# Complete Guide: Setting Up Hermes Relayer between Verana Devnet and Osmosis Mainnet

**Final Production-Ready Version**

This guide incorporates all fixes and optimizations discovered during setup. Follow these steps for a reliable relayer configuration.

---

## Prerequisites

Before starting, ensure you have:

✅ **Rust and Hermes installed** (see installation steps below)  
✅ Wallet with funds on both Verana devnet and Osmosis mainnet  
✅ Mnemonics or key files for both wallets

**Recommended Minimum Balances:**
- Verana devnet: 20 VNA (20,000,000 uvna)
- Osmosis mainnet: 20 OSMO

---

## Installation: Rust and Hermes

### Step 0.1: Install Rust

Hermes is written in Rust, so you need to install Rust first.

```bash
# Install Rust using rustup (official installer)
curl --proto '=https' --tlsv1.2 -sSf https://sh.rustup.rs | sh

# Follow the on-screen instructions and select option 1 (default installation)
# After installation completes, configure your current shell:
source $HOME/.cargo/env

# Verify Rust installation
rustc --version
cargo --version
```

**Expected output:**
```
rustc 1.xx.x (some hash)
cargo 1.xx.x (some hash)
```

**Note for macOS users:** You may need to install Xcode Command Line Tools first:
```bash
xcode-select --install
```

**Note for Linux users:** You may need to install build essentials:
```bash
# Ubuntu/Debian
sudo apt update
sudo apt install build-essential pkg-config libssl-dev

# Fedora/RHEL
sudo dnf install gcc openssl-devel
```

### Step 0.2: Install Hermes

There are two methods to install Hermes:

#### Method 1: Install from Source (Recommended)

```bash
# Clone the Hermes repository
git clone https://github.com/informalsystems/hermes.git
cd hermes

# Checkout the latest stable release (v1.9.0 or newer)
git checkout v1.9.0

# Build and install Hermes (this may take 5-10 minutes)
cargo install --path crates/relayer-cli --locked

# Verify installation
hermes version
```

**Expected output:**
```
hermes 1.9.0+a026d661
```

#### Method 2: Install Pre-built Binary (Faster)

```bash
# Download the latest release for your platform from GitHub
# For macOS (Intel):
wget https://github.com/informalsystems/hermes/releases/download/v1.9.0/hermes-v1.9.0-x86_64-apple-darwin.tar.gz
tar -xzf hermes-v1.9.0-x86_64-apple-darwin.tar.gz
sudo mv hermes /usr/local/bin/
chmod +x /usr/local/bin/hermes

# For macOS (Apple Silicon):
wget https://github.com/informalsystems/hermes/releases/download/v1.9.0/hermes-v1.9.0-aarch64-apple-darwin.tar.gz
tar -xzf hermes-v1.9.0-aarch64-apple-darwin.tar.gz
sudo mv hermes /usr/local/bin/
chmod +x /usr/local/bin/hermes

# For Linux (x86_64):
wget https://github.com/informalsystems/hermes/releases/download/v1.9.0/hermes-v1.9.0-x86_64-unknown-linux-gnu.tar.gz
tar -xzf hermes-v1.9.0-x86_64-unknown-linux-gnu.tar.gz
sudo mv hermes /usr/local/bin/
chmod +x /usr/local/bin/hermes

# Verify installation
hermes version
```

**Expected output:**
```
hermes 1.9.0+a026d661
```

### Step 0.3: Verify Hermes Installation

```bash
# Check that Hermes is accessible
hermes --help

# You should see the help menu with all available commands
```

**Expected output:**
```
hermes 1.9.0+a026d661
Informal Systems <hello@informal.systems>
  Hermes is an IBC Relayer written in Rust

USAGE:
    hermes [OPTIONS] [SUBCOMMAND]
...
```

### Troubleshooting Installation

**Issue: "command not found: hermes"**
- **Solution:** Make sure `/usr/local/bin` is in your PATH:
```bash
echo 'export PATH="/usr/local/bin:$PATH"' >> ~/.bashrc  # or ~/.zshrc for zsh
source ~/.bashrc  # or source ~/.zshrc
```

