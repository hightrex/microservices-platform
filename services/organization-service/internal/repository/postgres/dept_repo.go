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

// DeptRepo implements service.DeptRepository using PostgreSQL.
type DeptRepo struct {
	db *pgxpool.Pool
}

// NewDeptRepo creates a new DeptRepo.
func NewDeptRepo(db *pgxpool.Pool) *DeptRepo {
	return &DeptRepo{db: db}
}

// Create inserts a new department.
func (r *DeptRepo) Create(ctx context.Context, orgID uuid.UUID, dept *models.Department) error {
	tenantID, err := tenant.RequireTenant(ctx)
	if err != nil {
		return err
	}
	if tenantID != orgID {
		return errors.Forbidden("Access denied to this organization", nil)
	}

	dept.ID = uuid.New()
	dept.OrgID = orgID
	dept.CreatedAt = time.Now()
	dept.UpdatedAt = time.Now()

	_, err = r.db.Exec(ctx,
		`INSERT INTO departments (id, org_id, name, description, parent_id, created_at, updated_at)
		 VALUES ($1, $2, $3, $4, $5, $6, $7)`,
		dept.ID, dept.OrgID, dept.Name, dept.Description, dept.ParentID,
		dept.CreatedAt, dept.UpdatedAt,
	)
	if err != nil {
		if strings.Contains(err.Error(), "idx_departments_org_name") {
			return errors.BadRequest("Department name already exists in this organization", err)
		}
		return errors.InternalServerError("Failed to create department", err)
	}
	return nil
}

// List retrieves all departments for an organization with hierarchy.
func (r *DeptRepo) List(ctx context.Context, orgID uuid.UUID) ([]models.Department, error) {
	tenantID, err := tenant.RequireTenant(ctx)
	if err != nil {
		return nil, err
	}
	if tenantID != orgID {
		return nil, errors.Forbidden("Access denied to this organization", nil)
	}

	rows, err := r.db.Query(ctx,
		`SELECT id, org_id, name, description, parent_id, created_at, updated_at
		 FROM departments WHERE org_id = $1 ORDER BY name`,
		orgID,
	)
	if err != nil {
		return nil, errors.InternalServerError("Failed to list departments", err)
	}
	defer rows.Close()

	var depts []models.Department
	for rows.Next() {
		var d models.Department
		if err := rows.Scan(
			&d.ID, &d.OrgID, &d.Name, &d.Description, &d.ParentID,
			&d.CreatedAt, &d.UpdatedAt,
		); err != nil {
			return nil, errors.InternalServerError("Failed to scan department", err)
		}
		depts = append(depts, d)
	}

	return depts, nil
}

// Update performs a partial update on a department.
func (r *DeptRepo) Update(ctx context.Context, orgID, deptID uuid.UUID, fields map[string]interface{}) error {
	tenantID, err := tenant.RequireTenant(ctx)
	if err != nil {
		return err
	}
	if tenantID != orgID {
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

	args = append(args, deptID, orgID)

	query := fmt.Sprintf(
		"UPDATE departments SET %s WHERE id = $%d AND org_id = $%d",
		strings.Join(setClauses, ", "), argIdx, argIdx+1,
	)

	tag, err := r.db.Exec(ctx, query, args...)
	if err != nil {
		if strings.Contains(err.Error(), "idx_departments_org_name") {
			return errors.BadRequest("Department name already exists in this organization", err)
		}
		return errors.InternalServerError("Failed to update department", err)
	}
	if tag.RowsAffected() == 0 {
		return errors.NotFound("Department not found", nil)
	}
	return nil
}

// Delete removes a department.
func (r *DeptRepo) Delete(ctx context.Context, orgID, deptID uuid.UUID) error {
	tenantID, err := tenant.RequireTenant(ctx)
	if err != nil {
		return err
	}
	if tenantID != orgID {
		return errors.Forbidden("Access denied to this organization", nil)
	}

	tag, err := r.db.Exec(ctx,
		`DELETE FROM departments WHERE id = $1 AND org_id = $2`,
		deptID, orgID,
	)
	if err != nil {
		return errors.InternalServerError("Failed to delete department", err)
	}
	if tag.RowsAffected() == 0 {
		return errors.NotFound("Department not found", nil)
	}
	return nil
}

// GetByID retrieves a department by ID, scoped to the org.
func (r *DeptRepo) GetByID(ctx context.Context, orgID, deptID uuid.UUID) (*models.Department, error) {
	tenantID, err := tenant.RequireTenant(ctx)
	if err != nil {
		return nil, err
	}
	if tenantID != orgID {
		return nil, errors.Forbidden("Access denied to this organization", nil)
	}

	var d models.Department
	err = r.db.QueryRow(ctx,
		`SELECT id, org_id, name, description, parent_id, created_at, updated_at
		 FROM departments WHERE id = $1 AND org_id = $2`,
		deptID, orgID,
	).Scan(
		&d.ID, &d.OrgID, &d.Name, &d.Description, &d.ParentID,
		&d.CreatedAt, &d.UpdatedAt,
	)
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, errors.NotFound("Department not found", nil)
		}
		return nil, errors.InternalServerError("Failed to get department", err)
	}
	return &d, nil
}
