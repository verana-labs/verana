/**
 * Journey: Create Or Update Permission Session
 *
 * This script demonstrates how to create or update a Permission Session using the
 * TypeScript client and the generated protobuf types.
 *
 * Usage:
 *   ISSUER_PERM_ID=1 VERIFIER_PERM_ID=2 AGENT_PERM_ID=3 npm run test:create-perm-session
 *   # Or let it create permissions first, then create session
 *   npm run test:create-perm-session
 */

import {
  createDirectWallet,
  createSigningClient,
  getAccountInfo,
  calculateFeeWithSimulation,
  signAndBroadcastWithRetry,
  config,
  createQueryClient,
  getBlockTime,
} from "../helpers/client";
import { typeUrls } from "../helpers/registry";
import { MsgCreateOrUpdatePermissionSession } from "../../../src/codec/verana/perm/v1/tx";
import { createSchemaForTest, createPermissionForTest, createRootPermissionForTest } from "../helpers/permissionHelpers";
import { PermissionType } from "../../../src/codec/verana/perm/v1/types";

const TEST_MNEMONIC =
  process.env.MNEMONIC ||
  "pink glory help gown abstract eight nice crazy forward ketchup skill cheese";

// Generate UUID v4
function generateUUID(): string {
  return "xxxxxxxx-xxxx-4xxx-yxxx-xxxxxxxxxxxx".replace(/[xy]/g, function (c) {
    const r = (Math.random() * 16) | 0;
    const v = c === "x" ? r : (r & 0x3) | 0x8;
    return v.toString(16);
  });
}

