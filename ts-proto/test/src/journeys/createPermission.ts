/**
 * Journey: Create Permission
 *
 * This script demonstrates how to create a Permission using the
 * TypeScript client and the generated protobuf types.
 *
 * Based on Journey 18 from the test harness, which shows that:
 * 1. A root (ecosystem) permission MUST be created first for the schema
 * 2. Then regular permissions can be created for that schema
 *
 * Usage:
 *   SCHEMA_ID=1 DID="did:verana:example" npm run test:create-perm
 *   # Or let it create a schema first, then create permission
 *   npm run test:create-perm
 *
 * Or with environment variables:
 *   export MNEMONIC="your mnemonic here"
 *   export VERANA_RPC_ENDPOINT="http://localhost:26657"
 *   export SCHEMA_ID=1
 *   export DID="did:verana:example"
 *   npm run test:create-perm
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
} from "../helpers/client";
import { typeUrls } from "../helpers/registry";
import { MsgCreatePermission } from "../../../src/codec/verana/perm/v1/tx";
import { PermissionType } from "../../../src/codec/verana/perm/v1/types";
import { getActiveTRAndSchema, getRootPermissionId, savePermissionId } from "../helpers/journeyResults";
import { createSchemaForTest } from "../helpers/permissionHelpers";

// Master mnemonic - same for all accounts
const MASTER_MNEMONIC =
  process.env.MNEMONIC ||
  "pink glory help gown abstract eight nice crazy forward ketchup skill cheese";

// Account index for Journey 14 (Create Permission)
const ACCOUNT_INDEX = 14;

async function main() {
  console.log("=".repeat(60));
  console.log("Journey: Create Permission (TypeScript Client)");
  console.log("=".repeat(60));
  console.log();

  // Step 1: Setup cooluser account (for funding)
  console.log("Step 1: Setting up cooluser account (for funding)...");
  const cooluserWallet = await createWallet(MASTER_MNEMONIC);
  const cooluserAccount = await getAccountInfo(cooluserWallet);
  const cooluserClient = await createSigningClient(cooluserWallet);
  console.log(`  ✓ Cooluser address: ${cooluserAccount.address}`);
  console.log();

  // Step 2: Create account_14 from mnemonic with derivation path 14
  console.log(`Step 2: Creating account_${ACCOUNT_INDEX} from mnemonic (derivation path ${ACCOUNT_INDEX})...`);
  const account14Wallet = await createAccountFromMnemonic(MASTER_MNEMONIC, ACCOUNT_INDEX);
  const account14 = await getAccountInfo(account14Wallet);
  console.log(`  ✓ Account_${ACCOUNT_INDEX} address: ${account14.address}`);
  console.log();

  // Step 3: Fund account_14 from cooluser
  console.log("Step 3: Funding account_14 from cooluser...");
  const fundingAmount = "1000000000uvna"; // 1 VNA
  try {
    const fundResult = await fundAccount(
      MASTER_MNEMONIC,
      cooluserAccount.address,
      account14.address,
      fundingAmount
    );
    if (fundResult.code === 0) {
      console.log(`  ✓ Funded account_14 with ${fundingAmount}`);
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

  // Step 5: Connect account_14 to blockchain
  console.log("Step 5: Connecting account_14 to Verana blockchain...");
  console.log(`  RPC Endpoint: ${config.rpcEndpoint}`);
  const client = await createSigningClient(account14Wallet);
  console.log("  ✓ Connected successfully");
  
  // Verify balance
  const balance = await client.getBalance(account14.address, config.denom);
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
    schemaId = parseInt(process.env.SCHEMA_ID, 10);
    did = process.env.DID;
    if (isNaN(schemaId)) {
      console.log("  ❌ Invalid SCHEMA_ID provided");
      process.exit(1);
    }
    console.log(`Step 6: Using provided Schema ID: ${schemaId} and DID: ${did}`);
  } else {
    // Try to load from active TR/CS
    const trAndSchema = getActiveTRAndSchema();
    
    if (trAndSchema) {
      schemaId = trAndSchema.schemaId;
      did = trAndSchema.did;
      console.log(`Step 6: Using active TR/CS from journey results:`);
      console.log(`  - Trust Registry ID: ${trAndSchema.trustRegistryId}`);
      console.log(`  - Schema ID: ${schemaId}`);
      console.log(`  - DID: ${did}`);
    } else {
      console.log("Step 6: No active TR/CS found, creating new Trust Registry and Schema...");
      const schemaResult = await createSchemaForTest(cooluserClient, cooluserAccount.address);
      schemaId = schemaResult.schemaId;
      did = schemaResult.did;
      console.log(`  ✓ Created Credential Schema with ID: ${schemaId}`);
    }
  }

  if (!schemaId || !did) {
    console.log("  ❌ Schema ID and DID are required");
    process.exit(1);
  }

  console.log();

  // Step 7: Load Root Permission ID from Journey 13 (REQUIRED - ecosystem permission must exist)
  console.log("Step 7: Loading Root Permission ID from Journey 13...");
  const rootPermId = getRootPermissionId();
  if (!rootPermId) {
    console.log("  ❌ Root Permission not found. Journey 13 (Create Root Permission) must be run first.");
    process.exit(1);
  }
  console.log(`  ✓ Root Permission ID: ${rootPermId}`);
  console.log();

  // Step 8: Create Permission message
  console.log("Step 8: Creating Permission transaction...");
  // Set effectiveFrom to 10 seconds in the future as required by blockchain (matches test harness)
  const effectiveFrom = new Date(Date.now() + 10000);
  const effectiveUntil = new Date(effectiveFrom.getTime() + 360 * 24 * 60 * 60 * 1000); // 360 days from effectiveFrom
  const verificationFees = 1000;
  const validationFees = 1000;
  const country = "US";
  const permissionType = PermissionType.ISSUER; // Can be ISSUER, VERIFIER, etc.

  const msg = {
    typeUrl: typeUrls.MsgCreatePermission,
    value: MsgCreatePermission.fromPartial({
      creator: account14.address,
      schemaId: schemaId,
      type: permissionType,
      did: did,
      country: country,
      effectiveFrom: effectiveFrom,
      effectiveUntil: effectiveUntil,
      verificationFees: verificationFees,
      validationFees: validationFees,
      validatorPermId: rootPermId, // Use root permission from Journey 13
    }),
  };
  console.log("  Message details:");
  console.log(`    - Creator: ${account14.address} (account_${ACCOUNT_INDEX})`);
  console.log(`    - Schema ID: ${schemaId}`);
  console.log(`    - Permission Type: ${PermissionType[permissionType]} (${permissionType})`);
  console.log(`    - DID: ${did}`);
  console.log(`    - Country: ${country}`);
  console.log(`    - Validator Permission ID: ${rootPermId}`);
  console.log(`    - Effective From: ${effectiveFrom.toISOString()}`);
  console.log(`    - Effective Until: ${effectiveUntil.toISOString()}`);
  console.log(`    - Verification Fees: ${verificationFees}`);
  console.log(`    - Validation Fees: ${validationFees}`);
  console.log();

  // Step 9: Sign and broadcast
  console.log("Step 9: Signing and broadcasting transaction...");
  try {
    const fee = await calculateFeeWithSimulation(
      client,
      account14.address,
      [msg],
      "Creating Permission via TypeScript client"
    );
    console.log(`  Calculated gas: ${fee.gas}, fee: ${fee.amount[0].amount}${fee.amount[0].denom}`);
    
    // Use retry logic for consistency (matches frontend pattern)
    const result = await signAndBroadcastWithRetry(
      client,
      account14.address,
      [msg],
      fee,
      "Creating Permission via TypeScript client"
    );

    console.log();
    if (result.code === 0) {
      console.log("✅ SUCCESS! Permission created successfully!");
      console.log("=".repeat(60));
      console.log(`  Transaction Hash: ${result.transactionHash}`);
      console.log(`  Block Height: ${result.height}`);
      console.log(`  Gas Used: ${result.gasUsed}/${result.gasWanted}`);
      
      // Extract permission ID from events and save to journey results
      let permissionId: number | null = null;
      const events = result.events || [];
      for (const event of events) {
        if (event.type === "create_permission" || event.type === "verana.perm.v1.EventCreatePermission") {
          for (const attr of event.attributes) {
            if (attr.key === "permission_id" || attr.key === "id") {
              permissionId = parseInt(attr.value, 10);
              console.log(`  Permission ID: ${permissionId}`);
            }
          }
        }
      }
      
      // Save permission ID for reuse in Journeys 15-16
      if (permissionId !== null) {
        savePermissionId(permissionId, "create-permission");
        console.log(`  💾 Saved permission ID to journey results for reuse`);
      } else {
        console.log(`  ⚠️  Warning: Could not extract permission ID from events`);
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
    
    if (error.message?.includes("ecosystem permission not found")) {
      console.error("\n⚠️  Prerequisite Error: Ecosystem permission (root permission) not found.");
      console.error(`   This means the root permission for schema ${schemaId} was not created or committed properly.`);
      console.error("   The root permission must be created and committed before creating regular permissions.");
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

