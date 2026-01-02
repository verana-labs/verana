/**
 * Journey: Create Trust Registry
 *
 * This script demonstrates how to create a Trust Registry using the
 * TypeScript client and the generated protobuf types.
 *
 * Usage:
 *   MNEMONIC="your mnemonic here" npm run test:create-tr
 *
 * Or with environment variables:
 *   export MNEMONIC="your mnemonic here"
 *   export VERANA_RPC_ENDPOINT="http://localhost:26657"
 *   npm run test:create-tr
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
import { MsgCreateTrustRegistry } from "../../../src/codec/verana/tr/v1/tx";
import { saveJourneyResult } from "../helpers/journeyResults";

// Test mnemonic - Uses cooluser seed phrase (same as test harness)
// This account is pre-funded in local chains initialized with setup_primary_validator.sh
const TEST_MNEMONIC =
  (process.env.MNEMONIC && process.env.MNEMONIC.trim()) ||
  "pink glory help gown abstract eight nice crazy forward ketchup skill cheese";

async function main() {
  console.log("=".repeat(60));
  console.log("Journey: Create Trust Registry (TypeScript Client)");
  console.log("=".repeat(60));
  console.log();

  // Step 1: Setup wallet (using Amino Sign to match frontend)
  console.log("Step 1: Setting up wallet (Amino Sign mode)...");
  const wallet = await createWallet(TEST_MNEMONIC);
  const account = await getAccountInfo(wallet);
  console.log(`  ✓ Wallet address: ${account.address}`);
  console.log(`  ✓ Using Amino Sign (matches frontend)`);
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
    console.log(`     Use: veranad tx bank send <from> ${account.address} 10000000uvna --chain-id ${config.chainId}`);
  }
  console.log();

  // Step 4: Generate unique DID
  console.log("Step 4: Generating unique DID...");
  const did = generateUniqueDID();
  console.log(`  ✓ DID: ${did}`);
  console.log();

  // Step 5: Create Trust Registry message
  console.log("Step 5: Creating Trust Registry transaction...");
  const aka = "http://example-trust-registry.com";
  const msg = {
    typeUrl: typeUrls.MsgCreateTrustRegistry,
    value: MsgCreateTrustRegistry.fromPartial({
      creator: account.address,
      did: did,
      aka: aka,
      language: "en",
      docUrl: "https://example.com/governance-framework.pdf",
      docDigestSri: "sha384-MzNNbQTWCSUSi0bbz7dbua+RcENv7C6FvlmYJ1Y+I727HsPOHdzwELMYO9Mz68M26",
    }),
  };
  console.log("  Message details:");
  console.log(`    - Creator: ${account.address}`);
  console.log(`    - DID: ${did}`);
  console.log(`    - AKA: ${aka}`);
  console.log(`    - Language: en`);
  console.log();

  // Step 6: Sign and broadcast (using gas simulation like frontend)
  console.log("Step 6: Signing and broadcasting transaction...");
  try {
    // Use gas simulation to calculate fee (matches frontend approach)
    const fee = await calculateFeeWithSimulation(
      client,
      account.address,
      [msg],
      "Creating Trust Registry via TypeScript client"
    );
    console.log(`  Calculated gas: ${fee.gas}, fee: ${fee.amount[0].amount}${fee.amount[0].denom}`);
    
    const result = await client.signAndBroadcast(
      account.address,
      [msg],
      fee,
      "Creating Trust Registry via TypeScript client"
    );

    console.log();
    if (result.code === 0) {
      console.log("✅ SUCCESS! Trust Registry created successfully!");
      console.log("=".repeat(60));
      console.log(`  Transaction Hash: ${result.transactionHash}`);
      console.log(`  Block Height: ${result.height}`);
      console.log(`  Gas Used: ${result.gasUsed}/${result.gasWanted}`);
      console.log();

      // Try to extract the trust registry ID from events
      const events = result.events || [];
      let trId: number | undefined;
      for (const event of events) {
        if (event.type === "create_trust_registry" || event.type === "verana.tr.v1.EventCreateTrustRegistry") {
          for (const attr of event.attributes) {
            console.log(`  Event ${attr.key}: ${attr.value}`);
            if (attr.key === "trust_registry_id" || attr.key === "id" || attr.key === "tr_id") {
              trId = parseInt(attr.value, 10);
              if (!isNaN(trId)) {
                console.log(`  ✓ Trust Registry ID: ${trId}`);
              }
            }
          }
        }
      }
      
      // Save as active TR for reuse
      if (trId) {
        const { saveActiveTR } = await import("../helpers/journeyResults");
        saveActiveTR(trId, did);
      }
    } else {
      console.log("❌ FAILED! Transaction failed.");
      console.log(`  Error Code: ${result.code}`);
      console.log(`  Raw Log: ${result.rawLog}`);
    }
  } catch (error) {
    console.log("❌ ERROR! Transaction failed with exception:");
    console.error(error);
  }

  console.log();
  console.log("=".repeat(60));
}

main().catch((error: any) => {
  console.error("\n❌ Fatal error:", error.message || error);
  
  // Check if it's a connection error
  if (error.cause?.code === "ECONNREFUSED" || error.message?.includes("fetch failed")) {
    console.error("\n⚠️  Connection Error: Cannot connect to the blockchain.");
    console.error(`   Make sure the Verana blockchain is running at ${config.rpcEndpoint}`);
    console.error("   Start it with: ./scripts/setup_primary_validator.sh");
  }
  
  process.exit(1);
});
