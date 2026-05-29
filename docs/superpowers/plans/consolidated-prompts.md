# Consolidated Session-Starter Prompts — Comprehensive Test Overhaul

One copy-paste prompt per step. Start a **new Claude Code session** for each step. Each prompt is self-contained.

---

## Step 0 — StatefulBankMock Infrastructure

**Plan:** `docs/superpowers/plans/2026-05-06-step-0-stateful-bank-mock.md`

```
I'm working on issue #292 — comprehensive behavioral test overhaul for the Verana blockchain.

Step 0 is a hard gate: build the StatefulBankMock infrastructure that all subsequent steps depend on.

Read the plan at docs/superpowers/plans/2026-05-06-step-0-stateful-bank-mock.md and execute it task-by-task. Use the superpowers:executing-plans skill to track progress.

Key deliverable: testutil/keeper/bank.go with StatefulBankMock that enforces deductions, records call history, and maps module names to module account addresses.

Branch: test/step-0-stateful-bank-mock
```

---

## Step 1 — TD: ReclaimTrustDepositYield

**Plan:** `docs/superpowers/plans/2026-05-06-step-1-td-reclaim-trust-deposit-yield.md`

```
I'm working on issue #292 — comprehensive behavioral test overhaul (step 1 of 34).

Gate check: step 0 must be merged (testutil/keeper/bank.go must exist).

Read the plan at docs/superpowers/plans/2026-05-06-step-1-td-reclaim-trust-deposit-yield.md and execute it task-by-task. Use the superpowers:executing-plans skill.

Key deliverables:
- TrustdepositKeeperWithStatefulBank added to testutil/keeper/trustdeposit.go
- x/td/keeper/fixture_test.go created (shared Fixture for ALL TD steps)
- x/td/keeper/reclaim_trust_deposit_yield_test.go with fixture-based tests
- Old TestMsgReclaimTrustDepositYield deleted from msg_server_test.go

Branch: test/step-1-td-reclaim-yield
```

---

## Step 2 — TD: SlashTrustDeposit

**Plan:** `docs/superpowers/plans/2026-05-06-step-2-td-slash-trust-deposit.md`

```
I'm working on issue #292 — comprehensive behavioral test overhaul (step 2 of 34).

Gate check: step 1 must be merged. x/td/keeper/fixture_test.go must exist.

Read the plan at docs/superpowers/plans/2026-05-06-step-2-td-slash-trust-deposit.md and execute it task-by-task. Use the superpowers:executing-plans skill.

Key deliverable: x/td/keeper/slash_trust_deposit_test.go with fixture-based tests covering all preconditions. SlashTrustDeposit has NO bank transfer — deposit stays in module. Old test deleted in same PR.

Branch: test/step-2-td-slash-trust-deposit
```

---

## Step 3 — TD: RepaySlashedTrustDeposit

**Plan:** `docs/superpowers/plans/2026-05-06-step-3-td-repay-slashed-trust-deposit.md`

```
I'm working on issue #292 — comprehensive behavioral test overhaul (step 3 of 34).

Gate check: step 2 must be merged.

Read the plan at docs/superpowers/plans/2026-05-06-step-3-td-repay-slashed-trust-deposit.md and execute it task-by-task. Use the superpowers:executing-plans skill.

Key deliverable: x/td/keeper/repay_slashed_trust_deposit_test.go. RepaySlashedTrustDeposit transfers FROM corporation TO module AND burns same amount. Balance assertions must verify both directions. Old test deleted in same PR.

Branch: test/step-3-td-repay-slashed-trust-deposit
```

---

## Step 4 — TD: BurnEcosystemSlashedTrustDeposit

**Plan:** `docs/superpowers/plans/2026-05-06-step-4-td-burn-ecosystem-slashed-trust-deposit.md`

```
I'm working on issue #292 — comprehensive behavioral test overhaul (step 4 of 34).

Gate check: step 3 must be merged.

Read the plan at docs/superpowers/plans/2026-05-06-step-4-td-burn-ecosystem-slashed-trust-deposit.md and execute it task-by-task. Use the superpowers:executing-plans skill.

Key deliverable: x/td/keeper/burn_ecosystem_slashed_trust_deposit_test.go. This is a Keeper method (not MsgServer). Old test deleted in same PR. Coverage gate: x/td/keeper >= 95%.

Branch: test/step-4-td-burn-ecosystem-slashed
```

