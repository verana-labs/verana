/**
 * Journey: Update Trust Registry
 *
 * This script demonstrates how to update a Trust Registry using the
 * TypeScript client and the generated protobuf types.
 *
 * Usage:
 *   TR_ID=1 npm run test:update-tr
 *   # Or let it create a TR first, then update it
 *   npm run test:update-tr
 *
 * Or with environment variables:
 *   export MNEMONIC="your mnemonic here"
 *   export VERANA_RPC_ENDPOINT="http://localhost:26657"
 *   export TR_ID=1
 *   npm run test:update-tr
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
import { MsgCreateTrustRegistry, MsgUpdateTrustRegistry } from "../../../src/codec/verana/tr/v1/tx";

// Test mnemonic - Uses cooluser seed phrase (same as test harness)
const TEST_MNEMONIC =
  (process.env.MNEMONIC && process.env.MNEMONIC.trim()) ||
  "pink glory help gown abstract eight nice crazy forward ketchup skill cheese";

async function main() {
  console.log("=".repeat(60));
  console.log("Journey: Update Trust Registry (TypeScript Client)");
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

  // Step 4: Get or create Trust Registry ID
  let trId: number | undefined;
  if (process.env.TR_ID) {
    trId = parseInt(process.env.TR_ID, 10);
    if (isNaN(trId)) {
      console.log("  ❌ Invalid TR_ID provided");
      process.exit(1);
    }
    console.log(`Step 4: Using provided Trust Registry ID: ${trId}`);
  } else {
    console.log("Step 4: Creating a Trust Registry first (no TR_ID provided)...");
    const did = generateUniqueDID();
    const createMsg = {
      typeUrl: typeUrls.MsgCreateTrustRegistry,
      value: MsgCreateTrustRegistry.fromPartial({
        creator: account.address,
        did: did,
        aka: "http://example-trust-registry.com",
        language: "en",
        docUrl: "https://example.com/governance-framework.pdf",
        docDigestSri: "sha384-MzNNbQTWCSUSi0bbz7dbua+RcENv7C6FvlmYJ1Y+I727HsPOHdzwELMYO9Mz68M26",
      }),
    };

    const createFee = await calculateFeeWithSimulation(
      client,
      account.address,
      [createMsg],
      "Creating Trust Registry for update test"
    );

    const createResult = await client.signAndBroadcast(
      account.address,
      [createMsg],
      createFee,
      "Creating Trust Registry for update test"
    );

    if (createResult.code !== 0) {
      console.log("  ❌ Failed to create Trust Registry for update test");
      console.log(`  Error: ${createResult.rawLog}`);
      process.exit(1);
    }

    // Extract TR ID from events
    // Event type is "create_trust_registry" and attribute key is "trust_registry_id"
    const events = createResult.events || [];
    for (const event of events) {
      if (event.type === "create_trust_registry" || event.type === "verana.tr.v1.EventCreateTrustRegistry") {
        for (const attr of event.attributes) {
          // Try multiple possible attribute keys
          if (attr.key === "trust_registry_id" || attr.key === "id" || attr.key === "tr_id") {
            trId = parseInt(attr.value, 10);
            if (!isNaN(trId)) {
              console.log(`  ✓ Created Trust Registry with ID: ${trId}`);
              break;
            }
          }
        }
        if (trId) break;
      }
    }
    
    // Debug: print all events if ID not found
    if (!trId) {
      console.log("  Debug: Available events:");
      for (const event of events) {
        console.log(`    Event type: ${event.type}`);
        for (const attr of event.attributes) {
          console.log(`      ${attr.key}: ${attr.value}`);
        }
      }
    }

    if (!trId || isNaN(trId)) {
      console.log("  ❌ Could not extract TR ID from events");
      process.exit(1);
    }
  }

  // Ensure trId is defined
  if (!trId) {
    console.log("  ❌ Trust Registry ID is required");
    process.exit(1);
  }

  console.log();

  // Step 5: Update Trust Registry
  console.log("Step 5: Updating Trust Registry transaction...");
  const newDid = generateUniqueDID();
  const newAka = "http://updated-trust-registry.com";
  const msg = {
    typeUrl: typeUrls.MsgUpdateTrustRegistry,
    value: MsgUpdateTrustRegistry.fromPartial({
      creator: account.address,
      id: trId,
      did: newDid,
      aka: newAka,
    }),
  };
  console.log("  Message details:");
  console.log(`    - Creator: ${account.address}`);
  console.log(`    - Trust Registry ID: ${trId}`);
  console.log(`    - New DID: ${newDid}`);
  console.log(`    - New AKA: ${newAka}`);
  console.log();

  // Step 6: Sign and broadcast
  console.log("Step 6: Signing and broadcasting transaction...");
  try {
    const fee = await calculateFeeWithSimulation(
      client,
      account.address,
      [msg],
      "Updating Trust Registry via TypeScript client"
    );
    console.log(`  Calculated gas: ${fee.gas}, fee: ${fee.amount[0].amount}${fee.amount[0].denom}`);
    
    const result = await client.signAndBroadcast(
      account.address,
      [msg],
      fee,
      "Updating Trust Registry via TypeScript client"
    );

    console.log();
    if (result.code === 0) {
      console.log("✅ SUCCESS! Trust Registry updated successfully!");
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

