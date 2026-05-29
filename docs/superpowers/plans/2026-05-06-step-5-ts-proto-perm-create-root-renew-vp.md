# Step 5: TS-Proto Smoke Tests — CreateRootPermission Sign Mode + RenewPermissionVP Journey

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Issue #286 deliverables.
1. Confirm `permCreateRootPermission.ts` uses LEGACY_AMINO sign mode (fix `permStartPermissionVP.ts` which currently uses `createDirectAccountFromMnemonic`).
2. Create `permRenewPermissionVP.ts` — new journey file for `MsgRenewPermissionVP` using LEGACY_AMINO.
3. Add `test:perm-renew-vp` npm script.

**Architecture:** Two TS smoke test journeys. Both use `createAccountFromMnemonic` (`Secp256k1HdWallet`, LEGACY_AMINO). Each builds the message in TypeScript, signs via LEGACY_AMINO, broadcasts, and asserts `result.code === 0`. No business logic assertions.

**Rule:** ALL TS journeys MUST use `createAccountFromMnemonic` (LEGACY_AMINO). `createDirectAccountFromMnemonic` is FORBIDDEN in journey files. Keplr (the production frontend) uses LEGACY_AMINO.

**Branch name:** `test/step-5-ts-proto-perm-root-renew`

---

## Pre-flight

- [ ] **Worktree.** Create an isolated worktree. Use the `superpowers:using-git-worktrees` skill if not already in one. Do not work on `main`.
- [ ] **Branch.** Branch name: `test/step-5-ts-proto-perm-root-renew`.
- [ ] **Sanity check current tree builds.**

  ```bash
  cd ts-proto/test && npm run build
  ```
  Expected: no TypeScript errors.

---

## File Structure

- **Modify:** `ts-proto/test/src/journeys/permStartPermissionVP.ts` — replace `createDirectAccountFromMnemonic` with `createAccountFromMnemonic`
- **Verify:** `ts-proto/test/src/journeys/permCreateRootPermission.ts` — already uses `createAccountFromMnemonic` (confirmed in source, no edit needed)
- **Create:** `ts-proto/test/src/journeys/permRenewPermissionVP.ts`
- **Modify:** `ts-proto/test/package.json` — add `test:perm-renew-vp` script

---

## Task 1: Fix `permStartPermissionVP.ts` sign mode

`permStartPermissionVP.ts` currently imports and uses `createDirectAccountFromMnemonic`. This violates the LEGACY_AMINO requirement.

**File:** `ts-proto/test/src/journeys/permStartPermissionVP.ts`

- [ ] **Step 1.1: Replace the sign-mode import.**

  In `permStartPermissionVP.ts`, line 14 currently reads:
  ```ts
  import {
    createDirectAccountFromMnemonic,
  ```
  Replace `createDirectAccountFromMnemonic` with `createAccountFromMnemonic` in both the import statement and the usage on line 57:
  ```ts
  const wallet = await createAccountFromMnemonic(COOLUSER_MNEMONIC, OPERATOR_INDEX);
  ```

  Full import block after edit (replace the entire existing import from `../helpers/client`):
  ```ts
  import {
    createAccountFromMnemonic,
    createSigningClient,
    getAccountInfo,
    calculateFeeWithSimulation,
    signAndBroadcastWithRetry,
    waitForPermissionToBecomeEffective,
    createQueryClient,
    config,
  } from "../helpers/client";
  ```

- [ ] **Step 1.2: Verify no other journey files use `createDirectAccountFromMnemonic`.**

  ```bash
  grep -rn "createDirectAccountFromMnemonic" ts-proto/test/src/journeys/
  ```
  Expected: no output (zero matches). If any matches remain, replace them in the same branch.

- [ ] **Step 1.3: Verify TypeScript compiles.**

  ```bash
  cd ts-proto/test && npm run build
  ```
  Expected: no errors.

- [ ] **Step 1.4: Commit.**

  ```bash
  git add ts-proto/test/src/journeys/permStartPermissionVP.ts
  git commit -m "fix(ts-proto): use LEGACY_AMINO sign mode in permStartPermissionVP"
  ```

---

## Task 2: Create `permRenewPermissionVP.ts`

`MsgRenewPermissionVP` has three fields: `corporation`, `operator`, `id` (the permission ID that is in PENDING VP state). The journey loads from `perm-vp-setup` (written by `permStartPermissionVP.ts`).

**File to create:** `ts-proto/test/src/journeys/permRenewPermissionVP.ts`

- [ ] **Step 2.1: Create the file.**

