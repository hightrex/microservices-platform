# Phase 2: Modular Services (Weeks 11–18)

> **Status**: ⚪ NOT STARTED
> **Prerequisite**: Phase 1 complete (`m1-core-platform-mvp`)
> **Milestone**: M2 — All 8 services running, module gating working, full event-driven architecture
> **Dependency order**: Shared Rust Crate → (Billing + File + Notification + Audit in parallel) → Integration

---

## 2.1 Shared Rust Crate (`libs/rust/`)

The Rust shared library provides common functionality for Billing and File services. **Build first** — the Rust services depend on it.

### 2.1.1 Workspace Setup
- [ ] Create `libs/rust/Cargo.toml` — Cargo workspace definition
- [ ] Create workspace members:
  - [ ] `libs/rust/common/` — core utilities
  - [ ] `libs/rust/messaging/` — Redis Streams integration
  - [ ] `libs/rust/database/` — Postgres connection pooling
  - [ ] `libs/rust/middleware/` — Axum middleware
- [ ] Set edition = "2021" and consistent dependency versions across workspace
- [ ] Add shared dependencies in workspace `Cargo.toml`:
  - [ ] `tokio`, `serde`, `serde_json`, `anyhow`, `thiserror`
  - [ ] `tracing`, `tracing-subscriber`
  - [ ] `uuid`, `chrono`
- [ ] Verify `cargo build --workspace` compiles

### 2.1.2 Common Crate (`libs/rust/common/`)
- [ ] `src/config.rs` — configuration loading (environment variables, TOML)
  - [ ] `DatabaseConfig` struct (host, port, database, user, password, pool size)
  - [ ] `RedisConfig` struct (host, port, password, db)
  - [ ] `ServerConfig` struct (host, port, cors_origins)
  - [ ] `load_config()` function with validation
- [ ] `src/error.rs` — standardized error types
  - [ ] `AppError` enum (NotFound, Unauthorized, Forbidden, ValidationError, DatabaseError, ExternalError, InternalError)
  - [ ] Implement `std::fmt::Display` and `std::error::Error`
  - [ ] Conversion to Axum responses (status codes + JSON body)
  - [ ] Error code constants matching Go shared lib
- [ ] `src/logger.rs` — structured logging setup
  - [ ] Initialize `tracing_subscriber` with JSON formatter
  - [ ] Log level from environment variable
  - [ ] Add request ID to context
- [ ] `src/tenant.rs` — tenant context extraction
  - [ ] `TenantContext` struct with `tenant_id` and `org_id`
  - [ ] Extract from Axum headers (`X-Tenant-ID`, `X-Org-ID`)
  - [ ] `require_tenant()` helper that returns error if missing
- [ ] `src/validation.rs` — input validation
  - [ ] Integration with `validator` crate
  - [ ] `validate_request()` generic function
  - [ ] Validation error formatting to match standardized API error format
- [ ] Unit tests for each module

### 2.1.3 Database Crate (`libs/rust/database/`)
- [ ] `src/pool.rs` — Postgres connection pooling
  - [ ] Use `sqlx::PgPool` with configuration from `common::config`
  - [ ] Connection health checks
  - [ ] Connection lifecycle management
- [ ] `src/migrations.rs` — migration runner
  - [ ] Use `sqlx::migrate!()` macro
  - [ ] `run_migrations()` function
- [ ] `src/repository.rs` — base repository traits
  - [ ] `TenantScoped` trait for tenant isolation
  - [ ] Query builder helpers
- [ ] Unit tests with test containers

### 2.1.4 Messaging Crate (`libs/rust/messaging/`)
- [ ] `src/producer.rs` — Redis Streams producer
  - [ ] `Producer` struct with Redis connection
  - [ ] `publish()` method with event schema validation
  - [ ] Automatic timestamp and correlation ID injection
- [ ] `src/consumer.rs` — Redis Streams consumer
  - [ ] `Consumer` struct with consumer group support
  - [ ] `subscribe()` method with message acknowledgment
  - [ ] Dead letter queue (DLQ) support
  - [ ] Retry logic with exponential backoff
- [ ] `src/schema.rs` — event schema definitions
  - [ ] Common event structure (event_type, tenant_id, timestamp, data)
  - [ ] Serialization/deserialization with `serde`
- [ ] Integration tests against real Redis

### 2.1.5 Middleware Crate (`libs/rust/middleware/`)
- [ ] `src/auth.rs` — JWT validation middleware
  - [ ] Extract and validate JWT from `Authorization` header
  - [ ] Parse claims (sub, tid, org, roles, exp)
  - [ ] Add user context to Axum extensions
  - [ ] Token blacklist check (Redis)
- [ ] `src/tenant.rs` — tenant extraction middleware
  - [ ] Extract `X-Tenant-ID` header
  - [ ] Validate against JWT claims
  - [ ] Add to request extensions
- [ ] `src/logging.rs` — request logging middleware
  - [ ] Log request method, path, status, duration
  - [ ] Include request ID, tenant ID, user ID
- [ ] `src/metrics.rs` — Prometheus metrics middleware
  - [ ] Request duration histogram
  - [ ] Request counter by method/path/status
  - [ ] Active request gauge
- [ ] `src/cors.rs` — CORS middleware
  - [ ] Configurable allowed origins
  - [ ] Proper preflight handling
- [ ] `src/rate_limit.rs` — rate limiting middleware
  - [ ] Redis-backed sliding window
  - [ ] Per-tenant and per-user limits
  - [ ] Return `X-RateLimit-*` headers
  - [ ] 429 status with `Retry-After`
- [ ] Unit tests for each middleware

### 2.1.6 Health Check Module
- [ ] `libs/rust/common/src/health.rs` — health check framework
  - [ ] `HealthCheck` trait
  - [ ] `DatabaseHealthCheck` implementation
  - [ ] `RedisHealthCheck` implementation
  - [ ] Aggregate health status endpoint
  - [ ] Dependency status reporting

### 2.1.7 Tracing & Observability
- [ ] `libs/rust/common/src/tracing.rs` — OpenTelemetry setup
  - [ ] Initialize OpenTelemetry tracer
  - [ ] Jaeger exporter configuration
  - [ ] Context propagation helpers
  - [ ] Span creation macros
- [ ] `libs/rust/common/src/metrics.rs` — Prometheus metrics
  - [ ] Metrics registry
  - [ ] Counter, histogram, gauge helpers
  - [ ] `/metrics` endpoint handler

### 2.1.8 Documentation & Testing
- [ ] `libs/rust/README.md` — usage guide for each crate
- [ ] Examples in `libs/rust/examples/` directory
- [ ] Run `cargo test --workspace` — all tests passing
- [ ] Run `cargo clippy --workspace -- -D warnings` — zero warnings
- [ ] Run `cargo fmt --check` — all code formatted

---

## 2.2 Notification Service (Go/Gin — Port 8082)

Multi-channel notification service with template engine, preference management, and delivery tracking.

### 2.2.1 Scaffold & Configuration
- [ ] Run `scripts/create-service.sh notification-service`
- [ ] Create `internal/config/config.go` — service config struct:
  - [ ] Server config (port, timeouts)
  - [ ] Database and Redis config
  - [ ] SMTP config (host, port, username, password, from address)
  - [ ] Twilio config (account SID, auth token, from number)
  - [ ] Webhook config (timeout, retry attempts, max redirects)
  - [ ] Template config (default language, supported languages)
- [ ] Create `config.yaml` — default dev configuration
- [ ] Add `replace` directive in `go.mod` for `libs/go`
- [ ] Verify `go build ./...` compiles

### 2.2.2 Database Migrations
All tables include standard fields. Tenant-scoped tables include `tenant_id`.

- [ ] `migrations/001_create_notification_templates_table.up.sql`
  - Columns: `id`, `tenant_id`, `name`, `channel` (enum: email/sms/in_app/webhook), `subject_template`, `body_template`, `template_data_schema` (JSONB), `language` (default: en), `is_active`, `created_at`, `updated_at`, `created_by`
  - Indexes: `(tenant_id, name, channel)` UNIQUE, `(tenant_id, is_active)`
  - Template uses Go template syntax with safe functions
- [ ] `migrations/001_create_notification_templates_table.down.sql`
- [ ] `migrations/002_create_notification_preferences_table.up.sql`
  - Columns: `id`, `user_id`, `tenant_id`, `channel` (enum), `event_type`, `enabled`, `updated_at`
  - Purpose: per-user notification opt-in/opt-out
  - Indexes: `(user_id, tenant_id, channel, event_type)` UNIQUE
- [ ] `migrations/002_create_notification_preferences_table.down.sql`
- [ ] `migrations/003_create_notifications_table.up.sql`
  - Columns: `id`, `tenant_id`, `user_id`, `channel`, `event_type`, `subject`, `body`, `status` (enum: pending/sent/failed/read), `metadata` (JSONB), `sent_at`, `read_at`, `created_at`
  - Purpose: in-app notification storage + delivery audit trail
  - Indexes: `(user_id, tenant_id, status)`, `(tenant_id, created_at)`
