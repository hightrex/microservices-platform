package handlers

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/hightrex/microservices-platform/libs/go/pkg/errors"
	"github.com/hightrex/microservices-platform/libs/go/pkg/logger"
	"github.com/hightrex/microservices-platform/services/audit-service/internal/models"
)

// SuccessResponse returns a standardized success response.
func SuccessResponse(c *gin.Context, status int, data interface{}) {
	c.JSON(status, gin.H{
		"success": true,
		"data":    data,
	})
}

// SuccessWithMessage returns a standardized success response with a message.
func SuccessWithMessage(c *gin.Context, status int, data interface{}, message string) {
	c.JSON(status, gin.H{
		"success": true,
		"data":    data,
		"message": message,
	})
}

// HandleError maps domain errors to HTTP responses.
func HandleError(c *gin.Context, err error) {
	if appErr, ok := err.(*errors.AppError); ok {
		c.JSON(appErr.Code, gin.H{
			"success": false,
			"error": gin.H{
				"code":    errorCode(appErr.Code),
				"message": appErr.Message,
			},
		})
		return
	}

	logger.Error().Err(err).Msg("Unhandled error")
	c.JSON(http.StatusInternalServerError, gin.H{
		"success": false,
		"error": gin.H{
			"code":    "INTERNAL_ERROR",
			"message": "An internal error occurred",
		},
	})
}

func errorCode(status int) string {
	switch status {
	case http.StatusBadRequest:
		return "INVALID_REQUEST"
	case http.StatusUnauthorized:
		return "UNAUTHORIZED"
	case http.StatusForbidden:
		return "FORBIDDEN"
	case http.StatusNotFound:
		return "RESOURCE_NOT_FOUND"
	case http.StatusConflict:
		return "CONFLICT"
	case http.StatusTooManyRequests:
		return "RATE_LIMITED"
	default:
		return "INTERNAL_ERROR"
	}
}

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
