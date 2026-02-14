package postgres

import (
	"context"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/hightrex/microservices-platform/libs/go/pkg/errors"
	"github.com/hightrex/microservices-platform/services/organization-service/internal/models"
)

// PlanRepo implements service.PlanRepository using PostgreSQL.
type PlanRepo struct {
	db *pgxpool.Pool
}

// NewPlanRepo creates a new PlanRepo.
func NewPlanRepo(db *pgxpool.Pool) *PlanRepo {
	return &PlanRepo{db: db}
}

// GetByName retrieves a plan by its name.
func (r *PlanRepo) GetByName(ctx context.Context, name string) (*models.Plan, error) {
	var p models.Plan
	err := r.db.QueryRow(ctx,
		`SELECT id, name, display_name, max_users, max_storage_bytes, max_api_calls_per_minute,
		        available_modules, price_monthly_cents, price_annual_cents, is_active, created_at, updated_at
		 FROM plans WHERE name = $1 AND is_active = true`,
		name,
	).Scan(
		&p.ID, &p.Name, &p.DisplayName, &p.MaxUsers, &p.MaxStorageBytes,
		&p.MaxAPICallsPerMinute, &p.AvailableModules, &p.PriceMonthly,
		&p.PriceAnnual, &p.IsActive, &p.CreatedAt, &p.UpdatedAt,
	)
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, errors.NotFound("Plan not found", nil)
		}
		return nil, errors.InternalServerError("Failed to get plan", err)
	}
	return &p, nil
}

// List retrieves all available plans.
func (r *PlanRepo) List(ctx context.Context) ([]models.Plan, error) {
	rows, err := r.db.Query(ctx,
		`SELECT id, name, display_name, max_users, max_storage_bytes, max_api_calls_per_minute,
		        available_modules, price_monthly_cents, price_annual_cents, is_active, created_at, updated_at
		 FROM plans WHERE is_active = true ORDER BY price_monthly_cents ASC`,
	)
	if err != nil {
		return nil, errors.InternalServerError("Failed to list plans", err)
	}
	defer rows.Close()

	var plans []models.Plan
	for rows.Next() {
		var p models.Plan
		if err := rows.Scan(
			&p.ID, &p.Name, &p.DisplayName, &p.MaxUsers, &p.MaxStorageBytes,
			&p.MaxAPICallsPerMinute, &p.AvailableModules, &p.PriceMonthly,
			&p.PriceAnnual, &p.IsActive, &p.CreatedAt, &p.UpdatedAt,
		); err != nil {
			return nil, errors.InternalServerError("Failed to scan plan", err)
		}
		plans = append(plans, p)
	}

	return plans, nil
}
