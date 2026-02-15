package handlers

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/hightrex/microservices-platform/libs/go/pkg/validation"
	"github.com/hightrex/microservices-platform/services/notification-service/internal/models"
	"github.com/hightrex/microservices-platform/services/notification-service/internal/service"
)

// PreferenceHandler handles preference HTTP requests.
type PreferenceHandler struct {
	prefSvc *service.PreferenceService
}

// NewPreferenceHandler creates a new PreferenceHandler.
func NewPreferenceHandler(prefSvc *service.PreferenceService) *PreferenceHandler {
	return &PreferenceHandler{prefSvc: prefSvc}
}

// GetPreferences handles GET /api/v1/notifications/preferences
func (h *PreferenceHandler) GetPreferences(c *gin.Context) {
	userID, err := getUserID(c)
	if err != nil {
		return
	}

	prefs, err := h.prefSvc.GetPreferences(c.Request.Context(), userID)
	if err != nil {
		HandleError(c, err)
		return
	}

	if prefs == nil {
		prefs = h.prefSvc.GetDefaultPreferences()
	}

	SuccessResponse(c, http.StatusOK, gin.H{"preferences": prefs})
}

// BulkUpdate handles PUT /api/v1/notifications/preferences
func (h *PreferenceHandler) BulkUpdate(c *gin.Context) {
	userID, err := getUserID(c)
	if err != nil {
		return
	}

	var req models.BulkUpdatePreferencesRequest
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

	if err := h.prefSvc.BulkUpdatePreferences(c.Request.Context(), userID, req); err != nil {
		HandleError(c, err)
		return
	}

	SuccessWithMessage(c, http.StatusOK, nil, "Preferences updated successfully")
}

// UpdateSingle handles PUT /api/v1/notifications/preferences/:channel/:event_type
func (h *PreferenceHandler) UpdateSingle(c *gin.Context) {
	userID, err := getUserID(c)
	if err != nil {
		return
	}

	channel := models.Channel(c.Param("channel"))
	if !channel.IsValid() {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"error":   gin.H{"code": "INVALID_REQUEST", "message": "Invalid channel"},
		})
		return
	}

	eventType := c.Param("event_type")
	if eventType == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"error":   gin.H{"code": "INVALID_REQUEST", "message": "Event type is required"},
		})
		return
	}

	var req struct {
		Enabled bool `json:"enabled"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"error":   gin.H{"code": "INVALID_REQUEST", "message": "Invalid request body"},
		})
		return
	}

	prefReq := models.UpdatePreferenceRequest{
		Channel:   channel,
		EventType: eventType,
		Enabled:   req.Enabled,
	}

	if err := h.prefSvc.UpdatePreference(c.Request.Context(), userID, prefReq); err != nil {
		HandleError(c, err)
		return
	}

	SuccessWithMessage(c, http.StatusOK, nil, "Preference updated successfully")
}

// getUserID extracts and validates the user ID from the Gin context.
func getUserID(c *gin.Context) (uuid.UUID, error) {
	userIDStr, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{
			"success": false,
			"error":   gin.H{"code": "UNAUTHORIZED", "message": "User ID not found in context"},
		})
		return uuid.Nil, errUnauthorized
	}

	userID, err := uuid.Parse(userIDStr.(string))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"error":   gin.H{"code": "INVALID_REQUEST", "message": "Invalid user ID"},
		})
		return uuid.Nil, err
	}

	return userID, nil
}

var errUnauthorized = &unauthorizedError{}

type unauthorizedError struct{}

func (e *unauthorizedError) Error() string { return "unauthorized" }
