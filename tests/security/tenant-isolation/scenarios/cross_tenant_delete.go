// Package isolation - cross-tenant delete scenario
//
// Phase 1 TODO: Implement this test against real service endpoints.
package isolation

// CrossTenantDeleteScenario verifies that Tenant A cannot delete Tenant B's data.
//
// Pattern:
//   1. Create Tenant A and Tenant B contexts
//   2. Insert data into Tenant B's context
//   3. Attempt to delete using Tenant A's context
//   4. Assert: operation is denied (403/404)
//   5. Verify: Tenant B's data still exists
