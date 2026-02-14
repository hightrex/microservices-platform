package config

import (
	"github.com/hightrex/microservices-platform/libs/go/pkg/cache"
	"github.com/hightrex/microservices-platform/libs/go/pkg/database"
)

// Config holds all configuration for the organization-service.
type Config struct {
	ServerPort  int             `mapstructure:"server_port"`
	Environment string          `mapstructure:"environment"`
	LogLevel    string          `mapstructure:"log_level"`
	Database    database.Config `mapstructure:"database"`
	Redis       cache.Config    `mapstructure:"redis"`
	Tracing     TracingConfig   `mapstructure:"tracing"`
	AuthService AuthServiceConfig `mapstructure:"auth_service"`
}

// TracingConfig holds OpenTelemetry tracing configuration.
type TracingConfig struct {
	Enabled           bool   `mapstructure:"enabled"`
	CollectorEndpoint string `mapstructure:"collector_endpoint"`
}

// AuthServiceConfig holds configuration for calling the auth service.
type AuthServiceConfig struct {
	BaseURL string `mapstructure:"base_url"`
}
