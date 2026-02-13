package service

import (
	"context"
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha1"
	"encoding/base32"
	"fmt"
	"net/url"
	"strings"
	"time"

	"golang.org/x/crypto/bcrypt"

	"github.com/google/uuid"
	"github.com/hightrex/microservices-platform/libs/go/pkg/errors"
	"github.com/hightrex/microservices-platform/libs/go/pkg/logger"
	"github.com/hightrex/microservices-platform/libs/go/pkg/securitylog"
	"github.com/hightrex/microservices-platform/libs/go/pkg/validation"
	"github.com/hightrex/microservices-platform/services/auth-service/internal/config"
	"github.com/hightrex/microservices-platform/services/auth-service/internal/models"
)

const (
	authEventsStream = "auth-events"
)

// AuthService handles authentication and registration business logic.
type AuthService struct {
	users       UserRepository
	sessions    SessionRepository
	roles       RoleRepository
	pwHistory   PasswordHistoryRepository
	tokenCache  TokenCache
	tokens      *TokenService
	events      EventPublisher
	securityCfg config.SecurityConfig
	mfaCfg      config.MFAConfig
}

// NewAuthService creates a new AuthService.
func NewAuthService(
	users UserRepository,
	sessions SessionRepository,
	roles RoleRepository,
	pwHistory PasswordHistoryRepository,
	tokenCache TokenCache,
	tokens *TokenService,
	events EventPublisher,
	securityCfg config.SecurityConfig,
	mfaCfg config.MFAConfig,
) *AuthService {
	return &AuthService{
		users:       users,
		sessions:    sessions,
		roles:       roles,
		pwHistory:   pwHistory,
		tokenCache:  tokenCache,
		tokens:      tokens,
		events:      events,
		securityCfg: securityCfg,
		mfaCfg:      mfaCfg,
	}
}

// Register creates a new user account within the tenant.
func (s *AuthService) Register(ctx context.Context, req models.CreateUserRequest, ip string) (*models.UserResponse, error) {
	if errs := validation.Validate(req); errs != nil {
		return nil, errors.BadRequest("Validation failed", fmt.Errorf("%v", validation.ErrorResponse(errs)))
	}

	// Enforce password minimum length from security config
	if len(req.Password) < s.securityCfg.PasswordMinLength {
		return nil, errors.BadRequest(
			fmt.Sprintf("Password must be at least %d characters", s.securityCfg.PasswordMinLength), nil)
	}

	// Normalize email to lowercase
	req.Email = strings.ToLower(req.Email)

	// Hash password
	hash, err := bcrypt.GenerateFromPassword([]byte(req.Password), s.securityCfg.BcryptCost)
	if err != nil {
		return nil, errors.InternalServerError("Failed to hash password", err)
	}

	user := &models.User{
		Email:        req.Email,
		PasswordHash: string(hash),
		FirstName:    req.FirstName,
		LastName:     req.LastName,
		Status:       models.UserStatusActive,
	}

	if err := s.users.Create(ctx, user); err != nil {
		return nil, err
	}

	// Store initial password in history
	if err := s.pwHistory.Add(ctx, user.ID, user.PasswordHash); err != nil {
		logger.Warn().Err(err).Msg("Failed to store initial password in history")
	}

	// Assign default "member" role
	role, err := s.roles.GetByName(ctx, models.RoleMember)
	if err != nil {
		logger.Warn().Err(err).Msg("Default 'member' role not found, skipping role assignment")
	} else {
		if err := s.roles.AssignRole(ctx, user.ID, role.ID, nil); err != nil {
			logger.Warn().Err(err).Msg("Failed to assign default role")
		}
	}

	// Log security event
	securitylog.Log(ctx, securitylog.Event{
		Type:     securitylog.EventAccountCreated,
		Outcome:  securitylog.OutcomeSuccess,
		ActorID:  user.ID.String(),
		IP:       ip,
		Resource: "auth/register",
	})

	// Publish event
	if err := s.events.Publish(ctx, authEventsStream, "user.created", map[string]interface{}{
		"user_id": user.ID.String(),
		"email":   user.Email,
	}); err != nil {
		logger.Warn().Err(err).Msg("Failed to publish user.created event")
	}

	resp := user.ToResponse()
	return &resp, nil
}

