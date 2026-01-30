/**
 * Journey: Create Root Permission
 *
 * Step 2 of 2 for Create Root Permission.
 * Loads prerequisites from Step 1, creates the Root Permission transaction.
 *
 * Usage:
 *   # Using prerequisites from Step 1 (recommended)
 *   npm run test:create-root-perm
 *
 *   # Or provide specific values
 *   SCHEMA_ID=1 DID="did:verana:example" npm run test:create-root-perm
 *
 * Or with environment variables:
 *   export MNEMONIC="your mnemonic here"
 *   export VERANA_RPC_ENDPOINT="http://localhost:26657"
 *   export SCHEMA_ID=1
 *   export DID="did:verana:example"
 *   npm run test:create-root-perm
 */

import {
  createAccountFromMnemonic,
  createSigningClient,
  getAccountInfo,
  calculateFeeWithSimulation,
  signAndBroadcastWithRetry,
  config,
} from "../helpers/client";
import { typeUrls } from "../helpers/registry";
import { MsgCreateRootPermission } from "../../../src/codec/verana/perm/v1/tx";
import { getActiveTRAndSchema, saveRootPermissionId, loadJourneyResult } from "../helpers/journeyResults";

// Master mnemonic - same for all accounts
const MASTER_MNEMONIC =
  process.env.MNEMONIC ||
  "pink glory help gown abstract eight nice crazy forward ketchup skill cheese";

// Account index for Journey 13 (Create Root Permission)
const ACCOUNT_INDEX = 13;


async function main() {
  console.log("=".repeat(60));
  console.log("Journey: Create Root Permission (Step 2/2)");
  console.log("=".repeat(60));
  console.log();

  // Load prerequisites from Step 1
  console.log("Loading prerequisites from Step 1...");
  const prereqs = loadJourneyResult("create-root-perm-prereqs");

  // Create account wallet (needed for both prerequisites and connection)
  const account13Wallet = await createAccountFromMnemonic(MASTER_MNEMONIC, ACCOUNT_INDEX);
  const account13 = await getAccountInfo(account13Wallet);
  const account13Address = account13.address;

  let schemaId: number | undefined;
  let did: string | undefined;

  if (prereqs?.accountAddress) {
    // Use address from prerequisites (should match, but use loaded one for consistency)
    schemaId = prereqs.schemaId ? parseInt(prereqs.schemaId, 10) : undefined;
    did = prereqs.did || undefined;
    console.log(`  ✓ Loaded prerequisites:`);
    console.log(`    - Account address: ${account13Address}`);
    if (schemaId) {
      console.log(`    - Schema ID: ${schemaId}`);
    }
    if (did) {
      console.log(`    - DID: ${did}`);
    }
  } else {
    console.log("  ⚠️  Prerequisites not found. Run Step 1 first:");
    console.log("     npm run test:setup-create-root-perm-prereqs");
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

    console.log(`  Account address: ${account13Address}`);
  }

  console.log();

  // Step 1: Connect account_13 to blockchain
  console.log("Step 1: Connecting account_13 to Verana blockchain...");
  console.log(`  RPC Endpoint: ${config.rpcEndpoint}`);
  const client = await createSigningClient(account13Wallet);
  console.log("  ✓ Connected successfully");

  // Verify balance
  const balance = await client.getBalance(account13Address, config.denom);
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

  // Step 3: Create Root Permission message
  console.log("Step 2: Creating Root Permission transaction...");
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

  // Step 4: Sign and broadcast
  console.log("Step 3: Signing and broadcasting transaction...");
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

