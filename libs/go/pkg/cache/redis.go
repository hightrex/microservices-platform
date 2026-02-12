package cache

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/hightrex/microservices-platform/libs/go/pkg/logger"
	"github.com/redis/go-redis/v9"
)

// Config holds Redis configuration
type Config struct {
	Address  string `mapstructure:"address"`
	Password string `mapstructure:"password"`
	DB       int    `mapstructure:"db"`
}

// Client wraps the Redis client
type Client struct {
	rdb *redis.Client
}

// New creates a new Redis client
func New(ctx context.Context, cfg Config) (*Client, error) {
	rdb := redis.NewClient(&redis.Options{
		Addr:     cfg.Address,
		Password: cfg.Password,
		DB:       cfg.DB,
	})

	if err := rdb.Ping(ctx).Err(); err != nil {
		return nil, fmt.Errorf("failed to ping redis: %w", err)
	}

	logger.Info().Msg("Connected to Redis")
	return &Client{rdb: rdb}, nil
}

// Set stores a value in Redis with a TTL
func (c *Client) Set(ctx context.Context, key string, value interface{}, ttl time.Duration) error {
	if ttl == 0 {
		return fmt.Errorf("TTL is required for all cache operations")
	}

	bytes, err := json.Marshal(value)
	if err != nil {
		return fmt.Errorf("failed to marshal value: %w", err)
	}

	return c.rdb.Set(ctx, key, bytes, ttl).Err()
}

// Get retrieves a value from Redis
func (c *Client) Get(ctx context.Context, key string, dest interface{}) error {
	bytes, err := c.rdb.Get(ctx, key).Bytes()
	if err != nil {
		if err == redis.Nil {
			return fmt.Errorf("key not found: %s", key)
		}
		return fmt.Errorf("failed to get key: %w", err)
	}

	return json.Unmarshal(bytes, dest)
}

// Delete removes a value from Redis
func (c *Client) Delete(ctx context.Context, key string) error {
	return c.rdb.Del(ctx, key).Err()
}

// Close closes the Redis connection
func (c *Client) Close() error {
	return c.rdb.Close()
}
