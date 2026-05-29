# Step 34: TS-Proto Gap Fill — Missing Journey Files + Sign Mode Audit

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Close all remaining TS-proto journey file gaps identified in the design spec, plus fix the 9 existing journey files that incorrectly use `createDirectAccountFromMnemonic` (DIRECT signing) instead of `createAccountFromMnemonic` (LEGACY_AMINO). Update `runAll.ts` and `package.json` to include all new journeys.

**Sign mode rule (non-negotiable):** ALL journey files MUST use `createAccountFromMnemonic` (`Secp256k1HdWallet` — LEGACY_AMINO). This matches the Keplr wallet and frontend. Files using `createDirectAccountFromMnemonic` produce transactions that will fail in production. No exceptions.

**Scope analysis — what exists vs. what is missing:**

Current journey files: `csArchiveCredentialSchema`, `csCreateCredentialSchema`, `csUpdateCredentialSchema`, `deGrantCsOperatorAuthorization`, `deGrantOperatorAuthorization`, `deGrantPermOperatorAuthorization`, `permAdjustPermission`, `permCancelPermissionVPLastRequest`, `permCreateOrUpdatePermissionSession`, `permCreatePermission`, `permCreateRootPermission`, `permRepayPermissionSlashedTrustDeposit`, `permRevokePermission`, `permSetPermissionVPToValidated`, `permSlashPermissionTrustDeposit`, `permStartPermissionVP`, `runAll.ts`, `tdReclaimTrustDepositYield`, `tdRepaySlashedTrustDeposit`, `trAddGovernanceFrameworkDocument`, `trArchiveTrustRegistry`, `trCreateTrustRegistry`, `trIncreaseActiveGovernanceFrameworkVersion`, `trUpdateTrustRegistry`

**Missing (per design spec + signing.ts cross-reference):**
- CS: `csCreateSchemaAuthorizationPolicy.ts`, `csIncreaseActiveSAPVersion.ts`, `csRevokeSchemaAuthorizationPolicy.ts`
- TD: `tdSlashTrustDeposit.ts`
- DE: `deRevokeOperatorAuthorization.ts`
- XR: `xrCreateExchangeRate.ts`, `xrUpdateExchangeRate.ts`, `xrToggleExchangeRateState.ts`
- DI: `diStoreDigest.ts`

**Note — not in scope for this step (not yet registered in `signing.ts`):**
- `tdBurnEcosystemSlashedTrustDeposit.ts` — `MsgBurnEcosystemSlashedTrustDeposit` has no entry in `signing.ts`; add when proto/amino-converter exists
- `deGrantVsOperatorAuthorization.ts`, `deRevokeVsOperatorAuthorization.ts` — no Vs-specific message types in `signing.ts`; the DE module only exports `MsgGrantOperatorAuthorization` / `MsgRevokeOperatorAuthorization`
- `deGrantExchangeRateAuthorization.ts`, `deRevokeExchangeRateAuthorization.ts` — same reason
- `permRenewPermissionVP.ts` — covered by Step 5; verify it exists

**Journey dependency order (within this step):**
1. Sign mode fix (Task 1) — no new code, only imports changed
2. `deRevokeOperatorAuthorization.ts` — depends on deGrantOperatorAuthorization output
3. CS SAP journeys — depend on csCreateCredentialSchema + TR setup
4. `tdSlashTrustDeposit.ts` — depends on TD trust deposit existing
5. XR journeys — `xrCreateExchangeRate` first (needs governance tx pattern), then `xrUpdateExchangeRate`, then `xrToggleExchangeRateState`
6. `diStoreDigest.ts` — needs PERM authz setup
7. `runAll.ts` + `package.json` update (Task 10) — after all new files exist

---

## Pre-flight

- [ ] **Worktree.** Create an isolated git worktree.
- [ ] **Branch.** Branch name: `test/step-34-ts-proto-gap-fill`.
- [ ] **Sanity-check.** Run `cd ts-proto/test && npm run build` — exit 0.

---

## File Layout

| Action | Path |
|--------|------|
| Modify (sign mode) | `ts-proto/test/src/journeys/permCancelPermissionVPLastRequest.ts` |
| Modify (sign mode) | `ts-proto/test/src/journeys/permRepayPermissionSlashedTrustDeposit.ts` |
| Modify (sign mode) | `ts-proto/test/src/journeys/permStartPermissionVP.ts` |
| Modify (sign mode) | `ts-proto/test/src/journeys/permCreatePermission.ts` |
| Modify (sign mode) | `ts-proto/test/src/journeys/permCreateOrUpdatePermissionSession.ts` |
| Modify (sign mode) | `ts-proto/test/src/journeys/permSetPermissionVPToValidated.ts` |
| Modify (sign mode) | `ts-proto/test/src/journeys/permSlashPermissionTrustDeposit.ts` |
| Modify (sign mode) | `ts-proto/test/src/journeys/tdReclaimTrustDepositYield.ts` |
| Modify (sign mode) | `ts-proto/test/src/journeys/tdRepaySlashedTrustDeposit.ts` |
| Create | `ts-proto/test/src/journeys/deRevokeOperatorAuthorization.ts` |
| Create | `ts-proto/test/src/journeys/csCreateSchemaAuthorizationPolicy.ts` |
| Create | `ts-proto/test/src/journeys/csIncreaseActiveSAPVersion.ts` |
| Create | `ts-proto/test/src/journeys/csRevokeSchemaAuthorizationPolicy.ts` |
| Create | `ts-proto/test/src/journeys/tdSlashTrustDeposit.ts` |
| Create | `ts-proto/test/src/journeys/xrCreateExchangeRate.ts` |
| Create | `ts-proto/test/src/journeys/xrUpdateExchangeRate.ts` |
| Create | `ts-proto/test/src/journeys/xrToggleExchangeRateState.ts` |
| Create | `ts-proto/test/src/journeys/diStoreDigest.ts` |
| Modify | `ts-proto/test/src/journeys/runAll.ts` |
| Modify | `ts-proto/test/package.json` |

---

## Task 1: Sign mode audit and fix (9 files)

All 9 files use `createDirectAccountFromMnemonic` from `client.ts`. Replace ONLY the import and the wallet creation call. All other code stays unchanged.

**Pattern to fix in each file:**

From:
```typescript
import {
  createDirectAccountFromMnemonic,
  ...
} from "../helpers/client";
...
const wallet = await createDirectAccountFromMnemonic(COOLUSER_MNEMONIC, OPERATOR_INDEX);
```

To:
```typescript
import {
  createAccountFromMnemonic,
  ...
} from "../helpers/client";
...
const wallet = await createAccountFromMnemonic(COOLUSER_MNEMONIC, OPERATOR_INDEX);
```

- [ ] **Step 1.1: Fix `permCancelPermissionVPLastRequest.ts`.**

  Replace `createDirectAccountFromMnemonic` with `createAccountFromMnemonic` in both the import and the wallet creation call.

- [ ] **Step 1.2: Fix `permRepayPermissionSlashedTrustDeposit.ts`.**

  Same replacement.

- [ ] **Step 1.3: Fix `permStartPermissionVP.ts`.**

  Same replacement.

- [ ] **Step 1.4: Fix `permCreatePermission.ts`.**

  Same replacement.

- [ ] **Step 1.5: Fix `permCreateOrUpdatePermissionSession.ts`.**

  Same replacement.

- [ ] **Step 1.6: Fix `permSetPermissionVPToValidated.ts`.**

  Same replacement.

- [ ] **Step 1.7: Fix `permSlashPermissionTrustDeposit.ts`.**

  Same replacement.

- [ ] **Step 1.8: Fix `tdReclaimTrustDepositYield.ts`.**

  Same replacement.

- [ ] **Step 1.9: Fix `tdRepaySlashedTrustDeposit.ts`.**

  Same replacement.

- [ ] **Step 1.10: Verify no Direct sign mode remains.**

  Run: `grep -r "createDirectAccountFromMnemonic" ts-proto/test/src/journeys/`
  Expected: no output.

- [ ] **Step 1.11: Build check.**

  Run: `cd ts-proto/test && npm run build`
  Expected: exit 0.

- [ ] **Step 1.12: Commit.**

```bash
git add ts-proto/test/src/journeys/permCancelPermissionVPLastRequest.ts \
        ts-proto/test/src/journeys/permRepayPermissionSlashedTrustDeposit.ts \
        ts-proto/test/src/journeys/permStartPermissionVP.ts \
        ts-proto/test/src/journeys/permCreatePermission.ts \
        ts-proto/test/src/journeys/permCreateOrUpdatePermissionSession.ts \
        ts-proto/test/src/journeys/permSetPermissionVPToValidated.ts \
        ts-proto/test/src/journeys/permSlashPermissionTrustDeposit.ts \
        ts-proto/test/src/journeys/tdReclaimTrustDepositYield.ts \
        ts-proto/test/src/journeys/tdRepaySlashedTrustDeposit.ts
git commit -m "fix(ts-proto): standardize all journeys to LEGACY_AMINO sign mode"
```