**Issue: "cargo: command not found"**
- **Solution:** Rust wasn't properly installed. Run:
```bash
source $HOME/.cargo/env
```

**Issue: Build fails with "linker error"**
- **Solution:** Install build dependencies (see Step 0.1 notes for your OS)

---

Now that Rust and Hermes are installed, proceed with the relayer setup:

---

## Step 1: Create Hermes Configuration Directory

```bash
mkdir -p $HOME/.hermes
cd $HOME/.hermes
```

---

## Step 2: Create Configuration File FIRST

**Critical:** Create the config file BEFORE adding keys, otherwise Hermes won't recognize the chains.

```bash
nano $HOME/.hermes/config.toml
```

Paste this complete, tested configuration:

```toml
[global]
log_level = 'info'

[mode]

[mode.clients]
enabled = true
refresh = true
misbehaviour = true

[mode.connections]
enabled = false

[mode.channels]
enabled = false

[mode.packets]
enabled = true
clear_interval = 100
clear_on_start = true
tx_confirmation = false

[rest]
enabled = true
host = '127.0.0.1'
port = 3000

[telemetry]
enabled = true
host = '127.0.0.1'
port = 3001

# Verana Devnet Configuration
[[chains]]
id = 'vna-devnet-1'
type = 'CosmosSdk'
rpc_addr = 'https://rpc.devnet.verana.network'
event_source = { mode = 'push', url = 'ws://node1.devnet.verana.network:26657/websocket', batch_delay = '500ms' }
grpc_addr = 'http://node1.devnet.verana.network:9090'
rpc_timeout = '15s'
trusted_node = false
account_prefix = 'verana'
key_name = 'verana-relayer'
key_store_type = 'Test'
store_prefix = 'ibc'
default_gas = 2000000
max_gas = 10000000
gas_multiplier = 1.2
max_msg_num = 30
max_tx_size = 2097152
clock_drift = '600s'
max_block_time = '600s'
trusting_period = '14days'
memo_prefix = 'Relayed by Hermes'
sequential_batch_tx = false

[chains.trust_threshold]
numerator = '1'
denominator = '3'

[chains.gas_price]
price = 25
denom = 'uvna'

[chains.packet_filter]
policy = 'allow'
list = [
  ['transfer', '*']
]

[chains.address_type]
derivation = 'cosmos'

# Osmosis Mainnet Configuration
[[chains]]
id = 'osmosis-1'
type = 'CosmosSdk'
rpc_addr = 'https://osmosis.rpc.kjnodes.com'
event_source = { mode = 'push', url = 'wss://osmosis-rpc.polkachu.com/websocket', batch_delay = '500ms' }
grpc_addr = 'https://osmosis.grpc.kjnodes.com:443'
rpc_timeout = '15s'
trusted_node = false
account_prefix = 'osmo'
key_name = 'osmosis-relayer'
key_store_type = 'Test'
store_prefix = 'ibc'
default_gas = 5000000
max_gas = 15000000
gas_multiplier = 1.1
max_msg_num = 20
max_tx_size = 2097152
clock_drift = '600s'
max_block_time = '600s'
trusting_period = '10days'
memo_prefix = 'Relayed by Hermes'
sequential_batch_tx = false

[chains.trust_threshold]
numerator = '1'
denominator = '3'

[chains.gas_price]
price = 0.05
denom = 'uosmo'

[chains.packet_filter]
policy = 'allow'
list = [
  ['transfer', '*']
]

[chains.address_type]
derivation = 'cosmos'
```

**Key Configuration Notes:**
- ✅ `clock_drift = '600s'` - High value prevents timestamp validation issues
- ✅ kjnodes RPC + Polkachu WebSocket for Osmosis - Tested and reliable
- ✅ Verified Verana endpoints - Tested and working
- ✅ `packet_filter = '*'` - Will be updated after channel creation

Save and exit (Ctrl+X, then Y, then Enter)

---

## Step 3: Validate Configuration

```bash
hermes config validate
```

**Expected output:**
```
SUCCESS: "validation passed successfully"
```

