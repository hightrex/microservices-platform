# Rust Shared Libraries (`libs/rust/`)

Shared Rust crates for the platform's Rust microservices (Billing Service, File Service). These crates mirror the patterns established in `libs/go/pkg/` for the Go services, ensuring consistent behavior across languages.

## Workspace Structure

```
libs/rust/
├── Cargo.toml          # Workspace root with shared dependency versions
├── common/             # Core utilities (config, errors, logging, tenant, validation)
├── database/           # PostgreSQL connection pooling and repository helpers
├── messaging/          # Redis Streams producer/consumer for event-driven architecture
└── middleware/          # Axum middleware (auth, tenant, CORS, rate limiting, metrics)
```

## Crates

### `platform-common`

Core utilities shared across all Rust services.

| Module          | Description                                              |
|-----------------|----------------------------------------------------------|
| `config`        | TOML + environment variable configuration loading        |
| `error`         | Standardized `AppError` with HTTP status codes + Axum responses |
| `logger`        | Structured logging via `tracing` (JSON prod, pretty dev) |
| `tenant`        | Tenant context extraction from headers                   |
| `validation`    | Input validation using `validator` crate                 |
| `health`        | Pluggable health check framework with HTTP handler       |
| `tracing_setup` | Distributed tracing configuration (OpenTelemetry-ready)  |
| `metrics`       | Prometheus metrics registry and `/metrics` handler       |

### `platform-database`

PostgreSQL database utilities.

| Module        | Description                                            |
|---------------|--------------------------------------------------------|
| `pool`        | `sqlx::PgPool` creation with configurable connection limits |
| `migrations`  | Migration runner using `sqlx::migrate!()`              |
| `repository`  | `TenantScope` helper and `Pagination`/`PaginatedResponse` types |

### `platform-messaging`

Redis Streams event-driven messaging.

| Module     | Description                                                |
|------------|------------------------------------------------------------|
| `schema`   | `Event` struct (wire-compatible with Go `messaging.Event`) |
| `producer` | Publish events to Redis Streams with auto-generated IDs    |
| `consumer` | Consumer group support, PEL recovery, DLQ, retry logic     |

### `platform-middleware`

Axum middleware stack.

| Module             | Description                                          |
|--------------------|------------------------------------------------------|
| `auth`             | JWT validation, claims extraction, RBAC helpers      |
| `tenant`           | `X-Tenant-ID` extraction + strict tenant requirement |
| `logging`          | Request/response logging with structured fields      |
| `metrics_middleware` | Prometheus request duration/count/active gauges    |
| `cors`             | CORS with explicit origin allowlist (no wildcards)   |
| `rate_limit`       | Redis-backed sliding window rate limiter             |

## Usage

Add workspace crates as dependencies in your service's `Cargo.toml`:

```toml
[dependencies]
platform-common = { path = "../../libs/rust/common" }
platform-database = { path = "../../libs/rust/database" }
platform-messaging = { path = "../../libs/rust/messaging" }
platform-middleware = { path = "../../libs/rust/middleware" }
```

### Quick Start Example

```rust
use platform_common::{config, logger, error::AppError};
use platform_database::pool;
use platform_messaging::producer::Producer;
use platform_middleware::{auth, tenant, logging, cors};

#[derive(serde::Deserialize)]
struct ServiceConfig {
    server: config::ServerConfig,
    database: config::DatabaseConfig,
    redis: config::RedisConfig,
    logger: logger::LoggerConfig,
}

#[tokio::main]
async fn main() -> anyhow::Result<()> {
    // Load config from TOML + env vars
    let cfg: ServiceConfig = config::load_config("config/config.toml")?;

    // Initialize logging
    logger::init(&cfg.logger);

    // Connect to database
    let db_pool = pool::connect(&cfg.database).await?;

    // Create event producer
    let producer = Producer::new(&cfg.redis.connection_url(), "my-service")?;

    // Build Axum router with middleware
    let cors_layer = cors::create_cors_layer(&cfg.server.cors_origins);
    let auth_config = auth::AuthConfig::new("jwt-secret");

    let app = axum::Router::new()
        // ... routes ...
        .layer(axum::middleware::from_fn(logging::request_logger))
        .layer(axum::middleware::from_fn(tenant::tenant_middleware))
        .layer(cors_layer);

    tracing::info!("Service starting on {}", cfg.server.bind_address());
    let listener = tokio::net::TcpListener::bind(cfg.server.bind_address()).await?;
    axum::serve(listener, app).await?;

    Ok(())
}
```

## Event Schema

All events follow the standardized schema (wire-compatible with Go services):

```json
{
  "id": "uuid-v4",
  "type": "subscription.created",
  "version": "1.0",
  "source": "billing-service",
  "tenant_id": "uuid",
  "data": { ... },
  "metadata": {
    "correlation_id": "uuid-v4"
  },
  "timestamp": "2026-02-15T12:00:00Z"
}
```

## Error Response Format

Matches the platform standard:

```json
{
  "success": false,
  "error": {
    "code": "RESOURCE_NOT_FOUND",
    "message": "Subscription not found",
    "details": "Optional non-sensitive details"
  }
}
```

## Development

```bash
# Build all crates
cargo build --workspace

# Run all tests
cargo test --workspace

# Check for lint issues (requires clippy)
cargo clippy --workspace -- -D warnings

# Format code (requires rustfmt)
cargo fmt --check
```

## Design Principles

1. **Tenant isolation enforced at the data layer** via `TenantScope` and `require_tenant()`
2. **No `unwrap()`/`expect()` outside tests** — use `?` or explicit error handling
3. **No `println!`** — use structured `tracing` macros
4. **Mandatory TTL on Redis keys** — enforced by API design
5. **Fail-closed security** — missing auth/tenant context rejects the request
6. **Wire compatibility with Go services** — event schema, error format, API standards
