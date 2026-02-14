package e2e

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"testing"
	"time"
)

const (
	baseURL = "http://localhost:3000/api/v1"
)

// Data structs for requests/responses
type RegisterRequest struct {
	Email     string `json:"email"`
	Password  string `json:"password"`
	FirstName string `json:"first_name"`
	LastName  string `json:"last_name"`
}

type LoginRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

type LoginResponse struct {
	Success bool `json:"success"`
	Data    struct {
		AccessToken  string `json:"access_token"`
		RefreshToken string `json:"refresh_token"`
	} `json:"data"`
}

type CreateOrgRequest struct {
	Name string `json:"name"`
	Slug string `json:"slug"`
}

type OrgResponse struct {
	Success bool `json:"success"`
	Data    struct {
		ID   string `json:"id"`
		Name string `json:"name"`
		Slug string `json:"slug"`
	} `json:"data"`
}

func TestIntegrationFlow(t *testing.T) {
	// Generate unique email/slug/tenant
	uniqueID := fmt.Sprintf("%d", time.Now().UnixNano())
	// Format: xxxxxxxx-xxxx-xxxx-xxxx-xxxxxxxxxxxx
	// We use a prefix + part of uniqueID (digits are valid hex)
	paddedID := fmt.Sprintf("%012s", uniqueID)
	tenantID := fmt.Sprintf("123e4567-e89b-12d3-a456-%s", paddedID[:12])

	email := fmt.Sprintf("testuser_%s@example.com", uniqueID)
	orgName := fmt.Sprintf("Test Org %s", uniqueID)
	orgSlug := fmt.Sprintf("testorg%s", uniqueID) // Alphanumeric only

	password := "SecurePass123!"

	client := &http.Client{
		Timeout: 10 * time.Second,
	}

	// 1. Register
	t.Run("Register", func(t *testing.T) {
		reqBody, _ := json.Marshal(RegisterRequest{
			Email:     email,
			Password:  password,
			FirstName: "Test",
			LastName:  "User",
		})
		req, _ := http.NewRequest("POST", baseURL+"/auth/register", bytes.NewBuffer(reqBody))
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("X-Tenant-ID", tenantID)

		resp, err := client.Do(req)
		if err != nil {
			t.Fatalf("Failed to register: %v", err)
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusCreated && resp.StatusCode != http.StatusOK {
			body, _ := io.ReadAll(resp.Body)
			t.Fatalf("Expected 201/200, got %d. Body: %s", resp.StatusCode, string(body))
		}
	})

	// 2. Login
	var accessToken string
	t.Run("Login", func(t *testing.T) {
		reqBody, _ := json.Marshal(LoginRequest{
			Email:    email,
			Password: password,
		})
		req, _ := http.NewRequest("POST", baseURL+"/auth/login", bytes.NewBuffer(reqBody))
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("X-Tenant-ID", tenantID)

		resp, err := client.Do(req)
		if err != nil {
			t.Fatalf("Failed to login: %v", err)
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusOK {
			body, _ := io.ReadAll(resp.Body)
			t.Fatalf("Expected 200, got %d. Body: %s", resp.StatusCode, string(body))
		}

		var loginResp LoginResponse
		if err := json.NewDecoder(resp.Body).Decode(&loginResp); err != nil {
			t.Fatalf("Failed to decode login response: %v", err)
		}

		if !loginResp.Success || loginResp.Data.AccessToken == "" {
			t.Fatalf("Login failed or no token returned")
		}
		accessToken = loginResp.Data.AccessToken
	})

	// 3. Create Org (Protected Route)
	// This verifies: Gateway JWT validation -> Identity Header Injection -> Org Service RBAC
	t.Run("CreateOrg", func(t *testing.T) {
		reqBody, _ := json.Marshal(CreateOrgRequest{
			Name: orgName,
			Slug: orgSlug,
		})

		req, _ := http.NewRequest("POST", baseURL+"/organizations", bytes.NewBuffer(reqBody))
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Authorization", "Bearer "+accessToken)

		resp, err := client.Do(req)
		if err != nil {
			t.Fatalf("Failed to create org: %v", err)
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusCreated && resp.StatusCode != http.StatusOK {
			body, _ := io.ReadAll(resp.Body)
			t.Fatalf("Expected 201/200, got %d. Body: %s", resp.StatusCode, string(body))
		}

		var orgResp OrgResponse
		if err := json.NewDecoder(resp.Body).Decode(&orgResp); err != nil {
			t.Fatalf("Failed to decode org response: %v", err)
		}

		if orgResp.Data.Slug != orgSlug {
			t.Fatalf("Created org slug mismatch. Expected %s, got %s", orgSlug, orgResp.Data.Slug)
		}
	})

	// 4. Verify Gateway Gating (Module Check)
	// For now, checks that we get a valid response (even if denied) rather than a 404 or 500
	// Ideally, we'd toggle a module off and check 403, but default state is usually enabled/undefined.
	// Let's check a placeholder route is 503 as configured in the Gateway.
	t.Run("CheckPlaceholderRoute", func(t *testing.T) {
		req, _ := http.NewRequest("GET", baseURL+"/billing/invoices", nil)
		req.Header.Set("Authorization", "Bearer "+accessToken)

		resp, err := client.Do(req)
		if err != nil {
			t.Fatalf("Failed to call placeholder: %v", err)
		}
		defer resp.Body.Close()

		// Gateway is configured to return 503 for /api/v1/billing
		if resp.StatusCode != http.StatusServiceUnavailable {
			t.Fatalf("Expected 503 for placeholder route, got %d", resp.StatusCode)
		}
	})
}
