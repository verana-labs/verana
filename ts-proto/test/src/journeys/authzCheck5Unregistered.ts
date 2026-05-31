/**
 * Journey: AUTHZ-CHECK-5 negative — unregistered Corporation rejected.
 *
 * Per spec v4-rc2 AUTHZ-CHECK-5, a delegable Msg whose signing `corporation`
 * account is NOT the policy_address of a registered Corporation MUST abort with
 * ErrCorporationNotRegistered (see MOD-CO-MSG-1).
 *
 * Probe: MsgGrantOperatorAuthorization with operator="" (self-grant). AUTHZ-CHECK-1
 * short-circuits for the empty operator (corporation acting alone), so
 * AUTHZ-CHECK-5 is the primary gate. The message's signer IS `corporation`, so a
 * fresh account that was never registered via MOD-CO-MSG-1 signs it directly
 * (no group proposal needed) and MUST be rejected.
 *
 * Signing: SIGN_MODE_LEGACY_AMINO_JSON (Secp256k1HdWallet) — strict, no exceptions.
 * MsgGrantOperatorAuthorization is amino-encoded via its converter in
 * ts-proto/src/amino-converter/de.ts (aminoType verana/x/de/MsgGrantOpAuthorization).
 *
 * Self-contained: does NOT depend on test:co-create. It derives a fresh,
 * never-registered account, funds it from cooluser, and asserts the grant is
 * rejected by AUTHZ-CHECK-5.
 *
 * Usage:
 *   npm run test:authz-check5
 */

import {
  createAccountFromMnemonic,
  createSigningClient,
  createQueryClient,
  getAccountInfo,
  signAndBroadcastWithRetry,
  fundAccount,
} from "../helpers/client";
import { typeUrls } from "../helpers/registry";
import { MsgGrantOperatorAuthorization } from "../../../src/codec/verana/de/v1/tx";

const COOLUSER_MNEMONIC =
  (process.env.MNEMONIC && process.env.MNEMONIC.trim()) ||
  "pink glory help gown abstract eight nice crazy forward ketchup skill cheese";

// A high derivation index that no journey ever registers as a Corporation.
const UNREGISTERED_INDEX = 25;

function isCorporationNotRegistered(text: string): boolean {
  return text.includes("Corporation") &&
    (text.includes("policy_address") || text.includes("not been registered"));
}

async function main() {
  console.log("=".repeat(60));
  console.log("Journey: AUTHZ-CHECK-5 negative (unregistered Corporation)");
  console.log("=".repeat(60));
  console.log();

  // Step 1: a fresh amino wallet that was never registered as a Corporation.
  console.log("Step 1: Creating an unregistered account (SIGN_MODE_LEGACY_AMINO_JSON)...");
  const wallet = await createAccountFromMnemonic(COOLUSER_MNEMONIC, UNREGISTERED_INDEX);
  const account = await getAccountInfo(wallet);
  console.log(`  Unregistered account: ${account.address} (derivation index ${UNREGISTERED_INDEX})`);
  console.log();

  // Step 2: fund it so the tx clears the ante handler and reaches the msg handler.
  console.log("Step 2: Funding the unregistered account...");
  const cooluserWallet = await createAccountFromMnemonic(COOLUSER_MNEMONIC, 0);
  const cooluserAccount = await getAccountInfo(cooluserWallet);
  const fundResult = await fundAccount(
    COOLUSER_MNEMONIC,
    cooluserAccount.address,
    account.address,
    "10000000uvna",
  );
  if (fundResult.code !== 0) {
    console.log(`  ❌ Failed to fund: ${fundResult.rawLog}`);
    process.exit(1);
  }
  const qc = await createQueryClient();
  for (let i = 0; i < 30; i++) {
    try {
      const tx = await qc.getTx(fundResult.transactionHash);
      if (tx) break;
    } catch {
      // not in a block yet
    }
    await new Promise((r) => setTimeout(r, 1000));
  }
  qc.disconnect();
  console.log(`  ✓ Funded`);
  console.log();

  // Step 3: self-grant from the unregistered account → expect AUTHZ-CHECK-5 abort.
  console.log("Step 3: Self-grant from unregistered corporation (expect ErrCorporationNotRegistered)...");
  const grantMsg = {
    typeUrl: typeUrls.MsgGrantOperatorAuthorization,
    value: MsgGrantOperatorAuthorization.fromPartial({
      corporation: account.address,
      operator: "", // self-grant: AUTHZ-CHECK-1 short-circuits, AUTHZ-CHECK-5 is the gate
      grantee: account.address,
      msgTypes: [typeUrls.MsgStoreDigest],
      withFeegrant: false,
    }),
  };

  const client = await createSigningClient(wallet);
  // Fixed fee (skip simulation) so the message reaches deliverTx and is rejected
  // there by AUTHZ-CHECK-5, rather than failing at the simulate step.
  const fee = { amount: [{ denom: "uvna", amount: "10000" }], gas: "300000" };

  let failed = false;
  let errText = "";
  try {
    const result = await signAndBroadcastWithRetry(
      client, account.address, [grantMsg], fee, "authz-check-5 negative self-grant",
    );
    if (result.code !== 0) {
      failed = true;
      errText = String(result.rawLog);
    }
  } catch (e: any) {
    failed = true;
    errText = String(e?.rawLog || e?.message || e);
  }

  if (!failed) {
    console.log("  ❌ Grant unexpectedly SUCCEEDED — AUTHZ-CHECK-5 was not enforced");
    process.exit(1);
  }
  if (!isCorporationNotRegistered(errText)) {
    console.log(`  ❌ Grant failed, but for the wrong reason:\n     ${errText}`);
    process.exit(1);
  }

  console.log("  ✅ AUTHZ-CHECK-5 correctly rejected the unregistered corporation:");
  console.log(`     ${errText.split("\n")[0]}`);
  console.log();
  console.log("=".repeat(60));
  console.log("Journey PASSED: AUTHZ-CHECK-5 negative path verified (amino).");
  console.log("=".repeat(60));
}

main().catch((e) => {
  console.error(e);
  process.exit(1);
});
