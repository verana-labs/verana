/**
 * Journey: DE Revoke Operator Authorization (amino, x/group)
 *
 * Grants an operator authorization to a fresh grantee, then revokes it, both via
 * x/group proposals signed with SIGN_MODE_LEGACY_AMINO_JSON. Covers
 * de.MsgRevokeOperatorAuthorization.
 *
 * Requires: test:co-create (active corporation; signer index 10 is the sole group member).
 *
 * Usage: npm run test:de-revoke-auth
 */

import {
  createAccountFromMnemonic,
  createSigningClient,
  getAccountInfo,
} from "../helpers/client";
import { typeUrls } from "../helpers/registry";
import {
  MsgGrantOperatorAuthorization,
  MsgRevokeOperatorAuthorization,
} from "../../../src/codec/verana/de/v1/tx";
import { submitGroupProposalAndExec } from "../helpers/group";
import { getActiveCorporation } from "../helpers/journeyResults";

const COOLUSER_MNEMONIC =
  (process.env.MNEMONIC && process.env.MNEMONIC.trim()) ||
  "pink glory help gown abstract eight nice crazy forward ketchup skill cheese";

const SIGNER_INDEX = 10; // sole group member
const GRANTEE_INDEX = 12; // fresh grantee (authz record only; never signs)

async function main() {
  console.log("=".repeat(60));
  console.log("Journey: DE Revoke Operator Authorization (amino, x/group)");
  console.log("=".repeat(60));

  const corp = getActiveCorporation();
  if (!corp) {
    console.log("  No active corporation found. Run test:co-create first.");
    process.exit(1);
  }

  const signerWallet = await createAccountFromMnemonic(COOLUSER_MNEMONIC, SIGNER_INDEX);
  const signerAccount = await getAccountInfo(signerWallet);
  const granteeWallet = await createAccountFromMnemonic(COOLUSER_MNEMONIC, GRANTEE_INDEX);
  const granteeAccount = await getAccountInfo(granteeWallet);
  const client = await createSigningClient(signerWallet);

  console.log(`  Corporation: ${corp.policyAddress}`);
  console.log(`  Grantee:     ${granteeAccount.address}`);

  const msgType = typeUrls.MsgCreateEcosystem;

  try {
    // Step 1: Grant operator authorization to the grantee (group proposal)
    console.log("\nStep 1: Grant operator authorization to grantee...");
    const grantValue = MsgGrantOperatorAuthorization.encode(
      MsgGrantOperatorAuthorization.fromPartial({
        corporation: corp.policyAddress,
        operator: "",
        grantee: granteeAccount.address,
        msgTypes: [msgType],
        withFeegrant: false,
      }),
    ).finish();
    await submitGroupProposalAndExec(
      client, signerAccount.address, corp.policyAddress,
      [{ typeUrl: typeUrls.MsgGrantOperatorAuthorization, value: grantValue }],
      "Grant operator authz",
    );
    console.log("  Granted");

    // Step 2: Revoke that operator authorization (group proposal)
    console.log("\nStep 2: Revoke operator authorization...");
    const revokeValue = MsgRevokeOperatorAuthorization.encode(
      MsgRevokeOperatorAuthorization.fromPartial({
        corporation: corp.policyAddress,
        operator: "",
        grantee: granteeAccount.address,
      }),
    ).finish();
    await submitGroupProposalAndExec(
      client, signerAccount.address, corp.policyAddress,
      [{ typeUrl: typeUrls.MsgRevokeOperatorAuthorization, value: revokeValue }],
      "Revoke operator authz",
    );
    console.log("  Revoked");

    console.log("\n" + "=".repeat(60));
    console.log("SUCCESS! DE grant + revoke operator authorization (amino)");
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