---

## Step 5 — TS-Proto: permCreateRootPermission sign mode + permRenewPermissionVP

**Plan:** `docs/superpowers/plans/2026-05-06-step-5-ts-proto-perm-create-root-renew-vp.md`

```
I'm working on issue #292 / issue #286 — TS-proto smoke tests (step 5 of 34).

Read the plan at docs/superpowers/plans/2026-05-06-step-5-ts-proto-perm-create-root-renew-vp.md and execute it task-by-task. Use the superpowers:executing-plans skill.

CRITICAL: ALL TS journey files MUST use createAccountFromMnemonic (Secp256k1HdWallet — LEGACY_AMINO). Never use createDirectAccountFromMnemonic.

Key deliverables:
- Confirm permCreateRootPermission.ts uses LEGACY_AMINO (fix if not)
- Create ts-proto/test/src/journeys/permRenewPermissionVP.ts
- Add permRenewPermissionVP to runAll.ts

Branch: test/step-5-ts-perm-root-renew
```

---

## Step 6 — TR: CreateTrustRegistry

**Plan:** `docs/superpowers/plans/2026-05-06-step-6-tr-create-trust-registry.md`

```
I'm working on issue #292 — comprehensive behavioral test overhaul (step 6 of 34).

Gate check: step 0 must be merged.

Read the plan at docs/superpowers/plans/2026-05-06-step-6-tr-create-trust-registry.md and execute it task-by-task. Use the superpowers:executing-plans skill.

Key deliverables:
- TrustregistryKeeperWithDelegation added to testutil/keeper/trustregistry.go
- x/tr/keeper/fixture_test.go created (shared Fixture for ALL TR steps)
- x/tr/keeper/create_trust_registry_test.go
- Old test deleted in same PR

Branch: test/step-6-tr-create-trust-registry
```

---

## Step 7 — TR: AddGovernanceFrameworkDocument

**Plan:** `docs/superpowers/plans/2026-05-06-step-7-tr-add-governance-framework-document.md`

```
I'm working on issue #292 — comprehensive behavioral test overhaul (step 7 of 34).

Gate check: step 6 must be merged. x/tr/keeper/fixture_test.go must exist.

Read the plan at docs/superpowers/plans/2026-05-06-step-7-tr-add-governance-framework-document.md and execute it task-by-task.

Branch: test/step-7-tr-add-gf-document
```

---

## Step 8 — TR: IncreaseActiveGovernanceFrameworkVersion

**Plan:** `docs/superpowers/plans/2026-05-06-step-8-tr-increase-active-gf-version.md`

```
I'm working on issue #292 — comprehensive behavioral test overhaul (step 8 of 34).

Gate check: step 7 must be merged.

Read the plan at docs/superpowers/plans/2026-05-06-step-8-tr-increase-active-gf-version.md and execute it task-by-task.

Branch: test/step-8-tr-increase-gf-version
```

---

## Step 9 — TR: UpdateTrustRegistry

**Plan:** `docs/superpowers/plans/2026-05-06-step-9-tr-update-trust-registry.md`

```
I'm working on issue #292 — comprehensive behavioral test overhaul (step 9 of 34).

Gate check: step 8 must be merged.

Read the plan at docs/superpowers/plans/2026-05-06-step-9-tr-update-trust-registry.md and execute it task-by-task.

Branch: test/step-9-tr-update-trust-registry
```

---

## Step 10 — TR: ArchiveTrustRegistry

**Plan:** `docs/superpowers/plans/2026-05-06-step-10-tr-archive-trust-registry.md`

```
I'm working on issue #292 — comprehensive behavioral test overhaul (step 10 of 34).

Gate check: step 9 must be merged.

Read the plan at docs/superpowers/plans/2026-05-06-step-10-tr-archive-trust-registry.md and execute it task-by-task.

Coverage gate when step 10 merges: x/tr/keeper >= 95%. Run go test ./x/tr/keeper/ -cover -count=1 and include coverage delta in PR description.

Branch: test/step-10-tr-archive-trust-registry
```

---

## Step 11 — CS: CreateCredentialSchema