- [ ] `migrations/003_create_notifications_table.down.sql`
- [ ] `migrations/004_create_notification_delivery_log_table.up.sql`
  - Columns: `id`, `notification_id`, `tenant_id`, `channel`, `recipient`, `status` (enum: delivered/failed/bounced), `error_message`, `provider_response` (JSONB), `retry_count`, `delivered_at`, `created_at`
  - Purpose: track every delivery attempt with provider responses
  - Indexes: `(notification_id)`, `(tenant_id, status, created_at)`
- [ ] `migrations/004_create_notification_delivery_log_table.down.sql`
- [ ] `migrations/005_create_notification_dlq_table.up.sql`
  - Columns: `id`, `tenant_id`, `event_type`, `payload` (JSONB), `error`, `retry_count`, `created_at`, `retried_at`
  - Purpose: dead letter queue for failed notification processing
  - Index: `(tenant_id, created_at)`
- [ ] `migrations/005_create_notification_dlq_table.down.sql`
- [ ] Verify migrations run against local Postgres

### 2.2.3 Domain Models
- [ ] `internal/models/template.go`
  - [ ] `NotificationTemplate` struct
  - [ ] `CreateTemplateRequest`, `UpdateTemplateRequest` with validation tags
  - [ ] `TemplateResponse` (omit internal fields)
  - [ ] `Channel` enum type
- [ ] `internal/models/notification.go`
  - [ ] `Notification` struct
  - [ ] `SendNotificationRequest` with validation (channel, recipient, template_id, data)
  - [ ] `NotificationResponse`
  - [ ] `NotificationStatus` enum
- [ ] `internal/models/preference.go`
  - [ ] `NotificationPreference` struct
  - [ ] `UpdatePreferenceRequest`
  - [ ] `PreferenceResponse`
- [ ] `internal/models/delivery.go`
  - [ ] `DeliveryLog` struct
  - [ ] `DeliveryStatus` enum
  - [ ] `ProviderResponse` struct

### 2.2.4 Repository Layer
All methods use `tenant.RequireTenant(ctx)` or `tenant.NewScope(ctx)`.

- [ ] `internal/repository/postgres/template_repo.go`
  - [ ] `Create(ctx, template)` — tenant-scoped
  - [ ] `GetByID(ctx, id)` — tenant-scoped
  - [ ] `GetByName(ctx, name, channel)` — for lookup by name + channel
  - [ ] `List(ctx, filter)` — tenant-scoped, filterable by channel and status
  - [ ] `Update(ctx, id, fields)` — partial update
  - [ ] `Delete(ctx, id)` — soft delete (set is_active=false)
- [ ] `internal/repository/postgres/notification_repo.go`
  - [ ] `Create(ctx, notification)` — in-app notification storage
  - [ ] `GetByID(ctx, id)` — tenant-scoped
  - [ ] `List(ctx, userID, filter)` — user's in-app notifications with pagination
  - [ ] `MarkAsRead(ctx, id)` — set read_at timestamp
  - [ ] `CountUnread(ctx, userID)` — unread count for user
- [ ] `internal/repository/postgres/preference_repo.go`
  - [ ] `GetUserPreferences(ctx, userID)` — all preferences for user
  - [ ] `UpdatePreference(ctx, userID, eventType, channel, enabled)` — upsert
  - [ ] `CheckEnabled(ctx, userID, eventType, channel)` — opt-in check
- [ ] `internal/repository/postgres/delivery_repo.go`
  - [ ] `Create(ctx, log)` — record delivery attempt
  - [ ] `List(ctx, notificationID)` — delivery history for notification
  - [ ] `GetFailedDeliveries(ctx, threshold)` — for retry worker
- [ ] `internal/repository/redis/notification_cache.go`
  - [ ] Cache templates (1 hour TTL)
  - [ ] Cache user preferences (30 min TTL)
- [ ] Unit tests for each repository

### 2.2.5 Service Layer

#### Core Services
- [ ] `internal/service/template_service.go`
  - [ ] `CreateTemplate(ctx, req)` — validate template syntax, save, publish event
  - [ ] `GetTemplate(ctx, id)` — with caching
  - [ ] `ListTemplates(ctx, filter)` — tenant-scoped
  - [ ] `UpdateTemplate(ctx, id, req)` — invalidate cache
  - [ ] `DeleteTemplate(ctx, id)` — soft delete
  - [ ] `RenderTemplate(template, data)` — safe template rendering with sandbox
  - [ ] Validate template data schema against JSONB schema field

- [ ] `internal/service/notification_service.go`
  - [ ] `Send(ctx, req)` — orchestrate notification sending:
    1. Load template
    2. Check user preferences
    3. Render template with data
    4. Route to appropriate channel handler
    5. Create delivery log
    6. Return notification ID
  - [ ] `GetByID(ctx, id)` — retrieve notification
  - [ ] `List(ctx, userID, filter)` — user's in-app notifications
  - [ ] `MarkAsRead(ctx, id)` — update read status
  - [ ] `GetUnreadCount(ctx, userID)` — badge count
  - [ ] Validate webhook URLs to prevent SSRF (no private IPs, localhost, etc.)

- [ ] `internal/service/preference_service.go`
  - [ ] `GetPreferences(ctx, userID)` — with caching
  - [ ] `UpdatePreference(ctx, req)` — save, invalidate cache
  - [ ] `GetDefaultPreferences()` — system defaults for new users

#### Channel Implementations
- [ ] `internal/service/channels/email_channel.go`
  - [ ] `Send(recipient, subject, body, metadata)` — SMTP or SendGrid
  - [ ] HTML and plain text support
  - [ ] Attachment support
  - [ ] Retry logic with exponential backoff (3 attempts)
  - [ ] Track bounces and unsubscribes

- [ ] `internal/service/channels/sms_channel.go`
  - [ ] `Send(recipient, body, metadata)` — Twilio integration
  - [ ] Phone number validation (E.164 format)
  - [ ] Message truncation with warning
  - [ ] Delivery receipt handling
  - [ ] Retry logic

- [ ] `internal/service/channels/in_app_channel.go`
  - [ ] `Send(userID, subject, body, metadata)` — database storage
  - [ ] Real-time notification via Redis pub/sub (optional)
  - [ ] Auto-expire old notifications (90 days)

- [ ] `internal/service/channels/webhook_channel.go`
  - [ ] `Send(url, payload, metadata)` — HTTP POST
  - [ ] Signature generation (HMAC) for webhook verification
  - [ ] Timeout (10 seconds)
  - [ ] Retry with exponential backoff (5 attempts)
  - [ ] Follow redirects (max 3) with URL validation on each hop
  - [ ] SSRF prevention (block private IPs, localhost, metadata endpoints)

#### Event Processing
- [ ] `internal/service/event_consumer.go`
  - [ ] Subscribe to all service events (user.*, org.*, billing.*, etc.)
  - [ ] Map events to notification templates
  - [ ] Auto-send notifications based on event type
  - [ ] Support for event batching (digest notifications)
  - [ ] DLQ handling for failed processing

- [ ] `internal/service/retry_worker.go`
  - [ ] Background job to retry failed deliveries
  - [ ] Exponential backoff schedule
  - [ ] Move to DLQ after max retries
  - [ ] Configurable retry policy per channel

### 2.2.6 HTTP Handlers
- [ ] `internal/handlers/notification_handler.go`
  - [ ] `POST /api/v1/notifications/send` — manual notification sending
  - [ ] `GET /api/v1/notifications` — list in-app notifications (tenant-scoped, paginated)
  - [ ] `GET /api/v1/notifications/:id` — get notification by ID
  - [ ] `PUT /api/v1/notifications/:id/read` — mark as read
  - [ ] `GET /api/v1/notifications/unread/count` — unread count
- [ ] `internal/handlers/template_handler.go`
  - [ ] `POST /api/v1/notifications/templates` — create template (admin only)
  - [ ] `GET /api/v1/notifications/templates` — list templates
  - [ ] `GET /api/v1/notifications/templates/:id` — get template
  - [ ] `PUT /api/v1/notifications/templates/:id` — update template
  - [ ] `DELETE /api/v1/notifications/templates/:id` — deactivate template
- [ ] `internal/handlers/preference_handler.go`
  - [ ] `GET /api/v1/notifications/preferences` — get user preferences
  - [ ] `PUT /api/v1/notifications/preferences` — update preferences (bulk)
  - [ ] `PUT /api/v1/notifications/preferences/:channel/:event_type` — update single preference
- [ ] All handlers use standardized error responses

### 2.2.7 API Route Registration
- [ ] `api/routes.go` — register all routes with middleware:
  1. Recovery, RequestLogger, Tenant, RejectBodyIdentity
  2. Auth middleware (JWT validation)
  3. RBAC middleware (admin routes require admin role)
  4. Metrics