---

## Step 4: Add Keys to Hermes

### 4.1 Add Verana Key

```bash
# Create mnemonic file (replace with your actual mnemonic)
echo "your twelve or twenty four word mnemonic here" > mnemonic_verana.txt

# Add key to Hermes
hermes keys add --key-name verana-relayer --chain vna-devnet-1 --mnemonic-file mnemonic_verana.txt

# Remove mnemonic file for security
rm mnemonic_verana.txt
```

### 4.2 Add Osmosis Key

```bash
# Create mnemonic file (replace with your actual mnemonic)
echo "your twelve or twenty four word mnemonic here" > mnemonic_osmosis.txt

# Add key to Hermes
hermes keys add --key-name osmosis-relayer --chain osmosis-1 --mnemonic-file mnemonic_osmosis.txt

# Remove mnemonic file for security
rm mnemonic_osmosis.txt
```

### 4.3 Verify Keys

```bash
# List keys for both chains
hermes keys list --chain vna-devnet-1
hermes keys list --chain osmosis-1
```

**Expected output:** Each should show the address for the respective chain.

---

## Step 5: Health Check

```bash
hermes health-check
```

**Expected output:**
```
INFO [vna-devnet-1] performing health check...
WARN chain is healthy chain=vna-devnet-1
INFO [osmosis-1] performing health check...
INFO chain is healthy chain=osmosis-1
SUCCESS performed health check for all chains in the config
```

**Note:** A warning about gRPC is normal and won't affect functionality. As long as both chains show "healthy," you're good to proceed.

---

## Step 6: Check for Existing IBC Infrastructure

**IMPORTANT:** Always check if a path already exists before creating new clients/connections/channels!

```bash
# Check for existing channels
hermes query channels --chain vna-devnet-1
hermes query channels --chain osmosis-1

# Check for existing connections
hermes query connections --chain vna-devnet-1
hermes query connections --chain osmosis-1
```

**If you see existing channels between Verana and Osmosis:**
- Note the channel IDs
- Skip to Step 9 and use those existing channels
- **Do NOT create duplicate channels**

---

## Step 7: Create IBC Connection (If None Exists)

```bash
hermes create connection --a-chain vna-devnet-1 --b-chain osmosis-1
```

**Expected output:**
```
Creating new clients hosted on chains vna-devnet-1 and osmosis-1
client was created successfully id=07-tendermint-X
client was created successfully id=07-tendermint-Y
OpenInitConnection...
OpenTryConnection...
OpenAckConnection...
OpenConfirmConnection...
SUCCESS Connection {
    a_side: ConnectionSide {
        chain_id: vna-devnet-1,
        client_id: 07-tendermint-X,
        connection_id: Some(connection-N)
    },
    b_side: ConnectionSide {
        chain_id: osmosis-1,
        client_id: 07-tendermint-Y,
        connection_id: Some(connection-M)
    }
}
```

**Save these IDs:**
- Verana connection ID: `connection-N`
- Osmosis connection ID: `connection-M`
- Verana client ID: `07-tendermint-X`
- Osmosis client ID: `07-tendermint-Y`

---

## Step 8: Create IBC Channel

Use the connection ID from Verana (shown in Step 7):

```bash
hermes create channel --a-chain vna-devnet-1 --a-connection connection-N --a-port transfer --b-port transfer
```

**Replace `connection-N` with your actual Verana connection ID from Step 7!**

**Expected output:**
```
OpenInitChannel...
OpenTryChannel...
OpenAckChannel...
OpenConfirmChannel...
SUCCESS Channel {
    a_side: ChannelSide {
        chain_id: vna-devnet-1,
        channel_id: Some(channel-X)
    },
    b_side: ChannelSide {
        chain_id: osmosis-1,
        channel_id: Some(channel-Y)
    }
}
```

**Save these channel IDs:**
- Verana channel: `channel-X`
- Osmosis channel: `channel-Y`

---

## Step 9: Update Packet Filters in Config

Edit your config to specify the exact channels:

```bash
nano $HOME/.hermes/config.toml
```

