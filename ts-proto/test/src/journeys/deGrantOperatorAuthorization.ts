/**
 * Journey: DE Grant Operator Authorization
 *
 * This script grants operator authorization from an authority account to an
 * operator account for all 5 TR message types. The operator can then sign
 * TR messages on behalf of the authority.
 *
 * Key insight: When operator is empty in MsgGrantOperatorAuthorization,
 * the AUTHZ-CHECK is skipped — the authority acts alone and signs directly.
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
import { saveTrAuthzSetup, saveActiveTR } from "../helpers/journeyResults";

// Cooluser mnemonic (pre-funded in local chains)
const COOLUSER_MNEMONIC =
  (process.env.MNEMONIC && process.env.MNEMONIC.trim()) ||
  "pink glory help gown abstract eight nice crazy forward ketchup skill cheese";

// Derivation path indexes for authority and operator
const AUTHORITY_INDEX = 10;
const OPERATOR_INDEX = 11;

async function main() {
  console.log("=".repeat(60));
  console.log("Journey: DE Grant Operator Authorization (TypeScript Client)");
  console.log("=".repeat(60));
  console.log();

  // Step 1: Create authority and operator wallets
  console.log("Step 1: Creating authority and operator wallets...");
  const authorityWallet = await createAccountFromMnemonic(COOLUSER_MNEMONIC, AUTHORITY_INDEX);
  const operatorWallet = await createAccountFromMnemonic(COOLUSER_MNEMONIC, OPERATOR_INDEX);
  const cooluserWallet = await createWallet(COOLUSER_MNEMONIC);

  const authorityAccount = await getAccountInfo(authorityWallet);
  const operatorAccount = await getAccountInfo(operatorWallet);
  const cooluserAccount = await getAccountInfo(cooluserWallet);

  console.log(`  Cooluser:  ${cooluserAccount.address}`);
  console.log(`  Authority: ${authorityAccount.address} (derivation index ${AUTHORITY_INDEX})`);
  console.log(`  Operator:  ${operatorAccount.address} (derivation index ${OPERATOR_INDEX})`);
  console.log();

  // Step 2: Fund authority and operator from cooluser
  console.log("Step 2: Funding authority and operator accounts...");
  const fundAmount = "50000000uvna"; // 50 VNA

  const fundAuthResult = await fundAccount(
    COOLUSER_MNEMONIC,
    cooluserAccount.address,
    authorityAccount.address,
    fundAmount,
  );
  if (fundAuthResult.code !== 0) {
    console.log(`  ❌ Failed to fund authority: ${fundAuthResult.rawLog}`);
    process.exit(1);
  }
  console.log(`  ✓ Funded authority with ${fundAmount}`);

  // Wait for funding tx to be confirmed by polling for the tx hash
  const queryClient = await createQueryClient();
  console.log("  ⏳ Waiting for authority funding tx to confirm...");
  for (let i = 0; i < 30; i++) {
    try {
      const tx = await queryClient.getTx(fundAuthResult.transactionHash);
      if (tx) {
        console.log(`  ✓ Authority funding confirmed at block ${tx.height}`);
        break;
      }
    } catch {}
    await new Promise((resolve) => setTimeout(resolve, 1000));
  }

  // Wait for cooluser sequence to advance before second fund tx
  console.log("  ⏳ Waiting for cooluser sequence to advance...");
  for (let i = 0; i < 30; i++) {
    try {
      const seq = await queryClient.getSequence(cooluserAccount.address);
      if (seq.sequence >= 1) {
        console.log(`  ✓ Cooluser sequence: ${seq.sequence}`);
        break;
      }
    } catch {}
    await new Promise((resolve) => setTimeout(resolve, 1000));
  }

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

  // Wait for operator funding tx to confirm
  console.log("  ⏳ Waiting for operator funding tx to confirm...");
  for (let i = 0; i < 30; i++) {
    try {
      const tx = await queryClient.getTx(fundOpResult.transactionHash);
      if (tx) {
        console.log(`  ✓ Operator funding confirmed at block ${tx.height}`);
        break;
      }
    } catch {}
    await new Promise((resolve) => setTimeout(resolve, 1000));
  }

  // Verify authority account balance before proceeding
  console.log("  ⏳ Verifying authority balance...");
  for (let i = 0; i < 20; i++) {
    const balance = await queryClient.getBalance(authorityAccount.address, config.denom);
    if (BigInt(balance.amount) > 0) {
      console.log(`  ✓ Authority balance: ${balance.amount}${balance.denom}`);
      break;
    }
    await new Promise((resolve) => setTimeout(resolve, 1000));
  }
  queryClient.disconnect();
  console.log();

  // Step 3: Authority grants operator authorization for all 5 TR message types
  console.log("Step 3: Granting operator authorization for all TR message types...");

  const allTrMsgTypes = [
    typeUrls.MsgCreateTrustRegistry,
    typeUrls.MsgUpdateTrustRegistry,
    typeUrls.MsgArchiveTrustRegistry,
    typeUrls.MsgAddGovernanceFrameworkDocument,
    typeUrls.MsgIncreaseActiveGovernanceFrameworkVersion,
  ];

  console.log("  Message types being authorized:");
  for (const msgType of allTrMsgTypes) {
    console.log(`    - ${msgType}`);
  }

  const client = await createSigningClient(authorityWallet);

  const msg = {
    typeUrl: typeUrls.MsgGrantOperatorAuthorization,
    value: MsgGrantOperatorAuthorization.fromPartial({
      authority: authorityAccount.address,
      operator: "", // empty — authority acts alone (AUTHZ-CHECK skipped)
      grantee: operatorAccount.address,
      msgTypes: allTrMsgTypes,
      withFeegrant: false,
    }),
  };

  try {
    const fee = await calculateFeeWithSimulation(
      client,
      authorityAccount.address,
      [msg],
      "Granting operator authorization for TR messages",
    );
    console.log(`  Calculated gas: ${fee.gas}, fee: ${fee.amount[0].amount}${fee.amount[0].denom}`);

    const result = await signAndBroadcastWithRetry(
      client,
      authorityAccount.address,
      [msg],
      fee,
      "Granting operator authorization for TR messages",
    );

    console.log();
    if (result.code === 0) {
      console.log("✅ SUCCESS! Operator authorization granted successfully!");
      console.log("=".repeat(60));
      console.log(`  Transaction Hash: ${result.transactionHash}`);
      console.log(`  Block Height: ${result.height}`);
      console.log(`  Gas Used: ${result.gasUsed}/${result.gasWanted}`);

      // Print events
      const events = result.events || [];
      for (const event of events) {
        if (event.type.includes("grant") || event.type.includes("operator")) {
          console.log(`  Event: ${event.type}`);
          for (const attr of event.attributes) {
            console.log(`    ${attr.key}: ${attr.value}`);
          }
        }
      }

      // Save setup for TR journeys
      saveTrAuthzSetup(authorityAccount.address, operatorAccount.address);
      console.log();
      console.log("  💾 Saved authority and operator addresses for TR journeys");
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
