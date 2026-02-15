package isolation

import (
	"fmt"
	"net/http"
	"testing"
)

// CrossTenantReadScenario verifies that Tenant A cannot read Tenant B's data.
func CrossTenantReadScenario(t *testing.T, h *Harness) {
	t.Log("Starting CrossTenantReadScenario")

	// 1. Setup Tenants
	tenantA := h.CreateTenant("tenantA_read")
	tenantB := h.CreateTenant("tenantB_read")

	h.AssertDifferentTenants(tenantA, tenantB)

	// 2. Tenant B has a user (created in CreateTenant)
	targetUserID := tenantB.User.ID
	t.Logf("Target Resource: User ID %s (belongs to %s)", targetUserID, tenantB.Name)

	// 3. Tenant A tries to read Tenant B's user
	// GET /api/v1/users/:id
	url := fmt.Sprintf("%s/users/%s", GatewayURL, targetUserID)
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		t.Fatalf("failed to create request: %v", err)
	}

	req.Header.Set("Authorization", "Bearer "+tenantA.Token)

	resp, err := h.client.Do(req)
	if err != nil {
		t.Fatalf("failed to perform request: %v", err)
	}
	defer resp.Body.Close()

	// 4. Assert Access Denied
	// Should be 404 (Not Found) or 403 (Forbidden)
	// Ideally 404 so we don't leak existence, but 403 is also acceptable for isolation if scoped.
	// Since `GetByID` is tenant-scoped, it likely won't find the user in Tenant A's scope, so 404.

	if resp.StatusCode == http.StatusOK {
		t.Errorf("Security Violation: Tenant A was able to read Tenant B's user! Status: %d", resp.StatusCode)
	} else if resp.StatusCode != http.StatusNotFound && resp.StatusCode != http.StatusForbidden {
		t.Logf("Unexpected status code: %d (expected 404 or 403)", resp.StatusCode)
	} else {
		t.Logf("Success: Request denied with status %d", resp.StatusCode)
	}
}
