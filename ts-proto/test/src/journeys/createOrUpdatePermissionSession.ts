/**
 * Journey: Create Or Update Permission Session
 *
 * Step 3 of 3 for Permission Session creation.
 * Loads prerequisites from Steps 1 & 2, creates the Permission Session.
 *
 * Usage:
 *   # Using prerequisites from previous steps (recommended)
 *   npm run test:create-perm-session
 *
 *   # Or provide specific permission IDs
 *   ISSUER_PERM_ID=1 VERIFIER_PERM_ID=2 AGENT_PERM_ID=3 npm run test:create-perm-session
 */

import {
  createAminoWallet,
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
import { loadJourneyResult } from "../helpers/journeyResults";

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
  console.log("Journey: Create Or Update Permission Session (Step 3/3)");
  console.log("=".repeat(60));
  console.log();

  // Setup wallet and client
  const wallet = await createAminoWallet(TEST_MNEMONIC);
  const account = await getAccountInfo(wallet);
  const client = await createSigningClient(wallet);
  console.log(`  ✓ Wallet address: ${account.address}`);
  console.log(`  ✓ Connected to ${config.rpcEndpoint}`);
  console.log();

  // Check balance
  const balance = await client.getBalance(account.address, config.denom);
  if (BigInt(balance.amount) < BigInt(1000000)) {
    console.log("  ⚠️  Warning: Low balance.");
    process.exit(1);
  }

  let issuerPermId: number;
  let verifierPermId: number;
  let agentPermId: number;

  // Check for environment variables first
  if (process.env.ISSUER_PERM_ID && process.env.VERIFIER_PERM_ID && process.env.AGENT_PERM_ID) {
    issuerPermId = parseInt(process.env.ISSUER_PERM_ID, 10);
    verifierPermId = parseInt(process.env.VERIFIER_PERM_ID, 10);
    agentPermId = parseInt(process.env.AGENT_PERM_ID, 10);

    if (isNaN(issuerPermId) || isNaN(verifierPermId) || isNaN(agentPermId)) {
      console.log("  ❌ Invalid permission IDs provided");
      process.exit(1);
    }

    console.log("Using provided Permission IDs:");
    console.log(`  - Issuer: ${issuerPermId}`);
    console.log(`  - Verifier: ${verifierPermId}`);
    console.log(`  - Agent: ${agentPermId}`);
  } else {
    // Load from journey results (created by Steps 1 & 2)
    console.log("Loading permission IDs from previous steps...");
    const perms = loadJourneyResult("perm-session-perms");

    if (!perms?.issuerPermId || !perms?.verifierPermId) {
      console.log("  ❌ Permission IDs not found. Run the setup steps first:");
      console.log("     npm run test:setup-perm-session-prereqs");
      console.log("     npm run test:setup-perm-session-perms");
      process.exit(1);
    }

    issuerPermId = parseInt(perms.issuerPermId, 10);
    verifierPermId = parseInt(perms.verifierPermId, 10);
    agentPermId = perms.agentPermId ? parseInt(perms.agentPermId, 10) : issuerPermId;

    console.log(`  ✓ Loaded from journey_results/perm-session-perms.json:`);
    console.log(`    - Issuer: ${issuerPermId}`);
    console.log(`    - Verifier: ${verifierPermId}`);
    console.log(`    - Agent: ${agentPermId}`);
  }
  console.log();

  // Wait for permissions to become effective (created with 10s future effectiveFrom)
  console.log("Waiting for permissions to become effective...");
  const queryClient = await createQueryClient();
  try {
    const startTime = Date.now();
    const maxWait = 15000; // 15 seconds

    while (Date.now() - startTime < maxWait) {
      await new Promise((resolve) => setTimeout(resolve, 1000));
      const elapsed = Date.now() - startTime;
      if (elapsed >= 12000) {
        const currentBlockTime = await getBlockTime(queryClient);
        console.log(`  ✓ Waited ${Math.ceil(elapsed / 1000)} seconds, block time: ${currentBlockTime.toISOString()}`);
        break;
      }
    }
  } finally {
    queryClient.disconnect();
  }
  console.log();

  // Create Permission Session
  console.log("Creating Permission Session...");
  const sessionId = generateUUID();
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

  console.log(`  - Session ID: ${sessionId}`);
  console.log(`  - Issuer Permission ID: ${issuerPermId}`);
  console.log(`  - Verifier Permission ID: ${verifierPermId}`);
  console.log(`  - Agent Permission ID: ${agentPermId}`);
  console.log(`  - Wallet Agent Permission ID: ${walletAgentPermId}`);
  console.log();

  console.log("Signing and broadcasting transaction...");
  try {
    const fee = await calculateFeeWithSimulation(
      client,
      account.address,
      [msg],
      "Creating Permission Session via TypeScript client"
    );
    console.log(`  Calculated gas: ${fee.gas}, fee: ${fee.amount[0].amount}${fee.amount[0].denom}`);

    const result = await signAndBroadcastWithRetry(
      client,
      account.address,
      [msg],
      fee,
      "Creating Permission Session via TypeScript client"
    );

    console.log();
    if (result.code === 0) {
      console.log("=".repeat(60));
      console.log("✅ SUCCESS! Permission Session created successfully!");
      console.log("=".repeat(60));
      console.log(`  Transaction Hash: ${result.transactionHash}`);
      console.log(`  Block Height: ${result.height}`);
      console.log(`  Gas Used: ${result.gasUsed}/${result.gasWanted}`);
      console.log(`  Session ID: ${sessionId}`);
      console.log("=".repeat(60));
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
}

main().catch((error: any) => {
  console.error("\n❌ Fatal error:", error.message || error);
  if (error.cause?.code === "ECONNREFUSED" || error.message?.includes("fetch failed")) {
    console.error("\n⚠️  Connection Error: Cannot connect to the blockchain.");
    console.error(`   Make sure the Verana blockchain is running at ${config.rpcEndpoint}`);
  }
  process.exit(1);
});
