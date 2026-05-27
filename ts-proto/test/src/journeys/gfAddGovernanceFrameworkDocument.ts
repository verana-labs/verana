/**
 * Journey: GF Add Governance Framework Document (Operator-signed)
 *
 * Adds a GFD to an existing Ecosystem. Moved from `verana.tr.v1` to
 * `verana.gf.v1` in v4-rc2 and now uses `ecosystem_id` (replacing `tr_id`).
 * When `ecosystem_id == 0`, the GF target is the signing Corporation's own
 * CGF instead.
 *
 * Depends on: test:de-grant-auth, test:ec-create
 *
 * Usage:
 *   npm run test:gf-add-doc
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
import { MsgAddGovernanceFrameworkDocument } from "../../../src/codec/verana/gf/v1/tx";
import { getEcAuthzSetup, getActiveEC } from "../helpers/journeyResults";

const COOLUSER_MNEMONIC =
  (process.env.MNEMONIC && process.env.MNEMONIC.trim()) ||
  "pink glory help gown abstract eight nice crazy forward ketchup skill cheese";

const OPERATOR_INDEX = 11;

async function main() {
  console.log("=".repeat(60));
  console.log("Journey: GF Add Governance Framework Document (Operator-signed)");
  console.log("=".repeat(60));
  console.log();

  const setup = getEcAuthzSetup();
  if (!setup) {
    console.log("  ❌ No EC authz setup found. Run test:de-grant-auth first.");
    process.exit(1);
  }

  const activeEC = getActiveEC();
  if (!activeEC) {
    console.log("  ❌ No active EC found. Run test:ec-create first.");
    process.exit(1);
  }

  console.log(`  Corporation: ${setup.authorityAddress}`);
  console.log(`  EC ID:       ${activeEC.ecosystemId}`);
  console.log();

  const wallet = await createAccountFromMnemonic(COOLUSER_MNEMONIC, OPERATOR_INDEX);
  const account = await getAccountInfo(wallet);
  const client = await createSigningClient(wallet);

  console.log(`  Operator:  ${account.address}`);
  console.log();

  // Add GFD for version 2
  console.log("Adding Governance Framework Document for version 2...");
  const msg = {
    typeUrl: typeUrls.MsgAddGovernanceFrameworkDocument,
    value: MsgAddGovernanceFrameworkDocument.fromPartial({
      corporation: setup.authorityAddress,
      operator: account.address,
      ecosystemId: activeEC.ecosystemId,
      docLanguage: "en",
      docUrl: "https://example.com/governance-framework-v2.pdf",
      docDigestSri: "sha384-TsProtoTestDocHash1234567890123456789012345678901234567890123456789012345678",
      version: 2,
    }),
  };

  try {
    const fee = await calculateFeeWithSimulation(
      client, account.address, [msg],
      "Adding GFD via operator",
    );

    const result = await signAndBroadcastWithRetry(
      client, account.address, [msg], fee,
      "Adding GFD via operator",
    );

    if (result.code === 0) {
      console.log();
      console.log("✅ SUCCESS! Governance Framework Document added!");
      console.log(`  Tx Hash: ${result.transactionHash}`);
      console.log(`  Block:   ${result.height}`);
      console.log(`  Gas:     ${result.gasUsed}/${result.gasWanted}`);
    } else {
      console.log("❌ FAILED!");
      console.log(`  Code: ${result.code}`);
      console.log(`  Log:  ${result.rawLog}`);
      process.exit(1);
    }
  } catch (error: any) {
    console.log("❌ ERROR!");
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
  process.exit(1);
});
