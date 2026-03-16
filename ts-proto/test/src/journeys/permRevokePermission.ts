/**
 * Journey: PERM Revoke Permission
 *
 * Creates a fresh root permission and revokes it.
 * Uses a fresh CS to avoid conflicts with other journeys.
 *
 * Requires: test:de-grant-perm-auth must be run first.
 *
 * Usage:
 *   npm run test:perm-revoke
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
import { MsgRevokePermission } from "../../../src/codec/verana/perm/v1/tx";
import { CredentialSchemaPermManagementMode } from "../../../src/codec/verana/cs/v1/types";
import { getPermAuthzSetup } from "../helpers/journeyResults";
import { createPermPrerequisites } from "../helpers/permissionHelpers";

const COOLUSER_MNEMONIC =
  (process.env.MNEMONIC && process.env.MNEMONIC.trim()) ||
  "pink glory help gown abstract eight nice crazy forward ketchup skill cheese";

const OPERATOR_INDEX = 15;

async function main() {
  console.log("=".repeat(60));
  console.log("Journey: PERM Revoke Permission");
  console.log("=".repeat(60));
  console.log();

  // Step 1: Load setup
  console.log("Step 1: Loading PERM authz setup...");
  const setup = getPermAuthzSetup();
  if (!setup) {
    console.log("  No PERM authz setup found. Run test:de-grant-perm-auth first.");
    process.exit(1);
  }
  console.log(`  Authority: ${setup.authorityAddress}`);
  console.log(`  Operator:  ${setup.operatorAddress}`);
  console.log();

  // Step 2: Connect operator
  console.log("Step 2: Setting up operator wallet...");
  const wallet = await createAccountFromMnemonic(COOLUSER_MNEMONIC, OPERATOR_INDEX);
  const account = await getAccountInfo(wallet);
  const client = await createSigningClient(wallet);
  console.log(`  Connected as ${account.address}`);
  console.log();

  try {
    // Step 3: Create fresh prerequisites (TR + CS + Root Permission)
    console.log("Step 3: Creating fresh prerequisites for revoke test...");
    const { rootPermId, effectiveFrom } = await createPermPrerequisites(
      client,
      setup.authorityAddress,
      setup.operatorAddress,
      CredentialSchemaPermManagementMode.GRANTOR_VALIDATION,
    );
    console.log(`  Root Permission ID: ${rootPermId}`);
    console.log();

    // Step 4: Wait for root permission to become effective
    console.log("Step 4: Waiting for root permission to become effective...");
    const queryClient = await createQueryClient();
    try {
      await waitForPermissionToBecomeEffective(queryClient, effectiveFrom, 60000);
    } finally {
      queryClient.disconnect();
    }
    console.log("  Root permission is now effective");
    console.log();

    // Step 5: Revoke the root permission
    console.log("Step 5: Revoking root permission (MsgRevokePermission)...");
    const msg = {
      typeUrl: typeUrls.MsgRevokePermission,
      value: MsgRevokePermission.fromPartial({
        authority: setup.authorityAddress,
        operator: setup.operatorAddress,
        id: rootPermId,
      }),
    };

    const fee = await calculateFeeWithSimulation(client, account.address, [msg], "Revoking permission");
    const result = await signAndBroadcastWithRetry(client, account.address, [msg], fee, "Revoking permission");

    if (result.code !== 0) {
      throw new Error(`Failed to revoke permission: ${result.rawLog}`);
    }

    console.log();
    console.log("SUCCESS! Permission revoked!");
    console.log(`  Tx Hash: ${result.transactionHash}`);
    console.log(`  Block: ${result.height}`);
    console.log(`  Gas: ${result.gasUsed}/${result.gasWanted}`);
    console.log(`  Revoked Permission ID: ${rootPermId}`);
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
