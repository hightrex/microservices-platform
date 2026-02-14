package models

import (
	"time"

	"github.com/google/uuid"
)

// Department represents a department within an organization.
type Department struct {
	ID          uuid.UUID  `json:"id"`
	OrgID       uuid.UUID  `json:"org_id"`
	Name        string     `json:"name"`
	Description string     `json:"description"`
	ParentID    *uuid.UUID `json:"parent_id,omitempty"`
	CreatedAt   time.Time  `json:"created_at"`
	UpdatedAt   time.Time  `json:"updated_at"`
}

// CreateDeptRequest is the input for creating a department.
type CreateDeptRequest struct {
	Name        string     `json:"name"        validate:"required,min=1,max=255"`
	Description string     `json:"description" validate:"max=1000"`
	ParentID    *uuid.UUID `json:"parent_id,omitempty"`
}

// UpdateDeptRequest is the input for updating a department.
type UpdateDeptRequest struct {
	Name        *string    `json:"name,omitempty"        validate:"omitempty,min=1,max=255"`
	Description *string    `json:"description,omitempty" validate:"omitempty,max=1000"`
	ParentID    *uuid.UUID `json:"parent_id,omitempty"`
}

// DeptResponse is the external representation of a department.
type DeptResponse struct {
	ID          uuid.UUID     `json:"id"`
	Name        string        `json:"name"`
	Description string        `json:"description"`
	ParentID    *uuid.UUID    `json:"parent_id,omitempty"`
	Children    []DeptResponse `json:"children,omitempty"`
	CreatedAt   time.Time     `json:"created_at"`
	UpdatedAt   time.Time     `json:"updated_at"`
}

// ToResponse converts a Department to its external representation.
func (d *Department) ToResponse() DeptResponse {
	return DeptResponse{
		ID:          d.ID,
		Name:        d.Name,
		Description: d.Description,
		ParentID:    d.ParentID,
		CreatedAt:   d.CreatedAt,
		UpdatedAt:   d.UpdatedAt,
	}
}
