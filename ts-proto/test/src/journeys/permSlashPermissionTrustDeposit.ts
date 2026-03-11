/**
 * Journey: PERM Slash Permission Trust Deposit
 *
 * Creates a fresh validated ISSUER permission (full chain: TR → CS → Root → VP → Validate),
 * then slashes its trust deposit.
 *
 * Requires: test:de-grant-perm-auth must be run first.
 *
 * Usage:
 *   npm run test:perm-slash
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
import { MsgSlashPermissionTrustDeposit } from "../../../src/codec/verana/perm/v1/tx";
import { CredentialSchemaPermManagementMode } from "../../../src/codec/verana/cs/v1/types";
import { getPermAuthzSetup, savePermSlashSetup } from "../helpers/journeyResults";
import { createPermPrerequisites, createValidatedPermission } from "../helpers/permissionHelpers";

const COOLUSER_MNEMONIC =
  (process.env.MNEMONIC && process.env.MNEMONIC.trim()) ||
  "pink glory help gown abstract eight nice crazy forward ketchup skill cheese";

const OPERATOR_INDEX = 15;

async function main() {
  console.log("=".repeat(60));
  console.log("Journey: PERM Slash Permission Trust Deposit");
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
    // Step 3: Create fresh prerequisites
    console.log("Step 3: Creating fresh prerequisites (TR + CS + Root Permission)...");
    const { schemaId, rootPermId, did, effectiveFrom } = await createPermPrerequisites(
      client,
      setup.authorityAddress,
      setup.operatorAddress,
      CredentialSchemaPermManagementMode.ECOSYSTEM,
    );
    console.log(`  Schema ID: ${schemaId}, Root Permission ID: ${rootPermId}`);
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

    // Step 5: Create validated ISSUER permission (VP lifecycle)
    console.log("Step 5: Creating validated ISSUER permission (Start VP + Validate)...");
    const issuerPermId = await createValidatedPermission(
      client,
      setup.authorityAddress,
      setup.operatorAddress,
      schemaId,
      rootPermId,
      did,
    );
    console.log(`  Validated ISSUER Permission ID: ${issuerPermId}`);
    console.log();

    // Step 6: Slash the trust deposit
    console.log("Step 6: Slashing trust deposit (MsgSlashPermissionTrustDeposit)...");
    const slashAmount = 10;

    const msg = {
      typeUrl: typeUrls.MsgSlashPermissionTrustDeposit,
      value: MsgSlashPermissionTrustDeposit.fromPartial({
        authority: setup.authorityAddress,
        operator: setup.operatorAddress,
        id: issuerPermId,
        amount: slashAmount,
      }),
    };

    const fee = await calculateFeeWithSimulation(client, account.address, [msg], "Slashing trust deposit");
    const result = await signAndBroadcastWithRetry(client, account.address, [msg], fee, "Slashing trust deposit");

    if (result.code !== 0) {
      throw new Error(`Failed to slash trust deposit: ${result.rawLog}`);
    }

    console.log();
    console.log("SUCCESS! Trust deposit slashed!");
    console.log(`  Tx Hash: ${result.transactionHash}`);
    console.log(`  Block: ${result.height}`);
    console.log(`  Gas: ${result.gasUsed}/${result.gasWanted}`);
    console.log(`  Slashed Permission ID: ${issuerPermId}`);
    console.log(`  Slash Amount: ${slashAmount}`);

    savePermSlashSetup(issuerPermId, schemaId);
    console.log("  Saved perm-slash-setup");
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
