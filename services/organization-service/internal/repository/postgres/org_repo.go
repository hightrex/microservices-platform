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
	"github.com/hightrex/microservices-platform/services/organization-service/internal/models"
)

// OrgRepo implements service.OrgRepository using PostgreSQL.
type OrgRepo struct {
	db *pgxpool.Pool
}

// NewOrgRepo creates a new OrgRepo.
func NewOrgRepo(db *pgxpool.Pool) *OrgRepo {
	return &OrgRepo{db: db}
}

// Create inserts a new organization and auto-creates default modules.
func (r *OrgRepo) Create(ctx context.Context, org *models.Organization) error {
	org.ID = uuid.New()
	org.CreatedAt = time.Now()
	org.UpdatedAt = time.Now()
	if org.Status == "" {
		org.Status = models.OrgStatusActive
	}
	if org.Plan == "" {
		org.Plan = models.PlanFree
	}
	if org.Settings == nil {
		org.Settings = []byte("{}")
	}

	_, err := r.db.Exec(ctx,
		`INSERT INTO organizations (id, name, slug, owner_user_id, plan, status, settings, max_users, max_storage_bytes, created_at, updated_at)
		 VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11)`,
		org.ID, org.Name, org.Slug, org.OwnerUserID, org.Plan, org.Status,
		org.Settings, org.MaxUsers, org.MaxStorageBytes, org.CreatedAt, org.UpdatedAt,
	)
	if err != nil {
		if strings.Contains(err.Error(), "idx_organizations_slug") {
			return errors.BadRequest("Organization slug already taken", err)
		}
		return errors.InternalServerError("Failed to create organization", err)
	}

	// Auto-create module records for all available modules
	for _, moduleName := range models.AllModuleNames() {
		_, err := r.db.Exec(ctx,
			`INSERT INTO org_modules (id, org_id, module_name, enabled, config, created_at, updated_at)
			 VALUES ($1, $2, $3, $4, $5, $6, $7)`,
			uuid.New(), org.ID, string(moduleName), false, "{}", org.CreatedAt, org.UpdatedAt,
		)
		if err != nil {
			return errors.InternalServerError("Failed to create default modules", err)
		}
	}

	return nil
}

// GetByID retrieves an organization by ID.
// For the org service, org_id IS the tenant_id.
func (r *OrgRepo) GetByID(ctx context.Context, id uuid.UUID) (*models.Organization, error) {
	// Verify caller has access to this org via tenant context
	tenantID, err := tenant.RequireTenant(ctx)
	if err != nil {
		return nil, err
	}
	if tenantID != id {
		return nil, errors.Forbidden("Access denied to this organization", nil)
	}

	var org models.Organization
	err = r.db.QueryRow(ctx,
		`SELECT id, name, slug, owner_user_id, plan, status, settings, max_users, max_storage_bytes, created_at, updated_at
		 FROM organizations WHERE id = $1 AND status != 'deleted'`,
		id,
	).Scan(
		&org.ID, &org.Name, &org.Slug, &org.OwnerUserID, &org.Plan, &org.Status,
		&org.Settings, &org.MaxUsers, &org.MaxStorageBytes, &org.CreatedAt, &org.UpdatedAt,
	)
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, errors.NotFound("Organization not found", nil)
		}
		return nil, errors.InternalServerError("Failed to get organization", err)
	}
	return &org, nil
}

// GetBySlug retrieves an organization by its URL-friendly slug.
func (r *OrgRepo) GetBySlug(ctx context.Context, slug string) (*models.Organization, error) {
	var org models.Organization
	err := r.db.QueryRow(ctx,
		`SELECT id, name, slug, owner_user_id, plan, status, settings, max_users, max_storage_bytes, created_at, updated_at
		 FROM organizations WHERE slug = $1 AND status != 'deleted'`,
		slug,
	).Scan(
		&org.ID, &org.Name, &org.Slug, &org.OwnerUserID, &org.Plan, &org.Status,
		&org.Settings, &org.MaxUsers, &org.MaxStorageBytes, &org.CreatedAt, &org.UpdatedAt,
	)
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, errors.NotFound("Organization not found", nil)
		}
		return nil, errors.InternalServerError("Failed to get organization by slug", err)
	}
	return &org, nil
}

