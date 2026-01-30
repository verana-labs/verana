/**
 * Journey: Set Permission VP To Validated
 *
 * Step 2 of 2 for Set Permission VP To Validated.
 * Loads prerequisites from Step 1, sets the Permission VP to validated state.
 *
 * Usage:
 *   # Using prerequisites from Step 1 (recommended)
 *   npm run test:set-perm-vp-validated
 *
 *   # Or provide specific Permission ID
 *   PERM_ID=1 npm run test:set-perm-vp-validated
 */

import {
  createWallet,
  createSigningClient,
  getAccountInfo,
  calculateFeeWithSimulation,
  signAndBroadcastWithRetry,
  config,
} from "../helpers/client";
import { typeUrls } from "../helpers/registry";
import { MsgSetPermissionVPToValidated } from "../../../src/codec/verana/perm/v1/tx";
import { loadJourneyResult } from "../helpers/journeyResults";

const TEST_MNEMONIC =
  process.env.MNEMONIC ||
  "pink glory help gown abstract eight nice crazy forward ketchup skill cheese";

async function main() {
  console.log("=".repeat(60));
  console.log("Journey: Set Permission VP To Validated (Step 2/2)");
  console.log("=".repeat(60));
  console.log();

  // Load prerequisites from Step 1
  console.log("Loading prerequisites from Step 1...");
  const prereqs = loadJourneyResult("set-perm-vp-validated-prereqs");

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

  let permId: number | undefined;
  if (prereqs?.permissionId) {
    permId = parseInt(prereqs.permissionId, 10);
    console.log(`  ✓ Loaded prerequisites:`);
    console.log(`    - Permission ID: ${permId} (PENDING state)`);
    if (prereqs.schemaId) {
      console.log(`    - Schema ID: ${prereqs.schemaId}`);
    }
    if (prereqs.validatorPermId) {
      console.log(`    - Validator Permission ID: ${prereqs.validatorPermId}`);
    }
  } else if (process.env.PERM_ID) {
    permId = parseInt(process.env.PERM_ID, 10);
    if (isNaN(permId)) {
      console.log("  ❌ Invalid PERM_ID provided");
      process.exit(1);
    }
    console.log(`  Using provided Permission ID: ${permId}`);
  } else {
    console.log("  ❌ Permission ID not found. Run Step 1 first:");
    console.log("     npm run test:setup-set-perm-vp-validated-prereqs");
    console.log("  Or provide PERM_ID via environment variable.");
    process.exit(1);
  }

  if (!permId) {
    console.log("  ❌ Permission ID is required");
    process.exit(1);
  }

  console.log();

  console.log("Step 1: Setting Permission VP To Validated transaction...");
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

  console.log("Step 2: Signing and broadcasting transaction...");
  try {
    const fee = await calculateFeeWithSimulation(
      client,
      account.address,
      [msg],
      "Setting Permission VP To Validated via TypeScript client"
    );
    console.log(`  Calculated gas: ${fee.gas}, fee: ${fee.amount[0].amount}${fee.amount[0].denom}`);

    // Use retry logic for consistency (matches frontend pattern)
    const result = await signAndBroadcastWithRetry(
      client,
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