// Login authenticates a user and returns tokens.
func (s *AuthService) Login(ctx context.Context, req models.LoginRequest, ip, userAgent string) (*models.LoginResponse, error) {
	if errs := validation.Validate(req); errs != nil {
		return nil, errors.BadRequest("Validation failed", fmt.Errorf("%v", validation.ErrorResponse(errs)))
	}

	req.Email = strings.ToLower(req.Email)

	user, err := s.users.GetByEmail(ctx, req.Email)
	if err != nil {
		securitylog.LogFailure(ctx, securitylog.EventLoginFailed, ip, "auth/login", map[string]interface{}{
			"reason": "user_not_found",
			"email":  req.Email,
		})
		// Return generic error to avoid user enumeration
		return nil, errors.Unauthorized("Invalid email or password", nil)
	}

	// Check account status
	if user.Status == models.UserStatusDisabled {
		securitylog.LogFailure(ctx, securitylog.EventLoginFailed, ip, "auth/login", map[string]interface{}{
			"reason": "account_disabled",
		})
		return nil, errors.Forbidden("Account is disabled", nil)
	}

	// Check lockout
	if user.Status == models.UserStatusLocked && user.LockedUntil != nil && user.LockedUntil.After(time.Now()) {
		securitylog.LogFailure(ctx, securitylog.EventLoginFailed, ip, "auth/login", map[string]interface{}{
			"reason":       "account_locked",
			"locked_until": user.LockedUntil,
		})
		return nil, errors.Forbidden("Account is locked. Try again later.", nil)
	}

	// Auto-unlock if lockout period has expired
	if user.Status == models.UserStatusLocked && (user.LockedUntil == nil || user.LockedUntil.Before(time.Now())) {
		if err := s.users.ResetFailedLogins(ctx, user.ID); err != nil {
			logger.Warn().Err(err).Msg("Failed to auto-unlock user")
		}
		if err := s.users.Update(ctx, user.ID, map[string]interface{}{"status": models.UserStatusActive}); err != nil {
			logger.Warn().Err(err).Msg("Failed to update user status after unlock")
		}
	}

	// Verify password
	if err := bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(req.Password)); err != nil {
		// Increment failed login counter
		if incErr := s.users.IncrementFailedLogins(ctx, user.ID); incErr != nil {
			logger.Warn().Err(incErr).Msg("Failed to increment failed logins")
		}

		// Check if we should lock the account
		if user.FailedLoginCount+1 >= s.securityCfg.MaxFailedLogins {
			lockUntil := time.Now().Add(time.Duration(s.securityCfg.LockoutDurationMins) * time.Minute)
			if lockErr := s.users.Update(ctx, user.ID, map[string]interface{}{
				"status":       models.UserStatusLocked,
				"locked_until": lockUntil,
			}); lockErr != nil {
				logger.Warn().Err(lockErr).Msg("Failed to lock user account")
			}
			securitylog.Log(ctx, securitylog.Event{
				Type:     securitylog.EventAccountLocked,
				Outcome:  securitylog.OutcomeBlocked,
				ActorID:  user.ID.String(),
				IP:       ip,
				Resource: "auth/login",
				Details:  map[string]interface{}{"locked_until": lockUntil},
			})
		}

		securitylog.LogFailure(ctx, securitylog.EventLoginFailed, ip, "auth/login", map[string]interface{}{
			"reason":             "invalid_password",
			"failed_login_count": user.FailedLoginCount + 1,
		})

		// Publish auth.failed event
		if pubErr := s.events.Publish(ctx, authEventsStream, "auth.failed", map[string]interface{}{
			"user_id": user.ID.String(),
			"reason":  "invalid_password",
			"ip":      ip,
		}); pubErr != nil {
			logger.Warn().Err(pubErr).Msg("Failed to publish auth.failed event")
		}

		return nil, errors.Unauthorized("Invalid email or password", nil)
	}

	// Check MFA requirement
	if user.MFAEnabled {
		if req.MFACode == "" {
			return &models.LoginResponse{
				MFARequired: true,
				User:        user.ToResponse(),
			}, nil
		}
		// Verify MFA code using TOTP
		if !verifyTOTP(user.MFASecret, req.MFACode) {
			securitylog.LogFailure(ctx, securitylog.EventMFAFailed, ip, "auth/login", map[string]interface{}{
				"user_id": user.ID.String(),
			})
			return nil, errors.Unauthorized("Invalid MFA code", nil)
		}
		securitylog.LogSuccess(ctx, securitylog.EventMFASuccess, user.ID.String(), ip, "auth/login")
	}

	// Reset failed logins on successful auth
	if err := s.users.ResetFailedLogins(ctx, user.ID); err != nil {
		logger.Warn().Err(err).Msg("Failed to reset failed logins")
	}

	// Update last login
	if err := s.users.UpdateLastLogin(ctx, user.ID); err != nil {
		logger.Warn().Err(err).Msg("Failed to update last login")
	}

	// Get user roles for token claims
	userRoles, err := s.roles.GetUserRoles(ctx, user.ID)
	if err != nil {
		logger.Warn().Err(err).Msg("Failed to get user roles")
	}
	roleNames := make([]string, 0, len(userRoles))
	for _, ur := range userRoles {
		roleNames = append(roleNames, string(ur.RoleName))
	}

	// Generate tokens
	accessToken, err := s.tokens.GenerateAccessToken(user, roleNames)
	if err != nil {
		return nil, err
	}

	refreshTokenRaw, refreshTokenHash, err := s.tokens.GenerateRefreshToken()
	if err != nil {
		return nil, err
	}

	// Create session
	session := &models.Session{
		UserID:           user.ID,
		RefreshTokenHash: refreshTokenHash,
		IPAddress:        ip,
		UserAgent:        userAgent,
		ExpiresAt:        time.Now().Add(s.tokens.RefreshTokenTTL()),
	}
	if err := s.sessions.Create(ctx, session); err != nil {
		return nil, err
	}

	// Log security event
	securitylog.LogSuccess(ctx, securitylog.EventLoginSuccess, user.ID.String(), ip, "auth/login")

	// Publish event
	if err := s.events.Publish(ctx, authEventsStream, "user.login", map[string]interface{}{
		"user_id": user.ID.String(),
		"ip":      ip,
	}); err != nil {
		logger.Warn().Err(err).Msg("Failed to publish user.login event")
	}

	userResp := user.ToResponse()
	userResp.Roles = roleNames

	return &models.LoginResponse{
		AccessToken:  accessToken,
		RefreshToken: refreshTokenRaw,
		ExpiresIn:    s.tokens.AccessTokenTTLSeconds(),
		TokenType:    "Bearer",
		User:         userResp,
	}, nil
}

