# Microservices Platform

A focused, multi-tenant SaaS platform for organizations (hospitals, schools, businesses) to manage their operations. Built with a polyglot stack (Go, Rust, TypeScript) to demonstrate scalable microservices patterns.

## Architecture

The platform consists of 8 focused services enabling organizations to choose modules according to their needs (multi-tenancy with feature toggles).

```mermaid
graph TB
  subgraph clients [Clients]
    Web[Web Frontend]
    Mobile[Mobile App / API]
  end

  subgraph gateway [Infrastructure Layer]
    GW[API Gateway - TypeScript<br/>Port 3000]
  end

  subgraph core [Core Platform - Go]
    Auth[Auth & Identity Service<br/>Port 8080<br/>JWT, RBAC, MFA]
    Org[Organization Service<br/>Port 8081<br/>Multi-tenancy, Feature Toggles]
  end

  subgraph services [Modular Services]
    Notify[Notification Service - Go<br/>Port 8082<br/>Email + SMS + In-App + Webhook]
    Billing[Billing Service - Rust<br/>Port 8083<br/>Subscriptions, Invoicing]
    FileS[File Service - Rust<br/>Port 8084<br/>Documents, Storage]
    Audit[Audit Service - Go<br/>Port 8085<br/>Compliance, Security Logs]
    Analytics[Analytics Service - Go<br/>Port 8086<br/>Dashboards, Reports]
  end

  subgraph data [Data Layer]
    PG[(PostgreSQL<br/>per-service DBs)]
    Redis[(Redis<br/>Cache + Streams)]
    S3[(MinIO / S3<br/>Object Storage)]
  end

  Web --> GW
  Mobile --> GW
  GW --> Auth
  GW --> Org
  GW --> Notify
  GW --> Billing
  GW --> FileS
  GW --> Audit
  GW --> Analytics
  Auth --> Redis
  Org --> PG
  Billing --> PG
  FileS --> S3
  Audit --> PG
  Analytics --> PG
```

## Features

- **Multi-Tenancy:** First-class support for organizations with strict data isolation.
- **Module System:** Organizations enable only the services they need (e.g., Hospital = Auth+Files+Audit).
- **Polyglot Stack:**
  - **Go (Gin):** High-throughput core services (Auth, Org, Notification, Audit, Analytics).
  - **Rust (Axum):** Safety-critical and high-performance services (Billing, File).
  - **TypeScript (Express):** Flexible API Gateway layer.
- **Observability:** Full OpenTelemetry tracing, Prometheus metrics, and Grafana dashboards.
- **Security:** "Shift-left" security with SAST, DAST, and container scanning in every build.

## Prerequisites

- Go 1.22+
- Rust 1.75+
- Node.js 20+
- Podman & Podman Compose
- Git

## Quick Start

1. **Clone the repository:**
   ```bash
   git clone <repo-url>
   cd microservices-platform
   ```

2. **Start Infrastructure:**
   ```bash
   make infra-up
   ```

3. **Run Services (Dev Mode):**
   ```bash
   make services-up
   ```

## Documentation

- [Project Tracker](PROJECT_TRACKER.md)
- [Architecture Vision](docs/architecture/VISION.md)
- [Development Guide](docs/guides/DEVELOPMENT.md)
- [Security Testing](docs/guides/SECURITY.md)

## License

[License Name]
