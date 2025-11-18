#!/bin/bash

set -e  # Exit on any error

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

DOCKER_IMAGE="verana:dev-cosmovisor"

# Define timezones for each validator
VALIDATOR_NAMES=(validator1 validator2 validator3 validator4 validator5)
VALIDATOR_TIMEZONES=(
    "America/New_York"      # validator1
    "Europe/London"        # validator2
    "Asia/Tokyo"           # validator3
    "Australia/Sydney"     # validator4
    "America/Los_Angeles"  # validator5
)
CHAIN_ID="vna-local-1"

# Define mnemonics for each validator
VALIDATOR_MNEMONICS=(
    "pink glory help gown abstract eight nice crazy forward ketchup skill cheese"
    "real spring program old collect circle scout survey earth wall north town become lottery response submit shallow garage bird wedding dial loop original melody"
    "forum world antique join retire twelve input flame hole hold sample draft skull speed blossom fork peace opinion soon symbol two left flat prepare"
    "normal cousin seminar poem raccoon genius hope escape track course soup drift build orchard egg mango race loop squeeze someone lunar tail seven much"
    "speak virus flip siren brother drill biology spell section economy mutual spell embody balcony flash celery book disorder language sight lion aim wash clarify"
)

# Set a random base port between 30000 and 40000
BASE_PORT=$(( ( RANDOM % 10000 ) + 30000 ))

# Dynamically assign ports for each validator
for i in {1..5}; do
  P2P_PORT=$((BASE_PORT + (i-1)*10 + 0))
  RPC_PORT=$((BASE_PORT + (i-1)*10 + 1))
  PPROF_PORT=$((BASE_PORT + (i-1)*10 + 2))
  API_PORT=$((BASE_PORT + (i-1)*10 + 3))
  GRPC_PORT=$((BASE_PORT + (i-1)*10 + 4))
  VALIDATOR_PORTS[$i]="$P2P_PORT:$RPC_PORT:$PPROF_PORT:$API_PORT:$GRPC_PORT"
done

# Define addresses for genesis accounts (these will be generated from the mnemonics)
#GENESIS_ACCOUNTS=(
#    "verana16mzeyu9l6kua2cdg9x0jk5g6e7h0kk8q6uadu4"
#    "verana1elf8m94agzfmpg2lkvqd776ellz7fxtgqnhcaa"
#    "verana10nmtkxr4mm7xu0ryq2h5t9f77jelwqsye0heee"
#    "verana17qupxnsfc4l82m40hx6ys08ds9lcj0lqll5kaf"
#    "verana16jh6jcpxnz5l49p3fdmru8rf6ex3farztc2mqa"
#)

PASSPHRASE="testpass123"

# Function to print colored output
print_status() {
    echo -e "${GREEN}[INFO]${NC} $1"
}

print_warning() {
    echo -e "${YELLOW}[WARNING]${NC} $1"
}

print_error() {
    echo -e "${RED}[ERROR]${NC} $1"
}

# Function to print timezone info
print_timezone_info() {
    local validator_name=$1
    local validator_index=$(get_validator_index "$validator_name")
    local timezone=${VALIDATOR_TIMEZONES[$validator_index]}
    local current_time=$(TZ=$timezone date "+%Y-%m-%d %H:%M:%S %Z")
    print_status "$validator_name timezone: $timezone ($current_time)"
}

# Function to cleanup existing containers and directories
cleanup_existing() {
    print_warning "Cleaning up existing containers and data..."

    # Stop and remove existing containers if they exist
    for validator in validator1 validator2 validator3 validator4 validator5; do
        if docker ps -a --format 'table {{.Names}}' | grep -q "^${validator}$"; then
            print_status "Stopping and removing existing ${validator} container..."
            docker stop ${validator} 2>/dev/null || true
            docker rm ${validator} 2>/dev/null || true
        fi
    done

    # Remove existing directories
    for dir in validator1 validator2 validator3 validator4 validator5; do
        if [ -d "$dir" ]; then
            print_status "Removing existing ${dir} directory..."
            rm -rf "$dir"
        fi
    done

    print_status "Cleanup completed!"
}

