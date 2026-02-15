package channels

import (
	"bytes"
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/hightrex/microservices-platform/libs/go/pkg/logger"
	"github.com/hightrex/microservices-platform/services/notification-service/internal/models"
)

// WebhookConfig holds webhook delivery configuration.
type WebhookConfig struct {
	TimeoutSeconds int
	RetryAttempts  int
	MaxRedirects   int
	SigningSecret   string
}

// WebhookChannel sends notifications via HTTP POST webhook.
type WebhookChannel struct {
	config WebhookConfig
	client *http.Client
}

// NewWebhookChannel creates a new WebhookChannel.
func NewWebhookChannel(cfg WebhookConfig) *WebhookChannel {
	timeout := time.Duration(cfg.TimeoutSeconds) * time.Second
	if timeout == 0 {
		timeout = 10 * time.Second
	}

	return &WebhookChannel{
		config: cfg,
		client: &http.Client{
			Timeout: timeout,
			CheckRedirect: func(req *http.Request, via []*http.Request) error {
				maxRedirects := cfg.MaxRedirects
				if maxRedirects == 0 {
					maxRedirects = 3
				}
				if len(via) >= maxRedirects {
					return fmt.Errorf("stopped after %d redirects", maxRedirects)
				}
				return nil
			},
		},
	}
}

// Channel returns the channel type.
func (c *WebhookChannel) Channel() models.Channel {
	return models.ChannelWebhook
}

// Send delivers a notification via HTTP POST to the webhook URL with HMAC signature.
func (c *WebhookChannel) Send(ctx context.Context, recipient, subject, body string, metadata map[string]interface{}) error {
	retryAttempts := c.config.RetryAttempts
	if retryAttempts == 0 {
		retryAttempts = 5
	}

	var lastErr error
	for attempt := 0; attempt < retryAttempts; attempt++ {
		if ctx.Err() != nil {
			return ctx.Err()
		}

		lastErr = c.sendWebhook(ctx, recipient, subject, body, metadata)
		if lastErr == nil {
			return nil
		}

		logger.Warn().Err(lastErr).
			Int("attempt", attempt+1).
			Str("url", recipient).
			Msg("Webhook delivery attempt failed, retrying")

		// Exponential backoff: 1s, 2s, 4s, 8s, 16s
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(time.Duration(1<<attempt) * time.Second):
		}
	}

	return fmt.Errorf("webhook delivery failed after %d attempts: %w", retryAttempts, lastErr)
}

func (c *WebhookChannel) sendWebhook(ctx context.Context, url, subject, body string, metadata map[string]interface{}) error {
	payload := map[string]interface{}{
		"subject":   subject,
		"body":      body,
		"metadata":  metadata,
		"timestamp": time.Now().UTC().Format(time.RFC3339),
	}

	jsonPayload, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("failed to marshal webhook payload: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(jsonPayload))
	if err != nil {
		return fmt.Errorf("failed to create webhook request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("User-Agent", "MicroservicesPlatform-Webhook/1.0")

	// Add HMAC signature for webhook verification
	if c.config.SigningSecret != "" {
		signature := generateHMACSignature(jsonPayload, c.config.SigningSecret)
		req.Header.Set("X-Webhook-Signature", signature)
	}

	resp, err := c.client.Do(req)
	if err != nil {
		return fmt.Errorf("webhook request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		return fmt.Errorf("webhook returned status %d", resp.StatusCode)
	}

	return nil
}

// generateHMACSignature generates an HMAC-SHA256 signature for webhook verification.
func generateHMACSignature(payload []byte, secret string) string {
	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write(payload)
	return "sha256=" + hex.EncodeToString(mac.Sum(nil))
}
