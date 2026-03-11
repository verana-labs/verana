/**
 * Journey: PERM Repay Permission Slashed Trust Deposit
 *
 * Repays the slashed trust deposit from the slash journey.
 *
 * Requires: test:perm-slash must be run first.
 *
 * Usage:
 *   npm run test:perm-repay
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
import { MsgRepayPermissionSlashedTrustDeposit } from "../../../src/codec/verana/perm/v1/tx";
import { getPermAuthzSetup, getPermSlashSetup } from "../helpers/journeyResults";

const COOLUSER_MNEMONIC =
  (process.env.MNEMONIC && process.env.MNEMONIC.trim()) ||
  "pink glory help gown abstract eight nice crazy forward ketchup skill cheese";

const OPERATOR_INDEX = 15;

async function main() {
  console.log("=".repeat(60));
  console.log("Journey: PERM Repay Slashed Trust Deposit");
  console.log("=".repeat(60));
  console.log();

  // Step 1: Load setup
  console.log("Step 1: Loading PERM setup...");
  const authzSetup = getPermAuthzSetup();
  const slashSetup = getPermSlashSetup();
  if (!authzSetup || !slashSetup) {
    console.log("  Missing setup. Run test:de-grant-perm-auth and test:perm-slash first.");
    process.exit(1);
  }
  console.log(`  Authority: ${authzSetup.authorityAddress}`);
  console.log(`  Slashed Permission ID: ${slashSetup.slashedPermId}`);
  console.log();

  // Step 2: Connect operator
  console.log("Step 2: Setting up operator wallet...");
  const wallet = await createAccountFromMnemonic(COOLUSER_MNEMONIC, OPERATOR_INDEX);
  const account = await getAccountInfo(wallet);
  const client = await createSigningClient(wallet);
  console.log(`  Connected as ${account.address}`);
  console.log();

  try {
    // Step 3: Repay the slashed trust deposit
    console.log("Step 3: Repaying slashed trust deposit (MsgRepayPermissionSlashedTrustDeposit)...");
    const msg = {
      typeUrl: typeUrls.MsgRepayPermissionSlashedTrustDeposit,
      value: MsgRepayPermissionSlashedTrustDeposit.fromPartial({
        authority: authzSetup.authorityAddress,
        operator: authzSetup.operatorAddress,
        id: slashSetup.slashedPermId,
      }),
    };

    const fee = await calculateFeeWithSimulation(client, account.address, [msg], "Repaying slashed deposit");
    const result = await signAndBroadcastWithRetry(client, account.address, [msg], fee, "Repaying slashed deposit");

    if (result.code !== 0) {
      throw new Error(`Failed to repay slashed deposit: ${result.rawLog}`);
    }

    console.log();
    console.log("SUCCESS! Slashed trust deposit repaid!");
    console.log(`  Tx Hash: ${result.transactionHash}`);
    console.log(`  Block: ${result.height}`);
    console.log(`  Gas: ${result.gasUsed}/${result.gasWanted}`);
    console.log(`  Permission ID: ${slashSetup.slashedPermId}`);
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