# Function to wait for a service to be ready
wait_for_service() {
    local url=$1
    local timeout=${2:-30}
    local interval=2
    local count=0

    print_status "Waiting for service at $url to be ready..."

    while ! curl -s "$url" > /dev/null 2>&1; do
        if [ $count -ge $timeout ]; then
            print_error "Timeout waiting for service at $url"
            return 1
        fi
        sleep $interval
        count=$((count + interval))
    done

    print_status "Service at $url is ready!"
}

# Function to wait for block production
wait_for_blocks() {
    local rpc_url=$1
    local timeout=${2:-60}
    local interval=2
    local count=0

    print_status "Waiting for blocks to be produced..."

    while true; do
        if [ $count -ge $timeout ]; then
            print_error "Timeout waiting for blocks to be produced"
            return 1
        fi

        # Check if we can get block info and height > 0
        local height=$(curl -s "${rpc_url}/status" | jq -r '.result.sync_info.latest_block_height' 2>/dev/null)
        if [[ "$height" =~ ^[0-9]+$ ]] && [ "$height" -gt 0 ]; then
            print_status "Blocks are being produced! Current height: $height"
            return 0
        fi

        sleep $interval
        count=$((count + interval))
    done
}

# Function to start a validator with specific timezone
start_validator_with_timezone() {
    local validator_name=$1
    local validator_index=$(get_validator_index "$validator_name")
    local timezone=${VALIDATOR_TIMEZONES[$validator_index]}
    local ports=$2

    print_timezone_info $validator_name

    # Parse ports (format: "p1:p2:p3:p4:p5")
    IFS=':' read -ra ADDR <<< "$ports"
    local p2p_port=${ADDR[0]}
    local rpc_port=${ADDR[1]}
    local pprof_port=${ADDR[2]}
    local api_port=${ADDR[3]}
    local grpc_port=${ADDR[4]}

    print_status "Starting $validator_name with timezone $timezone..."

    if ! docker run -d --name $validator_name \
        -e TZ=$timezone \
        -v $(pwd)/$validator_name:/root/.verana \
        -p $p2p_port:26656 \
        -p $rpc_port:26657 \
        -p $pprof_port:1234 \
        -p $api_port:1317 \
        -p $grpc_port:9090 \
        $DOCKER_IMAGE start; then
        print_error "Failed to start $validator_name container"
        exit 1
    fi

    # Wait for validator to be ready
    wait_for_service "http://localhost:$rpc_port/status" 60
}

# Function to get RPC port from validator ports string
get_rpc_port() {
    local ports=$1
    IFS=':' read -ra ADDR <<< "$ports"
    echo ${ADDR[1]}
}

# Function to get P2P port from validator ports string
get_p2p_port() {
    local ports=$1
    IFS=':' read -ra ADDR <<< "$ports"
    echo ${ADDR[0]}
}

# Function to build persistent peers string
build_persistent_peers() {
    local current_validator=$1
    local peers=""

    # Get all validator numbers that come before current
    for i in {1..5}; do
        local validator="validator$i"
        if [ "$validator" = "$current_validator" ]; then
            break
        fi

        # Get node ID and P2P port for this validator
        local rpc_port=$(get_rpc_port "${VALIDATOR_PORTS[$i]}")
        local p2p_port=$(get_p2p_port "${VALIDATOR_PORTS[$i]}")
        local node_id=$(curl -s "http://localhost:$rpc_port/status" | jq -r '.result.node_info.id')

        if [ -n "$peers" ]; then
            peers="${peers},"
        fi
        peers="${peers}${node_id}@host.docker.internal:${p2p_port}"
    done

    echo "$peers"
}

# Use a helper function to get the index of a validator name
get_validator_index() {
    local name="$1"
    local i
    for i in "${!VALIDATOR_NAMES[@]}"; do
        if [ "${VALIDATOR_NAMES[$i]}" = "$name" ]; then
            echo $i
            return
        fi
    done
    echo -1
}

# Check if jq is installed
if ! command -v jq &> /dev/null; then
    print_error "jq is required but not installed. Please install jq first."
    exit 1
