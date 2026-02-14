package service

import (
	"context"

	"github.com/google/uuid"
	"github.com/hightrex/microservices-platform/libs/go/pkg/errors"
	"github.com/hightrex/microservices-platform/libs/go/pkg/logger"
	"github.com/hightrex/microservices-platform/services/organization-service/internal/models"
)

// ModuleService handles module business logic.
type ModuleService struct {
	modules ModuleRepository
	orgs    OrgRepository
	plans   PlanRepository
	cache   ModuleCache
	events  EventPublisher
}

// NewModuleService creates a new ModuleService.
func NewModuleService(
	modules ModuleRepository,
	orgs OrgRepository,
	plans PlanRepository,
	cache ModuleCache,
	events EventPublisher,
) *ModuleService {
	return &ModuleService{
		modules: modules,
		orgs:    orgs,
		plans:   plans,
		cache:   cache,
		events:  events,
	}
}

// GetModules returns all modules with their status for an organization.
// Used by the API Gateway to check module availability.
func (s *ModuleService) GetModules(ctx context.Context, orgID uuid.UUID) ([]models.ModuleResponse, error) {
	// Try cache first
	cached, err := s.cache.GetModules(ctx, orgID)
	if err == nil && cached != nil {
		return cached, nil
	}

	// Fallback to DB
	modules, err := s.modules.GetByOrg(ctx, orgID)
	if err != nil {
		return nil, err
	}

	responses := make([]models.ModuleResponse, 0, len(modules))
	for _, m := range modules {
		responses = append(responses, m.ToResponse())
	}

	// Cache for next time
	if cacheErr := s.cache.SetModules(ctx, orgID, responses); cacheErr != nil {
		logger.Warn().Err(cacheErr).Msg("Failed to cache modules")
	}

	return responses, nil
}

// ToggleModule enables or disables a module for an organization.
// Checks that the org's plan allows the module.
func (s *ModuleService) ToggleModule(ctx context.Context, orgID uuid.UUID, moduleName string, enabled bool) error {
	// Check if the plan allows this module
	if enabled {
		org, err := s.orgs.GetByID(ctx, orgID)
		if err != nil {
			return err
		}

		plan, err := s.plans.GetByName(ctx, string(org.Plan))
		if err != nil {
			return errors.InternalServerError("Failed to get plan details", err)
		}

		allowed := false
		for _, m := range plan.AvailableModules {
			if m == moduleName {
				allowed = true
				break
			}
		}
		if !allowed {
			return errors.Forbidden("Module not available on your current plan. Please upgrade.", nil)
		}
	}

	if err := s.modules.Toggle(ctx, orgID, moduleName, enabled); err != nil {
		return err
	}

	// Invalidate cache
	if err := s.cache.Invalidate(ctx, orgID); err != nil {
		logger.Warn().Err(err).Msg("Failed to invalidate module cache")
	}

	// Publish event
	if err := s.events.Publish(ctx, orgEventsStream, "org.module_toggled", map[string]interface{}{
		"org_id":      orgID.String(),
		"module_name": moduleName,
		"enabled":     enabled,
	}); err != nil {
		logger.Warn().Err(err).Msg("Failed to publish org.module_toggled event")
	}

	return nil
}

// GetModuleConfig retrieves per-module settings.
func (s *ModuleService) GetModuleConfig(ctx context.Context, orgID uuid.UUID, moduleName string) (*models.ModuleResponse, error) {
	module, err := s.modules.GetConfig(ctx, orgID, moduleName)
	if err != nil {
		return nil, err
	}

	resp := module.ToResponse()
	return &resp, nil
}
