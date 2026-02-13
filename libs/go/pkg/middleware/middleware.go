package middleware

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/hightrex/microservices-platform/libs/go/pkg/logger"
	"github.com/hightrex/microservices-platform/libs/go/pkg/tenant"
)

// Tenant middleware extracts the X-Tenant-ID header (or from generic context) and injects it into the request context.
// This example uses Gin, as per the project requirements.
func Tenant() gin.HandlerFunc {
	return func(c *gin.Context) {
		tenantHeader := c.GetHeader("X-Tenant-ID")

		// In a real generic gateway scenario, we might also look at the JWT.
		// For now, we respect the header propagated by the Gateway.

		if tenantHeader == "" {
			// In some cases (public endpoints), tenant might not be present.
			// We continue without it, but handlers requiring it should check.
			c.Next()
			return
		}

		id, err := uuid.Parse(tenantHeader)
		if err != nil {
			logger.Warn().Msgf("Invalid X-Tenant-ID header: %s", tenantHeader)
			// Depending on strictness, we could 400 here, or just ignore.
			// Let's ignore and let the handler decide if it needs a valid tenant.
			c.Next()
			return
		}

		// Store in Gin context and Request context
		c.Set("tenant_id", id)

		// Update the implementation of the request's context to include the tenant
		// This is crucial so that libraries accepting context.Context (like DB repos)
		// can extract the tenant ID using pkg/tenant.
		ctx := tenant.NewContext(c.Request.Context(), id)
		c.Request = c.Request.WithContext(ctx)

		c.Next()
	}
}

// RequestLogger is a simple logging middleware
func RequestLogger() gin.HandlerFunc {
	return func(c *gin.Context) {
		// Start timer
		// ... (implementation using zerolog)
		path := c.Request.URL.Path
		raw := c.Request.URL.RawQuery

		c.Next()

		// Log request details
		logger.Info().
			Str("method", c.Request.Method).
			Str("path", path).
			Str("query", raw).
			Int("status", c.Writer.Status()).
			Msg("Request")
	}
}

// Recovery middleware to recover from panics and log them
func Recovery() gin.HandlerFunc {
	return gin.CustomRecovery(func(c *gin.Context, recovered interface{}) {
		if err, ok := recovered.(string); ok {
			logger.Error().Msgf("panic recovered: %s", err)
		} else {
			logger.Error().Msgf("panic recovered: %v", recovered)
		}
		c.AbortWithStatus(http.StatusInternalServerError)
	})
}
