package isolation

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"testing"
)

// CrossTenantWriteScenario verifies that Tenant A cannot modify Tenant B's data.
func CrossTenantWriteScenario(t *testing.T, h *Harness) {
	t.Log("Starting CrossTenantWriteScenario")

	// 1. Setup Tenants
	tenantA := h.CreateTenant("tenantA_write")
	tenantB := h.CreateTenant("tenantB_write")

	// 2. Tenant B has a user
	targetUserID := tenantB.User.ID

	// 3. Tenant A tries to update Tenant B's user
	// PUT /api/v1/users/:id
	url := fmt.Sprintf("%s/users/%s", GatewayURL, targetUserID)

	payload := map[string]string{
		"first_name": "Hacked",
	}
	body, _ := json.Marshal(payload)
	req, err := http.NewRequest("PUT", url, bytes.NewBuffer(body))
	if err != nil {
		t.Fatalf("failed to create request: %v", err)
	}

	req.Header.Set("Authorization", "Bearer "+tenantA.Token)
	req.Header.Set("Content-Type", "application/json")

	resp, err := h.client.Do(req)
	if err != nil {
		t.Fatalf("failed to perform request: %v", err)
	}
	defer resp.Body.Close()

	// 4. Assert Access Denied (404/403)
	if resp.StatusCode == http.StatusOK {
		t.Errorf("Security Violation: Tenant A was able to update Tenant B's user! Status: %d", resp.StatusCode)
	} else if resp.StatusCode != http.StatusNotFound && resp.StatusCode != http.StatusForbidden {
		t.Logf("Unexpected status code: %d (expected 404/403)", resp.StatusCode)
	} else {
		t.Logf("Success: Update denied with status %d", resp.StatusCode)
	}
}
