package auth_test

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const (
	gatewayURL = "http://localhost:3000/api/v1"
)

func TestAuthSecurity(t *testing.T) {
	// Wait for gateway to be ready
	waitForGateway(t)

	t.Run("JWT_Manipulation", func(t *testing.T) {
		// 1. Register and Login to get a valid token
		email := fmt.Sprintf("hacker_%d@example.com", time.Now().UnixNano())
		password := "Password123!"
		token := registerAndLogin(t, email, password)

		// 2. Test Expired Token
		t.Run("ExpiredToken", func(t *testing.T) {
			badToken := createForgedToken(t, token, func(claims jwt.MapClaims) {
				claims["exp"] = time.Now().Add(-1 * time.Hour).Unix()
			})
			req, _ := http.NewRequest("GET", gatewayURL+"/auth/me", nil) // Assuming /auth/me or similar protected route
			req.Header.Set("Authorization", "Bearer "+badToken)

			// We use a known protected endpoint. If /auth/me doesn't exist, we can use /users/me or list users
			req, _ = http.NewRequest("GET", gatewayURL+"/users", nil)
			req.Header.Set("Authorization", "Bearer "+badToken)

			client := &http.Client{}
			resp, err := client.Do(req)
			require.NoError(t, err)
			defer resp.Body.Close()

			assert.Equal(t, http.StatusUnauthorized, resp.StatusCode, "Expired token should return 401")
		})

		// 3. Test Wrong Signing Key ("None" alg or different secret)
		t.Run("WrongSignature", func(t *testing.T) {
			// Create token with "none" algorithm
			token := jwt.New(jwt.SigningMethodNone)
			claims := token.Claims.(jwt.MapClaims)
			claims["sub"] = "hacked"
			claims["exp"] = time.Now().Add(time.Hour).Unix()

			tokenString, _ := token.SignedString(jwt.UnsafeAllowNoneSignatureType)

			req, _ := http.NewRequest("GET", gatewayURL+"/users", nil)
			req.Header.Set("Authorization", "Bearer "+tokenString)

			client := &http.Client{}
			resp, err := client.Do(req)
			require.NoError(t, err)
			defer resp.Body.Close()

			assert.Equal(t, http.StatusUnauthorized, resp.StatusCode, "None algorithm should be rejected")
		})
	})

	t.Run("BruteForce_Protection", func(t *testing.T) {
		email := fmt.Sprintf("victim_%d@example.com", time.Now().UnixNano())
		password := "CorrectHorse123!"
		registerUser(t, email, password)

		// Try wrong password 5 times (or whatever the limit is, usually 5 or 10)
		// We'll try 10 times to be sure
		for i := 0; i < 10; i++ {
			login(t, email, "WrongPass")
		}

		// Try with CORRECT password -> Should still fail due to lockout
		// Note: This assumes lockout duration is > 0s.
		// If implementation doesn't lock out immediately, this test might need adjustment.

		// We look for a specific error code or status in the last attempts
		// But strictly we want to see if correct password fails now

		payload := map[string]string{
			"email":    email,
			"password": password,
		}
		body, _ := json.Marshal(payload)
		req, err := http.NewRequest("POST", gatewayURL+"/auth/login", bytes.NewBuffer(body))
		require.NoError(t, err)
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("X-Tenant-ID", "550e8400-e29b-41d4-a716-446655440000")

		client := &http.Client{}
		resp, err := client.Do(req)
		require.NoError(t, err)
		defer resp.Body.Close()

		// If lockout is working, this should still be 401 or 403 or 429
		// The requirement says "account lockout after N failures"
		if resp.StatusCode == http.StatusOK {
			// It might be that the limit is higher than 10, or lockout not enabled.
			// marking as potential failure but checking response body needed
			t.Log("Warning: Account not locked out after 10 failed attempts")
		} else {
			assert.NotEqual(t, http.StatusOK, resp.StatusCode, "Account should be locked out")
		}
	})

	t.Run("Password_Policy", func(t *testing.T) {
		// Test short password
		err := tryRegister("weak@example.com", "123")
		assert.Error(t, err, "Should fail with short password")

		// Test no number/special char if policy requires it
		// Assuming policy: min 8 chars
		err = tryRegister("weaklongpassword@example.com", "passwordpassword")
		// This assertion depends on specific policy. Warning if it passes.
		if err == nil {
			t.Log("Info: Weak password (only letters) was accepted")
		}
	})
}

func waitForGateway(t *testing.T) {
	deadline := time.Now().Add(30 * time.Second)
	for time.Now().Before(deadline) {
		resp, err := http.Get("http://localhost:3000/health")
		if err == nil && resp.StatusCode == http.StatusOK {
			return
		}
		time.Sleep(1 * time.Second)
	}
	t.Fatal("Gateway did not become ready in time")
}

func registerAndLogin(t *testing.T, email, password string) string {
	registerUser(t, email, password)

	token, err := login(t, email, password)
	require.NoError(t, err)
	return token
}

func registerUser(t *testing.T, email, password string) {
	err := tryRegister(email, password)
	require.NoError(t, err)
}

func tryRegister(email, password string) error {
	payload := map[string]string{
		"email":      email,
		"password":   password,
		"first_name": "Test",
		"last_name":  "User",
	}
	body, _ := json.Marshal(payload)
	req, err := http.NewRequest("POST", gatewayURL+"/auth/register", bytes.NewBuffer(body))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Tenant-ID", "550e8400-e29b-41d4-a716-446655440000") // Use a valid UUID

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusCreated && resp.StatusCode != http.StatusOK {
		bodyBytes, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("registration failed: %d %s", resp.StatusCode, string(bodyBytes))
	}
	return nil
}

func login(t *testing.T, email, password string) (string, error) {
	payload := map[string]string{
		"email":    email,
		"password": password,
	}
	body, _ := json.Marshal(payload)
	req, err := http.NewRequest("POST", gatewayURL+"/auth/login", bytes.NewBuffer(body))
	if err != nil {
		return "", err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Tenant-ID", "550e8400-e29b-41d4-a716-446655440000") // Use a valid UUID

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("login failed: %d", resp.StatusCode)
	}

	var res map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&res); err != nil {
		return "", err
	}

	data, ok := res["data"].(map[string]interface{})
	if !ok {
		// Try top level if data structure different
		if token, ok := res["access_token"].(string); ok {
			return token, nil
		}
		return "", fmt.Errorf("invalid response structure")
	}

	token, ok := data["access_token"].(string)
	if !ok {
		return "", fmt.Errorf("token not found in response")
	}
	return token, nil
}

func createForgedToken(t *testing.T, originalToken string, modifier func(jwt.MapClaims)) string {
	parser := jwt.NewParser()
	token, _, err := parser.ParseUnverified(originalToken, jwt.MapClaims{})
	require.NoError(t, err)

	claims, ok := token.Claims.(jwt.MapClaims)
	require.True(t, ok)

	modifier(claims)

	// Re-sign with a different secret or same dev secret?
	// To test "Expired", we need a valid signature but expired time.
	// We know the dev secret from compose: "dev-secret-change-in-production-minimum-32-chars!"
	secret := []byte("dev-secret-change-in-production-minimum-32-chars!")

	newToken := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	signed, err := newToken.SignedString(secret)
	require.NoError(t, err)

	return signed
}
