package service

import (
	"context"

	"github.com/google/uuid"
	"github.com/hightrex/microservices-platform/services/notification-service/internal/models"
)

// TemplateRepository defines the interface for template data access.
type TemplateRepository interface {
	Create(ctx context.Context, template *models.NotificationTemplate) error
	GetByID(ctx context.Context, id uuid.UUID) (*models.NotificationTemplate, error)
	GetByName(ctx context.Context, name string, channel models.Channel) (*models.NotificationTemplate, error)
	List(ctx context.Context, filter models.TemplateFilter, page models.Pagination) ([]models.NotificationTemplate, int, error)
	Update(ctx context.Context, id uuid.UUID, fields map[string]interface{}) error
	Delete(ctx context.Context, id uuid.UUID) error
}

// NotificationRepository defines the interface for notification data access.
type NotificationRepository interface {
	Create(ctx context.Context, notification *models.Notification) error
	GetByID(ctx context.Context, id uuid.UUID) (*models.Notification, error)
	List(ctx context.Context, userID uuid.UUID, filter models.NotificationFilter, page models.Pagination) ([]models.Notification, int, error)
	MarkAsRead(ctx context.Context, id uuid.UUID) error
	CountUnread(ctx context.Context, userID uuid.UUID) (int, error)
	UpdateStatus(ctx context.Context, id uuid.UUID, status models.NotificationStatus) error
}

// PreferenceRepository defines the interface for preference data access.
type PreferenceRepository interface {
	GetUserPreferences(ctx context.Context, userID uuid.UUID) ([]models.NotificationPreference, error)
	UpdatePreference(ctx context.Context, userID uuid.UUID, eventType string, channel models.Channel, enabled bool) error
	CheckEnabled(ctx context.Context, userID uuid.UUID, eventType string, channel models.Channel) (bool, error)
}

// DeliveryRepository defines the interface for delivery log data access.
type DeliveryRepository interface {
	Create(ctx context.Context, log *models.DeliveryLog) error
	List(ctx context.Context, notificationID uuid.UUID) ([]models.DeliveryLog, error)
	GetFailedDeliveries(ctx context.Context, maxRetries int) ([]models.DeliveryLog, error)
	IncrementRetryCount(ctx context.Context, id uuid.UUID) error
	UpdateStatus(ctx context.Context, id uuid.UUID, status models.DeliveryStatus) error
}

// DLQRepository defines the interface for dead-letter queue data access.
type DLQRepository interface {
	Insert(ctx context.Context, tenantID uuid.UUID, eventType string, payload []byte, errMsg string) error
}

// TemplateCache defines the interface for Redis-based template caching.
type TemplateCache interface {
	GetTemplate(ctx context.Context, id uuid.UUID) (*models.NotificationTemplate, error)
	SetTemplate(ctx context.Context, template *models.NotificationTemplate) error
	InvalidateTemplate(ctx context.Context, id uuid.UUID) error
	GetUserPreferences(ctx context.Context, userID uuid.UUID) ([]models.PreferenceResponse, error)
	SetUserPreferences(ctx context.Context, userID uuid.UUID, prefs []models.PreferenceResponse) error
	InvalidateUserPreferences(ctx context.Context, userID uuid.UUID) error
}

// EventPublisher defines the interface for publishing domain events.
type EventPublisher interface {
	Publish(ctx context.Context, stream string, eventType string, data interface{}) error
}

// ChannelSender defines the interface for a notification channel implementation.
type ChannelSender interface {
	Send(ctx context.Context, recipient string, subject string, body string, metadata map[string]interface{}) error
	Channel() models.Channel
}
