/**
 * Journey: CS Create Credential Schema (Operator-signed)
 *
 * The operator signs MsgCreateCredentialSchema on behalf of the CS authority.
 * First creates a Trust Registry (controller = CS authority) as a prerequisite,
 * then creates the Credential Schema under that TR.
 *
 * Requires: test:de-grant-cs-auth must be run first.
 *
 * Usage:
 *   npm run test:cs-create
 */

import {
  createAccountFromMnemonic,
  createSigningClient,
  getAccountInfo,
  calculateFeeWithSimulation,
  signAndBroadcastWithRetry,
  generateUniqueDID,
  config,
} from "../helpers/client";
import { typeUrls } from "../helpers/registry";
import { MsgCreateTrustRegistry } from "../../../src/codec/verana/tr/v1/tx";
import {
  MsgCreateCredentialSchema,
  OptionalUInt32,
} from "../../../src/codec/verana/cs/v1/tx";
import {
  CredentialSchemaPermManagementMode,
  PricingAssetType,
} from "../../../src/codec/verana/cs/v1/types";
import { getCsAuthzSetup, saveCsActiveTR, saveActiveCS } from "../helpers/journeyResults";

const COOLUSER_MNEMONIC =
  (process.env.MNEMONIC && process.env.MNEMONIC.trim()) ||
  "pink glory help gown abstract eight nice crazy forward ketchup skill cheese";

const OPERATOR_INDEX = 13;

function generateSimpleSchema(): string {
  return JSON.stringify({
    $id: "vpr:verana:VPR_CHAIN_ID/cs/v1/js/VPR_CREDENTIAL_SCHEMA_ID",
    $schema: "https://json-schema.org/draft/2020-12/schema",
    title: "ExampleCredential",
    description: "ExampleCredential using JsonSchema",
    type: "object",
    properties: {
      credentialSubject: {
        type: "object",
        properties: {
          id: { type: "string", format: "uri" },
          firstName: { type: "string", minLength: 0, maxLength: 256 },
          lastName: { type: "string", minLength: 1, maxLength: 256 },
          expirationDate: { type: "string", format: "date" },
          countryOfResidence: { type: "string", minLength: 2, maxLength: 2 },
        },
        required: ["id", "lastName", "expirationDate", "countryOfResidence"],
      },
    },
  });
}