- [ ] Public routes: `/health`
- [ ] Protected routes: all API endpoints
- [ ] Admin routes: template management

### 2.2.8 Redis Streams Events
Publish events for other services:
- [ ] `notification.sent` — successful delivery
- [ ] `notification.failed` — delivery failure
- [ ] `notification.template_created` — new template
- [ ] `notification.preference_updated` — user preference change

### 2.2.9 Observability
- [ ] OpenTelemetry spans on all operations
- [ ] Prometheus metrics:
  - [ ] `notifications_sent_total` (channel, status)
  - [ ] `notifications_delivery_duration_seconds` (channel)
  - [ ] `notifications_retry_total` (channel)
  - [ ] `notifications_dlq_total`
- [ ] Health check with SMTP, Twilio, and database connectivity

### 2.2.10 Containerfile
- [ ] Multi-stage Go build
- [ ] Non-root user
- [ ] Pinned base image
- [ ] `HEALTHCHECK` instruction
- [ ] No secrets in environment variables (use mounted files)

### 2.2.11 Tests
- [ ] Unit tests:
  - [ ] Template rendering (valid, invalid, XSS attempts)
  - [ ] Template validation
  - [ ] SSRF prevention in webhook URLs
  - [ ] Channel implementations (mocked providers)
  - [ ] Preference checking logic
  - [ ] Retry logic
- [ ] Integration tests:
  - [ ] Full send flow (template → render → deliver)
  - [ ] Event consumer processing
  - [ ] DLQ handling
  - [ ] Delivery log creation
- [ ] Security tests:
  - [ ] Template injection attempts
  - [ ] SSRF in webhook URLs (private IPs, localhost, cloud metadata)
  - [ ] HTML/XSS in notification content
  - [ ] Phone number validation bypass attempts

### 2.2.12 Documentation
- [ ] `README.md` — architecture, setup, usage
- [ ] `libs/contracts/notification-service.yaml` — OpenAPI spec
- [ ] Template syntax documentation
- [ ] Channel configuration guide
- [ ] Event type to template mapping documentation

---

## 2.3 Billing Service (Rust/Axum — Port 8083)

Subscription management with Stripe integration, usage metering, invoicing, and proration.

### 2.3.1 Scaffold & Configuration
- [ ] Create `services/billing-service/` directory
- [ ] Initialize Cargo project: `cargo init --name billing-service`
- [ ] Add dependencies in `Cargo.toml`:
  - [ ] `axum`, `tokio`, `tower`, `tower-http`
  - [ ] `sqlx` with postgres feature
  - [ ] `redis`
  - [ ] `stripe-rust` (or `async-stripe`)
  - [ ] `serde`, `serde_json`
  - [ ] Workspace dependencies from `libs/rust`
- [ ] Add workspace member to `libs/rust/Cargo.toml`
- [ ] Create `config/config.toml` — service configuration:
  - [ ] Server config (host, port)
  - [ ] Database config
  - [ ] Redis config
  - [ ] Stripe config (secret key, webhook secret, publishable key)
  - [ ] Invoice config (due days, late fee percentage)
  - [ ] Feature flag for Stripe test mode
- [ ] Create `.env.example`
- [ ] Verify `cargo build` compiles

### 2.3.2 Database Migrations
Use `sqlx-cli` for migrations: `cargo install sqlx-cli`.

- [ ] `migrations/001_create_subscriptions_table.up.sql`
  - Columns: `id`, `tenant_id`, `org_id`, `plan_id`, `stripe_subscription_id`, `status` (enum: active/past_due/canceled/paused), `current_period_start`, `current_period_end`, `cancel_at`, `canceled_at`, `trial_end`, `created_at`, `updated_at`
  - Indexes: `(tenant_id)`, `(stripe_subscription_id)` UNIQUE
  - Only one active subscription per tenant
- [ ] `migrations/001_create_subscriptions_table.down.sql`
- [ ] `migrations/002_create_plans_table.up.sql`
  - Columns: `id`, `name`, `stripe_price_id`, `billing_interval` (enum: month/year), `price_cents`, `currency`, `features` (JSONB), `is_active`, `created_at`, `updated_at`
  - Purpose: mirror Stripe plans/prices locally
  - Index: `(stripe_price_id)` UNIQUE
  - Seed with starter, professional, enterprise plans
- [ ] `migrations/002_create_plans_table.down.sql`
- [ ] `migrations/003_create_invoices_table.up.sql`
  - Columns: `id`, `tenant_id`, `subscription_id`, `stripe_invoice_id`, `invoice_number`, `status` (enum: draft/open/paid/void/uncollectible), `amount_due_cents`, `amount_paid_cents`, `currency`, `due_date`, `paid_at`, `pdf_url`, `created_at`, `updated_at`
  - Indexes: `(tenant_id, created_at)`, `(stripe_invoice_id)` UNIQUE
- [ ] `migrations/003_create_invoices_table.down.sql`
- [ ] `migrations/004_create_payments_table.up.sql`
  - Columns: `id`, `tenant_id`, `invoice_id`, `stripe_payment_intent_id`, `amount_cents`, `currency`, `status` (enum: succeeded/pending/failed), `payment_method_type`, `receipt_url`, `failure_reason`, `paid_at`, `created_at`
  - Purpose: track all payment attempts
  - Indexes: `(tenant_id, created_at)`, `(stripe_payment_intent_id)` UNIQUE
  - NEVER store card numbers, CVV, or full PANs
- [ ] `migrations/004_create_payments_table.down.sql`
- [ ] `migrations/005_create_usage_records_table.up.sql`
  - Columns: `id`, `tenant_id`, `subscription_id`, `metric_name` (e.g., api_calls, storage_gb, users), `quantity`, `recorded_at`, `aggregated`, `created_at`
  - Purpose: metered billing data
  - Indexes: `(tenant_id, metric_name, recorded_at)`, `(subscription_id, aggregated)`
- [ ] `migrations/005_create_usage_records_table.down.sql`
- [ ] Run migrations: `sqlx migrate run`

### 2.3.3 Domain Models
- [ ] `src/models/subscription.rs`
  - [ ] `Subscription` struct
  - [ ] `CreateSubscriptionRequest` with validation
  - [ ] `UpdateSubscriptionRequest` (upgrade/downgrade)
  - [ ] `CancelSubscriptionRequest`
  - [ ] `SubscriptionResponse`
  - [ ] `SubscriptionStatus` enum
- [ ] `src/models/plan.rs`
  - [ ] `Plan` struct
  - [ ] `PlanResponse`
  - [ ] `BillingInterval` enum
  - [ ] `Features` struct (from JSONB)
- [ ] `src/models/invoice.rs`
  - [ ] `Invoice` struct
  - [ ] `InvoiceResponse`
  - [ ] `InvoiceStatus` enum
  - [ ] `InvoiceLineItem` struct
- [ ] `src/models/payment.rs`
  - [ ] `Payment` struct
  - [ ] `PaymentResponse` (without sensitive data)
  - [ ] `PaymentStatus` enum
- [ ] `src/models/usage.rs`
  - [ ] `UsageRecord` struct
  - [ ] `RecordUsageRequest`
  - [ ] `UsageResponse` with aggregation

### 2.3.4 Repository Layer
All queries use `libs/rust/database` helpers and tenant scoping.

- [ ] `src/repository/subscription_repo.rs`
  - [ ] `create(pool, subscription)` — tenant-scoped
  - [ ] `get_by_id(pool, id)` — tenant-scoped
  - [ ] `get_by_tenant(pool, tenant_id)` — current subscription
  - [ ] `update(pool, id, fields)` — partial update
  - [ ] `cancel(pool, id)` — set canceled_at
  - [ ] `get_expiring_trials(pool, days)` — for reminder notifications
- [ ] `src/repository/plan_repo.rs`
  - [ ] `get_by_id(pool, id)`
  - [ ] `get_by_stripe_price_id(pool, stripe_price_id)`
  - [ ] `list_active(pool)` — public plans
- [ ] `src/repository/invoice_repo.rs`
  - [ ] `create(pool, invoice)` — tenant-scoped
  - [ ] `get_by_id(pool, id)` — tenant-scoped
  - [ ] `list_by_tenant(pool, tenant_id, pagination)` — with filtering
  - [ ] `update_status(pool, id, status)`
- [ ] `src/repository/payment_repo.rs`
  - [ ] `create(pool, payment)` — tenant-scoped
  - [ ] `get_by_invoice(pool, invoice_id)`
  - [ ] `list_by_tenant(pool, tenant_id, pagination)`
- [ ] `src/repository/usage_repo.rs`
  - [ ] `record(pool, usage)` — tenant-scoped
  - [ ] `aggregate(pool, tenant_id, metric, start, end)` — sum quantities
  - [ ] `get_current_period(pool, subscription_id)` — usage this billing cycle
