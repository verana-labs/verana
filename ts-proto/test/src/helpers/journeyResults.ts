/**
 * Journey Results Storage
 * Stores and retrieves journey results for reuse across tests
 * Similar to the Go test harness journey_results system
 */

import * as fs from "fs";
import * as path from "path";

const JOURNEY_RESULTS_DIR = path.join(__dirname, "../../journey_results");

/**
 * Journey result structure matching Go test harness
 */
export interface JourneyResult {
  // Journey 1: Trust Registry and Schema
  trustRegistryId?: string;
  schemaId?: string;
  rootPermissionId?: string;
  did?: string;

  // Permission IDs for permission session setup
  issuerPermId?: string;
  verifierPermId?: string;
  agentPermId?: string;
  permissionId?: string;

  // Additional fields for other journeys
  issuerGrantorDid?: string;
  issuerGrantorPermId?: string;
  issuerDid?: string;
  verifierDid?: string;
  // ... add more as needed
}

/**
 * Ensures the journey_results directory exists
 */
function ensureResultsDir(): void {
  if (!fs.existsSync(JOURNEY_RESULTS_DIR)) {
    fs.mkdirSync(JOURNEY_RESULTS_DIR, { recursive: true });
  }
}

/**
 * Saves a journey result to a JSON file
 * @param journeyName - Name of the journey (e.g., "create-tr", "create-cs")
 * @param result - The result data to save
 */
export function saveJourneyResult(journeyName: string, result: JourneyResult): void {
  ensureResultsDir();
  const filePath = path.join(JOURNEY_RESULTS_DIR, `${journeyName}.json`);
  fs.writeFileSync(filePath, JSON.stringify(result, null, 2), "utf-8");
  console.log(`  💾 Saved journey result to: ${filePath}`);
}

/**
 * Loads a journey result from a JSON file
 * @param journeyName - Name of the journey (e.g., "create-tr", "create-cs")
 * @returns The result data, or null if not found
 */
export function loadJourneyResult(journeyName: string): JourneyResult | null {
  const filePath = path.join(JOURNEY_RESULTS_DIR, `${journeyName}.json`);

  if (!fs.existsSync(filePath)) {
    return null;
  }

  try {
    const content = fs.readFileSync(filePath, "utf-8");
    return JSON.parse(content) as JourneyResult;
  } catch (error) {
    console.error(`  ⚠️  Failed to load journey result from ${filePath}:`, error);
    return null;
  }
}

/**
 * Checks if a journey result exists
 * @param journeyName - Name of the journey
 * @returns true if the result file exists
 */
export function hasJourneyResult(journeyName: string): boolean {
  const filePath = path.join(JOURNEY_RESULTS_DIR, `${journeyName}.json`);
  return fs.existsSync(filePath);
}

/**
 * Gets the active TR and Schema from journey results
 * This is the standard TR/CS that tests should use
 * @returns The result with trustRegistryId and schemaId, or null if not found
 */
export function getActiveTRAndSchema(): { trustRegistryId: number; schemaId: number; did: string } | null {
  // Load active TR and CS
  const trResult = loadJourneyResult("active-tr");
  const csResult = loadJourneyResult("active-cs");

  if (trResult?.trustRegistryId && csResult?.schemaId && (trResult?.did || csResult?.did)) {
    return {
      trustRegistryId: parseInt(trResult.trustRegistryId, 10),
      schemaId: parseInt(csResult.schemaId, 10),
      did: trResult.did || csResult.did || "",
    };
  }

  return null;
}

/**
 * Saves the active Trust Registry (replaces any existing active TR)
 */
export function saveActiveTR(trustRegistryId: number, did: string): void {
  saveJourneyResult("active-tr", {
    trustRegistryId: trustRegistryId.toString(),
    did: did,
  });
}

/**
 * Saves the active Credential Schema (replaces any existing active CS)
 */
