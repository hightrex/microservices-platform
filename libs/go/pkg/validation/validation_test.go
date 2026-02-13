package validation

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type testCreateUser struct {
	Email     string `json:"email"      validate:"required,email"`
	FirstName string `json:"first_name" validate:"required,min=1,max=100"`
	LastName  string `json:"last_name"  validate:"required,min=1,max=100"`
	Role      string `json:"role"       validate:"required,oneof=admin member viewer"`
}

func TestValidate_ValidInput(t *testing.T) {
	req := testCreateUser{
		Email:     "alice@example.com",
		FirstName: "Alice",
		LastName:  "Smith",
		Role:      "admin",
	}
	errs := Validate(req)
	assert.Nil(t, errs)
}

func TestValidate_MissingRequired(t *testing.T) {
	req := testCreateUser{} // all empty
	errs := Validate(req)
	require.NotNil(t, errs)
	assert.Len(t, errs, 4) // Email, FirstName, LastName, Role

	// Check that all are "required" errors
	for _, e := range errs {
		assert.Equal(t, "required", e.Tag)
		assert.Contains(t, e.Message, "is required")
	}
}

func TestValidate_InvalidEmail(t *testing.T) {
	req := testCreateUser{
		Email:     "not-an-email",
		FirstName: "Alice",
		LastName:  "Smith",
		Role:      "admin",
	}
	errs := Validate(req)
	require.NotNil(t, errs)
	assert.Len(t, errs, 1)
	assert.Equal(t, "email", errs[0].Field)
	assert.Equal(t, "email", errs[0].Tag)
	assert.Contains(t, errs[0].Message, "valid email")
}

func TestValidate_InvalidOneof(t *testing.T) {
	req := testCreateUser{
		Email:     "alice@example.com",
		FirstName: "Alice",
		LastName:  "Smith",
		Role:      "superadmin", // not in oneof
	}
	errs := Validate(req)
	require.NotNil(t, errs)
	assert.Len(t, errs, 1)
	assert.Equal(t, "role", errs[0].Field)
	assert.Equal(t, "oneof", errs[0].Tag)
}

func TestValidate_MaxLength(t *testing.T) {
	req := testCreateUser{
		Email:     "alice@example.com",
		FirstName: string(make([]byte, 101)), // 101 chars, over max=100
		LastName:  "Smith",
		Role:      "admin",
	}
	errs := Validate(req)
	require.NotNil(t, errs)
	assert.Len(t, errs, 1)
	assert.Equal(t, "first_name", errs[0].Field)
	assert.Equal(t, "max", errs[0].Tag)
}

func TestErrorResponse(t *testing.T) {
	errs := []FieldError{
		{Field: "email", Message: "email is required", Tag: "required"},
	}
	resp := ErrorResponse(errs)
	assert.Equal(t, 400, resp.Code)
	assert.Equal(t, "validation_failed", resp.Error)
	assert.Len(t, resp.Details, 1)
}

func TestToJSONFieldName(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"FirstName", "first_name"},
		{"Email", "email"},
		{"ID", "i_d"}, // simple conversion, not perfect but consistent
		{"OrgID", "org_i_d"},
	}
	for _, tt := range tests {
		assert.Equal(t, tt.expected, toJSONFieldName(tt.input))
	}
}
