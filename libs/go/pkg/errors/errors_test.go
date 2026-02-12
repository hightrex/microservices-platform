package errors

import (
	"fmt"
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestAppError(t *testing.T) {
	baseErr := fmt.Errorf("underlying error")
	err := New(http.StatusBadRequest, "bad request", baseErr)

	assert.Equal(t, http.StatusBadRequest, err.Code)
	assert.Equal(t, "bad request", err.Message)
	assert.Equal(t, baseErr, err.Err)
	assert.Equal(t, "bad request: underlying error", err.Error())
	assert.Equal(t, baseErr, err.Unwrap())
}

func TestHelpers(t *testing.T) {
	err := fmt.Errorf("oops")

	assert.Equal(t, http.StatusBadRequest, BadRequest("msg", err).Code)
	assert.Equal(t, http.StatusUnauthorized, Unauthorized("msg", err).Code)
	assert.Equal(t, http.StatusForbidden, Forbidden("msg", err).Code)
	assert.Equal(t, http.StatusNotFound, NotFound("msg", err).Code)
	assert.Equal(t, http.StatusInternalServerError, InternalServerError("msg", err).Code)
}
