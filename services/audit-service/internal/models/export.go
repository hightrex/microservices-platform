package models

import (
	"time"

	"github.com/google/uuid"
)

// ExportStatus represents the status of an audit export job.
type ExportStatus string

const (
	ExportStatusPending    ExportStatus = "pending"
	ExportStatusProcessing ExportStatus = "processing"
	ExportStatusCompleted  ExportStatus = "completed"
	ExportStatusFailed     ExportStatus = "failed"
)

// ExportFormat represents the output format of an export.
type ExportFormat string

const (
	ExportFormatCSV  ExportFormat = "csv"
	ExportFormatJSON ExportFormat = "json"
)

// AuditExport represents an audit export job.
type AuditExport struct {
	ID           uuid.UUID    `json:"id"`
	TenantID     uuid.UUID    `json:"tenant_id"`
	RequestedBy  uuid.UUID    `json:"requested_by"`
	StartDate    time.Time    `json:"start_date"`
	EndDate      time.Time    `json:"end_date"`
	Format       ExportFormat `json:"format"`
	Status       ExportStatus `json:"status"`
	FileID       *uuid.UUID   `json:"file_id,omitempty"`
	ErrorMessage *string      `json:"error_message,omitempty"`
	CreatedAt    time.Time    `json:"created_at"`
	CompletedAt  *time.Time   `json:"completed_at,omitempty"`
}

// CreateExportRequest is the input for creating an export job.
type CreateExportRequest struct {
	StartDate time.Time    `json:"start_date" validate:"required"`
	EndDate   time.Time    `json:"end_date" validate:"required"`
	Format    ExportFormat `json:"format" validate:"required,oneof=csv json"`
}

// ExportResponse is the external representation of an export job.
type ExportResponse struct {
	ID          uuid.UUID    `json:"id"`
	StartDate   time.Time    `json:"start_date"`
	EndDate     time.Time    `json:"end_date"`
	Format      ExportFormat `json:"format"`
	Status      ExportStatus `json:"status"`
	FileID      *uuid.UUID   `json:"file_id,omitempty"`
	Error       *string      `json:"error,omitempty"`
	CreatedAt   time.Time    `json:"created_at"`
	CompletedAt *time.Time   `json:"completed_at,omitempty"`
}

// ToResponse converts an AuditExport to its external representation.
func (e *AuditExport) ToResponse() ExportResponse {
	return ExportResponse{
		ID:          e.ID,
		StartDate:   e.StartDate,
		EndDate:     e.EndDate,
		Format:      e.Format,
		Status:      e.Status,
		FileID:      e.FileID,
		Error:       e.ErrorMessage,
		CreatedAt:   e.CreatedAt,
		CompletedAt: e.CompletedAt,
	}
}
