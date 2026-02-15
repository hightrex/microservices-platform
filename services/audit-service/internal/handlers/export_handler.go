package handlers

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/hightrex/microservices-platform/libs/go/pkg/validation"
	"github.com/hightrex/microservices-platform/services/audit-service/internal/models"
	"github.com/hightrex/microservices-platform/services/audit-service/internal/service"
)

// ExportHandler handles audit export HTTP requests.
type ExportHandler struct {
	exportSvc *service.ExportService
}

// NewExportHandler creates a new ExportHandler.
func NewExportHandler(exportSvc *service.ExportService) *ExportHandler {
	return &ExportHandler{exportSvc: exportSvc}
}

// CreateExport handles POST /api/v1/audit/logs/export
func (h *ExportHandler) CreateExport(c *gin.Context) {
	var req models.CreateExportRequest
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
	requestedBy, err := uuid.Parse(userIDStr.(string))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"error":   gin.H{"code": "INVALID_REQUEST", "message": "Invalid user ID in context"},
		})
		return
	}

	resp, err := h.exportSvc.CreateExport(c.Request.Context(), req, requestedBy)
	if err != nil {
		HandleError(c, err)
		return
	}

	SuccessWithMessage(c, http.StatusCreated, resp, "Export job created")
}

// GetExport handles GET /api/v1/audit/logs/export/:id
func (h *ExportHandler) GetExport(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"error":   gin.H{"code": "INVALID_REQUEST", "message": "Invalid export ID format"},
		})
		return
	}

	resp, err := h.exportSvc.GetExport(c.Request.Context(), id)
	if err != nil {
		HandleError(c, err)
		return
	}

	SuccessResponse(c, http.StatusOK, resp)
}

// ListExports handles GET /api/v1/audit/logs/exports
func (h *ExportHandler) ListExports(c *gin.Context) {
	page := parsePagination(c)

	exports, total, err := h.exportSvc.ListExports(c.Request.Context(), page)
	if err != nil {
		HandleError(c, err)
		return
	}

	SuccessResponse(c, http.StatusOK, gin.H{
		"exports":   exports,
		"total":     total,
		"page":      page.Page,
		"page_size": page.PageSize,
	})
}
