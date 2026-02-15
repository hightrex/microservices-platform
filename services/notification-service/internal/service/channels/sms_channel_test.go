package channels

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestE164Validation(t *testing.T) {
	tests := []struct {
		name    string
		phone   string
		isValid bool
	}{
		{"valid US", "+15551234567", true},
		{"valid UK", "+447911123456", true},
		{"valid short", "+1234", true},
		{"missing plus", "15551234567", false},
		{"too long", "+123456789012345678", false},
		{"with spaces", "+1 555 123 4567", false},
		{"with dashes", "+1-555-123-4567", false},
		{"empty", "", false},
		{"just plus", "+", false},
		{"starts with 0", "+05551234567", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			valid := e164Regex.MatchString(tt.phone)
			assert.Equal(t, tt.isValid, valid)
		})
	}
}

func TestMaskPhone(t *testing.T) {
	tests := []struct {
		name     string
		phone    string
		expected string
	}{
		{"normal phone", "+15551234567", "********4567"},
		{"exactly 4", "1234", "****"},
		{"very short", "12", "****"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.expected, maskPhone(tt.phone))
		})
	}
}
