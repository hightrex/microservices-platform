# Threat Model — Microservices Platform

**Last Updated**: 2026-02-14
**Status**: Phase 1 (Core Platform MVP)
**Methodology**: Based on STRIDE + Attack Trees

---

## 1. Attack Surface

### External-Facing Components

| Component | Exposure | Protocol | Notes |
|-----------|----------|----------|-------|
| API Gateway | Internet | HTTPS (443) | Single entry point for all client traffic. Validates JWTs, enforces rate limits, and gates modules. |
| Frontend (SPA) | Internet | HTTPS (443) | Static assets; communicates with API Gateway |

### Internal Components (Not Directly Exposed)

| Component | Exposure | Protocol | Notes |
|-----------|----------|----------|-------|
| Auth Service | Internal (via Gateway) | HTTP | Manages users, credentials, sessions, MFA, and RBAC. |
| Org Service | Internal (via Gateway) | HTTP | Manages tenants, organizations, members, and module configurations. |
| File Storage Service | Internal (via Gateway) | HTTP | Proxies to MinIO (Future) |
| Notification Service | Internal | gRPC / HTTP | Email, push, in-app (Future) |
| Billing Service | Internal (via Gateway) | HTTP | Stripe integration, plan management (Future) |
| Audit Service | Internal | gRPC | Event consumer (Redis Streams) |
| Analytics Service | Internal | gRPC | Event consumer |
| PostgreSQL | Internal (`data-net`) | TCP 5432 | Primary data store |
| Redis | Internal (`data-net`) | TCP 6379 | Cache, sessions, event bus |
| MinIO | Internal (`data-net`) | TCP 9000 | Object storage |

### Development/Monitoring Tools (Not in Production)

| Component | Exposure | Notes |
|-----------|----------|-------|
| Adminer | localhost:8085 | DB admin (dev only) |
| Redis Commander | localhost:8086 | Redis admin (dev only) |
| Grafana | localhost:3001 | Monitoring dashboards |
| Prometheus | localhost:9090 | Metrics collection |
| Jaeger | localhost:16686 | Distributed tracing |
| OWASP ZAP | localhost:8095 | DAST scanning |

---

## 2. Trust Boundaries

```
┌─────────────────────────────────────────────────────────────────────┐
│                         INTERNET (Untrusted)                        │
│                                                                     │
│   [Browser/Mobile Client]  ──→  [CDN/WAF]                         │
│                                                                     │
└──────────────────────────────┬──────────────────────────────────────┘
                               │
                    ═══════════╪═══════════  BOUNDARY 1: Public Edge
                               │
┌──────────────────────────────┼──────────────────────────────────────┐
│                         DMZ / Gateway                               │
│                                                                     │
│   [API Gateway]                                                     │
│     ├── TLS termination                                            │
│     ├── JWT validation (Signature, Exp, Iss)                       │
│     ├── Rate limiting (Redis-backed)                               │
│     ├── Request routing                                            │
│     └── Tenant header injection (X-Tenant-ID, X-User-ID)           │
│                                                                     │
└──────────────────────────────┬──────────────────────────────────────┘
                               │
                    ═══════════╪═══════════  BOUNDARY 2: Service Mesh
                               │
┌──────────────────────────────┼──────────────────────────────────────┐
│                     Internal Service Network                        │
│                                                                     │
│   [Auth Service]  [Org Service]  [Future Services...]              │
│                                                                     │
│   Services trust:                                                   │
│     ✅ Context headers from Gateway (X-Tenant-ID, X-User-ID)        │
│     ✅ Internal traffic on `app-net`                               │
│     ❌ Client-supplied identity in request bodies (Rejected)       │
│                                                                     │
└──────────────────────────────┬──────────────────────────────────────┘
                               │
                    ═══════════╪═══════════  BOUNDARY 3: Data Layer
                               │
┌──────────────────────────────┼──────────────────────────────────────┐
│                        Data Network                                 │
│                                                                     │
│   [PostgreSQL]     [Redis]     [MinIO]                             │
│                                                                     │
│   Access controlled by:                                             │
│     ✅ Network segmentation (data-net, not exposed to host)        │
│     ✅ Credential-based authentication                             │
│     ✅ Per-service database isolation                              │
│     ✅ Row-level tenant scoping (tenant_id in every query)         │
│                                                                     │
└─────────────────────────────────────────────────────────────────────┘
```

### Trust Boundary Rules

