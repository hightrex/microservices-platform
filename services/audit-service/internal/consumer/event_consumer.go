package consumer

import (
	"context"
	"encoding/json"

	"github.com/google/uuid"
	"github.com/hightrex/microservices-platform/libs/go/pkg/logger"
	"github.com/hightrex/microservices-platform/libs/go/pkg/messaging"
	"github.com/hightrex/microservices-platform/libs/go/pkg/tenant"
	"github.com/hightrex/microservices-platform/services/audit-service/internal/models"
	"github.com/hightrex/microservices-platform/services/audit-service/internal/service"
)

// eventCategoryMap maps event type prefixes to categories.
var eventCategoryMap = map[string]models.EventCategory{
	"user.":         models.CategoryAuth,
	"auth.":         models.CategoryAuth,
	"org.":          models.CategoryData,
	"billing.":      models.CategoryData,
	"subscription.": models.CategoryData,
	"invoice.":      models.CategoryData,
	"file.":         models.CategoryData,
	"notification.": models.CategorySystem,
	"audit.":        models.CategorySystem,
	"security.":     models.CategorySecurity,
}

// EventConsumer captures ALL events from other services and logs them to the audit trail.
type EventConsumer struct {
	auditSvc *service.AuditService
}

// NewEventConsumer creates a new EventConsumer.
func NewEventConsumer(auditSvc *service.AuditService) *EventConsumer {
	return &EventConsumer{auditSvc: auditSvc}
}

// HandleAllEvents is a catch-all handler that maps any service event to an audit log entry.
func (ec *EventConsumer) HandleAllEvents(ctx context.Context, event messaging.Event) error {
	logger.Debug().
		Str("event_id", event.ID).
		Str("event_type", event.Type).
		Str("source", event.Source).
		Str("tenant_id", event.TenantID).
		Msg("Capturing event for audit log")

	tenantID, err := uuid.Parse(event.TenantID)
	if err != nil {
		logger.Error().Err(err).Str("tenant_id", event.TenantID).Msg("Invalid tenant ID in event, skipping")
		return nil // Don't retry bad data
	}

	// Set tenant context for repository calls
	ctx = tenant.NewContext(ctx, tenantID)

	// Extract actor and resource info from event data
	actorID, actorType, resourceType, resourceID, action, outcome := parseEventData(event)

	// Determine event category from event type prefix
	category := categorizeEvent(event.Type)

	auditLog := &models.AuditLog{
		TenantID:      tenantID,
		EventType:     event.Type,
		EventCategory: category,
		ActorID:       actorID,
		ActorType:     actorType,
		ResourceType:  resourceType,
		ResourceID:    resourceID,
		Action:        action,
		Outcome:       outcome,
		Metadata:      event.Data,
		Timestamp:     event.Timestamp,
	}

	if err := ec.auditSvc.CreateLog(ctx, auditLog); err != nil {
		logger.Error().Err(err).
			Str("event_type", event.Type).
			Str("event_id", event.ID).
			Msg("Failed to create audit log from event")
		return err
	}

	return nil
}

// categorizeEvent determines the event category from the event type.
func categorizeEvent(eventType string) models.EventCategory {
	for prefix, category := range eventCategoryMap {
		if len(eventType) >= len(prefix) && eventType[:len(prefix)] == prefix {
			return category
		}
	}
	return models.CategorySystem
}

// parseEventData extracts structured audit fields from the event data.
func parseEventData(event messaging.Event) (actorID string, actorType models.ActorType, resourceType *string, resourceID *string, action string, outcome models.Outcome) {
	actorID = "system"
	actorType = models.ActorSystem
	action = event.Type
	outcome = models.OutcomeSuccess

	var data map[string]interface{}
	if err := json.Unmarshal(event.Data, &data); err != nil {
		return
	}

	// Try to extract actor info
	if uid, ok := data["user_id"].(string); ok && uid != "" {
		actorID = uid
		actorType = models.ActorUser
	} else if uid, ok := data["actor_id"].(string); ok && uid != "" {
		actorID = uid
		actorType = models.ActorUser
	}

	// Try to extract resource info
	for _, key := range []string{"resource_type", "type"} {
		if rt, ok := data[key].(string); ok && rt != "" {
			resourceType = &rt
			break
		}
	}
	for _, key := range []string{"resource_id", "id", "org_id", "template_id", "notification_id", "file_id"} {
		if rid, ok := data[key].(string); ok && rid != "" {
			resourceID = &rid
			break
		}
	}

	// Try to extract action
	if a, ok := data["action"].(string); ok && a != "" {
		action = a
	}

	// Try to extract outcome
	if o, ok := data["outcome"].(string); ok {
		if o == "failure" || o == "failed" {
			outcome = models.OutcomeFailure
		}
	}
	if _, ok := data["error"].(string); ok {
		outcome = models.OutcomeFailure
	}

	return
}