**Plan:** `docs/superpowers/plans/2026-05-06-step-11-cs-create-credential-schema.md`

```
I'm working on issue #292 — comprehensive behavioral test overhaul (step 11 of 34).

Gate check: step 0 must be merged.

Read the plan at docs/superpowers/plans/2026-05-06-step-11-cs-create-credential-schema.md and execute it task-by-task.

Key deliverables:
- CredentialschemaKeeperWithDelegation added to testutil/keeper/credentialschema.go
- x/cs/keeper/fixture_test.go created (shared Fixture for ALL CS steps)
- x/cs/keeper/create_credential_schema_test.go

Branch: test/step-11-cs-create-credential-schema
```

---

## Step 12 — CS: UpdateCredentialSchema

**Plan:** `docs/superpowers/plans/2026-05-06-step-12-cs-update-credential-schema.md`

```
I'm working on issue #292 — comprehensive behavioral test overhaul (step 12 of 34).

Gate check: step 11 must be merged.

Read the plan at docs/superpowers/plans/2026-05-06-step-12-cs-update-credential-schema.md and execute it task-by-task.

Branch: test/step-12-cs-update-credential-schema
```

---

## Step 13 — CS: ArchiveCredentialSchema

**Plan:** `docs/superpowers/plans/2026-05-06-step-13-cs-archive-credential-schema.md`

```
I'm working on issue #292 — comprehensive behavioral test overhaul (step 13 of 34).

Gate check: step 12 must be merged.

Read the plan at docs/superpowers/plans/2026-05-06-step-13-cs-archive-credential-schema.md and execute it task-by-task.

Branch: test/step-13-cs-archive-credential-schema
```

---

## Step 14 — CS: CreateSchemaAuthorizationPolicy

**Plan:** `docs/superpowers/plans/2026-05-06-step-14-cs-create-schema-authorization-policy.md`

```
I'm working on issue #292 — comprehensive behavioral test overhaul (step 14 of 34).

Gate check: step 13 must be merged.

Read the plan at docs/superpowers/plans/2026-05-06-step-14-cs-create-schema-authorization-policy.md and execute it task-by-task.

Branch: test/step-14-cs-create-sap
```

---

## Step 15 — CS: IncreaseActiveSAPVersion

**Plan:** `docs/superpowers/plans/2026-05-06-step-15-cs-increase-active-sap-version.md`

```
I'm working on issue #292 — comprehensive behavioral test overhaul (step 15 of 34).

Gate check: step 14 must be merged.

Read the plan at docs/superpowers/plans/2026-05-06-step-15-cs-increase-active-sap-version.md and execute it task-by-task.

Branch: test/step-15-cs-increase-active-sap-version
```

---

## Step 16 — CS: RevokeSchemaAuthorizationPolicy

**Plan:** `docs/superpowers/plans/2026-05-06-step-16-cs-revoke-schema-authorization-policy.md`

```
I'm working on issue #292 — comprehensive behavioral test overhaul (step 16 of 34).

Gate check: step 15 must be merged.

Read the plan at docs/superpowers/plans/2026-05-06-step-16-cs-revoke-schema-authorization-policy.md and execute it task-by-task.

Coverage gate when step 16 merges: x/cs/keeper >= 95%. Include coverage delta in PR description.

Branch: test/step-16-cs-revoke-sap
```

---

## Step 17 — DE: GrantOperatorAuthorization

**Plan:** `docs/superpowers/plans/2026-05-06-step-17-de-grant-operator-authorization.md`

```
I'm working on issue #292 — comprehensive behavioral test overhaul (step 17 of 34).

Gate check: step 0 must be merged.

Read the plan at docs/superpowers/plans/2026-05-06-step-17-de-grant-operator-authorization.md and execute it task-by-task.

Key deliverables:
- x/de/keeper/fixture_test.go created (shared Fixture for ALL DE steps)
- x/de/keeper/grant_operator_authorization_test.go

Branch: test/step-17-de-grant-operator-auth
```

---

## Step 18 — DE: RevokeOperatorAuthorization

**Plan:** `docs/superpowers/plans/2026-05-06-step-18-de-revoke-operator-authorization.md`

