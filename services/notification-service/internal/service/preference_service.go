package service

import (
	"context"

	"github.com/google/uuid"
	"github.com/hightrex/microservices-platform/libs/go/pkg/logger"
	"github.com/hightrex/microservices-platform/services/notification-service/internal/models"
)

// PreferenceService handles notification preference business logic.
type PreferenceService struct {
	prefRepo  PreferenceRepository
	cache     TemplateCache
	publisher EventPublisher
}

// NewPreferenceService creates a new PreferenceService.
func NewPreferenceService(repo PreferenceRepository, cache TemplateCache, publisher EventPublisher) *PreferenceService {
	return &PreferenceService{
		prefRepo:  repo,
		cache:     cache,
		publisher: publisher,
	}
}

// GetPreferences retrieves all notification preferences for a user, with caching.
func (s *PreferenceService) GetPreferences(ctx context.Context, userID uuid.UUID) ([]models.PreferenceResponse, error) {
	// Try cache first
	cached, err := s.cache.GetUserPreferences(ctx, userID)
	if err == nil && cached != nil {
		return cached, nil
	}

	prefs, err := s.prefRepo.GetUserPreferences(ctx, userID)
	if err != nil {
		return nil, err
	}

	responses := make([]models.PreferenceResponse, len(prefs))
	for i, p := range prefs {
		responses[i] = p.ToResponse()
	}

	// Populate cache
	if cacheErr := s.cache.SetUserPreferences(ctx, userID, responses); cacheErr != nil {
		logger.Warn().Err(cacheErr).Str("user_id", userID.String()).Msg("Failed to cache user preferences")
	}

	return responses, nil
}

// UpdatePreference updates a single notification preference and invalidates cache.
func (s *PreferenceService) UpdatePreference(ctx context.Context, userID uuid.UUID, req models.UpdatePreferenceRequest) error {
	if err := s.prefRepo.UpdatePreference(ctx, userID, req.EventType, req.Channel, req.Enabled); err != nil {
		return err
	}

	// Invalidate cache
	if err := s.cache.InvalidateUserPreferences(ctx, userID); err != nil {
		logger.Warn().Err(err).Str("user_id", userID.String()).Msg("Failed to invalidate preference cache")
	}

	// Publish event
	if err := s.publisher.Publish(ctx, "notification-events", "notification.preference_updated", map[string]interface{}{
		"user_id":    userID,
		"channel":    req.Channel,
		"event_type": req.EventType,
		"enabled":    req.Enabled,
	}); err != nil {
		logger.Warn().Err(err).Msg("Failed to publish preference_updated event")
	}

	return nil
}

// BulkUpdatePreferences updates multiple preferences and invalidates cache.
func (s *PreferenceService) BulkUpdatePreferences(ctx context.Context, userID uuid.UUID, req models.BulkUpdatePreferencesRequest) error {
	for _, pref := range req.Preferences {
		if err := s.prefRepo.UpdatePreference(ctx, userID, pref.EventType, pref.Channel, pref.Enabled); err != nil {
			return err
		}
	}

	// Invalidate cache
	if err := s.cache.InvalidateUserPreferences(ctx, userID); err != nil {
		logger.Warn().Err(err).Str("user_id", userID.String()).Msg("Failed to invalidate preference cache")
	}

	// Publish event
	if err := s.publisher.Publish(ctx, "notification-events", "notification.preference_updated", map[string]interface{}{
		"user_id": userID,
		"count":   len(req.Preferences),
	}); err != nil {
		logger.Warn().Err(err).Msg("Failed to publish preference_updated event")
	}

	return nil
}

// GetDefaultPreferences returns system default preferences for new users.
func (s *PreferenceService) GetDefaultPreferences() []models.PreferenceResponse {
	defaults := []models.PreferenceResponse{}
	channels := models.ValidChannels()
	eventTypes := []string{
		"user.created", "user.updated", "user.password_changed",
		"org.created", "org.updated",
		"billing.invoice_paid", "billing.payment_failed",
		"file.uploaded", "file.quarantined",
	}

	for _, eventType := range eventTypes {
		for _, channel := range channels {
			defaults = append(defaults, models.PreferenceResponse{
				Channel:   channel,
				EventType: eventType,
				Enabled:   true,
			})
		}
	}

	return defaults
}
