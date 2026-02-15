.PHONY: \
	help \
	infra-up infra-core-up infra-down infra-restart infra-status infra-ps infra-logs infra-clean infra-purge infra-all \
	infra-dev-up infra-dev-down infra-dev-restart \
	infra-security-up infra-security-down infra-security-restart infra-security-logs \
	services-up services-down services-restart services-status services-ps services-logs services-build services-rebuild services-clean services-pull \
	services-up-auth services-up-org services-up-gateway services-up-notification services-up-audit services-up-billing services-up-file \
	core-up core-down core-logs core-status core-build core-rebuild core-reset \
	stack-up stack-down stack-restart stack-status stack-logs stack-build stack-rebuild stack-clean stack-reset \
	init-db setup test test-ts test-gateway test-go test-rust test-billing test-file test-notification test-audit test-tenant-isolation test-all lint lint-rust \
	security-sast security-trivy security-dast security-all security-up security-down \
	all

# -------------------------
# Help
# -------------------------
help:
	@echo ""
	@echo "========================"
	@echo " Microservices Platform"
	@echo "========================"
	@echo ""
	@echo "Infra:"
	@echo "  make infra-up               Start base infrastructure stack"
	@echo "  make infra-core-up          Start only core infra (postgres, redis, minio)"
	@echo "  make infra-down             Stop base infrastructure stack"
	@echo "  make infra-restart          Restart base infrastructure stack"
	@echo "  make infra-status           Show status of infra/dev/security stacks"
	@echo "  make infra-logs             Tail base infra logs"
	@echo "  make infra-clean            Remove project-scoped infra resources (safe-ish)"
	@echo "  make infra-purge            DANGER: remove ALL podman resources for this user"
	@echo "  make infra-all              Alias for infra-up"
	@echo ""
	@echo "Dev tools:"
	@echo "  make infra-dev-up           Start dev tools stack"
	@echo "  make infra-dev-down         Stop dev tools stack"
	@echo "  make infra-dev-restart      Restart dev tools stack"
	@echo ""
	@echo "Security tools (infrastructure):"
	@echo "  make infra-security-up      Start security tools stack"
	@echo "  make infra-security-down    Stop security tools stack"
	@echo "  make infra-security-restart Restart security tools stack"
	@echo "  make infra-security-logs    Tail security tools logs"
	@echo ""
	@echo "Services (microservices stack):"
	@echo "  make services-up            Start all services"
	@echo "  make services-down          Stop all services"
	@echo "  make services-restart       Restart all services"
	@echo "  make services-status        Show services status"
	@echo "  make services-logs          Tail services logs"
	@echo "  make services-build         Build services images"
	@echo "  make services-rebuild       Rebuild services images (no-cache) then start"
	@echo "  make services-clean         Remove services stack + volumes (project scoped)"
	@echo "  make services-pull          Pull referenced images (if any)"
	@echo "  make services-up-auth       Start only Auth Service"
	@echo "  make services-up-org        Start only Organization Service"
	@echo "  make services-up-notification  Start only Notification Service"
	@echo "  make services-up-billing    Start only Billing Service (Rust)"
	@echo "  make services-up-file       Start only File Service (Rust)"
	@echo "  make services-up-audit      Start only Audit Service"
	@echo "  make services-up-gateway    Start only API Gateway"
	@echo ""
	@echo "Core stack (infra + auth + org + gateway):"
	@echo "  make core-up                Start infra + core services (auth, org, gateway)"
	@echo "  make core-down              Stop core services"
	@echo "  make core-logs              Tail core services logs"
	@echo "  make core-status            Show core services status"
	@echo "  make core-build             Build core service images"
	@echo "  make core-rebuild           Rebuild core (no-cache) then start"
	@echo "  make core-reset             Clean -> up -> init-db -> healthy"
	@echo ""
	@echo "Full stack (infra + dev + security + services):"
	@echo "  make stack-up               Start everything"
	@echo "  make stack-down             Stop everything (safe order)"
	@echo "  make stack-restart          Restart everything"
	@echo "  make stack-status           Status of everything"
	@echo "  make stack-logs             Tail logs (infra + services)"
	@echo "  make stack-build            Build services images (keeps caches)"
	@echo "  make stack-rebuild          Rebuild services images (no-cache) then start everything"
	@echo "  make stack-clean            Remove project-scoped resources (infra + services)"
	@echo "  make stack-reset            Clean then start everything from scratch"
	@echo ""
	@echo "Dev workflow:"
	@echo "  make setup                  Run dev setup script"
	@echo "  make init-db                Initialize databases"
	@echo "  make test                   Run all tests (Go + TypeScript + Rust)"
	@echo "  make test-all               Alias for test"
	@echo "  make test-go                Run all Go tests"
	@echo "  make test-rust              Run all Rust tests"
	@echo "  make test-billing           Run Billing Service tests (Rust)"
	@echo "  make test-file              Run File Service tests (Rust)"
	@echo "  make test-notification      Run Notification Service tests"
	@echo "  make test-audit             Run Audit Service tests"
	@echo "  make lint                   Run all linters (Go + TypeScript + Rust)"
	@echo "  make lint-rust              Run Rust clippy + fmt check"
	@echo ""
	@echo "Security scans:"
	@echo "  make security-sast          Run SAST"
	@echo "  make security-trivy         Run Trivy image scans"
	@echo "  make security-dast          Run ZAP DAST"
	@echo "  make security-all           Run all scans"
	@echo ""
	@echo "Bootstrap:"
	@echo "  make all                    setup + infra-up + init-db + test + security-sast"
	@echo ""

