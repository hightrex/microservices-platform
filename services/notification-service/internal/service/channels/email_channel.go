package channels

import (
	"context"
	"crypto/tls"
	"fmt"
	"net"
	"net/smtp"
	"time"

	"github.com/hightrex/microservices-platform/libs/go/pkg/logger"
	"github.com/hightrex/microservices-platform/services/notification-service/internal/models"
)

// EmailConfig holds SMTP configuration.
type EmailConfig struct {
	Host        string
	Port        int
	Username    string
	Password    string
	FromAddress string
	FromName    string
	UseTLS      bool
}

// EmailChannel sends notifications via SMTP email.
type EmailChannel struct {
	config EmailConfig
}

// NewEmailChannel creates a new EmailChannel.
func NewEmailChannel(cfg EmailConfig) *EmailChannel {
	return &EmailChannel{config: cfg}
}

// Channel returns the channel type.
func (c *EmailChannel) Channel() models.Channel {
	return models.ChannelEmail
}

// Send sends an email notification with retry logic (3 attempts).
func (c *EmailChannel) Send(ctx context.Context, recipient, subject, body string, metadata map[string]interface{}) error {
	var lastErr error
	for attempt := 0; attempt < 3; attempt++ {
		if ctx.Err() != nil {
			return ctx.Err()
		}

		lastErr = c.sendEmail(recipient, subject, body)
		if lastErr == nil {
			return nil
		}

		logger.Warn().Err(lastErr).
			Int("attempt", attempt+1).
			Str("recipient", recipient).
			Msg("Email send attempt failed, retrying")

		// Exponential backoff: 1s, 2s, 4s
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(time.Duration(1<<attempt) * time.Second):
		}
	}

	return fmt.Errorf("email delivery failed after 3 attempts: %w", lastErr)
}

func (c *EmailChannel) sendEmail(to, subject, body string) error {
	addr := net.JoinHostPort(c.config.Host, fmt.Sprintf("%d", c.config.Port))

	from := c.config.FromAddress
	if c.config.FromName != "" {
		from = fmt.Sprintf("%s <%s>", c.config.FromName, c.config.FromAddress)
	}

	msg := fmt.Sprintf("From: %s\r\nTo: %s\r\nSubject: %s\r\nMIME-Version: 1.0\r\nContent-Type: text/html; charset=UTF-8\r\n\r\n%s",
		from, to, subject, body)

	// Connect with timeout
	conn, err := net.DialTimeout("tcp", addr, 10*time.Second)
	if err != nil {
		return fmt.Errorf("failed to connect to SMTP server: %w", err)
	}

	client, err := smtp.NewClient(conn, c.config.Host)
	if err != nil {
		conn.Close()
		return fmt.Errorf("failed to create SMTP client: %w", err)
	}
	defer client.Close()

	// STARTTLS if configured
	if c.config.UseTLS {
		tlsConfig := &tls.Config{
			ServerName: c.config.Host,
			MinVersion: tls.VersionTLS12,
		}
		if err := client.StartTLS(tlsConfig); err != nil {
			return fmt.Errorf("STARTTLS failed: %w", err)
		}
	}

	// Authenticate if credentials provided
	if c.config.Username != "" {
		auth := smtp.PlainAuth("", c.config.Username, c.config.Password, c.config.Host)
		if err := client.Auth(auth); err != nil {
			return fmt.Errorf("SMTP auth failed: %w", err)
		}
	}

	if err := client.Mail(c.config.FromAddress); err != nil {
		return fmt.Errorf("MAIL FROM failed: %w", err)
	}
	if err := client.Rcpt(to); err != nil {
		return fmt.Errorf("RCPT TO failed: %w", err)
	}

	w, err := client.Data()
	if err != nil {
		return fmt.Errorf("DATA command failed: %w", err)
	}

	if _, err := w.Write([]byte(msg)); err != nil {
		return fmt.Errorf("failed to write email body: %w", err)
	}

	if err := w.Close(); err != nil {
		return fmt.Errorf("failed to close email data: %w", err)
	}

	return client.Quit()
}
