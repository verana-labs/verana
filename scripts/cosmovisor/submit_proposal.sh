#!/usr/bin/env bash

# Script to submit a software upgrade proposal
# Usage: ./scripts/cosmovisor/submit_proposal.sh <upgrade-height> <upgrade-name>
# Example: ./scripts/cosmovisor/submit_proposal.sh 400 v0.9-dev.3

set -e

if [ $# -lt 2 ]; then
    echo "Usage: $0 <upgrade-height> <upgrade-name>"
    echo "Example: $0 400 v0.9-dev.3"
    exit 1
fi

UPGRADE_HEIGHT=$1
UPGRADE_NAME=$2
CHAIN_ID="${CHAIN_ID:-vna-testnet-1}"
BINARY="${BINARY:-veranad}"
NODE="${NODE:-tcp://localhost:26657}"
FROM="${FROM:-cooluser}"
KEYRING_BACKEND="${KEYRING_BACKEND:-test}"
DEPOSIT="${DEPOSIT:-10000000uvna}"
FEES="${FEES:-600000uvna}"

echo "=== Submitting Software Upgrade Proposal ==="
echo "Upgrade Name: $UPGRADE_NAME"
echo "Upgrade Height: $UPGRADE_HEIGHT"
echo "Chain ID: $CHAIN_ID"
echo "From: $FROM"
echo ""

# Detect architecture
ARCH=$(uname -m)
if [ "$ARCH" = "x86_64" ]; then
    ARCH="amd64"
elif [ "$ARCH" = "arm64" ] || [ "$ARCH" = "aarch64" ]; then
    ARCH="arm64"
fi

# Use v0.9 as the upgrade plan name (matches constants.go)
UPGRADE_PLAN_NAME="v0.9"

# Get the gov module address
GOV_AUTHORITY=$($BINARY query auth module-accounts --node $NODE --output json 2>/dev/null | jq -r '.accounts[] | select(.name == "gov") | .address' 2>/dev/null || echo "")
if [ -z "$GOV_AUTHORITY" ] || [ "$GOV_AUTHORITY" = "null" ]; then
    GOV_AUTHORITY=$($BINARY query auth module-accounts --node $NODE 2>/dev/null | grep -B 1 "name: gov" | grep "address:" | awk '{print $2}' || echo "")
fi
if [ -z "$GOV_AUTHORITY" ] || [ "$GOV_AUTHORITY" = "null" ]; then
    echo "Error: Could not determine gov module address"
    exit 1
fi
echo "Using gov authority: $GOV_AUTHORITY"

# Create metadata JSON (keep it short)
METADATA_JSON=$(cat <<EOF | jq -c .
{
  "title": "Upgrade to $UPGRADE_NAME",
  "summary": "Upgrade to $UPGRADE_NAME at height $UPGRADE_HEIGHT"
}
EOF
)
METADATA_B64=$(echo "$METADATA_JSON" | base64 | tr -d '\n')

# Create the proposal JSON
PROPOSAL_FILE="/tmp/upgrade_proposal_${UPGRADE_NAME}.json"
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
        "info": "{\"binaries\":{\"darwin/$ARCH\":\"https://github.com/verana-labs/verana/releases/download/$UPGRADE_NAME/veranad-$UPGRADE_NAME-darwin-$ARCH\"}}",
        "upgraded_client_state": null
      }
    }
  ],
  "metadata": "$METADATA_B64",
  "deposit": "$DEPOSIT",
  "title": "Upgrade to $UPGRADE_NAME",
  "summary": "Software upgrade to $UPGRADE_NAME at height $UPGRADE_HEIGHT",
  "expedited": false
}
EOF

echo ""
echo "Submitting proposal..."
$BINARY tx gov submit-proposal "$PROPOSAL_FILE" \
    --from "$FROM" \
    --chain-id "$CHAIN_ID" \
    --keyring-backend "$KEYRING_BACKEND" \
    --gas auto \
    --gas-adjustment 1.5 \
    --fees "$FEES" \
    --node "$NODE" \
    --yes

echo ""
echo "Proposal submitted! Waiting for transaction to be included..."
sleep 5

# Get the proposal ID
PROPOSAL_ID=$($BINARY query gov proposals --node $NODE --limit 1 --reverse --output json 2>/dev/null | jq -r '.proposals[0].id // empty' || echo "")
if [ -n "$PROPOSAL_ID" ] && [ "$PROPOSAL_ID" != "null" ] && [ "$PROPOSAL_ID" != "" ]; then
    echo "Proposal ID: $PROPOSAL_ID"
    echo "$PROPOSAL_ID" > /tmp/proposal_id.txt
    echo "Proposal ID saved to /tmp/proposal_id.txt"
else
    echo "Warning: Could not fetch proposal ID. Check manually:"
    echo "  $BINARY query gov proposals --node $NODE"
fi

