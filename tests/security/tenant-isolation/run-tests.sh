#!/bin/bash
set -euo pipefail

# =============================================================================
# Tenant Isolation Test Runner
# Runs all tenant-isolation tests against running services.
# =============================================================================

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(cd "$SCRIPT_DIR/../../.." && pwd)"

# Colors
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
CYAN='\033[0;36m'
NC='\033[0m'

log_info()  { echo -e "${CYAN}[INFO]${NC}  $*"; }
log_ok()    { echo -e "${GREEN}[  OK]${NC}  $*"; }
log_warn()  { echo -e "${YELLOW}[WARN]${NC}  $*"; }
log_error() { echo -e "${RED}[FAIL]${NC} $*"; }

# ---------------------------------------------------------------------------
# Pre-flight checks
# ---------------------------------------------------------------------------
check_prerequisites() {
    log_info "Checking prerequisites..."

    # Check if Go is available
    if ! command -v go &> /dev/null; then
        log_error "Go is not installed"
        exit 1
    fi
    log_ok "Go is available ($(go version | awk '{print $3}'))"

    # Check if infrastructure is running
    if command -v podman &> /dev/null; then
        local pg_running
        pg_running=$(podman ps --filter name=platform-postgres --format "{{.Names}}" 2>/dev/null || true)
        if [ -z "$pg_running" ]; then
            log_warn "Postgres does not appear to be running. Run 'make infra-up' first."
            log_warn "Continuing anyway (tests may fail)..."
        else
            log_ok "Postgres is running"
        fi
    fi
}

# ---------------------------------------------------------------------------
# Run tests
# ---------------------------------------------------------------------------
run_tests() {
    log_info "========================================="
    log_info "Tenant Isolation Test Suite"
    log_info "========================================="
    echo ""

    check_prerequisites

    log_info "Running tenant isolation tests..."
    echo ""

    cd "$PROJECT_ROOT"

    # Run Go tests with verbose output
    if go test -v -count=1 -timeout 120s ./tests/security/tenant-isolation/... 2>&1; then
        echo ""
        log_ok "All tenant isolation tests passed!"
    else
        echo ""
        log_error "Some tenant isolation tests failed!"
        exit 1
    fi
}

# ---------------------------------------------------------------------------
# Main
# ---------------------------------------------------------------------------
case "${1:-run}" in
    run)
        run_tests
        ;;
    check)
        check_prerequisites
        ;;
    --help|-h)
        echo "Usage: $0 [run|check]"
        echo ""
        echo "  run    Run all tenant isolation tests (default)"
        echo "  check  Check prerequisites only"
        ;;
    *)
        echo "Unknown command: $1"
        exit 1
        ;;
esac
