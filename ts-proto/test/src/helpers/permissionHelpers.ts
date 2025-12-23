/**
 * Permission Test Helpers
 * Shared utilities for permission-related tests
 */

import { SigningStargateClient } from "@cosmjs/stargate";
import { typeUrls } from "./registry";
import { MsgCreateRootPermission, MsgCreatePermission } from "../../../src/codec/verana/perm/v1/tx";
import { MsgCreateTrustRegistry } from "../../../src/codec/verana/tr/v1/tx";
import { MsgCreateCredentialSchema, OptionalUInt32 } from "../../../src/codec/verana/cs/v1/tx";
import { CredentialSchemaPermManagementMode } from "../../../src/codec/verana/cs/v1/types";
import { PermissionType } from "../../../src/codec/verana/perm/v1/types";
// Note: We use Date objects directly, not Timestamp objects
import { calculateFeeWithSimulation, generateUniqueDID } from "./client";

// Note: The generated protobuf code expects Date objects directly, not Timestamp objects.
// The toTimestamp conversion happens automatically during encoding in the generated code.

/**
 * Creates a root permission and returns its ID
 * This creates the ecosystem permission which is REQUIRED before creating any regular permissions
 * 
 * Note: Sets effectiveFrom to 10 seconds in the future as required by the blockchain
 * The permission will become effective after this time passes
 */
export async function createRootPermissionForTest(
  client: SigningStargateClient,
  address: string,
  schemaId: number,
  did: string
): Promise<number> {
  // Set effectiveFrom to 10 seconds in the future as required by blockchain (matches test harness)
  const effectiveFrom = new Date(Date.now() + 10000);
  const effectiveUntil = new Date(effectiveFrom.getTime() + 360 * 24 * 60 * 60 * 1000); // 360 days from effectiveFrom

  const msg = {
    typeUrl: typeUrls.MsgCreateRootPermission,
    value: MsgCreateRootPermission.fromPartial({
      creator: address,
      schemaId: schemaId,
      did: did,
      country: "US",
      effectiveFrom: effectiveFrom,
      effectiveUntil: effectiveUntil,
      validationFees: 5,
      verificationFees: 5,
      issuanceFees: 5,
    }),
  };

  try {
    const fee = await calculateFeeWithSimulation(client, address, [msg], "Creating root permission (ecosystem permission) for test");
    const result = await client.signAndBroadcast(address, [msg], fee, "Creating root permission (ecosystem permission) for test");

    if (result.code !== 0) {
      throw new Error(`Failed to create root permission (ecosystem permission) for schema ${schemaId}: ${result.rawLog}`);
    }

    // Extract permission ID from events
    const events = result.events || [];
    for (const event of events) {
      if (event.type === "create_root_permission" || event.type === "verana.perm.v1.EventCreateRootPermission") {
        for (const attr of event.attributes) {
          if (attr.key === "root_permission_id" || attr.key === "permission_id" || attr.key === "id") {
            const permId = parseInt(attr.value, 10);
            if (!isNaN(permId)) {
              // Wait a bit to ensure the transaction is fully committed to state
              // This is important because the next transaction needs to query for the ecosystem permission
              await new Promise((resolve) => setTimeout(resolve, 1000));
              return permId;
            }
          }
        }
      }
    }

    throw new Error(`Could not extract root permission ID from events for schema ${schemaId}`);
  } catch (error: any) {
    // Provide more context in error message
    if (error.message?.includes("simulate") || error.message?.includes("Query failed") || error.message?.includes("ecosystem permission not found")) {
      throw new Error(`Failed to create root permission (ecosystem permission) for schema ${schemaId} during simulation: ${error.message}. Make sure the schema exists and you have permission to create root permissions.`);
    }
    throw error;
  }
}

/**
 * Creates a permission and returns its ID
 * IMPORTANT: Creates a root (ecosystem) permission first as it's required by spec
 * The ecosystem permission must exist before any regular permissions can be created for a schema
 * 
 * Note: Sets effectiveFrom to 10 seconds in the future as required by the blockchain
 * For operations that need immediate effectiveness (like Extend/Revoke), wait for effectiveFrom to pass
 */