fi

# Automatically clean up existing containers and directories at the start of the script
cleanup_existing

# Handle command line arguments
if [ "$1" = "--clean" ] || [ "$1" = "-c" ]; then
    cleanup_existing
    print_status "Starting fresh setup..."
elif [ "$1" = "--help" ] || [ "$1" = "-h" ]; then
    echo "Usage: $0 [OPTION]"
    echo "Options:"
    echo "  --clean, -c    Clean up existing containers and directories before setup"
    echo "  --help, -h     Show this help message"
    echo ""
    echo "Timezone Configuration:"
    for i in "${!VALIDATOR_NAMES[@]}"; do
        echo "  ${VALIDATOR_NAMES[$i]}: ${VALIDATOR_TIMEZONES[$i]}"
    done
    exit 0
else
    # Check for existing containers and prompt user
    existing_containers=()
    for validator in validator1 validator2 validator3 validator4 validator5; do
        if docker ps -a --format 'table {{.Names}}' | grep -q "^${validator}$"; then
            existing_containers+=("$validator")
        fi
    done

    if [ ${#existing_containers[@]} -gt 0 ]; then
        print_warning "Found existing containers: ${existing_containers[*]}"
        print_warning "This script will fail if containers already exist."
        echo -n "Do you want to clean up existing containers? (y/N): "
        read -r response
        case "$response" in
            [yY][eE][sS]|[yY])
                cleanup_existing
                ;;
            *)
                print_error "Please clean up existing containers first or run with --clean option"
                exit 1
                ;;
        esac
    fi
fi

# Display timezone information
echo "=============================================="
echo "Validator Timezone Configuration:"
echo "=============================================="
for validator in validator1 validator2 validator3 validator4 validator5; do
    print_timezone_info $validator
done
echo "=============================================="
echo

# =============================================================================
# VALIDATOR 1 SETUP (Special case - genesis validator)
# =============================================================================
print_status "Setting up Validator 1 in ${VALIDATOR_TIMEZONES[0]}..."

# Create validator1 directory
mkdir -p validator1

# Setup cosmovisor directory structure BEFORE initialization (if using cosmovisor image)
if [[ "$DOCKER_IMAGE" == *"cosmovisor"* ]]; then
    print_status "Setting up cosmovisor directory structure for validator1 (before init)..."
    mkdir -p validator1/cosmovisor/genesis/bin
    mkdir -p validator1/cosmovisor/upgrades
    
    # Copy binary BEFORE init (cosmovisor may check for it)
    print_status "Copying binary to cosmovisor/genesis/bin (before init)..."
    if docker run --rm --entrypoint cat $DOCKER_IMAGE /usr/local/bin/veranad > validator1/cosmovisor/genesis/bin/veranad.tmp 2>&1; then
        mv validator1/cosmovisor/genesis/bin/veranad.tmp validator1/cosmovisor/genesis/bin/veranad
        chmod +x validator1/cosmovisor/genesis/bin/veranad
        print_status "Binary copied successfully ($(ls -lh validator1/cosmovisor/genesis/bin/veranad | awk '{print $5}'))"
    else
        print_error "Failed to copy binary before init"
        exit 1
    fi
fi

# Initialize the node
# Override entrypoint for setup commands (init, keys, etc.) when using cosmovisor image
if [[ "$DOCKER_IMAGE" == *"cosmovisor"* ]]; then
    docker run --rm --entrypoint veranad -v $(pwd)/validator1:/root/.verana $DOCKER_IMAGE init validator1 --chain-id $CHAIN_ID
else
    docker run --rm -v $(pwd)/validator1:/root/.verana $DOCKER_IMAGE init validator1 --chain-id $CHAIN_ID
fi

# Replace stake with uvna in genesis file
sed -i.bak 's/stake/uvna/g' validator1/config/genesis.json

# Update unbonding time to 60 seconds for testing
# sed -i.bak 's/172800s/60s/g' validator1/config/genesis.json

