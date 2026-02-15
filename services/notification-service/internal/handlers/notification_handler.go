package handlers

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/hightrex/microservices-platform/libs/go/pkg/validation"
	"github.com/hightrex/microservices-platform/services/notification-service/internal/models"
	"github.com/hightrex/microservices-platform/services/notification-service/internal/service"
)

// NotificationHandler handles notification HTTP requests.
type NotificationHandler struct {
	notifSvc *service.NotificationService
}

// NewNotificationHandler creates a new NotificationHandler.
func NewNotificationHandler(notifSvc *service.NotificationService) *NotificationHandler {
	return &NotificationHandler{notifSvc: notifSvc}
}

// Send handles POST /api/v1/notifications/send
func (h *NotificationHandler) Send(c *gin.Context) {
	var req models.SendNotificationRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"error":   gin.H{"code": "INVALID_REQUEST", "message": "Invalid request body"},
		})
		return
	}

	if errs := validation.Validate(req); errs != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"error":   gin.H{"code": "VALIDATION_FAILED", "message": "Validation failed", "details": errs},
		})
		return
	}

	notifID, err := h.notifSvc.Send(c.Request.Context(), req)
	if err != nil {
		HandleError(c, err)
		return
	}

	if notifID == nil {
		SuccessWithMessage(c, http.StatusOK, nil, "Notification skipped (user opted out)")
		return
	}

	SuccessWithMessage(c, http.StatusCreated, gin.H{"notification_id": notifID}, "Notification sent successfully")
}

// GetByID handles GET /api/v1/notifications/:id
func (h *NotificationHandler) GetByID(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"error":   gin.H{"code": "INVALID_REQUEST", "message": "Invalid notification ID format"},
		})
		return
	}

	notif, err := h.notifSvc.GetByID(c.Request.Context(), id)
	if err != nil {
		HandleError(c, err)
		return
	}

	SuccessResponse(c, http.StatusOK, notif)
}

// List handles GET /api/v1/notifications
func (h *NotificationHandler) List(c *gin.Context) {
	userIDStr, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{
			"success": false,
			"error":   gin.H{"code": "UNAUTHORIZED", "message": "User ID not found in context"},
		})
		return
	}

	userID, err := uuid.Parse(userIDStr.(string))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"error":   gin.H{"code": "INVALID_REQUEST", "message": "Invalid user ID"},
		})
		return
	}

	filter := models.NotificationFilter{}
	if status := c.Query("status"); status != "" {
		s := models.NotificationStatus(status)
		filter.Status = &s
	}
	if channel := c.Query("channel"); channel != "" {
		ch := models.Channel(channel)
		filter.Channel = &ch
	}

	page := parsePagination(c)

	notifications, total, err := h.notifSvc.List(c.Request.Context(), userID, filter, page)
	if err != nil {
		HandleError(c, err)
		return
	}

	SuccessResponse(c, http.StatusOK, gin.H{
		"notifications": notifications,
		"total":         total,
		"page":          page.Page,
		"page_size":     page.PageSize,
	})
}

// MarkAsRead handles PUT /api/v1/notifications/:id/read
func (h *NotificationHandler) MarkAsRead(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"error":   gin.H{"code": "INVALID_REQUEST", "message": "Invalid notification ID format"},
		})
		return
	}

	if err := h.notifSvc.MarkAsRead(c.Request.Context(), id); err != nil {
		HandleError(c, err)
		return
	}

	SuccessWithMessage(c, http.StatusOK, nil, "Notification marked as read")
}

// GetUnreadCount handles GET /api/v1/notifications/unread/count
func (h *NotificationHandler) GetUnreadCount(c *gin.Context) {
	userIDStr, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{
			"success": false,
			"error":   gin.H{"code": "UNAUTHORIZED", "message": "User ID not found in context"},
		})
		return
	}

	userID, err := uuid.Parse(userIDStr.(string))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"error":   gin.H{"code": "INVALID_REQUEST", "message": "Invalid user ID"},
		})
		return
	}

	count, err := h.notifSvc.GetUnreadCount(c.Request.Context(), userID)
	if err != nil {
		HandleError(c, err)
		return
	}

	SuccessResponse(c, http.StatusOK, gin.H{"unread_count": count})
}

// parsePagination extracts pagination parameters from query string.
func parsePagination(c *gin.Context) models.Pagination {
	page := models.DefaultPagination()

	if p := c.Query("page"); p != "" {
		if val, err := strconv.Atoi(p); err == nil && val > 0 {
			page.Page = val
		}
	}

	if ps := c.Query("page_size"); ps != "" {
		if val, err := strconv.Atoi(ps); err == nil && val > 0 && val <= 100 {
			page.PageSize = val
		}
	}

	return page
}
