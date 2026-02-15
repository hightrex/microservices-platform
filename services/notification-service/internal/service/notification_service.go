package service

import (
	"context"
	"encoding/json"
	"fmt"
	"net"
	"net/url"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/hightrex/microservices-platform/libs/go/pkg/errors"
	"github.com/hightrex/microservices-platform/libs/go/pkg/logger"
	"github.com/hightrex/microservices-platform/services/notification-service/internal/models"
)

// NotificationService handles notification sending business logic.
type NotificationService struct {
	notifRepo    NotificationRepository
	templateRepo TemplateRepository
	prefRepo     PreferenceRepository
	deliveryRepo DeliveryRepository
	cache        TemplateCache
	publisher    EventPublisher
	channels     map[models.Channel]ChannelSender
	templateSvc  *TemplateService
}

// NewNotificationService creates a new NotificationService.
func NewNotificationService(
	notifRepo NotificationRepository,
	templateRepo TemplateRepository,
	prefRepo PreferenceRepository,
	deliveryRepo DeliveryRepository,
	cache TemplateCache,
	publisher EventPublisher,
	templateSvc *TemplateService,
) *NotificationService {
	return &NotificationService{
		notifRepo:    notifRepo,
		templateRepo: templateRepo,
		prefRepo:     prefRepo,
		deliveryRepo: deliveryRepo,
		cache:        cache,
		publisher:    publisher,
		channels:     make(map[models.Channel]ChannelSender),
		templateSvc:  templateSvc,
	}
}

// RegisterChannel registers a channel sender implementation.
func (s *NotificationService) RegisterChannel(sender ChannelSender) {
	s.channels[sender.Channel()] = sender
}

// Send orchestrates notification delivery:
// 1. Load template (if template_id provided)
// 2. Check user preferences
// 3. Render template with data
// 4. Route to appropriate channel handler
// 5. Create delivery log
// 6. Return notification ID
func (s *NotificationService) Send(ctx context.Context, req models.SendNotificationRequest) (*uuid.UUID, error) {
	// Validate webhook URLs for SSRF prevention
	if req.Channel == models.ChannelWebhook {
		if err := validateWebhookURL(req.Recipient); err != nil {
			return nil, err
		}
	}

	// Check user preferences
	enabled, err := s.prefRepo.CheckEnabled(ctx, req.UserID, req.EventType, req.Channel)
	if err != nil {
		logger.Warn().Err(err).Msg("Failed to check user preferences, proceeding with send")
	} else if !enabled {
		logger.Info().
			Str("user_id", req.UserID.String()).
			Str("event_type", req.EventType).
			Str("channel", string(req.Channel)).
			Msg("Notification skipped — user has opted out")
		return nil, nil
	}

	var subject, body string

	// Render from template if template_id provided
	if req.TemplateID != nil {
		tmpl, err := s.templateRepo.GetByID(ctx, *req.TemplateID)
		if err != nil {
			return nil, errors.BadRequest("Template not found", err)
		}

		var data map[string]interface{}
		if req.Data != nil {
			if err := json.Unmarshal(req.Data, &data); err != nil {
				return nil, errors.BadRequest("Invalid template data", err)
			}
		}

		subject, body, err = s.templateSvc.RenderTemplate(tmpl, data)
		if err != nil {
			return nil, err
		}
	} else {
		// Use inline subject/body
		if req.Body == nil || *req.Body == "" {
			return nil, errors.BadRequest("Either template_id or body is required", nil)
		}
		body = *req.Body
		if req.Subject != nil {
			subject = *req.Subject
		}
	}

	// Create notification record
	notif := &models.Notification{
		UserID:    req.UserID,
		Channel:   req.Channel,
		EventType: req.EventType,
		Subject:   stringPtrIfNotEmpty(subject),
		Body:      body,
		Status:    models.StatusPending,
		Metadata:  req.Data,
	}

	if err := s.notifRepo.Create(ctx, notif); err != nil {
		return nil, err
	}

	// Route to channel handler
	sender, ok := s.channels[req.Channel]
	if !ok {
		logger.Error().Str("channel", string(req.Channel)).Msg("No sender registered for channel")
		if statusErr := s.notifRepo.UpdateStatus(ctx, notif.ID, models.StatusFailed); statusErr != nil {
			logger.Error().Err(statusErr).Msg("Failed to update notification status")
		}
		return &notif.ID, errors.BadRequest(fmt.Sprintf("Channel %s is not configured", req.Channel), nil)
	}

	// Attempt delivery
	metadata := make(map[string]interface{})
	if req.Data != nil {
		if err := json.Unmarshal(req.Data, &metadata); err != nil {
			logger.Warn().Err(err).Msg("Failed to parse metadata for channel sender")
		}
	}

	deliveryStatus := models.DeliveryStatusDelivered
	var deliveryErr *string
	var deliveredAt *time.Time

	if err := sender.Send(ctx, req.Recipient, subject, body, metadata); err != nil {
		deliveryStatus = models.DeliveryStatusFailed
		errMsg := err.Error()
		deliveryErr = &errMsg

		if statusErr := s.notifRepo.UpdateStatus(ctx, notif.ID, models.StatusFailed); statusErr != nil {
			logger.Error().Err(statusErr).Msg("Failed to update notification status after delivery failure")
		}

		// Publish failure event
		if pubErr := s.publisher.Publish(ctx, "notification-events", "notification.failed", map[string]interface{}{
			"notification_id": notif.ID,
			"channel":         req.Channel,
			"error":           err.Error(),
		}); pubErr != nil {
			logger.Warn().Err(pubErr).Msg("Failed to publish notification.failed event")
		}
	} else {
		now := time.Now()
		deliveredAt = &now

		if statusErr := s.notifRepo.UpdateStatus(ctx, notif.ID, models.StatusSent); statusErr != nil {
			logger.Error().Err(statusErr).Msg("Failed to update notification status after delivery success")
		}

		// Publish success event
		if pubErr := s.publisher.Publish(ctx, "notification-events", "notification.sent", map[string]interface{}{
			"notification_id": notif.ID,
			"channel":         req.Channel,
			"user_id":         req.UserID,
		}); pubErr != nil {
			logger.Warn().Err(pubErr).Msg("Failed to publish notification.sent event")
		}
	}

	// Record delivery log
	deliveryLog := &models.DeliveryLog{
		NotificationID: notif.ID,
		Channel:        req.Channel,
		Recipient:      req.Recipient,
		Status:         deliveryStatus,
		ErrorMessage:   deliveryErr,
		RetryCount:     0,
		DeliveredAt:    deliveredAt,
	}
	if logErr := s.deliveryRepo.Create(ctx, deliveryLog); logErr != nil {
		logger.Error().Err(logErr).Msg("Failed to create delivery log")
	}

	return &notif.ID, nil
}

