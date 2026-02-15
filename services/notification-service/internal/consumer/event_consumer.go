package consumer

import (
	"context"
	"encoding/json"

	"github.com/google/uuid"
	"github.com/hightrex/microservices-platform/libs/go/pkg/logger"
	"github.com/hightrex/microservices-platform/libs/go/pkg/messaging"
	"github.com/hightrex/microservices-platform/libs/go/pkg/tenant"
	"github.com/hightrex/microservices-platform/services/notification-service/internal/models"
	"github.com/hightrex/microservices-platform/services/notification-service/internal/service"
)

// EventConsumer handles incoming events from other services and routes them
// to notification templates for automatic notification delivery.
type EventConsumer struct {
	notifSvc    *service.NotificationService
	templateSvc *service.TemplateService
}

// NewEventConsumer creates a new EventConsumer.
func NewEventConsumer(notifSvc *service.NotificationService, templateSvc *service.TemplateService) *EventConsumer {
	return &EventConsumer{
		notifSvc:    notifSvc,
		templateSvc: templateSvc,
	}
}

// HandleUserCreated processes user.created events by sending a welcome notification.
func (ec *EventConsumer) HandleUserCreated(ctx context.Context, event messaging.Event) error {
	logger.Info().
		Str("event_id", event.ID).
		Str("event_type", event.Type).
		Str("tenant_id", event.TenantID).
		Msg("Processing user.created event for notification")

	tenantID, err := uuid.Parse(event.TenantID)
	if err != nil {
		logger.Error().Err(err).Str("tenant_id", event.TenantID).Msg("Invalid tenant ID in event")
		return nil // Don't retry — bad data
	}

	// Set tenant context for downstream repository calls
	ctx = tenant.NewContext(ctx, tenantID)

	var data struct {
		UserID string `json:"user_id"`
		Email  string `json:"email"`
	}
	if err := json.Unmarshal(event.Data, &data); err != nil {
		logger.Error().Err(err).Msg("Failed to unmarshal user.created event data")
		return nil // Don't retry — bad data
	}

	userID, err := uuid.Parse(data.UserID)
	if err != nil {
		logger.Error().Err(err).Str("user_id", data.UserID).Msg("Invalid user ID in event data")
		return nil
	}

	// Send in-app welcome notification
	body := "Welcome to the platform! Start by exploring the dashboard."
	subject := "Welcome aboard!"
	req := models.SendNotificationRequest{
		Channel:   models.ChannelInApp,
		Recipient: data.UserID,
		EventType: "user.created",
		UserID:    userID,
		Subject:   &subject,
		Body:      &body,
	}

	if _, err := ec.notifSvc.Send(ctx, req); err != nil {
		logger.Error().Err(err).Str("user_id", data.UserID).Msg("Failed to send welcome notification")
		return err
	}

	logger.Info().Str("user_id", data.UserID).Msg("Welcome notification sent")
	return nil
}

// HandleOrgCreated processes org.created events.
func (ec *EventConsumer) HandleOrgCreated(ctx context.Context, event messaging.Event) error {
	logger.Info().
		Str("event_id", event.ID).
		Str("event_type", event.Type).
		Msg("Processing org.created event for notification")

	tenantID, err := uuid.Parse(event.TenantID)
	if err != nil {
		logger.Error().Err(err).Msg("Invalid tenant ID in org.created event")
		return nil
	}

	ctx = tenant.NewContext(ctx, tenantID)

	var data struct {
		OrgID   string `json:"org_id"`
		OrgName string `json:"org_name"`
		OwnerID string `json:"owner_user_id"`
	}
	if err := json.Unmarshal(event.Data, &data); err != nil {
		logger.Error().Err(err).Msg("Failed to unmarshal org.created event data")
		return nil
	}

	ownerID, err := uuid.Parse(data.OwnerID)
	if err != nil {
		logger.Error().Err(err).Msg("Invalid owner ID in org.created event data")
		return nil
	}

	body := "Your organization '" + data.OrgName + "' has been created successfully."
	subject := "Organization Created"
	req := models.SendNotificationRequest{
		Channel:   models.ChannelInApp,
		Recipient: data.OwnerID,
		EventType: "org.created",
		UserID:    ownerID,
		Subject:   &subject,
		Body:      &body,
	}

	if _, err := ec.notifSvc.Send(ctx, req); err != nil {
		logger.Error().Err(err).Str("org_id", data.OrgID).Msg("Failed to send org created notification")
		return err
	}

	return nil
}
