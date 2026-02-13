package tenant

import (
	"context"
	"fmt"

	"github.com/google/uuid"
)

// ErrMissingTenantID is returned when a repository operation is attempted
// without a tenant ID in the context. This prevents accidental cross-tenant
// data access.
var ErrMissingTenantID = fmt.Errorf("tenant_id is required but missing from context")

// RequireTenant extracts the tenant ID from the context and returns an error
// if it is not set. Every repository method that accesses tenant-scoped data
// MUST call this before executing any query.
//
// Usage:
//
//	func (r *UserRepo) GetByID(ctx context.Context, id uuid.UUID) (*User, error) {
//	    tenantID, err := tenant.RequireTenant(ctx)
//	    if err != nil {
//	        return nil, err
//	    }
//	    row := r.db.QueryRow(ctx,
//	        "SELECT id, email FROM users WHERE id = $1 AND tenant_id = $2",
//	        id, tenantID,
//	    )
//	    // ...
//	}
func RequireTenant(ctx context.Context) (uuid.UUID, error) {
	id := FromContext(ctx)
	if id == uuid.Nil {
		return uuid.Nil, ErrMissingTenantID
	}
	return id, nil
}

// TenantScope holds a validated tenant ID and provides helper methods
// for building tenant-scoped queries. Obtain one via NewScope().
//
// Usage:
//
//	scope, err := tenant.NewScope(ctx)
//	if err != nil {
//	    return err
//	}
//	rows, err := db.Query(ctx, scope.SQL("SELECT * FROM orders WHERE tenant_id = $1 AND status = $2"), scope.ID, "active")
type TenantScope struct {
	ID uuid.UUID
}

// NewScope creates a TenantScope from the context, returning an error if
// the tenant ID is not present. This is the preferred entry point for
// repository methods that need tenant scoping.
func NewScope(ctx context.Context) (*TenantScope, error) {
	id, err := RequireTenant(ctx)
	if err != nil {
		return nil, err
	}
	return &TenantScope{ID: id}, nil
}

// SQL is a pass-through that returns the query string unchanged.
// It exists as a documentation marker so reviewers can see that a query
// was constructed through the TenantScope pattern. Semgrep rules and
// code review can look for queries NOT going through scope.SQL().
func (s *TenantScope) SQL(query string) string {
	return query
}
