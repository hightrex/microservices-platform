package httputil

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestNewClient(t *testing.T) {
	c := NewClient()
	assert.NotNil(t, c)
	assert.Equal(t, 30*time.Second, c.Timeout)
	assert.NotNil(t, c.Transport)
}
