package service

import (
	"context"

	"github.com/google/uuid"
	"github.com/hightrex/microservices-platform/services/auth-service/internal/models"
)

// UserRepository defines the interface for user data access.
type UserRepository interface {
	Create(ctx context.Context, user *models.User) error
	GetByID(ctx context.Context, id uuid.UUID) (*models.User, error)
	GetByEmail(ctx context.Context, email string) (*models.User, error)
	List(ctx context.Context, filter models.UserFilter, page models.Pagination) ([]models.User, int, error)
	Update(ctx context.Context, id uuid.UUID, fields map[string]interface{}) error
	Delete(ctx context.Context, id uuid.UUID) error
	IncrementFailedLogins(ctx context.Context, id uuid.UUID) error
	ResetFailedLogins(ctx context.Context, id uuid.UUID) error
	UpdateLastLogin(ctx context.Context, id uuid.UUID) error
}

// SessionRepository defines the interface for session data access.
type SessionRepository interface {
	Create(ctx context.Context, session *models.Session) error
	GetByRefreshToken(ctx context.Context, tokenHash string) (*models.Session, error)
	ListByUser(ctx context.Context, userID uuid.UUID) ([]models.Session, error)
	Delete(ctx context.Context, id uuid.UUID) error
	DeleteAllByUser(ctx context.Context, userID uuid.UUID) error
	CleanupExpired(ctx context.Context) (int64, error)
}

// APIKeyRepository defines the interface for API key data access.
type APIKeyRepository interface {
	Create(ctx context.Context, apiKey *models.APIKey) error
	GetByHash(ctx context.Context, keyHash string) (*models.APIKey, error)
	List(ctx context.Context, userID uuid.UUID) ([]models.APIKey, error)
	Revoke(ctx context.Context, id uuid.UUID) error
}

// RoleRepository defines the interface for role/permission data access.
type RoleRepository interface {
	GetByName(ctx context.Context, name models.RoleName) (*models.Role, error)
	ListRoles(ctx context.Context) ([]models.Role, error)
	AssignRole(ctx context.Context, userID, roleID uuid.UUID, grantedBy *uuid.UUID) error
	RevokeRole(ctx context.Context, userID, roleID uuid.UUID) error
	GetUserRoles(ctx context.Context, userID uuid.UUID) ([]models.UserRole, error)
	GetUserPermissions(ctx context.Context, userID uuid.UUID) ([]models.Permission, error)
}

// PasswordHistoryRepository defines the interface for password history.
type PasswordHistoryRepository interface {
	Add(ctx context.Context, userID uuid.UUID, passwordHash string) error
	GetRecent(ctx context.Context, userID uuid.UUID, count int) ([]string, error)
}

// TokenCache defines the interface for Redis-based token caching.
type TokenCache interface {
	BlacklistToken(ctx context.Context, tokenID string, ttl int64) error
	IsBlacklisted(ctx context.Context, tokenID string) (bool, error)
	CacheSession(ctx context.Context, sessionID string, session *models.Session) error
	GetCachedSession(ctx context.Context, sessionID string) (*models.Session, error)
	InvalidateSession(ctx context.Context, sessionID string) error
	StoreMFAToken(ctx context.Context, userID string, secret string) error
	GetMFAToken(ctx context.Context, userID string) (string, error)
	DeleteMFAToken(ctx context.Context, userID string) error
}

// EventPublisher defines the interface for publishing domain events.
type EventPublisher interface {
	Publish(ctx context.Context, stream string, eventType string, data interface{}) error
}
