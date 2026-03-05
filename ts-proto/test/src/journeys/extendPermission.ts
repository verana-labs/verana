/**
 * Journey: Adjust Permission
 *
 * This script demonstrates how to adjust a Permission's effective_until date using the
 * TypeScript client and the generated protobuf types.
 *
 * Usage:
 *   PERM_ID=1 npm run test:adjust-perm
 *   # Or let it create a permission first, then adjust it
 *   npm run test:adjust-perm
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
import { MsgAdjustPermission } from "../../../src/codec/verana/perm/v1/tx";
import { getActiveTRAndSchema, getPermissionId } from "../helpers/journeyResults";

// Master mnemonic - same for all accounts
const MASTER_MNEMONIC =
  process.env.MNEMONIC ||
  "pink glory help gown abstract eight nice crazy forward ketchup skill cheese";

// Account index for Journey 15 (Adjust Permission) - REUSE account_14
const ACCOUNT_INDEX = 14;

async function main() {
  console.log("=".repeat(60));
  console.log("Journey: Adjust Permission (TypeScript Client)");
  console.log("=".repeat(60));
  console.log();

  // Step 1: Create account_14 from mnemonic (REUSE from Journey 14)
  console.log(`Step 1: Creating account_${ACCOUNT_INDEX} from mnemonic (derivation path ${ACCOUNT_INDEX})...`);
  const account14Wallet = await createAccountFromMnemonic(MASTER_MNEMONIC, ACCOUNT_INDEX);
  const account14 = await getAccountInfo(account14Wallet);
  console.log(`  account_${ACCOUNT_INDEX} address: ${account14.address}`);
  console.log();

  // Step 2: Connect account_14 to blockchain
  console.log("Step 2: Connecting account_14 to Verana blockchain...");
  console.log(`  RPC Endpoint: ${config.rpcEndpoint}`);
  const client = await createSigningClient(account14Wallet);
  console.log("  Connected successfully");

  // Verify balance (account should already be funded from Journey 14)
  const balance = await client.getBalance(account14.address, config.denom);
  console.log(`  Balance: ${balance.amount} ${balance.denom}`);
  if (BigInt(balance.amount) < BigInt(1000000)) {
    console.log("  Warning: Low balance. Account may need funding.");
    process.exit(1);
  }
  console.log();

  // Step 3: Load permission ID from Journey 14
  let permId: number | undefined;
  if (process.env.PERM_ID) {
    permId = parseInt(process.env.PERM_ID, 10);
    if (isNaN(permId)) {
      console.log("  Invalid PERM_ID provided");
      process.exit(1);
    }
    console.log(`Step 3: Using provided Permission ID: ${permId}`);
  } else {
    // Load permission ID from Journey 14
    const loadedPermId = getPermissionId("create-permission");
    if (loadedPermId === null) {
      console.log("  Permission ID not found. Journey 14 (Create Permission) must be run first.");
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

  // Step 5: Wait for permission to become effective
  console.log(`Step 5: Waiting for permission to become effective...`);
  const queryClient = await createQueryClient();
  try {
    const startTime = Date.now();
    const maxWait = 20000; // 20 seconds max wait

    while (Date.now() - startTime < maxWait) {
      const waitElapsed = Date.now() - startTime;
      if (waitElapsed >= 15000) {
        const currentBlockTime = await getBlockTime(queryClient);
        console.log(`  Waited ${Math.ceil(waitElapsed / 1000)} seconds, block time: ${currentBlockTime.toISOString()}`);
        break;
      }
      await new Promise((resolve) => setTimeout(resolve, 1000));
    }
    console.log(`  Permission should now be effective`);
  } finally {
    queryClient.disconnect();
  }
  console.log();

  if (!permId) {
    console.log("  Permission ID is required");
    process.exit(1);
  }

  console.log();

  console.log("Step 6: Adjusting Permission transaction...");
  const newEffectiveUntil = new Date(Date.now() + 720 * 24 * 60 * 60 * 1000); // 720 days from now

  const msg = {
    typeUrl: typeUrls.MsgAdjustPermission,
    value: MsgAdjustPermission.fromPartial({
      authority: account14.address,
      operator: account14.address,
      id: permId,
      effectiveUntil: newEffectiveUntil,
    }),
  };
  console.log("  Message details:");
  console.log(`    - Authority: ${account14.address} (account_${ACCOUNT_INDEX})`);
  console.log(`    - Operator: ${account14.address} (account_${ACCOUNT_INDEX})`);
  console.log(`    - Permission ID: ${permId}`);
  console.log(`    - New Effective Until: ${newEffectiveUntil.toISOString()}`);
  console.log();

  console.log("Step 7: Signing and broadcasting transaction...");
  try {
    const fee = await calculateFeeWithSimulation(
      client,
      account14.address,
      [msg],
      "Adjusting Permission via TypeScript client"
    );
    console.log(`  Calculated gas: ${fee.gas}, fee: ${fee.amount[0].amount}${fee.amount[0].denom}`);

    const result = await signAndBroadcastWithRetry(
      client,
      account14.address,
      [msg],
      fee,
      "Adjusting Permission via TypeScript client"
    );

    console.log();
    if (result.code === 0) {
      console.log("SUCCESS! Permission adjusted successfully!");
      console.log("=".repeat(60));
      console.log(`  Transaction Hash: ${result.transactionHash}`);
      console.log(`  Block Height: ${result.height}`);
      console.log(`  Gas Used: ${result.gasUsed}/${result.gasWanted}`);
    } else {
      console.log("FAILED! Transaction failed.");
      console.log(`  Error Code: ${result.code}`);
      console.log(`  Raw Log: ${result.rawLog}`);
      process.exit(1);
    }
  } catch (error: any) {
    console.log("ERROR! Transaction failed with exception:");
    console.error(error);
    if (error.cause?.code === "ECONNREFUSED" || error.message?.includes("fetch failed")) {
      console.error("\nConnection Error: Cannot connect to the blockchain.");
      console.error(`   Make sure the Verana blockchain is running at ${config.rpcEndpoint}`);
    }
    process.exit(1);
  }

  console.log();
  console.log("=".repeat(60));
}

main().catch((error: any) => {
  console.error("\nFatal error:", error.message || error);
  if (error.cause?.code === "ECONNREFUSED" || error.message?.includes("fetch failed")) {
    console.error("\nConnection Error: Cannot connect to the blockchain.");
    console.error(`   Make sure the Verana blockchain is running at ${config.rpcEndpoint}`);
  }
  process.exit(1);
});
