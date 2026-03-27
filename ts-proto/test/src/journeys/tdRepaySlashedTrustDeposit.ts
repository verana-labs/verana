/**
 * Journey: TD Repay Slashed Trust Deposit
 *
 * Repays the outstanding slashed amount on the ecosystem trust deposit for the
 * authority (group policy address), executed by an authorized operator.
 *
 * Queries the trust deposit on-chain to determine the outstanding slash amount,
 * then submits MsgRepaySlashedTrustDeposit.
 *
 * Requires: test:de-grant-perm-auth must be run first (provides authority + operator).
 * Requires: The authority's trust deposit must have been slashed previously.
 *
 * Usage:
 *   npm run test:td-repay-slash
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
import { MsgRepaySlashedTrustDeposit } from "../../../src/codec/verana/td/v1/tx";
import { getPermAuthzSetup } from "../helpers/journeyResults";

const COOLUSER_MNEMONIC =
  (process.env.MNEMONIC && process.env.MNEMONIC.trim()) ||
  "pink glory help gown abstract eight nice crazy forward ketchup skill cheese";

const OPERATOR_INDEX = 15;

/**
 * Queries the trust deposit for the given account via LCD to get the outstanding slash amount.
 */
async function queryTrustDeposit(
  account: string,
): Promise<{ slashedDeposit: number; repaidDeposit: number; amount: number }> {
  const lcdEndpoint = process.env.VERANA_LCD_ENDPOINT || "http://localhost:1317";
  const url = `${lcdEndpoint}/verana/td/v1/trust_deposit/${account}`;
  console.log(`  Querying: ${url}`);

  const response = await fetch(url);
  if (!response.ok) {
    throw new Error(`Failed to query trust deposit: ${response.status} ${response.statusText}`);
  }

  const data = (await response.json()) as {
    trust_deposit?: {
      slashed_deposit?: string;
      repaid_deposit?: string;
      amount?: string;
    };
  };

  const td = data.trust_deposit;
  if (!td) {
    throw new Error("No trust deposit found for this account");
  }

  return {
    slashedDeposit: parseInt(td.slashed_deposit || "0", 10),
    repaidDeposit: parseInt(td.repaid_deposit || "0", 10),
    amount: parseInt(td.amount || "0", 10),
  };
}

async function main() {
  console.log("=".repeat(60));
  console.log("Journey: TD Repay Slashed Trust Deposit");
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
    // Step 3: Query trust deposit to determine outstanding slash amount
    console.log("Step 3: Querying trust deposit for outstanding slash...");
    const td = await queryTrustDeposit(setup.authorityAddress);
    const outstandingSlash = td.slashedDeposit - td.repaidDeposit;
    console.log(`  Deposit Amount: ${td.amount}`);
    console.log(`  Slashed Deposit: ${td.slashedDeposit}`);
    console.log(`  Repaid Deposit: ${td.repaidDeposit}`);
    console.log(`  Outstanding Slash: ${outstandingSlash}`);
    console.log();

    if (outstandingSlash <= 0) {
      console.log("No outstanding slash to repay. Exiting.");
      console.log();
      console.log("=".repeat(60));
      return;
    }

    // Step 4: Build and send MsgRepaySlashedTrustDeposit
    console.log("Step 4: Repaying slashed trust deposit (MsgRepaySlashedTrustDeposit)...");
    const msg = {
      typeUrl: typeUrls.MsgRepaySlashedTrustDeposit,
      value: MsgRepaySlashedTrustDeposit.fromPartial({
        authority: setup.authorityAddress,
        operator: setup.operatorAddress,
        amount: outstandingSlash,
      }),
    };

    const fee = await calculateFeeWithSimulation(client, account.address, [msg], "Repaying slashed trust deposit");
    const result = await signAndBroadcastWithRetry(client, account.address, [msg], fee, "Repaying slashed trust deposit");

    if (result.code !== 0) {
      throw new Error(`Failed to repay slashed trust deposit: ${result.rawLog}`);
    }

    console.log();
    console.log("SUCCESS! Slashed trust deposit repaid!");
    console.log(`  Tx Hash: ${result.transactionHash}`);
    console.log(`  Block: ${result.height}`);
    console.log(`  Gas: ${result.gasUsed}/${result.gasWanted}`);
    console.log(`  Repaid Amount: ${outstandingSlash}`);
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
