package metrics

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
)

func TestMetricsMiddleware(t *testing.T) {
	gin.SetMode(gin.TestMode)

	// We can't easily assert on global prometheus registry state without resetting it,
	// which is tricky in parallel tests.
	// For now, we just ensure the middleware doesn't panic and passes through.

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request, _ = http.NewRequest("GET", "/test", nil)

	handler := func(c *gin.Context) {
		c.Status(http.StatusOK)
	}

	middleware := Middleware()
	middleware(c)
	handler(c)

	assert.Equal(t, http.StatusOK, w.Code)
}
