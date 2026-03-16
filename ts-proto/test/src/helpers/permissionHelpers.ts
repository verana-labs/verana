/**
 * Permission Test Helpers
 * Shared utilities for permission-related tests
 */

import { SigningStargateClient } from "@cosmjs/stargate";
import { typeUrls } from "./registry";
import {
  MsgCreateRootPermission,
  MsgCreatePermission,
  MsgStartPermissionVP,
  MsgSetPermissionVPToValidated,
} from "../../../src/codec/verana/perm/v1/tx";
import { MsgCreateTrustRegistry } from "../../../src/codec/verana/tr/v1/tx";
import { MsgCreateCredentialSchema, OptionalUInt32 } from "../../../src/codec/verana/cs/v1/tx";
import { CredentialSchemaPermManagementMode, PricingAssetType } from "../../../src/codec/verana/cs/v1/types";
import { PermissionType, OptionalUInt64 } from "../../../src/codec/verana/perm/v1/types";
// Note: We use Date objects directly, not Timestamp objects
import { calculateFeeWithSimulation, generateUniqueDID, signAndBroadcastWithRetry, waitForPermissionToBecomeEffective, createQueryClient } from "./client";

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
      authority: address,
      operator: "",
      schemaId: schemaId,
      did: did,
      effectiveFrom: effectiveFrom,
      effectiveUntil: effectiveUntil,
      validationFees: 5,
      verificationFees: 5,
      issuanceFees: 5,
    }),
  };

  try {
    // Get sequence BEFORE transaction to track if it increments
    const sequenceBefore = await client.getSequence(address);
    
    const fee = await calculateFeeWithSimulation(client, address, [msg], "Creating root permission (ecosystem permission) for test");
    // Use retry logic for consistency (matches frontend pattern)
    const result = await signAndBroadcastWithRetry(client, address, [msg], fee, "Creating root permission (ecosystem permission) for test");

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
              // Wait for transaction to be included in a block and sequence to be updated
              // This is important because the next transaction needs the sequence to be incremented
              const txHash = result.transactionHash;
              const queryClient = await import("./client").then(m => m.createQueryClient());
              try {
                // Wait up to 10 seconds for transaction to be queryable (means it's in a block)
                let found = false;
                for (let i = 0; i < 10; i++) {
                  try {
                    const tx = await queryClient.getTx(txHash);
                    if (tx) {
                      found = true;
                      break;
                    }
                  } catch {
                    // Transaction not found yet, continue waiting
                  }
                  await new Promise((resolve) => setTimeout(resolve, 1000));
                }
                if (!found) {
                  console.log(`  ⚠️  Warning: Could not confirm transaction ${txHash} was included in a block, but continuing...`);
                }
                // Verify sequence has actually incremented on-chain
                // Compare to sequence BEFORE transaction
                let sequenceUpdated = false;
                for (let i = 0; i < 10; i++) {
                  await new Promise((resolve) => setTimeout(resolve, 500));
                  const currentSequence = await client.getSequence(address);
                  if (currentSequence.sequence > sequenceBefore.sequence) {
                    sequenceUpdated = true;
                    break;
                  }
                }
                if (!sequenceUpdated) {
                  const finalSequence = await client.getSequence(address);
                  console.log(`  ⚠️  Warning: Sequence may not have updated yet. Before: ${sequenceBefore.sequence}, After: ${finalSequence.sequence}`);
                }
              } finally {
                queryClient.disconnect();
              }
              // Force sequence refresh to ensure client cache is updated
              await client.getSequence(address);
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
 * IMPORTANT: Creates a root (ecosystem) permission first as it's required by spec (unless skipRoot is true)
 * The ecosystem permission must exist before any regular permissions can be created for a schema
 * 
 * Note: Sets effectiveFrom to 10 seconds in the future as required by the blockchain
 * For operations that need immediate effectiveness (like Extend/Revoke), wait for effectiveFrom to pass
 * 
 * @param skipRoot - If true, skip creating root permission (assumes it already exists)
 */
export async function createPermissionForTest(
  client: SigningStargateClient,
  address: string,
  schemaId: number,
  did: string,
  type: PermissionType = PermissionType.ISSUER,
  skipRoot: boolean = false
): Promise<number> {
  // First, create root (ecosystem) permission - REQUIRED before creating regular permissions
  // This is the prerequisite that must exist for the schema
  // Skip if skipRoot is true (root permission already created)
  if (!skipRoot) {
    console.log(`  Creating root (ecosystem) permission for schema ${schemaId} first (required prerequisite)...`);
    try {
      await createRootPermissionForTest(client, address, schemaId, did);
      console.log(`  ✓ Root (ecosystem) permission created successfully`);
      // Wait a bit to ensure the transaction is fully committed and sequence is updated
      // This prevents sequence synchronization issues when creating the regular permission immediately after
      await new Promise((resolve) => setTimeout(resolve, 2000));
      // Force sequence refresh multiple times to ensure cache is cleared
      await client.getSequence(address);
      await new Promise((resolve) => setTimeout(resolve, 500));
      await client.getSequence(address);
    } catch (error: any) {
      throw new Error(`Failed to create root (ecosystem) permission prerequisite for schema ${schemaId}: ${error.message}`);
    }
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
      authority: address,
      operator: "",
      type: type,
      validatorPermId: 0,
      did: did,
      effectiveFrom: effectiveFrom,
      effectiveUntil: effectiveUntil,
      verificationFees: verificationFees,
      validationFees: validationFees,
    }),
  };

  try {
    // Get sequence BEFORE transaction to track if it increments
    const sequenceBefore = await client.getSequence(address);
    
    const fee = await calculateFeeWithSimulation(client, address, [msg], "Creating permission for test");
    // Use retry logic for consistency (matches frontend pattern)
    const result = await signAndBroadcastWithRetry(client, address, [msg], fee, "Creating permission for test");

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
              // Wait for transaction to be included in a block and sequence to be updated
              const txHash = result.transactionHash;
              const queryClient = await import("./client").then(m => m.createQueryClient());
              try {
                // Wait up to 10 seconds for transaction to be queryable (means it's in a block)
                let found = false;
                for (let i = 0; i < 10; i++) {
                  try {
                    const tx = await queryClient.getTx(txHash);
                    if (tx) {
                      found = true;
                      break;
                    }
                  } catch {
                    // Transaction not found yet, continue waiting
                  }
                  await new Promise((resolve) => setTimeout(resolve, 1000));
                }
                if (!found) {
                  console.log(`  ⚠️  Warning: Could not confirm transaction ${txHash} was included in a block, but continuing...`);
                }
                // Verify sequence has actually incremented on-chain
                // Compare to sequence BEFORE transaction
                let sequenceUpdated = false;
                for (let i = 0; i < 10; i++) {
                  await new Promise((resolve) => setTimeout(resolve, 500));
                  const currentSequence = await client.getSequence(address);
                  if (currentSequence.sequence > sequenceBefore.sequence) {
                    sequenceUpdated = true;
                    break;
                  }
                }
                if (!sequenceUpdated) {
                  const finalSequence = await client.getSequence(address);
                  console.log(`  ⚠️  Warning: Sequence may not have updated yet. Before: ${sequenceBefore.sequence}, After: ${finalSequence.sequence}`);
                }
              } finally {
                queryClient.disconnect();
              }
              // Force sequence refresh to ensure client cache is updated
              await client.getSequence(address);
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
): Promise<{ schemaId: number; did: string; trustRegistryId: number }> {
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
      authority: address,
      operator: "",
      did: did,
      aka: "http://example-trust-registry.com",
      language: "en",
      docUrl: "https://example.com/governance-framework.pdf",
      docDigestSri: "sha384-MzNNbQTWCSUSi0bbz7dbua+RcENv7C6FvlmYJ1Y+I727HsPOHdzwELMYO9Mz68M26",
    }),
  };

  const createTrFee = await calculateFeeWithSimulation(client, address, [createTrMsg], "Creating TR for schema");
  // Use retry logic for consistency (matches frontend pattern)
  const createTrResult = await signAndBroadcastWithRetry(client, address, [createTrMsg], createTrFee, "Creating TR for schema");

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

  // Wait for TR transaction to be confirmed and sequence updated
  const trTxHash = createTrResult.transactionHash;
  const queryClient1 = await import("./client").then(m => m.createQueryClient());
  try {
    let found = false;
    for (let i = 0; i < 10; i++) {
      try {
        const tx = await queryClient1.getTx(trTxHash);
        if (tx) {
          found = true;
          break;
        }
      } catch {
        // Transaction not found yet, continue waiting
      }
      await new Promise((resolve) => setTimeout(resolve, 1000));
    }
    // Verify sequence has incremented
    const initialSeq = await client.getSequence(address);
    for (let i = 0; i < 10; i++) {
      await new Promise((resolve) => setTimeout(resolve, 500));
      const currentSeq = await client.getSequence(address);
      if (currentSeq.sequence > initialSeq.sequence) {
        break;
      }
    }
  } finally {
    queryClient1.disconnect();
  }
  
  // Force sequence refresh to ensure client cache is updated before creating CS
  await client.getSequence(address);
  await new Promise((resolve) => setTimeout(resolve, 500));
  await client.getSequence(address);

  // Create Credential Schema
  const createCsMsg = {
    typeUrl: typeUrls.MsgCreateCredentialSchema,
    value: MsgCreateCredentialSchema.fromPartial({
      authority: address,
      operator: "",
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

  // Get sequence BEFORE CS transaction to track if it increments
  const sequenceBeforeCs = await client.getSequence(address);
  
  const createCsFee = await calculateFeeWithSimulation(client, address, [createCsMsg], "Creating schema");
  // Use retry logic for consistency (matches frontend pattern)
  const createCsResult = await signAndBroadcastWithRetry(client, address, [createCsMsg], createCsFee, "Creating schema");

  if (createCsResult.code !== 0) {
    throw new Error(`Failed to create Credential Schema: ${createCsResult.rawLog}`);
  }

  // Extract Schema ID
  const csEvents = createCsResult.events || [];
  let schemaId: number | undefined;
  for (const event of csEvents) {
    if (event.type === "create_credential_schema" || event.type === "verana.cs.v1.EventCreateCredentialSchema") {
      for (const attr of event.attributes) {
        if (attr.key === "credential_schema_id" || attr.key === "id" || attr.key === "cs_id") {
          schemaId = parseInt(attr.value, 10);
          if (!isNaN(schemaId)) {
            break;
          }
        }
      }
      if (schemaId) break;
    }
  }

  if (!schemaId) {
    throw new Error("Could not extract Schema ID from events");
  }

  // Wait for CS transaction to be confirmed and sequence updated before returning
  const csTxHash = createCsResult.transactionHash;
  const queryClient2 = await import("./client").then(m => m.createQueryClient());
  try {
    let found = false;
    for (let i = 0; i < 10; i++) {
      try {
        const tx = await queryClient2.getTx(csTxHash);
        if (tx) {
          found = true;
          break;
        }
      } catch {
        // Transaction not found yet, continue waiting
      }
      await new Promise((resolve) => setTimeout(resolve, 1000));
    }
    // Verify sequence has incremented (compare to sequence BEFORE transaction)
    for (let i = 0; i < 10; i++) {
      await new Promise((resolve) => setTimeout(resolve, 500));
      const currentSeq = await client.getSequence(address);
      if (currentSeq.sequence > sequenceBeforeCs.sequence) {
        break;
      }
    }
  } finally {
    queryClient2.disconnect();
  }
  
  // Force sequence refresh to ensure client cache is updated
  await client.getSequence(address);

  // Save as active TR/CS so subsequent tests can reuse them
  const { saveActiveTR, saveActiveCS } = await import("./journeyResults");
  saveActiveTR(trId, did);
  saveActiveCS(schemaId, trId, did);

  return { schemaId, did, trustRegistryId: trId };
}

// ============================================================
// Operator-signed helpers for PERM journey tests
// ============================================================

/**
 * Extracts a numeric ID from transaction events.
 * Searches for the specified event type and attribute key.
 */
export function extractIdFromEvents(
  events: readonly { type: string; attributes: readonly { key: string; value: string }[] }[],
  eventType: string,
  attrKeys: string[],
): number | undefined {
  for (const event of events) {
    if (event.type === eventType || event.type.includes(eventType)) {
      for (const attr of event.attributes) {
        if (attrKeys.includes(attr.key)) {
          const id = parseInt(attr.value, 10);
          if (!isNaN(id)) return id;
        }
      }
    }
  }
  return undefined;
}

/**
 * Waits for a transaction to be confirmed and sequence to advance.
 */
async function waitForTxConfirmation(
  client: SigningStargateClient,
  txHash: string,
  address: string,
  sequenceBefore: number,
): Promise<void> {
  const queryClient = await createQueryClient();
  try {
    for (let i = 0; i < 15; i++) {
      try {
        const tx = await queryClient.getTx(txHash);
        if (tx) break;
      } catch {}
      await new Promise((r) => setTimeout(r, 1000));
    }
    for (let i = 0; i < 15; i++) {
      await new Promise((r) => setTimeout(r, 500));
      const seq = await client.getSequence(address);
      if (seq.sequence > sequenceBefore) break;
    }
  } finally {
    queryClient.disconnect();
  }
  await client.getSequence(address);
}

/**
 * Creates a Trust Registry using operator-signed pattern.
 * Returns the TR ID and DID.
 */
export async function createTRWithOperator(
  client: SigningStargateClient,
  authority: string,
  operator: string,
): Promise<{ trId: number; did: string }> {
  const did = generateUniqueDID();
  const msg = {
    typeUrl: typeUrls.MsgCreateTrustRegistry,
    value: MsgCreateTrustRegistry.fromPartial({
      authority,
      operator,
      did,
      aka: "http://perm-test-trust-registry.com",
      language: "en",
      docUrl: "https://example.com/perm-governance-framework.pdf",
      docDigestSri: "sha384-MzNNbQTWCSUSi0bbz7dbua+RcENv7C6FvlmYJ1Y+I727HsPOHdzwELMYO9Mz68M26",
    }),
  };

  const seqBefore = (await client.getSequence(operator)).sequence;
  const fee = await calculateFeeWithSimulation(client, operator, [msg], "Creating TR for perm test");
  const result = await signAndBroadcastWithRetry(client, operator, [msg], fee, "Creating TR for perm test");

  if (result.code !== 0) {
    throw new Error(`Failed to create TR: ${result.rawLog}`);
  }

  const trId = extractIdFromEvents(result.events || [], "create_trust_registry", ["trust_registry_id", "id", "tr_id"]);
  if (!trId) throw new Error("Could not extract TR ID from events");

  await waitForTxConfirmation(client, result.transactionHash, operator, seqBefore);
  return { trId, did };
}

/**
 * Creates a Credential Schema using operator-signed pattern.
 * Returns the schema ID.
 */
export async function createCSWithOperator(
  client: SigningStargateClient,
  authority: string,
  operator: string,
  trId: number,
  mode: CredentialSchemaPermManagementMode,
): Promise<number> {
  const jsonSchema = JSON.stringify({
    $id: "vpr:verana:VPR_CHAIN_ID/cs/v1/js/VPR_CREDENTIAL_SCHEMA_ID",
    $schema: "https://json-schema.org/draft/2020-12/schema",
    title: "PermTestCredential",
    description: "Credential schema for permission tests",
    type: "object",
    properties: {
      credentialSubject: {
        type: "object",
        properties: {
          id: { type: "string", format: "uri" },
          name: { type: "string", minLength: 1, maxLength: 256 },
        },
      },
    },
  });

  const msg = {
    typeUrl: typeUrls.MsgCreateCredentialSchema,
    value: MsgCreateCredentialSchema.fromPartial({
      authority,
      operator,
      trId,
      jsonSchema,
      issuerGrantorValidationValidityPeriod: OptionalUInt32.fromPartial({ value: 0 }),
      verifierGrantorValidationValidityPeriod: OptionalUInt32.fromPartial({ value: 0 }),
      issuerValidationValidityPeriod: OptionalUInt32.fromPartial({ value: 0 }),
      verifierValidationValidityPeriod: OptionalUInt32.fromPartial({ value: 0 }),
      holderValidationValidityPeriod: OptionalUInt32.fromPartial({ value: 0 }),
      issuerPermManagementMode: mode,
      verifierPermManagementMode: CredentialSchemaPermManagementMode.OPEN,
      pricingAssetType: PricingAssetType.TU,
      pricingAsset: "tu",
      digestAlgorithm: "sha256",
    }),
  };

  const seqBefore = (await client.getSequence(operator)).sequence;
  const fee = await calculateFeeWithSimulation(client, operator, [msg], "Creating CS for perm test");
  const result = await signAndBroadcastWithRetry(client, operator, [msg], fee, "Creating CS for perm test");

  if (result.code !== 0) {
    throw new Error(`Failed to create CS: ${result.rawLog}`);
  }

  const csId = extractIdFromEvents(result.events || [], "create_credential_schema", ["credential_schema_id", "id", "cs_id"]);
  if (!csId) throw new Error("Could not extract CS ID from events");

  await waitForTxConfirmation(client, result.transactionHash, operator, seqBefore);
  return csId;
}

/**
 * Creates a root (ECOSYSTEM) permission using operator-signed pattern.
 * Returns the root permission ID.
 */
export async function createRootPermWithOperator(
  client: SigningStargateClient,
  authority: string,
  operator: string,
  schemaId: number,
  did: string,
  opts?: { validationFees?: number; issuanceFees?: number; verificationFees?: number },
): Promise<{ rootPermId: number; effectiveFrom: Date }> {
  const effectiveFrom = new Date(Date.now() + 10000); // 10s in future
  const effectiveUntil = new Date(effectiveFrom.getTime() + 360 * 24 * 60 * 60 * 1000); // 360 days

  const msg = {
    typeUrl: typeUrls.MsgCreateRootPermission,
    value: MsgCreateRootPermission.fromPartial({
      authority,
      operator,
      schemaId,
      did,
      effectiveFrom,
      effectiveUntil,
      validationFees: opts?.validationFees ?? 5,
      issuanceFees: opts?.issuanceFees ?? 5,
      verificationFees: opts?.verificationFees ?? 5,
    }),
  };

  const seqBefore = (await client.getSequence(operator)).sequence;
  const fee = await calculateFeeWithSimulation(client, operator, [msg], "Creating root permission");
  const result = await signAndBroadcastWithRetry(client, operator, [msg], fee, "Creating root permission");

  if (result.code !== 0) {
    throw new Error(`Failed to create root permission: ${result.rawLog}`);
  }

  const rootPermId = extractIdFromEvents(result.events || [], "create_root_permission", ["root_permission_id", "permission_id", "id"]);
  if (!rootPermId) throw new Error("Could not extract root permission ID from events");

  await waitForTxConfirmation(client, result.transactionHash, operator, seqBefore);
  return { rootPermId, effectiveFrom };
}

/**
 * Creates a full prerequisite chain: TR → CS → Root Permission.
 * Used by multiple perm journey tests.
 */
export async function createPermPrerequisites(
  client: SigningStargateClient,
  authority: string,
  operator: string,
  mode: CredentialSchemaPermManagementMode = CredentialSchemaPermManagementMode.ECOSYSTEM,
): Promise<{ trId: number; schemaId: number; rootPermId: number; did: string; effectiveFrom: Date }> {
  console.log("  Creating TR...");
  const { trId, did } = await createTRWithOperator(client, authority, operator);
  console.log(`  ✓ TR created (ID: ${trId})`);

  console.log("  Creating CS...");
  const schemaId = await createCSWithOperator(client, authority, operator, trId, mode);
  console.log(`  ✓ CS created (ID: ${schemaId}, mode: ${mode === CredentialSchemaPermManagementMode.OPEN ? "OPEN" : mode === CredentialSchemaPermManagementMode.ECOSYSTEM ? "ECOSYSTEM" : "GRANTOR_VALIDATION"})`);

  console.log("  Creating root permission...");
  const { rootPermId, effectiveFrom } = await createRootPermWithOperator(client, authority, operator, schemaId, did);
  console.log(`  ✓ Root permission created (ID: ${rootPermId})`);

  return { trId, schemaId, rootPermId, did, effectiveFrom };
}

/**
 * Starts a VP and validates it, creating a child permission.
 * Returns the child permission ID.
 */
export async function createValidatedPermission(
  client: SigningStargateClient,
  authority: string,
  operator: string,
  schemaId: number,
  rootPermId: number,
  did: string,
): Promise<number> {
  // Start VP
  const startMsg = {
    typeUrl: typeUrls.MsgStartPermissionVP,
    value: MsgStartPermissionVP.fromPartial({
      authority,
      operator,
      type: PermissionType.ISSUER,
      validatorPermId: rootPermId,
      did,
      validationFees: OptionalUInt64.fromPartial({ value: 5 }),
      issuanceFees: OptionalUInt64.fromPartial({ value: 5 }),
      verificationFees: OptionalUInt64.fromPartial({ value: 5 }),
      vsOperator: "",
      vsOperatorAuthzEnabled: false,
    }),
  };

  const seqBefore1 = (await client.getSequence(operator)).sequence;
  const startFee = await calculateFeeWithSimulation(client, operator, [startMsg], "Starting VP");
  const startResult = await signAndBroadcastWithRetry(client, operator, [startMsg], startFee, "Starting VP");

  if (startResult.code !== 0) {
    throw new Error(`Failed to start VP: ${startResult.rawLog}`);
  }

  const vpPermId = extractIdFromEvents(startResult.events || [], "start_permission_vp", ["permission_id", "id"]);
  if (!vpPermId) throw new Error("Could not extract VP permission ID from events");

  await waitForTxConfirmation(client, startResult.transactionHash, operator, seqBefore1);

  // Validate VP
  // Must be <= validator_perm.effective_until (root uses effectiveFrom + 360 days)
  const effectiveUntil = new Date(Date.now() + 300 * 24 * 60 * 60 * 1000); // 300 days
  const validateMsg = {
    typeUrl: typeUrls.MsgSetPermissionVPToValidated,
    value: MsgSetPermissionVPToValidated.fromPartial({
      authority,
      operator,
      id: vpPermId,
      effectiveUntil,
      validationFees: 5,
      issuanceFees: 5,
      verificationFees: 5,
      vpSummaryDigestSri: "sha384-validationSummaryDigest123",
      issuanceFeeDiscount: 0,
      verificationFeeDiscount: 0,
    }),
  };

  const seqBefore2 = (await client.getSequence(operator)).sequence;
  const validateFee = await calculateFeeWithSimulation(client, operator, [validateMsg], "Validating VP");
  const validateResult = await signAndBroadcastWithRetry(client, operator, [validateMsg], validateFee, "Validating VP");

  if (validateResult.code !== 0) {
    throw new Error(`Failed to validate VP: ${validateResult.rawLog}`);
  }

  await waitForTxConfirmation(client, validateResult.transactionHash, operator, seqBefore2);
  return vpPermId;
}
