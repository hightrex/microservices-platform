package handlers

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/hightrex/microservices-platform/libs/go/pkg/validation"
	"github.com/hightrex/microservices-platform/services/audit-service/internal/models"
	"github.com/hightrex/microservices-platform/services/audit-service/internal/service"
)

// RetentionHandler handles retention policy HTTP requests.
type RetentionHandler struct {
	retentionSvc *service.RetentionService
}

// NewRetentionHandler creates a new RetentionHandler.
func NewRetentionHandler(retentionSvc *service.RetentionService) *RetentionHandler {
	return &RetentionHandler{retentionSvc: retentionSvc}
}

// List handles GET /api/v1/audit/retention-policies
func (h *RetentionHandler) List(c *gin.Context) {
	policies, err := h.retentionSvc.GetPolicies(c.Request.Context())
	if err != nil {
		HandleError(c, err)
		return
	}

	SuccessResponse(c, http.StatusOK, gin.H{"policies": policies})
}

// Create handles POST /api/v1/audit/retention-policies
func (h *RetentionHandler) Create(c *gin.Context) {
	var req models.CreatePolicyRequest
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

	resp, err := h.retentionSvc.CreatePolicy(c.Request.Context(), req)
	if err != nil {
		HandleError(c, err)
		return
	}

	SuccessWithMessage(c, http.StatusCreated, resp, "Retention policy created")
}

// Update handles PUT /api/v1/audit/retention-policies/:id
func (h *RetentionHandler) Update(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"error":   gin.H{"code": "INVALID_REQUEST", "message": "Invalid policy ID format"},
		})
		return
	}

	var req models.UpdatePolicyRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"error":   gin.H{"code": "INVALID_REQUEST", "message": "Invalid request body"},
		})
		return
	}

	if err := h.retentionSvc.UpdatePolicy(c.Request.Context(), id, req); err != nil {
		HandleError(c, err)
		return
	}

	SuccessWithMessage(c, http.StatusOK, nil, "Retention policy updated")
}

// Delete handles DELETE /api/v1/audit/retention-policies/:id
func (h *RetentionHandler) Delete(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"error":   gin.H{"code": "INVALID_REQUEST", "message": "Invalid policy ID format"},
		})
		return
	}

	if err := h.retentionSvc.DeletePolicy(c.Request.Context(), id); err != nil {
		HandleError(c, err)
		return
	}

	SuccessWithMessage(c, http.StatusOK, nil, "Retention policy deleted")
}
