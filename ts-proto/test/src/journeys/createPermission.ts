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
  createSigningClient,
  getAccountInfo,
  calculateFeeWithSimulation,
  signAndBroadcastWithRetry,
  config,
} from "../helpers/client";
import { typeUrls } from "../helpers/registry";
import { MsgCreatePermission } from "../../../src/codec/verana/perm/v1/tx";
import { PermissionType } from "../../../src/codec/verana/perm/v1/types";
import { createSchemaForTest, createRootPermissionForTest } from "../helpers/permissionHelpers";

// Test mnemonic - Uses cooluser seed phrase (same as test harness)
const TEST_MNEMONIC =
  process.env.MNEMONIC ||
  "pink glory help gown abstract eight nice crazy forward ketchup skill cheese";

async function main() {
  console.log("=".repeat(60));
  console.log("Journey: Create Permission (TypeScript Client)");
  console.log("=".repeat(60));
  console.log();

  // Step 1: Setup wallet (using Amino Sign to match frontend)
  console.log("Step 1: Setting up wallet (Amino Sign mode)...");
  const wallet = await createWallet(TEST_MNEMONIC);
  const account = await getAccountInfo(wallet);
  console.log(`  ✓ Wallet address: ${account.address}`);
  console.log(`  ✓ Using Amino Sign (matches frontend behavior)`);
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

  // Step 4: Get or create Schema ID and DID
  let schemaId: number | undefined;
  let did: string;
  
  if (process.env.SCHEMA_ID && process.env.DID) {
    schemaId = parseInt(process.env.SCHEMA_ID, 10);
    did = process.env.DID;
    if (isNaN(schemaId)) {
      console.log("  ❌ Invalid SCHEMA_ID provided");
      process.exit(1);
    }
    console.log(`Step 4: Using provided Schema ID: ${schemaId} and DID: ${did}`);
  } else {
    console.log("Step 4: Creating Trust Registry and Schema first...");
    const schemaResult = await createSchemaForTest(client, account.address);
    schemaId = schemaResult.schemaId;
    did = schemaResult.did;
    console.log(`  ✓ Created Credential Schema with ID: ${schemaId}`);
    // Refresh sequence after schema creation to ensure cache is updated
    await new Promise((resolve) => setTimeout(resolve, 2000));
    await client.getSequence(account.address);
    await new Promise((resolve) => setTimeout(resolve, 500));
    await client.getSequence(account.address);
  }

  if (!schemaId || !did) {
    console.log("  ❌ Schema ID and DID are required");
    process.exit(1);
  }

  console.log();

  // Step 5: Create Root Permission first (REQUIRED - ecosystem permission must exist before regular permissions)
  // This is the key prerequisite from the test harness (Journey 18)
  console.log("Step 5: Creating Root Permission first (required prerequisite)...");
  try {
    const rootPermId = await createRootPermissionForTest(client, account.address, schemaId, did);
    console.log(`  ✓ Root Permission (ecosystem permission) created with ID: ${rootPermId}`);
    // Refresh sequence after root permission creation to ensure cache is updated
    await new Promise((resolve) => setTimeout(resolve, 2000));
    await client.getSequence(account.address);
    await new Promise((resolve) => setTimeout(resolve, 500));
    await client.getSequence(account.address);
  } catch (error: any) {
    console.log("  ❌ Failed to create Root Permission (prerequisite)");
    console.error(`  Error: ${error.message}`);
    process.exit(1);
  }
  console.log();

  // Step 6: Create Permission message
  console.log("Step 6: Creating Permission transaction...");
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
      creator: account.address,
      schemaId: schemaId,
      type: permissionType,
      did: did,
      country: country,
      effectiveFrom: effectiveFrom,
      effectiveUntil: effectiveUntil,
      verificationFees: verificationFees,
      validationFees: validationFees,
    }),
  };
  console.log("  Message details:");
  console.log(`    - Creator: ${account.address}`);
  console.log(`    - Schema ID: ${schemaId}`);
  console.log(`    - Permission Type: ${PermissionType[permissionType]} (${permissionType})`);
  console.log(`    - DID: ${did}`);
  console.log(`    - Country: ${country}`);
  console.log(`    - Effective From: ${effectiveFrom.toISOString()}`);
  console.log(`    - Effective Until: ${effectiveUntil.toISOString()}`);
  console.log(`    - Verification Fees: ${verificationFees}`);
  console.log(`    - Validation Fees: ${validationFees}`);
  console.log();

  // Step 7: Sign and broadcast
  console.log("Step 7: Signing and broadcasting transaction...");
  try {
    // Refresh sequence one more time right before the transaction to ensure it's up to date
    await new Promise((resolve) => setTimeout(resolve, 1000));
    await client.getSequence(account.address);
    await new Promise((resolve) => setTimeout(resolve, 500));
    await client.getSequence(account.address);
    
    const fee = await calculateFeeWithSimulation(
      client,
      account.address,
      [msg],
      "Creating Permission via TypeScript client"
    );
    console.log(`  Calculated gas: ${fee.gas}, fee: ${fee.amount[0].amount}${fee.amount[0].denom}`);
    
    // Use retry logic for consistency (matches frontend pattern)
    const result = await signAndBroadcastWithRetry(
      client,
      account.address,
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
      
      // Try to extract permission ID from events
      const events = result.events || [];
      for (const event of events) {
        if (event.type === "create_permission" || event.type === "verana.perm.v1.EventCreatePermission") {
          for (const attr of event.attributes) {
            if (attr.key === "permission_id" || attr.key === "id") {
              console.log(`  Permission ID: ${attr.value}`);
            }
          }
        }
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