// Update performs a partial update on an organization, scoped to the tenant.
func (r *OrgRepo) Update(ctx context.Context, id uuid.UUID, fields map[string]interface{}) error {
	tenantID, err := tenant.RequireTenant(ctx)
	if err != nil {
		return err
	}
	if tenantID != id {
		return errors.Forbidden("Access denied to this organization", nil)
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

	args = append(args, id)

	query := fmt.Sprintf(
		"UPDATE organizations SET %s WHERE id = $%d AND status != 'deleted'",
		strings.Join(setClauses, ", "), argIdx,
	)

	tag, err := r.db.Exec(ctx, query, args...)
	if err != nil {
		return errors.InternalServerError("Failed to update organization", err)
	}
	if tag.RowsAffected() == 0 {
		return errors.NotFound("Organization not found", nil)
	}
	return nil
}

// Delete performs a soft delete by setting org status to deleted.
func (r *OrgRepo) Delete(ctx context.Context, id uuid.UUID) error {
	return r.Update(ctx, id, map[string]interface{}{"status": models.OrgStatusDeleted})
}

// List retrieves organizations with filtering and pagination (admin-only).
func (r *OrgRepo) List(ctx context.Context, filter models.OrgFilter, page models.Pagination) ([]models.Organization, int, error) {
	var conditions []string
	var args []interface{}
	argIdx := 1

	conditions = append(conditions, "status != 'deleted'")

	if filter.Status != nil {
		conditions = append(conditions, fmt.Sprintf("status = $%d", argIdx))
		args = append(args, *filter.Status)
		argIdx++
	}

	if filter.Search != nil && *filter.Search != "" {
		conditions = append(conditions, fmt.Sprintf("(name ILIKE $%d OR slug ILIKE $%d)", argIdx, argIdx))
		args = append(args, "%"+*filter.Search+"%")
		argIdx++
	}

	where := strings.Join(conditions, " AND ")

	// Count total
	var total int
	countQuery := fmt.Sprintf("SELECT COUNT(*) FROM organizations WHERE %s", where)
	if err := r.db.QueryRow(ctx, countQuery, args...).Scan(&total); err != nil {
		return nil, 0, errors.InternalServerError("Failed to count organizations", err)
	}

	// Fetch page
	offset := (page.Page - 1) * page.PageSize
	listQuery := fmt.Sprintf(
		`SELECT id, name, slug, owner_user_id, plan, status, settings, max_users, max_storage_bytes, created_at, updated_at
		 FROM organizations WHERE %s ORDER BY created_at DESC LIMIT $%d OFFSET $%d`,
		where, argIdx, argIdx+1,
	)
	args = append(args, page.PageSize, offset)

	rows, err := r.db.Query(ctx, listQuery, args...)
	if err != nil {
		return nil, 0, errors.InternalServerError("Failed to list organizations", err)
	}
	defer rows.Close()

	var orgs []models.Organization
	for rows.Next() {
		var o models.Organization
		if err := rows.Scan(
			&o.ID, &o.Name, &o.Slug, &o.OwnerUserID, &o.Plan, &o.Status,
			&o.Settings, &o.MaxUsers, &o.MaxStorageBytes, &o.CreatedAt, &o.UpdatedAt,
		); err != nil {
			return nil, 0, errors.InternalServerError("Failed to scan organization", err)
		}
		orgs = append(orgs, o)
	}

	return orgs, total, nil
}

// GetByIDUnscoped retrieves an organization by ID without tenant check.
// Used internally during org creation when tenant context is the new org itself.
func (r *OrgRepo) GetByIDUnscoped(ctx context.Context, id uuid.UUID) (*models.Organization, error) {
	var org models.Organization
	err := r.db.QueryRow(ctx,
		`SELECT id, name, slug, owner_user_id, plan, status, settings, max_users, max_storage_bytes, created_at, updated_at
		 FROM organizations WHERE id = $1 AND status != 'deleted'`,
		id,
	).Scan(
		&org.ID, &org.Name, &org.Slug, &org.OwnerUserID, &org.Plan, &org.Status,
		&org.Settings, &org.MaxUsers, &org.MaxStorageBytes, &org.CreatedAt, &org.UpdatedAt,
	)
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, errors.NotFound("Organization not found", nil)
		}
		return nil, errors.InternalServerError("Failed to get organization", err)
	}
	return &org, nil
}

// CountByOwnerID counts organizations owned by a specific user.
// This is used to enforce the one-org-per-user rule during registration.
func (r *OrgRepo) CountByOwnerID(ctx context.Context, ownerID uuid.UUID) (int, error) {
	var count int
	err := r.db.QueryRow(ctx,
		"SELECT COUNT(*) FROM organizations WHERE owner_user_id = $1 AND status != 'deleted'",
		ownerID,
	).Scan(&count)
	if err != nil {
		return 0, errors.InternalServerError("Failed to count organizations by owner", err)
	}
	return count, nil
}
