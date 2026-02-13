# Tenant Isolation Test Harness

This directory contains the test infrastructure for verifying tenant isolation across all services.

## Purpose

Tenant isolation is the **#1 security invariant** of the platform. These tests verify that:

1. **Data cannot leak across tenants** — Tenant A cannot read/modify Tenant B's data
2. **Queries are always scoped** — Every DB query includes `tenant_id` filtering
3. **API endpoints enforce boundaries** — Requests with Tenant A's token cannot access Tenant B's resources
4. **Bulk operations respect boundaries** — List/search/export operations only return data for the requesting tenant

## Directory Structure

```
tenant-isolation/
├── README.md           # This file
├── run-tests.sh        # Test runner script
├── harness.go          # Go test harness with shared setup/teardown
├── harness_test.go     # Example test using the harness
└── scenarios/
    ├── cross_tenant_read.go    # Test: Tenant A cannot read Tenant B's data
    ├── cross_tenant_write.go   # Test: Tenant A cannot modify Tenant B's data
    ├── cross_tenant_list.go    # Test: List endpoints only return own tenant's data
    └── cross_tenant_delete.go  # Test: Tenant A cannot delete Tenant B's data
```

## Running Tests

### Prerequisites
- Infrastructure running (`make infra-up`)
- Test databases seeded (`make init-db`)
- Services running (Phase 1+)

### Commands

```bash
# Run all tenant isolation tests
./tests/security/tenant-isolation/run-tests.sh

# Run via make
make test-tenant-isolation

# Run specific scenario
go test -v ./tests/security/tenant-isolation/... -run TestCrossTenantRead
```

## Test Pattern

Every tenant-isolation test follows this pattern:

1. **Setup**: Create two tenants (A and B) with separate contexts
2. **Seed**: Insert test data for both tenants
3. **Act**: Tenant A attempts to access Tenant B's data
4. **Assert**: The operation is denied or returns empty results
5. **Cleanup**: Remove test data

```go
func TestCrossTenantRead(t *testing.T) {
    h := harness.New(t)

    // Create two isolated tenants
    tenantA := h.CreateTenant("tenant-a")
    tenantB := h.CreateTenant("tenant-b")

    // Seed data for Tenant B
    h.SeedUser(tenantB, "secret-user@b.com")

    // Tenant A tries to read Tenant B's users
    users, err := h.ListUsers(tenantA)
    require.NoError(t, err)
    assert.Empty(t, users, "Tenant A should not see Tenant B's users")
}
```

## Adding New Tests

When adding a new service (Phase 1+), you MUST add tenant-isolation tests:

1. Create a new scenario file in `scenarios/`
2. Follow the Setup → Seed → Act → Assert → Cleanup pattern
3. Test all CRUD operations across tenant boundaries
4. Run the full suite to ensure no regressions
