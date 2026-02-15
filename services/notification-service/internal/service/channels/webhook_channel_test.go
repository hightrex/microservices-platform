package channels

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestGenerateHMACSignature(t *testing.T) {
	payload := []byte(`{"test": "data"}`)
	secret := "test-secret-key"

	sig := generateHMACSignature(payload, secret)

	// Verify format
	assert.Contains(t, sig, "sha256=")
	assert.Greater(t, len(sig), len("sha256="))

	// Same input produces same output (deterministic)
	sig2 := generateHMACSignature(payload, secret)
	assert.Equal(t, sig, sig2)

	// Different secret produces different signature
	sig3 := generateHMACSignature(payload, "different-secret")
	assert.NotEqual(t, sig, sig3)

	// Different payload produces different signature
	sig4 := generateHMACSignature([]byte(`{"different": "data"}`), secret)
	assert.NotEqual(t, sig, sig4)
}