async function main() {
  console.log("=".repeat(60));
  console.log("Journey: Create Or Update Permission Session (TypeScript Client)");
  console.log("=".repeat(60));
  console.log();

  // Using Direct Sign for Permission Session
  const wallet = await createDirectWallet(TEST_MNEMONIC);
  const account = await getAccountInfo(wallet);
  const client = await createSigningClient(wallet);
  console.log(`  ✓ Using Direct Sign`);

  console.log(`  ✓ Wallet address: ${account.address}`);
  console.log(`  ✓ Connected to ${config.rpcEndpoint}`);
  console.log();

  const balance = await client.getBalance(account.address, config.denom);
  if (BigInt(balance.amount) < BigInt(1000000)) {
    console.log("  ⚠️  Warning: Low balance.");
    process.exit(1);
  }

  let issuerPermId: number | undefined;
  let verifierPermId: number | undefined;
  let agentPermId: number | undefined;

  if (process.env.ISSUER_PERM_ID && process.env.VERIFIER_PERM_ID && process.env.AGENT_PERM_ID) {
    issuerPermId = parseInt(process.env.ISSUER_PERM_ID, 10);
    verifierPermId = parseInt(process.env.VERIFIER_PERM_ID, 10);
    agentPermId = parseInt(process.env.AGENT_PERM_ID, 10);
    if (isNaN(issuerPermId) || isNaN(verifierPermId) || isNaN(agentPermId)) {
      console.log("  ❌ Invalid permission IDs provided");
      process.exit(1);
    }
    console.log(`Step 4: Using provided Permission IDs:`);
    console.log(`    - Issuer: ${issuerPermId}`);
    console.log(`    - Verifier: ${verifierPermId}`);
    console.log(`    - Agent: ${agentPermId}`);
  } else {
    console.log("Step 4: Creating schema and permissions first...");
    const { schemaId, did } = await createSchemaForTest(client, account.address);
    
    // Create root permission once for the schema (required prerequisite)
    // This ensures we only create it once, not once per permission type
    console.log(`  Creating root (ecosystem) permission for schema ${schemaId} first (required prerequisite)...`);
    try {
      await createRootPermissionForTest(client, account.address, schemaId, did);
      console.log(`  ✓ Root (ecosystem) permission created successfully`);
      // Wait and refresh sequence before creating regular permissions
      // The createRootPermissionForTest already waits for transaction confirmation
      await client.getSequence(account.address);
    } catch (error: any) {
      throw new Error(`Failed to create root (ecosystem) permission prerequisite for schema ${schemaId}: ${error.message}`);
    }
    
    // Now create regular permissions (they won't try to create root permission again)
    issuerPermId = await createPermissionForTest(client, account.address, schemaId, did, PermissionType.ISSUER, true); // Pass skipRoot=true
    // The createPermissionForTest already waits for transaction confirmation
    await client.getSequence(account.address);
    verifierPermId = await createPermissionForTest(client, account.address, schemaId, did, PermissionType.VERIFIER, true); // Pass skipRoot=true
    // Agent permission must be ISSUER type (not ISSUER_GRANTOR)
    // Use issuer permission as agent permission (matching test harness pattern)
    agentPermId = issuerPermId;
    console.log(`  ✓ Created Permissions:`);
    console.log(`    - Issuer: ${issuerPermId}`);
    console.log(`    - Verifier: ${verifierPermId}`);
    console.log(`    - Agent: ${agentPermId} (using issuer permission)`);
    
    // Wait for permissions to become effective (permissions are created with effectiveFrom 10 seconds in future)
    console.log(`  ⏳ Waiting for permissions to become effective (permissions require effective_from to be in the future)...`);
    const queryClient = await createQueryClient();
    try {
      // Wait for blockchain block time to advance (check every second)
      const startTime = Date.now();
      const maxWait = 20000; // 20 seconds max wait
      
      while (Date.now() - startTime < maxWait) {
        const waitElapsed = Date.now() - startTime;
        if (waitElapsed >= 15000) {
          // Double-check block time has advanced sufficiently
          const currentBlockTime = await getBlockTime(queryClient);
          console.log(`  ✓ Waited ${Math.ceil(waitElapsed / 1000)} seconds, block time: ${currentBlockTime.toISOString()}`);
          break;
        }
        await new Promise((resolve) => setTimeout(resolve, 1000));
      }
      console.log(`  ✓ Permissions should now be effective`);
    } finally {
      queryClient.disconnect();
    }
  }

  if (!issuerPermId || !verifierPermId || !agentPermId) {
    console.log("  ❌ All permission IDs are required");
    process.exit(1);
  }

  console.log();

  console.log("Step 5: Creating Permission Session transaction...");
  const sessionId = generateUUID();
  // walletAgentPermId is mandatory - use issuer permission ID (matching test harness pattern)
  const walletAgentPermId = process.env.WALLET_AGENT_PERM_ID
    ? parseInt(process.env.WALLET_AGENT_PERM_ID, 10)
    : issuerPermId;

  const msg = {
    typeUrl: typeUrls.MsgCreateOrUpdatePermissionSession,
    value: MsgCreateOrUpdatePermissionSession.fromPartial({
      creator: account.address,
      id: sessionId,
      issuerPermId: issuerPermId,
      verifierPermId: verifierPermId,
      agentPermId: agentPermId,
      walletAgentPermId: walletAgentPermId,
    }),
  };
  console.log(`    - Session ID: ${sessionId}`);
  console.log(`    - Issuer Permission ID: ${issuerPermId}`);
  console.log(`    - Verifier Permission ID: ${verifierPermId}`);
  console.log(`    - Agent Permission ID: ${agentPermId}`);
  if (walletAgentPermId) {
    console.log(`    - Wallet Agent Permission ID: ${walletAgentPermId}`);
  }
  console.log();

  console.log("Step 6: Signing and broadcasting transaction...");
  try {
    const fee = await calculateFeeWithSimulation(
      client,
      account.address,
      [msg],
      "Creating Permission Session via TypeScript client"
    );
    console.log(`  Calculated gas: ${fee.gas}, fee: ${fee.amount[0].amount}${fee.amount[0].denom}`);

    // Use retry logic for consistency (matches frontend pattern)
    const result = await signAndBroadcastWithRetry(
      client,
      account.address,
      [msg],
      fee,
      "Creating Permission Session via TypeScript client"
    );

    console.log();
    if (result.code === 0) {
      console.log("✅ SUCCESS! Permission Session created/updated successfully!");
      console.log("=".repeat(60));
      console.log(`  Transaction Hash: ${result.transactionHash}`);
      console.log(`  Block Height: ${result.height}`);
      console.log(`  Gas Used: ${result.gasUsed}/${result.gasWanted}`);
      console.log(`  Session ID: ${sessionId}`);
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

