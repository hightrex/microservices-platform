package postgres

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/hightrex/microservices-platform/libs/go/pkg/errors"
	"github.com/hightrex/microservices-platform/libs/go/pkg/tenant"
	"github.com/hightrex/microservices-platform/services/audit-service/internal/models"
)

// ExportRepo implements service.ExportRepository using PostgreSQL.
type ExportRepo struct {
	db *pgxpool.Pool
}

// NewExportRepo creates a new ExportRepo.
func NewExportRepo(db *pgxpool.Pool) *ExportRepo {
	return &ExportRepo{db: db}
}

// Create inserts a new export job record.
func (r *ExportRepo) Create(ctx context.Context, export *models.AuditExport) error {
	tenantID, err := tenant.RequireTenant(ctx)
	if err != nil {
		return err
	}

	export.ID = uuid.New()
	export.TenantID = tenantID
	export.Status = models.ExportStatusPending
	export.CreatedAt = time.Now()

	_, err = r.db.Exec(ctx,
		`INSERT INTO audit_exports (id, tenant_id, requested_by, start_date, end_date, format, status, created_at)
		 VALUES ($1, $2, $3, $4, $5, $6, $7, $8)`,
		export.ID, export.TenantID, export.RequestedBy, export.StartDate,
		export.EndDate, export.Format, export.Status, export.CreatedAt,
	)
	if err != nil {
		return errors.InternalServerError("Failed to create export job", err)
	}
	return nil
}

// GetByID retrieves an export job by ID, scoped to tenant.
func (r *ExportRepo) GetByID(ctx context.Context, id uuid.UUID) (*models.AuditExport, error) {
	tenantID, err := tenant.RequireTenant(ctx)
	if err != nil {
		return nil, err
	}

	var export models.AuditExport
	err = r.db.QueryRow(ctx,
		`SELECT id, tenant_id, requested_by, start_date, end_date, format, status, file_id, error_message, created_at, completed_at
		 FROM audit_exports WHERE id = $1 AND tenant_id = $2`,
		id, tenantID,
	).Scan(
		&export.ID, &export.TenantID, &export.RequestedBy, &export.StartDate,
		&export.EndDate, &export.Format, &export.Status, &export.FileID,
		&export.ErrorMessage, &export.CreatedAt, &export.CompletedAt,
	)
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, errors.NotFound("Export job not found", nil)
		}
		return nil, errors.InternalServerError("Failed to get export job", err)
	}
	return &export, nil
}

// List retrieves export jobs with pagination, scoped to tenant.
func (r *ExportRepo) List(ctx context.Context, page models.Pagination) ([]models.AuditExport, int, error) {
	tenantID, err := tenant.RequireTenant(ctx)
	if err != nil {
		return nil, 0, err
	}

	var total int
	if err := r.db.QueryRow(ctx,
		"SELECT COUNT(*) FROM audit_exports WHERE tenant_id = $1", tenantID,
	).Scan(&total); err != nil {
		return nil, 0, errors.InternalServerError("Failed to count exports", err)
	}

	offset := (page.Page - 1) * page.PageSize
	rows, err := r.db.Query(ctx,
		fmt.Sprintf(`SELECT id, tenant_id, requested_by, start_date, end_date, format, status, file_id, error_message, created_at, completed_at
		 FROM audit_exports WHERE tenant_id = $1 ORDER BY created_at DESC LIMIT $2 OFFSET $3`),
		tenantID, page.PageSize, offset,
	)
	if err != nil {
		return nil, 0, errors.InternalServerError("Failed to list exports", err)
	}
	defer rows.Close()

	var exports []models.AuditExport
	for rows.Next() {
		var e models.AuditExport
		if err := rows.Scan(
			&e.ID, &e.TenantID, &e.RequestedBy, &e.StartDate,
			&e.EndDate, &e.Format, &e.Status, &e.FileID,
			&e.ErrorMessage, &e.CreatedAt, &e.CompletedAt,
		); err != nil {
			return nil, 0, errors.InternalServerError("Failed to scan export", err)
		}
		exports = append(exports, e)
	}

	return exports, total, nil
}

// UpdateStatus updates the status of an export job.
func (r *ExportRepo) UpdateStatus(ctx context.Context, id uuid.UUID, status models.ExportStatus, fileID *uuid.UUID, errMsg *string) error {
	tenantID, err := tenant.RequireTenant(ctx)
	if err != nil {
		return err
	}

	var completedAt *time.Time
	if status == models.ExportStatusCompleted || status == models.ExportStatusFailed {
		now := time.Now()
		completedAt = &now
	}

	tag, err := r.db.Exec(ctx,
		`UPDATE audit_exports SET status = $1, file_id = $2, error_message = $3, completed_at = $4
		 WHERE id = $5 AND tenant_id = $6`,
		status, fileID, errMsg, completedAt, id, tenantID,
	)
	if err != nil {
		return errors.InternalServerError("Failed to update export status", err)
	}
	if tag.RowsAffected() == 0 {
		return errors.NotFound("Export job not found", nil)
	}
	return nil
}
