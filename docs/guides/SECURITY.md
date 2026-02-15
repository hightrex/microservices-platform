# Security Guide

## Standards

### Authentication
- JWTs (Access Tokens) for API access.
- API Keys for service-to-service communication.
- MFA supported and encouraged.

### Authorization
- RBAC (Role-Based Access Control) within organizations.
- Tenants are strictly isolated.

### Data Protection
- All sensitive data encrypted at rest and in transit (TLS).
- Secrets management via environment variables / Vault.

### Coding Standards
- **No Raw SQL**: Always use parameterized queries (enforced by semgrep).
- **Tenant Context**: Always use `pkg/tenant.RequireTenant(ctx)` in repository methods. Never query without tenant scoping.
- **No Identity in Bodies**: Never read `tenant_id`, `org_id`, `user_id` from request bodies. Use `middleware.RejectBodyIdentity()`.
- **Input Validation**: Validate all inputs using `pkg/validation.Validate()` before business logic.
- **Security Logging**: Use `pkg/securitylog` for authentication, authorization, and data access events.
- **Rate Limiting**: Follow standards in `docs/guides/RATE_LIMITING.md` (429 + standard headers).
- **Threat Model**: See `docs/architecture/THREAT_MODEL.md` for attack surface and trust boundaries.

---

## Container Hardening

All containers in the platform follow these security controls:

| Control | Description |
|---------|-------------|
| `no-new-privileges` | Prevents privilege escalation inside containers |
| `cap_drop: ALL` | Drops all Linux capabilities; only required caps added back |
| `read_only: true` | Root filesystem is read-only where possible; tmpfs for writable dirs |
| `mem_limit` / `cpus` | Resource limits prevent noisy-neighbor and DoS scenarios |
| `127.0.0.1` port binding | Infra ports not exposed to the network; localhost only |
| Pinned image versions | No `latest` tags; all images use specific version pins |

### Network Segmentation

```
┌──────────────┐    ┌──────────────┐    ┌──────────────┐
│   data-net   │    │ monitoring   │    │  security    │
│  (internal)  │    │    -net      │    │    -net      │
├──────────────┤    ├──────────────┤    ├──────────────┤
│ Postgres     │    │ Jaeger       │    │ ZAP          │
│ Redis        │    │ Prometheus   │    │ Trivy        │
│ MinIO        │    │ Grafana      │    │ Kali         │
└──────┬───────┘    └──────┬───────┘    └──────────────┘
       │                   │
       └─────┬─────────────┘
             │
      ┌──────┴───────┐
      │   app-net    │
      │  (services)  │
      └──────────────┘
```

- **`data-net`** (internal): Postgres, Redis, MinIO. Not directly accessible from host except via bound ports.
- **`monitoring-net`**: Jaeger, Prometheus, Grafana. Accessible for dashboards.
- **`app-net`**: Bridge network for future application services to reach data and monitoring.
- **`security-net`**: Isolated network for security testing tools.

### Redis Authentication
Redis requires password authentication (`--requirepass`). The password is set via the `REDIS_PASSWORD` environment variable. All clients must provide the password to connect.

---

## Security Scanning

### Tools

| Tool | Type | Purpose |
|------|------|---------|
| **Gitleaks** | SAST | Secret detection in source code |
| **Semgrep** | SAST | Custom rules: tenant isolation, SQL injection |
| **Gosec** | SAST | Go-specific security analysis |
| **Hadolint** | SAST | Containerfile/Dockerfile linting |
| **Trivy** | Container | CVE scanning for container images |
| **OWASP ZAP** | DAST | Dynamic web application scanning |
| **Kali Linux** | Pentest | Manual penetration testing environment |

### Running Scans

```bash
# Run all SAST checks
make security-sast

# Scan container images for vulnerabilities
make security-trivy

# Start OWASP ZAP and run baseline scan
make security-dast

# Run everything
make security-all

# Start/stop security tool containers
make security-up
make security-down
```

### Custom Semgrep Rules

Located in `tests/security/sast/rules.yaml`:

- **`missing-tenant-id-in-sql`** — Flags SQL queries missing `tenant_id` (tenant isolation enforcement)
- **`raw-sql-concatenation`** — Flags string concatenation in SQL queries (SQL injection prevention)

### Reports

All scan reports are saved to `tests/security/reports/` (gitignored). Reports are generated in JSON format for machine processing.

## Functional Security Tests

### Auth-Specific Security Tests
These tests verify JWT manipulation, brute-force protection, and password policies.
```bash
# Run auth security tests
cd tests/security/auth
go test -v ./...
```

### Tenant Isolation Tests
These tests verify that data does not leak between tenants.
```bash
# Run tenant isolation scenarios
make test-tenant-isolation
# OR manually:
cd tests/security/tenant-isolation
go test -v ./...
```

