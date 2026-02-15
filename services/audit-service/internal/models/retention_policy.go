package models

import (
	"time"

	"github.com/google/uuid"
)

// RetentionPolicy configures how long to keep audit logs per event type.
type RetentionPolicy struct {
	ID            uuid.UUID `json:"id"`
	TenantID      uuid.UUID `json:"tenant_id"`
	EventType     string    `json:"event_type"`
	RetentionDays int       `json:"retention_days"`
	IsActive      bool      `json:"is_active"`
	CreatedAt     time.Time `json:"created_at"`
	UpdatedAt     time.Time `json:"updated_at"`
}

// CreatePolicyRequest is the input for creating a retention policy.
type CreatePolicyRequest struct {
	EventType     string `json:"event_type" validate:"required,max=255"`
	RetentionDays int    `json:"retention_days" validate:"required,min=30,max=2555"`
}

// UpdatePolicyRequest is the input for updating a retention policy.
type UpdatePolicyRequest struct {
	RetentionDays *int  `json:"retention_days,omitempty" validate:"omitempty,min=30,max=2555"`
	IsActive      *bool `json:"is_active,omitempty"`
}

// PolicyResponse is the external representation of a retention policy.
type PolicyResponse struct {
	ID            uuid.UUID `json:"id"`
	EventType     string    `json:"event_type"`
	RetentionDays int       `json:"retention_days"`
	IsActive      bool      `json:"is_active"`
	CreatedAt     time.Time `json:"created_at"`
	UpdatedAt     time.Time `json:"updated_at"`
}

// ToResponse converts a RetentionPolicy to its external representation.
func (p *RetentionPolicy) ToResponse() PolicyResponse {
	return PolicyResponse{
		ID:            p.ID,
		EventType:     p.EventType,
		RetentionDays: p.RetentionDays,
		IsActive:      p.IsActive,
		CreatedAt:     p.CreatedAt,
		UpdatedAt:     p.UpdatedAt,
	}
}
