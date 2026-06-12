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
import { getEcAuthzSetup, savePermAuthzSetup } from "../helpers/journeyResults";
import { MsgSubmitProposal, MsgVote, Exec } from "cosmjs-types/cosmos/group/v1/tx";
import { VoteOption } from "cosmjs-types/cosmos/group/v1/types";

const COOLUSER_MNEMONIC =
  (process.env.MNEMONIC && process.env.MNEMONIC.trim()) ||
  "pink glory help gown abstract eight nice crazy forward ketchup skill cheese";

const OPERATOR_INDEX = 15;
// Corp group member (proposer/voter) — matches co-create's SIGNER_INDEX.
const SIGNER_INDEX = 10;

async function main() {
  console.log("=".repeat(60));
  console.log("Journey: DE Grant PERM Operator Authorization");
  console.log("=".repeat(60));
  console.log();

  // Step 1: Load authority (Corporation policy_address) from EC authz setup
  console.log("Step 1: Loading authority from EC authz setup...");
  const ecSetup = getEcAuthzSetup();
  if (!ecSetup) {
    console.log("  No EC authz setup found. Run test:de-grant-auth first.");
    process.exit(1);
  }
  const authorityAddress = ecSetup.authorityAddress;
  console.log(`  Authority (corporation policy_address): ${authorityAddress}`);
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
    // EC messages (for creating prerequisite Ecosystems)
    typeUrls.MsgCreateEcosystem,
    typeUrls.MsgUpdateEcosystem,
    typeUrls.MsgArchiveEcosystem,
    // GF messages (moved from verana.tr.v1 to verana.gf.v1 in v4-rc2)
    typeUrls.MsgAddGovernanceFrameworkDocument,
    typeUrls.MsgIncreaseActiveGovernanceFrameworkVersion,
    // CS messages (for creating prerequisite CSs)
    typeUrls.MsgCreateCredentialSchema,
    typeUrls.MsgUpdateCredentialSchema,
    typeUrls.MsgArchiveCredentialSchema,
    // PERM messages
    typeUrls.MsgCreateRootParticipant,
    typeUrls.MsgSelfCreateParticipant,
    typeUrls.MsgSetParticipantEffectiveUntil,
    typeUrls.MsgRevokeParticipant,
    typeUrls.MsgStartParticipantOP,
    typeUrls.MsgRenewParticipantOP,
    typeUrls.MsgSetParticipantOPToValidated,
    typeUrls.MsgCancelParticipantOPLastRequest,
    // Note: MsgCreateOrUpdateParticipantSession is NOT DE-delegable
    // It uses VS operator authorization (AUTHZ-CHECK-3) instead
    typeUrls.MsgSlashParticipantTrustDeposit,
    typeUrls.MsgRepayParticipantSlashedTrustDeposit,
    // [MOD-PP-MSG-15] Trigger Resolver
    typeUrls.MsgTriggerResolver,
  ];

  // The corporation policy_address cannot sign directly; the corp's sole group
  // member (SIGNER_INDEX, matching co-create) submits an x/group proposal carrying
  // the inner MsgGrantOperatorAuthorization (operator="" = corporation acts alone)
  // and votes YES with EXEC_TRY so the policy executes it (threshold=1, weight=1).
  const signerWallet = await createAccountFromMnemonic(COOLUSER_MNEMONIC, SIGNER_INDEX);
  const signerAccount = await getAccountInfo(signerWallet);
  const client = await createSigningClient(signerWallet);

  const innerMsgValue = MsgGrantOperatorAuthorization.encode(
    MsgGrantOperatorAuthorization.fromPartial({
      corporation: authorityAddress,
      operator: "",
      grantee: operatorAccount.address,
      msgTypes: allMsgTypes,
      withFeegrant: false,
    }),
  ).finish();

  const proposalMsg = {
    typeUrl: "/cosmos.group.v1.MsgSubmitProposal",
    value: MsgSubmitProposal.fromPartial({
      groupPolicyAddress: authorityAddress,
      proposers: [signerAccount.address],
      metadata: "Grant PERM operator authz",
      title: "Grant PERM operator authz",
      summary: "Grant operator authorization for PERM/EC/GF/CS message types",
      messages: [{ typeUrl: typeUrls.MsgGrantOperatorAuthorization, value: innerMsgValue }],
      exec: Exec.EXEC_UNSPECIFIED,
    }),
  };

  try {
    // 4a: submit proposal
    const proposalFee = await calculateFeeWithSimulation(client, signerAccount.address, [proposalMsg], "Submit PERM grant proposal");
    const proposalResult = await signAndBroadcastWithRetry(client, signerAccount.address, [proposalMsg], proposalFee, "Submit PERM grant proposal");
    if (proposalResult.code !== 0) {
      if (String(proposalResult.rawLog).includes("already exists")) {
        savePermAuthzSetup(authorityAddress, operatorAccount.address);
        console.log("  Authorization already exists; saved setup.");
        client.disconnect();
        console.log("=".repeat(60));
        return;
      }
      console.log(`FAILED proposal: ${proposalResult.rawLog}`);
      process.exit(1);
    }
    let proposalId: bigint | undefined;
    for (const event of (proposalResult.events || [])) {
      for (const attr of event.attributes) {
        if (attr.key === "proposal_id") { proposalId = BigInt(String(attr.value).replace(/"/g, "")); break; }
      }
      if (proposalId !== undefined) break;
    }
    if (proposalId === undefined) { console.log("Could not extract proposal_id"); process.exit(1); }
    console.log(`  Proposal ID: ${proposalId}`);
    const qc2 = await createQueryClient();
    for (let i = 0; i < 30; i++) { try { if (await qc2.getTx(proposalResult.transactionHash)) break; } catch {} await new Promise((r) => setTimeout(r, 1000)); }
    qc2.disconnect();

    // 4b: vote YES with EXEC_TRY (auto-executes)
    const voteMsg = {
      typeUrl: "/cosmos.group.v1.MsgVote",
      value: MsgVote.fromPartial({ proposalId, voter: signerAccount.address, option: VoteOption.VOTE_OPTION_YES, exec: Exec.EXEC_TRY, metadata: "" }),
    };
    const voteFee = await calculateFeeWithSimulation(client, signerAccount.address, [voteMsg], "Vote YES PERM grant");
    const voteResult = await signAndBroadcastWithRetry(client, signerAccount.address, [voteMsg], voteFee, "Vote YES PERM grant");
    if (voteResult.code !== 0) { console.log(`FAILED vote: ${voteResult.rawLog}`); process.exit(1); }
    console.log(`  Voted YES + executed at block ${voteResult.height}; authorized ${allMsgTypes.length} msg types`);
    const qc3 = await createQueryClient();
    for (let i = 0; i < 30; i++) { try { if (await qc3.getTx(voteResult.transactionHash)) break; } catch {} await new Promise((r) => setTimeout(r, 1000)); }
    qc3.disconnect();

    savePermAuthzSetup(authorityAddress, operatorAccount.address);
    console.log("  Saved perm-authz-setup");
  } catch (error: any) {
    const errorMsg = error?.message || String(error);
    if (errorMsg.includes("already exists") || errorMsg.includes("mutual exclusivity")) {
      console.log("  Authorization already exists on chain. Saving setup and continuing.");
      savePermAuthzSetup(authorityAddress, operatorAccount.address);
    } else {
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
