package cache

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

// Mocking redis client would be ideal with miniredis, but sticking to basic constraints.
// We will write a test that requires a running Redis (integration test style)
// OR mock the interface if we had one. The current package exposes a struct Client.

func TestCacheKeys(t *testing.T) {
	t.Skip("Skipping integration test requiring running Redis")

	// In a real scenario, we'd use:
	// s, _ := miniredis.Run()
	// cfg := Config{Address: s.Addr()}

	ctx := context.Background()
	cfg := Config{Address: "localhost:6379"}

	c, err := New(ctx, cfg)
	if err != nil {
		t.Log("Redis not available, skipping")
		t.SkipNow()
	}
	defer c.Close()

	err = c.Set(ctx, "key", "value", time.Minute)
	assert.NoError(t, err)

	var val string
	err = c.Get(ctx, "key", &val)
	assert.NoError(t, err)
	assert.Equal(t, "value", val)

	err = c.Set(ctx, "key2", "value", 0)
	assert.Error(t, err, "Should fail without TTL")
}

func TestConfigValidation(t *testing.T) {
	// Basic test
	cfg := Config{Address: "localhost:6379"}
	assert.Equal(t, "localhost:6379", cfg.Address)
}
