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
	"github.com/hightrex/microservices-platform/services/audit-service/internal/models"
)

// RetentionRepo implements service.RetentionRepository using PostgreSQL.
type RetentionRepo struct {
	db *pgxpool.Pool
}

// NewRetentionRepo creates a new RetentionRepo.
func NewRetentionRepo(db *pgxpool.Pool) *RetentionRepo {
	return &RetentionRepo{db: db}
}

// Create inserts a new retention policy.
func (r *RetentionRepo) Create(ctx context.Context, policy *models.RetentionPolicy) error {
	tenantID, err := tenant.RequireTenant(ctx)
	if err != nil {
		return err
	}

	policy.ID = uuid.New()
	policy.TenantID = tenantID
	policy.IsActive = true
	policy.CreatedAt = time.Now()
	policy.UpdatedAt = time.Now()

	_, err = r.db.Exec(ctx,
		`INSERT INTO retention_policies (id, tenant_id, event_type, retention_days, is_active, created_at, updated_at)
		 VALUES ($1, $2, $3, $4, $5, $6, $7)`,
		policy.ID, policy.TenantID, policy.EventType, policy.RetentionDays,
		policy.IsActive, policy.CreatedAt, policy.UpdatedAt,
	)
	if err != nil {
		if strings.Contains(err.Error(), "idx_retention_policies_tenant_event_type") {
			return errors.BadRequest("Retention policy for this event type already exists", err)
		}
		return errors.InternalServerError("Failed to create retention policy", err)
	}
	return nil
}

// GetByEventType retrieves a retention policy by event type, scoped to tenant.
func (r *RetentionRepo) GetByEventType(ctx context.Context, eventType string) (*models.RetentionPolicy, error) {
	tenantID, err := tenant.RequireTenant(ctx)
	if err != nil {
		return nil, err
	}

	var policy models.RetentionPolicy
	err = r.db.QueryRow(ctx,
		`SELECT id, tenant_id, event_type, retention_days, is_active, created_at, updated_at
		 FROM retention_policies WHERE tenant_id = $1 AND event_type = $2`,
		tenantID, eventType,
	).Scan(
		&policy.ID, &policy.TenantID, &policy.EventType, &policy.RetentionDays,
		&policy.IsActive, &policy.CreatedAt, &policy.UpdatedAt,
	)
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, errors.NotFound("Retention policy not found", nil)
		}
		return nil, errors.InternalServerError("Failed to get retention policy", err)
	}
	return &policy, nil
}

// List retrieves all retention policies for the tenant.
func (r *RetentionRepo) List(ctx context.Context) ([]models.RetentionPolicy, error) {
	tenantID, err := tenant.RequireTenant(ctx)
	if err != nil {
		return nil, err
	}

	rows, err := r.db.Query(ctx,
		`SELECT id, tenant_id, event_type, retention_days, is_active, created_at, updated_at
		 FROM retention_policies WHERE tenant_id = $1 ORDER BY event_type`,
		tenantID,
	)
	if err != nil {
		return nil, errors.InternalServerError("Failed to list retention policies", err)
	}
	defer rows.Close()

	var policies []models.RetentionPolicy
	for rows.Next() {
		var p models.RetentionPolicy
		if err := rows.Scan(
			&p.ID, &p.TenantID, &p.EventType, &p.RetentionDays,
			&p.IsActive, &p.CreatedAt, &p.UpdatedAt,
		); err != nil {
			return nil, errors.InternalServerError("Failed to scan retention policy", err)
		}
		policies = append(policies, p)
	}

	return policies, nil
}

// ListAllActive returns all active retention policies across all tenants.
// This is used by the background retention worker which runs outside any tenant context.
func (r *RetentionRepo) ListAllActive(ctx context.Context) ([]models.RetentionPolicy, error) {
	rows, err := r.db.Query(ctx,
		`SELECT id, tenant_id, event_type, retention_days, is_active, created_at, updated_at
		 FROM retention_policies WHERE is_active = true ORDER BY tenant_id, event_type`,
	)
	if err != nil {
		return nil, errors.InternalServerError("Failed to list all active retention policies", err)
	}
	defer rows.Close()

	var policies []models.RetentionPolicy
	for rows.Next() {
		var p models.RetentionPolicy
		if err := rows.Scan(
			&p.ID, &p.TenantID, &p.EventType, &p.RetentionDays,
			&p.IsActive, &p.CreatedAt, &p.UpdatedAt,
		); err != nil {
			return nil, errors.InternalServerError("Failed to scan retention policy", err)
		}
		policies = append(policies, p)
	}

	return policies, nil
}

// Update performs a partial update on a retention policy.
func (r *RetentionRepo) Update(ctx context.Context, id uuid.UUID, fields map[string]interface{}) error {
	tenantID, err := tenant.RequireTenant(ctx)
	if err != nil {
		return err
	}

	if len(fields) == 0 {
		return nil
	}

	var setClauses []string
	var args []interface{}
	argIdx := 1

	for key, value := range fields {
		setClauses = append(setClauses, fmt.Sprintf("%s = $%d", key, argIdx))
		args = append(args, value)
		argIdx++
	}

	setClauses = append(setClauses, fmt.Sprintf("updated_at = $%d", argIdx))
	args = append(args, time.Now())
	argIdx++

	args = append(args, id, tenantID)

	query := fmt.Sprintf(
		"UPDATE retention_policies SET %s WHERE id = $%d AND tenant_id = $%d",
		strings.Join(setClauses, ", "), argIdx, argIdx+1,
	)

	tag, err := r.db.Exec(ctx, query, args...)
	if err != nil {
		return errors.InternalServerError("Failed to update retention policy", err)
	}
	if tag.RowsAffected() == 0 {
		return errors.NotFound("Retention policy not found", nil)
	}
	return nil
}

// Delete removes a retention policy.
func (r *RetentionRepo) Delete(ctx context.Context, id uuid.UUID) error {
	tenantID, err := tenant.RequireTenant(ctx)
	if err != nil {
		return err
	}

	tag, err := r.db.Exec(ctx,
		"DELETE FROM retention_policies WHERE id = $1 AND tenant_id = $2",
		id, tenantID,
	)
	if err != nil {
		return errors.InternalServerError("Failed to delete retention policy", err)
	}
	if tag.RowsAffected() == 0 {
		return errors.NotFound("Retention policy not found", nil)
	}
	return nil
}
