/**
 * Journey: XR ExchangeRateAuthorization (gov + operator, SIGN_MODE_LEGACY_AMINO_JSON)
 *
 * Exercises the v4-rc3 XR authorization flow entirely in legacy amino:
 *   1. CreateExchangeRate via gov proposal (amino)
 *   2. GrantExchangeRateAuthorization via gov proposal (amino) - validates the
 *      grant message amino encoding INCLUDING its expiration timestamp, which is
 *      the first amino-signed Timestamp exercised by this suite.
 *   3. UpdateExchangeRate by the authorized operator (amino) - expect success
 *   4. RevokeExchangeRateAuthorization via gov proposal (amino)
 *   5. UpdateExchangeRate after revoke - expect failure
 *
 * Self-contained: the validator account (cooluser, index 0) is proposer, voter,
 * and the authorized operator. Requires a running chain with a short gov voting
 * period (the local sim chain uses 30s).
 *
 * Usage:
 *   npm run test:xr-authz
 */

import {
  createAccountFromMnemonic,
  createSigningClient,
  getAccountInfo,
  calculateFeeWithSimulation,
  signAndBroadcastWithRetry,
  config,
} from "../helpers/client";
import { typeUrls } from "../helpers/registry";
import {
  MsgCreateExchangeRate,
  MsgGrantExchangeRateAuthorization,
  MsgRevokeExchangeRateAuthorization,
  MsgUpdateExchangeRate,
} from "../../../src/codec/verana/xr/v1/tx";
import { PricingAssetType } from "../../../src/codec/verana/cs/v1/types";
import { MsgSubmitProposal, MsgVote } from "cosmjs-types/cosmos/gov/v1/tx";
import { VoteOption } from "cosmjs-types/cosmos/gov/v1/gov";

const COOLUSER_MNEMONIC =
  (process.env.MNEMONIC && process.env.MNEMONIC.trim()) ||
  "pink glory help gown abstract eight nice crazy forward ketchup skill cheese";

// Distinct asset pair to avoid colliding with the Go harness journey 601 (TU/uvna).
const BASE_ASSET_TYPE = PricingAssetType.COIN;
const BASE_ASSET = "uvna";
const QUOTE_ASSET_TYPE = PricingAssetType.FIAT;
const QUOTE_ASSET = "EUR";

const sleep = (ms: number) => new Promise((r) => setTimeout(r, ms));

async function lcd(path: string): Promise<any> {
  const res = await fetch(`${config.lcdEndpoint}${path}`);
  if (!res.ok) {
    throw new Error(`LCD ${path} -> HTTP ${res.status}: ${await res.text()}`);
  }
  return res.json();
}

