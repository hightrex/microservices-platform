# Phase 2: Modular Services (Weeks 11–18)

> **Status**: 🟡 IN PROGRESS — Track A & B code complete (All services implemented), Integration done, Security pending
> **Prerequisite**: Phase 1 complete (`m1-core-platform-ready`)
> **Milestone**: M2 — All 7 services running, module gating working, full event-driven architecture
> **Dependency order**: Shared Rust Crate → (Billing + File in parallel) | (Notification + Audit in parallel, no Rust dependency) → Integration
> **Completed**: 2.1 Shared Rust Crate ✅ | 2.2 Notification Service ✅ | 2.3 Billing Service ✅ | 2.4 File Service ✅ | 2.5 Audit Service ✅ | 2.6 Integration ✅
> **Remaining**: 2.7 Phase 2 Security

---

## 2.1 Shared Rust Crate (`libs/rust/`)

The Rust shared library provides common functionality for Billing and File services. **Build first** — the Rust services depend on it.

### 2.1.1 Workspace Setup
- [x] Create `libs/rust/Cargo.toml` — Cargo workspace definition
- [x] Create workspace members:
  - [x] `libs/rust/common/` — core utilities
  - [x] `libs/rust/messaging/` — Redis Streams integration
  - [x] `libs/rust/database/` — Postgres connection pooling
  - [x] `libs/rust/middleware/` — Axum middleware
- [x] Set edition = "2021" and consistent dependency versions across workspace
- [x] Add shared dependencies in workspace `Cargo.toml`:
  - [x] `tokio`, `serde`, `serde_json`, `anyhow`, `thiserror`
  - [x] `tracing`, `tracing-subscriber`
  - [x] `uuid`, `chrono`
- [x] Verify `cargo build --workspace` compiles

### 2.1.2 Common Crate (`libs/rust/common/`)
- [x] `src/config.rs` — configuration loading (environment variables, TOML)
  - [x] `DatabaseConfig` struct (host, port, database, user, password, pool size)
  - [x] `RedisConfig` struct (host, port, password, db)
  - [x] `ServerConfig` struct (host, port, cors_origins)
  - [x] `load_config()` function with validation
- [x] `src/error.rs` — standardized error types
  - [x] `AppError` enum (NotFound, Unauthorized, Forbidden, ValidationError, DatabaseError, ExternalError, InternalError)
  - [x] Implement `std::fmt::Display` and `std::error::Error`
  - [x] Conversion to Axum responses (status codes + JSON body)
  - [x] Error code constants matching Go shared lib
- [x] `src/logger.rs` — structured logging setup
  - [x] Initialize `tracing_subscriber` with JSON formatter
  - [x] Log level from environment variable
  - [x] Add request ID to context
- [x] `src/tenant.rs` — tenant context extraction
  - [x] `TenantContext` struct with `tenant_id` and `org_id`
  - [x] Extract from Axum headers (`X-Tenant-ID`, `X-Org-ID`)
  - [x] `require_tenant()` helper that returns error if missing
- [x] `src/validation.rs` — input validation
  - [x] Integration with `validator` crate
  - [x] `validate_request()` generic function
  - [x] Validation error formatting to match standardized API error format
- [x] Unit tests for each module

### 2.1.3 Database Crate (`libs/rust/database/`)
- [x] `src/pool.rs` — Postgres connection pooling
  - [x] Use `sqlx::PgPool` with configuration from `common::config`
  - [x] Connection health checks
  - [x] Connection lifecycle management
- [x] `src/migrations.rs` — migration runner
  - [x] Use `sqlx::migrate!()` macro
  - [x] `run_migrations()` function
- [x] `src/repository.rs` — base repository traits
  - [x] `TenantScoped` trait for tenant isolation
  - [x] Query builder helpers
- [x] Unit tests with test containers

### 2.1.4 Messaging Crate (`libs/rust/messaging/`)
- [x] `src/producer.rs` — Redis Streams producer
  - [x] `Producer` struct with Redis connection
  - [x] `publish()` method with event schema validation
  - [x] Automatic timestamp and correlation ID injection
- [x] `src/consumer.rs` — Redis Streams consumer
  - [x] `Consumer` struct with consumer group support
  - [x] `subscribe()` method with message acknowledgment
  - [x] Dead letter queue (DLQ) support
  - [x] Retry logic with exponential backoff
- [x] `src/schema.rs` — event schema definitions
  - [x] Common event structure (event_type, tenant_id, timestamp, data)
  - [x] Serialization/deserialization with `serde`
- [x] Integration tests against real Redis

### 2.1.5 Middleware Crate (`libs/rust/middleware/`)
- [x] `src/auth.rs` — JWT validation middleware
  - [x] Extract and validate JWT from `Authorization` header
  - [x] Parse claims (sub, tid, org, roles, exp)
  - [x] Add user context to Axum extensions
  - [x] Token blacklist check (Redis)
- [x] `src/tenant.rs` — tenant extraction middleware
  - [x] Extract `X-Tenant-ID` header
  - [x] Validate against JWT claims
  - [x] Add to request extensions
- [x] `src/logging.rs` — request logging middleware
  - [x] Log request method, path, status, duration
  - [x] Include request ID, tenant ID, user ID
- [x] `src/metrics.rs` — Prometheus metrics middleware
  - [x] Request duration histogram
  - [x] Request counter by method/path/status
  - [x] Active request gauge
- [x] `src/cors.rs` — CORS middleware
  - [x] Configurable allowed origins
  - [x] Proper preflight handling
- [x] `src/rate_limit.rs` — rate limiting middleware
  - [x] Redis-backed sliding window
  - [x] Per-tenant and per-user limits
  - [x] Return `X-RateLimit-*` headers
  - [x] 429 status with `Retry-After`
- [x] Unit tests for each middleware

### 2.1.6 Health Check Module
- [x] `libs/rust/common/src/health.rs` — health check framework
  - [x] `HealthCheck` trait
  - [x] `DatabaseHealthCheck` implementation
  - [x] `RedisHealthCheck` implementation
  - [x] Aggregate health status endpoint
  - [x] Dependency status reporting

### 2.1.7 Tracing & Observability
- [x] `libs/rust/common/src/tracing.rs` — OpenTelemetry setup
  - [x] Initialize OpenTelemetry tracer
  - [x] Jaeger exporter configuration
  - [x] Context propagation helpers
  - [x] Span creation macros
- [x] `libs/rust/common/src/metrics.rs` — Prometheus metrics
  - [x] Metrics registry
  - [x] Counter, histogram, gauge helpers
  - [x] `/metrics` endpoint handler

### 2.1.8 Documentation & Testing
- [x] `libs/rust/README.md` — usage guide for each crate
- [x] Examples in `libs/rust/examples/` directory
- [x] Run `cargo test --workspace` — all tests passing
- [x] Run `cargo clippy --workspace -- -D warnings` — zero warnings
- [x] Run `cargo fmt --check` — all code formatted

---

## 2.2 Notification Service (Go/Gin — Port 8082) ✅

Multi-channel notification service with template engine, preference management, and delivery tracking.

### 2.2.1 Scaffold & Configuration ✅
- [x] Run `scripts/create-service.sh notification-service`
- [x] Create `internal/config/config.go` — service config struct:
  - [x] Server config (port, timeouts)
  - [x] Database and Redis config
  - [x] SMTP config (host, port, username, password, from address)
  - [x] Twilio config (account SID, auth token, from number)
  - [x] Webhook config (timeout, retry attempts, max redirects)
  - [x] Template config (default language, supported languages)
- [x] Create `config.yaml` — default dev configuration
- [x] Add `replace` directive in `go.mod` for `libs/go`
- [x] Verify `go build ./...` compiles

