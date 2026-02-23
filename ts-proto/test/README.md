# Verana TypeScript Client Tests

TypeScript tests for interacting with the Verana blockchain using generated protobuf types. These tests use the same transaction signing approach as our frontend application.

## Testing Strategy

### Overview

We use these TypeScript client tests as an **early warning system** to catch proto alignment issues immediately after proto generation. When proto files are updated, the frontend can become misaligned with the blockchain, causing hours of debugging. These tests validate that the frontend can sign and broadcast transactions correctly.

### Two-Layer Testing Approach

1. **Go Test Harness** (Comprehensive Logic Testing)
   - Tests all blockchain logic and business rules
   - Validates end-to-end workflows (19+ journeys)
   - Ensures the blockchain behaves correctly
   - Located in: `verana-test-harness` repository

2. **TypeScript Client Tests** (Frontend Alignment Testing)
   - Tests transaction signing and proto alignment
   - Validates that TypeScript protobuf types work correctly
   - Ensures frontend can sign and broadcast transactions
   - Located in: `ts-proto/test/` directory (this directory)

### Why Both?

- **Go tests** ensure the blockchain logic is correct
- **TS tests** ensure the frontend can interact with the blockchain correctly
- **Different concerns**: Logic correctness vs. API compatibility

### What These Tests Cover

#### ✅ DO Test:
- **Transaction Signing**: Can we sign each message type correctly?
- **Proto Alignment**: Do the TypeScript types match the proto definitions?
- **Registry Registration**: Are all message types registered correctly?
- **Gas Calculation**: Does gas simulation work with our messages?
- **Type URLs**: Are type URLs correct and match the blockchain?
- **Message Construction**: Can we build messages using the generated types?

#### ❌ DON'T Test:
- **Business Logic**: That's what the Go test harness is for
- **Complex Workflows**: Focus on individual transaction types
- **Edge Cases**: Keep it simple - just verify transactions can be sent

### CI/CD Integration

These tests run automatically in GitHub Actions when proto files change:

```
Proto Changes
    ↓
Generate TS Types (make proto-ts)
    ↓
Run TS Client Tests (tsClientTests.yml) ← Fast feedback (5-10 min)
    ↓
Run Go Test Harness (testHarness.yml) ← Comprehensive (20-30 min)
    ↓
Merge PR
```

### Benefits

1. **Early Detection**: Catch proto alignment issues before frontend integration
2. **Fast Feedback**: TS tests run in 5-10 minutes vs. full test harness 20-30 minutes
3. **Frontend Reference**: Frontend devs can see exactly how to sign transactions
4. **Documentation**: Tests serve as living documentation for transaction signing
5. **Confidence**: Know that proto changes won't break the frontend

## Prerequisites

1. **Node.js 18+** installed
2. A running **Verana blockchain** (local or testnet)
3. The `cooluser` account funded (used by default in tests)

## Setup

```bash
# From the repository root, build the parent ts-proto package first
cd ts-proto
npm install
npm run build

# Then navigate to the test directory and install test dependencies
cd test
npm install
```

## Configuration

We configure tests using environment variables. Defaults match the frontend configuration:

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

We use the `cooluser` account by default, which is the same account used in the Go test harness. This account is automatically funded when you initialize a local chain using `./scripts/setup_primary_validator.sh`.

**Default mnemonic (cooluser):**
```
pink glory help gown abstract eight nice crazy forward ketchup skill cheese
```

**Address:** `verana16mzeyu9l6kua2cdg9x0jk5g6e7h0kk8q6uadu4`

This account has 1,000,000,000,000,000,000,000 uvna (1 billion VNA) in genesis when using the setup scripts.

## Running Tests

### Run All Tests

```bash
# Run all TypeScript client tests sequentially
npm run test:all
```

This executes all test journeys and provides a summary report.

### Amino Sign Bench (Frontend Debug Aid)

When the frontend struggles with invalid tx signatures, use the Amino sign bench to verify the exact bytes being signed. It demonstrates why small JSON differences (like including zero fees) produce different sign bytes and fail verification.

```bash
# From ts-proto/test
npx tsx scripts/benches/amino/perm/ts.ts
```

