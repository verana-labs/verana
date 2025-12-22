/**
 * Journey: Touch DID
 *
 * This script demonstrates how to touch/update a DID's modified timestamp in the
 * DID Directory using the TypeScript client and the generated protobuf types.
 *
 * Usage:
 *   DID="did:verana:example" npm run test:touch-did
 *   # Or let it add a DID first, then touch it
 *   npm run test:touch-did
 *
 * Or with environment variables:
 *   export MNEMONIC="your mnemonic here"
 *   export VERANA_RPC_ENDPOINT="http://localhost:26657"
 *   export DID="did:verana:example"
 *   npm run test:touch-did
 */

import {
  createWallet,
  createSigningClient,
  getAccountInfo,
  calculateFeeWithSimulation,
  generateUniqueDID,
  config,
} from "../helpers/client";
import { typeUrls } from "../helpers/registry";
import { MsgAddDID, MsgTouchDID } from "../../../src/codec/verana/dd/v1/tx";

// Test mnemonic - Uses cooluser seed phrase (same as test harness)
const TEST_MNEMONIC =
  process.env.MNEMONIC ||
  "pink glory help gown abstract eight nice crazy forward ketchup skill cheese";

async function main() {
  console.log("=".repeat(60));
  console.log("Journey: Touch DID (TypeScript Client)");
  console.log("=".repeat(60));
  console.log();

  // Step 1: Setup wallet
  console.log("Step 1: Setting up wallet...");
  const wallet = await createWallet(TEST_MNEMONIC);
  const account = await getAccountInfo(wallet);
  console.log(`  ✓ Wallet address: ${account.address}`);
  console.log();

  // Step 2: Connect to blockchain
  console.log("Step 2: Connecting to Verana blockchain...");
  console.log(`  RPC Endpoint: ${config.rpcEndpoint}`);
  const client = await createSigningClient(wallet);
  console.log("  ✓ Connected successfully");
  console.log();

  // Step 3: Check account balance
  console.log("Step 3: Checking account balance...");
  const balance = await client.getBalance(account.address, config.denom);
  console.log(`  Balance: ${balance.amount} ${balance.denom}`);
  if (BigInt(balance.amount) < BigInt(1000000)) {
    console.log("  ⚠️  Warning: Low balance. You may need to fund this account.");
    process.exit(1);
  }
  console.log();

  // Step 4: Get or create DID
  let did: string;
  if (process.env.DID) {
    did = process.env.DID;
    console.log(`Step 4: Using provided DID: ${did}`);
  } else {
    console.log("Step 4: Adding a DID first (no DID provided)...");
    did = generateUniqueDID();
    const addMsg = {
      typeUrl: typeUrls.MsgAddDID,
      value: MsgAddDID.fromPartial({
        creator: account.address,
        did: did,
        years: 1,
      }),
    };

    const addFee = await calculateFeeWithSimulation(
      client,
      account.address,
      [addMsg],
      "Adding DID for touch test"
    );

    const addResult = await client.signAndBroadcast(
      account.address,
      [addMsg],
      addFee,
      "Adding DID for touch test"
    );

    if (addResult.code !== 0) {
      console.log("  ❌ Failed to add DID for touch test");
      console.log(`  Error: ${addResult.rawLog}`);
      process.exit(1);
    }

    console.log(`  ✓ Added DID: ${did}`);
  }
  console.log();

  // Step 5: Touch DID message
  console.log("Step 5: Creating Touch DID transaction...");
  const msg = {
    typeUrl: typeUrls.MsgTouchDID,
    value: MsgTouchDID.fromPartial({
      creator: account.address,
      did: did,
    }),
  };
  console.log("  Message details:");
  console.log(`    - Creator: ${account.address}`);
  console.log(`    - DID: ${did}`);
  console.log();

  // Step 6: Sign and broadcast
  console.log("Step 6: Signing and broadcasting transaction...");
  try {
    const fee = await calculateFeeWithSimulation(
      client,
      account.address,
      [msg],
      "Touching DID via TypeScript client"
    );
    console.log(`  Calculated gas: ${fee.gas}, fee: ${fee.amount[0].amount}${fee.amount[0].denom}`);
    
    const result = await client.signAndBroadcast(
      account.address,
      [msg],
      fee,
      "Touching DID via TypeScript client"
    );

    console.log();
    if (result.code === 0) {
      console.log("✅ SUCCESS! DID touched successfully!");
      console.log("=".repeat(60));
      console.log(`  Transaction Hash: ${result.transactionHash}`);
      console.log(`  Block Height: ${result.height}`);
      console.log(`  Gas Used: ${result.gasUsed}/${result.gasWanted}`);
    } else {
      console.log("❌ FAILED! Transaction failed.");
      console.log(`  Error Code: ${result.code}`);
      console.log(`  Raw Log: ${result.rawLog}`);
      process.exit(1);
    }
  } catch (error: any) {
    console.log("❌ ERROR! Transaction failed with exception:");
    console.error(error);
    
    if (error.cause?.code === "ECONNREFUSED" || error.message?.includes("fetch failed")) {
      console.error("\n⚠️  Connection Error: Cannot connect to the blockchain.");
      console.error(`   Make sure the Verana blockchain is running at ${config.rpcEndpoint}`);
      console.error("   Start it with: ./scripts/setup_primary_validator.sh");
    }
    
    process.exit(1);
  }

  console.log();
  console.log("=".repeat(60));
}

main().catch((error: any) => {
  console.error("\n❌ Fatal error:", error.message || error);
  
  if (error.cause?.code === "ECONNREFUSED" || error.message?.includes("fetch failed")) {
    console.error("\n⚠️  Connection Error: Cannot connect to the blockchain.");
    console.error(`   Make sure the Verana blockchain is running at ${config.rpcEndpoint}`);
    console.error("   Start it with: ./scripts/setup_primary_validator.sh");
  }
  
  process.exit(1);
});

