/**
 * Journey: Setup Create Root Permission Prerequisites
 *
 * Step 1 of 2 for Create Root Permission.
 * Sets up account, funds it, and ensures TR/CS exist.
 * Saves results to journey_results/ for Step 2.
 *
 * Usage:
 *   npm run test:setup-create-root-perm-prereqs
 */

import {
  createWallet,
  createAccountFromMnemonic,
  createSigningClient,
  getAccountInfo,
  fundAccount,
  config,
  waitForSequencePropagation,
} from "../helpers/client";
import { getActiveTRAndSchema, saveJourneyResult } from "../helpers/journeyResults";
import { createSchemaForTest } from "../helpers/permissionHelpers";

// Master mnemonic - same for all accounts
const MASTER_MNEMONIC =
  process.env.MNEMONIC ||
  "pink glory help gown abstract eight nice crazy forward ketchup skill cheese";

// Account index for Journey 13 (Create Root Permission)
const ACCOUNT_INDEX = 13;

async function main() {
  console.log("=".repeat(60));
  console.log("Journey: Setup Create Root Permission Prerequisites (Step 1/2)");
  console.log("=".repeat(60));
  console.log();

  // Step 1: Setup cooluser account (for funding)
  console.log("Step 1: Setting up cooluser account (for funding)...");
  const cooluserWallet = await createWallet(MASTER_MNEMONIC);
  const cooluserAccount = await getAccountInfo(cooluserWallet);
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

  // Step 5: Connect account_13 to blockchain and verify balance
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
          trustRegistryId = trAndSchema.trustRegistryId;
          console.log(`  ✓ Reusing TR/CS from journey results:`);
          console.log(`    - Trust Registry ID: ${trustRegistryId}`);
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
        trustRegistryId = newSchema.trustRegistryId;
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
      trustRegistryId = newSchema.trustRegistryId;
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

  // Save prerequisites for next step
  saveJourneyResult("create-root-perm-prereqs", {
    accountIndex: ACCOUNT_INDEX.toString(),
    accountAddress: account13.address,
    schemaId: schemaId.toString(),
    trustRegistryId: trustRegistryId?.toString(),
    did: did,
    // Save cooluser info for potential schema creation
    cooluserAddress: cooluserAccount.address,
  });

  console.log("=".repeat(60));
  console.log("✅ SUCCESS! Prerequisites setup completed!");
  console.log("=".repeat(60));
  console.log(`  Account_${ACCOUNT_INDEX} address: ${account13.address}`);
  console.log(`  Schema ID: ${schemaId}`);
  if (trustRegistryId) {
    console.log(`  Trust Registry ID: ${trustRegistryId}`);
  }
  console.log(`  DID: ${did}`);
  console.log();
  console.log("  💾 Results saved to journey_results/create-root-perm-prereqs.json");
  console.log("  ➡️  Run next step: npm run test:create-root-perm");
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
