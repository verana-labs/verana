/**
 * Journey: Create Credential Schema
 *
 * This script demonstrates how to create a Credential Schema using the
 * TypeScript client and the generated protobuf types.
 *
 * Usage:
 *   TR_ID=1 npm run test:create-cs
 *   # Or let it create a TR first, then create a schema
 *   npm run test:create-cs
 *
 * Or with environment variables:
 *   export MNEMONIC="your mnemonic here"
 *   export VERANA_RPC_ENDPOINT="http://localhost:26657"
 *   export TR_ID=1
 *   npm run test:create-cs
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
import { MsgCreateCredentialSchema, OptionalUInt32 } from "../../../src/codec/verana/cs/v1/tx";
import { MsgCreateTrustRegistry } from "../../../src/codec/verana/tr/v1/tx";
import { CredentialSchemaPermManagementMode } from "../../../src/codec/verana/cs/v1/types";

// Test mnemonic - Uses cooluser seed phrase (same as test harness)
const TEST_MNEMONIC =
  process.env.MNEMONIC ||
  "pink glory help gown abstract eight nice crazy forward ketchup skill cheese";

// Generate a simple JSON schema (matches test harness pattern)
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
          id: {
            type: "string",
            format: "uri",
          },
          firstName: {
            type: "string",
            minLength: 0,
            maxLength: 256,
          },
          lastName: {
            type: "string",
            minLength: 1,
            maxLength: 256,
          },
          expirationDate: {
            type: "string",
            format: "date",
          },
          countryOfResidence: {
            type: "string",
            minLength: 2,
            maxLength: 2,
          },
        },
        required: ["id", "lastName", "expirationDate", "countryOfResidence"],
      },
    },
  });
}

async function main() {
  console.log("=".repeat(60));
  console.log("Journey: Create Credential Schema (TypeScript Client)");
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

  // Step 4: Get or create Trust Registry ID
  let trId: number | undefined;
  if (process.env.TR_ID) {
    trId = parseInt(process.env.TR_ID, 10);
    if (isNaN(trId)) {
      console.log("  ❌ Invalid TR_ID provided");
      process.exit(1);
    }
    console.log(`Step 4: Using provided Trust Registry ID: ${trId}`);
  } else {
    console.log("Step 4: Creating a Trust Registry first (no TR_ID provided)...");
    const did = generateUniqueDID();
    const createMsg = {
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

    const createFee = await calculateFeeWithSimulation(
      client,
      account.address,
      [createMsg],
      "Creating Trust Registry for schema test"
    );

    const createResult = await client.signAndBroadcast(
      account.address,
      [createMsg],
      createFee,
      "Creating Trust Registry for schema test"
    );

    if (createResult.code !== 0) {
      console.log("  ❌ Failed to create Trust Registry for schema test");
      console.log(`  Error: ${createResult.rawLog}`);
      process.exit(1);
    }

    // Extract TR ID from events
    const events = createResult.events || [];
    for (const event of events) {
      if (event.type === "create_trust_registry" || event.type === "verana.tr.v1.EventCreateTrustRegistry") {
        for (const attr of event.attributes) {
          if (attr.key === "trust_registry_id" || attr.key === "id" || attr.key === "tr_id") {
            trId = parseInt(attr.value, 10);
            if (!isNaN(trId)) {
              console.log(`  ✓ Created Trust Registry with ID: ${trId}`);
              break;
            }
          }
        }
        if (trId) break;
      }
    }

    if (!trId || isNaN(trId)) {
      console.log("  ❌ Could not extract TR ID from events");
      process.exit(1);
    }
  }

  if (!trId) {
    console.log("  ❌ Trust Registry ID is required");
    process.exit(1);
  }

  console.log();

  // Step 5: Create Credential Schema message
  console.log("Step 5: Creating Credential Schema transaction...");
  const jsonSchema = generateSimpleSchema(trId.toString());
  const msg = {
    typeUrl: typeUrls.MsgCreateCredentialSchema,
    value: MsgCreateCredentialSchema.fromPartial({
      creator: account.address,
      trId: trId,
      jsonSchema: jsonSchema,
      // Validity periods: 0 means never expire (matches test harness)
      issuerGrantorValidationValidityPeriod: { value: 0 } as OptionalUInt32,
      verifierGrantorValidationValidityPeriod: { value: 0 } as OptionalUInt32,
      issuerValidationValidityPeriod: { value: 0 } as OptionalUInt32,
      verifierValidationValidityPeriod: { value: 0 } as OptionalUInt32,
      holderValidationValidityPeriod: { value: 0 } as OptionalUInt32,
      // Permission management modes: GRANTOR_VALIDATION = 2, OPEN = 1
      issuerPermManagementMode: CredentialSchemaPermManagementMode.GRANTOR_VALIDATION,
      verifierPermManagementMode: CredentialSchemaPermManagementMode.OPEN,
    }),
  };
  console.log("  Message details:");
  console.log(`    - Creator: ${account.address}`);
  console.log(`    - Trust Registry ID: ${trId}`);
  console.log(`    - Issuer Perm Management Mode: GRANTOR_VALIDATION (2)`);
  console.log(`    - Verifier Perm Management Mode: OPEN (1)`);
  console.log();

  // Step 6: Sign and broadcast
  console.log("Step 6: Signing and broadcasting transaction...");
  try {
    const fee = await calculateFeeWithSimulation(
      client,
      account.address,
      [msg],
      "Creating Credential Schema via TypeScript client"
    );
    console.log(`  Calculated gas: ${fee.gas}, fee: ${fee.amount[0].amount}${fee.amount[0].denom}`);
    
    const result = await client.signAndBroadcast(
      account.address,
      [msg],
      fee,
      "Creating Credential Schema via TypeScript client"
    );

    console.log();
    if (result.code === 0) {
      console.log("✅ SUCCESS! Credential Schema created successfully!");
      console.log("=".repeat(60));
      console.log(`  Transaction Hash: ${result.transactionHash}`);
      console.log(`  Block Height: ${result.height}`);
      console.log(`  Gas Used: ${result.gasUsed}/${result.gasWanted}`);
      
      // Try to extract schema ID from events
      const events = result.events || [];
      for (const event of events) {
        if (event.type === "create_credential_schema" || event.type === "verana.cs.v1.EventCreateCredentialSchema") {
          for (const attr of event.attributes) {
            if (attr.key === "credential_schema_id" || attr.key === "id" || attr.key === "cs_id") {
              console.log(`  Credential Schema ID: ${attr.value}`);
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

