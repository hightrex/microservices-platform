# Audit Service

Immutable audit log with event capture, hash chaining, retention policies, and compliance exports.

## Architecture

- **Language**: Go 1.24 with Gin framework
- **Port**: 8085
- **Database**: PostgreSQL (audit_service)
- **Cache**: Redis (DB 4 for cache, DB 0 for messaging)
- **Immutability**: Database triggers prevent UPDATE/DELETE on audit_logs table

## Features

- **Event Capture**: Subscribes to ALL Redis Streams events from every service
- **Hash Chaining**: SHA-256 hash chain prevents log tampering (compliance mode)
- **Integrity Verification**: API to verify hash chain integrity for any date range
- **Retention Policies**: Configurable per-event-type log retention (30–2555 days)
- **Export Jobs**: Async export of audit logs to CSV/JSON format
- **Statistics**: Event counts by type, category, and outcome
- **Immutable Storage**: Database triggers prevent any modification to audit entries

## API Endpoints

### Audit Logs (read-only — no CREATE/UPDATE/DELETE)
| Method | Endpoint | Description |
|--------|----------|-------------|
| GET | `/api/v1/audit/logs` | Search/filter audit logs (paginated) |
| GET | `/api/v1/audit/logs/:id` | Get single log entry |
| GET | `/api/v1/audit/stats` | Event statistics |
| POST | `/api/v1/audit/verify` | Verify hash chain integrity |

### Exports
| Method | Endpoint | Description |
|--------|----------|-------------|
| POST | `/api/v1/audit/logs/export` | Create export job |
| GET | `/api/v1/audit/logs/export/:id` | Get export status |
| GET | `/api/v1/audit/logs/exports` | List all exports |

### Retention Policies (admin only)
| Method | Endpoint | Description |
|--------|----------|-------------|
| GET | `/api/v1/audit/retention-policies` | List policies |
| POST | `/api/v1/audit/retention-policies` | Create policy |
| PUT | `/api/v1/audit/retention-policies/:id` | Update policy |
| DELETE | `/api/v1/audit/retention-policies/:id` | Delete policy |

## Event Streams

### Consumes (ALL events from ALL services)
- `auth-events`: `*` (all user/auth events)
- `org-events`: `*` (all organization events)
- `notification-events`: `*` (all notification events)

### Publishes (to `audit-events`)
- `audit.export_completed` — Export ready for download
- `audit.logs_archived` — Logs moved to cold storage
- `audit.chain_broken` — Integrity violation detected

## Hash Chain

Each audit log entry includes:
- `previous_hash`: SHA-256 hash of the prior entry for this tenant
- `current_hash`: SHA-256 of `{previous_hash}|{tenant_id}|{timestamp}|{event_type}|{actor_id}|{resource_id}|{action}|{outcome}`

This creates a tamper-evident chain similar to a blockchain.

## Setup

```bash
# Build
cd services/audit-service && go build ./...

# Test
go test ./...

# Run locally
go run ./cmd/main.go
```

## Compliance

- **SOC 2**: Immutable audit trail with integrity verification
- **GDPR**: Export functionality for data subject access requests
- **HIPAA**: Full event capture with configurable retention