- [ ] `src/repository/cache/` — Redis caching for plans and subscriptions (5 min TTL)
- [ ] Unit tests for each repository (use `sqlx::test` with test database)

### 2.3.5 Service Layer

#### Core Services
- [ ] `src/service/subscription_service.rs`
  - [ ] `create_subscription(ctx, req)` — create in Stripe, save locally, publish event
    - [ ] Validate plan exists
    - [ ] Check tenant doesn't have active subscription (or cancel existing)
    - [ ] Create Stripe subscription with trial if new customer
    - [ ] Handle Stripe payment failures gracefully
  - [ ] `get_subscription(ctx, tenant_id)` — with plan details
  - [ ] `upgrade_subscription(ctx, req)` — change plan with proration
    - [ ] Calculate proration amount
    - [ ] Update Stripe subscription
    - [ ] Sync status locally
  - [ ] `downgrade_subscription(ctx, req)` — schedule change for end of period
  - [ ] `cancel_subscription(ctx, immediate)` — cancel now or at period end
  - [ ] `pause_subscription(ctx)` — pause billing (if supported by plan)
  - [ ] `resume_subscription(ctx)` — resume from pause

- [ ] `src/service/invoice_service.rs`
  - [ ] `list_invoices(ctx, tenant_id, filter)` — paginated
  - [ ] `get_invoice(ctx, id)` — with line items
  - [ ] `generate_pdf(ctx, invoice_id)` — call File Service API
  - [ ] `send_invoice(ctx, invoice_id)` — trigger email notification
  - [ ] `mark_as_paid(ctx, id)` — manual payment confirmation

- [ ] `src/service/usage_service.rs`
  - [ ] `record_usage(ctx, req)` — save usage record, publish event
  - [ ] `get_usage(ctx, tenant_id, metric, period)` — aggregated usage
  - [ ] `get_quota_status(ctx, tenant_id)` — current usage vs plan limits
  - [ ] `enforce_quota(ctx, tenant_id, metric)` — check if over limit
  - [ ] Background job to sync usage to Stripe metered billing

