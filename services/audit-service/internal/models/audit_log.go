package models

import (
	"encoding/json"
	"time"

	"github.com/google/uuid"
)

// EventCategory represents the category of an audit event.
type EventCategory string

const (
	CategoryAuth     EventCategory = "auth"
	CategoryData     EventCategory = "data"
	CategorySystem   EventCategory = "system"
	CategorySecurity EventCategory = "security"
)

// ActorType represents who performed the action.
type ActorType string

const (
	ActorUser   ActorType = "user"
	ActorSystem ActorType = "system"
	ActorAPIKey ActorType = "api_key"
)

// Outcome represents the result of the audited action.
type Outcome string

const (
	OutcomeSuccess Outcome = "success"
	OutcomeFailure Outcome = "failure"
)

// AuditLog represents an immutable audit log entry.
type AuditLog struct {
	ID            uuid.UUID       `json:"id"`
	TenantID      uuid.UUID       `json:"tenant_id"`
	EventType     string          `json:"event_type"`
	EventCategory EventCategory   `json:"event_category"`
	ActorID       string          `json:"actor_id"`
	ActorType     ActorType       `json:"actor_type"`
	ResourceType  *string         `json:"resource_type,omitempty"`
	ResourceID    *string         `json:"resource_id,omitempty"`
	Action        string          `json:"action"`
	Outcome       Outcome         `json:"outcome"`
	IPAddress     *string         `json:"ip_address,omitempty"`
	UserAgent     *string         `json:"user_agent,omitempty"`
	Metadata      json.RawMessage `json:"metadata"`
	Timestamp     time.Time       `json:"timestamp"`
	PreviousHash  *string         `json:"previous_hash,omitempty"`
	CurrentHash   string          `json:"current_hash"`
}

// AuditLogResponse is the external representation of an audit log entry.
type AuditLogResponse struct {
	ID            uuid.UUID       `json:"id"`
	EventType     string          `json:"event_type"`
	EventCategory EventCategory   `json:"event_category"`
	ActorID       string          `json:"actor_id"`
	ActorType     ActorType       `json:"actor_type"`
	ResourceType  *string         `json:"resource_type,omitempty"`
	ResourceID    *string         `json:"resource_id,omitempty"`
	Action        string          `json:"action"`
	Outcome       Outcome         `json:"outcome"`
	IPAddress     *string         `json:"ip_address,omitempty"`
	Metadata      json.RawMessage `json:"metadata,omitempty"`
	Timestamp     time.Time       `json:"timestamp"`
}

// ToResponse converts an AuditLog to its external representation.
func (a *AuditLog) ToResponse() AuditLogResponse {
	return AuditLogResponse{
		ID:            a.ID,
		EventType:     a.EventType,
		EventCategory: a.EventCategory,
		ActorID:       a.ActorID,
		ActorType:     a.ActorType,
		ResourceType:  a.ResourceType,
		ResourceID:    a.ResourceID,
		Action:        a.Action,
		Outcome:       a.Outcome,
		IPAddress:     a.IPAddress,
		Metadata:      a.Metadata,
		Timestamp:     a.Timestamp,
	}
}

// SearchRequest holds filter parameters for searching audit logs.
type SearchRequest struct {
	EventType    *string    `json:"event_type,omitempty"`
	EventCategory *EventCategory `json:"event_category,omitempty"`
	ActorID      *string    `json:"actor_id,omitempty"`
	ResourceType *string    `json:"resource_type,omitempty"`
	ResourceID   *string    `json:"resource_id,omitempty"`
	Outcome      *Outcome   `json:"outcome,omitempty"`
	StartDate    *time.Time `json:"start_date,omitempty"`
	EndDate      *time.Time `json:"end_date,omitempty"`
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

// VerificationReport holds the result of a hash chain verification.
type VerificationReport struct {
	Valid       bool   `json:"valid"`
	TotalLogs   int    `json:"total_logs"`
	Verified    int    `json:"verified"`
	BrokenAt    *int   `json:"broken_at,omitempty"`
	ErrorDetail string `json:"error_detail,omitempty"`
}

// Statistics holds event count statistics.
type Statistics struct {
	TotalLogs       int                    `json:"total_logs"`
	ByEventType     map[string]int         `json:"by_event_type"`
	ByCategory      map[EventCategory]int  `json:"by_category"`
	ByOutcome       map[Outcome]int        `json:"by_outcome"`
}
