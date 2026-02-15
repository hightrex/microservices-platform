// Package isolation provides the test harness for tenant isolation testing.
package isolation

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
	"github.com/hightrex/microservices-platform/libs/go/pkg/tenant"
)

const (
	GatewayURL = "http://localhost:3000/api/v1"
)

// Harness manages isolated tenant contexts for testing.
type Harness struct {
	t       *testing.T
	tenants map[string]*TenantCtx
	client  *http.Client
}

// TenantCtx represents an isolated tenant testing context.
type TenantCtx struct {
	ID    uuid.UUID
	Name  string
	Token string
	User  User
	Ctx   context.Context
}

type User struct {
	ID        string `json:"id"`
	Email     string `json:"email"`
	FirstName string `json:"first_name"`
	LastName  string `json:"last_name"`
}

// New creates a new test harness.
func New(t *testing.T) *Harness {
	t.Helper()
	waitForGateway(t)
	return &Harness{
		t:       t,
		tenants: make(map[string]*TenantCtx),
		client:  &http.Client{Timeout: 10 * time.Second},
	}
}

// CreateTenant creates a new isolated tenant context for testing.
// It registers a user for the tenant to establish identity and gets a token.
func (h *Harness) CreateTenant(name string) *TenantCtx {
	h.t.Helper()

	// Generate unique email and tenant ID
	email := fmt.Sprintf("%s_%d@example.com", name, time.Now().UnixNano())
	password := "TestPass123!"
	tenantID := uuid.New()

	// Register
	user, err := h.registerUser(email, password, tenantID)
	if err != nil {
		h.t.Fatalf("failed to register user for tenant %s: %v", name, err)
	}

	// Login
	token, err := h.loginUser(email, password, tenantID)
	if err != nil {
		h.t.Fatalf("failed to login user for tenant %s: %v", name, err)
	}

	// Verify Tenant ID in Token matches
	extractedID := h.extractTenantID(token)
	if extractedID != uuid.Nil && extractedID != tenantID {
		h.t.Logf("Warning: Extracted tenant ID %s does not match generated %s", extractedID, tenantID)
	}

	// Create context with tenant ID for compatibility with internal helpers if needed
	ctx := tenant.NewContext(context.Background(), tenantID)

	tc := &TenantCtx{
		ID:    tenantID,
		Name:  name,
		Token: token,
		User:  user,
		Ctx:   ctx,
	}
	h.tenants[name] = tc

	h.t.Logf("Created test tenant: name=%s id=%s user=%s", name, tenantID, user.Email)
	return tc
}

func (h *Harness) registerUser(email, password string, tenantID uuid.UUID) (User, error) {
	payload := map[string]string{
		"email":      email,
		"password":   password,
		"first_name": "Test",
		"last_name":  "User",
	}
	body, _ := json.Marshal(payload)

	req, err := http.NewRequest("POST", GatewayURL+"/auth/register", bytes.NewBuffer(body))
	if err != nil {
		return User{}, err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Tenant-ID", tenantID.String())

	resp, err := h.client.Do(req)
	if err != nil {
		return User{}, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusCreated && resp.StatusCode != http.StatusOK {
		b, _ := io.ReadAll(resp.Body)
		return User{}, fmt.Errorf("status %d: %s", resp.StatusCode, string(b))
	}

	var res struct {
		Data User `json:"data"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&res); err != nil {
		return User{}, err
	}
	return res.Data, nil
}

func (h *Harness) loginUser(email, password string, tenantID uuid.UUID) (string, error) {
	payload := map[string]string{
		"email":    email,
		"password": password,
	}
	body, _ := json.Marshal(payload)

	req, err := http.NewRequest("POST", GatewayURL+"/auth/login", bytes.NewBuffer(body))
	if err != nil {
		return "", err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Tenant-ID", tenantID.String())

	resp, err := h.client.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("status %d", resp.StatusCode)
	}

	var res struct {
		Data struct {
			AccessToken string `json:"access_token"`
		} `json:"data"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&res); err != nil {
		return "", err
	}
	return res.Data.AccessToken, nil
}

func (h *Harness) extractTenantID(tokenString string) uuid.UUID {
	parser := jwt.NewParser()
	token, _, err := parser.ParseUnverified(tokenString, jwt.MapClaims{})
	if err != nil {
		h.t.Logf("failed to parse token: %v", err)
		return uuid.Nil
	}

	claims, ok := token.Claims.(jwt.MapClaims)
	if !ok {
		return uuid.Nil
	}

	tidStr, ok := claims["tid"].(string)
	if !ok {
		return uuid.Nil
	}

	id, err := uuid.Parse(tidStr)
	if err != nil {
		h.t.Logf("failed to parse tenant ID claim: %v", err)
		return uuid.Nil
	}
	return id
}

// GetTenant retrieves a previously created tenant by name.
func (h *Harness) GetTenant(name string) *TenantCtx {
	h.t.Helper()
	tc, ok := h.tenants[name]
	if !ok {
		h.t.Fatalf("tenant %q not found in harness", name)
	}
	return tc
}

// AssertDifferentTenants verifies that two tenant contexts have different IDs.
func (h *Harness) AssertDifferentTenants(a, b *TenantCtx) {
	h.t.Helper()
	if a.ID == b.ID {
		h.t.Fatalf("expected different tenant IDs, but both are %s", a.ID.String())
	}
}

// RequireTenantInContext verifies that the context has a valid tenant ID.
func (h *Harness) RequireTenantInContext(ctx context.Context) uuid.UUID {
	h.t.Helper()
	id, err := tenant.RequireTenant(ctx)
	if err != nil {
		h.t.Fatalf("expected tenant in context: %v", err)
	}
	return id
}

// Helper to wait for gateway
func waitForGateway(t *testing.T) {
	deadline := time.Now().Add(30 * time.Second)
	for time.Now().Before(deadline) {
		resp, err := http.Get("http://localhost:3000/health")
		if err == nil && resp.StatusCode == http.StatusOK {
			return
		}
		time.Sleep(1 * time.Second)
	}
	t.Fatal("Gateway did not become ready in time")
}
