package service

import (
	"context"
	"fmt"

	"golang.org/x/crypto/bcrypt"

	"github.com/google/uuid"
	"github.com/hightrex/microservices-platform/libs/go/pkg/errors"
	"github.com/hightrex/microservices-platform/libs/go/pkg/logger"
	"github.com/hightrex/microservices-platform/libs/go/pkg/securitylog"
	"github.com/hightrex/microservices-platform/libs/go/pkg/validation"
	"github.com/hightrex/microservices-platform/services/auth-service/internal/config"
	"github.com/hightrex/microservices-platform/services/auth-service/internal/models"
)

// UserService handles user management business logic.
type UserService struct {
	users       UserRepository
	sessions    SessionRepository
	roles       RoleRepository
	pwHistory   PasswordHistoryRepository
	events      EventPublisher
	securityCfg config.SecurityConfig
}

// NewUserService creates a new UserService.
func NewUserService(
	users UserRepository,
	sessions SessionRepository,
	roles RoleRepository,
	pwHistory PasswordHistoryRepository,
	events EventPublisher,
	securityCfg config.SecurityConfig,
) *UserService {
	return &UserService{
		users:       users,
		sessions:    sessions,
		roles:       roles,
		pwHistory:   pwHistory,
		events:      events,
		securityCfg: securityCfg,
	}
}

// GetByID retrieves a user by ID with role information.
func (s *UserService) GetByID(ctx context.Context, id uuid.UUID) (*models.UserResponse, error) {
	user, err := s.users.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}

	userRoles, err := s.roles.GetUserRoles(ctx, user.ID)
	if err != nil {
		logger.Warn().Err(err).Msg("Failed to get user roles")
	}

	resp := user.ToResponse()
	for _, ur := range userRoles {
		resp.Roles = append(resp.Roles, string(ur.RoleName))
	}

	return &resp, nil
}

