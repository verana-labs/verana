#!/bin/bash

# Build Docker image with cosmovisor support
# Usage: ./local-test/build-cosmovisor.sh
# Or from project root: ./local-test/build-cosmovisor.sh

set -e

# Get the directory where this script is located
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(cd "$SCRIPT_DIR/.." && pwd)"

DOCKER_IMAGE="verana:dev-cosmovisor"

echo "Building Docker image with cosmovisor: $DOCKER_IMAGE"
cd "$PROJECT_ROOT"
docker build -f local-test/Dockerfile.cosmovisor -t $DOCKER_IMAGE .

if [ $? -eq 0 ]; then
    echo ""
    echo "✓ Docker image built successfully: $DOCKER_IMAGE"
    echo ""
    echo "To use this image, update setup-validators.sh to use:"
    echo "  DOCKER_IMAGE=\"$DOCKER_IMAGE\""
else
    echo "❌ Build failed"
    exit 1
fi

