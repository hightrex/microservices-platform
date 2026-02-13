package models

import (
	"time"

	"github.com/google/uuid"
)

// UserStatus represents the user account status.
type UserStatus string

const (
	UserStatusActive   UserStatus = "active"
	UserStatusDisabled UserStatus = "disabled"
	UserStatusLocked   UserStatus = "locked"
)

// User represents a user in the system.
type User struct {
	ID               uuid.UUID  `json:"id"`
	TenantID         uuid.UUID  `json:"tenant_id"`
	Email            string     `json:"email"`
	PasswordHash     string     `json:"-"` // never expose
	FirstName        string     `json:"first_name"`
	LastName         string     `json:"last_name"`
	Status           UserStatus `json:"status"`
	MFAEnabled       bool       `json:"mfa_enabled"`
	MFASecret        string     `json:"-"` // never expose
	FailedLoginCount int        `json:"-"` // internal only
	LockedUntil      *time.Time `json:"locked_until,omitempty"`
	LastLoginAt      *time.Time `json:"last_login_at,omitempty"`
	CreatedAt        time.Time  `json:"created_at"`
	UpdatedAt        time.Time  `json:"updated_at"`
}

// CreateUserRequest is the input for user registration.
type CreateUserRequest struct {
	Email     string `json:"email"      validate:"required,email,max=255"`
	Password  string `json:"password"   validate:"required,min=8,max=128"`
	FirstName string `json:"first_name" validate:"required,min=1,max=100"`
	LastName  string `json:"last_name"  validate:"required,min=1,max=100"`
}

// UpdateUserRequest is the input for updating a user profile.
type UpdateUserRequest struct {
	FirstName *string `json:"first_name,omitempty" validate:"omitempty,min=1,max=100"`
	LastName  *string `json:"last_name,omitempty"  validate:"omitempty,min=1,max=100"`
	Status    *string `json:"status,omitempty"     validate:"omitempty,oneof=active disabled locked"`
}

// ChangePasswordRequest is the input for changing a user password.
type ChangePasswordRequest struct {
	OldPassword string `json:"old_password" validate:"required,min=8,max=128"`
	NewPassword string `json:"new_password" validate:"required,min=8,max=128"`
}

// UserResponse is the external representation of a user.
type UserResponse struct {
	ID          uuid.UUID  `json:"id"`
	Email       string     `json:"email"`
	FirstName   string     `json:"first_name"`
	LastName    string     `json:"last_name"`
	Status      UserStatus `json:"status"`
	MFAEnabled  bool       `json:"mfa_enabled"`
	LastLoginAt *time.Time `json:"last_login_at,omitempty"`
	CreatedAt   time.Time  `json:"created_at"`
	UpdatedAt   time.Time  `json:"updated_at"`
	Roles       []string   `json:"roles,omitempty"`
}

// ToResponse converts a User to its external representation.
func (u *User) ToResponse() UserResponse {
	return UserResponse{
		ID:          u.ID,
		Email:       u.Email,
		FirstName:   u.FirstName,
		LastName:    u.LastName,
		Status:      u.Status,
		MFAEnabled:  u.MFAEnabled,
		LastLoginAt: u.LastLoginAt,
		CreatedAt:   u.CreatedAt,
		UpdatedAt:   u.UpdatedAt,
	}
}

// UserFilter holds filtering criteria for listing users.
type UserFilter struct {
	Status *UserStatus `json:"status,omitempty"`
	Search *string     `json:"search,omitempty"` // search by email/name
}

// Pagination holds pagination parameters.
type Pagination struct {
	Page     int `json:"page"      validate:"gte=1"`
	PageSize int `json:"page_size" validate:"gte=1,lte=100"`
}

// DefaultPagination returns sensible defaults for pagination.
func DefaultPagination() Pagination {
	return Pagination{Page: 1, PageSize: 20}
}

// PaginatedResponse wraps a paginated list of items.
type PaginatedResponse struct {
	Items      interface{} `json:"items"`
	TotalCount int         `json:"total_count"`
	Page       int         `json:"page"`
	PageSize   int         `json:"page_size"`
	TotalPages int         `json:"total_pages"`
}
