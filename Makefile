.PHONY: infra-up infra-down services-up test lint \
       security-all security-sast security-dast security-trivy \
       security-up security-down test-tenant-isolation

# Infrastructure
infra-up:
	./scripts/manage-infra.sh up

infra-core-up:
	./scripts/manage-infra.sh core-up

infra-down:
	./scripts/manage-infra.sh down

infra-logs:
	./scripts/manage-infra.sh logs

infra-clean:
	./scripts/manage-infra.sh clean

# Services (placeholder for now)
services-up:
	@echo "Starting services..."
	# podman compose -f deploy/podman/compose.services.yml up -d

# Development
init-db:
	./scripts/init-databases.sh

setup:
	./scripts/setup-dev.sh

# Testing & Verification
test:
	@echo "Running tests..."
	cd libs/go && go test -v ./...
	# cd services/auth-service && go test -v ./...

test-tenant-isolation:
	@echo "Running tenant isolation tests..."
	cd tests/security/tenant-isolation && go test -v -count=1 ./...

lint:
	@echo "Running linters..."
	cd libs/go && golangci-lint run
	# cd services/auth-service && golangci-lint run

# Security
security-sast:
	@echo "Running SAST..."
	./scripts/run-security-scan.sh --sast-only

security-trivy:
	@echo "Running Trivy container image scans..."
	./scripts/run-security-scan.sh --trivy-only

security-dast:
	@echo "Running DAST (ZAP)..."
	./scripts/run-security-scan.sh --dast-only

security-all: security-sast security-trivy security-dast

security-up:
	./scripts/manage-infra.sh security-up

security-down:
	./scripts/manage-infra.sh security-down

# Helper to run everything from scratch
all: setup infra-up init-db test security-sast
