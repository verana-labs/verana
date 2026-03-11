/**
 * Journey: PERM Adjust Permission
 *
 * Adjusts the root permission's effective_until date.
 * Waits for the permission to become effective first.
 *
 * Requires: test:perm-create-root must be run first.
 *
 * Usage:
 *   npm run test:perm-adjust
 */

import {
  createAccountFromMnemonic,
  createSigningClient,
  getAccountInfo,
  calculateFeeWithSimulation,
  signAndBroadcastWithRetry,
  waitForPermissionToBecomeEffective,
  createQueryClient,
  config,
} from "../helpers/client";
import { typeUrls } from "../helpers/registry";
import { MsgAdjustPermission } from "../../../src/codec/verana/perm/v1/tx";
import { getPermAuthzSetup, getPermRootSetup } from "../helpers/journeyResults";

const COOLUSER_MNEMONIC =
  (process.env.MNEMONIC && process.env.MNEMONIC.trim()) ||
  "pink glory help gown abstract eight nice crazy forward ketchup skill cheese";

const OPERATOR_INDEX = 15;

async function main() {
  console.log("=".repeat(60));
  console.log("Journey: PERM Adjust Permission");
  console.log("=".repeat(60));
  console.log();

  // Step 1: Load setup
  console.log("Step 1: Loading PERM setup...");
  const authzSetup = getPermAuthzSetup();
  const rootSetup = getPermRootSetup();
  if (!authzSetup || !rootSetup) {
    console.log("  Missing setup. Run test:de-grant-perm-auth and test:perm-create-root first.");
    process.exit(1);
  }
  console.log(`  Authority: ${authzSetup.authorityAddress}`);
  console.log(`  Root Permission ID: ${rootSetup.rootPermId}`);
  console.log();

  // Step 2: Connect operator
  console.log("Step 2: Setting up operator wallet...");
  const wallet = await createAccountFromMnemonic(COOLUSER_MNEMONIC, OPERATOR_INDEX);
  const account = await getAccountInfo(wallet);
  const client = await createSigningClient(wallet);
  console.log(`  Connected as ${account.address}`);
  console.log();

  try {
    // Step 3: Wait for root permission to be effective
    console.log("Step 3: Waiting for root permission to become effective...");
    // Root perm was created with effectiveFrom = now + 10s, so we need to wait
    const effectiveFrom = new Date(Date.now() - 5000); // It may already be effective
    const queryClient = await createQueryClient();
    try {
      // Wait up to 30s — the root perm should already be or soon become effective
      await waitForPermissionToBecomeEffective(queryClient, effectiveFrom, 30000);
    } catch {
      // If it fails, wait a fixed amount and try anyway
      console.log("  Warning: Could not confirm effectiveness, proceeding anyway...");
    } finally {
      queryClient.disconnect();
    }
    console.log("  Root permission should be effective");
    console.log();

    // Step 4: Adjust permission — extend effective_until
    console.log("Step 4: Adjusting root permission (MsgAdjustPermission)...");
    const newEffectiveUntil = new Date(Date.now() + 720 * 24 * 60 * 60 * 1000); // 720 days from now

    const msg = {
      typeUrl: typeUrls.MsgAdjustPermission,
      value: MsgAdjustPermission.fromPartial({
        authority: authzSetup.authorityAddress,
        operator: authzSetup.operatorAddress,
        id: rootSetup.rootPermId,
        effectiveUntil: newEffectiveUntil,
      }),
    };

    const fee = await calculateFeeWithSimulation(client, account.address, [msg], "Adjusting permission");
    const result = await signAndBroadcastWithRetry(client, account.address, [msg], fee, "Adjusting permission");

    if (result.code !== 0) {
      throw new Error(`Failed to adjust permission: ${result.rawLog}`);
    }

    console.log();
    console.log("SUCCESS! Permission adjusted!");
    console.log(`  Tx Hash: ${result.transactionHash}`);
    console.log(`  Block: ${result.height}`);
    console.log(`  Gas: ${result.gasUsed}/${result.gasWanted}`);
    console.log(`  New effective_until: ${newEffectiveUntil.toISOString()}`);
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