# -------------------------
# Infra
# -------------------------
infra-up:
	@echo "==> Starting infrastructure..."
	./scripts/manage-infra.sh up

infra-all: infra-up

infra-core-up:
	@echo "==> Starting core infrastructure (postgres, redis, minio)..."
	./scripts/manage-infra.sh core-up

infra-down:
	@echo "==> Stopping infrastructure..."
	./scripts/manage-infra.sh down

infra-restart:
	@echo "==> Restarting infrastructure..."
	./scripts/manage-infra.sh down || true
	./scripts/manage-infra.sh up

infra-status infra-ps:
	@echo "==> Showing infra/dev/security status..."
	./scripts/manage-infra.sh status

infra-logs:
	@echo "==> Tailing infrastructure logs..."
	./scripts/manage-infra.sh logs

infra-clean:
	@echo "==> Cleaning project-scoped infrastructure resources..."
	./scripts/manage-infra.sh clean

infra-purge:
	@echo "==> DANGER: Purging ALL podman resources..."
	./scripts/manage-infra.sh purge

# -------------------------
# Dev tools (infra)
# -------------------------
infra-dev-up:
	@echo "==> Starting dev tools..."
	./scripts/manage-infra.sh dev-up

infra-dev-down:
	@echo "==> Stopping dev tools..."
	./scripts/manage-infra.sh dev-down

infra-dev-restart:
	@echo "==> Restarting dev tools..."
	./scripts/manage-infra.sh dev-down || true
	./scripts/manage-infra.sh dev-up

# -------------------------
# Security tools (infra)
# -------------------------
infra-security-up:
	@echo "==> Starting security tools..."
	./scripts/manage-infra.sh security-up

infra-security-down:
	@echo "==> Stopping security tools..."
	./scripts/manage-infra.sh security-down

infra-security-restart:
	@echo "==> Restarting security tools..."
	./scripts/manage-infra.sh security-down || true
	./scripts/manage-infra.sh security-up

infra-security-logs:
	@echo "==> Tailing security tools logs..."
	./scripts/manage-infra.sh security-logs

# -------------------------
# Services (compose.services.yml)
# -------------------------
SERVICES_COMPOSE := deploy/podman/compose.services.yml

services-up:
	@echo "==> Starting services..."
	podman compose -f $(SERVICES_COMPOSE) up -d

services-down:
	@echo "==> Stopping services..."
	podman compose -f $(SERVICES_COMPOSE) down

services-restart:
	@echo "==> Restarting services..."
	podman compose -f $(SERVICES_COMPOSE) down || true
	podman compose -f $(SERVICES_COMPOSE) up -d

services-status services-ps:
	@echo "==> Services status:"
	podman compose -f $(SERVICES_COMPOSE) ps

services-logs:
	@echo "==> Tailing services logs..."
	podman compose -f $(SERVICES_COMPOSE) logs -f

