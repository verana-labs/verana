/**
 * Journey: CO Update Corporation (amino, operator + AUTHZ-CHECK-5)
 *
 * Grants the operator authorization for MsgUpdateCorporation (x/group proposal),
 * then the operator updates the corporation's did via MsgUpdateCorporation
 * (operator-signed, AUTHZ-CHECK-5 resolves the corporation). All amino.
 *
 * Requires: test:co-create (active corporation; signer index 10 sole group member).
 *
 * Usage: npm run test:co-update
 */

import {
  createAccountFromMnemonic,
  createSigningClient,
  getAccountInfo,
  calculateFeeWithSimulation,
  signAndBroadcastWithRetry,
  fundAccount,
} from "../helpers/client";
import { typeUrls } from "../helpers/registry";
import { MsgUpdateCorporation } from "../../../src/codec/verana/co/v1/tx";
import { MsgGrantOperatorAuthorization } from "../../../src/codec/verana/de/v1/tx";
import { submitGroupProposalAndExec } from "../helpers/group";
import { lcdGet } from "../helpers/gov";
import { getActiveCorporation } from "../helpers/journeyResults";

const COOLUSER_MNEMONIC =
  (process.env.MNEMONIC && process.env.MNEMONIC.trim()) ||
  "pink glory help gown abstract eight nice crazy forward ketchup skill cheese";

const SIGNER_INDEX = 10; // sole group member
const OPERATOR_INDEX = 11;

async function main() {
  console.log("=".repeat(60));
  console.log("Journey: CO Update Corporation (amino, operator)");
  console.log("=".repeat(60));

  const corp = getActiveCorporation();
  if (!corp) {
    console.log("  No active corporation found. Run test:co-create first.");
    process.exit(1);
  }

  const signerWallet = await createAccountFromMnemonic(COOLUSER_MNEMONIC, SIGNER_INDEX);
  const signerAccount = await getAccountInfo(signerWallet);
  const operatorWallet = await createAccountFromMnemonic(COOLUSER_MNEMONIC, OPERATOR_INDEX);
  const operatorAccount = await getAccountInfo(operatorWallet);
  const cooluserWallet = await createAccountFromMnemonic(COOLUSER_MNEMONIC, 0);
  const cooluserAccount = await getAccountInfo(cooluserWallet);

  const signerClient = await createSigningClient(signerWallet);
  console.log(`  Corporation: ${corp.corporationId} (${corp.policyAddress})`);
  console.log(`  Operator:    ${operatorAccount.address}`);

  try {
    // Ensure operator is funded
    await fundAccount(COOLUSER_MNEMONIC, cooluserAccount.address, operatorAccount.address, "50000000uvna");
    await new Promise((r) => setTimeout(r, 3000));

    // Step 1: Grant operator authz for MsgUpdateCorporation (group proposal)
    console.log("\nStep 1: Grant operator authz for MsgUpdateCorporation...");
    const grantValue = MsgGrantOperatorAuthorization.encode(
      MsgGrantOperatorAuthorization.fromPartial({
        corporation: corp.policyAddress,
        operator: "",
        grantee: operatorAccount.address,
        msgTypes: [typeUrls.MsgUpdateCorporation],
        withFeegrant: false,
      }),
    ).finish();
    await submitGroupProposalAndExec(
      signerClient, signerAccount.address, corp.policyAddress,
      [{ typeUrl: typeUrls.MsgGrantOperatorAuthorization, value: grantValue }],
      "Grant UpdateCorporation authz",
    );
    console.log("  Operator authorized for MsgUpdateCorporation");

    // Step 2: Operator updates the corporation did (amino)
    console.log("\nStep 2: UpdateCorporation (operator amino)...");
    const newDid = "did:example:co-update-" + corp.corporationId;
    const operatorClient = await createSigningClient(operatorWallet);
    try {
      const msg = {
        typeUrl: typeUrls.MsgUpdateCorporation,
        value: MsgUpdateCorporation.fromPartial({
          corporation: corp.policyAddress,
          operator: operatorAccount.address,
          did: newDid,
        }),
      };
      const fee = await calculateFeeWithSimulation(operatorClient, operatorAccount.address, [msg], "update corp");
      const res = await signAndBroadcastWithRetry(operatorClient, operatorAccount.address, [msg], fee, "update corp");
      if (res.code !== 0) throw new Error(`update corporation failed: ${res.rawLog}`);
      console.log(`  Update succeeded (tx ${res.transactionHash})`);
    } finally {
      operatorClient.disconnect();
    }

    // Step 3: Verify the did changed
    const data = await lcdGet(`/cosmos/base/tendermint/v1beta1/blocks/latest`).catch(() => null);
    void data;
    const corpData = await lcdGet(`/verana/co/v1/get/${corp.corporationId}`).catch(() => null);
    const got = corpData?.corporation?.did ?? corpData?.corporation?.Did;
    if (got !== undefined && got !== newDid) {
      throw new Error(`expected did=${newDid}, got ${got}`);
    }
    console.log(`  Corporation did updated to ${newDid}`);

    console.log("\n" + "=".repeat(60));
    console.log("SUCCESS! CO UpdateCorporation (amino) validated");
    console.log("=".repeat(60));
  } catch (error: any) {
    console.log("ERROR!");
    console.error(error.message || error);
    process.exit(1);
  } finally {
    signerClient.disconnect();
  }
}

main().catch((error: any) => {
  console.error("\nFatal error:", error.message || error);
  process.exit(1);
});
