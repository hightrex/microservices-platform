# API Gateway (Phase 1 Complete)

The central entry point for the Microservices Platform, handling routing, authentication, rate limiting, and module gating for all incoming requests.

**Port:** 3000 | **Language:** TypeScript/Express | **Cache:** Redis

## Architecture

The Gateway sits at the edge and enforces security policies before traffic reaches internal services.

```
       Internet
          │
    [API Gateway] (:3000)
          │
    ┌─────┴─────┐
    ▼           ▼
[Auth Service] [Org Service]
 (:8080)        (:8081)
```

## Core Responsibilities (Phase 1)

1.  **JWT Validation**: Validates `Authorization: Bearer <token>` signature, expiry, and issuer.
2.  **Context Injection**: Extracts `sub`, `tid`, `roles` from JWT and injects `X-User-ID`, `X-Tenant-ID`, `X-User-Roles` headers for downstream services.
3.  **Module Gating**: Checks if the target module (e.g., `billing`) is enabled for the tenant before proxying. Returns `403 MODULE_NOT_ENABLED` if disabled.
4.  **Rate Limiting**: Redis-backed sliding window limiter per tenant/IP.
5.  **Proxy Routing**: Forwards requests to appropriate backend services.

## Configuration

Configuration is loaded from environment variables (validated via Zod).

| Variable | Description | Default |
|----------|-------------|---------|
| `PORT` | Server port | `3000` |
| `AUTH_BASE_URL` | Upstream Auth Service URL | `http://auth-service:8080` |
| `ORG_BASE_URL` | Upstream Org Service URL | `http://organization-service:8081` |
| `REDIS_URL` | Redis connection string | `redis://redis:6379` |
| `JWT_SECRET` | Secret for verifying tokens | `dev-secret-key...` |
| `CORS_ORIGINS` | Allowed CORS origins | `http://localhost:3000` |

## Routes

### Public
- `GET /health` - Aggregate health check
- `POST /api/v1/auth/login` -> Auth Service
- `POST /api/v1/auth/register` -> Auth Service

### Protected
- `/api/v1/auth/*` -> Auth Service
- `/api/v1/users/*` -> Auth Service
- `/api/v1/organizations/*` -> Org Service

## Development

```bash
# Install dependencies
npm install

# Run in dev mode (hot reload)
npm run dev

# Run tests
npm test

# Build for production
npm run build
```

## Security

- **Helmet**: Sets secure HTTP headers.
- **Cors**: Restricts cross-origin access.
- **Circuit Breaker**: Fails fast if upstream services are down.
- **Identity Guard**: Overwrites identity headers to prevent spoofing.
