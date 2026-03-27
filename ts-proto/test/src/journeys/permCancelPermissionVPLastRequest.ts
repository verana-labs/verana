/**
 * Journey: PERM Renew + Cancel Permission VP
 *
 * Renews a validated VP (MsgRenewPermissionVP), then cancels the
 * pending request (MsgCancelPermissionVPLastRequest).
 * Tests both message types in one journey.
 *
 * Requires: test:perm-validate-vp must be run first.
 *
 * Usage:
 *   npm run test:perm-cancel-vp
 */

import {
  createDirectAccountFromMnemonic,
  createSigningClient,
  getAccountInfo,
  calculateFeeWithSimulation,
  signAndBroadcastWithRetry,
  config,
} from "../helpers/client";
import { typeUrls } from "../helpers/registry";
import { MsgRenewPermissionVP, MsgCancelPermissionVPLastRequest } from "../../../src/codec/verana/perm/v1/tx";
import { getPermAuthzSetup, loadJourneyResult } from "../helpers/journeyResults";

const COOLUSER_MNEMONIC =
  (process.env.MNEMONIC && process.env.MNEMONIC.trim()) ||
  "pink glory help gown abstract eight nice crazy forward ketchup skill cheese";

const OPERATOR_INDEX = 15;

async function main() {
  console.log("=".repeat(60));
  console.log("Journey: PERM Renew + Cancel Permission VP");
  console.log("=".repeat(60));
  console.log();

  // Step 1: Load setup
  console.log("Step 1: Loading PERM setup...");
  const authzSetup = getPermAuthzSetup();
  const validatedSetup = loadJourneyResult("perm-validated-setup");
  if (!authzSetup || !validatedSetup?.permissionId) {
    console.log("  Missing setup. Run test:perm-validate-vp first.");
    process.exit(1);
  }
  const permId = parseInt(validatedSetup.permissionId, 10);
  console.log(`  Authority: ${authzSetup.authorityAddress}`);
  console.log(`  Validated Permission ID: ${permId}`);
  console.log();

  // Step 2: Connect operator
  console.log("Step 2: Setting up operator wallet...");
  const wallet = await createDirectAccountFromMnemonic(COOLUSER_MNEMONIC, OPERATOR_INDEX);
  const account = await getAccountInfo(wallet);
  const client = await createSigningClient(wallet);
  console.log(`  Connected as ${account.address}`);
  console.log();

  try {
    // Step 3: Renew the validated VP
    console.log("Step 3: Renewing validated VP (MsgRenewPermissionVP)...");
    const renewMsg = {
      typeUrl: typeUrls.MsgRenewPermissionVP,
      value: MsgRenewPermissionVP.fromPartial({
        authority: authzSetup.authorityAddress,
        operator: authzSetup.operatorAddress,
        id: permId,
      }),
    };

    const renewFee = await calculateFeeWithSimulation(client, account.address, [renewMsg], "Renewing VP");
    const renewResult = await signAndBroadcastWithRetry(client, account.address, [renewMsg], renewFee, "Renewing VP");

    if (renewResult.code !== 0) {
      throw new Error(`Failed to renew VP: ${renewResult.rawLog}`);
    }

    console.log("  VP renewed successfully (now in PENDING state)");
    console.log(`  Tx Hash: ${renewResult.transactionHash}`);
    console.log();

    // Step 4: Cancel the pending VP request
    console.log("Step 4: Cancelling VP last request (MsgCancelPermissionVPLastRequest)...");
    const cancelMsg = {
      typeUrl: typeUrls.MsgCancelPermissionVPLastRequest,
      value: MsgCancelPermissionVPLastRequest.fromPartial({
        authority: authzSetup.authorityAddress,
        operator: authzSetup.operatorAddress,
        id: permId,
      }),
    };

    const cancelFee = await calculateFeeWithSimulation(client, account.address, [cancelMsg], "Cancelling VP");
    const cancelResult = await signAndBroadcastWithRetry(client, account.address, [cancelMsg], cancelFee, "Cancelling VP");

    if (cancelResult.code !== 0) {
      throw new Error(`Failed to cancel VP: ${cancelResult.rawLog}`);
    }

    console.log();
    console.log("SUCCESS! VP renewed then cancelled!");
    console.log(`  Renew Tx: ${renewResult.transactionHash}`);
    console.log(`  Cancel Tx: ${cancelResult.transactionHash}`);
    console.log(`  Block: ${cancelResult.height}`);
    console.log(`  Gas: ${cancelResult.gasUsed}/${cancelResult.gasWanted}`);
    console.log(`  Permission ID: ${permId} (now TERMINATED)`);
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
