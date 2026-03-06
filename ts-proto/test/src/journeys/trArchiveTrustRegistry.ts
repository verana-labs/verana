/**
 * Journey: TR Archive Trust Registry (Operator-signed)
 *
 * Archives a Trust Registry.
 * Depends on: test:de-grant-auth, test:tr-create
 *
 * Usage:
 *   npm run test:tr-archive
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
import { MsgArchiveTrustRegistry } from "../../../src/codec/verana/tr/v1/tx";
import { getTrAuthzSetup, getActiveTR } from "../helpers/journeyResults";

const COOLUSER_MNEMONIC =
  (process.env.MNEMONIC && process.env.MNEMONIC.trim()) ||
  "pink glory help gown abstract eight nice crazy forward ketchup skill cheese";

const OPERATOR_INDEX = 11;

async function main() {
  console.log("=".repeat(60));
  console.log("Journey: TR Archive Trust Registry (Operator-signed)");
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

  // Archive Trust Registry
  console.log("Archiving Trust Registry...");
  const msg = {
    typeUrl: typeUrls.MsgArchiveTrustRegistry,
    value: MsgArchiveTrustRegistry.fromPartial({
      authority: setup.authorityAddress,
      operator: account.address,
      id: activeTR.trustRegistryId,
      archive: true,
    }),
  };

  try {
    const fee = await calculateFeeWithSimulation(
      client, account.address, [msg],
      "Archiving Trust Registry via operator",
    );

    const result = await signAndBroadcastWithRetry(
      client, account.address, [msg], fee,
      "Archiving Trust Registry via operator",
    );

    if (result.code === 0) {
      console.log();
      console.log("✅ SUCCESS! Trust Registry archived!");
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
