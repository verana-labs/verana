/**
 * Journey: Create Root Permission
 *
 * This script demonstrates how to create a Root Permission using the
 * TypeScript client and the generated protobuf types.
 *
 * Usage:
 *   # Reuse existing TR/CS from journey results (recommended)
 *   npm run test:create-root-perm
 *   
 *   # Or provide specific Schema ID and DID
 *   SCHEMA_ID=1 DID="did:verana:example" npm run test:create-root-perm
 *   
 *   # First create TR and CS if they don't exist:
 *   npm run test:create-tr
 *   npm run test:create-cs
 *
 * Or with environment variables:
 *   export MNEMONIC="your mnemonic here"
 *   export VERANA_RPC_ENDPOINT="http://localhost:26657"
 *   export SCHEMA_ID=1
 *   export DID="did:verana:example"
 *   npm run test:create-root-perm
 */

import {
  createWallet,
  createAccountFromMnemonic,
  createSigningClient,
  getAccountInfo,
  calculateFeeWithSimulation,
  signAndBroadcastWithRetry,
  fundAccount,
  config,
  waitForSequencePropagation,
} from "../helpers/client";
import { typeUrls } from "../helpers/registry";
import { MsgCreateRootPermission } from "../../../src/codec/verana/perm/v1/tx";
import { getActiveTRAndSchema, saveRootPermissionId } from "../helpers/journeyResults";
import { createSchemaForTest } from "../helpers/permissionHelpers";

// Master mnemonic - same for all accounts
const MASTER_MNEMONIC =
  process.env.MNEMONIC ||
  "pink glory help gown abstract eight nice crazy forward ketchup skill cheese";

// Account index for Journey 13 (Create Root Permission)
const ACCOUNT_INDEX = 13;


