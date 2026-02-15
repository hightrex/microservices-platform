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

	"github.com/hightrex/microservices-platform/services/auth-service/api"
	"github.com/hightrex/microservices-platform/services/auth-service/internal/config"
	"github.com/hightrex/microservices-platform/services/auth-service/internal/handlers"
	"github.com/hightrex/microservices-platform/services/auth-service/internal/repository/postgres"
	redisrepo "github.com/hightrex/microservices-platform/services/auth-service/internal/repository/redis"
	"github.com/hightrex/microservices-platform/services/auth-service/internal/service"
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
		tp, err := tracing.InitTracer(ctx, "auth-service", cfg.Tracing.CollectorEndpoint)
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

	// 7. Wire repositories (concrete implementations)
	userRepo := postgres.NewUserRepo(pool)
	sessionRepo := postgres.NewSessionRepo(pool)
	apiKeyRepo := postgres.NewAPIKeyRepo(pool)
	roleRepo := postgres.NewRoleRepo(pool)
	pwHistoryRepo := postgres.NewPasswordHistoryRepo(pool)
	tokenCache := redisrepo.NewTokenCache(redisClient)

	// 8. Event publisher (Redis Streams)
	// Access underlying redis client for messaging producer
	eventPublisher := messaging.NewProducer(getRedisClient(cfg.Redis), "auth-service")

	// 9. Wire services (accept interfaces)
	tokenSvc := service.NewTokenService(cfg.JWT)
	authSvc := service.NewAuthService(
		userRepo, sessionRepo, roleRepo, pwHistoryRepo,
		tokenCache, tokenSvc, eventPublisher,
		cfg.Security, cfg.MFA,
	)
	userSvc := service.NewUserService(
		userRepo, sessionRepo, roleRepo, pwHistoryRepo,
		eventPublisher, cfg.Security,
	)

	// Suppress unused variable warnings — apiKeyRepo is used in later phases
	_ = apiKeyRepo

	// 10. Wire handlers
	authHandler := handlers.NewAuthHandler(authSvc)
	userHandler := handlers.NewUserHandler(userSvc)

	// 11. Health checks
	healthManager := health.NewManager()
	healthManager.AddCheck(handlers.NewPostgresCheck(pool))
	healthManager.AddCheck(handlers.NewRedisCheck(redisClient))

	// 12. Setup router and register routes
	r := gin.New()
	api.RegisterRoutes(r, authHandler, userHandler, tokenSvc, tokenCache, healthManager)

	// 13. Start server with graceful shutdown
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
		logger.Info().Int("port", cfg.ServerPort).Msg("Auth service starting")
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			logger.Fatal().Err(err).Msg("Server listen failed")
		}
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit
	logger.Info().Msg("Shutting down server...")

	shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := srv.Shutdown(shutdownCtx); err != nil {
		logger.Fatal().Err(err).Msg("Server forced to shutdown")
	}

	logger.Info().Msg("Auth service exited")
}

// getRedisClient creates a raw redis.Client for the messaging producer.
func getRedisClient(cfg cache.Config) *redis.Client {
	return redis.NewClient(&redis.Options{
		Addr:     cfg.Address,
		Password: cfg.Password,
		DB:       cfg.DB,
	})
}
