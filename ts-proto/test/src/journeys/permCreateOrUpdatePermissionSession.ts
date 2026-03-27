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
  createDirectAccountFromMnemonic,
  createSigningClient,
  getAccountInfo,
  calculateFeeWithSimulation,
  signAndBroadcastWithRetry,
  waitForPermissionToBecomeEffective,
  createQueryClient,
  generateUniqueDID,
  fundAccount,
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
  const wallet = await createDirectAccountFromMnemonic(COOLUSER_MNEMONIC, OPERATOR_INDEX);
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
    // Use a distinct vs_operator account (derivation index 16) to avoid mutual
    // exclusivity conflict with the OperatorAuthorization for setup.operatorAddress.
    console.log("Step 5: Starting VP with vs_operator enabled...");
    const vsOperatorWallet = await createDirectAccountFromMnemonic(COOLUSER_MNEMONIC, 16);
    const vsOperatorAccount = await getAccountInfo(vsOperatorWallet);
    console.log(`  VS Operator: ${vsOperatorAccount.address}`);

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
        vsOperator: vsOperatorAccount.address,
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

    // Step 6b: Fund VS operator so it can sign transactions
    console.log("  Funding VS operator...");
    const fundVsResult = await client.sendTokens(
      account.address,
      vsOperatorAccount.address,
      [{ denom: config.denom, amount: "50000000" }],
      "auto",
    );
    if (fundVsResult.code !== 0) {
      throw new Error(`Failed to fund VS operator: ${fundVsResult.rawLog}`);
    }
    console.log(`  ✓ VS operator funded`);
    // Wait for funding to confirm
    await new Promise(resolve => setTimeout(resolve, 5000));

    // Step 7: Create Permission Session (signed by vs_operator)
    console.log("Step 7: Creating permission session (MsgCreateOrUpdatePermissionSession)...");
    const vsClient = await createSigningClient(vsOperatorWallet);
    const sessionId = crypto.randomUUID();

    const cspsMsg = {
      typeUrl: typeUrls.MsgCreateOrUpdatePermissionSession,
      value: MsgCreateOrUpdatePermissionSession.fromPartial({
        authority: setup.authorityAddress,
        operator: vsOperatorAccount.address,
        id: sessionId,
        issuerPermId: issuerPermId,
        verifierPermId: 0,
        agentPermId: issuerPermId,
        walletAgentPermId: issuerPermId,
        digest: "sha384-sessionDigest123",
      }),
    };

    const cspsFee = await calculateFeeWithSimulation(vsClient, vsOperatorAccount.address, [cspsMsg], "Creating permission session");
    const cspsResult = await signAndBroadcastWithRetry(vsClient, vsOperatorAccount.address, [cspsMsg], cspsFee, "Creating permission session");

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
        operator: vsOperatorAccount.address,
        id: sessionId,
        issuerPermId: issuerPermId,
        verifierPermId: 0,
        agentPermId: issuerPermId,
        walletAgentPermId: issuerPermId,
        digest: "sha384-updatedSessionDigest456",
      }),
    };

    const updateFee = await calculateFeeWithSimulation(vsClient, vsOperatorAccount.address, [updateMsg], "Updating permission session");
    const updateResult = await signAndBroadcastWithRetry(vsClient, vsOperatorAccount.address, [updateMsg], updateFee, "Updating permission session");

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
    try { vsClient?.disconnect(); } catch (_) {}
  }

  console.log();
  console.log("=".repeat(60));
}

main().catch((error: any) => {
  console.error("\nFatal error:", error.message || error);
  process.exit(1);
});
