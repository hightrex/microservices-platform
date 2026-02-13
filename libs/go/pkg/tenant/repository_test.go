package tenant

import (
	"context"
	"errors"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRequireTenant(t *testing.T) {
	t.Run("returns error when tenant missing", func(t *testing.T) {
		ctx := context.Background()
		id, err := RequireTenant(ctx)
		assert.Equal(t, uuid.Nil, id)
		assert.Error(t, err)
		assert.True(t, errors.Is(err, ErrMissingTenantID))
	})

	t.Run("returns tenant ID when present", func(t *testing.T) {
		expected := uuid.New()
		ctx := NewContext(context.Background(), expected)
		id, err := RequireTenant(ctx)
		require.NoError(t, err)
		assert.Equal(t, expected, id)
	})

	t.Run("returns error for nil UUID in context", func(t *testing.T) {
		ctx := NewContext(context.Background(), uuid.Nil)
		id, err := RequireTenant(ctx)
		assert.Equal(t, uuid.Nil, id)
		assert.Error(t, err)
	})
}

func TestNewScope(t *testing.T) {
	t.Run("creates scope with valid tenant", func(t *testing.T) {
		expected := uuid.New()
		ctx := NewContext(context.Background(), expected)
		scope, err := NewScope(ctx)
		require.NoError(t, err)
		assert.Equal(t, expected, scope.ID)
	})

	t.Run("fails without tenant", func(t *testing.T) {
		scope, err := NewScope(context.Background())
		assert.Nil(t, scope)
		assert.Error(t, err)
	})

	t.Run("SQL pass-through works", func(t *testing.T) {
		expected := uuid.New()
		ctx := NewContext(context.Background(), expected)
		scope, err := NewScope(ctx)
		require.NoError(t, err)

		query := "SELECT * FROM users WHERE tenant_id = $1 AND id = $2"
		assert.Equal(t, query, scope.SQL(query))
	})
}
