/**
 * Journey: Setup Permission Session Prerequisites
 *
 * Step 1 of 3 for Permission Session creation.
 * Creates Trust Registry, Credential Schema, and Root Permission.
 * Saves results to journey_results/ for subsequent steps.
 *
 * Usage:
 *   npm run test:setup-perm-session-prereqs
 */

import {
    createAminoWallet,
    createSigningClient,
    getAccountInfo,
    calculateFeeWithSimulation,
    signAndBroadcastWithRetry,
    generateUniqueDID,
    config,
    waitForSequencePropagation,
} from "../helpers/client";
import { typeUrls } from "../helpers/registry";
import { MsgCreateTrustRegistry } from "../../../src/codec/verana/tr/v1/tx";
import { MsgCreateCredentialSchema, OptionalUInt32 } from "../../../src/codec/verana/cs/v1/tx";
import { MsgCreateRootPermission } from "../../../src/codec/verana/perm/v1/tx";
import { CredentialSchemaPermManagementMode } from "../../../src/codec/verana/cs/v1/types";
import { saveJourneyResult } from "../helpers/journeyResults";

const TEST_MNEMONIC =
    process.env.MNEMONIC ||
    "pink glory help gown abstract eight nice crazy forward ketchup skill cheese";

async function main() {
    console.log("=".repeat(60));
    console.log("Journey: Setup Permission Session Prerequisites (Step 1/3)");
    console.log("=".repeat(60));
    console.log();

    // Setup wallet and client
    const wallet = await createAminoWallet(TEST_MNEMONIC);
    const account = await getAccountInfo(wallet);
    const client = await createSigningClient(wallet);
    console.log(`  ✓ Wallet address: ${account.address}`);
    console.log(`  ✓ Connected to ${config.rpcEndpoint}`);
    console.log();

    // Check balance
    const balance = await client.getBalance(account.address, config.denom);
    if (BigInt(balance.amount) < BigInt(1000000)) {
        console.log("  ⚠️  Warning: Low balance.");
        process.exit(1);
    }

    // Step 1: Create Trust Registry
    console.log("Step 1: Creating Trust Registry...");
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

    const trFee = await calculateFeeWithSimulation(client, account.address, [createTrMsg], "Creating TR for perm session");
    const trResult = await signAndBroadcastWithRetry(client, account.address, [createTrMsg], trFee, "Creating TR for perm session");

    if (trResult.code !== 0) {
        console.log(`  ❌ Failed to create Trust Registry: ${trResult.rawLog}`);
        process.exit(1);
    }

    // Extract TR ID
    let trId: number | undefined;
    for (const event of trResult.events || []) {
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

    console.log(`  ✓ Trust Registry created: ID=${trId}, DID=${did}`);
    console.log();

    // Wait for sequence propagation
    await waitForSequencePropagation(client, account.address);

    // Step 2: Create Credential Schema
    console.log("Step 2: Creating Credential Schema...");

    const schemaJson = JSON.stringify({
        $id: `vpr:verana:VPR_CHAIN_ID/cs/v1/js/VPR_CREDENTIAL_SCHEMA_ID`,
        $schema: "https://json-schema.org/draft/2020-12/schema",
        title: "PermSessionTestCredential",
        description: "Test credential for permission session",
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

    const createCsMsg = {
        typeUrl: typeUrls.MsgCreateCredentialSchema,
        value: MsgCreateCredentialSchema.fromPartial({
            creator: account.address,
            trId: trId,
            jsonSchema: schemaJson,
            issuerGrantorValidationValidityPeriod: { value: 0 } as OptionalUInt32,
            verifierGrantorValidationValidityPeriod: { value: 0 } as OptionalUInt32,
            issuerValidationValidityPeriod: { value: 0 } as OptionalUInt32,
            verifierValidationValidityPeriod: { value: 0 } as OptionalUInt32,
            holderValidationValidityPeriod: { value: 0 } as OptionalUInt32,
            issuerPermManagementMode: CredentialSchemaPermManagementMode.OPEN,
            verifierPermManagementMode: CredentialSchemaPermManagementMode.OPEN,
        }),
    };

    const csFee = await calculateFeeWithSimulation(client, account.address, [createCsMsg], "Creating CS for perm session");
    const csResult = await signAndBroadcastWithRetry(client, account.address, [createCsMsg], csFee, "Creating CS for perm session");

    if (csResult.code !== 0) {
        console.log(`  ❌ Failed to create Credential Schema: ${csResult.rawLog}`);
        process.exit(1);
    }

    // Extract Schema ID
    let schemaId: number | undefined;
    for (const event of csResult.events || []) {
        if (event.type === "create_credential_schema" || event.type === "verana.cs.v1.EventCreateCredentialSchema") {
            for (const attr of event.attributes) {
                if (attr.key === "credential_schema_id" || attr.key === "id" || attr.key === "cs_id") {
                    schemaId = parseInt(attr.value, 10);
                    if (!isNaN(schemaId)) break;
                }
            }
            if (schemaId) break;
        }
    }

    if (!schemaId) {
        console.log("  ❌ Could not extract Schema ID from events");
        process.exit(1);
    }

    console.log(`  ✓ Credential Schema created: ID=${schemaId}`);
    console.log();

    // Wait for sequence propagation
    await waitForSequencePropagation(client, account.address);

    // Step 3: Create Root Permission (Ecosystem)
    console.log("Step 3: Creating Root Permission (Ecosystem)...");

    const effectiveFrom = new Date(Date.now() + 10000); // 10 seconds in future
    const effectiveUntil = new Date(effectiveFrom.getTime() + 360 * 24 * 60 * 60 * 1000); // 360 days

    const createRootPermMsg = {
        typeUrl: typeUrls.MsgCreateRootPermission,
        value: MsgCreateRootPermission.fromPartial({
            creator: account.address,
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

    const rootPermFee = await calculateFeeWithSimulation(client, account.address, [createRootPermMsg], "Creating root perm for perm session");
    const rootPermResult = await signAndBroadcastWithRetry(client, account.address, [createRootPermMsg], rootPermFee, "Creating root perm for perm session");

    if (rootPermResult.code !== 0) {
        console.log(`  ❌ Failed to create Root Permission: ${rootPermResult.rawLog}`);
        process.exit(1);
    }

    // Extract Root Permission ID
    let rootPermId: number | undefined;
    for (const event of rootPermResult.events || []) {
        if (event.type === "create_root_permission" || event.type === "verana.perm.v1.EventCreateRootPermission") {
            for (const attr of event.attributes) {
                if (attr.key === "root_permission_id" || attr.key === "permission_id" || attr.key === "id") {
                    rootPermId = parseInt(attr.value, 10);
                    if (!isNaN(rootPermId)) break;
                }
            }
            if (rootPermId) break;
        }
    }

    if (!rootPermId) {
        console.log("  ❌ Could not extract Root Permission ID from events");
        process.exit(1);
    }

    console.log(`  ✓ Root Permission created: ID=${rootPermId}`);
    console.log();

    // Save results for next journey step
    saveJourneyResult("perm-session-prereqs", {
        trustRegistryId: trId.toString(),
        schemaId: schemaId.toString(),
        rootPermissionId: rootPermId.toString(),
        did: did,
    });

    console.log("=".repeat(60));
    console.log("✅ SUCCESS! Prerequisites created successfully!");
    console.log("=".repeat(60));
    console.log(`  Trust Registry ID: ${trId}`);
    console.log(`  Schema ID: ${schemaId}`);
    console.log(`  Root Permission ID: ${rootPermId}`);
    console.log(`  DID: ${did}`);
    console.log();
    console.log("  💾 Results saved to journey_results/perm-session-prereqs.json");
    console.log("  ➡️  Run next step: npm run test:setup-perm-session-perms");
    console.log("=".repeat(60));
}

main().catch((error: any) => {
    console.error("\n❌ Fatal error:", error.message || error);
    if (error.cause?.code === "ECONNREFUSED" || error.message?.includes("fetch failed")) {
        console.error("\n⚠️  Connection Error: Cannot connect to the blockchain.");
        console.error(`   Make sure the Verana blockchain is running at ${config.rpcEndpoint}`);
    }
    process.exit(1);
});
