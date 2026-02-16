package api

import (
	"github.com/gin-gonic/gin"
	"github.com/prometheus/client_golang/prometheus/promhttp"

	"github.com/hightrex/microservices-platform/libs/go/pkg/health"
	"github.com/hightrex/microservices-platform/libs/go/pkg/metrics"
	"github.com/hightrex/microservices-platform/libs/go/pkg/middleware"

	authmw "github.com/hightrex/microservices-platform/services/auth-service/api/middleware"
	"github.com/hightrex/microservices-platform/services/auth-service/internal/handlers"
	"github.com/hightrex/microservices-platform/services/auth-service/internal/service"
)

// RegisterRoutes sets up all routes with the appropriate middleware stack.
func RegisterRoutes(
	r *gin.Engine,
	authHandler *handlers.AuthHandler,
	userHandler *handlers.UserHandler,
	tokenService *service.TokenService,
	tokenCache service.TokenCache,
	healthManager *health.Manager,
) {
	// Global middleware (order matters)
	r.Use(middleware.Recovery())
	r.Use(middleware.RequestLogger())
	r.Use(middleware.Tenant())
	r.Use(middleware.RejectBodyIdentity())
	r.Use(metrics.Middleware())

	// Infrastructure endpoints
	r.GET("/health", healthManager.Handler())
	r.GET("/metrics", gin.WrapH(promhttp.Handler()))

	// API v1 group
	v1 := r.Group("/api/v1")

	// Public auth routes (no JWT required)
	auth := v1.Group("/auth")
	{
		auth.POST("/register", authHandler.Register)
		auth.POST("/login", authHandler.Login)
		auth.POST("/refresh", authHandler.RefreshToken)
	}

	// Protected routes (JWT required)
	protected := v1.Group("")
	protected.Use(authmw.JWTAuth(tokenService, tokenCache))
	{
		// Auth routes that require authentication
		protectedAuth := protected.Group("/auth")
		{
			protectedAuth.POST("/logout", authHandler.Logout)
			protectedAuth.POST("/mfa/setup", authHandler.SetupMFA)
			protectedAuth.POST("/mfa/verify", authHandler.VerifyMFA)
		}

		// User management routes
		users := protected.Group("/users")
		{
			// Listing users requires admin privileges
			users.GET("",
				authmw.RequireRole("org_owner", "org_admin"),
				userHandler.List,
			)
			// Any authenticated user can view their own profile; admins can view others
			users.GET("/:id", userHandler.GetByID)
			// Update and delete: self-only or org_admin/org_owner
			users.PUT("/:id",
				authmw.SelfOrRole("org_owner", "org_admin"),
				userHandler.Update,
			)
			users.DELETE("/:id",
				authmw.RequireRole("org_owner", "org_admin"),
				userHandler.Delete,
			)
			users.PUT("/:id/role",
				authmw.RequireRole("org_owner", "org_admin"),
				userHandler.AssignRole,
			)
			// Sessions: self-only or admin
			users.GET("/:id/sessions",
				authmw.SelfOrRole("org_owner", "org_admin"),
				userHandler.ListSessions,
			)
			// Password change: self-only (users change their own password)
			users.PUT("/:id/password",
				authmw.SelfOrRole("org_owner", "org_admin"),
				userHandler.ChangePassword,
			)
		}
	}
}
