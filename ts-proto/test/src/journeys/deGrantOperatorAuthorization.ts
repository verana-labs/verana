/**
 * Journey: DE Grant Operator Authorization (EC + GF message types)
 *
 * Grants operator authorization from the active Corporation (created by
 * coCreateCorporation) to an operator account, covering all EC and GF
 * message types. The operator can then sign EC/GF messages on behalf of
 * the Corporation.
 *
 * Flow: x/group proposal submitted by the sole group member (index 10),
 * who votes YES with EXEC_TRY — proposal auto-executes since threshold=1.
 *
 * Signing: all messages use SIGN_MODE_LEGACY_AMINO_JSON. Group coordination
 * (MsgSubmitProposal, MsgVote) amino converters are implemented in
 * ts-proto/src/amino-converter/group.ts with recursive Any transcoding.
 * MsgGrantOperatorAuthorization is the inner executed message and is
 * amino-encoded as part of the proposal messages array.
 *
 * Requires: test:co-create must be run first.
 *
 * Usage:
 *   npm run test:de-grant-auth
 */

import {
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
import { MsgSubmitProposal, MsgVote, Exec } from "cosmjs-types/cosmos/group/v1/tx";
import { VoteOption } from "cosmjs-types/cosmos/group/v1/types";
import { saveEcAuthzSetup, getActiveCorporation } from "../helpers/journeyResults";

const COOLUSER_MNEMONIC =
  (process.env.MNEMONIC && process.env.MNEMONIC.trim()) ||
  "pink glory help gown abstract eight nice crazy forward ketchup skill cheese";

// Signer (index 10) is the sole group member and acts as proposer + voter.
// Operator (index 11) is the grantee.
const SIGNER_INDEX = 10;
const OPERATOR_INDEX = 11;

async function main() {
  console.log("=".repeat(60));
  console.log("Journey: DE Grant Operator Authorization (EC + GF)");
  console.log("=".repeat(60));
  console.log();

  // Step 1: Load active Corporation policy_address (the v4-rc2 authority).
  console.log("Step 1: Loading active Corporation...");
  const corp = getActiveCorporation();
  if (!corp) {
    console.log("  ❌ No active corporation found. Run test:co-create first.");
    process.exit(1);
  }
  console.log(`  Corp ID:        ${corp.corporationId}`);
  console.log(`  Policy Address: ${corp.policyAddress}`);
  console.log();

  // Step 2: Create wallets (both amino — SIGN_MODE_LEGACY_AMINO_JSON).
  console.log("Step 2: Creating wallets...");
  const signerWallet = await createAccountFromMnemonic(COOLUSER_MNEMONIC, SIGNER_INDEX);
  const operatorWallet = await createAccountFromMnemonic(COOLUSER_MNEMONIC, OPERATOR_INDEX);

  const signerAccount = await getAccountInfo(signerWallet);
  const operatorAccount = await getAccountInfo(operatorWallet);

  console.log(`  Signer:   ${signerAccount.address} (derivation index ${SIGNER_INDEX}, SIGN_MODE_LEGACY_AMINO_JSON)`);
  console.log(`  Operator: ${operatorAccount.address} (derivation index ${OPERATOR_INDEX})`);
  console.log();

  // Step 3: Fund operator (signer was already funded by test:co-create)
  console.log("Step 3: Funding operator...");
  const cooluserWallet = await createAccountFromMnemonic(COOLUSER_MNEMONIC, 0);
  const cooluserAccount = await getAccountInfo(cooluserWallet);

  const fundOpResult = await fundAccount(
    COOLUSER_MNEMONIC,
    cooluserAccount.address,
    operatorAccount.address,
    "50000000uvna",
  );
  if (fundOpResult.code !== 0) {
    console.log(`  ❌ Failed to fund operator: ${fundOpResult.rawLog}`);
    process.exit(1);
  }
  console.log(`  ✓ Funded operator with 50 VNA`);

  const qc1 = await createQueryClient();
  console.log("  ⏳ Waiting for operator funding tx to confirm...");
  for (let i = 0; i < 30; i++) {
    try {
      const tx = await qc1.getTx(fundOpResult.transactionHash);
      if (tx) { console.log(`  ✓ Operator funding confirmed at block ${tx.height}`); break; }
    } catch {}
    await new Promise((r) => setTimeout(r, 1000));
  }
  qc1.disconnect();
  console.log();

  // Step 4: Grant operator authorization via x/group proposal flow.
  // The policy_address (corp.policyAddress) is both the group policy and the
  // `corporation` field of MsgGrantOperatorAuthorization. Since the signer
  // (index 10) is the sole group member with weight=1 and threshold=1, a single
  // proposal + YES vote with EXEC_TRY executes immediately.
  console.log("Step 4: Building MsgGrantOperatorAuthorization...");

  const allMsgTypes = [
    typeUrls.MsgCreateEcosystem,
    typeUrls.MsgUpdateEcosystem,
    typeUrls.MsgArchiveEcosystem,
    typeUrls.MsgAddGovernanceFrameworkDocument,
    typeUrls.MsgIncreaseActiveGovernanceFrameworkVersion,
  ];

  console.log("  Message types being authorized:");
  for (const msgType of allMsgTypes) {
    console.log(`    - ${msgType}`);
  }
  console.log();

  // Encode MsgGrantOperatorAuthorization as raw bytes for the Any wrapper.
  const innerMsgValue = MsgGrantOperatorAuthorization.encode(
    MsgGrantOperatorAuthorization.fromPartial({
      corporation: corp.policyAddress,
      operator: "",
      grantee: operatorAccount.address,
      msgTypes: allMsgTypes,
      withFeegrant: false,
    }),
  ).finish();

  const proposalMsg = {
    typeUrl: "/cosmos.group.v1.MsgSubmitProposal",
    value: MsgSubmitProposal.fromPartial({
      groupPolicyAddress: corp.policyAddress,
      proposers: [signerAccount.address],
      metadata: "Grant EC+GF operator authz",
      title: "Grant EC+GF operator authz",
      summary: "Grant operator authorization for EC and GF message types",
      messages: [{ typeUrl: typeUrls.MsgGrantOperatorAuthorization, value: innerMsgValue }],
      exec: Exec.EXEC_UNSPECIFIED,
    }),
  };

  const client = await createSigningClient(signerWallet);

  try {
    // Step 4a: Submit proposal
    console.log("Step 4a: Submitting x/group proposal (SIGN_MODE_LEGACY_AMINO_JSON)...");
    const proposalFee = await calculateFeeWithSimulation(
      client, signerAccount.address, [proposalMsg], "Submit grant operator authz proposal",
    );
    console.log(`  Gas: ${proposalFee.gas}, Fee: ${proposalFee.amount[0].amount}${proposalFee.amount[0].denom}`);

    const proposalResult = await signAndBroadcastWithRetry(
      client, signerAccount.address, [proposalMsg], proposalFee, "Submit grant operator authz proposal",
    );

    if (proposalResult.code !== 0) {
      console.log(`❌ Proposal submission failed: ${proposalResult.rawLog}`);
      process.exit(1);
    }
    console.log(`✅ Step 4a: Proposal submitted at block ${proposalResult.height}`);
    console.log(`  Tx: ${proposalResult.transactionHash}`);

    // Extract proposal_id from events
    let proposalId: bigint | undefined;
    for (const event of (proposalResult.events || [])) {
      for (const attr of event.attributes) {
        if (attr.key === "proposal_id") {
          proposalId = BigInt(String(attr.value).replace(/"/g, ""));
          break;
        }
      }
      if (proposalId !== undefined) break;
    }
    if (proposalId === undefined) {
      console.log("❌ Could not extract proposal_id from events");
      console.log("Events:", JSON.stringify(proposalResult.events?.slice(0, 5), null, 2));
      process.exit(1);
    }
    console.log(`  Proposal ID: ${proposalId}`);

    // Wait for proposal tx to confirm
    const qc2 = await createQueryClient();
    for (let i = 0; i < 30; i++) {
      try {
        const tx = await qc2.getTx(proposalResult.transactionHash);
        if (tx) { console.log(`  Proposal confirmed at block ${tx.height}`); break; }
      } catch {}
      await new Promise((r) => setTimeout(r, 1000));
    }
    qc2.disconnect();
    console.log();

    // Step 4b: Vote YES with EXEC_TRY — auto-executes (threshold=1, weight=1)
    console.log("Step 4b: Voting YES on proposal (EXEC_TRY, SIGN_MODE_LEGACY_AMINO_JSON)...");
    const voteMsg = {
      typeUrl: "/cosmos.group.v1.MsgVote",
      value: MsgVote.fromPartial({
        proposalId,
        voter: signerAccount.address,
        option: VoteOption.VOTE_OPTION_YES,
        exec: Exec.EXEC_TRY,
        metadata: "",
      }),
    };

    const voteFee = await calculateFeeWithSimulation(
      client, signerAccount.address, [voteMsg], "Vote YES on grant proposal",
    );
    console.log(`  Gas: ${voteFee.gas}, Fee: ${voteFee.amount[0].amount}${voteFee.amount[0].denom}`);

    const voteResult = await signAndBroadcastWithRetry(
      client, signerAccount.address, [voteMsg], voteFee, "Vote YES on grant proposal",
    );

    if (voteResult.code !== 0) {
      console.log(`❌ Vote failed: ${voteResult.rawLog}`);
      process.exit(1);
    }
    console.log(`✅ Step 4b: Voted YES + executed at block ${voteResult.height}`);
    console.log(`  Tx: ${voteResult.transactionHash}`);

    // Wait for vote to confirm
    const qc3 = await createQueryClient();
    for (let i = 0; i < 30; i++) {
      try {
        const tx = await qc3.getTx(voteResult.transactionHash);
        if (tx) { console.log(`  Vote confirmed at block ${tx.height}`); break; }
      } catch {}
      await new Promise((r) => setTimeout(r, 1000));
    }
    qc3.disconnect();

    console.log();
    console.log("✅ SUCCESS! Operator authorization granted via x/group proposal!");
    console.log("=".repeat(60));
    console.log(`  Proposal Tx: ${proposalResult.transactionHash}`);
    console.log(`  Vote Tx:     ${voteResult.transactionHash}`);

    saveEcAuthzSetup(corp.policyAddress, operatorAccount.address);
    console.log("  💾 Saved EC authz setup (corporation + operator) for EC/GF journeys");

  } catch (error: any) {
    console.log("❌ ERROR! Transaction failed with exception:");
    console.error(error);
    process.exit(1);
  } finally {
    client.disconnect();
  }

  console.log();
  console.log("=".repeat(60));
}

main().catch((error: any) => {
  console.error("\n❌ Fatal error:", error.message || error);

  if (error.cause?.code === "ECONNREFUSED" || error.message?.includes("fetch failed")) {
    console.error("\n⚠️  Connection Error: Cannot connect to the blockchain.");
    console.error(`   Make sure the Verana blockchain is running at ${config.rpcEndpoint}`);
  }

  process.exit(1);
});
