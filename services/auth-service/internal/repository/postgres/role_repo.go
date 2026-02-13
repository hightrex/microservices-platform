package postgres

import (
	"context"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/hightrex/microservices-platform/libs/go/pkg/errors"
	"github.com/hightrex/microservices-platform/libs/go/pkg/tenant"
	"github.com/hightrex/microservices-platform/services/auth-service/internal/models"
)

// RoleRepo implements service.RoleRepository using PostgreSQL.
type RoleRepo struct {
	db *pgxpool.Pool
}

// NewRoleRepo creates a new RoleRepo.
func NewRoleRepo(db *pgxpool.Pool) *RoleRepo {
	return &RoleRepo{db: db}
}

// GetByName retrieves a role by name, scoped to the tenant.
func (r *RoleRepo) GetByName(ctx context.Context, name models.RoleName) (*models.Role, error) {
	tenantID, err := tenant.RequireTenant(ctx)
	if err != nil {
		return nil, err
	}

	var role models.Role
	err = r.db.QueryRow(ctx,
		`SELECT id, tenant_id, name, description, is_system, created_at, updated_at
		 FROM roles WHERE name = $1 AND tenant_id = $2`,
		name, tenantID,
	).Scan(
		&role.ID, &role.TenantID, &role.Name, &role.Description,
		&role.IsSystem, &role.CreatedAt, &role.UpdatedAt,
	)
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, errors.NotFound("Role not found", nil)
		}
		return nil, errors.InternalServerError("Failed to get role", err)
	}
	return &role, nil
}

// ListRoles retrieves all roles for the tenant.
func (r *RoleRepo) ListRoles(ctx context.Context) ([]models.Role, error) {
	tenantID, err := tenant.RequireTenant(ctx)
	if err != nil {
		return nil, err
	}

	rows, err := r.db.Query(ctx,
		`SELECT id, tenant_id, name, description, is_system, created_at, updated_at
		 FROM roles WHERE tenant_id = $1 ORDER BY name`,
		tenantID,
	)
	if err != nil {
		return nil, errors.InternalServerError("Failed to list roles", err)
	}
	defer rows.Close()

	var roles []models.Role
	for rows.Next() {
		var role models.Role
		if err := rows.Scan(
			&role.ID, &role.TenantID, &role.Name, &role.Description,
			&role.IsSystem, &role.CreatedAt, &role.UpdatedAt,
		); err != nil {
			return nil, errors.InternalServerError("Failed to scan role", err)
		}
		roles = append(roles, role)
	}
	return roles, nil
}

// AssignRole assigns a role to a user within the tenant.
func (r *RoleRepo) AssignRole(ctx context.Context, userID, roleID uuid.UUID, grantedBy *uuid.UUID) error {
	tenantID, err := tenant.RequireTenant(ctx)
	if err != nil {
		return err
	}

	_, err = r.db.Exec(ctx,
		`INSERT INTO user_roles (user_id, role_id, tenant_id, granted_by, granted_at)
		 VALUES ($1, $2, $3, $4, $5)
		 ON CONFLICT (user_id, role_id, tenant_id) DO UPDATE SET granted_by = $4, granted_at = $5`,
		userID, roleID, tenantID, grantedBy, time.Now(),
	)
	if err != nil {
		return errors.InternalServerError("Failed to assign role", err)
	}
	return nil
}

// RevokeRole removes a role from a user within the tenant.
func (r *RoleRepo) RevokeRole(ctx context.Context, userID, roleID uuid.UUID) error {
	tenantID, err := tenant.RequireTenant(ctx)
	if err != nil {
		return err
	}

	tag, err := r.db.Exec(ctx,
		`DELETE FROM user_roles WHERE user_id = $1 AND role_id = $2 AND tenant_id = $3`,
		userID, roleID, tenantID,
	)
	if err != nil {
		return errors.InternalServerError("Failed to revoke role", err)
	}
	if tag.RowsAffected() == 0 {
		return errors.NotFound("Role assignment not found", nil)
	}
	return nil
}

// GetUserRoles retrieves all roles assigned to a user within the tenant.
func (r *RoleRepo) GetUserRoles(ctx context.Context, userID uuid.UUID) ([]models.UserRole, error) {
	tenantID, err := tenant.RequireTenant(ctx)
	if err != nil {
		return nil, err
	}

	rows, err := r.db.Query(ctx,
		`SELECT ur.user_id, ur.role_id, ur.tenant_id, ur.granted_by, ur.granted_at, ro.name
		 FROM user_roles ur
		 JOIN roles ro ON ro.id = ur.role_id
		 WHERE ur.user_id = $1 AND ur.tenant_id = $2`,
		userID, tenantID,
	)
	if err != nil {
		return nil, errors.InternalServerError("Failed to get user roles", err)
	}
	defer rows.Close()

	var userRoles []models.UserRole
	for rows.Next() {
		var ur models.UserRole
		if err := rows.Scan(
			&ur.UserID, &ur.RoleID, &ur.TenantID, &ur.GrantedBy, &ur.GrantedAt, &ur.RoleName,
		); err != nil {
			return nil, errors.InternalServerError("Failed to scan user role", err)
		}
		userRoles = append(userRoles, ur)
	}
	return userRoles, nil
}

// GetUserPermissions retrieves the flattened permission set for a user.
func (r *RoleRepo) GetUserPermissions(ctx context.Context, userID uuid.UUID) ([]models.Permission, error) {
	tenantID, err := tenant.RequireTenant(ctx)
	if err != nil {
		return nil, err
	}

	rows, err := r.db.Query(ctx,
		`SELECT DISTINCT p.id, p.resource, p.action, p.description
		 FROM permissions p
		 JOIN role_permissions rp ON rp.permission_id = p.id
		 JOIN user_roles ur ON ur.role_id = rp.role_id
		 WHERE ur.user_id = $1 AND ur.tenant_id = $2`,
		userID, tenantID,
	)
	if err != nil {
		return nil, errors.InternalServerError("Failed to get user permissions", err)
	}
	defer rows.Close()

	var perms []models.Permission
	for rows.Next() {
		var p models.Permission
		if err := rows.Scan(&p.ID, &p.Resource, &p.Action, &p.Description); err != nil {
			return nil, errors.InternalServerError("Failed to scan permission", err)
		}
		perms = append(perms, p)
	}
	return perms, nil
}
