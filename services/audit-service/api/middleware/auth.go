package middleware

import (
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/hightrex/microservices-platform/libs/go/pkg/securitylog"
	"github.com/hightrex/microservices-platform/libs/go/pkg/tenant"
)

// GatewayAuth extracts identity from headers set by the API Gateway.
func GatewayAuth() gin.HandlerFunc {
	return func(c *gin.Context) {
		userID := c.GetHeader("X-User-ID")
		tenantIDStr := c.GetHeader("X-Tenant-ID")
		userRoles := c.GetHeader("X-User-Roles")

		if userID == "" || tenantIDStr == "" {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{
				"success": false,
				"error":   gin.H{"code": "UNAUTHORIZED", "message": "Missing authentication headers"},
			})
			return
		}

		tid, err := uuid.Parse(tenantIDStr)
		if err != nil {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{
				"success": false,
				"error":   gin.H{"code": "UNAUTHORIZED", "message": "Invalid tenant ID"},
			})
			return
		}

		ctx := tenant.NewContext(c.Request.Context(), tid)
		c.Request = c.Request.WithContext(ctx)

		var roles []string
		if userRoles != "" {
			roles = strings.Split(userRoles, ",")
			for i := range roles {
				roles[i] = strings.TrimSpace(roles[i])
			}
		}

		c.Set("user_id", userID)
		c.Set("tenant_id", tenantIDStr)
		c.Set("roles", roles)

		c.Next()
	}
}

// RequireRole checks if the user has one of the required roles.
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
