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

// APIKeyRepo implements service.APIKeyRepository using PostgreSQL.
type APIKeyRepo struct {
	db *pgxpool.Pool
}

// NewAPIKeyRepo creates a new APIKeyRepo.
func NewAPIKeyRepo(db *pgxpool.Pool) *APIKeyRepo {
	return &APIKeyRepo{db: db}
}

// Create inserts a new API key into the database.
func (r *APIKeyRepo) Create(ctx context.Context, apiKey *models.APIKey) error {
	tenantID, err := tenant.RequireTenant(ctx)
	if err != nil {
		return err
	}

	apiKey.TenantID = tenantID
	apiKey.ID = uuid.New()
	apiKey.CreatedAt = time.Now()
	apiKey.UpdatedAt = time.Now()

	_, err = r.db.Exec(ctx,
		`INSERT INTO api_keys (id, tenant_id, user_id, name, key_hash, key_prefix, scopes, last_used_at, expires_at, revoked_at, created_at, updated_at)
		 VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12)`,
		apiKey.ID, apiKey.TenantID, apiKey.UserID, apiKey.Name,
		apiKey.KeyHash, apiKey.KeyPrefix, apiKey.Scopes,
		apiKey.LastUsedAt, apiKey.ExpiresAt, apiKey.RevokedAt,
		apiKey.CreatedAt, apiKey.UpdatedAt,
	)
	if err != nil {
		return errors.InternalServerError("Failed to create API key", err)
	}
	return nil
}

// GetByHash retrieves an API key by its hashed value.
func (r *APIKeyRepo) GetByHash(ctx context.Context, keyHash string) (*models.APIKey, error) {
	var apiKey models.APIKey
	// nosemgrep: missing-tenant-id-in-sql
	err := r.db.QueryRow(ctx,
		`SELECT id, tenant_id, user_id, name, key_hash, key_prefix, scopes, last_used_at, expires_at, revoked_at, created_at, updated_at
     FROM api_keys WHERE key_hash = $1 AND revoked_at IS NULL`,
		keyHash,
	).Scan(
		&apiKey.ID, &apiKey.TenantID, &apiKey.UserID, &apiKey.Name,
		&apiKey.KeyHash, &apiKey.KeyPrefix, &apiKey.Scopes,
		&apiKey.LastUsedAt, &apiKey.ExpiresAt, &apiKey.RevokedAt,
		&apiKey.CreatedAt, &apiKey.UpdatedAt,
	)
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, errors.NotFound("API key not found", nil)
		}
		return nil, errors.InternalServerError("Failed to get API key", err)
	}

	// Check expiry
	if apiKey.ExpiresAt != nil && apiKey.ExpiresAt.Before(time.Now()) {
		return nil, errors.Unauthorized("API key expired", nil)
	}

	return &apiKey, nil
}

// List retrieves all API keys for a user, scoped to the tenant.
func (r *APIKeyRepo) List(ctx context.Context, userID uuid.UUID) ([]models.APIKey, error) {
	tenantID, err := tenant.RequireTenant(ctx)
	if err != nil {
		return nil, err
	}

	rows, err := r.db.Query(ctx,
		`SELECT id, tenant_id, user_id, name, key_hash, key_prefix, scopes, last_used_at, expires_at, revoked_at, created_at, updated_at
		 FROM api_keys WHERE user_id = $1 AND tenant_id = $2 ORDER BY created_at DESC`,
		userID, tenantID,
	)
	if err != nil {
		return nil, errors.InternalServerError("Failed to list API keys", err)
	}
	defer rows.Close()

	var keys []models.APIKey
	for rows.Next() {
		var k models.APIKey
		if err := rows.Scan(
			&k.ID, &k.TenantID, &k.UserID, &k.Name,
			&k.KeyHash, &k.KeyPrefix, &k.Scopes,
			&k.LastUsedAt, &k.ExpiresAt, &k.RevokedAt,
			&k.CreatedAt, &k.UpdatedAt,
		); err != nil {
			return nil, errors.InternalServerError("Failed to scan API key", err)
		}
		keys = append(keys, k)
	}
	return keys, nil
}

// Revoke marks an API key as revoked.
func (r *APIKeyRepo) Revoke(ctx context.Context, id uuid.UUID) error {
	tenantID, err := tenant.RequireTenant(ctx)
	if err != nil {
		return err
	}

	now := time.Now()
	tag, err := r.db.Exec(ctx,
		`UPDATE api_keys SET revoked_at = $1, updated_at = $2 WHERE id = $3 AND tenant_id = $4 AND revoked_at IS NULL`,
		now, now, id, tenantID,
	)
	if err != nil {
		return errors.InternalServerError("Failed to revoke API key", err)
	}
	if tag.RowsAffected() == 0 {
		return errors.NotFound("API key not found or already revoked", nil)
	}
	return nil
}
