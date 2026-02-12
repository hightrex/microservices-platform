# Phase 0: Foundation Setup (CHECKLIST)

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

## 0.9 Verification Gate
- [x] infra-up clean
- [x] Go lib build/test
- [x] Scripts work
- [x] SAST error-free
- [x] **Milestone M0**: Ready.