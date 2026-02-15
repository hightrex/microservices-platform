package service

import (
	"context"
	"time"

	"github.com/google/uuid"
	"github.com/hightrex/microservices-platform/libs/go/pkg/logger"
	"github.com/hightrex/microservices-platform/libs/go/pkg/tenant"
	"github.com/hightrex/microservices-platform/services/audit-service/internal/models"
)

// AuditService handles audit log business logic.
type AuditService struct {
	auditRepo           AuditRepository
	publisher           EventPublisher
	hashChainingEnabled bool
}

// NewAuditService creates a new AuditService.
func NewAuditService(repo AuditRepository, publisher EventPublisher, hashChainingEnabled bool) *AuditService {
	return &AuditService{
		auditRepo:           repo,
		publisher:           publisher,
		hashChainingEnabled: hashChainingEnabled,
	}
}

// CreateLog appends a new audit log entry with hash chaining.
func (s *AuditService) CreateLog(ctx context.Context, log *models.AuditLog) error {
	if log.Timestamp.IsZero() {
		log.Timestamp = time.Now()
	}

	if s.hashChainingEnabled {
		// Load last hash for tenant
		prevHash, err := s.auditRepo.GetLastHash(ctx, log.TenantID)
		if err != nil {
			logger.Error().Err(err).Str("tenant_id", log.TenantID.String()).Msg("Failed to get last hash for chain")
			// Continue without chaining rather than lose the audit event
		}

		log.PreviousHash = prevHash
		log.CurrentHash = GenerateHash(prevHash, log)
	} else {
		log.CurrentHash = GenerateHash(nil, log)
	}

	if err := s.auditRepo.Create(ctx, log); err != nil {
		return err
	}

	return nil
}

// Search queries audit logs with filtering and pagination.
func (s *AuditService) Search(ctx context.Context, filter models.SearchRequest, page models.Pagination) ([]models.AuditLogResponse, int, error) {
	logs, total, err := s.auditRepo.Search(ctx, filter, page)
	if err != nil {
		return nil, 0, err
	}

	responses := make([]models.AuditLogResponse, len(logs))
	for i, l := range logs {
		responses[i] = l.ToResponse()
	}
	return responses, total, nil
}

// GetByID retrieves a single audit log entry.
func (s *AuditService) GetByID(ctx context.Context, id uuid.UUID) (*models.AuditLogResponse, error) {
	log, err := s.auditRepo.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}
	resp := log.ToResponse()
	return &resp, nil
}

// VerifyIntegrity verifies the hash chain integrity for a tenant's audit logs in a date range.
func (s *AuditService) VerifyIntegrity(ctx context.Context, start, end time.Time) (*models.VerificationReport, error) {
	tenantID, err := tenant.RequireTenant(ctx)
	if err != nil {
		return nil, err
	}

	logs, err := s.auditRepo.GetLogsForVerification(ctx, tenantID, start, end)
	if err != nil {
		return nil, err
	}

	report := VerifyChain(logs)

	// If chain is broken, publish alert event
	if !report.Valid {
		if pubErr := s.publisher.Publish(ctx, "audit-events", "audit.chain_broken", map[string]interface{}{
			"tenant_id": tenantID,
			"detail":    report.ErrorDetail,
		}); pubErr != nil {
			logger.Warn().Err(pubErr).Msg("Failed to publish audit.chain_broken event")
		}
	}

	return report, nil
}

// GetStatistics retrieves event count statistics for the tenant.
func (s *AuditService) GetStatistics(ctx context.Context) (*models.Statistics, error) {
	tenantID, err := tenant.RequireTenant(ctx)
	if err != nil {
		return nil, err
	}

	return s.auditRepo.GetStatistics(ctx, tenantID)
}
