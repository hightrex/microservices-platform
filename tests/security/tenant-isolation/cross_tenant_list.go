package isolation

import (
	"encoding/json"
	"net/http"
	"testing"
)

// CrossTenantListScenario verifies that Tenant A's list endpoints return only Tenant A's data.
func CrossTenantListScenario(t *testing.T, h *Harness) {
	t.Log("Starting CrossTenantListScenario")

	// 1. Setup Tenants
	tenantA := h.CreateTenant("tenantA_list")
	tenantB := h.CreateTenant("tenantB_list")

	// 2. Tenant A lists users
	// GET /api/v1/users
	url := GatewayURL + "/users"
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

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("failed to list users: status %d", resp.StatusCode)
	}

	var res struct {
		Data struct {
			Items []User `json:"items"`
		} `json:"data"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&res); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	// 3. Assert Tenant B's user is NOT in the list
	foundB := false
	for _, u := range res.Data.Items {
		if u.ID == tenantB.User.ID {
			foundB = true
			break
		}
	}

	if foundB {
		t.Errorf("Security Violation: Tenant B's user %s found in Tenant A's list results!", tenantB.User.ID)
	} else {
		t.Log("Success: Tenant B's user not visible to Tenant A")
	}
}
