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

// TestCrossTenantContextIsolation is a Phase 0 skeleton test demonstrating
// the pattern for cross-tenant isolation testing. Phase 1 will add
// real database and API assertions here.
func TestCrossTenantContextIsolation(t *testing.T) {
	h := New(t)

	tenantA := h.CreateTenant("org-alpha")
	tenantB := h.CreateTenant("org-beta")

	// Verify contexts are isolated
	idFromA := tenant.FromContext(tenantA.Ctx)
	idFromB := tenant.FromContext(tenantB.Ctx)

	assert.Equal(t, tenantA.ID, idFromA, "Tenant A context should contain Tenant A's ID")
	assert.Equal(t, tenantB.ID, idFromB, "Tenant B context should contain Tenant B's ID")
	assert.NotEqual(t, idFromA, idFromB, "Tenant contexts should be isolated")

	// Phase 1 TODO: Add real cross-tenant data access tests:
	// - Create a user in Tenant B's context
	// - Try to list users in Tenant A's context
	// - Assert Tenant B's user is NOT visible
	t.Log("Phase 0: Context isolation verified. Phase 1 will add service-level tests.")
}
