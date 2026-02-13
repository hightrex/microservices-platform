# Phase 1: Core Platform (Weeks 3–10)

> **Status**: 🟡 IN PROGRESS (1.1 complete)
> **Prerequisite**: Phase 0 complete (`m0-foundation-ready`)
> **Milestone**: M1 — Auth + Org + Gateway working, tenant isolation proven
> **Dependency order**: Auth Service → Organization Service → API Gateway → Integration

---

## 1.1 Auth & Identity Service (Go/Gin — Port 8080)

This is the identity backbone. **Build first** — everything else depends on it.

### 1.1.1 Scaffold & Configuration ✅
- [x] Run `scripts/create-service.sh auth-service`
- [x] Fix generated `cmd/main.go` (add missing `net/http` and `fmt` imports, fix logger.Fatal)
- [x] Create `internal/config/config.go` — service config struct (port, DB, Redis, JWT secrets, token TTLs)
- [x] Create `config.yaml` — default dev configuration
- [x] Add `replace` directive in `go.mod` for `libs/go`
- [x] Verify `go build ./...` compiles

### 1.1.2 Database Migrations ✅
All tables include `id` (UUID PK), `created_at`, `updated_at`. Tenant-scoped tables include `tenant_id`.

- [x] `migrations/001_create_users_table.up.sql`
  - Columns: `id`, `tenant_id`, `email`, `password_hash`, `first_name`, `last_name`, `status` (enum: active/disabled/locked), `mfa_enabled`, `mfa_secret`, `failed_login_count`, `locked_until`, `last_login_at`, `created_at`, `updated_at`
  - Indexes: `(tenant_id, email)` UNIQUE, `(tenant_id, status)`
  - Constraint: `email` lowercase, `tenant_id` NOT NULL
- [x] `migrations/001_create_users_table.down.sql`
- [x] `migrations/002_create_sessions_table.up.sql`
  - Columns: `id`, `user_id`, `tenant_id`, `refresh_token_hash`, `ip_address`, `user_agent`, `expires_at`, `created_at`
  - Indexes: `(refresh_token_hash)` UNIQUE, `(user_id, tenant_id)`, `(expires_at)` for cleanup
- [x] `migrations/002_create_sessions_table.down.sql`
- [x] `migrations/003_create_api_keys_table.up.sql`
  - Columns: `id`, `tenant_id`, `user_id`, `name`, `key_hash`, `key_prefix` (first 8 chars for identification), `scopes`, `last_used_at`, `expires_at`, `revoked_at`, `created_at`, `updated_at`
  - Indexes: `(key_hash)` UNIQUE, `(tenant_id)`
