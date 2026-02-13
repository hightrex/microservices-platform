// Package securitylog provides a structured security event logging system
// for the microservices platform. It wraps the standard logger with
// predefined security event types and standardized fields to ensure
// consistent security audit trails across all services.
//
// Phase 0 establishes the patterns; Phase 1 services log real events.
//
// Usage:
//
//	securitylog.Log(ctx, securitylog.Event{
//	    Type:     securitylog.EventLoginFailed,
//	    Outcome:  securitylog.OutcomeFailure,
//	    ActorID:  "",  // unknown actor on failed login
//	    TenantID: "",
//	    IP:       c.ClientIP(),
//	    Resource: "auth/login",
//	    Details:  map[string]interface{}{"reason": "invalid_credentials", "email": email},
//	})
package securitylog

import (
	"context"

	"github.com/hightrex/microservices-platform/libs/go/pkg/logger"
	"github.com/hightrex/microservices-platform/libs/go/pkg/tenant"
)

// EventType identifies the category of security event.
type EventType string

const (
	// Authentication events
	EventLoginSuccess     EventType = "auth.login.success"
	EventLoginFailed      EventType = "auth.login.failed"
	EventLogout           EventType = "auth.logout"
	EventTokenRefreshed   EventType = "auth.token.refreshed"
	EventTokenInvalid     EventType = "auth.token.invalid"
	EventTokenExpired     EventType = "auth.token.expired"
	EventMFASuccess       EventType = "auth.mfa.success"
	EventMFAFailed        EventType = "auth.mfa.failed"
	EventPasswordChanged  EventType = "auth.password.changed"
	EventPasswordResetReq EventType = "auth.password.reset_requested"

	// Authorization events
	EventPermissionDenied EventType = "authz.permission_denied"
	EventRoleChanged      EventType = "authz.role.changed"
	EventAPIKeyCreated    EventType = "authz.apikey.created"
	EventAPIKeyRevoked    EventType = "authz.apikey.revoked"

	// Tenant boundary events
	EventTenantViolation EventType = "tenant.boundary_violation"
	EventTenantCreated   EventType = "tenant.created"
	EventTenantDeleted   EventType = "tenant.deleted"

	// Data access events
	EventDataExported    EventType = "data.exported"
	EventBulkDelete      EventType = "data.bulk_delete"
	EventSensitiveAccess EventType = "data.sensitive_access"

	// Account events
	EventAccountCreated  EventType = "account.created"
	EventAccountDisabled EventType = "account.disabled"
	EventAccountDeleted  EventType = "account.deleted"
	EventAccountLocked   EventType = "account.locked"

	// Rate limiting
	EventRateLimited EventType = "ratelimit.exceeded"

	// Input validation / abuse
	EventInputRejected       EventType = "input.rejected"
	EventSQLInjectionAttempt EventType = "input.sqli_attempt"
)

// Outcome indicates whether the security event succeeded or failed.
type Outcome string

const (
	OutcomeSuccess Outcome = "success"
	OutcomeFailure Outcome = "failure"
	OutcomeBlocked Outcome = "blocked"
)

// Event represents a security event to be logged.
type Event struct {
	// Type categorizes the event (required).
	Type EventType

	// Outcome indicates success/failure/blocked (required).
	Outcome Outcome

	// ActorID is the user/service performing the action (if known).
	ActorID string

	// TenantID is the tenant context. If empty, will be extracted from ctx.
	TenantID string

	// IP is the source IP address of the request (if applicable).
	IP string

	// Resource is the target resource (endpoint, entity, etc.).
	Resource string

	// Details contains additional event-specific metadata.
	Details map[string]interface{}
}

// Log writes a structured security event to the log output.
// It automatically enriches the event with tenant context if available.
func Log(ctx context.Context, event Event) {
	// Auto-populate tenant from context if not explicitly set
	if event.TenantID == "" {
		if tid := tenant.FromContext(ctx); tid.String() != "00000000-0000-0000-0000-000000000000" {
			event.TenantID = tid.String()
		}
	}

	logEvent := logger.Info()
	if event.Outcome == OutcomeFailure || event.Outcome == OutcomeBlocked {
		logEvent = logger.Warn()
	}

	// Always include these standard fields for searchability
	logEvent.
		Str("security_event", string(event.Type)).
		Str("outcome", string(event.Outcome)).
		Str("actor_id", event.ActorID).
		Str("tenant_id", event.TenantID).
		Str("ip", event.IP).
		Str("resource", event.Resource)

	// Include any additional details
	if event.Details != nil {
		for k, v := range event.Details {
			logEvent.Interface(k, v)
		}
	}

	logEvent.Msg("security_event")
}

// LogFailure is a convenience wrapper for logging failed security events.
func LogFailure(ctx context.Context, eventType EventType, ip, resource string, details map[string]interface{}) {
	Log(ctx, Event{
		Type:     eventType,
		Outcome:  OutcomeFailure,
		IP:       ip,
		Resource: resource,
		Details:  details,
	})
}

// LogSuccess is a convenience wrapper for logging successful security events.
func LogSuccess(ctx context.Context, eventType EventType, actorID, ip, resource string) {
	Log(ctx, Event{
		Type:     eventType,
		Outcome:  OutcomeSuccess,
		ActorID:  actorID,
		IP:       ip,
		Resource: resource,
	})
}
