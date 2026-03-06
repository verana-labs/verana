/**
 * Journey: TR Create Trust Registry (Operator-signed)
 *
 * The operator signs MsgCreateTrustRegistry on behalf of the authority.
 * Requires DE grant operator authorization journey to run first.
 *
 * Usage:
 *   npm run test:tr-create
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
import { MsgCreateTrustRegistry } from "../../../src/codec/verana/tr/v1/tx";
import { getTrAuthzSetup, saveActiveTR } from "../helpers/journeyResults";

const COOLUSER_MNEMONIC =
  (process.env.MNEMONIC && process.env.MNEMONIC.trim()) ||
  "pink glory help gown abstract eight nice crazy forward ketchup skill cheese";

const OPERATOR_INDEX = 11;

async function main() {
  console.log("=".repeat(60));
  console.log("Journey: TR Create Trust Registry (Operator-signed)");
  console.log("=".repeat(60));
  console.log();

  // Step 1: Load authz setup
  console.log("Step 1: Loading authz setup...");
  const setup = getTrAuthzSetup();
  if (!setup) {
    console.log("  ❌ No authz setup found. Run test:de-grant-auth first.");
    process.exit(1);
  }
  console.log(`  Authority: ${setup.authorityAddress}`);
  console.log(`  Operator:  ${setup.operatorAddress}`);
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

  // Step 4: Create Trust Registry
  console.log("Step 4: Creating Trust Registry...");
  const did = generateUniqueDID();
  const aka = "http://ts-proto-test-trust-registry.com";

  const msg = {
    typeUrl: typeUrls.MsgCreateTrustRegistry,
    value: MsgCreateTrustRegistry.fromPartial({
      authority: setup.authorityAddress,
      operator: account.address,
      did: did,
      aka: aka,
      language: "en",
      docUrl: "https://example.com/governance-framework.pdf",
      docDigestSri: "sha384-MzNNbQTWCSUSi0bbz7dbua+RcENv7C6FvlmYJ1Y+I727HsPOHdzwELMYO9Mz68M26",
    }),
  };

  console.log(`  Authority: ${setup.authorityAddress}`);
  console.log(`  Operator:  ${account.address}`);
  console.log(`  DID:       ${did}`);
  console.log(`  AKA:       ${aka}`);
  console.log();

  try {
    const fee = await calculateFeeWithSimulation(
      client, account.address, [msg],
      "Creating Trust Registry via operator",
    );
    console.log(`  Gas: ${fee.gas}, Fee: ${fee.amount[0].amount}${fee.amount[0].denom}`);

    const result = await signAndBroadcastWithRetry(
      client, account.address, [msg], fee,
      "Creating Trust Registry via operator",
    );

    if (result.code === 0) {
      console.log();
      console.log("✅ SUCCESS! Trust Registry created!");
      console.log(`  Tx Hash: ${result.transactionHash}`);
      console.log(`  Block:   ${result.height}`);
      console.log(`  Gas:     ${result.gasUsed}/${result.gasWanted}`);

      // Extract TR ID from events
      let trId: number | undefined;
      for (const event of (result.events || [])) {
        if (event.type === "create_trust_registry" || event.type === "verana.tr.v1.EventCreateTrustRegistry") {
          for (const attr of event.attributes) {
            if (attr.key === "trust_registry_id" || attr.key === "id" || attr.key === "tr_id") {
              trId = parseInt(attr.value, 10);
              if (!isNaN(trId)) {
                console.log(`  TR ID:   ${trId}`);
              }
            }
          }
        }
      }

      if (trId) {
        saveActiveTR(trId, did);
        console.log("  💾 Saved active TR for subsequent journeys");
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
