/**
 * Journey: CS Schema Authorization Policy (amino)
 *
 * Grants the operator authorization for the 3 schema-auth-policy message types
 * (x/group proposal), then the operator creates -> increases version -> revokes a
 * schema authorization policy via MsgCreate/Increase/RevokeSchemaAuthorizationPolicy
 * (operator-signed, all amino). Covers MOD-CS-MSG-5/6/7.
 *
 * Requires: test:co-create (active corporation + operator index 11) and a created
 * credential schema (test:cs-create -> getActiveCS).
 *
 * Usage: npm run test:cs-policy
 */

import {
  createAccountFromMnemonic,
  createSigningClient,
  getAccountInfo,
  calculateFeeWithSimulation,
  signAndBroadcastWithRetry,
  fundAccount,
} from "../helpers/client";
import { typeUrls } from "../helpers/registry";
import {
  MsgCreateSchemaAuthorizationPolicy,
  MsgIncreaseActiveSchemaAuthorizationPolicyVersion,
  MsgRevokeSchemaAuthorizationPolicy,
} from "../../../src/codec/verana/cs/v1/tx";
import { SchemaAuthorizationPolicyRole } from "../../../src/codec/verana/cs/v1/types";
import { MsgGrantOperatorAuthorization } from "../../../src/codec/verana/de/v1/tx";
import { submitGroupProposalAndExec } from "../helpers/group";
import { getActiveCorporation, getActiveCS } from "../helpers/journeyResults";

const COOLUSER_MNEMONIC =
  (process.env.MNEMONIC && process.env.MNEMONIC.trim()) ||
  "pink glory help gown abstract eight nice crazy forward ketchup skill cheese";

const SIGNER_INDEX = 10; // sole group member of the active corporation
const OPERATOR_INDEX = 11;

