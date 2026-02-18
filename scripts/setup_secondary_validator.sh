#!/usr/bin/env bash

set -euo pipefail

log() {
  echo "$(date '+%Y-%m-%d %H:%M:%S') - $1"
}

if [[ "${OSTYPE:-}" == "darwin"* ]]; then
  sed_inplace() { sed -i '' "$@"; }
else
  sed_inplace() { sed -i "$@"; }
fi

VALIDATOR_NUM="${1:-2}"

BINARY="${BINARY:-veranad}"
CHAIN_ID="${CHAIN_ID:-${VERANA_CHAIN_ID:-vna-testnet-1}}"
PRIMARY_HOME="${PRIMARY_HOME:-$HOME/.verana}"
PRIMARY_RPC="${PRIMARY_RPC:-tcp://127.0.0.1:26657}"
PRIMARY_HTTP_RPC="${PRIMARY_HTTP_RPC:-http://127.0.0.1:26657}"

SECONDARY_HOME="${SECONDARY_HOME:-$HOME/.verana${VALIDATOR_NUM}}"
VALIDATOR_NAME="${VALIDATOR_NAME:-validator${VALIDATOR_NUM}}"
MONIKER="${MONIKER:-validator${VALIDATOR_NUM}}"

PORT_OFFSET=$(( (VALIDATOR_NUM - 1) * 100 ))
SECONDARY_P2P_PORT="${SECONDARY_P2P_PORT:-$((26656 + PORT_OFFSET))}"
SECONDARY_RPC_PORT="${SECONDARY_RPC_PORT:-$((26657 + PORT_OFFSET))}"
SECONDARY_API_PORT="${SECONDARY_API_PORT:-$((1317 + PORT_OFFSET))}"
SECONDARY_GRPC_PORT="${SECONDARY_GRPC_PORT:-$((9090 + PORT_OFFSET))}"
SECONDARY_GRPC_WEB_PORT="${SECONDARY_GRPC_WEB_PORT:-$((9091 + PORT_OFFSET))}"

FUND_AMOUNT="${FUND_AMOUNT:-1500000000uvna}"
STAKE_AMOUNT="${STAKE_AMOUNT:-1000000000uvna}"
TX_FEES="${TX_FEES:-800000uvna}"
SECONDARY_LOG_PATH="${SECONDARY_LOG_PATH:-/tmp/verana-validator${VALIDATOR_NUM}.log}"

SECONDARY_MNEMONIC="${SECONDARY_MNEMONIC:-real spring program old collect circle scout survey earth wall north town become lottery response submit shallow garage bird wedding dial loop original melody}"

wait_for_rpc() {
  local endpoint="$1"
  local attempts="${2:-120}"

  for i in $(seq 1 "$attempts"); do
    if curl -fsS "${endpoint}/status" >/dev/null 2>&1; then
      return 0
    fi
    sleep 2
  done
  return 1
}

validator_count() {
  curl -fsS "${PRIMARY_HTTP_RPC}/validators" | tr -d '\n' | awk '{print gsub(/"address"/, "&")}'
}

wait_for_validator_set_size() {
  local expected="$1"
  local attempts="${2:-120}"

  for i in $(seq 1 "$attempts"); do
    local count
    count="$(validator_count || echo 0)"
    if [[ "${count:-0}" -ge "$expected" ]]; then
      log "Validator set size is ${count} (expected >= ${expected})"
      return 0
    fi
    sleep 2
  done
  return 1
}

send_funds_with_retry() {
  local to_address="$1"
  local max_attempts=5

  for attempt in $(seq 1 "$max_attempts"); do
    if "$BINARY" tx bank send \
      cooluser \
      "$to_address" \
      "$FUND_AMOUNT" \
      --chain-id "$CHAIN_ID" \
      --keyring-backend test \
      --home "$PRIMARY_HOME" \
      --node "$PRIMARY_RPC" \
      --fees "$TX_FEES" \
      --gas 800000 \
      --gas-adjustment 1.3 \
      -y >/tmp/secondary-validator-fund.log 2>&1; then
      return 0
    fi

    log "Funding tx failed (attempt ${attempt}/${max_attempts}), retrying..."
    sleep 3
  done

  return 1
}

create_validator_with_retry() {
  local validator_file="$1"
  local max_attempts=5

  for attempt in $(seq 1 "$max_attempts"); do
    if "$BINARY" tx staking create-validator "$validator_file" \
      --from "$VALIDATOR_NAME" \
      --chain-id "$CHAIN_ID" \
      --keyring-backend test \
      --home "$SECONDARY_HOME" \
      --node "$PRIMARY_RPC" \
      --fees "$TX_FEES" \
      --gas 800000 \
      --gas-adjustment 1.3 \
      -y >/tmp/secondary-validator-create.log 2>&1; then
      return 0
    fi

    log "Create-validator tx failed (attempt ${attempt}/${max_attempts}), retrying..."
    sleep 3
  done

  return 1
}

log "Bootstrapping secondary validator ${VALIDATOR_NUM}..."
log "Chain ID: ${CHAIN_ID}"
log "Primary RPC: ${PRIMARY_HTTP_RPC}"
log "Secondary home: ${SECONDARY_HOME}"

if ! wait_for_rpc "$PRIMARY_HTTP_RPC" 120; then
  log "Error: primary RPC did not become ready at ${PRIMARY_HTTP_RPC}"
  exit 1
