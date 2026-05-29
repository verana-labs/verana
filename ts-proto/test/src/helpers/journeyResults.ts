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
  // Journey 1: Ecosystem and Schema
  ecosystemId?: string;
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

  // Create Permission prerequisites
  accountIndex?: string;
  accountAddress?: string;
  cooluserAddress?: string;

  // Cancel Permission VP prerequisites
  validatorPermId?: string;
  applicantDid?: string;

  // DE + EC authz setup
  authorityAddress?: string;
  operatorAddress?: string;

  // MOD-CO Corporation setup
  corporationId?: string;
  policyAddress?: string;

  // Effective-from timestamp for root permissions
  effectiveFrom?: string;
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
 * Gets the active Ecosystem and Schema from journey results
 * This is the standard EC/CS that tests should use
 * @returns The result with ecosystemId and schemaId, or null if not found
 */
export function getActiveECAndSchema(): { ecosystemId: number; schemaId: number; did: string } | null {
  const ecResult = loadJourneyResult("active-ec");
  const csResult = loadJourneyResult("active-cs");

  if (ecResult?.ecosystemId && csResult?.schemaId && (ecResult?.did || csResult?.did)) {
    return {
      ecosystemId: parseInt(ecResult.ecosystemId, 10),
      schemaId: parseInt(csResult.schemaId, 10),
      did: ecResult.did || csResult.did || "",
    };
  }

  return null;
}

/**
 * Saves the active Ecosystem (replaces any existing active EC)
 */
export function saveActiveEC(ecosystemId: number, did: string): void {
  saveJourneyResult("active-ec", {
    ecosystemId: ecosystemId.toString(),
    did: did,
  });
}

/**
 * Saves the active Credential Schema (replaces any existing active CS)
 */
export function saveActiveCS(schemaId: number, ecosystemId: number, did: string): void {
  saveJourneyResult("active-cs", {
    schemaId: schemaId.toString(),
    ecosystemId: ecosystemId.toString(),
    did: did,
  });
}

/**
 * Gets the active Ecosystem
 */
export function getActiveEC(): { ecosystemId: number; did: string } | null {
  const ecResult = loadJourneyResult("active-ec");
  if (ecResult?.ecosystemId && ecResult?.did) {
    return {
      ecosystemId: parseInt(ecResult.ecosystemId, 10),
      did: ecResult.did,
    };
  }
  return null;
}

/**
 * Gets the active Credential Schema
 */
export function getActiveCS(): { schemaId: number; ecosystemId: number; did: string } | null {
  const csResult = loadJourneyResult("active-cs");
  if (csResult?.schemaId && csResult?.ecosystemId && csResult?.did) {
    return {
      schemaId: parseInt(csResult.schemaId, 10),
      ecosystemId: parseInt(csResult.ecosystemId, 10),
      did: csResult.did,
    };
  }
  return null;
}

/**
 * Checks if an Ecosystem is archived by querying the chain
 * This is used to determine if we should reuse saved results or create new ones
 */
