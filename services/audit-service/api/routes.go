package api

import (
	"github.com/gin-gonic/gin"
	"github.com/prometheus/client_golang/prometheus/promhttp"

	"github.com/hightrex/microservices-platform/libs/go/pkg/health"
	"github.com/hightrex/microservices-platform/libs/go/pkg/metrics"
	"github.com/hightrex/microservices-platform/libs/go/pkg/middleware"

	authmw "github.com/hightrex/microservices-platform/services/audit-service/api/middleware"
	"github.com/hightrex/microservices-platform/services/audit-service/internal/handlers"
)

// RegisterRoutes sets up the audit service routes with middleware.
func RegisterRoutes(
	r *gin.Engine,
	auditHandler *handlers.AuditHandler,
	exportHandler *handlers.ExportHandler,
	retentionHandler *handlers.RetentionHandler,
	healthManager *health.Manager,
) {
	// Global middleware (order matters)
	r.Use(middleware.Recovery())
	r.Use(middleware.RequestLogger())
	r.Use(middleware.Tenant())
	r.Use(metrics.Middleware())

	// Infrastructure endpoints (public)
	r.GET("/health", healthManager.Handler())
	r.GET("/metrics", gin.WrapH(promhttp.Handler()))

	// API v1 group — all routes require authentication
	v1 := r.Group("/api/v1")
	v1.Use(authmw.GatewayAuth())

	audit := v1.Group("/audit")
	{
		// Audit log search and retrieval
		audit.GET("/logs", auditHandler.Search)
		audit.GET("/logs/:id", auditHandler.GetByID)
		audit.GET("/stats", auditHandler.GetStats)
		audit.POST("/verify", auditHandler.VerifyIntegrity)

		// Export management
		audit.POST("/logs/export", exportHandler.CreateExport)
		audit.GET("/logs/export/:id", exportHandler.GetExport)
		audit.GET("/logs/exports", exportHandler.ListExports)

		// Retention policy management (admin only)
		retention := audit.Group("/retention-policies")
		retention.Use(authmw.RequireRole("org_owner", "org_admin"))
		{
			retention.GET("", retentionHandler.List)
			retention.POST("", retentionHandler.Create)
			retention.PUT("/:id", retentionHandler.Update)
			retention.DELETE("/:id", retentionHandler.Delete)
		}
	}

	// NOTE: No CREATE endpoint for audit logs — only via event consumer
	// NOTE: No UPDATE or DELETE endpoints for audit logs — immutable
}
