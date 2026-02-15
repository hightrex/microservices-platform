#!/bin/bash
# scripts/security/kali/modules/01-reconnaissance.sh
set -euo pipefail
RESULTS_DIR="/pentest/results/recon"
mkdir -p "$RESULTS_DIR"
TARGET_HOST="api-gateway"
log() { echo "[$(date +'%H:%M:%S')] $*"; }

log "Starting Nmap scan on $TARGET_HOST..."
nmap -sV -p- --open --unprivileged -oX "$RESULTS_DIR/nmap-scan.xml" "$TARGET_HOST" > "$RESULTS_DIR/nmap-scan.txt"
log "Recon complete"
