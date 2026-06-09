/**
 * Journey: TD SlashTrustDeposit (governance, amino)
 *
 * Slashes the active corporation's trust deposit by corporation_id via a gov
 * proposal (td.MsgSlashTrustDeposit, signer = gov authority), legacy amino.
 *
 * Requires: test:co-create (active corporation) and prior perm flows so the
 * corporation has a non-zero trust deposit.
 *
 * Usage: npm run test:td-slash
 */

import {
  createAccountFromMnemonic,
  createSigningClient,
  getAccountInfo,
} from "../helpers/client";
import { typeUrls } from "../helpers/registry";
import { MsgSlashTrustDeposit } from "../../../src/codec/verana/td/v1/tx";
import { getGovModuleAddress, lcdGet, submitGovProposalAndPass } from "../helpers/gov";
import { getActiveCorporation } from "../helpers/journeyResults";

const COOLUSER_MNEMONIC =
  (process.env.MNEMONIC && process.env.MNEMONIC.trim()) ||
  "pink glory help gown abstract eight nice crazy forward ketchup skill cheese";

async function main() {
  console.log("=".repeat(60));
  console.log("Journey: TD SlashTrustDeposit (governance, amino)");
  console.log("=".repeat(60));

  const corp = getActiveCorporation();
  if (!corp) {
    console.log("  No active corporation found. Run test:co-create + perm flows first.");
    process.exit(1);
  }
  const corpId = Number(corp.corporationId);

  const wallet = await createAccountFromMnemonic(COOLUSER_MNEMONIC, 0);
  const account = await getAccountInfo(wallet);
  const client = await createSigningClient(wallet);
  const govAddr = await getGovModuleAddress();
  console.log(`  Corporation id: ${corpId} (${corp.policyAddress})`);

  try {
    // Read the corporation's trust deposit
    const before = await lcdGet(`/verana/td/v1/get/${corpId}`);
    const tdBefore = before.trust_deposit ?? before.trustDeposit;
    const depositBefore = Number(tdBefore?.deposit ?? 0);
    const slashedBefore = Number(tdBefore?.slashed_deposit ?? tdBefore?.slashedDeposit ?? 0);
    console.log(`  Deposit: ${depositBefore}, SlashedDeposit: ${slashedBefore}`);
    if (depositBefore === 0) throw new Error("corporation has no trust deposit to slash");

    const slashAmount = Math.max(1, Math.floor(depositBefore / 10));
    console.log(`  Slashing amount: ${slashAmount}`);

    const slashValue = MsgSlashTrustDeposit.encode(
      MsgSlashTrustDeposit.fromPartial({
        authority: govAddr,
        corporationId: corpId,
        deposit: String(slashAmount),
        reason: "coverage journey gov slash",
      }),
    ).finish();
    await submitGovProposalAndPass(client, account.address, typeUrls.MsgSlashTrustDeposit, slashValue,
      "Slash trust deposit", "Governance slash by corporation_id");

    const after = await lcdGet(`/verana/td/v1/get/${corpId}`);
    const tdAfter = after.trust_deposit ?? after.trustDeposit;
    const slashedAfter = Number(tdAfter?.slashed_deposit ?? tdAfter?.slashedDeposit ?? 0);
    console.log(`  SlashedDeposit: ${slashedBefore} -> ${slashedAfter}`);
    if (slashedAfter <= slashedBefore) throw new Error(`slashed_deposit did not increase (${slashedBefore} -> ${slashedAfter})`);

    console.log("\n" + "=".repeat(60));
    console.log("SUCCESS! TD SlashTrustDeposit (gov, amino) validated");
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
