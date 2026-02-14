#!/bin/bash
set -euo pipefail

# Manage infrastructure containers with Podman Compose.
#
# Stacks:
#  - Base infra:     deploy/podman/compose.base.yml
#  - Dev tools:      deploy/podman/compose.dev.yml
#  - Security tools: deploy/podman/compose.security.yml
#
# Notes:
#  - clean: removes THIS compose project's containers/networks/volumes (+ optional prune)
#  - purge: removes ALL podman containers/images/volumes/networks (DANGEROUS)

COMPOSE_DIR="deploy/podman"
COMPOSE_FILE="$COMPOSE_DIR/compose.base.yml"
COMPOSE_DEV="$COMPOSE_DIR/compose.dev.yml"
SECURITY_COMPOSE_FILE="$COMPOSE_DIR/compose.security.yml"
ENV_FILE=".env"

# Stable project name so compose resources are predictable and namespaced.
export COMPOSE_PROJECT_NAME="${COMPOSE_PROJECT_NAME:-core-platform}"

COMPOSE_OPTS="--env-file $ENV_FILE"

require_env_file() {
  if [[ ! -f "$ENV_FILE" ]]; then
    echo "ERROR: $ENV_FILE not found."
    echo "       Create it at repo root or adjust ENV_FILE in scripts/manage-infra.sh"
    exit 1
  fi
}

banner() {
  echo ""
  echo "============================================================"
  echo "$1"
  echo "Project: COMPOSE_PROJECT_NAME=$COMPOSE_PROJECT_NAME"
  echo "============================================================"
}

usage() {
  echo "Usage: $0 {up|down|restart|logs|clean|purge|core-up|dev-up|dev-down|dev-restart|security-up|security-down|security-restart|security-logs|status|ps|help}"
  echo ""
  echo "Base infrastructure:"
  echo "  up               Start base infrastructure stack"
  echo "  down             Stop base infrastructure stack"
  echo "  restart          Restart base infrastructure stack"
  echo "  logs             Tail base infrastructure logs"
  echo "  core-up          Start only core infra (postgres, redis, minio)"
  echo ""
  echo "Dev tools:"
  echo "  dev-up           Start dev tools stack"
  echo "  dev-down         Stop dev tools stack"
  echo "  dev-restart      Restart dev tools stack"
  echo ""
  echo "Security tools:"
  echo "  security-up      Start security tools stack"
  echo "  security-down    Stop security tools stack"
  echo "  security-restart Restart security tools stack"
  echo "  security-logs    Tail security tools logs"
  echo ""
  echo "Inspection:"
  echo "  status           Show status of base infra + dev tools + security tools"
  echo "  ps               Alias for status"
  echo ""
  echo "Cleanup:"
  echo "  clean            Remove project-scoped resources (containers/networks/volumes) + prune dangling nets/vols"
  echo "  purge            DANGER: Remove ALL podman resources for this user"
  echo ""
}

cmd="${1:-help}"

