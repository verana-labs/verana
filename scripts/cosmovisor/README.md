# Cosmovisor Upgrade Testing Guide

This directory contains scripts and documentation for testing chain upgrades using cosmovisor.

## Prerequisites

Before starting, ensure you have:

1. **Cosmovisor installed**
   ```bash
   go install cosmossdk.io/tools/cosmovisor/cmd/cosmovisor@latest
   ```

2. **Current chain binary** (`veranad`) in your PATH

3. **Upgrade binary downloaded manually**
   - Go to: https://github.com/verana-labs/verana/releases/tag/v0.9-dev.3
   - Download the binary for your platform:
     - `darwin-arm64` for Mac M1/M2
     - `darwin-amd64` for Intel Mac
   - Place it at: `~/.verana/cosmovisor/upgrades/v0.9/bin/veranad`
   
   **macOS Security Note:** After downloading, you may need to allow the binary:
   ```bash
   # Remove quarantine attribute
   xattr -d com.apple.quarantine ~/.verana/cosmovisor/upgrades/v0.9/bin/veranad
   
   # Make executable
   chmod +x ~/.verana/cosmovisor/upgrades/v0.9/bin/veranad
   ```
   
   If you still get "killed" error, go to: **System Settings → Privacy & Security → Allow "veranad"**

## Step-by-Step Guide

### Step 1: Query Current Block Height

First, check the current block height to determine the upgrade height:

```bash
./scripts/cosmovisor/query_height.sh
```

This will output the current block height. Add ~50-100 blocks to get your upgrade height.
For example, if current height is 350, set upgrade height to 400-450.

### Step 2: Setup Cosmovisor

Setup the cosmovisor directory structure:

```bash
./scripts/cosmovisor/setup_cosmovisor.sh
```

This script will:
- Check if cosmovisor is installed
- Create the directory structure
- Copy your current binary to `genesis/bin/`
- Check if the upgrade binary exists (will prompt if missing)
- Create environment file and start script

**Important:** Make sure you've downloaded the upgrade binary before running this script.

### Step 3: Start Chain with Cosmovisor

Stop your current chain (if running), then start it with cosmovisor:

```bash
source ~/.verana/cosmovisor/cosmovisor.env
cosmovisor run start
```

Or use the start script:
```bash
~/.verana/cosmovisor/start.sh
```

### Step 4: Submit Upgrade Proposal

Once the chain is running with cosmovisor, submit the upgrade proposal:

```bash
./scripts/cosmovisor/submit_proposal.sh <upgrade-height> <upgrade-name>
```

Example:
```bash
./scripts/cosmovisor/submit_proposal.sh 400 v0.9
```

**Note:** The upgrade height must be in the future (higher than current block height).

### Step 5: Query Proposal

Check the proposal details:

```bash
# Query latest proposal
veranad query gov proposals --node tcp://localhost:26657

# Query specific proposal by ID
veranad query gov proposal <proposal-id> --node tcp://localhost:26657

# Get proposal ID from latest proposal
PROPOSAL_ID=$(veranad query gov proposals --node tcp://localhost:26657 --output json | jq -r '.proposals[0].id')
echo "Proposal ID: $PROPOSAL_ID"
```

### Step 6: Vote on Proposal (WITHIN 100 SECONDS!)

**IMPORTANT:** The voting period is only 100 seconds, so vote immediately:

```bash
# Vote on a specific proposal
veranad tx gov vote <proposal-id> yes \
    --from cooluser \
    --chain-id vna-testnet-1 \
    --keyring-backend test \
    --fees 600000uvna \
    --node tcp://localhost:26657 \
    --yes

# Or get proposal ID and vote in one command
PROPOSAL_ID=$(veranad query gov proposals --node tcp://localhost:26657 --limit 1 --reverse --output json | jq -r '.proposals[0].id')
veranad tx gov vote $PROPOSAL_ID yes \
    --from cooluser \
    --chain-id vna-testnet-1 \
    --keyring-backend test \
    --fees 600000uvna \
    --node tcp://localhost:26657 \
    --yes
```

### Step 7: Monitor Upgrade

Monitor the chain until it reaches the upgrade height:

```bash
# Watch block height
watch -n 1 'veranad status --node tcp://localhost:26657 | jq -r .sync_info.latest_block_height'

# Or query periodically
veranad status --node tcp://localhost:26657 | jq -r .sync_info.latest_block_height
```

When the chain reaches the upgrade height, cosmovisor will automatically:
1. Stop the chain
2. Switch to the upgrade binary
3. Restart the chain

## Scripts Overview

- **`query_height.sh`** - Query current block height
- **`setup_cosmovisor.sh`** - Setup cosmovisor directory structure
- **`submit_proposal.sh`** - Submit software upgrade proposal

## Environment Variables

You can override default values:

```bash
export CHAIN_ID="vna-testnet-1"
export BINARY="veranad"
export NODE="tcp://localhost:26657"
export FROM="cooluser"
export KEYRING_BACKEND="test"
export UPGRADE_NAME="v0.9"  # Used for directory name (matches upgrade plan name)
```