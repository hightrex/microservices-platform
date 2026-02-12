package tenant

import (
	"context"

	"github.com/google/uuid"
)

type contextKey string

const (
	tenantIDKey contextKey = "tenant_id"
)

// FromContext extracts the tenant ID from the context.
// Returns uuid.Nil if not found.
func FromContext(ctx context.Context) uuid.UUID {
	id, ok := ctx.Value(tenantIDKey).(uuid.UUID)
	if !ok {
		return uuid.Nil
	}
	return id
}

// NewContext returns a new context with the given tenant ID.
func NewContext(ctx context.Context, tenantID uuid.UUID) context.Context {
	return context.WithValue(ctx, tenantIDKey, tenantID)
}

// IsSet checks if a valid tenant ID exists in the context.
func IsSet(ctx context.Context) bool {
	return FromContext(ctx) != uuid.Nil
}
