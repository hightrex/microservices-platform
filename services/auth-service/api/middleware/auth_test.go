package middleware

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/hightrex/microservices-platform/services/auth-service/internal/config"
	"github.com/hightrex/microservices-platform/services/auth-service/internal/models"
	"github.com/hightrex/microservices-platform/services/auth-service/internal/service"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func init() {
	gin.SetMode(gin.TestMode)
}

func newTestTokenService() *service.TokenService {
	return service.NewTokenService(config.JWTConfig{
		Secret:          "test-secret-minimum-32-characters-long!",
		AccessTokenTTL:  15 * time.Minute,
		RefreshTokenTTL: 168 * time.Hour,
		Issuer:          "test",
	})
}

// mockTokenCache implements service.TokenCache for tests.
type mockTokenCache struct {
	blacklisted map[string]bool
}

func newMockTokenCache() *mockTokenCache {
	return &mockTokenCache{blacklisted: make(map[string]bool)}
}

func (m *mockTokenCache) BlacklistToken(_ context.Context, tokenID string, _ int64) error {
	m.blacklisted[tokenID] = true
	return nil
}

func (m *mockTokenCache) IsBlacklisted(_ context.Context, tokenID string) (bool, error) {
	return m.blacklisted[tokenID], nil
}

func (m *mockTokenCache) CacheSession(_ context.Context, _ string, _ *models.Session) error {
	return nil
}
func (m *mockTokenCache) GetCachedSession(_ context.Context, _ string) (*models.Session, error) {
	return nil, nil
}
func (m *mockTokenCache) InvalidateSession(_ context.Context, _ string) error { return nil }
func (m *mockTokenCache) StoreMFAToken(_ context.Context, _, _ string) error   { return nil }
func (m *mockTokenCache) GetMFAToken(_ context.Context, _ string) (string, error) {
	return "", nil
}
func (m *mockTokenCache) DeleteMFAToken(_ context.Context, _ string) error { return nil }

func TestJWTAuth_MissingHeader(t *testing.T) {
	tokenSvc := newTestTokenService()
	cache := newMockTokenCache()

	r := gin.New()
	r.Use(JWTAuth(tokenSvc, cache))
	r.GET("/test", func(c *gin.Context) {
		c.JSON(200, gin.H{"ok": true})
	})

	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusUnauthorized, w.Code)
}

func TestJWTAuth_InvalidFormat(t *testing.T) {
	tokenSvc := newTestTokenService()
	cache := newMockTokenCache()

	r := gin.New()
	r.Use(JWTAuth(tokenSvc, cache))
	r.GET("/test", func(c *gin.Context) {
		c.JSON(200, gin.H{"ok": true})
	})

	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	req.Header.Set("Authorization", "InvalidFormat")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusUnauthorized, w.Code)
}

func TestJWTAuth_ValidToken(t *testing.T) {
	tokenSvc := newTestTokenService()
	cache := newMockTokenCache()
	user := &models.User{
		ID:       uuid.New(),
		TenantID: uuid.New(),
	}
	token, err := tokenSvc.GenerateAccessToken(user, []string{"member"})
	require.NoError(t, err)

	var capturedUserID string
	r := gin.New()
	r.Use(JWTAuth(tokenSvc, cache))
	r.GET("/test", func(c *gin.Context) {
		uid, _ := c.Get("user_id")
		capturedUserID = uid.(string)
		c.JSON(200, gin.H{"ok": true})
	})

	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.Equal(t, user.ID.String(), capturedUserID)
}

func TestJWTAuth_BlacklistedToken(t *testing.T) {
	tokenSvc := newTestTokenService()
	cache := newMockTokenCache()
	user := &models.User{
		ID:       uuid.New(),
		TenantID: uuid.New(),
	}
	token, err := tokenSvc.GenerateAccessToken(user, nil)
	require.NoError(t, err)

	// Parse claims to get token ID for blacklisting
	claims, err := tokenSvc.ValidateAccessToken(token)
	require.NoError(t, err)
	cache.blacklisted[claims.ID] = true

	r := gin.New()
	r.Use(JWTAuth(tokenSvc, cache))
	r.GET("/test", func(c *gin.Context) {
		c.JSON(200, gin.H{"ok": true})
	})

	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusUnauthorized, w.Code)
}

func TestRequireRole_Authorized(t *testing.T) {
	r := gin.New()
	r.Use(func(c *gin.Context) {
		c.Set("roles", []string{"org_admin", "member"})
		c.Next()
	})
	r.Use(RequireRole("org_admin", "org_owner"))
	r.GET("/test", func(c *gin.Context) {
		c.JSON(200, gin.H{"ok": true})
	})

	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
}

func TestRequireRole_Unauthorized(t *testing.T) {
	r := gin.New()
	r.Use(func(c *gin.Context) {
		c.Set("roles", []string{"member", "viewer"})
		c.Next()
	})
	r.Use(RequireRole("org_admin", "org_owner"))
	r.GET("/test", func(c *gin.Context) {
		c.JSON(200, gin.H{"ok": true})
	})

	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusForbidden, w.Code)
}

func TestRequireRole_NoRoles(t *testing.T) {
	r := gin.New()
	r.Use(RequireRole("org_admin"))
	r.GET("/test", func(c *gin.Context) {
		c.JSON(200, gin.H{"ok": true})
	})

	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusForbidden, w.Code)
}
