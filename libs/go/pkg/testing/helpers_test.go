package testing

import (
	"testing"

	"github.com/google/uuid"
	"github.com/hightrex/microservices-platform/libs/go/pkg/tenant"
	"github.com/stretchr/testify/assert"
)

func TestTestContext(t *testing.T) {
	ctx := TestContext(t)
	assert.True(t, tenant.IsSet(ctx))
}

func TestTestContextWithTenantID(t *testing.T) {
	id := uuid.New()
	ctx := TestContextWithTenantID(t, id)
	assert.Equal(t, id, tenant.FromContext(ctx))
}
