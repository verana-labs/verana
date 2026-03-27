/**
 * Journey: TD Reclaim Trust Deposit Yield
 *
 * Reclaims any accrued yield on the ecosystem trust deposit for the authority
 * (group policy address), executed by an authorized operator.
 *
 * Requires: test:de-grant-perm-auth must be run first (provides authority + operator).
 *
 * Usage:
 *   npm run test:td-reclaim-yield
 */

import {
  createDirectAccountFromMnemonic,
  createSigningClient,
  getAccountInfo,
  calculateFeeWithSimulation,
  signAndBroadcastWithRetry,
  config,
} from "../helpers/client";
import { typeUrls } from "../helpers/registry";
import { MsgReclaimTrustDepositYield } from "../../../src/codec/verana/td/v1/tx";
import { getPermAuthzSetup } from "../helpers/journeyResults";

const COOLUSER_MNEMONIC =
  (process.env.MNEMONIC && process.env.MNEMONIC.trim()) ||
  "pink glory help gown abstract eight nice crazy forward ketchup skill cheese";

const OPERATOR_INDEX = 15;

async function main() {
  console.log("=".repeat(60));
  console.log("Journey: TD Reclaim Trust Deposit Yield");
  console.log("=".repeat(60));
  console.log();

  // Step 1: Load DE auth setup
  console.log("Step 1: Loading DE authz setup...");
  const setup = getPermAuthzSetup();
  if (!setup) {
    console.log("  No DE authz setup found. Run test:de-grant-perm-auth first.");
    process.exit(1);
  }
  console.log(`  Authority: ${setup.authorityAddress}`);
  console.log(`  Operator:  ${setup.operatorAddress}`);
  console.log();

  // Step 2: Connect operator wallet
  console.log("Step 2: Setting up operator wallet...");
  const wallet = await createDirectAccountFromMnemonic(COOLUSER_MNEMONIC, OPERATOR_INDEX);
  const account = await getAccountInfo(wallet);
  const client = await createSigningClient(wallet);
  console.log(`  Connected as ${account.address}`);
  console.log();

  try {
    // Step 3: Build and send MsgReclaimTrustDepositYield
    console.log("Step 3: Reclaiming trust deposit yield (MsgReclaimTrustDepositYield)...");
    const msg = {
      typeUrl: typeUrls.MsgReclaimTrustDepositYield,
      value: MsgReclaimTrustDepositYield.fromPartial({
        authority: setup.authorityAddress,
        operator: setup.operatorAddress,
      }),
    };

    const fee = await calculateFeeWithSimulation(client, account.address, [msg], "Reclaiming trust deposit yield");
    const result = await signAndBroadcastWithRetry(client, account.address, [msg], fee, "Reclaiming trust deposit yield");

    if (result.code !== 0) {
      const rawLog = result.rawLog || "";
      // "no claimable yield" is an acceptable outcome
      if (rawLog.includes("no claimable yield") || rawLog.includes("nothing to claim")) {
        console.log();
        console.log("No claimable yield available (this is acceptable).");
        console.log(`  Tx Hash: ${result.transactionHash}`);
        console.log(`  Raw Log: ${rawLog}`);
      } else {
        throw new Error(`Failed to reclaim trust deposit yield: ${rawLog}`);
      }
    } else {
      console.log();
      console.log("SUCCESS! Trust deposit yield reclaimed!");
      console.log(`  Tx Hash: ${result.transactionHash}`);
      console.log(`  Block: ${result.height}`);
      console.log(`  Gas: ${result.gasUsed}/${result.gasWanted}`);
    }
  } catch (error: any) {
    console.log("ERROR!");
    console.error(error);
    process.exit(1);
  } finally {
    client.disconnect();
  }

  console.log();
  console.log("=".repeat(60));
}

main().catch((error: any) => {
  console.error("\nFatal error:", error.message || error);
  process.exit(1);
});