// Logout revokes a session and blacklists the access token.
func (s *AuthService) Logout(ctx context.Context, sessionID string, accessTokenID string, accessTokenTTLRemaining int64, ip string) error {
	// Blacklist the access token for its remaining lifetime
	if accessTokenID != "" {
		if err := s.tokenCache.BlacklistToken(ctx, accessTokenID, accessTokenTTLRemaining); err != nil {
			logger.Warn().Err(err).Msg("Failed to blacklist access token")
		}
	}

	// Delete session from cache
	if err := s.tokenCache.InvalidateSession(ctx, sessionID); err != nil {
		logger.Warn().Err(err).Msg("Failed to invalidate session cache")
	}

	securitylog.LogSuccess(ctx, securitylog.EventLogout, "", ip, "auth/logout")

	// Publish event
	if err := s.events.Publish(ctx, authEventsStream, "user.logout", map[string]interface{}{
		"session_id": sessionID,
	}); err != nil {
		logger.Warn().Err(err).Msg("Failed to publish user.logout event")
	}

	return nil
}

// RefreshToken validates a refresh token and issues a new token pair.
func (s *AuthService) RefreshToken(ctx context.Context, refreshToken string, ip, userAgent string) (*models.LoginResponse, error) {
	tokenHash := HashToken(refreshToken)

	session, err := s.sessions.GetByRefreshToken(ctx, tokenHash)
	if err != nil {
		securitylog.LogFailure(ctx, securitylog.EventTokenInvalid, ip, "auth/refresh", nil)
		return nil, errors.Unauthorized("Invalid or expired refresh token", nil)
	}

	// Get user
	user, err := s.users.GetByID(ctx, session.UserID)
	if err != nil {
		return nil, err
	}

	if user.Status != models.UserStatusActive {
		return nil, errors.Forbidden("Account is not active", nil)
	}

	// Delete old session (token rotation)
	if err := s.sessions.Delete(ctx, session.ID); err != nil {
		logger.Warn().Err(err).Msg("Failed to delete old session during refresh")
	}

	// Get user roles
	userRoles, err := s.roles.GetUserRoles(ctx, user.ID)
	if err != nil {
		logger.Warn().Err(err).Msg("Failed to get user roles")
	}
	roleNames := make([]string, 0, len(userRoles))
	for _, ur := range userRoles {
		roleNames = append(roleNames, string(ur.RoleName))
	}

	// Generate new tokens
	accessToken, err := s.tokens.GenerateAccessToken(user, roleNames)
	if err != nil {
		return nil, err
	}

	newRefreshRaw, newRefreshHash, err := s.tokens.GenerateRefreshToken()
	if err != nil {
		return nil, err
	}

	// Create new session
	newSession := &models.Session{
		UserID:           user.ID,
		RefreshTokenHash: newRefreshHash,
		IPAddress:        ip,
		UserAgent:        userAgent,
		ExpiresAt:        time.Now().Add(s.tokens.RefreshTokenTTL()),
	}
	if err := s.sessions.Create(ctx, newSession); err != nil {
		return nil, err
	}

	securitylog.LogSuccess(ctx, securitylog.EventTokenRefreshed, user.ID.String(), ip, "auth/refresh")

	userResp := user.ToResponse()
	userResp.Roles = roleNames

	return &models.LoginResponse{
		AccessToken:  accessToken,
		RefreshToken: newRefreshRaw,
		ExpiresIn:    s.tokens.AccessTokenTTLSeconds(),
		TokenType:    "Bearer",
		User:         userResp,
	}, nil
}

