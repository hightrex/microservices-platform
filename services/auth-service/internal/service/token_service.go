package service

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"

	"github.com/hightrex/microservices-platform/libs/go/pkg/errors"
	"github.com/hightrex/microservices-platform/services/auth-service/internal/config"
	"github.com/hightrex/microservices-platform/services/auth-service/internal/models"
)

// TokenClaims holds the JWT claims for an access token.
type TokenClaims struct {
	jwt.RegisteredClaims
	TenantID string   `json:"tid"`
	OrgID    string   `json:"org"`
	Roles    []string `json:"roles"`
}

// TokenService handles JWT and refresh token operations.
type TokenService struct {
	cfg config.JWTConfig
}

// NewTokenService creates a new TokenService.
func NewTokenService(cfg config.JWTConfig) *TokenService {
	return &TokenService{cfg: cfg}
}

// GenerateAccessToken creates a signed JWT access token with user claims.
func (s *TokenService) GenerateAccessToken(user *models.User, roles []string) (string, error) {
	now := time.Now()
	claims := TokenClaims{
		RegisteredClaims: jwt.RegisteredClaims{
			Subject:   user.ID.String(),
			Issuer:    s.cfg.Issuer,
			IssuedAt:  jwt.NewNumericDate(now),
			ExpiresAt: jwt.NewNumericDate(now.Add(s.cfg.AccessTokenTTL)),
			ID:        uuid.New().String(),
		},
		TenantID: user.TenantID.String(),
		OrgID:    user.TenantID.String(), // org_id == tenant_id
		Roles:    roles,
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	signed, err := token.SignedString([]byte(s.cfg.Secret))
	if err != nil {
		return "", errors.InternalServerError("Failed to generate access token", err)
	}
	return signed, nil
}

// GenerateRefreshToken creates a cryptographically random refresh token.
// Returns the raw token (for the client) and its SHA-256 hash (for storage).
func (s *TokenService) GenerateRefreshToken() (raw string, hash string, err error) {
	bytes := make([]byte, 32)
	if _, err := rand.Read(bytes); err != nil {
		return "", "", errors.InternalServerError("Failed to generate refresh token", err)
	}
	raw = hex.EncodeToString(bytes)
	hash = HashToken(raw)
	return raw, hash, nil
}

// ValidateAccessToken parses and validates a JWT access token.
func (s *TokenService) ValidateAccessToken(tokenString string) (*TokenClaims, error) {
	token, err := jwt.ParseWithClaims(tokenString, &TokenClaims{}, func(token *jwt.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
		}
		return []byte(s.cfg.Secret), nil
	})
	if err != nil {
		return nil, errors.Unauthorized("Invalid or expired token", err)
	}

	claims, ok := token.Claims.(*TokenClaims)
	if !ok || !token.Valid {
		return nil, errors.Unauthorized("Invalid token claims", nil)
	}

	return claims, nil
}

// AccessTokenTTLSeconds returns the access token TTL in seconds.
func (s *TokenService) AccessTokenTTLSeconds() int64 {
	return int64(s.cfg.AccessTokenTTL.Seconds())
}

// RefreshTokenTTL returns the refresh token TTL as a duration.
func (s *TokenService) RefreshTokenTTL() time.Duration {
	return s.cfg.RefreshTokenTTL
}

// HashToken returns the SHA-256 hash of a token string.
func HashToken(token string) string {
	h := sha256.Sum256([]byte(token))
	return hex.EncodeToString(h[:])
}
