/**
 * Journey: TR Update Trust Registry (Operator-signed)
 *
 * Updates a Trust Registry's DID and AKA fields.
 * Depends on: test:de-grant-auth, test:tr-create
 *
 * Usage:
 *   npm run test:tr-update
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
import { MsgUpdateTrustRegistry } from "../../../src/codec/verana/tr/v1/tx";
import { getTrAuthzSetup, getActiveTR } from "../helpers/journeyResults";

const COOLUSER_MNEMONIC =
  (process.env.MNEMONIC && process.env.MNEMONIC.trim()) ||
  "pink glory help gown abstract eight nice crazy forward ketchup skill cheese";

const OPERATOR_INDEX = 11;

async function main() {
  console.log("=".repeat(60));
  console.log("Journey: TR Update Trust Registry (Operator-signed)");
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

  // Update Trust Registry
  const newDid = generateUniqueDID();
  const newAka = "http://updated-ts-proto-trust-registry.com";

  console.log("Updating Trust Registry...");
  console.log(`  New DID: ${newDid}`);
  console.log(`  New AKA: ${newAka}`);

  const msg = {
    typeUrl: typeUrls.MsgUpdateTrustRegistry,
    value: MsgUpdateTrustRegistry.fromPartial({
      authority: setup.authorityAddress,
      operator: account.address,
      id: activeTR.trustRegistryId,
      did: newDid,
      aka: newAka,
    }),
  };

  try {
    const fee = await calculateFeeWithSimulation(
      client, account.address, [msg],
      "Updating Trust Registry via operator",
    );

    const result = await signAndBroadcastWithRetry(
      client, account.address, [msg], fee,
      "Updating Trust Registry via operator",
    );

    if (result.code === 0) {
      console.log();
      console.log("✅ SUCCESS! Trust Registry updated!");
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
