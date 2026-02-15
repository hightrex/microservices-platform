package isolation

import (
	"fmt"
	"net/http"
	"testing"
)

// CrossTenantDeleteScenario verifies that Tenant A cannot delete Tenant B's resources.
func CrossTenantDeleteScenario(t *testing.T, h *Harness) {
	t.Log("Starting CrossTenantDeleteScenario")

	// 1. Setup Tenants
	tenantA := h.CreateTenant("tenantA_delete")
	tenantB := h.CreateTenant("tenantB_delete")

	// 2. Tenant B has a user
	targetUserID := tenantB.User.ID

	// 3. Tenant A tries to delete Tenant B's user
	// DELETE /api/v1/users/:id
	url := fmt.Sprintf("%s/users/%s", GatewayURL, targetUserID)
	req, err := http.NewRequest("DELETE", url, nil)
	if err != nil {
		t.Fatalf("failed to create request: %v", err)
	}
	req.Header.Set("Authorization", "Bearer "+tenantA.Token)

	resp, err := h.client.Do(req)
	if err != nil {
		t.Fatalf("failed to perform request: %v", err)
	}
	defer resp.Body.Close()

	// 4. Assert Access Denied (404/403)
	if resp.StatusCode == http.StatusOK || resp.StatusCode == http.StatusNoContent {
		t.Errorf("Security Violation: Tenant A was able to delete Tenant B's user! Status: %d", resp.StatusCode)
	} else if resp.StatusCode != http.StatusNotFound && resp.StatusCode != http.StatusForbidden {
		t.Logf("Unexpected status code: %d (expected 404/403)", resp.StatusCode)
	} else {
		t.Logf("Success: Delete denied with status %d", resp.StatusCode)
	}
}
