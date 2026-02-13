#!/bin/bash
set -uo pipefail

# =============================================================================
# Security Scan Runner (Containerized)
# All tools run inside containers via Podman — nothing installed on the host.
# =============================================================================

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(cd "$SCRIPT_DIR/.." && pwd)"
REPORTS_DIR="$PROJECT_ROOT/tests/security/reports"
SEMGREP_RULES="tests/security/sast/rules.yaml"
COMPOSE_SECURITY="$PROJECT_ROOT/deploy/podman/compose.security.yml"

# Tool image versions (pinned)
GITLEAKS_IMAGE="docker.io/zricethezav/gitleaks:v8.23.3"
SEMGREP_IMAGE="docker.io/semgrep/semgrep:1.109.0"
GOSEC_IMAGE="docker.io/securego/gosec:2.22.1"
HADOLINT_IMAGE="docker.io/hadolint/hadolint:v2.12.1-beta"
TRIVY_IMAGE="docker.io/aquasec/trivy:0.59.1"

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

mkdir -p "$REPORTS_DIR"

PASS_COUNT=0
WARN_COUNT=0
FAIL_COUNT=0

inc_pass() { PASS_COUNT=$((PASS_COUNT + 1)); }
inc_warn() { WARN_COUNT=$((WARN_COUNT + 1)); }
inc_fail() { FAIL_COUNT=$((FAIL_COUNT + 1)); }

# ---------------------------------------------------------------------------
# SAST: Static Application Security Testing (all containerized)
# ---------------------------------------------------------------------------
run_gitleaks() {
    log_info "Running gitleaks (secret detection)..."
    if podman run --rm \
        -v "$PROJECT_ROOT:/src:ro,z" \
        "$GITLEAKS_IMAGE" \
        detect --source=/src --no-git \
        --report-path=/dev/stderr \
        --report-format=json 2>"$REPORTS_DIR/gitleaks-report.json"; then
        log_ok "gitleaks: No secrets detected"
        inc_pass
    else
        local exit_code=$?
        if [ "$exit_code" -eq 1 ]; then
            log_warn "gitleaks: Potential secrets found — see tests/security/reports/gitleaks-report.json"
            inc_warn
        else
            log_error "gitleaks: Failed to run (exit $exit_code)"
            inc_fail
        fi
    fi
}

