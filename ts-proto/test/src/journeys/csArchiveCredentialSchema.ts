/**
 * Journey: CS Archive Credential Schema (Operator-signed)
 *
 * Archives a Credential Schema.
 * Depends on: test:de-grant-cs-auth, test:cs-create
 *
 * Usage:
 *   npm run test:cs-archive
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
import { MsgArchiveCredentialSchema } from "../../../src/codec/verana/cs/v1/tx";
import { getCsAuthzSetup, getActiveCS } from "../helpers/journeyResults";

const COOLUSER_MNEMONIC =
  (process.env.MNEMONIC && process.env.MNEMONIC.trim()) ||
  "pink glory help gown abstract eight nice crazy forward ketchup skill cheese";

const OPERATOR_INDEX = 13;

async function main() {
  console.log("=".repeat(60));
  console.log("Journey: CS Archive Credential Schema (Operator-signed)");
  console.log("=".repeat(60));
  console.log();

  // Step 1: Load CS authz setup and active CS
  const setup = getCsAuthzSetup();
  if (!setup) {
    console.log("  ❌ No CS authz setup found. Run test:de-grant-cs-auth first.");
    process.exit(1);
  }

  const activeCS = getActiveCS();
  if (!activeCS) {
    console.log("  ❌ No active CS found. Run test:cs-create first.");
    process.exit(1);
  }

  console.log(`  CS Authority: ${setup.authorityAddress}`);
  console.log(`  CS ID:        ${activeCS.schemaId}`);
  console.log();

  // Step 2: Create operator wallet and connect
  const wallet = await createAccountFromMnemonic(COOLUSER_MNEMONIC, OPERATOR_INDEX);
  const account = await getAccountInfo(wallet);
  const client = await createSigningClient(wallet);

  console.log(`  Operator:  ${account.address}`);
  console.log();

  // Step 3: Check balance
  console.log("Step 3: Checking CS operator balance...");
  const balance = await client.getBalance(account.address, config.denom);
  console.log(`  Balance: ${balance.amount} ${balance.denom}`);
  if (BigInt(balance.amount) < BigInt(1000000)) {
    console.log("  ❌ Insufficient balance.");
    process.exit(1);
  }
  console.log();

  // Step 4: Archive Credential Schema
  console.log("Step 4: Archiving Credential Schema...");
  const msg = {
    typeUrl: typeUrls.MsgArchiveCredentialSchema,
    value: MsgArchiveCredentialSchema.fromPartial({
      authority: setup.authorityAddress,
      operator: account.address,
      id: activeCS.schemaId,
      archive: true,
    }),
  };

  try {
    const fee = await calculateFeeWithSimulation(
      client, account.address, [msg],
      "Archiving Credential Schema via operator",
    );
    console.log(`  Gas: ${fee.gas}, Fee: ${fee.amount[0].amount}${fee.amount[0].denom}`);

    const result = await signAndBroadcastWithRetry(
      client, account.address, [msg], fee,
      "Archiving Credential Schema via operator",
    );

    if (result.code === 0) {
      console.log();
      console.log("✅ SUCCESS! Credential Schema archived!");
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
