package models

import (
	"time"

	"github.com/google/uuid"
)

// NotificationPreference represents a user's notification preference.
type NotificationPreference struct {
	ID        uuid.UUID `json:"id"`
	UserID    uuid.UUID `json:"user_id"`
	TenantID  uuid.UUID `json:"tenant_id"`
	Channel   Channel   `json:"channel"`
	EventType string    `json:"event_type"`
	Enabled   bool      `json:"enabled"`
	UpdatedAt time.Time `json:"updated_at"`
}

// UpdatePreferenceRequest is the input for updating a notification preference.
type UpdatePreferenceRequest struct {
	Channel   Channel `json:"channel" validate:"required,oneof=email sms in_app webhook"`
	EventType string  `json:"event_type" validate:"required,max=255"`
	Enabled   bool    `json:"enabled"`
}

// BulkUpdatePreferencesRequest is the input for bulk-updating preferences.
type BulkUpdatePreferencesRequest struct {
	Preferences []UpdatePreferenceRequest `json:"preferences" validate:"required,min=1,dive"`
}

// PreferenceResponse is the external representation of a notification preference.
type PreferenceResponse struct {
	Channel   Channel `json:"channel"`
	EventType string  `json:"event_type"`
	Enabled   bool    `json:"enabled"`
}

// ToResponse converts a NotificationPreference to its external representation.
func (p *NotificationPreference) ToResponse() PreferenceResponse {
	return PreferenceResponse{
		Channel:   p.Channel,
		EventType: p.EventType,
		Enabled:   p.Enabled,
	}
}