| From → To | Trust Level | Enforcement |
|-----------|-------------|-------------|
| Internet → Gateway | **None** | TLS, rate limiting, input validation, JWT, CORS |
| Gateway → Services | **Partial** | JWT validated at gateway, Identity headers injected via trusted middleware |
| Service → Service | **High** | API keys (future), same network transparency |
| Service → Database | **High** | Credentials, network isolation, tenant scoping in every query |
| Service → Redis | **High** | Password auth, network isolation |

---

## 3. Data Classification

### Critical (C4 — Highest Sensitivity)

| Data | Location | Encryption | Access Control |
|------|----------|------------|----------------|
| User passwords | PostgreSQL (auth DB) | bcrypt/argon2 hashed | Auth Service only |
| JWT signing keys | Environment / Vault | N/A (in-memory) | Auth Service & Gateway only |
| API keys | PostgreSQL (auth DB) | SHA-256 hashed | Auth Service only |
| Stripe API keys | Environment / Vault | N/A (in-memory) | Billing Service only |
| Stripe webhook secrets | Environment / Vault | N/A (in-memory) | Billing Service only |

### Confidential (C3)

| Data | Location | Encryption | Access Control |
|------|----------|------------|----------------|
| User PII (email, name) | PostgreSQL (auth/org DB) | At rest (future) | Owner tenant + admins |
| Session tokens | Redis | N/A (in-memory, TTL) | Auth Service |
| MFA secrets (TOTP) | PostgreSQL (auth DB) | Encrypted at rest | Auth Service, owner user |
| Billing info | PostgreSQL (billing DB) | At rest | Billing Service, tenant admins |
| Uploaded files | MinIO | At rest (SSE) | File Service, authorized users |

### Internal (C2)

| Data | Location | Notes |
|------|----------|-------|
| Organization metadata | PostgreSQL (org DB) | Name, plan, settings |
| Audit logs | PostgreSQL (audit DB) | Immutable, append-only |
| Notification records | PostgreSQL (notification DB) | Templates, delivery status |
| Application metrics | Prometheus | Non-sensitive operational data |
| Distributed traces | Jaeger | May contain request paths |

### Public (C1)

| Data | Location | Notes |
|------|----------|-------|
| API documentation | Static files / CDN | OpenAPI specs |
| Health check endpoints | Services | `/health`, `/ready` |
| Frontend static assets | CDN | HTML, CSS, JS |

---

## 4. Authentication Flow

### JWT-Based Authentication

```
           Client                Gateway              Auth Service         PostgreSQL
             │                     │                      │                    │
             │  POST /auth/login   │                      │                    │
             │  {email, password}  │                      │                    │
             │ ──────────────────► │                      │                    │
             │                     │  Forward to auth     │                    │
             │                     │ ──────────────────►  │                    │
             │                     │                      │  Lookup user       │
             │                     │                      │ ──────────────►    │
             │                     │                      │  ◄──────────────   │
             │                     │                      │  Verify password   │
             │                     │                      │                    │
             │                     │                      │  [If MFA enabled]  │
             │                     │                      │  Return MFA token  │
             │                     │ ◄──────────────────  │                    │
             │  MFA required       │                      │                    │
             │ ◄────────────────── │                      │                    │
             │                     │                      │                    │
             │  POST /auth/mfa     │                      │                    │
             │  {mfa_token, code}  │                      │                    │
             │ ──────────────────► │ ──────────────────►  │                    │
             │                     │                      │  Verify TOTP       │
             │                     │                      │                    │
             │                     │                      │  Generate JWT      │
             │                     │                      │  (access + refresh)│
             │                     │                      │  Store session     │
             │                     │                      │ ──────────────►    │
             │                     │ ◄──────────────────  │                    │
             │  {access_token,     │                      │                    │
             │   refresh_token}    │                      │                    │
             │ ◄────────────────── │                      │                    │
             │                     │                      │                    │
             │  GET /api/resource  │                      │                    │
             │  Authorization:     │                      │                    │
             │  Bearer <jwt>       │                      │                    │
             │ ──────────────────► │  Validate JWT        │                    │
             │                     │  Extract tenant_id   │                    │
             │                     │  Set X-Tenant-ID     │                    │
             │                     │  Forward             │                    │
             │                     │ ──────────────── ►   │                    │
```

### JWT Claims

```json
{
  "sub": "user-uuid",
  "tid": "tenant-uuid",
  "org": "org-uuid",
  "roles": ["admin"],
  "iat": 1707830400,
  "exp": 1707834000,
  "iss": "auth-service",
  "aud": "microservices-platform"
}
```

