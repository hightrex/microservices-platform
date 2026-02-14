package models

import (
	"time"

	"github.com/google/uuid"
)

// Plan represents a billing plan with its limits and pricing.
type Plan struct {
	ID                   uuid.UUID `json:"id"`
	Name                 string    `json:"name"`
	DisplayName          string    `json:"display_name"`
	MaxUsers             int       `json:"max_users"`
	MaxStorageBytes      int64     `json:"max_storage_bytes"`
	MaxAPICallsPerMinute int       `json:"max_api_calls_per_minute"`
	AvailableModules     []string  `json:"available_modules"`
	PriceMonthly         int       `json:"price_monthly_cents"`
	PriceAnnual          int       `json:"price_annual_cents"`
	IsActive             bool      `json:"is_active"`
	CreatedAt            time.Time `json:"created_at"`
	UpdatedAt            time.Time `json:"updated_at"`
}

// ChangePlanRequest is the input for changing an organization's plan.
type ChangePlanRequest struct {
	PlanName string `json:"plan_name" validate:"required,oneof=free starter business enterprise"`
}

// PlanResponse is the external representation of a plan.
type PlanResponse struct {
	ID                   uuid.UUID `json:"id"`
	Name                 string    `json:"name"`
	DisplayName          string    `json:"display_name"`
	MaxUsers             int       `json:"max_users"`
	MaxStorageBytes      int64     `json:"max_storage_bytes"`
	MaxAPICallsPerMinute int       `json:"max_api_calls_per_minute"`
	AvailableModules     []string  `json:"available_modules"`
	PriceMonthly         int       `json:"price_monthly_cents"`
	PriceAnnual          int       `json:"price_annual_cents"`
	IsActive             bool      `json:"is_active"`
}

// ToResponse converts a Plan to its external representation.
func (p *Plan) ToResponse() PlanResponse {
	return PlanResponse{
		ID:                   p.ID,
		Name:                 p.Name,
		DisplayName:          p.DisplayName,
		MaxUsers:             p.MaxUsers,
		MaxStorageBytes:      p.MaxStorageBytes,
		MaxAPICallsPerMinute: p.MaxAPICallsPerMinute,
		AvailableModules:     p.AvailableModules,
		PriceMonthly:         p.PriceMonthly,
		PriceAnnual:          p.PriceAnnual,
		IsActive:             p.IsActive,
	}
}
