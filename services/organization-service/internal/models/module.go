package models

import (
	"encoding/json"
	"time"

	"github.com/google/uuid"
)

// ModuleName represents the available modules for an organization.
type ModuleName string

const (
	ModuleNotifications  ModuleName = "notifications"
	ModuleBilling        ModuleName = "billing"
	ModuleFileManagement ModuleName = "file_management"
	ModuleAuditLogging   ModuleName = "audit_logging"
	ModuleAnalytics      ModuleName = "analytics"
)

// AllModuleNames returns all available module names.
func AllModuleNames() []ModuleName {
	return []ModuleName{
		ModuleNotifications,
		ModuleBilling,
		ModuleFileManagement,
		ModuleAuditLogging,
		ModuleAnalytics,
	}
}

// Module represents an organization's module configuration.
type Module struct {
	ID         uuid.UUID       `json:"id"`
	OrgID      uuid.UUID       `json:"org_id"`
	ModuleName ModuleName      `json:"module_name"`
	Enabled    bool            `json:"enabled"`
	Config     json.RawMessage `json:"config"`
	EnabledAt  *time.Time      `json:"enabled_at,omitempty"`
	DisabledAt *time.Time      `json:"disabled_at,omitempty"`
	CreatedAt  time.Time       `json:"created_at"`
	UpdatedAt  time.Time       `json:"updated_at"`
}

// ToggleModuleRequest is the input for enabling/disabling a module.
type ToggleModuleRequest struct {
	ModuleName string `json:"module_name" validate:"required,oneof=notifications billing file_management audit_logging analytics"`
	Enabled    bool   `json:"enabled"`
}

// UpdateModuleConfigRequest is the input for updating a module's config.
type UpdateModuleConfigRequest struct {
	Config json.RawMessage `json:"config" validate:"required"`
}

// ModuleResponse is the external representation of a module.
type ModuleResponse struct {
	ID         uuid.UUID       `json:"id"`
	ModuleName ModuleName      `json:"module_name"`
	Enabled    bool            `json:"enabled"`
	Config     json.RawMessage `json:"config"`
	EnabledAt  *time.Time      `json:"enabled_at,omitempty"`
	DisabledAt *time.Time      `json:"disabled_at,omitempty"`
}

// ToResponse converts a Module to its external representation.
func (m *Module) ToResponse() ModuleResponse {
	return ModuleResponse{
		ID:         m.ID,
		ModuleName: m.ModuleName,
		Enabled:    m.Enabled,
		Config:     m.Config,
		EnabledAt:  m.EnabledAt,
		DisabledAt: m.DisabledAt,
	}
}
