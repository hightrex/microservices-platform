package redis

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/hightrex/microservices-platform/libs/go/pkg/cache"
	"github.com/hightrex/microservices-platform/services/notification-service/internal/models"
)

const (
	templateCacheTTL    = 1 * time.Hour
	preferenceCacheTTL  = 30 * time.Minute
	templateKeyPrefix   = "notif:template:"
	preferenceKeyPrefix = "notif:prefs:"
)

// NotificationCache implements service.TemplateCache using Redis.
type NotificationCache struct {
	client *cache.Client
}

// NewNotificationCache creates a new NotificationCache.
func NewNotificationCache(client *cache.Client) *NotificationCache {
	return &NotificationCache{client: client}
}

// GetTemplate retrieves a cached template by ID.
func (c *NotificationCache) GetTemplate(ctx context.Context, id uuid.UUID) (*models.NotificationTemplate, error) {
	var tmpl models.NotificationTemplate
	err := c.client.Get(ctx, templateKey(id), &tmpl)
	if err != nil {
		return nil, err
	}
	return &tmpl, nil
}

// SetTemplate caches a template with TTL.
func (c *NotificationCache) SetTemplate(ctx context.Context, template *models.NotificationTemplate) error {
	return c.client.Set(ctx, templateKey(template.ID), template, templateCacheTTL)
}

// InvalidateTemplate removes a template from cache.
func (c *NotificationCache) InvalidateTemplate(ctx context.Context, id uuid.UUID) error {
	return c.client.Delete(ctx, templateKey(id))
}

// GetUserPreferences retrieves cached user preferences.
func (c *NotificationCache) GetUserPreferences(ctx context.Context, userID uuid.UUID) ([]models.PreferenceResponse, error) {
	var prefs []models.PreferenceResponse
	err := c.client.Get(ctx, preferenceKey(userID), &prefs)
	if err != nil {
		return nil, err
	}
	return prefs, nil
}

// SetUserPreferences caches user preferences with TTL.
func (c *NotificationCache) SetUserPreferences(ctx context.Context, userID uuid.UUID, prefs []models.PreferenceResponse) error {
	return c.client.Set(ctx, preferenceKey(userID), prefs, preferenceCacheTTL)
}

// InvalidateUserPreferences removes user preferences from cache.
func (c *NotificationCache) InvalidateUserPreferences(ctx context.Context, userID uuid.UUID) error {
	return c.client.Delete(ctx, preferenceKey(userID))
}

func templateKey(id uuid.UUID) string {
	return fmt.Sprintf("%s%s", templateKeyPrefix, id.String())
}

func preferenceKey(userID uuid.UUID) string {
	return fmt.Sprintf("%s%s", preferenceKeyPrefix, userID.String())
}
