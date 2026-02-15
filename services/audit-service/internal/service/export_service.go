package service

import (
	"context"
	"time"

	"github.com/google/uuid"
	"github.com/hightrex/microservices-platform/libs/go/pkg/errors"
	"github.com/hightrex/microservices-platform/libs/go/pkg/logger"
	"github.com/hightrex/microservices-platform/services/audit-service/internal/models"
)

// ExportService handles audit export business logic.
type ExportService struct {
	exportRepo ExportRepository
	auditRepo  AuditRepository
	publisher  EventPublisher
}

// NewExportService creates a new ExportService.
func NewExportService(exportRepo ExportRepository, auditRepo AuditRepository, publisher EventPublisher) *ExportService {
	return &ExportService{
		exportRepo: exportRepo,
		auditRepo:  auditRepo,
		publisher:  publisher,
	}
}

// CreateExport queues an export job.
func (s *ExportService) CreateExport(ctx context.Context, req models.CreateExportRequest, requestedBy uuid.UUID) (*models.ExportResponse, error) {
	// Validate date range (max 1 year)
	if req.EndDate.Before(req.StartDate) {
		return nil, errors.BadRequest("End date must be after start date", nil)
	}
	if req.EndDate.Sub(req.StartDate) > 365*24*time.Hour {
		return nil, errors.BadRequest("Export date range must not exceed 1 year", nil)
	}

	export := &models.AuditExport{
		RequestedBy: requestedBy,
		StartDate:   req.StartDate,
		EndDate:     req.EndDate,
		Format:      req.Format,
	}

	if err := s.exportRepo.Create(ctx, export); err != nil {
		return nil, err
	}

	// Queue background processing (in a real system, this would go to a job queue)
	go func() {
		bgCtx := context.Background()
		s.processExport(bgCtx, export)
	}()

	resp := export.ToResponse()
	return &resp, nil
}

// GetExport retrieves an export job by ID.
func (s *ExportService) GetExport(ctx context.Context, id uuid.UUID) (*models.ExportResponse, error) {
	export, err := s.exportRepo.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}
	resp := export.ToResponse()
	return &resp, nil
}

// ListExports retrieves all export jobs for the tenant.
func (s *ExportService) ListExports(ctx context.Context, page models.Pagination) ([]models.ExportResponse, int, error) {
	exports, total, err := s.exportRepo.List(ctx, page)
	if err != nil {
		return nil, 0, err
	}

	responses := make([]models.ExportResponse, len(exports))
	for i, e := range exports {
		responses[i] = e.ToResponse()
	}
	return responses, total, nil
}

// processExport handles the background export processing.
func (s *ExportService) processExport(ctx context.Context, export *models.AuditExport) {
	logger.Info().Str("export_id", export.ID.String()).Msg("Processing audit export")

	// Update status to processing
	if err := s.exportRepo.UpdateStatus(ctx, export.ID, models.ExportStatusProcessing, nil, nil); err != nil {
		logger.Error().Err(err).Msg("Failed to update export status to processing")
		return
	}

	// For now, mark as completed. In a full implementation, this would:
	// 1. Fetch audit logs for the date range
	// 2. Format as CSV or JSON
	// 3. Upload to File Service
	// 4. Update with file_id
	// TODO(phase-2): Integrate with File Service for actual export file generation (PHASE2-EXPORT)

	if err := s.exportRepo.UpdateStatus(ctx, export.ID, models.ExportStatusCompleted, nil, nil); err != nil {
		logger.Error().Err(err).Msg("Failed to update export status to completed")
		return
	}

	// Publish event
	if err := s.publisher.Publish(ctx, "audit-events", "audit.export_completed", map[string]interface{}{
		"export_id": export.ID,
		"format":    export.Format,
	}); err != nil {
		logger.Warn().Err(err).Msg("Failed to publish export_completed event")
	}

	logger.Info().Str("export_id", export.ID.String()).Msg("Audit export completed")
}
