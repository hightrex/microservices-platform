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

// MemberService handles member management business logic.
type MemberService struct {
	members MemberRepository
	orgs    OrgRepository
	events  EventPublisher
}

// NewMemberService creates a new MemberService.
func NewMemberService(
	members MemberRepository,
	orgs OrgRepository,
	events EventPublisher,
) *MemberService {
	return &MemberService{
		members: members,
		orgs:    orgs,
		events:  events,
	}
}

// Invite adds a new member to an organization, checking plan limits.
func (s *MemberService) Invite(ctx context.Context, orgID uuid.UUID, req models.InviteMemberRequest, invitedBy uuid.UUID, ip string) (*models.MemberResponse, error) {
	if errs := validation.Validate(req); errs != nil {
		return nil, errors.BadRequest("Validation failed", fmt.Errorf("%v", validation.ErrorResponse(errs)))
	}

	// Check plan limit for max users
	org, err := s.orgs.GetByID(ctx, orgID)
	if err != nil {
		return nil, err
	}

	activeCount, err := s.members.CountActive(ctx, orgID)
	if err != nil {
		return nil, errors.InternalServerError("Failed to check member count", err)
	}

	if activeCount >= org.MaxUsers {
		return nil, errors.Forbidden(
			fmt.Sprintf("Organization has reached its member limit (%d). Please upgrade your plan.", org.MaxUsers), nil)
	}

	member := &models.Member{
		UserID:    req.UserID,
		Role:      req.Role,
		InvitedBy: &invitedBy,
		Status:    models.MemberStatusInvited,
	}

	if err := s.members.Add(ctx, orgID, member); err != nil {
		return nil, err
	}

	securitylog.Log(ctx, securitylog.Event{
		Type:     securitylog.EventAccountCreated,
		Outcome:  securitylog.OutcomeSuccess,
		ActorID:  invitedBy.String(),
		TenantID: orgID.String(),
		IP:       ip,
		Resource: fmt.Sprintf("organizations/%s/members/%s", orgID, req.UserID),
		Details: map[string]interface{}{
			"role":    req.Role,
			"user_id": req.UserID.String(),
		},
	})

	// Publish event
	if err := s.events.Publish(ctx, orgEventsStream, "org.member_invited", map[string]interface{}{
		"org_id":     orgID.String(),
		"user_id":    req.UserID.String(),
		"role":       req.Role,
		"invited_by": invitedBy.String(),
	}); err != nil {
		logger.Warn().Err(err).Msg("Failed to publish org.member_invited event")
	}

	resp := member.ToResponse()
	return &resp, nil
}

// List retrieves members with filtering and pagination.
func (s *MemberService) List(ctx context.Context, orgID uuid.UUID, filter models.MemberFilter, page models.Pagination) (*models.PaginatedResponse, error) {
	members, total, err := s.members.List(ctx, orgID, filter, page)
	if err != nil {
		return nil, err
	}

	responses := make([]models.MemberResponse, 0, len(members))
	for i := range members {
		responses = append(responses, members[i].ToResponse())
	}

	totalPages := total / page.PageSize
	if total%page.PageSize > 0 {
		totalPages++
	}

	return &models.PaginatedResponse{
		Items:      responses,
		TotalCount: total,
		Page:       page.Page,
		PageSize:   page.PageSize,
		TotalPages: totalPages,
	}, nil
}

// Remove removes a member from an organization.
func (s *MemberService) Remove(ctx context.Context, orgID, userID uuid.UUID, ip string) error {
	if err := s.members.Remove(ctx, orgID, userID); err != nil {
		return err
	}

	securitylog.Log(ctx, securitylog.Event{
		Type:     securitylog.EventAccountDisabled,
		Outcome:  securitylog.OutcomeSuccess,
		TenantID: orgID.String(),
		IP:       ip,
		Resource: fmt.Sprintf("organizations/%s/members/%s", orgID, userID),
	})

	// Publish event
	if err := s.events.Publish(ctx, orgEventsStream, "org.member_removed", map[string]interface{}{
		"org_id":  orgID.String(),
		"user_id": userID.String(),
	}); err != nil {
		logger.Warn().Err(err).Msg("Failed to publish org.member_removed event")
	}

	return nil
}
