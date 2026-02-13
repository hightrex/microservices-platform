package models

import (
	"time"

	"github.com/google/uuid"
)

// RoleName represents valid role names.
type RoleName string

const (
	RoleOrgOwner RoleName = "org_owner"
	RoleOrgAdmin RoleName = "org_admin"
	RoleManager  RoleName = "manager"
	RoleMember   RoleName = "member"
	RoleViewer   RoleName = "viewer"
)

// Role represents a role in the RBAC system.
type Role struct {
	ID          uuid.UUID `json:"id"`
	TenantID    uuid.UUID `json:"tenant_id"`
	Name        RoleName  `json:"name"`
	Description string    `json:"description"`
	IsSystem    bool      `json:"is_system"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

// Permission represents a permission on a resource.
type Permission struct {
	ID          uuid.UUID `json:"id"`
	Resource    string    `json:"resource"`
	Action      string    `json:"action"`
	Description string    `json:"description"`
}

// UserRole represents the assignment of a role to a user.
type UserRole struct {
	UserID    uuid.UUID  `json:"user_id"`
	RoleID    uuid.UUID  `json:"role_id"`
	TenantID  uuid.UUID  `json:"tenant_id"`
	GrantedBy *uuid.UUID `json:"granted_by,omitempty"`
	GrantedAt time.Time  `json:"granted_at"`
	RoleName  RoleName   `json:"role_name,omitempty"` // populated on read
}

// AssignRoleRequest is the input for assigning a role to a user.
type AssignRoleRequest struct {
	RoleName string `json:"role_name" validate:"required,oneof=org_owner org_admin manager member viewer"`
}

// RoleWithPermissions represents a role with its associated permissions.
type RoleWithPermissions struct {
	Role        Role         `json:"role"`
	Permissions []Permission `json:"permissions"`
}
