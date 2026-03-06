/**
 * Journey: CS Update Credential Schema (Operator-signed)
 *
 * Updates a Credential Schema's validity periods.
 * Depends on: test:de-grant-cs-auth, test:cs-create
 *
 * Usage:
 *   npm run test:cs-update
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
import {
  MsgUpdateCredentialSchema,
  OptionalUInt32,
} from "../../../src/codec/verana/cs/v1/tx";
import { getCsAuthzSetup, getActiveCS } from "../helpers/journeyResults";

const COOLUSER_MNEMONIC =
  (process.env.MNEMONIC && process.env.MNEMONIC.trim()) ||
  "pink glory help gown abstract eight nice crazy forward ketchup skill cheese";

const OPERATOR_INDEX = 13;

async function main() {
  console.log("=".repeat(60));
  console.log("Journey: CS Update Credential Schema (Operator-signed)");
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
  console.log(`  TR ID:        ${activeCS.trustRegistryId}`);
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

  // Step 4: Update Credential Schema validity periods
  console.log("Step 4: Updating Credential Schema validity periods...");
  console.log("  IssuerGrantorValidationValidityPeriod:   730 days (2 years)");
  console.log("  VerifierGrantorValidationValidityPeriod: 730 days (2 years)");
  console.log("  IssuerValidationValidityPeriod:          365 days (1 year)");
  console.log("  VerifierValidationValidityPeriod:        365 days (1 year)");
  console.log("  HolderValidationValidityPeriod:          180 days (6 months)");

  const msg = {
    typeUrl: typeUrls.MsgUpdateCredentialSchema,
    value: MsgUpdateCredentialSchema.fromPartial({
      authority: setup.authorityAddress,
      operator: account.address,
      id: activeCS.schemaId,
      issuerGrantorValidationValidityPeriod: OptionalUInt32.fromPartial({ value: 730 }),
      verifierGrantorValidationValidityPeriod: OptionalUInt32.fromPartial({ value: 730 }),
      issuerValidationValidityPeriod: OptionalUInt32.fromPartial({ value: 365 }),
      verifierValidationValidityPeriod: OptionalUInt32.fromPartial({ value: 365 }),
      holderValidationValidityPeriod: OptionalUInt32.fromPartial({ value: 180 }),
    }),
  };

  try {
    const fee = await calculateFeeWithSimulation(
      client, account.address, [msg],
      "Updating Credential Schema via operator",
    );
    console.log(`  Gas: ${fee.gas}, Fee: ${fee.amount[0].amount}${fee.amount[0].denom}`);

    const result = await signAndBroadcastWithRetry(
      client, account.address, [msg], fee,
      "Updating Credential Schema via operator",
    );

    if (result.code === 0) {
      console.log();
      console.log("✅ SUCCESS! Credential Schema updated!");
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