**Update Verana chain section:**
```toml
[chains.packet_filter]
policy = 'allow'
list = [
  ['transfer', 'channel-X']  # Replace X with your actual channel number
]
```

**Update Osmosis chain section:**
```toml
[chains.packet_filter]
policy = 'allow'
list = [
  ['transfer', 'channel-Y']  # Replace Y with your actual channel number
]
```

**Example:** If Verana is `channel-0` and Osmosis is `channel-107781`:
```toml
# Verana
list = [['transfer', 'channel-0']]

# Osmosis
list = [['transfer', 'channel-107781']]
```

Save the file and validate:

```bash
hermes config validate
```

---

## Step 10: Start the Relayer

### Option A: Run in Foreground (for testing)

```bash
hermes start
```

### Option B: Run in Background (for production)

```bash
nohup hermes start > $HOME/.hermes/hermes.log 2>&1 &

# View logs in real-time
tail -f $HOME/.hermes/hermes.log
```

**Expected output (successful start):**
```
INFO using default configuration from '$HOME/.hermes/config.toml'
INFO Hermes has started
INFO [vna-devnet-1] chain driver spawned
INFO [osmosis-1] chain driver spawned
INFO spawned workers for 2 chains
```

---

## Step 11: Test IBC Transfer

Send a test transfer from Verana to Osmosis:

```bash
# Using veranad CLI (adjust based on your setup)
veranad tx ibc-transfer transfer transfer channel-X \
  <OSMOSIS_ADDRESS> 1000000uvna \
  --from <YOUR_VERANA_KEY> \
  --chain-id vna-devnet-1 \
  --node https://rpc.devnet.verana.network \
  --fees 600000uvna \
  --gas auto
```

**Replace:**
- `channel-X` with your Verana channel ID
- `<OSMOSIS_ADDRESS>` with recipient address on Osmosis
- `<YOUR_VERANA_KEY>` with your Verana wallet key name

### Monitor the Relayer

Watch the logs for successful relaying:

```bash
tail -f $HOME/.hermes/hermes.log
```

**Look for these SUCCESS indicators:**
```
recv_packet
write_acknowledgement  
SUCCESS
```

The transfer should complete in 10-30 seconds!

---

## Monitoring & Maintenance

### Check Relayer Status

```bash
# Health check
hermes health-check

# Check specific client
hermes query client status --chain vna-devnet-1 --client <CLIENT_ID>

# View pending packets
hermes query packet pending --chain vna-devnet-1 --port transfer --channel <CHANNEL_ID>
```

### Clear Stuck Packets (if needed)

```bash
hermes clear packets --chain vna-devnet-1 --port transfer --channel <CHANNEL_ID>
```

### Update Client Manually (if needed)

```bash
# Update Osmosis client with latest Verana state
hermes update client --host-chain osmosis-1 --client <OSMOSIS_CLIENT_ID>

# Update Verana client with latest Osmosis state
hermes update client --host-chain vna-devnet-1 --client <VERANA_CLIENT_ID>
```

### Check Telemetry

```bash
# View metrics (if telemetry enabled)
curl http://127.0.0.1:3001/metrics
```

---

## Troubleshooting

### Issue: "chain is not healthy"
**Solution:**
```bash
# Test RPC endpoints manually
curl https://rpc.devnet.verana.network/status
curl https://osmosis.rpc.kjnodes.com/status
```

### Issue: "timeout" or "clock drift" warnings
**Solution:** Already fixed in config with `clock_drift = '600s'`

### Issue: Packets timeout instead of relaying
**Cause:** Relayer took too long due to stale state  
**Solution:**
```bash
# Restart relayer
pkill hermes
hermes start

# Or manually update client
hermes update client --host-chain osmosis-1 --client <CLIENT_ID>
```

### Issue: "WebSocket connection failed"
**Solution:** The config uses kjnodes RPC with Polkachu WebSocket for optimal reliability. If issues persist, try these alternatives:

**Option A - LavendrFive:**
```toml
rpc_addr = 'https://osmosis-rpc.lavenderfive.com'
event_source = { mode = 'push', url = 'wss://osmosis-rpc.lavenderfive.com/websocket', batch_delay = '500ms' }
grpc_addr = 'https://osmosis-grpc.lavenderfive.com:443'
```

