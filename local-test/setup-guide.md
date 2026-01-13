# Verana Multi-Validator Development Guide

Quick setup for testing Verana blockchain changes with 3 validators in Docker.

## Prerequisites

- Docker installed
- `jq` installed (`brew install jq` or `apt install jq`)

## Setup

All required files are already in the `local-test/` directory:
- `Dockerfile` - Docker image definition
- `build.sh` - Build script for Docker image
- `setup-validators.sh` - Setup script for 5 validators (regular, no cosmovisor)
- `setup-validators-cosmovisor.sh` - Setup script for 5 validators with cosmovisor
- `cleanup.sh` - Cleanup script

**First build:**
   ```bash
# From project root
./local-test/build.sh
./local-test/setup-validators.sh
   ```

Or if you're in the `local-test/` directory:
   ```bash
cd local-test
   ./build.sh
   ./setup-validators.sh
   ```

## Development Workflow

Every time you make code changes:

```bash
# 1. Build new Docker image with your changes
./local-test/build.sh

# 2. Clean old environment
./local-test/cleanup.sh

# 3. Start fresh 5-validator network
./local-test/setup-validators.sh

# 4. Test your changes
```

## Testing Your Changes

```bash
# Check if validators are working
curl -s http://localhost:26657/status | jq '.result.sync_info.latest_block_height'

# Check your binary version
docker exec val1 veranad version

# View logs
docker logs val1 -f

# Send test transaction
VAL2_ADDR=$(docker exec val2 veranad keys show val2 -a --keyring-backend test)
docker exec val1 veranad tx bank send val1 $VAL2_ADDR 1000000uvna \
  --chain-id vna-local-1 --keyring-backend test --fees 250uvna -y

# Check balance
docker exec val2 veranad query bank balances $VAL2_ADDR
```

## Network Info

- **3 validators:** val1, val2, val3
- **Chain ID:** `vna-local-1`
- **Ports:**
    - val1: RPC=26657, API=1317
    - val2: RPC=27657, API=2317
    - val3: RPC=28657, API=3317

## Quick Commands

```bash
# Full rebuild cycle (from project root)
./local-test/build.sh && ./local-test/cleanup.sh && ./local-test/setup-validators.sh

# Check all validator heights
for port in 26657 27657 28657; do
  echo "Port $port: $(curl -s http://localhost:$port/status | jq -r '.result.sync_info.latest_block_height')"
done

# Restart all validators (useful after upgrades)
docker restart validator1 validator2 validator3 validator4 validator5

# Check all validator heights (for cosmovisor setup with 5 validators)
for i in {1..5}; do
  echo "Validator $i: $(docker exec validator$i veranad status --node tcp://localhost:26657 2>/dev/null | jq -r '.sync_info.latest_block_height')"
done

# Reset everything if stuck
./local-test/cleanup.sh
docker rmi verana:dev
./local-test/build.sh && ./local-test/setup-validators.sh
```

## Testing Upgrades with Cosmovisor

To test chain upgrades in the local-test environment:

1. **Build cosmovisor-enabled image:**
   ```bash
   ./local-test/build-cosmovisor.sh
   ```

2. **Setup validators with cosmovisor:**
   ```bash
   # Use the cosmovisor-specific setup script
   ./local-test/setup-validators-cosmovisor.sh
   ```
   
   **Note:** This script is separate from `setup-validators.sh` and automatically uses the `verana:dev-cosmovisor` image.

3. **Download and prepare upgrade binary:**
   ```bash
   # Download the Linux binary for Docker (not darwin!)
   # Docker containers run Linux, so you need:
   #   - linux-amd64 for Intel/AMD
   #   - linux-arm64 for ARM (M1/M2 Macs with Linux containers)
   # 
   # Download from: https://github.com/verana-labs/verana/releases/tag/v0.9-dev.3
   # Then prepare for all validators:
   UPGRADE_BINARY_PATH=/path/to/veranad-linux-amd64 ./local-test/setup-cosmovisor-upgrade.sh
   ```
   
   **Note:** The setup script automatically makes the binary executable (`chmod +x`), so you don't need to do this manually.

4. **Query current block height:**
   ```bash
   # Get the current height to set upgrade height
   docker exec validator1 veranad status --node tcp://localhost:26657 | jq -r '.sync_info.latest_block_height'
   ```

5. **Submit upgrade proposal and vote from all validators:**
   ```bash
   # This script submits the proposal and votes from all 5 validators
   ./local-test/submit-upgrade-proposal-docker.sh <upgrade-height> <release-tag>
   # Example:
   ./local-test/submit-upgrade-proposal-docker.sh 400 v0.9
   ```
   
   **Note:** The voting period is only 100 seconds, so the script votes immediately from all validators.

6. **Monitor upgrade:** When the upgrade height is reached, cosmovisor will automatically switch binaries in all validators.
   ```bash
   # Watch logs to see the upgrade happen
   docker logs validator1 -f
   ```

7. **Restart validators after upgrade (if chain stops producing blocks):**
   
   **Why validators get stuck after upgrades:**
   - When Cosmovisor applies an upgrade, it restarts each validator with the new binary
   - Validators restart at slightly different times, causing temporary P2P connection losses
   - In Docker environments with dynamic ports, these connection issues can persist
   - Without P2P connections, validators cannot reach consensus and blocks stop being produced
   
   **Solution:** Restart all validators to re-establish P2P connections:
   ```bash
   # Restart all validators at once
   docker restart validator1 validator2 validator3 validator4 validator5
   
   # Wait a few seconds for them to start
   sleep 10
   
   # Verify they're producing blocks again
   for i in {1..5}; do
     echo "Validator $i: $(docker exec validator$i veranad status --node tcp://localhost:26657 2>/dev/null | jq -r '.sync_info.latest_block_height')"
   done
   ```
   
   **Note:** This is normal behavior in Docker environments. The restart allows validators to reconnect and continue block production.

## Troubleshooting

- **Build fails:** Run `go mod tidy` in your source code
- **Ports busy:** Run `./cleanup.sh` first
- **Validators not syncing:** Check `docker logs val1`
- **Complete reset:** `./local-test/cleanup.sh && docker rmi verana:dev && ./local-test/build.sh`
- **Cosmovisor not switching:** Verify upgrade binary exists in each validator's `cosmovisor/upgrades/<name>/bin/` directory
- **Wrong binary architecture:** Docker containers run Linux, so download `linux-amd64` or `linux-arm64`, not `darwin-arm64` or `darwin-amd64`
- **Binary not executable:** The `setup-cosmovisor-upgrade.sh` script automatically makes binaries executable, but if you manually copy, run `chmod +x` on the binary
- **Chain stops producing blocks after upgrade:** This is normal in Docker environments. Restart all validators: `docker restart validator1 validator2 validator3 validator4 validator5` (see "Testing Upgrades with Cosmovisor" section above for details)

That's it! Build → Clean → Setup → Test → Repeat for fast development cycles.