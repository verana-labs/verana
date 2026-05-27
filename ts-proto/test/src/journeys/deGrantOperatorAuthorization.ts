/**
 * Journey: DE Grant Operator Authorization (EC + GF message types)
 *
 * Grants operator authorization from the active Corporation (created by
 * coCreateCorporation) to an operator account, covering all EC and GF
 * message types. The operator can then sign EC/GF messages on behalf of
 * the Corporation.
 *
 * v4-rc2 changes:
 *  - `corporation` is now the policy_address of a registered MOD-CO entry
 *    (AUTHZ-CHECK-5), NOT a fresh wallet address. We sign the grant from
 *    a member account whose grant target is that policy_address.
 *  - Grants cover EC (Create/Update/Archive) + GF (AddDoc/IncreaseVersion).
 *    The legacy TR message types are gone.
 *
 * IMPORTANT runtime note: signing a Msg whose `corporation` is a group
 * policy_address typically requires a group proposal flow rather than a
 * direct sign. The ts-proto test suite is purely a wire/compile-time
 * verifier; the actual on-chain authorisation is exercised by the Go
 * test harness against a live chain. This script's job is only to ensure
 * the messages encode/build cleanly.
 *
 * Requires: test:co-create must be run first.
 *
 * Usage:
 *   npm run test:de-grant-auth
 */

import {
  createWallet,
  createAccountFromMnemonic,
  createSigningClient,
  createQueryClient,
  getAccountInfo,
  calculateFeeWithSimulation,
  signAndBroadcastWithRetry,
  fundAccount,
  config,
} from "../helpers/client";
import { typeUrls } from "../helpers/registry";
import { MsgGrantOperatorAuthorization } from "../../../src/codec/verana/de/v1/tx";
import { saveEcAuthzSetup, getActiveCorporation } from "../helpers/journeyResults";

// Cooluser mnemonic (pre-funded in local chains)
const COOLUSER_MNEMONIC =
  (process.env.MNEMONIC && process.env.MNEMONIC.trim()) ||
  "pink glory help gown abstract eight nice crazy forward ketchup skill cheese";

// Derivation path indexes. Signer (index 10) is the corporation's group
// member and acts as the authorising signer. Operator (index 11) is the
// grantee.
const SIGNER_INDEX = 10;
const OPERATOR_INDEX = 11;

