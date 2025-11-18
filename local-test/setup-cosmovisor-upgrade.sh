#!/bin/bash

# Script to prepare cosmovisor for upgrade testing in local-test environment
# This script should be run after setup-validators.sh to prepare for an upgrade

set -e

UPGRADE_NAME="${UPGRADE_NAME:-v0.9}"
UPGRADE_BINARY_PATH="${UPGRADE_BINARY_PATH:-}"

if [ -z "$UPGRADE_BINARY_PATH" ]; then
    echo "Error: UPGRADE_BINARY_PATH must be set"
    echo "Usage: UPGRADE_BINARY_PATH=/path/to/veranad-binary ./setup-cosmovisor-upgrade.sh"
    echo ""
    echo "Note: Docker containers run Linux, so download linux-amd64 or linux-arm64 (not darwin!)"
    echo "Example: UPGRADE_BINARY_PATH=./veranad-linux-amd64 ./setup-cosmovisor-upgrade.sh"
    exit 1
fi

if [ ! -f "$UPGRADE_BINARY_PATH" ]; then
    echo "Error: Upgrade binary not found at: $UPGRADE_BINARY_PATH"
    exit 1
fi

echo "=== Setting up Cosmovisor Upgrade ==="
echo "Upgrade Name: $UPGRADE_NAME"
echo "Upgrade Binary: $UPGRADE_BINARY_PATH"
echo ""

# Copy upgrade binary to each validator's cosmovisor directory
for i in {1..5}; do
    validator="validator$i"
    
    if [ ! -d "$validator" ]; then
        echo "Warning: $validator directory not found, skipping..."
        continue
    fi
    
    echo "Setting up upgrade for $validator..."
    
    # Create upgrade directory
    mkdir -p "$validator/cosmovisor/upgrades/${UPGRADE_NAME}/bin"
    
    # Copy upgrade binary
    cp "$UPGRADE_BINARY_PATH" "$validator/cosmovisor/upgrades/${UPGRADE_NAME}/bin/veranad"
    chmod +x "$validator/cosmovisor/upgrades/${UPGRADE_NAME}/bin/veranad"
    
    echo "✓ $validator upgrade binary installed"
done

echo ""
echo "=== Cosmovisor Upgrade Setup Complete ==="
echo ""
echo "All validators are now ready for upgrade testing."
echo "The upgrade binary is installed at:"
for i in {1..5}; do
    validator="validator$i"
    if [ -d "$validator" ]; then
        echo "  $validator/cosmovisor/upgrades/${UPGRADE_NAME}/bin/veranad"
    fi
done
echo ""
echo "Next steps:"
echo "1. Query current block height:"
echo "   docker exec validator1 veranad status --node tcp://localhost:26657 | jq -r '.sync_info.latest_block_height'"
echo ""
echo "2. Submit upgrade proposal and vote (from all validators):"
echo "   ./local-test/submit-upgrade-proposal-docker.sh <upgrade-height> <release-tag>"
echo "   Example: ./local-test/submit-upgrade-proposal-docker.sh 400 v0.9-dev.3"
echo ""
echo "3. When the upgrade height is reached, cosmovisor will automatically switch binaries"
echo ""
echo "Note: The binary has been made executable automatically (chmod +x)"