async function main() {
  console.log("=".repeat(60));
  console.log("Journey: Create Root Permission (TypeScript Client)");
  console.log("=".repeat(60));
  console.log();

  // Step 1: Setup cooluser account (for funding)
  console.log("Step 1: Setting up cooluser account (for funding)...");
  const cooluserWallet = await createWallet(MASTER_MNEMONIC);
  const cooluserAccount = await getAccountInfo(cooluserWallet);
  const cooluserClient = await createSigningClient(cooluserWallet);
  console.log(`  ✓ Cooluser address: ${cooluserAccount.address}`);
  console.log();

  // Step 2: Create account_13 from mnemonic with derivation path 13
  console.log(`Step 2: Creating account_${ACCOUNT_INDEX} from mnemonic (derivation path ${ACCOUNT_INDEX})...`);
  const account13Wallet = await createAccountFromMnemonic(MASTER_MNEMONIC, ACCOUNT_INDEX);
  const account13 = await getAccountInfo(account13Wallet);
  console.log(`  ✓ Account_${ACCOUNT_INDEX} address: ${account13.address}`);
  console.log();

  // Step 3: Fund account_13 from cooluser
  console.log("Step 3: Funding account_13 from cooluser...");
  const fundingAmount = "1000000000uvna"; // 1 VNA
  try {
    const fundResult = await fundAccount(
      MASTER_MNEMONIC,
      cooluserAccount.address,
      account13.address,
      fundingAmount
    );
    if (fundResult.code === 0) {
      console.log(`  ✓ Funded account_13 with ${fundingAmount}`);
      console.log(`  Transaction Hash: ${fundResult.transactionHash}`);
    } else {
      console.log(`  ❌ Funding failed: ${fundResult.rawLog}`);
      process.exit(1);
    }
  } catch (error: any) {
    console.log(`  ❌ Funding failed: ${error.message}`);
    process.exit(1);
  }
  console.log();

  // Step 4: Wait for balance to be reflected (10 seconds)
  console.log("Step 4: Waiting 10 seconds for balance to be reflected...");
  await new Promise((resolve) => setTimeout(resolve, 10000));
  console.log("  ✓ Wait complete");
  console.log();

  // Step 5: Connect account_13 to blockchain
  console.log("Step 5: Connecting account_13 to Verana blockchain...");
  console.log(`  RPC Endpoint: ${config.rpcEndpoint}`);
  const client = await createSigningClient(account13Wallet);
  console.log("  ✓ Connected successfully");
  
  // Verify balance
  const balance = await client.getBalance(account13.address, config.denom);
  console.log(`  Balance: ${balance.amount} ${balance.denom}`);
  if (BigInt(balance.amount) < BigInt(1000000)) {
    console.log("  ⚠️  Warning: Low balance. Funding may not have completed.");
    process.exit(1);
  }
  console.log();

  // Step 6: Get Schema ID and DID from journey results or create new ones
  let schemaId: number | undefined;
  let did: string;
  
  if (process.env.SCHEMA_ID && process.env.DID) {
    // Use provided values
    schemaId = parseInt(process.env.SCHEMA_ID, 10);
    did = process.env.DID;
    if (isNaN(schemaId)) {
      console.log("  ❌ Invalid SCHEMA_ID provided");
      process.exit(1);
    }
    console.log(`Step 6: Using provided Schema ID: ${schemaId} and DID: ${did}`);
  } else {
    // Try to load active TR/CS from journey results (reuse existing TR/CS from earlier journeys)
    console.log("Step 6: Loading active Trust Registry and Schema from journey results...");
    const trAndSchema = getActiveTRAndSchema();
    
    if (trAndSchema) {
      // Try to verify the schema exists on-chain by querying LCD endpoint
      try {
        const lcdEndpoint = process.env.VERANA_LCD_ENDPOINT || "http://localhost:1317";
        const response = await fetch(`${lcdEndpoint}/verana/cs/v1/credential_schema/${trAndSchema.schemaId}`);
        
        if (response.ok) {
          // Schema exists, reuse it
          schemaId = trAndSchema.schemaId;
          did = trAndSchema.did;
          console.log(`  ✓ Reusing TR/CS from journey results:`);
          console.log(`    - Trust Registry ID: ${trAndSchema.trustRegistryId}`);
          console.log(`    - Schema ID: ${schemaId}`);
          console.log(`    - DID: ${did}`);
        } else {
          throw new Error("Schema not found");
        }
      } catch (error) {
        // Schema doesn't exist on-chain, create a new one with account_13
        // This ensures account_13 is the TR controller and can create root permission
        console.log("  ⚠️  Schema from journey results doesn't exist on-chain, creating new Trust Registry and Schema with account_13...");
        const newSchema = await createSchemaForTest(client, account13.address);
        schemaId = newSchema.schemaId;
        did = newSchema.did;
        // Wait for sequence to fully propagate after creating TR and CS (poll with 60s timeout)
        await waitForSequencePropagation(client, account13.address);
        console.log(`  ✓ Created new Schema ID: ${schemaId}, DID: ${did}`);
      }
    } else {
      // No journey results found - create new TR/CS using account_13
      // This ensures account_13 is the TR controller and can create root permission
      console.log("  No journey results found, creating new Trust Registry and Schema with account_13...");
      const newSchema = await createSchemaForTest(client, account13.address);
      schemaId = newSchema.schemaId;
      did = newSchema.did;
      // Wait for sequence to fully propagate after creating TR and CS (poll with 60s timeout)
      await waitForSequencePropagation(client, account13.address);
      console.log(`  ✓ Created new Schema ID: ${schemaId}, DID: ${did}`);
    }
  }

  if (!schemaId || !did) {
    console.log("  ❌ Schema ID and DID are required");
    process.exit(1);
  }

  console.log();

  // Step 7: Create Root Permission message
  console.log("Step 7: Creating Root Permission transaction...");
  // Set effectiveFrom to 10 seconds in the future as required by blockchain (matches test harness)
  const effectiveFrom = new Date(Date.now() + 10000);
  const effectiveUntil = new Date(effectiveFrom.getTime() + 360 * 24 * 60 * 60 * 1000); // 360 days from effectiveFrom
  const validationFees = 5;
  const verificationFees = 5;
  const issuanceFees = 5;
  const country = "US";

  const msg = {
    typeUrl: typeUrls.MsgCreateRootPermission,
    value: MsgCreateRootPermission.fromPartial({
      creator: account13.address,
      schemaId: schemaId,
      did: did,
      country: country,
      effectiveFrom: effectiveFrom,
      effectiveUntil: effectiveUntil,
      validationFees: validationFees,
      verificationFees: verificationFees,
      issuanceFees: issuanceFees,
    }),
  };
  console.log("  Message details:");
  console.log(`    - Creator: ${account13.address} (account_${ACCOUNT_INDEX})`);
  console.log(`    - Schema ID: ${schemaId}`);
  console.log(`    - DID: ${did}`);
  console.log(`    - Country: ${country}`);
  console.log(`    - Effective From: ${effectiveFrom.toISOString()}`);
  console.log(`    - Effective Until: ${effectiveUntil.toISOString()}`);
  console.log(`    - Validation Fees: ${validationFees}`);
  console.log(`    - Verification Fees: ${verificationFees}`);
  console.log(`    - Issuance Fees: ${issuanceFees}`);
  console.log();

  // Step 8: Sign and broadcast
  console.log("Step 8: Signing and broadcasting transaction...");
  try {
    const fee = await calculateFeeWithSimulation(
      client,
      account13.address,
      [msg],
      "Creating Root Permission via TypeScript client"
    );
    console.log(`  Calculated gas: ${fee.gas}, fee: ${fee.amount[0].amount}${fee.amount[0].denom}`);
    
    // Use retry logic for consistency (matches frontend pattern)
    const result = await signAndBroadcastWithRetry(
      client,
      account13.address,
      [msg],
      fee,
      "Creating Root Permission via TypeScript client"
    );

    console.log();
    if (result.code === 0) {
      console.log("✅ SUCCESS! Root Permission created successfully!");
      console.log("=".repeat(60));
      console.log(`  Transaction Hash: ${result.transactionHash}`);
      console.log(`  Block Height: ${result.height}`);
      console.log(`  Gas Used: ${result.gasUsed}/${result.gasWanted}`);
      
      // Extract permission ID from events and save to journey results
      let rootPermissionId: number | null = null;
      const events = result.events || [];
      for (const event of events) {
        if (event.type === "create_root_permission" || event.type === "verana.perm.v1.EventCreateRootPermission") {
          for (const attr of event.attributes) {
            if (attr.key === "root_permission_id" || attr.key === "permission_id" || attr.key === "id") {
              rootPermissionId = parseInt(attr.value, 10);
              console.log(`  Root Permission ID: ${rootPermissionId}`);
            }
          }
        }
      }
      
      // Save root permission ID for reuse in other journeys
      if (rootPermissionId !== null) {
        saveRootPermissionId(rootPermissionId);
        console.log(`  💾 Saved root permission ID to journey results for reuse`);
      } else {
        console.log(`  ⚠️  Warning: Could not extract root permission ID from events`);
      }
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

