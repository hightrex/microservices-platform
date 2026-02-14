package postgres

import (
	"context"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/google/uuid"
	"github.com/hightrex/microservices-platform/libs/go/pkg/errors"
	"github.com/hightrex/microservices-platform/libs/go/pkg/tenant"
	"github.com/hightrex/microservices-platform/services/organization-service/internal/models"
)

// ModuleRepo implements service.ModuleRepository using PostgreSQL.
type ModuleRepo struct {
	db *pgxpool.Pool
}

// NewModuleRepo creates a new ModuleRepo.
func NewModuleRepo(db *pgxpool.Pool) *ModuleRepo {
	return &ModuleRepo{db: db}
}

// GetByOrg retrieves all modules for an organization.
func (r *ModuleRepo) GetByOrg(ctx context.Context, orgID uuid.UUID) ([]models.Module, error) {
	tenantID, err := tenant.RequireTenant(ctx)
	if err != nil {
		return nil, err
	}
	if tenantID != orgID {
		return nil, errors.Forbidden("Access denied to this organization's modules", nil)
	}

	rows, err := r.db.Query(ctx,
		`SELECT id, org_id, module_name, enabled, config, enabled_at, disabled_at, created_at, updated_at
		 FROM org_modules WHERE org_id = $1 ORDER BY module_name`,
		orgID,
	)
	if err != nil {
		return nil, errors.InternalServerError("Failed to get modules", err)
	}
	defer rows.Close()

	var modules []models.Module
	for rows.Next() {
		var m models.Module
		if err := rows.Scan(
			&m.ID, &m.OrgID, &m.ModuleName, &m.Enabled, &m.Config,
			&m.EnabledAt, &m.DisabledAt, &m.CreatedAt, &m.UpdatedAt,
		); err != nil {
			return nil, errors.InternalServerError("Failed to scan module", err)
		}
		modules = append(modules, m)
	}

	return modules, nil
}

// Toggle enables or disables a module for an organization.
func (r *ModuleRepo) Toggle(ctx context.Context, orgID uuid.UUID, moduleName string, enabled bool) error {
	tenantID, err := tenant.RequireTenant(ctx)
	if err != nil {
		return err
	}
	if tenantID != orgID {
		return errors.Forbidden("Access denied to this organization's modules", nil)
	}

	now := time.Now()
	var query string
	if enabled {
		query = `UPDATE org_modules SET enabled = true, enabled_at = $1, disabled_at = NULL, updated_at = $1 WHERE org_id = $2 AND module_name = $3`
	} else {
		query = `UPDATE org_modules SET enabled = false, disabled_at = $1, updated_at = $1 WHERE org_id = $2 AND module_name = $3`
	}

	tag, err := r.db.Exec(ctx, query, now, orgID, moduleName)
	if err != nil {
		return errors.InternalServerError("Failed to toggle module", err)
	}
	if tag.RowsAffected() == 0 {
		return errors.NotFound("Module not found", nil)
	}
	return nil
}

// GetConfig retrieves per-module config for an organization.
func (r *ModuleRepo) GetConfig(ctx context.Context, orgID uuid.UUID, moduleName string) (*models.Module, error) {
	tenantID, err := tenant.RequireTenant(ctx)
	if err != nil {
		return nil, err
	}
	if tenantID != orgID {
		return nil, errors.Forbidden("Access denied to this organization's modules", nil)
	}

	var m models.Module
	err = r.db.QueryRow(ctx,
		`SELECT id, org_id, module_name, enabled, config, enabled_at, disabled_at, created_at, updated_at
		 FROM org_modules WHERE org_id = $1 AND module_name = $2`,
		orgID, moduleName,
	).Scan(
		&m.ID, &m.OrgID, &m.ModuleName, &m.Enabled, &m.Config,
		&m.EnabledAt, &m.DisabledAt, &m.CreatedAt, &m.UpdatedAt,
	)
	if err != nil {
		return nil, errors.NotFound("Module not found", nil)
	}
	return &m, nil
}

// UpdateConfig updates per-module config for an organization.
func (r *ModuleRepo) UpdateConfig(ctx context.Context, orgID uuid.UUID, moduleName string, config []byte) error {
	tenantID, err := tenant.RequireTenant(ctx)
	if err != nil {
		return err
	}
	if tenantID != orgID {
		return errors.Forbidden("Access denied to this organization's modules", nil)
	}

	tag, err := r.db.Exec(ctx,
		`UPDATE org_modules SET config = $1, updated_at = $2 WHERE org_id = $3 AND module_name = $4`,
		config, time.Now(), orgID, moduleName,
	)
	if err != nil {
		return errors.InternalServerError("Failed to update module config", err)
	}
	if tag.RowsAffected() == 0 {
		return errors.NotFound("Module not found", nil)
	}
	return nil
}

// GetByOrgUnscoped retrieves all modules for an organization without tenant check.
// Used internally after org creation when setting up default modules.
func (r *ModuleRepo) GetByOrgUnscoped(ctx context.Context, orgID uuid.UUID) ([]models.Module, error) {
	rows, err := r.db.Query(ctx,
		`SELECT id, org_id, module_name, enabled, config, enabled_at, disabled_at, created_at, updated_at
		 FROM org_modules WHERE org_id = $1 ORDER BY module_name`,
		orgID,
	)
	if err != nil {
		return nil, errors.InternalServerError("Failed to get modules", err)
	}
	defer rows.Close()

	var modules []models.Module
	for rows.Next() {
		var m models.Module
		if err := rows.Scan(
			&m.ID, &m.OrgID, &m.ModuleName, &m.Enabled, &m.Config,
			&m.EnabledAt, &m.DisabledAt, &m.CreatedAt, &m.UpdatedAt,
		); err != nil {
			return nil, errors.InternalServerError("Failed to scan module", err)
		}
		modules = append(modules, m)
	}

	return modules, nil
}
