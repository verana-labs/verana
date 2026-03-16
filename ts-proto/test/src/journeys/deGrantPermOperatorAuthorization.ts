/**
 * Journey: DE Grant PERM Operator Authorization
 *
 * Grants operator authorization from the authority account to a perm-specific
 * operator for all PERM message types, plus TR and CS message types needed
 * for creating prerequisites in perm journeys.
 *
 * Uses: authority=index 10 (same as TR/CS), operator=index 15
 *
 * Usage:
 *   npm run test:de-grant-perm-auth
 */

import {
  createWallet,
  createAccountFromMnemonic,
  createSigningClient,
  createQueryClient,
  getAccountInfo,
  calculateFeeWithSimulation,
  signAndBroadcastWithRetry,
  fundAccount,
  config,
} from "../helpers/client";
import { typeUrls } from "../helpers/registry";
import { MsgGrantOperatorAuthorization } from "../../../src/codec/verana/de/v1/tx";
import { getTrAuthzSetup, savePermAuthzSetup } from "../helpers/journeyResults";

const COOLUSER_MNEMONIC =
  (process.env.MNEMONIC && process.env.MNEMONIC.trim()) ||
  "pink glory help gown abstract eight nice crazy forward ketchup skill cheese";

const OPERATOR_INDEX = 15;

async function main() {
  console.log("=".repeat(60));
  console.log("Journey: DE Grant PERM Operator Authorization");
  console.log("=".repeat(60));
  console.log();

  // Step 1: Load authority from TR authz setup
  console.log("Step 1: Loading authority from TR authz setup...");
  const trSetup = getTrAuthzSetup();
  if (!trSetup) {
    console.log("  No TR authz setup found. Run test:de-grant-auth first.");
    process.exit(1);
  }
  const authorityAddress = trSetup.authorityAddress;
  console.log(`  Authority: ${authorityAddress}`);
  console.log();

  // Step 2: Create perm operator wallet
  console.log("Step 2: Creating PERM operator wallet...");
  const operatorWallet = await createAccountFromMnemonic(COOLUSER_MNEMONIC, OPERATOR_INDEX);
  const cooluserWallet = await createWallet(COOLUSER_MNEMONIC);
  const operatorAccount = await getAccountInfo(operatorWallet);
  const cooluserAccount = await getAccountInfo(cooluserWallet);
  console.log(`  PERM Operator: ${operatorAccount.address} (index ${OPERATOR_INDEX})`);
  console.log();

  // Step 3: Fund operator
  console.log("Step 3: Funding PERM operator...");
  const fundAmount = "5000000000uvna"; // 5000 VNA (enough for perm operations with trust deposits + fees)

  const fundResult = await fundAccount(
    COOLUSER_MNEMONIC,
    cooluserAccount.address,
    operatorAccount.address,
    fundAmount,
  );
  if (fundResult.code !== 0) {
    console.log(`  Failed to fund operator: ${fundResult.rawLog}`);
    process.exit(1);
  }
  console.log(`  Funded operator with ${fundAmount}`);

  // Wait for funding to confirm
  const queryClient = await createQueryClient();
  console.log("  Waiting for funding tx to confirm...");
  for (let i = 0; i < 30; i++) {
    try {
      const tx = await queryClient.getTx(fundResult.transactionHash);
      if (tx) {
        console.log(`  Funding confirmed at block ${tx.height}`);
        break;
      }
    } catch {}
    await new Promise((r) => setTimeout(r, 1000));
  }

  // Verify balance
  for (let i = 0; i < 20; i++) {
    const balance = await queryClient.getBalance(operatorAccount.address, config.denom);
    if (BigInt(balance.amount) > 0) {
      console.log(`  Operator balance: ${balance.amount}${balance.denom}`);
      break;
    }
    await new Promise((r) => setTimeout(r, 1000));
  }
  queryClient.disconnect();
  console.log();

  // Step 4: Grant operator authorization for all PERM + TR + CS message types
  console.log("Step 4: Granting operator authorization...");

  const allMsgTypes = [
    // TR messages (for creating prerequisite TRs)
    typeUrls.MsgCreateTrustRegistry,
    typeUrls.MsgUpdateTrustRegistry,
    typeUrls.MsgArchiveTrustRegistry,
    typeUrls.MsgAddGovernanceFrameworkDocument,
    typeUrls.MsgIncreaseActiveGovernanceFrameworkVersion,
    // CS messages (for creating prerequisite CSs)
    typeUrls.MsgCreateCredentialSchema,
    typeUrls.MsgUpdateCredentialSchema,
    typeUrls.MsgArchiveCredentialSchema,
    // PERM messages
    typeUrls.MsgCreateRootPermission,
    typeUrls.MsgCreatePermission,
    typeUrls.MsgAdjustPermission,
    typeUrls.MsgRevokePermission,
    typeUrls.MsgStartPermissionVP,
    typeUrls.MsgRenewPermissionVP,
    typeUrls.MsgSetPermissionVPToValidated,
    typeUrls.MsgCancelPermissionVPLastRequest,
    // Note: MsgCreateOrUpdatePermissionSession is NOT DE-delegable
    // It uses VS operator authorization (AUTHZ-CHECK-3) instead
    typeUrls.MsgSlashPermissionTrustDeposit,
    typeUrls.MsgRepayPermissionSlashedTrustDeposit,
  ];

  // Authority signs the grant directly (no operator for DE grant)
  const authorityWallet = await createAccountFromMnemonic(COOLUSER_MNEMONIC, 10);
  const client = await createSigningClient(authorityWallet);

  const msg = {
    typeUrl: typeUrls.MsgGrantOperatorAuthorization,
    value: MsgGrantOperatorAuthorization.fromPartial({
      authority: authorityAddress,
      operator: "", // authority acts alone
      grantee: operatorAccount.address,
      msgTypes: allMsgTypes,
      withFeegrant: false,
    }),
  };

  try {
    const fee = await calculateFeeWithSimulation(
      client, authorityAddress, [msg],
      "Granting PERM operator authorization",
    );
    console.log(`  Gas: ${fee.gas}, Fee: ${fee.amount[0].amount}${fee.amount[0].denom}`);

    const result = await signAndBroadcastWithRetry(
      client, authorityAddress, [msg], fee,
      "Granting PERM operator authorization",
    );

    if (result.code === 0) {
      console.log();
      console.log("SUCCESS! PERM operator authorization granted!");
      console.log(`  Tx Hash: ${result.transactionHash}`);
      console.log(`  Block: ${result.height}`);
      console.log(`  Gas: ${result.gasUsed}/${result.gasWanted}`);
      console.log(`  Authorized ${allMsgTypes.length} message types`);

      savePermAuthzSetup(authorityAddress, operatorAccount.address);
      console.log("  Saved perm-authz-setup");
    } else {
      console.log("FAILED!");
      console.log(`  Code: ${result.code}`);
      console.log(`  Log: ${result.rawLog}`);
      process.exit(1);
    }
  } catch (error: any) {
    const errorMsg = error?.message || String(error);
    // If authorization already exists (from a previous run on same chain), save setup and continue
    if (errorMsg.includes("already exists") || errorMsg.includes("mutual exclusivity")) {
      console.log("  Authorization already exists on chain (from previous run). Saving setup and continuing.");
      savePermAuthzSetup(authorityAddress, operatorAccount.address);
      console.log("  Saved perm-authz-setup");
    } else {
      console.log("ERROR!");
      console.error(error);
      process.exit(1);
    }
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