---

## Task 2: `deRevokeOperatorAuthorization.ts`

Revokes the TR operator authorization granted by `deGrantOperatorAuthorization.ts`. Uses the authority wallet (index 10).

**File:** `ts-proto/test/src/journeys/deRevokeOperatorAuthorization.ts`

- [ ] **Step 2.1: Create the file.**

```typescript
/**
 * Journey: DE Revoke Operator Authorization
 *
 * Revokes the operator authorization previously granted by deGrantOperatorAuthorization.
 * The authority signs MsgRevokeOperatorAuthorization directly (no operator needed).
 *
 * Requires: test:de-grant-auth must be run first.
 *
 * Usage:
 *   npm run test:de-revoke-auth
 */

import {
  createAccountFromMnemonic,
  createSigningClient,
  getAccountInfo,
  calculateFeeWithSimulation,
  signAndBroadcastWithRetry,
  config,
} from "../helpers/client";
import { typeUrls } from "../helpers/registry";
import { MsgRevokeOperatorAuthorization } from "../../../src/codec/verana/de/v1/tx";
import { getTrAuthzSetup } from "../helpers/journeyResults";

const COOLUSER_MNEMONIC =
  (process.env.MNEMONIC && process.env.MNEMONIC.trim()) ||
  "pink glory help gown abstract eight nice crazy forward ketchup skill cheese";

const AUTHORITY_INDEX = 10;
const OPERATOR_INDEX = 11;

async function main() {
  console.log("=".repeat(60));
  console.log("Journey: DE Revoke Operator Authorization");
  console.log("=".repeat(60));
  console.log();

  // Step 1: Load authz setup
  console.log("Step 1: Loading TR authz setup...");
  const setup = getTrAuthzSetup();
  if (!setup) {
    console.log("  No TR authz setup found. Run test:de-grant-auth first.");
    process.exit(1);
  }
  console.log(`  Authority: ${setup.authorityAddress}`);
  console.log(`  Operator:  ${setup.operatorAddress}`);
  console.log();

  // Step 2: Create authority wallet
  console.log("Step 2: Setting up authority wallet...");
  const authorityWallet = await createAccountFromMnemonic(COOLUSER_MNEMONIC, AUTHORITY_INDEX);
  const authorityAccount = await getAccountInfo(authorityWallet);
  console.log(`  Authority wallet: ${authorityAccount.address}`);

  if (authorityAccount.address !== setup.authorityAddress) {
    console.log("  Authority address mismatch!");
    process.exit(1);
  }

  const operatorWallet = await createAccountFromMnemonic(COOLUSER_MNEMONIC, OPERATOR_INDEX);
  const operatorAccount = await getAccountInfo(operatorWallet);

  const client = await createSigningClient(authorityWallet);
  console.log("  Connected to blockchain");
  console.log();

  // Step 3: Check authority balance
  console.log("Step 3: Checking authority balance...");
  const balance = await client.getBalance(authorityAccount.address, config.denom);
  console.log(`  Balance: ${balance.amount} ${balance.denom}`);
  if (BigInt(balance.amount) < BigInt(1000000)) {
    console.log("  Insufficient balance.");
    process.exit(1);
  }
  console.log();

  // Step 4: Revoke TR operator authorization
  console.log("Step 4: Revoking TR operator authorization...");

  const allTrMsgTypes = [
    typeUrls.MsgCreateTrustRegistry,
    typeUrls.MsgUpdateTrustRegistry,
    typeUrls.MsgArchiveTrustRegistry,
    typeUrls.MsgAddGovernanceFrameworkDocument,
    typeUrls.MsgIncreaseActiveGovernanceFrameworkVersion,
  ];

  const msg = {
    typeUrl: typeUrls.MsgRevokeOperatorAuthorization,
    value: MsgRevokeOperatorAuthorization.fromPartial({
      corporation: authorityAccount.address,
      operator: "",  // authority acts alone
      grantee: operatorAccount.address,
      msgTypes: allTrMsgTypes,
    }),
  };

  try {
    const fee = await calculateFeeWithSimulation(
      client, authorityAccount.address, [msg],
      "Revoking TR operator authorization",
    );
    console.log(`  Gas: ${fee.gas}, Fee: ${fee.amount[0].amount}${fee.amount[0].denom}`);

    const result = await signAndBroadcastWithRetry(
      client, authorityAccount.address, [msg], fee,
      "Revoking TR operator authorization",
    );

    if (result.code === 0) {
      console.log();
      console.log("SUCCESS! TR operator authorization revoked.");
      console.log(`  Tx Hash: ${result.transactionHash}`);
      console.log(`  Block:   ${result.height}`);
      console.log(`  Gas:     ${result.gasUsed}/${result.gasWanted}`);
    } else {
      console.log("FAILED!");
      console.log(`  Code: ${result.code}`);
      console.log(`  Log:  ${result.rawLog}`);
      process.exit(1);
    }
  } catch (error: any) {
    console.log("ERROR!");
    console.error(error);
    process.exit(1);
  } finally {
    client.disconnect();
  }

  console.log();
  console.log("=".repeat(60));
}

main().catch((error: any) => {
  console.error("\nFatal error:", error.message || error);
  process.exit(1);
});
```

- [ ] **Step 2.2: Build check.**

  Run: `cd ts-proto/test && npm run build`
  Expected: exit 0.

- [ ] **Step 2.3: Commit.**

```bash
git add ts-proto/test/src/journeys/deRevokeOperatorAuthorization.ts
git commit -m "feat(ts-proto): add deRevokeOperatorAuthorization journey"
```

---

## Task 3: CS SAP journeys

These three journeys operate on SchemaAuthorizationPolicy (SAP). They depend on an active CS and TR (set up by the existing CS journey chain). The CS operator authorization is via `deGrantCsOperatorAuthorization.ts` which saves to `getCsAuthzSetup()`.

### Step 3a: `csCreateSchemaAuthorizationPolicy.ts`

**File:** `ts-proto/test/src/journeys/csCreateSchemaAuthorizationPolicy.ts`

- [ ] **Step 3.1: Create the file.**

