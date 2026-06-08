/**
 * Journey: PERM Renew Participant OP (RenewParticipantOP)
 *
 * Self-contained: creates fresh prerequisites (EC + CS + root), a dedicated
 * VALIDATED participant, then renews it. Independent of the shared VP so it
 * does not collide (overlap) with the validate/cancel journeys. Signs in
 * SIGN_MODE_LEGACY_AMINO_JSON.
 *
 * Requires: test:de-grant-perm-auth first.
 *
 * Usage:
 *   npm run test:perm-renew
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
import { MsgRenewParticipantOP } from "../../../src/codec/verana/pp/v1/tx";
import { IssuerOnboardingMode } from "../../../src/codec/verana/cs/v1/types";
import { getPermAuthzSetup } from "../helpers/journeyResults";
import { createPermPrerequisites, createValidatedPermission } from "../helpers/permissionHelpers";

const COOLUSER_MNEMONIC =
  (process.env.MNEMONIC && process.env.MNEMONIC.trim()) ||
  "pink glory help gown abstract eight nice crazy forward ketchup skill cheese";

const OPERATOR_INDEX = 15;

async function main() {
  console.log("=".repeat(60));
  console.log("Journey: PERM Renew Participant OP");
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
    // Step 3: Fresh prerequisites (avoids overlap with the shared VP).
    console.log("Step 3: Creating fresh prerequisites (EC + CS + Root)...");
    const { schemaId, rootPermId, effectiveFrom } = await createPermPrerequisites(
      client,
      setup.authorityAddress,
      setup.operatorAddress,
      IssuerOnboardingMode.ISSUER_ONBOARDING_MODE_ECOSYSTEM_VALIDATION_PROCESS,
    );
    console.log(`  Root Permission ID: ${rootPermId}`);

    console.log("Step 4: Waiting for root permission to become effective...");
    const queryClient = await createQueryClient();
    try {
      await waitForPermissionToBecomeEffective(queryClient, effectiveFrom, 60000);
    } finally {
      queryClient.disconnect();
    }
    console.log("  Root permission is now effective");

    // Step 5: Create a dedicated VALIDATED participant to renew.
    console.log("Step 5: Creating a VALIDATED participant to renew...");
    const participantId = await createValidatedPermission(
      client,
      setup.authorityAddress,
      setup.operatorAddress,
      schemaId,
      rootPermId,
      generateUniqueDID(),
    );
    console.log(`  Validated participant ID: ${participantId}`);
    console.log();

    // Step 6: Renew the validation process.
    console.log("Step 6: Renewing participant OP (MsgRenewParticipantOP)...");
    const msg = {
      typeUrl: typeUrls.MsgRenewParticipantOP,
      value: MsgRenewParticipantOP.fromPartial({
        corporation: setup.authorityAddress,
        operator: setup.operatorAddress,
        id: participantId,
      }),
    };

    const fee = await calculateFeeWithSimulation(client, account.address, [msg], "Renewing participant OP");
    const result = await signAndBroadcastWithRetry(client, account.address, [msg], fee, "Renewing participant OP");

    if (result.code !== 0) {
      throw new Error(`Failed to renew participant OP: ${result.rawLog}`);
    }

    console.log();
    console.log("SUCCESS! Participant OP renewed!");
    console.log(`  Tx Hash: ${result.transactionHash}`);
    console.log(`  Block: ${result.height}`);
    console.log(`  Renewed Participant ID: ${participantId}`);
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
