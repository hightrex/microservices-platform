#!/bin/bash
set -euo pipefail

# Managing infrastructure containers with Podman Compose
# - "clean" removes project containers + volumes + project networks
# - "purge" removes ALL podman containers/images/volumes/networks (DANGEROUS)

COMPOSE_DIR="deploy/podman"
COMPOSE_FILE="$COMPOSE_DIR/compose.base.yml"
COMPOSE_DEV="$COMPOSE_DIR/compose.dev.yml"
SECURITY_COMPOSE_FILE="$COMPOSE_DIR/compose.security.yml"
ENV_FILE=".env"

# Set a stable project name so compose resources are namespaced/predictable.
# This ensures down/clean targets only this project (unless you run 'purge').
export COMPOSE_PROJECT_NAME="${COMPOSE_PROJECT_NAME:-core-platform}"

COMPOSE_OPTS="--env-file $ENV_FILE"

require_env_file() {
  if [[ ! -f "$ENV_FILE" ]]; then
    echo "ERROR: $ENV_FILE not found. Create it or adjust ENV_FILE in scripts/manage-infra.sh"
    exit 1
  fi
}

case "${1:-}" in
  up)
    require_env_file
    echo "Starting infrastructure..."
    podman compose -f "$COMPOSE_FILE" $COMPOSE_OPTS up -d
    ;;

  down)
    require_env_file
    echo "Stopping infrastructure..."
    podman compose -f "$COMPOSE_FILE" $COMPOSE_OPTS down
    ;;

  logs)
    require_env_file
    echo "Tailing logs..."
    podman compose -f "$COMPOSE_FILE" $COMPOSE_OPTS logs -f
    ;;

  clean)
    require_env_file
    echo "Cleaning up infrastructure (project-scoped containers, networks, volumes)..."
    # Remove compose-managed containers, project networks, and volumes
    podman compose -f "$COMPOSE_FILE" $COMPOSE_OPTS down -v --remove-orphans || true

    # Also stop dev + security stacks if they exist (project-scoped)
    podman compose -f "$COMPOSE_DEV" $COMPOSE_OPTS down -v --remove-orphans 2>/dev/null || true
    podman compose -f "$SECURITY_COMPOSE_FILE" $COMPOSE_OPTS down -v --remove-orphans 2>/dev/null || true

    # Prune dangling networks/volumes created by this user (safe-ish)
    # If you want strictly only compose project networks, comment these out.
    podman network prune -f || true
    podman volume prune -f || true

    echo "✅ Clean complete for COMPOSE_PROJECT_NAME=$COMPOSE_PROJECT_NAME"
    ;;

  core-up)
    require_env_file
    echo "Starting core infrastructure (Postgres, Redis, MinIO)..."
    podman compose -f "$COMPOSE_FILE" $COMPOSE_OPTS up -d postgres redis minio
    ;;

  dev-up)
    require_env_file
    echo "Starting dev tools (Adminer, Redis Commander)..."
    podman compose -f "$COMPOSE_FILE" $COMPOSE_OPTS up -d
    podman compose -f "$COMPOSE_DEV" $COMPOSE_OPTS up -d
    ;;

  dev-down)
    require_env_file
    echo "Stopping dev tools..."
    podman compose -f "$COMPOSE_DEV" $COMPOSE_OPTS down
    ;;

  security-up)
    require_env_file
    echo "Starting security tools..."
    podman compose -f "$SECURITY_COMPOSE_FILE" $COMPOSE_OPTS up -d
    ;;

  security-down)
    require_env_file
    echo "Stopping security tools..."
    podman compose -f "$SECURITY_COMPOSE_FILE" $COMPOSE_OPTS down
    ;;

  security-logs)
    require_env_file
    echo "Tailing security tool logs..."
    podman compose -f "$SECURITY_COMPOSE_FILE" $COMPOSE_OPTS logs -f
    ;;

  status)
    require_env_file
    echo "=== Infrastructure (project: $COMPOSE_PROJECT_NAME) ==="
    podman compose -f "$COMPOSE_FILE" $COMPOSE_OPTS ps
    echo ""
    echo "=== Security Tools (project: $COMPOSE_PROJECT_NAME) ==="
    podman compose -f "$SECURITY_COMPOSE_FILE" $COMPOSE_OPTS ps 2>/dev/null || echo "(not running)"
    ;;

  purge)
    echo "🔥 DANGER: PURGING ALL PODMAN RESOURCES (containers/images/volumes/networks)"
    echo "This will remove EVERYTHING Podman knows about for this user."
    echo "Proceeding..."

    # Stop and remove all containers
    podman stop -a 2>/dev/null || true
    podman rm -a -f 2>/dev/null || true

    # Remove all volumes
    podman volume rm -a -f 2>/dev/null || true

    # Remove all non-default networks (podman has default ones that may fail removal)
    # This attempts removal and ignores failures.
    podman network ls -q | xargs -r podman network rm 2>/dev/null || true

    # Remove all images
    podman image rm -a -f 2>/dev/null || true

    # Prune build cache + anything dangling
    podman system prune -a -f 2>/dev/null || true

    echo "✅ Podman purge complete."
    ;;

  *)
    echo "Usage: $0 {up|down|logs|clean|purge|core-up|dev-up|dev-down|security-up|security-down|security-logs|status}"
    echo ""
    echo "Notes:"
    echo "  - clean: removes THIS compose project (COMPOSE_PROJECT_NAME=$COMPOSE_PROJECT_NAME) + prunes dangling nets/vols"
    echo "  - purge: removes EVERYTHING podman has (use only when you're totally stuck)"
    exit 1
    ;;
esac