```typescript
/**
 * Journey: CS Create Schema Authorization Policy
 *
 * Creates a SchemaAuthorizationPolicy on an existing credential schema.
 * Operator signs on behalf of the authority.
 *
 * Requires: test:de-grant-cs-auth and test:cs-create must run first.
 *
 * Usage:
 *   npm run test:cs-create-sap
 */

import {
  createAccountFromMnemonic,
  createSigningClient,
  getAccountInfo,
  calculateFeeWithSimulation,
  signAndBroadcastWithRetry,
  config,
} from "../helpers/client";
import { typeUrls } from "../helpers/registry";
import { MsgCreateSchemaAuthorizationPolicy } from "../../../src/codec/verana/cs/v1/tx";
import { getCsAuthzSetup, getActiveCS, saveJourneyResult } from "../helpers/journeyResults";

const COOLUSER_MNEMONIC =
  (process.env.MNEMONIC && process.env.MNEMONIC.trim()) ||
  "pink glory help gown abstract eight nice crazy forward ketchup skill cheese";

const OPERATOR_INDEX = 13; // CS operator derivation index (matches deGrantCsOperatorAuthorization)

async function main() {
  console.log("=".repeat(60));
  console.log("Journey: CS Create Schema Authorization Policy");
  console.log("=".repeat(60));
  console.log();

  // Step 1: Load CS authz setup and active schema
  console.log("Step 1: Loading CS authz setup and active schema...");
  const setup = getCsAuthzSetup();
  if (!setup) {
    console.log("  No CS authz setup found. Run test:de-grant-cs-auth first.");
    process.exit(1);
  }
  const cs = getActiveCS();
  if (!cs) {
    console.log("  No active CS found. Run test:cs-create first.");
    process.exit(1);
  }
  console.log(`  Authority:  ${setup.authorityAddress}`);
  console.log(`  Operator:   ${setup.operatorAddress}`);
  console.log(`  Schema ID:  ${cs.schemaId}`);
  console.log(`  TR ID:      ${cs.trustRegistryId}`);
  console.log();

  // Step 2: Create operator wallet
  console.log("Step 2: Setting up operator wallet...");
  const wallet = await createAccountFromMnemonic(COOLUSER_MNEMONIC, OPERATOR_INDEX);
  const account = await getAccountInfo(wallet);
  console.log(`  Operator wallet: ${account.address}`);

  if (account.address !== setup.operatorAddress) {
    console.log("  Operator address mismatch!");
    process.exit(1);
  }

  const client = await createSigningClient(wallet);
  console.log("  Connected to blockchain");
  console.log();

  // Step 3: Check balance
  console.log("Step 3: Checking operator balance...");
  const balance = await client.getBalance(account.address, config.denom);
  console.log(`  Balance: ${balance.amount} ${balance.denom}`);
  if (BigInt(balance.amount) < BigInt(1000000)) {
    console.log("  Insufficient balance.");
    process.exit(1);
  }
  console.log();

  // Step 4: Create Schema Authorization Policy
  console.log("Step 4: Creating Schema Authorization Policy...");
  const docUrl = `https://ts-proto-test-sap-v1-${Date.now()}.example.com`;
  const docDigestSri = "sha384-MzNNbQTWCSUSi0bbz7dbua+RcENv7C6FvlmYJ1Y+I727HsPOHdzwELMYO9Mz68M26";

  const msg = {
    typeUrl: typeUrls.MsgCreateSchemaAuthorizationPolicy,
    value: MsgCreateSchemaAuthorizationPolicy.fromPartial({
      corporation: setup.authorityAddress,
      operator: account.address,
      schemaId: cs.schemaId,
      docUrl: docUrl,
      docDigestSri: docDigestSri,
    }),
  };

  console.log(`  Schema ID:  ${cs.schemaId}`);
  console.log(`  Doc URL:    ${docUrl}`);
  console.log();

  try {
    const fee = await calculateFeeWithSimulation(
      client, account.address, [msg],
      "Creating Schema Authorization Policy",
    );
    console.log(`  Gas: ${fee.gas}, Fee: ${fee.amount[0].amount}${fee.amount[0].denom}`);

    const result = await signAndBroadcastWithRetry(
      client, account.address, [msg], fee,
      "Creating Schema Authorization Policy",
    );

    if (result.code === 0) {
      console.log();
      console.log("SUCCESS! Schema Authorization Policy created.");
      console.log(`  Tx Hash: ${result.transactionHash}`);
      console.log(`  Block:   ${result.height}`);
      console.log(`  Gas:     ${result.gasUsed}/${result.gasWanted}`);

      saveJourneyResult("csCreateSchemaAuthorizationPolicy", {
        success: true,
        transactionHash: result.transactionHash,
        height: result.height,
        schemaId: cs.schemaId,
      });
      console.log("  Saved SAP creation result for subsequent journeys");
    } else {
      console.log("FAILED!");
      console.log(`  Code: ${result.code}`);
      console.log(`  Log:  ${result.rawLog}`);
      process.exit(1);
    }
  } catch (error: any) {
    console.log("ERROR!");
    console.error(error);
    process.exit(1);
  } finally {
    client.disconnect();
  }

  console.log();
  console.log("=".repeat(60));
}

main().catch((error: any) => {
  console.error("\nFatal error:", error.message || error);
  process.exit(1);
});
```

### Step 3b: `csIncreaseActiveSAPVersion.ts`

**File:** `ts-proto/test/src/journeys/csIncreaseActiveSAPVersion.ts`

- [ ] **Step 3.2: Create the file.**

```typescript
/**
 * Journey: CS Increase Active SAP Version
 *
 * Increases the active Schema Authorization Policy version for a credential schema.
 * Operator signs on behalf of the authority.
 *
 * Requires: test:cs-create-sap must run first.
 *
 * Usage:
 *   npm run test:cs-increase-sap-version
 */

import {
  createAccountFromMnemonic,
  createSigningClient,
  getAccountInfo,
  calculateFeeWithSimulation,
  signAndBroadcastWithRetry,
  config,
} from "../helpers/client";
import { typeUrls } from "../helpers/registry";
import { MsgIncreaseActiveSchemaAuthorizationPolicyVersion } from "../../../src/codec/verana/cs/v1/tx";
import { getCsAuthzSetup, getActiveCS } from "../helpers/journeyResults";

const COOLUSER_MNEMONIC =
  (process.env.MNEMONIC && process.env.MNEMONIC.trim()) ||
  "pink glory help gown abstract eight nice crazy forward ketchup skill cheese";

const OPERATOR_INDEX = 13;

async function main() {
  console.log("=".repeat(60));
  console.log("Journey: CS Increase Active SAP Version");
  console.log("=".repeat(60));
  console.log();

  // Step 1: Load CS authz setup
  console.log("Step 1: Loading CS authz setup and active schema...");
  const setup = getCsAuthzSetup();
  if (!setup) {
    console.log("  No CS authz setup found. Run test:de-grant-cs-auth first.");
    process.exit(1);
  }
  const cs = getActiveCS();
  if (!cs) {
    console.log("  No active CS found. Run test:cs-create first.");
    process.exit(1);
  }
  console.log(`  Authority:  ${setup.authorityAddress}`);
  console.log(`  Operator:   ${setup.operatorAddress}`);
  console.log(`  Schema ID:  ${cs.schemaId}`);
  console.log();

  // Step 2: Create operator wallet
  console.log("Step 2: Setting up operator wallet...");
  const wallet = await createAccountFromMnemonic(COOLUSER_MNEMONIC, OPERATOR_INDEX);
  const account = await getAccountInfo(wallet);
  console.log(`  Operator wallet: ${account.address}`);

  if (account.address !== setup.operatorAddress) {
    console.log("  Operator address mismatch!");
    process.exit(1);
  }

  const client = await createSigningClient(wallet);
  console.log("  Connected to blockchain");
  console.log();

  // Step 3: Check balance
  console.log("Step 3: Checking operator balance...");
  const balance = await client.getBalance(account.address, config.denom);
  console.log(`  Balance: ${balance.amount} ${balance.denom}`);
  if (BigInt(balance.amount) < BigInt(1000000)) {
    console.log("  Insufficient balance.");
    process.exit(1);
  }
  console.log();

  // Step 4: Increase active SAP version
  console.log("Step 4: Increasing active SAP version...");

  const msg = {
    typeUrl: typeUrls.MsgIncreaseActiveSchemaAuthorizationPolicyVersion,
    value: MsgIncreaseActiveSchemaAuthorizationPolicyVersion.fromPartial({
      corporation: setup.authorityAddress,
      operator: account.address,
      schemaId: cs.schemaId,
    }),
  };

  try {
    const fee = await calculateFeeWithSimulation(
      client, account.address, [msg],
      "Increasing active SAP version",
    );
    console.log(`  Gas: ${fee.gas}, Fee: ${fee.amount[0].amount}${fee.amount[0].denom}`);

    const result = await signAndBroadcastWithRetry(
      client, account.address, [msg], fee,
      "Increasing active SAP version",
    );

    if (result.code === 0) {
      console.log();
      console.log("SUCCESS! Active SAP version increased.");
      console.log(`  Tx Hash: ${result.transactionHash}`);
      console.log(`  Block:   ${result.height}`);
      console.log(`  Gas:     ${result.gasUsed}/${result.gasWanted}`);
    } else {
      console.log("FAILED!");
      console.log(`  Code: ${result.code}`);
      console.log(`  Log:  ${result.rawLog}`);
      process.exit(1);
    }
  } catch (error: any) {
    console.log("ERROR!");
    console.error(error);
    process.exit(1);
  } finally {
    client.disconnect();
  }

  console.log();
  console.log("=".repeat(60));
}

main().catch((error: any) => {
  console.error("\nFatal error:", error.message || error);
  process.exit(1);
});
```

### Step 3c: `csRevokeSchemaAuthorizationPolicy.ts`

**File:** `ts-proto/test/src/journeys/csRevokeSchemaAuthorizationPolicy.ts`

- [ ] **Step 3.3: Create the file.**

```typescript
/**
 * Journey: CS Revoke Schema Authorization Policy
 *
 * Revokes the Schema Authorization Policy on a credential schema.
 * Operator signs on behalf of the authority.
 *
 * Requires: test:cs-create-sap must run first.
 *
 * Usage:
 *   npm run test:cs-revoke-sap
 */

import {
  createAccountFromMnemonic,
  createSigningClient,
  getAccountInfo,
  calculateFeeWithSimulation,
  signAndBroadcastWithRetry,
  config,
} from "../helpers/client";
import { typeUrls } from "../helpers/registry";
import { MsgRevokeSchemaAuthorizationPolicy } from "../../../src/codec/verana/cs/v1/tx";
import { getCsAuthzSetup, getActiveCS } from "../helpers/journeyResults";

