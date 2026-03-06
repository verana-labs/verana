/**
 * Journey: TR Add Governance Framework Document (Operator-signed)
 *
 * Adds a GFD to an existing Trust Registry. Requires:
 * - DE grant operator auth journey (test:de-grant-auth)
 * - TR create journey (test:tr-create)
 *
 * Usage:
 *   npm run test:tr-add-gfd
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
import { MsgAddGovernanceFrameworkDocument } from "../../../src/codec/verana/tr/v1/tx";
import { getTrAuthzSetup, getActiveTR } from "../helpers/journeyResults";

const COOLUSER_MNEMONIC =
  (process.env.MNEMONIC && process.env.MNEMONIC.trim()) ||
  "pink glory help gown abstract eight nice crazy forward ketchup skill cheese";

const OPERATOR_INDEX = 11;

async function main() {
  console.log("=".repeat(60));
  console.log("Journey: TR Add Governance Framework Document (Operator-signed)");
  console.log("=".repeat(60));
  console.log();

  // Load setup
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

  // Setup operator
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
      authority: setup.authorityAddress,
      operator: account.address,
      id: activeTR.trustRegistryId,
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
