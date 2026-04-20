set -e

# Function to log messages
log() {
    echo "$(date '+%Y-%m-%d %H:%M:%S') - $1"
}

# Variables
CHAIN_ID="vna-testnet-1"
MONIKER="validator1"
BINARY="veranad"
HOME_DIR="$HOME/.verana"
GENESIS_JSON_PATH="$HOME_DIR/config/genesis.json"
APP_TOML_PATH="$HOME_DIR/config/app.toml"
CONFIG_TOML_PATH="$HOME_DIR/config/config.toml"
VALIDATOR_NAME="cooluser"
VALIDATOR_AMOUNT="1000000000000000000000uvna"
GENTX_AMOUNT="1000000000uvna"
SEED_PHRASE_COOLUSER="pink glory help gown abstract eight nice crazy forward ketchup skill cheese"

# Default ports for primary validator
P2P_PORT="26656"
RPC_PORT="26657"
API_PORT="1317"
GRPC_PORT="9090"
GRPC_WEB_PORT="9091"

log "Starting Primary Validator setup..."

# Initialize the chain with uvna as default denom
log "Initializing the chain..."
$BINARY init $MONIKER --chain-id $CHAIN_ID --default-denom uvna
if [ $? -ne 0 ]; then
    log "Error: Failed to initialize the chain."
    exit 1
fi

# Add a validator key
log "Adding validator key..."
echo "$SEED_PHRASE_COOLUSER" | $BINARY keys add $VALIDATOR_NAME --recover --keyring-backend test
if [ $? -ne 0 ]; then
    log "Error: Failed to add validator key."
    exit 1
fi

# Add genesis account
log "Adding genesis account..."
$BINARY add-genesis-account $VALIDATOR_NAME $VALIDATOR_AMOUNT --keyring-backend test
if [ $? -ne 0 ]; then
    log "Error: Failed to add genesis account."
    exit 1
fi

# Replace remaining "stake" references and update governance params BEFORE gentx
# Use Python for reliable cross-platform editing
log "Replacing 'stake' with 'uvna' in genesis.json and updating governance params..."
python3 - <<'PYEOF'
import json, os

home = os.path.expanduser("~")
genesis_path = os.path.join(home, ".verana", "config", "genesis.json")

with open(genesis_path) as f:
    content = f.read()

# Replace any remaining "stake" denom references
content = content.replace('"stake"', '"uvna"')

with open(genesis_path, "w") as f:
    f.write(content)

# Now update governance params via JSON
with open(genesis_path) as f:
    g = json.load(f)

g['app_state']['gov']['params']['max_deposit_period'] = '100s'
g['app_state']['gov']['params']['voting_period'] = '100s'
g['app_state']['gov']['params']['expedited_voting_period'] = '90s'

with open(genesis_path, "w") as f:
    json.dump(g, f, indent=2)
PYEOF

# Create gentx (bond_denom=uvna is now set in genesis)
log "Creating genesis transaction..."
$BINARY gentx $VALIDATOR_NAME $GENTX_AMOUNT \
    --chain-id $CHAIN_ID \
    --moniker $MONIKER \
    --commission-rate "0.10" \
    --commission-max-rate "0.20" \
    --commission-max-change-rate "0.01" \
    --min-self-delegation "1" \
    --keyring-backend test
if [ $? -ne 0 ]; then
    log "Error: Failed to create genesis transaction."
    exit 1
fi

# Configure app.toml and config.toml using Python for reliable cross-platform editing
log "Updating app.toml and config.toml..."
python3 - <<PYEOF
import os

home = os.path.expanduser("~")
app_toml = os.path.join(home, ".verana", "config", "app.toml")
config_toml = os.path.join(home, ".verana", "config", "config.toml")

with open(app_toml) as f:
    c = f.read()
c = c.replace('minimum-gas-prices = ""', 'minimum-gas-prices = "0.25uvna"')
c = c.replace('enable = false', 'enable = true')
c = c.replace('swagger = false', 'swagger = true')
c = c.replace('enabled-unsafe-cors = false', 'enabled-unsafe-cors = true')
c = c.replace(':1317', ':${API_PORT}')
c = c.replace(':9090', ':${GRPC_PORT}')
c = c.replace(':9091', ':${GRPC_WEB_PORT}')
c = c.replace('address = "tcp://localhost:', 'address = "tcp://0.0.0.0:')
c = c.replace('address = "localhost:', 'address = "0.0.0.0:')
with open(app_toml, 'w') as f:
    f.write(c)

