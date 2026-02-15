package api

import (
	"github.com/gin-gonic/gin"
	"github.com/prometheus/client_golang/prometheus/promhttp"

	"github.com/hightrex/microservices-platform/libs/go/pkg/health"
	"github.com/hightrex/microservices-platform/libs/go/pkg/metrics"
	"github.com/hightrex/microservices-platform/libs/go/pkg/middleware"

	authmw "github.com/hightrex/microservices-platform/services/notification-service/api/middleware"
	"github.com/hightrex/microservices-platform/services/notification-service/internal/handlers"
)

// RegisterRoutes sets up the notification service routes with middleware.
func RegisterRoutes(
	r *gin.Engine,
	notifHandler *handlers.NotificationHandler,
	templateHandler *handlers.TemplateHandler,
	preferenceHandler *handlers.PreferenceHandler,
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

	// Notification routes
	notifs := v1.Group("/notifications")
	{
		notifs.POST("/send", notifHandler.Send)
		notifs.GET("", notifHandler.List)
		notifs.GET("/:id", notifHandler.GetByID)
		notifs.PUT("/:id/read", notifHandler.MarkAsRead)
		notifs.GET("/unread/count", notifHandler.GetUnreadCount)

		// Template management (admin only)
		templates := notifs.Group("/templates")
		{
			templates.GET("", templateHandler.List)
			templates.GET("/:id", templateHandler.GetByID)
			templates.POST("",
				authmw.RequireRole("org_owner", "org_admin"),
				templateHandler.Create,
			)
			templates.PUT("/:id",
				authmw.RequireRole("org_owner", "org_admin"),
				templateHandler.Update,
			)
			templates.DELETE("/:id",
				authmw.RequireRole("org_owner", "org_admin"),
				templateHandler.Delete,
			)
		}

		// Preference management
		prefs := notifs.Group("/preferences")
		{
			prefs.GET("", preferenceHandler.GetPreferences)
			prefs.PUT("", preferenceHandler.BulkUpdate)
			prefs.PUT("/:channel/:event_type", preferenceHandler.UpdateSingle)
		}
	}
}
