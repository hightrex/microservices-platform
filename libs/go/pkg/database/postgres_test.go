package database

import (
	"context"
	"testing"
)

func TestConnectValidation(t *testing.T) {
	t.Skip("Skipping integration test requiring running Postgres")

	cfg := Config{
		Host:     "localhost",
		Port:     5432,
		User:     "postgres",
		Password: "password",
		Name:     "test_db",
	}

	_, err := Connect(context.Background(), cfg)
	if err != nil {
		t.Skip("Database not available")
	}
}
