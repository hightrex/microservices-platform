package consumer

import (
	"context"
	"encoding/json"
	"errors"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/hightrex/microservices-platform/libs/go/pkg/messaging"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

// MockOrgService is a mock implementation of the OrgCreator interface
type MockOrgService struct {
	mock.Mock
}

func (m *MockOrgService) CreateFromUserEvent(ctx context.Context, userID, email string) error {
	args := m.Called(ctx, userID, email)
	return args.Error(0)
}

func TestUserCreatedHandler_HandleUserCreated(t *testing.T) {
	tests := []struct {
		name          string
		eventData     map[string]interface{}
		mockReturnErr error
		expectCall    bool // whether CreateFromUserEvent should be called
		expectError   bool
	}{
		{
			name: "Success — creates organization for new user",
			eventData: map[string]interface{}{
				"user_id": uuid.New().String(),
				"email":   "test@example.com",
				"source":  "registration",
			},
			mockReturnErr: nil,
			expectCall:    true,
			expectError:   false,
		},
		{
			name: "Missing user_id — skips silently (no error)",
			eventData: map[string]interface{}{
				"email": "test@example.com",
			},
			mockReturnErr: nil,
			expectCall:    false,
			expectError:   false,
		},
		{
			name: "Empty user_id — skips silently (no error)",
			eventData: map[string]interface{}{
				"user_id": "",
				"email":   "test@example.com",
			},
			mockReturnErr: nil,
			expectCall:    false,
			expectError:   false,
		},
		{
			name: "Invalid user_id type — unmarshal error",
			eventData: map[string]interface{}{
				"user_id": 12345, // Number instead of string
				"email":   "test@example.com",
			},
			mockReturnErr: nil,
			expectCall:    false,
			expectError:   true, // json.Unmarshal fails on type mismatch
		},
		{
			name: "OrgService failure — returns error for retry",
			eventData: map[string]interface{}{
				"user_id": uuid.New().String(),
				"email":   "test@example.com",
			},
			mockReturnErr: errors.New("database connection failed"),
			expectCall:    true,
			expectError:   true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockOrgService := new(MockOrgService)
			uc := NewUserConsumer(mockOrgService)

			dataBytes, err := json.Marshal(tt.eventData)
			assert.NoError(t, err)

			event := messaging.Event{
				ID:        uuid.New().String(),
				Type:      "user.created",
				Version:   "1.0",
				Source:    "auth-service",
				TenantID:  uuid.Nil.String(),
				Data:      dataBytes,
				Timestamp: time.Now(),
			}

			if tt.expectCall {
				userID := tt.eventData["user_id"].(string)
				email := tt.eventData["email"].(string)
				mockOrgService.On("CreateFromUserEvent", mock.Anything, userID, email).Return(tt.mockReturnErr)
			}

			err = uc.HandleUserCreated(context.Background(), event)

			if tt.expectError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}

			if tt.expectCall {
				mockOrgService.AssertExpectations(t)
			} else {
				mockOrgService.AssertNotCalled(t, "CreateFromUserEvent", mock.Anything, mock.Anything, mock.Anything)
			}
		})
	}
}

func TestUserCreatedHandler_TenantContextInjection(t *testing.T) {
	tenantID := uuid.New()
	mockOrgService := new(MockOrgService)
	uc := NewUserConsumer(mockOrgService)

	userID := uuid.New().String()
	dataBytes, _ := json.Marshal(map[string]interface{}{
		"user_id": userID,
		"email":   "tenant@example.com",
	})

	event := messaging.Event{
		ID:        uuid.New().String(),
		Type:      "user.created",
		Version:   "1.0",
		Source:    "auth-service",
		TenantID:  tenantID.String(),
		Data:      dataBytes,
		Timestamp: time.Now(),
	}

	mockOrgService.On("CreateFromUserEvent", mock.Anything, userID, "tenant@example.com").Return(nil)

	err := uc.HandleUserCreated(context.Background(), event)
	assert.NoError(t, err)
	mockOrgService.AssertExpectations(t)
}
