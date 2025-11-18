#!/usr/bin/env bash

# Script to submit and vote on a software upgrade proposal in the Docker local-test environment
# Usage: ./local-test/submit-upgrade-proposal-docker.sh <upgrade-height> <upgrade-name-release-tag>
# Example: ./local-test/submit-upgrade-proposal-docker.sh 400 v0.9-dev.3

set -e

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

print_status() {
    echo -e "${GREEN}[INFO]${NC} $1"
}

print_warning() {
    echo -e "${YELLOW}[WARNING]${NC} $1"
}

print_error() {
    echo -e "${RED}[ERROR]${NC} $1"
}

if [ $# -lt 2 ]; then
    echo "Usage: $0 <upgrade-height> <upgrade-name-release-tag>"
    echo "Example: $0 400 v0.9-dev.3"
    exit 1
fi

UPGRADE_HEIGHT=$1
UPGRADE_NAME_RELEASE_TAG=$2  # e.g., v0.9-dev.3
UPGRADE_PLAN_NAME="v0.9"     # Upgrade plan name (matches constants.go)
CHAIN_ID="vna-local-1"
VALIDATOR="validator1"        # Use validator1 for submitting proposal
WALLET="wallet1"              # Use wallet1 for submitting proposal
KEYRING_BACKEND="test"
DEPOSIT="10000000uvna"
FEES="600000uvna"
GAS="auto"
GAS_ADJUSTMENT="1.5"

# Get RPC port for validator1 (first validator) - for host queries
# Inside containers, we use localhost:26657, but for host queries we need the mapped port
HOST_RPC_PORT=$(docker port $VALIDATOR 26657/tcp 2>/dev/null | cut -d: -f2)
if [ -z "$HOST_RPC_PORT" ]; then
    print_error "Could not determine RPC port for $VALIDATOR"
    print_error "Make sure the validator container is running"
    exit 1
fi

# For commands inside containers, use localhost:26657 (container's internal port)
# For queries from host, use the mapped port
NODE_HOST="tcp://localhost:$HOST_RPC_PORT"  # For host queries
NODE_CONTAINER="tcp://localhost:26657"       # For container commands

print_status "=== Submitting Software Upgrade Proposal ==="
print_status "Upgrade Name (Release Tag): $UPGRADE_NAME_RELEASE_TAG"
print_status "Upgrade Plan Name: $UPGRADE_PLAN_NAME"
print_status "Upgrade Height: $UPGRADE_HEIGHT"
print_status "Chain ID: $CHAIN_ID"
print_status "From: $WALLET (in $VALIDATOR)"
print_status "RPC Node (host): $NODE_HOST"
print_status "RPC Node (container): $NODE_CONTAINER"
print_status ""

# Detect architecture for binary URL (Linux for Docker, not host OS)
# Docker containers run Linux, so we need linux-amd64 or linux-arm64
DOCKER_ARCH=$(docker exec $VALIDATOR uname -m 2>/dev/null || echo "x86_64")
if [ "$DOCKER_ARCH" = "x86_64" ]; then
    DOCKER_ARCH="amd64"
elif [ "$DOCKER_ARCH" = "aarch64" ] || [ "$DOCKER_ARCH" = "arm64" ]; then
    DOCKER_ARCH="arm64"
else
    DOCKER_ARCH="amd64"  # Default to amd64
fi

print_status "Docker container architecture: $DOCKER_ARCH"
print_status "Binary URL will use: linux-$DOCKER_ARCH"
print_status ""

# Get the gov module address (authority for upgrades)
print_status "Querying gov module address..."
GOV_AUTHORITY=$(docker exec $VALIDATOR veranad query auth module-accounts --node $NODE_CONTAINER --output json 2>/dev/null | jq -r '.accounts[] | select(.value.name == "gov") | .value.address' 2>/dev/null || echo "")
if [ -z "$GOV_AUTHORITY" ] || [ "$GOV_AUTHORITY" = "null" ]; then
    # Fallback: try text output parsing
    print_warning "JSON query failed, trying text output..."
    GOV_AUTHORITY=$(docker exec $VALIDATOR veranad query auth module-accounts --node $NODE_CONTAINER 2>/dev/null | grep -A 5 "name: gov" | grep "address:" | awk '{print $2}' || echo "")
    if [ -z "$GOV_AUTHORITY" ]; then
        print_error "Could not determine gov module address"
        print_error "Make sure the chain is running and accessible"
        exit 1
    fi
fi
print_status "Gov authority: $GOV_AUTHORITY"

# Create metadata JSON (keep it short to avoid "metadata too long" error)
METADATA_JSON=$(cat <<EOF | jq -c .
{
  "title": "Upgrade to $UPGRADE_NAME_RELEASE_TAG",
  "summary": "Upgrade to $UPGRADE_NAME_RELEASE_TAG at height $UPGRADE_HEIGHT"
}
EOF
)

# Base64 encode metadata
METADATA_B64=$(echo "$METADATA_JSON" | base64 | tr -d '\n')

# Create the proposal JSON with software upgrade message
PROPOSAL_FILE="/tmp/upgrade_proposal_${UPGRADE_NAME_RELEASE_TAG}.json"
cat > "$PROPOSAL_FILE" <<EOF
{
  "messages": [
    {
      "@type": "/cosmos.upgrade.v1beta1.MsgSoftwareUpgrade",
      "authority": "$GOV_AUTHORITY",
      "plan": {
        "name": "$UPGRADE_PLAN_NAME",
        "time": "0001-01-01T00:00:00Z",
        "height": "$UPGRADE_HEIGHT",
        "info": "{\"binaries\":{\"linux/$DOCKER_ARCH\":\"https://github.com/verana-labs/verana/releases/download/$UPGRADE_NAME_RELEASE_TAG/veranad-$UPGRADE_NAME_RELEASE_TAG-linux-$DOCKER_ARCH\"}}",
        "upgraded_client_state": null
      }
    }
  ],
  "metadata": "$METADATA_B64",
  "deposit": "$DEPOSIT",
  "title": "Upgrade to $UPGRADE_NAME_RELEASE_TAG",
  "summary": "Software upgrade to $UPGRADE_NAME_RELEASE_TAG at height $UPGRADE_HEIGHT",
  "expedited": false
}
EOF

print_status "Proposal JSON created at: $PROPOSAL_FILE"
print_status ""

# Copy proposal file into container
docker cp "$PROPOSAL_FILE" "$VALIDATOR:/tmp/upgrade_proposal.json"

# Submit the proposal
print_status "Submitting proposal from $VALIDATOR..."
OUTPUT=$(docker exec $VALIDATOR veranad tx gov submit-proposal /tmp/upgrade_proposal.json \
    --from "$WALLET" \
    --chain-id "$CHAIN_ID" \
    --keyring-backend "$KEYRING_BACKEND" \
    --gas "$GAS" \
    --gas-adjustment "$GAS_ADJUSTMENT" \
    --fees "$FEES" \
    --node "$NODE_CONTAINER" \
    --yes 2>&1)
echo "$OUTPUT"

print_status ""
print_status "Proposal submitted! Waiting for transaction to be included in a block..."

# Wait for transaction to be included in a block
TXHASH=$(echo "$OUTPUT" | grep -o 'txhash: [0-9A-F]\+' | awk '{print $2}' || echo "")
if [ -n "$TXHASH" ]; then
    print_status "Transaction hash: $TXHASH"
    print_status "Waiting for transaction to be included in a block..."
    
    # Wait up to 30 seconds for transaction to be included
    TX_FOUND=false
    for i in {1..15}; do
        sleep 2
        # Use set +e temporarily to prevent exit on error
        set +e
        TX_QUERY=$(docker exec $VALIDATOR veranad query tx $TXHASH --node $NODE_CONTAINER --output json 2>/dev/null)
        TX_QUERY_EXIT=$?
        set -e
        
        if [ $TX_QUERY_EXIT -eq 0 ] && [ -n "$TX_QUERY" ]; then
            set +e
            TX_HEIGHT=$(echo "$TX_QUERY" | jq -r '.height // "0"' 2>/dev/null || echo "0")
            TX_CODE=$(echo "$TX_QUERY" | jq -r '.code // "null"' 2>/dev/null || echo "null")
            set -e
            
            # Check if transaction is included (height > 0)
            if [ "$TX_HEIGHT" != "0" ] && [ "$TX_HEIGHT" != "null" ] && [ -n "$TX_HEIGHT" ]; then
                TX_FOUND=true
                if [ "$TX_CODE" = "0" ] || [ "$TX_CODE" = "null" ]; then
                    print_status "Transaction included in block at height $TX_HEIGHT"
                    break
                else
                    print_error "Transaction failed with code: $TX_CODE"
                    set +e
                    TX_ERROR=$(echo "$TX_QUERY" | jq -r '.raw_log // ""' 2>/dev/null || echo "")
                    set -e
                    if [ -n "$TX_ERROR" ] && [ "$TX_ERROR" != "null" ]; then
                        print_error "Error details: $TX_ERROR"
                    fi
                    exit 1
                fi
            fi
        fi
        
        if [ $i -eq 15 ]; then
            print_warning "Transaction not found after 30 seconds, continuing anyway..."
            print_warning "The transaction may still be pending. Check manually:"
            print_warning "  docker exec $VALIDATOR veranad query tx $TXHASH --node $NODE_CONTAINER"
        fi
    done
else
    print_warning "Could not extract transaction hash, waiting 10 seconds..."
    sleep 10
fi

# Get the proposal ID - try multiple methods
print_status "Fetching proposal ID..."
# Wait a bit for proposal to be indexed
sleep 3

set +e
PROPOSAL_ID=$(docker exec $VALIDATOR veranad query gov proposals --node $NODE_CONTAINER --output json 2>/dev/null | jq -r '.proposals | sort_by(.id | tonumber) | reverse | .[0].id' 2>/dev/null || echo "")
set -e

if [ -z "$PROPOSAL_ID" ] || [ "$PROPOSAL_ID" = "null" ]; then
    # Try without reverse/sort
    set +e
    PROPOSAL_ID=$(docker exec $VALIDATOR veranad query gov proposals --node $NODE_CONTAINER --output json 2>/dev/null | jq -r '.proposals[0].id' 2>/dev/null || echo "")
    set -e
fi
if [ -z "$PROPOSAL_ID" ] || [ "$PROPOSAL_ID" = "null" ]; then
    # Try getting the highest ID
    set +e
    PROPOSAL_ID=$(docker exec $VALIDATOR veranad query gov proposals --node $NODE_CONTAINER --output json 2>/dev/null | jq -r '.proposals | map(.id | tonumber) | max | tostring' 2>/dev/null || echo "")
    set -e
fi

if [ -n "$PROPOSAL_ID" ] && [ "$PROPOSAL_ID" != "null" ] && [ "$PROPOSAL_ID" != "" ]; then
    print_status "Proposal ID: $PROPOSAL_ID"
    echo "$PROPOSAL_ID" > /tmp/proposal_id.txt
    print_status "Proposal ID saved to /tmp/proposal_id.txt"
    print_status ""
    
    # Vote from all validators
    print_status "=== Voting on Proposal from All Validators ==="
    print_warning "Voting period is 100 seconds - voting immediately!"
    print_status ""
    
    for i in {1..5}; do
        validator="validator$i"
        wallet="wallet$i"
        
        # Check if validator container exists
        if ! docker ps --format '{{.Names}}' | grep -q "^${validator}$"; then
            print_warning "$validator container not running, skipping vote"
            continue
        fi
        
        # Use localhost:26657 inside container (container's internal port)
        val_node="tcp://localhost:26657"
        
        print_status "Voting from $validator ($wallet)..."
        set +e
        VOTE_OUTPUT=$(docker exec $validator veranad tx gov vote $PROPOSAL_ID yes \
            --from "$wallet" \
            --chain-id "$CHAIN_ID" \
            --keyring-backend "$KEYRING_BACKEND" \
            --fees "$FEES" \
            --node "$val_node" \
            --yes 2>&1)
        VOTE_EXIT=$?
        set -e
        
        # Check if vote was successful
        if [ $VOTE_EXIT -eq 0 ] && echo "$VOTE_OUTPUT" | grep -q "code: 0"; then
            print_status "✓ $validator voted successfully"
        elif echo "$VOTE_OUTPUT" | grep -qi "already voted"; then
            print_warning "⚠ $validator already voted (this is OK)"
        else
            print_warning "⚠ $validator vote may have failed (exit code: $VOTE_EXIT)"
            echo "$VOTE_OUTPUT" | tail -10
        fi
        sleep 2
    done
    
    print_status ""
    print_status "=== Voting Complete ==="
    print_status ""
    print_status "To check proposal status:"
    print_status "  docker exec $VALIDATOR veranad query gov proposal $PROPOSAL_ID --node $NODE_CONTAINER"
    print_status ""
    print_status "To monitor upgrade:"
    print_status "  docker logs $VALIDATOR -f"
    print_status ""
    print_status "The upgrade will automatically execute at height $UPGRADE_HEIGHT"
else
    print_error "Could not fetch proposal ID. Please check manually:"
    print_error "  docker exec $VALIDATOR veranad query gov proposals --node $NODE_CONTAINER"
    exit 1
fi