run_semgrep() {
    log_info "Running semgrep (custom tenant isolation + SQL injection rules)..."

    # Build scan targets — only include dirs that have code
    local targets=()
    if [ -d "$PROJECT_ROOT/libs" ] && [ -n "$(find "$PROJECT_ROOT/libs" -name '*.go' -o -name '*.rs' -o -name '*.ts' 2>/dev/null | head -1)" ]; then
        targets+=("/src/libs")
    fi
    if [ -d "$PROJECT_ROOT/services" ] && [ -n "$(find "$PROJECT_ROOT/services" -name '*.go' -o -name '*.rs' -o -name '*.ts' 2>/dev/null | head -1)" ]; then
        targets+=("/src/services")
    fi

    if [ ${#targets[@]} -eq 0 ]; then
        log_info "semgrep: No source code found to scan — skipping"
        return
    fi

    if podman run --rm \
        -v "$PROJECT_ROOT:/src:ro,z" \
        "$SEMGREP_IMAGE" \
        semgrep --config=/src/$SEMGREP_RULES \
        "${targets[@]}" \
        --json --output=/dev/stderr 2>"$REPORTS_DIR/semgrep-report.json"; then
        log_ok "semgrep: No issues found"
        inc_pass
    else
        local exit_code=$?
        if [ "$exit_code" -eq 1 ]; then
            log_warn "semgrep: Issues found — see tests/security/reports/semgrep-report.json"
            inc_warn
        else
            log_error "semgrep: Failed to run (exit $exit_code)"
            inc_fail
        fi
    fi
}

run_gosec() {
    log_info "Running gosec (Go security analysis)..."

    # Check if there's Go code to scan
    if [ ! -f "$PROJECT_ROOT/libs/go/go.mod" ]; then
        log_info "gosec: No Go modules found — skipping"
        return
    fi

    # gosec needs writable go cache, mount project read-only
    if podman run --rm \
        -v "$PROJECT_ROOT:/src:ro,z" \
        -w /src/libs/go \
        -e GOFLAGS="-buildvcs=false" \
        "$GOSEC_IMAGE" \
        -fmt=json -out=/dev/stderr \
        ./... 2>"$REPORTS_DIR/gosec-report.json"; then
        log_ok "gosec: No issues found"
        inc_pass
    else
        local exit_code=$?
        if [ "$exit_code" -eq 1 ]; then
            log_warn "gosec: Issues found — see tests/security/reports/gosec-report.json"
            inc_warn
        else
            log_error "gosec: Failed to run (exit $exit_code)"
            inc_fail
        fi
    fi
}

run_hadolint() {
    log_info "Running hadolint (Containerfile linting)..."
    local containerfiles
    containerfiles=$(find "$PROJECT_ROOT" -name "Containerfile" -o -name "Dockerfile" 2>/dev/null || true)

    if [ -z "$containerfiles" ]; then
        log_info "hadolint: No Containerfiles found — skipping"
        return
    fi

    local hadolint_pass=true
    while IFS= read -r cf; do
        local relative="${cf#$PROJECT_ROOT/}"
        log_info "  Scanning $relative..."
        if podman run --rm \
            -v "$cf:/src/Containerfile:ro,z" \
            "$HADOLINT_IMAGE" \
            hadolint /src/Containerfile; then
            log_ok "  $relative: passed"
        else
            log_warn "  $relative: issues found"
            hadolint_pass=false
        fi
    done <<< "$containerfiles"

    if $hadolint_pass; then
        log_ok "hadolint: All Containerfiles pass"
        inc_pass
    else
        inc_warn
    fi
}

# ---------------------------------------------------------------------------
# Trivy: Container image vulnerability scanning
# ---------------------------------------------------------------------------
run_trivy() {
    log_info "=== Trivy: Container Image Scanning ==="

    # Scan the infra images we're actually using
    local images=(
        "docker.io/library/postgres:17-alpine"
        "docker.io/library/redis:7.4-alpine"
        "docker.io/minio/minio:RELEASE.2025-01-20T14-44-11Z"
    )

    # Also scan any platform service images if they exist
    local svc_images
    svc_images=$(podman images --format "{{.Repository}}:{{.Tag}}" 2>/dev/null \
        | grep -E "(platform-|auth-|org-|gateway|notification|billing|file-|audit|analytics)" || true)
    if [ -n "$svc_images" ]; then
        while IFS= read -r img; do
            images+=("$img")
        done <<< "$svc_images"
    fi

    for img in "${images[@]}"; do
        local report_name
        report_name=$(echo "$img" | tr '/:.' '_')
        log_info "Scanning: $img"
        if podman run --rm \
            -v trivy_cache:/root/.cache/:z \
            -v /run/user/$(id -u)/podman/podman.sock:/var/run/docker.sock:ro,z \
            "$TRIVY_IMAGE" \
            image --severity CRITICAL,HIGH \
            --format json \
            "$img" > "$REPORTS_DIR/trivy-${report_name}.json" 2>&1; then
            log_ok "Trivy: $img — scan complete"
            inc_pass
        else
            log_warn "Trivy: $img — vulnerabilities found (see report)"
            inc_warn
        fi
    done
}

# ---------------------------------------------------------------------------
# DAST: Dynamic Application Security Testing (ZAP)
# ---------------------------------------------------------------------------
run_dast() {
    log_info "=== DAST: OWASP ZAP Baseline Scan ==="

    local zap_running
    zap_running=$(podman ps --filter name=platform-zap --format "{{.Names}}" 2>/dev/null || true)

    if [ -z "$zap_running" ]; then
        log_info "Starting ZAP proxy..."
        podman compose --env-file "$PROJECT_ROOT/.env" -f "$COMPOSE_SECURITY" up -d zaproxy
    fi

    local zap_api_key="${ZAP_API_KEY:-zap-dev-api-key}"
    local zap_url="http://127.0.0.1:8095"

    log_info "Waiting for ZAP to be ready..."
    local max_retries=60
    local count=0
    while ! curl -sf "$zap_url/JSON/core/view/version/?apikey=$zap_api_key" &>/dev/null; do
        sleep 2
        count=$((count + 1))
        if [ $count -ge $max_retries ]; then
            log_error "ZAP is not responding at $zap_url — skipping DAST"
            inc_fail
            return
        fi
    done
    log_ok "ZAP is running and accessible"
    inc_pass

    # Scan any running services
    local targets=(
        "http://host.containers.internal:8080/health"
        "http://host.containers.internal:3000/health"
    )

    for target in "${targets[@]}"; do
        if curl -sf "$target" &>/dev/null 2>&1; then
            log_info "Running ZAP spider against $target ..."
            curl -sf "$zap_url/JSON/spider/action/scan/?apikey=$zap_api_key&url=$target&maxChildren=10" &>/dev/null || true
        else
            log_info "Target $target not reachable — skipping (no services running yet)"
        fi
    done
}

# ---------------------------------------------------------------------------
# Summary
# ---------------------------------------------------------------------------
print_summary() {
    echo ""
    log_info "========================================="
    log_info "Security Scan Summary"
    log_info "========================================="
    echo -e "  ${GREEN}Passed:${NC}   $PASS_COUNT"
    echo -e "  ${YELLOW}Warnings:${NC} $WARN_COUNT"
    echo -e "  ${RED}Failed:${NC}   $FAIL_COUNT"
    log_info "Reports:  $REPORTS_DIR/"
    echo ""

    if [ "$FAIL_COUNT" -gt 0 ]; then
        exit 1
    fi
}

# ---------------------------------------------------------------------------
# Main
# ---------------------------------------------------------------------------
usage() {
    echo "Usage: $0 [--sast-only|--trivy-only|--dast-only|--all]"
    echo ""
    echo "All tools run inside containers via Podman. Nothing is installed on the host."
    echo ""
    echo "Options:"
    echo "  --sast-only    SAST: gitleaks, semgrep, gosec, hadolint (containerized)"
    echo "  --trivy-only   Trivy: container image CVE scanning"
    echo "  --dast-only    DAST: OWASP ZAP baseline scan"
    echo "  --all          Run all security scans (default)"
}

main() {
    local mode="${1:---all}"

    log_info "Microservices Platform — Security Scanner (Containerized)"
    log_info "========================================================="
    echo ""

    case "$mode" in
        --sast-only)
            run_gitleaks
            run_semgrep
            run_gosec
            run_hadolint
            ;;
        --trivy-only)
            run_trivy
            ;;
        --dast-only)
            run_dast
            ;;
        --all)
            run_gitleaks
            run_semgrep
            run_gosec
            run_hadolint
            run_trivy
            run_dast
            ;;
        --help|-h)
            usage
            exit 0
            ;;
        *)
            log_error "Unknown option: $mode"
            usage
            exit 1
            ;;
    esac

    print_summary
}

main "$@"
