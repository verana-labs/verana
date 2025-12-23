/**
 * Journey: Create Root Permission
 *
 * This script demonstrates how to create a Root Permission using the
 * TypeScript client and the generated protobuf types.
 *
 * Usage:
 *   SCHEMA_ID=1 DID="did:verana:example" npm run test:create-root-perm
 *   # Or let it create a schema first, then create root permission
 *   npm run test:create-root-perm
 *
 * Or with environment variables:
 *   export MNEMONIC="your mnemonic here"
 *   export VERANA_RPC_ENDPOINT="http://localhost:26657"
 *   export SCHEMA_ID=1
 *   export DID="did:verana:example"
 *   npm run test:create-root-perm
 */

import {
  createWallet,
  createSigningClient,
  getAccountInfo,
  calculateFeeWithSimulation,
  generateUniqueDID,
  config,
} from "../helpers/client";
import { typeUrls } from "../helpers/registry";
import { MsgCreateRootPermission } from "../../../src/codec/verana/perm/v1/tx";
import { MsgCreateTrustRegistry } from "../../../src/codec/verana/tr/v1/tx";
import { MsgCreateCredentialSchema, OptionalUInt32 } from "../../../src/codec/verana/cs/v1/tx";
import { CredentialSchemaPermManagementMode } from "../../../src/codec/verana/cs/v1/types";
// Test mnemonic - Uses cooluser seed phrase (same as test harness)
const TEST_MNEMONIC =
  process.env.MNEMONIC ||
  "pink glory help gown abstract eight nice crazy forward ketchup skill cheese";

// Generate a simple JSON schema
function generateSimpleSchema(trustRegistryId: string): string {
  return JSON.stringify({
    $id: `vpr:verana:VPR_CHAIN_ID/cs/v1/js/VPR_CREDENTIAL_SCHEMA_ID`,
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
        },
      },
    },
  });
}

