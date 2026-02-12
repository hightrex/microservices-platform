# Microservices Platform — Project Tracker

> **Last updated:** 2026-02-12
> **Total services:** 8 | **Phases:** 4 | **Target:** ~26 weeks

---

## Prerequisites Checklist

Before scaffolding, verify these are installed. Run each command to check.

### Required Tools

| Tool | Min Version | Check Command | Status |
|------|-------------|---------------|--------|
| Go | 1.22+ | `go version` | [ ] |
| Rust | 1.75+ (stable) | `rustc --version` | [ ] |
| Cargo | (comes with Rust) | `cargo --version` | [ ] |
| Node.js | 20 LTS+ | `node --version` | [ ] |
| npm | 10+ | `npm --version` | [ ] |
| Podman | 4.0+ | `podman --version` | [ ] |
| Podman Compose | 1.0+ | `podman compose version` | [ ] |
| Git | 2.40+ | `git --version` | [ ] |
| Make | any | `make --version` | [ ] |

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
- [ ] **Go module path:** e.g., `github.com/<username>/microservices-platform`
- [ ] **GitHub repo:** public or private initially?
- [ ] **License:** MIT / Apache 2.0 / AGPL for open-source intent?
- [ ] **Container registry:** GitHub Container Registry (ghcr.io) or DockerHub?

---

## Phase 0: Foundation (Week 1-2)

### 0.1 Repository & Structure
- [ ] Initialize git repo (`git init`)
- [ ] Create `.gitignore` (Go, Rust, Node, IDE files, .env, reports/)
- [ ] Create full directory tree:
  - [ ] `services/` (8 service directories)
  - [ ] `libs/go/`, `libs/rust/`, `libs/typescript/`, `libs/contracts/`
  - [ ] `deploy/podman/`, `deploy/k8s/`
  - [ ] `docs/architecture/`, `docs/api-specs/`, `docs/runbooks/`, `docs/guides/`
  - [ ] `tests/e2e/`, `tests/contract/`, `tests/load/`, `tests/security/`
  - [ ] `scripts/`
  - [ ] `frontend/`
- [ ] Create root `README.md` (project overview, architecture diagram, quick start)
- [ ] Create `.env.example` with all documented variables

