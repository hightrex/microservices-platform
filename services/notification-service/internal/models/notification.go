package models

import (
	"encoding/json"
	"time"

	"github.com/google/uuid"
)

// NotificationStatus represents the delivery status of a notification.
type NotificationStatus string

const (
	StatusPending NotificationStatus = "pending"
	StatusSent    NotificationStatus = "sent"
	StatusFailed  NotificationStatus = "failed"
	StatusRead    NotificationStatus = "read"
)

// Notification represents a notification stored in the database.
type Notification struct {
	ID        uuid.UUID          `json:"id"`
	TenantID  uuid.UUID          `json:"tenant_id"`
	UserID    uuid.UUID          `json:"user_id"`
	Channel   Channel            `json:"channel"`
	EventType string             `json:"event_type"`
	Subject   *string            `json:"subject,omitempty"`
	Body      string             `json:"body"`
	Status    NotificationStatus `json:"status"`
	Metadata  json.RawMessage    `json:"metadata"`
	SentAt    *time.Time         `json:"sent_at,omitempty"`
	ReadAt    *time.Time         `json:"read_at,omitempty"`
	CreatedAt time.Time          `json:"created_at"`
}

// SendNotificationRequest is the input for sending a notification.
type SendNotificationRequest struct {
	Channel    Channel         `json:"channel" validate:"required,oneof=email sms in_app webhook"`
	Recipient  string          `json:"recipient" validate:"required"`
	TemplateID *uuid.UUID      `json:"template_id,omitempty"`
	EventType  string          `json:"event_type" validate:"required,max=255"`
	Data       json.RawMessage `json:"data,omitempty"`
	Subject    *string         `json:"subject,omitempty"`
	Body       *string         `json:"body,omitempty"`
	UserID     uuid.UUID       `json:"user_id" validate:"required"`
}

// NotificationResponse is the external representation of a notification.
type NotificationResponse struct {
	ID        uuid.UUID          `json:"id"`
	Channel   Channel            `json:"channel"`
	EventType string             `json:"event_type"`
	Subject   *string            `json:"subject,omitempty"`
	Body      string             `json:"body"`
	Status    NotificationStatus `json:"status"`
	Metadata  json.RawMessage    `json:"metadata,omitempty"`
	SentAt    *time.Time         `json:"sent_at,omitempty"`
	ReadAt    *time.Time         `json:"read_at,omitempty"`
	CreatedAt time.Time          `json:"created_at"`
}

// ToResponse converts a Notification to its external representation.
func (n *Notification) ToResponse() NotificationResponse {
	return NotificationResponse{
		ID:        n.ID,
		Channel:   n.Channel,
		EventType: n.EventType,
		Subject:   n.Subject,
		Body:      n.Body,
		Status:    n.Status,
		Metadata:  n.Metadata,
		SentAt:    n.SentAt,
		ReadAt:    n.ReadAt,
		CreatedAt: n.CreatedAt,
	}
}

// NotificationFilter holds filter parameters for listing notifications.
type NotificationFilter struct {
	Status    *NotificationStatus `json:"status,omitempty"`
	Channel   *Channel            `json:"channel,omitempty"`
	EventType *string             `json:"event_type,omitempty"`
}

// Pagination holds pagination parameters.
type Pagination struct {
	Page     int `json:"page"`
	PageSize int `json:"page_size"`
}

// DefaultPagination returns default pagination values.
func DefaultPagination() Pagination {
	return Pagination{Page: 1, PageSize: 20}
}