export async function isEcosystemArchived(
  client: any,
  ecId: number
): Promise<boolean> {
  try {
    const lcdEndpoint = process.env.VERANA_LCD_ENDPOINT || "http://localhost:1317";
    const response = await fetch(`${lcdEndpoint}/verana/ec/v1/ecosystem/${ecId}`);

    if (!response.ok) {
      // If not found, consider it as "needs creation"
      return true;
    }

    // In v4-rc2 `archived` is a bool (was nullable timestamp).
    const data = (await response.json()) as { ecosystem?: { archived?: boolean | null } };
    return data.ecosystem?.archived === true;
  } catch (error) {
    // On error, assume it might be archived (safer to create new)
    console.log(`  ⚠️  Could not check EC archive status: ${error}`);
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

/**
 * Saves the EC authz setup (authority/corporation policy_address + operator).
 *
 * In v4-rc2 the "authority" is the policy_address of a registered Corporation
 * (see AUTHZ-CHECK-5). We persist it under the legacy `authorityAddress` key
 * for compatibility with existing consumers.
 */
export function saveEcAuthzSetup(authorityAddress: string, operatorAddress: string): void {
  saveJourneyResult("ec-authz-setup", {
    authorityAddress,
    operatorAddress,
  });
}

/**
 * Gets the EC authz setup (corporation policy_address + operator address).
 */
export function getEcAuthzSetup(): { authorityAddress: string; operatorAddress: string } | null {
  const result = loadJourneyResult("ec-authz-setup");
  if (result?.authorityAddress && result?.operatorAddress) {
    return {
      authorityAddress: result.authorityAddress,
      operatorAddress: result.operatorAddress,
    };
  }
  return null;
}

/**
 * Saves the active Corporation (id + policy_address) created via MOD-CO.
 * Subsequent EC/GF/CS/PERM journeys use `policyAddress` as the signing
 * `corporation` (AUTHZ-CHECK-5).
 */
export function saveActiveCorporation(corporationId: number, policyAddress: string): void {
  saveJourneyResult("active-corporation", {
    corporationId: corporationId.toString(),
    policyAddress,
  });
}

/**
 * Gets the active Corporation (id + policy_address).
 */
export function getActiveCorporation(): { corporationId: number; policyAddress: string } | null {
  const result = loadJourneyResult("active-corporation");
  if (result?.corporationId && result?.policyAddress) {
    return {
      corporationId: parseInt(result.corporationId, 10),
      policyAddress: result.policyAddress,
    };
  }
  return null;
}

/**
 * Saves the CS authz setup (authority + operator addresses)
 */
export function saveCsAuthzSetup(authorityAddress: string, operatorAddress: string): void {
  saveJourneyResult("cs-authz-setup", {
    authorityAddress,
    operatorAddress,
  });
}

/**
 * Gets the CS authz setup (authority + operator addresses)
 */
export function getCsAuthzSetup(): { authorityAddress: string; operatorAddress: string } | null {
  const result = loadJourneyResult("cs-authz-setup");
  if (result?.authorityAddress && result?.operatorAddress) {
    return {
      authorityAddress: result.authorityAddress,
      operatorAddress: result.operatorAddress,
    };
  }
  return null;
}

/**
 * Saves the active Ecosystem created for CS journeys
 */
export function saveCsActiveEC(ecosystemId: number): void {
  saveJourneyResult("cs-active-ec", {
    ecosystemId: ecosystemId.toString(),
  });
}

/**
 * Gets the active Ecosystem created for CS journeys
 */
export function getCsActiveEC(): { ecosystemId: number } | null {
  const result = loadJourneyResult("cs-active-ec");
  if (result?.ecosystemId) {
    return {
      ecosystemId: parseInt(result.ecosystemId, 10),
    };
  }
  return null;
}

// ============================================================
// PERM Module Journey Results
// ============================================================

/**
 * Saves the PERM authz setup (authority + operator addresses)
 */
export function savePermAuthzSetup(authorityAddress: string, operatorAddress: string): void {
  saveJourneyResult("perm-authz-setup", {
    authorityAddress,
    operatorAddress,
  });
}

/**
 * Gets the PERM authz setup
 */
export function getPermAuthzSetup(): { authorityAddress: string; operatorAddress: string } | null {
  const result = loadJourneyResult("perm-authz-setup");
  if (result?.authorityAddress && result?.operatorAddress) {
    return {
      authorityAddress: result.authorityAddress,
      operatorAddress: result.operatorAddress,
    };
  }
  return null;
}

/**
 * Saves the PERM root setup (EC, CS, root permission, DID)
 */
export function savePermRootSetup(ecId: number, schemaId: number, rootPermId: number, did: string, effectiveFrom?: Date): void {
  saveJourneyResult("perm-root-setup", {
    ecosystemId: ecId.toString(),
    schemaId: schemaId.toString(),
    rootPermissionId: rootPermId.toString(),
    did,
    effectiveFrom: effectiveFrom?.toISOString(),
  });
}

/**
 * Gets the PERM root setup
 */
export function getPermRootSetup(): { ecId: number; schemaId: number; rootPermId: number; did: string; effectiveFrom?: Date } | null {
  const result = loadJourneyResult("perm-root-setup");
  if (result?.ecosystemId && result?.schemaId && result?.rootPermissionId && result?.did) {
    return {
      ecId: parseInt(result.ecosystemId, 10),
      schemaId: parseInt(result.schemaId, 10),
      rootPermId: parseInt(result.rootPermissionId, 10),
      did: result.did,
      effectiveFrom: result.effectiveFrom ? new Date(result.effectiveFrom) : undefined,
    };
  }
  return null;
}

/**
 * Saves the PERM VP setup (permission in VP lifecycle)
 */
export function savePermVPSetup(vpPermId: number, schemaId: number): void {
  saveJourneyResult("perm-vp-setup", {
    permissionId: vpPermId.toString(),
    schemaId: schemaId.toString(),
  });
}

/**
 * Gets the PERM VP setup
 */
export function getPermVPSetup(): { vpPermId: number; schemaId: number } | null {
  const result = loadJourneyResult("perm-vp-setup");
  if (result?.permissionId && result?.schemaId) {
    return {
      vpPermId: parseInt(result.permissionId, 10),
      schemaId: parseInt(result.schemaId, 10),
    };
  }
  return null;
}

/**
 * Saves the PERM slash setup
 */
export function savePermSlashSetup(slashedPermId: number, schemaId: number): void {
  saveJourneyResult("perm-slash-setup", {
    permissionId: slashedPermId.toString(),
    schemaId: schemaId.toString(),
  });
}

/**
 * Gets the PERM slash setup
 */
export function getPermSlashSetup(): { slashedPermId: number; schemaId: number } | null {
  const result = loadJourneyResult("perm-slash-setup");
  if (result?.permissionId && result?.schemaId) {
    return {
      slashedPermId: parseInt(result.permissionId, 10),
      schemaId: parseInt(result.schemaId, 10),
    };
  }
  return null;
}

