package api

import (
	"github.com/gin-gonic/gin"
	"github.com/prometheus/client_golang/prometheus/promhttp"

	"github.com/hightrex/microservices-platform/libs/go/pkg/health"
	"github.com/hightrex/microservices-platform/libs/go/pkg/metrics"
	"github.com/hightrex/microservices-platform/libs/go/pkg/middleware"

	authmw "github.com/hightrex/microservices-platform/services/organization-service/api/middleware"
	"github.com/hightrex/microservices-platform/services/organization-service/internal/handlers"
)

// RegisterRoutes sets up the organization service routes with middleware.
func RegisterRoutes(
	r *gin.Engine,
	orgHandler *handlers.OrgHandler,
	moduleHandler *handlers.ModuleHandler,
	memberHandler *handlers.MemberHandler,
	planHandler *handlers.PlanHandler,
	deptHandler *handlers.DeptHandler,
	healthManager *health.Manager,
) {
	// Global middleware (order matters)
	r.Use(middleware.Recovery())
	r.Use(middleware.RequestLogger())
	r.Use(middleware.Tenant())
	// r.Use(middleware.RejectBodyIdentity()) // Breaks Invite Member which needs user_id in body
	r.Use(metrics.Middleware())

	// Infrastructure endpoints (public)
	r.GET("/health", healthManager.Handler())
	r.GET("/metrics", gin.WrapH(promhttp.Handler()))

	// API v1 group — all routes require authentication
	v1 := r.Group("/api/v1")
	v1.Use(authmw.GatewayAuth())

	// Organization routes
	orgs := v1.Group("/organizations")
	{
		// Org creation requires admin role to prevent unbounded org proliferation
		orgs.POST("",
			authmw.RequireRole("org_owner", "org_admin", "admin", "super_admin"),
			orgHandler.Create,
		)

		// Org-specific routes (scoped by tenant context)
		org := orgs.Group("/:id")
		{
			org.GET("", orgHandler.GetByID)

			// Org settings: org_owner or org_admin only
			org.PUT("",
				authmw.RequireRole("org_owner", "org_admin"),
				orgHandler.Update,
			)

			// Module management: org_owner or org_admin only
			org.GET("/modules", moduleHandler.ListModules)
			org.PUT("/modules",
				authmw.RequireRole("org_owner", "org_admin"),
				moduleHandler.ToggleModule,
			)
			org.GET("/modules/:name/config", moduleHandler.GetModuleConfig)

			// Member management: org_admin or higher
			org.GET("/members", memberHandler.List)
			org.POST("/members/invite",
				authmw.RequireRole("org_owner", "org_admin"),
				memberHandler.Invite,
			)
			org.DELETE("/members/:userId",
				authmw.RequireRole("org_owner", "org_admin"),
				memberHandler.Remove,
			)

			// Plan management: org_owner or org_admin only
			org.GET("/plan", planHandler.GetCurrent)
			org.PUT("/plan",
				authmw.RequireRole("org_owner", "org_admin"),
				planHandler.Change,
			)

			// Department management: manager or higher
			org.GET("/departments", deptHandler.List)
			org.POST("/departments",
				authmw.RequireRole("org_owner", "org_admin", "manager"),
				deptHandler.Create,
			)
			org.PUT("/departments/:deptId",
				authmw.RequireRole("org_owner", "org_admin", "manager"),
				deptHandler.Update,
			)
			org.DELETE("/departments/:deptId",
				authmw.RequireRole("org_owner", "org_admin", "manager"),
				deptHandler.Delete,
			)
		}
	}

	// Plan listing (available to any authenticated user)
	v1.GET("/plans", planHandler.ListAvailable)
}
