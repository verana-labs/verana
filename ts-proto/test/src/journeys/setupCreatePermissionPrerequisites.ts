/**
 * Journey: Setup Create Permission Prerequisites
 *
 * Step 1 of 2 for Create Permission.
 * Sets up account, funds it, and ensures schema/root permission exist.
 * Saves results to journey_results/ for Step 2.
 *
 * Usage:
 *   npm run test:setup-create-perm-prereqs
 */

import {
  createWallet,
  createAccountFromMnemonic,
  createSigningClient,
  getAccountInfo,
  fundAccount,
  config,
} from "../helpers/client";
import { getActiveTRAndSchema, getRootPermissionId, saveJourneyResult } from "../helpers/journeyResults";

// Master mnemonic - same for all accounts
const MASTER_MNEMONIC =
  process.env.MNEMONIC ||
  "pink glory help gown abstract eight nice crazy forward ketchup skill cheese";

// Account index for Journey 14 (Create Permission)
const ACCOUNT_INDEX = 14;

async function main() {
  console.log("=".repeat(60));
  console.log("Journey: Setup Create Permission Prerequisites (Step 1/2)");
  console.log("=".repeat(60));
  console.log();

  // Step 1: Setup cooluser account (for funding)
  console.log("Step 1: Setting up cooluser account (for funding)...");
  const cooluserWallet = await createWallet(MASTER_MNEMONIC);
  const cooluserAccount = await getAccountInfo(cooluserWallet);
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

  // Step 5: Connect account_14 to blockchain and verify balance
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
  let did: string | undefined;
  let trustRegistryId: number | undefined;

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
      trustRegistryId = trAndSchema.trustRegistryId;
      console.log(`Step 6: Using active TR/CS from journey results:`);
      console.log(`  - Trust Registry ID: ${trustRegistryId}`);
      console.log(`  - Schema ID: ${schemaId}`);
      console.log(`  - DID: ${did}`);
    } else {
      console.log("Step 6: No active TR/CS found, will create in next step...");
      // Don't create here - let the next step handle it
      // This avoids multiple transactions in one journey
    }
  }

  if (!schemaId || !did) {
    console.log("  ⚠️  Schema ID and DID not found. Will be created in next step if needed.");
    // Don't exit - let next step handle creation
  }

  console.log();

  // Step 7: Check Root Permission ID from Journey 13 (REQUIRED - ecosystem permission must exist)
  console.log("Step 7: Checking Root Permission ID from Journey 13...");
  const rootPermId = getRootPermissionId();
  if (!rootPermId) {
    console.log("  ⚠️  Root Permission not found. Journey 13 (Create Root Permission) must be run first.");
    console.log("     The next step will fail if root permission is not available.");
  } else {
    console.log(`  ✓ Root Permission ID: ${rootPermId}`);
  }
  console.log();

  // Save prerequisites for next step
  saveJourneyResult("create-perm-prereqs", {
    accountIndex: ACCOUNT_INDEX.toString(),
    accountAddress: account14.address,
    schemaId: schemaId?.toString(),
    trustRegistryId: trustRegistryId?.toString(),
    did: did || undefined,
    rootPermissionId: rootPermId?.toString(),
    // Save cooluser info for potential schema creation
    cooluserAddress: cooluserAccount.address,
  });

  console.log("=".repeat(60));
  console.log("✅ SUCCESS! Prerequisites setup completed!");
  console.log("=".repeat(60));
  console.log(`  Account_${ACCOUNT_INDEX} address: ${account14.address}`);
  if (schemaId) {
    console.log(`  Schema ID: ${schemaId}`);
  } else {
    console.log(`  Schema ID: Will be created in next step`);
  }
  if (rootPermId) {
    console.log(`  Root Permission ID: ${rootPermId}`);
  } else {
    console.log(`  Root Permission ID: Not found (must run Journey 13 first)`);
  }
  console.log();
  console.log("  💾 Results saved to journey_results/create-perm-prereqs.json");
  console.log("  ➡️  Run next step: npm run test:create-perm");
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
