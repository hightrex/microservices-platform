package models

import (
	"time"

	"github.com/google/uuid"
)

// MemberStatus represents a member's status within an organization.
type MemberStatus string

const (
	MemberStatusInvited MemberStatus = "invited"
	MemberStatusActive  MemberStatus = "active"
	MemberStatusRemoved MemberStatus = "removed"
)

// Member represents a member of an organization.
type Member struct {
	ID        uuid.UUID    `json:"id"`
	OrgID     uuid.UUID    `json:"org_id"`
	UserID    uuid.UUID    `json:"user_id"`
	Role      string       `json:"role"`
	InvitedBy *uuid.UUID   `json:"invited_by,omitempty"`
	InvitedAt *time.Time   `json:"invited_at,omitempty"`
	JoinedAt  *time.Time   `json:"joined_at,omitempty"`
	Status    MemberStatus `json:"status"`
	CreatedAt time.Time    `json:"created_at"`
	UpdatedAt time.Time    `json:"updated_at"`
}

// InviteMemberRequest is the input for inviting a member to an organization.
type InviteMemberRequest struct {
	UserID uuid.UUID `json:"user_id" validate:"required,uuid"`
	Role   string    `json:"role"    validate:"required,oneof=org_owner org_admin manager member viewer"`
}

// UpdateMemberRoleRequest is the input for updating a member's role.
type UpdateMemberRoleRequest struct {
	Role string `json:"role" validate:"required,oneof=org_owner org_admin manager member viewer"`
}

// MemberResponse is the external representation of a member.
type MemberResponse struct {
	ID        uuid.UUID    `json:"id"`
	UserID    uuid.UUID    `json:"user_id"`
	Role      string       `json:"role"`
	InvitedBy *uuid.UUID   `json:"invited_by,omitempty"`
	InvitedAt *time.Time   `json:"invited_at,omitempty"`
	JoinedAt  *time.Time   `json:"joined_at,omitempty"`
	Status    MemberStatus `json:"status"`
	CreatedAt time.Time    `json:"created_at"`
}

// ToResponse converts a Member to its external representation.
func (m *Member) ToResponse() MemberResponse {
	return MemberResponse{
		ID:        m.ID,
		UserID:    m.UserID,
		Role:      m.Role,
		InvitedBy: m.InvitedBy,
		InvitedAt: m.InvitedAt,
		JoinedAt:  m.JoinedAt,
		Status:    m.Status,
		CreatedAt: m.CreatedAt,
	}
}

// MemberFilter holds filtering criteria for listing members.
type MemberFilter struct {
	Status *MemberStatus `json:"status,omitempty"`
	Role   *string       `json:"role,omitempty"`
}
