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

func NewPostgresCheck(pool *pgxpool.Pool) *PostgresCheck {
	return &PostgresCheck{pool: pool}
}

func (c *PostgresCheck) Name() string           { return "postgres" }
func (c *PostgresCheck) Check(ctx context.Context) error { return c.pool.Ping(ctx) }

// RedisCheck implements health.Check for Redis.
type RedisCheck struct {
	client *cache.Client
}

func NewRedisCheck(client *cache.Client) *RedisCheck {
	return &RedisCheck{client: client}
}

func (c *RedisCheck) Name() string { return "redis" }
func (c *RedisCheck) Check(ctx context.Context) error {
	var dummy string
	err := c.client.Get(ctx, "health:ping", &dummy)
	if err != nil && err.Error() == "key not found: health:ping" {
		return nil
	}
	return err
}
