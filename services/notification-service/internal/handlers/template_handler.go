package handlers

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/hightrex/microservices-platform/libs/go/pkg/validation"
	"github.com/hightrex/microservices-platform/services/notification-service/internal/models"
	"github.com/hightrex/microservices-platform/services/notification-service/internal/service"
)

// TemplateHandler handles template HTTP requests.
type TemplateHandler struct {
	templateSvc *service.TemplateService
}

// NewTemplateHandler creates a new TemplateHandler.
func NewTemplateHandler(templateSvc *service.TemplateService) *TemplateHandler {
	return &TemplateHandler{templateSvc: templateSvc}
}

// Create handles POST /api/v1/notifications/templates
func (h *TemplateHandler) Create(c *gin.Context) {
	var req models.CreateTemplateRequest
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

	userIDStr, _ := c.Get("user_id")
	createdBy, err := uuid.Parse(userIDStr.(string))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"error":   gin.H{"code": "INVALID_REQUEST", "message": "Invalid user ID in context"},
		})
		return
	}

	resp, err := h.templateSvc.CreateTemplate(c.Request.Context(), req, createdBy)
	if err != nil {
		HandleError(c, err)
		return
	}

	SuccessWithMessage(c, http.StatusCreated, resp, "Template created successfully")
}

// List handles GET /api/v1/notifications/templates
func (h *TemplateHandler) List(c *gin.Context) {
	filter := models.TemplateFilter{}
	if channel := c.Query("channel"); channel != "" {
		ch := models.Channel(channel)
		filter.Channel = &ch
	}
	if active := c.Query("is_active"); active != "" {
		a := active == "true"
		filter.IsActive = &a
	}

	page := parsePagination(c)

	templates, total, err := h.templateSvc.ListTemplates(c.Request.Context(), filter, page)
	if err != nil {
		HandleError(c, err)
		return
	}

	SuccessResponse(c, http.StatusOK, gin.H{
		"templates": templates,
		"total":     total,
		"page":      page.Page,
		"page_size": page.PageSize,
	})
}

// GetByID handles GET /api/v1/notifications/templates/:id
func (h *TemplateHandler) GetByID(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"error":   gin.H{"code": "INVALID_REQUEST", "message": "Invalid template ID format"},
		})
		return
	}

	resp, err := h.templateSvc.GetTemplate(c.Request.Context(), id)
	if err != nil {
		HandleError(c, err)
		return
	}

	SuccessResponse(c, http.StatusOK, resp)
}

// Update handles PUT /api/v1/notifications/templates/:id
func (h *TemplateHandler) Update(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"error":   gin.H{"code": "INVALID_REQUEST", "message": "Invalid template ID format"},
		})
		return
	}

	var req models.UpdateTemplateRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"error":   gin.H{"code": "INVALID_REQUEST", "message": "Invalid request body"},
		})
		return
	}

	if err := h.templateSvc.UpdateTemplate(c.Request.Context(), id, req); err != nil {
		HandleError(c, err)
		return
	}

	SuccessWithMessage(c, http.StatusOK, nil, "Template updated successfully")
}

// Delete handles DELETE /api/v1/notifications/templates/:id
func (h *TemplateHandler) Delete(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"error":   gin.H{"code": "INVALID_REQUEST", "message": "Invalid template ID format"},
		})
		return
	}

	if err := h.templateSvc.DeleteTemplate(c.Request.Context(), id); err != nil {
		HandleError(c, err)
		return
	}

	SuccessWithMessage(c, http.StatusOK, nil, "Template deactivated successfully")
}
