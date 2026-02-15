package isolation

import (
	"testing"

	"github.com/hightrex/microservices-platform/libs/go/pkg/tenant"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestHarnessCreatesTenants verifies the harness correctly creates isolated tenants.
func TestHarnessCreatesTenants(t *testing.T) {
	h := New(t)

	tenantA := h.CreateTenant("tenant-a")
	tenantB := h.CreateTenant("tenant-b")

	// Tenants should have different IDs
	h.AssertDifferentTenants(tenantA, tenantB)

	// Each context should have the correct tenant ID
	idA := tenant.FromContext(tenantA.Ctx)
	idB := tenant.FromContext(tenantB.Ctx)
	assert.Equal(t, tenantA.ID, idA)
	assert.Equal(t, tenantB.ID, idB)
	assert.NotEqual(t, idA, idB)
}

// TestHarnessGetTenant verifies tenant retrieval by name.
func TestHarnessGetTenant(t *testing.T) {
	h := New(t)

	created := h.CreateTenant("my-org")
	retrieved := h.GetTenant("my-org")
	assert.Equal(t, created.ID, retrieved.ID)
}

// TestHarnessRequireTenant verifies context tenant validation.
func TestHarnessRequireTenant(t *testing.T) {
	h := New(t)
	tc := h.CreateTenant("test-org")
	id := h.RequireTenantInContext(tc.Ctx)
	require.Equal(t, tc.ID, id)
}

// TestCrossTenantContextIsolation verified basic context isolation.
// Now we run the real service-level scenarios.
func TestTenantIsolationScenarios(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	h := New(t)

	// Run scenarios as subtests
	t.Run("CrossTenantRead", func(t *testing.T) {
		CrossTenantReadScenario(t, h)
	})
	t.Run("CrossTenantWrite", func(t *testing.T) {
		CrossTenantWriteScenario(t, h)
	})
	t.Run("CrossTenantList", func(t *testing.T) {
		CrossTenantListScenario(t, h)
	})
	t.Run("CrossTenantDelete", func(t *testing.T) {
		CrossTenantDeleteScenario(t, h)
	})
}
