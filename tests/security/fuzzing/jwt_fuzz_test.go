package fuzzing

import (
	"crypto/rand"
	"crypto/rsa"
	"encoding/json"
	"fmt"
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/stretchr/testify/assert"
)

// Simple JWT structure for testing
type TestClaims struct {
	Sub   string   `json:"sub"`
	Tid   string   `json:"tid"`
	Roles []string `json:"roles"`
	jwt.RegisteredClaims
}

// FuzzJWTValidation helps discover edge cases in JWT parsing/validation logic.
// While we rely on the robust golang-jwt library, this tests our wrapper logic/expectations.
func FuzzJWTValidation(f *testing.F) {
	// Seed corpus with valid and semi-valid tokens
	testKey := []byte("fuzz-secret")
	
	validClaims := TestClaims{
		Sub:   "user-123",
		Tid:   "tenant-456",
		Roles: []string{"admin"},
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(1 * time.Hour)),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
			Issuer:    "auth-service",
		},
	}
	
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, validClaims)
	validTokenString, _ := token.SignedString(testKey)
	
	f.Add(validTokenString)
	f.Add("eyJhbGciOiJub25lIn0.eyJzdWIiOiIxMjM0NTY3ODkwIiwibmFtZSI6IkpvaG4gRG9lIiwiaWF0IjoxNTE2MjM5MDIyfQ.") // Alg: none
	f.Add("invalid.token.structure")
	f.Add("")

	f.Fuzz(func(t *testing.T, tokenStr string) {
		// Mimic the validation logic used in our services/gateway
		// We expect this to NOT panic, regardless of input.
		
		parsedToken, err := jwt.ParseWithClaims(tokenStr, &TestClaims{}, func(token *jwt.Token) (interface{}, error) {
			// Enforce HS256 (as Gateway does)
			if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
				return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
			}
			return testKey, nil
		})

		// If it parses successfully, it MUST have valid claims
		if err == nil && parsedToken.Valid {
			claims, ok := parsedToken.Claims.(*TestClaims)
			assert.True(t, ok, "Claims should be castable")
			assert.NotEmpty(t, claims.Sub, "Subject should not be empty in valid token")
		} else {
			// Expected failure for invalid input
			assert.Error(t, err)
		}
	})
}
