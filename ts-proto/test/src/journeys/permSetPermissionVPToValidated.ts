/**
 * Journey: PERM Set Permission VP To Validated
 *
 * Validates a pending VP from the Start VP journey.
 *
 * Requires: test:perm-start-vp must be run first.
 *
 * Usage:
 *   npm run test:perm-validate-vp
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
import { MsgSetPermissionVPToValidated } from "../../../src/codec/verana/perm/v1/tx";
import { getPermAuthzSetup, getPermVPSetup, saveJourneyResult } from "../helpers/journeyResults";

const COOLUSER_MNEMONIC =
  (process.env.MNEMONIC && process.env.MNEMONIC.trim()) ||
  "pink glory help gown abstract eight nice crazy forward ketchup skill cheese";

const OPERATOR_INDEX = 15;

async function main() {
  console.log("=".repeat(60));
  console.log("Journey: PERM Set Permission VP To Validated");
  console.log("=".repeat(60));
  console.log();

  // Step 1: Load setup
  console.log("Step 1: Loading PERM setup...");
  const authzSetup = getPermAuthzSetup();
  const vpSetup = getPermVPSetup();
  if (!authzSetup || !vpSetup) {
    console.log("  Missing setup. Run test:de-grant-perm-auth and test:perm-start-vp first.");
    process.exit(1);
  }
  console.log(`  Authority: ${authzSetup.authorityAddress}`);
  console.log(`  VP Permission ID: ${vpSetup.vpPermId}`);
  console.log();

  // Step 2: Connect operator
  console.log("Step 2: Setting up operator wallet...");
  const wallet = await createAccountFromMnemonic(COOLUSER_MNEMONIC, OPERATOR_INDEX);
  const account = await getAccountInfo(wallet);
  const client = await createSigningClient(wallet);
  console.log(`  Connected as ${account.address}`);
  console.log();

  try {
    // Step 3: Validate the VP
    console.log("Step 3: Setting VP to validated (MsgSetPermissionVPToValidated)...");
    const effectiveUntil = new Date(Date.now() + 300 * 24 * 60 * 60 * 1000); // 300 days (must be <= validator_perm.effective_until)

    const msg = {
      typeUrl: typeUrls.MsgSetPermissionVPToValidated,
      value: MsgSetPermissionVPToValidated.fromPartial({
        authority: authzSetup.authorityAddress,
        operator: authzSetup.operatorAddress,
        id: vpSetup.vpPermId,
        effectiveUntil,
        validationFees: 5,
        issuanceFees: 5,
        verificationFees: 5,
        vpSummaryDigestSri: "sha384-validationSummaryDigest123456",
        issuanceFeeDiscount: 0,
        verificationFeeDiscount: 0,
      }),
    };

    const fee = await calculateFeeWithSimulation(client, account.address, [msg], "Validating VP");
    const result = await signAndBroadcastWithRetry(client, account.address, [msg], fee, "Validating VP");

    if (result.code !== 0) {
      throw new Error(`Failed to validate VP: ${result.rawLog}`);
    }

    console.log();
    console.log("SUCCESS! VP validated!");
    console.log(`  Tx Hash: ${result.transactionHash}`);
    console.log(`  Block: ${result.height}`);
    console.log(`  Gas: ${result.gasUsed}/${result.gasWanted}`);
    console.log(`  Validated Permission ID: ${vpSetup.vpPermId}`);
    console.log(`  Effective Until: ${effectiveUntil.toISOString()}`);

    saveJourneyResult("perm-validated-setup", {
      permissionId: vpSetup.vpPermId.toString(),
      schemaId: vpSetup.schemaId.toString(),
    });
    console.log("  Saved perm-validated-setup");
  } catch (error: any) {
    console.log("ERROR!");
    console.error(error);
    process.exit(1);
  } finally {
    client.disconnect();
  }

  console.log();
  console.log("=".repeat(60));
}

main().catch((error: any) => {
  console.error("\nFatal error:", error.message || error);
  process.exit(1);
});
