package service

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/hightrex/microservices-platform/libs/go/pkg/errors"
	"github.com/hightrex/microservices-platform/libs/go/pkg/tenant"
	"github.com/hightrex/microservices-platform/services/organization-service/internal/models"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestOrgService_Create_Success(t *testing.T) {
	orgRepo := &mockOrgRepo{}
	moduleRepo := &mockModuleRepo{}
	planRepo := &mockPlanRepo{}
	memberRepo := &mockMemberRepo{}
	cache := &mockModuleCache{}
	events := &mockEventPublisher{}

	svc := NewOrgService(orgRepo, moduleRepo, planRepo, memberRepo, cache, events)

	ctx := context.Background()
	ownerID := uuid.New()

	req := models.CreateOrgRequest{
		Name: "Test Org",
		Slug: "testorg",
	}

	resp, err := svc.Create(ctx, req, ownerID, "127.0.0.1")
	require.NoError(t, err)
	assert.NotNil(t, resp)
	assert.Equal(t, "Test Org", resp.Name)
	assert.Equal(t, "testorg", resp.Slug)
	assert.Equal(t, models.PlanFree, resp.Plan)
	assert.Equal(t, models.OrgStatusActive, resp.Status)

	// Verify event was published
	assert.Len(t, events.events, 1)
	assert.Equal(t, "org.created", events.events[0].EventType)
}

func TestOrgService_Create_ValidationFailure(t *testing.T) {
	svc := NewOrgService(&mockOrgRepo{}, &mockModuleRepo{}, &mockPlanRepo{}, &mockMemberRepo{}, &mockModuleCache{}, &mockEventPublisher{})

	ctx := context.Background()
	ownerID := uuid.New()

	// Missing required name
	req := models.CreateOrgRequest{
		Name: "",
		Slug: "testorg",
	}

	resp, err := svc.Create(ctx, req, ownerID, "127.0.0.1")
	assert.Error(t, err)
	assert.Nil(t, resp)
}

func TestOrgService_GetByID_Success(t *testing.T) {
	orgID := uuid.New()
	orgRepo := &mockOrgRepo{
		getByIDFn: func(ctx context.Context, id uuid.UUID) (*models.Organization, error) {
			return &models.Organization{
				ID:              orgID,
				Name:            "Test Org",
				Slug:            "testorg",
				OwnerUserID:     uuid.New(),
				Plan:            models.PlanFree,
				Status:          models.OrgStatusActive,
				Settings:        json.RawMessage("{}"),
				MaxUsers:        5,
				MaxStorageBytes: 1073741824,
				CreatedAt:       time.Now(),
				UpdatedAt:       time.Now(),
			}, nil
		},
	}
	cache := &mockModuleCache{}
	moduleRepo := &mockModuleRepo{
		getByOrgFn: func(ctx context.Context, id uuid.UUID) ([]models.Module, error) {
			return []models.Module{
				{ID: uuid.New(), ModuleName: models.ModuleAuditLogging, Enabled: false},
			}, nil
		},
	}

	svc := NewOrgService(orgRepo, moduleRepo, &mockPlanRepo{}, &mockMemberRepo{}, cache, &mockEventPublisher{})

	ctx := tenant.NewContext(context.Background(), orgID)
	resp, err := svc.GetByID(ctx, orgID)
	require.NoError(t, err)
	assert.Equal(t, "Test Org", resp.Name)
}

func TestOrgService_GetByID_NotFound(t *testing.T) {
	orgRepo := &mockOrgRepo{
		getByIDFn: func(ctx context.Context, id uuid.UUID) (*models.Organization, error) {
			return nil, errors.NotFound("Organization not found", nil)
		},
	}

	svc := NewOrgService(orgRepo, &mockModuleRepo{}, &mockPlanRepo{}, &mockMemberRepo{}, &mockModuleCache{}, &mockEventPublisher{})

	ctx := tenant.NewContext(context.Background(), uuid.New())
	resp, err := svc.GetByID(ctx, uuid.New())
	assert.Error(t, err)
	assert.Nil(t, resp)
}

func TestOrgService_Update_Success(t *testing.T) {
	orgID := uuid.New()
	updatedName := "Updated Org"

	orgRepo := &mockOrgRepo{
		getByIDFn: func(ctx context.Context, id uuid.UUID) (*models.Organization, error) {
			return &models.Organization{
				ID:       orgID,
				Name:     updatedName,
				Slug:     "testorg",
				Plan:     models.PlanFree,
				Status:   models.OrgStatusActive,
				Settings: json.RawMessage("{}"),
			}, nil
		},
	}
	moduleRepo := &mockModuleRepo{}
	cache := &mockModuleCache{}
	events := &mockEventPublisher{}

	svc := NewOrgService(orgRepo, moduleRepo, &mockPlanRepo{}, &mockMemberRepo{}, cache, events)

	ctx := tenant.NewContext(context.Background(), orgID)
	req := models.UpdateOrgRequest{Name: &updatedName}

	resp, err := svc.Update(ctx, orgID, req, "127.0.0.1")
	require.NoError(t, err)
	assert.Equal(t, updatedName, resp.Name)
	assert.Len(t, events.events, 1)
	assert.Equal(t, "org.updated", events.events[0].EventType)
}

func TestOrgService_Update_NoFields(t *testing.T) {
	svc := NewOrgService(&mockOrgRepo{}, &mockModuleRepo{}, &mockPlanRepo{}, &mockMemberRepo{}, &mockModuleCache{}, &mockEventPublisher{})

	ctx := tenant.NewContext(context.Background(), uuid.New())
	req := models.UpdateOrgRequest{}

	resp, err := svc.Update(ctx, uuid.New(), req, "127.0.0.1")
	assert.Error(t, err)
	assert.Nil(t, resp)
}

func TestOrgService_Delete_Success(t *testing.T) {
	orgID := uuid.New()
	events := &mockEventPublisher{}
	cache := &mockModuleCache{}

	svc := NewOrgService(&mockOrgRepo{}, &mockModuleRepo{}, &mockPlanRepo{}, &mockMemberRepo{}, cache, events)

	ctx := tenant.NewContext(context.Background(), orgID)
	err := svc.Delete(ctx, orgID, "127.0.0.1")
	require.NoError(t, err)
	assert.Len(t, events.events, 1)
	assert.Equal(t, "org.deleted", events.events[0].EventType)
}
