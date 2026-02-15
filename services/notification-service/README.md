# Notification Service

Multi-channel notification service with template engine, preference management, and delivery tracking.

## Architecture

- **Language**: Go 1.24 with Gin framework
- **Port**: 8082
- **Database**: PostgreSQL (notification_service)
- **Cache**: Redis (DB 2 for cache, DB 0 for messaging)
- **Channels**: Email (SMTP), SMS (Twilio), In-App (DB), Webhook (HTTP POST)

## Features

- **Template Engine**: Go html/template with safe function whitelist (XSS prevention)
- **Multi-Channel Delivery**: Email, SMS, in-app, and webhook notifications
- **User Preferences**: Per-user opt-in/opt-out for each channel and event type
- **Delivery Tracking**: Full audit trail of every delivery attempt
- **Event-Driven**: Consumes events from other services and auto-sends notifications
- **Retry Logic**: Exponential backoff with configurable retry attempts per channel
- **SSRF Prevention**: Webhook URLs validated against private IPs, localhost, and cloud metadata endpoints
- **Dead Letter Queue**: Failed notifications tracked for investigation

## API Endpoints

### Notifications
| Method | Endpoint | Description |
|--------|----------|-------------|
| POST | `/api/v1/notifications/send` | Send a notification |
| GET | `/api/v1/notifications` | List in-app notifications (paginated) |
| GET | `/api/v1/notifications/:id` | Get notification by ID |
| PUT | `/api/v1/notifications/:id/read` | Mark as read |
| GET | `/api/v1/notifications/unread/count` | Get unread count |

### Templates (admin only for write operations)
| Method | Endpoint | Description |
|--------|----------|-------------|
| POST | `/api/v1/notifications/templates` | Create template |
| GET | `/api/v1/notifications/templates` | List templates |
| GET | `/api/v1/notifications/templates/:id` | Get template |
| PUT | `/api/v1/notifications/templates/:id` | Update template |
| DELETE | `/api/v1/notifications/templates/:id` | Deactivate template |

### Preferences
| Method | Endpoint | Description |
|--------|----------|-------------|
| GET | `/api/v1/notifications/preferences` | Get user preferences |
| PUT | `/api/v1/notifications/preferences` | Bulk update preferences |
| PUT | `/api/v1/notifications/preferences/:channel/:event_type` | Update single preference |

## Event Streams

### Consumes
- `auth-events`: `user.created` → Welcome notification
- `org-events`: `org.created` → Organization created notification

### Publishes (to `notification-events`)
- `notification.sent` — Successful delivery
- `notification.failed` — Delivery failure
- `notification.template_created` — New template created
- `notification.preference_updated` — User preference change

## Setup

```bash
# Run scaffold (already done)
scripts/create-service.sh notification-service

# Build
cd services/notification-service && go build ./...

# Test
go test ./...

# Run locally
go run ./cmd/main.go
```

## Configuration

See `config.yaml` for all configuration options. Environment variables override YAML config.

### Channel Configuration

- **Email**: Configure SMTP settings (MailHog at localhost:1025 for local dev)
- **SMS**: Configure Twilio credentials
- **Webhook**: Configure timeout, retry attempts, max redirects
- **In-App**: No additional config (uses database storage)

## Template Syntax

Templates use Go's `html/template` syntax with a safe function whitelist:

```
Hello {{.Name}}, welcome to {{.OrgName}}!
Your role: {{upper .Role}}
```

Available functions: `upper`, `lower`, `title`, `trim`
