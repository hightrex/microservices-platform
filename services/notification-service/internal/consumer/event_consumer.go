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

// HandleBillingEvent processes billing-related events (subscription changes, invoices, usage).
func (ec *EventConsumer) HandleBillingEvent(ctx context.Context, event messaging.Event) error {
	logger.Info().
		Str("event_id", event.ID).
		Str("event_type", event.Type).
		Str("tenant_id", event.TenantID).
		Msg("Processing billing event for notification")

	tenantID, err := uuid.Parse(event.TenantID)
	if err != nil {
		logger.Error().Err(err).Str("tenant_id", event.TenantID).Msg("Invalid tenant ID in billing event")
		return nil
	}

	ctx = tenant.NewContext(ctx, tenantID)

	// Common data structure — all billing events should include user_id for recipient targeting
	var baseData struct {
		UserID string `json:"user_id"`
	}
	if err := json.Unmarshal(event.Data, &baseData); err != nil {
		logger.Error().Err(err).Msg("Failed to unmarshal base billing event data")
		return nil
	}

	// Extract user_id from event data; tenant_id is NOT a valid recipient
	recipientID := baseData.UserID
	if recipientID == "" {
		logger.Warn().
			Str("event_type", event.Type).
			Str("tenant_id", event.TenantID).
			Msg("Billing event missing user_id, skipping notification — tenant_id is not a valid recipient")
		return nil
	}

	var subject, body string

	switch event.Type {
	case "subscription.created":
		var data struct {
			SubscriptionID string `json:"subscription_id"`
			PlanName       string `json:"plan_name"`
		}
		if err := json.Unmarshal(event.Data, &data); err != nil {
			logger.Error().Err(err).Msg("Failed to unmarshal subscription.created data")
			return nil
		}
		subject = "Subscription Activated"
		body = "Your subscription to the '" + data.PlanName + "' plan is now active."

	case "subscription.updated":
		var data struct {
			SubscriptionID string `json:"subscription_id"`
			PlanName       string `json:"plan_name"`
		}
		if err := json.Unmarshal(event.Data, &data); err != nil {
			logger.Error().Err(err).Msg("Failed to unmarshal subscription.updated data")
			return nil
		}
		subject = "Subscription Updated"
		if data.PlanName != "" {
			body = "Your subscription has been changed to the '" + data.PlanName + "' plan."
		} else {
			body = "Your subscription has been updated."
		}

	case "subscription.canceled":
		var data struct {
			SubscriptionID string `json:"subscription_id"`
			Immediate      bool   `json:"immediate"`
		}
		if err := json.Unmarshal(event.Data, &data); err != nil {
			logger.Error().Err(err).Msg("Failed to unmarshal subscription.canceled data")
			return nil
		}
		subject = "Subscription Canceled"
		if data.Immediate {
			body = "Your subscription has been canceled immediately."
		} else {
			body = "Your subscription has been canceled and will end at the current billing period."
		}

	case "invoice.paid":
		var data struct {
			InvoiceID string `json:"invoice_id"`
			Amount    int64  `json:"amount"`
		}
		if err := json.Unmarshal(event.Data, &data); err != nil {
			logger.Error().Err(err).Msg("Failed to unmarshal invoice.paid data")
			return nil
		}
		subject = "Payment Received"
		body = "Payment for invoice " + data.InvoiceID + " has been received."

	case "invoice.failed":
		var data struct {
			InvoiceID string `json:"invoice_id"`
			Reason    string `json:"reason"`
		}
		if err := json.Unmarshal(event.Data, &data); err != nil {
			logger.Error().Err(err).Msg("Failed to unmarshal invoice.failed data")
			return nil
		}
		subject = "Payment Failed"
		body = "Payment for invoice " + data.InvoiceID + " has failed. Please update your payment method."

	case "usage.quota_exceeded":
		var data struct {
			MetricName string `json:"metric_name"`
		}
		if err := json.Unmarshal(event.Data, &data); err != nil {
			logger.Error().Err(err).Msg("Failed to unmarshal usage.quota_exceeded data")
			return nil
		}
		subject = "Usage Quota Exceeded"
		body = "Your usage quota for '" + data.MetricName + "' has been exceeded. Consider upgrading your plan."

	default:
		logger.Debug().Str("event_type", event.Type).Msg("Unhandled billing event type, skipping notification")
		return nil
	}

	userID, err := uuid.Parse(recipientID)
	if err != nil {
		logger.Error().Err(err).Str("recipient_id", recipientID).Msg("Invalid recipient user ID for billing notification")
		return nil
	}

	req := models.SendNotificationRequest{
		Channel:   models.ChannelInApp,
		Recipient: recipientID,
		EventType: event.Type,
		UserID:    userID,
		Subject:   &subject,
		Body:      &body,
	}

	if _, err := ec.notifSvc.Send(ctx, req); err != nil {
		logger.Error().Err(err).Str("event_type", event.Type).Msg("Failed to send billing notification")
		return err
	}

	logger.Info().Str("event_type", event.Type).Str("user_id", recipientID).Msg("Billing notification sent")
	return nil
}

