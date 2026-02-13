# Threat Model — Microservices Platform

**Last Updated**: 2026-02-13
**Status**: Phase 0 (Initial)
**Methodology**: Based on STRIDE + Attack Trees

---

## 1. Attack Surface

### External-Facing Components

| Component | Exposure | Protocol | Notes |
|-----------|----------|----------|-------|
| API Gateway | Internet | HTTPS (443) | Single entry point for all client traffic |
| Frontend (SPA) | Internet | HTTPS (443) | Static assets; communicates with API Gateway |

### Internal Components (Not Directly Exposed)

| Component | Exposure | Protocol | Notes |
|-----------|----------|----------|-------|
| Auth Service | Internal (via Gateway) | gRPC / HTTP | Issues JWTs, manages sessions |
| Org Service | Internal (via Gateway) | gRPC / HTTP | Manages tenants, users, roles |
| File Storage Service | Internal (via Gateway) | HTTP | Proxies to MinIO |
| Notification Service | Internal | gRPC / HTTP | Email, push, in-app |
| Billing Service | Internal (via Gateway) | HTTP | Stripe integration, plan management |
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
│     ├── JWT validation                                             │
│     ├── Rate limiting                                              │
│     ├── Request routing                                            │
│     └── Tenant header injection                                    │
│                                                                     │
└──────────────────────────────┬──────────────────────────────────────┘
                               │
                    ═══════════╪═══════════  BOUNDARY 2: Service Mesh
                               │
┌──────────────────────────────┼──────────────────────────────────────┐
│                     Internal Service Network                        │
│                                                                     │
│   [Auth Service]  [Org Service]  [File Service]  [Billing Service] │
│                                                                     │
│   Services trust:                                                   │
│     ✅ Tenant ID from Gateway headers (X-Tenant-ID)               │
│     ✅ User ID from validated JWT claims                           │
│     ❌ Client-supplied identity in request bodies                  │
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
| Internet → Gateway | **None** | TLS, rate limiting, input validation |
| Gateway → Services | **Partial** | JWT validated at gateway, tenant injected via header |
| Service → Service | **High** | API keys, mTLS (future), same network |
| Service → Database | **High** | Credentials, network isolation, tenant scoping in every query |
| Service → Redis | **High** | Password auth, network isolation |

---

## 3. Data Classification

### Critical (C4 — Highest Sensitivity)

| Data | Location | Encryption | Access Control |
|------|----------|------------|----------------|
| User passwords | PostgreSQL (auth DB) | bcrypt/argon2 hashed | Auth Service only |
| JWT signing keys | Environment / Vault | N/A (in-memory) | Auth Service only |
| API keys | PostgreSQL (auth DB) | SHA-256 hashed | Auth Service only |
| Stripe API keys | Environment / Vault | N/A (in-memory) | Billing Service only |
| Stripe webhook secrets | Environment / Vault | N/A (in-memory) | Billing Service only |

### Confidential (C3)

| Data | Location | Encryption | Access Control |
|------|----------|------------|----------------|
| User PII (email, name) | PostgreSQL (auth/org DB) | At rest (pgcrypto, future) | Owner tenant + admins |
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
             │ ──────────────────► │                      │                    │
             │                     │  Validate JWT        │                    │
             │                     │  Extract tenant_id   │                    │
             │                     │  Set X-Tenant-ID     │                    │
             │                     │  Forward to service  │                    │
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
  "iss": "microservices-platform",
  "aud": "platform-api"
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

| Threat | Category | Risk | Mitigation |
|--------|----------|------|------------|
| Tenant A reads Tenant B's data | **Information Disclosure** | **CRITICAL** | `tenant_id` in every query (enforced by `RequireTenant()`), semgrep rules, tenant-isolation tests |
| Tenant A modifies Tenant B's data | **Tampering** | **CRITICAL** | Same as above + write operations also scoped by tenant |
| Tenant A impersonates Tenant B | **Spoofing** | **CRITICAL** | Tenant derived from JWT only (never from request body), `RejectBodyIdentity()` middleware |
| Tenant A exhausts shared resources | **Denial of Service** | **HIGH** | Per-tenant rate limiting, resource quotas (billing plan), container resource limits |
| Tenant A elevates privileges | **Elevation of Privilege** | **HIGH** | RBAC enforcement, role changes logged as security events, `no-new-privileges` in containers |
| Tenant data visible in logs/traces | **Information Disclosure** | **MEDIUM** | Structured logging with PII filtering, trace sampling, log access controls |
| Tenant data mixed in shared caches | **Information Disclosure** | **HIGH** | Cache keys include `tenant_id` prefix, TTL enforcement |
| Deleted tenant's data remains | **Information Disclosure** | **MEDIUM** | Tenant deletion workflow with cascading data cleanup, audit trail |

