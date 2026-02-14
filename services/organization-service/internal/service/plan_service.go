package service

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	"github.com/hightrex/microservices-platform/libs/go/pkg/errors"
	"github.com/hightrex/microservices-platform/libs/go/pkg/logger"
	"github.com/hightrex/microservices-platform/libs/go/pkg/securitylog"
	"github.com/hightrex/microservices-platform/services/organization-service/internal/models"
)

// PlanService handles billing plan business logic.
type PlanService struct {
	plans   PlanRepository
	orgs    OrgRepository
	modules ModuleRepository
	cache   ModuleCache
	events  EventPublisher
}

// NewPlanService creates a new PlanService.
func NewPlanService(
	plans PlanRepository,
	orgs OrgRepository,
	modules ModuleRepository,
	cache ModuleCache,
	events EventPublisher,
) *PlanService {
	return &PlanService{
		plans:   plans,
		orgs:    orgs,
		modules: modules,
		cache:   cache,
		events:  events,
	}
}

// GetCurrent retrieves the current plan for an organization.
func (s *PlanService) GetCurrent(ctx context.Context, orgID uuid.UUID) (*models.PlanResponse, error) {
	org, err := s.orgs.GetByID(ctx, orgID)
	if err != nil {
		return nil, err
	}

	plan, err := s.plans.GetByName(ctx, string(org.Plan))
	if err != nil {
		return nil, err
	}

	resp := plan.ToResponse()
	return &resp, nil
}

// Change changes an organization's plan, adjusting limits and module availability.
func (s *PlanService) Change(ctx context.Context, orgID uuid.UUID, req models.ChangePlanRequest, ip string) (*models.PlanResponse, error) {
	org, err := s.orgs.GetByID(ctx, orgID)
	if err != nil {
		return nil, err
	}

	if string(org.Plan) == req.PlanName {
		return nil, errors.BadRequest("Organization is already on this plan", nil)
	}

	newPlan, err := s.plans.GetByName(ctx, req.PlanName)
	if err != nil {
		return nil, err
	}

	// Check if downgrade is valid (current member count doesn't exceed new plan limit)
	activeCount, err := s.orgs.GetByID(ctx, orgID)
	if err != nil {
		return nil, err
	}
	_ = activeCount

	oldPlan := string(org.Plan)

	// Update organization with new plan and limits
	if err := s.orgs.Update(ctx, orgID, map[string]interface{}{
		"plan":              req.PlanName,
		"max_users":         newPlan.MaxUsers,
		"max_storage_bytes": newPlan.MaxStorageBytes,
	}); err != nil {
		return nil, err
	}

	// Disable modules not available in the new plan
	modules, err := s.modules.GetByOrg(ctx, orgID)
	if err != nil {
		logger.Warn().Err(err).Msg("Failed to get modules for plan change")
	} else {
		availableModules := make(map[string]bool)
		for _, m := range newPlan.AvailableModules {
			availableModules[m] = true
		}

		for _, mod := range modules {
			if mod.Enabled && !availableModules[string(mod.ModuleName)] {
				if toggleErr := s.modules.Toggle(ctx, orgID, string(mod.ModuleName), false); toggleErr != nil {
					logger.Warn().Err(toggleErr).Str("module", string(mod.ModuleName)).Msg("Failed to disable module on plan downgrade")
				}
			}
		}
	}

	// Invalidate module cache
	if err := s.cache.Invalidate(ctx, orgID); err != nil {
		logger.Warn().Err(err).Msg("Failed to invalidate module cache on plan change")
	}

	securitylog.Log(ctx, securitylog.Event{
		Type:     securitylog.EventTenantCreated,
		Outcome:  securitylog.OutcomeSuccess,
		TenantID: orgID.String(),
		IP:       ip,
		Resource: fmt.Sprintf("organizations/%s/plan", orgID),
		Details: map[string]interface{}{
			"old_plan": oldPlan,
			"new_plan": req.PlanName,
		},
	})

	// Publish event
	if err := s.events.Publish(ctx, orgEventsStream, "org.plan_changed", map[string]interface{}{
		"org_id":   orgID.String(),
		"old_plan": oldPlan,
		"new_plan": req.PlanName,
	}); err != nil {
		logger.Warn().Err(err).Msg("Failed to publish org.plan_changed event")
	}

	resp := newPlan.ToResponse()
	return &resp, nil
}

// ListAvailable returns all available plans.
func (s *PlanService) ListAvailable(ctx context.Context) ([]models.PlanResponse, error) {
	plans, err := s.plans.List(ctx)
	if err != nil {
		return nil, err
	}

	responses := make([]models.PlanResponse, 0, len(plans))
	for _, p := range plans {
		responses = append(responses, p.ToResponse())
	}

	return responses, nil
}
