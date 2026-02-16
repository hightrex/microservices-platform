package main

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/redis/go-redis/v9"

	"github.com/hightrex/microservices-platform/libs/go/pkg/cache"
	libconfig "github.com/hightrex/microservices-platform/libs/go/pkg/config"
	"github.com/hightrex/microservices-platform/libs/go/pkg/database"
	"github.com/hightrex/microservices-platform/libs/go/pkg/health"
	"github.com/hightrex/microservices-platform/libs/go/pkg/logger"
	"github.com/hightrex/microservices-platform/libs/go/pkg/messaging"
	"github.com/hightrex/microservices-platform/libs/go/pkg/tracing"

	"github.com/hightrex/microservices-platform/services/notification-service/api"
	"github.com/hightrex/microservices-platform/services/notification-service/internal/config"
	"github.com/hightrex/microservices-platform/services/notification-service/internal/consumer"
	"github.com/hightrex/microservices-platform/services/notification-service/internal/handlers"
	"github.com/hightrex/microservices-platform/services/notification-service/internal/repository/postgres"
	redisrepo "github.com/hightrex/microservices-platform/services/notification-service/internal/repository/redis"
	"github.com/hightrex/microservices-platform/services/notification-service/internal/service"
	"github.com/hightrex/microservices-platform/services/notification-service/internal/service/channels"
)

