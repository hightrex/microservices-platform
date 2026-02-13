package redis

import (
	"context"
	"fmt"
	"time"

	"github.com/hightrex/microservices-platform/libs/go/pkg/cache"
	"github.com/hightrex/microservices-platform/services/auth-service/internal/models"
)

const (
	blacklistPrefix = "auth:blacklist:"
	sessionPrefix   = "auth:session:"
	mfaPrefix       = "auth:mfa:"

	sessionCacheTTL = 15 * time.Minute
	mfaTokenTTL     = 5 * time.Minute
)

// TokenCache implements service.TokenCache using Redis.
type TokenCache struct {
	client *cache.Client
}

// NewTokenCache creates a new TokenCache.
func NewTokenCache(client *cache.Client) *TokenCache {
	return &TokenCache{client: client}
}

// BlacklistToken adds a token to the blacklist with a TTL matching the token's remaining lifetime.
func (c *TokenCache) BlacklistToken(ctx context.Context, tokenID string, ttlSeconds int64) error {
	if ttlSeconds <= 0 {
		return nil // token already expired, no need to blacklist
	}
	return c.client.Set(ctx, blacklistPrefix+tokenID, true, time.Duration(ttlSeconds)*time.Second)
}

// IsBlacklisted checks if a token has been blacklisted.
func (c *TokenCache) IsBlacklisted(ctx context.Context, tokenID string) (bool, error) {
	var blacklisted bool
	err := c.client.Get(ctx, blacklistPrefix+tokenID, &blacklisted)
	if err != nil {
		// Key not found means not blacklisted
		return false, nil
	}
	return blacklisted, nil
}

// CacheSession caches a session for fast lookup.
func (c *TokenCache) CacheSession(ctx context.Context, sessionID string, session *models.Session) error {
	return c.client.Set(ctx, sessionPrefix+sessionID, session, sessionCacheTTL)
}

// GetCachedSession retrieves a cached session.
func (c *TokenCache) GetCachedSession(ctx context.Context, sessionID string) (*models.Session, error) {
	var session models.Session
	err := c.client.Get(ctx, sessionPrefix+sessionID, &session)
	if err != nil {
		return nil, fmt.Errorf("session not in cache: %w", err)
	}
	return &session, nil
}

// InvalidateSession removes a session from the cache.
func (c *TokenCache) InvalidateSession(ctx context.Context, sessionID string) error {
	return c.client.Delete(ctx, sessionPrefix+sessionID)
}

// StoreMFAToken stores a temporary MFA secret for verification flow (5-minute TTL).
func (c *TokenCache) StoreMFAToken(ctx context.Context, userID string, secret string) error {
	return c.client.Set(ctx, mfaPrefix+userID, secret, mfaTokenTTL)
}

// GetMFAToken retrieves a stored MFA secret.
func (c *TokenCache) GetMFAToken(ctx context.Context, userID string) (string, error) {
	var secret string
	err := c.client.Get(ctx, mfaPrefix+userID, &secret)
	if err != nil {
		return "", fmt.Errorf("MFA token not found or expired: %w", err)
	}
	return secret, nil
}

// DeleteMFAToken removes a stored MFA secret.
func (c *TokenCache) DeleteMFAToken(ctx context.Context, userID string) error {
	return c.client.Delete(ctx, mfaPrefix+userID)
}
