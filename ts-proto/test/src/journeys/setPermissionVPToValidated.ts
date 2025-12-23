/**
 * Journey: Set Permission VP To Validated
 *
 * This script demonstrates how to set a Permission Validation Process to validated state using the
 * TypeScript client and the generated protobuf types.
 *
 * Usage:
 *   PERM_ID=1 npm run test:set-perm-vp-validated
 *   # Or let it create a permission and start VP first, then validate it
 *   npm run test:set-perm-vp-validated
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
import { MsgSetPermissionVPToValidated, MsgStartPermissionVP } from "../../../src/codec/verana/perm/v1/tx";
import { PermissionType } from "../../../src/codec/verana/perm/v1/types";
import { createSchemaForTest, createRootPermissionForTest } from "../helpers/permissionHelpers";
import { generateUniqueDID } from "../helpers/client";

const TEST_MNEMONIC =
  process.env.MNEMONIC ||
  "pink glory help gown abstract eight nice crazy forward ketchup skill cheese";

async function main() {
  console.log("=".repeat(60));
  console.log("Journey: Set Permission VP To Validated (TypeScript Client)");
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
    console.log("Step 4: Creating schema, validator permission, and starting VP first...");
    const { schemaId, did } = await createSchemaForTest(client, account.address);
    const validatorPermId = await createRootPermissionForTest(client, account.address, schemaId, did);
    
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
    
    // Start a VP to get a permission in pending state
    const startVPMsg = {
      typeUrl: typeUrls.MsgStartPermissionVP,
      value: MsgStartPermissionVP.fromPartial({
        creator: account.address,
        type: PermissionType.ISSUER,
        validatorPermId: validatorPermId,
        country: "US",
        did: generateUniqueDID(),
      }),
    };
    const startVPFee = await calculateFeeWithSimulation(
      client,
      account.address,
      [startVPMsg],
      "Starting VP for validation test"
    );
    const startVPResult = await client.signAndBroadcast(
      account.address,
      [startVPMsg],
      startVPFee,
      "Starting VP for validation test"
    );
    if (startVPResult.code !== 0) {
      console.log("  ❌ Failed to start VP");
      console.log(`  Error: ${startVPResult.rawLog}`);
      process.exit(1);
    }
    
    // Extract permission ID from events
    const events = startVPResult.events || [];
    for (const event of events) {
      if (event.type === "start_permission_vp" || event.type === "verana.perm.v1.EventStartPermissionVP") {
        for (const attr of event.attributes) {
          if (attr.key === "permission_id" || attr.key === "id") {
            permId = parseInt(attr.value, 10);
            if (!isNaN(permId)) {
              console.log(`  ✓ Started VP, Permission ID: ${permId}`);
              break;
            }
          }
        }
        if (permId) break;
      }
    }
    
    if (!permId) {
      console.log("  ❌ Could not extract Permission ID from VP start events");
      process.exit(1);
    }
  }

  if (!permId) {
    console.log("  ❌ Permission ID is required");
    process.exit(1);
  }

  console.log();

  console.log("Step 5: Setting Permission VP To Validated transaction...");
  const effectiveUntil = new Date(Date.now() + 360 * 24 * 60 * 60 * 1000); // 360 days from now
  const validationFees = 1000;
  const issuanceFees = 1000;
  const verificationFees = 1000;
  const country = "US";
  const vpSummaryDigestSri = "sha384-ExampleVPSummaryDigest123456789012345678901234567890123456789012345678901234567890";

  const msg = {
    typeUrl: typeUrls.MsgSetPermissionVPToValidated,
    value: MsgSetPermissionVPToValidated.fromPartial({
      creator: account.address,
      id: permId,
      effectiveUntil: effectiveUntil,
      validationFees: validationFees,
      issuanceFees: issuanceFees,
      verificationFees: verificationFees,
      country: country,
      vpSummaryDigestSri: vpSummaryDigestSri,
    }),
  };
  console.log(`    - Permission ID: ${permId}`);
  console.log(`    - Effective Until: ${effectiveUntil.toISOString()}`);
  console.log(`    - Validation Fees: ${validationFees}`);
  console.log(`    - Issuance Fees: ${issuanceFees}`);
  console.log(`    - Verification Fees: ${verificationFees}`);
  console.log(`    - Country: ${country}`);
  console.log(`    - VP Summary Digest SRI: ${vpSummaryDigestSri}`);
  console.log();

  console.log("Step 6: Signing and broadcasting transaction...");
  try {
    const fee = await calculateFeeWithSimulation(
      client,
      account.address,
      [msg],
      "Setting Permission VP To Validated via TypeScript client"
    );
    console.log(`  Calculated gas: ${fee.gas}, fee: ${fee.amount[0].amount}${fee.amount[0].denom}`);

    const result = await client.signAndBroadcast(
      account.address,
      [msg],
      fee,
      "Setting Permission VP To Validated via TypeScript client"
    );

    console.log();
    if (result.code === 0) {
      console.log("✅ SUCCESS! Permission VP set to validated successfully!");
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