services-pull:
	@echo "==> Pulling service images (if applicable)..."
	podman compose -f $(SERVICES_COMPOSE) pull || true

services-up-auth:
	@echo "==> Starting Auth Service..."
	podman compose -f $(SERVICES_COMPOSE) up -d auth-service

services-up-org:
	@echo "==> Starting Organization Service..."
	podman compose -f $(SERVICES_COMPOSE) up -d organization-service

services-up-notification:
	@echo "==> Starting Notification Service..."
	podman compose -f $(SERVICES_COMPOSE) up -d notification-service

services-up-billing:
	@echo "==> Starting Billing Service..."
	podman compose -f $(SERVICES_COMPOSE) up -d billing-service

services-up-file:
	@echo "==> Starting File Service..."
	podman compose -f $(SERVICES_COMPOSE) up -d file-service

services-up-audit:
	@echo "==> Starting Audit Service..."
	podman compose -f $(SERVICES_COMPOSE) up -d audit-service

services-up-gateway:
	@echo "==> Starting API Gateway..."
	podman compose -f $(CORE_COMPOSE) up -d api-gateway

gateway-build:
	@echo "==> Building API Gateway image..."
	podman compose -f $(CORE_COMPOSE) build api-gateway

gateway-rebuild:
	@echo "==> Rebuilding API Gateway image (no cache) and starting..."
	podman compose -f $(CORE_COMPOSE) build --no-cache api-gateway
	podman compose -f $(CORE_COMPOSE) up -d api-gateway

gateway-restart:
	@echo "==> Restarting API Gateway..."
	podman compose -f $(CORE_COMPOSE) restart api-gateway

gateway-clean:
	@echo "==> Removing API Gateway stack (project scoped: containers + volumes + orphans)..."
	podman compose -f $(CORE_COMPOSE) down api-gateway -v --remove-orphans || true

gateway-stop:
	@echo "==> Stopping API Gateway..."
	podman compose -f $(CORE_COMPOSE) stop api-gateway

gateway-start:
	@echo "==> Starting API Gateway..."
	podman compose -f $(CORE_COMPOSE) start api-gateway

gateway-status:
	@echo "==> API Gateway status:"
	podman compose -f $(CORE_COMPOSE) ps api-gateway

gateway-logs:
	@echo "==> Tailing API Gateway logs..."
	podman compose -f $(CORE_COMPOSE) logs -f api-gateway

services-build:
	@echo "==> Building service images..."
	podman compose -f $(SERVICES_COMPOSE) build

services-rebuild:
	@echo "==> Rebuilding service images (no cache) and starting..."
	podman compose -f $(SERVICES_COMPOSE) build --no-cache
	podman compose -f $(SERVICES_COMPOSE) up -d

services-clean:
	@echo "==> Removing services stack (project scoped: containers + volumes + orphans)..."
	podman compose -f $(SERVICES_COMPOSE) down -v --remove-orphans || true
	@echo "✅ Services clean complete."

# -------------------------
# Core stack (infra + auth + org + gateway)
# -------------------------
CORE_COMPOSE := deploy/podman/compose.core.yml

core-up:
	@echo "==> Starting core stack (infra + auth + org + gateway)..."
	./scripts/manage-infra.sh up
	@sleep 3
	podman compose -f $(CORE_COMPOSE) up -d
	@echo "✅ Core stack is running. Gateway at http://localhost:3000"

core-down:
	@echo "==> Stopping core services..."
	podman compose -f $(CORE_COMPOSE) down

core-logs:
	@echo "==> Tailing core services logs..."
	podman compose -f $(CORE_COMPOSE) logs -f

core-status:
	@echo "==> Core services status:"
	podman compose -f $(CORE_COMPOSE) ps

core-build:
	@echo "==> Building core service images..."
	podman compose -f $(CORE_COMPOSE) build

core-rebuild:
	@echo "==> Rebuilding core services (no-cache) then starting..."
	podman compose -f $(CORE_COMPOSE) build --no-cache
	$(MAKE) core-up

core-reset:
	@echo "==> RESET core: clean -> up -> init-db -> healthy..."
	podman compose -f $(CORE_COMPOSE) down -v --remove-orphans || true
	./scripts/manage-infra.sh clean || true
	$(MAKE) core-up
	@sleep 5
	$(MAKE) init-db
	@echo "✅ Core stack reset complete."

