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
	"github.com/hightrex/microservices-platform/services/auth-service/internal/models"
)

// UserRepo implements service.UserRepository using PostgreSQL.
type UserRepo struct {
	db *pgxpool.Pool
}

// NewUserRepo creates a new UserRepo.
func NewUserRepo(db *pgxpool.Pool) *UserRepo {
	return &UserRepo{db: db}
}

// Create inserts a new user into the database.
func (r *UserRepo) Create(ctx context.Context, user *models.User) error {
	tenantID, err := tenant.RequireTenant(ctx)
	if err != nil {
		return err
	}

	user.TenantID = tenantID
	user.ID = uuid.New()
	user.CreatedAt = time.Now()
	user.UpdatedAt = time.Now()
	if user.Status == "" {
		user.Status = models.UserStatusActive
	}

	_, err = r.db.Exec(ctx,
		`INSERT INTO users (id, tenant_id, email, password_hash, first_name, last_name, status, mfa_enabled, mfa_secret, failed_login_count, locked_until, last_login_at, created_at, updated_at)
		 VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14)`,
		user.ID, user.TenantID, user.Email, user.PasswordHash,
		user.FirstName, user.LastName, user.Status, user.MFAEnabled,
		user.MFASecret, user.FailedLoginCount, user.LockedUntil,
		user.LastLoginAt, user.CreatedAt, user.UpdatedAt,
	)
	if err != nil {
		if strings.Contains(err.Error(), "uniq_users_tenant_id_email") {
			return errors.BadRequest("Email already registered", err)
		}
		return errors.InternalServerError("Failed to create user", err)
	}
	return nil
}

// GetByID retrieves a user by ID, scoped to the tenant.
func (r *UserRepo) GetByID(ctx context.Context, id uuid.UUID) (*models.User, error) {
	tenantID, err := tenant.RequireTenant(ctx)
	if err != nil {
		return nil, err
	}

	var user models.User
	err = r.db.QueryRow(ctx,
		`SELECT id, tenant_id, email, password_hash, first_name, last_name, status, mfa_enabled, mfa_secret, failed_login_count, locked_until, last_login_at, created_at, updated_at
		 FROM users WHERE id = $1 AND tenant_id = $2`,
		id, tenantID,
	).Scan(
		&user.ID, &user.TenantID, &user.Email, &user.PasswordHash,
		&user.FirstName, &user.LastName, &user.Status, &user.MFAEnabled,
		&user.MFASecret, &user.FailedLoginCount, &user.LockedUntil,
		&user.LastLoginAt, &user.CreatedAt, &user.UpdatedAt,
	)
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, errors.NotFound("User not found", nil)
		}
		return nil, errors.InternalServerError("Failed to get user", err)
	}
	return &user, nil
}

// GetByEmail retrieves a user by email, scoped to the tenant.
func (r *UserRepo) GetByEmail(ctx context.Context, email string) (*models.User, error) {
	tenantID, err := tenant.RequireTenant(ctx)
	if err != nil {
		return nil, err
	}

	var user models.User
	err = r.db.QueryRow(ctx,
		`SELECT id, tenant_id, email, password_hash, first_name, last_name, status, mfa_enabled, mfa_secret, failed_login_count, locked_until, last_login_at, created_at, updated_at
		 FROM users WHERE email = $1 AND tenant_id = $2`,
		strings.ToLower(email), tenantID,
	).Scan(
		&user.ID, &user.TenantID, &user.Email, &user.PasswordHash,
		&user.FirstName, &user.LastName, &user.Status, &user.MFAEnabled,
		&user.MFASecret, &user.FailedLoginCount, &user.LockedUntil,
		&user.LastLoginAt, &user.CreatedAt, &user.UpdatedAt,
	)
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, errors.NotFound("User not found", nil)
		}
		return nil, errors.InternalServerError("Failed to get user", err)
	}
	return &user, nil
}

