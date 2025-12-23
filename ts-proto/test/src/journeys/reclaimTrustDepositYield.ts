/**
 * Journey: Reclaim Trust Deposit Yield
 *
 * This script demonstrates how to reclaim trust deposit yield using the
 * TypeScript client and the generated protobuf types.
 *
 * Prerequisites:
 * - A continuous fund governance proposal must exist to fund the Yield Intermediate Pool
 *   (This funds the pool that distributes yield to trust deposit holders)
 *   - If SKIP_PROPOSAL_SETUP=true, skips proposal setup (assumes it already exists)
 *   - Otherwise, you must set up the proposal manually first
 * - Account must have a trust deposit (created when DID is registered, trust registry created, or permission created)
 * - Yield must have accumulated (share value must have increased)
 *
 * Usage:
 *   # Skip proposal setup if it already exists
 *   SKIP_PROPOSAL_SETUP=true npm run test:reclaim-td-yield
 *   # Or run normally (will fail if proposal doesn't exist)
 *   npm run test:reclaim-td-yield
 */

import {
  createWallet,
  createSigningClient,
  getAccountInfo,
  calculateFeeWithSimulation,
  config,
  generateUniqueDID,
} from "../helpers/client";
import { typeUrls } from "../helpers/registry";
import { MsgReclaimTrustDepositYield } from "../../../src/codec/verana/td/v1/tx";
import { MsgAddDID } from "../../../src/codec/verana/dd/v1/tx";

const TEST_MNEMONIC =
  process.env.MNEMONIC ||
  "pink glory help gown abstract eight nice crazy forward ketchup skill cheese";