# -------------------------
# Full stack orchestration
# -------------------------
stack-up:
	@echo "==> Starting FULL STACK (infra + dev + security + services)..."
	./scripts/manage-infra.sh up
	./scripts/manage-infra.sh dev-up
	./scripts/manage-infra.sh security-up
	podman compose -f $(CORE_COMPOSE) up -d
	@echo "✅ Full stack is running."

stack-down:
	@echo "==> Stopping FULL STACK (services -> security -> dev -> infra)..."
	podman compose -f $(CORE_COMPOSE) down || true
	./scripts/manage-infra.sh security-down || true
	./scripts/manage-infra.sh dev-down || true
	./scripts/manage-infra.sh down || true
	@echo "✅ Full stack stopped."

stack-restart:
	@echo "==> Restarting FULL STACK..."
	$(MAKE) stack-down
	$(MAKE) stack-up

stack-status:
	@echo "==> Full stack status:"
	./scripts/manage-infra.sh status
	@echo ""
	@echo "=== Services ==="
	podman compose -f $(SERVICES_COMPOSE) ps

stack-logs:
	@echo "==> Tailing FULL STACK logs (infra then services)..."
	@echo "---- INFRA LOGS (Ctrl+C to stop) ----"
	./scripts/manage-infra.sh logs

stack-build:
	@echo "==> Building services (keeps cache)..."
	podman compose -f $(CORE_COMPOSE) build

stack-rebuild:
	@echo "==> Rebuilding services (no-cache) then starting FULL STACK..."
	podman compose -f $(CORE_COMPOSE) build --no-cache
	$(MAKE) stack-up

stack-clean:
	@echo "==> Cleaning FULL STACK (project scoped)..."
	podman compose -f $(CORE_COMPOSE) down -v --remove-orphans || true
	./scripts/manage-infra.sh clean
	@echo "✅ Full stack clean complete."

stack-reset:
	@echo "==> RESET: cleaning then starting everything..."
	$(MAKE) stack-clean
	$(MAKE) stack-up

# -------------------------
# Development
# -------------------------
init-db:
	@echo "==> Initializing databases..."
	./scripts/init-databases.sh

setup:
	@echo "==> Running dev setup..."
	./scripts/setup-dev.sh

# -------------------------
# Testing & lint
# -------------------------
test:
	@echo "==> Running ALL tests (Go + TypeScript + Rust)..."
	@echo ""
	@echo "========== Go Tests =========="
	@echo "--- libs/go ---"
	cd libs/go && go test -v ./...
	@echo "--- auth-service ---"
	cd services/auth-service && go test -v ./...
	@echo "--- organization-service ---"
	cd services/organization-service && go test -v ./...
	@echo "--- notification-service ---"
	cd services/notification-service && go test -v ./...
	@echo "--- audit-service ---"
	cd services/audit-service && go test -v ./...
	@echo ""
	@echo "========== TypeScript Tests =========="
	@echo "--- libs/typescript ---"
	cd libs/typescript && npm test
	@echo "--- api-gateway ---"
	cd services/api-gateway && npm test
	@echo ""
	@echo "========== Rust Tests =========="
	@echo "--- libs/rust (shared crates) ---"
	cd libs/rust && cargo test --workspace
	@echo "--- billing-service ---"
	cd services/billing-service && cargo test
	@echo "--- file-service ---"
	cd services/file-service && cargo test

test-all: test

test-go:
	@echo "==> Running all Go tests..."
	@echo "--- libs/go ---"
	cd libs/go && go test -v ./...
	@echo "--- auth-service ---"
	cd services/auth-service && go test -v ./...
	@echo "--- organization-service ---"
	cd services/organization-service && go test -v ./...
	@echo "--- notification-service ---"
	cd services/notification-service && go test -v ./...
	@echo "--- audit-service ---"
	cd services/audit-service && go test -v ./...

test-rust:
	@echo "==> Running all Rust tests..."
	@echo "--- libs/rust (shared crates) ---"
	cd libs/rust && cargo test --workspace
	@echo "--- billing-service ---"
	cd services/billing-service && cargo test
	@echo "--- file-service ---"
	cd services/file-service && cargo test

