// Package isolation provides the test harness for tenant isolation testing.
// It creates isolated tenant contexts and provides helpers for verifying
// that data does not leak across tenant boundaries.
//
// This is the Phase 0 skeleton. Phase 1 will add real service-specific tests.
package isolation

import (
	"context"
	"testing"

	"github.com/google/uuid"
	"github.com/hightrex/microservices-platform/libs/go/pkg/tenant"
)

// Harness manages isolated tenant contexts for testing.
// Phase 1 will extend this with database connections and HTTP clients.
type Harness struct {
	t       *testing.T
	tenants map[string]*TenantCtx
}

// TenantCtx represents an isolated tenant testing context.
type TenantCtx struct {
	ID   uuid.UUID
	Name string
	Ctx  context.Context
}

// New creates a new test harness.
func New(t *testing.T) *Harness {
	t.Helper()
	return &Harness{
		t:       t,
		tenants: make(map[string]*TenantCtx),
	}
}

// CreateTenant creates a new isolated tenant context for testing.
// Returns a TenantCtx that can be used to make API calls or DB queries
// scoped to that tenant.
func (h *Harness) CreateTenant(name string) *TenantCtx {
	h.t.Helper()

	id := uuid.New()
	ctx := tenant.NewContext(context.Background(), id)

	tc := &TenantCtx{
		ID:   id,
		Name: name,
		Ctx:  ctx,
	}
	h.tenants[name] = tc

	h.t.Logf("Created test tenant: name=%s id=%s", name, id.String())
	return tc
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