func main() {
	// 1. Logger
	logger.Setup(logger.Config{Level: "debug", Environment: "dev"})

	// 2. Config
	var cfg config.Config
	if err := libconfig.Load(".", "config", &cfg); err != nil {
		logger.Fatal().Err(err).Msg("Failed to load config")
	}

	// Override logger level from config
	logger.Setup(logger.Config{Level: cfg.LogLevel, Environment: cfg.Environment})

	// 3. Database connection
	ctx := context.Background()
	pool, err := database.Connect(ctx, cfg.Database)
	if err != nil {
		logger.Fatal().Err(err).Msg("Failed to connect to database")
	}
	defer pool.Close()

	// 4. Run migrations
	if err := database.Migrate(cfg.Database, "migrations"); err != nil {
		logger.Fatal().Err(err).Msg("Failed to run migrations")
	}

	// 5. Redis connection
	redisClient, err := cache.New(ctx, cfg.Redis)
	if err != nil {
		logger.Fatal().Err(err).Msg("Failed to connect to Redis")
	}
	defer redisClient.Close()

	// 6. Tracing (optional)
	if cfg.Tracing.Enabled {
		tp, err := tracing.InitTracer(ctx, "notification-service", cfg.Tracing.CollectorEndpoint)
		if err != nil {
			logger.Warn().Err(err).Msg("Failed to initialize tracer, continuing without tracing")
		} else {
			defer func() {
				if err := tp.Shutdown(ctx); err != nil {
					logger.Warn().Err(err).Msg("Failed to shutdown tracer")
				}
			}()
		}
	}

	// 7. Wire repositories
	templateRepo := postgres.NewTemplateRepo(pool)
	notifRepo := postgres.NewNotificationRepo(pool)
	prefRepo := postgres.NewPreferenceRepo(pool)
	deliveryRepo := postgres.NewDeliveryRepo(pool)
	notifCache := redisrepo.NewNotificationCache(redisClient)

	// 8. Event publisher (Redis Streams — shared DB 0)
	messagingRedis := getMessagingRedisClient(cfg.Redis)
	defer messagingRedis.Close()
	eventPublisher := messaging.NewProducer(messagingRedis, "notification-service")

	// 9. Wire services
	templateSvc := service.NewTemplateService(templateRepo, notifCache, eventPublisher)
	notifSvc := service.NewNotificationService(
		notifRepo, templateRepo, prefRepo, deliveryRepo, notifCache, eventPublisher, templateSvc,
	)
	prefSvc := service.NewPreferenceService(prefRepo, notifCache, eventPublisher)

	// Register channel implementations
	notifSvc.RegisterChannel(channels.NewEmailChannel(channels.EmailConfig{
		Host:        cfg.SMTP.Host,
		Port:        cfg.SMTP.Port,
		Username:    cfg.SMTP.Username,
		Password:    cfg.SMTP.Password,
		FromAddress: cfg.SMTP.FromAddress,
		FromName:    cfg.SMTP.FromName,
		UseTLS:      cfg.SMTP.UseTLS,
	}))
	notifSvc.RegisterChannel(channels.NewSMSChannel(channels.SMSConfig{
		AccountSID: cfg.Twilio.AccountSID,
		AuthToken:  cfg.Twilio.AuthToken,
		FromNumber: cfg.Twilio.FromNumber,
	}))
	notifSvc.RegisterChannel(channels.NewInAppChannel())
	notifSvc.RegisterChannel(channels.NewWebhookChannel(channels.WebhookConfig{
		TimeoutSeconds: cfg.Webhook.TimeoutSeconds,
		RetryAttempts:  cfg.Webhook.RetryAttempts,
		MaxRedirects:   cfg.Webhook.MaxRedirects,
	}))

	// 10. Wire handlers
	notifHandler := handlers.NewNotificationHandler(notifSvc)
	templateHandler := handlers.NewTemplateHandler(templateSvc)
	preferenceHandler := handlers.NewPreferenceHandler(prefSvc)

	// 11. Health checks
	healthManager := health.NewManager()
	healthManager.AddCheck(handlers.NewPostgresCheck(pool))
	healthManager.AddCheck(handlers.NewRedisCheck(redisClient))

	// 12. Setup router and register routes
	r := gin.New()
	api.RegisterRoutes(r, notifHandler, templateHandler, preferenceHandler, healthManager)

	// 13. Initialize and start event consumer
	eventConsumer := consumer.NewEventConsumer(notifSvc, templateSvc)
	consumerManager := consumer.NewConsumer(messagingRedis, consumer.Config{
		GroupName:    "notification-service-consumer-group",
		ConsumerName: fmt.Sprintf("notification-service-%s", os.Getenv("HOSTNAME")),
		Concurrency:  1,
	})
	healthManager.AddCheck(consumerManager)

	// Subscribe to events from other services
	consumerManager.RegisterHandler("auth-events", "user.created", eventConsumer.HandleUserCreated)
	consumerManager.RegisterHandler("org-events", "org.created", eventConsumer.HandleOrgCreated)
	consumerManager.RegisterHandler("billing-events", "*", eventConsumer.HandleBillingEvent)
	consumerManager.RegisterHandler("file-events", "*", eventConsumer.HandleFileEvent)

	if err := consumerManager.Start(ctx); err != nil {
		logger.Error().Err(err).Msg("Failed to start event consumer")
	}

	// 14. Start retry worker (background)
	dlqRepo := postgres.NewDLQRepo(pool)
	retryWorker := service.NewRetryWorker(deliveryRepo, notifRepo, dlqRepo, notifSvc, 5)
	retryCtx, retryCancel := context.WithCancel(ctx)
	defer retryCancel()
	go retryWorker.Start(retryCtx)

	// 15. Start server with graceful shutdown
	srv := &http.Server{
		Addr:              fmt.Sprintf(":%d", cfg.ServerPort),
		Handler:           r,
		ReadHeaderTimeout: 10 * time.Second,
		ReadTimeout:       30 * time.Second,
		WriteTimeout:      30 * time.Second,
		IdleTimeout:       120 * time.Second,
		MaxHeaderBytes:    1 << 20, // 1 MB
	}

	go func() {
		logger.Info().Int("port", cfg.ServerPort).Msg("Notification service starting")
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			logger.Fatal().Err(err).Msg("Server listen failed")
		}
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit
	logger.Info().Msg("Shutting down server...")

	// Stop consumer and retry worker
	consumerManager.Stop()
	retryCancel()

	shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := srv.Shutdown(shutdownCtx); err != nil {
		logger.Fatal().Err(err).Msg("Server forced to shutdown")
	}

	logger.Info().Msg("Notification service exited")
}

// getMessagingRedisClient returns a Redis client for the shared messaging bus (always DB 0).
func getMessagingRedisClient(cfg cache.Config) *redis.Client {
	return redis.NewClient(&redis.Options{
		Addr:     cfg.Address,
		Password: cfg.Password,
		DB:       0, // Shared messaging bus — all services use DB 0 for Redis Streams
	})
}
