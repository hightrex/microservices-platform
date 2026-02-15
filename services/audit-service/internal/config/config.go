package config

import (
	"github.com/hightrex/microservices-platform/libs/go/pkg/cache"
	"github.com/hightrex/microservices-platform/libs/go/pkg/database"
)

// Config holds all configuration for the audit-service.
type Config struct {
	ServerPort  int              `mapstructure:"server_port"`
	Environment string           `mapstructure:"environment"`
	LogLevel    string           `mapstructure:"log_level"`
	Database    database.Config  `mapstructure:"database"`
	Redis       cache.Config     `mapstructure:"redis"`
	Tracing     TracingConfig    `mapstructure:"tracing"`
	Retention   RetentionConfig  `mapstructure:"retention"`
	Export      ExportConfig     `mapstructure:"export"`
	Compliance  ComplianceConfig `mapstructure:"compliance"`
}

// TracingConfig holds OpenTelemetry tracing configuration.
type TracingConfig struct {
	Enabled           bool   `mapstructure:"enabled"`
	CollectorEndpoint string `mapstructure:"collector_endpoint"`
}

// RetentionConfig holds audit log retention defaults.
type RetentionConfig struct {
	DefaultDays        int `mapstructure:"default_days"`
	SecurityEventsDays int `mapstructure:"security_events_days"`
}

// ExportConfig holds audit export settings.
type ExportConfig struct {
	MaxExportSizeMB int      `mapstructure:"max_export_size_mb"`
	AllowedFormats  []string `mapstructure:"allowed_formats"`
}

// ComplianceConfig holds compliance mode settings.
type ComplianceConfig struct {
	HashChainingEnabled bool `mapstructure:"hash_chaining_enabled"`
}
