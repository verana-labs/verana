# Verana TypeScript Client Tests

This directory contains TypeScript tests that demonstrate how to interact with the Verana blockchain using the generated protobuf types. The tests use the same transaction signing approach as the frontend application.

## Prerequisites

1. **Node.js 18+** installed
2. A running **Verana blockchain** (local or testnet)
3. The `cooluser` account funded (used by default in tests)

## Setup

```bash
# Navigate to the test directory
cd ts-proto/test

# Install dependencies
npm install

# Build the parent ts-proto package first (if not already built)
cd ..
npm install
npm run build
cd test
```

## Configuration

The tests use environment variables for configuration. Defaults match the frontend configuration:

| Variable | Default | Description |
|----------|---------|-------------|
| `VERANA_RPC_ENDPOINT` | `http://localhost:26657` | Tendermint RPC endpoint |
| `VERANA_LCD_ENDPOINT` | `http://localhost:1317` | LCD REST API endpoint |
| `VERANA_CHAIN_ID` | `verana` | Chain ID |
| `VERANA_ADDRESS_PREFIX` | `verana` | Bech32 address prefix |
| `VERANA_DENOM` | `uvna` | Token denomination |
| `VERANA_GAS_PRICE` | `3uvna` | Gas price (matches frontend) |
| `VERANA_GAS_LIMIT` | `300000` | Default gas limit (matches frontend) |
| `VERANA_GAS_ADJUSTMENT` | `2` | Gas adjustment multiplier (matches frontend) |
| `MNEMONIC` | `cooluser` seed phrase | Wallet mnemonic (see below) |

### Default Test Account

The tests use the `cooluser` account by default, which is the same account used in the Go test harness. This account is automatically funded when you initialize a local chain using `./scripts/setup_primary_validator.sh`.

**Default mnemonic (cooluser):**
```
pink glory help gown abstract eight nice crazy forward ketchup skill cheese
```

**Address:** `verana16mzeyu9l6kua2cdg9x0jk5g6e7h0kk8q6uadu4`

This account has 1,000,000,000,000,000,000,000 uvna (1 billion VNA) in genesis when using the setup scripts.

## Running Tests

### Create a Trust Registry

```bash
# Using the default cooluser mnemonic (recommended for local testing)
npm run test:create-tr

# Using your own mnemonic
MNEMONIC="your twelve word mnemonic phrase goes here" npm run test:create-tr

# With custom endpoint
VERANA_RPC_ENDPOINT="https://rpc.testnet.verana.network:443" \
MNEMONIC="your mnemonic" \
npm run test:create-tr
```

### Query Trust Registries

```bash
# Query all trust registries
npm run test:query-tr

# With custom LCD endpoint
VERANA_LCD_ENDPOINT="https://lcd.testnet.verana.network" npm run test:query-tr
```

## Local Testing Setup

### Quick Start (Recommended)

1. **Initialize and start a local chain** (from repo root):
   ```bash
   ./scripts/setup_primary_validator.sh
   ```
   
   This script:
   - Initializes the chain with `cooluser` as the validator
   - Funds `cooluser` with 1 billion VNA in genesis
   - Starts the blockchain node
   - The `cooluser` account is ready to use immediately

2. **Run the test**:
   ```bash
   cd ts-proto/test
   npm run test:create-tr
   ```

The `cooluser` account is already funded, so no additional funding is needed!

### Manual Setup

If you're using a different chain setup:

1. **Start the local chain**:
   ```bash
   # From the repo root
   veranad start
   ```

2. **Fund the cooluser account** (if not already funded):
   ```bash
   # Get the cooluser address
   veranad keys show cooluser -a --keyring-backend test
   
   # Fund from another account (if needed)
   veranad tx bank send <validator-address> verana16mzeyu9l6kua2cdg9x0jk5g6e7h0kk8q6uadu4 10000000uvna \
     --chain-id verana \
     --keyring-backend test \
     --fees 5000uvna \
     --yes
   ```

3. **Run the test**:
   ```bash
   cd ts-proto/test
   npm run test:create-tr
   ```

## How It Works

The test harness uses the same transaction signing approach as the frontend:

