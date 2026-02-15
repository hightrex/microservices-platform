package postgres

import (
	"context"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/hightrex/microservices-platform/libs/go/pkg/errors"
	"github.com/hightrex/microservices-platform/libs/go/pkg/tenant"
	"github.com/hightrex/microservices-platform/services/notification-service/internal/models"
)

// DeliveryRepo implements service.DeliveryRepository using PostgreSQL.
type DeliveryRepo struct {
	db *pgxpool.Pool
}

// NewDeliveryRepo creates a new DeliveryRepo.
func NewDeliveryRepo(db *pgxpool.Pool) *DeliveryRepo {
	return &DeliveryRepo{db: db}
}

// Create inserts a new delivery log entry.
func (r *DeliveryRepo) Create(ctx context.Context, log *models.DeliveryLog) error {
	tenantID, err := tenant.RequireTenant(ctx)
	if err != nil {
		return err
	}

	log.ID = uuid.New()
	log.TenantID = tenantID
	log.CreatedAt = time.Now()
	if log.ProviderResponse == nil {
		log.ProviderResponse = []byte("{}")
	}

	_, err = r.db.Exec(ctx,
		`INSERT INTO notification_delivery_log (id, notification_id, tenant_id, channel, recipient, status, error_message, provider_response, retry_count, delivered_at, created_at)
		 VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11)`,
		log.ID, log.NotificationID, log.TenantID, log.Channel, log.Recipient,
		log.Status, log.ErrorMessage, log.ProviderResponse, log.RetryCount,
		log.DeliveredAt, log.CreatedAt,
	)
	if err != nil {
		return errors.InternalServerError("Failed to create delivery log", err)
	}
	return nil
}

// List retrieves delivery log entries for a notification.
func (r *DeliveryRepo) List(ctx context.Context, notificationID uuid.UUID) ([]models.DeliveryLog, error) {
	tenantID, err := tenant.RequireTenant(ctx)
	if err != nil {
		return nil, err
	}

	rows, err := r.db.Query(ctx,
		`SELECT id, notification_id, tenant_id, channel, recipient, status, error_message, provider_response, retry_count, delivered_at, created_at
		 FROM notification_delivery_log WHERE notification_id = $1 AND tenant_id = $2
		 ORDER BY created_at DESC`,
		notificationID, tenantID,
	)
	if err != nil {
		return nil, errors.InternalServerError("Failed to list delivery logs", err)
	}
	defer rows.Close()

	var logs []models.DeliveryLog
	for rows.Next() {
		var l models.DeliveryLog
		if err := rows.Scan(
			&l.ID, &l.NotificationID, &l.TenantID, &l.Channel, &l.Recipient,
			&l.Status, &l.ErrorMessage, &l.ProviderResponse, &l.RetryCount,
			&l.DeliveredAt, &l.CreatedAt,
		); err != nil {
			return nil, errors.InternalServerError("Failed to scan delivery log", err)
		}
		logs = append(logs, l)
	}

	return logs, nil
}

// GetFailedDeliveries retrieves failed deliveries eligible for retry.
func (r *DeliveryRepo) GetFailedDeliveries(ctx context.Context, maxRetries int) ([]models.DeliveryLog, error) {
	rows, err := r.db.Query(ctx,
		`SELECT id, notification_id, tenant_id, channel, recipient, status, error_message, provider_response, retry_count, delivered_at, created_at
		 FROM notification_delivery_log WHERE status = 'failed' AND retry_count < $1
		 ORDER BY created_at ASC LIMIT 100`,
		maxRetries,
	)
	if err != nil {
		return nil, errors.InternalServerError("Failed to get failed deliveries", err)
	}
	defer rows.Close()

	var logs []models.DeliveryLog
	for rows.Next() {
		var l models.DeliveryLog
		if err := rows.Scan(
			&l.ID, &l.NotificationID, &l.TenantID, &l.Channel, &l.Recipient,
			&l.Status, &l.ErrorMessage, &l.ProviderResponse, &l.RetryCount,
			&l.DeliveredAt, &l.CreatedAt,
		); err != nil {
			return nil, errors.InternalServerError("Failed to scan delivery log", err)
		}
		logs = append(logs, l)
	}

	return logs, nil
}
