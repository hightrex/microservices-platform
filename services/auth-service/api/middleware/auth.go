package middleware

import (
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/hightrex/microservices-platform/libs/go/pkg/securitylog"
	"github.com/hightrex/microservices-platform/services/auth-service/internal/service"
)

// JWTAuth returns middleware that validates JWT access tokens.
func JWTAuth(tokenService *service.TokenService, tokenCache service.TokenCache) gin.HandlerFunc {
	return func(c *gin.Context) {
		authHeader := c.GetHeader("Authorization")
		if authHeader == "" {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{
				"success": false,
				"error":   gin.H{"code": "UNAUTHORIZED", "message": "Missing authorization header"},
			})
			return
		}

		parts := strings.SplitN(authHeader, " ", 2)
		if len(parts) != 2 || strings.ToLower(parts[0]) != "bearer" {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{
				"success": false,
				"error":   gin.H{"code": "UNAUTHORIZED", "message": "Invalid authorization header format"},
			})
			return
		}

		claims, err := tokenService.ValidateAccessToken(parts[1])
		if err != nil {
			securitylog.LogFailure(c.Request.Context(), securitylog.EventTokenInvalid, c.ClientIP(), c.Request.URL.Path, nil)
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{
				"success": false,
				"error":   gin.H{"code": "UNAUTHORIZED", "message": "Invalid or expired token"},
			})
			return
		}

		// Check if token is blacklisted
		if claims.ID != "" {
			blacklisted, err := tokenCache.IsBlacklisted(c.Request.Context(), claims.ID)
			if err == nil && blacklisted {
				securitylog.LogFailure(c.Request.Context(), securitylog.EventTokenInvalid, c.ClientIP(), c.Request.URL.Path, map[string]interface{}{
					"reason": "token_blacklisted",
				})
				c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{
					"success": false,
					"error":   gin.H{"code": "UNAUTHORIZED", "message": "Token has been revoked"},
				})
				return
			}
		}

		// Set user context for downstream handlers
		c.Set("user_id", claims.Subject)
		c.Set("tenant_id", claims.TenantID)
		c.Set("org_id", claims.OrgID)
		c.Set("roles", claims.Roles)
		c.Set("token_id", claims.ID)
		if claims.ExpiresAt != nil {
			c.Set("token_exp", claims.ExpiresAt.Unix())
		}

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

// SelfOrRole returns middleware that allows access if the authenticated user is acting
// on their own resource (matching :id param) OR if they have one of the specified roles.
func SelfOrRole(roles ...string) gin.HandlerFunc {
	return func(c *gin.Context) {
		userID, exists := c.Get("user_id")
		if !exists {
			c.AbortWithStatusJSON(http.StatusForbidden, gin.H{
				"success": false,
				"error":   gin.H{"code": "FORBIDDEN", "message": "Authentication context missing"},
			})
			return
		}

		// Check if the user is accessing their own resource
		pathID := c.Param("id")
		if pathID != "" && userID.(string) == pathID {
			c.Next()
			return
		}

		// Fall back to role check
		userRoles, exists := c.Get("roles")
		if exists {
			roleList, ok := userRoles.([]string)
			if ok {
				for _, required := range roles {
					for _, userRole := range roleList {
						if required == userRole {
							c.Next()
							return
						}
					}
				}
			}
		}

		securitylog.LogFailure(c.Request.Context(), securitylog.EventPermissionDenied, c.ClientIP(), c.Request.URL.Path, map[string]interface{}{
			"required": "self or roles",
			"user_id":  userID,
			"path_id":  pathID,
		})

		c.AbortWithStatusJSON(http.StatusForbidden, gin.H{
			"success": false,
			"error":   gin.H{"code": "FORBIDDEN", "message": "Insufficient permissions"},
		})
	}
}
