package handlers

import (
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/hightrex/microservices-platform/services/audit-service/internal/models"
	"github.com/hightrex/microservices-platform/services/audit-service/internal/service"
)

// AuditHandler handles audit log HTTP requests.
type AuditHandler struct {
	auditSvc *service.AuditService
}

// NewAuditHandler creates a new AuditHandler.
func NewAuditHandler(auditSvc *service.AuditService) *AuditHandler {
	return &AuditHandler{auditSvc: auditSvc}
}

// Search handles GET /api/v1/audit/logs
func (h *AuditHandler) Search(c *gin.Context) {
	filter := models.SearchRequest{}

	if et := c.Query("event_type"); et != "" {
		filter.EventType = &et
	}
	if cat := c.Query("event_category"); cat != "" {
		ec := models.EventCategory(cat)
		filter.EventCategory = &ec
	}
	if actor := c.Query("actor_id"); actor != "" {
		filter.ActorID = &actor
	}
	if rt := c.Query("resource_type"); rt != "" {
		filter.ResourceType = &rt
	}
	if rid := c.Query("resource_id"); rid != "" {
		filter.ResourceID = &rid
	}
	if o := c.Query("outcome"); o != "" {
		oc := models.Outcome(o)
		filter.Outcome = &oc
	}
	if sd := c.Query("start_date"); sd != "" {
		if t, err := time.Parse(time.RFC3339, sd); err == nil {
			filter.StartDate = &t
		}
	}
	if ed := c.Query("end_date"); ed != "" {
		if t, err := time.Parse(time.RFC3339, ed); err == nil {
			filter.EndDate = &t
		}
	}

	page := parsePagination(c)

	logs, total, err := h.auditSvc.Search(c.Request.Context(), filter, page)
	if err != nil {
		HandleError(c, err)
		return
	}

	SuccessResponse(c, http.StatusOK, gin.H{
		"logs":      logs,
		"total":     total,
		"page":      page.Page,
		"page_size": page.PageSize,
	})
}

// GetByID handles GET /api/v1/audit/logs/:id
func (h *AuditHandler) GetByID(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"error":   gin.H{"code": "INVALID_REQUEST", "message": "Invalid audit log ID format"},
		})
		return
	}

	log, err := h.auditSvc.GetByID(c.Request.Context(), id)
	if err != nil {
		HandleError(c, err)
		return
	}

	SuccessResponse(c, http.StatusOK, log)
}

// GetStats handles GET /api/v1/audit/stats
func (h *AuditHandler) GetStats(c *gin.Context) {
	stats, err := h.auditSvc.GetStatistics(c.Request.Context())
	if err != nil {
		HandleError(c, err)
		return
	}

	SuccessResponse(c, http.StatusOK, stats)
}

// VerifyIntegrity handles POST /api/v1/audit/verify
func (h *AuditHandler) VerifyIntegrity(c *gin.Context) {
	var req struct {
		StartDate time.Time `json:"start_date" binding:"required"`
		EndDate   time.Time `json:"end_date" binding:"required"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"error":   gin.H{"code": "INVALID_REQUEST", "message": "Invalid request body — start_date and end_date required (RFC3339)"},
		})
		return
	}

	report, err := h.auditSvc.VerifyIntegrity(c.Request.Context(), req.StartDate, req.EndDate)
	if err != nil {
		HandleError(c, err)
		return
	}

	SuccessResponse(c, http.StatusOK, report)
}
