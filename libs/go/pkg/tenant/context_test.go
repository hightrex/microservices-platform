package tenant

import (
	"context"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
)

func TestContextOperations(t *testing.T) {
	t.Run("FromContext Empty", func(t *testing.T) {
		ctx := context.Background()
		id := FromContext(ctx)
		assert.Equal(t, uuid.Nil, id)
		assert.False(t, IsSet(ctx))
	})

	t.Run("NewContext and FromContext", func(t *testing.T) {
		expectedID := uuid.New()
		ctx := NewContext(context.Background(), expectedID)

		id := FromContext(ctx)
		assert.Equal(t, expectedID, id)
		assert.True(t, IsSet(ctx))
	})
}
