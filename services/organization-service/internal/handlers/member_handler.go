package handlers

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/hightrex/microservices-platform/libs/go/pkg/validation"
	"github.com/hightrex/microservices-platform/services/organization-service/internal/models"
	"github.com/hightrex/microservices-platform/services/organization-service/internal/service"
)

// MemberHandler handles member HTTP requests.
type MemberHandler struct {
	memberService *service.MemberService
}

// NewMemberHandler creates a new MemberHandler.
func NewMemberHandler(memberService *service.MemberService) *MemberHandler {
	return &MemberHandler{memberService: memberService}
}

// List handles GET /api/v1/organizations/:id/members
func (h *MemberHandler) List(c *gin.Context) {
	orgID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"error":   gin.H{"code": "INVALID_REQUEST", "message": "Invalid organization ID format"},
		})
		return
	}

	page := models.DefaultPagination()
	if p := c.Query("page"); p != "" {
		if v, err := strconv.Atoi(p); err == nil && v > 0 {
			page.Page = v
		}
	}
	if ps := c.Query("page_size"); ps != "" {
		if v, err := strconv.Atoi(ps); err == nil && v > 0 && v <= 100 {
			page.PageSize = v
		}
	}

	var filter models.MemberFilter
	if status := c.Query("status"); status != "" {
		s := models.MemberStatus(status)
		filter.Status = &s
	}
	if role := c.Query("role"); role != "" {
		filter.Role = &role
	}

	resp, err := h.memberService.List(c.Request.Context(), orgID, filter, page)
	if err != nil {
		HandleError(c, err)
		return
	}

	SuccessResponse(c, http.StatusOK, resp)
}

// Invite handles POST /api/v1/organizations/:id/members/invite
func (h *MemberHandler) Invite(c *gin.Context) {
	orgID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"error":   gin.H{"code": "INVALID_REQUEST", "message": "Invalid organization ID format"},
		})
		return
	}

	var req models.InviteMemberRequest
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

	// Get the inviter's user ID from JWT context
	var invitedBy uuid.UUID
	if actorID, exists := c.Get("user_id"); exists {
		if uid, parseErr := uuid.Parse(actorID.(string)); parseErr == nil {
			invitedBy = uid
		}
	}

	resp, err := h.memberService.Invite(c.Request.Context(), orgID, req, invitedBy, c.ClientIP())
	if err != nil {
		HandleError(c, err)
		return
	}

	SuccessWithMessage(c, http.StatusCreated, resp, "Member invited successfully")
}

// Remove handles DELETE /api/v1/organizations/:id/members/:userId
func (h *MemberHandler) Remove(c *gin.Context) {
	orgID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"error":   gin.H{"code": "INVALID_REQUEST", "message": "Invalid organization ID format"},
		})
		return
	}

	userID, err := uuid.Parse(c.Param("userId"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"error":   gin.H{"code": "INVALID_REQUEST", "message": "Invalid user ID format"},
		})
		return
	}

	if err := h.memberService.Remove(c.Request.Context(), orgID, userID, c.ClientIP()); err != nil {
		HandleError(c, err)
		return
	}

	SuccessWithMessage(c, http.StatusOK, nil, "Member removed successfully")
}
