package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/hightrex/microservices-platform/libs/go/pkg/tenant"
	"github.com/stretchr/testify/assert"
)

func TestTenantMiddleware(t *testing.T) {
	gin.SetMode(gin.TestMode)

	tests := []struct {
		name           string
		headerValue    string
		expectedStatus int
		expectTenant   bool
	}{
		{
			name:           "Valid Tenant ID",
			headerValue:    uuid.New().String(),
			expectedStatus: http.StatusOK,
			expectTenant:   true,
		},
		{
			name:           "Invalid Tenant ID",
			headerValue:    "invalid-uuid",
			expectedStatus: http.StatusOK, // Currently ignoring invalid
			expectTenant:   false,
		},
		{
			name:           "Missing Header",
			headerValue:    "",
			expectedStatus: http.StatusOK,
			expectTenant:   false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			w := httptest.NewRecorder()
			c, _ := gin.CreateTestContext(w)
			c.Request, _ = http.NewRequest("GET", "/", nil)
			if tt.headerValue != "" {
				c.Request.Header.Set("X-Tenant-ID", tt.headerValue)
			}

			// Capture context in handler
			var capturedTenantID uuid.UUID
			handler := func(c *gin.Context) {
				if val, exists := c.Get("tenant_id"); exists {
					capturedTenantID = val.(uuid.UUID)
				}
				// Verify it's also in the request context
				if tenant.IsSet(c.Request.Context()) {
					// Check consistency
				}
				c.Status(http.StatusOK)
			}

			middleware := Tenant()
			middleware(c)
			handler(c)

			assert.Equal(t, tt.expectedStatus, w.Code)
			if tt.expectTenant {
				assert.NotEqual(t, uuid.Nil, capturedTenantID)
				assert.Equal(t, tt.headerValue, capturedTenantID.String())
			} else {
				assert.Equal(t, uuid.Nil, capturedTenantID)
			}
		})
	}
}