const COOLUSER_MNEMONIC =
  (process.env.MNEMONIC && process.env.MNEMONIC.trim()) ||
  "pink glory help gown abstract eight nice crazy forward ketchup skill cheese";

const OPERATOR_INDEX = 13;

async function main() {
  console.log("=".repeat(60));
  console.log("Journey: CS Revoke Schema Authorization Policy");
  console.log("=".repeat(60));
  console.log();

  // Step 1: Load CS authz setup
  console.log("Step 1: Loading CS authz setup and active schema...");
  const setup = getCsAuthzSetup();
  if (!setup) {
    console.log("  No CS authz setup found. Run test:de-grant-cs-auth first.");
    process.exit(1);
  }
  const cs = getActiveCS();
  if (!cs) {
    console.log("  No active CS found. Run test:cs-create first.");
    process.exit(1);
  }
  console.log(`  Authority:  ${setup.authorityAddress}`);
  console.log(`  Operator:   ${setup.operatorAddress}`);
  console.log(`  Schema ID:  ${cs.schemaId}`);
  console.log();

  // Step 2: Create operator wallet
  console.log("Step 2: Setting up operator wallet...");
  const wallet = await createAccountFromMnemonic(COOLUSER_MNEMONIC, OPERATOR_INDEX);
  const account = await getAccountInfo(wallet);

  if (account.address !== setup.operatorAddress) {
    console.log("  Operator address mismatch!");
    process.exit(1);
  }

  const client = await createSigningClient(wallet);
  console.log(`  Connected as ${account.address}`);
  console.log();

  // Step 3: Check balance
  console.log("Step 3: Checking operator balance...");
  const balance = await client.getBalance(account.address, config.denom);
  console.log(`  Balance: ${balance.amount} ${balance.denom}`);
  if (BigInt(balance.amount) < BigInt(1000000)) {
    console.log("  Insufficient balance.");
    process.exit(1);
  }
  console.log();

  // Step 4: Revoke SAP
  console.log("Step 4: Revoking Schema Authorization Policy...");

  const msg = {
    typeUrl: typeUrls.MsgRevokeSchemaAuthorizationPolicy,
    value: MsgRevokeSchemaAuthorizationPolicy.fromPartial({
      corporation: setup.authorityAddress,
      operator: account.address,
      schemaId: cs.schemaId,
    }),
  };

  try {
    const fee = await calculateFeeWithSimulation(
      client, account.address, [msg],
      "Revoking Schema Authorization Policy",
    );
    console.log(`  Gas: ${fee.gas}, Fee: ${fee.amount[0].amount}${fee.amount[0].denom}`);

    const result = await signAndBroadcastWithRetry(
      client, account.address, [msg], fee,
      "Revoking Schema Authorization Policy",
    );

    if (result.code === 0) {
      console.log();
      console.log("SUCCESS! Schema Authorization Policy revoked.");
      console.log(`  Tx Hash: ${result.transactionHash}`);
      console.log(`  Block:   ${result.height}`);
      console.log(`  Gas:     ${result.gasUsed}/${result.gasWanted}`);
    } else {
      console.log("FAILED!");
      console.log(`  Code: ${result.code}`);
      console.log(`  Log:  ${result.rawLog}`);
      process.exit(1);
    }
  } catch (error: any) {
    console.log("ERROR!");
    console.error(error);
    process.exit(1);
  } finally {
    client.disconnect();
  }

  console.log();
  console.log("=".repeat(60));
}

main().catch((error: any) => {
  console.error("\nFatal error:", error.message || error);
  process.exit(1);
});
```

- [ ] **Step 3.4: Add type URLs for CS SAP messages to `signing.ts`.**

  Check `signing.ts` for `MsgCreateSchemaAuthorizationPolicy`, `MsgIncreaseActiveSchemaAuthorizationPolicyVersion`, `MsgRevokeSchemaAuthorizationPolicy`. If they are NOT present in `veranaTypeUrls`, `veranaRegistryTypes`, and `createVeranaAminoTypes`, add them following the same pattern as the existing CS entries. Import from `./codec/verana/cs/v1/tx` and from `./amino-converter/cs`. Verify the amino converter file exports the needed converters.

  Run: `grep -n "CreateSchemaAuth\|IncreaseActiveSAP\|IncreaseActiveSchema\|RevokeSchemaAuth" ts-proto/src/signing.ts`
  If the output is empty, add the entries.

- [ ] **Step 3.5: Build check.**

  Run: `cd ts-proto/test && npm run build`
  Expected: exit 0.

- [ ] **Step 3.6: Commit.**

```bash
git add ts-proto/test/src/journeys/csCreateSchemaAuthorizationPolicy.ts \
        ts-proto/test/src/journeys/csIncreaseActiveSAPVersion.ts \
        ts-proto/test/src/journeys/csRevokeSchemaAuthorizationPolicy.ts
git commit -m "feat(ts-proto): add CS SAP journey files (create, increase, revoke)"
```

---

## Task 4: `tdSlashTrustDeposit.ts`

`MsgSlashTrustDeposit` is governance-signed (authority only, no operator). It requires an existing trust deposit entry. The `tdReclaimTrustDepositYield` journey uses PERM authz setup (index 15 operator). `MsgSlashTrustDeposit` has: `authority`, `operator`, `slashedAddress`, `amount`.

**File:** `ts-proto/test/src/journeys/tdSlashTrustDeposit.ts`

- [ ] **Step 4.1: Create the file.**

```typescript
/**
 * Journey: TD Slash Trust Deposit
 *
 * Slashes a trust deposit entry via governance. The authority signs
 * MsgSlashTrustDeposit on behalf of an operator.
 *
 * Requires: test:de-grant-perm-auth must run first (provides authority + operator).
 *
 * Usage:
 *   npm run test:td-slash
 */

import {
  createAccountFromMnemonic,
  createSigningClient,
  getAccountInfo,
  calculateFeeWithSimulation,
  signAndBroadcastWithRetry,
  config,
} from "../helpers/client";
import { typeUrls } from "../helpers/registry";
import { MsgSlashTrustDeposit } from "../../../src/codec/verana/td/v1/tx";
import { getPermAuthzSetup } from "../helpers/journeyResults";

const COOLUSER_MNEMONIC =
  (process.env.MNEMONIC && process.env.MNEMONIC.trim()) ||
  "pink glory help gown abstract eight nice crazy forward ketchup skill cheese";

const OPERATOR_INDEX = 15; // matches deGrantPermOperatorAuthorization

async function main() {
  console.log("=".repeat(60));
  console.log("Journey: TD Slash Trust Deposit");
  console.log("=".repeat(60));
  console.log();

  // Step 1: Load PERM authz setup
  console.log("Step 1: Loading PERM authz setup...");
  const setup = getPermAuthzSetup();
  if (!setup) {
    console.log("  No PERM authz setup found. Run test:de-grant-perm-auth first.");
    process.exit(1);
  }
  console.log(`  Authority: ${setup.authorityAddress}`);
  console.log(`  Operator:  ${setup.operatorAddress}`);
  console.log();

  // Step 2: Connect operator wallet
  console.log("Step 2: Setting up operator wallet...");
  const wallet = await createAccountFromMnemonic(COOLUSER_MNEMONIC, OPERATOR_INDEX);
  const account = await getAccountInfo(wallet);
  const client = await createSigningClient(wallet);
  console.log(`  Connected as ${account.address}`);
  console.log();

  // Step 3: Check balance
  console.log("Step 3: Checking operator balance...");
  const balance = await client.getBalance(account.address, config.denom);
  console.log(`  Balance: ${balance.amount} ${balance.denom}`);
  if (BigInt(balance.amount) < BigInt(1000000)) {
    console.log("  Insufficient balance.");
    process.exit(1);
  }
  console.log();

  // Step 4: Build and broadcast MsgSlashTrustDeposit
  console.log("Step 4: Broadcasting MsgSlashTrustDeposit...");

  // Slash a small amount from the authority's trust deposit.
  // The authority address holds the trust deposit entry.
  const slashAmount = "1uvna";

  const msg = {
    typeUrl: typeUrls.MsgSlashTrustDeposit,
    value: MsgSlashTrustDeposit.fromPartial({
      authority: setup.authorityAddress,
      operator: account.address,
      slashedAddress: setup.authorityAddress,
      amount: slashAmount,
    }),
  };

  console.log(`  Slashing ${slashAmount} from ${setup.authorityAddress}`);
  console.log();

  try {
    const fee = await calculateFeeWithSimulation(
      client, account.address, [msg],
      "Slashing trust deposit",
    );
    console.log(`  Gas: ${fee.gas}, Fee: ${fee.amount[0].amount}${fee.amount[0].denom}`);

    const result = await signAndBroadcastWithRetry(
      client, account.address, [msg], fee,
      "Slashing trust deposit",
    );

    if (result.code === 0) {
      console.log();
      console.log("SUCCESS! Trust deposit slashed.");
      console.log(`  Tx Hash: ${result.transactionHash}`);
      console.log(`  Block:   ${result.height}`);
      console.log(`  Gas:     ${result.gasUsed}/${result.gasWanted}`);
    } else {
      console.log("FAILED!");
      console.log(`  Code: ${result.code}`);
      console.log(`  Log:  ${result.rawLog}`);
      process.exit(1);
    }
  } catch (error: any) {
    console.log("ERROR!");
    console.error(error);
    process.exit(1);
  } finally {
    client.disconnect();
  }

  console.log();
  console.log("=".repeat(60));
}

