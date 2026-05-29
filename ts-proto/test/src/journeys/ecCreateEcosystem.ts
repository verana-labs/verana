/**
 * Journey: EC Create Ecosystem (Operator-signed)
 *
 * The operator signs MsgCreateEcosystem on behalf of the Corporation. The
 * Corporation policy_address (from coCreateCorporation) is the `corporation`
 * field; AUTHZ-CHECK-5 verifies it resolves to a registered MOD-CO entry.
 *
 * Replaces the v3 `trCreateTrustRegistry` journey. Drops `aka`. Extracts
 * `ecosystem_id` from `create_ecosystem` events.
 *
 * Requires: test:de-grant-auth must be run first.
 *
 * Usage:
 *   npm run test:ec-create
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
import { MsgCreateEcosystem } from "../../../src/codec/verana/ec/v1/tx";
import { getEcAuthzSetup, saveActiveEC } from "../helpers/journeyResults";

const COOLUSER_MNEMONIC =
  (process.env.MNEMONIC && process.env.MNEMONIC.trim()) ||
  "pink glory help gown abstract eight nice crazy forward ketchup skill cheese";

const OPERATOR_INDEX = 11;

async function main() {
  console.log("=".repeat(60));
  console.log("Journey: EC Create Ecosystem (Operator-signed)");
  console.log("=".repeat(60));
  console.log();

  // Step 1: Load EC authz setup
  console.log("Step 1: Loading EC authz setup...");
  const setup = getEcAuthzSetup();
  if (!setup) {
    console.log("  ❌ No EC authz setup found. Run test:de-grant-auth first.");
    process.exit(1);
  }
  console.log(`  Corporation (policy_address): ${setup.authorityAddress}`);
  console.log(`  Operator:                     ${setup.operatorAddress}`);
  console.log();

  // Step 2: Create operator wallet and connect
  console.log("Step 2: Setting up operator wallet...");
  const wallet = await createAccountFromMnemonic(COOLUSER_MNEMONIC, OPERATOR_INDEX);
  const account = await getAccountInfo(wallet);
  console.log(`  Operator wallet: ${account.address}`);

  if (account.address !== setup.operatorAddress) {
    console.log("  ❌ Operator address mismatch!");
    process.exit(1);
  }

  const client = await createSigningClient(wallet);
  console.log("  ✓ Connected to blockchain");
  console.log();

  // Step 3: Check balance
  console.log("Step 3: Checking operator balance...");
  const balance = await client.getBalance(account.address, config.denom);
  console.log(`  Balance: ${balance.amount} ${balance.denom}`);
  if (BigInt(balance.amount) < BigInt(1000000)) {
    console.log("  ❌ Insufficient balance.");
    process.exit(1);
  }
  console.log();

  // Step 4: Create Ecosystem
  console.log("Step 4: Creating Ecosystem...");
  const did = generateUniqueDID();

  const msg = {
    typeUrl: typeUrls.MsgCreateEcosystem,
    value: MsgCreateEcosystem.fromPartial({
      corporation: setup.authorityAddress,
      operator: account.address,
      did,
      language: "en",
      docUrl: "http://ts-proto-test-ecosystem.com/doc-v1",
      docDigestSri: "sha384-MzNNbQTWCSUSi0bbz7dbua+RcENv7C6FvlmYJ1Y+I727HsPOHdzwELMYO9Mz68M26",
    }),
  };

  console.log(`  Corporation: ${setup.authorityAddress}`);
  console.log(`  Operator:    ${account.address}`);
  console.log(`  DID:         ${did}`);
  console.log();

  try {
    const fee = await calculateFeeWithSimulation(
      client, account.address, [msg],
      "Creating Ecosystem via operator",
    );
    console.log(`  Gas: ${fee.gas}, Fee: ${fee.amount[0].amount}${fee.amount[0].denom}`);

    const result = await signAndBroadcastWithRetry(
      client, account.address, [msg], fee,
      "Creating Ecosystem via operator",
    );

    if (result.code === 0) {
      console.log();
      console.log("✅ SUCCESS! Ecosystem created!");
      console.log(`  Tx Hash: ${result.transactionHash}`);
      console.log(`  Block:   ${result.height}`);
      console.log(`  Gas:     ${result.gasUsed}/${result.gasWanted}`);

      // Extract EC ID from events (v4-rc2 event type: `create_ecosystem`)
      let ecId: number | undefined;
      for (const event of (result.events || [])) {
        if (event.type === "create_ecosystem" || event.type === "verana.ec.v1.EventCreateEcosystem") {
          for (const attr of event.attributes) {
            if (attr.key === "ecosystem_id" || attr.key === "id") {
              ecId = parseInt(attr.value, 10);
              if (!isNaN(ecId)) {
                console.log(`  EC ID:   ${ecId}`);
              }
            }
          }
        }
      }

      if (ecId) {
        saveActiveEC(ecId, did);
        console.log("  💾 Saved active EC for subsequent journeys");
      }
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