case "$cmd" in
  up)
    require_env_file
    banner "Starting base infrastructure"
    podman compose -f "$COMPOSE_FILE" $COMPOSE_OPTS up -d
    echo "✅ Base infrastructure is up."
    ;;

  down)
    require_env_file
    banner "Stopping base infrastructure"
    podman compose -f "$COMPOSE_FILE" $COMPOSE_OPTS down
    echo "✅ Base infrastructure is down."
    ;;

  restart)
    require_env_file
    banner "Restarting base infrastructure"
    podman compose -f "$COMPOSE_FILE" $COMPOSE_OPTS down || true
    podman compose -f "$COMPOSE_FILE" $COMPOSE_OPTS up -d
    echo "✅ Base infrastructure restarted."
    ;;

  logs)
    require_env_file
    banner "Tailing base infrastructure logs"
    podman compose -f "$COMPOSE_FILE" $COMPOSE_OPTS logs -f
    ;;

  core-up)
    require_env_file
    banner "Starting core infrastructure (postgres, redis, minio)"
    podman compose -f "$COMPOSE_FILE" $COMPOSE_OPTS up -d postgres redis minio
    echo "✅ Core infrastructure is up."
    ;;

  dev-up)
    require_env_file
    banner "Starting dev tools"
    # Bring base infra up so dev tools can connect (safe if already running)
    podman compose -f "$COMPOSE_FILE" $COMPOSE_OPTS up -d
    podman compose -f "$COMPOSE_DEV" $COMPOSE_OPTS up -d
    echo "✅ Dev tools are up."
    ;;

  dev-down)
    require_env_file
    banner "Stopping dev tools"
    podman compose -f "$COMPOSE_DEV" $COMPOSE_OPTS down
    echo "✅ Dev tools are down."
    ;;

  dev-restart)
    require_env_file
    banner "Restarting dev tools"
    podman compose -f "$COMPOSE_DEV" $COMPOSE_OPTS down || true
    podman compose -f "$COMPOSE_DEV" $COMPOSE_OPTS up -d
    echo "✅ Dev tools restarted."
    ;;

  security-up)
    require_env_file
    banner "Starting security tools"
    podman compose -f "$SECURITY_COMPOSE_FILE" $COMPOSE_OPTS up -d
    echo "✅ Security tools are up."
    ;;

  security-down)
    require_env_file
    banner "Stopping security tools"
    podman compose -f "$SECURITY_COMPOSE_FILE" $COMPOSE_OPTS down
    echo "✅ Security tools are down."
    ;;

  security-restart)
    require_env_file
    banner "Restarting security tools"
    podman compose -f "$SECURITY_COMPOSE_FILE" $COMPOSE_OPTS down || true
    podman compose -f "$SECURITY_COMPOSE_FILE" $COMPOSE_OPTS up -d
    echo "✅ Security tools restarted."
    ;;

  security-logs)
    require_env_file
    banner "Tailing security tools logs"
    podman compose -f "$SECURITY_COMPOSE_FILE" $COMPOSE_OPTS logs -f
    ;;

  status|ps)
    require_env_file
    banner "Status"
    echo "=== Base infrastructure ==="
    podman compose -f "$COMPOSE_FILE" $COMPOSE_OPTS ps
    echo ""
    echo "=== Dev tools ==="
    podman compose -f "$COMPOSE_DEV" $COMPOSE_OPTS ps 2>/dev/null || echo "(dev tools not running)"
    echo ""
    echo "=== Security tools ==="
    podman compose -f "$SECURITY_COMPOSE_FILE" $COMPOSE_OPTS ps 2>/dev/null || echo "(security tools not running)"
    ;;

  clean)
    require_env_file
    banner "Cleaning project-scoped resources (containers/networks/volumes)"
    echo "-> Stopping base infra + removing volumes/orphans..."
    podman compose -f "$COMPOSE_FILE" $COMPOSE_OPTS down -v --remove-orphans || true

    echo "-> Stopping dev tools + removing volumes/orphans..."
    podman compose -f "$COMPOSE_DEV" $COMPOSE_OPTS down -v --remove-orphans 2>/dev/null || true

    echo "-> Stopping security tools + removing volumes/orphans..."
    podman compose -f "$SECURITY_COMPOSE_FILE" $COMPOSE_OPTS down -v --remove-orphans 2>/dev/null || true

    echo "-> Pruning dangling networks/volumes (safe-ish)..."
    # If you want strictly only compose project networks/volumes, comment these out.
    podman network prune -f || true
    podman volume prune -f || true

    echo "✅ Clean complete."
    ;;

  purge)
    banner "🔥 DANGER: PURGING ALL PODMAN RESOURCES"
    echo "This will remove EVERYTHING Podman knows about for this user:"
    echo "  - all containers"
    echo "  - all images"
    echo "  - all volumes"
    echo "  - all networks"
    echo ""
    echo "Proceeding..."

    podman stop -a 2>/dev/null || true
    podman rm -a -f 2>/dev/null || true
    podman volume rm -a -f 2>/dev/null || true
    podman network ls -q | xargs -r podman network rm 2>/dev/null || true
    podman image rm -a -f 2>/dev/null || true
    podman system prune -a -f 2>/dev/null || true

    echo "✅ Podman purge complete."
    ;;

  help|*)
    usage
    ;;
esac
