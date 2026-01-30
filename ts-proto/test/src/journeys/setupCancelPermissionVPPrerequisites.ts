/**
 * Journey: Setup Cancel Permission VP Prerequisites
 *
 * Step 1 of 2 for Cancel Permission VP Last Request.
 * Sets up account, creates schema, root permission, and starts VP (creates permission in PENDING state).
 * Saves results to journey_results/ for Step 2.
 *
 * Usage:
 *   npm run test:setup-cancel-perm-vp-prereqs
 */

import {
  createWallet,
  createSigningClient,
  getAccountInfo,
  calculateFeeWithSimulation,
  signAndBroadcastWithRetry,
  config,
  generateUniqueDID,
  createQueryClient,
  getBlockTime,
  waitForSequencePropagation,
} from "../helpers/client";
import { typeUrls } from "../helpers/registry";
import { MsgStartPermissionVP } from "../../../src/codec/verana/perm/v1/tx";
import { PermissionType } from "../../../src/codec/verana/perm/v1/types";
import { createSchemaForTest, createRootPermissionForTest } from "../helpers/permissionHelpers";
import { saveJourneyResult } from "../helpers/journeyResults";

const TEST_MNEMONIC =
  process.env.MNEMONIC ||
  "pink glory help gown abstract eight nice crazy forward ketchup skill cheese";

async function main() {
  console.log("=".repeat(60));
  console.log("Journey: Setup Cancel Permission VP Prerequisites (Step 1/2)");
  console.log("=".repeat(60));
  console.log();

  // Using Amino Sign to match frontend
  const wallet = await createWallet(TEST_MNEMONIC);
  const account = await getAccountInfo(wallet);
  const client = await createSigningClient(wallet);

  console.log(`  ✓ Wallet address: ${account.address}`);
  console.log(`  ✓ Using Amino Sign (matches frontend behavior)`);
  console.log(`  ✓ Connected to ${config.rpcEndpoint}`);
  console.log();

  const balance = await client.getBalance(account.address, config.denom);
  if (BigInt(balance.amount) < BigInt(1000000)) {
    console.log("  ⚠️  Warning: Low balance.");
    process.exit(1);
  }

  console.log("Step 1: Creating schema, validator permission, and starting VP (creates permission in PENDING state)...");
  // Create schema and root permission (validator)
  const { schemaId, did } = await createSchemaForTest(client, account.address);
  // Wait for sequence to propagate after schema creation (poll with 60s timeout)
  await waitForSequencePropagation(client, account.address);
  const validatorPermId = await createRootPermissionForTest(client, account.address, schemaId, did);
  // Wait for sequence to propagate after root permission creation (poll with 60s timeout)
  await waitForSequencePropagation(client, account.address);
  console.log(`  ✓ Created Validator Permission (Root) with ID: ${validatorPermId}`);

  // Wait for validator permission to become effective (permissions are created with effectiveFrom 10 seconds in future)
  console.log(`  ⏳ Waiting for validator permission to become effective (permissions require effective_from to be in the future)...`);
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
    console.log(`  ✓ Validator permission should now be effective`);
  } finally {
    queryClient.disconnect();
  }

  // Start Permission VP - this creates a permission in PENDING state
  const applicantDid = generateUniqueDID();
  const startVPMsg = {
    typeUrl: typeUrls.MsgStartPermissionVP,
    value: MsgStartPermissionVP.fromPartial({
      creator: account.address,
      type: PermissionType.ISSUER,
      validatorPermId: validatorPermId,
      country: "US",
      did: applicantDid,
    }),
  };

  const startVPFee = await calculateFeeWithSimulation(
    client,
    account.address,
    [startVPMsg],
    "Starting Permission VP for cancel test"
  );
  // Use retry logic for consistency (matches frontend pattern)
  const startVPResult = await signAndBroadcastWithRetry(
    client,
    account.address,
    [startVPMsg],
    startVPFee,
    "Starting Permission VP for cancel test"
  );

  if (startVPResult.code !== 0) {
    console.log("  ❌ Failed to start Permission VP");
    console.log(`  Error: ${startVPResult.rawLog}`);
    process.exit(1);
  }

  // Extract permission ID from events
  let permId: number | undefined;
  const events = startVPResult.events || [];
  for (const event of events) {
    if (event.type === "start_permission_vp" || event.type === "verana.perm.v1.EventStartPermissionVP") {
      for (const attr of event.attributes) {
        if (attr.key === "permission_id" || attr.key === "id") {
          permId = parseInt(attr.value, 10);
          if (!isNaN(permId)) {
            console.log(`  ✓ Started Permission VP - Permission ID: ${permId} (PENDING state)`);
            break;
          }
        }
      }
      if (permId) break;
    }
  }

  if (!permId) {
    console.log("  ❌ Could not extract Permission ID from StartPermissionVP events");
    process.exit(1);
  }

  // Wait for sequence to propagate after starting VP
  await waitForSequencePropagation(client, account.address);

  // Save prerequisites for next step
  saveJourneyResult("cancel-perm-vp-prereqs", {
    accountAddress: account.address,
    permissionId: permId.toString(),
    schemaId: schemaId.toString(),
    did: did,
    validatorPermId: validatorPermId.toString(),
    applicantDid: applicantDid,
  });

  console.log();
  console.log("=".repeat(60));
  console.log("✅ SUCCESS! Prerequisites setup completed!");
  console.log("=".repeat(60));
  console.log(`  Account address: ${account.address}`);
  console.log(`  Permission ID: ${permId} (PENDING state)`);
  console.log(`  Schema ID: ${schemaId}`);
  console.log(`  Validator Permission ID: ${validatorPermId}`);
  console.log();
  console.log("  💾 Results saved to journey_results/cancel-perm-vp-prereqs.json");
  console.log("  ➡️  Run next step: npm run test:cancel-perm-vp");
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
