package channels

import (
	"context"

	"github.com/hightrex/microservices-platform/libs/go/pkg/logger"
	"github.com/hightrex/microservices-platform/services/notification-service/internal/models"
)

// InAppChannel stores notifications in the database for in-app display.
// The actual database write is handled by the NotificationService, so this
// channel sender is a no-op that simply records that in-app delivery succeeded.
type InAppChannel struct{}

// NewInAppChannel creates a new InAppChannel.
func NewInAppChannel() *InAppChannel {
	return &InAppChannel{}
}

// Channel returns the channel type.
func (c *InAppChannel) Channel() models.Channel {
	return models.ChannelInApp
}

// Send is a no-op for in-app notifications since the notification record
// in the database IS the delivery. The NotificationService.Send method
// already creates the notification record before calling this.
func (c *InAppChannel) Send(ctx context.Context, recipient, subject, body string, metadata map[string]interface{}) error {
	logger.Debug().
		Str("recipient", recipient).
		Msg("In-app notification delivered (stored in database)")
	return nil
}