// GetByID retrieves a notification by ID.
func (s *NotificationService) GetByID(ctx context.Context, id uuid.UUID) (*models.NotificationResponse, error) {
	notif, err := s.notifRepo.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}
	resp := notif.ToResponse()
	return &resp, nil
}

// List retrieves in-app notifications for a user.
func (s *NotificationService) List(ctx context.Context, userID uuid.UUID, filter models.NotificationFilter, page models.Pagination) ([]models.NotificationResponse, int, error) {
	notifications, total, err := s.notifRepo.List(ctx, userID, filter, page)
	if err != nil {
		return nil, 0, err
	}

	responses := make([]models.NotificationResponse, len(notifications))
	for i, n := range notifications {
		responses[i] = n.ToResponse()
	}
	return responses, total, nil
}

// MarkAsRead marks a notification as read.
func (s *NotificationService) MarkAsRead(ctx context.Context, id uuid.UUID) error {
	return s.notifRepo.MarkAsRead(ctx, id)
}

// GetUnreadCount returns the number of unread in-app notifications for a user.
func (s *NotificationService) GetUnreadCount(ctx context.Context, userID uuid.UUID) (int, error) {
	return s.notifRepo.CountUnread(ctx, userID)
}

// validateWebhookURL checks that a webhook URL is not targeting private/internal addresses (SSRF prevention).
func validateWebhookURL(rawURL string) error {
	parsed, err := url.Parse(rawURL)
	if err != nil {
		return errors.BadRequest("Invalid webhook URL", err)
	}

	if parsed.Scheme != "https" && parsed.Scheme != "http" {
		return errors.BadRequest("Webhook URL must use http or https scheme", nil)
	}

	hostname := parsed.Hostname()

	// Block localhost
	if strings.EqualFold(hostname, "localhost") || hostname == "127.0.0.1" || hostname == "::1" || hostname == "0.0.0.0" {
		return errors.BadRequest("Webhook URL must not target localhost", nil)
	}

	// Resolve hostname and check for private IPs
	ips, err := net.LookupIP(hostname)
	if err != nil {
		return errors.BadRequest("Failed to resolve webhook URL hostname", err)
	}

	for _, ip := range ips {
		if ip.IsLoopback() || ip.IsPrivate() || ip.IsLinkLocalUnicast() || ip.IsLinkLocalMulticast() {
			return errors.BadRequest("Webhook URL must not target private or internal addresses", nil)
		}
		// Block cloud metadata endpoints (169.254.169.254)
		if ip.Equal(net.ParseIP("169.254.169.254")) {
			return errors.BadRequest("Webhook URL must not target cloud metadata endpoints", nil)
		}
	}

	return nil
}

func stringPtrIfNotEmpty(s string) *string {
	if s == "" {
		return nil
	}
	return &s
}
