/**
 * Journey: PERM Create Root Permission
 *
 * Creates a Trust Registry, Credential Schema (GRANTOR_VALIDATION mode),
 * and then creates a root (ECOSYSTEM) permission.
 *
 * Requires: test:de-grant-perm-auth must be run first.
 *
 * Usage:
 *   npm run test:perm-create-root
 */

import {
  createAccountFromMnemonic,
  createSigningClient,
  getAccountInfo,
  config,
} from "../helpers/client";
import { getPermAuthzSetup, savePermRootSetup } from "../helpers/journeyResults";
import { createPermPrerequisites } from "../helpers/permissionHelpers";
import { CredentialSchemaPermManagementMode } from "../../../src/codec/verana/cs/v1/types";

const COOLUSER_MNEMONIC =
  (process.env.MNEMONIC && process.env.MNEMONIC.trim()) ||
  "pink glory help gown abstract eight nice crazy forward ketchup skill cheese";

const OPERATOR_INDEX = 15;

async function main() {
  console.log("=".repeat(60));
  console.log("Journey: PERM Create Root Permission");
  console.log("=".repeat(60));
  console.log();

  // Step 1: Load perm authz setup
  console.log("Step 1: Loading PERM authz setup...");
  const setup = getPermAuthzSetup();
  if (!setup) {
    console.log("  No PERM authz setup found. Run test:de-grant-perm-auth first.");
    process.exit(1);
  }
  console.log(`  Authority: ${setup.authorityAddress}`);
  console.log(`  Operator:  ${setup.operatorAddress}`);
  console.log();

  // Step 2: Connect operator
  console.log("Step 2: Setting up operator wallet...");
  const wallet = await createAccountFromMnemonic(COOLUSER_MNEMONIC, OPERATOR_INDEX);
  const account = await getAccountInfo(wallet);
  if (account.address !== setup.operatorAddress) {
    console.log("  Operator address mismatch!");
    process.exit(1);
  }
  const client = await createSigningClient(wallet);
  console.log(`  Connected as ${account.address}`);

  const balance = await client.getBalance(account.address, config.denom);
  console.log(`  Balance: ${balance.amount}${balance.denom}`);
  console.log();

  // Step 3: Create TR → CS (GRANTOR_VALIDATION) → Root Permission
  console.log("Step 3: Creating prerequisites (TR + CS + Root Permission)...");
  try {
    const { trId, schemaId, rootPermId, did, effectiveFrom } = await createPermPrerequisites(
      client,
      setup.authorityAddress,
      setup.operatorAddress,
      CredentialSchemaPermManagementMode.ECOSYSTEM,
    );

    console.log();
    console.log("SUCCESS! Root permission created!");
    console.log(`  TR ID: ${trId}`);
    console.log(`  CS ID: ${schemaId} (ECOSYSTEM mode)`);
    console.log(`  Root Permission ID: ${rootPermId}`);
    console.log(`  DID: ${did}`);
    console.log(`  Effective From: ${effectiveFrom.toISOString()}`);

    savePermRootSetup(trId, schemaId, rootPermId, did);
    console.log("  Saved perm-root-setup");
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
