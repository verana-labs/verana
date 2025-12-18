# Variables
CHAIN_ID="vna-local-1"
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
SEED_PHRASE_CONTROLLER_A="simple stuff order coach cliff advance ugly dial right forward boring rhythm comfort initial girl either universe genre pony sort own cycle hurt grit"
SEED_PHRASE_CONTROLLER_B="nut dune cigar refuse buzz tone stage movie mix write flame melt coach minute notice shed foil tilt skate legend half clay high drop"

# Default ports for primary validator
P2P_PORT="26656"
RPC_PORT="26657"
API_PORT="1317"
GRPC_PORT="9090"
GRPC_WEB_PORT="9091"


# $BINARY keys add controllerC --recover --keyring-backend test <<EOF
# $SEED_PHRASE_CONTROLLER_A
# EOF

echo "$SEED_PHRASE_CONTROLLER_A" | $BINARY keys add controllerA --recover --keyring-backend test | true
echo "$SEED_PHRASE_CONTROLLER_B" | $BINARY keys add controllerB --recover --keyring-backend test | true


echo done