1. **Gas Simulation**: Uses `client.simulate()` to estimate gas usage
2. **Gas Adjustment**: Applies a 2x multiplier for safety (matches frontend)
3. **Fee Calculation**: Uses `calculateFee()` from `@cosmjs/stargate`
4. **Registry**: Custom registry with all Verana message types registered
5. **Direct Signing**: Uses `SigningStargateClient` with direct signing mode

This ensures compatibility with the frontend application and validates that the TypeScript protobuf types work correctly.

## Test Structure

```
test/
├── package.json           # Dependencies and scripts
├── tsconfig.json          # TypeScript configuration
├── README.md              # This file
└── src/
    ├── helpers/
    │   ├── client.ts      # CosmJS client setup utilities (matches frontend)
    │   ├── registry.ts    # Custom type registry for Verana messages
    │   └── index.ts       # Helper exports
    └── journeys/
        ├── createTrustRegistry.ts  # Create a trust registry
        └── queryTrustRegistry.ts   # Query trust registries
```

## Using in Your Frontend

The same patterns work in a frontend application. The test harness code is designed to match the frontend's transaction signing approach:

```typescript
import { DirectSecp256k1HdWallet } from "@cosmjs/proto-signing";
import { SigningStargateClient } from "@cosmjs/stargate";
import { createVeranaRegistry, typeUrls } from "./helpers/registry";
import { MsgCreateTrustRegistry } from "@verana-labs/verana-types/codec/verana/tr/v1/tx";
import { calculateFeeWithSimulation } from "./helpers/client";

// Create wallet (in browser, use Keplr or similar)
const wallet = await DirectSecp256k1HdWallet.fromMnemonic(mnemonic, {
  prefix: "verana",
});

// Create client with custom registry
const client = await SigningStargateClient.connectWithSigner(
  "http://localhost:26657",
  wallet,
  { registry: createVeranaRegistry() }
);

// Create message
const msg = {
  typeUrl: typeUrls.MsgCreateTrustRegistry,
  value: MsgCreateTrustRegistry.fromPartial({
    creator: address,
    did: "did:verana:example",
    aka: "http://example.com", // Must be a valid URI
    language: "en",
    docUrl: "https://example.com/gf.pdf",
    docDigestSri: "sha384-...",
  }),
};

// Calculate fee using simulation (matches frontend)
const fee = await calculateFeeWithSimulation(
  client,
  address,
  [msg],
  "Creating Trust Registry"
);

// Sign and broadcast
const result = await client.signAndBroadcast(address, [msg], fee, "Creating Trust Registry");
```

## Integrating with Keplr (Browser)

For browser-based frontends using Keplr, see the frontend implementation in `/Users/pratik/verana-frontend/app/msg/util/sendTxDetectingMode.ts` for the complete pattern.

## Troubleshooting

### "Account not found"
The `cooluser` account hasn't been funded yet. Make sure you've initialized the chain using `./scripts/setup_primary_validator.sh` which automatically funds this account.

### "Insufficient fees"
The gas simulation should handle this automatically. If you see this error, check:
- The account has sufficient balance
- The gas price is set correctly (default: `3uvna`)
- The chain is producing blocks

### "Invalid AKA URI"
The `aka` field must be a valid URI (e.g., `http://example.com`), not plain text.

### "Invalid type URL"
Make sure you're using the correct type URL from `typeUrls` and that the message is registered in the registry.

### Connection refused
Check that the blockchain node is running and accessible at the configured endpoints:
- RPC: `http://localhost:26657`
- REST: `http://localhost:1317`

### Gas simulation fails
If gas simulation fails, you can fall back to a fixed fee:
```typescript
import { getDefaultFee } from "./helpers/client";
const fee = getDefaultFee("300000"); // Use fixed 300k gas
```

## Alignment with Frontend

This test harness is designed to match the frontend's transaction signing approach:

- ✅ Same gas price: `3uvna`
- ✅ Same gas adjustment: `2`
- ✅ Same gas simulation approach
- ✅ Same registry registration pattern
- ✅ Same `SigningStargateClient` usage

The frontend uses a more manual signing flow (see `signAndBroadcastManualDirect.ts`), but both approaches use the same gas calculation and configuration, ensuring compatibility.
