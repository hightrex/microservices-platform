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

func TestMemberService_Invite_Success(t *testing.T) {
	orgID := uuid.New()
	inviterID := uuid.New()
	events := &mockEventPublisher{}

	orgRepo := &mockOrgRepo{
		getByIDFn: func(ctx context.Context, id uuid.UUID) (*models.Organization, error) {
			return &models.Organization{
				ID:       orgID,
				MaxUsers: 10,
				Plan:     models.PlanStarter,
				Status:   models.OrgStatusActive,
				Settings: json.RawMessage("{}"),
			}, nil
		},
	}
	memberRepo := &mockMemberRepo{
		countActiveFn: func(ctx context.Context, id uuid.UUID) (int, error) {
			return 2, nil // Under limit
		},
	}

	svc := NewMemberService(memberRepo, orgRepo, events)

	ctx := tenant.NewContext(context.Background(), orgID)
	req := models.InviteMemberRequest{
		UserID: uuid.New(),
		Role:   "member",
	}

	resp, err := svc.Invite(ctx, orgID, req, inviterID, "127.0.0.1")
	require.NoError(t, err)
	assert.NotNil(t, resp)
	assert.Equal(t, models.MemberStatusInvited, resp.Status)
	assert.Len(t, events.events, 1)
	assert.Equal(t, "org.member_invited", events.events[0].EventType)
}

func TestMemberService_Invite_ExceedsPlanLimit(t *testing.T) {
	orgID := uuid.New()
	orgRepo := &mockOrgRepo{
		getByIDFn: func(ctx context.Context, id uuid.UUID) (*models.Organization, error) {
			return &models.Organization{
				ID:       orgID,
				MaxUsers: 5,
				Plan:     models.PlanFree,
				Status:   models.OrgStatusActive,
				Settings: json.RawMessage("{}"),
			}, nil
		},
	}
	memberRepo := &mockMemberRepo{
		countActiveFn: func(ctx context.Context, id uuid.UUID) (int, error) {
			return 5, nil // At limit
		},
	}

	svc := NewMemberService(memberRepo, orgRepo, &mockEventPublisher{})

	ctx := tenant.NewContext(context.Background(), orgID)
	req := models.InviteMemberRequest{
		UserID: uuid.New(),
		Role:   "member",
	}

	resp, err := svc.Invite(ctx, orgID, req, uuid.New(), "127.0.0.1")
	assert.Error(t, err)
	assert.Nil(t, resp)
	assert.Contains(t, err.Error(), "member limit")
}

func TestMemberService_List_Success(t *testing.T) {
	orgID := uuid.New()
	now := time.Now()
	memberRepo := &mockMemberRepo{
		listFn: func(ctx context.Context, id uuid.UUID, filter models.MemberFilter, page models.Pagination) ([]models.Member, int, error) {
			return []models.Member{
				{ID: uuid.New(), UserID: uuid.New(), Role: "member", Status: models.MemberStatusActive, CreatedAt: now},
				{ID: uuid.New(), UserID: uuid.New(), Role: "org_admin", Status: models.MemberStatusActive, CreatedAt: now},
			}, 2, nil
		},
	}

	svc := NewMemberService(memberRepo, &mockOrgRepo{}, &mockEventPublisher{})

	ctx := tenant.NewContext(context.Background(), orgID)
	resp, err := svc.List(ctx, orgID, models.MemberFilter{}, models.DefaultPagination())
	require.NoError(t, err)
	assert.Equal(t, 2, resp.TotalCount)
}

func TestMemberService_Remove_Success(t *testing.T) {
	orgID := uuid.New()
	events := &mockEventPublisher{}

	svc := NewMemberService(&mockMemberRepo{}, &mockOrgRepo{}, events)

	ctx := tenant.NewContext(context.Background(), orgID)
	err := svc.Remove(ctx, orgID, uuid.New(), "127.0.0.1")
	require.NoError(t, err)
	assert.Len(t, events.events, 1)
	assert.Equal(t, "org.member_removed", events.events[0].EventType)
}

func TestMemberService_Remove_NotFound(t *testing.T) {
	orgID := uuid.New()
	memberRepo := &mockMemberRepo{
		removeFn: func(ctx context.Context, oid, uid uuid.UUID) error {
			return errors.NotFound("Member not found", nil)
		},
	}

	svc := NewMemberService(memberRepo, &mockOrgRepo{}, &mockEventPublisher{})

	ctx := tenant.NewContext(context.Background(), orgID)
	err := svc.Remove(ctx, orgID, uuid.New(), "127.0.0.1")
	assert.Error(t, err)
}
