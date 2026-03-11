/**
 * Journey: PERM Create Or Update Permission Session
 *
 * Creates a permission session (CSPS) and then updates it.
 * This requires a validated ISSUER permission with vs_operator enabled.
 *
 * Creates its own prerequisite chain:
 * TR → CS (GRANTOR_VALIDATION) → Root → StartVP (with vs_operator) → Validate → CSPS
 *
 * Requires: test:de-grant-perm-auth must be run first.
 *
 * Usage:
 *   npm run test:perm-csps
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
  config,
} from "../helpers/client";
import { typeUrls } from "../helpers/registry";
import {
  MsgStartPermissionVP,
  MsgSetPermissionVPToValidated,
  MsgCreateOrUpdatePermissionSession,
} from "../../../src/codec/verana/perm/v1/tx";
import { PermissionType, OptionalUInt64 } from "../../../src/codec/verana/perm/v1/types";
import { CredentialSchemaPermManagementMode } from "../../../src/codec/verana/cs/v1/types";
import { getPermAuthzSetup } from "../helpers/journeyResults";
import { createPermPrerequisites, extractIdFromEvents } from "../helpers/permissionHelpers";

const COOLUSER_MNEMONIC =
  (process.env.MNEMONIC && process.env.MNEMONIC.trim()) ||
  "pink glory help gown abstract eight nice crazy forward ketchup skill cheese";

const OPERATOR_INDEX = 15;

async function main() {
  console.log("=".repeat(60));
  console.log("Journey: PERM Create Or Update Permission Session");
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
    // Step 3: Create prerequisites (TR + CS + Root)
    console.log("Step 3: Creating prerequisites...");
    const { schemaId, rootPermId, did, effectiveFrom } = await createPermPrerequisites(
      client,
      setup.authorityAddress,
      setup.operatorAddress,
      CredentialSchemaPermManagementMode.ECOSYSTEM,
    );
    console.log(`  Schema ID: ${schemaId}, Root Permission ID: ${rootPermId}`);
    console.log();

    // Step 4: Wait for root to be effective
    console.log("Step 4: Waiting for root permission to become effective...");
    const queryClient = await createQueryClient();
    try {
      await waitForPermissionToBecomeEffective(queryClient, effectiveFrom, 60000);
    } finally {
      queryClient.disconnect();
    }
    console.log("  Root permission is now effective");
    console.log();

    // Step 5: Start VP for ISSUER with vs_operator enabled
    console.log("Step 5: Starting VP with vs_operator enabled...");
    const startMsg = {
      typeUrl: typeUrls.MsgStartPermissionVP,
      value: MsgStartPermissionVP.fromPartial({
        authority: setup.authorityAddress,
        operator: setup.operatorAddress,
        type: PermissionType.ISSUER,
        validatorPermId: rootPermId,
        did,
        validationFees: OptionalUInt64.fromPartial({ value: 5 }),
        issuanceFees: OptionalUInt64.fromPartial({ value: 5 }),
        verificationFees: OptionalUInt64.fromPartial({ value: 5 }),
        vsOperator: setup.operatorAddress,
        vsOperatorAuthzEnabled: true,
      }),
    };

    const startFee = await calculateFeeWithSimulation(client, account.address, [startMsg], "Starting VP for CSPS");
    const startResult = await signAndBroadcastWithRetry(client, account.address, [startMsg], startFee, "Starting VP for CSPS");

    if (startResult.code !== 0) {
      throw new Error(`Failed to start VP: ${startResult.rawLog}`);
    }

    const issuerPermId = extractIdFromEvents(startResult.events || [], "start_permission_vp", ["permission_id", "id"]);
    if (!issuerPermId) throw new Error("Could not extract ISSUER perm ID");
    console.log(`  ISSUER Permission ID: ${issuerPermId}`);

    // Step 6: Validate the VP
    console.log("Step 6: Validating VP...");
    const effectiveUntil = new Date(Date.now() + 300 * 24 * 60 * 60 * 1000);
    const validateMsg = {
      typeUrl: typeUrls.MsgSetPermissionVPToValidated,
      value: MsgSetPermissionVPToValidated.fromPartial({
        authority: setup.authorityAddress,
        operator: setup.operatorAddress,
        id: issuerPermId,
        effectiveUntil,
        validationFees: 5,
        issuanceFees: 5,
        verificationFees: 5,
        vpSummaryDigestSri: "sha384-cspsValidationDigest",
        issuanceFeeDiscount: 0,
        verificationFeeDiscount: 0,
      }),
    };

    const validateFee = await calculateFeeWithSimulation(client, account.address, [validateMsg], "Validating VP for CSPS");
    const validateResult = await signAndBroadcastWithRetry(client, account.address, [validateMsg], validateFee, "Validating VP for CSPS");

    if (validateResult.code !== 0) {
      throw new Error(`Failed to validate VP: ${validateResult.rawLog}`);
    }
    console.log("  VP validated");
    console.log();

    // Step 7: Create Permission Session
    console.log("Step 7: Creating permission session (MsgCreateOrUpdatePermissionSession)...");
    const sessionId = `session-${Date.now()}-${Math.random().toString(36).substring(2, 8)}`;

    const cspsMsg = {
      typeUrl: typeUrls.MsgCreateOrUpdatePermissionSession,
      value: MsgCreateOrUpdatePermissionSession.fromPartial({
        authority: setup.authorityAddress,
        operator: setup.operatorAddress,
        id: sessionId,
        issuerPermId: issuerPermId,
        verifierPermId: 0,
        agentPermId: issuerPermId,
        walletAgentPermId: issuerPermId,
        digest: "sha384-sessionDigest123",
      }),
    };

    const cspsFee = await calculateFeeWithSimulation(client, account.address, [cspsMsg], "Creating permission session");
    const cspsResult = await signAndBroadcastWithRetry(client, account.address, [cspsMsg], cspsFee, "Creating permission session");

    if (cspsResult.code !== 0) {
      throw new Error(`Failed to create permission session: ${cspsResult.rawLog}`);
    }

    console.log();
    console.log("SUCCESS! Permission session created!");
    console.log(`  Tx Hash: ${cspsResult.transactionHash}`);
    console.log(`  Block: ${cspsResult.height}`);
    console.log(`  Gas: ${cspsResult.gasUsed}/${cspsResult.gasWanted}`);
    console.log(`  Session ID: ${sessionId}`);

    // Step 8: Update the session
    console.log();
    console.log("Step 8: Updating permission session...");
    const updateMsg = {
      typeUrl: typeUrls.MsgCreateOrUpdatePermissionSession,
      value: MsgCreateOrUpdatePermissionSession.fromPartial({
        authority: setup.authorityAddress,
        operator: setup.operatorAddress,
        id: sessionId,
        issuerPermId: issuerPermId,
        verifierPermId: 0,
        agentPermId: issuerPermId,
        walletAgentPermId: issuerPermId,
        digest: "sha384-updatedSessionDigest456",
      }),
    };

    const updateFee = await calculateFeeWithSimulation(client, account.address, [updateMsg], "Updating permission session");
    const updateResult = await signAndBroadcastWithRetry(client, account.address, [updateMsg], updateFee, "Updating permission session");

    if (updateResult.code !== 0) {
      throw new Error(`Failed to update permission session: ${updateResult.rawLog}`);
    }

    console.log("  Session updated!");
    console.log(`  Tx Hash: ${updateResult.transactionHash}`);
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
