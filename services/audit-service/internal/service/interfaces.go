package service

import (
	"context"
	"time"

	"github.com/google/uuid"
	"github.com/hightrex/microservices-platform/services/audit-service/internal/models"
)

// AuditRepository defines the interface for audit log data access.
// NO UPDATE or DELETE methods — audit logs are immutable (except retention-based purging).
type AuditRepository interface {
	Create(ctx context.Context, log *models.AuditLog) error
	GetByID(ctx context.Context, id uuid.UUID) (*models.AuditLog, error)
	Search(ctx context.Context, filter models.SearchRequest, page models.Pagination) ([]models.AuditLog, int, error)
	GetLastHash(ctx context.Context, tenantID uuid.UUID) (*string, error)
	GetLogsForVerification(ctx context.Context, tenantID uuid.UUID, start, end time.Time) ([]models.AuditLog, error)
	GetStatistics(ctx context.Context, tenantID uuid.UUID) (*models.Statistics, error)
	// PurgeExpiredByEventType deletes audit logs older than the given cutoff date
	// for a specific event type and tenant. Used exclusively by the retention enforcement worker.
	PurgeExpiredByEventType(ctx context.Context, tenantID uuid.UUID, eventType string, olderThan time.Time) (int64, error)
}

// RetentionRepository defines the interface for retention policy data access.
type RetentionRepository interface {
	Create(ctx context.Context, policy *models.RetentionPolicy) error
	GetByEventType(ctx context.Context, eventType string) (*models.RetentionPolicy, error)
	List(ctx context.Context) ([]models.RetentionPolicy, error)
	// ListAllActive returns all active retention policies across all tenants.
	// Used by the background retention worker which runs outside of any tenant context.
	ListAllActive(ctx context.Context) ([]models.RetentionPolicy, error)
	Update(ctx context.Context, id uuid.UUID, fields map[string]interface{}) error
	Delete(ctx context.Context, id uuid.UUID) error
}

// ExportRepository defines the interface for export job data access.
type ExportRepository interface {
	Create(ctx context.Context, export *models.AuditExport) error
	GetByID(ctx context.Context, id uuid.UUID) (*models.AuditExport, error)
	List(ctx context.Context, page models.Pagination) ([]models.AuditExport, int, error)
	UpdateStatus(ctx context.Context, id uuid.UUID, status models.ExportStatus, fileID *uuid.UUID, errMsg *string) error
}

// EventPublisher defines the interface for publishing domain events.
type EventPublisher interface {
	Publish(ctx context.Context, stream string, eventType string, data interface{}) error
}
