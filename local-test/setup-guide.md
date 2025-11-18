# Verana Multi-Validator Development Guide

Quick setup for testing Verana blockchain changes with 3 validators in Docker.

## Prerequisites

- Docker installed
- `jq` installed (`brew install jq` or `apt install jq`)

## Setup

All required files are already in the `local-test/` directory:
- `Dockerfile` - Docker image definition
- `build.sh` - Build script for Docker image
- `setup-validators.sh` - Setup script for 5 validators
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

2. **Update `local-test/setup-validators.sh` to use cosmovisor image:**
   ```bash
   # Change line 11 from:
   DOCKER_IMAGE="verana:dev"
   # To:
   DOCKER_IMAGE="verana:dev-cosmovisor"
   ```

3. **Setup validators (they will use cosmovisor):**
   ```bash
   ./local-test/setup-validators.sh
   ```

4. **Prepare upgrade binary:**
   ```bash
   # Download upgrade binary and prepare for all validators
   UPGRADE_BINARY_PATH=/path/to/veranad-binary ./local-test/setup-cosmovisor-upgrade.sh
   ```

5. **Submit upgrade proposal and vote** (use scripts from `scripts/cosmovisor/`)

6. **Monitor upgrade:** When the upgrade height is reached, cosmovisor will automatically switch binaries in all validators.

## Troubleshooting

- **Build fails:** Run `go mod tidy` in your source code
- **Ports busy:** Run `./cleanup.sh` first
- **Validators not syncing:** Check `docker logs val1`
- **Complete reset:** `./local-test/cleanup.sh && docker rmi verana:dev && ./local-test/build.sh`
- **Cosmovisor not switching:** Verify upgrade binary exists in each validator's `cosmovisor/upgrades/<name>/bin/` directory

That's it! Build → Clean → Setup → Test → Repeat for fast development cycles.