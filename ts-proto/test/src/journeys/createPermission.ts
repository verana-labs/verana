/**
 * Journey: Create Permission
 *
 * Step 2 of 2 for Create Permission.
 * Loads prerequisites from Step 1, creates the Permission transaction.
 *
 * Based on Journey 18 from the test harness, which shows that:
 * 1. A root (ecosystem) permission MUST be created first for the schema
 * 2. Then regular permissions can be created for that schema
 *
 * Usage:
 *   # Using prerequisites from Step 1 (recommended)
 *   npm run test:create-perm
 *
 *   # Or provide specific values
 *   SCHEMA_ID=1 DID="did:verana:example" npm run test:create-perm
 *
 * Or with environment variables:
 *   export MNEMONIC="your mnemonic here"
 *   export VERANA_RPC_ENDPOINT="http://localhost:26657"
 *   export SCHEMA_ID=1
 *   export DID="did:verana:example"
 *   npm run test:create-perm
 */

import {
  createAccountFromMnemonic,
  createSigningClient,
  getAccountInfo,
  calculateFeeWithSimulation,
  signAndBroadcastWithRetry,
  config,
  waitForSequencePropagation,
} from "../helpers/client";
import { typeUrls } from "../helpers/registry";
import { MsgCreatePermission } from "../../../src/codec/verana/perm/v1/tx";
import { PermissionType } from "../../../src/codec/verana/perm/v1/types";
import { getActiveTRAndSchema, getRootPermissionId, savePermissionId, loadJourneyResult } from "../helpers/journeyResults";

// Master mnemonic - same for all accounts
const MASTER_MNEMONIC =
  process.env.MNEMONIC ||
  "pink glory help gown abstract eight nice crazy forward ketchup skill cheese";

// Account index for Journey 14 (Create Permission)
const ACCOUNT_INDEX = 14;

async function main() {
  console.log("=".repeat(60));
  console.log("Journey: Create Permission (Step 2/2)");
  console.log("=".repeat(60));
  console.log();

  // Load prerequisites from Step 1
  console.log("Loading prerequisites from Step 1...");
  const prereqs = loadJourneyResult("create-perm-prereqs");

  // Create account wallet (needed for both prerequisites and connection)
  const account14Wallet = await createAccountFromMnemonic(MASTER_MNEMONIC, ACCOUNT_INDEX);
  const account14 = await getAccountInfo(account14Wallet);
  const account14Address = account14.address;

  let schemaId: number | undefined;
  let did: string;
  let rootPermId: number | null = null;

  if (prereqs?.accountAddress) {
    // Use address from prerequisites (should match, but use loaded one for consistency)
    schemaId = prereqs.schemaId ? parseInt(prereqs.schemaId, 10) : undefined;
    did = prereqs.did || "";
    rootPermId = prereqs.rootPermissionId ? parseInt(prereqs.rootPermissionId, 10) : null;
    console.log(`  ✓ Loaded prerequisites:`);
    console.log(`    - Account address: ${account14Address}`);
    if (schemaId) {
      console.log(`    - Schema ID: ${schemaId}`);
    }
    if (did) {
      console.log(`    - DID: ${did}`);
    }
    if (rootPermId) {
      console.log(`    - Root Permission ID: ${rootPermId}`);
    }
  } else {
    console.log("  ⚠️  Prerequisites not found. Run Step 1 first:");
    console.log("     npm run test:setup-create-perm-prereqs");
    console.log("  Or provide SCHEMA_ID and DID via environment variables.");
    
    // Fallback to environment variables or active TR/CS
    if (process.env.SCHEMA_ID && process.env.DID) {
      schemaId = parseInt(process.env.SCHEMA_ID, 10);
      did = process.env.DID;
      if (isNaN(schemaId)) {
        console.log("  ❌ Invalid SCHEMA_ID provided");
        process.exit(1);
      }
      console.log(`  Using provided Schema ID: ${schemaId} and DID: ${did}`);
    } else {
      // Try to load from active TR/CS
      const trAndSchema = getActiveTRAndSchema();
      if (trAndSchema) {
        schemaId = trAndSchema.schemaId;
        did = trAndSchema.did;
        console.log(`  Using active TR/CS from journey results:`);
        console.log(`    - Schema ID: ${schemaId}`);
        console.log(`    - DID: ${did}`);
      } else {
        console.log("  ❌ Schema ID and DID are required. Run Step 1 first or provide via environment variables.");
        process.exit(1);
      }
    }

    console.log(`  Account address: ${account14Address}`);
  }

  console.log();

  // Step 1: Connect account_14 to blockchain
  console.log("Step 1: Connecting account_14 to Verana blockchain...");
  console.log(`  RPC Endpoint: ${config.rpcEndpoint}`);
  const client = await createSigningClient(account14Wallet);
  console.log("  ✓ Connected successfully");

  // Verify balance
  const balance = await client.getBalance(account14Address, config.denom);
  console.log(`  Balance: ${balance.amount} ${balance.denom}`);
  if (BigInt(balance.amount) < BigInt(1000000)) {
    console.log("  ⚠️  Warning: Low balance. Run Step 1 to fund the account.");
    process.exit(1);
  }
  console.log();

  // Step 2: Validate prerequisites
  if (!schemaId || !did) {
    console.log("  ❌ Schema ID and DID are required");
    process.exit(1);
  }

  // Step 3: Load Root Permission ID (REQUIRED - ecosystem permission must exist)
  console.log("Step 2: Loading Root Permission ID...");
  if (!rootPermId) {
    rootPermId = getRootPermissionId();
  }
  if (!rootPermId) {
    console.log("  ❌ Root Permission not found. Journey 13 (Create Root Permission) must be run first.");
    process.exit(1);
  }
  console.log(`  ✓ Root Permission ID: ${rootPermId}`);
  console.log();

  // Step 4: Create Permission message
  console.log("Step 3: Creating Permission transaction...");
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
      creator: account14Address,
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
  console.log(`    - Creator: ${account14Address} (account_${ACCOUNT_INDEX})`);
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

  // Step 5: Sign and broadcast
  console.log("Step 4: Signing and broadcasting transaction...");
  try {
    const fee = await calculateFeeWithSimulation(
      client,
      account14Address,
      [msg],
      "Creating Permission via TypeScript client"
    );
    console.log(`  Calculated gas: ${fee.gas}, fee: ${fee.amount[0].amount}${fee.amount[0].denom}`);

    // Use retry logic for consistency (matches frontend pattern)
    const result = await signAndBroadcastWithRetry(
      client,
      account14Address,
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

