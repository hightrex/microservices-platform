package logger

import (
	"bytes"
	"context"
	"testing"

	"github.com/rs/zerolog"
	"github.com/stretchr/testify/assert"
)

func TestLoggerSetup(t *testing.T) {
	// Test setup doesn't panic
	Setup(Config{Level: "debug", Environment: "test"})
	assert.NotNil(t, Get())
}

func TestContextLogger(t *testing.T) {
	ctx := context.Background()
	l := FromContext(ctx)
	// Default zerolog behavior returns disabled logger if not found?
	// Actually zerolog.Ctx returns a pointer to the logger in context or a disabled logger.
	assert.NotNil(t, l)

	// With context
	logger := zerolog.New(zerolog.NewConsoleWriter())
	ctx = logger.WithContext(ctx)
	l2 := FromContext(ctx)
	assert.NotNil(t, l2)
}

func TestLogHelpers(t *testing.T) {
	// Basic test to ensure helpers don't panic
	// Capturing output is tricky with the global logger, but we verify no panic
	memLog := &bytes.Buffer{}
	validLogger := zerolog.New(memLog)

	// Swap global logger temporarily (not thread safe but okay for this unit test)
	oldLog := log
	log = validLogger
	defer func() { log = oldLog }()

	Info().Msg("info message")
	assert.Contains(t, memLog.String(), "info message")
}
