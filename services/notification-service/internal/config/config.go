package config

import (
	"github.com/hightrex/microservices-platform/libs/go/pkg/cache"
	"github.com/hightrex/microservices-platform/libs/go/pkg/database"
)

// Config holds all configuration for the notification-service.
type Config struct {
	ServerPort  int             `mapstructure:"server_port"`
	Environment string          `mapstructure:"environment"`
	LogLevel    string          `mapstructure:"log_level"`
	Database    database.Config `mapstructure:"database"`
	Redis       cache.Config    `mapstructure:"redis"`
	Tracing     TracingConfig   `mapstructure:"tracing"`
	SMTP        SMTPConfig      `mapstructure:"smtp"`
	Twilio      TwilioConfig    `mapstructure:"twilio"`
	Webhook     WebhookConfig   `mapstructure:"webhook"`
	Template    TemplateConfig  `mapstructure:"template"`
}

// TracingConfig holds OpenTelemetry tracing configuration.
type TracingConfig struct {
	Enabled           bool   `mapstructure:"enabled"`
	CollectorEndpoint string `mapstructure:"collector_endpoint"`
}

// SMTPConfig holds email delivery configuration.
type SMTPConfig struct {
	Host        string `mapstructure:"host"`
	Port        int    `mapstructure:"port"`
	Username    string `mapstructure:"username"`
	Password    string `mapstructure:"password"`
	FromAddress string `mapstructure:"from_address"`
	FromName    string `mapstructure:"from_name"`
	UseTLS      bool   `mapstructure:"use_tls"`
}

// TwilioConfig holds SMS delivery configuration.
type TwilioConfig struct {
	AccountSID string `mapstructure:"account_sid"`
	AuthToken  string `mapstructure:"auth_token"`
	FromNumber string `mapstructure:"from_number"`
}

// WebhookConfig holds webhook delivery configuration.
type WebhookConfig struct {
	TimeoutSeconds int `mapstructure:"timeout_seconds"`
	RetryAttempts  int `mapstructure:"retry_attempts"`
	MaxRedirects   int `mapstructure:"max_redirects"`
}

// TemplateConfig holds template engine configuration.
type TemplateConfig struct {
	DefaultLanguage    string   `mapstructure:"default_language"`
	SupportedLanguages []string `mapstructure:"supported_languages"`
}
