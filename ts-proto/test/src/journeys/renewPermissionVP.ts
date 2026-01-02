/**
 * Journey: Renew Permission VP
 *
 * This script demonstrates how to renew a Permission Validation Process using the
 * TypeScript client and the generated protobuf types.
 *
 * Usage:
 *   PERM_ID=1 npm run test:renew-perm-vp
 *   # Or let it create a permission first, then renew it
 *   npm run test:renew-perm-vp
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
import { MsgRenewPermissionVP } from "../../../src/codec/verana/perm/v1/tx";
import { getActiveTRAndSchema, getPermissionId } from "../helpers/journeyResults";

// Master mnemonic - same for all accounts
const MASTER_MNEMONIC =
  process.env.MNEMONIC ||
  "pink glory help gown abstract eight nice crazy forward ketchup skill cheese";

// Account index for Journey 18 (Renew Permission VP) - REUSE account_17
const ACCOUNT_INDEX = 17;

async function main() {
  console.log("=".repeat(60));
  console.log("Journey: Renew Permission VP (TypeScript Client)");
  console.log("=".repeat(60));
  console.log();

  // Step 1: Create account_17 from mnemonic (REUSE from Journey 17)
  console.log(`Step 1: Creating account_${ACCOUNT_INDEX} from mnemonic (derivation path ${ACCOUNT_INDEX})...`);
  const account17Wallet = await createAccountFromMnemonic(MASTER_MNEMONIC, ACCOUNT_INDEX);
  const account17 = await getAccountInfo(account17Wallet);
  console.log(`  ✓ Account_${ACCOUNT_INDEX} address: ${account17.address}`);
  console.log();

  // Step 2: Connect account_17 to blockchain
  console.log("Step 2: Connecting account_17 to Verana blockchain...");
  console.log(`  RPC Endpoint: ${config.rpcEndpoint}`);
  const client = await createSigningClient(account17Wallet);
  console.log("  ✓ Connected successfully");
  
  // Verify balance (account should already be funded from Journey 17)
  const balance = await client.getBalance(account17.address, config.denom);
  console.log(`  Balance: ${balance.amount} ${balance.denom}`);
  if (BigInt(balance.amount) < BigInt(1000000)) {
    console.log("  ⚠️  Warning: Low balance. Account may need funding.");
    process.exit(1);
  }
  console.log();

  // Step 3: Load permission ID from Journey 17
  let permId: number | undefined;
  if (process.env.PERM_ID) {
    permId = parseInt(process.env.PERM_ID, 10);
    if (isNaN(permId)) {
      console.log("  ❌ Invalid PERM_ID provided");
      process.exit(1);
    }
    console.log(`Step 3: Using provided Permission ID: ${permId}`);
  } else {
    // Load permission ID from Journey 17
    const loadedPermId = getPermissionId("start-perm-vp");
    if (loadedPermId === null) {
      console.log("  ❌ Permission ID not found. Journey 17 (Start Permission VP) must be run first.");
      process.exit(1);
    }
    permId = loadedPermId;
    console.log(`Step 3: Loaded Permission ID from Journey 17: ${permId}`);
  }
  console.log();

  // Step 4: Verify TR/CS exist (for reference)
  const trAndSchema = getActiveTRAndSchema();
  if (trAndSchema) {
    console.log(`Step 4: Active TR/CS:`);
    console.log(`  - Trust Registry ID: ${trAndSchema.trustRegistryId}`);
    console.log(`  - Schema ID: ${trAndSchema.schemaId}`);
    console.log(`  - DID: ${trAndSchema.did}`);
  }
  console.log();

  if (!permId) {
    console.log("  ❌ Permission ID is required");
    process.exit(1);
  }

  // Step 5: Renew Permission VP transaction
  console.log("Step 5: Renewing Permission VP transaction...");
  const msg = {
    typeUrl: typeUrls.MsgRenewPermissionVP,
    value: MsgRenewPermissionVP.fromPartial({
      creator: account17.address,
      id: permId,
    }),
  };
  console.log("  Message details:");
  console.log(`    - Creator: ${account17.address} (account_${ACCOUNT_INDEX})`);
  console.log(`    - Permission ID: ${permId}`);
  console.log();

  // Step 6: Sign and broadcast
  console.log("Step 6: Signing and broadcasting transaction...");
  try {
    const fee = await calculateFeeWithSimulation(
      client,
      account17.address,
      [msg],
      "Renewing Permission VP via TypeScript client"
    );
    console.log(`  Calculated gas: ${fee.gas}, fee: ${fee.amount[0].amount}${fee.amount[0].denom}`);

    // Use retry logic for consistency (matches frontend pattern)
    const result = await signAndBroadcastWithRetry(
      client,
      account17.address,
      [msg],
      fee,
      "Renewing Permission VP via TypeScript client"
    );

    console.log();
    if (result.code === 0) {
      console.log("✅ SUCCESS! Permission VP renewed successfully!");
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