**Option B - All Polkachu:**
```toml
rpc_addr = 'https://osmosis-rpc.polkachu.com'
event_source = { mode = 'push', url = 'wss://osmosis-rpc.polkachu.com/websocket', batch_delay = '500ms' }
grpc_addr = 'https://osmosis-grpc.polkachu.com:12590'
```

### Issue: "insufficient fees"
**Solution:** Increase gas prices in config:
```toml
[chains.gas_price]
price = 50  # Increase from 25 for Verana
denom = 'uvna'
```

---

## Alternative Osmosis Endpoints

If the current kjnodes/Polkachu combination has issues, try these alternatives:

**LavendrFive (Very Reliable):**
```toml
rpc_addr = 'https://osmosis-rpc.lavenderfive.com'
grpc_addr = 'https://osmosis-grpc.lavenderfive.com:443'
event_source = { mode = 'push', url = 'wss://osmosis-rpc.lavenderfive.com/websocket', batch_delay = '500ms' }
```

**Stakely:**
```toml
rpc_addr = 'https://osmosis-rpc.stakely.io'
grpc_addr = 'https://osmosis-grpc.stakely.io:443'
event_source = { mode = 'push', url = 'wss://osmosis-rpc.stakely.io/websocket', batch_delay = '500ms' }
```

**PublicNode:**
```toml
rpc_addr = 'https://osmosis-rpc.publicnode.com'
grpc_addr = 'https://osmosis-grpc.publicnode.com:443'
event_source = { mode = 'push', url = 'wss://osmosis-rpc.publicnode.com/websocket', batch_delay = '500ms' }
```

---

## Production Setup: Run as System Service

For production, set up Hermes as a systemd service:

```bash
sudo nano /etc/systemd/system/hermes.service
```

```ini
[Unit]
Description=Hermes IBC Relayer
After=network.target

[Service]
Type=simple
User=YOUR_USERNAME
WorkingDirectory=/home/YOUR_USERNAME
ExecStart=/usr/local/bin/hermes start
Restart=always
RestartSec=10
StandardOutput=append:/home/YOUR_USERNAME/.hermes/hermes.log
StandardError=append:/home/YOUR_USERNAME/.hermes/hermes.log

[Install]
WantedBy=multi-user.target
```

```bash
# Enable and start service
sudo systemctl daemon-reload
sudo systemctl enable hermes
sudo systemctl start hermes

# Check status
sudo systemctl status hermes

# View logs
journalctl -u hermes -f
```

---

## Key Takeaways

✅ **Install Rust before Hermes**  
✅ **Create config BEFORE adding keys**  
✅ **Use high `clock_drift` values (600s) for mixed networks**  
✅ **Always check for existing IBC paths before creating new ones**  
✅ **Update packet filters with specific channel IDs after creation**  
✅ **kjnodes + Polkachu combination provides reliable Osmosis connectivity**  
✅ **Monitor relayer logs regularly**  
✅ **Keep sufficient balance on both chains for gas fees**

---

## Summary of Your Working Setup

**IBC Path: Verana Devnet ↔ Osmosis Mainnet**

| Component | Verana | Osmosis |
|-----------|--------|---------|
| Chain ID | vna-devnet-1 | osmosis-1 |
| Client ID | 07-tendermint-6 | 07-tendermint-3627 |
| Connection ID | connection-0 | connection-10967 |
| Channel ID | channel-0 | channel-107781 |
| Port | transfer | transfer |
| RPC | https://rpc.devnet.verana.network | https://osmosis.rpc.kjnodes.com |
| gRPC | http://node1.devnet.verana.network:9090 | https://osmosis.grpc.kjnodes.com:443 |
| WebSocket | ws://node1.devnet.verana.network:26657/websocket | wss://osmosis-rpc.polkachu.com/websocket |

**Your relayer is now successfully configured and running!** 🎉

---

## Useful Commands Reference

