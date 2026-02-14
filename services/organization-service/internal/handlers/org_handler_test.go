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

func TestCreateOrg_InvalidJSON(t *testing.T) {
	r := gin.New()
	handler := &OrgHandler{}
	r.POST("/api/v1/organizations", handler.Create)

	body := bytes.NewBufferString(`{invalid json}`)
	req := httptest.NewRequest(http.MethodPost, "/api/v1/organizations", body)
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)

	var resp map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &resp)
	assert.NoError(t, err)
	assert.Equal(t, false, resp["success"])
}

func TestCreateOrg_ValidationFailure(t *testing.T) {
	r := gin.New()
	handler := &OrgHandler{}
	r.POST("/api/v1/organizations", handler.Create)

	payload := map[string]string{"name": "", "slug": ""}
	bodyBytes, _ := json.Marshal(payload)
	req := httptest.NewRequest(http.MethodPost, "/api/v1/organizations", bytes.NewBuffer(bodyBytes))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestCreateOrg_MissingUserID(t *testing.T) {
	r := gin.New()
	handler := &OrgHandler{}
	r.POST("/api/v1/organizations", handler.Create)

	payload := map[string]string{"name": "Test Org", "slug": "testorg"}
	bodyBytes, _ := json.Marshal(payload)
	req := httptest.NewRequest(http.MethodPost, "/api/v1/organizations", bytes.NewBuffer(bodyBytes))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusUnauthorized, w.Code)
}

func TestGetOrg_InvalidID(t *testing.T) {
	r := gin.New()
	handler := &OrgHandler{}
	r.GET("/api/v1/organizations/:id", handler.GetByID)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/organizations/not-a-uuid", nil)
	w := httptest.NewRecorder()

	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestUpdateOrg_InvalidID(t *testing.T) {
	r := gin.New()
	handler := &OrgHandler{}
	r.PUT("/api/v1/organizations/:id", handler.Update)

	req := httptest.NewRequest(http.MethodPut, "/api/v1/organizations/not-a-uuid", nil)
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestUpdateOrg_InvalidJSON(t *testing.T) {
	r := gin.New()
	handler := &OrgHandler{}
	r.PUT("/api/v1/organizations/:id", handler.Update)

	body := bytes.NewBufferString(`{invalid}`)
	req := httptest.NewRequest(http.MethodPut, "/api/v1/organizations/11111111-1111-1111-1111-111111111111", body)
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestSuccessResponse_Format(t *testing.T) {
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
		HandleError(c, fmt.Errorf("some internal error"))
	})

	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	w := httptest.NewRecorder()

	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusInternalServerError, w.Code)
}