async function main() {
  console.log("=".repeat(60));
  console.log("Journey: Reclaim Trust Deposit Yield (TypeScript Client)");
  console.log("=".repeat(60));
  console.log();

  const wallet = await createWallet(TEST_MNEMONIC);
  const account = await getAccountInfo(wallet);
  const client = await createSigningClient(wallet);

  console.log(`  ✓ Wallet address: ${account.address}`);
  console.log(`  ✓ Connected to ${config.rpcEndpoint}`);
  console.log();

  const balance = await client.getBalance(account.address, config.denom);
  if (BigInt(balance.amount) < BigInt(1000000)) {
    console.log("  ⚠️  Warning: Low balance.");
    process.exit(1);
  }

  // Step 0: Note about continuous fund proposal
  // Note: When running via test:all, Journey 20 (Setup TD Yield Funding Proposal) runs automatically before this
  // If running this test standalone, ensure the continuous fund proposal exists
  const skipProposalSetup = process.env.SKIP_PROPOSAL_SETUP === "true";
  if (!skipProposalSetup && !process.env.RUNNING_ALL_TESTS) {
    console.log("Step 0: Note about continuous fund proposal...");
    console.log("  ℹ️  A continuous fund governance proposal must exist to fund the Yield Intermediate Pool");
    console.log("  ℹ️  This proposal funds the pool that distributes yield to trust deposit holders");
    console.log("  ℹ️  When running via 'npm run test:all', Journey 20 sets this up automatically");
    console.log("  ℹ️  If running standalone, ensure the proposal exists or set SKIP_PROPOSAL_SETUP=true");
    console.log();
  }

  // Step 1: Ensure account has trust deposit
  // Trust deposit is created when:
  // - DID is registered (AddDID)
  // - Trust Registry is created
  // - Permission is created (for grantee and executor)
  console.log("Step 1: Ensuring account has trust deposit...");
  console.log("  Note: Trust deposit is created automatically when DID is registered");
  
  // Create a DID to ensure trust deposit exists
  const did = generateUniqueDID();
  const addDidMsg = {
    typeUrl: typeUrls.MsgAddDID,
    value: MsgAddDID.fromPartial({
      creator: account.address,
      did: did,
      years: 1, // 1 year
    }),
  };

  try {
    const addDidFee = await calculateFeeWithSimulation(
      client,
      account.address,
      [addDidMsg],
      "Adding DID to create trust deposit"
    );
    const addDidResult = await client.signAndBroadcast(
      account.address,
      [addDidMsg],
      addDidFee,
      "Adding DID to create trust deposit"
    );

    if (addDidResult.code === 0) {
      console.log(`  ✓ Created DID: ${did}`);
      console.log(`  ✓ Trust deposit created automatically`);
    } else {
      console.log(`  ⚠️  Failed to create DID: ${addDidResult.rawLog}`);
      console.log(`  Note: Account may already have trust deposit from previous operations`);
    }
  } catch (error: any) {
    console.log(`  ⚠️  Error creating DID: ${error.message}`);
    console.log(`  Note: Account may already have trust deposit from previous operations`);
  }

  console.log();

  // Step 2: Wait for yield to accumulate (optional - yield accumulates over time)
  console.log("Step 2: Checking yield accumulation...");
  console.log("  Note: Yield accumulates over time as share value increases");
  console.log("  Note: Yield must be >= 1 uvna (after truncation) to be reclaimable");
  console.log("  ⏳ Waiting a few seconds for potential yield accumulation...");
  await new Promise((resolve) => setTimeout(resolve, 5000));

  console.log();

  // Step 3: Reclaim Trust Deposit Yield
  console.log("Step 3: Reclaiming Trust Deposit Yield transaction...");
  const msg = {
    typeUrl: typeUrls.MsgReclaimTrustDepositYield,
    value: MsgReclaimTrustDepositYield.fromPartial({
      creator: account.address,
    }),
  };
  console.log(`    - Creator: ${account.address}`);
  console.log();

  console.log("Step 4: Signing and broadcasting transaction...");
  try {
    const fee = await calculateFeeWithSimulation(
      client,
      account.address,
      [msg],
      "Reclaiming Trust Deposit Yield via TypeScript client"
    );
    console.log(`  Calculated gas: ${fee.gas}, fee: ${fee.amount[0].amount}${fee.amount[0].denom}`);

    const result = await client.signAndBroadcast(
      account.address,
      [msg],
      fee,
      "Reclaiming Trust Deposit Yield via TypeScript client"
    );

    console.log();
    if (result.code === 0) {
      console.log("✅ SUCCESS! Trust Deposit Yield reclaimed successfully!");
      console.log("=".repeat(60));
      console.log(`  Transaction Hash: ${result.transactionHash}`);
      console.log(`  Block Height: ${result.height}`);
      console.log(`  Gas Used: ${result.gasUsed}/${result.gasWanted}`);

      // Try to extract claimed amount from events
      const events = result.events || [];
      for (const event of events) {
        if (
          event.type === "reclaim_trust_deposit_yield" ||
          event.type === "verana.td.v1.EventReclaimTrustDepositYield"
        ) {
          for (const attr of event.attributes) {
            if (attr.key === "claimed_yield" || attr.key === "claimedYield") {
              console.log(`  Claimed Yield: ${attr.value} uvna`);
            }
          }
        }
      }
    } else {
      console.log("❌ FAILED! Transaction failed.");
      console.log(`  Error Code: ${result.code}`);
      console.log(`  Raw Log: ${result.rawLog}`);
      if (result.rawLog?.includes("no claimable yield available")) {
        console.log(`  Note: Yield may not have accumulated yet. Wait longer for yield to accumulate.`);
      }
      process.exit(1);
    }
  } catch (error: any) {
    console.log("❌ ERROR! Transaction failed with exception:");
    console.error(error);
    if (error.cause?.code === "ECONNREFUSED" || error.message?.includes("fetch failed")) {
      console.error("\n⚠️  Connection Error: Cannot connect to the blockchain.");
      console.error(`   Make sure the Verana blockchain is running at ${config.rpcEndpoint}`);
    }
    process.exit(1);
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

