package config

import (
	"time"

	"github.com/hightrex/microservices-platform/libs/go/pkg/cache"
	"github.com/hightrex/microservices-platform/libs/go/pkg/database"
)

// Config holds all configuration for the auth-service.
type Config struct {
	ServerPort  int             `mapstructure:"server_port"`
	Environment string          `mapstructure:"environment"`
	LogLevel    string          `mapstructure:"log_level"`
	Database    database.Config `mapstructure:"database"`
	Redis       cache.Config    `mapstructure:"redis"`
	JWT         JWTConfig       `mapstructure:"jwt"`
	Tracing     TracingConfig   `mapstructure:"tracing"`
	MFA         MFAConfig       `mapstructure:"mfa"`
	Security    SecurityConfig  `mapstructure:"security"`
}

// JWTConfig holds JWT signing and TTL configuration.
type JWTConfig struct {
	Secret          string        `mapstructure:"secret"`
	AccessTokenTTL  time.Duration `mapstructure:"access_token_ttl"`
	RefreshTokenTTL time.Duration `mapstructure:"refresh_token_ttl"`
	Issuer          string        `mapstructure:"issuer"`
}

// TracingConfig holds OpenTelemetry tracing configuration.
type TracingConfig struct {
	Enabled           bool   `mapstructure:"enabled"`
	CollectorEndpoint string `mapstructure:"collector_endpoint"`
}

// MFAConfig holds MFA-related configuration.
type MFAConfig struct {
	Issuer string `mapstructure:"issuer"`
}

// SecurityConfig holds security-related settings.
type SecurityConfig struct {
	BcryptCost           int `mapstructure:"bcrypt_cost"`
	MaxFailedLogins      int `mapstructure:"max_failed_logins"`
	LockoutDurationMins  int `mapstructure:"lockout_duration_mins"`
	PasswordMinLength    int `mapstructure:"password_min_length"`
	PasswordHistoryCount int `mapstructure:"password_history_count"`
}
