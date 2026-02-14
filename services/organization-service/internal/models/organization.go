package models

import (
	"encoding/json"
	"time"

	"github.com/google/uuid"
)

// OrgPlan represents the billing plan for an organization.
type OrgPlan string

const (
	PlanFree       OrgPlan = "free"
	PlanStarter    OrgPlan = "starter"
	PlanBusiness   OrgPlan = "business"
	PlanEnterprise OrgPlan = "enterprise"
)

// OrgStatus represents the organization lifecycle status.
type OrgStatus string

const (
	OrgStatusActive    OrgStatus = "active"
	OrgStatusSuspended OrgStatus = "suspended"
	OrgStatusDeleted   OrgStatus = "deleted"
)

// Organization represents an organization (tenant) in the system.
// The organization ID serves as the tenant_id for all downstream services.
type Organization struct {
	ID              uuid.UUID       `json:"id"`
	Name            string          `json:"name"`
	Slug            string          `json:"slug"`
	OwnerUserID     uuid.UUID       `json:"owner_user_id"`
	Plan            OrgPlan         `json:"plan"`
	Status          OrgStatus       `json:"status"`
	Settings        json.RawMessage `json:"settings"`
	MaxUsers        int             `json:"max_users"`
	MaxStorageBytes int64           `json:"max_storage_bytes"`
	CreatedAt       time.Time       `json:"created_at"`
	UpdatedAt       time.Time       `json:"updated_at"`
}

// CreateOrgRequest is the input for creating a new organization.
type CreateOrgRequest struct {
	Name string  `json:"name" validate:"required,min=2,max=255"`
	Slug string  `json:"slug" validate:"required,min=2,max=255,alphanum"`
	Plan OrgPlan `json:"plan" validate:"omitempty,oneof=free starter business enterprise"`
}

// UpdateOrgRequest is the input for updating organization settings.
type UpdateOrgRequest struct {
	Name     *string          `json:"name,omitempty"     validate:"omitempty,min=2,max=255"`
	Settings *json.RawMessage `json:"settings,omitempty"`
}

// OrgResponse is the external representation of an organization.
type OrgResponse struct {
	ID              uuid.UUID        `json:"id"`
	Name            string           `json:"name"`
	Slug            string           `json:"slug"`
	OwnerUserID     uuid.UUID        `json:"owner_user_id"`
	Plan            OrgPlan          `json:"plan"`
	Status          OrgStatus        `json:"status"`
	Settings        json.RawMessage  `json:"settings"`
	MaxUsers        int              `json:"max_users"`
	MaxStorageBytes int64            `json:"max_storage_bytes"`
	CreatedAt       time.Time        `json:"created_at"`
	UpdatedAt       time.Time        `json:"updated_at"`
	Modules         []ModuleResponse `json:"modules,omitempty"`
}

// ToResponse converts an Organization to its external representation.
func (o *Organization) ToResponse() OrgResponse {
	return OrgResponse{
		ID:              o.ID,
		Name:            o.Name,
		Slug:            o.Slug,
		OwnerUserID:     o.OwnerUserID,
		Plan:            o.Plan,
		Status:          o.Status,
		Settings:        o.Settings,
		MaxUsers:        o.MaxUsers,
		MaxStorageBytes: o.MaxStorageBytes,
		CreatedAt:       o.CreatedAt,
		UpdatedAt:       o.UpdatedAt,
	}
}

// OrgFilter holds filtering criteria for listing organizations.
type OrgFilter struct {
	Status *OrgStatus `json:"status,omitempty"`
	Search *string    `json:"search,omitempty"`
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
