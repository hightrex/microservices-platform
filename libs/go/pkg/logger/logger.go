package logger

import (
	"context"
	"io"
	"os"
	"time"

	"github.com/rs/zerolog"
	"github.com/rs/zerolog/pkgerrors"
)

// Logger is a wrapper around zerolog.Logger
type Logger struct {
	logger zerolog.Logger
}

var (
	// Default logger instance
	log zerolog.Logger
)

func init() {
	// Default configuration
	zerolog.ErrorStackMarshaler = pkgerrors.MarshalStack
	zerolog.TimeFieldFormat = time.RFC3339

	// Default to console output for development, can be changed via config
	output := zerolog.ConsoleWriter{Out: os.Stdout, TimeFormat: time.RFC3339}
	log = zerolog.New(output).With().Timestamp().Logger()
}

// Config holds logger configuration
type Config struct {
	Level       string `mapstructure:"level"`
	Environment string `mapstructure:"environment"`
}

// Setup initializes the logger with the given configuration
func Setup(cfg Config) {
	level, err := zerolog.ParseLevel(cfg.Level)
	if err != nil {
		level = zerolog.InfoLevel
	}

	var output io.Writer
	if cfg.Environment == "local" || cfg.Environment == "dev" {
		output = zerolog.ConsoleWriter{Out: os.Stdout, TimeFormat: time.RFC3339}
	} else {
		output = os.Stdout
	}

	log = zerolog.New(output).Level(level).With().Timestamp().Caller().Logger()
}

// Get check returns the global logger instance
func Get() zerolog.Logger {
	return log
}

// FromContext retrieves the logger from the context
func FromContext(ctx context.Context) *zerolog.Logger {
	return zerolog.Ctx(ctx)
}

// WithContext adds the logger to the context
func WithContext(ctx context.Context) context.Context {
	return log.WithContext(ctx)
}

// Info logs a message at level Info
func Info() *zerolog.Event {
	return log.Info()
}

// Debug logs a message at level Debug
func Debug() *zerolog.Event {
	return log.Debug()
}

// Warn logs a message at level Warn
func Warn() *zerolog.Event {
	return log.Warn()
}

// Error logs a message at level Error
func Error() *zerolog.Event {
	return log.Error()
}

// Fatal logs a message at level Fatal
func Fatal() *zerolog.Event {
	return log.Fatal()
}
