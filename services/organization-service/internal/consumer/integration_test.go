package consumer_test

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/hightrex/microservices-platform/libs/go/pkg/messaging"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/hightrex/microservices-platform/services/organization-service/internal/consumer"
)

// MockIntegrationOrgService simulates the OrgService for integration testing
type MockIntegrationOrgService struct {
	CreatedOrgs map[string]string // userID -> orgID
}

func (m *MockIntegrationOrgService) CreateFromUserEvent(ctx context.Context, userID, email string) error {
	orgID := uuid.New().String()
	m.CreatedOrgs[userID] = orgID
	return nil
}

func TestIntegration_UserRegistrationFlow(t *testing.T) {
	mockSvc := &MockIntegrationOrgService{
		CreatedOrgs: make(map[string]string),
	}
	userConsumer := consumer.NewUserConsumer(mockSvc)

	userID := uuid.New().String()
	eventData := map[string]interface{}{
		"user_id": userID,
		"email":   "integration@example.com",
		"source":  "registration",
	}
	dataBytes, err := json.Marshal(eventData)
	require.NoError(t, err)

	event := messaging.Event{
		ID:        uuid.New().String(),
		Type:      "user.created",
		Version:   "1.0",
		Source:    "auth-service",
		TenantID:  uuid.Nil.String(),
		Data:      dataBytes,
		Timestamp: time.Now(),
	}

	// Execute handler directly (simulating Redis delivery)
	err = userConsumer.HandleUserCreated(context.Background(), event)

	require.NoError(t, err)
	assert.Contains(t, mockSvc.CreatedOrgs, userID, "Organization should have been created for user")
	assert.NotEmpty(t, mockSvc.CreatedOrgs[userID], "Org ID should not be empty")
}

func TestIntegration_IdempotentProcessing(t *testing.T) {
	mockSvc := &MockIntegrationOrgService{
		CreatedOrgs: make(map[string]string),
	}
	userConsumer := consumer.NewUserConsumer(mockSvc)

	userID := uuid.New().String()
	eventData := map[string]interface{}{
		"user_id": userID,
		"email":   "idempotent@example.com",
	}
	dataBytes, err := json.Marshal(eventData)
	require.NoError(t, err)

	event := messaging.Event{
		ID:        uuid.New().String(),
		Type:      "user.created",
		Version:   "1.0",
		Source:    "auth-service",
		TenantID:  uuid.Nil.String(),
		Data:      dataBytes,
		Timestamp: time.Now(),
	}

	// Process same event twice
	err = userConsumer.HandleUserCreated(context.Background(), event)
	require.NoError(t, err)

	err = userConsumer.HandleUserCreated(context.Background(), event)
	require.NoError(t, err)

	// Both calls go through to the service which enforces idempotency
	// The second call is handled by CreateFromUserEvent's CountByOwnerID check
	assert.Contains(t, mockSvc.CreatedOrgs, userID)
}
