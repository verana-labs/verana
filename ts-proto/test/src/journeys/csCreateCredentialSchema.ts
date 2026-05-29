/**
 * Journey: CS Create Credential Schema (Operator-signed)
 *
 * The operator signs MsgCreateCredentialSchema on behalf of the CS authority.
 * First creates an Ecosystem (controller = CS authority) as a prerequisite,
 * then creates the Credential Schema under that Ecosystem.
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
import { MsgCreateEcosystem } from "../../../src/codec/verana/ec/v1/tx";
import {
  MsgCreateCredentialSchema,
  OptionalUInt32,
} from "../../../src/codec/verana/cs/v1/tx";
import {
  IssuerOnboardingMode,
  VerifierOnboardingMode,
  HolderOnboardingMode,
  PricingAssetType,
} from "../../../src/codec/verana/cs/v1/types";
import { getCsAuthzSetup, saveCsActiveEC, saveActiveCS } from "../helpers/journeyResults";

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

  // Step 4: Create Ecosystem (prerequisite — controller = CS authority).
  // The CS authority address here MUST be a registered MOD-CO policy_address
  // (AUTHZ-CHECK-5). In this wire-level test, we pass it through verbatim;
  // the live-chain CI run is responsible for the real Corporation setup.
  console.log("Step 4: Creating Ecosystem (controller = CS authority)...");
  const ecDid = generateUniqueDID();

  const ecMsg = {
    typeUrl: typeUrls.MsgCreateEcosystem,
    value: MsgCreateEcosystem.fromPartial({
      corporation: setup.authorityAddress,
      operator: account.address,
      did: ecDid,
      language: "en",
      docUrl: "http://cs-ts-proto-test-ecosystem.com/doc-v1",
      docDigestSri: "sha384-MzNNbQTWCSUSi0bbz7dbua+RcENv7C6FvlmYJ1Y+I727HsPOHdzwELMYO9Mz68M26",
    }),
  };

  console.log(`  Authority: ${setup.authorityAddress}`);
  console.log(`  Operator:  ${account.address}`);
  console.log(`  EC DID:    ${ecDid}`);
  console.log();

  let ecId: number | undefined;
  try {
    const ecFee = await calculateFeeWithSimulation(
      client, account.address, [ecMsg],
      "Creating Ecosystem for CS tests via operator",
    );
    console.log(`  Gas: ${ecFee.gas}, Fee: ${ecFee.amount[0].amount}${ecFee.amount[0].denom}`);

    const ecResult = await signAndBroadcastWithRetry(
      client, account.address, [ecMsg], ecFee,
      "Creating Ecosystem for CS tests via operator",
    );

    if (ecResult.code === 0) {
      console.log("  ✅ Ecosystem created!");
      console.log(`  Tx Hash: ${ecResult.transactionHash}`);

      for (const event of (ecResult.events || [])) {
        if (event.type === "create_ecosystem" || event.type === "verana.ec.v1.EventCreateEcosystem") {
          for (const attr of event.attributes) {
            if (attr.key === "ecosystem_id" || attr.key === "id") {
              ecId = parseInt(attr.value, 10);
              if (!isNaN(ecId)) {
                console.log(`  EC ID: ${ecId}`);
              }
            }
          }
        }
      }

      if (!ecId) {
        console.log("  ❌ Could not extract EC ID from events");
        process.exit(1);
      }

      saveCsActiveEC(ecId);
      console.log("  💾 Saved CS active EC");
    } else {
      console.log("  ❌ Ecosystem creation failed!");
      console.log(`  Code: ${ecResult.code}`);
      console.log(`  Log:  ${ecResult.rawLog}`);
      process.exit(1);
    }
  } catch (error: any) {
    console.log("  ❌ ERROR creating Ecosystem!");
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
      corporation: setup.authorityAddress,
      operator: account.address,
      ecosystemId: ecId!,
      jsonSchema: jsonSchema,
      issuerGrantorValidationValidityPeriod: OptionalUInt32.fromPartial({ value: 0 }),
      verifierGrantorValidationValidityPeriod: OptionalUInt32.fromPartial({ value: 0 }),
      issuerValidationValidityPeriod: OptionalUInt32.fromPartial({ value: 0 }),
      verifierValidationValidityPeriod: OptionalUInt32.fromPartial({ value: 0 }),
      holderValidationValidityPeriod: OptionalUInt32.fromPartial({ value: 0 }),
      issuerOnboardingMode: IssuerOnboardingMode.ISSUER_ONBOARDING_MODE_GRANTOR_VALIDATION_PROCESS,
      verifierOnboardingMode: VerifierOnboardingMode.VERIFIER_ONBOARDING_MODE_OPEN,
      holderOnboardingMode: HolderOnboardingMode.HOLDER_ONBOARDING_MODE_PERMISSIONLESS,
      pricingAssetType: PricingAssetType.TU,
      pricingAsset: "tu",
      digestAlgorithm: "sha256",
    }),
  };

  console.log(`  Authority: ${setup.authorityAddress}`);
  console.log(`  Operator:  ${account.address}`);
  console.log(`  EC ID:     ${ecId}`);
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
        saveActiveCS(csId, ecId!, ecDid);
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