// List retrieves users with filtering and pagination.
func (s *UserService) List(ctx context.Context, filter models.UserFilter, page models.Pagination) (*models.PaginatedResponse, error) {
	users, total, err := s.users.List(ctx, filter, page)
	if err != nil {
		return nil, err
	}

	responses := make([]models.UserResponse, 0, len(users))
	for i := range users {
		responses = append(responses, users[i].ToResponse())
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

// Update performs a partial update on a user.
func (s *UserService) Update(ctx context.Context, id uuid.UUID, req models.UpdateUserRequest, ip string) (*models.UserResponse, error) {
	if errs := validation.Validate(req); errs != nil {
		return nil, errors.BadRequest("Validation failed", fmt.Errorf("%v", validation.ErrorResponse(errs)))
	}

	fields := make(map[string]interface{})
	if req.FirstName != nil {
		fields["first_name"] = *req.FirstName
	}
	if req.LastName != nil {
		fields["last_name"] = *req.LastName
	}
	if req.Status != nil {
		fields["status"] = *req.Status
	}

	if len(fields) == 0 {
		return nil, errors.BadRequest("No fields to update", nil)
	}

	if err := s.users.Update(ctx, id, fields); err != nil {
		return nil, err
	}

	// Publish event
	if err := s.events.Publish(ctx, authEventsStream, "user.updated", map[string]interface{}{
		"user_id": id.String(),
		"fields":  fields,
	}); err != nil {
		logger.Warn().Err(err).Msg("Failed to publish user.updated event")
	}

	return s.GetByID(ctx, id)
}

// Delete soft-deletes a user and revokes all their sessions.
func (s *UserService) Delete(ctx context.Context, id uuid.UUID, ip string) error {
	if err := s.users.Delete(ctx, id); err != nil {
		return err
	}

	// Revoke all sessions
	if err := s.sessions.DeleteAllByUser(ctx, id); err != nil {
		logger.Warn().Err(err).Msg("Failed to revoke all sessions for deleted user")
	}

	securitylog.Log(ctx, securitylog.Event{
		Type:     securitylog.EventAccountDisabled,
		Outcome:  securitylog.OutcomeSuccess,
		ActorID:  id.String(),
		IP:       ip,
		Resource: fmt.Sprintf("users/%s", id),
	})

	// Publish event
	if err := s.events.Publish(ctx, authEventsStream, "user.deleted", map[string]interface{}{
		"user_id": id.String(),
	}); err != nil {
		logger.Warn().Err(err).Msg("Failed to publish user.deleted event")
	}

	return nil
}

// ChangePassword verifies the old password, checks history, and sets a new one.
func (s *UserService) ChangePassword(ctx context.Context, id uuid.UUID, req models.ChangePasswordRequest, ip string) error {
	if errs := validation.Validate(req); errs != nil {
		return errors.BadRequest("Validation failed", fmt.Errorf("%v", validation.ErrorResponse(errs)))
	}

	if len(req.NewPassword) < s.securityCfg.PasswordMinLength {
		return errors.BadRequest(
			fmt.Sprintf("Password must be at least %d characters", s.securityCfg.PasswordMinLength), nil)
	}

	user, err := s.users.GetByID(ctx, id)
	if err != nil {
		return err
	}

	// Verify old password
	if err := bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(req.OldPassword)); err != nil {
		securitylog.LogFailure(ctx, securitylog.EventPasswordChanged, ip, fmt.Sprintf("users/%s/password", id), nil)
		return errors.Unauthorized("Current password is incorrect", nil)
	}

	// Check password history
	recentHashes, err := s.pwHistory.GetRecent(ctx, id, s.securityCfg.PasswordHistoryCount)
	if err != nil {
		logger.Warn().Err(err).Msg("Failed to get password history")
	}
	for _, oldHash := range recentHashes {
		if bcrypt.CompareHashAndPassword([]byte(oldHash), []byte(req.NewPassword)) == nil {
			return errors.BadRequest("Password was recently used. Please choose a different password.", nil)
		}
	}

	// Hash new password
	newHash, err := bcrypt.GenerateFromPassword([]byte(req.NewPassword), s.securityCfg.BcryptCost)
	if err != nil {
		return errors.InternalServerError("Failed to hash password", err)
	}

	// Update password
	if err := s.users.Update(ctx, id, map[string]interface{}{"password_hash": string(newHash)}); err != nil {
		return err
	}

	// Add to password history
	if err := s.pwHistory.Add(ctx, id, string(newHash)); err != nil {
		logger.Warn().Err(err).Msg("Failed to store password in history")
	}

	// Revoke all existing sessions (force re-login)
	if err := s.sessions.DeleteAllByUser(ctx, id); err != nil {
		logger.Warn().Err(err).Msg("Failed to revoke sessions after password change")
	}

	securitylog.LogSuccess(ctx, securitylog.EventPasswordChanged, id.String(), ip, fmt.Sprintf("users/%s/password", id))

	// Publish event
	if err := s.events.Publish(ctx, authEventsStream, "auth.password_changed", map[string]interface{}{
		"user_id": id.String(),
	}); err != nil {
		logger.Warn().Err(err).Msg("Failed to publish auth.password_changed event")
	}

	return nil
}

// AssignRole assigns a role to a user with authorization check.
func (s *UserService) AssignRole(ctx context.Context, userID uuid.UUID, roleName string, grantedBy *uuid.UUID, ip string) error {
	role, err := s.roles.GetByName(ctx, models.RoleName(roleName))
	if err != nil {
		return err
	}

	if err := s.roles.AssignRole(ctx, userID, role.ID, grantedBy); err != nil {
		return err
	}

	securitylog.Log(ctx, securitylog.Event{
		Type:     securitylog.EventRoleChanged,
		Outcome:  securitylog.OutcomeSuccess,
		ActorID:  userID.String(),
		IP:       ip,
		Resource: fmt.Sprintf("users/%s/role", userID),
		Details:  map[string]interface{}{"new_role": roleName},
	})

	// Publish event
	if err := s.events.Publish(ctx, authEventsStream, "user.role_changed", map[string]interface{}{
		"user_id":  userID.String(),
		"new_role": roleName,
	}); err != nil {
		logger.Warn().Err(err).Msg("Failed to publish user.role_changed event")
	}

	return nil
}

// GetUserSessions retrieves all active sessions for a user.
func (s *UserService) GetUserSessions(ctx context.Context, userID uuid.UUID) ([]models.SessionResponse, error) {
	sessions, err := s.sessions.ListByUser(ctx, userID)
	if err != nil {
		return nil, err
	}

	responses := make([]models.SessionResponse, 0, len(sessions))
	for _, sess := range sessions {
		responses = append(responses, sess.ToResponse())
	}
	return responses, nil
}
