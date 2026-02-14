package service

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	"github.com/hightrex/microservices-platform/libs/go/pkg/errors"
	"github.com/hightrex/microservices-platform/libs/go/pkg/logger"
	"github.com/hightrex/microservices-platform/libs/go/pkg/securitylog"
	"github.com/hightrex/microservices-platform/libs/go/pkg/validation"
	"github.com/hightrex/microservices-platform/services/organization-service/internal/models"
)

const orgEventsStream = "org-events"

// OrgService handles organization business logic.
type OrgService struct {
	orgs    OrgRepository
	modules ModuleRepository
	plans   PlanRepository
	members MemberRepository
	cache   ModuleCache
	events  EventPublisher
}

// NewOrgService creates a new OrgService.
func NewOrgService(
	orgs OrgRepository,
	modules ModuleRepository,
	plans PlanRepository,
	members MemberRepository,
	cache ModuleCache,
	events EventPublisher,
) *OrgService {
	return &OrgService{
		orgs:    orgs,
		modules: modules,
		plans:   plans,
		members: members,
		cache:   cache,
		events:  events,
	}
}

// Create creates a new organization, sets the owner, and seeds default modules per plan.
func (s *OrgService) Create(ctx context.Context, req models.CreateOrgRequest, ownerUserID uuid.UUID, ip string) (*models.OrgResponse, error) {
	if errs := validation.Validate(req); errs != nil {
		return nil, errors.BadRequest("Validation failed", fmt.Errorf("%v", validation.ErrorResponse(errs)))
	}

	// Get the plan defaults
	planName := string(models.PlanFree)
	if req.Plan != "" {
		planName = string(req.Plan)
	}

	plan, err := s.plans.GetByName(ctx, planName)
	if err != nil {
		logger.Warn().Err(err).Msgf("Failed to get plan %s, using defaults", planName)
		plan = &models.Plan{MaxUsers: 5, MaxStorageBytes: 1073741824}
	}

	org := &models.Organization{
		Name:            req.Name,
		Slug:            req.Slug,
		OwnerUserID:     ownerUserID,
		Plan:            models.OrgPlan(planName),
		Status:          models.OrgStatusActive,
		Settings:        []byte("{}"),
		MaxUsers:        plan.MaxUsers,
		MaxStorageBytes: plan.MaxStorageBytes,
	}

	if err := s.orgs.Create(ctx, org); err != nil {
		return nil, err
	}

	// Log security event
	securitylog.Log(ctx, securitylog.Event{
		Type:     securitylog.EventTenantCreated,
		Outcome:  securitylog.OutcomeSuccess,
		ActorID:  ownerUserID.String(),
		TenantID: org.ID.String(),
		IP:       ip,
		Resource: fmt.Sprintf("organizations/%s", org.ID),
		Details: map[string]interface{}{
			"name": org.Name,
			"slug": org.Slug,
		},
	})

	// Publish event
	if err := s.events.Publish(ctx, orgEventsStream, "org.created", map[string]interface{}{
		"org_id":   org.ID.String(),
		"name":     org.Name,
		"slug":     org.Slug,
		"owner_id": ownerUserID.String(),
		"plan":     string(org.Plan),
	}); err != nil {
		logger.Warn().Err(err).Msg("Failed to publish org.created event")
	}

	resp := org.ToResponse()
	return &resp, nil
}

// GetByID retrieves an organization by ID with module summary.
func (s *OrgService) GetByID(ctx context.Context, id uuid.UUID) (*models.OrgResponse, error) {
	org, err := s.orgs.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}

	resp := org.ToResponse()

	// Try to get modules from cache first
	cached, err := s.cache.GetModules(ctx, id)
	if err == nil && cached != nil {
		resp.Modules = cached
	} else {
		// Fallback to DB
		modules, err := s.modules.GetByOrg(ctx, id)
		if err == nil {
			for _, m := range modules {
				resp.Modules = append(resp.Modules, m.ToResponse())
			}
			// Cache for next time
			if cacheErr := s.cache.SetModules(ctx, id, resp.Modules); cacheErr != nil {
				logger.Warn().Err(cacheErr).Msg("Failed to cache modules")
			}
		}
	}

	return &resp, nil
}

// Update updates organization settings.
func (s *OrgService) Update(ctx context.Context, id uuid.UUID, req models.UpdateOrgRequest, ip string) (*models.OrgResponse, error) {
	if errs := validation.Validate(req); errs != nil {
		return nil, errors.BadRequest("Validation failed", fmt.Errorf("%v", validation.ErrorResponse(errs)))
	}

	fields := make(map[string]interface{})
	if req.Name != nil {
		fields["name"] = *req.Name
	}
	if req.Settings != nil {
		fields["settings"] = *req.Settings
	}

	if len(fields) == 0 {
		return nil, errors.BadRequest("No fields to update", nil)
	}

	if err := s.orgs.Update(ctx, id, fields); err != nil {
		return nil, err
	}

	// Publish event
	if err := s.events.Publish(ctx, orgEventsStream, "org.updated", map[string]interface{}{
		"org_id": id.String(),
		"fields": fields,
	}); err != nil {
		logger.Warn().Err(err).Msg("Failed to publish org.updated event")
	}

	return s.GetByID(ctx, id)
}

// Delete soft-deletes an organization.
func (s *OrgService) Delete(ctx context.Context, id uuid.UUID, ip string) error {
	if err := s.orgs.Delete(ctx, id); err != nil {
		return err
	}

	// Invalidate cache
	if err := s.cache.Invalidate(ctx, id); err != nil {
		logger.Warn().Err(err).Msg("Failed to invalidate module cache on org delete")
	}

	securitylog.Log(ctx, securitylog.Event{
		Type:     securitylog.EventTenantDeleted,
		Outcome:  securitylog.OutcomeSuccess,
		TenantID: id.String(),
		IP:       ip,
		Resource: fmt.Sprintf("organizations/%s", id),
	})

	// Publish event
	if err := s.events.Publish(ctx, orgEventsStream, "org.deleted", map[string]interface{}{
		"org_id": id.String(),
	}); err != nil {
		logger.Warn().Err(err).Msg("Failed to publish org.deleted event")
	}

	return nil
}
