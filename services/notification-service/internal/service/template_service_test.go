package service

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestValidateTemplateSyntax_ValidTemplates(t *testing.T) {
	tests := []struct {
		name     string
		template string
	}{
		{"simple text", "Hello World"},
		{"variable", "Hello {{.Name}}"},
		{"with function", "Hello {{upper .Name}}"},
		{"conditional", "{{if .Active}}Active{{else}}Inactive{{end}}"},
		{"range", "{{range .Items}}{{.}}{{end}}"},
		{"nested", "Hello {{.User.Name}}, welcome to {{.Org.Name}}"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validateTemplateSyntax(tt.template)
			assert.NoError(t, err)
		})
	}
}

func TestValidateTemplateSyntax_InvalidTemplates(t *testing.T) {
	tests := []struct {
		name     string
		template string
	}{
		{"unclosed action", "Hello {{.Name"},
		{"unclosed range", "{{range .Items}}{{.}}"},
		{"invalid function", "{{invalidFunc .Name}}"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validateTemplateSyntax(tt.template)
			assert.Error(t, err)
		})
	}
}

func TestRenderSafe_BasicRendering(t *testing.T) {
	data := map[string]interface{}{
		"Name":  "Alice",
		"Email": "alice@example.com",
	}

	result, err := renderSafe("Hello {{.Name}}, your email is {{.Email}}", data)
	require.NoError(t, err)
	assert.Equal(t, "Hello Alice, your email is alice@example.com", result)
}

func TestRenderSafe_WithFunctions(t *testing.T) {
	data := map[string]interface{}{
		"Name": "alice",
	}

	result, err := renderSafe("Hello {{upper .Name}}", data)
	require.NoError(t, err)
	assert.Equal(t, "Hello ALICE", result)
}

func TestRenderSafe_WithConditional(t *testing.T) {
	tests := []struct {
		name     string
		data     map[string]interface{}
		expected string
	}{
		{
			"active true",
			map[string]interface{}{"Active": true},
			"Status: Active",
		},
		{
			"active false",
			map[string]interface{}{"Active": false},
			"Status: Inactive",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := renderSafe("Status: {{if .Active}}Active{{else}}Inactive{{end}}", tt.data)
			require.NoError(t, err)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestRenderSafe_MissingData(t *testing.T) {
	data := map[string]interface{}{}

	// html/template renders missing map keys as empty string
	result, err := renderSafe("Hello {{.Name}}", data)
	require.NoError(t, err)
	assert.Equal(t, "Hello ", result)
}

func TestRenderSafe_XSSPrevention(t *testing.T) {
	data := map[string]interface{}{
		"Name": "<script>alert('xss')</script>",
	}

	// Go html/template auto-escapes HTML in {{.Name}}
	result, err := renderSafe("Hello {{.Name}}", data)
	require.NoError(t, err)
	assert.NotContains(t, result, "<script>")
	assert.Contains(t, result, "&lt;script&gt;")
}

func TestSafeFuncMap_ContainsExpectedFunctions(t *testing.T) {
	funcMap := safeFuncMap()
	assert.Contains(t, funcMap, "upper")
	assert.Contains(t, funcMap, "lower")
	assert.Contains(t, funcMap, "title")
	assert.Contains(t, funcMap, "trim")
}
