# Phase 0: Foundation Setup (CHECKLIST)

> **Status**: ✅ COMPLETE — committed and pushed as `m0-foundation-ready`
> **Last verified:** 2026-02-13

## 0.1 Shared Go Library (`libs/go`)
### Structure & Core
- [x] `go.mod` setup (workspace or module)
- [x] `pkg/errors/` — standardized error types (HTTP mapped)
- [x] `pkg/logger/` — structured logging (zerolog/slog)
- [x] `pkg/config/` — configuration loading (Viper/Env)
- [x] `pkg/tenant/` — tenant context propagation (Middleware)
- [x] `pkg/middleware/` — Gin middleware (Auth placeholder, logging, recovery)

### Data & Infrastructure Drivers
- [x] `pkg/database/` — Postgres connection (pgx pool) + migration runner
- [x] `pkg/cache/` — Redis client + TTL enforcement
- [x] `pkg/messaging/` — Redis Streams consumer/producer wrapper
- [x] `pkg/health/` — health check framework with dependency status
- [x] `pkg/tracing/` — OpenTelemetry setup and span helpers
- [x] `pkg/metrics/` — Prometheus metrics registration
- [x] `pkg/httputil/` — HTTP client with tracing, timeouts, retries
- [x] `pkg/testing/` — test helpers, fixtures, mock builders

### Security Guardrails (Code-Level)
- [x] `pkg/tenant/repository.go` — `RequireTenant()`, `TenantScope`, `NewScope()`
- [x] `pkg/middleware/identity_guard.go` — `RejectBodyIdentity()` middleware
- [x] `pkg/validation/validation.go` — input validation (go-playground/validator)
- [x] `pkg/securitylog/securitylog.go` — structured security event logging (25+ event types)

## 0.4 Infrastructure (Compose)
- [x] `compose.base.yml`: Postgres 16, Redis 7, MinIO, Jaeger, Prometheus, Grafana
- [x] `compose.security.yml`: Kali, ZAP, Trivy
- [x] `compose.dev.yml`: Adminer, Redis Commander
- [x] Verify `infra-up` works clean (Prometheus fixed)

## 0.5 Scripts
- [x] `setup-dev.sh`
- [x] `init-databases.sh`
- [x] `create-service.sh`
- [x] `manage-infra.sh`

## 0.6 Root Makefile
- [x] infra-up/down/core-up
- [x] services-up
- [x] test/lint
- [x] security checks
- [x] test-tenant-isolation

## 0.7 Security Setup
- [x] Install gosec, gitleaks, semgrep, hadolint
- [x] Create `tests/security/` structure with README
- [x] Write semgrep rules (missing tenant_id, raw SQL, identity-from-body)
- [x] Set up pre-commit hooks for gitleaks
- [x] Tenant-isolation test harness + scenarios

## 0.8 Documentation
- [x] `docs/architecture/VISION.md`
- [x] `docs/architecture/THREAT_MODEL.md` — STRIDE analysis, risk register
- [x] `docs/guides/DEVELOPMENT.md`
- [x] `docs/guides/SECURITY.md` — updated with Phase 0 packages
- [x] `docs/guides/MESSAGING.md`
- [x] `docs/guides/RATE_LIMITING.md` — standards, tiers, config schema

## 0.9 Verification Gate
- [x] infra-up clean
- [x] Go lib build/test (15 packages, 0 failures)
- [x] Tenant isolation tests pass (4 tests)
- [x] Scripts work
- [x] SAST error-free
- [x] **Milestone M0**: Ready.