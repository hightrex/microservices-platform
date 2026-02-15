package service

import (
	"context"

	"github.com/google/uuid"
	"github.com/hightrex/microservices-platform/libs/go/pkg/errors"
	"github.com/hightrex/microservices-platform/services/audit-service/internal/models"
)

// RetentionService handles retention policy business logic.
type RetentionService struct {
	retentionRepo RetentionRepository
	publisher     EventPublisher
}

// NewRetentionService creates a new RetentionService.
func NewRetentionService(repo RetentionRepository, publisher EventPublisher) *RetentionService {
	return &RetentionService{
		retentionRepo: repo,
		publisher:     publisher,
	}
}

// GetPolicies retrieves all retention policies for the tenant.
func (s *RetentionService) GetPolicies(ctx context.Context) ([]models.PolicyResponse, error) {
	policies, err := s.retentionRepo.List(ctx)
	if err != nil {
		return nil, err
	}

	responses := make([]models.PolicyResponse, len(policies))
	for i, p := range policies {
		responses[i] = p.ToResponse()
	}
	return responses, nil
}

// CreatePolicy creates a new retention policy.
func (s *RetentionService) CreatePolicy(ctx context.Context, req models.CreatePolicyRequest) (*models.PolicyResponse, error) {
	if req.RetentionDays < 30 {
		return nil, errors.BadRequest("Retention days must be at least 30", nil)
	}
	if req.RetentionDays > 2555 {
		return nil, errors.BadRequest("Retention days must not exceed 2555 (7 years)", nil)
	}

	policy := &models.RetentionPolicy{
		EventType:     req.EventType,
		RetentionDays: req.RetentionDays,
	}

	if err := s.retentionRepo.Create(ctx, policy); err != nil {
		return nil, err
	}

	resp := policy.ToResponse()
	return &resp, nil
}

// UpdatePolicy performs a partial update on a retention policy.
func (s *RetentionService) UpdatePolicy(ctx context.Context, id uuid.UUID, req models.UpdatePolicyRequest) error {
	fields := make(map[string]interface{})

	if req.RetentionDays != nil {
		if *req.RetentionDays < 30 || *req.RetentionDays > 2555 {
			return errors.BadRequest("Retention days must be between 30 and 2555", nil)
		}
		fields["retention_days"] = *req.RetentionDays
	}
	if req.IsActive != nil {
		fields["is_active"] = *req.IsActive
	}

	return s.retentionRepo.Update(ctx, id, fields)
}

// DeletePolicy removes a retention policy.
func (s *RetentionService) DeletePolicy(ctx context.Context, id uuid.UUID) error {
	return s.retentionRepo.Delete(ctx, id)
}
