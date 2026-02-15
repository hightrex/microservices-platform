package models

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestChannelIsValid(t *testing.T) {
	tests := []struct {
		channel Channel
		valid   bool
	}{
		{ChannelEmail, true},
		{ChannelSMS, true},
		{ChannelInApp, true},
		{ChannelWebhook, true},
		{Channel("carrier_pigeon"), false},
		{Channel(""), false},
	}

	for _, tt := range tests {
		t.Run(string(tt.channel), func(t *testing.T) {
			assert.Equal(t, tt.valid, tt.channel.IsValid())
		})
	}
}

func TestValidChannels(t *testing.T) {
	channels := ValidChannels()
	assert.Len(t, channels, 4)
	assert.Contains(t, channels, ChannelEmail)
	assert.Contains(t, channels, ChannelSMS)
	assert.Contains(t, channels, ChannelInApp)
	assert.Contains(t, channels, ChannelWebhook)
}

func TestNotificationTemplateToResponse(t *testing.T) {
	subject := "Test Subject"
	tmpl := &NotificationTemplate{
		Name:            "test-template",
		Channel:         ChannelEmail,
		SubjectTemplate: &subject,
		BodyTemplate:    "Hello {{.Name}}",
		Language:        "en",
		IsActive:        true,
	}

	resp := tmpl.ToResponse()

	assert.Equal(t, tmpl.Name, resp.Name)
	assert.Equal(t, tmpl.Channel, resp.Channel)
	assert.Equal(t, tmpl.SubjectTemplate, resp.SubjectTemplate)
	assert.Equal(t, tmpl.BodyTemplate, resp.BodyTemplate)
	assert.Equal(t, tmpl.Language, resp.Language)
	assert.Equal(t, tmpl.IsActive, resp.IsActive)
}
