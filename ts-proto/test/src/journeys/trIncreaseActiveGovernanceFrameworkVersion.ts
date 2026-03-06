/**
 * Journey: TR Increase Active Governance Framework Version (Operator-signed)
 *
 * Increases the active GF version. Requires a GFD for the next version to exist.
 * Depends on: test:de-grant-auth, test:tr-create, test:tr-add-gfd
 *
 * Usage:
 *   npm run test:tr-increase-gf-version
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
import { MsgIncreaseActiveGovernanceFrameworkVersion } from "../../../src/codec/verana/tr/v1/tx";
import { getTrAuthzSetup, getActiveTR } from "../helpers/journeyResults";

const COOLUSER_MNEMONIC =
  (process.env.MNEMONIC && process.env.MNEMONIC.trim()) ||
  "pink glory help gown abstract eight nice crazy forward ketchup skill cheese";

const OPERATOR_INDEX = 11;

async function main() {
  console.log("=".repeat(60));
  console.log("Journey: TR Increase Active GF Version (Operator-signed)");
  console.log("=".repeat(60));
  console.log();

  const setup = getTrAuthzSetup();
  if (!setup) {
    console.log("  ❌ No authz setup found. Run test:de-grant-auth first.");
    process.exit(1);
  }

  const activeTR = getActiveTR();
  if (!activeTR) {
    console.log("  ❌ No active TR found. Run test:tr-create first.");
    process.exit(1);
  }

  console.log(`  Authority: ${setup.authorityAddress}`);
  console.log(`  TR ID:     ${activeTR.trustRegistryId}`);
  console.log();

  const wallet = await createAccountFromMnemonic(COOLUSER_MNEMONIC, OPERATOR_INDEX);
  const account = await getAccountInfo(wallet);
  const client = await createSigningClient(wallet);

  console.log(`  Operator:  ${account.address}`);
  console.log();

  // Increase active GF version (from 1 to 2)
  console.log("Increasing active Governance Framework version...");
  const msg = {
    typeUrl: typeUrls.MsgIncreaseActiveGovernanceFrameworkVersion,
    value: MsgIncreaseActiveGovernanceFrameworkVersion.fromPartial({
      authority: setup.authorityAddress,
      operator: account.address,
      id: activeTR.trustRegistryId,
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
