package handlers

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
)

func TestGetByID_InvalidUUID(t *testing.T) {
	r := gin.New()
	handler := &UserHandler{}

	r.GET("/api/v1/users/:id", handler.GetByID)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/users/not-a-uuid", nil)
	w := httptest.NewRecorder()

	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)

	var resp map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &resp)
	assert.NoError(t, err)
	assert.Equal(t, false, resp["success"])
}

func TestUpdate_InvalidUUID(t *testing.T) {
	r := gin.New()
	handler := &UserHandler{}

	r.PUT("/api/v1/users/:id", handler.Update)

	req := httptest.NewRequest(http.MethodPut, "/api/v1/users/not-a-uuid", nil)
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestDelete_InvalidUUID(t *testing.T) {
	r := gin.New()
	handler := &UserHandler{}

	r.DELETE("/api/v1/users/:id", handler.Delete)

	req := httptest.NewRequest(http.MethodDelete, "/api/v1/users/not-a-uuid", nil)
	w := httptest.NewRecorder()

	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestAssignRole_InvalidUUID(t *testing.T) {
	r := gin.New()
	handler := &UserHandler{}

	r.PUT("/api/v1/users/:id/role", handler.AssignRole)

	req := httptest.NewRequest(http.MethodPut, "/api/v1/users/not-a-uuid/role", nil)
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestListSessions_InvalidUUID(t *testing.T) {
	r := gin.New()
	handler := &UserHandler{}

	r.GET("/api/v1/users/:id/sessions", handler.ListSessions)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/users/not-a-uuid/sessions", nil)
	w := httptest.NewRecorder()

	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestChangePassword_InvalidUUID(t *testing.T) {
	r := gin.New()
	handler := &UserHandler{}

	r.PUT("/api/v1/users/:id/password", handler.ChangePassword)

	req := httptest.NewRequest(http.MethodPut, "/api/v1/users/not-a-uuid/password", nil)
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}