```
I'm working on issue #292 — comprehensive behavioral test overhaul (step 18 of 34).

Gate check: step 17 must be merged.

Read the plan at docs/superpowers/plans/2026-05-06-step-18-de-revoke-operator-authorization.md and execute it task-by-task.

Coverage gate when step 18 merges: x/de/keeper >= 95%. Include coverage delta in PR description.

Branch: test/step-18-de-revoke-operator-auth
```

---

## Steps 19 & 20 — DE: GrantVsOperatorAuthorization / RevokeVsOperatorAuthorization

**Plan:** `docs/superpowers/plans/2026-05-06-step-19-de-grant-vs-operator-authorization.md`

```
I'm working on issue #292 — comprehensive behavioral test overhaul (steps 19-20).

Steps 19 and 20 have no standalone implementation. Read docs/superpowers/plans/2026-05-06-step-19-de-grant-vs-operator-authorization.md and docs/superpowers/plans/2026-05-06-step-20-de-revoke-vs-operator-authorization.md for explanation.

VS operator authorization is orchestrated from PERM and covered by steps 26, 27, 29, 31. No code changes needed for these steps. Simply verify the DE proto does not define these messages:

grep -r "GrantVsOperator\|RevokeVsOperator" proto/verana/de/

Expected: no output. If proto has been updated since this plan was written, implement the steps.
```

---

## Step 21 — PERM: CreateRootPermission

**Plan:** `docs/superpowers/plans/2026-05-06-step-21-perm-create-root-permission.md`

```
I'm working on issue #292 — comprehensive behavioral test overhaul (step 21 of 34).

Gate checks:
- Step 0 must be merged (testutil/keeper/bank.go exists)
- Steps 1-4 (TD) must be merged so TD keeper is available for PERM wiring

Read the plan at docs/superpowers/plans/2026-05-06-step-21-perm-create-root-permission.md and execute it task-by-task. Use the superpowers:executing-plans skill.

Key deliverables (this is the first PERM step):
- PermissionKeeperWithStatefulBank added to testutil/keeper/permission.go
- x/perm/keeper/fixture_test.go created (shared Fixture for ALL PERM steps 21-31)
- x/perm/keeper/create_root_permission_test.go
- Old TestMsgCreateRootPermission deleted from msg_server_test.go

CreateRootPermission has NO bank transfers — no fees for root permissions.
The fixture must include RequirePermission(id, want types.Permission) for full struct comparison.

Branch: test/step-21-perm-create-root-permission
```

---

## Step 22 — PERM: StartPermissionVP

**Plan:** `docs/superpowers/plans/2026-05-06-step-22-perm-start-permission-vp.md`

```
I'm working on issue #292 — comprehensive behavioral test overhaul (step 22 of 34).

Gate check: step 21 must be merged. x/perm/keeper/fixture_test.go must exist.

Read the plan at docs/superpowers/plans/2026-05-06-step-22-perm-start-permission-vp.md and execute it task-by-task. Use the superpowers:executing-plans skill.

StartPermissionVP fee formula (CRITICAL — must be computed independently of implementation):
  fees = validatorPerm.ValidationFees * trustUnitPrice
  deposit = fees * trustDepositRate (LegacyDec truncated)

Bank assertion: corporation balance decremented by fees; module account incremented by fees.
These deltas must be independently computed via spec formula functions — not derived from the implementation.

Branch: test/step-22-perm-start-permission-vp
```

---

## Step 23 — PERM: RenewPermissionVP

**Plan:** `docs/superpowers/plans/2026-05-06-step-23-perm-renew-permission-vp.md`

```
I'm working on issue #292 — comprehensive behavioral test overhaul (step 23 of 34).

Gate check: step 22 must be merged.

Read the plan at docs/superpowers/plans/2026-05-06-step-23-perm-renew-permission-vp.md and execute it task-by-task. Use the superpowers:executing-plans skill.

RenewPermissionVP requires:
- applicant perm exists AND vp_state == VALIDATED AND perm is active
- Same fee formula as StartPermissionVP
- Bank: corporation pays fees to module escrow

Branch: test/step-23-perm-renew-permission-vp
```

---

## Step 24 — PERM: SetPermissionVPToValidated

**Plan:** `docs/superpowers/plans/2026-05-06-step-24-perm-set-permission-vp-validated.md`

