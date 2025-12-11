#!/usr/bin/env bash

# Simple script to query current block height
# Usage: ./scripts/cosmovisor/query_height.sh

set -e

CHAIN_ID="${CHAIN_ID:-vna-testnet-1}"
BINARY="${BINARY:-veranad}"
NODE="${NODE:-tcp://localhost:26657}"

echo "Querying chain status..."
echo "CHAIN_ID="${CHAIN_ID}"
echo "NODE=${NODE}"
STATUS_JSON=$($BINARY status --node $NODE 2>/dev/null)
if [ $? -eq 0 ] && [ -n "$STATUS_JSON" ]; then
    HEIGHT=$(echo "$STATUS_JSON" | jq -r '.sync_info.latest_block_height // .latest_block_height // empty')
    if [ -n "$HEIGHT" ] && [ "$HEIGHT" != "null" ] && [ "$HEIGHT" != "" ]; then
        echo "Current Block Height: $HEIGHT"
        echo "$HEIGHT" > /tmp/current_height.txt
        echo "Height saved to /tmp/current_height.txt"
    else
        echo "Error: Could not extract block height"
        exit 1
    fi
else
    echo "Error: Could not query chain status"
    exit 1
fi

