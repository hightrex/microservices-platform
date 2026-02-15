package models

import (
	"encoding/json"
	"time"

	"github.com/google/uuid"
)

// DeliveryStatus represents the delivery status of a notification attempt.
type DeliveryStatus string

const (
	DeliveryStatusDelivered DeliveryStatus = "delivered"
	DeliveryStatusFailed    DeliveryStatus = "failed"
	DeliveryStatusBounced   DeliveryStatus = "bounced"
)

// DeliveryLog represents a delivery attempt record.
type DeliveryLog struct {
	ID               uuid.UUID       `json:"id"`
	NotificationID   uuid.UUID       `json:"notification_id"`
	TenantID         uuid.UUID       `json:"tenant_id"`
	Channel          Channel         `json:"channel"`
	Recipient        string          `json:"recipient"`
	Status           DeliveryStatus  `json:"status"`
	ErrorMessage     *string         `json:"error_message,omitempty"`
	ProviderResponse json.RawMessage `json:"provider_response,omitempty"`
	RetryCount       int             `json:"retry_count"`
	DeliveredAt      *time.Time      `json:"delivered_at,omitempty"`
	CreatedAt        time.Time       `json:"created_at"`
}

// DeliveryLogResponse is the external representation of a delivery log entry.
type DeliveryLogResponse struct {
	ID          uuid.UUID      `json:"id"`
	Channel     Channel        `json:"channel"`
	Recipient   string         `json:"recipient"`
	Status      DeliveryStatus `json:"status"`
	Error       *string        `json:"error,omitempty"`
	RetryCount  int            `json:"retry_count"`
	DeliveredAt *time.Time     `json:"delivered_at,omitempty"`
	CreatedAt   time.Time      `json:"created_at"`
}

// ToResponse converts a DeliveryLog to its external representation.
func (d *DeliveryLog) ToResponse() DeliveryLogResponse {
	return DeliveryLogResponse{
		ID:          d.ID,
		Channel:     d.Channel,
		Recipient:   d.Recipient,
		Status:      d.Status,
		Error:       d.ErrorMessage,
		RetryCount:  d.RetryCount,
		DeliveredAt: d.DeliveredAt,
		CreatedAt:   d.CreatedAt,
	}
}