### 2.2.2 Database Migrations ✅
All tables include standard fields. Tenant-scoped tables include `tenant_id`.

- [x] `migrations/001_create_notification_templates_table.up.sql`
  - Columns: `id`, `tenant_id`, `name`, `channel` (enum: email/sms/in_app/webhook), `subject_template`, `body_template`, `template_data_schema` (JSONB), `language` (default: en), `is_active`, `created_at`, `updated_at`, `created_by`
  - Indexes: `(tenant_id, name, channel)` UNIQUE, `(tenant_id, is_active)`
  - Template uses Go template syntax with safe functions
- [x] `migrations/001_create_notification_templates_table.down.sql`
- [x] `migrations/002_create_notification_preferences_table.up.sql`
  - Columns: `id`, `user_id`, `tenant_id`, `channel` (enum), `event_type`, `enabled`, `updated_at`
  - Purpose: per-user notification opt-in/opt-out
  - Indexes: `(user_id, tenant_id, channel, event_type)` UNIQUE
- [x] `migrations/002_create_notification_preferences_table.down.sql`
- [x] `migrations/003_create_notifications_table.up.sql`
  - Columns: `id`, `tenant_id`, `user_id`, `channel`, `event_type`, `subject`, `body`, `status` (enum: pending/sent/failed/read), `metadata` (JSONB), `sent_at`, `read_at`, `created_at`
  - Purpose: in-app notification storage + delivery audit trail
  - Indexes: `(user_id, tenant_id, status)`, `(tenant_id, created_at)`
- [x] `migrations/003_create_notifications_table.down.sql`
- [x] `migrations/004_create_notification_delivery_log_table.up.sql`
  - Columns: `id`, `notification_id`, `tenant_id`, `channel`, `recipient`, `status` (enum: delivered/failed/bounced), `error_message`, `provider_response` (JSONB), `retry_count`, `delivered_at`, `created_at`
  - Purpose: track every delivery attempt with provider responses
  - Indexes: `(notification_id)`, `(tenant_id, status, created_at)`
- [x] `migrations/004_create_notification_delivery_log_table.down.sql`
- [x] `migrations/005_create_notification_dlq_table.up.sql`
  - Columns: `id`, `tenant_id`, `event_type`, `payload` (JSONB), `error`, `retry_count`, `created_at`, `retried_at`
  - Purpose: dead letter queue for failed notification processing
  - Index: `(tenant_id, created_at)`
- [x] `migrations/005_create_notification_dlq_table.down.sql`
- [x] Verify migrations run against local Postgres

### 2.2.3 Domain Models ✅
- [x] `internal/models/template.go`
  - [x] `NotificationTemplate` struct
  - [x] `CreateTemplateRequest`, `UpdateTemplateRequest` with validation tags
  - [x] `TemplateResponse` (omit internal fields)
  - [x] `Channel` enum type
- [x] `internal/models/notification.go`
  - [x] `Notification` struct
  - [x] `SendNotificationRequest` with validation (channel, recipient, template_id, data)
  - [x] `NotificationResponse`
  - [x] `NotificationStatus` enum
- [x] `internal/models/preference.go`
  - [x] `NotificationPreference` struct
  - [x] `UpdatePreferenceRequest`
  - [x] `PreferenceResponse`
- [x] `internal/models/delivery.go`
  - [x] `DeliveryLog` struct
  - [x] `DeliveryStatus` enum
  - [x] `ProviderResponse` struct

### 2.2.4 Repository Layer ✅
All methods use `tenant.RequireTenant(ctx)` or `tenant.NewScope(ctx)`.

- [x] `internal/repository/postgres/template_repo.go`
  - [x] `Create(ctx, template)` — tenant-scoped
  - [x] `GetByID(ctx, id)` — tenant-scoped
  - [x] `GetByName(ctx, name, channel)` — for lookup by name + channel
  - [x] `List(ctx, filter)` — tenant-scoped, filterable by channel and status
  - [x] `Update(ctx, id, fields)` — partial update
  - [x] `Delete(ctx, id)` — soft delete (set is_active=false)
- [x] `internal/repository/postgres/notification_repo.go`
  - [x] `Create(ctx, notification)` — in-app notification storage
  - [x] `GetByID(ctx, id)` — tenant-scoped
  - [x] `List(ctx, userID, filter)` — user's in-app notifications with pagination
  - [x] `MarkAsRead(ctx, id)` — set read_at timestamp
  - [x] `CountUnread(ctx, userID)` — unread count for user
- [x] `internal/repository/postgres/preference_repo.go`
  - [x] `GetUserPreferences(ctx, userID)` — all preferences for user
  - [x] `UpdatePreference(ctx, userID, eventType, channel, enabled)` — upsert
  - [x] `CheckEnabled(ctx, userID, eventType, channel)` — opt-in check
- [x] `internal/repository/postgres/delivery_repo.go`
  - [x] `Create(ctx, log)` — record delivery attempt
  - [x] `List(ctx, notificationID)` — delivery history for notification
  - [x] `GetFailedDeliveries(ctx, threshold)` — for retry worker
- [x] `internal/repository/redis/notification_cache.go`
  - [x] Cache templates (1 hour TTL)
  - [x] Cache user preferences (30 min TTL)
- [ ] Unit tests for each repository

### 2.2.5 Service Layer ✅

#### Core Services
- [x] `internal/service/template_service.go`
  - [x] `CreateTemplate(ctx, req)` — validate template syntax, save, publish event
  - [x] `GetTemplate(ctx, id)` — with caching
  - [x] `ListTemplates(ctx, filter)` — tenant-scoped
  - [x] `UpdateTemplate(ctx, id, req)` — invalidate cache
  - [x] `DeleteTemplate(ctx, id)` — soft delete
  - [x] `RenderTemplate(template, data)` — safe template rendering with sandbox (html/template)
  - [x] Validate template data schema against JSONB schema field

- [x] `internal/service/notification_service.go`
  - [x] `Send(ctx, req)` — orchestrate notification sending:
    1. Load template
    2. Check user preferences
    3. Render template with data
    4. Route to appropriate channel handler
    5. Create delivery log
    6. Return notification ID
  - [x] `GetByID(ctx, id)` — retrieve notification
  - [x] `List(ctx, userID, filter)` — user's in-app notifications
  - [x] `MarkAsRead(ctx, id)` — update read status
  - [x] `GetUnreadCount(ctx, userID)` — badge count
  - [x] Validate webhook URLs to prevent SSRF (no private IPs, localhost, etc.)

- [x] `internal/service/preference_service.go`
  - [x] `GetPreferences(ctx, userID)` — with caching
  - [x] `UpdatePreference(ctx, req)` — save, invalidate cache
  - [x] `GetDefaultPreferences()` — system defaults for new users

#### Channel Implementations
- [x] `internal/service/channels/email_channel.go`
  - [x] `Send(recipient, subject, body, metadata)` — SMTP with TLS
  - [x] HTML and plain text support
  - [x] Retry logic with exponential backoff (3 attempts)

- [x] `internal/service/channels/sms_channel.go`
  - [x] `Send(recipient, body, metadata)` — Twilio integration
  - [x] Phone number validation (E.164 format)
  - [x] Message truncation with warning
  - [x] Retry logic

- [x] `internal/service/channels/in_app_channel.go`
  - [x] `Send(userID, subject, body, metadata)` — database storage

- [x] `internal/service/channels/webhook_channel.go`
  - [x] `Send(url, payload, metadata)` — HTTP POST
  - [x] Signature generation (HMAC-SHA256) for webhook verification
  - [x] Timeout (10 seconds)
  - [x] Retry with exponential backoff (5 attempts)
  - [x] Follow redirects (max 3) with URL validation on each hop
  - [x] SSRF prevention (block private IPs, localhost, metadata endpoints)

