/**
 * Journey: Start Permission VP
 *
 * This script demonstrates how to start a Permission Validation Process using the
 * TypeScript client and the generated protobuf types.
 *
 * Usage:
 *   VALIDATOR_PERM_ID=1 TYPE=ISSUER COUNTRY=US npm run test:start-perm-vp
 *   # Or let it create a validator permission first, then start VP
 *   npm run test:start-perm-vp
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
  generateUniqueDID,
} from "../helpers/client";
import { typeUrls } from "../helpers/registry";
import { MsgStartPermissionVP } from "../../../src/codec/verana/perm/v1/tx";
import { PermissionType } from "../../../src/codec/verana/perm/v1/types";
import { getActiveTRAndSchema, getRootPermissionId, savePermissionId } from "../helpers/journeyResults";

// Master mnemonic - same for all accounts
const MASTER_MNEMONIC =
  process.env.MNEMONIC ||
  "pink glory help gown abstract eight nice crazy forward ketchup skill cheese";

// Account index for Journey 17 (Start Permission VP)
const ACCOUNT_INDEX = 17;

async function main() {
  console.log("=".repeat(60));
  console.log("Journey: Start Permission VP (TypeScript Client)");
  console.log("=".repeat(60));
  console.log();

  // Step 1: Setup cooluser account (for funding)
  console.log("Step 1: Setting up cooluser account (for funding)...");
  const cooluserWallet = await createWallet(MASTER_MNEMONIC);
  const cooluserAccount = await getAccountInfo(cooluserWallet);
  const cooluserClient = await createSigningClient(cooluserWallet);
  console.log(`  ✓ Cooluser address: ${cooluserAccount.address}`);
  console.log();

  // Step 2: Create account_17 from mnemonic with derivation path 17
  console.log(`Step 2: Creating account_${ACCOUNT_INDEX} from mnemonic (derivation path ${ACCOUNT_INDEX})...`);
  const account17Wallet = await createAccountFromMnemonic(MASTER_MNEMONIC, ACCOUNT_INDEX);
  const account17 = await getAccountInfo(account17Wallet);
  console.log(`  ✓ Account_${ACCOUNT_INDEX} address: ${account17.address}`);
  console.log();

  // Step 3: Fund account_17 from cooluser
  console.log("Step 3: Funding account_17 from cooluser...");
  const fundingAmount = "1000000000uvna"; // 1 VNA
  try {
    const fundResult = await fundAccount(
      MASTER_MNEMONIC,
      cooluserAccount.address,
      account17.address,
      fundingAmount
    );
    if (fundResult.code === 0) {
      console.log(`  ✓ Funded account_17 with ${fundingAmount}`);
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

  // Step 5: Connect account_17 to blockchain
  console.log("Step 5: Connecting account_17 to Verana blockchain...");
  console.log(`  RPC Endpoint: ${config.rpcEndpoint}`);
  const client = await createSigningClient(account17Wallet);
  console.log("  ✓ Connected successfully");
  
  // Verify balance
  const balance = await client.getBalance(account17.address, config.denom);
  console.log(`  Balance: ${balance.amount} ${balance.denom}`);
  if (BigInt(balance.amount) < BigInt(1000000)) {
    console.log("  ⚠️  Warning: Low balance. Funding may not have completed.");
    process.exit(1);
  }
  console.log();

  // Step 6: Load root permission ID from Journey 13
  const permissionType = process.env.TYPE === "VERIFIER" ? PermissionType.VERIFIER : PermissionType.ISSUER;
  const country = process.env.COUNTRY || "US";
  
  let validatorPermId: number | undefined;
  if (process.env.VALIDATOR_PERM_ID) {
    validatorPermId = parseInt(process.env.VALIDATOR_PERM_ID, 10);
    if (isNaN(validatorPermId)) {
      console.log("  ❌ Invalid VALIDATOR_PERM_ID provided");
      process.exit(1);
    }
    console.log(`Step 6: Using provided Validator Permission ID: ${validatorPermId}`);
  } else {
    // Load root permission ID from Journey 13
    const loadedRootPermId = getRootPermissionId();
    if (loadedRootPermId === null) {
      console.log("  ❌ Root Permission not found. Journey 13 (Create Root Permission) must be run first.");
      process.exit(1);
    }
    validatorPermId = loadedRootPermId;
    console.log(`Step 6: Loaded Root Permission ID from Journey 13: ${validatorPermId}`);
  }
  console.log();

  // Step 7: Verify TR/CS exist (for reference)
  const trAndSchema = getActiveTRAndSchema();
  if (trAndSchema) {
    console.log(`Step 7: Active TR/CS:`);
    console.log(`  - Trust Registry ID: ${trAndSchema.trustRegistryId}`);
    console.log(`  - Schema ID: ${trAndSchema.schemaId}`);
    console.log(`  - DID: ${trAndSchema.did}`);
  }
  console.log();

  if (!validatorPermId) {
    console.log("  ❌ Validator Permission ID is required");
    process.exit(1);
  }

  // Step 8: Start Permission VP transaction
  console.log("Step 8: Starting Permission VP transaction...");
  const did = process.env.DID || generateUniqueDID();
  const msg = {
    typeUrl: typeUrls.MsgStartPermissionVP,
    value: MsgStartPermissionVP.fromPartial({
      creator: account17.address,
      type: permissionType,
      validatorPermId: validatorPermId,
      country: country,
      did: did,
    }),
  };
  console.log("  Message details:");
  console.log(`    - Creator: ${account17.address} (account_${ACCOUNT_INDEX})`);
  console.log(`    - Permission Type: ${PermissionType[permissionType]} (${permissionType})`);
  console.log(`    - Validator Permission ID: ${validatorPermId}`);
  console.log(`    - Country: ${country}`);
  console.log(`    - DID: ${did}`);
  console.log();

  // Step 9: Sign and broadcast
  console.log("Step 9: Signing and broadcasting transaction...");
  try {
    const fee = await calculateFeeWithSimulation(
      client,
      account17.address,
      [msg],
      "Starting Permission VP via TypeScript client"
    );
    console.log(`  Calculated gas: ${fee.gas}, fee: ${fee.amount[0].amount}${fee.amount[0].denom}`);

    // Use retry logic for consistency (matches frontend pattern)
    const result = await signAndBroadcastWithRetry(
      client,
      account17.address,
      [msg],
      fee,
      "Starting Permission VP via TypeScript client"
    );

    console.log();
    if (result.code === 0) {
      console.log("✅ SUCCESS! Permission VP started successfully!");
      console.log("=".repeat(60));
      console.log(`  Transaction Hash: ${result.transactionHash}`);
      console.log(`  Block Height: ${result.height}`);
      console.log(`  Gas Used: ${result.gasUsed}/${result.gasWanted}`);
      
      // Extract permission ID from events and save to journey results
      let permissionId: number | null = null;
      const events = result.events || [];
      for (const event of events) {
        if (event.type === "start_permission_vp" || event.type === "verana.perm.v1.EventStartPermissionVP") {
          for (const attr of event.attributes) {
            if (attr.key === "permission_id" || attr.key === "id") {
              permissionId = parseInt(attr.value, 10);
              console.log(`  Permission ID: ${permissionId}`);
            }
          }
        }
      }
      
      // Save permission ID for reuse in Journeys 18 and 20
      if (permissionId !== null) {
        savePermissionId(permissionId, "start-perm-vp");
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
  }
  process.exit(1);
});

