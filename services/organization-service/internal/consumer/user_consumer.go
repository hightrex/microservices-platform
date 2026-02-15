package consumer

import (
	"context"
	"encoding/json"
	"time"

	"github.com/google/uuid"
	"github.com/hightrex/microservices-platform/libs/go/pkg/logger"
	"github.com/hightrex/microservices-platform/libs/go/pkg/messaging"
	"github.com/hightrex/microservices-platform/libs/go/pkg/metrics"
	"github.com/hightrex/microservices-platform/libs/go/pkg/tenant"
)

// OrgCreator defines the interface for creating organizations from user events.
// This is a narrow interface so the consumer only depends on what it needs.
type OrgCreator interface {
	CreateFromUserEvent(ctx context.Context, userID, email string) error
}

// UserConsumer handles user-related events from the auth service.
type UserConsumer struct {
	orgService OrgCreator
}

// NewUserConsumer creates a new user consumer
func NewUserConsumer(orgService OrgCreator) *UserConsumer {
	return &UserConsumer{
		orgService: orgService,
	}
}

// userCreatedData matches the event payload published by auth service
type userCreatedData struct {
	UserID string `json:"user_id"`
	Email  string `json:"email"`
	Source string `json:"source,omitempty"` // "registration" or "admin"
}

// HandleUserCreated processes user.created events from the auth-events stream.
// It auto-creates an organization for newly registered users.
func (c *UserConsumer) HandleUserCreated(ctx context.Context, event messaging.Event) error {
	start := time.Now()
	status := "success"
	defer func() {
		duration := time.Since(start).Seconds()
		metrics.EventProcessingDuration.WithLabelValues("auth-events", "user_created_handler", status).Observe(duration)
		metrics.EventsProcessed.WithLabelValues("auth-events", "user_created_handler", status).Inc()
	}()

	logger.Info().
		Str("event_id", event.ID).
		Str("tenant_id", event.TenantID).
		Msg("Processing user.created event")

	// Parse event data
	var data userCreatedData
	if err := json.Unmarshal(event.Data, &data); err != nil {
		status = "error"
		logger.Error().Err(err).
			Str("event_id", event.ID).
			Msg("Failed to unmarshal user.created event data")
		return err
	}

	if data.UserID == "" {
		status = "ignored"
		logger.Warn().
			Str("event_id", event.ID).
			Msg("Received user.created event without user_id, skipping")
		return nil // Not an error — just skip malformed events
	}

	// Inject tenant context from event if available.
	// For registration events, tenant may be nil (org doesn't exist yet),
	// which is fine — CreateFromUserEvent handles the unscoped creation path.
	if event.TenantID != "" && event.TenantID != uuid.Nil.String() {
		tenantID, err := uuid.Parse(event.TenantID)
		if err == nil {
			ctx = tenant.NewContext(ctx, tenantID)
		}
	}

	// Create organization for the user with a bounded timeout
	ctx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()

	if err := c.orgService.CreateFromUserEvent(ctx, data.UserID, data.Email); err != nil {
		status = "error"
		logger.Error().Err(err).
			Str("user_id", data.UserID).
			Str("event_id", event.ID).
			Msg("Failed to auto-create organization for user")
		return err
	}

	logger.Info().
		Str("user_id", data.UserID).
		Str("event_id", event.ID).
		Msg("Successfully processed user.created event — organization auto-created")
	return nil
}