```
I'm working on issue #292 — comprehensive behavioral test overhaul (step 24 of 34).

Gate check: step 23 must be merged.

Read the plan at docs/superpowers/plans/2026-05-06-step-24-perm-set-permission-vp-validated.md and execute it task-by-task. Use the superpowers:executing-plans skill.

SetPermissionVPToValidated is complex:
- Called by the VALIDATOR (not applicant) to approve a pending VP
- Bank: sends VpCurrentFees FROM module TO validator corporation
- Sets EffectiveFrom, EffectiveUntil, VpExp, IssuanceFeeDiscount, VerificationFeeDiscount
- Read x/perm/keeper/perm_validated.go before writing tests

Branch: test/step-24-perm-set-vp-validated
```

---

## Step 25 — PERM: CancelPermissionVPLastRequest

**Plan:** `docs/superpowers/plans/2026-05-06-step-25-perm-cancel-permission-vp-last-request.md`

```
I'm working on issue #292 — comprehensive behavioral test overhaul (step 25 of 34).

Gate check: step 24 must be merged.

Read the plan at docs/superpowers/plans/2026-05-06-step-25-perm-cancel-permission-vp-last-request.md and execute it task-by-task. Use the superpowers:executing-plans skill.

CancelPermissionVPLastRequest:
- Refunds VpCurrentFees from module BACK to corporation
- If vp_exp is nil: VpState=TERMINATED, else VpState=VALIDATED
- Bank: module decremented by fees, corporation incremented by fees

Branch: test/step-25-perm-cancel-vp-last-request
```

---

## Step 26 — PERM: AdjustPermission

**Plan:** `docs/superpowers/plans/2026-05-06-step-26-perm-adjust-permission.md`

```
I'm working on issue #292 — comprehensive behavioral test overhaul (step 26 of 34).

Gate check: step 25 must be merged.

Read the plan at docs/superpowers/plans/2026-05-06-step-26-perm-adjust-permission.md and execute it task-by-task. Use the superpowers:executing-plans skill.

AdjustPermission has NO bank transfers. Tests must call RequireNoBalanceChange.
Three authority cases: ECOSYSTEM perm (self-adjust), self-created perm (self-adjust), VP-managed perm (validator adjusts).

Branch: test/step-26-perm-adjust-permission
```

---

## Step 27 — PERM: RevokePermission

**Plan:** `docs/superpowers/plans/2026-05-06-step-27-perm-revoke-permission.md`

```
I'm working on issue #292 — comprehensive behavioral test overhaul (step 27 of 34).

Gate check: step 26 must be merged.

Read the plan at docs/superpowers/plans/2026-05-06-step-27-perm-revoke-permission.md and execute it task-by-task. Use the superpowers:executing-plans skill.

RevokePermission has NO bank transfers directly (deposit released via AdjustTrustDeposit mock).
Three authority options: validator ancestor, TR controller, or perm corporation.
Test all three paths in happy path variants.

Branch: test/step-27-perm-revoke-permission
```

---

## Step 28 — PERM: CreateOrUpdatePermissionSession

**Plan:** `docs/superpowers/plans/2026-05-06-step-28-perm-create-or-update-permission-session.md`

```
I'm working on issue #292 — comprehensive behavioral test overhaul (step 28 of 34).

Gate check: step 27 must be merged.

Read the plan at docs/superpowers/plans/2026-05-06-step-28-perm-create-or-update-permission-session.md and execute it task-by-task. Use the superpowers:executing-plans skill.

CreateOrUpdatePermissionSession is the most complex PERM message — it distributes fees across multiple participants (issuer, verifier, agent, wallet-agent). Read x/perm/keeper/session.go (or wherever the fee distribution logic lives) BEFORE writing tests.

Branch: test/step-28-perm-create-or-update-session
```

---

## Step 29 — PERM: SlashPermissionTrustDeposit

**Plan:** `docs/superpowers/plans/2026-05-06-step-29-perm-slash-permission-trust-deposit.md`

```
I'm working on issue #292 — comprehensive behavioral test overhaul (step 29 of 34).

Gate check: step 28 must be merged.

Read the plan at docs/superpowers/plans/2026-05-06-step-29-perm-slash-permission-trust-deposit.md and execute it task-by-task. Use the superpowers:executing-plans skill.

SlashPermissionTrustDeposit:
- NO bank transfers (calls BurnEcosystemSlashedTrustDeposit via mock TD keeper)
- Sets perm.Slashed, increments SlashedDeposit, decrements Deposit
- Authorized by validator ancestor OR TR controller (not perm corporation)

Branch: test/step-29-perm-slash-permission-td
```

