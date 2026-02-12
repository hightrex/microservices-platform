#!/bin/bash
set -e

# Colors
GREEN='\033[0;32m'
RED='\033[0;31m'
NC='\033[0m' # No Color

echo -e "${GREEN}Setting up development environment...${NC}"

# Check dependencies
check_cmd() {
    if ! command -v "$1" &> /dev/null; then
        echo -e "${RED}Error: $1 is not installed.${NC}"
        exit 1
    fi
}

check_cmd go
check_cmd podman
check_cmd make

# Install Go tools
echo "Installing Go tools..."
go install github.com/air-verse/air@latest
go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest
go install github.com/securego/gosec/v2/cmd/gosec@latest
go install github.com/vektra/mockery/v2@latest

# Setup git hooks
if [ -d ".git" ]; then
    echo "Setting up git hooks..."
    # Add gitleaks pre-commit if installed
    if command -v gitleaks &> /dev/null; then
        # This is a simplified hook setup. In a real scenario we might use pre-commit framework
        echo -e "#!/bin/bash\ngitleaks detect --source . -v" > .git/hooks/pre-commit
        chmod +x .git/hooks/pre-commit
    fi
fi

# Create env file if not exists
if [ ! -f .env ]; then
    cp .env.example .env
    echo "Created .env from .env.example"
fi

echo -e "${GREEN}Setup complete!${NC}"