fi

log "Preparing secondary home..."
rm -rf "$SECONDARY_HOME"
"$BINARY" init "$MONIKER" --chain-id "$CHAIN_ID" --home "$SECONDARY_HOME" >/tmp/secondary-validator-init.log 2>&1

if ! "$BINARY" keys show "$VALIDATOR_NAME" --keyring-backend test --home "$SECONDARY_HOME" >/dev/null 2>&1; then
  echo "$SECONDARY_MNEMONIC" | "$BINARY" keys add "$VALIDATOR_NAME" --recover --keyring-backend test --home "$SECONDARY_HOME" >/tmp/secondary-validator-key.log 2>&1
fi

SECONDARY_ADDRESS="$("$BINARY" keys show "$VALIDATOR_NAME" -a --keyring-backend test --home "$SECONDARY_HOME")"
SECONDARY_OPERATOR_ADDRESS="$("$BINARY" keys show "$VALIDATOR_NAME" -a --bech val --keyring-backend test --home "$SECONDARY_HOME")"
PRIMARY_NODE_ID="$("$BINARY" tendermint show-node-id --home "$PRIMARY_HOME")"

cp "$PRIMARY_HOME/config/genesis.json" "$SECONDARY_HOME/config/genesis.json"

APP_TOML_PATH="$SECONDARY_HOME/config/app.toml"
CONFIG_TOML_PATH="$SECONDARY_HOME/config/config.toml"

sed_inplace "s/^minimum-gas-prices = \".*\"/minimum-gas-prices = \"0.25uvna\"/" "$APP_TOML_PATH"
sed_inplace "s/:1317/:${SECONDARY_API_PORT}/" "$APP_TOML_PATH"
sed_inplace "s/:9090/:${SECONDARY_GRPC_PORT}/" "$APP_TOML_PATH"
sed_inplace "s/:9091/:${SECONDARY_GRPC_WEB_PORT}/" "$APP_TOML_PATH"

sed_inplace "s/:26656/:${SECONDARY_P2P_PORT}/" "$CONFIG_TOML_PATH"
sed_inplace "s/:26657/:${SECONDARY_RPC_PORT}/" "$CONFIG_TOML_PATH"
sed_inplace "s|^persistent_peers = \".*\"|persistent_peers = \"${PRIMARY_NODE_ID}@127.0.0.1:26656\"|" "$CONFIG_TOML_PATH"

sed_inplace 's/enable = false/enable = true/' "$APP_TOML_PATH"
sed_inplace 's/swagger = false/swagger = true/' "$APP_TOML_PATH"
sed_inplace 's/enabled-unsafe-cors = false/enabled-unsafe-cors = true/' "$APP_TOML_PATH"
sed_inplace 's/cors_allowed_origins = \[\]/cors_allowed_origins = \["*"\]/' "$CONFIG_TOML_PATH"

if "$BINARY" query staking validator "$SECONDARY_OPERATOR_ADDRESS" --node "$PRIMARY_RPC" >/dev/null 2>&1; then
  log "Validator ${SECONDARY_OPERATOR_ADDRESS} already exists on-chain, skipping create-validator tx."
else
  log "Funding secondary validator address ${SECONDARY_ADDRESS}..."
  if ! send_funds_with_retry "$SECONDARY_ADDRESS"; then
    log "Error: failed to fund secondary validator after retries."
    cat /tmp/secondary-validator-fund.log || true
    exit 1
  fi

  sleep 5

  log "Submitting create-validator for ${SECONDARY_OPERATOR_ADDRESS}..."
  cat > "$SECONDARY_HOME/validator.json" <<EOF
{
  "pubkey": $("$BINARY" tendermint show-validator --home "$SECONDARY_HOME"),
  "amount": "${STAKE_AMOUNT}",
  "moniker": "${MONIKER}",
  "identity": "",
  "website": "",
  "security": "",
  "details": "Validator ${VALIDATOR_NUM}",
  "commission-rate": "0.10",
  "commission-max-rate": "0.20",
  "commission-max-change-rate": "0.01",
  "min-self-delegation": "1"
}
EOF

  if ! create_validator_with_retry "$SECONDARY_HOME/validator.json"; then
    log "Error: failed to create secondary validator after retries."
    cat /tmp/secondary-validator-create.log || true
    exit 1
  fi
fi

log "Starting secondary validator process..."
nohup "$BINARY" start --home "$SECONDARY_HOME" > "$SECONDARY_LOG_PATH" 2>&1 &
SECONDARY_PID=$!
echo "$SECONDARY_PID" > "/tmp/verana-validator${VALIDATOR_NUM}.pid"

sleep 2
if ! kill -0 "$SECONDARY_PID" >/dev/null 2>&1; then
  log "Error: secondary validator process did not stay alive."
  tail -n 200 "$SECONDARY_LOG_PATH" || true
  exit 1
fi

if ! wait_for_rpc "http://127.0.0.1:${SECONDARY_RPC_PORT}" 60; then
  log "Error: secondary RPC did not become ready on port ${SECONDARY_RPC_PORT}."
  tail -n 200 "$SECONDARY_LOG_PATH" || true
  exit 1
fi

if ! wait_for_validator_set_size 2 120; then
  log "Error: validator set did not reach 2 validators."
  tail -n 200 "$SECONDARY_LOG_PATH" || true
  exit 1
fi

log "Secondary validator bootstrap complete."
