package postgres

import (
	"context"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/hightrex/microservices-platform/libs/go/pkg/errors"
)

// DLQRepo implements service.DLQRepository using PostgreSQL.
type DLQRepo struct {
	db *pgxpool.Pool
}

// NewDLQRepo creates a new DLQRepo.
func NewDLQRepo(db *pgxpool.Pool) *DLQRepo {
	return &DLQRepo{db: db}
}

// Insert adds a failed notification event to the dead-letter queue.
func (r *DLQRepo) Insert(ctx context.Context, tenantID uuid.UUID, eventType string, payload []byte, errMsg string) error {
	_, err := r.db.Exec(ctx,
		`INSERT INTO notification_dlq (tenant_id, event_type, payload, error) VALUES ($1, $2, $3, $4)`,
		tenantID, eventType, payload, errMsg,
	)
	if err != nil {
		return errors.InternalServerError("Failed to insert into notification DLQ", err)
	}
	return nil
}