async function getGovModuleAddress(): Promise<string> {
  const data = await lcd("/cosmos/auth/v1beta1/module_accounts/gov");
  // base_account is nested under the module account.
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
  // Local chain voting period is 30s; poll well past it.
  for (let i = 0; i < 30; i++) {
    await sleep(3000);
    const data = await lcd(`/cosmos/gov/v1/proposals/${proposalId}`);
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

async function submitAndPass(
  client: any,
  address: string,
  govAddr: string,
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

async function getRateIdByPair(): Promise<number> {
  const qs =
    `?base_asset_type=${BASE_ASSET_TYPE}&base_asset=${BASE_ASSET}` +
    `&quote_asset_type=${QUOTE_ASSET_TYPE}&quote_asset=${QUOTE_ASSET}`;
  const data = await lcd(`/verana/xr/v1/get${qs}`);
  const xr = data.exchange_rate ?? data.exchangeRate;
  if (!xr?.id) throw new Error(`could not resolve exchange rate id: ${JSON.stringify(data)}`);
  return Number(xr.id);
}

async function main() {
  console.log("=".repeat(60));
  console.log("Journey: XR ExchangeRateAuthorization (gov + operator amino)");
  console.log("=".repeat(60));

  const wallet = await createAccountFromMnemonic(COOLUSER_MNEMONIC, 0);
  const account = await getAccountInfo(wallet);
  const client = await createSigningClient(wallet);
  const govAddr = await getGovModuleAddress();
  console.log(`  Validator/operator: ${account.address}`);
  console.log(`  Gov module:         ${govAddr}`);

  try {
    // Step 1: Create exchange rate via gov proposal (amino)
    console.log("\nStep 1: CreateExchangeRate via gov proposal...");
    const createValue = MsgCreateExchangeRate.encode(
      MsgCreateExchangeRate.fromPartial({
        authority: govAddr,
        baseAssetType: BASE_ASSET_TYPE,
        baseAsset: BASE_ASSET,
        quoteAssetType: QUOTE_ASSET_TYPE,
        quoteAsset: QUOTE_ASSET,
        rate: "1000000",
        rateScale: 6,
        // ts-proto-generated Duration uses `seconds: number` (not bigint).
        validityDuration: { seconds: 24 * 3600, nanos: 0 },
      }),
    ).finish();
    await submitAndPass(client, account.address, govAddr, typeUrls.MsgCreateExchangeRate, createValue,
      "Create uvna/EUR rate", "Create exchange rate for amino journey");

    const rateId = await getRateIdByPair();
    console.log(`  Exchange rate id: ${rateId}`);

    // Step 2: Grant ExchangeRateAuthorization via gov proposal (amino + timestamp)
    console.log("\nStep 2: GrantExchangeRateAuthorization via gov proposal (validates amino timestamp)...");
    const grantValue = MsgGrantExchangeRateAuthorization.encode(
      MsgGrantExchangeRateAuthorization.fromPartial({
        authority: govAddr,
        xrId: rateId,
        operator: account.address,
        expiration: new Date(Date.now() + 24 * 3600 * 1000),
      }),
    ).finish();
    await submitAndPass(client, account.address, govAddr, typeUrls.MsgGrantExchangeRateAuthorization, grantValue,
      "Grant XR authz", "Authorize operator to update the rate");

    // Step 3: Operator updates the rate (amino) - expect success
    console.log("\nStep 3: UpdateExchangeRate by authorized operator (amino)...");
    const updateMsg = {
      typeUrl: typeUrls.MsgUpdateExchangeRate,
      value: MsgUpdateExchangeRate.fromPartial({
        operator: account.address,
        id: rateId,
        rate: "2000000",
      }),
    };
    const upFee = await calculateFeeWithSimulation(client, account.address, [updateMsg], "update");
    const upRes = await signAndBroadcastWithRetry(client, account.address, [updateMsg], upFee, "update");
    if (upRes.code !== 0) throw new Error(`update failed: ${upRes.rawLog}`);
    console.log(`  Update succeeded (tx ${upRes.transactionHash})`);

    // Step 4: Revoke ExchangeRateAuthorization via gov proposal (amino)
    console.log("\nStep 4: RevokeExchangeRateAuthorization via gov proposal...");
    const revokeValue = MsgRevokeExchangeRateAuthorization.encode(
      MsgRevokeExchangeRateAuthorization.fromPartial({
        authority: govAddr,
        xrId: rateId,
        operator: account.address,
      }),
    ).finish();
    await submitAndPass(client, account.address, govAddr, typeUrls.MsgRevokeExchangeRateAuthorization, revokeValue,
      "Revoke XR authz", "Revoke operator authorization");

    // Step 5: Update after revoke - expect failure
    console.log("\nStep 5: UpdateExchangeRate after revoke (expect failure)...");
    const upMsg2 = {
      typeUrl: typeUrls.MsgUpdateExchangeRate,
      value: MsgUpdateExchangeRate.fromPartial({
        operator: account.address,
        id: rateId,
        rate: "3000000",
      }),
    };
    let rejected = false;
    try {
      const upFee2 = await calculateFeeWithSimulation(client, account.address, [upMsg2], "update2");
      const upRes2 = await signAndBroadcastWithRetry(client, account.address, [upMsg2], upFee2, "update2");
      if (upRes2.code !== 0) rejected = true;
    } catch (_e) {
      rejected = true;
    }
    if (!rejected) throw new Error("update after revoke unexpectedly succeeded");
    console.log("  Update correctly rejected after revoke");

    console.log("\n" + "=".repeat(60));
    console.log("SUCCESS! XR authorization amino flow validated:");
    console.log("  - CreateExchangeRate (gov, amino)");
    console.log("  - GrantExchangeRateAuthorization (gov, amino, timestamp)");
    console.log("  - UpdateExchangeRate (operator, amino) succeeded");
    console.log("  - RevokeExchangeRateAuthorization (gov, amino)");
    console.log("  - UpdateExchangeRate rejected after revoke");
    console.log("=".repeat(60));
  } catch (error: any) {
    console.log("ERROR!");
    console.error(error.message || error);
    process.exit(1);
  } finally {
    client.disconnect();
  }
}

main().catch((error: any) => {
  console.error("\nFatal error:", error.message || error);
  process.exit(1);
});
