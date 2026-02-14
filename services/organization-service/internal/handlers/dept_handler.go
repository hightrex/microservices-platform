package handlers

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/hightrex/microservices-platform/libs/go/pkg/validation"
	"github.com/hightrex/microservices-platform/services/organization-service/internal/models"
	"github.com/hightrex/microservices-platform/services/organization-service/internal/service"
)

// DeptHandler handles department HTTP requests.
type DeptHandler struct {
	deptService *service.DeptService
}

// NewDeptHandler creates a new DeptHandler.
func NewDeptHandler(deptService *service.DeptService) *DeptHandler {
	return &DeptHandler{deptService: deptService}
}

// List handles GET /api/v1/organizations/:id/departments
func (h *DeptHandler) List(c *gin.Context) {
	orgID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"error":   gin.H{"code": "INVALID_REQUEST", "message": "Invalid organization ID format"},
		})
		return
	}

	depts, err := h.deptService.List(c.Request.Context(), orgID)
	if err != nil {
		HandleError(c, err)
		return
	}

	SuccessResponse(c, http.StatusOK, depts)
}

// Create handles POST /api/v1/organizations/:id/departments
func (h *DeptHandler) Create(c *gin.Context) {
	orgID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"error":   gin.H{"code": "INVALID_REQUEST", "message": "Invalid organization ID format"},
		})
		return
	}

	var req models.CreateDeptRequest
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

	resp, err := h.deptService.Create(c.Request.Context(), orgID, req)
	if err != nil {
		HandleError(c, err)
		return
	}

	SuccessWithMessage(c, http.StatusCreated, resp, "Department created successfully")
}

// Update handles PUT /api/v1/organizations/:id/departments/:deptId
func (h *DeptHandler) Update(c *gin.Context) {
	orgID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"error":   gin.H{"code": "INVALID_REQUEST", "message": "Invalid organization ID format"},
		})
		return
	}

	deptID, err := uuid.Parse(c.Param("deptId"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"error":   gin.H{"code": "INVALID_REQUEST", "message": "Invalid department ID format"},
		})
		return
	}

	var req models.UpdateDeptRequest
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

	resp, err := h.deptService.Update(c.Request.Context(), orgID, deptID, req)
	if err != nil {
		HandleError(c, err)
		return
	}

	SuccessResponse(c, http.StatusOK, resp)
}

// Delete handles DELETE /api/v1/organizations/:id/departments/:deptId
func (h *DeptHandler) Delete(c *gin.Context) {
	orgID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"error":   gin.H{"code": "INVALID_REQUEST", "message": "Invalid organization ID format"},
		})
		return
	}

	deptID, err := uuid.Parse(c.Param("deptId"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"error":   gin.H{"code": "INVALID_REQUEST", "message": "Invalid department ID format"},
		})
		return
	}

	if err := h.deptService.Delete(c.Request.Context(), orgID, deptID); err != nil {
		HandleError(c, err)
		return
	}

	SuccessWithMessage(c, http.StatusOK, nil, "Department deleted successfully")
}
