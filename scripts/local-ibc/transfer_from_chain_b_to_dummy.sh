#!/usr/bin/env bash
set -euo pipefail

log() { echo "$(date '+%Y-%m-%d %H:%M:%S') - $*"; }

BINARY="veranad"
RLY="rly"

# Chain endpoints
CHAIN_A_ID="vna-testnet-1"
CHAIN_B_ID="vna-testnet-2"
CHAIN_A_RPC="http://localhost:26657"
CHAIN_B_RPC="http://localhost:36657"

# Dummy account details (Chain A keyring)
DUMMY_NAME="dummy-acc"
DUMMY_MNEMONIC="length hollow spawn want borrow rude hub potato verify degree bracket ecology fragile view argue priority neutral course update maple face acquire hole ladder"

# Amount to transfer from Chain B -> Chain A
AMOUNT="1000000uvna"

ensure_tools() {
  command -v "$BINARY" >/dev/null || { echo "veranad not found in PATH"; exit 1; }
  command -v "$RLY" >/dev/null || { echo "rly not found in PATH"; exit 1; }
}

create_dummy_key_if_missing() {
  if ! $BINARY keys show "$DUMMY_NAME" --keyring-backend test >/dev/null 2>&1; then
    log "Creating dummy account on Chain A keyring..."
    echo "$DUMMY_MNEMONIC" | $BINARY keys add "$DUMMY_NAME" --recover --keyring-backend test >/dev/null
  else
    log "Dummy account already exists in keyring."
  fi
}

get_dummy_address() {
  DUMMY_ADDR=$($BINARY keys show "$DUMMY_NAME" -a --keyring-backend test)
  if [[ -z "${DUMMY_ADDR:-}" ]]; then
    echo "Failed to obtain dummy address" >&2; exit 1
  fi
  echo "$DUMMY_ADDR"
}

get_chainb_channel() {
  # Query channels on Chain B and get first channel id
  CID=$("$BINARY" q ibc channel channels --node "$CHAIN_B_RPC" -o json | grep -o 'channel-[0-9]\+' | head -n1 || true)
  if [[ -z "${CID:-}" ]]; then
    echo ""; return 1
  fi
  echo "$CID"
}

send_transfer() {
  local dst_addr="$1" channel_id="$2"
  log "Sending $AMOUNT from Chain B -> Chain A on $channel_id to $dst_addr ..."
  $RLY tx transfer chainB chainA "$AMOUNT" "$dst_addr" "$channel_id" --path verana-local
}

wait_for_balance() {
  local addr="$1"; shift
  >&2 log "Waiting for IBC balance to arrive at $addr on Chain A..."
  for i in {1..30}; do
    BAL=$($BINARY q bank balances "$addr" --node "$CHAIN_A_RPC" -o json | grep -Eo 'ibc/[A-F0-9]+' | head -n1 || true)
    if [[ -n "${BAL:-}" ]]; then
      echo "$BAL"; return 0
    fi
    sleep 2
  done
  echo ""; return 1
}

show_ibc_denom_trace() {
  local fullhash="$1"
  # denom-trace expects only the hex hash without the ibc/ prefix
  local hash="${fullhash#ibc/}"
  log "Denom trace for ibc/$hash (on Chain A):"
  $BINARY q ibc-transfer denom-trace "$hash" --node "$CHAIN_A_RPC" || true
}

main() {
  ensure_tools

  # Quick health checks
  curl -sf "$CHAIN_A_RPC/status" >/dev/null || { echo "Chain A RPC not reachable at $CHAIN_A_RPC"; exit 1; }
  curl -sf "$CHAIN_B_RPC/status" >/dev/null || { echo "Chain B RPC not reachable at $CHAIN_B_RPC"; exit 1; }

  create_dummy_key_if_missing
  DST_ADDR=$(get_dummy_address)
  log "Dummy address (Chain A): $DST_ADDR"

  CH_B_CHAN=$(get_chainb_channel) || { echo "Could not determine Chain B channel id"; exit 1; }
  log "Using Chain B src channel: $CH_B_CHAN"

  send_transfer "$DST_ADDR" "$CH_B_CHAN"

  IBC_HASH=$(wait_for_balance "$DST_ADDR") || {
    echo "Timed out waiting for IBC funds on Chain A"; exit 1; }
  log "Received IBC denom on Chain A: $IBC_HASH"
  show_ibc_denom_trace "$IBC_HASH"

  log "All done."
}

main "$@"