```bash
# Installation
rustc --version                                # Check Rust version
cargo --version                                # Check Cargo version
hermes version                                 # Check Hermes version

# Configuration
hermes config validate

# Health & Status
hermes health-check
hermes query channels --chain <CHAIN_ID>
hermes query connections --chain <CHAIN_ID>

# Operations
hermes start                                    # Start relayer
hermes clear packets --chain <CHAIN_ID>        # Clear packets
hermes update client --host-chain <CHAIN_ID>   # Update client

# Keys
hermes keys list --chain <CHAIN_ID>
hermes keys add --chain <CHAIN_ID> --key-name <NAME>

# Monitoring
tail -f $HOME/.hermes/hermes.log               # View logs
curl http://127.0.0.1:3001/metrics             # Telemetry
```

---

**Questions or issues?** Check the [Hermes documentation](https://hermes.informal.systems/) or join the Verana/Osmosis Discord communities for support.


## How to test the bridge?

```bash
osmosisd q bank balances $OSMOS_ACC_LIT --node $OSMO_RPC

osmosisd q bank balance $(osmosisd keys show $USER_ACC -a --keyring-backend test) "ibc/A29BAC98BBD293A8BEEB6413123CD3788035F27676DCD066FE0A869D5B39474C" --node $OSMO_RPC

osmosisd q bank balance $OSMOS_ACC_LIT "ibc/A29BAC98BBD293A8BEEB6413123CD3788035F27676DCD066FE0A869D5B39474C" --node $OSMO_RPC

veranad q bank balances $(veranad keys show $USER_ACC  -a --keyring-backend test) --node $NODE_RPC

veranad q bank balances $(veranad keys show cooluser -a --keyring-backend test) --node $NODE_RPC

osmosisd q ibc-transfer denom-trace \
  A29BAC98BBD293A8BEEB6413123CD3788035F27676DCD066FE0A869D5B39474C \
  --node $OSMO_RPC

```

```bash
NODE_RPC=https://rpc.devnet.verana.network
CHAIN_ID=vna-devnet-1
OSMOS_ACC=mat-test-acc
OSMOS_ACC_LIT=osmo1sxau0xyttphpck7vhlvt8s82ez70nlzwjgvynf
veranad tx ibc-transfer transfer transfer channel-1 $OSMOS_ACC_LIT 1000000uvna --chain-id $CHAIN_ID --keyring-backend test  --from cooluser --fees 800000uvna --gas 800000 -y --node $NODE_RPC


VERANA_GRPC=http://node1.devnet.verana.network:9090  # or set an env var
veranad tx ibc-transfer transfer transfer channel-1 \
  "$OSMOS_ACC_LIT" 1000000uvna \
  --chain-id "$CHAIN_ID" \
  --keyring-backend test \
  --from cooluser \
  --fees 800000uvna \
  --gas 800000 \
  --node "$NODE_RPC" \
  -y
```



```bash
OSMO_RPC=https://osmosis.rpc.kjnodes.com
OSMO_CHAIN_ID=osmosis-1
USER_ACC_LIT=verana1sxau0xyttphpck7vhlvt8s82ez70nlzw2mhya0

echo "$TEST_WALLET_MNEMO" | osmosisd keys add $USER_ACC --keyring-backend test --recover 



osmosisd tx ibc-transfer transfer transfer channel-107808 $USER_ACC_LIT 100000ibc/A29BAC98BBD293A8BEEB6413123CD3788035F27676DCD066FE0A869D5B39474C --chain-id $OSMO_CHAIN_ID --keyring-backend test  --from $USER_ACC --fees 80000uosmo --gas 800000 -y --node $OSMO_RPC
```

## troubleshooting tips

```bash
hermes update client --host-chain vna-devnet-1 --client 07-tendermint-7
hermes update client --host-chain osmosis-1 --client 07-tendermint-3629


hermes query packet pending --chain $CHAIN_ID --port transfer --channel channel-1

hermes clear packets --chain vna-devnet-1 --port transfer --channel channel-1

```

## How to remove a key

```bash
hermes keys delete --key-name osmosis-relayer  --chain osmosis-1
```