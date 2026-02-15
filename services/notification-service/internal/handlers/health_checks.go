package handlers

import (
	"context"

	"github.com/hightrex/microservices-platform/libs/go/pkg/cache"
	"github.com/jackc/pgx/v5/pgxpool"
)

// PostgresCheck implements health.Check for the database.
type PostgresCheck struct {
	pool *pgxpool.Pool
}

// NewPostgresCheck creates a new PostgresCheck.
func NewPostgresCheck(pool *pgxpool.Pool) *PostgresCheck {
	return &PostgresCheck{pool: pool}
}

// Name returns the name of this health check.
func (c *PostgresCheck) Name() string {
	return "postgres"
}

// Check verifies the database connection is healthy.
func (c *PostgresCheck) Check(ctx context.Context) error {
	return c.pool.Ping(ctx)
}

// RedisCheck implements health.Check for Redis.
type RedisCheck struct {
	client *cache.Client
}

// NewRedisCheck creates a new RedisCheck.
func NewRedisCheck(client *cache.Client) *RedisCheck {
	return &RedisCheck{client: client}
}

// Name returns the name of this health check.
func (c *RedisCheck) Name() string {
	return "redis"
}

// Check verifies the Redis connection is healthy.
func (c *RedisCheck) Check(ctx context.Context) error {
	var dummy string
	err := c.client.Get(ctx, "health:ping", &dummy)
	if err != nil && err.Error() == "key not found: health:ping" {
		return nil
	}
	return err
}