main().catch((error: any) => {
  console.error("\nFatal error:", error.message || error);
  process.exit(1);
});
```

- [ ] **Step 4.2: Build check.**

  Run: `cd ts-proto/test && npm run build`
  Expected: exit 0.

- [ ] **Step 4.3: Commit.**

```bash
git add ts-proto/test/src/journeys/tdSlashTrustDeposit.ts
git commit -m "feat(ts-proto): add tdSlashTrustDeposit journey"
```

---

## Task 5: XR journeys

XR messages `CreateExchangeRate` and `SetExchangeRateState` are governance-signed (authority address directly, no operator). `UpdateExchangeRate` is operator-signed. For the journey tests, we use the governance module address — but since this is an E2E smoke test on a local chain that uses a group account as authority, we model the authority as a standard account (derivation index 10) that also plays the role of governance. `UpdateExchangeRate` uses the authority (index 10) as both `authority` and `operator` to keep the journey self-contained.

**Journey result helpers needed:** `saveActiveXR(id: number)` and `getActiveXR(): { id: number } | null`. Add these to `journeyResults.ts` before creating the XR journey files, OR handle in-file via a local JSON file. Use the in-file approach for simplicity (write to a temp file).

Actually, since `journeyResults.ts` is shared and we should not require changes to it, we use a `saveJourneyResult` + `loadJourneyResult` pattern instead.

### Step 5a: `xrCreateExchangeRate.ts`

**File:** `ts-proto/test/src/journeys/xrCreateExchangeRate.ts`

- [ ] **Step 5.1: Create the file.**

```typescript
/**
 * Journey: XR Create Exchange Rate
 *
 * Creates an exchange rate entry via governance authority.
 * The authority signs MsgCreateExchangeRate directly (no operator).
 *
 * Requires: no prior journey needed (governance authority signs directly).
 * Note: On local test chains, the "authority" is the governance module address.
 *       In this E2E test, we use account index 10 as the authority.
 *
 * Usage:
 *   npm run test:xr-create
 */

import {
  createAccountFromMnemonic,
  createSigningClient,
  getAccountInfo,
  calculateFeeWithSimulation,
  signAndBroadcastWithRetry,
  config,
} from "../helpers/client";
import { typeUrls } from "../helpers/registry";
import { MsgCreateExchangeRate } from "../../../src/codec/verana/xr/v1/tx";
import { PricingAssetType } from "../../../src/codec/verana/cs/v1/types";
import { saveJourneyResult, loadJourneyResult } from "../helpers/journeyResults";

const COOLUSER_MNEMONIC =
  (process.env.MNEMONIC && process.env.MNEMONIC.trim()) ||
  "pink glory help gown abstract eight nice crazy forward ketchup skill cheese";

const AUTHORITY_INDEX = 10; // acts as governance authority in local test chain

async function main() {
  console.log("=".repeat(60));
  console.log("Journey: XR Create Exchange Rate");
  console.log("=".repeat(60));
  console.log();

  // Step 1: Create authority wallet
  console.log("Step 1: Setting up authority wallet...");
  const wallet = await createAccountFromMnemonic(COOLUSER_MNEMONIC, AUTHORITY_INDEX);
  const account = await getAccountInfo(wallet);
  console.log(`  Authority wallet: ${account.address}`);

  const client = await createSigningClient(wallet);
  console.log("  Connected to blockchain");
  console.log();

  // Step 2: Check balance
  console.log("Step 2: Checking authority balance...");
  const balance = await client.getBalance(account.address, config.denom);
  console.log(`  Balance: ${balance.amount} ${balance.denom}`);
  if (BigInt(balance.amount) < BigInt(1000000)) {
    console.log("  Insufficient balance.");
    process.exit(1);
  }
  console.log();

  // Step 3: Create exchange rate
  console.log("Step 3: Creating exchange rate (uvna / USD)...");

  // 30-day validity duration expressed as google.protobuf.Duration
  const validityDurationSeconds = 30 * 24 * 60 * 60; // 30 days in seconds

  const msg = {
    typeUrl: typeUrls.MsgCreateExchangeRate,
    value: MsgCreateExchangeRate.fromPartial({
      authority: account.address,
      baseAssetType: PricingAssetType.PRICING_ASSET_TYPE_TU,
      baseAsset: "uvna",
      quoteAssetType: PricingAssetType.PRICING_ASSET_TYPE_FIAT,
      quoteAsset: "USD",
      rate: "1.5",
      rateScale: 6,
      validityDuration: { seconds: validityDurationSeconds, nanos: 0 },
      state: true,
    }),
  };

  console.log(`  Authority: ${account.address}`);
  console.log(`  Pair:      uvna / USD`);
  console.log(`  Rate:      1.5 (scale 6)`);
  console.log();

  try {
    const fee = await calculateFeeWithSimulation(
      client, account.address, [msg],
      "Creating exchange rate",
    );
    console.log(`  Gas: ${fee.gas}, Fee: ${fee.amount[0].amount}${fee.amount[0].denom}`);

    const result = await signAndBroadcastWithRetry(
      client, account.address, [msg], fee,
      "Creating exchange rate",
    );

    if (result.code === 0) {
      console.log();
      console.log("SUCCESS! Exchange rate created.");
      console.log(`  Tx Hash: ${result.transactionHash}`);
      console.log(`  Block:   ${result.height}`);
      console.log(`  Gas:     ${result.gasUsed}/${result.gasWanted}`);

      // Extract XR ID from events
      let xrId: number | undefined;
      for (const event of (result.events || [])) {
        if (event.type === "create_exchange_rate") {
          for (const attr of event.attributes) {
            if (attr.key === "id") {
              xrId = parseInt(attr.value, 10);
              if (!isNaN(xrId)) {
                console.log(`  XR ID: ${xrId}`);
              }
            }
          }
        }
      }

      saveJourneyResult("xrCreateExchangeRate", {
        success: true,
        transactionHash: result.transactionHash,
        height: result.height,
        xrId: xrId,
        authorityAddress: account.address,
      });
      console.log("  Saved XR creation result for subsequent journeys");
    } else {
      console.log("FAILED!");
      console.log(`  Code: ${result.code}`);
      console.log(`  Log:  ${result.rawLog}`);
      process.exit(1);
    }
  } catch (error: any) {
    console.log("ERROR!");
    console.error(error);
    process.exit(1);
  } finally {
    client.disconnect();
  }

  console.log();
  console.log("=".repeat(60));
}

main().catch((error: any) => {
  console.error("\nFatal error:", error.message || error);
  process.exit(1);
});
```

### Step 5b: `xrUpdateExchangeRate.ts`

**File:** `ts-proto/test/src/journeys/xrUpdateExchangeRate.ts`

- [ ] **Step 5.2: Create the file.**

```typescript
/**
 * Journey: XR Update Exchange Rate
 *
 * Updates an existing exchange rate's rate value.
 * Operator signs MsgUpdateExchangeRate on behalf of the authority.
 *
 * Requires: test:xr-create must run first.
 *
 * Usage:
 *   npm run test:xr-update
 */

import {
  createAccountFromMnemonic,
  createSigningClient,
  getAccountInfo,
  calculateFeeWithSimulation,
  signAndBroadcastWithRetry,
  config,
} from "../helpers/client";
import { typeUrls } from "../helpers/registry";
import { MsgUpdateExchangeRate } from "../../../src/codec/verana/xr/v1/tx";
import { loadJourneyResult } from "../helpers/journeyResults";

const COOLUSER_MNEMONIC =
  (process.env.MNEMONIC && process.env.MNEMONIC.trim()) ||
  "pink glory help gown abstract eight nice crazy forward ketchup skill cheese";

const AUTHORITY_INDEX = 10;

