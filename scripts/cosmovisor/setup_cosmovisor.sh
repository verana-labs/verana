#!/usr/bin/env bash

# Script to setup cosmovisor directory structure
# Prerequisites:
# 1. Current binary must be in PATH (veranad)
# 2. Upgrade binary must be manually downloaded and placed in the upgrades directory
# Usage: ./scripts/cosmovisor/setup_cosmovisor.sh

set -e

HOME_DIR="${HOME_DIR:-$HOME/.verana}"
BINARY_NAME="${BINARY_NAME:-veranad}"
UPGRADE_NAME="${UPGRADE_NAME:-v0.9}"

echo "=== Setting up Cosmovisor ==="
echo "Home Directory: $HOME_DIR"
echo "Binary Name: $BINARY_NAME"
echo "Upgrade Name: $UPGRADE_NAME"
echo ""

# Check if cosmovisor is installed
if ! command -v cosmovisor &> /dev/null; then
    echo "Error: cosmovisor is not installed."
    echo "Please install it first:"
    echo "  go install cosmossdk.io/tools/cosmovisor/cmd/cosmovisor@latest"
    exit 1
fi

echo "Cosmovisor found: $(which cosmovisor)"
echo ""

# Create cosmovisor directory structure
echo "Creating cosmovisor directory structure..."
mkdir -p "$HOME_DIR/cosmovisor"
mkdir -p "$HOME_DIR/cosmovisor/genesis/bin"
mkdir -p "$HOME_DIR/cosmovisor/upgrades/${UPGRADE_NAME}/bin"

# Copy current binary to genesis
if [ -f "$(which $BINARY_NAME)" ]; then
    CURRENT_BINARY=$(which $BINARY_NAME)
    echo "Copying current binary ($CURRENT_BINARY) to genesis..."
    cp "$CURRENT_BINARY" "$HOME_DIR/cosmovisor/genesis/bin/$BINARY_NAME"
    chmod +x "$HOME_DIR/cosmovisor/genesis/bin/$BINARY_NAME"
    echo "✓ Genesis binary installed"
else
    echo "Error: Current binary not found. Please ensure $BINARY_NAME is in PATH"
    exit 1
fi

# Check if upgrade binary exists
if [ ! -f "$HOME_DIR/cosmovisor/upgrades/${UPGRADE_NAME}/bin/$BINARY_NAME" ]; then
    echo ""
    echo "⚠️  Upgrade binary not found!"
    echo "Please download the upgrade binary and place it at:"
    echo "  $HOME_DIR/cosmovisor/upgrades/${UPGRADE_NAME}/bin/$BINARY_NAME"
    echo ""
    echo "To download:"
    echo "  1. Go to: https://github.com/verana-labs/verana/releases/tag/v0.9-dev.3"
    echo "  2. Download the binary for your platform (darwin-arm64 for Mac M1/M2, darwin-amd64 for Intel Mac)"
    echo "  3. Place it at the path above"
    echo ""
    read -p "Press Enter once you've placed the binary, or Ctrl+C to cancel..."
    
    if [ ! -f "$HOME_DIR/cosmovisor/upgrades/${UPGRADE_NAME}/bin/$BINARY_NAME" ]; then
        echo "Error: Binary still not found at expected location"
        exit 1
    fi
fi

# Make binary executable and remove macOS quarantine attribute
if [ -f "$HOME_DIR/cosmovisor/upgrades/${UPGRADE_NAME}/bin/$BINARY_NAME" ]; then
    echo "Setting up upgrade binary..."
    
    # Remove macOS quarantine attribute (allows execution without Gatekeeper warning)
    if [[ "$OSTYPE" == "darwin"* ]]; then
        # Remove quarantine attribute if present
        xattr -d com.apple.quarantine "$HOME_DIR/cosmovisor/upgrades/${UPGRADE_NAME}/bin/$BINARY_NAME" 2>/dev/null || true
        echo "✓ Removed macOS quarantine attribute"
    fi
    
    # Make executable
    chmod +x "$HOME_DIR/cosmovisor/upgrades/${UPGRADE_NAME}/bin/$BINARY_NAME"
    
    # Verify it's executable
    if [ -x "$HOME_DIR/cosmovisor/upgrades/${UPGRADE_NAME}/bin/$BINARY_NAME" ]; then
        echo "✓ Upgrade binary is executable"
    else
        echo "⚠️  Warning: Binary may not be executable. If you get 'killed' error, see README for macOS security settings."
    fi
fi

# Create cosmovisor environment file
ENV_FILE="$HOME_DIR/cosmovisor/cosmovisor.env"
cat > "$ENV_FILE" <<EOF
# Cosmovisor environment variables
export DAEMON_NAME=$BINARY_NAME
export DAEMON_HOME=$HOME_DIR
export DAEMON_RESTART_AFTER_UPGRADE=true
export DAEMON_ALLOW_DOWNLOAD_BINARIES=false
EOF

echo ""
echo "✓ Cosmovisor environment file created at: $ENV_FILE"

# Create start script
START_SCRIPT="$HOME_DIR/cosmovisor/start.sh"
cat > "$START_SCRIPT" <<'EOF'
#!/usr/bin/env bash
# Start cosmovisor with the chain

set -e

source "$HOME/.verana/cosmovisor/cosmovisor.env"

exec cosmovisor run start
EOF

chmod +x "$START_SCRIPT"
echo "✓ Start script created at: $START_SCRIPT"

echo ""
echo "=== Cosmovisor Setup Complete ==="
echo ""
echo "Directory structure:"
echo "  $HOME_DIR/cosmovisor/genesis/bin/$BINARY_NAME (current version)"
echo "  $HOME_DIR/cosmovisor/upgrades/${UPGRADE_NAME}/bin/$BINARY_NAME (upgrade version)"
echo ""
echo "To start the chain with cosmovisor:"
echo "  1. Stop your current chain (if running)"
echo "  2. Run: $START_SCRIPT"
echo "  Or manually: source $ENV_FILE && cosmovisor run start"