#### Event Processing
- [x] `internal/consumer/event_consumer.go`
  - [x] Subscribe to auth-events and org-events streams
  - [x] HandleUserCreated — send welcome notification
  - [x] HandleOrgCreated — send org created notification

- [x] `internal/service/retry_worker.go`
  - [x] Background job to retry failed deliveries
  - [x] Exponential backoff schedule
  - [x] Configurable retry policy per channel

### 2.2.6 HTTP Handlers ✅
- [x] `internal/handlers/notification_handler.go`
  - [x] `POST /api/v1/notifications/send` — manual notification sending
  - [x] `GET /api/v1/notifications` — list in-app notifications (tenant-scoped, paginated)
  - [x] `GET /api/v1/notifications/:id` — get notification by ID
  - [x] `PUT /api/v1/notifications/:id/read` — mark as read
  - [x] `GET /api/v1/notifications/unread/count` — unread count
- [x] `internal/handlers/template_handler.go`
  - [x] `POST /api/v1/notifications/templates` — create template (admin only)
  - [x] `GET /api/v1/notifications/templates` — list templates
  - [x] `GET /api/v1/notifications/templates/:id` — get template
  - [x] `PUT /api/v1/notifications/templates/:id` — update template
  - [x] `DELETE /api/v1/notifications/templates/:id` — deactivate template
- [x] `internal/handlers/preference_handler.go`
  - [x] `GET /api/v1/notifications/preferences` — get user preferences
  - [x] `PUT /api/v1/notifications/preferences` — update preferences (bulk)
  - [x] `PUT /api/v1/notifications/preferences/:channel/:event_type` — update single preference
- [x] All handlers use standardized error responses

### 2.2.7 API Route Registration ✅
- [x] `api/routes.go` — register all routes with middleware:
  1. Recovery, RequestLogger, Tenant, Metrics
  2. GatewayAuth middleware (identity from gateway headers)
  3. RBAC middleware (admin routes require org_owner/org_admin role)
- [x] Public routes: `/health`, `/metrics`
- [x] Protected routes: all API endpoints
- [x] Admin routes: template management

### 2.2.8 Redis Streams Events ✅
Publish events for other services:
- [x] `notification.sent` — successful delivery
- [x] `notification.failed` — delivery failure
- [x] `notification.template_created` — new template
- [x] `notification.preference_updated` — user preference change

### 2.2.9 Observability ✅
- [x] OpenTelemetry tracing (optional, configured via config)
- [x] Prometheus metrics:
  - [x] `notifications_sent_total` (channel, status)
  - [x] `notifications_delivery_duration_seconds` (channel)
  - [x] `notifications_retry_total` (channel)
  - [x] `notifications_dlq_total`
- [x] Health check with Postgres, Redis, and event consumer status

### 2.2.10 Containerfile ✅
- [x] Multi-stage Go build (golang:1.25-alpine → alpine:3.21.2)
- [x] Non-root user (`appuser`)
- [x] Pinned base image
- [x] `HEALTHCHECK` instruction
- [x] No secrets in image

### 2.2.11 Tests (partial)
- [x] Unit tests:
  - [x] Template rendering (valid, invalid, XSS attempts)
  - [x] Template validation (syntax checking)
  - [x] SSRF prevention in webhook URLs (private IPs, localhost, invalid schemes)
  - [x] E.164 phone number validation
  - [x] HMAC signature generation
  - [x] Channel enum validation
- [ ] Integration tests:
  - [ ] Full send flow (template → render → deliver)
  - [ ] Event consumer processing
  - [ ] DLQ handling
  - [ ] Delivery log creation
- [ ] Security tests:
  - [ ] Template injection attempts
  - [ ] SSRF in webhook URLs (DNS rebinding)
  - [ ] Phone number validation bypass attempts

### 2.2.12 Documentation (partial)
- [x] `README.md` — architecture, setup, usage, API, events
- [ ] `libs/contracts/notification-service.yaml` — OpenAPI spec
- [ ] Template syntax documentation
- [ ] Channel configuration guide

---

## 2.3 Billing Service (Rust/Axum — Port 8083)

Subscription management with Stripe integration, usage metering, invoicing, and proration.

### 2.3.1 Scaffold & Configuration
- [x] Create `services/billing-service/` directory
- [x] Initialize Cargo project: `cargo init --name billing-service`
- [x] Add dependencies in `Cargo.toml`:
  - [x] `axum`, `tokio`, `tower`, `tower-http`
  - [x] `sqlx` with postgres feature
  - [x] `redis`
  - [x] `stripe-rust` (or `async-stripe`)
  - [x] `serde`, `serde_json`
  - [x] Workspace dependencies from `libs/rust`
- [x] Add workspace member to `libs/rust/Cargo.toml`
- [x] Create `config/config.toml` — service configuration:
  - [x] Server config (host, port)
  - [x] Database config
  - [x] Redis config
  - [x] Stripe config (secret key, webhook secret, publishable key)
  - [x] Invoice config (due days, late fee percentage)
  - [x] Feature flag for Stripe test mode
- [x] Create `.env.example`
- [x] Verify `cargo build` compiles

### 2.3.2 Database Migrations
Use `sqlx-cli` for migrations: `cargo install sqlx-cli`.

- [x] `migrations/001_create_subscriptions_table.up.sql`
  - Columns: `id`, `tenant_id`, `org_id`, `plan_id`, `stripe_subscription_id`, `status` (enum: active/past_due/canceled/paused), `current_period_start`, `current_period_end`, `cancel_at`, `canceled_at`, `trial_end`, `created_at`, `updated_at`
  - Indexes: `(tenant_id)`, `(stripe_subscription_id)` UNIQUE
  - Only one active subscription per tenant
- [x] `migrations/001_create_subscriptions_table.down.sql`
- [x] `migrations/002_create_plans_table.up.sql`
  - Columns: `id`, `name`, `stripe_price_id`, `billing_interval` (enum: month/year), `price_cents`, `currency`, `features` (JSONB), `is_active`, `created_at`, `updated_at`
  - Purpose: mirror Stripe plans/prices locally
  - Index: `(stripe_price_id)` UNIQUE
  - Seed with starter, professional, enterprise plans
- [x] `migrations/002_create_plans_table.down.sql`
- [x] `migrations/003_create_invoices_table.up.sql`
  - Columns: `id`, `tenant_id`, `subscription_id`, `stripe_invoice_id`, `invoice_number`, `status` (enum: draft/open/paid/void/uncollectible), `amount_due_cents`, `amount_paid_cents`, `currency`, `due_date`, `paid_at`, `pdf_url`, `created_at`, `updated_at`
  - Indexes: `(tenant_id, created_at)`, `(stripe_invoice_id)` UNIQUE
- [x] `migrations/003_create_invoices_table.down.sql`
- [x] `migrations/004_create_payments_table.up.sql`
  - Columns: `id`, `tenant_id`, `invoice_id`, `stripe_payment_intent_id`, `amount_cents`, `currency`, `status` (enum: succeeded/pending/failed), `payment_method_type`, `receipt_url`, `failure_reason`, `paid_at`, `created_at`
  - Purpose: track all payment attempts
  - Indexes: `(tenant_id, created_at)`, `(stripe_payment_intent_id)` UNIQUE
  - NEVER store card numbers, CVV, or full PANs
