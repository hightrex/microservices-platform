# Phase 0: Foundation (Weeks 1-2; ~3-5 days with parallelism)
Focus: Monorepo setup, rules, shared Go lib, infra, scripts, docs. Complete fully before Phase 1.
**Parallelism Tips**: Use 3-5 agents for lib packages (one per pkg), docs, and scripts/infra. Sync via Makefile/tests.
### 0.1 Repository & Structure
- [x] Initialize git repo (`git init`)
- [x] Create `.gitignore` (ignore Go/Rust/Node/IDE files, .env, reports/)
- [x] Create full directory tree (from build_plan.md)
- [x] Create root `README.md` (overview, diagram, quick start)
- [x] Create `.env.example` with all variables
### 0.2 Cursor Rules
- [x] Write `foundation-rules.mdc` (alwaysApply: true; completeness, quality, shared libs, security, multi-tenancy, API standards, DB, infra, testing, communication)
- [x] Write `service-rules.mdc` (globs: services/**, libs/**; split by language: Go, Rust, TS, SQL, compose)
- [x] Write `production-rules.mdc` (alwaysApply: false; activate in Phase 3)
### 0.3 Shared Go Library (`libs/go/`) [PARALLEL: One agent per package]
- [x] Initialize Go module (`go mod init`)
- [x] `pkg/logger/` — zerolog + tests
- [x] `pkg/errors/` — error types/codes + tests
- [x] `pkg/config/` — viper/env vars + tests
- [x] `pkg/database/` — Postgres pooling/migrations + tests
- [x] `pkg/cache/` — Redis with TTL + tests
- [x] `pkg/messaging/` — Redis Streams/DLQ + tests
- [x] `pkg/middleware/` — auth/rate/CORS/recovery/ID/tenant + tests
- [x] `pkg/tenant/` — context/scoped queries + tests
- [x] `pkg/health/` — checks/dependencies + tests
- [x] `pkg/tracing/` — OpenTelemetry spans + tests
- [x] `pkg/metrics/` — Prometheus + tests
- [x] `pkg/httputil/` — client/timeouts/retries + tests
- [x] `pkg/testing/` — helpers/fixtures/mocks + tests
- [ ] All tests: `go test ./...`
### 0.4 Infrastructure (Compose) [PARALLEL with scripts]
- [ ] `compose.base.yml`: Postgres/Redis/MinIO/Jaeger/Prometheus/Grafana
- [ ] `compose.security.yml`: Kali/ZAP/Trivy
- [ ] Verify `podman compose -f compose.base.yml up`

### 0.5 Scripts [PARALLEL with infra]
- [ ] `setup-dev.sh` — dev setup
- [ ] `init-databases.sh` — per-service DBs
- [ ] `create-service.sh` — service scaffold
### 0.6 Root Makefile
- [ ] infra-up/down
- [ ] services-up SVC="..."
- [ ] dev-<service>
- [ ] test/lint
- [ ] security-sast (gosec/gitleaks/semgrep/hadolint/cargo-audit/npm audit)
- [ ] security-dast (ZAP/fuzz)
- [ ] security-all
### 0.7 Security Setup [PARALLEL with Makefile]
- [ ] Install tools
- [ ] `tests/security/` structure/README
- [ ] Semgrep custom rules (tenant_id/raw SQL)
- [ ] Pre-commit hooks (gitleaks)
### 0.8 Documentation [PARALLEL: Agents for each doc]
- [ ] `VISION.md` — overview/diagrams
- [ ] `DEVELOPMENT.md` — workflow/replace/hot-reload
- [ ] `SECURITY.md` — standards/language
- [ ] `MESSAGING.md` — Streams patterns/schema
- [ ] `libs/contracts/` — OpenAPI templates
### 0.9 Verification Gate
- [ ] infra-up clean
- [ ] Go lib build/test
- [ ] Scripts work
- [ ] SAST error-free
- [ ] Commit; tag `m0-foundation-ready`
**Milestone M0**: Foundation ready. Block Phase 1 until complete.