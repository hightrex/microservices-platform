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

// PreferenceRepo implements service.PreferenceRepository using PostgreSQL.
type PreferenceRepo struct {
	db *pgxpool.Pool
}

// NewPreferenceRepo creates a new PreferenceRepo.
func NewPreferenceRepo(db *pgxpool.Pool) *PreferenceRepo {
	return &PreferenceRepo{db: db}
}

// GetUserPreferences retrieves all notification preferences for a user, scoped to tenant.
func (r *PreferenceRepo) GetUserPreferences(ctx context.Context, userID uuid.UUID) ([]models.NotificationPreference, error) {
	tenantID, err := tenant.RequireTenant(ctx)
	if err != nil {
		return nil, err
	}

	rows, err := r.db.Query(ctx,
		`SELECT id, user_id, tenant_id, channel, event_type, enabled, updated_at
		 FROM notification_preferences WHERE user_id = $1 AND tenant_id = $2
		 ORDER BY event_type, channel`,
		userID, tenantID,
	)
	if err != nil {
		return nil, errors.InternalServerError("Failed to get user preferences", err)
	}
	defer rows.Close()

	var prefs []models.NotificationPreference
	for rows.Next() {
		var p models.NotificationPreference
		if err := rows.Scan(&p.ID, &p.UserID, &p.TenantID, &p.Channel, &p.EventType, &p.Enabled, &p.UpdatedAt); err != nil {
			return nil, errors.InternalServerError("Failed to scan preference", err)
		}
		prefs = append(prefs, p)
	}

	return prefs, nil
}

// UpdatePreference upserts a notification preference for a user, scoped to tenant.
func (r *PreferenceRepo) UpdatePreference(ctx context.Context, userID uuid.UUID, eventType string, channel models.Channel, enabled bool) error {
	tenantID, err := tenant.RequireTenant(ctx)
	if err != nil {
		return err
	}

	_, err = r.db.Exec(ctx,
		`INSERT INTO notification_preferences (id, user_id, tenant_id, channel, event_type, enabled, updated_at)
		 VALUES ($1, $2, $3, $4, $5, $6, $7)
		 ON CONFLICT (user_id, tenant_id, channel, event_type)
		 DO UPDATE SET enabled = $6, updated_at = $7`,
		uuid.New(), userID, tenantID, channel, eventType, enabled, time.Now(),
	)
	if err != nil {
		return errors.InternalServerError("Failed to update preference", err)
	}
	return nil
}

// CheckEnabled checks if a notification channel is enabled for a specific event type and user.
// Returns true (enabled) if no preference is found (opt-in by default).
func (r *PreferenceRepo) CheckEnabled(ctx context.Context, userID uuid.UUID, eventType string, channel models.Channel) (bool, error) {
	tenantID, err := tenant.RequireTenant(ctx)
	if err != nil {
		return false, err
	}

	var enabled bool
	err = r.db.QueryRow(ctx,
		`SELECT enabled FROM notification_preferences
		 WHERE user_id = $1 AND tenant_id = $2 AND channel = $3 AND event_type = $4`,
		userID, tenantID, channel, eventType,
	).Scan(&enabled)
	if err != nil {
		// No preference found — default is enabled
		return true, nil
	}
	return enabled, nil
}