async function main() {
  console.log("=".repeat(60));
  console.log("Journey: XR Update Exchange Rate");
  console.log("=".repeat(60));
  console.log();

  // Step 1: Load XR create result
  console.log("Step 1: Loading XR creation result...");
  const xrResult = loadJourneyResult("xrCreateExchangeRate");
  if (!xrResult || !xrResult.xrId) {
    console.log("  No XR creation result found. Run test:xr-create first.");
    process.exit(1);
  }
  const xrId: number = xrResult.xrId as number;
  console.log(`  XR ID:      ${xrId}`);
  console.log(`  Authority:  ${xrResult.authorityAddress}`);
  console.log();

  // Step 2: Connect operator wallet (same as authority for this journey)
  console.log("Step 2: Setting up operator wallet...");
  const wallet = await createAccountFromMnemonic(COOLUSER_MNEMONIC, AUTHORITY_INDEX);
  const account = await getAccountInfo(wallet);
  const client = await createSigningClient(wallet);
  console.log(`  Connected as ${account.address}`);
  console.log();

  // Step 3: Check balance
  console.log("Step 3: Checking balance...");
  const balance = await client.getBalance(account.address, config.denom);
  console.log(`  Balance: ${balance.amount} ${balance.denom}`);
  if (BigInt(balance.amount) < BigInt(1000000)) {
    console.log("  Insufficient balance.");
    process.exit(1);
  }
  console.log();

  // Step 4: Update exchange rate
  console.log("Step 4: Updating exchange rate...");

  const msg = {
    typeUrl: typeUrls.MsgUpdateExchangeRate,
    value: MsgUpdateExchangeRate.fromPartial({
      authority: account.address,
      operator: account.address, // authority acts as own operator in this test
      id: xrId,
      rate: "2.0",
      rateScale: 0, // 0 means keep existing scale
    }),
  };

  console.log(`  XR ID: ${xrId}`);
  console.log(`  New rate: 2.0`);
  console.log();

  try {
    const fee = await calculateFeeWithSimulation(
      client, account.address, [msg],
      "Updating exchange rate",
    );
    console.log(`  Gas: ${fee.gas}, Fee: ${fee.amount[0].amount}${fee.amount[0].denom}`);

    const result = await signAndBroadcastWithRetry(
      client, account.address, [msg], fee,
      "Updating exchange rate",
    );

    if (result.code === 0) {
      console.log();
      console.log("SUCCESS! Exchange rate updated.");
      console.log(`  Tx Hash: ${result.transactionHash}`);
      console.log(`  Block:   ${result.height}`);
      console.log(`  Gas:     ${result.gasUsed}/${result.gasWanted}`);
    } else {
      console.log("FAILED!");
      console.log(`  Code: ${result.code}`);
      console.log(`  Log:  ${result.rawLog}`);
      process.exit(1);
    }
  } catch (error: any) {
    console.log("ERROR!");
    console.error(error);
    process.exit(1);
  } finally {
    client.disconnect();
  }

  console.log();
  console.log("=".repeat(60));
}

main().catch((error: any) => {
  console.error("\nFatal error:", error.message || error);
  process.exit(1);
});
```

### Step 5c: `xrToggleExchangeRateState.ts`

**File:** `ts-proto/test/src/journeys/xrToggleExchangeRateState.ts`

- [ ] **Step 5.3: Create the file.**

```typescript
/**
 * Journey: XR Toggle Exchange Rate State
 *
 * Toggles the state of an exchange rate (active ↔ inactive).
 * The authority signs MsgSetExchangeRateState directly.
 *
 * Requires: test:xr-create must run first.
 *
 * Usage:
 *   npm run test:xr-toggle-state
 */

import {
  createAccountFromMnemonic,
  createSigningClient,
  getAccountInfo,
  calculateFeeWithSimulation,
  signAndBroadcastWithRetry,
  config,
} from "../helpers/client";
import { typeUrls } from "../helpers/registry";
import { MsgSetExchangeRateState } from "../../../src/codec/verana/xr/v1/tx";
import { loadJourneyResult } from "../helpers/journeyResults";

const COOLUSER_MNEMONIC =
  (process.env.MNEMONIC && process.env.MNEMONIC.trim()) ||
  "pink glory help gown abstract eight nice crazy forward ketchup skill cheese";

const AUTHORITY_INDEX = 10;

async function main() {
  console.log("=".repeat(60));
  console.log("Journey: XR Toggle Exchange Rate State");
  console.log("=".repeat(60));
  console.log();

  // Step 1: Load XR create result
  console.log("Step 1: Loading XR creation result...");
  const xrResult = loadJourneyResult("xrCreateExchangeRate");
  if (!xrResult || !xrResult.xrId) {
    console.log("  No XR creation result found. Run test:xr-create first.");
    process.exit(1);
  }
  const xrId: number = xrResult.xrId as number;
  console.log(`  XR ID:     ${xrId}`);
  console.log();

  // Step 2: Connect authority wallet
  console.log("Step 2: Setting up authority wallet...");
  const wallet = await createAccountFromMnemonic(COOLUSER_MNEMONIC, AUTHORITY_INDEX);
  const account = await getAccountInfo(wallet);
  const client = await createSigningClient(wallet);
  console.log(`  Connected as ${account.address}`);
  console.log();

  // Step 3: Check balance
  console.log("Step 3: Checking balance...");
  const balance = await client.getBalance(account.address, config.denom);
  console.log(`  Balance: ${balance.amount} ${balance.denom}`);
  if (BigInt(balance.amount) < BigInt(1000000)) {
    console.log("  Insufficient balance.");
    process.exit(1);
  }
  console.log();

  // Step 4: Toggle exchange rate state
  console.log("Step 4: Toggling exchange rate state...");

  const msg = {
    typeUrl: typeUrls.MsgSetExchangeRateState,
    value: MsgSetExchangeRateState.fromPartial({
      authority: account.address,
      id: xrId,
    }),
  };

  console.log(`  XR ID: ${xrId} (will toggle active ↔ inactive)`);
  console.log();

  try {
    const fee = await calculateFeeWithSimulation(
      client, account.address, [msg],
      "Toggling exchange rate state",
    );
    console.log(`  Gas: ${fee.gas}, Fee: ${fee.amount[0].amount}${fee.amount[0].denom}`);

    const result = await signAndBroadcastWithRetry(
      client, account.address, [msg], fee,
      "Toggling exchange rate state",
    );

    if (result.code === 0) {
      console.log();
      console.log("SUCCESS! Exchange rate state toggled.");
      console.log(`  Tx Hash: ${result.transactionHash}`);
      console.log(`  Block:   ${result.height}`);
      console.log(`  Gas:     ${result.gasUsed}/${result.gasWanted}`);

      for (const event of (result.events || [])) {
        if (event.type === "set_exchange_rate_state") {
          for (const attr of event.attributes) {
            if (attr.key === "state") {
              console.log(`  New state: ${attr.value}`);
            }
          }
        }
      }
    } else {
      console.log("FAILED!");
      console.log(`  Code: ${result.code}`);
      console.log(`  Log:  ${result.rawLog}`);
      process.exit(1);
    }
  } catch (error: any) {
    console.log("ERROR!");
    console.error(error);
    process.exit(1);
  } finally {
    client.disconnect();
  }

  console.log();
  console.log("=".repeat(60));
}

main().catch((error: any) => {
  console.error("\nFatal error:", error.message || error);
  process.exit(1);
});
```

- [ ] **Step 5.4: Build check.**

  Run: `cd ts-proto/test && npm run build`
  Expected: exit 0.

- [ ] **Step 5.5: Commit.**

```bash
git add ts-proto/test/src/journeys/xrCreateExchangeRate.ts \
        ts-proto/test/src/journeys/xrUpdateExchangeRate.ts \
        ts-proto/test/src/journeys/xrToggleExchangeRateState.ts
git commit -m "feat(ts-proto): add XR journey files (create, update, toggle)"
```

---

## Task 6: `diStoreDigest.ts`

`MsgStoreDigest` is operator-signed. The DI operator authorization is granted via the PERM authz setup (same authority as perm).

**File:** `ts-proto/test/src/journeys/diStoreDigest.ts`

- [ ] **Step 6.1: Create the file.**

```typescript
/**
 * Journey: DI Store Digest
 *
 * Stores a digest on-chain. The operator signs MsgStoreDigest on behalf of
 * the authority (corporation).
 *
 * Requires: test:de-grant-perm-auth must run first (provides authority + operator).
 *
 * Usage:
 *   npm run test:di-store-digest
 */

