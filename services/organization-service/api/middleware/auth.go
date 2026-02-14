package middleware

import (
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/hightrex/microservices-platform/libs/go/pkg/securitylog"
	"github.com/hightrex/microservices-platform/libs/go/pkg/tenant"
)

// GatewayAuth is a middleware that extracts identity from headers set by the
// API Gateway. The gateway has already validated the JWT; we trust these headers
// for service-to-service communication within the private network.
//
// Headers expected:
//   - X-User-ID: authenticated user's UUID
//   - X-Tenant-ID: organization/tenant UUID
//   - X-User-Roles: comma-separated list of roles
func GatewayAuth() gin.HandlerFunc {
	return func(c *gin.Context) {
		userID := c.GetHeader("X-User-ID")
		tenantIDStr := c.GetHeader("X-Tenant-ID")
		userRoles := c.GetHeader("X-User-Roles")

		// For internal service, we trust the gateway headers
		// If no headers, the request must have an Authorization header
		if userID == "" || tenantIDStr == "" {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{
				"success": false,
				"error":   gin.H{"code": "UNAUTHORIZED", "message": "Missing authentication headers"},
			})
			return
		}

		// Validate tenant ID format
		tid, err := uuid.Parse(tenantIDStr)
		if err != nil {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{
				"success": false,
				"error":   gin.H{"code": "UNAUTHORIZED", "message": "Invalid tenant ID"},
			})
			return
		}

		// Set tenant context (critical for tenant scoping in repositories)
		ctx := tenant.NewContext(c.Request.Context(), tid)
		c.Request = c.Request.WithContext(ctx)

		// Parse roles
		var roles []string
		if userRoles != "" {
			roles = strings.Split(userRoles, ",")
			for i := range roles {
				roles[i] = strings.TrimSpace(roles[i])
			}
		}

		// Set in Gin context for handlers
		c.Set("user_id", userID)
		c.Set("tenant_id", tenantIDStr)
		c.Set("roles", roles)

		c.Next()
	}
}

// RequireRole returns middleware that checks if the user has one of the required roles.
func RequireRole(roles ...string) gin.HandlerFunc {
	return func(c *gin.Context) {
		userRoles, exists := c.Get("roles")
		if !exists {
			c.AbortWithStatusJSON(http.StatusForbidden, gin.H{
				"success": false,
				"error":   gin.H{"code": "FORBIDDEN", "message": "Insufficient permissions"},
			})
			return
		}

		roleList, ok := userRoles.([]string)
		if !ok {
			c.AbortWithStatusJSON(http.StatusForbidden, gin.H{
				"success": false,
				"error":   gin.H{"code": "FORBIDDEN", "message": "Invalid role data"},
			})
			return
		}

		for _, required := range roles {
			for _, userRole := range roleList {
				if required == userRole {
					c.Next()
					return
				}
			}
		}

		securitylog.LogFailure(c.Request.Context(), securitylog.EventPermissionDenied, c.ClientIP(), c.Request.URL.Path, map[string]interface{}{
			"required_roles": roles,
			"user_roles":     roleList,
		})

		c.AbortWithStatusJSON(http.StatusForbidden, gin.H{
			"success": false,
			"error":   gin.H{"code": "FORBIDDEN", "message": "Insufficient permissions"},
		})
	}
}