// List retrieves users with filtering and pagination, scoped to the tenant.
func (r *UserRepo) List(ctx context.Context, filter models.UserFilter, page models.Pagination) ([]models.User, int, error) {
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

	if filter.Status != nil {
		conditions = append(conditions, fmt.Sprintf("status = $%d", argIdx))
		args = append(args, *filter.Status)
		argIdx++
	}

	if filter.Search != nil && *filter.Search != "" {
		conditions = append(conditions, fmt.Sprintf("(email ILIKE $%d OR first_name ILIKE $%d OR last_name ILIKE $%d)", argIdx, argIdx, argIdx))
		args = append(args, "%"+*filter.Search+"%")
		argIdx++
	}

	where := strings.Join(conditions, " AND ")

	// Count total
	var total int
	countQuery := fmt.Sprintf("SELECT COUNT(*) FROM users WHERE %s", where)
	if err := r.db.QueryRow(ctx, countQuery, args...).Scan(&total); err != nil {
		return nil, 0, errors.InternalServerError("Failed to count users", err)
	}

	// Fetch page
	offset := (page.Page - 1) * page.PageSize
	listQuery := fmt.Sprintf(
		`SELECT id, tenant_id, email, password_hash, first_name, last_name, status, mfa_enabled, mfa_secret, failed_login_count, locked_until, last_login_at, created_at, updated_at
		 FROM users WHERE %s ORDER BY created_at DESC LIMIT $%d OFFSET $%d`,
		where, argIdx, argIdx+1,
	)
	args = append(args, page.PageSize, offset)

	rows, err := r.db.Query(ctx, listQuery, args...)
	if err != nil {
		return nil, 0, errors.InternalServerError("Failed to list users", err)
	}
	defer rows.Close()

	var users []models.User
	for rows.Next() {
		var u models.User
		if err := rows.Scan(
			&u.ID, &u.TenantID, &u.Email, &u.PasswordHash,
			&u.FirstName, &u.LastName, &u.Status, &u.MFAEnabled,
			&u.MFASecret, &u.FailedLoginCount, &u.LockedUntil,
			&u.LastLoginAt, &u.CreatedAt, &u.UpdatedAt,
		); err != nil {
			return nil, 0, errors.InternalServerError("Failed to scan user", err)
		}
		users = append(users, u)
	}

	return users, total, nil
}

// Update performs a partial update on a user, scoped to the tenant.
func (r *UserRepo) Update(ctx context.Context, id uuid.UUID, fields map[string]interface{}) error {
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
		"UPDATE users SET %s WHERE id = $%d AND tenant_id = $%d",
		strings.Join(setClauses, ", "), argIdx, argIdx+1,
	)

	tag, err := r.db.Exec(ctx, query, args...)
	if err != nil {
		return errors.InternalServerError("Failed to update user", err)
	}
	if tag.RowsAffected() == 0 {
		return errors.NotFound("User not found", nil)
	}
	return nil
}

// Delete performs a soft delete by setting user status to disabled.
func (r *UserRepo) Delete(ctx context.Context, id uuid.UUID) error {
	return r.Update(ctx, id, map[string]interface{}{"status": models.UserStatusDisabled})
}

// IncrementFailedLogins increments the failed login counter for a user.
func (r *UserRepo) IncrementFailedLogins(ctx context.Context, id uuid.UUID) error {
	tenantID, err := tenant.RequireTenant(ctx)
	if err != nil {
		return err
	}

	_, err = r.db.Exec(ctx,
		`UPDATE users SET failed_login_count = failed_login_count + 1, updated_at = $1 WHERE id = $2 AND tenant_id = $3`,
		time.Now(), id, tenantID,
	)
	if err != nil {
		return errors.InternalServerError("Failed to increment failed logins", err)
	}
	return nil
}

// ResetFailedLogins resets the failed login counter to zero.
func (r *UserRepo) ResetFailedLogins(ctx context.Context, id uuid.UUID) error {
	tenantID, err := tenant.RequireTenant(ctx)
	if err != nil {
		return err
	}

	_, err = r.db.Exec(ctx,
		`UPDATE users SET failed_login_count = 0, locked_until = NULL, updated_at = $1 WHERE id = $2 AND tenant_id = $3`,
		time.Now(), id, tenantID,
	)
	if err != nil {
		return errors.InternalServerError("Failed to reset failed logins", err)
	}
	return nil
}

// UpdateLastLogin sets the last_login_at timestamp to now.
func (r *UserRepo) UpdateLastLogin(ctx context.Context, id uuid.UUID) error {
	tenantID, err := tenant.RequireTenant(ctx)
	if err != nil {
		return err
	}

	now := time.Now()
	_, err = r.db.Exec(ctx,
		`UPDATE users SET last_login_at = $1, updated_at = $2 WHERE id = $3 AND tenant_id = $4`,
		now, now, id, tenantID,
	)
	if err != nil {
		return errors.InternalServerError("Failed to update last login", err)
	}
	return nil
}
