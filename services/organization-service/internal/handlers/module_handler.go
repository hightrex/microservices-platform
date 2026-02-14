package handlers

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/hightrex/microservices-platform/libs/go/pkg/validation"
	"github.com/hightrex/microservices-platform/services/organization-service/internal/models"
	"github.com/hightrex/microservices-platform/services/organization-service/internal/service"
)

// ModuleHandler handles module HTTP requests.
type ModuleHandler struct {
	moduleService *service.ModuleService
}

// NewModuleHandler creates a new ModuleHandler.
func NewModuleHandler(moduleService *service.ModuleService) *ModuleHandler {
	return &ModuleHandler{moduleService: moduleService}
}

// ListModules handles GET /api/v1/organizations/:id/modules
func (h *ModuleHandler) ListModules(c *gin.Context) {
	orgID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"error":   gin.H{"code": "INVALID_REQUEST", "message": "Invalid organization ID format"},
		})
		return
	}

	modules, err := h.moduleService.GetModules(c.Request.Context(), orgID)
	if err != nil {
		HandleError(c, err)
		return
	}

	SuccessResponse(c, http.StatusOK, modules)
}

// ToggleModule handles PUT /api/v1/organizations/:id/modules
func (h *ModuleHandler) ToggleModule(c *gin.Context) {
	orgID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"error":   gin.H{"code": "INVALID_REQUEST", "message": "Invalid organization ID format"},
		})
		return
	}

	var req models.ToggleModuleRequest
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

	if err := h.moduleService.ToggleModule(c.Request.Context(), orgID, req.ModuleName, req.Enabled); err != nil {
		HandleError(c, err)
		return
	}

	action := "disabled"
	if req.Enabled {
		action = "enabled"
	}
	SuccessWithMessage(c, http.StatusOK, nil, "Module "+req.ModuleName+" "+action+" successfully")
}

// GetModuleConfig handles GET /api/v1/organizations/:id/modules/:name/config
func (h *ModuleHandler) GetModuleConfig(c *gin.Context) {
	orgID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"error":   gin.H{"code": "INVALID_REQUEST", "message": "Invalid organization ID format"},
		})
		return
	}

	moduleName := c.Param("name")
	if moduleName == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"error":   gin.H{"code": "INVALID_REQUEST", "message": "Module name is required"},
		})
		return
	}

	config, err := h.moduleService.GetModuleConfig(c.Request.Context(), orgID, moduleName)
	if err != nil {
		HandleError(c, err)
		return
	}

	SuccessResponse(c, http.StatusOK, config)
}
