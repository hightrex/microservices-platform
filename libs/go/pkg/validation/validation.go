// Package validation provides shared input validation patterns for the
// microservices platform. It wraps go-playground/validator with standardized
// error formatting so every service returns consistent validation errors.
//
// Phase 0 establishes the patterns; Phase 1 services use them on every endpoint.
//
// Usage:
//
//	type CreateUserRequest struct {
//	    Email     string `json:"email"     validate:"required,email"`
//	    FirstName string `json:"first_name" validate:"required,min=1,max=100"`
//	    LastName  string `json:"last_name"  validate:"required,min=1,max=100"`
//	}
//
//	func (h *Handler) CreateUser(c *gin.Context) {
//	    var req CreateUserRequest
//	    if err := c.ShouldBindJSON(&req); err != nil {
//	        c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
//	        return
//	    }
//	    if errs := validation.Validate(req); errs != nil {
//	        c.JSON(http.StatusBadRequest, validation.ErrorResponse(errs))
//	        return
//	    }
//	    // ... business logic
//	}
package validation

import (
	"fmt"
	"net/http"
	"strings"
	"sync"

	"github.com/go-playground/validator/v10"
)

// validate is the singleton validator instance.
var (
	validate *validator.Validate
	once     sync.Once
)

// getValidator returns the singleton validator, initializing it lazily.
func getValidator() *validator.Validate {
	once.Do(func() {
		validate = validator.New(validator.WithRequiredStructEnabled())
		// Register custom validators here as the platform grows.
		_ = validate.RegisterValidation("slug", slugValidator)
	})
	return validate
}

// slugValidator checks if a string is a valid URL-friendly slug.
func slugValidator(fl validator.FieldLevel) bool {
	slug := fl.Field().String()
	if slug == "" {
		return true // skip empty, use 'required' tag for that
	}
	// Regex: lowercase letters, numbers, and hyphens, starting/ending with letter/number
	for _, r := range slug {
		if (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9') || r == '-' {
			continue
		}
		return false
	}
	if slug[0] == '-' || slug[len(slug)-1] == '-' {
		return false
	}
	return true
}

// FieldError represents a single field validation error in a standardized format.
type FieldError struct {
	Field   string `json:"field"`
	Message string `json:"message"`
	Tag     string `json:"tag"`
	Value   string `json:"value,omitempty"`
}

// ValidationError is the top-level validation error response.
type ValidationError struct {
	Code    int          `json:"code"`
	Error   string       `json:"error"`
	Details []FieldError `json:"details"`
}

// Validate checks a struct against its validate tags and returns a slice of
// FieldError if validation fails, or nil if the input is valid.
func Validate(s interface{}) []FieldError {
	err := getValidator().Struct(s)
	if err == nil {
		return nil
	}

	validationErrors, ok := err.(validator.ValidationErrors)
	if !ok {
		// Non-validation error (e.g. nil pointer)
		return []FieldError{{
			Field:   "unknown",
			Message: err.Error(),
			Tag:     "internal",
		}}
	}

	fieldErrors := make([]FieldError, 0, len(validationErrors))
	for _, ve := range validationErrors {
		fieldErrors = append(fieldErrors, FieldError{
			Field:   toJSONFieldName(ve.Field()),
			Message: buildMessage(ve),
			Tag:     ve.Tag(),
			Value:   fmt.Sprintf("%v", ve.Value()),
		})
	}
	return fieldErrors
}

// ErrorResponse creates a standardized validation error response body.
func ErrorResponse(errs []FieldError) ValidationError {
	return ValidationError{
		Code:    http.StatusBadRequest,
		Error:   "validation_failed",
		Details: errs,
	}
}

// buildMessage generates a human-readable message from a validator.FieldError.
func buildMessage(fe validator.FieldError) string {
	field := toJSONFieldName(fe.Field())

	switch fe.Tag() {
	case "required":
		return fmt.Sprintf("%s is required", field)
	case "email":
		return fmt.Sprintf("%s must be a valid email address", field)
	case "min":
		return fmt.Sprintf("%s must be at least %s characters", field, fe.Param())
	case "max":
		return fmt.Sprintf("%s must be at most %s characters", field, fe.Param())
	case "uuid":
		return fmt.Sprintf("%s must be a valid UUID", field)
	case "oneof":
		return fmt.Sprintf("%s must be one of: %s", field, fe.Param())
	case "url":
		return fmt.Sprintf("%s must be a valid URL", field)
	case "gte":
		return fmt.Sprintf("%s must be greater than or equal to %s", field, fe.Param())
	case "lte":
		return fmt.Sprintf("%s must be less than or equal to %s", field, fe.Param())
	case "alphanum":
		return fmt.Sprintf("%s must contain only alphanumeric characters", field)
	default:
		return fmt.Sprintf("%s failed validation: %s", field, fe.Tag())
	}
}

// toJSONFieldName converts PascalCase Go struct field names to snake_case
// for JSON-friendly error messages.
func toJSONFieldName(field string) string {
	var result strings.Builder
	for i, r := range field {
		if r >= 'A' && r <= 'Z' {
			if i > 0 {
				result.WriteRune('_')
			}
			result.WriteRune(r + 32) // to lowercase
		} else {
			result.WriteRune(r)
		}
	}
	return result.String()
}
