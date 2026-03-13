/**
 * Journey: PERM Start Permission VP
 *
 * Starts a Validation Process (VP) for an ISSUER permission.
 * Uses the GRANTOR_VALIDATION schema from permCreateRootPermission.
 *
 * Requires: test:perm-create-root must be run first.
 *
 * Usage:
 *   npm run test:perm-start-vp
 */

import {
  createDirectAccountFromMnemonic,
  createSigningClient,
  getAccountInfo,
  calculateFeeWithSimulation,
  signAndBroadcastWithRetry,
  waitForPermissionToBecomeEffective,
  createQueryClient,
  config,
} from "../helpers/client";
import { typeUrls } from "../helpers/registry";
import { MsgStartPermissionVP } from "../../../src/codec/verana/perm/v1/tx";
import { PermissionType, OptionalUInt64 } from "../../../src/codec/verana/perm/v1/types";
import { getPermAuthzSetup, getPermRootSetup, savePermVPSetup } from "../helpers/journeyResults";
import { extractIdFromEvents } from "../helpers/permissionHelpers";

const COOLUSER_MNEMONIC =
  (process.env.MNEMONIC && process.env.MNEMONIC.trim()) ||
  "pink glory help gown abstract eight nice crazy forward ketchup skill cheese";

const OPERATOR_INDEX = 15;

async function main() {
  console.log("=".repeat(60));
  console.log("Journey: PERM Start Permission VP");
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
  console.log(`  Schema ID: ${rootSetup.schemaId}`);
  console.log(`  Root Permission ID: ${rootSetup.rootPermId}`);
  console.log(`  DID: ${rootSetup.did}`);
  console.log();

  // Step 2: Connect operator
  console.log("Step 2: Setting up operator wallet...");
  const wallet = await createDirectAccountFromMnemonic(COOLUSER_MNEMONIC, OPERATOR_INDEX);
  const account = await getAccountInfo(wallet);
  const client = await createSigningClient(wallet);
  console.log(`  Connected as ${account.address}`);
  console.log();

  try {
    // Step 3: Wait for root permission to become effective
    console.log("Step 3: Waiting for root permission to become effective...");
    if (rootSetup.effectiveFrom) {
      const queryClient = await createQueryClient();
      try {
        await waitForPermissionToBecomeEffective(queryClient, rootSetup.effectiveFrom, 60000);
      } finally {
        queryClient.disconnect();
      }
    }
    console.log("  Root permission is now effective");
    console.log();

    // Step 4: Start VP for ISSUER type
    console.log("Step 4: Starting Permission VP (MsgStartPermissionVP)...");
    const msg = {
      typeUrl: typeUrls.MsgStartPermissionVP,
      value: MsgStartPermissionVP.fromPartial({
        authority: authzSetup.authorityAddress,
        operator: authzSetup.operatorAddress,
        type: PermissionType.ISSUER,
        validatorPermId: rootSetup.rootPermId,
        did: rootSetup.did,
        validationFees: OptionalUInt64.fromPartial({ value: 5 }),
        issuanceFees: OptionalUInt64.fromPartial({ value: 5 }),
        verificationFees: OptionalUInt64.fromPartial({ value: 5 }),
        vsOperator: "",
        vsOperatorAuthzEnabled: false,
      }),
    };

    const fee = await calculateFeeWithSimulation(client, account.address, [msg], "Starting VP");
    const result = await signAndBroadcastWithRetry(client, account.address, [msg], fee, "Starting VP");

    if (result.code !== 0) {
      throw new Error(`Failed to start VP: ${result.rawLog}`);
    }

    const vpPermId = extractIdFromEvents(result.events || [], "start_permission_vp", ["permission_id", "id"]);

    console.log();
    console.log("SUCCESS! Permission VP started!");
    console.log(`  Tx Hash: ${result.transactionHash}`);
    console.log(`  Block: ${result.height}`);
    console.log(`  Gas: ${result.gasUsed}/${result.gasWanted}`);
    console.log(`  VP Permission ID: ${vpPermId}`);

    if (vpPermId) {
      savePermVPSetup(vpPermId, rootSetup.schemaId);
      console.log("  Saved perm-vp-setup");
    }
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
