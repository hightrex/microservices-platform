package service

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/hightrex/microservices-platform/libs/go/pkg/tenant"
	"github.com/hightrex/microservices-platform/services/organization-service/internal/models"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestModuleService_GetModules_CacheHit(t *testing.T) {
	orgID := uuid.New()
	cachedModules := []models.ModuleResponse{
		{ID: uuid.New(), ModuleName: models.ModuleAuditLogging, Enabled: true},
	}

	cache := &mockModuleCache{
		getModulesFn: func(ctx context.Context, id uuid.UUID) ([]models.ModuleResponse, error) {
			return cachedModules, nil
		},
	}

	svc := NewModuleService(&mockModuleRepo{}, &mockOrgRepo{}, &mockPlanRepo{}, cache, &mockEventPublisher{})

	ctx := tenant.NewContext(context.Background(), orgID)
	result, err := svc.GetModules(ctx, orgID)
	require.NoError(t, err)
	assert.Len(t, result, 1)
	assert.Equal(t, models.ModuleAuditLogging, result[0].ModuleName)
}

func TestModuleService_GetModules_CacheMiss_FallsToDB(t *testing.T) {
	orgID := uuid.New()
	dbModules := []models.Module{
		{
			ID:         uuid.New(),
			OrgID:      orgID,
			ModuleName: models.ModuleNotifications,
			Enabled:    false,
			Config:     json.RawMessage("{}"),
			CreatedAt:  time.Now(),
			UpdatedAt:  time.Now(),
		},
	}

	moduleRepo := &mockModuleRepo{
		getByOrgFn: func(ctx context.Context, id uuid.UUID) ([]models.Module, error) {
			return dbModules, nil
		},
	}
	cache := &mockModuleCache{} // Will return error (cache miss)
	cacheCalled := false
	cache.setModulesFn = func(ctx context.Context, id uuid.UUID, modules []models.ModuleResponse) error {
		cacheCalled = true
		return nil
	}

	svc := NewModuleService(moduleRepo, &mockOrgRepo{}, &mockPlanRepo{}, cache, &mockEventPublisher{})

	ctx := tenant.NewContext(context.Background(), orgID)
	result, err := svc.GetModules(ctx, orgID)
	require.NoError(t, err)
	assert.Len(t, result, 1)
	assert.True(t, cacheCalled, "Cache should be populated after DB fetch")
}

func TestModuleService_ToggleModule_PlanAllows(t *testing.T) {
	orgID := uuid.New()

	orgRepo := &mockOrgRepo{
		getByIDFn: func(ctx context.Context, id uuid.UUID) (*models.Organization, error) {
			return &models.Organization{
				ID:   orgID,
				Plan: models.PlanBusiness,
			}, nil
		},
	}
	planRepo := &mockPlanRepo{
		getByNameFn: func(ctx context.Context, name string) (*models.Plan, error) {
			return &models.Plan{
				Name: "business",
				AvailableModules: []string{
					"notifications", "billing", "file_management", "audit_logging", "analytics",
				},
			}, nil
		},
	}
	events := &mockEventPublisher{}
	cache := &mockModuleCache{}

	svc := NewModuleService(&mockModuleRepo{}, orgRepo, planRepo, cache, events)

	ctx := tenant.NewContext(context.Background(), orgID)
	err := svc.ToggleModule(ctx, orgID, "notifications", true)
	require.NoError(t, err)
	assert.Len(t, events.events, 1)
	assert.Equal(t, "org.module_toggled", events.events[0].EventType)
}

func TestModuleService_ToggleModule_PlanDenies(t *testing.T) {
	orgID := uuid.New()

	orgRepo := &mockOrgRepo{
		getByIDFn: func(ctx context.Context, id uuid.UUID) (*models.Organization, error) {
			return &models.Organization{
				ID:   orgID,
				Plan: models.PlanFree,
			}, nil
		},
	}
	planRepo := &mockPlanRepo{
		getByNameFn: func(ctx context.Context, name string) (*models.Plan, error) {
			return &models.Plan{
				Name:             "free",
				AvailableModules: []string{"audit_logging"},
			}, nil
		},
	}

	svc := NewModuleService(&mockModuleRepo{}, orgRepo, planRepo, &mockModuleCache{}, &mockEventPublisher{})

	ctx := tenant.NewContext(context.Background(), orgID)
	err := svc.ToggleModule(ctx, orgID, "notifications", true)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "not available on your current plan")
}

func TestModuleService_ToggleModule_Disable_NoPlanCheck(t *testing.T) {
	orgID := uuid.New()
	events := &mockEventPublisher{}

	svc := NewModuleService(&mockModuleRepo{}, &mockOrgRepo{}, &mockPlanRepo{}, &mockModuleCache{}, events)

	ctx := tenant.NewContext(context.Background(), orgID)
	err := svc.ToggleModule(ctx, orgID, "notifications", false)
	require.NoError(t, err)
	assert.Len(t, events.events, 1)
}
