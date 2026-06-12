/**
 * Journey: PP Trigger Resolver (MsgTriggerResolver, MOD-PP-MSG-15)
 *
 * Self-contained: creates fresh prerequisites (EC + CS + root), a dedicated
 * VALIDATED child participant whose validator ancestor is the root, then
 * broadcasts MsgTriggerResolver and asserts the trigger_resolver event.
 * Authorization resolves via Path 2 (the child's active ancestor validator +
 * AUTHZ-CHECK-1). Signs in SIGN_MODE_LEGACY_AMINO_JSON.
 *
 * Requires: test:de-grant-perm-auth first (which now also grants
 * MsgTriggerResolver to the operator).
 *
 * Usage:
 *   npm run test:perm-trigger-resolver
 */

import {
  createAccountFromMnemonic,
  createSigningClient,
  getAccountInfo,
  calculateFeeWithSimulation,
  signAndBroadcastWithRetry,
  waitForPermissionToBecomeEffective,
  createQueryClient,
  generateUniqueDID,
} from "../helpers/client";
import { typeUrls } from "../helpers/registry";
import { MsgTriggerResolver } from "../../../src/codec/verana/pp/v1/tx";
import { IssuerOnboardingMode } from "../../../src/codec/verana/cs/v1/types";
import { getPermAuthzSetup } from "../helpers/journeyResults";
import { createPermPrerequisites, createValidatedPermission } from "../helpers/permissionHelpers";

const COOLUSER_MNEMONIC =
  (process.env.MNEMONIC && process.env.MNEMONIC.trim()) ||
  "pink glory help gown abstract eight nice crazy forward ketchup skill cheese";

const OPERATOR_INDEX = 15;

async function main() {
  console.log("=".repeat(60));
  console.log("Journey: PP Trigger Resolver (MOD-PP-MSG-15)");
  console.log("=".repeat(60));
  console.log();

  console.log("Step 1: Loading PERM authz setup...");
  const setup = getPermAuthzSetup();
  if (!setup) {
    console.log("  Missing setup. Run test:de-grant-perm-auth first.");
    process.exit(1);
  }
  console.log(`  Authority: ${setup.authorityAddress}`);
  console.log();

  console.log("Step 2: Setting up operator wallet (legacy amino)...");
  const wallet = await createAccountFromMnemonic(COOLUSER_MNEMONIC, OPERATOR_INDEX);
  const account = await getAccountInfo(wallet);
  const client = await createSigningClient(wallet);
  console.log(`  Connected as ${account.address}`);
  console.log();

  try {
    // Step 3: Fresh prerequisites (EC + CS + Root). Root effective_from = now+10s.
    console.log("Step 3: Creating fresh prerequisites (EC + CS + Root)...");
    const { schemaId, rootPermId, effectiveFrom } = await createPermPrerequisites(
      client,
      setup.authorityAddress,
      setup.operatorAddress,
      IssuerOnboardingMode.ISSUER_ONBOARDING_MODE_ECOSYSTEM_VALIDATION_PROCESS,
    );
    console.log(`  Root Permission ID: ${rootPermId}`);

    // Step 4: Wait for the root to become active (it is the Path 2 ancestor and
    // also validates the child).
    console.log("Step 4: Waiting for root permission to become effective...");
    const queryClient = await createQueryClient();
    try {
      await waitForPermissionToBecomeEffective(queryClient, effectiveFrom, 60000);
    } finally {
      queryClient.disconnect();
    }
    console.log("  Root permission is now effective");

    // Step 5: Create a VALIDATED child participant under the root.
    console.log("Step 5: Creating a VALIDATED child participant...");
    const participantId = await createValidatedPermission(
      client,
      setup.authorityAddress,
      setup.operatorAddress,
      schemaId,
      rootPermId,
      generateUniqueDID(),
    );
    console.log(`  Validated child participant ID: ${participantId}`);
    console.log();

    // Step 6: Trigger the resolver on the child (authorized via Path 2).
    console.log("Step 6: Broadcasting MsgTriggerResolver...");
    const msg = {
      typeUrl: typeUrls.MsgTriggerResolver,
      value: MsgTriggerResolver.fromPartial({
        corporation: setup.authorityAddress,
        operator: setup.operatorAddress,
        id: participantId,
      }),
    };

    const fee = await calculateFeeWithSimulation(client, account.address, [msg], "Triggering resolver");
    const result = await signAndBroadcastWithRetry(client, account.address, [msg], fee, "Triggering resolver");

    if (result.code !== 0) {
      throw new Error(`Failed to trigger resolver: ${result.rawLog}`);
    }

    // Step 7: Assert the trigger_resolver event with participant_id == childId.
    console.log("Step 7: Asserting trigger_resolver event...");
    let found = false;
    for (const event of (result.events || [])) {
      if (event.type !== "trigger_resolver") continue;
      for (const attr of event.attributes) {
        const key = String(attr.key);
        const value = String(attr.value).replace(/"/g, "");
        if (key === "participant_id" && value === String(participantId)) {
          found = true;
        }
      }
    }
    if (!found) {
      throw new Error(`trigger_resolver event with participant_id=${participantId} not found`);
    }

    console.log();
    console.log("SUCCESS! Trigger resolver event emitted!");
    console.log(`  Tx Hash: ${result.transactionHash}`);
    console.log(`  Block: ${result.height}`);
    console.log(`  Participant ID: ${participantId}`);
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
