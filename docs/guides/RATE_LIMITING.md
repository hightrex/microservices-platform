# Rate Limiting Standards

This document defines the conventions for **rate limiting** across the microservices platform. Phase 0 sets the standards; Phase 1 implements them in the API Gateway.

---

## Error Code

All rate-limited responses MUST return HTTP **429 Too Many Requests**.

### Response Body

```json
{
  "code": 429,
  "error": "rate_limit_exceeded",
  "message": "Too many requests. Please retry after the indicated time.",
  "retry_after_seconds": 60
}
```

---

## Response Headers

Every API response (not just rate-limited ones) MUST include these headers so clients can self-regulate:

| Header              | Description                                   | Example      |
|---------------------|-----------------------------------------------|--------------|
| `X-RateLimit-Limit`  | Maximum requests allowed in the window        | `100`        |
| `X-RateLimit-Remaining` | Remaining requests in the current window   | `42`         |
| `X-RateLimit-Reset`  | Unix epoch timestamp when the window resets   | `1707830400` |
| `Retry-After`        | Seconds until the client should retry (only on 429) | `60`    |

---

## Rate Limit Tiers

Rate limits are **per-organization** and derived from the organization's billing plan:

| Plan       | Requests/Minute | Burst Limit | Notes                        |
|-----------|-----------------|-------------|------------------------------|
| Free       | 60              | 10          | Suitable for testing         |
| Starter    | 300             | 50          | Small teams                  |
| Business   | 1,000           | 100         | Production workloads         |
| Enterprise | 5,000           | 500         | Custom limits negotiable     |

### Per-Endpoint Overrides

Some endpoints have stricter limits regardless of plan:

| Endpoint Pattern        | Max Requests/Minute | Reason                      |
|------------------------|--------------------|-----------------------------|
| `POST /auth/login`     | 10                 | Brute-force protection      |
| `POST /auth/register`  | 5                  | Abuse prevention            |
| `POST /auth/forgot-password` | 3            | Email spam prevention       |
| `POST /*/import`       | 5                  | Resource-intensive          |

---

## Configuration Schema

Rate limit config in the Gateway's config file:

```yaml
rate_limiting:
  enabled: true
  # Default for all endpoints
  default:
    requests_per_minute: 100
    burst: 20
  # Per-plan overrides (looked up by org's plan)
  plans:
    free:
      requests_per_minute: 60
      burst: 10
    starter:
      requests_per_minute: 300
      burst: 50
    business:
      requests_per_minute: 1000
      burst: 100
    enterprise:
      requests_per_minute: 5000
      burst: 500
  # Per-endpoint overrides (highest priority)
  endpoints:
    - pattern: "POST /auth/login"
      requests_per_minute: 10
      burst: 3
    - pattern: "POST /auth/register"
      requests_per_minute: 5
      burst: 2
    - pattern: "POST /auth/forgot-password"
      requests_per_minute: 3
      burst: 1
  # Redis-backed sliding window
  backend: redis
  redis:
    key_prefix: "ratelimit:"
    # Window type: "sliding" or "fixed"
    window: sliding
```

---

## Implementation Notes (for Phase 1)

1. **Store**: Use Redis with sliding window counters (key = `ratelimit:{org_id}:{endpoint}:{window}`)
2. **Identity**: Rate limit by `org_id` (from JWT/context), NOT by IP only. Anonymous requests rate-limit by IP.
3. **Gateway-level**: Rate limiting is a gateway concern. Services behind the gateway do NOT implement their own rate limiting.
4. **Bypass**: Internal service-to-service calls (using API keys) can bypass rate limits. This is configurable.
5. **Monitoring**: Emit Prometheus metrics: `gateway_ratelimit_total{org_id, endpoint, status}` (allowed/rejected).

---

## Security Considerations

- Rate limiting MUST be stateless from the client's perspective (no cookies required).
- The `Retry-After` header MUST accurately reflect the reset window.
- Rate limiting MUST NOT leak information about other tenants.
- Logs of rate-limited requests SHOULD include `org_id`, `endpoint`, `ip_address` for abuse analysis.