export async function createPermissionForTest(
  client: SigningStargateClient,
  address: string,
  schemaId: number,
  did: string,
  type: PermissionType = PermissionType.ISSUER
): Promise<number> {
  // First, create root (ecosystem) permission - REQUIRED before creating regular permissions
  // This is the prerequisite that must exist for the schema
  console.log(`  Creating root (ecosystem) permission for schema ${schemaId} first (required prerequisite)...`);
  try {
    await createRootPermissionForTest(client, address, schemaId, did);
    console.log(`  ✓ Root (ecosystem) permission created successfully`);
  } catch (error: any) {
    throw new Error(`Failed to create root (ecosystem) permission prerequisite for schema ${schemaId}: ${error.message}`);
  }

  // Set effectiveFrom to 10 seconds in the future as required by blockchain (matches test harness)
  const effectiveFrom = new Date(Date.now() + 10000);
  const effectiveUntil = new Date(effectiveFrom.getTime() + 360 * 24 * 60 * 60 * 1000); // 360 days from effectiveFrom

  // VERIFIER permissions cannot have verification_fees or validation_fees (must be 0)
  // Only ISSUER permissions can have these fees
  const verificationFees = type === PermissionType.ISSUER ? 1000 : 0;
  const validationFees = type === PermissionType.ISSUER ? 1000 : 0;

  const msg = {
    typeUrl: typeUrls.MsgCreatePermission,
    value: MsgCreatePermission.fromPartial({
      creator: address,
      schemaId: schemaId,
      type: type,
      did: did,
      country: "US",
      effectiveFrom: effectiveFrom,
      effectiveUntil: effectiveUntil,
      verificationFees: verificationFees,
      validationFees: validationFees,
    }),
  };

  try {
    const fee = await calculateFeeWithSimulation(client, address, [msg], "Creating permission for test");
    const result = await client.signAndBroadcast(address, [msg], fee, "Creating permission for test");

    if (result.code !== 0) {
      throw new Error(`Failed to create permission for schema ${schemaId}: ${result.rawLog}`);
    }

    // Extract permission ID from events
    const events = result.events || [];
    for (const event of events) {
      if (event.type === "create_permission" || event.type === "verana.perm.v1.EventCreatePermission") {
        for (const attr of event.attributes) {
          if (attr.key === "permission_id" || attr.key === "id") {
            const permId = parseInt(attr.value, 10);
            if (!isNaN(permId)) {
              // Wait a bit to ensure the transaction is fully committed
              await new Promise((resolve) => setTimeout(resolve, 500));
              return permId;
            }
          }
        }
      }
    }

    throw new Error(`Could not extract permission ID from events for schema ${schemaId}`);
  } catch (error: any) {
    // Provide more context in error message
    if (error.message?.includes("ecosystem permission not found")) {
      throw new Error(`Failed to create permission: Ecosystem permission (root permission) not found for schema ${schemaId}. This should have been created as a prerequisite. Error: ${error.message}`);
    }
    throw error;
  }
}

/**
 * Creates a schema and returns its ID (creates TR first if needed)
 */
export async function createSchemaForTest(
  client: SigningStargateClient,
  address: string
): Promise<{ schemaId: number; did: string }> {
  // Generate schema JSON
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

  // Create Trust Registry
  const did = generateUniqueDID();
  const createTrMsg = {
    typeUrl: typeUrls.MsgCreateTrustRegistry,
    value: MsgCreateTrustRegistry.fromPartial({
      creator: address,
      did: did,
      aka: "http://example-trust-registry.com",
      language: "en",
      docUrl: "https://example.com/governance-framework.pdf",
      docDigestSri: "sha384-MzNNbQTWCSUSi0bbz7dbua+RcENv7C6FvlmYJ1Y+I727HsPOHdzwELMYO9Mz68M26",
    }),
  };

  const createTrFee = await calculateFeeWithSimulation(client, address, [createTrMsg], "Creating TR for schema");
  const createTrResult = await client.signAndBroadcast(address, [createTrMsg], createTrFee, "Creating TR for schema");

  if (createTrResult.code !== 0) {
    throw new Error(`Failed to create Trust Registry: ${createTrResult.rawLog}`);
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
    throw new Error("Could not extract TR ID from events");
  }

  // Create Credential Schema
  const createCsMsg = {
    typeUrl: typeUrls.MsgCreateCredentialSchema,
    value: MsgCreateCredentialSchema.fromPartial({
      creator: address,
      trId: trId,
      jsonSchema: generateSimpleSchema(trId.toString()),
      issuerGrantorValidationValidityPeriod: { value: 0 } as OptionalUInt32,
      verifierGrantorValidationValidityPeriod: { value: 0 } as OptionalUInt32,
      issuerValidationValidityPeriod: { value: 0 } as OptionalUInt32,
      verifierValidationValidityPeriod: { value: 0 } as OptionalUInt32,
      holderValidationValidityPeriod: { value: 0 } as OptionalUInt32,
      issuerPermManagementMode: CredentialSchemaPermManagementMode.OPEN,
      verifierPermManagementMode: CredentialSchemaPermManagementMode.OPEN,
    }),
  };

  const createCsFee = await calculateFeeWithSimulation(client, address, [createCsMsg], "Creating schema");
  const createCsResult = await client.signAndBroadcast(address, [createCsMsg], createCsFee, "Creating schema");

  if (createCsResult.code !== 0) {
    throw new Error(`Failed to create Credential Schema: ${createCsResult.rawLog}`);
  }

  // Extract Schema ID
  const csEvents = createCsResult.events || [];
  for (const event of csEvents) {
    if (event.type === "create_credential_schema" || event.type === "verana.cs.v1.EventCreateCredentialSchema") {
      for (const attr of event.attributes) {
        if (attr.key === "credential_schema_id" || attr.key === "id" || attr.key === "cs_id") {
          const schemaId = parseInt(attr.value, 10);
          if (!isNaN(schemaId)) {
            return { schemaId, did };
          }
        }
      }
    }
  }

  throw new Error("Could not extract Schema ID from events");
}

