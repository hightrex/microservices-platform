# Organization Service (Phase 1 Complete)

Multi-tenancy management service for the Microservices Platform. Manages organizations (tenants), module toggles, billing plans, member management, and departments.

**Port:** 8081 | **Language:** Go/Gin | **Database:** PostgreSQL | **Cache:** Redis

## Architecture

The organization `id` serves as the `tenant_id` for all downstream services. This service is the source of truth for:
- Organization lifecycle (create, update, suspend, delete)
- Module availability (which features are enabled per org)
- Billing plan management (free/starter/business/enterprise)
- Member management (invite, remove, role assignment)
- Department hierarchy

## API Endpoints

### Organizations
| Method | Endpoint | Description | Auth |
|--------|----------|-------------|------|
| POST | `/api/v1/organizations` | Create organization | Any authenticated user |
| GET | `/api/v1/organizations/:id` | Get organization details | Authenticated |
| PUT | `/api/v1/organizations/:id` | Update organization settings | org_owner, org_admin |

### Modules
| Method | Endpoint | Description | Auth |
|--------|----------|-------------|------|
| GET | `/api/v1/organizations/:id/modules` | List all modules | Authenticated |
| PUT | `/api/v1/organizations/:id/modules` | Toggle module on/off | org_owner, org_admin |
| GET | `/api/v1/organizations/:id/modules/:name/config` | Get module config | Authenticated |

### Members
| Method | Endpoint | Description | Auth |
|--------|----------|-------------|------|
| GET | `/api/v1/organizations/:id/members` | List members | Authenticated |
| POST | `/api/v1/organizations/:id/members/invite` | Invite member | org_owner, org_admin |
| DELETE | `/api/v1/organizations/:id/members/:userId` | Remove member | org_owner, org_admin |

### Plans
| Method | Endpoint | Description | Auth |
|--------|----------|-------------|------|
| GET | `/api/v1/organizations/:id/plan` | Get current plan | Authenticated |
| PUT | `/api/v1/organizations/:id/plan` | Change plan | org_owner, org_admin |
| GET | `/api/v1/plans` | List available plans | Authenticated |

### Departments
| Method | Endpoint | Description | Auth |
|--------|----------|-------------|------|
| GET | `/api/v1/organizations/:id/departments` | List departments (hierarchy) | Authenticated |
| POST | `/api/v1/organizations/:id/departments` | Create department | org_owner, org_admin, manager |
| PUT | `/api/v1/organizations/:id/departments/:deptId` | Update department | org_owner, org_admin, manager |
| DELETE | `/api/v1/organizations/:id/departments/:deptId` | Delete department | org_owner, org_admin, manager |

### Infrastructure
| Method | Endpoint | Description |
|--------|----------|-------------|
| GET | `/health` | Health check (DB + Redis) |
| GET | `/metrics` | Prometheus metrics |

## Database Migrations

| Migration | Description |
|-----------|-------------|
| 001 | Organizations table with plan, status, settings |
| 002 | Org modules table (per-org feature toggles) |
| 003 | Org members table (invitations, roles) |
| 004 | Plans table (seeded: free/starter/business/enterprise) |
| 005 | Departments table (hierarchical, self-referential) |

## Plans & Limits

| Plan | Max Users | Storage | API Calls/min | Available Modules |
|------|-----------|---------|---------------|-------------------|
| Free | 5 | 1 GB | 60 | audit_logging |
| Starter | 25 | 10 GB | 300 | notifications, audit_logging |
| Business | 100 | 100 GB | 1,000 | All modules |
| Enterprise | 1,000 | 1 TB | 5,000 | All modules |

## Redis Streams Events

| Event | Published When |
|-------|---------------|
| `org.created` | Organization created |
| `org.updated` | Organization settings updated |
| `org.deleted` | Organization soft-deleted |
| `org.module_toggled` | Module enabled/disabled |
| `org.plan_changed` | Plan upgrade/downgrade |
| `org.member_invited` | Member invited |
| `org.member_removed` | Member removed |

## Module Caching

Module configurations are cached in Redis with a 5-minute TTL (key: `org:modules:{orgID}`). The cache is invalidated on:
- Module toggle
- Plan change (may disable modules)
- Module config update

This is the hot path used by the API Gateway for module gating.

## Configuration

See `config.yaml` for all configuration options. Environment variables override YAML (e.g., `DATABASE_HOST`).

## Development

```bash
# Run locally (requires Postgres + Redis)
make infra-core-up
cd services/organization-service
go run ./cmd/main.go

# Run tests
go test ./...

# Build container
podman build -f services/organization-service/Containerfile -t organization-service .
```

## Security

- All data access is tenant-scoped (org_id = tenant_id)
- Identity comes from gateway-forwarded headers, never from request body
- RBAC enforced per endpoint (org_owner > org_admin > manager > member > viewer)
- Plan limits enforced on member invitations
- Module availability enforced by plan
- Security events logged via `pkg/securitylog`
