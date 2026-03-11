/**
 * Journey: PERM Create Permission (Self-Create, OPEN mode)
 *
 * Creates a new OPEN-mode CS, root permission, waits for effectiveness,
 * then creates a child ISSUER permission via MsgCreatePermission.
 *
 * Requires: test:de-grant-perm-auth must be run first.
 *
 * Usage:
 *   npm run test:perm-create
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
import { MsgCreatePermission } from "../../../src/codec/verana/perm/v1/tx";
import { PermissionType } from "../../../src/codec/verana/perm/v1/types";
import { CredentialSchemaPermManagementMode } from "../../../src/codec/verana/cs/v1/types";
import { getPermAuthzSetup, saveJourneyResult } from "../helpers/journeyResults";
import { createPermPrerequisites, extractIdFromEvents } from "../helpers/permissionHelpers";

const COOLUSER_MNEMONIC =
  (process.env.MNEMONIC && process.env.MNEMONIC.trim()) ||
  "pink glory help gown abstract eight nice crazy forward ketchup skill cheese";

const OPERATOR_INDEX = 15;

async function main() {
  console.log("=".repeat(60));
  console.log("Journey: PERM Create Permission (Self-Create, OPEN mode)");
  console.log("=".repeat(60));
  console.log();

  // Step 1: Load perm authz setup
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
  const wallet = await createDirectAccountFromMnemonic(COOLUSER_MNEMONIC, OPERATOR_INDEX);
  const account = await getAccountInfo(wallet);
  const client = await createSigningClient(wallet);
  console.log(`  Connected as ${account.address}`);
  console.log();

  try {
    // Step 3: Create prerequisites (TR + CS OPEN + Root Permission)
    console.log("Step 3: Creating prerequisites (TR + CS OPEN + Root Permission)...");
    const { schemaId, rootPermId, did, effectiveFrom } = await createPermPrerequisites(
      client,
      setup.authorityAddress,
      setup.operatorAddress,
      CredentialSchemaPermManagementMode.OPEN,
    );
    console.log(`  CS ID: ${schemaId} (OPEN mode)`);
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

    // Step 5: Create child ISSUER permission
    console.log("Step 5: Creating child ISSUER permission (MsgCreatePermission)...");
    const childEffectiveFrom = new Date(Date.now() + 30000); // 30s in future
    // Must be <= validator_perm.effective_until (root uses effectiveFrom + 360 days)
    const childEffectiveUntil = new Date(childEffectiveFrom.getTime() + 300 * 24 * 60 * 60 * 1000);

    const msg = {
      typeUrl: typeUrls.MsgCreatePermission,
      value: MsgCreatePermission.fromPartial({
        authority: setup.authorityAddress,
        operator: setup.operatorAddress,
        type: PermissionType.ISSUER,
        validatorPermId: rootPermId,
        did,
        effectiveFrom: childEffectiveFrom,
        effectiveUntil: childEffectiveUntil,
        validationFees: 5,
        verificationFees: 5,
        vsOperator: "",
        vsOperatorAuthzEnabled: false,
      }),
    };

    const fee = await calculateFeeWithSimulation(client, account.address, [msg], "Creating child permission");
    const result = await signAndBroadcastWithRetry(client, account.address, [msg], fee, "Creating child permission");

    if (result.code !== 0) {
      throw new Error(`Failed to create permission: ${result.rawLog}`);
    }

    const childPermId = extractIdFromEvents(result.events || [], "create_permission", ["permission_id", "id"]);

    console.log();
    console.log("SUCCESS! Child ISSUER permission created!");
    console.log(`  Tx Hash: ${result.transactionHash}`);
    console.log(`  Block: ${result.height}`);
    console.log(`  Gas: ${result.gasUsed}/${result.gasWanted}`);
    console.log(`  Child Permission ID: ${childPermId}`);

    saveJourneyResult("perm-child-setup", {
      permissionId: childPermId?.toString(),
      schemaId: schemaId.toString(),
    });
    console.log("  Saved perm-child-setup");
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