export function saveActiveCS(schemaId: number, trustRegistryId: number, did: string): void {
  saveJourneyResult("active-cs", {
    schemaId: schemaId.toString(),
    trustRegistryId: trustRegistryId.toString(),
    did: did,
  });
}

/**
 * Gets the active Trust Registry
 */
export function getActiveTR(): { trustRegistryId: number; did: string } | null {
  const trResult = loadJourneyResult("active-tr");
  if (trResult?.trustRegistryId && trResult?.did) {
    return {
      trustRegistryId: parseInt(trResult.trustRegistryId, 10),
      did: trResult.did,
    };
  }
  return null;
}

/**
 * Gets the active Credential Schema
 */
export function getActiveCS(): { schemaId: number; trustRegistryId: number; did: string } | null {
  const csResult = loadJourneyResult("active-cs");
  if (csResult?.schemaId && csResult?.trustRegistryId && csResult?.did) {
    return {
      schemaId: parseInt(csResult.schemaId, 10),
      trustRegistryId: parseInt(csResult.trustRegistryId, 10),
      did: csResult.did,
    };
  }
  return null;
}

/**
 * Checks if a Trust Registry or Credential Schema is archived by querying the chain
 * This is used to determine if we should reuse saved results or create new ones
 */
export async function isTrustRegistryArchived(
  client: any,
  trId: number
): Promise<boolean> {
  try {
    // Query TR via LCD endpoint
    const lcdEndpoint = process.env.VERANA_LCD_ENDPOINT || "http://localhost:1317";
    const response = await fetch(`${lcdEndpoint}/verana/tr/v1/trust_registry/${trId}`);

    if (!response.ok) {
      // If not found, consider it as "needs creation"
      return true;
    }

    const data = await response.json() as { trust_registry?: { archived?: string | null } };
    // Check if archived field exists and is not null
    return data.trust_registry?.archived != null;
  } catch (error) {
    // On error, assume it might be archived (safer to create new)
    console.log(`  ⚠️  Could not check TR archive status: ${error}`);
    return true;
  }
}

/**
 * Checks if a Credential Schema is archived by querying the chain
 */
export async function isCredentialSchemaArchived(
  client: any,
  schemaId: number
): Promise<boolean> {
  try {
    // Query CS via LCD endpoint
    const lcdEndpoint = process.env.VERANA_LCD_ENDPOINT || "http://localhost:1317";
    const response = await fetch(`${lcdEndpoint}/verana/cs/v1/credential_schema/${schemaId}`);

    if (!response.ok) {
      // If not found, consider it as "needs creation"
      return true;
    }

    const data = await response.json() as { credential_schema?: { archived?: string | null } };
    // Check if archived field exists and is not null
    return data.credential_schema?.archived != null;
  } catch (error) {
    // On error, assume it might be archived (safer to create new)
    console.log(`  ⚠️  Could not check CS archive status: ${error}`);
    return true;
  }
}

/**
 * Saves the root permission ID for reuse in other journeys
 */
export function saveRootPermissionId(permissionId: number): void {
  saveJourneyResult("root-permission", {
    rootPermissionId: permissionId.toString(),
  });
}

/**
 * Gets the root permission ID from journey results
 */
export function getRootPermissionId(): number | null {
  const result = loadJourneyResult("root-permission");
  if (result?.rootPermissionId) {
    return parseInt(result.rootPermissionId, 10);
  }
  return null;
}

/**
 * Saves a permission ID for a specific journey
 */
export function savePermissionId(permissionId: number, journeyName: string): void {
  saveJourneyResult(`permission-${journeyName}`, {
    permissionId: permissionId.toString(),
  });
}

/**
 * Gets a permission ID for a specific journey
 */
export function getPermissionId(journeyName: string): number | null {
  const result = loadJourneyResult(`permission-${journeyName}`);
  if (result?.permissionId) {
    return parseInt(result.permissionId, 10);
  }
  return null;
}

