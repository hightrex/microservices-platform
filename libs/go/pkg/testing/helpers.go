package testing

import (
	"context"
	"testing"

	"github.com/google/uuid"
	"github.com/hightrex/microservices-platform/libs/go/pkg/tenant"
)

// TestContext returns a context with a tenant ID for testing
func TestContext(t *testing.T) context.Context {
	return tenant.NewContext(context.Background(), uuid.New())
}

// TestContextWithTenantID returns a context with a specific tenant ID for testing
func TestContextWithTenantID(t *testing.T, tenantID uuid.UUID) context.Context {
	return tenant.NewContext(context.Background(), tenantID)
}