### Token Lifecycle

| Token | Lifetime | Storage | Revocation |
|-------|----------|---------|------------|
| Access Token (JWT) | 15 minutes | Client-side (memory) | Short-lived; no server revocation needed |
| Refresh Token | 7 days | Redis (session store) | Explicit revocation on logout/password change |
| MFA Token | 5 minutes | Redis (ephemeral) | Single-use, auto-expires |
| API Key | No expiry (configurable) | PostgreSQL (hashed) | Manual revocation |

---

## 5. Multi-Tenant Risks

### STRIDE Analysis for Multi-Tenancy

| Threat | Category | Risk | Mitigation | Status |
|--------|----------|------|------------|--------|
| Tenant A reads Tenant B's data | **Information Disclosure** | **CRITICAL** | `tenant_id` in every query (enforced by `RequireTenant()`), semgrep rules, tenant-isolation tests | ✅ Verified |
| Tenant A modifies Tenant B's data | **Tampering** | **CRITICAL** | Same as above + write operations also scoped by tenant | ✅ Verified |
| Tenant A impersonates Tenant B | **Spoofing** | **CRITICAL** | Tenant derived from JWT only (never from request body), `RejectBodyIdentity()` middleware | ✅ Verified |
| Tenant A exhausts shared resources | **Denial of Service** | **HIGH** | Per-tenant rate limiting, resource quotas (billing plan), container resource limits | ✅ Verified |
| Tenant A elevates privileges | **Elevation of Privilege** | **HIGH** | RBAC enforcement, role changes logged as security events, `no-new-privileges` in containers | ✅ Verified |
| Tenant data visible in logs/traces | **Information Disclosure** | **MEDIUM** | Structured logging with PII filtering, trace sampling, log access controls | ✅ Verified |
| Tenant data mixed in shared caches | **Information Disclosure** | **HIGH** | Cache keys include `tenant_id` prefix, TTL enforcement | ✅ Verified |
| Deleted tenant's data remains | **Information Disclosure** | **MEDIUM** | Tenant deletion workflow with cascading data cleanup, audit trail | ✅ Verified |

### Tenant Isolation Controls (Defense in Depth)

```
Layer 1: Gateway           → JWT validation, tenant extraction, rate limiting
Layer 2: Middleware         → RejectBodyIdentity(), Tenant() middleware on every service
Layer 3: Application Logic → RequireTenant() before any DB access
Layer 4: Database           → tenant_id column on every tenant-scoped table
Layer 5: Static Analysis    → Semgrep rules flag queries missing tenant_id
Layer 6: Testing            → Tenant-isolation test suite (tests/security/tenant-isolation)
```

---

## 6. Event Replay Risks

### Overview

The platform uses **Redis Streams** for event-driven communication. Event replay attacks could cause:

- **Duplicate operations** (e.g., double-charging via billing events)
- **State corruption** (e.g., replaying a "user deleted" event after re-creation)
- **Authorization bypass** (e.g., replaying an old "role granted" event)

### Mitigations

| Control | Implementation | Status |
|---------|---------------|--------|
| **Idempotency keys** | Every event MUST have a unique `event_id` (UUID). Consumers MUST track processed IDs. | ✅ Phase 1 (Implemented in `libs/go/messaging`) |
| **Event timestamps** | Events include `created_at`. Consumers reject events older than a threshold (e.g., 5 mins). | ✅ Phase 1 |
| **Consumer groups** | Redis Streams consumer groups ensure each event is processed exactly once per consumer group. | ✅ Phase 0 |
| **Monotonic ordering** | Use Redis Stream IDs (time-based) to detect out-of-order or replayed events. | ✅ Phase 1 |
| **Write-ahead audit log** | All state-changing events written to audit log BEFORE processing. | ✅ Phase 1 |

---

## 7. Additional Threats

### Infrastructure Threats

| Threat | Category | Risk | Mitigation |
|--------|----------|------|------------|
| Container escape | EoP | HIGH | `no-new-privileges`, `cap_drop: ALL`, read-only root FS |
| Secret leakage in logs | ID | MEDIUM | Gitleaks pre-commit hook, structured logging without PII |
| Dependency vulnerabilities | Tampering | MEDIUM | Trivy scanning, pinned image versions, `go.sum` verification |
| DNS rebinding against internal services | Spoofing | LOW | Services bound to `127.0.0.1`, network segmentation |
| DoS via large file uploads | DoS | MEDIUM | Request body limits at gateway, MinIO quotas |

