/**
 * Journey: Query Trust Registries
 *
 * This script demonstrates how to query Trust Registries using the
 * REST API (LCD endpoint).
 *
 * Usage:
 *   npm run test:query-tr
 *
 * Or with environment variables:
 *   export VERANA_LCD_ENDPOINT="http://localhost:1317"
 *   npm run test:query-tr
 */

import { config } from "../helpers/client";
import {
  QueryListTrustRegistriesResponse,
  QueryGetTrustRegistryResponse,
} from "../../../src/codec/verana/tr/v1/query";
import { TrustRegistryWithVersions } from "../../../src/codec/verana/tr/v1/types";

/**
 * Query all trust registries via REST API
 */
async function listTrustRegistries(): Promise<TrustRegistryWithVersions[]> {
  const url = `${config.lcdEndpoint}/verana/tr/v1/trust_registries`;

  console.log(`  Fetching from: ${url}`);
  const response = await fetch(url);

  if (!response.ok) {
    throw new Error(`HTTP error! status: ${response.status}`);
  }

  const data = await response.json();
  // Use the generated type to parse the response
  const parsed = QueryListTrustRegistriesResponse.fromJSON(data);
  return parsed.trustRegistries;
}

/**
 * Query a specific trust registry by ID
 */
async function getTrustRegistry(trId: number): Promise<TrustRegistryWithVersions | undefined> {
  const url = `${config.lcdEndpoint}/verana/tr/v1/trust_registry/${trId}`;

  console.log(`  Fetching from: ${url}`);
  const response = await fetch(url);

  if (!response.ok) {
    if (response.status === 404) {
      return undefined;
    }
    throw new Error(`HTTP error! status: ${response.status}`);
  }

  const data = await response.json();
  const parsed = QueryGetTrustRegistryResponse.fromJSON(data);
  return parsed.trustRegistry;
}

/**
 * Format a trust registry for display
 */
function formatTrustRegistry(tr: TrustRegistryWithVersions): string {
  const lines = [
    `  ID: ${tr.id}`,
    `  DID: ${tr.did}`,
    `  AKA: ${tr.aka}`,
    `  Controller: ${tr.controller}`,
    `  Language: ${tr.language}`,
    `  Active Version: ${tr.activeVersion}`,
    `  Created: ${tr.created?.toISOString() || "N/A"}`,
    `  Modified: ${tr.modified?.toISOString() || "N/A"}`,
    `  Archived: ${tr.archived?.toISOString() || "Not archived"}`,
  ];

  if (tr.versions && tr.versions.length > 0) {
    lines.push(`  Versions (${tr.versions.length}):`);
    for (const version of tr.versions) {
      lines.push(`    - Version ${version.version}: ID=${version.id}, Docs=${version.documents?.length || 0}`);
    }
  }

  return lines.join("\n");
}

async function main() {
  console.log("=".repeat(60));
  console.log("Journey: Query Trust Registries (TypeScript Client)");
  console.log("=".repeat(60));
  console.log();
  console.log(`LCD Endpoint: ${config.lcdEndpoint}`);
  console.log();

  // Step 1: List all trust registries
  console.log("Step 1: Listing all Trust Registries...");
  try {
    const trustRegistries = await listTrustRegistries();

    if (trustRegistries.length === 0) {
      console.log("  No trust registries found.");
    } else {
      console.log(`  Found ${trustRegistries.length} trust registries:\n`);

      for (const tr of trustRegistries) {
        console.log("-".repeat(50));
        console.log(formatTrustRegistry(tr));
      }
    }
  } catch (error) {
    console.log("  ❌ Failed to list trust registries:");
    console.error(error);
  }

  console.log();

  // Step 2: Query specific trust registry (ID 1 if exists)
  console.log("Step 2: Querying Trust Registry with ID 1...");
  try {
    const tr = await getTrustRegistry(1);

    if (!tr) {
      console.log("  Trust Registry with ID 1 not found.");
    } else {
      console.log("  Found Trust Registry:\n");
      console.log(formatTrustRegistry(tr));
    }
  } catch (error) {
    console.log("  ❌ Failed to get trust registry:");
    console.error(error);
  }

  console.log();
  console.log("=".repeat(60));
}

main().catch((error) => {
  console.error("Fatal error:", error);
  process.exit(1);
});
