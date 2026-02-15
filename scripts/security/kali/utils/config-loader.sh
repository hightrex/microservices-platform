#!/bin/bash
# scripts/security/kali/utils/config-loader.sh
#
# Loads scan configuration from scan-targets.yaml.
# Uses Python + PyYAML (both present in the Kali image) for reliable YAML parsing.

CONFIG_FILE="/pentest/config/scan-targets.yaml"

if [ -f "$CONFIG_FILE" ]; then
    # Use Python for reliable YAML parsing instead of fragile grep pipelines
    _gateway_url=$(python3 -c "
import yaml, sys
try:
    cfg = yaml.safe_load(open('$CONFIG_FILE'))
    print(cfg['targets']['api_gateway']['url'])
except Exception:
    sys.exit(1)
" 2>/dev/null) || true

    if [ -n "$_gateway_url" ]; then
        export API_GATEWAY_URL="$_gateway_url"
    else
        export API_GATEWAY_URL="${API_GATEWAY_URL:-http://api-gateway:3000}"
    fi

    # Load scan options
    _threads=$(python3 -c "
import yaml, sys
try:
    cfg = yaml.safe_load(open('$CONFIG_FILE'))
    print(cfg.get('scan_options', {}).get('threads', 5))
except Exception:
    sys.exit(1)
" 2>/dev/null) || true
    export SCAN_THREADS="${_threads:-5}"

    _timeout=$(python3 -c "
import yaml, sys
try:
    cfg = yaml.safe_load(open('$CONFIG_FILE'))
    print(cfg.get('scan_options', {}).get('timeout', 120))
except Exception:
    sys.exit(1)
" 2>/dev/null) || true
    export SCAN_TIMEOUT="${_timeout:-120}"
else
    export API_GATEWAY_URL="${API_GATEWAY_URL:-http://api-gateway:3000}"
    export SCAN_THREADS="${SCAN_THREADS:-5}"
    export SCAN_TIMEOUT="${SCAN_TIMEOUT:-120}"
fi
