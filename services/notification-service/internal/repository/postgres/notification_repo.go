package postgres

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/hightrex/microservices-platform/libs/go/pkg/errors"
	"github.com/hightrex/microservices-platform/libs/go/pkg/tenant"
	"github.com/hightrex/microservices-platform/services/notification-service/internal/models"
)

// NotificationRepo implements service.NotificationRepository using PostgreSQL.
type NotificationRepo struct {
	db *pgxpool.Pool
}

// NewNotificationRepo creates a new NotificationRepo.
func NewNotificationRepo(db *pgxpool.Pool) *NotificationRepo {
	return &NotificationRepo{db: db}
}

// Create inserts a new notification record.
func (r *NotificationRepo) Create(ctx context.Context, notif *models.Notification) error {
	tenantID, err := tenant.RequireTenant(ctx)
	if err != nil {
		return err
	}

	notif.ID = uuid.New()
	notif.TenantID = tenantID
	notif.CreatedAt = time.Now()
	if notif.Status == "" {
		notif.Status = models.StatusPending
	}
	if notif.Metadata == nil {
		notif.Metadata = []byte("{}")
	}

	_, err = r.db.Exec(ctx,
		`INSERT INTO notifications (id, tenant_id, user_id, channel, event_type, subject, body, status, metadata, sent_at, read_at, created_at)
		 VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12)`,
		notif.ID, notif.TenantID, notif.UserID, notif.Channel, notif.EventType,
		notif.Subject, notif.Body, notif.Status, notif.Metadata, notif.SentAt, notif.ReadAt, notif.CreatedAt,
	)
	if err != nil {
		return errors.InternalServerError("Failed to create notification", err)
	}
	return nil
}

// GetByID retrieves a notification by ID, scoped to tenant.
func (r *NotificationRepo) GetByID(ctx context.Context, id uuid.UUID) (*models.Notification, error) {
	tenantID, err := tenant.RequireTenant(ctx)
	if err != nil {
		return nil, err
	}

	var notif models.Notification
	err = r.db.QueryRow(ctx,
		`SELECT id, tenant_id, user_id, channel, event_type, subject, body, status, metadata, sent_at, read_at, created_at
		 FROM notifications WHERE id = $1 AND tenant_id = $2`,
		id, tenantID,
	).Scan(
		&notif.ID, &notif.TenantID, &notif.UserID, &notif.Channel, &notif.EventType,
		&notif.Subject, &notif.Body, &notif.Status, &notif.Metadata, &notif.SentAt, &notif.ReadAt, &notif.CreatedAt,
	)
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, errors.NotFound("Notification not found", nil)
		}
		return nil, errors.InternalServerError("Failed to get notification", err)
	}
	return &notif, nil
}

