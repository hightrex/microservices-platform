# Security Testing

This directory contains the security testing infrastructure for the Microservices Platform.

## Structure

- `sast/`: Static Application Security Testing configuration (Semgrep, Gosec)
- `dast/`: Dynamic Application Security Testing configuration (OWASP ZAP)
- `pentest/`: Manual pentesting scripts and resources (Kali)

## Running Tests

### SAST
Run static analysis tools:
```bash
make security-sast
```

### DAST
Run dynamic analysis tools (requires running services):
```bash
make security-dast
```

## Rules

### Tenant Isolation
All database queries MUST include `tenant_id` in the WHERE clause. This is enforced by Semgrep rules in `sast/rules.yaml`.

### SQL Injection
Raw SQL string concatenation is forbidden. Use parameterized queries. Enforced by Semgrep.
