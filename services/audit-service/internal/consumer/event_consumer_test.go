package consumer

import (
	"testing"

	"github.com/hightrex/microservices-platform/services/audit-service/internal/models"
	"github.com/stretchr/testify/assert"
)

func TestCategorizeEvent(t *testing.T) {
	tests := []struct {
		eventType string
		expected  models.EventCategory
	}{
		{"user.created", models.CategoryAuth},
		{"user.updated", models.CategoryAuth},
		{"auth.login", models.CategoryAuth},
		{"org.created", models.CategoryData},
		{"billing.payment", models.CategoryData},
		{"file.uploaded", models.CategoryData},
		{"notification.sent", models.CategorySystem},
		{"audit.export_completed", models.CategorySystem},
		{"security.breach", models.CategorySecurity},
		{"unknown.event", models.CategorySystem}, // Default
	}

	for _, tt := range tests {
		t.Run(tt.eventType, func(t *testing.T) {
			result := categorizeEvent(tt.eventType)
			assert.Equal(t, tt.expected, result)
		})
	}
}
