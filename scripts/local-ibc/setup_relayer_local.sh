#!/usr/bin/env bash
set -euo pipefail

# Sets up the Go relayer (rly) between two local Verana chains:
# - Chain A: vna-testnet-1 (RPC :26657, home ~/.verana)
# - Chain B: vna-testnet-2 (RPC :36657, home ~/.verana-b)

log() { echo "$(date '+%Y-%m-%d %H:%M:%S') - $*"; }

if [[ "${OSTYPE:-}" == darwin* ]]; then
  SED="sed -i ''"
else
  SED="sed -i"
fi

# Binaries and versions
RLY_BIN="rly"
RLY_VERSION="v2.4.2"

# Chain metadata
CHAIN_A_ID="vna-testnet-1"
CHAIN_B_ID="vna-testnet-2"
CHAIN_A_RPC="http://localhost:26657"
CHAIN_B_RPC="http://localhost:36657"
ACC_PREFIX="verana"
GAS_PRICES="0.25uvna"
GAS_ADJ="1.3"
TIMEOUT="20s"

# Keys for relayer
RELAYER_A_NAME="relayera"
RELAYER_B_NAME="relayerb"

# Optional mnemonics (leave empty to auto-generate in rly)
RELAYER_A_MNEMONIC=""
RELAYER_B_MNEMONIC=""

# Funding amounts
RELAYER_A_FUND="50000000000000000000uvna" # 5e19 uvna

ensure_rly_installed() {
  if ! command -v "$RLY_BIN" >/dev/null 2>&1; then
    log "Installing relayer $RLY_VERSION..."
    TMPDIR="$(mktemp -d)"
    pushd "$TMPDIR" >/dev/null
    git clone https://github.com/cosmos/relayer.git
    cd relayer
    git checkout "$RLY_VERSION"
    make install
    popd >/dev/null
    rm -rf "$TMPDIR"
  else
    log "Relayer already installed: $($RLY_BIN version 2>/dev/null || echo present)"
  fi
}

write_chain_json() {
  local chain_id="$1" rpc_addr="$2" out_file="$3"
  cat >"$out_file" <<EOF
{
  "type": "cosmos",
  "value": {
    "key": "default",
    "chain-id": "$chain_id",
    "rpc-addr": "$rpc_addr",
    "account-prefix": "$ACC_PREFIX",
    "keyring-backend": "test",
    "gas-adjustment": $GAS_ADJ,
    "gas-prices": "$GAS_PRICES",
    "debug": true,
    "timeout": "$TIMEOUT",
    "output-format": "json",
    "sign-mode": "direct"
  }
}
EOF
}

