/**
 * Journey: Extend Permission
 *
 * This script demonstrates how to extend a Permission's effective_until date using the
 * TypeScript client and the generated protobuf types.
 *
 * Usage:
 *   PERM_ID=1 npm run test:extend-perm
 *   # Or let it create a permission first, then extend it
 *   npm run test:extend-perm
 */

import {
  createAccountFromMnemonic,
  createSigningClient,
  getAccountInfo,
  calculateFeeWithSimulation,
  signAndBroadcastWithRetry,
  config,
  createQueryClient,
  getBlockTime,
} from "../helpers/client";
import { typeUrls } from "../helpers/registry";
import { MsgExtendPermission } from "../../../src/codec/verana/perm/v1/tx";
import { getActiveTRAndSchema, getPermissionId } from "../helpers/journeyResults";

// Master mnemonic - same for all accounts
const MASTER_MNEMONIC =
  process.env.MNEMONIC ||
  "pink glory help gown abstract eight nice crazy forward ketchup skill cheese";

// Account index for Journey 15 (Extend Permission) - REUSE account_14
const ACCOUNT_INDEX = 14;

async function main() {
  console.log("=".repeat(60));
  console.log("Journey: Extend Permission (TypeScript Client)");
  console.log("=".repeat(60));
  console.log();

  // Step 1: Create account_14 from mnemonic (REUSE from Journey 14)
  console.log(`Step 1: Creating account_${ACCOUNT_INDEX} from mnemonic (derivation path ${ACCOUNT_INDEX})...`);
  const account14Wallet = await createAccountFromMnemonic(MASTER_MNEMONIC, ACCOUNT_INDEX);
  const account14 = await getAccountInfo(account14Wallet);
  console.log(`  ✓ Account_${ACCOUNT_INDEX} address: ${account14.address}`);
  console.log();

  // Step 2: Connect account_14 to blockchain
  console.log("Step 2: Connecting account_14 to Verana blockchain...");
  console.log(`  RPC Endpoint: ${config.rpcEndpoint}`);
  const client = await createSigningClient(account14Wallet);
  console.log("  ✓ Connected successfully");
  
  // Verify balance (account should already be funded from Journey 14)
  const balance = await client.getBalance(account14.address, config.denom);
  console.log(`  Balance: ${balance.amount} ${balance.denom}`);
  if (BigInt(balance.amount) < BigInt(1000000)) {
    console.log("  ⚠️  Warning: Low balance. Account may need funding.");
    process.exit(1);
  }
  console.log();

  // Step 3: Load permission ID from Journey 14
  let permId: number | undefined;
  if (process.env.PERM_ID) {
    permId = parseInt(process.env.PERM_ID, 10);
    if (isNaN(permId)) {
      console.log("  ❌ Invalid PERM_ID provided");
      process.exit(1);
    }
    console.log(`Step 3: Using provided Permission ID: ${permId}`);
  } else {
    // Load permission ID from Journey 14
    const loadedPermId = getPermissionId("create-permission");
    if (loadedPermId === null) {
      console.log("  ❌ Permission ID not found. Journey 14 (Create Permission) must be run first.");
      process.exit(1);
    }
    permId = loadedPermId;
    console.log(`Step 3: Loaded Permission ID from Journey 14: ${permId}`);
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

  // Step 5: Wait for permission to become effective (permissions are created with effectiveFrom 10 seconds in future)
  // We need to wait for blockchain block time to pass the effectiveFrom time
  // Since permissions are created with effectiveFrom = Date.now() + 10000, we wait 15 seconds to ensure
  // blockchain block time has advanced past that point
  console.log(`Step 5: Waiting for permission to become effective (permissions require effective_from to be in the future)...`);
  const queryClient = await createQueryClient();
  try {
    // Wait for blockchain block time to advance (check every second)
    const startTime = Date.now();
    const maxWait = 20000; // 20 seconds max wait
    
    while (Date.now() - startTime < maxWait) {
      const blockTime = await getBlockTime(queryClient);
      
      // Permissions are created with effectiveFrom = Date.now() + 10000 (10 seconds in future)
      // We need block time to be at least 10 seconds after the creation time
      // Since we don't know exact creation time, wait 15 seconds from now to be safe
      const waitElapsed = Date.now() - startTime;
      if (waitElapsed >= 15000) {
        // Double-check block time has advanced sufficiently
        const currentBlockTime = await getBlockTime(queryClient);
        console.log(`  ✓ Waited ${Math.ceil(waitElapsed / 1000)} seconds, block time: ${currentBlockTime.toISOString()}`);
        break;
      }
      await new Promise((resolve) => setTimeout(resolve, 1000));
    }
    console.log(`  ✓ Permission should now be effective`);
  } finally {
    queryClient.disconnect();
  }
  console.log();

  if (!permId) {
    console.log("  ❌ Permission ID is required");
    process.exit(1);
  }

  console.log();

  console.log("Step 6: Extending Permission transaction...");
  const newEffectiveUntil = new Date(Date.now() + 720 * 24 * 60 * 60 * 1000); // 720 days from now

  const msg = {
    typeUrl: typeUrls.MsgExtendPermission,
    value: MsgExtendPermission.fromPartial({
      creator: account14.address,
      id: permId,
      effectiveUntil: newEffectiveUntil,
    }),
  };
  console.log("  Message details:");
  console.log(`    - Creator: ${account14.address} (account_${ACCOUNT_INDEX})`);
  console.log(`    - Permission ID: ${permId}`);
  console.log(`    - New Effective Until: ${newEffectiveUntil.toISOString()}`);
  console.log();

  console.log("Step 7: Signing and broadcasting transaction...");
  try {
    const fee = await calculateFeeWithSimulation(
      client,
      account14.address,
      [msg],
      "Extending Permission via TypeScript client"
    );
    console.log(`  Calculated gas: ${fee.gas}, fee: ${fee.amount[0].amount}${fee.amount[0].denom}`);

    // Use retry logic for consistency (matches frontend pattern)
    const result = await signAndBroadcastWithRetry(
      client,
      account14.address,
      [msg],
      fee,
      "Extending Permission via TypeScript client"
    );

    console.log();
    if (result.code === 0) {
      console.log("✅ SUCCESS! Permission extended successfully!");
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

