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

// SessionRepo implements service.SessionRepository using PostgreSQL.
type SessionRepo struct {
	db *pgxpool.Pool
}

// NewSessionRepo creates a new SessionRepo.
func NewSessionRepo(db *pgxpool.Pool) *SessionRepo {
	return &SessionRepo{db: db}
}

// Create inserts a new session into the database.
func (r *SessionRepo) Create(ctx context.Context, session *models.Session) error {
	tenantID, err := tenant.RequireTenant(ctx)
	if err != nil {
		return err
	}

	session.TenantID = tenantID
	session.ID = uuid.New()
	session.CreatedAt = time.Now()

	_, err = r.db.Exec(ctx,
		`INSERT INTO sessions (id, user_id, tenant_id, refresh_token_hash, ip_address, user_agent, expires_at, created_at)
		 VALUES ($1, $2, $3, $4, $5, $6, $7, $8)`,
		session.ID, session.UserID, session.TenantID, session.RefreshTokenHash,
		session.IPAddress, session.UserAgent, session.ExpiresAt, session.CreatedAt,
	)
	if err != nil {
		return errors.InternalServerError("Failed to create session", err)
	}
	return nil
}

// GetByRefreshToken retrieves a session by its refresh token hash.
func (r *SessionRepo) GetByRefreshToken(ctx context.Context, tokenHash string) (*models.Session, error) {
	tenantID, err := tenant.RequireTenant(ctx)
	if err != nil {
		return nil, err
	}

	var session models.Session
	err = r.db.QueryRow(ctx,
		`SELECT id, user_id, tenant_id, refresh_token_hash, ip_address, user_agent, expires_at, created_at
		 FROM sessions WHERE refresh_token_hash = $1 AND tenant_id = $2 AND expires_at > $3`,
		tokenHash, tenantID, time.Now(),
	).Scan(
		&session.ID, &session.UserID, &session.TenantID, &session.RefreshTokenHash,
		&session.IPAddress, &session.UserAgent, &session.ExpiresAt, &session.CreatedAt,
	)
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, errors.NotFound("Session not found or expired", nil)
		}
		return nil, errors.InternalServerError("Failed to get session", err)
	}
	return &session, nil
}

// ListByUser retrieves all active sessions for a user.
func (r *SessionRepo) ListByUser(ctx context.Context, userID uuid.UUID) ([]models.Session, error) {
	tenantID, err := tenant.RequireTenant(ctx)
	if err != nil {
		return nil, err
	}

	rows, err := r.db.Query(ctx,
		`SELECT id, user_id, tenant_id, refresh_token_hash, ip_address, user_agent, expires_at, created_at
		 FROM sessions WHERE user_id = $1 AND tenant_id = $2 AND expires_at > $3 ORDER BY created_at DESC`,
		userID, tenantID, time.Now(),
	)
	if err != nil {
		return nil, errors.InternalServerError("Failed to list sessions", err)
	}
	defer rows.Close()

	var sessions []models.Session
	for rows.Next() {
		var s models.Session
		if err := rows.Scan(
			&s.ID, &s.UserID, &s.TenantID, &s.RefreshTokenHash,
			&s.IPAddress, &s.UserAgent, &s.ExpiresAt, &s.CreatedAt,
		); err != nil {
			return nil, errors.InternalServerError("Failed to scan session", err)
		}
		sessions = append(sessions, s)
	}
	return sessions, nil
}

// Delete removes a specific session.
func (r *SessionRepo) Delete(ctx context.Context, id uuid.UUID) error {
	tenantID, err := tenant.RequireTenant(ctx)
	if err != nil {
		return err
	}

	tag, err := r.db.Exec(ctx,
		`DELETE FROM sessions WHERE id = $1 AND tenant_id = $2`,
		id, tenantID,
	)
	if err != nil {
		return errors.InternalServerError("Failed to delete session", err)
	}
	if tag.RowsAffected() == 0 {
		return errors.NotFound("Session not found", nil)
	}
	return nil
}

// DeleteAllByUser removes all sessions for a user (e.g., on password change).
func (r *SessionRepo) DeleteAllByUser(ctx context.Context, userID uuid.UUID) error {
	tenantID, err := tenant.RequireTenant(ctx)
	if err != nil {
		return err
	}

	_, err = r.db.Exec(ctx,
		`DELETE FROM sessions WHERE user_id = $1 AND tenant_id = $2`,
		userID, tenantID,
	)
	if err != nil {
		return errors.InternalServerError("Failed to delete all sessions", err)
	}
	return nil
}

// CleanupExpired removes expired sessions and returns the number deleted.
func (r *SessionRepo) CleanupExpired(ctx context.Context) (int64, error) {
	tag, err := r.db.Exec(ctx,
		`DELETE FROM sessions WHERE expires_at < $1`,
		time.Now(),
	)
	if err != nil {
		return 0, errors.InternalServerError("Failed to cleanup expired sessions", err)
	}
	return tag.RowsAffected(), nil
}