### Application Threats

| Threat | Category | Risk | Mitigation |
|--------|----------|------|------------|
| SQL injection | Tampering | HIGH | Parameterized queries only (pgx), semgrep rule |
| XSS via stored content | Tampering | MEDIUM | Input validation, output encoding, CSP headers |
| CSRF | Spoofing | LOW | JWT in Authorization header (not cookies), SameSite cookies |
| Insecure direct object references (IDOR) | ID | HIGH | Tenant scoping + authorization checks on every endpoint |
| Mass assignment | Tampering | MEDIUM | `RejectBodyIdentity()` middleware, explicit struct binding |

---

## 8. Risk Register Summary

| # | Risk | Severity | Likelihood | Status | Owner |
|---|------|----------|------------|--------|-------|
| R1 | Cross-tenant data leakage | **Critical** | Medium | Mitigated | Platform Team |
| R2 | SQL injection | **High** | Low | Mitigated | Platform Team |
| R3 | Identity spoofing via request body | **High** | Medium | Mitigated | Platform Team |
| R4 | Brute-force authentication | **High** | High | Mitigated (Rate Limiting) | Platform Team |
| R5 | Event replay attacks | **Medium** | Low | Mitigated (Idempotency) | Platform Team |
| R6 | Container escape | **High** | Low | Mitigated | Platform Team |
| R7 | Dependency CVEs | **Medium** | Medium | Mitigated (Trivy/Dependabot) | Platform Team |
| R8 | PII in logs | **Medium** | Medium | Mitigated (Log Filters) | Platform Team |

---

## 9. Review Schedule

- **Phase 0**: Initial threat model (this document) ✅
- **Phase 1**: Updated with Auth/Org/Gateway specific threats ✅
- **Phase 2**: Update after all services, add penetration testing results
- **Quarterly**: Review and update threat model
- **On-demand**: Review after any security incident or major architecture change

---

## 10. Phase 1 Specific Threats (Auth & Identity)

### 10.1 Authentication & Token Management

| Threat | Description | Mitigation | Status |
|--------|-------------|------------|--------|
| **Token Theft (XSS)** | Attacker steals JWT from local storage/cookies via XSS. | **Short-lived Access Tokens (15m)**: Limits window of opportunity. <br> **HttpOnly Cookies (Recommended)**: Store refresh tokens in HttpOnly cookies to prevent JS access. | ✅ Partially Mitigated |
| **Token Replay** | Attacker intercepts valid JWT and replays it. | **TLS Usage**: All traffic over HTTPS. <br> **Token Expiry**: Short lifespan. | ✅ Mitigated |
| **Signing Key Leak** | Attacker obtains JWT signing key and mints fake tokens. | **Key rotation**: Support for key rotation. <br> **Secret Management**: Keys injected via env vars, not committed to code. | ✅ Mitigated |
| **Algorithm Confusion** | Attacker changes JWT alg to `None` or `HS256` (when `RS256` expected). | **Strict Validation**: Gateway enforces specific signing algorithm. | ✅ Mitigated |

### 10.2 Authorization & RBAC

| Threat | Description | Mitigation | Status |
|--------|-------------|------------|--------|
| **Privilege Escalation** | User modifies their own role ID in request body. | **RejectBodyIdentity Middleware**: Ignores/rejects body fields for identity. <br> **Trusted Headers**: Only `X-User-Roles` from Gateway is trusted. | ✅ Verified |
| **Horizontal Escalation** | User accesses another user's data within same tenant. | **Owner Checks**: Endpoints verify `userID` matches token `sub`. | ✅ Verified |
| **Tenant Hopping** | User accesses another tenant's data. | **Tenant Middleware**: Enforces `X-Tenant-ID` matches token `tid`. <br> **DB Scoping**: All queries filter by `tenant_id`. | ✅ Verified |

### 10.3 Gateway & API Security

| Threat | Description | Mitigation | Status |
|--------|-------------|------------|--------|
| **Rate Limit Bypass** | Attacker rotates IPs to bypass rate limits. | **Authenticated Limits**: Limits applied per `org_id` or `user_id`, not just IP. | ✅ Mitigated |
| **Module Bypass** | Attacker accesses disabled module endpoints. | **Module Gating Middleware**: Checks Redis cache for enabled modules before proxying. | ✅ Verified |
| **Header Spoofing** | Attacker sends `X-Tenant-ID` header directly. | **Header Overwrite**: Gateway unconditionally overwrites these headers from verified JWT claims. | ✅ Verified |

