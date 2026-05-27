/**
 * Journey: GF Increase Active Governance Framework Version (Operator-signed)
 *
 * Activates the next governance-framework version for an Ecosystem. Moved
 * from `verana.tr.v1` to `verana.gf.v1` in v4-rc2; `ecosystem_id == 0`
 * targets the signing Corporation's own CGF.
 *
 * Depends on: test:de-grant-auth, test:ec-create, test:gf-add-doc
 *
 * Usage:
 *   npm run test:gf-increase-version
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
import { MsgIncreaseActiveGovernanceFrameworkVersion } from "../../../src/codec/verana/gf/v1/tx";
import { getEcAuthzSetup, getActiveEC } from "../helpers/journeyResults";

const COOLUSER_MNEMONIC =
  (process.env.MNEMONIC && process.env.MNEMONIC.trim()) ||
  "pink glory help gown abstract eight nice crazy forward ketchup skill cheese";

const OPERATOR_INDEX = 11;

async function main() {
  console.log("=".repeat(60));
  console.log("Journey: GF Increase Active GF Version (Operator-signed)");
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

  console.log("Increasing active Governance Framework version...");
  const msg = {
    typeUrl: typeUrls.MsgIncreaseActiveGovernanceFrameworkVersion,
    value: MsgIncreaseActiveGovernanceFrameworkVersion.fromPartial({
      corporation: setup.authorityAddress,
      operator: account.address,
      ecosystemId: activeEC.ecosystemId,
    }),
  };

  try {
    const fee = await calculateFeeWithSimulation(
      client, account.address, [msg],
      "Increasing GF version via operator",
    );

    const result = await signAndBroadcastWithRetry(
      client, account.address, [msg], fee,
      "Increasing GF version via operator",
    );

    if (result.code === 0) {
      console.log();
      console.log("✅ SUCCESS! Active GF version increased!");
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
