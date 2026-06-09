/**
 * Shared cosmos x/group proposal helper for ts-proto journeys (amino).
 * Submits a group proposal (single sole-member group, threshold=1) and votes YES
 * with EXEC_TRY so it auto-executes. Inner messages are amino-encoded via the
 * group MsgSubmitProposal converter (recursive Any).
 */
import {
  calculateFeeWithSimulation,
  signAndBroadcastWithRetry,
  createQueryClient,
} from "./client";
import { MsgSubmitProposal, MsgVote, Exec } from "cosmjs-types/cosmos/group/v1/tx";
import { VoteOption } from "cosmjs-types/cosmos/group/v1/types";

async function waitForTx(hash: string): Promise<void> {
  const qc = await createQueryClient();
  try {
    for (let i = 0; i < 30; i++) {
      try {
        const tx = await qc.getTx(hash);
        if (tx) return;
      } catch {}
      await new Promise((r) => setTimeout(r, 1000));
    }
  } finally {
    qc.disconnect();
  }
}

/**
 * Submit a group proposal with the given inner messages, then vote YES with
 * EXEC_TRY (auto-executes since the sole group member has weight=1, threshold=1).
 */
export async function submitGroupProposalAndExec(
  client: any,
  signerAddress: string,
  groupPolicyAddress: string,
  messages: { typeUrl: string; value: Uint8Array }[],
  title: string,
): Promise<void> {
  const proposalMsg = {
    typeUrl: "/cosmos.group.v1.MsgSubmitProposal",
    value: MsgSubmitProposal.fromPartial({
      groupPolicyAddress,
      proposers: [signerAddress],
      metadata: title,
      title,
      summary: title,
      messages,
      exec: Exec.EXEC_UNSPECIFIED,
    }),
  };
  const pFee = await calculateFeeWithSimulation(client, signerAddress, [proposalMsg], title);
  const pRes = await signAndBroadcastWithRetry(client, signerAddress, [proposalMsg], pFee, title);
  if (pRes.code !== 0) throw new Error(`group proposal submit failed: ${pRes.rawLog}`);

  let proposalId: bigint | undefined;
  for (const event of pRes.events || []) {
    for (const attr of event.attributes || []) {
      if (attr.key === "proposal_id") {
        proposalId = BigInt(String(attr.value).replace(/"/g, ""));
        break;
      }
    }
    if (proposalId !== undefined) break;
  }
  if (proposalId === undefined) throw new Error("could not extract group proposal_id");
  await waitForTx(pRes.transactionHash);

  const voteMsg = {
    typeUrl: "/cosmos.group.v1.MsgVote",
    value: MsgVote.fromPartial({
      proposalId,
      voter: signerAddress,
      option: VoteOption.VOTE_OPTION_YES,
      exec: Exec.EXEC_TRY,
      metadata: "",
    }),
  };
  const vFee = await calculateFeeWithSimulation(client, signerAddress, [voteMsg], "vote");
  const vRes = await signAndBroadcastWithRetry(client, signerAddress, [voteMsg], vFee, "vote");
  if (vRes.code !== 0) throw new Error(`group vote failed: ${vRes.rawLog}`);
  await waitForTx(vRes.transactionHash);
  console.log(`  Group proposal ${proposalId} executed`);
}
