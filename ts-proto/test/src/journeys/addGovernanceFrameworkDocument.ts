/**
 * Journey: Add Governance Framework Document
 *
 * This script demonstrates how to add a governance framework document to a
 * Trust Registry using the TypeScript client and the generated protobuf types.
 *
 * Usage:
 *   TR_ID=1 npm run test:add-gf-doc
 *   # Or let it create a TR first, then add a document
 *   npm run test:add-gf-doc
 *
 * Or with environment variables:
 *   export MNEMONIC="your mnemonic here"
 *   export VERANA_RPC_ENDPOINT="http://localhost:26657"
 *   export TR_ID=1
 *   npm run test:add-gf-doc
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
import { MsgCreateTrustRegistry, MsgAddGovernanceFrameworkDocument } from "../../../src/codec/verana/tr/v1/tx";

// Test mnemonic - Uses cooluser seed phrase (same as test harness)
const TEST_MNEMONIC =
  process.env.MNEMONIC ||
  "pink glory help gown abstract eight nice crazy forward ketchup skill cheese";

async function main() {
  console.log("=".repeat(60));
  console.log("Journey: Add Governance Framework Document (TypeScript Client)");
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
    // Try to reuse active TR from journey results first (for sequential runs)
    const { getActiveTR } = await import("../helpers/journeyResults");
    const trResult = getActiveTR();
    
    if (trResult) {
      trId = trResult.trustRegistryId;
      console.log(`Step 4: Reusing active Trust Registry from journey results: ${trId}`);
    }
    
    if (!trId) {
      console.log("Step 4: Creating a Trust Registry first (no TR_ID provided and no journey results found)...");
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
      "Creating Trust Registry for add document test"
    );

    const createResult = await client.signAndBroadcast(
      account.address,
      [createMsg],
      createFee,
      "Creating Trust Registry for add document test"
    );

    if (createResult.code !== 0) {
      console.log("  ❌ Failed to create Trust Registry for add document test");
      console.log(`  Error: ${createResult.rawLog}`);
      process.exit(1);
    }

    // Extract TR ID from events
    const events = createResult.events || [];
    for (const event of events) {
      if (event.type === "create_trust_registry" || event.type === "verana.tr.v1.EventCreateTrustRegistry") {
        for (const attr of event.attributes) {
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

      if (!trId || isNaN(trId)) {
        console.log("  ❌ Could not extract TR ID from events");
        process.exit(1);
      }
      
      // Save new TR as active TR so subsequent tests can reuse it
      const { saveActiveTR } = await import("../helpers/journeyResults");
      saveActiveTR(trId, did);
    }
  }

  if (!trId) {
    console.log("  ❌ Trust Registry ID is required");
    process.exit(1);
  }

  console.log();

  // Step 5: Add Governance Framework Document
  console.log("Step 5: Adding Governance Framework Document transaction...");
  const docLanguage = "en";
  const docUrl = "https://example.com/governance-framework-v2.pdf";
  const docDigestSri = "sha384-NewDocumentHash123456789012345678901234567890123456789012345678901234567890";
  // Note: When a TR is created, version 1 already exists. New documents must be for version 2 or higher.
  const version = 2;
  const msg = {
    typeUrl: typeUrls.MsgAddGovernanceFrameworkDocument,
    value: MsgAddGovernanceFrameworkDocument.fromPartial({
      creator: account.address,
      id: trId,
      docLanguage: docLanguage,
      docUrl: docUrl,
      docDigestSri: docDigestSri,
      version: version,
    }),
  };
  console.log("  Message details:");
  console.log(`    - Creator: ${account.address}`);
  console.log(`    - Trust Registry ID: ${trId}`);
  console.log(`    - Document Language: ${docLanguage}`);
  console.log(`    - Document URL: ${docUrl}`);
  console.log(`    - Document Digest SRI: ${docDigestSri}`);
  console.log(`    - Version: ${version}`);
  console.log();

  // Step 6: Sign and broadcast
  console.log("Step 6: Signing and broadcasting transaction...");
  try {
    const fee = await calculateFeeWithSimulation(
      client,
      account.address,
      [msg],
      "Adding Governance Framework Document via TypeScript client"
    );
    console.log(`  Calculated gas: ${fee.gas}, fee: ${fee.amount[0].amount}${fee.amount[0].denom}`);
    
    const result = await client.signAndBroadcast(
      account.address,
      [msg],
      fee,
      "Adding Governance Framework Document via TypeScript client"
    );

    console.log();
    if (result.code === 0) {
      console.log("✅ SUCCESS! Governance Framework Document added successfully!");
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