---

## Step 30 — PERM: RepayPermissionSlashedTrustDeposit

**Plan:** `docs/superpowers/plans/2026-05-06-step-30-perm-repay-permission-slashed-trust-deposit.md`

```
I'm working on issue #292 — comprehensive behavioral test overhaul (step 30 of 34).

Gate check: step 29 must be merged.

Read the plan at docs/superpowers/plans/2026-05-06-step-30-perm-repay-permission-slashed-trust-deposit.md and execute it task-by-task. Use the superpowers:executing-plans skill.

RepayPermissionSlashedTrustDeposit:
- Calls AdjustTrustDeposit (mocked) — no direct bank transfer in PERM module
- Updates RepaidDeposit, Deposit
- Sets Repaid timestamp when RepaidDeposit >= SlashedDeposit (partial repay is allowed)

Branch: test/step-30-perm-repay-permission-slashed-td
```

---

## Step 31 — PERM: SelfCreatePermission

**Plan:** `docs/superpowers/plans/2026-05-06-step-31-perm-self-create-permission.md`

```
I'm working on issue #292 — comprehensive behavioral test overhaul (step 31 of 34).

Gate check: step 30 must be merged.

Read the plan at docs/superpowers/plans/2026-05-06-step-31-perm-self-create-permission.md and execute it task-by-task. Use the superpowers:executing-plans skill.

SelfCreatePermission:
- No fees, no bank transfers
- validator_perm must be ECOSYSTEM type (active or future)
- Schema must have OPEN onboarding mode for the requested type
- Coverage gate when step 31 merges: x/perm/keeper >= 95%

Branch: test/step-31-perm-self-create-permission
```

---

## Step 32 — XR: CreateExchangeRate, UpdateExchangeRate, ToggleExchangeRateState

**Plan:** `docs/superpowers/plans/2026-05-06-step-32-xr-exchange-rates.md`

```
I'm working on issue #292 — comprehensive behavioral test overhaul (step 32 of 34).

Gate check: step 0 must be merged.

Read the plan at docs/superpowers/plans/2026-05-06-step-32-xr-exchange-rates.md and execute it task-by-task. Use the superpowers:executing-plans skill.

XR module covers all 3 exchange rate messages in one step. No bank transfers — governance-only messages. Coverage gate: x/xr/keeper >= 95%.

Branch: test/step-32-xr-exchange-rates
```

---

## Step 33 — DI: StoreDigest

**Plan:** `docs/superpowers/plans/2026-05-06-step-33-di-store-digest.md`

```
I'm working on issue #292 — comprehensive behavioral test overhaul (step 33 of 34).

Gate check: step 0 must be merged.

Read the plan at docs/superpowers/plans/2026-05-06-step-33-di-store-digest.md and execute it task-by-task. Use the superpowers:executing-plans skill.

DI module: StoreDigest, no bank transfers. Key invariant: no duplicate digest hashes. Coverage gate: x/di/keeper >= 95%.

Branch: test/step-33-di-store-digest
```

---

## Step 34 — TS-Proto: Fill Remaining Journey Gaps

**Plan:** `docs/superpowers/plans/2026-05-06-step-34-ts-proto-gap-fill.md`

```
I'm working on issue #292 — comprehensive behavioral test overhaul (step 34 of 34).

Read the plan at docs/superpowers/plans/2026-05-06-step-34-ts-proto-gap-fill.md and execute it task-by-task. Use the superpowers:executing-plans skill.

CRITICAL: ALL TS journey files MUST use createAccountFromMnemonic (Secp256k1HdWallet — LEGACY_AMINO). Never createDirectAccountFromMnemonic.

Remaining journey gaps to fill:
- CS: csCreateSchemaAuthorizationPolicy.ts, csIncreaseActiveSAPVersion.ts, csRevokeSchemaAuthorizationPolicy.ts
- TD: tdSlashTrustDeposit.ts, tdBurnEcosystemSlashedTrustDeposit.ts
- DE: deGrantVsOperatorAuthorization.ts, deRevokeVsOperatorAuthorization.ts, deGrantExchangeRateAuthorization.ts, deRevokeExchangeRateAuthorization.ts
- XR: xrCreateExchangeRate.ts, xrUpdateExchangeRate.ts, xrToggleExchangeRateState.ts
- Update runAll.ts to include all new journeys in dependency order

Branch: test/step-34-ts-proto-gap-fill
```

