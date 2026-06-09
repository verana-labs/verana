/**
 * Shared cosmos x/gov v1 proposal helper for ts-proto journeys (amino).
 * Submits a governance proposal wrapping a single inner message, votes YES from
 * the validator, and polls the LCD until the proposal passes. The inner message
 * is amino-encoded via the gov MsgSubmitProposal converter (recursive Any).
 */
import {
  calculateFeeWithSimulation,
  signAndBroadcastWithRetry,
  config,
} from "./client";
import { MsgSubmitProposal, MsgVote } from "cosmjs-types/cosmos/gov/v1/tx";
import { VoteOption } from "cosmjs-types/cosmos/gov/v1/gov";

const sleep = (ms: number) => new Promise((r) => setTimeout(r, ms));

export async function lcdGet(path: string): Promise<any> {
  const res = await fetch(`${config.lcdEndpoint}${path}`);
  if (!res.ok) {
    throw new Error(`LCD ${path} -> HTTP ${res.status}: ${await res.text()}`);
  }
  return res.json();
}

export async function getGovModuleAddress(): Promise<string> {
  const data = await lcdGet("/cosmos/auth/v1beta1/module_accounts/gov");
  const acct = data.account?.base_account ?? data.account?.baseAccount ?? data.account;
  const addr = acct?.address;
  if (!addr) throw new Error(`could not resolve gov module address: ${JSON.stringify(data)}`);
  return addr;
}

function extractProposalId(result: any): bigint {
  for (const event of result.events || []) {
    for (const attr of event.attributes || []) {
      if (attr.key === "proposal_id") {
        return BigInt(String(attr.value).replace(/"/g, ""));
      }
    }
  }
  throw new Error("could not extract proposal_id from submit result events");
}

async function waitProposalPassed(proposalId: bigint): Promise<void> {
  for (let i = 0; i < 30; i++) {
    await sleep(3000);
    const data = await lcdGet(`/cosmos/gov/v1/proposals/${proposalId}`);
    const status = data.proposal?.status as string;
    if (status && status !== "PROPOSAL_STATUS_VOTING_PERIOD" && status !== "PROPOSAL_STATUS_DEPOSIT_PERIOD") {
      if (status !== "PROPOSAL_STATUS_PASSED") {
        throw new Error(`proposal ${proposalId} did not pass: ${status}`);
      }
      return;
    }
  }
  throw new Error(`proposal ${proposalId} still pending after timeout`);
}

/** Submit a gov proposal with one inner message (amino), vote YES, wait to pass. */
export async function submitGovProposalAndPass(
  client: any,
  address: string,
  innerTypeUrl: string,
  innerValue: Uint8Array,
  title: string,
  summary: string,
): Promise<void> {
  const submit = {
    typeUrl: "/cosmos.gov.v1.MsgSubmitProposal",
    value: MsgSubmitProposal.fromPartial({
      messages: [{ typeUrl: innerTypeUrl, value: innerValue }],
      initialDeposit: [{ denom: "uvna", amount: "10000000" }],
      proposer: address,
      metadata: "ipfs://CID",
      title,
      summary,
    }),
  };
  const submitFee = await calculateFeeWithSimulation(client, address, [submit], title);
  const submitRes = await signAndBroadcastWithRetry(client, address, [submit], submitFee, title);
  if (submitRes.code !== 0) throw new Error(`submit failed: ${submitRes.rawLog}`);
  const proposalId = extractProposalId(submitRes);
  console.log(`  Proposal ${proposalId} submitted, voting YES...`);
  await sleep(2000);

  const vote = {
    typeUrl: "/cosmos.gov.v1.MsgVote",
    value: MsgVote.fromPartial({
      proposalId,
      voter: address,
      option: VoteOption.VOTE_OPTION_YES,
      metadata: "",
    }),
  };
  const voteFee = await calculateFeeWithSimulation(client, address, [vote], "vote");
  const voteRes = await signAndBroadcastWithRetry(client, address, [vote], voteFee, "vote");
  if (voteRes.code !== 0) throw new Error(`vote failed: ${voteRes.rawLog}`);
  console.log(`  Voted YES on ${proposalId}, waiting for it to pass...`);
  await waitProposalPassed(proposalId);
  console.log(`  Proposal ${proposalId} PASSED`);
}