async function main() {
  console.log("=".repeat(60));
  console.log("Journey: Create Root Permission (TypeScript Client)");
  console.log("=".repeat(60));
  console.log();

  // Step 1: Setup wallet
  console.log("Step 1: Setting up wallet...");
  const wallet = await createWallet(TEST_MNEMONIC);
  const account = await getAccountInfo(wallet);
  console.log(`  ✓ Wallet address: ${account.address}`);
  console.log();

  // Step 2: Connect to blockchain
  console.log("Step 2: Connecting to Verana blockchain...");
  console.log(`  RPC Endpoint: ${config.rpcEndpoint}`);
  const client = await createSigningClient(wallet);
  console.log("  ✓ Connected successfully");
  console.log();

  // Step 3: Check account balance
  console.log("Step 3: Checking account balance...");
  const balance = await client.getBalance(account.address, config.denom);
  console.log(`  Balance: ${balance.amount} ${balance.denom}`);
  if (BigInt(balance.amount) < BigInt(1000000)) {
    console.log("  ⚠️  Warning: Low balance. You may need to fund this account.");
    process.exit(1);
  }
  console.log();

  // Step 4: Get or create Schema ID and DID
  let schemaId: number | undefined;
  let did: string;
  
  if (process.env.SCHEMA_ID && process.env.DID) {
    schemaId = parseInt(process.env.SCHEMA_ID, 10);
    did = process.env.DID;
    if (isNaN(schemaId)) {
      console.log("  ❌ Invalid SCHEMA_ID provided");
      process.exit(1);
    }
    console.log(`Step 4: Using provided Schema ID: ${schemaId} and DID: ${did}`);
  } else {
    console.log("Step 4: Creating Trust Registry and Schema first...");
    
    // Create Trust Registry
    did = generateUniqueDID();
    const createTrMsg = {
      typeUrl: typeUrls.MsgCreateTrustRegistry,
      value: MsgCreateTrustRegistry.fromPartial({
        creator: account.address,
        did: did,
        aka: "http://example-trust-registry.com",
        language: "en",
        docUrl: "https://example.com/governance-framework.pdf",
        docDigestSri: "sha384-MzNNbQTWCSUSi0bbz7dbua+RcENv7C6FvlmYJ1Y+I727HsPOHdzwELMYO9Mz68M26",
      }),
    };

    const createTrFee = await calculateFeeWithSimulation(
      client,
      account.address,
      [createTrMsg],
      "Creating Trust Registry for root permission test"
    );

    const createTrResult = await client.signAndBroadcast(
      account.address,
      [createTrMsg],
      createTrFee,
      "Creating Trust Registry for root permission test"
    );

    if (createTrResult.code !== 0) {
      console.log("  ❌ Failed to create Trust Registry");
      console.log(`  Error: ${createTrResult.rawLog}`);
      process.exit(1);
    }

    // Extract TR ID
    let trId: number | undefined;
    const trEvents = createTrResult.events || [];
    for (const event of trEvents) {
      if (event.type === "create_trust_registry" || event.type === "verana.tr.v1.EventCreateTrustRegistry") {
        for (const attr of event.attributes) {
          if (attr.key === "trust_registry_id" || attr.key === "id" || attr.key === "tr_id") {
            trId = parseInt(attr.value, 10);
            if (!isNaN(trId)) break;
          }
        }
        if (trId) break;
      }
    }

    if (!trId) {
      console.log("  ❌ Could not extract TR ID from events");
      process.exit(1);
    }

    // Create Credential Schema
    const createCsMsg = {
      typeUrl: typeUrls.MsgCreateCredentialSchema,
      value: MsgCreateCredentialSchema.fromPartial({
        creator: account.address,
        trId: trId,
        jsonSchema: generateSimpleSchema(trId.toString()),
        issuerGrantorValidationValidityPeriod: { value: 0 } as OptionalUInt32,
        verifierGrantorValidationValidityPeriod: { value: 0 } as OptionalUInt32,
        issuerValidationValidityPeriod: { value: 0 } as OptionalUInt32,
        verifierValidationValidityPeriod: { value: 0 } as OptionalUInt32,
        holderValidationValidityPeriod: { value: 0 } as OptionalUInt32,
        issuerPermManagementMode: CredentialSchemaPermManagementMode.GRANTOR_VALIDATION,
        verifierPermManagementMode: CredentialSchemaPermManagementMode.OPEN,
      }),
    };

    const createCsFee = await calculateFeeWithSimulation(
      client,
      account.address,
      [createCsMsg],
      "Creating Credential Schema for root permission test"
    );

    const createCsResult = await client.signAndBroadcast(
      account.address,
      [createCsMsg],
      createCsFee,
      "Creating Credential Schema for root permission test"
    );

    if (createCsResult.code !== 0) {
      console.log("  ❌ Failed to create Credential Schema");
      console.log(`  Error: ${createCsResult.rawLog}`);
      process.exit(1);
    }

    // Extract Schema ID from events
    const csEvents = createCsResult.events || [];
    for (const event of csEvents) {
      if (event.type === "create_credential_schema" || event.type === "verana.cs.v1.EventCreateCredentialSchema") {
        for (const attr of event.attributes) {
          if (attr.key === "credential_schema_id" || attr.key === "id" || attr.key === "cs_id") {
            schemaId = parseInt(attr.value, 10);
            if (!isNaN(schemaId)) {
              console.log(`  ✓ Created Credential Schema with ID: ${schemaId}`);
              break;
            }
          }
        }
        if (schemaId) break;
      }
    }

    if (!schemaId || isNaN(schemaId)) {
      console.log("  ❌ Could not extract Schema ID from events");
      process.exit(1);
    }
  }

  if (!schemaId || !did) {
    console.log("  ❌ Schema ID and DID are required");
    process.exit(1);
  }

  console.log();

  // Step 5: Create Root Permission message
  console.log("Step 5: Creating Root Permission transaction...");
  // Set effectiveFrom to 10 seconds in the future as required by blockchain (matches test harness)
  const effectiveFrom = new Date(Date.now() + 10000);
  const effectiveUntil = new Date(effectiveFrom.getTime() + 360 * 24 * 60 * 60 * 1000); // 360 days from effectiveFrom
  const validationFees = 5;
  const verificationFees = 5;
  const issuanceFees = 5;
  const country = "US";

  const msg = {
    typeUrl: typeUrls.MsgCreateRootPermission,
    value: MsgCreateRootPermission.fromPartial({
      creator: account.address,
      schemaId: schemaId,
      did: did,
      country: country,
      effectiveFrom: effectiveFrom,
      effectiveUntil: effectiveUntil,
      validationFees: validationFees,
      verificationFees: verificationFees,
      issuanceFees: issuanceFees,
    }),
  };
  console.log("  Message details:");
  console.log(`    - Creator: ${account.address}`);
  console.log(`    - Schema ID: ${schemaId}`);
  console.log(`    - DID: ${did}`);
  console.log(`    - Country: ${country}`);
  console.log(`    - Effective From: ${effectiveFrom.toISOString()}`);
  console.log(`    - Effective Until: ${effectiveUntil.toISOString()}`);
  console.log(`    - Validation Fees: ${validationFees}`);
  console.log(`    - Verification Fees: ${verificationFees}`);
  console.log(`    - Issuance Fees: ${issuanceFees}`);
  console.log();

  // Step 6: Sign and broadcast
  console.log("Step 6: Signing and broadcasting transaction...");
  try {
    const fee = await calculateFeeWithSimulation(
      client,
      account.address,
      [msg],
      "Creating Root Permission via TypeScript client"
    );
    console.log(`  Calculated gas: ${fee.gas}, fee: ${fee.amount[0].amount}${fee.amount[0].denom}`);
    
    const result = await client.signAndBroadcast(
      account.address,
      [msg],
      fee,
      "Creating Root Permission via TypeScript client"
    );

    console.log();
    if (result.code === 0) {
      console.log("✅ SUCCESS! Root Permission created successfully!");
      console.log("=".repeat(60));
      console.log(`  Transaction Hash: ${result.transactionHash}`);
      console.log(`  Block Height: ${result.height}`);
      console.log(`  Gas Used: ${result.gasUsed}/${result.gasWanted}`);
      
      // Try to extract permission ID from events
      const events = result.events || [];
      for (const event of events) {
        if (event.type === "create_root_permission" || event.type === "verana.perm.v1.EventCreateRootPermission") {
          for (const attr of event.attributes) {
            if (attr.key === "root_permission_id" || attr.key === "permission_id" || attr.key === "id") {
              console.log(`  Root Permission ID: ${attr.value}`);
            }
          }
        }
      }
    } else {
      console.log("❌ FAILED! Transaction failed.");
      console.log(`  Error Code: ${result.code}`);
      console.log(`  Raw Log: ${result.rawLog}`);
      process.exit(1);
    }
  } catch (error: any) {
    console.log("❌ ERROR! Transaction failed with exception:");
    console.error(error);
    
    if (error.cause?.code === "ECONNREFUSED" || error.message?.includes("fetch failed")) {
      console.error("\n⚠️  Connection Error: Cannot connect to the blockchain.");
      console.error(`   Make sure the Verana blockchain is running at ${config.rpcEndpoint}`);
      console.error("   Start it with: ./scripts/setup_primary_validator.sh");
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
    console.error("   Start it with: ./scripts/setup_primary_validator.sh");
  }
  
  process.exit(1);
});

