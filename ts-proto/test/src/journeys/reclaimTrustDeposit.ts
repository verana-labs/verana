/**
 * Journey: Reclaim Trust Deposit
 *
 * This script demonstrates how to reclaim trust deposit principal using the
 * TypeScript client and the generated protobuf types.
 *
 * Prerequisites:
 * - Account must have a trust deposit with claimable amount > 0
 * - Claimable amount comes from:
 *   - Permission termination (when permission is revoked)
 *   - Permission slashing (when permission trust deposit is slashed)
 *
 * Usage:
 *   CLAIMED=1000 npm run test:reclaim-td
 *   # Or let it use default amount
 *   npm run test:reclaim-td
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
import { MsgReclaimTrustDeposit } from "../../../src/codec/verana/td/v1/tx";
import { MsgAddDID, MsgRemoveDID } from "../../../src/codec/verana/dd/v1/tx";

const TEST_MNEMONIC =
  process.env.MNEMONIC ||
  "pink glory help gown abstract eight nice crazy forward ketchup skill cheese";

async function main() {
  console.log("=".repeat(60));
  console.log("Journey: Reclaim Trust Deposit (TypeScript Client)");
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

  let claimableAmount = 0;
  let didToRemove: string | undefined;

  // Step 1: Setup - Create DID and remove it to create claimable amount
  // Removing a DID moves its deposit to claimable (per spec MOD-DD-MSG-4)
  if (process.env.DID_TO_REMOVE) {
    didToRemove = process.env.DID_TO_REMOVE;
    console.log(`Step 1: Using provided DID to remove: ${didToRemove}`);
    console.log(`  Note: This DID should already exist and removing it will create claimable amount`);
  } else {
    console.log("Step 1: Creating DID first...");
    console.log("  Note: We'll create a DID, then remove it to create claimable amount");
    console.log("  Note: Removing a DID moves its deposit to claimable (per spec MOD-DD-MSG-4)");
    
    // Create a DID (this creates trust deposit)
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
        didToRemove = did;
        // Wait a bit for state to update
        await new Promise((resolve) => setTimeout(resolve, 2000));
      } else {
        console.log(`  ❌ Failed to create DID: ${addDidResult.rawLog}`);
        process.exit(1);
      }
    } catch (error: any) {
      console.log(`  ❌ Error creating DID: ${error.message}`);
      process.exit(1);
    }
    
    // Remove the DID (this moves deposit to claimable)
    console.log("Step 2: Removing DID to create claimable amount...");
    const removeDidMsg = {
      typeUrl: typeUrls.MsgRemoveDID,
      value: MsgRemoveDID.fromPartial({
        creator: account.address,
        did: didToRemove,
      }),
    };
    
    try {
      const removeDidFee = await calculateFeeWithSimulation(
        client,
        account.address,
        [removeDidMsg],
        "Removing DID to create claimable amount"
      );
      const removeDidResult = await client.signAndBroadcast(
        account.address,
        [removeDidMsg],
        removeDidFee,
        "Removing DID to create claimable amount"
      );
      
      if (removeDidResult.code === 0) {
        console.log(`  ✓ Removed DID: ${didToRemove}`);
        console.log(`  ✓ Trust deposit is now claimable`);
        // Wait a bit for state to update
        await new Promise((resolve) => setTimeout(resolve, 2000));
      } else {
        console.log(`  ❌ Failed to remove DID: ${removeDidResult.rawLog}`);
        process.exit(1);
      }
    } catch (error: any) {
      console.log(`  ❌ Error removing DID: ${error.message}`);
      process.exit(1);
    }
  }

  if (!didToRemove) {
    console.log("  ❌ DID is required");
    process.exit(1);
  }

  // Step 3: Query trust deposit to get actual claimable amount (after DID removal)
  console.log();
  console.log("Step 3: Querying trust deposit to get claimable amount...");
  try {
    const response = await fetch(`${config.lcdEndpoint}/verana/td/v1/get/${account.address}`);
    if (!response.ok) {
      throw new Error(`Failed to query trust deposit: ${response.statusText}`);
    }
    const data = (await response.json()) as any;
    const trustDeposit = data.trustDeposit || data.trust_deposit;
    
    if (trustDeposit) {
      const availableClaimable = trustDeposit.claimable || trustDeposit.Claimable || 0;
      console.log(`  ✓ Trust Deposit Amount: ${trustDeposit.amount || trustDeposit.Amount || 0} uvna`);
      console.log(`  ✓ Claimable Amount: ${availableClaimable} uvna`);
      
      if (availableClaimable === 0) {
        console.log(`  ❌ No claimable amount available. DID removal may not have created claimable balance yet.`);
        console.log(`  Note: Wait a bit longer or check if DID was properly removed.`);
        process.exit(1);
      }
      
      // Use requested amount if provided, otherwise use available claimable amount
      const requestedAmount = process.env.CLAIMED ? parseInt(process.env.CLAIMED, 10) : undefined;
      if (requestedAmount) {
        if (requestedAmount > availableClaimable) {
          console.log(`  ⚠️  Requested amount (${requestedAmount}) exceeds available claimable (${availableClaimable})`);
          console.log(`  Using available claimable amount: ${availableClaimable} uvna`);
          claimableAmount = availableClaimable;
        } else {
          claimableAmount = requestedAmount;
        }
      } else {
        // Use all available claimable amount
        claimableAmount = availableClaimable;
      }
      
      console.log(`  ✓ Will reclaim: ${claimableAmount} uvna`);
    } else {
      throw new Error("Trust deposit data not found in response");
    }
  } catch (error: any) {
    console.log(`  ⚠️  Failed to query trust deposit: ${error.message}`);
    console.log(`  Using default amount: 1000 uvna`);
    claimableAmount = 1000;
  }
  console.log();

  // Step 4: Reclaim Trust Deposit
  console.log("Step 4: Reclaiming Trust Deposit transaction...");
  const msg = {
    typeUrl: typeUrls.MsgReclaimTrustDeposit,
    value: MsgReclaimTrustDeposit.fromPartial({
      creator: account.address,
      claimed: claimableAmount,
    }),
  };
  console.log(`    - Creator: ${account.address}`);
  console.log(`    - Amount to reclaim: ${claimableAmount} uvna`);
  console.log();

  console.log("Step 5: Signing and broadcasting transaction...");
  try {
    const fee = await calculateFeeWithSimulation(
      client,
      account.address,
      [msg],
      "Reclaiming Trust Deposit via TypeScript client"
    );
    console.log(`  Calculated gas: ${fee.gas}, fee: ${fee.amount[0].amount}${fee.amount[0].denom}`);

    const result = await client.signAndBroadcast(
      account.address,
      [msg],
      fee,
      "Reclaiming Trust Deposit via TypeScript client"
    );

    console.log();
    if (result.code === 0) {
      console.log("✅ SUCCESS! Trust Deposit reclaimed successfully!");
      console.log("=".repeat(60));
      console.log(`  Transaction Hash: ${result.transactionHash}`);
      console.log(`  Block Height: ${result.height}`);
      console.log(`  Gas Used: ${result.gasUsed}/${result.gasWanted}`);

      // Try to extract amounts from events
      const events = result.events || [];
      for (const event of events) {
        if (
          event.type === "reclaim_trust_deposit" ||
          event.type === "verana.td.v1.EventReclaimTrustDeposit"
        ) {
          for (const attr of event.attributes) {
            if (attr.key === "burned_amount" || attr.key === "burnedAmount") {
              console.log(`  Burned Amount: ${attr.value} uvna`);
            }
            if (attr.key === "claimed_amount" || attr.key === "claimedAmount") {
              console.log(`  Claimed Amount: ${attr.value} uvna`);
            }
          }
        }
      }
    } else {
      console.log("❌ FAILED! Transaction failed.");
      console.log(`  Error Code: ${result.code}`);
      console.log(`  Raw Log: ${result.rawLog}`);
      if (result.rawLog?.includes("insufficient claimable")) {
        console.log(`  Note: Account may not have sufficient claimable amount.`);
        console.log(`  Note: Claimable amount comes from permission termination or slashing.`);
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

