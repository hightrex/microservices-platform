package service

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/hightrex/microservices-platform/libs/go/pkg/tenant"
	"github.com/hightrex/microservices-platform/services/auth-service/internal/config"
	"github.com/hightrex/microservices-platform/services/auth-service/internal/models"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func newTestAuthService() (*AuthService, *mockUserRepo, *mockSessionRepo, *mockEventPublisher) {
	users := newMockUserRepo()
	sessions := newMockSessionRepo()
	roles := newMockRoleRepo()
	pwHistory := newMockPasswordHistoryRepo()
	tokenCacheMock := newMockTokenCache()
	events := newMockEventPublisher()
	tokenSvc := newTestTokenService()

	svc := NewAuthService(
		users, sessions, roles, pwHistory, tokenCacheMock,
		tokenSvc, events,
		config.SecurityConfig{
			BcryptCost:           4, // low cost for test speed
			MaxFailedLogins:      5,
			LockoutDurationMins:  30,
			PasswordMinLength:    8,
			PasswordHistoryCount: 5,
		},
		config.MFAConfig{Issuer: "TestApp"},
	)
	return svc, users, sessions, events
}

func testContext() context.Context {
	return tenant.NewContext(context.Background(), uuid.New())
}

func TestRegister_Success(t *testing.T) {
	svc, users, _, events := newTestAuthService()
	ctx := testContext()

	resp, err := svc.Register(ctx, models.CreateUserRequest{
		Email:     "user@example.com",
		Password:  "securepassword123",
		FirstName: "John",
		LastName:  "Doe",
	}, "127.0.0.1")

	require.NoError(t, err)
	require.NotNil(t, resp)
	assert.Equal(t, "user@example.com", resp.Email)
	assert.Equal(t, "John", resp.FirstName)
	assert.Equal(t, "Doe", resp.LastName)
	assert.Len(t, users.users, 1)

	// Verify event was published
	lastEvt := events.lastEvent()
	require.NotNil(t, lastEvt)
	assert.Equal(t, "user.created", lastEvt.EventType)
}

func TestRegister_ValidationFailure(t *testing.T) {
	svc, _, _, _ := newTestAuthService()
	ctx := testContext()

	// Missing required fields
	_, err := svc.Register(ctx, models.CreateUserRequest{
		Email: "bad",
	}, "127.0.0.1")
	assert.Error(t, err)
}

func TestRegister_ShortPassword(t *testing.T) {
	svc, _, _, _ := newTestAuthService()
	ctx := testContext()

	_, err := svc.Register(ctx, models.CreateUserRequest{
		Email:     "user@example.com",
		Password:  "short",
		FirstName: "John",
		LastName:  "Doe",
	}, "127.0.0.1")
	assert.Error(t, err)
}

func TestLogin_Success(t *testing.T) {
	svc, _, _, events := newTestAuthService()
	ctx := testContext()

	// Register first
	_, err := svc.Register(ctx, models.CreateUserRequest{
		Email:     "login@example.com",
		Password:  "securepassword123",
		FirstName: "Jane",
		LastName:  "Doe",
	}, "127.0.0.1")
	require.NoError(t, err)

	// Login
	resp, err := svc.Login(ctx, models.LoginRequest{
		Email:    "login@example.com",
		Password: "securepassword123",
	}, "127.0.0.1", "TestAgent/1.0")
	require.NoError(t, err)
	require.NotNil(t, resp)
	assert.NotEmpty(t, resp.AccessToken)
	assert.NotEmpty(t, resp.RefreshToken)
	assert.Equal(t, "Bearer", resp.TokenType)
	assert.Greater(t, resp.ExpiresIn, int64(0))

	lastEvt := events.lastEvent()
	require.NotNil(t, lastEvt)
	assert.Equal(t, "user.login", lastEvt.EventType)
}

func TestLogin_InvalidPassword(t *testing.T) {
	svc, _, _, _ := newTestAuthService()
	ctx := testContext()

	_, err := svc.Register(ctx, models.CreateUserRequest{
		Email:     "test@example.com",
		Password:  "correctpassword",
		FirstName: "Test",
		LastName:  "User",
	}, "127.0.0.1")
	require.NoError(t, err)

	_, err = svc.Login(ctx, models.LoginRequest{
		Email:    "test@example.com",
		Password: "wrongpassword",
	}, "127.0.0.1", "TestAgent")
	assert.Error(t, err)
}

func TestLogin_NonExistentUser(t *testing.T) {
	svc, _, _, _ := newTestAuthService()
	ctx := testContext()

	_, err := svc.Login(ctx, models.LoginRequest{
		Email:    "nobody@example.com",
		Password: "password123",
	}, "127.0.0.1", "TestAgent")
	assert.Error(t, err)
}

func TestLogin_DisabledAccount(t *testing.T) {
	svc, users, _, _ := newTestAuthService()
	ctx := testContext()

	_, err := svc.Register(ctx, models.CreateUserRequest{
		Email:     "disabled@example.com",
		Password:  "securepassword123",
		FirstName: "Disabled",
		LastName:  "User",
	}, "127.0.0.1")
	require.NoError(t, err)

	// Disable the user
	for _, u := range users.users {
		u.Status = models.UserStatusDisabled
	}

	_, err = svc.Login(ctx, models.LoginRequest{
		Email:    "disabled@example.com",
		Password: "securepassword123",
	}, "127.0.0.1", "TestAgent")
	assert.Error(t, err)
}

func TestLogin_LockedAccount(t *testing.T) {
	svc, users, _, _ := newTestAuthService()
	ctx := testContext()

	_, err := svc.Register(ctx, models.CreateUserRequest{
		Email:     "locked@example.com",
		Password:  "securepassword123",
		FirstName: "Locked",
		LastName:  "User",
	}, "127.0.0.1")
	require.NoError(t, err)

	// Lock the user
	lockUntil := time.Now().Add(30 * time.Minute)
	for _, u := range users.users {
		u.Status = models.UserStatusLocked
		u.LockedUntil = &lockUntil
	}

	_, err = svc.Login(ctx, models.LoginRequest{
		Email:    "locked@example.com",
		Password: "securepassword123",
	}, "127.0.0.1", "TestAgent")
	assert.Error(t, err)
}

func TestLogout(t *testing.T) {
	svc, _, _, events := newTestAuthService()
	ctx := testContext()

	err := svc.Logout(ctx, "session-123", "token-id-abc", 300, "127.0.0.1")
	require.NoError(t, err)

	lastEvt := events.lastEvent()
	require.NotNil(t, lastEvt)
	assert.Equal(t, "user.logout", lastEvt.EventType)
}

func TestSetupMFA(t *testing.T) {
	svc, _, _, _ := newTestAuthService()
	ctx := testContext()

	resp, err := svc.SetupMFA(ctx, uuid.New().String(), "127.0.0.1")
	require.NoError(t, err)
	require.NotNil(t, resp)
	assert.NotEmpty(t, resp.Secret)
	assert.NotEmpty(t, resp.QRCodeURL)
	assert.Contains(t, resp.QRCodeURL, "otpauth://totp/")
	assert.Equal(t, "TestApp", resp.Issuer)
}

func TestRefreshToken_InvalidToken(t *testing.T) {
	svc, _, _, _ := newTestAuthService()
	ctx := testContext()

	_, err := svc.RefreshToken(ctx, "nonexistent-refresh-token", "127.0.0.1", "TestAgent")
	assert.Error(t, err)
}
