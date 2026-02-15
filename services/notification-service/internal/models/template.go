package models

import (
	"encoding/json"
	"time"

	"github.com/google/uuid"
)

// Channel represents the notification delivery channel.
type Channel string

const (
	ChannelEmail   Channel = "email"
	ChannelSMS     Channel = "sms"
	ChannelInApp   Channel = "in_app"
	ChannelWebhook Channel = "webhook"
)

// ValidChannels returns all valid channel values.
func ValidChannels() []Channel {
	return []Channel{ChannelEmail, ChannelSMS, ChannelInApp, ChannelWebhook}
}

// IsValid checks if the channel value is valid.
func (c Channel) IsValid() bool {
	for _, valid := range ValidChannels() {
		if c == valid {
			return true
		}
	}
	return false
}

// NotificationTemplate represents a notification template stored in the database.
type NotificationTemplate struct {
	ID                 uuid.UUID       `json:"id"`
	TenantID           uuid.UUID       `json:"tenant_id"`
	Name               string          `json:"name"`
	Channel            Channel         `json:"channel"`
	SubjectTemplate    *string         `json:"subject_template,omitempty"`
	BodyTemplate       string          `json:"body_template"`
	TemplateDataSchema json.RawMessage `json:"template_data_schema"`
	Language           string          `json:"language"`
	IsActive           bool            `json:"is_active"`
	CreatedAt          time.Time       `json:"created_at"`
	UpdatedAt          time.Time       `json:"updated_at"`
	CreatedBy          *uuid.UUID      `json:"created_by,omitempty"`
}

// CreateTemplateRequest is the input for creating a notification template.
type CreateTemplateRequest struct {
	Name               string          `json:"name" validate:"required,min=1,max=255"`
	Channel            Channel         `json:"channel" validate:"required,oneof=email sms in_app webhook"`
	SubjectTemplate    *string         `json:"subject_template,omitempty" validate:"omitempty,max=1000"`
	BodyTemplate       string          `json:"body_template" validate:"required,min=1"`
	TemplateDataSchema json.RawMessage `json:"template_data_schema,omitempty"`
	Language           string          `json:"language,omitempty" validate:"omitempty,min=2,max=10"`
}

// UpdateTemplateRequest is the input for updating a notification template.
type UpdateTemplateRequest struct {
	Name               *string         `json:"name,omitempty" validate:"omitempty,min=1,max=255"`
	SubjectTemplate    *string         `json:"subject_template,omitempty" validate:"omitempty,max=1000"`
	BodyTemplate       *string         `json:"body_template,omitempty" validate:"omitempty,min=1"`
	TemplateDataSchema json.RawMessage `json:"template_data_schema,omitempty"`
	Language           *string         `json:"language,omitempty" validate:"omitempty,min=2,max=10"`
	IsActive           *bool           `json:"is_active,omitempty"`
}

// TemplateResponse is the external representation of a notification template.
type TemplateResponse struct {
	ID                 uuid.UUID       `json:"id"`
	Name               string          `json:"name"`
	Channel            Channel         `json:"channel"`
	SubjectTemplate    *string         `json:"subject_template,omitempty"`
	BodyTemplate       string          `json:"body_template"`
	TemplateDataSchema json.RawMessage `json:"template_data_schema,omitempty"`
	Language           string          `json:"language"`
	IsActive           bool            `json:"is_active"`
	CreatedAt          time.Time       `json:"created_at"`
	UpdatedAt          time.Time       `json:"updated_at"`
}

// ToResponse converts a NotificationTemplate to its external representation.
func (t *NotificationTemplate) ToResponse() TemplateResponse {
	return TemplateResponse{
		ID:                 t.ID,
		Name:               t.Name,
		Channel:            t.Channel,
		SubjectTemplate:    t.SubjectTemplate,
		BodyTemplate:       t.BodyTemplate,
		TemplateDataSchema: t.TemplateDataSchema,
		Language:           t.Language,
		IsActive:           t.IsActive,
		CreatedAt:          t.CreatedAt,
		UpdatedAt:          t.UpdatedAt,
	}
}

// TemplateFilter holds filter parameters for listing templates.
type TemplateFilter struct {
	Channel  *Channel `json:"channel,omitempty"`
	IsActive *bool    `json:"is_active,omitempty"`
	Language *string  `json:"language,omitempty"`
}
