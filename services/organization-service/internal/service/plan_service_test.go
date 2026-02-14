package service

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/google/uuid"
	"github.com/hightrex/microservices-platform/libs/go/pkg/tenant"
	"github.com/hightrex/microservices-platform/services/organization-service/internal/models"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestPlanService_GetCurrent_Success(t *testing.T) {
	orgID := uuid.New()
	orgRepo := &mockOrgRepo{
		getByIDFn: func(ctx context.Context, id uuid.UUID) (*models.Organization, error) {
			return &models.Organization{
				ID:       orgID,
				Plan:     models.PlanStarter,
				Settings: json.RawMessage("{}"),
			}, nil
		},
	}
	planRepo := &mockPlanRepo{
		getByNameFn: func(ctx context.Context, name string) (*models.Plan, error) {
			return &models.Plan{
				ID:          uuid.New(),
				Name:        "starter",
				DisplayName: "Starter",
				MaxUsers:    25,
			}, nil
		},
	}

	svc := NewPlanService(planRepo, orgRepo, &mockModuleRepo{}, &mockModuleCache{}, &mockEventPublisher{})

	ctx := tenant.NewContext(context.Background(), orgID)
	resp, err := svc.GetCurrent(ctx, orgID)
	require.NoError(t, err)
	assert.Equal(t, "starter", resp.Name)
	assert.Equal(t, 25, resp.MaxUsers)
}

func TestPlanService_Change_Success(t *testing.T) {
	orgID := uuid.New()
	orgRepo := &mockOrgRepo{
		getByIDFn: func(ctx context.Context, id uuid.UUID) (*models.Organization, error) {
			return &models.Organization{
				ID:       orgID,
				Plan:     models.PlanFree,
				Settings: json.RawMessage("{}"),
			}, nil
		},
	}
	planRepo := &mockPlanRepo{
		getByNameFn: func(ctx context.Context, name string) (*models.Plan, error) {
			return &models.Plan{
				ID:               uuid.New(),
				Name:             "business",
				DisplayName:      "Business",
				MaxUsers:         100,
				MaxStorageBytes:  107374182400,
				AvailableModules: []string{"notifications", "billing", "file_management", "audit_logging", "analytics"},
			}, nil
		},
	}
	moduleRepo := &mockModuleRepo{
		getByOrgFn: func(ctx context.Context, id uuid.UUID) ([]models.Module, error) {
			return []models.Module{}, nil
		},
	}
	events := &mockEventPublisher{}
	cache := &mockModuleCache{}

	svc := NewPlanService(planRepo, orgRepo, moduleRepo, cache, events)

	ctx := tenant.NewContext(context.Background(), orgID)
	req := models.ChangePlanRequest{PlanName: "business"}

	resp, err := svc.Change(ctx, orgID, req, "127.0.0.1")
	require.NoError(t, err)
	assert.Equal(t, "business", resp.Name)
	assert.Len(t, events.events, 1)
	assert.Equal(t, "org.plan_changed", events.events[0].EventType)
}

func TestPlanService_Change_AlreadyOnPlan(t *testing.T) {
	orgID := uuid.New()
	orgRepo := &mockOrgRepo{
		getByIDFn: func(ctx context.Context, id uuid.UUID) (*models.Organization, error) {
			return &models.Organization{
				ID:       orgID,
				Plan:     models.PlanBusiness,
				Settings: json.RawMessage("{}"),
			}, nil
		},
	}

	svc := NewPlanService(&mockPlanRepo{}, orgRepo, &mockModuleRepo{}, &mockModuleCache{}, &mockEventPublisher{})

	ctx := tenant.NewContext(context.Background(), orgID)
	req := models.ChangePlanRequest{PlanName: "business"}

	resp, err := svc.Change(ctx, orgID, req, "127.0.0.1")
	assert.Error(t, err)
	assert.Nil(t, resp)
	assert.Contains(t, err.Error(), "already on this plan")
}

func TestPlanService_ListAvailable_Success(t *testing.T) {
	planRepo := &mockPlanRepo{
		listFn: func(ctx context.Context) ([]models.Plan, error) {
			return []models.Plan{
				{ID: uuid.New(), Name: "free", DisplayName: "Free"},
				{ID: uuid.New(), Name: "starter", DisplayName: "Starter"},
				{ID: uuid.New(), Name: "business", DisplayName: "Business"},
				{ID: uuid.New(), Name: "enterprise", DisplayName: "Enterprise"},
			}, nil
		},
	}

	svc := NewPlanService(planRepo, &mockOrgRepo{}, &mockModuleRepo{}, &mockModuleCache{}, &mockEventPublisher{})

	ctx := context.Background()
	plans, err := svc.ListAvailable(ctx)
	require.NoError(t, err)
	assert.Len(t, plans, 4)
}