- [x] `migrations/004_create_payments_table.down.sql`
- [x] `migrations/005_create_usage_records_table.up.sql`
  - Columns: `id`, `tenant_id`, `subscription_id`, `metric_name` (e.g., api_calls, storage_gb, users), `quantity`, `recorded_at`, `aggregated`, `created_at`
  - Purpose: metered billing data
  - Indexes: `(tenant_id, metric_name, recorded_at)`, `(subscription_id, aggregated)`
- [x] `migrations/005_create_usage_records_table.down.sql`
- [x] Run migrations: `sqlx migrate run`

### 2.3.3 Domain Models
- [x] `src/models/subscription.rs`
  - [x] `Subscription` struct
  - [x] `CreateSubscriptionRequest` with validation
  - [x] `UpdateSubscriptionRequest` (upgrade/downgrade)
  - [x] `CancelSubscriptionRequest`
  - [x] `SubscriptionResponse`
  - [x] `SubscriptionStatus` enum
- [x] `src/models/plan.rs`
  - [x] `Plan` struct
  - [x] `PlanResponse`
  - [x] `BillingInterval` enum
  - [x] `Features` struct (from JSONB)
- [x] `src/models/invoice.rs`
  - [x] `Invoice` struct
  - [x] `InvoiceResponse`
  - [x] `InvoiceStatus` enum
  - [x] `InvoiceLineItem` struct
- [x] `src/models/payment.rs`
  - [x] `Payment` struct
  - [x] `PaymentResponse` (without sensitive data)
  - [x] `PaymentStatus` enum
- [x] `src/models/usage.rs`
  - [x] `UsageRecord` struct
  - [x] `RecordUsageRequest`
  - [x] `UsageResponse` with aggregation

### 2.3.4 Repository Layer
All queries use `libs/rust/database` helpers and tenant scoping.

- [x] `src/repository/subscription_repo.rs`
  - [x] `create(pool, subscription)` — tenant-scoped
  - [x] `get_by_id(pool, id)` — tenant-scoped
  - [x] `get_by_tenant(pool, tenant_id)` — current subscription
  - [x] `update(pool, id, fields)` — partial update
  - [x] `cancel(pool, id)` — set canceled_at
  - [x] `get_expiring_trials(pool, days)` — for reminder notifications
- [x] `src/repository/plan_repo.rs`
  - [x] `get_by_id(pool, id)`
  - [x] `get_by_stripe_price_id(pool, stripe_price_id)`
  - [x] `list_active(pool)` — public plans
- [x] `src/repository/invoice_repo.rs`
  - [x] `create(pool, invoice)` — tenant-scoped
  - [x] `get_by_id(pool, id)` — tenant-scoped
  - [x] `list_by_tenant(pool, tenant_id, pagination)` — with filtering
  - [x] `update_status(pool, id, status)`
- [x] `src/repository/payment_repo.rs`
  - [x] `create(pool, payment)` — tenant-scoped
  - [x] `get_by_invoice(pool, invoice_id)`
  - [x] `list_by_tenant(pool, tenant_id, pagination)`
- [x] `src/repository/usage_repo.rs`
  - [x] `record(pool, usage)` — tenant-scoped
  - [x] `aggregate(pool, tenant_id, metric, start, end)` — sum quantities
  - [x] `get_current_period(pool, subscription_id)` — usage this billing cycle
- [x] `src/repository/cache/` — Redis caching for plans and subscriptions (5 min TTL)
- [x] Unit tests for each repository (use `sqlx::test` with test database)

### 2.3.5 Service Layer

#### Core Services
- [x] `src/service/subscription_service.rs`
  - [x] `create_subscription(ctx, req)` — create in Stripe, save locally, publish event
    - [x] Validate plan exists
    - [x] Check tenant doesn't have active subscription (or cancel existing)
    - [x] Create Stripe subscription with trial if new customer
    - [x] Handle Stripe payment failures gracefully
  - [x] `get_subscription(ctx, tenant_id)` — with plan details
  - [x] `upgrade_subscription(ctx, req)` — change plan with proration
    - [x] Calculate proration amount
    - [x] Update Stripe subscription
    - [x] Sync status locally
  - [x] `downgrade_subscription(ctx, req)` — schedule change for end of period
  - [x] `cancel_subscription(ctx, immediate)` — cancel now or at period end
  - [x] `pause_subscription(ctx)` — pause billing (if supported by plan)
  - [x] `resume_subscription(ctx)` — resume from pause

- [x] `src/service/invoice_service.rs`
  - [x] `list_invoices(ctx, tenant_id, filter)` — paginated
  - [x] `get_invoice(ctx, id)` — with line items
  - [x] `generate_pdf(ctx, invoice_id)` — call File Service API
  - [x] `send_invoice(ctx, invoice_id)` — trigger email notification
  - [x] `mark_as_paid(ctx, id)` — manual payment confirmation

- [x] `src/service/usage_service.rs`
  - [x] `record_usage(ctx, req)` — save usage record, publish event
  - [x] `get_usage(ctx, tenant_id, metric, period)` — aggregated usage
  - [x] `get_quota_status(ctx, tenant_id)` — current usage vs plan limits
  - [x] `enforce_quota(ctx, tenant_id, metric)` — check if over limit
  - [x] Background job to sync usage to Stripe metered billing