# Update governance parameters for faster testing
# sed -i.bak 's/"max_deposit_period": ".*"/"max_deposit_period": "100s"/' validator1/config/genesis.json
# sed -i.bak 's/"voting_period": ".*"/"voting_period": "100s"/' validator1/config/genesis.json

# Set minimum gas prices
sed -i.bak 's/minimum-gas-prices = ""/minimum-gas-prices = "0.25uvna"/g' validator1/config/app.toml

# Update RPC to bind to all interfaces
sed -i.bak 's/laddr = "tcp:\/\/127.0.0.1:26657"/laddr = "tcp:\/\/0.0.0.0:26657"/g' validator1/config/config.toml

# Enable and configure API server
sed -i.bak 's/enable = false/enable = true/g' validator1/config/app.toml
sed -i.bak 's/address = "tcp:\/\/localhost:1317"/address = "tcp:\/\/0.0.0.0:1317"/g' validator1/config/app.toml
sed -i.bak 's/swagger = false/swagger = true/g' validator1/config/app.toml
sed -i.bak 's/enabled-unsafe-cors = false/enabled-unsafe-cors = true/g' validator1/config/app.toml

# Enable and configure gRPC server
sed -i.bak 's/address = "localhost:9090"/address = "0.0.0.0:9090"/g' validator1/config/app.toml

# Update CORS settings
sed -i.bak 's/cors_allowed_origins = \[\]/cors_allowed_origins = \["*"\]/' validator1/config/config.toml

# Fix the minimum gas prices if they're still set to stake
sed -i.bak 's/minimum-gas-prices = "0stake"/minimum-gas-prices = "0.25uvna"/g' validator1/config/app.toml

# Update moniker
sed -i.bak 's/moniker = "validator1"/moniker = "validator1"/g' validator1/config/config.toml

# Import all wallets using predefined mnemonics
# Override entrypoint for setup commands when using cosmovisor image
ENTRYPOINT_OVERRIDE=""
if [[ "$DOCKER_IMAGE" == *"cosmovisor"* ]]; then
    ENTRYPOINT_OVERRIDE="--entrypoint veranad"
fi

for i in {1..5}; do
    wallet="wallet$i"
    mnemonic="${VALIDATOR_MNEMONICS[$((i-1))]}"
    print_status "[DEBUG] About to import $wallet for validator1 using predefined mnemonic"
    echo "$mnemonic" | docker run --rm -i $ENTRYPOINT_OVERRIDE \
        -v "$(pwd)/validator1:/root/.verana" \
        $DOCKER_IMAGE \
        keys add "$wallet" --recover --keyring-backend test
    print_status "[DEBUG] Finished importing $wallet for validator1"
    docker run --rm $ENTRYPOINT_OVERRIDE -v $(pwd)/validator1:/root/.verana $DOCKER_IMAGE add-genesis-account "$wallet" 10000000000000000000uvna --keyring-backend test
done

# Generate validator transaction
printf "$PASSPHRASE" \
| docker run --rm -i $ENTRYPOINT_OVERRIDE -v $(pwd)/validator1:/root/.verana $DOCKER_IMAGE gentx wallet1 1000000000uvna --chain-id $CHAIN_ID --keyring-backend test

# Collect genesis transactions
docker run --rm $ENTRYPOINT_OVERRIDE -v $(pwd)/validator1:/root/.verana $DOCKER_IMAGE collect-gentxs

# Validate genesis
docker run --rm $ENTRYPOINT_OVERRIDE -v $(pwd)/validator1:/root/.verana $DOCKER_IMAGE validate-genesis

# Binary was already copied before init, so we just verify it exists
if [[ "$DOCKER_IMAGE" == *"cosmovisor"* ]]; then
    if [ ! -f validator1/cosmovisor/genesis/bin/veranad ]; then
        print_error "Binary not found in cosmovisor/genesis/bin!"
        exit 1
    fi
    print_status "Binary verified in cosmovisor/genesis/bin"
fi

# Start validator 1 with timezone
start_validator_with_timezone "validator1" "${VALIDATOR_PORTS[1]}"