### 0.2 Cursor Rules
- [ ] Write `foundation-rules.mdc` (alwaysApply: true)
- [ ] Write `service-rules.mdc` (globs: services/**, libs/**)
- [ ] Write `production-rules.mdc` (alwaysApply: false)

### 0.3 Shared Go Library (`libs/go/`)
- [ ] Initialize Go module (`go mod init`)
- [ ] `pkg/logger/` — structured logging (zerolog)
- [ ] `pkg/errors/` — standardized error types with codes
- [ ] `pkg/config/` — config loading (viper, env vars)
- [ ] `pkg/database/` — Postgres connection, pooling, migration runner
- [ ] `pkg/cache/` — Redis client abstraction with TTL
- [ ] `pkg/messaging/` — Redis Streams producer/consumer with DLQ
- [ ] `pkg/middleware/` — auth, rate-limit, CORS, recovery, request ID
- [ ] `pkg/tenant/` — tenant context extraction, scoped query helpers
- [ ] `pkg/health/` — health check framework with dependency status
- [ ] `pkg/tracing/` — OpenTelemetry setup and span helpers
- [ ] `pkg/metrics/` — Prometheus metrics registration
- [ ] `pkg/httputil/` — HTTP client with tracing, timeouts, retries
- [ ] `pkg/testing/` — test helpers, fixtures, mock builders
- [ ] Unit tests for each package

### 0.4 Infrastructure (Compose)
- [ ] `deploy/podman/compose.base.yml`:
  - [ ] PostgreSQL 16 (with `init-databases.sh` for per-service DBs)
  - [ ] Redis 7
  - [ ] MinIO (S3-compatible storage)
  - [ ] Jaeger (distributed tracing)
  - [ ] Prometheus (metrics)
  - [ ] Grafana (dashboards)
- [ ] `deploy/podman/compose.security.yml`:
  - [ ] Kali Linux container
  - [ ] OWASP ZAP container
  - [ ] Trivy scanner container
- [ ] `deploy/podman/compose.dev.yml`:
  - [ ] Adminer (DB admin UI)
  - [ ] Redis Commander (Redis UI)
- [ ] Verify `podman compose -f compose.base.yml up` works

### 0.5 Scripts
- [ ] `scripts/setup-dev.sh` — one-command dev environment setup
- [ ] `scripts/init-databases.sh` — create per-service Postgres databases
- [ ] `scripts/create-service.sh` — scaffold new service from template

### 0.6 Root Makefile
- [ ] `make infra-up` / `make infra-down` — start/stop infrastructure
- [ ] `make services-up SVC="..."` — start specific services in containers
- [ ] `make dev-<service>` — run a service natively with hot-reload
- [ ] `make test` — run all tests
- [ ] `make lint` — run all linters
- [ ] `make security-sast` — run SAST tools
- [ ] `make security-dast` — run DAST tools
- [ ] `make security-all` — run all security scans

### 0.7 Security Setup
- [ ] Install gosec, gitleaks, semgrep, hadolint
- [ ] Create `tests/security/` structure with README
- [ ] Write semgrep custom rules (missing auth middleware, raw SQL, missing tenant_id)
- [ ] Set up pre-commit hooks for gitleaks

### 0.8 Documentation
- [ ] `docs/architecture/VISION.md` — architecture overview with diagrams
- [ ] `docs/guides/DEVELOPMENT.md` — local dev workflow, `replace` directives, hot-reload
- [ ] `docs/guides/SECURITY.md` — secure coding standards per language
- [ ] `docs/guides/MESSAGING.md` — Redis Streams patterns, event schema

### 0.9 Phase 0 Verification
- [ ] `make infra-up` starts all containers cleanly
- [ ] Go shared lib compiles with `go build ./...`
- [ ] All shared lib tests pass with `go test ./...`
- [ ] `scripts/create-service.sh` generates a valid service skeleton
- [ ] gitleaks + gosec + semgrep run without config errors

---

## Phase 1: Core Platform (Week 3-10)

### 1.1 Shared TypeScript Package (`libs/typescript/`)
- [ ] Initialize npm package
- [ ] `logger.ts` — structured logging
- [ ] `errors.ts` — standardized error types
- [ ] `health.ts` — health check helpers
- [ ] `tenant.ts` — tenant context extraction
- [ ] Unit tests

### 1.2 Auth & Identity Service (Go/Gin — Port 8080)
- [ ] Scaffold with `create-service.sh`
- [ ] Database migrations:
  - [ ] `001_create_users_table.sql`
  - [ ] `002_create_sessions_table.sql`
  - [ ] `003_create_api_keys_table.sql`
  - [ ] `004_create_roles_permissions_table.sql`
- [ ] Handlers:
  - [ ] `POST /api/v1/auth/register`
  - [ ] `POST /api/v1/auth/login`
  - [ ] `POST /api/v1/auth/logout`
  - [ ] `POST /api/v1/auth/refresh`
  - [ ] `POST /api/v1/auth/mfa/setup`
  - [ ] `POST /api/v1/auth/mfa/verify`
  - [ ] `GET /api/v1/users` (list, tenant-scoped)
  - [ ] `GET /api/v1/users/:id`
  - [ ] `PUT /api/v1/users/:id`
  - [ ] `DELETE /api/v1/users/:id`
  - [ ] `PUT /api/v1/users/:id/role`
  - [ ] `GET /api/v1/users/:id/sessions`
- [ ] Service layer (business logic, no stubs)
- [ ] Repository layer (Postgres + Redis cache)
- [ ] Redis Streams events: `user.created`, `user.login`, `auth.failed`
- [ ] OpenTelemetry instrumentation
- [ ] Containerfile (multi-stage, non-root)
- [ ] Unit tests (all handlers)
- [ ] Integration tests (against real DB)
- [ ] OpenAPI spec in `libs/contracts/auth-service.yaml`
- [ ] README

### 1.3 Organization Service (Go/Gin — Port 8081)
- [ ] Scaffold with `create-service.sh`
- [ ] Database migrations:
  - [ ] `001_create_organizations_table.sql`
  - [ ] `002_create_org_modules_table.sql`
  - [ ] `003_create_org_members_table.sql`
  - [ ] `004_create_org_plans_table.sql`
  - [ ] `005_create_departments_table.sql`
- [ ] Handlers:
  - [ ] `POST /api/v1/organizations` (register)
  - [ ] `GET /api/v1/organizations/:id`
  - [ ] `PUT /api/v1/organizations/:id`
  - [ ] `GET /api/v1/organizations/:id/modules` (enabled modules)
  - [ ] `PUT /api/v1/organizations/:id/modules` (toggle modules)
  - [ ] `GET /api/v1/organizations/:id/members`
  - [ ] `POST /api/v1/organizations/:id/members/invite`
  - [ ] `DELETE /api/v1/organizations/:id/members/:userId`
  - [ ] `GET /api/v1/organizations/:id/plan`
  - [ ] `PUT /api/v1/organizations/:id/plan` (upgrade/downgrade)
  - [ ] `GET /api/v1/organizations/:id/departments`
  - [ ] `POST /api/v1/organizations/:id/departments`
- [ ] Service layer
- [ ] Repository layer
- [ ] Redis Streams events: `org.created`, `org.module_toggled`, `org.plan_changed`
- [ ] Redis caching for module config (hot path for gateway)
- [ ] Seed data: default plans (free/starter/professional/enterprise)
- [ ] OpenTelemetry instrumentation
- [ ] Containerfile
- [ ] Unit + integration tests
- [ ] OpenAPI spec in `libs/contracts/organization-service.yaml`
- [ ] README

### 1.4 API Gateway (TypeScript/Express — Port 3000)
- [ ] Initialize project (Express + TypeScript strict mode)
- [ ] Module-aware routing middleware
- [ ] JWT validation middleware (calls auth service)
- [ ] Tenant context extraction and forwarding
- [ ] Rate limiting (per-org, configurable)
- [ ] CORS, security headers (helmet)
- [ ] Circuit breaker per downstream service
- [ ] Request/response logging
- [ ] Correlation ID propagation
- [ ] OpenTelemetry instrumentation
- [ ] Health aggregation endpoint (`/health`)
- [ ] Proxy routes:
  - [ ] `/api/v1/auth/*` → Auth Service
  - [ ] `/api/v1/organizations/*` → Organization Service
  - [ ] `/api/v1/notifications/*` → Notification Service (Phase 2)
  - [ ] `/api/v1/billing/*` → Billing Service (Phase 2)
  - [ ] `/api/v1/files/*` → File Service (Phase 2)
  - [ ] `/api/v1/audit/*` → Audit Service (Phase 2)
  - [ ] `/api/v1/analytics/*` → Analytics Service (Phase 3)
- [ ] Containerfile
- [ ] Unit tests
- [ ] OpenAPI spec
- [ ] README

### 1.5 Phase 1 Integration
- [ ] `deploy/podman/compose.core.yml` (Auth, Org, Gateway)
- [ ] Cross-service integration tests (Auth ↔ Org ↔ Gateway)
- [ ] API contract tests
- [ ] Verify module gating works (disabled module returns 403)
- [ ] Verify tenant isolation (org A can't see org B)

### 1.6 Phase 1 Security
- [ ] gosec + semgrep on all Go code
- [ ] npm audit on gateway
- [ ] OWASP ZAP baseline scan
- [ ] Auth pentest scripts (JWT manipulation, brute force)
- [ ] Tenant isolation tests (cross-org access attempts)
- [ ] Trivy scan all containers
- [ ] Fuzz test auth token parsing

---

## Phase 2: Modular Services (Week 11-18)

### 2.1 Shared Rust Crate (`libs/rust/`)
- [ ] Initialize Cargo workspace
- [ ] `common/` — error types, config, logging
- [ ] `messaging/` — Redis Streams for Rust
- [ ] Tests

### 2.2 Notification Service (Go/Gin — Port 8082)
- [ ] Scaffold
- [ ] Migrations: templates, preferences, delivery log, DLQ
- [ ] Handlers:
  - [ ] `POST /api/v1/notifications/send`
  - [ ] `GET /api/v1/notifications` (in-app, tenant-scoped)
  - [ ] `PUT /api/v1/notifications/:id/read`
  - [ ] `GET /api/v1/notifications/preferences`
  - [ ] `PUT /api/v1/notifications/preferences`
  - [ ] `POST /api/v1/notifications/templates`
  - [ ] `GET /api/v1/notifications/templates`
- [ ] Channel implementations:
  - [ ] Email (SMTP / SendGrid)
  - [ ] SMS (Twilio)
  - [ ] In-app (database-backed)
  - [ ] Webhook (HTTP POST with retry)
- [ ] Template engine
- [ ] Event consumer (listens to all service events)
- [ ] Delivery tracking + retry with exponential backoff
- [ ] Webhook URL validation (SSRF prevention)
- [ ] Containerfile, tests, OpenAPI spec, README

### 2.3 Billing Service (Rust/Axum — Port 8083)
- [ ] Scaffold
- [ ] Migrations: subscriptions, invoices, payments, usage
- [ ] Handlers:
  - [ ] `GET /api/v1/billing/subscription`
  - [ ] `POST /api/v1/billing/subscription`
  - [ ] `PUT /api/v1/billing/subscription` (upgrade/downgrade)
  - [ ] `DELETE /api/v1/billing/subscription` (cancel)
  - [ ] `GET /api/v1/billing/invoices`
  - [ ] `GET /api/v1/billing/invoices/:id`
  - [ ] `GET /api/v1/billing/usage`
  - [ ] `POST /api/v1/billing/webhook` (Stripe webhook)
- [ ] Stripe integration
- [ ] Usage metering (API calls, storage, users)
- [ ] Invoice PDF generation (via File Service)
- [ ] Proration logic
- [ ] No card data in logs, encryption at rest
- [ ] Containerfile, tests, OpenAPI spec, README

### 2.4 File Service (Rust/Axum — Port 8084)
- [ ] Scaffold
- [ ] Migrations: file metadata
- [ ] Handlers:
  - [ ] `POST /api/v1/files/upload`
  - [ ] `GET /api/v1/files/:id`
  - [ ] `GET /api/v1/files/:id/download`
  - [ ] `DELETE /api/v1/files/:id`
  - [ ] `GET /api/v1/files` (list, tenant-scoped)
  - [ ] `GET /api/v1/files/quota` (storage usage)
- [ ] MinIO (S3) integration
- [ ] Per-org bucket isolation
- [ ] Storage quota enforcement
- [ ] File type validation (magic bytes)
- [ ] Signed URL generation
- [ ] Thumbnail generation for images
- [ ] Containerfile, tests, OpenAPI spec, README

### 2.5 Audit Service (Go/Gin — Port 8085)
- [ ] Scaffold
- [ ] Migrations: audit_logs (append-only), retention_policies
- [ ] Handlers:
  - [ ] `GET /api/v1/audit/logs` (search/filter, tenant-scoped)
  - [ ] `GET /api/v1/audit/logs/export` (CSV/JSON)
  - [ ] `GET /api/v1/audit/stats`
- [ ] Event consumer (auto-captures all service events)
- [ ] Hash chaining for tamper detection
- [ ] Retention policy enforcement
- [ ] No update/delete endpoints (immutable)
- [ ] Containerfile, tests, OpenAPI spec, README

### 2.6 Phase 2 Integration
- [ ] `deploy/podman/compose.services.yml`
- [ ] Integration tests across all services
- [ ] Module gating verified for all new services

### 2.7 Phase 2 Security
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
| **M0: Foundation Ready** | Week 2 | Shared libs compile, infra runs, rules written |
| **M1: Core Platform MVP** | Week 10 | Auth + Org + Gateway working, tenant isolation proven |
| **M2: Full Module Suite** | Week 18 | All 8 services running, module gating working |
| **M3: Production Ready** | Week 26 | Frontend, K8s, security audit, documentation complete |

---

## Service Dependency Order

```
Phase 0: libs/go (no service dependencies)
    ↓
Phase 1: Auth → Organization → Gateway (each depends on previous)
    ↓
Phase 2: libs/rust → Billing + File (parallel, independent)
         Notification + Audit (parallel, depend on event streams from Phase 1)
    ↓
Phase 3: Analytics (depends on events from all services)
         Frontend (depends on all service APIs)
```
