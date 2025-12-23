/**
 * Journey: Revoke Permission
 *
 * This script demonstrates how to revoke a Permission using the
 * TypeScript client and the generated protobuf types.
 *
 * Usage:
 *   PERM_ID=1 npm run test:revoke-perm
 *   # Or let it create a permission first, then revoke it
 *   npm run test:revoke-perm
 */

import {
  createWallet,
  createSigningClient,
  getAccountInfo,
  calculateFeeWithSimulation,
  config,
  createQueryClient,
  getBlockTime,
} from "../helpers/client";
import { typeUrls } from "../helpers/registry";
import { MsgRevokePermission } from "../../../src/codec/verana/perm/v1/tx";
import { createSchemaForTest, createPermissionForTest } from "../helpers/permissionHelpers";

const TEST_MNEMONIC =
  process.env.MNEMONIC ||
  "pink glory help gown abstract eight nice crazy forward ketchup skill cheese";

async function main() {
  console.log("=".repeat(60));
  console.log("Journey: Revoke Permission (TypeScript Client)");
  console.log("=".repeat(60));
  console.log();

  const wallet = await createWallet(TEST_MNEMONIC);
  const account = await getAccountInfo(wallet);
  const client = await createSigningClient(wallet);

  console.log(`  ✓ Wallet address: ${account.address}`);
  console.log(`  ✓ Connected to ${config.rpcEndpoint}`);
  console.log();

  const balance = await client.getBalance(account.address, config.denom);
  if (BigInt(balance.amount) < BigInt(1000000)) {
    console.log("  ⚠️  Warning: Low balance.");
    process.exit(1);
  }

  let permId: number | undefined;
  if (process.env.PERM_ID) {
    permId = parseInt(process.env.PERM_ID, 10);
    if (isNaN(permId)) {
      console.log("  ❌ Invalid PERM_ID provided");
      process.exit(1);
    }
    console.log(`Step 4: Using provided Permission ID: ${permId}`);
  } else {
    console.log("Step 4: Creating schema and permission first...");
    const { schemaId, did } = await createSchemaForTest(client, account.address);
    permId = await createPermissionForTest(client, account.address, schemaId, did);
    console.log(`  ✓ Created Permission with ID: ${permId}`);
    
    // Wait for permission to become effective (permissions are created with effectiveFrom 10 seconds in future)
    // We need to wait for blockchain block time to pass the effectiveFrom time
    // Since permissions are created with effectiveFrom = Date.now() + 10000, we wait 15 seconds to ensure
    // blockchain block time has advanced past that point
    console.log(`  ⏳ Waiting for permission to become effective (permissions require effective_from to be in the future)...`);
    const queryClient = await createQueryClient();
    try {
      // Wait for blockchain block time to advance (check every second)
      const startTime = Date.now();
      let lastBlockTime: Date | null = null;
      const maxWait = 20000; // 20 seconds max wait
      
      while (Date.now() - startTime < maxWait) {
        const blockTime = await getBlockTime(queryClient);
        lastBlockTime = blockTime;
        
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
  }

  if (!permId) {
    console.log("  ❌ Permission ID is required");
    process.exit(1);
  }

  console.log();

  console.log("Step 5: Revoking Permission transaction...");
  const msg = {
    typeUrl: typeUrls.MsgRevokePermission,
    value: MsgRevokePermission.fromPartial({
      creator: account.address,
      id: permId,
    }),
  };
  console.log(`    - Permission ID: ${permId}`);
  console.log();

  console.log("Step 6: Signing and broadcasting transaction...");
  try {
    const fee = await calculateFeeWithSimulation(
      client,
      account.address,
      [msg],
      "Revoking Permission via TypeScript client"
    );
    console.log(`  Calculated gas: ${fee.gas}, fee: ${fee.amount[0].amount}${fee.amount[0].denom}`);

    const result = await client.signAndBroadcast(account.address, [msg], fee, "Revoking Permission via TypeScript client");

    console.log();
    if (result.code === 0) {
      console.log("✅ SUCCESS! Permission revoked successfully!");
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