setup_relayers() {
  log "Initializing relayer config..."
  $RLY_BIN config init 2>/dev/null || true

  # Add both chains
  TMPD="$(mktemp -d)"
  write_chain_json "$CHAIN_A_ID" "$CHAIN_A_RPC" "$TMPD/chainA.json"
  write_chain_json "$CHAIN_B_ID" "$CHAIN_B_RPC" "$TMPD/chainB.json"

  log "Adding local chains to relayer..."
  $RLY_BIN chains add --file "$TMPD/chainA.json" chainA || $RLY_BIN chains update --file "$TMPD/chainA.json" chainA
  $RLY_BIN chains add --file "$TMPD/chainB.json" chainB || $RLY_BIN chains update --file "$TMPD/chainB.json" chainB

  rm -rf "$TMPD"

  # Restore or create keys
  log "Configuring relayer keys..."
  if [[ -n "$RELAYER_A_MNEMONIC" ]]; then
    echo "$RELAYER_A_MNEMONIC" | $RLY_BIN keys restore chainA "$RELAYER_A_NAME" --coin-type 118 >/dev/null 2>&1 || true
  fi
  if ! $RLY_BIN keys show chainA "$RELAYER_A_NAME" >/dev/null 2>&1; then
    $RLY_BIN keys add chainA "$RELAYER_A_NAME" >/dev/null
  fi

  if [[ -n "$RELAYER_B_MNEMONIC" ]]; then
    echo "$RELAYER_B_MNEMONIC" | $RLY_BIN keys restore chainB "$RELAYER_B_NAME" --coin-type 118 >/dev/null 2>&1 || true
  fi
  if ! $RLY_BIN keys show chainB "$RELAYER_B_NAME" >/dev/null 2>&1; then
    $RLY_BIN keys add chainB "$RELAYER_B_NAME" >/dev/null
  fi

  # Set keys in config
  CFG="$HOME/.relayer/config/config.yaml"
  if [[ -f "$CFG" ]]; then
    $SED "s/key: .*/key: $RELAYER_A_NAME/" "$CFG"
    # Narrowly set for chainA and chainB sections if present
    $SED "s/\(chain-id: $CHAIN_A_ID[\s\S]*\)key: .*/\1key: $RELAYER_A_NAME/" "$CFG" || true
    $SED "s/\(chain-id: $CHAIN_B_ID[\s\S]*\)key: .*/\1key: $RELAYER_B_NAME/" "$CFG" || true
  fi

  # Explicitly set keys for each chain in relayer config
  $RLY_BIN chains set-key chainA "$RELAYER_A_NAME" >/dev/null
  $RLY_BIN chains set-key chainB "$RELAYER_B_NAME" >/dev/null

  # Ensure the active key selection per chain (relayer v2 convention)
  $RLY_BIN keys use chainA "$RELAYER_A_NAME" >/dev/null 2>&1 || true
  $RLY_BIN keys use chainB "$RELAYER_B_NAME" >/dev/null 2>&1 || true

  # Wait for both RPCs to be live before funding
  log "Waiting for chain RPCs to be ready..."
  for i in {1..30}; do
    AOK=$(curl -sf "$CHAIN_A_RPC/status" >/dev/null && echo ok || true)
    BOK=$(curl -sf "$CHAIN_B_RPC/status" >/dev/null && echo ok || true)
    if [[ "$AOK" == ok && "$BOK" == ok ]]; then break; fi
    sleep 1
  done

  # Fund relayer on Chain A from cooluser
  log "Funding Chain A relayer account from cooluser..."
  RELAYER_A_ADDR=$($RLY_BIN keys show chainA "$RELAYER_A_NAME" 2>/dev/null | grep -Eo 'verana1[0-9a-z]+' | head -n1)
  if [[ -z "${RELAYER_A_ADDR:-}" ]]; then
    # Try listing keys and parse address as fallback
    RELAYER_A_ADDR=$($RLY_BIN keys list chainA 2>/dev/null | grep -Eo 'verana1[0-9a-z]+' | head -n1)
  fi
  if [[ -z "${RELAYER_A_ADDR:-}" ]]; then
    # Ensure key exists then re-query
    $RLY_BIN keys add chainA "$RELAYER_A_NAME" >/dev/null
    RELAYER_A_ADDR=$($RLY_BIN keys show chainA "$RELAYER_A_NAME" 2>/dev/null | grep -Eo 'verana1[0-9a-z]+' | head -n1)
  fi
  if [[ -z "${RELAYER_A_ADDR:-}" ]]; then
    echo "Failed to get relayer A address" >&2; exit 1
  fi
  log "Relayer A address: $RELAYER_A_ADDR"

  # Fund relayer on Chain B from validatorb (created by start_chain_b.sh)
  log "Funding Chain B relayer account from validatorb..."
  RELAYER_B_ADDR=$($RLY_BIN keys show chainB "$RELAYER_B_NAME" 2>/dev/null | grep -Eo 'verana1[0-9a-z]+' | head -n1)
  if [[ -z "${RELAYER_B_ADDR:-}" ]]; then
    RELAYER_B_ADDR=$($RLY_BIN keys list chainB 2>/dev/null | grep -Eo 'verana1[0-9a-z]+' | head -n1)
  fi
  if [[ -z "${RELAYER_B_ADDR:-}" ]]; then
    echo "Failed to get relayer B address" >&2; exit 1
  fi
  log "Relayer B address: $RELAYER_B_ADDR"
  veranad tx bank send \
    validatorb \
    "$RELAYER_B_ADDR" \
    "$RELAYER_A_FUND" \
    --chain-id "$CHAIN_B_ID" \
    --keyring-backend test \
    --home "$HOME/.verana-b" \
    --node "$CHAIN_B_RPC" \
    --fees 800000uvna \
    --gas 800000 \
    --gas-adjustment 1.3 \
    -y >/dev/null || true

  log "Waiting 5s for Chain B funds to be processed..."
  sleep 5
  veranad tx bank send \
    cooluser \
    "$RELAYER_A_ADDR" \
    "$RELAYER_A_FUND" \
    --chain-id "$CHAIN_A_ID" \
    --keyring-backend test \
    --node "$CHAIN_A_RPC" \
    --fees 800000uvna \
    --gas 800000 \
    --gas-adjustment 1.3 \
    -y >/dev/null

  log "Waiting 5s for funds to be processed..."
  sleep 5

  # Create path definition
  log "Creating local path definition (chainA <-> chainB)..."
  mkdir -p "$HOME/.relayer/paths"
  cat > "$HOME/.relayer/paths/verana-local.json" <<EOF
{
  "src": { "chain-id": "$CHAIN_A_ID", "client-id": "", "connection-id": "" },
  "dst": { "chain-id": "$CHAIN_B_ID", "client-id": "", "connection-id": "" },
  "src-channel-filter": { "rule": "", "channel-list": [] }
}
EOF

  $RLY_BIN paths add-dir "$HOME/.relayer/paths" >/dev/null

  # Link first with explicit command, then start the relayer loop
  log "Linking chains (one-time)..."
  $RLY_BIN tx link verana-local --override || true

  log "Starting relayer (foreground)..."
  exec $RLY_BIN start verana-local
}

ensure_rly_installed
setup_relayers