import {
  createAccountFromMnemonic,
  createSigningClient,
  getAccountInfo,
  calculateFeeWithSimulation,
  signAndBroadcastWithRetry,
  config,
} from "../helpers/client";
import { typeUrls } from "../helpers/registry";
import { MsgStoreDigest } from "../../../src/codec/verana/di/v1/tx";
import { getPermAuthzSetup } from "../helpers/journeyResults";

const COOLUSER_MNEMONIC =
  (process.env.MNEMONIC && process.env.MNEMONIC.trim()) ||
  "pink glory help gown abstract eight nice crazy forward ketchup skill cheese";

const OPERATOR_INDEX = 15; // matches deGrantPermOperatorAuthorization

async function main() {
  console.log("=".repeat(60));
  console.log("Journey: DI Store Digest");
  console.log("=".repeat(60));
  console.log();

  // Step 1: Load PERM authz setup
  console.log("Step 1: Loading PERM authz setup...");
  const setup = getPermAuthzSetup();
  if (!setup) {
    console.log("  No PERM authz setup found. Run test:de-grant-perm-auth first.");
    process.exit(1);
  }
  console.log(`  Authority: ${setup.authorityAddress}`);
  console.log(`  Operator:  ${setup.operatorAddress}`);
  console.log();

  // Step 2: Connect operator wallet
  console.log("Step 2: Setting up operator wallet...");
  const wallet = await createAccountFromMnemonic(COOLUSER_MNEMONIC, OPERATOR_INDEX);
  const account = await getAccountInfo(wallet);
  const client = await createSigningClient(wallet);
  console.log(`  Connected as ${account.address}`);
  console.log();

  // Step 3: Check balance
  console.log("Step 3: Checking operator balance...");
  const balance = await client.getBalance(account.address, config.denom);
  console.log(`  Balance: ${balance.amount} ${balance.denom}`);
  if (BigInt(balance.amount) < BigInt(1000000)) {
    console.log("  Insufficient balance.");
    process.exit(1);
  }
  console.log();

  // Step 4: Store digest
  console.log("Step 4: Storing digest (MsgStoreDigest)...");

  // Use a unique digest to avoid duplicate errors across runs
  const digestValue = `sha256:ts-proto-test-${Date.now()}-${Math.random().toString(36).slice(2, 8)}`;

  const msg = {
    typeUrl: typeUrls.MsgStoreDigest,
    value: MsgStoreDigest.fromPartial({
      authority: setup.authorityAddress,
      operator: account.address,
      digest: digestValue,
      digestAlgorithm: "sha2-256",
    }),
  };

  console.log(`  Authority: ${setup.authorityAddress}`);
  console.log(`  Operator:  ${account.address}`);
  console.log(`  Digest:    ${digestValue}`);
  console.log();

  try {
    const fee = await calculateFeeWithSimulation(
      client, account.address, [msg],
      "Storing digest",
    );
    console.log(`  Gas: ${fee.gas}, Fee: ${fee.amount[0].amount}${fee.amount[0].denom}`);

    const result = await signAndBroadcastWithRetry(
      client, account.address, [msg], fee,
      "Storing digest",
    );

    if (result.code === 0) {
      console.log();
      console.log("SUCCESS! Digest stored.");
      console.log(`  Tx Hash: ${result.transactionHash}`);
      console.log(`  Block:   ${result.height}`);
      console.log(`  Gas:     ${result.gasUsed}/${result.gasWanted}`);
      console.log(`  Digest:  ${digestValue}`);
    } else {
      console.log("FAILED!");
      console.log(`  Code: ${result.code}`);
      console.log(`  Log:  ${result.rawLog}`);
      process.exit(1);
    }
  } catch (error: any) {
    console.log("ERROR!");
    console.error(error);
    process.exit(1);
  } finally {
    client.disconnect();
  }

  console.log();
  console.log("=".repeat(60));
}

