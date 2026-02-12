package errors

import (
	"fmt"
	"net/http"
)

// AppError represents a standardized application error
type AppError struct {
	Code    int         `json:"code"`
	Message string      `json:"message"`
	Details interface{} `json:"details,omitempty"`
	Err     error       `json:"-"`
}

// Error implements the error interface
func (e *AppError) Error() string {
	if e.Err != nil {
		return fmt.Sprintf("%s: %v", e.Message, e.Err)
	}
	return e.Message
}

// Unwrap returns the underlying error
func (e *AppError) Unwrap() error {
	return e.Err
}

// New creates a new AppError
func New(code int, message string, err error) *AppError {
	return &AppError{
		Code:    code,
		Message: message,
		Err:     err,
	}
}

// Wrap wraps an existing error into an AppError
func Wrap(err error, code int, message string) *AppError {
	return &AppError{
		Code:    code,
		Message: message,
		Err:     err,
	}
}

// Common error constructors

func BadRequest(message string, err error) *AppError {
	return New(http.StatusBadRequest, message, err)
}

func Unauthorized(message string, err error) *AppError {
	return New(http.StatusUnauthorized, message, err)
}

func Forbidden(message string, err error) *AppError {
	return New(http.StatusForbidden, message, err)
}

func NotFound(message string, err error) *AppError {
	return New(http.StatusNotFound, message, err)
}

func InternalServerError(message string, err error) *AppError {
	return New(http.StatusInternalServerError, message, err)
}

func ServiceUnavailable(message string, err error) *AppError {
	return New(http.StatusServiceUnavailable, message, err)
}