---

## Quick Reference

| Step | Module | Message | Plan File |
|------|--------|---------|-----------|
| 0 | infra | StatefulBankMock | step-0-stateful-bank-mock.md |
| 1 | TD | ReclaimTrustDepositYield | step-1-td-reclaim-trust-deposit-yield.md |
| 2 | TD | SlashTrustDeposit | step-2-td-slash-trust-deposit.md |
| 3 | TD | RepaySlashedTrustDeposit | step-3-td-repay-slashed-trust-deposit.md |
| 4 | TD | BurnEcosystemSlashedTrustDeposit | step-4-td-burn-ecosystem-slashed-trust-deposit.md |
| 5 | TS-proto | permCreateRootPermission + permRenewPermissionVP | step-5-ts-proto-perm-create-root-renew-vp.md |
| 6 | TR | CreateTrustRegistry | step-6-tr-create-trust-registry.md |
| 7 | TR | AddGovernanceFrameworkDocument | step-7-tr-add-governance-framework-document.md |
| 8 | TR | IncreaseActiveGovernanceFrameworkVersion | step-8-tr-increase-active-gf-version.md |
| 9 | TR | UpdateTrustRegistry | step-9-tr-update-trust-registry.md |
| 10 | TR | ArchiveTrustRegistry | step-10-tr-archive-trust-registry.md |
| 11 | CS | CreateCredentialSchema | step-11-cs-create-credential-schema.md |
| 12 | CS | UpdateCredentialSchema | step-12-cs-update-credential-schema.md |
| 13 | CS | ArchiveCredentialSchema | step-13-cs-archive-credential-schema.md |
| 14 | CS | CreateSchemaAuthorizationPolicy | step-14-cs-create-schema-authorization-policy.md |
| 15 | CS | IncreaseActiveSAPVersion | step-15-cs-increase-active-sap-version.md |
| 16 | CS | RevokeSchemaAuthorizationPolicy | step-16-cs-revoke-schema-authorization-policy.md |
| 17 | DE | GrantOperatorAuthorization | step-17-de-grant-operator-authorization.md |
| 18 | DE | RevokeOperatorAuthorization | step-18-de-revoke-operator-authorization.md |
| 19 | DE | GrantVsOperatorAuthorization | step-19-de-grant-vs-operator-authorization.md |
| 20 | DE | RevokeVsOperatorAuthorization | step-20-de-revoke-vs-operator-authorization.md |
| 21 | PERM | CreateRootPermission | step-21-perm-create-root-permission.md |
| 22 | PERM | StartPermissionVP | step-22-perm-start-permission-vp.md |
| 23 | PERM | RenewPermissionVP | step-23-perm-renew-permission-vp.md |
| 24 | PERM | SetPermissionVPToValidated | step-24-perm-set-permission-vp-validated.md |
| 25 | PERM | CancelPermissionVPLastRequest | step-25-perm-cancel-permission-vp-last-request.md |
| 26 | PERM | AdjustPermission | step-26-perm-adjust-permission.md |
| 27 | PERM | RevokePermission | step-27-perm-revoke-permission.md |
| 28 | PERM | CreateOrUpdatePermissionSession | step-28-perm-create-or-update-permission-session.md |
| 29 | PERM | SlashPermissionTrustDeposit | step-29-perm-slash-permission-trust-deposit.md |
| 30 | PERM | RepayPermissionSlashedTrustDeposit | step-30-perm-repay-permission-slashed-trust-deposit.md |
| 31 | PERM | SelfCreatePermission | step-31-perm-self-create-permission.md |
| 32 | XR | CreateExchangeRate + UpdateExchangeRate + ToggleExchangeRateState | step-32-xr-exchange-rates.md |
| 33 | DI | StoreDigest | step-33-di-store-digest.md |
| 34 | TS-proto | Fill remaining journey gaps | step-34-ts-proto-gap-fill.md |
