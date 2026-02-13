package models

import (
	"time"

	"github.com/google/uuid"
)

// Session represents an active user session.
type Session struct {
	ID               uuid.UUID `json:"id"`
	UserID           uuid.UUID `json:"user_id"`
	TenantID         uuid.UUID `json:"tenant_id"`
	RefreshTokenHash string    `json:"-"` // never expose
	IPAddress        string    `json:"ip_address"`
	UserAgent        string    `json:"user_agent"`
	ExpiresAt        time.Time `json:"expires_at"`
	CreatedAt        time.Time `json:"created_at"`
}

// SessionResponse is the external representation of a session.
type SessionResponse struct {
	ID        uuid.UUID `json:"id"`
	IPAddress string    `json:"ip_address"`
	UserAgent string    `json:"user_agent"`
	ExpiresAt time.Time `json:"expires_at"`
	CreatedAt time.Time `json:"created_at"`
}

// ToResponse converts a Session to its external representation.
func (s *Session) ToResponse() SessionResponse {
	return SessionResponse{
		ID:        s.ID,
		IPAddress: s.IPAddress,
		UserAgent: s.UserAgent,
		ExpiresAt: s.ExpiresAt,
		CreatedAt: s.CreatedAt,
	}
}

// LoginRequest is the input for user authentication.
type LoginRequest struct {
	Email    string `json:"email"    validate:"required,email"`
	Password string `json:"password" validate:"required"`
	MFACode  string `json:"mfa_code,omitempty" validate:"omitempty,len=6"`
}

// LoginResponse is returned after successful authentication.
type LoginResponse struct {
	AccessToken  string       `json:"access_token"`
	RefreshToken string       `json:"refresh_token"`
	ExpiresIn    int64        `json:"expires_in"` // seconds
	TokenType    string       `json:"token_type"`
	User         UserResponse `json:"user"`
	MFARequired  bool         `json:"mfa_required,omitempty"`
}

// RefreshTokenRequest is the input for refreshing an access token.
type RefreshTokenRequest struct {
	RefreshToken string `json:"refresh_token" validate:"required"`
}

// TokenPair holds an access and refresh token pair.
type TokenPair struct {
	AccessToken  string `json:"access_token"`
	RefreshToken string `json:"refresh_token"`
	ExpiresIn    int64  `json:"expires_in"`
}
