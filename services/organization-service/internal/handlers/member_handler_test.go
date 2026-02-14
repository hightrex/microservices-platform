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

func TestListMembers_InvalidOrgID(t *testing.T) {
	r := gin.New()
	handler := &MemberHandler{}
	r.GET("/api/v1/organizations/:id/members", handler.List)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/organizations/bad-id/members", nil)
	w := httptest.NewRecorder()

	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestInviteMember_InvalidJSON(t *testing.T) {
	r := gin.New()
	handler := &MemberHandler{}
	r.POST("/api/v1/organizations/:id/members/invite", handler.Invite)

	body := bytes.NewBufferString(`{invalid}`)
	req := httptest.NewRequest(http.MethodPost, "/api/v1/organizations/11111111-1111-1111-1111-111111111111/members/invite", body)
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestInviteMember_ValidationFailure(t *testing.T) {
	r := gin.New()
	handler := &MemberHandler{}
	r.POST("/api/v1/organizations/:id/members/invite", handler.Invite)

	// Missing required role
	payload := map[string]string{"user_id": "11111111-1111-1111-1111-111111111111"}
	bodyBytes, _ := json.Marshal(payload)
	req := httptest.NewRequest(http.MethodPost, "/api/v1/organizations/11111111-1111-1111-1111-111111111111/members/invite", bytes.NewBuffer(bodyBytes))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestRemoveMember_InvalidUserID(t *testing.T) {
	r := gin.New()
	handler := &MemberHandler{}
	r.DELETE("/api/v1/organizations/:id/members/:userId", handler.Remove)

	req := httptest.NewRequest(http.MethodDelete, "/api/v1/organizations/11111111-1111-1111-1111-111111111111/members/bad-uuid", nil)
	w := httptest.NewRecorder()

	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}