main().catch((error: any) => {
  console.error("\nFatal error:", error.message || error);
  process.exit(1);
});
```

- [ ] **Step 6.2: Build check.**

  Run: `cd ts-proto/test && npm run build`
  Expected: exit 0.

- [ ] **Step 6.3: Commit.**

```bash
git add ts-proto/test/src/journeys/diStoreDigest.ts
git commit -m "feat(ts-proto): add diStoreDigest journey"
```

---

## Task 7: Verify `signing.ts` completeness

The CS SAP message types (`MsgCreateSchemaAuthorizationPolicy`, `MsgIncreaseActiveSchemaAuthorizationPolicyVersion`, `MsgRevokeSchemaAuthorizationPolicy`) may not be in `signing.ts`. This task ensures they are registered so the amino encoding works.

- [ ] **Step 7.1: Check current state of `signing.ts` for CS SAP entries.**

  Run: `grep -n "SchemaAuthorizationPolicy\|IncreaseSAP\|IncreaseActiveSAP" ts-proto/src/signing.ts`
  Expected: If output is empty, proceed to Step 7.2. If all 3 are present, skip to Step 7.4.

- [ ] **Step 7.2: Add imports to `ts-proto/src/signing.ts`.**

  In the CS import block (currently importing `MsgArchiveCredentialSchema`, `MsgCreateCredentialSchema`, `MsgUpdateCredentialSchema`), add:
  ```typescript
  MsgCreateSchemaAuthorizationPolicy,
  MsgIncreaseActiveSchemaAuthorizationPolicyVersion,
  MsgRevokeSchemaAuthorizationPolicy,
  ```

- [ ] **Step 7.3: Add type URLs, registry entries, and amino type entries for the 3 CS SAP messages.**

  Follow the exact same pattern as the existing CS entries. The type URLs are:
  - `"/verana.cs.v1.MsgCreateSchemaAuthorizationPolicy"`
  - `"/verana.cs.v1.MsgIncreaseActiveSchemaAuthorizationPolicyVersion"`
  - `"/verana.cs.v1.MsgRevokeSchemaAuthorizationPolicy"`

  Import the amino converters from `./amino-converter/cs` (verify they exist: `grep "SchemaAuthorizationPolicy" ts-proto/src/amino-converter/cs.ts`). If they don't exist, create stub converters following the same pattern as `MsgArchiveCredentialSchemaAminoConverter` in that file.

- [ ] **Step 7.4: Build check.**

  Run: `cd ts-proto && npm run build` (builds the `src` library)
  Then: `cd ts-proto/test && npm run build`
  Both expected: exit 0.

- [ ] **Step 7.5: Commit if changes were made.**

```bash
git add ts-proto/src/signing.ts
git commit -m "feat(ts-proto): register CS SAP message types in signing.ts"
```

---

## Task 8: `typeUrls` additions in `signing.ts` for CS SAP

The journey files reference `typeUrls.MsgCreateSchemaAuthorizationPolicy`, `typeUrls.MsgIncreaseActiveSchemaAuthorizationPolicyVersion`, `typeUrls.MsgRevokeSchemaAuthorizationPolicy`. These must be present in the `veranaTypeUrls` constant in `signing.ts`.

- [ ] **Step 8.1: Verify the type URL constants exist.**

  Run: `grep "MsgCreateSchemaAuthorizationPolicy\|MsgIncreaseActive\|MsgRevokeSchemaAuthorizationPolicy" ts-proto/src/signing.ts`
  If missing, add them to the `veranaTypeUrls` object in Task 7.3.

- [ ] **Step 8.2: Build check.**

  Run: `cd ts-proto/test && npm run build`
  Expected: exit 0.

---

## Task 9: Verify `permRenewPermissionVP.ts` exists (Step 5 gate)

- [ ] **Step 9.1: Check if permRenewPermissionVP.ts exists.**

  Run: `ls ts-proto/test/src/journeys/permRenewPermissionVP.ts 2>/dev/null && echo "EXISTS" || echo "MISSING"`

  If MISSING: create it following the same pattern as other PERM journeys (using `createAccountFromMnemonic`, `MsgRenewPermissionVP`, loading from `getPermVPSetup()`). This is out of scope for this step but must be noted.

---

## Task 10: Update `runAll.ts` and `package.json`

Add all new journeys to the test runner in dependency order.

- [ ] **Step 10.1: Update `package.json` scripts section.**

  Add these entries to the `scripts` object:

```json
"test:de-revoke-auth": "npx tsx src/journeys/deRevokeOperatorAuthorization.ts",
"test:cs-create-sap": "npx tsx src/journeys/csCreateSchemaAuthorizationPolicy.ts",
"test:cs-increase-sap-version": "npx tsx src/journeys/csIncreaseActiveSAPVersion.ts",
"test:cs-revoke-sap": "npx tsx src/journeys/csRevokeSchemaAuthorizationPolicy.ts",
"test:td-slash": "npx tsx src/journeys/tdSlashTrustDeposit.ts",
"test:xr-create": "npx tsx src/journeys/xrCreateExchangeRate.ts",
"test:xr-update": "npx tsx src/journeys/xrUpdateExchangeRate.ts",
"test:xr-toggle-state": "npx tsx src/journeys/xrToggleExchangeRateState.ts",
"test:di-store-digest": "npx tsx src/journeys/diStoreDigest.ts"
```

- [ ] **Step 10.2: Update `runAll.ts` tests array.**

  Insert the new journeys in dependency order. The full updated `tests` array (additions shown inline):

```typescript
const tests: TestConfig[] = [
  // Delegation Engine (DE) module
  { name: "DE: Grant Operator Authorization", script: "test:de-grant-auth" },
  // Trust Registry (TR) module
  { name: "TR: Create Trust Registry", script: "test:tr-create" },
  { name: "TR: Add GF Document", script: "test:tr-add-gfd" },
  { name: "TR: Increase Active GF Version", script: "test:tr-increase-gf-version" },
  { name: "TR: Update Trust Registry", script: "test:tr-update" },
  { name: "TR: Archive Trust Registry", script: "test:tr-archive" },
  // Credential Schema (CS) module
  { name: "DE: Grant CS Operator Authorization", script: "test:de-grant-cs-auth" },
  { name: "CS: Create Credential Schema", script: "test:cs-create" },
  { name: "CS: Update Credential Schema", script: "test:cs-update" },
  { name: "CS: Archive Credential Schema", script: "test:cs-archive" },
  // CS Schema Authorization Policy (SAP)
  { name: "CS: Create SAP", script: "test:cs-create-sap" },
  { name: "CS: Increase Active SAP Version", script: "test:cs-increase-sap-version" },
  { name: "CS: Revoke SAP", script: "test:cs-revoke-sap" },
  // Permission (PERM) module
  { name: "DE: Grant PERM Operator Authorization", script: "test:de-grant-perm-auth" },
  { name: "PERM: Create Root Permission", script: "test:perm-create-root" },
  { name: "PERM: Create Permission (Self-Create)", script: "test:perm-create" },
  { name: "PERM: Adjust Permission", script: "test:perm-adjust" },
  { name: "PERM: Revoke Permission", script: "test:perm-revoke" },
  { name: "PERM: Start Permission VP", script: "test:perm-start-vp" },
  { name: "PERM: Set Permission VP To Validated", script: "test:perm-validate-vp" },
  { name: "PERM: Renew + Cancel Permission VP", script: "test:perm-cancel-vp" },
  { name: "PERM: Create/Update Permission Session", script: "test:perm-csps" },
  { name: "PERM: Slash Permission Trust Deposit", script: "test:perm-slash" },
  { name: "PERM: Repay Slashed Trust Deposit", script: "test:perm-repay" },
  // Trust Deposit (TD) module
  { name: "TD: Reclaim Trust Deposit Yield", script: "test:td-reclaim-yield" },
  { name: "TD: Slash Trust Deposit", script: "test:td-slash" },
  { name: "TD: Repay Slashed Trust Deposit", script: "test:td-repay-slashed" },
  // Digest (DI) module
  { name: "DI: Store Digest", script: "test:di-store-digest" },
  // Exchange Rate (XR) module
  { name: "XR: Create Exchange Rate", script: "test:xr-create" },
  { name: "XR: Update Exchange Rate", script: "test:xr-update" },
  { name: "XR: Toggle Exchange Rate State", script: "test:xr-toggle-state" },
  // DE Revoke (cleanup — placed last so prior journeys succeed)
  { name: "DE: Revoke Operator Authorization", script: "test:de-revoke-auth" },
];
```

- [ ] **Step 10.3: Build check.**

  Run: `cd ts-proto/test && npm run build`
  Expected: exit 0.

- [ ] **Step 10.4: Commit.**

```bash
git add ts-proto/test/src/journeys/runAll.ts ts-proto/test/package.json
git commit -m "feat(ts-proto): update runAll.ts and package.json with all new journeys"
```

---

## Task 11: Final pass

- [ ] **Step 11.1: Confirm no Direct sign mode remains.**

  Run: `grep -r "createDirectAccountFromMnemonic" ts-proto/test/src/journeys/`
  Expected: no output.

- [ ] **Step 11.2: Full TypeScript build.**

  Run: `cd ts-proto && npm run build && cd test && npm run build`
  Both expected: exit 0.

- [ ] **Step 11.3: Verify all new journey files exist.**

  Run: `ls ts-proto/test/src/journeys/ | sort`
  Expected output includes all files from the "File Layout" table above.

- [ ] **Step 11.4: Push and open PR.**

```bash
git push -u origin test/step-34-ts-proto-gap-fill
gh pr create --title "feat(ts-proto): fill remaining journey gaps + fix sign mode (step 34)" --body "$(cat <<'EOF'
## Summary
- Fixed sign mode in 9 journey files: replaced createDirectAccountFromMnemonic with createAccountFromMnemonic (LEGACY_AMINO)
- New journey files: deRevokeOperatorAuthorization, csCreateSchemaAuthorizationPolicy, csIncreaseActiveSAPVersion, csRevokeSchemaAuthorizationPolicy, tdSlashTrustDeposit, xrCreateExchangeRate, xrUpdateExchangeRate, xrToggleExchangeRateState, diStoreDigest
- Updated runAll.ts with all new journeys in dependency order
- Updated package.json with npm scripts for all new journeys
- Verified signing.ts registers all required message types and amino converters

## Test plan
- [ ] npm run build passes in both ts-proto/ and ts-proto/test/
- [ ] grep createDirectAccountFromMnemonic ts-proto/test/src/journeys/ returns no output
- [ ] All 9 new journey files exist and import createAccountFromMnemonic
- [ ] runAll.ts includes all new journeys in correct dependency order
EOF
)"
```

---

## "Done" Criteria — Step 34

- [ ] 9 existing journey files updated: `createDirectAccountFromMnemonic` → `createAccountFromMnemonic`.
- [ ] 9 new journey files created: `deRevokeOperatorAuthorization`, `csCreateSchemaAuthorizationPolicy`, `csIncreaseActiveSAPVersion`, `csRevokeSchemaAuthorizationPolicy`, `tdSlashTrustDeposit`, `xrCreateExchangeRate`, `xrUpdateExchangeRate`, `xrToggleExchangeRateState`, `diStoreDigest`.
- [ ] `signing.ts` registers all CS SAP message types and amino converters.
- [ ] `runAll.ts` includes all new journeys in dependency order.
- [ ] `package.json` scripts include all new journey run targets.
- [ ] `npm run build` passes in both `ts-proto/` and `ts-proto/test/`.
- [ ] No file imports `createDirectAccountFromMnemonic`.

---

## Self-Review Notes

- **Sign mode:** The `createDirectAccountFromMnemonic` function returns `DirectSecp256k1HdWallet`, which uses proto encoding. Keplr and the Verana frontend use LEGACY_AMINO. A transaction signed with DIRECT mode will fail amino encoding and produce wrong signatures when the chain expects AMINO. This is the root of the inconsistency.
- **CS SAP message type URLs:** The `typeUrls` object in `signing.ts` must be updated before the CS SAP journey files will compile. Task 7 and 8 handle this.
- **XR governance signing:** On a live chain, `MsgCreateExchangeRate` and `MsgSetExchangeRateState` require the governance module address as `authority`. In local E2E tests, account index 10 is used as the test "governance" account, consistent with the existing DE grant journey pattern. This is intentional — the journey tests validate the TypeScript encoding, not governance policy enforcement.
- **DI authz prerequisite:** `MsgStoreDigest` requires AUTHZ from authority to operator. In the journey, we use the PERM authz setup (authority + operator at index 15) which has already been granted broad DE authorization. If `MsgStoreDigest` is not in that authorization grant, a prior `deGrantPermOperatorAuthorization` journey must add it. Check `deGrantPermOperatorAuthorization.ts` to confirm `typeUrls.MsgStoreDigest` is in its `msgTypes` list; if not, document this as a dependency gap.
- **tdSlashTrustDeposit dependencies:** `MsgSlashTrustDeposit` slashes a trust deposit entry for `slashedAddress`. On a fresh local chain, the authority's trust deposit may be zero. This journey will fail if no deposit exists. The correct prerequisite is that a permission has been created (which auto-creates a trust deposit entry). Given `permCreatePermission` runs before `tdSlashTrustDeposit` in `runAll.ts`, the deposit should exist when running the full suite.
- **`deRevokeOperatorAuthorization` placement:** Placed last in `runAll.ts` because it revokes the TR operator authorization — if placed earlier, all TR-dependent journeys would fail. Revocation is a cleanup step.
