package service

import (
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/hightrex/microservices-platform/services/auth-service/internal/config"
	"github.com/hightrex/microservices-platform/services/auth-service/internal/models"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func newTestTokenService() *TokenService {
	return NewTokenService(config.JWTConfig{
		Secret:          "test-secret-minimum-32-characters-long!",
		AccessTokenTTL:  15 * time.Minute,
		RefreshTokenTTL: 168 * time.Hour,
		Issuer:          "test-auth-service",
	})
}

func TestGenerateAccessToken(t *testing.T) {
	svc := newTestTokenService()
	user := &models.User{
		ID:       uuid.New(),
		TenantID: uuid.New(),
		Email:    "test@example.com",
	}

	token, err := svc.GenerateAccessToken(user, []string{"member"})
	require.NoError(t, err)
	assert.NotEmpty(t, token)
}

func TestValidateAccessToken_Valid(t *testing.T) {
	svc := newTestTokenService()
	user := &models.User{
		ID:       uuid.New(),
		TenantID: uuid.New(),
		Email:    "test@example.com",
	}
	roles := []string{"member", "org_admin"}

	token, err := svc.GenerateAccessToken(user, roles)
	require.NoError(t, err)

	claims, err := svc.ValidateAccessToken(token)
	require.NoError(t, err)
	assert.Equal(t, user.ID.String(), claims.Subject)
	assert.Equal(t, user.TenantID.String(), claims.TenantID)
	assert.Equal(t, roles, claims.Roles)
	assert.Equal(t, "test-auth-service", claims.Issuer)
	assert.NotEmpty(t, claims.ID)
}

func TestValidateAccessToken_Expired(t *testing.T) {
	svc := NewTokenService(config.JWTConfig{
		Secret:          "test-secret-minimum-32-characters-long!",
		AccessTokenTTL:  -1 * time.Minute, // already expired
		RefreshTokenTTL: 168 * time.Hour,
		Issuer:          "test",
	})
	user := &models.User{
		ID:       uuid.New(),
		TenantID: uuid.New(),
	}

	token, err := svc.GenerateAccessToken(user, nil)
	require.NoError(t, err)

	_, err = svc.ValidateAccessToken(token)
	assert.Error(t, err)
}

func TestValidateAccessToken_WrongSecret(t *testing.T) {
	svc1 := NewTokenService(config.JWTConfig{
		Secret:         "secret-one-minimum-32-characters-long!",
		AccessTokenTTL: 15 * time.Minute,
		Issuer:         "test",
	})
	svc2 := NewTokenService(config.JWTConfig{
		Secret:         "secret-two-minimum-32-characters-long!",
		AccessTokenTTL: 15 * time.Minute,
		Issuer:         "test",
	})

	user := &models.User{ID: uuid.New(), TenantID: uuid.New()}
	token, err := svc1.GenerateAccessToken(user, nil)
	require.NoError(t, err)

	_, err = svc2.ValidateAccessToken(token)
	assert.Error(t, err)
}

func TestValidateAccessToken_Tampered(t *testing.T) {
	svc := newTestTokenService()
	user := &models.User{ID: uuid.New(), TenantID: uuid.New()}

	token, err := svc.GenerateAccessToken(user, nil)
	require.NoError(t, err)

	// Tamper with the token
	tampered := token[:len(token)-5] + "XXXXX"
	_, err = svc.ValidateAccessToken(tampered)
	assert.Error(t, err)
}

func TestGenerateRefreshToken(t *testing.T) {
	svc := newTestTokenService()

	raw1, hash1, err := svc.GenerateRefreshToken()
	require.NoError(t, err)
	assert.NotEmpty(t, raw1)
	assert.NotEmpty(t, hash1)
	assert.NotEqual(t, raw1, hash1) // raw != hash

	raw2, hash2, err := svc.GenerateRefreshToken()
	require.NoError(t, err)
	assert.NotEqual(t, raw1, raw2)  // different each time
	assert.NotEqual(t, hash1, hash2)
}

func TestHashToken(t *testing.T) {
	hash1 := HashToken("test-token-1")
	hash2 := HashToken("test-token-1")
	hash3 := HashToken("test-token-2")

	assert.Equal(t, hash1, hash2)    // deterministic
	assert.NotEqual(t, hash1, hash3) // different input = different hash
	assert.Len(t, hash1, 64)         // SHA-256 hex = 64 chars
}

func TestAccessTokenTTLSeconds(t *testing.T) {
	svc := newTestTokenService()
	assert.Equal(t, int64(900), svc.AccessTokenTTLSeconds()) // 15 min = 900s
}

func TestRefreshTokenTTL(t *testing.T) {
	svc := newTestTokenService()
	assert.Equal(t, 168*time.Hour, svc.RefreshTokenTTL())
}
