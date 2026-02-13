// Package isolation - cross-tenant list scenario
//
// Phase 1 TODO: Implement this test against real service endpoints.
package isolation

// CrossTenantListScenario verifies that list/search endpoints only return
// data belonging to the requesting tenant.
//
// Pattern:
//   1. Create Tenant A and Tenant B contexts
//   2. Insert N records for Tenant A and M records for Tenant B
//   3. List using Tenant A's context
//   4. Assert: exactly N results returned, none from Tenant B
