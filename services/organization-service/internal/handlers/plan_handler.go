package handlers

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/hightrex/microservices-platform/libs/go/pkg/validation"
	"github.com/hightrex/microservices-platform/services/organization-service/internal/models"
	"github.com/hightrex/microservices-platform/services/organization-service/internal/service"
)

// PlanHandler handles plan HTTP requests.
type PlanHandler struct {
	planService *service.PlanService
}

// NewPlanHandler creates a new PlanHandler.
func NewPlanHandler(planService *service.PlanService) *PlanHandler {
	return &PlanHandler{planService: planService}
}

// GetCurrent handles GET /api/v1/organizations/:id/plan
func (h *PlanHandler) GetCurrent(c *gin.Context) {
	orgID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"error":   gin.H{"code": "INVALID_REQUEST", "message": "Invalid organization ID format"},
		})
		return
	}

	resp, err := h.planService.GetCurrent(c.Request.Context(), orgID)
	if err != nil {
		HandleError(c, err)
		return
	}

	SuccessResponse(c, http.StatusOK, resp)
}

// Change handles PUT /api/v1/organizations/:id/plan
func (h *PlanHandler) Change(c *gin.Context) {
	orgID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"error":   gin.H{"code": "INVALID_REQUEST", "message": "Invalid organization ID format"},
		})
		return
	}

	var req models.ChangePlanRequest
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

	resp, err := h.planService.Change(c.Request.Context(), orgID, req, c.ClientIP())
	if err != nil {
		HandleError(c, err)
		return
	}

	SuccessWithMessage(c, http.StatusOK, resp, "Plan changed successfully")
}

// ListAvailable handles GET /api/v1/plans
func (h *PlanHandler) ListAvailable(c *gin.Context) {
	resp, err := h.planService.ListAvailable(c.Request.Context())
	if err != nil {
		HandleError(c, err)
		return
	}

	SuccessResponse(c, http.StatusOK, resp)
}
