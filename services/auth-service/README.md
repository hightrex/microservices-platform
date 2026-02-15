# Auth & Identity Service (Phase 1 Complete)

The authentication and identity backbone for the microservices platform. All other services depend on this for user identity, JWT validation, and RBAC.

## Overview

- **Language:** Go 1.25+ with Gin
- **Port:** 8080
- **Database:** PostgreSQL 16
- **Cache:** Redis 7
- **Events:** Redis Streams

## Architecture

```
cmd/main.go                    # Entry point, wiring
internal/
  config/config.go             # Service configuration
  models/                      # Domain models + request/response types
  handlers/                    # HTTP handlers (Bind -> Validate -> Service -> Respond)
  service/                     # Business logic + repository interfaces
  repository/
    postgres/                  # PostgreSQL implementations
    redis/                     # Redis cache implementations
api/
  routes.go                    # Route registration with middleware
  middleware/                  # JWT auth + RBAC middleware
migrations/                    # SQL migration files (golang-migrate)
```

## Endpoints

### Public (No Authentication)

| Method | Path | Description |
|--------|------|-------------|
| GET | `/health` | Health check (DB + Redis) |
| GET | `/metrics` | Prometheus metrics |
| POST | `/api/v1/auth/register` | Register new user |
| POST | `/api/v1/auth/login` | Authenticate, get tokens |
| POST | `/api/v1/auth/refresh` | Refresh access token |

### Protected (JWT Required)

| Method | Path | Description |
|--------|------|-------------|
| POST | `/api/v1/auth/logout` | Revoke session |
| POST | `/api/v1/auth/mfa/setup` | Initiate MFA enrollment |
| POST | `/api/v1/auth/mfa/verify` | Complete MFA verification |
| GET | `/api/v1/users` | List users (paginated) |
| GET | `/api/v1/users/:id` | Get user by ID |
| PUT | `/api/v1/users/:id` | Update user |
| DELETE | `/api/v1/users/:id` | Deactivate user |
| PUT | `/api/v1/users/:id/role` | Assign role (admin+) |
| GET | `/api/v1/users/:id/sessions` | List active sessions |
| PUT | `/api/v1/users/:id/password` | Change password |

## Authentication Flow

1. **Register** - Create account with email/password within a tenant
2. **Login** - Verify credentials, receive JWT access token + opaque refresh token
3. **Access** - Include `Authorization: Bearer <token>` on all requests
4. **Refresh** - Exchange refresh token for new token pair (rotation)
5. **Logout** - Revoke session, blacklist access token

## Token Architecture

- **Access Token**: JWT (HS256), 15-minute TTL, contains `sub`, `tid`, `org`, `roles`, `jti`, `iat`, `exp`
- **Refresh Token**: Opaque 64-char hex string, stored as SHA-256 hash in DB, 7-day TTL, rotated on use

## Security Features

- **Account Lockout**: After 5 failed login attempts, account locked for 30 minutes
- **Password Policy**: Minimum 8 characters, history check (last 5 passwords)
- **MFA**: TOTP-based (Google Authenticator compatible)
- **Token Blacklisting**: Access tokens blacklisted on logout via Redis
- **Tenant Isolation**: All data access scoped by `tenant_id` in repository layer
- **Identity Guard**: `tenant_id`, `user_id` rejected from request bodies
- **Security Event Logging**: All auth events logged with structured fields

## Database Schema

| Table | Description |
|-------|-------------|
| `users` | User accounts (tenant-scoped) |
| `sessions` | Refresh token sessions |
| `api_keys` | Programmatic API keys |
| `roles` | RBAC roles (per-tenant) |
| `permissions` | Resource/action permissions |
| `role_permissions` | Role-to-permission mapping |
| `user_roles` | User-to-role assignment |
| `password_history` | Password reuse prevention |

## Events Published (Redis Streams)

| Event | Stream | When |
|-------|--------|------|
| `user.created` | `auth-events` | User registration |
| `user.login` | `auth-events` | Successful login |
| `user.logout` | `auth-events` | Logout |
| `user.updated` | `auth-events` | Profile update |
| `user.deleted` | `auth-events` | Account deactivation |
| `user.role_changed` | `auth-events` | Role assignment |
| `auth.failed` | `auth-events` | Failed login attempt |
| `auth.mfa_enabled` | `auth-events` | MFA enrollment |
| `auth.password_changed` | `auth-events` | Password change |

## Development

```bash
# Start infrastructure
make infra-up

# Run with hot-reload
make dev-auth-service

# Run tests
cd services/auth-service && go test ./...

# Build
cd services/auth-service && go build ./cmd/main.go
```

## Configuration

See `config.yaml` for default development configuration. All values can be overridden via environment variables (e.g., `DATABASE_HOST`, `JWT_SECRET`).