Notes:
- Uses `VERANA_RPC_ENDPOINT` to fetch on-chain `account_number` and `sequence`.
- Falls back to `0, 0` if the node is unreachable (still useful for comparing sign bytes).
- Edit the message payload in `ts-proto/test/scripts/benches/amino/perm/ts.ts` to test new message types.
- Root cause example: the chain omits zero-value fields in legacy Amino JSON (Go `omitempty`), while the client originally included `"verification_fees": "0"` / `"validation_fees": "0"`. That changes the sign bytes and causes signature verification to fail. The bench shows “client-style” (zeros included) vs “server-style” (zeros omitted) bytes.
- After running both benches, compare outputs with `node ts-proto/test/scripts/benches/amino/perm/compare.js`. The script normalizes JSON by sorting keys, so raw JSON strings can differ even when the bytes match.

### Go Corollary (Server-Style Sign Bytes)

If you want to reproduce the server's legacy Amino sign bytes in Go (and compare them to a client-style payload), run:

```bash
# From repo root
go run ts-proto/test/scripts/benches/amino/perm/go.go
```

This prints the server-style sign bytes (omitting zero fees) and a client-style variant that includes zeros, showing the exact byte mismatch.

### Credential Schema Bench (CS)

The Credential Schema bench follows the same structure under `ts-proto/test/scripts/benches/amino/cs/` with `ts.ts`, `go.go`, and `compare.js`.

### Recommended Run Sequence (TS → Go → Compare)

Run the three scripts in this order so the comparison uses fresh outputs:

```bash
# From repo root
npx tsx ts-proto/test/scripts/benches/amino/perm/ts.ts
go run ts-proto/test/scripts/benches/amino/perm/go.go
node ts-proto/test/scripts/benches/amino/perm/compare.js
```

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
   - Makes the `cooluser` account ready to use immediately

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

We use the same transaction signing approach as the frontend:

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
```

Runtime artifacts:
- `journey_results/*.json` are generated/overwritten by journeys to pass IDs between steps.
- `out/` is generated by Amino bench scripts.
- Both are intentionally gitignored.

## Using in Your Frontend

You can use these same patterns in your frontend application. The test code matches our frontend's transaction signing approach:

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

For browser-based frontends using Keplr, see our frontend implementation in `verana-frontend/app/msg/util/sendTxDetectingMode.ts` for the complete pattern.

## Troubleshooting

### "Account not found"
The `cooluser` account hasn't been funded yet. Initialize the chain using `./scripts/setup_primary_validator.sh` which automatically funds this account.

### "Insufficient fees"
Gas simulation should handle this automatically. If you see this error, check:
- The account has sufficient balance
- The gas price is set correctly (default: `3uvna`)
- The chain is producing blocks

### "Invalid AKA URI"
The `aka` field must be a valid URI (e.g., `http://example.com`), not plain text.

### "Invalid type URL"
Use the correct type URL from `typeUrls` and ensure the message is registered in the registry.

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

These tests match our frontend's transaction signing approach:

- ✅ Same gas price: `3uvna`
- ✅ Same gas adjustment: `2`
- ✅ Same gas simulation approach
- ✅ Same registry registration pattern
- ✅ Same `SigningStargateClient` usage

Our frontend uses a more manual signing flow (see `signAndBroadcastManualDirect.ts`), but both approaches use the same gas calculation and configuration, ensuring compatibility.

## Adding New Tests

To add a new transaction type test:

1. **Create a journey file** in `src/journeys/`
   - Name: `create<MessageName>.ts` or `send<MessageName>.ts`
   - Follow the pattern from `createTrustRegistry.ts`

2. **Add to package.json** scripts:
   ```json
   "test:create-<name>": "npx tsx src/journeys/create<Name>.ts"
   ```

3. **Add to runAll.ts**:
   ```typescript
   { name: "Create <Name>", script: "test:create-<name>" }
   ```

4. **Register in registry.ts** (if not already):
   - Add import
   - Add type URL
   - Register in `createVeranaRegistry()`

## Priority Test Coverage

We focus on testing transaction types that:
1. Are used frequently by the frontend
2. Have complex message structures
3. Have been problematic in the past
4. Are critical for user workflows

Our priority order:
1. ✅ Trust Registry (create, update, archive)
2. ⏳ Credential Schema (create, update, archive)
3. ⏳ DID Directory (add, renew, remove, touch)
4. ⏳ Permission (create, extend, revoke, VP operations)
5. ⏳ Trust Deposit (reclaim, slash, repay)

## Maintenance

- **Keep tests simple**: Don't over-engineer - just verify transactions work
- **Update when proto changes**: Add tests for new message types
- **Remove obsolete tests**: If a message type is deprecated, remove its test
- **Sync with Go tests**: We use the same test account and chain setup