test-billing:
	@echo "==> Running Billing Service tests (Rust)..."
	cd services/billing-service && cargo test

test-file:
	@echo "==> Running File Service tests (Rust)..."
	cd services/file-service && cargo test

test-notification:
	@echo "==> Running Notification Service tests..."
	cd services/notification-service && go test -v ./...

test-audit:
	@echo "==> Running Audit Service tests..."
	cd services/audit-service && go test -v ./...

test-ts:
	@echo "==> Running TypeScript tests..."
	@echo "--- libs/typescript ---"
	cd libs/typescript && npm test
	@echo "--- api-gateway ---"
	cd services/api-gateway && npm test

test-gateway:
	@echo "==> Running API Gateway tests..."
	cd services/api-gateway && npm test

test-tenant-isolation:
	@echo "==> Running tenant isolation tests..."
	cd tests/security/tenant-isolation && go test -v -count=1 ./...

lint:
	@echo "==> Running ALL linters..."
	@echo ""
	@echo "========== Go Lint =========="
	@echo "--- libs/go ---"
	cd libs/go && golangci-lint run
	@echo "--- auth-service ---"
	cd services/auth-service && golangci-lint run
	@echo "--- organization-service ---"
	cd services/organization-service && golangci-lint run
	@echo "--- notification-service ---"
	cd services/notification-service && golangci-lint run
	@echo "--- audit-service ---"
	cd services/audit-service && golangci-lint run
	@echo ""
	@echo "========== TypeScript Lint =========="
	@echo "--- libs/typescript ---"
	cd libs/typescript && npx tsc --noEmit
	@echo "--- api-gateway ---"
	cd services/api-gateway && npx tsc --noEmit
	@echo ""
	@echo "========== Rust Lint =========="
	@echo "--- libs/rust (clippy + fmt) ---"
	cd libs/rust && cargo clippy --workspace -- -D warnings && cargo fmt --check
	@echo "--- billing-service ---"
	cd services/billing-service && cargo clippy -- -D warnings && cargo fmt --check
	@echo "--- file-service ---"
	cd services/file-service && cargo clippy -- -D warnings && cargo fmt --check

lint-rust:
	@echo "==> Running Rust linters..."
	@echo "--- libs/rust ---"
	cd libs/rust && cargo clippy --workspace -- -D warnings && cargo fmt --check
	@echo "--- billing-service ---"
	cd services/billing-service && cargo clippy -- -D warnings && cargo fmt --check
	@echo "--- file-service ---"
	cd services/file-service && cargo clippy -- -D warnings && cargo fmt --check

# -------------------------
# Security scans (scripts)
# -------------------------
security-sast:
	@echo "==> Running SAST..."
	./scripts/run-security-scan.sh --sast-only

security-trivy:
	@echo "==> Running Trivy image scans..."
	./scripts/run-security-scan.sh --trivy-only

security-dast:
	@echo "==> Running DAST (ZAP)..."
	./scripts/run-security-scan.sh --dast-only

security-all: security-sast security-trivy security-dast

security-up: infra-security-up
security-down: infra-security-down

# -------------------------
# Bootstrap helper
# -------------------------
all: setup infra-up init-db test security-sast

# -------------------------
# Phase 1 Security Testing
# -------------------------
.PHONY: security-phase1 security-phase1-quick security-phase1-ci

security-phase1: ## Run complete Phase 1 security scan
	@echo "🔒 Running Phase 1 automated security tests..."
	@echo "   This includes: DAST, Auth tests, Tenant isolation, Fuzz testing"
	@PHASE=1 bash scripts/security/run-automated-pentest.sh

security-phase1-quick: ## Quick Phase 1 security scan (no aggressive tests)
	@echo "🔒 Running quick Phase 1 security scan..."
	@PHASE=1 QUICK_MODE=true bash scripts/security/run-automated-pentest.sh

security-phase1-ci: ## Phase 1 security scan for CI (fails on findings)
	@echo "🔒 Running Phase 1 CI security scan..."
	@PHASE=1 SCAN_MODE=ci bash scripts/security/run-automated-pentest.sh

security-report-server: ## Serve latest security report
	@LATEST=$$(ls -td reports/pentest/* | head -1); \
	echo "📊 Serving report from $$LATEST"; \
	python3 -m http.server --directory "$$LATEST" 8888