- [x] `migrations/003_create_api_keys_table.down.sql`
- [x] `migrations/004_create_roles_permissions_table.up.sql`
  - `roles` table: `id`, `tenant_id`, `name` (enum: org_owner/org_admin/manager/member/viewer), `description`, `is_system` (default roles can't be deleted), `created_at`, `updated_at`
  - `permissions` table: `id`, `resource`, `action` (enum: create/read/update/delete/manage), `description`
  - `role_permissions` join table: `role_id`, `permission_id`
  - `user_roles` join table: `user_id`, `role_id`, `tenant_id`, `granted_by`, `granted_at`
  - Seed default system roles and permissions
- [x] `migrations/004_create_roles_permissions_table.down.sql`
- [x] `migrations/005_create_password_history_table.up.sql`
  - Columns: `id`, `user_id`, `password_hash`, `created_at`
  - Purpose: prevent password reuse (keep last N)
- [x] `migrations/005_create_password_history_table.down.sql`
- [x] Verify migrations run via `database.Migrate()` against local Postgres

### 1.1.3 Domain Models ✅
- [x] `internal/models/user.go` — User, CreateUserRequest, UpdateUserRequest, UserResponse
- [x] `internal/models/session.go` — Session, LoginRequest, LoginResponse, TokenPair
- [x] `internal/models/apikey.go` — APIKey, CreateAPIKeyRequest, APIKeyResponse
- [x] `internal/models/role.go` — Role, Permission, UserRole, AssignRoleRequest
- [x] `internal/models/mfa.go` — MFASetupResponse, MFAVerifyRequest
- [x] All request structs use `validate` tags (using `pkg/validation`)
- [x] All response structs omit sensitive fields (password_hash, mfa_secret)

### 1.1.4 Repository Layer (Postgres + Redis) ✅
Every method MUST call `tenant.RequireTenant(ctx)` or `tenant.NewScope(ctx)`.

- [x] `internal/repository/postgres/user_repo.go`
  - `Create(ctx, user)` — insert with bcrypt-hashed password
  - `GetByID(ctx, id)` — tenant-scoped lookup
  - `GetByEmail(ctx, email)` — tenant-scoped (for login)
  - `List(ctx, filter, page)` — tenant-scoped with pagination
  - `Update(ctx, id, fields)` — tenant-scoped partial update
  - `Delete(ctx, id)` — soft delete (set status=disabled)
  - `IncrementFailedLogins(ctx, id)` — for lockout logic
  - `ResetFailedLogins(ctx, id)` — on successful login
  - `UpdateLastLogin(ctx, id)` — timestamp update
- [x] `internal/repository/postgres/session_repo.go`
  - `Create(ctx, session)` — store refresh token
  - `GetByRefreshToken(ctx, tokenHash)` — for token refresh
  - `ListByUser(ctx, userID)` — active sessions
  - `Delete(ctx, id)` — revoke session (logout)
  - `DeleteAllByUser(ctx, userID)` — revoke all (password change)
  - `CleanupExpired(ctx)` — periodic cleanup
- [x] `internal/repository/postgres/apikey_repo.go`
  - `Create(ctx, apiKey)`
  - `GetByHash(ctx, keyHash)` — for API key validation
  - `List(ctx, userID)` — tenant-scoped
  - `Revoke(ctx, id)` — set revoked_at
- [x] `internal/repository/postgres/role_repo.go`
  - `GetByName(ctx, name)` — lookup role
  - `ListRoles(ctx)` — all tenant roles
  - `AssignRole(ctx, userID, roleID)` — user ↔ role mapping
  - `RevokeRole(ctx, userID, roleID)`
  - `GetUserRoles(ctx, userID)` — with permissions
  - `GetUserPermissions(ctx, userID)` — flattened permission set
- [x] `internal/repository/redis/token_cache.go`
  - Token blacklist (for logout before expiry)
  - Session cache
  - MFA temporary token storage (5min TTL)
- [x] Unit tests for each repository (using `pkg/testing` helpers)

### 1.1.5 Service Layer (Business Logic) ✅
No stubs — real working logic.

- [x] `internal/service/auth_service.go`
  - `Register(ctx, req)` — validate input, check email uniqueness, hash password, create user, assign default role, publish `user.created` event
  - `Login(ctx, req)` — validate credentials, check lockout, check MFA, generate tokens, create session, log security event, publish `user.login` event
  - `Logout(ctx, sessionID)` — revoke session, blacklist access token
  - `RefreshToken(ctx, refreshToken)` — validate, rotate refresh token, issue new access token
  - `SetupMFA(ctx, userID)` — generate TOTP secret, return QR code data
  - `VerifyMFA(ctx, userID, code)` — verify TOTP, enable MFA on user
- [x] `internal/service/user_service.go`
  - `GetByID(ctx, id)` — with role info
  - `List(ctx, filter, page)` — with pagination
  - `Update(ctx, id, req)` — partial update, identity guard (can't change own role)
  - `Delete(ctx, id)` — soft delete, revoke all sessions
  - `ChangePassword(ctx, id, oldPw, newPw)` — verify old, check history, hash new
  - `AssignRole(ctx, userID, roleID)` — authorization check, log event
- [x] `internal/service/token_service.go`
  - `GenerateAccessToken(user)` — JWT with claims (sub, tid, org, roles, iat, exp)
  - `GenerateRefreshToken()` — opaque, stored hashed in DB
  - `ValidateAccessToken(tokenString)` — parse, verify signature, check expiry
  - `ValidateAPIKey(key)` — hash lookup, scope validation
- [x] All service methods use `securitylog.Log()` for security-relevant events
- [x] All service methods use `validation.Validate()` for input validation

### 1.1.6 HTTP Handlers ✅
- [x] `internal/handlers/auth_handler.go`
  - `POST /api/v1/auth/register` — register new user (within tenant)
  - `POST /api/v1/auth/login` — authenticate, return tokens
  - `POST /api/v1/auth/logout` — revoke current session
  - `POST /api/v1/auth/refresh` — refresh access token
  - `POST /api/v1/auth/mfa/setup` — initiate MFA enrollment
  - `POST /api/v1/auth/mfa/verify` — complete MFA verification
- [x] `internal/handlers/user_handler.go`
  - `GET /api/v1/users` — list users (tenant-scoped, paginated)
  - `GET /api/v1/users/:id` — get user by ID
  - `PUT /api/v1/users/:id` — update user
  - `DELETE /api/v1/users/:id` — deactivate user
  - `PUT /api/v1/users/:id/role` — assign/change role
  - `GET /api/v1/users/:id/sessions` — list active sessions
  - `PUT /api/v1/users/:id/password` — change password
- [x] All handlers return standardized responses via `pkg/errors` and `pkg/httputil`
- [x] All handlers validate input via `pkg/validation`

### 1.1.7 API Route Registration & Middleware ✅
- [x] `api/routes.go` — register all routes with middleware stack:
  1. `middleware.Recovery()`
  2. `middleware.RequestLogger()`
  3. `middleware.Tenant()` (on all routes)
  4. `middleware.RejectBodyIdentity()` (on POST/PUT/PATCH)
  5. `metrics.Middleware()`
  6. Auth middleware (JWT validation on protected routes)
  7. RBAC middleware (role check on role-protected routes)
- [x] Public routes: `/health`, `/api/v1/auth/login`, `/api/v1/auth/register`
- [x] Protected routes: everything else

### 1.1.8 Redis Streams Events ✅
- [x] Publish events using `pkg/messaging`:
  - `user.created` — on registration
  - `user.updated` — on profile update
  - `user.deleted` — on deactivation
  - `user.login` — on successful login
  - `user.logout` — on logout
  - `user.role_changed` — on role assignment
  - `auth.failed` — on failed login attempt
  - `auth.mfa_enabled` — on MFA setup
  - `auth.password_changed` — on password change

### 1.1.9 Observability ✅
- [x] OpenTelemetry instrumentation (spans on handlers, DB queries, Redis ops)
- [x] Prometheus metrics: `http_request_duration_seconds` with method/path/status labels
- [x] Health check endpoint with DB + Redis dependency checks

### 1.1.10 Containerfile ✅
- [x] Multi-stage build (builder → runtime)
- [x] Non-root user (`appuser`)
- [x] Pinned Go base image version
- [x] `HEALTHCHECK` instruction
- [x] `no-new-privileges` security option (applied via compose)
- [x] Minimal final image (alpine)

### 1.1.11 Tests ✅
- [x] Unit tests for all handlers (happy path + error cases)
- [x] Unit tests for all service methods
- [x] Unit tests for token generation/validation
- [x] Integration tests against real Postgres (using test containers or local DB)
- [x] Integration tests for login → refresh → logout flow
- [x] MFA enrollment and verification tests
- [x] Password policy tests (min length, history check)
- [x] Account lockout tests
- [x] RBAC permission check tests

### 1.1.12 Documentation ✅
- [x] `services/auth-service/README.md` — API overview, endpoints, authentication flow
- [x] `libs/contracts/auth-service.yaml` — OpenAPI 3.0 spec for all endpoints

---

## 1.2 Organization Service (Go/Gin — Port 8081)

Manages multi-tenancy, module toggles, and billing plans. **Depends on Auth Service** for user identity.

### 1.2.1 Scaffold & Configuration
- [ ] Run `scripts/create-service.sh organization-service`
- [ ] Create `internal/config/config.go` — service config struct
- [ ] Create `config.yaml` — default dev configuration
- [ ] Verify `go build ./...` compiles

### 1.2.2 Database Migrations
- [ ] `migrations/001_create_organizations_table.up.sql`
  - Columns: `id`, `name`, `slug` (URL-friendly unique), `owner_user_id`, `plan` (enum: free/starter/business/enterprise), `status` (active/suspended/deleted), `settings` (JSONB — timezone, locale, branding), `max_users`, `max_storage_bytes`, `created_at`, `updated_at`
  - Indexes: `(slug)` UNIQUE, `(status)`
  - Note: `id` IS the `tenant_id` for downstream services
- [ ] `migrations/001_create_organizations_table.down.sql`
- [ ] `migrations/002_create_org_modules_table.up.sql`
  - Columns: `id`, `org_id`, `module_name` (enum: notifications/billing/file_management/audit_logging/analytics), `enabled`, `config` (JSONB — per-module settings), `enabled_at`, `disabled_at`, `created_at`, `updated_at`
  - Indexes: `(org_id, module_name)` UNIQUE
- [ ] `migrations/002_create_org_modules_table.down.sql`
- [ ] `migrations/003_create_org_members_table.up.sql`
  - Columns: `id`, `org_id`, `user_id`, `role` (from auth roles), `invited_by`, `invited_at`, `joined_at`, `status` (invited/active/removed), `created_at`, `updated_at`
  - Indexes: `(org_id, user_id)` UNIQUE, `(org_id, status)`
- [ ] `migrations/003_create_org_members_table.down.sql`
- [ ] `migrations/004_create_plans_table.up.sql`
  - Columns: `id`, `name`, `display_name`, `max_users`, `max_storage_bytes`, `max_api_calls_per_minute`, `available_modules` (text array), `price_monthly_cents`, `price_annual_cents`, `is_active`, `created_at`, `updated_at`
  - Seed data: free/starter/business/enterprise plans
- [ ] `migrations/004_create_plans_table.down.sql`
- [ ] `migrations/005_create_departments_table.up.sql`
  - Columns: `id`, `org_id`, `name`, `description`, `parent_id` (self-referential for hierarchy), `created_at`, `updated_at`
  - Indexes: `(org_id, name)` UNIQUE, `(org_id, parent_id)`
- [ ] `migrations/005_create_departments_table.down.sql`
- [ ] Verify migrations run cleanly

### 1.2.3 Domain Models
- [ ] `internal/models/organization.go` — Organization, CreateOrgRequest, UpdateOrgRequest, OrgResponse
- [ ] `internal/models/module.go` — Module, ToggleModuleRequest, ModuleResponse
- [ ] `internal/models/member.go` — Member, InviteMemberRequest, MemberResponse
- [ ] `internal/models/plan.go` — Plan, ChangePlanRequest, PlanResponse
- [ ] `internal/models/department.go` — Department, CreateDeptRequest, DeptResponse
- [ ] All request structs use `validate` tags

### 1.2.4 Repository Layer
- [ ] `internal/repository/postgres/org_repo.go`
  - `Create(ctx, org)` — create org, auto-create default modules
  - `GetByID(ctx, id)` — note: org_id IS tenant_id here
  - `GetBySlug(ctx, slug)` — for URL-based access
  - `Update(ctx, id, fields)`
  - `Delete(ctx, id)` — soft delete (set status=deleted)
  - `List(ctx, filter, page)` — admin-only
- [ ] `internal/repository/postgres/module_repo.go`
  - `GetByOrg(ctx, orgID)` — all modules for an org
  - `Toggle(ctx, orgID, moduleName, enabled)` — enable/disable
  - `GetConfig(ctx, orgID, moduleName)` — per-module config
  - `UpdateConfig(ctx, orgID, moduleName, config)`
- [ ] `internal/repository/postgres/member_repo.go`
  - `Add(ctx, orgID, member)` — invite member
  - `List(ctx, orgID, filter, page)` — with pagination
  - `Remove(ctx, orgID, userID)` — set status=removed
  - `UpdateRole(ctx, orgID, userID, role)`
- [ ] `internal/repository/postgres/plan_repo.go`
  - `GetByName(ctx, name)` — look up plan details
  - `List(ctx)` — all available plans
- [ ] `internal/repository/postgres/dept_repo.go`
  - `Create(ctx, orgID, dept)`
  - `List(ctx, orgID)` — with hierarchy
  - `Update(ctx, orgID, deptID, fields)`
  - `Delete(ctx, orgID, deptID)`
- [ ] `internal/repository/redis/module_cache.go`
  - Cache org module config (hot path for gateway lookups)
  - Invalidate on toggle
  - TTL: 5 minutes
- [ ] Unit tests for each repository

### 1.2.5 Service Layer
- [ ] `internal/service/org_service.go`
  - `Create(ctx, req)` — validate, create org, set owner, seed default modules per plan, publish `org.created`
  - `GetByID(ctx, id)` — with module summary
  - `Update(ctx, id, req)` — validate, update, publish `org.updated`
  - `Delete(ctx, id)` — soft-delete, cascade notifications, publish `org.deleted`
- [ ] `internal/service/module_service.go`
  - `GetModules(ctx, orgID)` — return all modules with status (used by gateway)
  - `ToggleModule(ctx, orgID, moduleName, enabled)` — check plan allows module, invalidate cache, publish `org.module_toggled`
  - `GetModuleConfig(ctx, orgID, moduleName)` — per-module settings
- [ ] `internal/service/member_service.go`
  - `Invite(ctx, orgID, req)` — check max users (plan limit), publish `org.member_invited`
  - `List(ctx, orgID, filter, page)` — with roles and status
  - `Remove(ctx, orgID, userID)` — publish `org.member_removed`
- [ ] `internal/service/plan_service.go`
  - `GetCurrent(ctx, orgID)` — org's current plan
  - `Change(ctx, orgID, newPlan)` — validate upgrade/downgrade rules, adjust limits, publish `org.plan_changed`
  - `ListAvailable(ctx)` — all plans

### 1.2.6 HTTP Handlers
- [ ] `internal/handlers/org_handler.go`
  - `POST /api/v1/organizations` — create org (during registration)
  - `GET /api/v1/organizations/:id` — get org details
  - `PUT /api/v1/organizations/:id` — update org settings
- [ ] `internal/handlers/module_handler.go`
  - `GET /api/v1/organizations/:id/modules` — list modules
  - `PUT /api/v1/organizations/:id/modules` — toggle module(s)
  - `GET /api/v1/organizations/:id/modules/:name/config` — module config
- [ ] `internal/handlers/member_handler.go`
  - `GET /api/v1/organizations/:id/members` — list members
  - `POST /api/v1/organizations/:id/members/invite` — invite member
  - `DELETE /api/v1/organizations/:id/members/:userId` — remove member
- [ ] `internal/handlers/plan_handler.go`
  - `GET /api/v1/organizations/:id/plan` — current plan
  - `PUT /api/v1/organizations/:id/plan` — change plan
  - `GET /api/v1/plans` — list all available plans
- [ ] `internal/handlers/dept_handler.go`
  - `GET /api/v1/organizations/:id/departments` — list departments
  - `POST /api/v1/organizations/:id/departments` — create department
  - `PUT /api/v1/organizations/:id/departments/:deptId` — update department
  - `DELETE /api/v1/organizations/:id/departments/:deptId` — delete department
- [ ] All handlers use validation, identity guard, tenant scoping

### 1.2.7 API Route Registration & Middleware
- [ ] `api/routes.go` — same middleware stack as Auth, plus RBAC checks:
  - Org creation: any authenticated user
  - Org settings/modules/plan: `org_owner` or `org_admin` only
  - Member management: `org_admin` or higher
  - Department management: `manager` or higher

### 1.2.8 Redis Streams Events
- [ ] Publish events:
  - `org.created` — on org creation
  - `org.updated` — on settings update
  - `org.deleted` — on org deletion
  - `org.module_toggled` — with module name and new state
  - `org.plan_changed` — with old/new plan
  - `org.member_invited` — with user info
  - `org.member_removed` — with user info

### 1.2.9 Observability
- [ ] OpenTelemetry instrumentation
- [ ] Prometheus metrics: `org_created_total`, `org_module_toggled_total`, `org_plan_changed_total`
- [ ] Health check with DB + Redis checks

### 1.2.10 Containerfile
- [ ] Same hardening standards as Auth Service

### 1.2.11 Tests
- [ ] Unit tests for all handlers and service methods
- [ ] Integration tests: org creation → module toggle → plan change flow
- [ ] Plan limit enforcement tests (max users, max storage)
- [ ] Module config caching/invalidation tests
- [ ] RBAC authorization tests (owner vs admin vs member)

### 1.2.12 Documentation
- [ ] `services/organization-service/README.md`
- [ ] `libs/contracts/organization-service.yaml` — OpenAPI 3.0 spec

---

## 1.3 Shared TypeScript Package (`libs/typescript/`)

Needed by the API Gateway. Build in parallel with Auth/Org services.

### 1.3.1 Initialize
- [ ] `npm init` with TypeScript strict mode
- [ ] Configure `tsconfig.json` — strict, ES2022, module resolution
- [ ] Add dev dependencies: typescript, `@types/node`, jest, ts-jest
- [ ] Create `src/index.ts` — barrel exports

### 1.3.2 Packages
- [ ] `src/logger.ts` — structured logging (pino or winston, JSON format)
- [ ] `src/errors.ts` — standardized error types matching Go `pkg/errors`
- [ ] `src/health.ts` — health check helper (check downstream service health)
- [ ] `src/tenant.ts` — tenant context extraction from JWT/headers
- [ ] `src/types.ts` — shared TypeScript types (API responses, pagination, etc.)

### 1.3.3 Tests
- [ ] Unit tests for each module (jest)
- [ ] 100% type coverage (no `any`)

---

## 1.4 API Gateway (TypeScript/Express — Port 3000)

The public-facing entry point. **Depends on Auth + Org services**.

### 1.4.1 Initialize
- [ ] Initialize Express + TypeScript project
- [ ] Configure strict TypeScript, ESLint, Prettier
- [ ] Add dependencies: express, helmet, cors, http-proxy-middleware, jsonwebtoken, ioredis, zod, pino, opossum (circuit breaker)

### 1.4.2 Core Middleware Stack (in order)
- [ ] Request ID / Correlation ID generation (`X-Request-ID`)
- [ ] Request logging (pino)
- [ ] Security headers (helmet)
- [ ] CORS (configurable origins, no `*`)
- [ ] Request body limit (configurable, default 1MB)
- [ ] JWT validation middleware
  - Extract Bearer token from Authorization header
  - Verify signature using Auth Service's public key/shared secret
  - Extract claims: `sub`, `tid`, `org`, `roles`
  - Set `X-Tenant-ID`, `X-User-ID`, `X-User-Roles` headers for downstream
- [ ] Tenant context extraction and forwarding
- [ ] Module gating middleware
  - Fetch org's enabled modules from Org Service (cached in Redis, 5min TTL)
  - If target module disabled → `403 { code: "MODULE_NOT_ENABLED" }`
  - Map route prefix to module name
- [ ] Rate limiting middleware (per `docs/guides/RATE_LIMITING.md`)
  - Redis-backed sliding window
  - Per-org limits from plan
  - Per-endpoint overrides (auth login: 10/min, register: 5/min)
  - Return `X-RateLimit-*` headers on every response
  - Return `429` with `Retry-After` when exceeded
  - Log rate-limited requests via securitylog pattern
- [ ] Circuit breaker per downstream service (opossum)

### 1.4.3 Proxy Routes
- [ ] `/api/v1/auth/*` → Auth Service (`:8080`)
- [ ] `/api/v1/organizations/*` → Organization Service (`:8081`)
- [ ] `/api/v1/users/*` → Auth Service (`:8080`)
- [ ] `/api/v1/notifications/*` → Notification Service (`:8082`) — Phase 2, route registered but returns 503
- [ ] `/api/v1/billing/*` → Billing Service (`:8083`) — Phase 2
- [ ] `/api/v1/files/*` → File Service (`:8084`) — Phase 2
- [ ] `/api/v1/audit/*` → Audit Service (`:8085`) — Phase 2
- [ ] `/api/v1/analytics/*` → Analytics Service (`:8086`) — Phase 3

### 1.4.4 Health & Observability
- [ ] `GET /health` — aggregate health from all registered services
- [ ] OpenTelemetry instrumentation (trace propagation to downstream)
- [ ] Prometheus metrics: `gateway_requests_total`, `gateway_latency_seconds`, `gateway_ratelimit_total`
- [ ] Request/response logging (sanitized — no auth tokens in logs)

### 1.4.5 Containerfile
- [ ] Multi-stage Node.js build
- [ ] Non-root user
- [ ] Pinned Node base image
- [ ] HEALTHCHECK instruction

### 1.4.6 Tests
- [ ] Unit tests for all middleware (JWT validation, module gating, rate limiting)
- [ ] Unit tests for route mapping and forwarding logic
- [ ] Integration tests with mock upstream services
- [ ] Rate limiting behavior tests (sliding window, burst, per-endpoint)

### 1.4.7 Documentation
- [ ] `services/api-gateway/README.md`
- [ ] `libs/contracts/api-gateway.yaml` — OpenAPI spec (aggregated routes)

---

## 1.5 Phase 1 Integration

### 1.5.1 Compose Stack
- [ ] `deploy/podman/compose.core.yml` — Auth, Org, Gateway containers
  - Network segmentation matching `compose.base.yml` (service-net, data-net)
  - Resource limits, `no-new-privileges`, `cap_drop: ALL`
  - Health checks on all three services
  - Depends-on: postgres, redis from `compose.base.yml`

### 1.5.2 Cross-Service Integration Tests
- [ ] Full registration flow: Gateway → Auth (register) → Auth (login) → Gateway (authenticated request)
- [ ] Org creation flow: Gateway → Auth (authenticated) → Org (create org) → Org (toggle modules)
- [ ] Module gating: Gateway → Org (toggle module off) → Gateway (request to disabled module → 403)
- [ ] Token lifecycle: login → get access token → use token → refresh token → use new token → logout → token rejected
- [ ] Tenant isolation: create two orgs → each org sees only its own data

### 1.5.3 API Contract Tests
- [ ] Auth Service endpoints match `auth-service.yaml` OpenAPI spec
- [ ] Org Service endpoints match `organization-service.yaml` OpenAPI spec
- [ ] Gateway routes match `api-gateway.yaml` OpenAPI spec
- [ ] Response formats match standardized structure

### 1.5.4 Verification Gate
- [ ] `make test` passes for all Phase 1 services
- [ ] `make lint` passes for all Phase 1 services
- [ ] All three services start cleanly in `compose.core.yml`
- [ ] Gateway successfully proxies to Auth and Org services
- [ ] Module gating works (disabled module → 403)
- [ ] Tenant isolation verified (org A can't see org B's data)
- [ ] Rate limiting works (returns 429 with correct headers)

---

## 1.6 Phase 1 Security

### 1.6.1 SAST
- [ ] `gosec` on all Go code (auth-service, organization-service, libs/go) — zero HIGH
- [ ] `semgrep` with custom rules — zero errors
- [ ] `npm audit` on gateway — zero HIGH/CRITICAL
- [ ] `hadolint` on all Containerfiles — zero errors

### 1.6.2 DAST
- [ ] OWASP ZAP baseline scan on Auth Service
- [ ] OWASP ZAP baseline scan on Gateway
- [ ] Trivy scan all Phase 1 containers — zero CRITICAL CVEs

### 1.6.3 Auth-Specific Security Tests
- [ ] JWT manipulation tests (tampered token, expired token, wrong signing key)
- [ ] Brute force protection tests (account lockout after N failures)
- [ ] Session hijacking tests (stolen refresh token detection)
- [ ] RBAC boundary tests (member can't perform admin actions)
- [ ] Password policy enforcement tests
- [ ] MFA bypass attempt tests

### 1.6.4 Tenant Isolation Tests (Service-Level)
Update `tests/security/tenant-isolation/scenarios/` with real service tests:
- [ ] `cross_tenant_read.go` — org A can't read org B's users/orgs/sessions
- [ ] `cross_tenant_write.go` — org A can't modify org B's data
- [ ] `cross_tenant_list.go` — org A's list endpoints return only org A's data
- [ ] `cross_tenant_delete.go` — org A can't delete org B's resources
- [ ] Run all via `make test-tenant-isolation`

### 1.6.5 Fuzz Testing
- [ ] Fuzz test JWT parsing (malformed tokens, oversized payloads)
- [ ] Fuzz test user input validation (registration, login endpoints)
- [ ] Fuzz test API Gateway route matching

### 1.6.6 Security Event Logging Verification
- [ ] Verify `securitylog.Log()` is called on: login failure, permission denied, tenant violation, rate limiting
- [ ] Verify security events contain required fields: `security_event`, `outcome`, `actor_id`, `tenant_id`, `ip`
- [ ] Verify no PII (passwords, tokens) in log output

---

## 1.7 Phase 1 Documentation Completion

- [ ] Update `docs/architecture/THREAT_MODEL.md` — add Auth/Org-specific threats
- [ ] Update `docs/guides/SECURITY.md` — add Phase 1 patterns (JWT, RBAC)
- [ ] Update `README.md` — add Phase 1 quick start instructions
- [ ] All 3 service READMEs complete
- [ ] All OpenAPI specs in `libs/contracts/` complete and matching endpoints

---

## Milestone: M1 — Core Platform MVP

**Criteria for M1 completion:**
- [ ] Auth Service: all endpoints working, tested, documented
- [ ] Org Service: all endpoints working, tested, documented
- [ ] API Gateway: proxying, JWT validation, module gating, rate limiting all working
- [ ] All 3 services containerized and running via `compose.core.yml`
- [ ] Tenant isolation proven across Auth and Org services
- [ ] All SAST/DAST scans passing
- [ ] OpenAPI specs complete and matching
- [ ] Git tag: `m1-core-platform-mvp`