async function main() {
  console.log("=".repeat(60));
  console.log("Journey: DE Grant Operator Authorization (EC + GF)");
  console.log("=".repeat(60));
  console.log();

  // Step 1: Load active Corporation policy_address (the v4-rc2 authority).
  console.log("Step 1: Loading active Corporation...");
  const corp = getActiveCorporation();
  if (!corp) {
    console.log("  ❌ No active corporation found. Run test:co-create first.");
    process.exit(1);
  }
  console.log(`  Corp ID:        ${corp.corporationId}`);
  console.log(`  Policy Address: ${corp.policyAddress}`);
  console.log();

  // Step 2: Create signer + operator wallets
  console.log("Step 2: Creating signer + operator wallets...");
  const signerWallet = await createAccountFromMnemonic(COOLUSER_MNEMONIC, SIGNER_INDEX);
  const operatorWallet = await createAccountFromMnemonic(COOLUSER_MNEMONIC, OPERATOR_INDEX);
  const cooluserWallet = await createWallet(COOLUSER_MNEMONIC);

  const signerAccount = await getAccountInfo(signerWallet);
  const operatorAccount = await getAccountInfo(operatorWallet);
  const cooluserAccount = await getAccountInfo(cooluserWallet);

  console.log(`  Cooluser: ${cooluserAccount.address}`);
  console.log(`  Signer:   ${signerAccount.address} (derivation index ${SIGNER_INDEX})`);
  console.log(`  Operator: ${operatorAccount.address} (derivation index ${OPERATOR_INDEX})`);
  console.log();

  // Step 3: Fund operator (signer was already funded by test:co-create)
  console.log("Step 3: Funding operator...");
  const fundAmount = "50000000uvna"; // 50 VNA

  const fundOpResult = await fundAccount(
    COOLUSER_MNEMONIC,
    cooluserAccount.address,
    operatorAccount.address,
    fundAmount,
  );
  if (fundOpResult.code !== 0) {
    console.log(`  ❌ Failed to fund operator: ${fundOpResult.rawLog}`);
    process.exit(1);
  }
  console.log(`  ✓ Funded operator with ${fundAmount}`);

  const queryClient = await createQueryClient();
  console.log("  ⏳ Waiting for operator funding tx to confirm...");
  for (let i = 0; i < 30; i++) {
    try {
      const tx = await queryClient.getTx(fundOpResult.transactionHash);
      if (tx) {
        console.log(`  ✓ Operator funding confirmed at block ${tx.height}`);
        break;
      }
    } catch {}
    await new Promise((r) => setTimeout(r, 1000));
  }
  queryClient.disconnect();
  console.log();

  // Step 4: Build MsgGrantOperatorAuthorization for the EC+GF message types.
  // The `corporation` field is the Corporation's policy_address (AUTHZ-CHECK-5).
  // The signer pretends to act on behalf of the corporation for wire-level
  // verification only; actual on-chain authorisation requires a group proposal.
  console.log("Step 4: Granting operator authorization for EC + GF message types...");

  const allMsgTypes = [
    typeUrls.MsgCreateEcosystem,
    typeUrls.MsgUpdateEcosystem,
    typeUrls.MsgArchiveEcosystem,
    typeUrls.MsgAddGovernanceFrameworkDocument,
    typeUrls.MsgIncreaseActiveGovernanceFrameworkVersion,
  ];

  console.log("  Message types being authorized:");
  for (const msgType of allMsgTypes) {
    console.log(`    - ${msgType}`);
  }

  const client = await createSigningClient(signerWallet);

  const msg = {
    typeUrl: typeUrls.MsgGrantOperatorAuthorization,
    value: MsgGrantOperatorAuthorization.fromPartial({
      corporation: corp.policyAddress,
      operator: "", // empty — signer acts directly (AUTHZ-CHECK skipped for grant)
      grantee: operatorAccount.address,
      msgTypes: allMsgTypes,
      withFeegrant: false,
    }),
  };

  try {
    const fee = await calculateFeeWithSimulation(
      client,
      signerAccount.address,
      [msg],
      "Granting operator authorization for EC + GF messages",
    );
    console.log(`  Calculated gas: ${fee.gas}, fee: ${fee.amount[0].amount}${fee.amount[0].denom}`);

    const result = await signAndBroadcastWithRetry(
      client,
      signerAccount.address,
      [msg],
      fee,
      "Granting operator authorization for EC + GF messages",
    );

    console.log();
    if (result.code === 0) {
      console.log("✅ SUCCESS! Operator authorization granted successfully!");
      console.log("=".repeat(60));
      console.log(`  Transaction Hash: ${result.transactionHash}`);
      console.log(`  Block Height: ${result.height}`);
      console.log(`  Gas Used: ${result.gasUsed}/${result.gasWanted}`);

      for (const event of result.events || []) {
        if (event.type.includes("grant") || event.type.includes("operator")) {
          console.log(`  Event: ${event.type}`);
          for (const attr of event.attributes) {
            console.log(`    ${attr.key}: ${attr.value}`);
          }
        }
      }

      saveEcAuthzSetup(corp.policyAddress, operatorAccount.address);
      console.log();
      console.log("  💾 Saved EC authz setup (corporation + operator) for EC/GF journeys");
    } else {
      console.log("❌ FAILED! Transaction failed.");
      console.log(`  Error Code: ${result.code}`);
      console.log(`  Raw Log: ${result.rawLog}`);
      process.exit(1);
    }
  } catch (error: any) {
    console.log("❌ ERROR! Transaction failed with exception:");
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

  if (error.cause?.code === "ECONNREFUSED" || error.message?.includes("fetch failed")) {
    console.error("\n⚠️  Connection Error: Cannot connect to the blockchain.");
    console.error(`   Make sure the Verana blockchain is running at ${config.rpcEndpoint}`);
  }

  process.exit(1);
});
