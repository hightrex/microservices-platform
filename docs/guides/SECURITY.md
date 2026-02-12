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
- secrets management via environment variables / Vault.

### Coding Standards
- **No Raw SQL**: Always use parameterized queries.
- **Tenant Context**: Always use `pkg/tenant` to propagate context.
- **Input Validation**: Validate at the edge (Gateway/Handlers).
