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
  createSigningClient,
  getAccountInfo,
  calculateFeeWithSimulation,
  signAndBroadcastWithRetry,
  config,
  createQueryClient,
  getBlockTime,
} from "../helpers/client";
import { typeUrls } from "../helpers/registry";
import { MsgStartPermissionVP } from "../../../src/codec/verana/perm/v1/tx";
import { PermissionType } from "../../../src/codec/verana/perm/v1/types";
import { createSchemaForTest, createRootPermissionForTest } from "../helpers/permissionHelpers";
import { generateUniqueDID } from "../helpers/client";

const TEST_MNEMONIC =
  process.env.MNEMONIC ||
  "pink glory help gown abstract eight nice crazy forward ketchup skill cheese";

async function main() {
  console.log("=".repeat(60));
  console.log("Journey: Start Permission VP (TypeScript Client)");
  console.log("=".repeat(60));
  console.log();

  // Using Amino Sign to match frontend
  const wallet = await createWallet(TEST_MNEMONIC);
  const account = await getAccountInfo(wallet);
  console.log(`  ✓ Using Amino Sign (matches frontend behavior)`);
  const client = await createSigningClient(wallet);

  console.log(`  ✓ Wallet address: ${account.address}`);
  console.log(`  ✓ Connected to ${config.rpcEndpoint}`);
  console.log();

  const balance = await client.getBalance(account.address, config.denom);
  if (BigInt(balance.amount) < BigInt(1000000)) {
    console.log("  ⚠️  Warning: Low balance.");
    process.exit(1);
  }

  let validatorPermId: number | undefined;
  const permissionType = process.env.TYPE === "VERIFIER" ? PermissionType.VERIFIER : PermissionType.ISSUER;
  const country = process.env.COUNTRY || "US";

  if (process.env.VALIDATOR_PERM_ID) {
    validatorPermId = parseInt(process.env.VALIDATOR_PERM_ID, 10);
    if (isNaN(validatorPermId)) {
      console.log("  ❌ Invalid VALIDATOR_PERM_ID provided");
      process.exit(1);
    }
    console.log(`Step 4: Using provided Validator Permission ID: ${validatorPermId}`);
  } else {
    console.log("Step 4: Creating schema and validator permission first...");
    const { schemaId, did } = await createSchemaForTest(client, account.address);
    // Refresh sequence after schema creation to ensure cache is updated
    await client.getSequence(account.address);
    await new Promise((resolve) => setTimeout(resolve, 500));
    await client.getSequence(account.address);
    // Create a root permission as validator (ECOSYSTEM type)
    validatorPermId = await createRootPermissionForTest(client, account.address, schemaId, did);
    console.log(`  ✓ Created Validator Permission (Root) with ID: ${validatorPermId}`);
    
    // Wait for validator permission to become effective (permissions are created with effectiveFrom 10 seconds in future)
    // We need to wait for blockchain block time to pass the effectiveFrom time
    // Since permissions are created with effectiveFrom = Date.now() + 10000, we wait 15 seconds to ensure
    // blockchain block time has advanced past that point
    console.log(`  ⏳ Waiting for validator permission to become effective (permissions require effective_from to be in the future)...`);
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
      console.log(`  ✓ Validator permission should now be effective`);
    } finally {
      queryClient.disconnect();
    }
    
    // Refresh sequence after waiting to ensure it's up to date
    await client.getSequence(account.address);
  }

  if (!validatorPermId) {
    console.log("  ❌ Validator Permission ID is required");
    process.exit(1);
  }

  console.log();

  console.log("Step 5: Starting Permission VP transaction...");
  const did = process.env.DID || generateUniqueDID();
  const msg = {
    typeUrl: typeUrls.MsgStartPermissionVP,
    value: MsgStartPermissionVP.fromPartial({
      creator: account.address,
      type: permissionType,
      validatorPermId: validatorPermId,
      country: country,
      did: did,
    }),
  };
  console.log(`    - Creator: ${account.address}`);
  console.log(`    - Permission Type: ${PermissionType[permissionType]} (${permissionType})`);
  console.log(`    - Validator Permission ID: ${validatorPermId}`);
  console.log(`    - Country: ${country}`);
  console.log(`    - DID: ${did}`);
  console.log();

  console.log("Step 6: Signing and broadcasting transaction...");
  try {
    const fee = await calculateFeeWithSimulation(
      client,
      account.address,
      [msg],
      "Starting Permission VP via TypeScript client"
    );
    console.log(`  Calculated gas: ${fee.gas}, fee: ${fee.amount[0].amount}${fee.amount[0].denom}`);

    // Use retry logic for consistency (matches frontend pattern)
    const result = await signAndBroadcastWithRetry(
      client,
      account.address,
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
      
      // Try to extract permission ID from events
      const events = result.events || [];
      for (const event of events) {
        if (event.type === "start_permission_vp" || event.type === "verana.perm.v1.EventStartPermissionVP") {
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

