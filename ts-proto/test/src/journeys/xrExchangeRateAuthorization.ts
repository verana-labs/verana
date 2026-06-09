/**
 * Journey: XR ExchangeRateAuthorization + SetExchangeRateState (gov + operator, amino)
 *
 * Exercises the v4-rc3 XR flow entirely in legacy amino:
 *   1. CreateExchangeRate via gov proposal
 *   2. GrantExchangeRateAuthorization via gov proposal (amino-signed timestamp)
 *   3. UpdateExchangeRate by the authorized operator - expect success
 *   4. RevokeExchangeRateAuthorization via gov proposal
 *   5. UpdateExchangeRate after revoke - expect failure
 *   6. SetExchangeRateState (gov) - deactivate the rate
 *
 * Self-contained: the validator account (index 0) is proposer, voter, and operator.
 * Requires a running chain with a short gov voting period (local sim: 30s).
 *
 * Usage: npm run test:xr-authz
 */

import {
  createAccountFromMnemonic,
  createSigningClient,
  getAccountInfo,
  calculateFeeWithSimulation,
  signAndBroadcastWithRetry,
} from "../helpers/client";
import { typeUrls } from "../helpers/registry";
import {
  MsgCreateExchangeRate,
  MsgGrantExchangeRateAuthorization,
  MsgRevokeExchangeRateAuthorization,
  MsgUpdateExchangeRate,
  MsgSetExchangeRateState,
} from "../../../src/codec/verana/xr/v1/tx";
import { PricingAssetType } from "../../../src/codec/verana/cs/v1/types";
import { getGovModuleAddress, lcdGet, submitGovProposalAndPass } from "../helpers/gov";

const COOLUSER_MNEMONIC =
  (process.env.MNEMONIC && process.env.MNEMONIC.trim()) ||
  "pink glory help gown abstract eight nice crazy forward ketchup skill cheese";

// Distinct asset pair to avoid colliding with the Go harness journey 601 (TU/uvna).
const BASE_ASSET_TYPE = PricingAssetType.COIN;
const BASE_ASSET = "uvna";
const QUOTE_ASSET_TYPE = PricingAssetType.FIAT;
const QUOTE_ASSET = "EUR";

async function getRateIdByPair(): Promise<number> {
  const qs =
    `?base_asset_type=${BASE_ASSET_TYPE}&base_asset=${BASE_ASSET}` +
    `&quote_asset_type=${QUOTE_ASSET_TYPE}&quote_asset=${QUOTE_ASSET}`;
  const data = await lcdGet(`/verana/xr/v1/get${qs}`);
  const xr = data.exchange_rate ?? data.exchangeRate;
  if (!xr?.id) throw new Error(`could not resolve exchange rate id: ${JSON.stringify(data)}`);
  return Number(xr.id);
}

async function main() {
  console.log("=".repeat(60));
  console.log("Journey: XR ExchangeRateAuthorization + SetState (gov + operator amino)");
  console.log("=".repeat(60));

  const wallet = await createAccountFromMnemonic(COOLUSER_MNEMONIC, 0);
  const account = await getAccountInfo(wallet);
  const client = await createSigningClient(wallet);
  const govAddr = await getGovModuleAddress();
  console.log(`  Validator/operator: ${account.address}`);
  console.log(`  Gov module:         ${govAddr}`);

  try {
    // Step 1: Create exchange rate via gov proposal
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
        validityDuration: { seconds: 24 * 3600, nanos: 0 },
      }),
    ).finish();
    await submitGovProposalAndPass(client, account.address, typeUrls.MsgCreateExchangeRate, createValue,
      "Create uvna/EUR rate", "Create exchange rate for amino journey");

    const rateId = await getRateIdByPair();
    console.log(`  Exchange rate id: ${rateId}`);

    // Step 2: Grant ExchangeRateAuthorization via gov (amino + timestamp)
    console.log("\nStep 2: GrantExchangeRateAuthorization via gov proposal (amino timestamp)...");
    const grantValue = MsgGrantExchangeRateAuthorization.encode(
      MsgGrantExchangeRateAuthorization.fromPartial({
        authority: govAddr,
        xrId: rateId,
        operator: account.address,
        expiration: new Date(Date.now() + 24 * 3600 * 1000),
      }),
    ).finish();
    await submitGovProposalAndPass(client, account.address, typeUrls.MsgGrantExchangeRateAuthorization, grantValue,
      "Grant XR authz", "Authorize operator to update the rate");

    // Step 3: Operator updates the rate (amino) - expect success
    console.log("\nStep 3: UpdateExchangeRate by authorized operator (amino)...");
    const updateMsg = {
      typeUrl: typeUrls.MsgUpdateExchangeRate,
      value: MsgUpdateExchangeRate.fromPartial({ operator: account.address, id: rateId, rate: "2000000" }),
    };
    const upFee = await calculateFeeWithSimulation(client, account.address, [updateMsg], "update");
    const upRes = await signAndBroadcastWithRetry(client, account.address, [updateMsg], upFee, "update");
    if (upRes.code !== 0) throw new Error(`update failed: ${upRes.rawLog}`);
    console.log(`  Update succeeded (tx ${upRes.transactionHash})`);

    // Step 4: Revoke ExchangeRateAuthorization via gov
    console.log("\nStep 4: RevokeExchangeRateAuthorization via gov proposal...");
    const revokeValue = MsgRevokeExchangeRateAuthorization.encode(
      MsgRevokeExchangeRateAuthorization.fromPartial({
        authority: govAddr,
        xrId: rateId,
        operator: account.address,
      }),
    ).finish();
    await submitGovProposalAndPass(client, account.address, typeUrls.MsgRevokeExchangeRateAuthorization, revokeValue,
      "Revoke XR authz", "Revoke operator authorization");

    // Step 5: Update after revoke - expect failure
    console.log("\nStep 5: UpdateExchangeRate after revoke (expect failure)...");
    const upMsg2 = {
      typeUrl: typeUrls.MsgUpdateExchangeRate,
      value: MsgUpdateExchangeRate.fromPartial({ operator: account.address, id: rateId, rate: "3000000" }),
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

    // Step 6: SetExchangeRateState (gov) - deactivate the rate
    console.log("\nStep 6: SetExchangeRateState (gov) - deactivate...");
    const setStateValue = MsgSetExchangeRateState.encode(
      MsgSetExchangeRateState.fromPartial({ authority: govAddr, id: rateId, state: false }),
    ).finish();
    await submitGovProposalAndPass(client, account.address, typeUrls.MsgSetExchangeRateState, setStateValue,
      "Disable XR", "Set exchange rate state to inactive");
    const after = await lcdGet(`/verana/xr/v1/get?id=${rateId}`);
    const xrAfter = after.exchange_rate ?? after.exchangeRate;
    if (xrAfter?.state !== false) throw new Error(`expected state=false after SetExchangeRateState, got ${xrAfter?.state}`);
    console.log("  Exchange rate state set to inactive");

    console.log("\n" + "=".repeat(60));
    console.log("SUCCESS! XR amino flow validated:");
    console.log("  - CreateExchangeRate / Grant / Update / Revoke / reject-after-revoke / SetState");
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
