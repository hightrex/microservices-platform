package health

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
)

type mockCheck struct {
	name string
	err  error
}

func (m *mockCheck) Name() string                    { return m.name }
func (m *mockCheck) Check(ctx context.Context) error { return m.err }

func TestHealthHandler(t *testing.T) {
	gin.SetMode(gin.TestMode)

	t.Run("All Healthy", func(t *testing.T) {
		m := NewManager()
		m.AddCheck(&mockCheck{name: "db", err: nil})

		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)
		c.Request, _ = http.NewRequest("GET", "/health", nil)

		m.Handler()(c)

		assert.Equal(t, http.StatusOK, w.Code)
		assert.Contains(t, w.Body.String(), "UP")
	})

	t.Run("One Unhealthy", func(t *testing.T) {
		m := NewManager()
		m.AddCheck(&mockCheck{name: "db", err: errors.New("connection failed")})

		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)
		c.Request, _ = http.NewRequest("GET", "/health", nil)

		m.Handler()(c)

		// We decided 503 for unhealthy
		assert.Equal(t, http.StatusServiceUnavailable, w.Code)
		assert.Contains(t, w.Body.String(), "DOWN")
		assert.Contains(t, w.Body.String(), "connection failed")
	})
}
