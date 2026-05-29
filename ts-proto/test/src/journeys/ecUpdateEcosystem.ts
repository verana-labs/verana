/**
 * Journey: EC Update Ecosystem (Operator-signed)
 *
 * Rotates an Ecosystem's DID. `language` is immutable per MOD-ES-MSG-2.
 * Depends on: test:de-grant-auth, test:ec-create
 *
 * Usage:
 *   npm run test:ec-update
 */

import {
  createAccountFromMnemonic,
  createSigningClient,
  getAccountInfo,
  calculateFeeWithSimulation,
  signAndBroadcastWithRetry,
  generateUniqueDID,
  config,
} from "../helpers/client";
import { typeUrls } from "../helpers/registry";
import { MsgUpdateEcosystem } from "../../../src/codec/verana/ec/v1/tx";
import { getEcAuthzSetup, getActiveEC } from "../helpers/journeyResults";

const COOLUSER_MNEMONIC =
  (process.env.MNEMONIC && process.env.MNEMONIC.trim()) ||
  "pink glory help gown abstract eight nice crazy forward ketchup skill cheese";

const OPERATOR_INDEX = 11;

async function main() {
  console.log("=".repeat(60));
  console.log("Journey: EC Update Ecosystem (Operator-signed)");
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

  // Update Ecosystem — only DID is mutable
  const newDid = generateUniqueDID();

  console.log("Updating Ecosystem...");
  console.log(`  New DID: ${newDid}`);

  const msg = {
    typeUrl: typeUrls.MsgUpdateEcosystem,
    value: MsgUpdateEcosystem.fromPartial({
      corporation: setup.authorityAddress,
      operator: account.address,
      id: activeEC.ecosystemId,
      did: newDid,
    }),
  };

  try {
    const fee = await calculateFeeWithSimulation(
      client, account.address, [msg],
      "Updating Ecosystem via operator",
    );

    const result = await signAndBroadcastWithRetry(
      client, account.address, [msg], fee,
      "Updating Ecosystem via operator",
    );

    if (result.code === 0) {
      console.log();
      console.log("✅ SUCCESS! Ecosystem updated!");
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
