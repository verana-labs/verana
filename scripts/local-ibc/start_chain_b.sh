#!/usr/bin/env bash
set -euo pipefail

# This script starts a second local Verana chain (Chain B) on different ports
# so you can run two independent chains locally for IBC testing.

log() { echo "$(date '+%Y-%m-%d %H:%M:%S') - $*"; }

if [[ "${OSTYPE:-}" == darwin* ]]; then
  SED="sed -i ''"
else
  SED="sed -i"
fi

BINARY="veranad"
CHAIN_ID_B="vna-testnet-2"
MONIKER_B="validatorB"
HOME_B="$HOME/.verana-b"
APP_TOML_B="$HOME_B/config/app.toml"
CONFIG_TOML_B="$HOME_B/config/config.toml"
GENESIS_B="$HOME_B/config/genesis.json"

# Distinct ports for Chain B (avoid conflicts with Chain A defaults)
P2P_PORT_B=36656
RPC_PORT_B=36657
API_PORT_B=2317
GRPC_PORT_B=19090
GRPC_WEB_PORT_B=19091

# Accounts
RELAYER_NAME_B="relayerb"
VALIDATOR_NAME_B="validatorb"

# Token amounts
GENESIS_SUPPLY_B="1000000000000000000000000uvna"   # 1e24 uvna
RELAYER_FUND_B="100000000000000000000uvna"          # 1e20 uvna

log "Starting setup for Chain B ($CHAIN_ID_B) in $HOME_B"

# Initialize chain B
log "Initializing Chain B..."
"$BINARY" init "$MONIKER_B" --chain-id "$CHAIN_ID_B" --home "$HOME_B"

# Keys: add relayer key (recover from fixed mnemonic for reproducibility)
log "Creating relayer key on Chain B (new mnemonic will be generated)..."
"$BINARY" keys add "$RELAYER_NAME_B" --keyring-backend test --home "$HOME_B" --output json >/dev/null

# Create validator key for Chain B
log "Creating validator key on Chain B..."
"$BINARY" keys add "$VALIDATOR_NAME_B" --keyring-backend test --home "$HOME_B" --output json >/dev/null

# Add a large genesis balance for the relayer and a generic faucet
log "Funding genesis accounts on Chain B..."
"$BINARY" add-genesis-account "$RELAYER_NAME_B" "$RELAYER_FUND_B" --keyring-backend test --home "$HOME_B"
"$BINARY" add-genesis-account "$VALIDATOR_NAME_B" "1000000000000000000000uvna" --keyring-backend test --home "$HOME_B"

# Ensure default denom is uvna BEFORE gentx
log "Setting denom to uvna in genesis (if needed)..."
$SED 's/stake/uvna/g' "$GENESIS_B"

# Faster governance (optional; mirrors primary script behavior)
log "Tweaking governance params for quick proposals..."
$SED 's/"max_deposit_period": ".*"/"max_deposit_period": "100s"/' "$GENESIS_B"
$SED 's/"voting_period": ".*"/"voting_period": "100s"/' "$GENESIS_B"
$SED 's/"expedited_voting_period": ".*"/"expedited_voting_period": "90s"/' "$GENESIS_B"

# Create gentx for validator and collect
log "Creating gentx for Chain B validator..."
"$BINARY" gentx "$VALIDATOR_NAME_B" 1000000000uvna \
  --chain-id "$CHAIN_ID_B" \
  --moniker "$MONIKER_B" \
  --commission-rate "0.10" \
  --commission-max-rate "0.20" \
  --commission-max-change-rate "0.01" \
  --min-self-delegation "1" \
  --keyring-backend test \
  --home "$HOME_B" >/dev/null

log "Collecting gentxs for Chain B..."
"$BINARY" collect-gentxs --home "$HOME_B" >/dev/null

# Minimum gas price and ports
log "Configuring app.toml and config.toml for Chain B ports..."
$SED 's/^minimum-gas-prices = ""/minimum-gas-prices = "0.25uvna"/' "$APP_TOML_B"
$SED "s/:1317/:$API_PORT_B/" "$APP_TOML_B"
$SED "s/:9090/:$GRPC_PORT_B/" "$APP_TOML_B"
$SED "s/:9091/:$GRPC_WEB_PORT_B/" "$APP_TOML_B"

$SED "s/:26656/:$P2P_PORT_B/" "$CONFIG_TOML_B"
$SED "s/:26657/:$RPC_PORT_B/" "$CONFIG_TOML_B"

# Enable API, Swagger, and permissive CORS for local testing
log "Enabling API, Swagger, and CORS on Chain B..."
$SED 's/enable = false/enable = true/' "$APP_TOML_B"
$SED 's/swagger = false/swagger = true/' "$APP_TOML_B"
$SED 's/enabled-unsafe-cors = false/enabled-unsafe-cors = true/' "$APP_TOML_B"
$SED 's/cors_allowed_origins = \[\]/cors_allowed_origins = ["*"]/' "$CONFIG_TOML_B"

# Finalize genesis
log "Validating genesis for Chain B..."
"$BINARY" validate-genesis --home "$HOME_B"

# Start Chain B (foreground unless NO_START is set)
if [[ "${NO_START:-}" == "1" ]]; then
  log "Chain B initialized. Skipping start due to NO_START=1."
else
  log "Starting Chain B..."
  exec "$BINARY" start --home "$HOME_B"
fi