# Check if the container is running before proceeding
if ! docker ps --format '{{.Names}}' | grep -q "^validator1$"; then
    print_error "validator1 container failed to start. Printing logs:"
    docker logs validator1 || true
    exit 1
fi

# Wait for blocks to be produced
wait_for_blocks "http://localhost:$(get_rpc_port "${VALIDATOR_PORTS[1]}")" 60

print_status "Validator 1 is ready!"

# =============================================================================
# VALIDATORS 2-5 SETUP (Loop through remaining validators)
# =============================================================================
for i in {2..5}; do
    validator="validator$i"
    wallet="wallet$i"

    print_status "Setting up $validator in ${VALIDATOR_TIMEZONES[$((i-1))]}..."
    
    # Override entrypoint for setup commands when using cosmovisor image
    # (ENTRYPOINT_OVERRIDE is already defined for validator1, but ensure it's available here too)
    if [[ -z "$ENTRYPOINT_OVERRIDE" ]]; then
        ENTRYPOINT_OVERRIDE=""
        if [[ "$DOCKER_IMAGE" == *"cosmovisor"* ]]; then
            ENTRYPOINT_OVERRIDE="--entrypoint veranad"
        fi
    fi

    # Create directory for validator
    mkdir -p "$validator"

    # Setup cosmovisor directory structure BEFORE initialization (if using cosmovisor image)
    if [[ "$DOCKER_IMAGE" == *"cosmovisor"* ]]; then
        print_status "Setting up cosmovisor directory structure for $validator (before init)..."
        mkdir -p "$validator/cosmovisor/genesis/bin"
        mkdir -p "$validator/cosmovisor/upgrades"
        
        # Copy binary BEFORE init (cosmovisor may check for it)
        print_status "Copying binary to cosmovisor/genesis/bin for $validator (before init)..."
        if docker run --rm --entrypoint cat $DOCKER_IMAGE /usr/local/bin/veranad > "$validator/cosmovisor/genesis/bin/veranad.tmp" 2>&1; then
            mv "$validator/cosmovisor/genesis/bin/veranad.tmp" "$validator/cosmovisor/genesis/bin/veranad"
            chmod +x "$validator/cosmovisor/genesis/bin/veranad"
            print_status "Binary copied successfully for $validator ($(ls -lh "$validator/cosmovisor/genesis/bin/veranad" | awk '{print $5}'))"
        else
            print_error "Failed to copy binary before init for $validator"
            exit 1
        fi
    fi

    # Initialize validator node
    # Override entrypoint for setup commands when using cosmovisor image
    if [[ "$DOCKER_IMAGE" == *"cosmovisor"* ]]; then
        docker run --rm --entrypoint veranad -v "$(pwd)/$validator:/root/.verana" $DOCKER_IMAGE init "$validator" --chain-id $CHAIN_ID
    else
        docker run --rm -v "$(pwd)/$validator:/root/.verana" $DOCKER_IMAGE init "$validator" --chain-id $CHAIN_ID
    fi

    # Copy configuration files from validator 1
    cp validator1/config/genesis.json "$validator/config/genesis.json"
    cp validator1/config/app.toml "$validator/config/app.toml"
    cp validator1/config/config.toml "$validator/config/config.toml"

    # Build persistent peers for this validator
    persistent_peers=$(build_persistent_peers "$validator")
    print_status "$validator persistent peers: $persistent_peers"

    # Set persistent peers
    sed -i.bak "s/persistent_peers = \"\"/persistent_peers = \"$persistent_peers\"/g" "$validator/config/config.toml"

    # Update moniker
    sed -i.bak "s/moniker = \"validator1\"/moniker = \"$validator\"/g" "$validator/config/config.toml"

    # Import wallet using the SAME mnemonic that was added to genesis
    # Override entrypoint for setup commands when using cosmovisor image
    mnemonic="${VALIDATOR_MNEMONICS[$((i-1))]}"
    echo "$mnemonic" | docker run --rm -i $ENTRYPOINT_OVERRIDE \
        -v "$(pwd)/$validator:/root/.verana" \
        $DOCKER_IMAGE \
        keys add "$wallet" --recover --keyring-backend test

    # Get pubkey using a one-off docker run, not docker exec
    PUBKEY=$(docker run --rm $ENTRYPOINT_OVERRIDE -v "$(pwd)/$validator:/root/.verana" $DOCKER_IMAGE tendermint show-validator | jq -c '.')
    cat > $validator/validator.json <<EOF
{
  "pubkey": $PUBKEY,
  "amount": "1000000000uvna",
  "moniker": "$validator",
  "identity": "",
  "website": "",
  "security": "",
  "details": "",
  "commission-rate": "0.10",
  "commission-max-rate": "0.20",
  "commission-max-change-rate": "0.01",
  "min-self-delegation": "1"
}
EOF

    # Extract the RPC port for this validator from the dynamic port assignment
    ports=${VALIDATOR_PORTS[$i]}
    IFS=':' read -ra PORT_ARRAY <<< "$ports"
    RPC_PORT=${PORT_ARRAY[1]}

    # Setup cosmovisor directory structure (if using cosmovisor image)
    if [[ "$DOCKER_IMAGE" == *"cosmovisor"* ]]; then
        print_status "Setting up cosmovisor directory structure for $validator..."
        mkdir -p "$validator/cosmovisor/genesis/bin"
        mkdir -p "$validator/cosmovisor/upgrades"
        
        # Binary was already copied before init, so we just verify it exists
        if [ ! -f "$validator/cosmovisor/genesis/bin/veranad" ]; then
            print_error "Binary not found in cosmovisor/genesis/bin for $validator!"
            exit 1
        fi
        print_status "Binary verified in cosmovisor/genesis/bin for $validator"
    fi
    
    # Start the validator container before running create-validator
    start_validator_with_timezone "$validator" "${VALIDATOR_PORTS[$i]}"
    sleep 20
    # Run the create-validator transaction inside the running container
    print_status "Creating $validator..."
    printf "$PASSPHRASE" \
    | docker exec -i $validator veranad tx staking create-validator /root/.verana/validator.json \
        --from="$wallet" \
        --chain-id $CHAIN_ID \
        --broadcast-mode=sync \
        --fees=65000uvna \
        --keyring-backend test \
        --yes \
        --node tcp://localhost:26657

    # Check if the container is running before proceeding
    if ! docker ps --format '{{.Names}}' | grep -q "^$validator$"; then
        print_error "$validator container failed to start. Printing logs:"
        docker logs $validator || true
        exit 1
    fi

    print_status "$validator is ready!"

    # Wait before setting up next validator
    sleep 20
