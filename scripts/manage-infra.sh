#!/bin/bash

# Managing infrastructure containers with Podman Compose

COMPOSE_FILE="deploy/podman/compose.base.yml"
SECURITY_COMPOSE_FILE="deploy/podman/compose.security.yml"

case "$1" in
    up)
        echo "Starting infrastructure..."
        podman compose -f "$COMPOSE_FILE" up -d
        ;;
    down)
        echo "Stopping infrastructure..."
        podman compose -f "$COMPOSE_FILE" down
        ;;
    logs)
        echo "Tailing logs..."
        podman compose -f "$COMPOSE_FILE" logs -f
        ;;
    clean)
        echo "Cleaning up infrastructure..."
        podman compose -f "$COMPOSE_FILE" down -v
        ;;
    security-up)
        echo "Starting security tools..."
        podman compose -f "$SECURITY_COMPOSE_FILE" up -d
        ;;
    security-down)
        echo "Stopping security tools..."
        podman compose -f "$SECURITY_COMPOSE_FILE" down
        ;;
    *)
        echo "Usage: $0 {up|down|logs|clean|security-up|security-down}"
        exit 1
        ;;
esac
