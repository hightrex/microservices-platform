// Package isolation - cross-tenant write scenario
//
// Phase 1 TODO: Implement this test against real service endpoints.
package isolation

// CrossTenantWriteScenario verifies that Tenant A cannot modify Tenant B's data.
//
// Pattern:
//   1. Create Tenant A and Tenant B contexts
//   2. Insert data into Tenant B's context
//   3. Attempt to update/create using Tenant A's context targeting Tenant B's data
//   4. Assert: operation is denied (403) or data is unchanged
