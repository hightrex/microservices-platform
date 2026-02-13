package middleware

import (
	"bytes"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
)

func TestRejectBodyIdentity(t *testing.T) {
	gin.SetMode(gin.TestMode)

	tests := []struct {
		name           string
		method         string
		body           string
		contentType    string
		expectedStatus int
		expectBlocked  bool
	}{
		{
			name:           "POST with tenant_id is rejected",
			method:         "POST",
			body:           `{"tenant_id": "some-id", "name": "test"}`,
			contentType:    "application/json",
			expectedStatus: http.StatusBadRequest,
			expectBlocked:  true,
		},
		{
			name:           "POST with org_id is rejected",
			method:         "POST",
			body:           `{"org_id": "some-org", "name": "test"}`,
			contentType:    "application/json",
			expectedStatus: http.StatusBadRequest,
			expectBlocked:  true,
		},
		{
			name:           "POST with user_id is rejected",
			method:         "POST",
			body:           `{"user_id": "some-user", "email": "a@b.com"}`,
			contentType:    "application/json",
			expectedStatus: http.StatusBadRequest,
			expectBlocked:  true,
		},
		{
			name:           "POST with owner_id is rejected",
			method:         "POST",
			body:           `{"owner_id": "owner-123", "title": "test"}`,
			contentType:    "application/json",
			expectedStatus: http.StatusBadRequest,
			expectBlocked:  true,
		},
		{
			name:           "POST without identity fields is allowed",
			method:         "POST",
			body:           `{"name": "test", "email": "a@b.com"}`,
			contentType:    "application/json",
			expectedStatus: http.StatusOK,
			expectBlocked:  false,
		},
		{
			name:           "GET requests are skipped",
			method:         "GET",
			body:           "",
			contentType:    "",
			expectedStatus: http.StatusOK,
			expectBlocked:  false,
		},
		{
			name:           "DELETE requests are skipped",
			method:         "DELETE",
			body:           "",
			contentType:    "",
			expectedStatus: http.StatusOK,
			expectBlocked:  false,
		},
		{
			name:           "PUT with identity fields is rejected",
			method:         "PUT",
			body:           `{"tenant_id": "abc", "name": "updated"}`,
			contentType:    "application/json",
			expectedStatus: http.StatusBadRequest,
			expectBlocked:  true,
		},
		{
			name:           "POST with empty body is allowed",
			method:         "POST",
			body:           "",
			contentType:    "application/json",
			expectedStatus: http.StatusOK,
			expectBlocked:  false,
		},
		{
			name:           "POST with non-JSON content type is skipped",
			method:         "POST",
			body:           "tenant_id=abc",
			contentType:    "application/x-www-form-urlencoded",
			expectedStatus: http.StatusOK,
			expectBlocked:  false,
		},
		{
			name:           "POST with invalid JSON is passed through",
			method:         "POST",
			body:           `{not valid json}`,
			contentType:    "application/json",
			expectedStatus: http.StatusOK,
			expectBlocked:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			w := httptest.NewRecorder()
			c, _ := gin.CreateTestContext(w)

			var bodyReader *bytes.Buffer
			if tt.body != "" {
				bodyReader = bytes.NewBufferString(tt.body)
			} else {
				bodyReader = bytes.NewBuffer(nil)
			}

			c.Request, _ = http.NewRequest(tt.method, "/test", bodyReader)
			if tt.contentType != "" {
				c.Request.Header.Set("Content-Type", tt.contentType)
			}
			if tt.body != "" {
				c.Request.ContentLength = int64(len(tt.body))
			}

			handlerCalled := false
			router := gin.New()
			router.Use(RejectBodyIdentity())
			router.Handle(tt.method, "/test", func(c *gin.Context) {
				handlerCalled = true
				c.Status(http.StatusOK)
			})

			router.ServeHTTP(w, c.Request)

			assert.Equal(t, tt.expectedStatus, w.Code)
			if tt.expectBlocked {
				assert.False(t, handlerCalled, "handler should not have been called")
			}
		})
	}
}