with open(config_toml) as f:
    c = f.read()
c = c.replace('cors_allowed_origins = []', 'cors_allowed_origins = ["*"]')
c = c.replace(':26656', ':${P2P_PORT}')
c = c.replace(':26657', ':${RPC_PORT}')
c = c.replace('laddr = "tcp://127.0.0.1:', 'laddr = "tcp://0.0.0.0:')
with open(config_toml, 'w') as f:
    f.write(c)
PYEOF

# Initialize YieldIntermediatePool module account with 1 uvna to prevent invariant violations
# This addresses the issue where empty module accounts break invariants when receiving funds
# See: https://github.com/cosmos/cosmos-sdk/issues/25315
log "Initializing YieldIntermediatePool module account with 1 uvna..."

# Hardcoded YieldIntermediatePool module account address (derived from module name)
YIELD_POOL_ADDR="verana1wjnrmvjlgxvs098cnu3jaczzjjm4csmqep067h"

if command -v python3 &> /dev/null; then
    python3 << PYEOF
import json

genesis_path = "$GENESIS_JSON_PATH"
yield_addr = "verana1wjnrmvjlgxvs098cnu3jaczzjjm4csmqep067h"

with open(genesis_path, "r") as f:
    genesis = json.load(f)

app_state = genesis.setdefault("app_state", {})
bank = app_state.setdefault("bank", {})
balances = bank.setdefault("balances", [])

# Find existing balance entry for the yield pool address
entry = next((b for b in balances if b.get("address") == yield_addr), None)

balance_added = False
if entry is None:
    # Create a new balance entry with 1 uvna
    balances.append({
        "address": yield_addr,
        "coins": [{"denom": "uvna", "amount": "1"}],
    })
    balance_added = True
else:
    # Update existing entry: increment or add uvna
    coins = entry.setdefault("coins", [])
    uvna = next((c for c in coins if c.get("denom") == "uvna"), None)
    if uvna is None:
        coins.append({"denom": "uvna", "amount": "1"})
        balance_added = True
    else:
        old_amount = int(uvna.get("amount", "0"))
        if old_amount == 0:
            uvna["amount"] = "1"
            balance_added = True
        else:
            uvna["amount"] = str(old_amount + 1)
            balance_added = True

# Update supply to match the added balance
if balance_added:
    supply = bank.setdefault("supply", [])
    uvna_supply = next((s for s in supply if s.get("denom") == "uvna"), None)
    if uvna_supply:
        # Increment existing supply by 1
        current_supply = int(uvna_supply.get("amount", "0"))
        uvna_supply["amount"] = str(current_supply + 1)
    else:
        # Add new supply entry
        supply.append({"denom": "uvna", "amount": "1"})

with open(genesis_path, "w") as f:
    json.dump(genesis, f, indent=2)

print(f"Ensured 1 uvna exists for YieldIntermediatePool account: {yield_addr}")
PYEOF
else
    log "Warning: python3 not found; cannot auto-initialize YieldIntermediatePool."
    log "Please add 1uvna to $YIELD_POOL_ADDR in $GENESIS_JSON_PATH manually."
fi

# Collect genesis transactions
log "Collecting genesis transactions..."
$BINARY collect-gentxs

# Validate genesis file
log "Validating genesis file..."
$BINARY validate-genesis

# Save the genesis file for other validators
cp $GENESIS_JSON_PATH $HOME/genesis.json
log "Genesis file saved to $HOME/genesis.json"

# Get validator node ID
NODE_ID=$($BINARY tendermint show-node-id)
log "Primary Validator Node ID: $NODE_ID"
echo "Node ID: $NODE_ID" > $HOME/primary_validator_info.txt
echo "P2P Address: localhost:$P2P_PORT" >> $HOME/primary_validator_info.txt

# Start the chain
log "Starting the Primary Validator..."
$BINARY start

log "Primary Validator setup complete. If you encounter any issues, please check the logs above."