async function main() {
  console.log("=".repeat(60));
  console.log("Journey: CS Schema Authorization Policy (amino)");
  console.log("=".repeat(60));

  const corp = getActiveCorporation();
  if (!corp) {
    console.log("  No active corporation found. Run test:co-create first.");
    process.exit(1);
  }
  const cs = getActiveCS();
  if (!cs) {
    console.log("  No active credential schema found. Run test:cs-create first.");
    process.exit(1);
  }

  const signerWallet = await createAccountFromMnemonic(COOLUSER_MNEMONIC, SIGNER_INDEX);
  const signerAccount = await getAccountInfo(signerWallet);
  const operatorWallet = await createAccountFromMnemonic(COOLUSER_MNEMONIC, OPERATOR_INDEX);
  const operatorAccount = await getAccountInfo(operatorWallet);
  const cooluserWallet = await createAccountFromMnemonic(COOLUSER_MNEMONIC, 0);
  const cooluserAccount = await getAccountInfo(cooluserWallet);

  const signerClient = await createSigningClient(signerWallet);
  console.log(`  Corporation: ${corp.policyAddress}`);
  console.log(`  Operator:    ${operatorAccount.address}`);
  console.log(`  Schema id:   ${cs.schemaId}`);

  const role = SchemaAuthorizationPolicyRole.SCHEMA_AUTHORIZATION_POLICY_ROLE_ISSUER;

  try {
    await fundAccount(COOLUSER_MNEMONIC, cooluserAccount.address, operatorAccount.address, "50000000uvna");
    await new Promise((r) => setTimeout(r, 3000));

    // Step 1: Grant operator authz for the 3 CS policy msg types (group proposal)
    console.log("\nStep 1: Grant operator authz for CS policy msgs...");
    const grantValue = MsgGrantOperatorAuthorization.encode(
      MsgGrantOperatorAuthorization.fromPartial({
        corporation: corp.policyAddress,
        operator: "",
        grantee: operatorAccount.address,
        msgTypes: [
          typeUrls.MsgCreateSchemaAuthorizationPolicy,
          typeUrls.MsgIncreaseActiveSchemaAuthorizationPolicyVersion,
          typeUrls.MsgRevokeSchemaAuthorizationPolicy,
        ],
        withFeegrant: false,
      }),
    ).finish();
    await submitGroupProposalAndExec(
      signerClient, signerAccount.address, corp.policyAddress,
      [{ typeUrl: typeUrls.MsgGrantOperatorAuthorization, value: grantValue }],
      "Grant CS policy authz",
    );
    console.log("  Operator authorized for CS policy msgs");

    const operatorClient = await createSigningClient(operatorWallet);
    try {
      // Step 2: Create schema authorization policy
      console.log("\nStep 2: CreateSchemaAuthorizationPolicy (ISSUER, amino)...");
      const createMsg = {
        typeUrl: typeUrls.MsgCreateSchemaAuthorizationPolicy,
        value: MsgCreateSchemaAuthorizationPolicy.fromPartial({
          corporation: corp.policyAddress,
          operator: operatorAccount.address,
          schemaId: cs.schemaId,
          role,
          url: "https://example.com/issuer-policy.json",
          digestSri: "sha384-AbCdEf0123456789AbCdEf0123456789AbCdEf0123456789AbCdEf0123456789",
        }),
      };
      let fee = await calculateFeeWithSimulation(operatorClient, operatorAccount.address, [createMsg], "create policy");
      let res = await signAndBroadcastWithRetry(operatorClient, operatorAccount.address, [createMsg], fee, "create policy");
      if (res.code !== 0) throw new Error(`create policy failed: ${res.rawLog}`);
      console.log("  Created (version 1)");

      // Step 3: Increase active version
      console.log("\nStep 3: IncreaseActiveSchemaAuthorizationPolicyVersion (amino)...");
      const incMsg = {
        typeUrl: typeUrls.MsgIncreaseActiveSchemaAuthorizationPolicyVersion,
        value: MsgIncreaseActiveSchemaAuthorizationPolicyVersion.fromPartial({
          corporation: corp.policyAddress,
          operator: operatorAccount.address,
          schemaId: cs.schemaId,
          role,
        }),
      };
      fee = await calculateFeeWithSimulation(operatorClient, operatorAccount.address, [incMsg], "increase policy");
      res = await signAndBroadcastWithRetry(operatorClient, operatorAccount.address, [incMsg], fee, "increase policy");
      if (res.code !== 0) throw new Error(`increase policy failed: ${res.rawLog}`);
      console.log("  Active version increased");

      // Step 4: Revoke version 1
      console.log("\nStep 4: RevokeSchemaAuthorizationPolicy (version 1, amino)...");
      const revokeMsg = {
        typeUrl: typeUrls.MsgRevokeSchemaAuthorizationPolicy,
        value: MsgRevokeSchemaAuthorizationPolicy.fromPartial({
          corporation: corp.policyAddress,
          operator: operatorAccount.address,
          schemaId: cs.schemaId,
          role,
          version: 1,
        }),
      };
      fee = await calculateFeeWithSimulation(operatorClient, operatorAccount.address, [revokeMsg], "revoke policy");
      res = await signAndBroadcastWithRetry(operatorClient, operatorAccount.address, [revokeMsg], fee, "revoke policy");
      if (res.code !== 0) throw new Error(`revoke policy failed: ${res.rawLog}`);
      console.log("  Version 1 revoked");
    } finally {
      operatorClient.disconnect();
    }

    console.log("\n" + "=".repeat(60));
    console.log("SUCCESS! CS schema authorization policy (amino): create -> increase -> revoke");
    console.log("=".repeat(60));
  } catch (error: any) {
    console.log("ERROR!");
    console.error(error.message || error);
    process.exit(1);
  } finally {
    signerClient.disconnect();
  }
}

main().catch((error: any) => {
  console.error("\nFatal error:", error.message || error);
  process.exit(1);
});
