package handlers

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
)

func TestListModules_InvalidOrgID(t *testing.T) {
	r := gin.New()
	handler := &ModuleHandler{}
	r.GET("/api/v1/organizations/:id/modules", handler.ListModules)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/organizations/bad-id/modules", nil)
	w := httptest.NewRecorder()

	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestToggleModule_InvalidJSON(t *testing.T) {
	r := gin.New()
	handler := &ModuleHandler{}
	r.PUT("/api/v1/organizations/:id/modules", handler.ToggleModule)

	body := bytes.NewBufferString(`{invalid}`)
	req := httptest.NewRequest(http.MethodPut, "/api/v1/organizations/11111111-1111-1111-1111-111111111111/modules", body)
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestToggleModule_InvalidModuleName(t *testing.T) {
	r := gin.New()
	handler := &ModuleHandler{}
	r.PUT("/api/v1/organizations/:id/modules", handler.ToggleModule)

	payload := map[string]interface{}{"module_name": "nonexistent_module", "enabled": true}
	bodyBytes, _ := json.Marshal(payload)
	req := httptest.NewRequest(http.MethodPut, "/api/v1/organizations/11111111-1111-1111-1111-111111111111/modules", bytes.NewBuffer(bodyBytes))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestGetModuleConfig_MissingName(t *testing.T) {
	r := gin.New()
	handler := &ModuleHandler{}
	// Route without :name param
	r.GET("/api/v1/organizations/:id/modules//config", handler.GetModuleConfig)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/organizations/11111111-1111-1111-1111-111111111111/modules//config", nil)
	w := httptest.NewRecorder()

	r.ServeHTTP(w, req)

	// This will be 404 from Gin because the route doesn't match empty segment
	assert.True(t, w.Code == http.StatusNotFound || w.Code == http.StatusBadRequest || w.Code == http.StatusMovedPermanently)
}
