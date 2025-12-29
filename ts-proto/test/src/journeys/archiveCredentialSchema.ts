/**
 * Journey: Archive Credential Schema
 *
 * This script demonstrates how to archive/unarchive a Credential Schema using the
 * TypeScript client and the generated protobuf types.
 *
 * Usage:
 *   CS_ID=1 npm run test:archive-cs
 *   # Or let it create a schema first, then archive it
 *   npm run test:archive-cs
 *
 * Or with environment variables:
 *   export MNEMONIC="your mnemonic here"
 *   export VERANA_RPC_ENDPOINT="http://localhost:26657"
 *   export CS_ID=1
 *   npm run test:archive-cs
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
import { MsgCreateCredentialSchema, MsgArchiveCredentialSchema, OptionalUInt32 } from "../../../src/codec/verana/cs/v1/tx";
import { MsgCreateTrustRegistry } from "../../../src/codec/verana/tr/v1/tx";
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
  console.log("Journey: Archive Credential Schema (TypeScript Client)");
  console.log("=".repeat(60));
  console.log();

  // Step 1: Setup wallet (using Amino Sign to match frontend)
  console.log("Step 1: Setting up wallet (Amino Sign mode)...");
  const wallet = await createWallet(TEST_MNEMONIC);
  const account = await getAccountInfo(wallet);
  console.log(`  ✓ Wallet address: ${account.address}`);
  console.log(`  ✓ Using Amino Sign (matches frontend)`);
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

  // Step 4: Get or create Credential Schema ID
  let csId: number | undefined;
  if (process.env.CS_ID) {
    csId = parseInt(process.env.CS_ID, 10);
    if (isNaN(csId)) {
      console.log("  ❌ Invalid CS_ID provided");
      process.exit(1);
    }
    console.log(`Step 4: Using provided Credential Schema ID: ${csId}`);
  } else {
    console.log("Step 4: Creating a Credential Schema first (no CS_ID provided)...");
    
    // First create a Trust Registry
    const did = generateUniqueDID();
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
      "Creating Trust Registry for archive schema test"
    );

    const createTrResult = await client.signAndBroadcast(
      account.address,
      [createTrMsg],
      createTrFee,
      "Creating Trust Registry for archive schema test"
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

    // Now create a Credential Schema
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
      "Creating Credential Schema for archive test"
    );

    const createCsResult = await client.signAndBroadcast(
      account.address,
      [createCsMsg],
      createCsFee,
      "Creating Credential Schema for archive test"
    );

    if (createCsResult.code !== 0) {
      console.log("  ❌ Failed to create Credential Schema");
      console.log(`  Error: ${createCsResult.rawLog}`);
      process.exit(1);
    }

    // Extract CS ID from events
    const csEvents = createCsResult.events || [];
    for (const event of csEvents) {
      if (event.type === "create_credential_schema" || event.type === "verana.cs.v1.EventCreateCredentialSchema") {
        for (const attr of event.attributes) {
          if (attr.key === "credential_schema_id" || attr.key === "id" || attr.key === "cs_id") {
            csId = parseInt(attr.value, 10);
            if (!isNaN(csId)) {
              console.log(`  ✓ Created Credential Schema with ID: ${csId}`);
              break;
            }
          }
        }
        if (csId) break;
      }
    }

    if (!csId || isNaN(csId)) {
      console.log("  ❌ Could not extract CS ID from events");
      process.exit(1);
    }
  }

  if (!csId) {
    console.log("  ❌ Credential Schema ID is required");
    process.exit(1);
  }

  console.log();

  // Step 5: Archive Credential Schema message
  console.log("Step 5: Archiving Credential Schema transaction...");
  const archive = true; // Set to true to archive, false to unarchive
  const msg = {
    typeUrl: typeUrls.MsgArchiveCredentialSchema,
    value: MsgArchiveCredentialSchema.fromPartial({
      creator: account.address,
      id: csId,
      archive: archive,
    }),
  };
  console.log("  Message details:");
  console.log(`    - Creator: ${account.address}`);
  console.log(`    - Credential Schema ID: ${csId}`);
  console.log(`    - Archive: ${archive}`);
  console.log();

  // Step 6: Sign and broadcast
  console.log("Step 6: Signing and broadcasting transaction...");
  try {
    const fee = await calculateFeeWithSimulation(
      client,
      account.address,
      [msg],
      "Archiving Credential Schema via TypeScript client"
    );
    console.log(`  Calculated gas: ${fee.gas}, fee: ${fee.amount[0].amount}${fee.amount[0].denom}`);
    
    const result = await client.signAndBroadcast(
      account.address,
      [msg],
      fee,
      "Archiving Credential Schema via TypeScript client"
    );

    console.log();
    if (result.code === 0) {
      console.log(`✅ SUCCESS! Credential Schema ${archive ? "archived" : "unarchived"} successfully!`);
      console.log("=".repeat(60));
      console.log(`  Transaction Hash: ${result.transactionHash}`);
      console.log(`  Block Height: ${result.height}`);
      console.log(`  Gas Used: ${result.gasUsed}/${result.gasWanted}`);
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