```ts
/**
 * Journey: PERM Renew Permission VP
 *
 * Renews a Validation Process (VP) that is in PENDING state.
 * Uses the VP permission ID saved by permStartPermissionVP.
 *
 * Requires: test:perm-start-vp must be run first.
 *
 * Usage:
 *   npm run test:perm-renew-vp
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
import { MsgRenewPermissionVP } from "../../../src/codec/verana/perm/v1/tx";
import { getPermAuthzSetup, getPermVPSetup } from "../helpers/journeyResults";

const COOLUSER_MNEMONIC =
  (process.env.MNEMONIC && process.env.MNEMONIC.trim()) ||
  "pink glory help gown abstract eight nice crazy forward ketchup skill cheese";

const OPERATOR_INDEX = 15;

async function main() {
  console.log("=".repeat(60));
  console.log("Journey: PERM Renew Permission VP");
  console.log("=".repeat(60));
  console.log();

  // Step 1: Load setup
  console.log("Step 1: Loading PERM VP setup...");
  const authzSetup = getPermAuthzSetup();
  const vpSetup = getPermVPSetup();
  if (!authzSetup || !vpSetup) {
    console.log("  Missing setup. Run test:de-grant-perm-auth, test:perm-create-root, and test:perm-start-vp first.");
    process.exit(1);
  }
  console.log(`  Authority:       ${authzSetup.authorityAddress}`);
  console.log(`  Operator:        ${authzSetup.operatorAddress}`);
  console.log(`  VP Permission ID: ${vpSetup.vpPermId}`);
  console.log();

  // Step 2: Connect operator (LEGACY_AMINO)
  console.log("Step 2: Setting up operator wallet (LEGACY_AMINO)...");
  const wallet = await createAccountFromMnemonic(COOLUSER_MNEMONIC, OPERATOR_INDEX);
  const account = await getAccountInfo(wallet);
  if (account.address !== authzSetup.operatorAddress) {
    console.log(`  Operator address mismatch! wallet=${account.address} setup=${authzSetup.operatorAddress}`);
    process.exit(1);
  }
  const client = await createSigningClient(wallet);
  console.log(`  Connected as ${account.address}`);

  const balance = await client.getBalance(account.address, config.denom);
  console.log(`  Balance: ${balance.amount}${balance.denom}`);
  console.log();

  try {
    // Step 3: Build and broadcast MsgRenewPermissionVP
    console.log("Step 3: Broadcasting MsgRenewPermissionVP...");
    const msg = {
      typeUrl: typeUrls.MsgRenewPermissionVP,
      value: MsgRenewPermissionVP.fromPartial({
        corporation: authzSetup.authorityAddress,
        operator: authzSetup.operatorAddress,
        id: vpSetup.vpPermId,
      }),
    };

    const fee = await calculateFeeWithSimulation(client, account.address, [msg], "Renewing Permission VP");
    const result = await signAndBroadcastWithRetry(client, account.address, [msg], fee, "Renewing Permission VP");

    if (result.code !== 0) {
      throw new Error(`MsgRenewPermissionVP failed (code ${result.code}): ${result.rawLog}`);
    }

    console.log();
    console.log("SUCCESS! Permission VP renewed!");
    console.log(`  Tx Hash: ${result.transactionHash}`);
    console.log(`  Block:   ${result.height}`);
    console.log(`  Gas:     ${result.gasUsed}/${result.gasWanted}`);
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

- [ ] **Step 2.2: Verify TypeScript compiles.**

  ```bash
  cd ts-proto/test && npm run build
  ```
  Expected: no errors.

- [ ] **Step 2.3: Commit.**

  ```bash
  git add ts-proto/test/src/journeys/permRenewPermissionVP.ts
  git commit -m "feat(ts-proto): add permRenewPermissionVP journey (LEGACY_AMINO)"
  ```

---

## Task 3: Add `test:perm-renew-vp` script to `package.json`

**File:** `ts-proto/test/package.json`

- [ ] **Step 3.1: Add the script entry.**

  In `package.json`, locate the `"test:perm-start-vp"` line and add the new script immediately after it:
  ```json
  "test:perm-start-vp": "npx tsx src/journeys/permStartPermissionVP.ts",
  "test:perm-renew-vp": "npx tsx src/journeys/permRenewPermissionVP.ts",
  "test:perm-validate-vp": "npx tsx src/journeys/permSetPermissionVPToValidated.ts",
  ```

- [ ] **Step 3.2: Verify JSON is valid.**

  ```bash
  node -e "JSON.parse(require('fs').readFileSync('ts-proto/test/package.json', 'utf8'))"
  ```
  Expected: no output (no parse errors).

- [ ] **Step 3.3: Commit.**

  ```bash
  git add ts-proto/test/package.json
  git commit -m "chore(ts-proto): add test:perm-renew-vp npm script"
  ```

---

## Task 4: End-to-end run (requires live chain)

This task requires a running Verana devnet (`VERANA_RPC_ENDPOINT` set, funded accounts).

- [ ] **Step 4.1: Confirm `permCreateRootPermission.ts` still passes.**

  ```bash
  cd ts-proto/test && npm run test:perm-create-root
  ```
  Expected: `SUCCESS! Root permission created!`, exit 0.

- [ ] **Step 4.2: Run `permStartPermissionVP.ts` with new LEGACY_AMINO wallet.**

  ```bash
  cd ts-proto/test && npm run test:perm-start-vp
  ```
  Expected: `SUCCESS! Permission VP started!`, exit 0. A `perm-vp-setup.json` is written to the journey results directory.

- [ ] **Step 4.3: Run `permRenewPermissionVP.ts`.**

  ```bash
  cd ts-proto/test && npm run test:perm-renew-vp
  ```
  Expected: `SUCCESS! Permission VP renewed!`, `result.code === 0`, exit 0.

- [ ] **Step 4.4: Commit any fixes from the run.**

  If the live run surfaces a bug (wrong typeUrl, wrong field name, sequence issues), fix it and commit with:
  ```bash
  git commit -m "fix(ts-proto): <description of fix>"
  ```

---

## Task 5: Push branch and open PR

- [ ] **Step 5.1: Push.**

  ```bash
  git push -u origin test/step-5-ts-proto-perm-root-renew
  ```

- [ ] **Step 5.2: Open PR.**

  ```bash
  gh pr create --title "feat(ts-proto): LEGACY_AMINO fix + permRenewPermissionVP journey (issue #286 step 5)" --body "$(cat <<'EOF'
  ## Summary
  - Fix `permStartPermissionVP.ts` to use `createAccountFromMnemonic` (LEGACY_AMINO) instead of `createDirectAccountFromMnemonic`
  - Confirm `permCreateRootPermission.ts` already uses LEGACY_AMINO (no change needed)
  - Add new `permRenewPermissionVP.ts` journey for `MsgRenewPermissionVP`
  - Add `test:perm-renew-vp` npm script to `package.json`

  All journey files now consistently use `Secp256k1HdWallet` (LEGACY_AMINO), matching Keplr's sign mode.

  ## Test plan
  - [ ] `npm run build` in `ts-proto/test` passes with no TypeScript errors
  - [ ] `grep -rn "createDirectAccountFromMnemonic" ts-proto/test/src/journeys/` returns zero matches
  - [ ] `npm run test:perm-create-root` passes on devnet
  - [ ] `npm run test:perm-start-vp` passes on devnet
  - [ ] `npm run test:perm-renew-vp` passes on devnet (result.code === 0)
  EOF
  )"
  ```

---

## "Done" Criteria — Step 5

- [ ] `permCreateRootPermission.ts` uses `createAccountFromMnemonic` (LEGACY_AMINO). Confirmed, no change needed.
- [ ] `permStartPermissionVP.ts` uses `createAccountFromMnemonic` (LEGACY_AMINO), NOT `createDirectAccountFromMnemonic`.
- [ ] `grep -rn "createDirectAccountFromMnemonic" ts-proto/test/src/journeys/` returns zero matches.
- [ ] `permRenewPermissionVP.ts` exists at `ts-proto/test/src/journeys/permRenewPermissionVP.ts`.
- [ ] `permRenewPermissionVP.ts` uses `createAccountFromMnemonic` (LEGACY_AMINO).
- [ ] `permRenewPermissionVP.ts` loads `vpPermId` from `getPermVPSetup()` (written by `permStartPermissionVP.ts`).
- [ ] `permRenewPermissionVP.ts` builds `MsgRenewPermissionVP` with fields `{ corporation, operator, id: vpPermId }`.
- [ ] `permRenewPermissionVP.ts` asserts `result.code === 0`.
- [ ] `package.json` contains `"test:perm-renew-vp": "npx tsx src/journeys/permRenewPermissionVP.ts"`.
- [ ] `npm run build` passes with no TypeScript errors.
- [ ] End-to-end run on devnet succeeds for all three scripts (perm-create-root, perm-start-vp, perm-renew-vp).

---

## Self-Review Notes

- **Sign mode standardization:** `createAccountFromMnemonic` returns `Secp256k1HdWallet` from `@cosmjs/amino`, which forces LEGACY_AMINO sign mode through `createSigningClient`. The `createDirectAccountFromMnemonic` returns `DirectSecp256k1HdWallet` from `@cosmjs/proto-signing` which uses DIRECT. The fix in Task 1 is surgical — one import and one usage line.
- **`MsgRenewPermissionVP` fields:** Confirmed from `ts-proto/src/codec/verana/perm/v1/tx.ts` lines 66-74: only `corporation`, `operator`, `id` — no optional fields. The journey correctly uses all three.
- **Dependency chain:** `permRenewPermissionVP.ts` loads from `getPermVPSetup()` which reads `perm-vp-setup.json`. This file is written by `permStartPermissionVP.ts`. The journey exits early with a clear error message if the prerequisite is missing.
- **`typeUrls.MsgRenewPermissionVP`:** Confirmed present in `ts-proto/src/signing.ts` line 98, value `"/verana.perm.v1.MsgRenewPermissionVP"`.
- **No business logic assertions:** The journey only checks `result.code === 0`. It does not query state or verify fee deductions — those are the job of the Go unit test (Step 23).
- **Out of scope:** Step 23 (Go unit test for `RenewPermissionVP`) is a separate step. This step covers only the TS smoke test.