async function main() {
  console.log("=".repeat(60));
  console.log("Journey: CS Create Credential Schema (Operator-signed)");
  console.log("=".repeat(60));
  console.log();

  // Step 1: Load CS authz setup
  console.log("Step 1: Loading CS authz setup...");
  const setup = getCsAuthzSetup();
  if (!setup) {
    console.log("  ❌ No CS authz setup found. Run test:de-grant-cs-auth first.");
    process.exit(1);
  }
  console.log(`  CS Authority: ${setup.authorityAddress}`);
  console.log(`  CS Operator:  ${setup.operatorAddress}`);
  console.log();

  // Step 2: Create operator wallet and connect
  console.log("Step 2: Setting up CS operator wallet...");
  const wallet = await createAccountFromMnemonic(COOLUSER_MNEMONIC, OPERATOR_INDEX);
  const account = await getAccountInfo(wallet);
  console.log(`  Operator wallet: ${account.address}`);

  if (account.address !== setup.operatorAddress) {
    console.log("  ❌ CS operator address mismatch!");
    process.exit(1);
  }

  const client = await createSigningClient(wallet);
  console.log("  ✓ Connected to blockchain");
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

  // Step 4: Create Trust Registry (prerequisite — controller = CS authority)
  console.log("Step 4: Creating Trust Registry (controller = CS authority)...");
  const trDid = generateUniqueDID();
  const trAka = "http://cs-ts-proto-test-trust-registry.com";

  const trMsg = {
    typeUrl: typeUrls.MsgCreateTrustRegistry,
    value: MsgCreateTrustRegistry.fromPartial({
      authority: setup.authorityAddress,
      operator: account.address,
      did: trDid,
      aka: trAka,
      language: "en",
      docUrl: "https://example.com/cs-governance-framework.pdf",
      docDigestSri: "sha384-MzNNbQTWCSUSi0bbz7dbua+RcENv7C6FvlmYJ1Y+I727HsPOHdzwELMYO9Mz68M26",
    }),
  };

  console.log(`  Authority: ${setup.authorityAddress}`);
  console.log(`  Operator:  ${account.address}`);
  console.log(`  TR DID:    ${trDid}`);
  console.log();

  let trId: number | undefined;
  try {
    const trFee = await calculateFeeWithSimulation(
      client, account.address, [trMsg],
      "Creating Trust Registry for CS tests via operator",
    );
    console.log(`  Gas: ${trFee.gas}, Fee: ${trFee.amount[0].amount}${trFee.amount[0].denom}`);

    const trResult = await signAndBroadcastWithRetry(
      client, account.address, [trMsg], trFee,
      "Creating Trust Registry for CS tests via operator",
    );

    if (trResult.code === 0) {
      console.log("  ✅ Trust Registry created!");
      console.log(`  Tx Hash: ${trResult.transactionHash}`);

      for (const event of (trResult.events || [])) {
        if (event.type === "create_trust_registry" || event.type === "verana.tr.v1.EventCreateTrustRegistry") {
          for (const attr of event.attributes) {
            if (attr.key === "trust_registry_id" || attr.key === "id" || attr.key === "tr_id") {
              trId = parseInt(attr.value, 10);
              if (!isNaN(trId)) {
                console.log(`  TR ID: ${trId}`);
              }
            }
          }
        }
      }

      if (!trId) {
        console.log("  ❌ Could not extract TR ID from events");
        process.exit(1);
      }

      saveCsActiveTR(trId);
      console.log("  💾 Saved CS active TR");
    } else {
      console.log("  ❌ Trust Registry creation failed!");
      console.log(`  Code: ${trResult.code}`);
      console.log(`  Log:  ${trResult.rawLog}`);
      process.exit(1);
    }
  } catch (error: any) {
    console.log("  ❌ ERROR creating Trust Registry!");
    console.error(error);
    process.exit(1);
  }
  console.log();

  // Step 5: Create Credential Schema
  console.log("Step 5: Creating Credential Schema...");
  const jsonSchema = generateSimpleSchema();

  const csMsg = {
    typeUrl: typeUrls.MsgCreateCredentialSchema,
    value: MsgCreateCredentialSchema.fromPartial({
      authority: setup.authorityAddress,
      operator: account.address,
      trId: trId!,
      jsonSchema: jsonSchema,
      issuerGrantorValidationValidityPeriod: OptionalUInt32.fromPartial({ value: 0 }),
      verifierGrantorValidationValidityPeriod: OptionalUInt32.fromPartial({ value: 0 }),
      issuerValidationValidityPeriod: OptionalUInt32.fromPartial({ value: 0 }),
      verifierValidationValidityPeriod: OptionalUInt32.fromPartial({ value: 0 }),
      holderValidationValidityPeriod: OptionalUInt32.fromPartial({ value: 0 }),
      issuerPermManagementMode: CredentialSchemaPermManagementMode.GRANTOR_VALIDATION,
      verifierPermManagementMode: CredentialSchemaPermManagementMode.OPEN,
      pricingAssetType: PricingAssetType.TU,
      pricingAsset: "tu",
      digestAlgorithm: "sha256",
    }),
  };

  console.log(`  Authority: ${setup.authorityAddress}`);
  console.log(`  Operator:  ${account.address}`);
  console.log(`  TR ID:     ${trId}`);
  console.log();

  try {
    const csFee = await calculateFeeWithSimulation(
      client, account.address, [csMsg],
      "Creating Credential Schema via operator",
    );
    console.log(`  Gas: ${csFee.gas}, Fee: ${csFee.amount[0].amount}${csFee.amount[0].denom}`);

    const csResult = await signAndBroadcastWithRetry(
      client, account.address, [csMsg], csFee,
      "Creating Credential Schema via operator",
    );

    if (csResult.code === 0) {
      console.log();
      console.log("✅ SUCCESS! Credential Schema created!");
      console.log(`  Tx Hash: ${csResult.transactionHash}`);
      console.log(`  Block:   ${csResult.height}`);
      console.log(`  Gas:     ${csResult.gasUsed}/${csResult.gasWanted}`);

      // Extract CS ID from events
      let csId: number | undefined;
      for (const event of (csResult.events || [])) {
        if (event.type === "create_credential_schema" || event.type === "verana.cs.v1.EventCreateCredentialSchema") {
          for (const attr of event.attributes) {
            if (attr.key === "credential_schema_id" || attr.key === "id") {
              csId = parseInt(attr.value, 10);
              if (!isNaN(csId)) {
                console.log(`  CS ID:   ${csId}`);
              }
            }
          }
        }
      }

      if (csId) {
        saveActiveCS(csId, trId!, trDid);
        console.log("  💾 Saved active CS for subsequent journeys");
      }
    } else {
      console.log("❌ FAILED!");
      console.log(`  Code: ${csResult.code}`);
      console.log(`  Log:  ${csResult.rawLog}`);
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