- [x] `src/service/webhook_service.rs`
  - [x] `handle_webhook(signature, payload)` — verify and process Stripe webhooks:
    - [x] `invoice.payment_succeeded` → update invoice status, publish event
    - [x] `invoice.payment_failed` → update status, trigger notification
    - [x] `customer.subscription.updated` → sync subscription status
    - [x] `customer.subscription.deleted` → mark as canceled
    - [x] `payment_intent.succeeded` → record payment
    - [x] `payment_intent.payment_failed` → log failure
  - [x] Verify webhook signature (Stripe's HMAC)
  - [x] Idempotency handling (process each event only once)

#### Stripe Integration
- [x] `src/service/stripe_client.rs`
  - [x] Wrapper around `stripe-rust` SDK
  - [x] `create_customer(email, metadata)` — for new tenants
  - [x] `create_subscription(customer_id, price_id, options)`
  - [x] `update_subscription(subscription_id, price_id)`
  - [x] `cancel_subscription(subscription_id, immediately)`
  - [x] `retrieve_invoice(invoice_id)`
  - [x] `create_usage_record(subscription_item_id, quantity)`
  - [x] Error handling and retry logic
  - [x] Logging (no sensitive data)

#### Proration Logic
- [x] `src/service/proration.rs`
  - [x] Calculate prorated amount for upgrades/downgrades
  - [x] Credit calculation for unused time
  - [x] Generate preview invoice before applying change
  - [x] Handle different billing intervals (monthly vs annual)

### 2.3.6 HTTP Handlers
- [x] `src/handlers/subscription_handler.rs`
  - [x] `GET /api/v1/billing/subscription` — get current subscription
  - [x] `POST /api/v1/billing/subscription` — create subscription
  - [x] `PUT /api/v1/billing/subscription` — upgrade/downgrade
  - [x] `DELETE /api/v1/billing/subscription` — cancel subscription
  - [x] `POST /api/v1/billing/subscription/pause` — pause subscription
  - [x] `POST /api/v1/billing/subscription/resume` — resume subscription
- [x] `src/handlers/invoice_handler.rs`
  - [x] `GET /api/v1/billing/invoices` — list invoices (tenant-scoped)
  - [x] `GET /api/v1/billing/invoices/:id` — get invoice details
  - [x] `GET /api/v1/billing/invoices/:id/pdf` — download PDF
  - [x] `POST /api/v1/billing/invoices/:id/pay` — manual payment
- [x] `src/handlers/usage_handler.rs`
  - [x] `GET /api/v1/billing/usage` — get current usage
  - [x] `GET /api/v1/billing/usage/quota` — quota status
  - [x] `POST /api/v1/billing/usage/record` — record usage (internal API)
- [x] `src/handlers/webhook_handler.rs`
  - [x] `POST /api/v1/billing/webhook` — Stripe webhook endpoint (public)
  - [x] Signature verification
  - [x] Async processing (queue events if needed)
- [x] `src/handlers/plan_handler.rs`
  - [x] `GET /api/v1/billing/plans` — list available plans

### 2.3.7 API Route Registration
- [x] `src/routes.rs` — configure Axum router
  - [x] Apply middleware stack from `libs/rust/middleware`
  - [x] Auth middleware on all routes except webhook
  - [x] Tenant middleware
  - [x] Logging and metrics
- [x] Public routes: `/health`, `/api/v1/billing/webhook`
- [x] Protected routes: all other endpoints
- [x] Rate limiting on webhook endpoint (prevent abuse)

### 2.3.8 Redis Streams Events
- [x] `subscription.created` — new subscription
- [x] `subscription.updated` — plan change, status change
- [x] `subscription.canceled` — subscription canceled
- [x] `invoice.paid` — payment succeeded
- [x] `invoice.failed` — payment failed
- [x] `usage.recorded` — usage data point
- [x] `usage.quota_exceeded` — over plan limits

### 2.3.9 Observability
- [x] OpenTelemetry tracing on all operations
- [x] Prometheus metrics:
  - [x] `billing_subscriptions_total` (status)
  - [x] `billing_revenue_cents` (period)
  - [x] `billing_invoices_total` (status)
  - [x] `billing_payments_total` (status)
  - [x] `billing_usage_total` (metric_name)
  - [x] `billing_webhook_events_total` (event_type, status)
  - [x] `billing_stripe_api_duration_seconds` (operation)
- [x] Health check with Stripe API connectivity and database

### 2.3.10 Containerfile
- [x] Multi-stage Rust build (builder → runtime)
- [x] Use latest stable `rust:<version>-slim` for builder (check current stable at build time)
- [x] Use `debian:bookworm-slim` for runtime
- [x] Non-root user
- [x] `HEALTHCHECK` instruction
- [x] Copy only necessary binaries
- [x] No secrets in image

### 2.3.11 Tests
- [x] Unit tests:
  - [x] Proration calculations
  - [x] Quota enforcement logic
  - [x] Webhook signature verification
  - [x] Plan upgrade/downgrade scenarios
  - [x] Payment status transitions
- [x] Integration tests:
  - [x] Full subscription lifecycle (create → upgrade → cancel)
  - [x] Invoice generation flow
  - [x] Usage recording and aggregation
  - [x] Webhook processing (mock Stripe)
- [x] Security tests:
  - [x] PCI DSS compliance checks (no card data storage)
  - [x] Webhook signature tampering attempts
  - [x] Tenant isolation (can't access other tenant's billing)
  - [x] SQL injection in usage records
  - [x] Amount manipulation attempts

### 2.3.12 Documentation
- [x] `README.md` — architecture, Stripe setup, testing with test mode
- [x] `libs/contracts/billing-service.yaml` — OpenAPI spec
- [x] Stripe webhook configuration guide
- [x] Plan configuration documentation
- [x] PCI DSS compliance notes
- [x] Usage metering guide for other services

---

## 2.4 File Service (Rust/Axum — Port 8084)

S3-compatible file storage with MinIO, quota enforcement, virus scanning, and signed URL generation.

### 2.4.1 Scaffold & Configuration
- [x] Create `services/file-service/` directory
- [x] Initialize Cargo project: `cargo init --name file-service`
- [x] Add dependencies:
  - [x] `axum`, `tokio`, `tower`, `tower-http`
  - [x] `sqlx` with postgres feature
  - [x] `redis`
  - [x] `aws-sdk-s3` (MinIO is S3-compatible)
  - [x] `image` (for thumbnail generation)
  - [x] `mime_guess`
  - [x] `sha2` (for file hashing)
  - [x] Workspace dependencies from `libs/rust`
- [x] Create `config/config.toml`:
  - [x] Server config
  - [x] Database and Redis config
  - [x] S3 config (endpoint, region, bucket, access key, secret key)
  - [x] Quota config (default per-tenant limit, per-file size limit)
  - [x] Thumbnail config (max dimensions, quality)
  - [x] Allowed file types (MIME types whitelist)
  - [x] Virus scanning config (ClamAV endpoint, enabled flag)
- [x] Verify `cargo build` compiles

### 2.4.2 Database Migrations
- [x] `migrations/001_create_files_table.up.sql`
  - Columns: `id`, `tenant_id`, `user_id`, `org_id`, `filename`, `content_type`, `size_bytes`, `s3_key`, `s3_bucket`, `checksum_sha256`, `status` (enum: uploading/available/deleted/quarantined), `virus_scan_status` (enum: pending/clean/infected/error), `thumbnail_s3_key`, `uploaded_at`, `deleted_at`, `created_at`, `updated_at`
  - Indexes: `(tenant_id, status)`, `(s3_key)` UNIQUE, `(tenant_id, user_id)`
  - Foreign key: `user_id` references Auth Service (logical, not enforced)
- [x] `migrations/001_create_files_table.down.sql`
- [x] `migrations/002_create_storage_quotas_table.up.sql`
  - Columns: `id`, `tenant_id`, `quota_bytes`, `used_bytes`, `updated_at`
  - Purpose: track per-tenant storage usage
  - Index: `(tenant_id)` UNIQUE
  - Trigger to update `used_bytes` on file insert/delete
- [x] `migrations/002_create_storage_quotas_table.down.sql`
- [x] `migrations/003_create_file_access_log_table.up.sql`
  - Columns: `id`, `file_id`, `tenant_id`, `user_id`, `action` (enum: upload/download/delete/view), `ip_address`, `user_agent`, `created_at`
  - Purpose: audit trail for compliance
  - Indexes: `(file_id)`, `(tenant_id, created_at)`
- [x] `migrations/003_create_file_access_log_table.down.sql`
- [x] Run migrations

### 2.4.3 Domain Models
- [x] `src/models/file.rs`
  - [x] `File` struct
  - [x] `UploadRequest` (multipart form data handling)
  - [x] `FileResponse`
  - [x] `FileStatus` enum
  - [x] `VirusScanStatus` enum
- [x] `src/models/quota.rs`
  - [x] `StorageQuota` struct
  - [x] `QuotaResponse`
  - [x] `QuotaExceededError`
- [x] `src/models/access_log.rs`
  - [x] `FileAccessLog` struct
  - [x] `AccessAction` enum

### 2.4.4 Repository Layer
- [x] `src/repository/file_repo.rs`
  - [x] `create(pool, file)` — tenant-scoped
  - [x] `get_by_id(pool, id)` — tenant-scoped
  - [x] `list(pool, tenant_id, filter, pagination)` — with status filtering
  - [x] `update_status(pool, id, status)`
  - [x] `update_scan_status(pool, id, scan_status)`
  - [x] `delete(pool, id)` — soft delete (set status=deleted)
  - [x] `get_by_s3_key(pool, s3_key)`
- [x] `src/repository/quota_repo.rs`
  - [x] `get_quota(pool, tenant_id)` — with upsert if missing
  - [x] `increment_usage(pool, tenant_id, bytes)` — atomic update
  - [x] `decrement_usage(pool, tenant_id, bytes)` — atomic update
  - [x] `check_quota(pool, tenant_id, additional_bytes)` — would exceed?
- [x] `src/repository/access_log_repo.rs`
  - [x] `create(pool, log)` — tenant-scoped
  - [x] `list(pool, file_id, pagination)` — audit trail
- [x] `src/repository/cache/` — Redis caching for file metadata (10 min TTL)
- [x] Unit tests for repositories

### 2.4.5 Service Layer

#### Core Services
- [x] `src/service/file_service.rs`
  - [x] `upload(ctx, stream, metadata)` — full upload flow:
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
  - [x] `get_file(ctx, id)` — with download URL generation
  - [x] `list_files(ctx, filter)` — tenant-scoped, paginated
  - [x] `delete_file(ctx, id)` — soft delete + decrement quota
  - [x] `get_download_url(ctx, id, expiry)` — signed URL (default 1 hour)
  - [x] `get_upload_url(ctx, filename, content_type)` — presigned upload URL
  - [x] Validate no path traversal in filenames

- [x] `src/service/quota_service.rs`
  - [x] `get_quota_status(ctx, tenant_id)` — current usage and limit
  - [x] `check_quota(ctx, tenant_id, size)` — enforce before upload
  - [x] `update_quota_limit(ctx, tenant_id, new_limit)` — admin only
  - [x] Background job to recalculate quotas (daily reconciliation)

- [x] `src/service/thumbnail_service.rs`
  - [x] `generate(file_id, s3_key)` — for images only
  - [x] Download from S3
  - [x] Resize to 200x200 (maintain aspect ratio)
  - [x] Save back to S3 with `-thumb` suffix
  - [x] Update file record with thumbnail S3 key
  - [x] Supported formats: JPEG, PNG, GIF, WebP
  - [x] Error handling (skip if not image or generation fails)

- [x] `src/service/virus_scan_service.rs`
  - [x] `scan(file_id, s3_key)` — async scan
  - [x] Download file from S3 (stream)
  - [x] Send to ClamAV via TCP/Unix socket
  - [x] Parse scan result
  - [x] Update virus_scan_status
  - [x] If infected: set status=quarantined, publish alert event
  - [x] If clean: set status=available
  - [x] Queue scan on upload, process in background worker

#### S3 Client
- [x] `src/service/s3_client.rs`
  - [x] Wrapper around `aws-sdk-s3`
  - [x] Configure for MinIO (custom endpoint)
  - [x] Per-org bucket isolation (bucket name: `{org_id}-files`)
  - [x] Create bucket if not exists (on service startup)
  - [x] Bucket versioning enabled
  - [x] Lifecycle policy: delete files with status=deleted after 30 days
  - [x] `upload(bucket, key, stream, content_type)` — multipart for large files
  - [x] `download(bucket, key)` — stream
  - [x] `delete(bucket, key)`
  - [x] `generate_presigned_get_url(bucket, key, expiry)` — signed download
  - [x] `generate_presigned_put_url(bucket, key, expiry)` — signed upload
  - [x] Error handling and retry logic

### 2.4.6 HTTP Handlers
- [x] `src/handlers/file_handler.rs`
  - [x] `POST /api/v1/files/upload` — multipart form upload
    - [x] Accept `file` field (binary)
    - [x] Optional metadata fields (tags, description)
    - [x] Return file ID and metadata
  - [x] `GET /api/v1/files/:id` — get file metadata
  - [x] `GET /api/v1/files/:id/download` — redirect to signed S3 URL
  - [x] `GET /api/v1/files/:id/thumbnail` — redirect to thumbnail URL
  - [x] `DELETE /api/v1/files/:id` — soft delete
  - [x] `GET /api/v1/files` — list files (tenant-scoped, paginated)
  - [x] `GET /api/v1/files/:id/access-log` — audit trail
- [x] `src/handlers/quota_handler.rs`
  - [x] `GET /api/v1/files/quota` — current quota status
- [x] `src/handlers/presigned_handler.rs`
  - [x] `POST /api/v1/files/presigned-upload-url` — generate upload URL
  - [x] Return URL + required headers for client-side upload

### 2.4.7 API Route Registration
- [x] `src/routes.rs` — Axum router with middleware
  - [x] Auth, tenant, logging, metrics middleware
  - [x] File upload size limit (e.g., 100MB per request)
  - [x] Multipart form data handling
- [x] Public routes: `/health`
- [x] Protected routes: all file operations

### 2.4.8 Redis Streams Events
- [x] `file.uploaded` — new file uploaded
- [x] `file.deleted` — file deleted
- [x] `file.scan_completed` — virus scan result
- [x] `file.quarantined` — infected file detected
- [x] `quota.exceeded` — tenant over quota

### 2.4.9 Observability
- [x] OpenTelemetry tracing
- [x] Prometheus metrics:
  - [x] `files_uploads_total` (status)
  - [x] `files_downloads_total`
  - [x] `files_storage_bytes` (tenant_id)
  - [x] `files_scan_duration_seconds`
  - [x] `files_upload_duration_seconds`
  - [x] `files_virus_detections_total`
- [x] Health check with MinIO and database connectivity

### 2.4.10 Containerfile
- [x] Multi-stage Rust build
- [x] Non-root user
- [x] `HEALTHCHECK` instruction
- [x] No secrets in image

### 2.4.11 Tests
- [x] Unit tests:
  - [x] File type validation (magic bytes)
  - [x] Quota enforcement
  - [x] Path traversal prevention
  - [x] S3 key generation
  - [x] Thumbnail generation
- [x] Integration tests:
  - [x] Full upload → scan → download flow
  - [x] Quota exceeded scenario
  - [x] Presigned URL generation and usage
  - [x] Virus detection (mock ClamAV)
- [x] Security tests:
  - [x] Upload malicious file types (executables, scripts)
  - [x] Filename path traversal (../../etc/passwd)
  - [x] XXE attacks in XML files
  - [x] Zip bomb detection
  - [x] SSRF via file URLs
  - [x] Tenant isolation (can't access other tenant's files)
  - [x] Large file DoS (quota enforcement)

### 2.4.12 Documentation
- [x] `README.md` — architecture, MinIO setup, virus scanning setup
- [x] `libs/contracts/file-service.yaml` — OpenAPI spec
- [x] Client-side upload guide (using presigned URLs)
- [x] Supported file types documentation
- [x] Quota management guide

---

## 2.5 Audit Service (Go/Gin — Port 8085) ✅

Immutable audit log with event capture, hash chaining, retention policies, and compliance exports.

### 2.5.1 Scaffold & Configuration ✅
- [x] Run `scripts/create-service.sh audit-service`
- [x] Create `internal/config/config.go`:
  - [x] Server config
  - [x] Database and Redis config
  - [x] Retention config (default days, per-event-type overrides)
  - [x] Export config (max export size, allowed formats)
  - [x] Compliance mode flag (enables hash chaining)
- [x] Create `config.yaml`
- [x] Add `replace` directive for `libs/go`
- [x] Verify `go build ./...` compiles

### 2.5.2 Database Migrations ✅
- [x] `migrations/001_create_audit_logs_table.up.sql`
  - Columns: `id`, `tenant_id`, `event_type`, `event_category` (enum: auth/data/system/security/compliance), `actor_id`, `actor_type` (enum: user/system/api_key), `resource_type`, `resource_id`, `action`, `outcome` (enum: success/failure), `ip_address`, `user_agent`, `metadata` (JSONB), `timestamp`, `previous_hash`, `current_hash`
  - Purpose: append-only immutable audit trail
  - Indexes: `(tenant_id, timestamp DESC)`, `(tenant_id, event_type)`, `(tenant_id, resource_id)`, `(actor_id)`, `(current_hash)`
  - PostgreSQL trigger `prevent_audit_modification` — blocks UPDATE/DELETE
- [x] `migrations/001_create_audit_logs_table.down.sql`
- [x] `migrations/002_create_retention_policies_table.up.sql`
  - Columns: `id`, `tenant_id`, `event_type`, `retention_days`, `is_active`, `created_at`, `updated_at`
  - Index: `(tenant_id, event_type)` UNIQUE
- [x] `migrations/002_create_retention_policies_table.down.sql`
- [x] `migrations/003_create_audit_exports_table.up.sql`
  - Columns: `id`, `tenant_id`, `requested_by`, `start_date`, `end_date`, `format` (enum: csv/json), `status` (enum: pending/processing/completed/failed), `file_id`, `error_message`, `created_at`, `completed_at`
  - Index: `(tenant_id, created_at)`
- [x] `migrations/003_create_audit_exports_table.down.sql`
- [x] Verify migrations run

### 2.5.3 Domain Models ✅
- [x] `internal/models/audit_log.go`
  - [x] `AuditLog` struct (matches DB schema)
  - [x] `SearchRequest` with filters (event_type, actor, resource, date range)
  - [x] `AuditLogResponse`
  - [x] `EventCategory` enum
  - [x] `ActorType` enum
  - [x] `Outcome` enum
  - [x] `VerificationReport` and `Statistics` structs
- [x] `internal/models/retention_policy.go`
  - [x] `RetentionPolicy` struct
  - [x] `CreatePolicyRequest`, `UpdatePolicyRequest`
  - [x] `PolicyResponse`
- [x] `internal/models/export.go`
  - [x] `AuditExport` struct
  - [x] `CreateExportRequest`
  - [x] `ExportResponse`
  - [x] `ExportStatus` enum

### 2.5.4 Repository Layer ✅
All queries tenant-scoped. NO UPDATE or DELETE on audit_logs table.

- [x] `internal/repository/postgres/audit_repo.go`
  - [x] `Create(ctx, log)` — append-only insert with hash chaining
  - [x] `GetByID(ctx, id)` — tenant-scoped
  - [x] `Search(ctx, filter)` — advanced search with pagination (event_type, actor, resource, date range, outcome)
  - [x] `GetLastHash(ctx, tenant_id)` — for hash chaining
  - [x] `GetLogsForVerification(ctx, start, end)` — verify integrity
  - [x] `GetStatistics(ctx)` — event counts by type, category, outcome
  - [x] NO update or delete methods
- [x] `internal/repository/postgres/retention_repo.go`
  - [x] `Create(ctx, policy)` — tenant-scoped
  - [x] `GetByEventType(ctx, eventType)` — tenant-scoped
  - [x] `List(ctx)` — tenant-scoped
  - [x] `Update(ctx, id, fields)`
  - [x] `Delete(ctx, id)`
- [x] `internal/repository/postgres/export_repo.go`
  - [x] `Create(ctx, export)` — tenant-scoped
  - [x] `GetByID(ctx, id)` — tenant-scoped
  - [x] `List(ctx)` — tenant-scoped with pagination
  - [x] `UpdateStatus(ctx, id, status, fileID, error)`
- [x] No Redis caching (audit logs must be authoritative source)
- [ ] Unit tests for repositories

### 2.5.5 Service Layer ✅

#### Core Services
- [x] `internal/service/audit_service.go`
  - [x] `CreateLog(ctx, log)` — append audit entry with hash chaining:
    1. Load last hash for tenant
    2. Generate current hash (SHA-256 of: previous_hash + log data)
    3. Insert with previous_hash and current_hash
    4. Return log ID
  - [x] `Search(ctx, filter)` — with pagination
  - [x] `GetByID(ctx, id)` — single log entry
  - [x] `VerifyIntegrity(ctx, start, end)` — verify hash chain, publishes `audit.chain_broken` on failure
  - [x] `GetStatistics(ctx)` — event counts by type, category, outcome

- [x] `internal/service/retention_service.go`
  - [x] `GetPolicies(ctx)` — tenant-scoped
  - [x] `CreatePolicy(ctx, req)` — validate retention days (min 30, max 2555)
  - [x] `UpdatePolicy(ctx, id, req)`
  - [x] `DeletePolicy(ctx, id)`
  - [ ] Background job: `CleanupExpiredLogs()` — placeholder for File Service integration

- [x] `internal/service/export_service.go`
  - [x] `CreateExport(ctx, req)` — queue export job
  - [x] `GetExport(ctx, id)` — check export status
  - [x] `ListExports(ctx)` — tenant-scoped
  - [ ] Background worker: `ProcessExport(exportID)` — placeholder for File Service integration

- [x] `internal/consumer/event_consumer.go`
  - [x] Subscribe to ALL Redis Streams events (auth-events, org-events, notification-events)
  - [x] `HandleAllEvents` — wildcard handler maps any event to audit log entry
  - [x] `categorizeEvent` — maps event type prefix to category (auth/data/system/security/compliance)
  - [x] `parseEventData` — extracts actor, resource, metadata from event payload
  - [x] Call `audit_service.CreateLog()` for each event

#### Hash Chain Implementation
- [x] `internal/service/hash_chain.go`
  - [x] `GenerateHash(prevHash, logData)` — SHA-256
  - [x] `VerifyHash(log)` — recalculate and compare
  - [x] `VerifyChain(logs)` — verify entire sequence
  - [x] Use crypto/sha256 from Go stdlib
  - [x] Log data format: `{tenant_id}|{timestamp}|{event_type}|{actor_id}|{resource_id}|{action}|{outcome}`

### 2.5.6 HTTP Handlers ✅
- [x] `internal/handlers/audit_handler.go`
  - [x] `GET /api/v1/audit/logs` — search/filter audit logs (tenant-scoped, paginated)
  - [x] `GET /api/v1/audit/logs/:id` — get single log entry
  - [x] `GET /api/v1/audit/stats` — event statistics
  - [x] `POST /api/v1/audit/verify` — verify hash chain integrity (start_date, end_date)
- [x] `internal/handlers/export_handler.go`
  - [x] `POST /api/v1/audit/logs/export` — create export job
  - [x] `GET /api/v1/audit/logs/export/:id` — get export status
  - [x] `GET /api/v1/audit/logs/exports` — list all exports
- [x] `internal/handlers/retention_handler.go` (admin only)
  - [x] `GET /api/v1/audit/retention-policies` — list policies
  - [x] `POST /api/v1/audit/retention-policies` — create policy
  - [x] `PUT /api/v1/audit/retention-policies/:id` — update policy
  - [x] `DELETE /api/v1/audit/retention-policies/:id` — delete policy
- [x] NO CREATE endpoint for audit logs (only via event consumer)
- [x] NO UPDATE or DELETE endpoints for audit logs (immutable)

### 2.5.7 API Route Registration ✅
- [x] `api/routes.go` — register routes with middleware
  - [x] Recovery, logging, tenant, metrics
  - [x] GatewayAuth middleware (all routes protected)
  - [x] RBAC (retention policies require org_owner/org_admin)
- [x] Public routes: `/health`, `/metrics`
- [x] Protected routes: all audit endpoints
- [x] Admin routes: retention policy management

### 2.5.8 Redis Streams Events ✅
Subscribe to events from ALL services:
- [x] `auth-events` (`user.*`, `auth.*`) — auto-capture and log
- [x] `org-events` (`org.*`) — auto-capture and log
- [x] `notification-events` (`notification.*`) — auto-capture and log

Publish own events:
- [x] `audit.chain_broken` — integrity violation detected (verified working)
- [ ] `audit.export_completed` — export ready for download (pending File Service)
- [ ] `audit.logs_archived` — logs moved to cold storage (pending retention cleanup)

### 2.5.9 Observability ✅
- [x] OpenTelemetry tracing (optional, configured via config)
- [x] Prometheus metrics:
  - [x] `audit_logs_total` (event_type, outcome)
  - [x] `audit_events_consumed_total` (source_service)
  - [x] `audit_chain_verification_duration_seconds`
- [x] Health check with Postgres, Redis, and event consumer status

### 2.5.10 Containerfile ✅
- [x] Multi-stage Go build (golang:1.25-alpine → alpine:3.21.2)
- [x] Non-root user (`appuser`)
- [x] Pinned base image
- [x] `HEALTHCHECK` instruction

### 2.5.11 Tests (partial)
- [x] Unit tests:
  - [x] Hash chain generation (deterministic, previous hash affects output)
  - [x] Hash chain verification (valid chain, tampered detection, broken chain)
  - [x] Event categorization (auth, data, system, security, compliance)
- [ ] Integration tests:
  - [ ] Full event capture flow
  - [ ] Hash chain integrity over multiple inserts
  - [ ] Export job processing
  - [ ] Retention cleanup
- [ ] Security tests:
  - [ ] Attempt to update audit log (should fail — DB trigger enforced)
  - [ ] Attempt to delete audit log (should fail — DB trigger enforced)
  - [ ] Hash tampering detection
  - [ ] Tenant isolation

### 2.5.12 Documentation (partial)
- [x] `README.md` — architecture, compliance features, hash chaining, API, events
- [ ] `libs/contracts/audit-service.yaml` — OpenAPI spec
- [ ] Compliance guide (SOC 2, GDPR, HIPAA considerations)
- [ ] Hash chain verification guide

---

## 2.6 Phase 2 Integration

Bring all Phase 2 services together with Phase 1 core platform.

### 2.6.1 Compose Stack (partial — Track B services done)
- [x] Update `deploy/podman/compose.services.yml`:
  - [x] Add `notification-service` container (port 8082)
  - [x] Add `billing-service` container (port 8083)
  - [x] Add `file-service` container (port 8084)
  - [x] Add `audit-service` container (port 8085)
  - [x] Configure service dependencies (wait for DB, Redis)
  - [x] Apply security hardening (cap_drop ALL, no-new-privileges, mem_limit, cpus)
  - [x] Add healthchecks for new services
- [x] Update `deploy/podman/compose.core.yml`:
  - [x] Add notification-service and audit-service (internal only, expose not ports)
  - [x] Gateway depends on notification-service and audit-service (service_healthy)
  - [x] Gateway env vars: NOTIFICATION_BASE_URL, AUDIT_BASE_URL
- [x] Update `deploy/podman/prometheus/prometheus.yml` — per-service scrape targets
- [x] Update `.env.example` with Billing/File service environment variables
- [x] Add ClamAV to `deploy/podman/compose.base.yml`
- [x] Verify containers start and all healthchecks pass

### 2.6.2 API Gateway Integration (partial — Track B services done)
- [x] Update API Gateway to proxy Track B services:
  - [x] `/api/v1/notifications/*` → notification-service:8082
  - [x] `/api/v1/billing/*` → billing-service:8083
  - [x] `/api/v1/files/*` → file-service:8084
  - [x] `/api/v1/audit/*` → audit-service:8085
- [x] Gateway config: `notificationBaseUrl`, `auditBaseUrl` (optional, dynamic availability)
- [x] Module gating routes updated for notification and audit prefixes
- [x] Add service health checks to Gateway aggregated health endpoint (Track A)
- [x] Update rate limits (per-service tiers)

### 2.6.3 Makefile Updates (partial — Track B targets done)
- [x] Add targets:
  - [x] `services-up-notification` — start Notification Service
  - [x] `services-up-audit` — start Audit Service
  - [x] `test-notification` — run Notification Service tests
  - [x] `test-audit` — run Audit Service tests
  - [x] `test-go` — run all Go tests (libs + all services)
- [x] Update `make test` to include notification-service and audit-service
- [x] Update `make lint` to include notification-service and audit-service
- [ ] Add `full-up`, `full-down`, `full-reset` targets (after all services complete)

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
- [ ] Notification Service (Go) — all channels working, templates rendering, events consumed
- [ ] Billing Service (Rust) — Stripe integration working, subscriptions, invoicing, usage tracking
- [ ] File Service (Rust) — MinIO storage, virus scanning, quotas, thumbnails
- [ ] Audit Service (Go) — event capture, hash chaining, exports, retention

### Integration
- [ ] All 7 services (Auth, Org, Gateway, Notification, Billing, File, Audit) running via Compose
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
- [ ] `make full-up` starts infra + all 6 services + gateway cleanly
- [ ] All services containerized and healthy
- [ ] Service health monitoring working
- [ ] Circuit breakers functioning

### Git Tag
- [ ] **Git tag: `m2-full-module-suite`** — committed and pushed

---

## Notes & Recommendations

### Build Order (Optimized for Parallelism)
1. **Track A — Rust**: Start Shared Rust Crate (`libs/rust/`) first → then Billing + File services (Rust/Axum)
2. **Track B — Go (can start immediately, no Rust dependency)**: Notification + Audit services (Go/Gin) use existing `libs/go/pkg/`
3. **Integrate incrementally**: Add one service at a time to the gateway
4. **Test continuously**: Run integration tests after each service addition
5. **Track A and B are fully independent** — maximize parallelism

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
- ✅ All 7 services deployed and healthy (Auth, Org, Gateway + Notification, Billing, File, Audit)
- ✅ Module gating enforced by Gateway
- ✅ Tenant isolation proven across all services
- ✅ Security scans clean (SAST + DAST)
- ✅ Event-driven architecture fully functional
- ✅ Audit trail capturing all service events
- ✅ Documentation complete and accurate
- ✅ M2 milestone tagged in Git

**Estimated timeline**: 8 weeks (parallel development reduces time vs sequential)

Ready to proceed to Phase 3: Analytics, Frontend, and Production Hardening! 🚀

## 2.8 Service Integration & Event Consumers

> **Status**: 🟡 IN PROGRESS (Phase 1 + Track B consumers done, Track A pending)
> **Goal**: Implement missing event consumers and verify cross-service workflows.

### 2.8.1 Organization Service Consumers (Go) — ✅ COMPLETED (Phase 1)
- [x] `internal/consumer/consumer.go` — Consumer manager with event type routing
- [x] `internal/consumer/user_consumer.go` — Handles user.created events
  - [x] Implement `HandleUserCreated` handler
  - [x] Logic: Check if user is first in tenant → Create default organization (idempotent)
  - [x] Logic: Publish org.created event after creation
  - [x] DLQ support, pending recovery, metrics, health check
- [x] Logic: Send welcome notification (via Notification Service — verified: `user.created` → in-app welcome notification)

### 2.8.2 Auth Service Consumers (Go) — Pending (Phase 2)
- [ ] `internal/service/event_consumer.go`
  - [ ] Implement `ConsumeOrgDeleted` handler
  - [ ] Logic: Deactivate all users in deleted organization (or reassign)

### 2.8.3 End-to-End Integration Testing
- [x] **Workflow 1: User Registration** — ✅ VERIFIED
  - [x] Trigger: `POST /api/v1/auth/register`
  - [x] Verify: `user.created` event published to Redis `auth-events` stream
  - [x] Verify: Org Service consumes event → Creates Organization automatically
  - [x] Verify: `org.created` event published to `org-events` stream
  - [x] Verify: Consumer health check reports UP
  - [x] Verify: Consumer metrics visible in Prometheus
- [ ] **Workflow 2: Subscription Change** — Pending (requires Billing Service)
  - [ ] Trigger: Billing Service update
  - [ ] Verify: `subscription.updated` event published
  - [ ] Verify: Org Service consumes → Updates module access