done

# =============================================================================
# SUMMARY WITH TIMEZONE INFO
# =============================================================================
echo
echo "=============================================="
echo "All Verana validators are now running in different timezones!"
echo "=============================================="

for i in {1..5}; do
    validator="validator$i"
    ports=${VALIDATOR_PORTS[$i]}
    IFS=':' read -ra PORT_ARRAY <<< "$ports"
    rpc_port=${PORT_ARRAY[1]}
    api_port=${PORT_ARRAY[3]}
    grpc_port=${PORT_ARRAY[4]}
    timezone=${VALIDATOR_TIMEZONES[$((i-1))]}

    echo "$validator ($timezone):"
    echo "  RPC: http://localhost:$rpc_port"
    echo "  API: http://localhost:$api_port"
    echo "  gRPC: http://localhost:$grpc_port"
    echo "  Current time: $(TZ=$timezone date)"
    echo
done

echo "=============================================="
echo
echo "To check validator time and status:"
for validator in validator1 validator2 validator3 validator4 validator5; do
    echo "docker exec $validator date && docker exec $validator veranad status"
done
echo
echo "To check network status:"
echo "curl -s http://localhost:36657/validators | jq '.result.validators[] | {moniker: .moniker, voting_power: .voting_power}'"
echo
echo "To stop all validators:"
echo "docker stop validator1 validator2 validator3 validator4 validator5"
echo
echo "To remove all validators:"
echo "docker rm validator1 validator2 validator3 validator4 validator5"
echo
echo "To clean up everything (containers + data):"
echo "./setup_validators.sh --clean"
echo "=============================================="