// List retrieves notifications for a user with filtering and pagination, scoped to tenant.
func (r *NotificationRepo) List(ctx context.Context, userID uuid.UUID, filter models.NotificationFilter, page models.Pagination) ([]models.Notification, int, error) {
	tenantID, err := tenant.RequireTenant(ctx)
	if err != nil {
		return nil, 0, err
	}

	var conditions []string
	var args []interface{}
	argIdx := 1

	conditions = append(conditions, fmt.Sprintf("tenant_id = $%d", argIdx))
	args = append(args, tenantID)
	argIdx++

	conditions = append(conditions, fmt.Sprintf("user_id = $%d", argIdx))
	args = append(args, userID)
	argIdx++

	if filter.Status != nil {
		conditions = append(conditions, fmt.Sprintf("status = $%d", argIdx))
		args = append(args, *filter.Status)
		argIdx++
	}

	if filter.Channel != nil {
		conditions = append(conditions, fmt.Sprintf("channel = $%d", argIdx))
		args = append(args, *filter.Channel)
		argIdx++
	}

	if filter.EventType != nil {
		conditions = append(conditions, fmt.Sprintf("event_type = $%d", argIdx))
		args = append(args, *filter.EventType)
		argIdx++
	}

	where := strings.Join(conditions, " AND ")

	// Count total
	var total int
	countQuery := fmt.Sprintf("SELECT COUNT(*) FROM notifications WHERE %s", where)
	if err := r.db.QueryRow(ctx, countQuery, args...).Scan(&total); err != nil {
		return nil, 0, errors.InternalServerError("Failed to count notifications", err)
	}

	// Fetch page
	offset := (page.Page - 1) * page.PageSize
	listQuery := fmt.Sprintf(
		`SELECT id, tenant_id, user_id, channel, event_type, subject, body, status, metadata, sent_at, read_at, created_at
		 FROM notifications WHERE %s ORDER BY created_at DESC LIMIT $%d OFFSET $%d`,
		where, argIdx, argIdx+1,
	)
	args = append(args, page.PageSize, offset)

	rows, err := r.db.Query(ctx, listQuery, args...)
	if err != nil {
		return nil, 0, errors.InternalServerError("Failed to list notifications", err)
	}
	defer rows.Close()

	var notifications []models.Notification
	for rows.Next() {
		var n models.Notification
		if err := rows.Scan(
			&n.ID, &n.TenantID, &n.UserID, &n.Channel, &n.EventType,
			&n.Subject, &n.Body, &n.Status, &n.Metadata, &n.SentAt, &n.ReadAt, &n.CreatedAt,
		); err != nil {
			return nil, 0, errors.InternalServerError("Failed to scan notification", err)
		}
		notifications = append(notifications, n)
	}

	return notifications, total, nil
}

// MarkAsRead sets the read_at timestamp on a notification, scoped to tenant.
func (r *NotificationRepo) MarkAsRead(ctx context.Context, id uuid.UUID) error {
	tenantID, err := tenant.RequireTenant(ctx)
	if err != nil {
		return err
	}

	now := time.Now()
	tag, err := r.db.Exec(ctx,
		`UPDATE notifications SET status = $1, read_at = $2 WHERE id = $3 AND tenant_id = $4 AND status != 'read'`,
		models.StatusRead, now, id, tenantID,
	)
	if err != nil {
		return errors.InternalServerError("Failed to mark notification as read", err)
	}
	if tag.RowsAffected() == 0 {
		return errors.NotFound("Notification not found or already read", nil)
	}
	return nil
}

// CountUnread returns the count of unread notifications for a user, scoped to tenant.
func (r *NotificationRepo) CountUnread(ctx context.Context, userID uuid.UUID) (int, error) {
	tenantID, err := tenant.RequireTenant(ctx)
	if err != nil {
		return 0, err
	}

	var count int
	err = r.db.QueryRow(ctx,
		`SELECT COUNT(*) FROM notifications WHERE user_id = $1 AND tenant_id = $2 AND status IN ('pending', 'sent') AND channel = 'in_app'`,
		userID, tenantID,
	).Scan(&count)
	if err != nil {
		return 0, errors.InternalServerError("Failed to count unread notifications", err)
	}
	return count, nil
}

// UpdateStatus updates the status of a notification, scoped to tenant.
func (r *NotificationRepo) UpdateStatus(ctx context.Context, id uuid.UUID, status models.NotificationStatus) error {
	tenantID, err := tenant.RequireTenant(ctx)
	if err != nil {
		return err
	}

	var sentAt *time.Time
	if status == models.StatusSent {
		now := time.Now()
		sentAt = &now
	}

	tag, err := r.db.Exec(ctx,
		`UPDATE notifications SET status = $1, sent_at = COALESCE($2, sent_at) WHERE id = $3 AND tenant_id = $4`,
		status, sentAt, id, tenantID,
	)
	if err != nil {
		return errors.InternalServerError("Failed to update notification status", err)
	}
	if tag.RowsAffected() == 0 {
		return errors.NotFound("Notification not found", nil)
	}
	return nil
}
