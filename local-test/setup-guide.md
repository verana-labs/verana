# Verana Multi-Validator Development Guide

Quick setup for testing Verana blockchain changes with 3 validators in Docker.

## Prerequisites

- Docker installed
- `jq` installed (`brew install jq` or `apt install jq`)

## Setup

1. **Create the 4 files** (Dockerfile, build.sh, setup-validators.sh, cleanup.sh) in your Verana project root

2. **Make scripts executable:**
   ```bash
   chmod +x build.sh setup-validators.sh cleanup.sh
   ```

3. **First build:**
   ```bash
   ./build.sh
   ./setup-validators.sh
   ```

## Development Workflow

Every time you make code changes:

```bash
# 1. Build new Docker image with your changes
./build.sh

# 2. Clean old environment
./cleanup.sh

# 3. Start fresh 5-validator network
./setup-validators.sh

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
# Full rebuild cycle
./build.sh && ./cleanup.sh && ./setup-validators.sh

# Check all validator heights
for port in 26657 27657 28657; do
  echo "Port $port: $(curl -s http://localhost:$port/status | jq -r '.result.sync_info.latest_block_height')"
done

# Reset everything if stuck
./cleanup.sh
docker rmi verana:dev
./build.sh && ./setup-validators.sh
```

## Troubleshooting

- **Build fails:** Run `go mod tidy` in your source code
- **Ports busy:** Run `./cleanup.sh` first
- **Validators not syncing:** Check `docker logs val1`
- **Complete reset:** `./cleanup.sh && docker rmi verana:dev && ./build.sh`

That's it! Build → Clean → Setup → Test → Repeat for fast development cycles.