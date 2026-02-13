package handlers

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/hightrex/microservices-platform/libs/go/pkg/errors"
	"github.com/stretchr/testify/assert"
)

func init() {
	gin.SetMode(gin.TestMode)
}

func TestRegisterHandler_InvalidJSON(t *testing.T) {
	r := gin.New()
	handler := &AuthHandler{} // authService is nil, but we'll fail before calling it

	r.POST("/api/v1/auth/register", handler.Register)

	body := bytes.NewBufferString(`{invalid json}`)
	req := httptest.NewRequest(http.MethodPost, "/api/v1/auth/register", body)
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)

	var resp map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &resp)
	assert.NoError(t, err)
	assert.Equal(t, false, resp["success"])
}

func TestLoginHandler_InvalidJSON(t *testing.T) {
	r := gin.New()
	handler := &AuthHandler{}

	r.POST("/api/v1/auth/login", handler.Login)

	body := bytes.NewBufferString(`not json`)
	req := httptest.NewRequest(http.MethodPost, "/api/v1/auth/login", body)
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestRefreshTokenHandler_MissingBody(t *testing.T) {
	r := gin.New()
	handler := &AuthHandler{}

	r.POST("/api/v1/auth/refresh", handler.RefreshToken)

	req := httptest.NewRequest(http.MethodPost, "/api/v1/auth/refresh", nil)
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestSetupMFA_NoAuth(t *testing.T) {
	r := gin.New()
	handler := &AuthHandler{}

	r.POST("/api/v1/auth/mfa/setup", handler.SetupMFA)

	req := httptest.NewRequest(http.MethodPost, "/api/v1/auth/mfa/setup", nil)
	w := httptest.NewRecorder()

	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusUnauthorized, w.Code)
}

func TestVerifyMFA_NoAuth(t *testing.T) {
	r := gin.New()
	handler := &AuthHandler{}

	r.POST("/api/v1/auth/mfa/verify", handler.VerifyMFA)

	req := httptest.NewRequest(http.MethodPost, "/api/v1/auth/mfa/verify", nil)
	w := httptest.NewRecorder()

	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusUnauthorized, w.Code)
}

func TestSuccessResponse(t *testing.T) {
	r := gin.New()
	r.GET("/test", func(c *gin.Context) {
		SuccessResponse(c, http.StatusOK, gin.H{"key": "value"})
	})

	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var resp map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &resp)
	assert.NoError(t, err)
	assert.Equal(t, true, resp["success"])
	assert.NotNil(t, resp["data"])
}

func TestHandleError_AppError(t *testing.T) {
	r := gin.New()
	r.GET("/test", func(c *gin.Context) {
		HandleError(c, errors.NotFound("Resource not found", nil))
	})

	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusNotFound, w.Code)

	var resp map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &resp)
	assert.NoError(t, err)
	assert.Equal(t, false, resp["success"])
}

func TestHandleError_GenericError(t *testing.T) {
	r := gin.New()
	r.GET("/test", func(c *gin.Context) {
		HandleError(c, fmt.Errorf("unexpected error"))
	})

	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusInternalServerError, w.Code)
}
