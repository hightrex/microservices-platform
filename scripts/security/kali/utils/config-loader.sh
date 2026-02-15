#!/bin/bash
# scripts/security/kali/utils/config-loader.sh

# Load scan targets if available
if [ -f "/pentest/config/scan-targets.yaml" ]; then
    # Simple yq-like parsing using grep/sed for basic values
    # In a real scenario we might use yq or python
    export API_GATEWAY_URL=$(grep "url:" /pentest/config/scan-targets.yaml | grep "api_gateway" -A 1 | grep "url" | awk '{print $2}')
    if [ -z "$API_GATEWAY_URL" ]; then
        export API_GATEWAY_URL="http://api-gateway:3000"
    fi
fi