// SetupMFA generates a new TOTP secret for the user.
func (s *AuthService) SetupMFA(ctx context.Context, userID string, ip string) (*models.MFASetupResponse, error) {
	secret := make([]byte, 20)
	if _, err := rand.Read(secret); err != nil {
		return nil, errors.InternalServerError("Failed to generate MFA secret", err)
	}
	encodedSecret := base32.StdEncoding.WithPadding(base32.NoPadding).EncodeToString(secret)

	// Store the secret temporarily in Redis (not in DB until verified)
	if err := s.tokenCache.StoreMFAToken(ctx, userID, encodedSecret); err != nil {
		return nil, errors.InternalServerError("Failed to store MFA token", err)
	}

	qrURL := fmt.Sprintf("otpauth://totp/%s:%s?secret=%s&issuer=%s&algorithm=SHA1&digits=6&period=30",
		url.PathEscape(s.mfaCfg.Issuer),
		url.PathEscape(userID),
		encodedSecret,
		url.QueryEscape(s.mfaCfg.Issuer),
	)

	return &models.MFASetupResponse{
		Secret:    encodedSecret,
		QRCodeURL: qrURL,
		Issuer:    s.mfaCfg.Issuer,
	}, nil
}

// VerifyMFA completes MFA enrollment by verifying a TOTP code.
func (s *AuthService) VerifyMFA(ctx context.Context, userID string, code string, ip string) error {
	if errs := validation.Validate(models.MFAVerifyRequest{Code: code}); errs != nil {
		return errors.BadRequest("Validation failed", fmt.Errorf("%v", validation.ErrorResponse(errs)))
	}

	secret, err := s.tokenCache.GetMFAToken(ctx, userID)
	if err != nil {
		return errors.BadRequest("MFA setup not initiated or expired. Please start MFA setup again.", nil)
	}

	if !verifyTOTP(secret, code) {
		securitylog.LogFailure(ctx, securitylog.EventMFAFailed, ip, "auth/mfa/verify", map[string]interface{}{
			"user_id": userID,
		})
		return errors.Unauthorized("Invalid MFA code", nil)
	}

	uid, err := uuid.Parse(userID)
	if err != nil {
		return errors.BadRequest("Invalid user ID", err)
	}

	if err := s.users.Update(ctx, uid, map[string]interface{}{
		"mfa_enabled": true,
		"mfa_secret":  secret,
	}); err != nil {
		return err
	}

	if err := s.tokenCache.DeleteMFAToken(ctx, userID); err != nil {
		logger.Warn().Err(err).Msg("Failed to delete MFA token from cache")
	}

	securitylog.LogSuccess(ctx, securitylog.EventMFASuccess, userID, ip, "auth/mfa/verify")

	if err := s.events.Publish(ctx, authEventsStream, "auth.mfa_enabled", map[string]interface{}{
		"user_id": userID,
	}); err != nil {
		logger.Warn().Err(err).Msg("Failed to publish auth.mfa_enabled event")
	}

	return nil
}

// verifyTOTP verifies a TOTP code against a secret.
func verifyTOTP(secret, code string) bool {
	if secret == "" || code == "" || len(code) != 6 {
		return false
	}

	key, err := base32.StdEncoding.WithPadding(base32.NoPadding).DecodeString(secret)
	if err != nil {
		return false
	}

	now := time.Now().Unix()
	timeStep := int64(30)

	for _, offset := range []int64{-1, 0, 1} {
		counter := (now / timeStep) + offset
		expected := generateHOTP(key, counter)
		if expected == code {
			return true
		}
	}
	return false
}

// generateHOTP generates a 6-digit HOTP code for the given counter.
func generateHOTP(key []byte, counter int64) string {
	msg := make([]byte, 8)
	for i := 7; i >= 0; i-- {
		msg[i] = byte(counter & 0xff)
		counter >>= 8
	}

	mac := hmac.New(sha1.New, key)
	mac.Write(msg)
	hmacResult := mac.Sum(nil)

	offset := hmacResult[len(hmacResult)-1] & 0x0f
	code := int64(hmacResult[offset]&0x7f)<<24 |
		int64(hmacResult[offset+1]&0xff)<<16 |
		int64(hmacResult[offset+2]&0xff)<<8 |
		int64(hmacResult[offset+3]&0xff)

	return fmt.Sprintf("%06d", code%1000000)
}
