package middleware

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
)

func init() {
	gin.SetMode(gin.TestMode)
}

func TestGatewayAuth_MissingHeaders(t *testing.T) {
	r := gin.New()
	r.Use(GatewayAuth())
	r.GET("/test", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"ok": true})
	})

	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusUnauthorized, w.Code)
}

func TestGatewayAuth_InvalidTenantID(t *testing.T) {
	r := gin.New()
	r.Use(GatewayAuth())
	r.GET("/test", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"ok": true})
	})

	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	req.Header.Set("X-User-ID", uuid.New().String())
	req.Header.Set("X-Tenant-ID", "not-a-uuid")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusUnauthorized, w.Code)
}

func TestGatewayAuth_ValidHeaders(t *testing.T) {
	r := gin.New()
	r.Use(GatewayAuth())
	r.GET("/test", func(c *gin.Context) {
		userID, _ := c.Get("user_id")
		tenantID, _ := c.Get("tenant_id")
		roles, _ := c.Get("roles")
		c.JSON(http.StatusOK, gin.H{
			"user_id":   userID,
			"tenant_id": tenantID,
			"roles":     roles,
		})
	})

	uid := uuid.New().String()
	tid := uuid.New().String()

	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	req.Header.Set("X-User-ID", uid)
	req.Header.Set("X-Tenant-ID", tid)
	req.Header.Set("X-User-Roles", "org_admin,member")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var resp map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &resp)
	assert.NoError(t, err)
	assert.Equal(t, uid, resp["user_id"])
	assert.Equal(t, tid, resp["tenant_id"])
}

func TestRequireRole_HasRole(t *testing.T) {
	r := gin.New()
	r.Use(func(c *gin.Context) {
		c.Set("roles", []string{"org_admin", "member"})
		c.Next()
	})
	r.Use(RequireRole("org_admin"))
	r.GET("/test", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"ok": true})
	})

	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
}

func TestRequireRole_MissingRole(t *testing.T) {
	r := gin.New()
	r.Use(func(c *gin.Context) {
		c.Set("roles", []string{"member"})
		c.Next()
	})
	r.Use(RequireRole("org_owner", "org_admin"))
	r.GET("/test", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"ok": true})
	})

	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusForbidden, w.Code)
}

func TestRequireRole_NoRolesSet(t *testing.T) {
	r := gin.New()
	r.Use(RequireRole("org_owner"))
	r.GET("/test", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"ok": true})
	})

	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusForbidden, w.Code)
}
