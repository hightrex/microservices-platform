.PHONY: infra-up infra-down services-up test lint security-all

# Infrastructure
infra-up:
	./scripts/manage-infra.sh up

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

lint:
	@echo "Running linters..."
	cd libs/go && golangci-lint run
	# cd services/auth-service && golangci-lint run

# Security
security-sast:
	gosec -fmt=text ./...
	# semgrep --config=p/golang .

security-dast:
	@echo "Running DAST (ZAP)..."
	# podman compose -f deploy/podman/compose.security.yml up -d zaproxy
	# Trigger generic scan

security-all: security-sast security-dast

# Helper to run everything from scratch
all: setup infra-up init-db test security-sast
