#!/bin/bash
set -e

SERVICE_NAME=$1

if [ -z "$SERVICE_NAME" ]; then
    echo "Usage: ./scripts/create-service.sh <service-name>"
    exit 1
fi

TARGET_DIR="services/$SERVICE_NAME"

if [ -d "$TARGET_DIR" ]; then
    echo "Service $SERVICE_NAME already exists."
    exit 1
fi

echo "Creating service: $SERVICE_NAME"

mkdir -p "$TARGET_DIR/cmd"
mkdir -p "$TARGET_DIR/internal/config"
mkdir -p "$TARGET_DIR/internal/handlers"
mkdir -p "$TARGET_DIR/internal/service"
mkdir -p "$TARGET_DIR/internal/repository/postgres"
mkdir -p "$TARGET_DIR/internal/models"
mkdir -p "$TARGET_DIR/api/middleware"

# Create main.go
cat > "$TARGET_DIR/cmd/main.go" <<EOF
package main

import (
	"context"
	"os"
	"os/signal"
	"syscall"
	"time"

    "github.com/gin-gonic/gin"
	"github.com/hightrex/microservices-platform/libs/go/pkg/logger"
    "github.com/hightrex/microservices-platform/libs/go/pkg/config"
)

type Config struct {
    ServerPort int \`mapstructure:"server_port"\`
}

func main() {
    // 1. Setup Logger
    logger.Setup(logger.Config{Level: "debug", Environment: "dev"})

    // 2. Load Config
    var cfg Config
    if err := config.Load(".", "config", &cfg); err != nil {
        // Fallback or exit
        logger.Warn().Err(err).Msg("Failed to load config, using defaults")
        cfg.ServerPort = 8080
    }

    // 3. Setup Router
    r := gin.New()
    r.Use(gin.Recovery())

    r.GET("/health", func(c *gin.Context) {
        c.JSON(200, gin.H{"status": "UP"})
    })

    // 4. Start Server
    srv := &http.Server{
        Addr:    fmt.Sprintf(":%d", cfg.ServerPort),
        Handler: r,
    }

    go func() {
        if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
            logger.Fatal().Err(err).Msg("listen: %s\n", err)
        }
    }()

    // 5. Graceful Shutdown
    quit := make(chan os.Signal, 1)
    signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
    <-quit
    logger.Info().Msg("Shutting down server...")

    ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
    defer cancel()
    if err := srv.Shutdown(ctx); err != nil {
        logger.Fatal().Err(err).Msg("Server forced to shutdown")
    }

    logger.Info().Msg("Server exiting")
}
EOF

# Create go.mod
cd "$TARGET_DIR"
go mod init "github.com/hightrex/microservices-platform/services/$SERVICE_NAME"
go mod edit -replace github.com/hightrex/microservices-platform/libs/go=../../libs/go
go get github.com/gin-gonic/gin
go get github.com/hightrex/microservices-platform/libs/go

echo "Service $SERVICE_NAME scaffolded."