- [ ] `src/service/webhook_service.rs`
  - [ ] `handle_webhook(signature, payload)` — verify and process Stripe webhooks:
    - [ ] `invoice.payment_succeeded` → update invoice status, publish event
    - [ ] `invoice.payment_failed` → update status, trigger notification
    - [ ] `customer.subscription.updated` → sync subscription status
    - [ ] `customer.subscription.deleted` → mark as canceled
    - [ ] `payment_intent.succeeded` → record payment
    - [ ] `payment_intent.payment_failed` → log failure
  - [ ] Verify webhook signature (Stripe's HMAC)
  - [ ] Idempotency handling (process each event only once)

#### Stripe Integration
- [ ] `src/service/stripe_client.rs`
  - [ ] Wrapper around `stripe-rust` SDK
  - [ ] `create_customer(email, metadata)` — for new tenants
  - [ ] `create_subscription(customer_id, price_id, options)`
  - [ ] `update_subscription(subscription_id, price_id)`
  - [ ] `cancel_subscription(subscription_id, immediately)`
  - [ ] `retrieve_invoice(invoice_id)`
  - [ ] `create_usage_record(subscription_item_id, quantity)`
  - [ ] Error handling and retry logic
  - [ ] Logging (no sensitive data)

#### Proration Logic
- [ ] `src/service/proration.rs`
  - [ ] Calculate prorated amount for upgrades/downgrades
  - [ ] Credit calculation for unused time
  - [ ] Generate preview invoice before applying change
  - [ ] Handle different billing intervals (monthly vs annual)

### 2.3.6 HTTP Handlers
- [ ] `src/handlers/subscription_handler.rs`
  - [ ] `GET /api/v1/billing/subscription` — get current subscription
  - [ ] `POST /api/v1/billing/subscription` — create subscription
  - [ ] `PUT /api/v1/billing/subscription` — upgrade/downgrade
  - [ ] `DELETE /api/v1/billing/subscription` — cancel subscription
  - [ ] `POST /api/v1/billing/subscription/pause` — pause subscription
  - [ ] `POST /api/v1/billing/subscription/resume` — resume subscription
- [ ] `src/handlers/invoice_handler.rs`
  - [ ] `GET /api/v1/billing/invoices` — list invoices (tenant-scoped)
  - [ ] `GET /api/v1/billing/invoices/:id` — get invoice details
  - [ ] `GET /api/v1/billing/invoices/:id/pdf` — download PDF
  - [ ] `POST /api/v1/billing/invoices/:id/pay` — manual payment
- [ ] `src/handlers/usage_handler.rs`
  - [ ] `GET /api/v1/billing/usage` — get current usage
  - [ ] `GET /api/v1/billing/usage/quota` — quota status
  - [ ] `POST /api/v1/billing/usage/record` — record usage (internal API)
- [ ] `src/handlers/webhook_handler.rs`
  - [ ] `POST /api/v1/billing/webhook` — Stripe webhook endpoint (public)
  - [ ] Signature verification
  - [ ] Async processing (queue events if needed)
- [ ] `src/handlers/plan_handler.rs`
  - [ ] `GET /api/v1/billing/plans` — list available plans

### 2.3.7 API Route Registration
- [ ] `src/routes.rs` — configure Axum router
  - [ ] Apply middleware stack from `libs/rust/middleware`
  - [ ] Auth middleware on all routes except webhook
  - [ ] Tenant middleware
  - [ ] Logging and metrics
- [ ] Public routes: `/health`, `/api/v1/billing/webhook`
- [ ] Protected routes: all other endpoints
- [ ] Rate limiting on webhook endpoint (prevent abuse)

### 2.3.8 Redis Streams Events
- [ ] `subscription.created` — new subscription
- [ ] `subscription.updated` — plan change, status change
- [ ] `subscription.canceled` — subscription canceled
- [ ] `invoice.paid` — payment succeeded
- [ ] `invoice.failed` — payment failed
- [ ] `usage.recorded` — usage data point
- [ ] `usage.quota_exceeded` — over plan limits

### 2.3.9 Observability
- [ ] OpenTelemetry tracing on all operations
- [ ] Prometheus metrics:
  - [ ] `billing_subscriptions_total` (status)
  - [ ] `billing_revenue_cents` (period)
  - [ ] `billing_invoices_total` (status)
  - [ ] `billing_payments_total` (status)
  - [ ] `billing_usage_total` (metric_name)
  - [ ] `billing_webhook_events_total` (event_type, status)
  - [ ] `billing_stripe_api_duration_seconds` (operation)
- [ ] Health check with Stripe API connectivity and database

### 2.3.10 Containerfile
- [ ] Multi-stage Rust build (builder → runtime)
- [ ] Use `rust:1.75-slim` for builder
- [ ] Use `debian:bookworm-slim` for runtime
- [ ] Non-root user
- [ ] `HEALTHCHECK` instruction
- [ ] Copy only necessary binaries
- [ ] No secrets in image

### 2.3.11 Tests
- [ ] Unit tests:
  - [ ] Proration calculations
  - [ ] Quota enforcement logic
  - [ ] Webhook signature verification
  - [ ] Plan upgrade/downgrade scenarios
  - [ ] Payment status transitions
- [ ] Integration tests:
  - [ ] Full subscription lifecycle (create → upgrade → cancel)
  - [ ] Invoice generation flow
  - [ ] Usage recording and aggregation
  - [ ] Webhook processing (mock Stripe)
- [ ] Security tests:
  - [ ] PCI DSS compliance checks (no card data storage)
  - [ ] Webhook signature tampering attempts
  - [ ] Tenant isolation (can't access other tenant's billing)
  - [ ] SQL injection in usage records
  - [ ] Amount manipulation attempts

### 2.3.12 Documentation
- [ ] `README.md` — architecture, Stripe setup, testing with test mode
- [ ] `libs/contracts/billing-service.yaml` — OpenAPI spec
- [ ] Stripe webhook configuration guide
- [ ] Plan configuration documentation
- [ ] PCI DSS compliance notes
- [ ] Usage metering guide for other services

---

## 2.4 File Service (Rust/Axum — Port 8084)

S3-compatible file storage with MinIO, quota enforcement, virus scanning, and signed URL generation.

### 2.4.1 Scaffold & Configuration
- [ ] Create `services/file-service/` directory
- [ ] Initialize Cargo project: `cargo init --name file-service`
- [ ] Add dependencies:
  - [ ] `axum`, `tokio`, `tower`, `tower-http`
  - [ ] `sqlx` with postgres feature
  - [ ] `redis`
  - [ ] `aws-sdk-s3` (MinIO is S3-compatible)
  - [ ] `image` (for thumbnail generation)
  - [ ] `mime_guess`
  - [ ] `sha2` (for file hashing)
  - [ ] Workspace dependencies from `libs/rust`
- [ ] Create `config/config.toml`:
  - [ ] Server config
  - [ ] Database and Redis config
  - [ ] S3 config (endpoint, region, bucket, access key, secret key)
  - [ ] Quota config (default per-tenant limit, per-file size limit)
  - [ ] Thumbnail config (max dimensions, quality)
  - [ ] Allowed file types (MIME types whitelist)
  - [ ] Virus scanning config (ClamAV endpoint, enabled flag)
- [ ] Verify `cargo build` compiles

### 2.4.2 Database Migrations
- [ ] `migrations/001_create_files_table.up.sql`
  - Columns: `id`, `tenant_id`, `user_id`, `org_id`, `filename`, `content_type`, `size_bytes`, `s3_key`, `s3_bucket`, `checksum_sha256`, `status` (enum: uploading/available/deleted/quarantined), `virus_scan_status` (enum: pending/clean/infected/error), `thumbnail_s3_key`, `uploaded_at`, `deleted_at`, `created_at`, `updated_at`
  - Indexes: `(tenant_id, status)`, `(s3_key)` UNIQUE, `(tenant_id, user_id)`
  - Foreign key: `user_id` references Auth Service (logical, not enforced)
- [ ] `migrations/001_create_files_table.down.sql`
- [ ] `migrations/002_create_storage_quotas_table.up.sql`
  - Columns: `id`, `tenant_id`, `quota_bytes`, `used_bytes`, `updated_at`
  - Purpose: track per-tenant storage usage
  - Index: `(tenant_id)` UNIQUE
  - Trigger to update `used_bytes` on file insert/delete
- [ ] `migrations/002_create_storage_quotas_table.down.sql`
- [ ] `migrations/003_create_file_access_log_table.up.sql`
  - Columns: `id`, `file_id`, `tenant_id`, `user_id`, `action` (enum: upload/download/delete/view), `ip_address`, `user_agent`, `created_at`
  - Purpose: audit trail for compliance
  - Indexes: `(file_id)`, `(tenant_id, created_at)`
- [ ] `migrations/003_create_file_access_log_table.down.sql`
- [ ] Run migrations

### 2.4.3 Domain Models
- [ ] `src/models/file.rs`
  - [ ] `File` struct
  - [ ] `UploadRequest` (multipart form data handling)
  - [ ] `FileResponse`
  - [ ] `FileStatus` enum
  - [ ] `VirusScanStatus` enum
- [ ] `src/models/quota.rs`
  - [ ] `StorageQuota` struct
  - [ ] `QuotaResponse`
  - [ ] `QuotaExceededError`
- [ ] `src/models/access_log.rs`
  - [ ] `FileAccessLog` struct
  - [ ] `AccessAction` enum

### 2.4.4 Repository Layer
- [ ] `src/repository/file_repo.rs`
  - [ ] `create(pool, file)` — tenant-scoped
  - [ ] `get_by_id(pool, id)` — tenant-scoped
  - [ ] `list(pool, tenant_id, filter, pagination)` — with status filtering
  - [ ] `update_status(pool, id, status)`
  - [ ] `update_scan_status(pool, id, scan_status)`
  - [ ] `delete(pool, id)` — soft delete (set status=deleted)
  - [ ] `get_by_s3_key(pool, s3_key)`
- [ ] `src/repository/quota_repo.rs`
  - [ ] `get_quota(pool, tenant_id)` — with upsert if missing
  - [ ] `increment_usage(pool, tenant_id, bytes)` — atomic update
  - [ ] `decrement_usage(pool, tenant_id, bytes)` — atomic update
  - [ ] `check_quota(pool, tenant_id, additional_bytes)` — would exceed?
- [ ] `src/repository/access_log_repo.rs`
  - [ ] `create(pool, log)` — tenant-scoped
  - [ ] `list(pool, file_id, pagination)` — audit trail
- [ ] `src/repository/cache/` — Redis caching for file metadata (10 min TTL)
- [ ] Unit tests for repositories

### 2.4.5 Service Layer

#### Core Services
- [ ] `src/service/file_service.rs`
  - [ ] `upload(ctx, stream, metadata)` — full upload flow:
    1. Validate file type (magic bytes, not just extension)
    2. Check quota
    3. Generate S3 key (tenant_id/uuid/filename)
    4. Stream upload to MinIO
    5. Calculate SHA-256 checksum
    6. Save metadata to database
    7. Increment quota usage
    8. Queue virus scan (async)
    9. Generate thumbnail (if image)
    10. Publish `file.uploaded` event
  - [ ] `get_file(ctx, id)` — with download URL generation
  - [ ] `list_files(ctx, filter)` — tenant-scoped, paginated
  - [ ] `delete_file(ctx, id)` — soft delete + decrement quota
  - [ ] `get_download_url(ctx, id, expiry)` — signed URL (default 1 hour)
  - [ ] `get_upload_url(ctx, filename, content_type)` — presigned upload URL
  - [ ] Validate no path traversal in filenames

- [ ] `src/service/quota_service.rs`
  - [ ] `get_quota_status(ctx, tenant_id)` — current usage and limit
  - [ ] `check_quota(ctx, tenant_id, size)` — enforce before upload
  - [ ] `update_quota_limit(ctx, tenant_id, new_limit)` — admin only
  - [ ] Background job to recalculate quotas (daily reconciliation)

- [ ] `src/service/thumbnail_service.rs`
  - [ ] `generate(file_id, s3_key)` — for images only
  - [ ] Download from S3
  - [ ] Resize to 200x200 (maintain aspect ratio)
  - [ ] Save back to S3 with `-thumb` suffix
  - [ ] Update file record with thumbnail S3 key
  - [ ] Supported formats: JPEG, PNG, GIF, WebP
  - [ ] Error handling (skip if not image or generation fails)

- [ ] `src/service/virus_scan_service.rs`
  - [ ] `scan(file_id, s3_key)` — async scan
  - [ ] Download file from S3 (stream)
  - [ ] Send to ClamAV via TCP/Unix socket
  - [ ] Parse scan result
  - [ ] Update virus_scan_status
  - [ ] If infected: set status=quarantined, publish alert event
  - [ ] If clean: set status=available
  - [ ] Queue scan on upload, process in background worker

#### S3 Client
- [ ] `src/service/s3_client.rs`
  - [ ] Wrapper around `aws-sdk-s3`
  - [ ] Configure for MinIO (custom endpoint)
  - [ ] Per-org bucket isolation (bucket name: `{org_id}-files`)
  - [ ] Create bucket if not exists (on service startup)
  - [ ] Bucket versioning enabled
  - [ ] Lifecycle policy: delete files with status=deleted after 30 days
  - [ ] `upload(bucket, key, stream, content_type)` — multipart for large files
  - [ ] `download(bucket, key)` — stream
  - [ ] `delete(bucket, key)`
  - [ ] `generate_presigned_get_url(bucket, key, expiry)` — signed download
  - [ ] `generate_presigned_put_url(bucket, key, expiry)` — signed upload
  - [ ] Error handling and retry logic

### 2.4.6 HTTP Handlers
- [ ] `src/handlers/file_handler.rs`
  - [ ] `POST /api/v1/files/upload` — multipart form upload
    - [ ] Accept `file` field (binary)
    - [ ] Optional metadata fields (tags, description)
    - [ ] Return file ID and metadata
  - [ ] `GET /api/v1/files/:id` — get file metadata
  - [ ] `GET /api/v1/files/:id/download` — redirect to signed S3 URL
  - [ ] `GET /api/v1/files/:id/thumbnail` — redirect to thumbnail URL
  - [ ] `DELETE /api/v1/files/:id` — soft delete
  - [ ] `GET /api/v1/files` — list files (tenant-scoped, paginated)
  - [ ] `GET /api/v1/files/:id/access-log` — audit trail
- [ ] `src/handlers/quota_handler.rs`
  - [ ] `GET /api/v1/files/quota` — current quota status
- [ ] `src/handlers/presigned_handler.rs`
  - [ ] `POST /api/v1/files/presigned-upload-url` — generate upload URL
  - [ ] Return URL + required headers for client-side upload

### 2.4.7 API Route Registration
- [ ] `src/routes.rs` — Axum router with middleware
  - [ ] Auth, tenant, logging, metrics middleware
  - [ ] File upload size limit (e.g., 100MB per request)
  - [ ] Multipart form data handling
- [ ] Public routes: `/health`
- [ ] Protected routes: all file operations

### 2.4.8 Redis Streams Events
- [ ] `file.uploaded` — new file uploaded
- [ ] `file.deleted` — file deleted
- [ ] `file.scan_completed` — virus scan result
- [ ] `file.quarantined` — infected file detected
- [ ] `quota.exceeded` — tenant over quota

### 2.4.9 Observability
- [ ] OpenTelemetry tracing
- [ ] Prometheus metrics:
  - [ ] `files_uploads_total` (status)
  - [ ] `files_downloads_total`
  - [ ] `files_storage_bytes` (tenant_id)
  - [ ] `files_scan_duration_seconds`
  - [ ] `files_upload_duration_seconds`
  - [ ] `files_virus_detections_total`
- [ ] Health check with MinIO and database connectivity

### 2.4.10 Containerfile
- [ ] Multi-stage Rust build
- [ ] Non-root user
- [ ] `HEALTHCHECK` instruction
- [ ] No secrets in image

### 2.4.11 Tests
- [ ] Unit tests:
  - [ ] File type validation (magic bytes)
  - [ ] Quota enforcement
  - [ ] Path traversal prevention
  - [ ] S3 key generation
  - [ ] Thumbnail generation
- [ ] Integration tests:
  - [ ] Full upload → scan → download flow
  - [ ] Quota exceeded scenario
  - [ ] Presigned URL generation and usage
  - [ ] Virus detection (mock ClamAV)
- [ ] Security tests:
  - [ ] Upload malicious file types (executables, scripts)
  - [ ] Filename path traversal (../../etc/passwd)
  - [ ] XXE attacks in XML files
  - [ ] Zip bomb detection
  - [ ] SSRF via file URLs
  - [ ] Tenant isolation (can't access other tenant's files)
  - [ ] Large file DoS (quota enforcement)

### 2.4.12 Documentation
- [ ] `README.md` — architecture, MinIO setup, virus scanning setup
- [ ] `libs/contracts/file-service.yaml` — OpenAPI spec
- [ ] Client-side upload guide (using presigned URLs)
- [ ] Supported file types documentation
- [ ] Quota management guide

---

## 2.5 Audit Service (Go/Gin — Port 8085)

Immutable audit log with event capture, hash chaining, retention policies, and compliance exports.

### 2.5.1 Scaffold & Configuration
- [ ] Run `scripts/create-service.sh audit-service`
- [ ] Create `internal/config/config.go`:
  - [ ] Server config
  - [ ] Database and Redis config
  - [ ] Retention config (default days, per-event-type overrides)
  - [ ] Export config (max export size, allowed formats)
  - [ ] Compliance mode flag (enables hash chaining)
- [ ] Create `config.yaml`
- [ ] Add `replace` directive for `libs/go`
- [ ] Verify `go build ./...` compiles

### 2.5.2 Database Migrations
- [ ] `migrations/001_create_audit_logs_table.up.sql`
  - Columns: `id`, `tenant_id`, `event_type`, `event_category` (enum: auth/data/system/security), `actor_id`, `actor_type` (enum: user/system/api_key), `resource_type`, `resource_id`, `action`, `outcome` (enum: success/failure), `ip_address`, `user_agent`, `metadata` (JSONB), `timestamp`, `previous_hash`, `current_hash`
  - Purpose: append-only immutable audit trail
  - Indexes: `(tenant_id, timestamp DESC)`, `(tenant_id, event_type)`, `(tenant_id, resource_id)`, `(actor_id)`, `(current_hash)` for verification
  - NO UPDATE or DELETE allowed (append-only)
  - Partition by month for performance
- [ ] `migrations/001_create_audit_logs_table.down.sql`
- [ ] `migrations/002_create_retention_policies_table.up.sql`
  - Columns: `id`, `tenant_id`, `event_type`, `retention_days`, `is_active`, `created_at`, `updated_at`
  - Purpose: configure how long to keep audit logs per event type
  - Default retention: 90 days (compliance), 365 days (security events)
  - Index: `(tenant_id, event_type)` UNIQUE
- [ ] `migrations/002_create_retention_policies_table.down.sql`
- [ ] `migrations/003_create_audit_exports_table.up.sql`
  - Columns: `id`, `tenant_id`, `requested_by`, `start_date`, `end_date`, `format` (enum: csv/json), `status` (enum: pending/processing/completed/failed), `file_id`, `error_message`, `created_at`, `completed_at`
  - Purpose: track export jobs
  - Index: `(tenant_id, created_at)`
- [ ] `migrations/003_create_audit_exports_table.down.sql`
- [ ] Verify migrations run

### 2.5.3 Domain Models
- [ ] `internal/models/audit_log.go`
  - [ ] `AuditLog` struct (matches DB schema)
  - [ ] `SearchRequest` with filters (event_type, actor, resource, date range)
  - [ ] `AuditLogResponse`
  - [ ] `EventCategory` enum
  - [ ] `ActorType` enum
  - [ ] `Outcome` enum
- [ ] `internal/models/retention_policy.go`
  - [ ] `RetentionPolicy` struct
  - [ ] `CreatePolicyRequest`, `UpdatePolicyRequest`
  - [ ] `PolicyResponse`
- [ ] `internal/models/export.go`
  - [ ] `AuditExport` struct
  - [ ] `CreateExportRequest`
  - [ ] `ExportResponse`
  - [ ] `ExportStatus` enum

### 2.5.4 Repository Layer
All queries tenant-scoped. NO UPDATE or DELETE on audit_logs table.

- [ ] `internal/repository/postgres/audit_repo.go`
  - [ ] `Create(ctx, log)` — append-only insert with hash chaining
  - [ ] `GetByID(ctx, id)` — tenant-scoped
  - [ ] `Search(ctx, filter)` — advanced search with pagination:
    - [ ] Filter by event_type, actor, resource, date range, outcome
    - [ ] Full-text search on metadata JSONB
    - [ ] Sort by timestamp DESC
  - [ ] `Count(ctx, filter)` — for pagination
  - [ ] `GetLastHash(ctx, tenant_id)` — for hash chaining
  - [ ] `VerifyChain(ctx, tenant_id, start, end)` — verify integrity
  - [ ] NO update or delete methods
- [ ] `internal/repository/postgres/retention_repo.go`
  - [ ] `Create(ctx, policy)` — tenant-scoped
  - [ ] `GetByEventType(ctx, eventType)` — tenant-scoped
  - [ ] `List(ctx)` — tenant-scoped
  - [ ] `Update(ctx, id, fields)`
  - [ ] `Delete(ctx, id)`
- [ ] `internal/repository/postgres/export_repo.go`
  - [ ] `Create(ctx, export)` — tenant-scoped
  - [ ] `GetByID(ctx, id)` — tenant-scoped
  - [ ] `List(ctx)` — tenant-scoped with pagination
  - [ ] `UpdateStatus(ctx, id, status, fileID, error)`
- [ ] No Redis caching (audit logs must be authoritative source)
- [ ] Unit tests for repositories (especially hash chaining)

### 2.5.5 Service Layer

#### Core Services
- [ ] `internal/service/audit_service.go`
  - [ ] `CreateLog(ctx, log)` — append audit entry with hash chaining:
    1. Load last hash for tenant
    2. Generate current hash (SHA-256 of: previous_hash + log data)
    3. Insert with previous_hash and current_hash
    4. Return log ID
  - [ ] `Search(ctx, filter)` — with pagination
  - [ ] `GetByID(ctx, id)` — single log entry
  - [ ] `VerifyIntegrity(ctx, tenantID, start, end)` — verify hash chain
    - [ ] Recalculate hashes for date range
    - [ ] Compare with stored hashes
    - [ ] Return verification report (valid/invalid/broken chain)
  - [ ] `GetStatistics(ctx, tenantID)` — event counts by type, category, outcome

- [ ] `internal/service/retention_service.go`
  - [ ] `GetPolicies(ctx)` — tenant-scoped
  - [ ] `CreatePolicy(ctx, req)` — validate retention days (min 30, max 2555)
  - [ ] `UpdatePolicy(ctx, id, req)`
  - [ ] `DeletePolicy(ctx, id)`
  - [ ] Background job: `CleanupExpiredLogs()` — run daily:
    - [ ] For each tenant, find logs older than retention period
    - [ ] Archive to cold storage (optional)
    - [ ] Delete from primary database
    - [ ] Publish `audit.logs_archived` event

- [ ] `internal/service/export_service.go`
  - [ ] `CreateExport(ctx, req)` — queue export job:
    1. Validate date range (max 1 year)
    2. Create export record with status=pending
    3. Queue background job
    4. Return export ID
  - [ ] `GetExport(ctx, id)` — check export status
  - [ ] `ListExports(ctx)` — tenant-scoped
  - [ ] Background worker: `ProcessExport(exportID)`:
    1. Update status=processing
    2. Fetch audit logs for date range
    3. Format as CSV or JSON
    4. Upload to File Service
    5. Update status=completed, store file_id
    6. Publish `audit.export_completed` event
  - [ ] Handle export size limits (split into multiple files if needed)

- [ ] `internal/service/event_consumer.go`
  - [ ] Subscribe to ALL Redis Streams events
  - [ ] Map each event to audit log entry
  - [ ] Auto-capture events from: Auth, Org, Billing, File, Notification services
  - [ ] Enrich with actor info, IP, user agent from event metadata
  - [ ] Call `audit_service.CreateLog()` for each event
  - [ ] DLQ for failed audit log writes (critical)

#### Hash Chain Implementation
- [ ] `internal/service/hash_chain.go`
  - [ ] `GenerateHash(prevHash, logData)` — SHA-256
  - [ ] `VerifyHash(log)` — recalculate and compare
  - [ ] `VerifyChain(logs)` — verify entire sequence
  - [ ] Use crypto/sha256 from Go stdlib
  - [ ] Log data format for hashing: `{tenant_id}|{timestamp}|{event_type}|{actor_id}|{resource_id}|{action}|{outcome}`

### 2.5.6 HTTP Handlers
- [ ] `internal/handlers/audit_handler.go`
  - [ ] `GET /api/v1/audit/logs` — search/filter audit logs (tenant-scoped, paginated)
    - [ ] Query params: event_type, actor, resource, start_date, end_date, outcome
  - [ ] `GET /api/v1/audit/logs/:id` — get single log entry
  - [ ] `GET /api/v1/audit/stats` — event statistics
  - [ ] `POST /api/v1/audit/verify` — verify hash chain integrity
    - [ ] Request body: start_date, end_date
    - [ ] Return verification report
- [ ] `internal/handlers/export_handler.go`
  - [ ] `POST /api/v1/audit/logs/export` — create export job
    - [ ] Request body: start_date, end_date, format
  - [ ] `GET /api/v1/audit/logs/export/:id` — get export status
  - [ ] `GET /api/v1/audit/logs/export/:id/download` — download via File Service
  - [ ] `GET /api/v1/audit/logs/exports` — list all exports
- [ ] `internal/handlers/retention_handler.go` (admin only)
  - [ ] `GET /api/v1/audit/retention-policies` — list policies
  - [ ] `POST /api/v1/audit/retention-policies` — create policy
  - [ ] `PUT /api/v1/audit/retention-policies/:id` — update policy
  - [ ] `DELETE /api/v1/audit/retention-policies/:id` — delete policy
- [ ] NO CREATE endpoint for audit logs (only via event consumer)
- [ ] NO UPDATE or DELETE endpoints for audit logs (immutable)

### 2.5.7 API Route Registration
- [ ] `api/routes.go` — register routes with middleware
  - [ ] Recovery, logging, tenant, metrics
  - [ ] Auth middleware (all routes protected)
  - [ ] RBAC (retention policies admin-only)
- [ ] Public routes: `/health`
- [ ] Protected routes: all audit endpoints
- [ ] Admin routes: retention policy management

### 2.5.8 Redis Streams Events
Subscribe to events from ALL services:
- [ ] `user.*`, `org.*`, `billing.*`, `file.*`, `notification.*`
- [ ] Auto-capture and log

Publish own events:
- [ ] `audit.export_completed` — export ready for download
- [ ] `audit.logs_archived` — logs moved to cold storage
- [ ] `audit.chain_broken` — integrity violation detected

### 2.5.9 Observability
- [ ] OpenTelemetry spans
- [ ] Prometheus metrics:
  - [ ] `audit_logs_total` (event_type, outcome)
  - [ ] `audit_events_consumed_total` (source_service)
  - [ ] `audit_export_duration_seconds`
  - [ ] `audit_chain_verification_duration_seconds`
  - [ ] `audit_retention_cleanup_total`
- [ ] Health check with database connectivity

### 2.5.10 Containerfile
- [ ] Multi-stage Go build
- [ ] Non-root user
- [ ] Pinned base image
- [ ] `HEALTHCHECK` instruction

### 2.5.11 Tests
- [ ] Unit tests:
  - [ ] Hash chain generation and verification
  - [ ] Search filtering logic
  - [ ] Retention policy enforcement
  - [ ] Export formatting (CSV, JSON)
- [ ] Integration tests:
  - [ ] Full event capture flow
  - [ ] Hash chain integrity over multiple inserts
  - [ ] Export job processing
  - [ ] Retention cleanup
- [ ] Security tests:
  - [ ] Attempt to update audit log (should fail)
  - [ ] Attempt to delete audit log (should fail)
  - [ ] Hash tampering detection
  - [ ] Tenant isolation (can't read other tenant's logs)
  - [ ] SQL injection in search filters

### 2.5.12 Documentation
- [ ] `README.md` — architecture, compliance features, hash chaining
- [ ] `libs/contracts/audit-service.yaml` — OpenAPI spec
- [ ] Compliance guide (SOC 2, GDPR, HIPAA considerations)
- [ ] Retention policy configuration guide
- [ ] Hash chain verification guide

---

## 2.6 Phase 2 Integration

Bring all Phase 2 services together with Phase 1 core platform.

### 2.6.1 Compose Stack
- [ ] Update `deploy/podman/compose.services.yml`:
  - [ ] Add `notification-service` container (port 8082)
  - [ ] Add `billing-service` container (port 8083)
  - [ ] Add `file-service` container (port 8084)
  - [ ] Add `audit-service` container (port 8085)
  - [ ] Configure service dependencies (wait for DB, Redis)
  - [ ] Apply security hardening (cap_drop, resource limits)
  - [ ] Add healthchecks
- [ ] Update `.env` with new service environment variables:
  - [ ] Notification: SMTP, Twilio config
  - [ ] Billing: Stripe keys
  - [ ] File: MinIO endpoint, ClamAV endpoint
  - [ ] Audit: retention defaults
- [ ] Create `deploy/podman/compose.full.yml` — includes base + core + services

### 2.6.2 API Gateway Integration
- [ ] Update API Gateway to proxy Phase 2 services:
  - [ ] `/api/v1/notifications/*` → notification-service:8082
  - [ ] `/api/v1/billing/*` → billing-service:8083
  - [ ] `/api/v1/files/*` → file-service:8084
  - [ ] `/api/v1/audit/*` → audit-service:8085
- [ ] Add service health checks to Gateway aggregated health endpoint
- [ ] Module gating for Phase 2 services:
  - [ ] `notifications` module
  - [ ] `billing` module
  - [ ] `files` module
  - [ ] `audit` module (always enabled for compliance)
- [ ] Update rate limits (per-service tiers)

### 2.6.3 Makefile Updates
- [ ] Add targets:
  - [ ] `services-up-phase2` — start all Phase 2 services
  - [ ] `services-down-phase2`
  - [ ] `full-up` — start infra + all 5 services + gateway
  - [ ] `full-down`
  - [ ] `full-logs` / `full-status`
  - [ ] `full-reset`
- [ ] Update `make test` to include Phase 2 services

### 2.6.4 Cross-Service Integration Tests
- [ ] Notification integration:
  - [ ] Auth service login → notification sent
  - [ ] Org created → welcome email
  - [ ] Billing invoice paid → receipt email
- [ ] Billing integration:
  - [ ] Create org → create subscription
  - [ ] Module gating based on subscription plan
  - [ ] Usage tracking from File Service
- [ ] File integration:
  - [ ] Upload file → audit log created
  - [ ] Quota exceeded → billing event
  - [ ] Invoice PDF generation via File Service
- [ ] Audit integration:
  - [ ] All service events captured
  - [ ] Hash chain maintained across all services
  - [ ] Export includes events from all services

### 2.6.5 End-to-End Workflows
Test complete multi-service workflows:
- [ ] User registration → welcome email → audit log
- [ ] Org creation → subscription creation → payment → invoice → receipt
- [ ] File upload → virus scan → notification → audit log
- [ ] Subscription upgrade → proration → invoice → payment → notification
- [ ] Module toggle → audit log → gating enforcement

### 2.6.6 Service Discovery & Health
- [ ] All services register health status in Redis
- [ ] Gateway polls service health every 30 seconds
- [ ] Circuit breaker opens if service unhealthy
- [ ] Automated service recovery (restart unhealthy containers)

---

## 2.7 Phase 2 Security

Comprehensive security testing across all Phase 2 services and the full platform.

### 2.7.1 SAST (Static Application Security Testing)
- [ ] `gosec` on Notification and Audit services — zero HIGH
- [ ] `cargo audit` on Billing and File services — zero HIGH/CRITICAL
- [ ] `cargo clippy` with security lints — zero warnings
- [ ] `semgrep` with custom rules on all services
- [ ] `hadolint` on all Phase 2 Containerfiles

### 2.7.2 DAST (Dynamic Application Security Testing)
- [ ] OWASP ZAP baseline scan on each Phase 2 service
- [ ] OWASP ZAP active scan on Notification Service (webhook SSRF)
- [ ] OWASP ZAP active scan on File Service (upload attacks)
- [ ] Trivy scan all Phase 2 containers — zero CRITICAL CVEs

### 2.7.3 Service-Specific Security Tests

#### Notification Service
- [ ] Template injection tests (XSS, code execution)
- [ ] SSRF tests on webhook URLs:
  - [ ] Private IP ranges (10.0.0.0/8, 172.16.0.0/12, 192.168.0.0/16)
  - [ ] Localhost (127.0.0.1, ::1)
  - [ ] Cloud metadata endpoints (169.254.169.254)
  - [ ] DNS rebinding attacks
- [ ] Email header injection tests
- [ ] SMS injection tests (premium number abuse)
- [ ] Rate limiting (prevent notification spam)

#### Billing Service
- [ ] PCI DSS compliance checklist:
  - [ ] No card data stored locally
  - [ ] No card data in logs
  - [ ] All Stripe communication over HTTPS
  - [ ] Webhook signature verification
- [ ] Amount manipulation tests (tampered prices, negative amounts)
- [ ] Proration calculation tests (edge cases)
- [ ] Stripe webhook replay attacks
- [ ] Subscription status manipulation

#### File Service
- [ ] File upload attacks:
  - [ ] Path traversal (../../etc/passwd)
  - [ ] Malicious file types (executables, scripts)
  - [ ] Zip bombs (compression bombs)
  - [ ] XXE in XML files
  - [ ] SVG with embedded scripts
  - [ ] EICAR test file (virus scanner validation)
- [ ] Magic bytes validation (file type spoofing)
- [ ] Filename injection tests
- [ ] Large file DoS (quota enforcement)
- [ ] Signed URL tampering
- [ ] Thumbnail generation exploits (ImageTragick)

#### Audit Service
- [ ] Immutability tests (attempt UPDATE/DELETE on audit_logs)
- [ ] Hash chain tampering detection
- [ ] SQL injection in search filters
- [ ] Export size limits (prevent memory exhaustion)
- [ ] Time-based attacks (backdating logs)

### 2.7.4 Tenant Isolation Tests
Extend tenant isolation test suite to Phase 2 services:
- [ ] `cross_tenant_notifications.go` — can't send notification to other tenant's users
- [ ] `cross_tenant_billing.go` — can't view/modify other tenant's subscriptions
- [ ] `cross_tenant_files.go` — can't access other tenant's files
- [ ] `cross_tenant_audit.go` — can't read other tenant's audit logs
- [ ] Run via `make test-tenant-isolation`

### 2.7.5 OWASP Top 10 Testing
- [ ] A01: Broken Access Control — tenant isolation, RBAC
- [ ] A02: Cryptographic Failures — TLS, encrypted storage, hash chaining
- [ ] A03: Injection — SQL injection, template injection, command injection
- [ ] A04: Insecure Design — threat model review
- [ ] A05: Security Misconfiguration — default credentials, debug mode off
- [ ] A06: Vulnerable Components — dependency scanning (Trivy, cargo-audit)
- [ ] A07: Authentication Failures — tested in Phase 1
- [ ] A08: Software Integrity Failures — hash chaining, signed containers
- [ ] A09: Logging Failures — security events logged to Audit Service
- [ ] A10: SSRF — webhook URLs, file URLs, redirect validation

### 2.7.6 Load Testing with Security Focus
- [ ] DDoS resilience:
  - [ ] Rate limiting holds under sustained load
  - [ ] Circuit breakers prevent cascade failures
  - [ ] Resource limits prevent container takeover
- [ ] Concurrency attacks:
  - [ ] Race conditions in quota enforcement
  - [ ] Concurrent subscription changes
  - [ ] Parallel file uploads

### 2.7.7 Container Security
- [ ] All containers run as non-root user
- [ ] `no-new-privileges` security option
- [ ] `cap_drop: ALL` (drop all Linux capabilities)
- [ ] Read-only root filesystem where possible
- [ ] Resource limits (CPU, memory)
- [ ] Network segmentation (services can't reach each other directly)

### 2.7.8 Secrets Management Audit
- [ ] No secrets in source code
- [ ] No secrets in container images
- [ ] All secrets in environment variables or mounted files
- [ ] Stripe keys never logged
- [ ] SMTP passwords never logged
- [ ] Database passwords rotated regularly

### 2.7.9 Fuzz Testing
- [ ] Fuzz test notification template rendering
- [ ] Fuzz test file upload handling
- [ ] Fuzz test audit log search filters
- [ ] Fuzz test Stripe webhook parsing

### 2.7.10 Automated Full Platform Pentest
- [ ] Set up automated security testing framework (see `AUTOMATED_SECURITY_TESTING.md`):
  - [ ] Build custom Kali container with all tools
  - [ ] Configure scan targets in `scan-targets.yaml`
  - [ ] Set severity thresholds
- [ ] Run automated penetration test: `make security-pentest`
  - [ ] Network scanning (nmap, masscan)
  - [ ] Web vulnerability scanning (nikto, whatweb, wapiti)
  - [ ] SQL injection (sqlmap)
  - [ ] XSS testing (XSSer)
  - [ ] SSRF testing (custom automated scripts)
  - [ ] API security testing (ffuf, wfuzz)
  - [ ] Authentication testing (JWT attacks)
  - [ ] File upload attacks
- [ ] Review automated HTML/JSON reports in `reports/pentest/`
- [ ] Remediate all Critical and High findings
- [ ] Rerun scan to confirm fixes: `make security-pentest`
- [ ] Add to CI/CD pipeline for continuous security testing

---

## Milestone: M2 — Full Module Suite

**Criteria for M2 completion:**

### Service Completion
- [ ] Shared Rust library (`libs/rust/`) — all crates compiling and tested
- [ ] Notification Service — all channels working, templates rendering, events consumed
- [ ] Billing Service — Stripe integration working, subscriptions, invoicing, usage tracking
- [ ] File Service — MinIO storage, virus scanning, quotas, thumbnails
- [ ] Audit Service — event capture, hash chaining, exports, retention

### Integration
- [ ] All 5 services (Auth, Org, Notification, Billing, File, Audit) running via Compose
- [ ] API Gateway proxying all services with module gating
- [ ] Cross-service event flows working (e.g., login → audit log → notification)
- [ ] End-to-end workflows tested and passing

### Security
- [ ] All SAST scans passing (zero HIGH findings)
- [ ] All DAST scans passing (zero CRITICAL findings)
- [ ] Tenant isolation verified across all services
- [ ] OWASP Top 10 testing complete
- [ ] PCI DSS compliance for Billing Service
- [ ] Full platform pentest complete and remediated

### Documentation
- [ ] All service READMEs complete
- [ ] All OpenAPI specs in `libs/contracts/` matching endpoints
- [ ] Security documentation updated
- [ ] Compliance guide (SOC 2, GDPR, PCI DSS)

### Testing
- [ ] `make test` passes for all services and shared libs
- [ ] `make lint` passes for all services
- [ ] `make test-tenant-isolation` passes
- [ ] Integration tests passing
- [ ] Load tests passing (< 100ms p95)

### Infrastructure
- [ ] `make full-up` starts infra + all 5 services + gateway cleanly
- [ ] All services containerized and healthy
- [ ] Service health monitoring working
- [ ] Circuit breakers functioning

### Git Tag
- [ ] **Git tag: `m2-full-module-suite`** — committed and pushed

---

## Notes & Recommendations

### Build Order
1. **Start with Shared Rust Crate** — Billing and File services depend on it
2. **Parallelize service development**: Notification and Audit (Go) can be built simultaneously with Billing and File (Rust)
3. **Integrate incrementally**: Add one service at a time to the gateway
4. **Test continuously**: Run integration tests after each service addition

### Security Focus
- **Notification Service**: SSRF prevention is critical — validate all webhook URLs
- **Billing Service**: PCI DSS compliance — never log or store card data
- **File Service**: File upload attacks are common — validate magic bytes, not just extensions
- **Audit Service**: Immutability is essential — hash chaining prevents tampering

### Common Pitfalls to Avoid
- Don't expose Phase 2 service ports directly — route through Gateway only
- Don't skip virus scanning — infected files can compromise the system
- Don't trust file extensions — always validate magic bytes
- Don't store secrets in code or images — use environment or mounted files
- Don't skip tenant isolation tests — cross-tenant data leaks are critical vulnerabilities

### Performance Considerations
- Cache notification templates and preferences (Redis, 30min TTL)
- Use presigned URLs for file uploads/downloads (reduce server load)
- Partition audit logs by month (improve query performance)
- Batch notification sending (reduce external API calls)
- Use connection pooling for Postgres and Redis (all services)

### Compliance Reminders
- **SOC 2**: Audit Service provides audit trail for controls
- **GDPR**: Support data export via Audit exports; file deletion must cascade
- **PCI DSS**: Billing Service must never touch card data directly
- **HIPAA** (if applicable): Enable encryption at rest for File Service

---

## Success Metrics

Phase 2 is complete when:
- ✅ All 8 services deployed and healthy
- ✅ Module gating enforced by Gateway
- ✅ Tenant isolation proven across all services
- ✅ Security scans clean (SAST + DAST)
- ✅ Event-driven architecture fully functional
- ✅ Audit trail capturing all service events
- ✅ Documentation complete and accurate
- ✅ M2 milestone tagged in Git

**Estimated timeline**: 8 weeks (parallel development reduces time vs sequential)

Ready to proceed to Phase 3: Analytics, Frontend, and Production Hardening! 🚀
