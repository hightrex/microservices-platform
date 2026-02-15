package service

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestValidateWebhookURL_ValidSchemes(t *testing.T) {
	// NOTE: DNS resolution may fail in CI/local environments for external domains.
	// These tests focus on the scheme and localhost validation which don't require DNS.
	tests := []struct {
		name string
		url  string
	}{
		{"https scheme", "https://8.8.8.8/webhook"},
		{"http scheme", "http://8.8.8.8/webhook"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validateWebhookURL(tt.url)
			assert.NoError(t, err)
		})
	}
}

func TestValidateWebhookURL_BlocksLocalhost(t *testing.T) {
	tests := []struct {
		name string
		url  string
	}{
		{"localhost", "http://localhost/webhook"},
		{"127.0.0.1", "http://127.0.0.1/webhook"},
		{"::1", "http://[::1]/webhook"},
		{"0.0.0.0", "http://0.0.0.0/webhook"},
		{"localhost with port", "http://localhost:8080/webhook"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validateWebhookURL(tt.url)
			assert.Error(t, err)
			assert.Contains(t, err.Error(), "localhost")
		})
	}
}

func TestValidateWebhookURL_BlocksInvalidSchemes(t *testing.T) {
	tests := []struct {
		name string
		url  string
	}{
		{"ftp", "ftp://example.com/file"},
		{"file", "file:///etc/passwd"},
		{"gopher", "gopher://example.com"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validateWebhookURL(tt.url)
			assert.Error(t, err)
			assert.Contains(t, err.Error(), "http or https")
		})
	}
}

func TestValidateWebhookURL_BlocksInvalidURL(t *testing.T) {
	// Missing scheme
	err := validateWebhookURL("not-a-url")
	assert.Error(t, err)

	// Empty
	err = validateWebhookURL("")
	assert.Error(t, err)
}

func TestStringPtrIfNotEmpty(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected *string
	}{
		{"non-empty", "hello", strPtr("hello")},
		{"empty", "", nil},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := stringPtrIfNotEmpty(tt.input)
			if tt.expected == nil {
				assert.Nil(t, result)
			} else {
				assert.Equal(t, *tt.expected, *result)
			}
		})
	}
}

func strPtr(s string) *string {
	return &s
}
