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

	"github.com/hightrex/microservices-platform/services/audit-service/api"
	"github.com/hightrex/microservices-platform/services/audit-service/internal/config"
	"github.com/hightrex/microservices-platform/services/audit-service/internal/consumer"
	"github.com/hightrex/microservices-platform/services/audit-service/internal/handlers"
	"github.com/hightrex/microservices-platform/services/audit-service/internal/repository/postgres"
	"github.com/hightrex/microservices-platform/services/audit-service/internal/service"
)

func main() {
	// 1. Logger
	logger.Setup(logger.Config{Level: "debug", Environment: "dev"})

	// 2. Config
	var cfg config.Config
	if err := libconfig.Load(".", "config", &cfg); err != nil {
		logger.Fatal().Err(err).Msg("Failed to load config")
	}

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
		tp, err := tracing.InitTracer(ctx, "audit-service", cfg.Tracing.CollectorEndpoint)
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
	auditRepo := postgres.NewAuditRepo(pool)
	retentionRepo := postgres.NewRetentionRepo(pool)
	exportRepo := postgres.NewExportRepo(pool)

	// 8. Event publisher (Redis Streams — shared DB 0)
	messagingRedis := getMessagingRedisClient(cfg.Redis)
	defer messagingRedis.Close()
	eventPublisher := messaging.NewProducer(messagingRedis, "audit-service")

	// 9. Wire services
	auditSvc := service.NewAuditService(auditRepo, eventPublisher, cfg.Compliance.HashChainingEnabled)
	retentionSvc := service.NewRetentionService(retentionRepo, eventPublisher)
	exportSvc := service.NewExportService(exportRepo, auditRepo, eventPublisher)

	// 10. Wire handlers
	auditHandler := handlers.NewAuditHandler(auditSvc)
	exportHandler := handlers.NewExportHandler(exportSvc)
	retentionHandler := handlers.NewRetentionHandler(retentionSvc)

	// 11. Health checks
	healthManager := health.NewManager()
	healthManager.AddCheck(handlers.NewPostgresCheck(pool))
	healthManager.AddCheck(handlers.NewRedisCheck(redisClient))

	// 12. Setup router and register routes
	r := gin.New()
	api.RegisterRoutes(r, auditHandler, exportHandler, retentionHandler, healthManager)

	// 13. Initialize and start event consumer
	// The audit service subscribes to ALL event streams to capture everything
	eventConsumer := consumer.NewEventConsumer(auditSvc)
	consumerManager := consumer.NewConsumer(messagingRedis, consumer.Config{
		GroupName:    "audit-service-consumer-group",
		ConsumerName: fmt.Sprintf("audit-service-%s", os.Getenv("HOSTNAME")),
		Concurrency:  1,
	})
	healthManager.AddCheck(consumerManager)

	// Subscribe to ALL service event streams with catch-all handler
	consumerManager.RegisterHandler("auth-events", "*", eventConsumer.HandleAllEvents)
	consumerManager.RegisterHandler("org-events", "*", eventConsumer.HandleAllEvents)
	consumerManager.RegisterHandler("notification-events", "*", eventConsumer.HandleAllEvents)
	consumerManager.RegisterHandler("billing-events", "*", eventConsumer.HandleAllEvents)
	consumerManager.RegisterHandler("file-events", "*", eventConsumer.HandleAllEvents)

	if err := consumerManager.Start(ctx); err != nil {
		logger.Error().Err(err).Msg("Failed to start event consumer")
	}

	// 13b. Start retention enforcement worker (background)
	retentionWorker := service.NewRetentionWorker(auditRepo, retentionRepo)
	retentionCtx, retentionCancel := context.WithCancel(ctx)
	defer retentionCancel()
	go retentionWorker.Start(retentionCtx)

	// 14. Start server with graceful shutdown
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
		logger.Info().Int("port", cfg.ServerPort).Msg("Audit service starting")
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			logger.Fatal().Err(err).Msg("Server listen failed")
		}
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit
	logger.Info().Msg("Shutting down server...")

	consumerManager.Stop()
	retentionCancel()

	shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := srv.Shutdown(shutdownCtx); err != nil {
		logger.Fatal().Err(err).Msg("Server forced to shutdown")
	}

	logger.Info().Msg("Audit service exited")
}

// getMessagingRedisClient returns a Redis client for the shared messaging bus (always DB 0).
func getMessagingRedisClient(cfg cache.Config) *redis.Client {
	return redis.NewClient(&redis.Options{
		Addr:     cfg.Address,
		Password: cfg.Password,
		DB:       0, // Shared messaging bus — all services use DB 0 for Redis Streams
	})
}
