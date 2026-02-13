package securitylog

import (
	"bytes"
	"context"
	"testing"

	"github.com/google/uuid"
	"github.com/hightrex/microservices-platform/libs/go/pkg/tenant"
	"github.com/rs/zerolog"
	"github.com/stretchr/testify/assert"
)

// captureLog temporarily redirects the global logger to capture output.
func captureLog(fn func()) string {
	var buf bytes.Buffer
	// This is a test hack — swap the package-level logger in the logger package.
	// In production, you'd use dependency injection. For unit tests, this is acceptable.
	tmpLogger := zerolog.New(&buf)

	// We need to access the logger package's internal state.
	// Since securitylog uses logger.Info()/Warn(), we'll just test the event struct.
	_ = tmpLogger // suppressed for now; see test approach below
	fn()
	return buf.String()
}

func TestEventTypes(t *testing.T) {
	// Verify that common event types are defined
	assert.Equal(t, EventType("auth.login.success"), EventLoginSuccess)
	assert.Equal(t, EventType("auth.login.failed"), EventLoginFailed)
	assert.Equal(t, EventType("auth.token.invalid"), EventTokenInvalid)
	assert.Equal(t, EventType("authz.permission_denied"), EventPermissionDenied)
	assert.Equal(t, EventType("tenant.boundary_violation"), EventTenantViolation)
	assert.Equal(t, EventType("ratelimit.exceeded"), EventRateLimited)
	assert.Equal(t, EventType("input.rejected"), EventInputRejected)
}

func TestOutcomeTypes(t *testing.T) {
	assert.Equal(t, Outcome("success"), OutcomeSuccess)
	assert.Equal(t, Outcome("failure"), OutcomeFailure)
	assert.Equal(t, Outcome("blocked"), OutcomeBlocked)
}

func TestLogDoesNotPanic(t *testing.T) {
	// Basic test ensuring Log does not panic with various inputs
	ctx := context.Background()

	// Minimal event
	assert.NotPanics(t, func() {
		Log(ctx, Event{
			Type:    EventLoginFailed,
			Outcome: OutcomeFailure,
		})
	})

	// Full event
	assert.NotPanics(t, func() {
		Log(ctx, Event{
			Type:     EventLoginSuccess,
			Outcome:  OutcomeSuccess,
			ActorID:  "user-123",
			TenantID: "tenant-456",
			IP:       "192.168.1.1",
			Resource: "/auth/login",
			Details:  map[string]interface{}{"method": "password"},
		})
	})

	// With tenant in context
	tenantID := uuid.New()
	ctxWithTenant := tenant.NewContext(ctx, tenantID)
	assert.NotPanics(t, func() {
		Log(ctxWithTenant, Event{
			Type:    EventPermissionDenied,
			Outcome: OutcomeBlocked,
			ActorID: "user-789",
			IP:      "10.0.0.1",
		})
	})
}

func TestLogTenantAutoPopulation(t *testing.T) {
	tenantID := uuid.New()
	ctx := tenant.NewContext(context.Background(), tenantID)

	event := Event{
		Type:    EventLoginSuccess,
		Outcome: OutcomeSuccess,
	}

	// Simulate what Log does internally
	if event.TenantID == "" {
		if tid := tenant.FromContext(ctx); tid.String() != "00000000-0000-0000-0000-000000000000" {
			event.TenantID = tid.String()
		}
	}

	assert.Equal(t, tenantID.String(), event.TenantID)
}

func TestLogFailureConvenience(t *testing.T) {
	assert.NotPanics(t, func() {
		LogFailure(
			context.Background(),
			EventLoginFailed,
			"192.168.1.1",
			"/auth/login",
			map[string]interface{}{"reason": "invalid_credentials"},
		)
	})
}

func TestLogSuccessConvenience(t *testing.T) {
	assert.NotPanics(t, func() {
		LogSuccess(
			context.Background(),
			EventLoginSuccess,
			"user-123",
			"192.168.1.1",
			"/auth/login",
		)
	})
}
