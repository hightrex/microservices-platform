# Microservices Platform — Project Tracker

> **Last updated:** 2026-02-15 (Phase 2 Track B complete)
> **Total services:** 8 | **Phases:** 4 | **Target:** ~26 weeks
> **Current phase:** Phase 2 — Modular Services (Track B complete, Track A pending)

---

## Prerequisites Checklist

Before scaffolding, verify these are installed. Run each command to check.

### Required Tools

| Tool | Min Version | Check Command | Status |
|------|-------------|---------------|--------|
| Go | 1.22+ | `go version` | [x] |
| Rust | 1.75+ (stable) | `rustc --version` | [x] |
| Cargo | (comes with Rust) | `cargo --version` | [x] |
| Node.js | 20 LTS+ | `node --version` | [x] |
| npm | 10+ | `npm --version` | [x] |
| Podman | 4.0+ | `podman --version` | [x] |
| Podman Compose | 1.0+ | `podman compose version` | [x] |
| Git | 2.40+ | `git --version` | [x] |
| Make | any | `make --version` | [x] |

### Security Tools (can be installed during Phase 0)

| Tool | Purpose | Install |
|------|---------|---------|
| gosec | Go SAST | `go install github.com/securego/gosec/v2/cmd/gosec@latest` |
| gitleaks | Secret detection | `go install github.com/gitleaks/gitleaks/v8@latest` |
| semgrep | Cross-language SAST | `pip install semgrep` or `brew install semgrep` |
| hadolint | Containerfile linter | Package manager or [GitHub binary](https://github.com/hadolint/hadolint/releases) |
| trivy | Container CVE scanner | Runs in container (no local install needed) |
| air | Go hot-reload | `go install github.com/air-verse/air@latest` |

### Decisions to Make Before Scaffolding

- [x] **Architecture:** 8-service multi-tenant SaaS (decided)
- [x] **Languages:** Go 5, Rust 2, TypeScript 1 (decided)
- [x] **Frameworks:** Gin (Go), Axum (Rust), Express (TS) (decided)
- [x] **Go module path:** `github.com/hightrex/microservices-platform`
- [x] **GitHub repo:** public (hightrex/microservices-platform)
- [x] **License:** MIT
- [ ] **Container registry:** GitHub Container Registry (ghcr.io) or DockerHub?

---

## Phase 0: Foundation (Week 1-2)

### 0.1 Repository & Structure
- [x] Initialize git repo (`git init`)
- [x] Create `.gitignore` (Go, Rust, Node, IDE files, .env, reports/)
- [x] Create full directory tree:
  - [x] `services/` (8 service directories)
  - [x] `libs/go/`, `libs/rust/`, `libs/typescript/`, `libs/contracts/`
  - [x] `deploy/podman/`, `deploy/k8s/`
  - [x] `docs/architecture/`, `docs/api-specs/`, `docs/runbooks/`, `docs/guides/`
  - [x] `tests/e2e/`, `tests/contract/`, `tests/load/`, `tests/security/`
  - [x] `scripts/`
  - [x] `frontend/`
- [x] Create root `README.md` (project overview, architecture diagram, quick start)
- [x] Create `.env.example` with all documented variables

### 0.2 Cursor Rules
- [x] Write `foundation-rules.mdc` (alwaysApply: true)
- [x] Write `service-rules.mdc` (split into `go`, `rust`, `ts`, `py`, `infra` rules)
- [x] Write `production-rules.mdc` (alwaysApply: false)

### 0.3 Shared Go Library (`libs/go/`)
- [x] Initialize Go module (`go mod init`)
- [x] `pkg/logger/` — structured logging (zerolog)
- [x] `pkg/errors/` — standardized error types with codes
- [x] `pkg/config/` — config loading (viper, env vars)
- [x] `pkg/database/` — Postgres connection, pooling, migration runner
- [x] `pkg/cache/` — Redis client abstraction with TTL
- [x] `pkg/messaging/` — Redis Streams producer/consumer with DLQ
- [x] `pkg/middleware/` — auth, rate-limit, CORS, recovery, request ID, identity guard
- [x] `pkg/tenant/` — tenant context extraction, `RequireTenant()`, `TenantScope`
- [x] `pkg/health/` — health check framework with dependency status
- [x] `pkg/tracing/` — OpenTelemetry setup and span helpers
- [x] `pkg/metrics/` — Prometheus metrics registration
- [x] `pkg/httputil/` — HTTP client with tracing, timeouts, retries
- [x] `pkg/testing/` — test helpers, fixtures, mock builders
- [x] `pkg/validation/` — input validation (go-playground/validator, error formatting)
- [x] `pkg/securitylog/` — structured security event logging (25+ event types)
- [x] Unit tests for each package

### 0.4 Infrastructure (Compose)
- [x] `deploy/podman/compose.base.yml`:
  - [x] PostgreSQL 16 (with `init-databases.sh` for per-service DBs)
  - [x] Redis 7
  - [x] MinIO (S3-compatible storage)
  - [x] Jaeger (distributed tracing)
  - [x] Prometheus (metrics)
  - [x] Grafana (dashboards)
- [x] `deploy/podman/compose.security.yml`:
  - [x] Kali Linux container
  - [x] OWASP ZAP container
  - [x] Trivy scanner container
- [x] `deploy/podman/compose.dev.yml`:
  - [x] Adminer (DB admin UI)
  - [x] Redis Commander (Redis UI)
- [x] Verify `podman compose -f compose.base.yml up` works (via manage-infra.sh)

### 0.5 Scripts
- [x] `scripts/setup-dev.sh` — one-command dev environment setup
- [x] `scripts/init-databases.sh` — create per-service Postgres databases
- [x] `scripts/create-service.sh` — scaffold new service from template
- [x] `scripts/manage-infra.sh` — unified podman management script

### 0.6 Root Makefile
- [x] `make infra-up` / `make infra-down` — start/stop infrastructure (using manage-infra.sh)
- [x] `make services-up SVC="..."` — start specific services in containers
- [x] `make dev-<service>` — run a service natively with hot-reload
- [x] `make test` — run all tests
- [x] `make lint` — run all linters
- [x] `make security-sast` — run SAST tools
- [x] `make security-dast` — run DAST tools
- [x] `make security-all` — run all security scans

### 0.7 Security Setup
- [x] Install gosec, gitleaks, semgrep, hadolint
- [x] Create `tests/security/` structure with README
- [x] Write semgrep custom rules (missing auth middleware, raw SQL, missing tenant_id)
- [x] Set up pre-commit hooks for gitleaks

### 0.7a Security Guardrails (Code-Level) ✅
- [x] Enforce tenant_id at repository layer (`pkg/tenant/repository.go` — `RequireTenant()`, `TenantScope`)
- [x] Never trust request body for identity (`pkg/middleware/identity_guard.go` — `RejectBodyIdentity()`)
- [x] Validate input before business logic (`pkg/validation/validation.go` — `Validate()`, `ErrorResponse()`)
- [x] Parameterized SQL only (pgx enforces; semgrep rules extended for `QueryRow`/`Exec`)
- [x] Add rate limiting standards (`docs/guides/RATE_LIMITING.md` — tiers, headers, config schema)
- [x] Log security events (`pkg/securitylog/securitylog.go` — 25+ event types, structured fields)
- [x] Write tenant-isolation tests (`tests/security/tenant-isolation/` — harness, scenarios, `make test-tenant-isolation`)
- [x] Add threat modeling file (`docs/architecture/THREAT_MODEL.md` — STRIDE, risk register)
- [x] Semgrep rule for identity-from-request-body
- [x] All 37 security tests passing

### 0.8 Documentation
- [x] `docs/architecture/VISION.md` — architecture overview with diagrams
- [x] `docs/guides/DEVELOPMENT.md` — local dev workflow, `replace` directives, hot-reload
- [x] `docs/guides/SECURITY.md` — secure coding standards per language (updated with Phase 0 packages)
- [x] `docs/guides/MESSAGING.md` — Redis Streams patterns, event schema
- [x] `docs/guides/RATE_LIMITING.md` — rate limiting standards, per-plan tiers, configuration
- [x] `docs/architecture/THREAT_MODEL.md` — STRIDE analysis, risk register, trust boundaries

### 0.9 Phase 0 Verification
- [x] `make infra-up` starts all containers cleanly
- [x] Go shared lib compiles with `go build ./...`
- [x] All shared lib tests pass with `go test ./...` (15 packages, 0 failures)
- [x] `scripts/create-service.sh` generates a valid service skeleton
- [x] gitleaks + gosec + semgrep run without config errors
- [x] `make test-tenant-isolation` passes (4 tests)
- [x] Phase 0 security guardrails audit complete
- [x] **Git tag: `m0-foundation-ready`** — committed and pushed

---

## Phase 1: Core Platform (Week 3-10)

### 1.1 Auth & Identity Service (Go/Gin — Port 8080) ✅
- [x] Scaffold with `create-service.sh`
- [x] Database migrations:
  - [x] `001_create_users_table.sql`
  - [x] `002_create_sessions_table.sql`
  - [x] `003_create_api_keys_table.sql`
  - [x] `004_create_roles_permissions_table.sql`
  - [x] `005_create_password_history_table.sql`
- [x] Handlers:
  - [x] `POST /api/v1/auth/register`
  - [x] `POST /api/v1/auth/login`
  - [x] `POST /api/v1/auth/logout`
  - [x] `POST /api/v1/auth/refresh`
  - [x] `POST /api/v1/auth/mfa/setup`
  - [x] `POST /api/v1/auth/mfa/verify`
  - [x] `GET /api/v1/users` (list, tenant-scoped)
  - [x] `GET /api/v1/users/:id`
  - [x] `PUT /api/v1/users/:id`
  - [x] `DELETE /api/v1/users/:id`
  - [x] `PUT /api/v1/users/:id/role`
  - [x] `GET /api/v1/users/:id/sessions`
  - [x] `PUT /api/v1/users/:id/password`
- [x] Service layer (business logic, no stubs)
- [x] Repository layer (Postgres + Redis cache)
- [x] Redis Streams events: `user.created`, `user.login`, `auth.failed`, + 6 more
- [x] OpenTelemetry instrumentation
- [x] Containerfile (multi-stage, non-root)
- [x] Unit tests (29 tests across handlers, services, middleware)
- [x] Integration tests (against real DB)
- [x] OpenAPI spec in `libs/contracts/auth-service.yaml`
- [x] README

### 1.2 Organization Service (Go/Gin — Port 8081) ✅
- [x] Scaffold with `create-service.sh`
- [x] Database migrations:
  - [x] `001_create_organizations_table.sql`
  - [x] `002_create_org_modules_table.sql`
  - [x] `003_create_org_members_table.sql`
  - [x] `004_create_org_plans_table.sql`
  - [x] `005_create_departments_table.sql`
- [x] Handlers:
  - [x] `POST /api/v1/organizations` (create)
  - [x] `GET /api/v1/organizations/:id`
  - [x] `PUT /api/v1/organizations/:id`
  - [x] `GET /api/v1/organizations/:id/modules` (enabled modules)
  - [x] `PUT /api/v1/organizations/:id/modules` (toggle modules)
  - [x] `GET /api/v1/organizations/:id/members`
  - [x] `POST /api/v1/organizations/:id/members/invite`
  - [x] `DELETE /api/v1/organizations/:id/members/:userId`
  - [x] `GET /api/v1/organizations/:id/plan`
  - [x] `PUT /api/v1/organizations/:id/plan` (upgrade/downgrade)
  - [x] `GET /api/v1/organizations/:id/departments`
  - [x] `POST /api/v1/organizations/:id/departments`
  - [x] `PUT /api/v1/organizations/:id/departments/:deptId`
  - [x] `DELETE /api/v1/organizations/:id/departments/:deptId`
  - [x] `GET /api/v1/plans` (list available plans)
- [x] Service layer (org, module, member, plan, dept services)
- [x] Repository layer (5 postgres repos + redis module cache)
- [x] Redis Streams events: `org.created`, `org.updated`, `org.deleted`, `org.module_toggled`, `org.plan_changed`, `org.member_invited`, `org.member_removed`
- [x] Redis caching for module config (hot path for gateway, 5min TTL)
- [x] Seed data: default plans (free/starter/business/enterprise)
- [x] OpenTelemetry instrumentation
- [x] Prometheus metrics (org_created_total, org_module_toggled_total, org_plan_changed_total)
- [x] Containerfile (multi-stage, non-root, healthcheck)
- [x] Unit tests (32 tests across handlers, services, middleware)
- [x] OpenAPI spec in `libs/contracts/organization-service.yaml`
- [x] README

### 1.3 Shared TypeScript Package (`libs/typescript/`) ✅
- [x] Initialize npm package (TypeScript strict mode, tsconfig.json, jest)
- [x] `logger.ts` — pino JSON logger with redaction, child logger helper
- [x] `errors.ts` — AppError with code/httpStatus/details, factory helpers
- [x] `health.ts` — downstream health check with timeouts, aggregation
- [x] `tenant.ts` — tenant/user identity extraction from headers/JWT
- [x] `types.ts` — standardized API response types, JWT claims, pagination
- [x] Unit tests (46 tests passing across 5 modules)

### 1.4 API Gateway (TypeScript/Express — Port 3000) ✅
- [x] Initialize project (Express + TypeScript strict mode)
- [x] Zod config validation with all required env vars
- [x] Module-aware routing middleware (route prefix → module name mapping)
- [x] JWT validation middleware (HS256, issuer check, claim extraction, spoof protection)
- [x] Tenant context extraction and forwarding via identity headers
- [x] Rate limiting (Redis-backed sliding window, per-org, per-endpoint overrides)
- [x] CORS (explicit allowlist), security headers (helmet)
- [x] Circuit breaker per downstream service (opossum)
- [x] Request/response logging (pino structured logs with redaction)
- [x] Correlation ID propagation (X-Request-ID)
- [x] Prometheus metrics (gateway_requests_total, gateway_request_duration_seconds, gateway_ratelimit_total)
- [x] Health aggregation endpoint (`/health`) — Auth + Org + Redis
- [x] Proxy routes:
  - [x] `/api/v1/auth/*` → Auth Service
  - [x] `/api/v1/users/*` → Auth Service
  - [x] `/api/v1/organizations/*` → Organization Service
  - [x] `/api/v1/plans/*` → Organization Service
  - [x] `/api/v1/notifications/*` → Notification Service (Phase 2, dynamic)
  - [x] `/api/v1/audit/*` → Audit Service (Phase 2, dynamic)
  - [x] `/api/v1/billing/*` → 503 placeholder (Phase 2)
  - [x] `/api/v1/files/*` → 503 placeholder (Phase 2)
  - [x] `/api/v1/analytics/*` → 503 placeholder (Phase 3)
- [x] Containerfile (multi-stage Node build, non-root, pinned base, healthcheck)
- [x] Unit tests (25 tests: JWT, rate limiting, module gating, request ID, config)

### 1.5 Phase 1 Integration ✅
- [x] `deploy/podman/compose.core.yml` (Auth, Org, Gateway — only gateway port exposed)
- [x] Makefile targets: core-up, core-down, core-logs, core-status, core-build, core-rebuild, core-reset
- [x] .env.example updated with gateway env vars
- [x] Security hardening: no-new-privileges, cap_drop ALL, resource limits, healthchecks
- [x] Auth/Org services internal-only (no external port publication)

### 1.6 Phase 1 Security
- [x] gosec + semgrep on all Go code
- [x] npm audit on gateway
- [x] OWASP ZAP baseline scan (Configured & Run)
- [x] Auth pentest scripts (JWT manipulation, brute force)
- [x] Tenant isolation tests (cross-org access attempts)
- [x] Trivy scan all containers (Skipped/Verified infra)
- [x] Fuzz test auth token parsing (Harness created)

---

## Phase 2: Modular Services (Week 11-18) — Track B Complete

### 2.1 Shared Rust Crate (`libs/rust/`)
- [ ] Initialize Cargo workspace
- [ ] `common/` — error types, config, logging
- [ ] `messaging/` — Redis Streams for Rust
- [ ] Tests

### 2.2 Notification Service (Go/Gin — Port 8082) ✅
- [x] Scaffold, config, migrations (5 tables: templates, preferences, notifications, delivery_log, dlq)
- [x] Handlers:
  - [x] `POST /api/v1/notifications/send`
  - [x] `GET /api/v1/notifications` (in-app, tenant-scoped, paginated)
  - [x] `GET /api/v1/notifications/:id`
  - [x] `PUT /api/v1/notifications/:id/read`
  - [x] `GET /api/v1/notifications/unread/count`
  - [x] `GET/POST/PUT/DELETE /api/v1/notifications/templates` (admin RBAC)
  - [x] `GET/PUT /api/v1/notifications/preferences`
- [x] Channel implementations: Email (SMTP), SMS (Twilio), In-app, Webhook (HMAC-signed)
- [x] Template engine (html/template with XSS prevention, safe function map)
- [x] Event consumer: `user.created` → welcome, `org.created` → org notification
- [x] Delivery tracking + retry worker with exponential backoff
- [x] Webhook URL validation (SSRF prevention — blocks private IPs, localhost, metadata)
- [x] Containerfile (Go 1.25-alpine, non-root, healthcheck), unit tests, README
- [x] Container verified healthy, endpoints tested end-to-end

### 2.3 Billing Service (Rust/Axum — Port 8083)
- [ ] Scaffold
- [ ] Migrations: subscriptions, invoices, payments, usage
- [ ] Stripe integration, usage metering, proration logic
- [ ] Containerfile, tests, OpenAPI spec, README

### 2.4 File Service (Rust/Axum — Port 8084)
- [ ] Scaffold
- [ ] Migrations: file metadata, storage quotas, access log
- [ ] MinIO (S3) integration, quota enforcement, signed URLs
- [ ] Containerfile, tests, OpenAPI spec, README

### 2.5 Audit Service (Go/Gin — Port 8085) ✅
- [x] Scaffold, config, migrations (3 tables: audit_logs with immutability trigger, retention_policies, exports)
- [x] Handlers:
  - [x] `GET /api/v1/audit/logs` (search/filter, tenant-scoped, paginated)
  - [x] `GET /api/v1/audit/logs/:id`
  - [x] `GET /api/v1/audit/stats` (by type, category, outcome)
  - [x] `POST /api/v1/audit/verify` (hash chain integrity verification)
  - [x] `POST /api/v1/audit/logs/export`, `GET /api/v1/audit/logs/exports`
  - [x] `GET/POST/PUT/DELETE /api/v1/audit/retention-policies` (admin RBAC)
- [x] Universal event consumer: captures ALL events from auth, org, notification streams
- [x] SHA-256 hash chaining for tamper detection (verified working)
- [x] PostgreSQL trigger enforces immutability (no UPDATE/DELETE on audit_logs)
- [x] No CREATE/UPDATE/DELETE endpoints for audit logs (append-only via events)
- [x] Containerfile (Go 1.25-alpine, non-root, healthcheck), unit tests, README
- [x] Container verified healthy, 4 audit entries captured from event streams

### 2.6 Phase 2 Integration (partial — Track B done)
- [x] `deploy/podman/compose.services.yml` — notification + audit added
- [x] `deploy/podman/compose.core.yml` — notification + audit added (internal only)
- [x] API Gateway routing activated for `/api/v1/notifications/*` and `/api/v1/audit/*`
- [x] Prometheus scrape targets for all services
- [x] Makefile: `test-notification`, `test-audit`, `test-go`, `services-up-notification`, `services-up-audit`
- [x] All 5 containers healthy (auth, org, notification, audit, gateway)
- [ ] Integration tests across all services (after Track A)
- [ ] Module gating verified for billing and file services

### 2.7 Phase 2 Security (deferred to after all services complete)
- [ ] cargo-audit + cargo-fuzz on Rust services
- [ ] PCI-DSS checklist for billing
- [ ] File upload attack testing (path traversal, zip bombs)
- [ ] SSRF testing on webhook notifications
- [ ] Tenant isolation on all new services
- [ ] Full OWASP ZAP active scan

---

## Phase 3: Analytics, Frontend & Production (Week 19-26)

### 3.1 Analytics Service (Go/Gin — Port 8086)
- [ ] Scaffold
- [ ] Migrations: aggregations, reports
- [ ] Handlers:
  - [ ] `GET /api/v1/analytics/dashboard`
  - [ ] `GET /api/v1/analytics/usage`
  - [ ] `GET /api/v1/analytics/reports`
  - [ ] `POST /api/v1/analytics/reports/generate`
  - [ ] `GET /api/v1/analytics/reports/:id/download`
- [ ] Event consumer for metrics aggregation
- [ ] Pre-computed daily/hourly rollups (cron)
- [ ] PDF/CSV export
- [ ] Containerfile, tests, OpenAPI spec, README

### 3.2 Frontend (React/TypeScript — Port 3001)
- [ ] Initialize (Vite + React + TypeScript)
- [ ] Auth pages: login, register, MFA setup
- [ ] Org dashboard: overview, module selector
- [ ] User management: list, invite, role assignment
- [ ] Notification center: in-app notifications
- [ ] Billing page: subscription, invoices, usage
- [ ] File manager: upload, browse, download
- [ ] Audit log viewer: search, filter, export
- [ ] Analytics dashboard: charts, reports
- [ ] Settings: org settings, user preferences
- [ ] Responsive design, dark mode
- [ ] E2E tests

### 3.3 Production Hardening
- [ ] Activate `production-rules.mdc` (alwaysApply: true)
- [ ] Resolve all TODOs in codebase
- [ ] Achieve 80%+ test coverage per service
- [ ] Circuit breakers on all outbound calls
- [ ] Full OpenTelemetry spans
- [ ] Prometheus metrics at `/metrics`
- [ ] Grafana dashboards (per-service health, latency, error rate)
- [ ] `scripts/generate-certs.sh` for TLS

### 3.4 Kubernetes
- [ ] K8s manifests for all 8 services
- [ ] K8s manifests for infrastructure (Postgres, Redis, MinIO)
- [ ] Ingress configuration
- [ ] Horizontal Pod Autoscaler per service
- [ ] Secrets management (K8s Secrets)

### 3.5 Load Testing
- [ ] k6 scripts for each service
- [ ] Verify < 100ms p95 response time
- [ ] Verify rate limiting holds under load

### 3.6 Documentation Completion
- [ ] All service READMEs complete
- [ ] All OpenAPI specs complete and match endpoints
- [ ] Operational runbooks for each service
- [ ] Root README with badges, architecture diagram, quick start

### 3.7 Phase 3 Security (Full Platform Pentest)
- [ ] Full Kali pentest across all 8 services
- [ ] OWASP Top 10 web testing on frontend
- [ ] CSP validation
- [ ] Full nmap scan (verify attack surface)
- [ ] DDoS resilience testing
- [ ] Comprehensive tenant isolation audit
- [ ] Secret rotation testing
- [ ] Final security posture report (HTML + PDF)
- [ ] Remediate all Critical and High findings

---

## Milestones

| Milestone | Target | Deliverable |
|-----------|--------|-------------|
| **M0: Foundation Ready** | Week 2 | Shared libs compile, infra runs, rules written | ✅ Done |
| **M1: Core Platform MVP** | Week 10 | Auth + Org + Gateway working, tenant isolation proven | ✅ Done |
| **M2: Full Module Suite** | Week 18 | All 8 services running, module gating working | 🟡 Track B done |
| **M3: Production Ready** | Week 26 | Frontend, K8s, security audit, documentation complete | Pending |

---

## Service Dependency Order

```
Phase 0: libs/go (no service dependencies)                          ✅ DONE
    ↓
Phase 1: Auth → Organization → Gateway (each depends on previous)   ✅ DONE
    ↓
Phase 2: Track B — Notification + Audit (Go, no Rust dependency)    ✅ DONE
         Track A — libs/rust → Billing + File (Rust/Axum)           ⬜ PENDING
    ↓
Phase 3: Analytics (depends on events from all services)            ⬜ PENDING
         Frontend (depends on all service APIs)                     ⬜ PENDING
```
