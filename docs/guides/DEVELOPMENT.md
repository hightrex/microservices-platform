# Development Guide

## Prerequisites
- Go 1.21+
- Rust 1.75+
- Node.js 20+
- Podman & Podman Compose
- Make

## Quick Start
1.  **Setup Environment**:
    ```bash
    make setup
    ```

2.  **Start Infrastructure**:
    ```bash
    make infra-up
    ```

3.  **Initialize Databases**:
    ```bash
    make init-db
    ```

4.  **Create a New Service**:
    ```bash
    ./scripts/create-service.sh my-service
    ```

## Workflow
- **Local Dev**: Run services natively or in containers.
- **Testing**: `make test` runs unit tests.
- **Linting**: `make lint` runs golangci-lint.
- **Security**: `make security-sast` checks for vulnerabilities.
