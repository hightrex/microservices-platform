package channels

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"regexp"
	"strings"
	"time"

	"github.com/hightrex/microservices-platform/libs/go/pkg/logger"
	"github.com/hightrex/microservices-platform/services/notification-service/internal/models"
)

const (
	// maxSMSLength is the maximum character length for a single SMS message.
	maxSMSLength = 160
	// twilioAPIBaseURL is the Twilio REST API base URL.
	twilioAPIBaseURL = "https://api.twilio.com/2010-04-01"
)

// e164Regex validates phone numbers in E.164 format.
var e164Regex = regexp.MustCompile(`^\+[1-9]\d{1,14}$`)

// SMSConfig holds Twilio configuration.
type SMSConfig struct {
	AccountSID string
	AuthToken  string
	FromNumber string
}

// SMSChannel sends notifications via Twilio SMS.
type SMSChannel struct {
	config SMSConfig
	client *http.Client
}

// NewSMSChannel creates a new SMSChannel.
func NewSMSChannel(cfg SMSConfig) *SMSChannel {
	return &SMSChannel{
		config: cfg,
		client: &http.Client{
			Timeout: 30 * time.Second,
		},
	}
}

// Channel returns the channel type.
func (c *SMSChannel) Channel() models.Channel {
	return models.ChannelSMS
}

// Send sends an SMS notification via Twilio with retry logic.
func (c *SMSChannel) Send(ctx context.Context, recipient, subject, body string, metadata map[string]interface{}) error {
	// Validate phone number format (E.164)
	if !e164Regex.MatchString(recipient) {
		return fmt.Errorf("invalid phone number format: must be E.164 (e.g., +15551234567)")
	}

	// Truncate message if too long
	message := body
	if len(message) > maxSMSLength {
		logger.Warn().
			Int("original_length", len(message)).
			Int("max_length", maxSMSLength).
			Msg("SMS message truncated")
		message = message[:maxSMSLength-3] + "..."
	}

	var lastErr error
	for attempt := 0; attempt < 3; attempt++ {
		if ctx.Err() != nil {
			return ctx.Err()
		}

		lastErr = c.sendTwilioSMS(ctx, recipient, message)
		if lastErr == nil {
			return nil
		}

		logger.Warn().Err(lastErr).
			Int("attempt", attempt+1).
			Str("recipient", maskPhone(recipient)).
			Msg("SMS send attempt failed, retrying")

		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(time.Duration(1<<attempt) * time.Second):
		}
	}

	return fmt.Errorf("SMS delivery failed after 3 attempts: %w", lastErr)
}

func (c *SMSChannel) sendTwilioSMS(ctx context.Context, to, body string) error {
	apiURL := fmt.Sprintf("%s/Accounts/%s/Messages.json", twilioAPIBaseURL, c.config.AccountSID)

	data := url.Values{}
	data.Set("To", to)
	data.Set("From", c.config.FromNumber)
	data.Set("Body", body)

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, apiURL, strings.NewReader(data.Encode()))
	if err != nil {
		return fmt.Errorf("failed to create Twilio request: %w", err)
	}

	req.SetBasicAuth(c.config.AccountSID, c.config.AuthToken)
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	resp, err := c.client.Do(req)
	if err != nil {
		return fmt.Errorf("Twilio API call failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		var twilioErr struct {
			Message string `json:"message"`
			Code    int    `json:"code"`
		}
		if err := json.NewDecoder(resp.Body).Decode(&twilioErr); err == nil {
			return fmt.Errorf("Twilio error (code %d): %s", twilioErr.Code, twilioErr.Message)
		}
		return fmt.Errorf("Twilio API returned status %d", resp.StatusCode)
	}

	return nil
}

// maskPhone masks a phone number for logging (show last 4 digits only).
func maskPhone(phone string) string {
	if len(phone) <= 4 {
		return "****"
	}
	return strings.Repeat("*", len(phone)-4) + phone[len(phone)-4:]
}