// HandleFileEvent processes file-related events (upload, delete, quota).
func (ec *EventConsumer) HandleFileEvent(ctx context.Context, event messaging.Event) error {
	logger.Info().
		Str("event_id", event.ID).
		Str("event_type", event.Type).
		Str("tenant_id", event.TenantID).
		Msg("Processing file event for notification")

	tenantID, err := uuid.Parse(event.TenantID)
	if err != nil {
		logger.Error().Err(err).Str("tenant_id", event.TenantID).Msg("Invalid tenant ID in file event")
		return nil
	}

	ctx = tenant.NewContext(ctx, tenantID)

	// Common data structure — file events should include user_id for recipient targeting
	var baseData struct {
		UserID string `json:"user_id"`
	}
	if err := json.Unmarshal(event.Data, &baseData); err != nil {
		logger.Error().Err(err).Msg("Failed to unmarshal base file event data")
		return nil
	}

	// Extract user_id from event data; tenant_id is NOT a valid recipient
	recipientID := baseData.UserID
	if recipientID == "" {
		logger.Warn().
			Str("event_type", event.Type).
			Str("tenant_id", event.TenantID).
			Msg("File event missing user_id, skipping notification — tenant_id is not a valid recipient")
		return nil
	}

	var subject, body string

	switch event.Type {
	case "file.uploaded":
		var data struct {
			FileID   string `json:"file_id"`
			Filename string `json:"filename"`
		}
		if err := json.Unmarshal(event.Data, &data); err != nil {
			logger.Error().Err(err).Msg("Failed to unmarshal file.uploaded data")
			return nil
		}
		subject = "File Uploaded"
		body = "File '" + data.Filename + "' has been uploaded successfully."

	case "file.deleted":
		var data struct {
			FileID string `json:"file_id"`
		}
		if err := json.Unmarshal(event.Data, &data); err != nil {
			logger.Error().Err(err).Msg("Failed to unmarshal file.deleted data")
			return nil
		}
		subject = "File Deleted"
		body = "A file has been deleted from your storage."

	case "file.quarantined":
		var data struct {
			FileID   string `json:"file_id"`
			Filename string `json:"filename"`
		}
		if err := json.Unmarshal(event.Data, &data); err != nil {
			logger.Error().Err(err).Msg("Failed to unmarshal file.quarantined data")
			return nil
		}
		subject = "File Quarantined"
		body = "File '" + data.Filename + "' has been quarantined due to a security concern."

	case "quota.exceeded":
		var data struct {
			AttemptedBytes int64 `json:"attempted_bytes"`
		}
		if err := json.Unmarshal(event.Data, &data); err != nil {
			logger.Error().Err(err).Msg("Failed to unmarshal quota.exceeded data")
			return nil
		}
		subject = "Storage Quota Exceeded"
		body = "Your storage quota has been exceeded. Please free up space or upgrade your plan."

	default:
		logger.Debug().Str("event_type", event.Type).Msg("Unhandled file event type, skipping notification")
		return nil
	}

	userID, err := uuid.Parse(recipientID)
	if err != nil {
		logger.Error().Err(err).Str("recipient_id", recipientID).Msg("Invalid recipient user ID for file notification")
		return nil
	}

	req := models.SendNotificationRequest{
		Channel:   models.ChannelInApp,
		Recipient: recipientID,
		EventType: event.Type,
		UserID:    userID,
		Subject:   &subject,
		Body:      &body,
	}

	if _, err := ec.notifSvc.Send(ctx, req); err != nil {
		logger.Error().Err(err).Str("event_type", event.Type).Msg("Failed to send file notification")
		return err
	}

	logger.Info().Str("event_type", event.Type).Str("user_id", recipientID).Msg("File notification sent")
	return nil
}