### Tenant Isolation Controls (Defense in Depth)

```
Layer 1: Gateway           → JWT validation, tenant extraction, rate limiting
Layer 2: Middleware         → RejectBodyIdentity(), Tenant() middleware
Layer 3: Application Logic → RequireTenant() before any DB access
Layer 4: Database           → tenant_id column on every tenant-scoped table
Layer 5: Static Analysis    → Semgrep rules flag queries missing tenant_id
Layer 6: Testing            → Tenant-isolation test suite
```

---

## 6. Event Replay Risks

### Overview

The platform uses **Redis Streams** for event-driven communication. Event replay attacks could cause:

- **Duplicate operations** (e.g., double-charging via billing events)
- **State corruption** (e.g., replaying a "user deleted" event after re-creation)
- **Authorization bypass** (e.g., replaying an old "role granted" event)

### Attack Scenarios

| Scenario | Impact | Likelihood |
|----------|--------|------------|
| Billing event replayed | Double charge, revenue loss | Low (internal only) |
| "User role granted" replayed | Privilege escalation | Low (internal only) |
| "Account created" replayed | Duplicate accounts | Low |
| "Password reset" replayed | Account takeover | Low (tokens expire) |
| "Org settings changed" replayed | Config corruption | Medium (if streams are compromised) |

### Mitigations

| Control | Implementation | Status |
|---------|---------------|--------|
| **Idempotency keys** | Every event MUST have a unique `event_id` (UUID). Consumers MUST track processed IDs. | Phase 1 |
| **Event timestamps** | Events include `created_at`. Consumers reject events older than a threshold. | Phase 1 |
| **Consumer groups** | Redis Streams consumer groups ensure each event is processed exactly once per consumer group. | Phase 0 ✅ (messaging pkg) |
| **Event signing** | Critical events (billing, role changes) signed with HMAC. Consumers verify signature. | Phase 2 (future) |
| **Monotonic ordering** | Use Redis Stream IDs (time-based) to detect out-of-order or replayed events. | Phase 1 |
| **Write-ahead audit log** | All state-changing events written to audit log BEFORE processing. Enables detection of replay. | Phase 1 |

### Event Schema (Standard Fields)

```json
{
  "event_id": "uuid-v4",
  "event_type": "user.created",
  "tenant_id": "uuid-v4",
  "actor_id": "uuid-v4",
  "created_at": "2026-02-13T09:00:00Z",
  "version": 1,
  "data": { ... },
  "metadata": {
    "source_service": "auth-service",
    "correlation_id": "uuid-v4",
    "idempotency_key": "uuid-v4"
  }
}
```

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
| R1 | Cross-tenant data leakage | **Critical** | Medium | Mitigated (Phase 0 controls) | Platform Team |
| R2 | SQL injection | **High** | Low | Mitigated (pgx + semgrep) | Platform Team |
| R3 | Identity spoofing via request body | **High** | Medium | Mitigated (Phase 0 middleware) | Platform Team |
| R4 | Brute-force authentication | **High** | High | Planned (Phase 1 rate limiting) | Platform Team |
| R5 | Event replay attacks | **Medium** | Low | Planned (Phase 1 idempotency) | Platform Team |
| R6 | Container escape | **High** | Low | Mitigated (hardened containers) | Platform Team |
| R7 | Dependency CVEs | **Medium** | Medium | Mitigated (Trivy scanning) | Platform Team |
| R8 | PII in logs | **Medium** | Medium | Planned (Phase 1 log filtering) | Platform Team |

---

## 9. Review Schedule

- **Phase 0**: Initial threat model (this document) ✅
- **Phase 1**: Update after Auth/Org services implemented
- **Phase 2**: Update after all services, add penetration testing results
- **Quarterly**: Review and update threat model
- **On-demand**: Review after any security incident or major architecture change
