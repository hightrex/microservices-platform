#!/bin/bash

# Managing infrastructure containers with Podman Compose

COMPOSE_DIR="deploy/podman"
COMPOSE_FILE="$COMPOSE_DIR/compose.base.yml"
COMPOSE_DEV="$COMPOSE_DIR/compose.dev.yml"
SECURITY_COMPOSE_FILE="$COMPOSE_DIR/compose.security.yml"
ENV_FILE=".env"

COMPOSE_OPTS="--env-file $ENV_FILE"

case "$1" in
    up)
        echo "Starting infrastructure..."
        podman compose -f "$COMPOSE_FILE" $COMPOSE_OPTS up -d
        ;;
    down)
        echo "Stopping infrastructure..."
        podman compose -f "$COMPOSE_FILE" $COMPOSE_OPTS down
        ;;
    logs)
        echo "Tailing logs..."
        podman compose -f "$COMPOSE_FILE" $COMPOSE_OPTS logs -f
        ;;
    clean)
        echo "Cleaning up infrastructure..."
        podman compose -f "$COMPOSE_FILE" $COMPOSE_OPTS down -v
        ;;
    core-up)
        echo "Starting core infrastructure (Postgres, Redis, MinIO)..."
        podman compose -f "$COMPOSE_FILE" $COMPOSE_OPTS up -d postgres redis minio
        ;;
    dev-up)
        echo "Starting dev tools (Adminer, Redis Commander)..."
        podman compose -f "$COMPOSE_FILE" $COMPOSE_OPTS up -d
        podman compose -f "$COMPOSE_DEV" $COMPOSE_OPTS up -d
        ;;
    dev-down)
        echo "Stopping dev tools..."
        podman compose -f "$COMPOSE_DEV" $COMPOSE_OPTS down
        ;;
    security-up)
        echo "Starting security tools..."
        podman compose -f "$SECURITY_COMPOSE_FILE" $COMPOSE_OPTS up -d
        ;;
    security-down)
        echo "Stopping security tools..."
        podman compose -f "$SECURITY_COMPOSE_FILE" $COMPOSE_OPTS down
        ;;
    security-logs)
        echo "Tailing security tool logs..."
        podman compose -f "$SECURITY_COMPOSE_FILE" $COMPOSE_OPTS logs -f
        ;;
    status)
        echo "=== Infrastructure ==="
        podman compose -f "$COMPOSE_FILE" $COMPOSE_OPTS ps
        echo ""
        echo "=== Security Tools ==="
        podman compose -f "$SECURITY_COMPOSE_FILE" $COMPOSE_OPTS ps 2>/dev/null || echo "(not running)"
        ;;
    *)
        echo "Usage: $0 {up|down|logs|clean|core-up|dev-up|dev-down|security-up|security-down|security-logs|status}"
        exit 1
        ;;
esac
