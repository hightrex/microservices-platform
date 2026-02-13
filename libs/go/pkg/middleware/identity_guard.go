package middleware

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/hightrex/microservices-platform/libs/go/pkg/logger"
)

// ProtectedIdentityFields are the fields that MUST NOT be accepted from
// request bodies. Identity is always derived from JWT/context, never from
// client-supplied payloads. This prevents privilege escalation and
// tenant-boundary violations.
var ProtectedIdentityFields = []string{
	"tenant_id",
	"org_id",
	"user_id",
	"owner_id",
}

// RejectBodyIdentity middleware inspects JSON request bodies and rejects
// requests that contain any of the protected identity fields. This enforces
// the principle that identity comes from JWT/context only, never from the
// request body.
//
// This runs as an early middleware, before handlers can bind the body.
//
// If a protected field is found, returns 400 Bad Request with a clear error.
// GET/DELETE/HEAD/OPTIONS requests (no body expected) are skipped.
func RejectBodyIdentity() gin.HandlerFunc {
	return func(c *gin.Context) {
		// Skip methods that typically have no body
		if c.Request.Method == http.MethodGet ||
			c.Request.Method == http.MethodDelete ||
			c.Request.Method == http.MethodHead ||
			c.Request.Method == http.MethodOptions {
			c.Next()
			return
		}

		// Skip if no body or content type is not JSON
		contentType := c.ContentType()
		if contentType != "application/json" && contentType != "" {
			c.Next()
			return
		}

		if c.Request.Body == nil || c.Request.ContentLength == 0 {
			c.Next()
			return
		}

		// Read the body (and restore it for downstream handlers)
		bodyBytes, err := io.ReadAll(c.Request.Body)
		if err != nil {
			c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{
				"error":   "invalid_request",
				"message": "Failed to read request body",
			})
			return
		}
		// Restore the body for downstream handlers
		c.Request.Body = io.NopCloser(bytes.NewBuffer(bodyBytes))

		// Skip empty bodies
		if len(bytes.TrimSpace(bodyBytes)) == 0 {
			c.Next()
			return
		}

		// Parse as generic JSON object
		var body map[string]interface{}
		if err := json.Unmarshal(bodyBytes, &body); err != nil {
			// If it's not valid JSON, let the handler deal with it
			c.Next()
			return
		}

		// Check for protected fields
		for _, field := range ProtectedIdentityFields {
			if _, exists := body[field]; exists {
				logger.Warn().
					Str("field", field).
					Str("path", c.Request.URL.Path).
					Str("method", c.Request.Method).
					Msg("Rejected request: identity field in body")

				c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{
					"error":   "invalid_request",
					"message": "Identity field '" + field + "' must not be included in request body. Identity is derived from authentication context.",
				})
				return
			}
		}

		c.Next()
	}
}
