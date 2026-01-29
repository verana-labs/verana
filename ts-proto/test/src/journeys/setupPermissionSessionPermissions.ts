/**
 * Journey: Setup Permission Session Permissions
 *
 * Step 2 of 3 for Permission Session creation.
 * Loads prerequisites from Step 1, creates Issuer and Verifier permissions.
 * Saves results to journey_results/ for Step 3.
 *
 * Usage:
 *   npm run test:setup-perm-session-perms
 */

import {
    createAminoWallet,
    createSigningClient,
    getAccountInfo,
    calculateFeeWithSimulation,
    signAndBroadcastWithRetry,
    config,
    waitForSequencePropagation,
    createQueryClient,
    getBlockTime,
} from "../helpers/client";
import { typeUrls } from "../helpers/registry";
import { MsgCreatePermission } from "../../../src/codec/verana/perm/v1/tx";
import { PermissionType } from "../../../src/codec/verana/perm/v1/types";
import { loadJourneyResult, saveJourneyResult } from "../helpers/journeyResults";

const TEST_MNEMONIC =
    process.env.MNEMONIC ||
    "pink glory help gown abstract eight nice crazy forward ketchup skill cheese";

async function main() {
    console.log("=".repeat(60));
    console.log("Journey: Setup Permission Session Permissions (Step 2/3)");
    console.log("=".repeat(60));
    console.log();

    // Load prerequisites from Step 1
    console.log("Loading prerequisites from Step 1...");
    const prereqs = loadJourneyResult("perm-session-prereqs");

    if (!prereqs?.schemaId || !prereqs?.did || !prereqs?.rootPermissionId) {
        console.log("  ❌ Prerequisites not found. Run Step 1 first:");
        console.log("     npm run test:setup-perm-session-prereqs");
        process.exit(1);
    }

    const schemaId = parseInt(prereqs.schemaId, 10);
    const did = prereqs.did;
    console.log(`  ✓ Loaded prerequisites:`);
    console.log(`    - Schema ID: ${schemaId}`);
    console.log(`    - DID: ${did}`);
    console.log(`    - Root Permission ID: ${prereqs.rootPermissionId}`);
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

    // Wait for root permission to become effective (10 seconds after creation)
    console.log("Waiting for root permission to become effective...");
    const queryClient = await createQueryClient();
    try {
        const startTime = Date.now();
        const maxWait = 15000; // 15 seconds

        while (Date.now() - startTime < maxWait) {
            await new Promise((resolve) => setTimeout(resolve, 1000));
            const elapsed = Date.now() - startTime;
            if (elapsed >= 12000) {
                const currentBlockTime = await getBlockTime(queryClient);
                console.log(`  ✓ Waited ${Math.ceil(elapsed / 1000)} seconds, block time: ${currentBlockTime.toISOString()}`);
                break;
            }
        }
    } finally {
        queryClient.disconnect();
    }
    console.log();

    // Step 1: Create Issuer Permission
    console.log("Step 1: Creating Issuer Permission...");

    const effectiveFrom = new Date(Date.now() + 10000); // 10 seconds in future
    const effectiveUntil = new Date(effectiveFrom.getTime() + 360 * 24 * 60 * 60 * 1000); // 360 days

    const createIssuerMsg = {
        typeUrl: typeUrls.MsgCreatePermission,
        value: MsgCreatePermission.fromPartial({
            creator: account.address,
            schemaId: schemaId,
            type: PermissionType.ISSUER,
            did: did,
            country: "US",
            effectiveFrom: effectiveFrom,
            effectiveUntil: effectiveUntil,
            verificationFees: 1000,
            validationFees: 1000,
        }),
    };

    const issuerFee = await calculateFeeWithSimulation(client, account.address, [createIssuerMsg], "Creating issuer perm for perm session");
    const issuerResult = await signAndBroadcastWithRetry(client, account.address, [createIssuerMsg], issuerFee, "Creating issuer perm for perm session");

    if (issuerResult.code !== 0) {
        console.log(`  ❌ Failed to create Issuer Permission: ${issuerResult.rawLog}`);
        process.exit(1);
    }

    // Extract Issuer Permission ID
    let issuerPermId: number | undefined;
    for (const event of issuerResult.events || []) {
        if (event.type === "create_permission" || event.type === "verana.perm.v1.EventCreatePermission") {
            for (const attr of event.attributes) {
                if (attr.key === "permission_id" || attr.key === "id") {
                    issuerPermId = parseInt(attr.value, 10);
                    if (!isNaN(issuerPermId)) break;
                }
            }
            if (issuerPermId) break;
        }
    }

    if (!issuerPermId) {
        console.log("  ❌ Could not extract Issuer Permission ID from events");
        process.exit(1);
    }

    console.log(`  ✓ Issuer Permission created: ID=${issuerPermId}`);
    console.log();

    // Wait for sequence propagation
    await waitForSequencePropagation(client, account.address);

    // Step 2: Create Verifier Permission
    console.log("Step 2: Creating Verifier Permission...");

    // Verifier permissions cannot have verification/validation fees
    const createVerifierMsg = {
        typeUrl: typeUrls.MsgCreatePermission,
        value: MsgCreatePermission.fromPartial({
            creator: account.address,
            schemaId: schemaId,
            type: PermissionType.VERIFIER,
            did: did,
            country: "US",
            effectiveFrom: effectiveFrom,
            effectiveUntil: effectiveUntil,
            verificationFees: 0,
            validationFees: 0,
        }),
    };

    const verifierFee = await calculateFeeWithSimulation(client, account.address, [createVerifierMsg], "Creating verifier perm for perm session");
    const verifierResult = await signAndBroadcastWithRetry(client, account.address, [createVerifierMsg], verifierFee, "Creating verifier perm for perm session");

    if (verifierResult.code !== 0) {
        console.log(`  ❌ Failed to create Verifier Permission: ${verifierResult.rawLog}`);
        process.exit(1);
    }

    // Extract Verifier Permission ID
    let verifierPermId: number | undefined;
    for (const event of verifierResult.events || []) {
        if (event.type === "create_permission" || event.type === "verana.perm.v1.EventCreatePermission") {
            for (const attr of event.attributes) {
                if (attr.key === "permission_id" || attr.key === "id") {
                    verifierPermId = parseInt(attr.value, 10);
                    if (!isNaN(verifierPermId)) break;
                }
            }
            if (verifierPermId) break;
        }
    }

    if (!verifierPermId) {
        console.log("  ❌ Could not extract Verifier Permission ID from events");
        process.exit(1);
    }

    console.log(`  ✓ Verifier Permission created: ID=${verifierPermId}`);
    console.log();

    // Save results for next journey step (include issuer as agent perm)
    saveJourneyResult("perm-session-perms", {
        issuerPermId: issuerPermId.toString(),
        verifierPermId: verifierPermId.toString(),
        // Agent permission uses issuer permission (matching test harness pattern)
        agentPermId: issuerPermId.toString(),
        schemaId: schemaId.toString(),
        did: did,
    });

    console.log("=".repeat(60));
    console.log("✅ SUCCESS! Permissions created successfully!");
    console.log("=".repeat(60));
    console.log(`  Issuer Permission ID: ${issuerPermId}`);
    console.log(`  Verifier Permission ID: ${verifierPermId}`);
    console.log(`  Agent Permission ID: ${issuerPermId} (using issuer permission)`);
    console.log();
    console.log("  💾 Results saved to journey_results/perm-session-perms.json");
    console.log("  ➡️  Run next step: npm run test:create-perm-session");
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
