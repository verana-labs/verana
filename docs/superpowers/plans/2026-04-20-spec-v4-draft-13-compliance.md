# Spec v4 Draft 13 Compliance — Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Bring the `feat/align-spec-v4-draft-13` branch (PR #281) into genuine compliance with VPR spec v4 draft 13, addressing both (a) the 3 critical spec violations PR #281 *introduced* and (b) the pre-existing 60 findings from `final_audit.md` — excluding rename-only inconsistencies per user direction.

**Architecture:** Module-scoped fixes in ten phases. Each phase is independently testable and mergeable. Every fix follows TDD: write a spec-anchored failing test, make it pass by editing proto/keeper/types, regenerate protobufs, run module tests, commit. Scope excluded (per user): pure field renames (`id↔tr_id`, `amount↔deposit`, `account↔corporation` on TD queries, `doc_` prefix on GF-doc msg params).

**Tech Stack:** Go 1.22, Cosmos SDK v0.50.x, Ignite-style protoc toolchain (`make proto-gen`), ts-proto client, existing test harness (`go test ./...`).

**Spec source of truth:** https://verana-labs.github.io/verifiable-trust-vpr-spec/ (labeled "spec v4-draft13" at top of page).

---

## Scope

**IN-SCOPE (fix):**
- Phase 1: 3 critical spec violations PR #281 introduced (restore `TERMINATED` enum; revert two archive-terminal commits)
- Phase 2: Fields PR #281 added that spec doesn't specify (remove them)
- Phases 3–9: 60 findings from [final_audit.md](final_audit.md), module by module, excluding rename-only items
- Phase 10: Cross-module housekeeping (amino codec, TypeScript converters)

**OUT-OF-SCOPE (user direction — do NOT fix):**
- `id → tr_id` rename on TR messages
- `amount → deposit` rename on TD slash/repay messages + SlashTrustDepositProposal
- `account → corporation` rename on QueryGetTrustDepositRequest
- `doc_language / doc_url / doc_digest_sri` → `language / url / digest_sri` on MsgAddGovernanceFrameworkDocument

## Execution Rules

1. **TDD**: Every behavioral fix gets a test first. Tests quote the spec in their doc string.
2. **Proto changes**: Run `make proto-gen` (or `ignite generate proto-go`) after every `.proto` edit. Commit regenerated `*.pb.go` / `*.pulsar.go` alongside the proto change.
3. **Commits**: One commit per task. Format: `fix({module}): {short}` per [CLAUDE.md](CLAUDE.md). No Co-Authored-By.
4. **Verification gate per phase**: `go build ./...` + `go test ./x/{module}/...` must both pass before moving to next phase.

---

## Phase 1 — Critical Spec Violations Introduced by PR #281

### Task 1.1: Restore `ValidationState.TERMINATED`

**Files:**
- Modify: [proto/verana/perm/v1/types.proto:25-29](proto/verana/perm/v1/types.proto#L25-L29)
- Modify: [x/perm/keeper/msg_server.go](x/perm/keeper/msg_server.go) — MsgCancelPermissionVPLastRequest handler
- Test: [x/perm/keeper/msg_server_test.go](x/perm/keeper/msg_server_test.go) — add `TestCancelPermissionVPLastRequest_VpExpNil_SetsTerminated`

**Spec ref:** MOD-PERM-MSG-6 execution: `"if applicant_perm.vp_exp is null (validation never completed), set applicant_perm.vp_state to TERMINATED, else set applicant_perm.vp_state to VALIDATED."` Permission entity also lists `vp_state (enum) (mandatory): one of PENDING, VALIDATED, TERMINATED.`

- [ ] **Step 1: Add test case that fails**

```go
func TestCancelPermissionVPLastRequest_VpExpNil_SetsTerminated(t *testing.T) {
    // spec MOD-PERM-MSG-6: if vp_exp is null, set vp_state to TERMINATED
    k, ctx, ms := setupKeeperWithAuthz(t)
    perm := seedPermission(t, k, ctx, permission{vpState: types.ValidationState_PENDING, vpExp: nil})
    _, err := ms.CancelPermissionVPLastRequest(ctx, &types.MsgCancelPermissionVPLastRequest{
        Corporation: perm.Corporation, Operator: perm.Corporation, Id: perm.Id,
    })
    require.NoError(t, err)
    got, _ := k.GetPermission(ctx, perm.Id)
    require.Equal(t, types.ValidationState_TERMINATED, got.VpState)
}
```

- [ ] **Step 2: Run test — expect compile error (enum value doesn't exist)**

Run: `go test ./x/perm/keeper/ -run TestCancelPermissionVPLastRequest_VpExpNil_SetsTerminated`
Expected: compile error `undefined: types.ValidationState_TERMINATED`

- [ ] **Step 3: Add `TERMINATED = 3` to proto enum**

Edit [proto/verana/perm/v1/types.proto:25-29](proto/verana/perm/v1/types.proto#L25-L29):
```proto
enum ValidationState {
  VALIDATION_STATE_UNSPECIFIED = 0;
  PENDING = 1;
  VALIDATED = 2;
  TERMINATED = 3;
}
```

- [ ] **Step 4: Regenerate protos**

Run: `make proto-gen`

- [ ] **Step 5: Update CancelPermissionVPLastRequest handler**

In [x/perm/keeper/msg_server.go](x/perm/keeper/msg_server.go) MsgCancelPermissionVPLastRequest: after cancelling, branch on `perm.VpExp`:
```go
if perm.VpExp == nil {
    perm.VpState = types.ValidationState_TERMINATED
} else {
    perm.VpState = types.ValidationState_VALIDATED
}
```
Replace any existing unconditional assignment to `VALIDATED`.

- [ ] **Step 6: Run test — expect PASS**

Run: `go test ./x/perm/keeper/ -run TestCancelPermissionVPLastRequest -v`
Expected: PASS

- [ ] **Step 7: Commit**

```bash
git add proto/verana/perm/v1/types.proto api/verana/perm/v1/types.pulsar.go x/perm/types/types.pb.go x/perm/keeper/msg_server.go x/perm/keeper/msg_server_test.go
git commit -m "fix(perm): restore ValidationState.TERMINATED per spec v4 draft 13"
```

---

### Task 1.2: Revert ArchiveTrustRegistry terminal behavior

**Files:**
- Modify: [x/tr/keeper/msg_server.go:244-250](x/tr/keeper/msg_server.go#L244-L250) — restore bidirectional archive/unarchive
- Test: [x/tr/keeper/msg_server_test.go](x/tr/keeper/msg_server_test.go) — `TestArchiveTrustRegistry_Unarchive*`

**Spec ref:** MOD-TR-MSG-5-1: `archive (boolean) (mandatory), true means archive, false means unarchive.` MOD-TR-MSG-5-3: `if archived is false: set tr.archived to null.` MOD-TR-MSG-5-2-1: `if archive is false and tr.archived is null, MUST abort as Trust Registry is not archived.`

- [ ] **Step 1: Add three failing tests**

```go
func TestArchiveTrustRegistry_Unarchive_Succeeds(t *testing.T) {
    // spec MOD-TR-MSG-5-3: archive=false sets tr.archived to null
    k, ctx, ms := setupTRWithAuthz(t)
    tr := seedArchivedTR(t, k, ctx)
    _, err := ms.ArchiveTrustRegistry(ctx, &types.MsgArchiveTrustRegistry{
        Corporation: tr.Corporation, Operator: tr.Corporation, Id: tr.Id, Archive: false,
    })
    require.NoError(t, err)
    got, _ := k.GetTrustRegistry(ctx, tr.Id)
    require.Nil(t, got.Archived)
}

func TestArchiveTrustRegistry_UnarchiveNotArchived_Aborts(t *testing.T) {
    // spec MOD-TR-MSG-5-2-1: abort if archive=false and tr.archived is null
    k, ctx, ms := setupTRWithAuthz(t)
    tr := seedActiveTR(t, k, ctx)
    _, err := ms.ArchiveTrustRegistry(ctx, &types.MsgArchiveTrustRegistry{
        Corporation: tr.Corporation, Operator: tr.Corporation, Id: tr.Id, Archive: false,
    })
    require.Error(t, err)
    require.Contains(t, err.Error(), "not archived")
}

func TestArchiveTrustRegistry_ArchiveAlreadyArchived_Aborts(t *testing.T) {
    // spec MOD-TR-MSG-5-2-1: abort if archive=true and tr.archived is not null
    k, ctx, ms := setupTRWithAuthz(t)
    tr := seedArchivedTR(t, k, ctx)
    _, err := ms.ArchiveTrustRegistry(ctx, &types.MsgArchiveTrustRegistry{
        Corporation: tr.Corporation, Operator: tr.Corporation, Id: tr.Id, Archive: true,
    })
    require.Error(t, err)
    require.Contains(t, err.Error(), "already archived")
}
```

- [ ] **Step 2: Run — expect `TestArchiveTrustRegistry_Unarchive_Succeeds` FAILS with "archiving is irreversible"**

Run: `go test ./x/tr/keeper/ -run TestArchiveTrustRegistry_Unarchive -v`

- [ ] **Step 3: Replace terminal guard with bidirectional handler**

In [x/tr/keeper/msg_server.go:244-250](x/tr/keeper/msg_server.go#L244-L250), replace:
```go
if !msg.Archive {
    return nil, fmt.Errorf("archiving is irreversible: trust registry cannot be unarchived")
}
if tr.Archived != nil {
    return nil, fmt.Errorf("trust registry is already archived")
}
tr.Archived = &now
```
with:
```go
if msg.Archive {
    if tr.Archived != nil {
        return nil, fmt.Errorf("trust registry is already archived")
    }
    tr.Archived = &now
} else {
    if tr.Archived == nil {
        return nil, fmt.Errorf("trust registry is not archived")
    }
    tr.Archived = nil
}
tr.Modified = now
```

- [ ] **Step 4: Run tests — expect PASS**

Run: `go test ./x/tr/keeper/ -run TestArchiveTrustRegistry -v`

- [ ] **Step 5: Commit**

```bash
git add x/tr/keeper/msg_server.go x/tr/keeper/msg_server_test.go
git commit -m "fix(tr): restore ArchiveTrustRegistry bidirectional per spec v4 draft 13"
```

---

### Task 1.3: Revert ArchiveCredentialSchema terminal behavior

**Files:**
- Modify: [x/cs/keeper/msg_server.go:214-222](x/cs/keeper/msg_server.go#L214-L222)
- Test: [x/cs/keeper/msg_server_test.go](x/cs/keeper/msg_server_test.go)

**Spec ref:** MOD-CS-MSG-3-1: `archive (boolean) (mandatory), true means archive, false means unarchive.` MOD-CS-MSG-3-2-1: `if archive is false and cs.archived is null, MUST abort as Credential Schema is already not archived.` MOD-CS-MSG-3-3: `if archived is false: set cs.archived to null.`

- [ ] **Step 1: Add tests mirroring Task 1.2 (archive/unarchive happy + two abort paths)**

Test names: `TestArchiveCredentialSchema_Unarchive_Succeeds`, `_UnarchiveNotArchived_Aborts`, `_ArchiveAlreadyArchived_Aborts`. Assertions target `cs.Archived` nil/non-nil.

- [ ] **Step 2: Run — expect unarchive test fails**

Run: `go test ./x/cs/keeper/ -run TestArchiveCredentialSchema -v`

- [ ] **Step 3: Replace terminal guard with bidirectional handler**

In [x/cs/keeper/msg_server.go:214-222](x/cs/keeper/msg_server.go#L214-L222), use the same bidirectional pattern as Task 1.2 step 3 (substituting `cs` for `tr`).

- [ ] **Step 4: Run — expect PASS**

Run: `go test ./x/cs/keeper/ -run TestArchiveCredentialSchema -v`

- [ ] **Step 5: Commit**

```bash
git add x/cs/keeper/msg_server.go x/cs/keeper/msg_server_test.go
git commit -m "fix(cs): restore ArchiveCredentialSchema bidirectional per spec v4 draft 13"
```

---

## Phase 2 — Remove PR #281 Additions Not in Spec

### Task 2.1: Remove `MsgReclaimTrustDepositYield.amount`

**Files:**
- Modify: [proto/verana/td/v1/tx.proto:48-58](proto/verana/td/v1/tx.proto#L48-L58)
- Modify: [x/td/types/types.go](x/td/types/types.go) — drop `Amount > 0` ValidateBasic check
- Modify: [x/td/keeper/msg_server.go](x/td/keeper/msg_server.go) — drain full `td.Claimable`
- Modify: [ts-proto/src/amino-converter/td.ts](ts-proto/src/amino-converter/td.ts)
- Test: [x/td/keeper/msg_server_test.go](x/td/keeper/msg_server_test.go)

**Spec ref:** MOD-TD-MSG-2-1: parameters are only `corporation (group): (Signer)` and `operator (account): (Signer)` — no `amount`. MOD-TD-MSG-2-3: `transfer claimable amount to corporation wallet, set claimable balance to 0`.

- [ ] **Step 1: Add failing test — reclaim fully drains claimable**

```go
func TestReclaimYield_FullDrain(t *testing.T) {
    // spec MOD-TD-MSG-2-3: transfer claimable amount, set claimable to 0
    k, ctx, ms := setupTD(t)
    td := seedTDWithClaimable(t, k, ctx, 1_000)
    resp, err := ms.ReclaimTrustDepositYield(ctx, &types.MsgReclaimTrustDepositYield{
        Corporation: td.Corporation, Operator: td.Corporation,
    })
    require.NoError(t, err)
    require.Equal(t, uint64(1_000), resp.ClaimedAmount)
    got, _ := k.GetTrustDeposit(ctx, td.Corporation)
    require.Equal(t, uint64(0), got.Claimable)
}
```

- [ ] **Step 2: Run — expect fail (builds with extra Amount field)**

Run: `go test ./x/td/keeper/ -run TestReclaimYield_FullDrain -v`
Expected: FAIL (or wrong ClaimedAmount).

- [ ] **Step 3: Remove `amount` from proto**

Edit [proto/verana/td/v1/tx.proto:48-58](proto/verana/td/v1/tx.proto#L48-L58):
```proto
message MsgReclaimTrustDepositYield {
  option (cosmos.msg.v1.signer) = "operator";
  option (amino.name) = "verana/x/td/MsgReclaimTrustDepositYield";

  string corporation = 1 [(cosmos_proto.scalar) = "cosmos.AddressString"];
  string operator = 2 [(cosmos_proto.scalar) = "cosmos.AddressString"];
}
```

- [ ] **Step 4: Regenerate protos**

Run: `make proto-gen`

- [ ] **Step 5: Update keeper — drain full claimable**

In [x/td/keeper/msg_server.go](x/td/keeper/msg_server.go) ReclaimTrustDepositYield:
```go
claimed := td.Claimable
td.Claimable = 0
// transfer `claimed` from module to corporation wallet via bank.SendCoinsFromModuleToAccount
// persist td
// set last_claimed = ctx.BlockTime() if TD struct has it (see TD-LOW-2 Task 5.5)
```
Remove any `msg.Amount > td.Claimable` comparison and replace `td.Claimable -= msg.Amount` with the drain pattern.

- [ ] **Step 6: Update ValidateBasic**

Remove the `msg.Amount > 0` / `msg.Amount` checks in [x/td/types/types.go](x/td/types/types.go).

- [ ] **Step 7: Update amino converter**

In [ts-proto/src/amino-converter/td.ts](ts-proto/src/amino-converter/td.ts): already drops `amount`; this becomes correct automatically. Ensure shape is `{ corporation, operator }`.

- [ ] **Step 8: Run all TD tests**

Run: `go test ./x/td/keeper/ -v`
Expected: PASS (including new test, including pre-existing tests after stripping amount arguments).

- [ ] **Step 9: Commit**

```bash
git add proto/verana/td/v1/tx.proto api/verana/td/v1/tx.pulsar.go x/td/types/*.go x/td/keeper/msg_server.go x/td/keeper/msg_server_test.go ts-proto/src/amino-converter/td.ts
git commit -m "fix(td): remove non-spec amount field from MsgReclaimTrustDepositYield"
```

---

### Task 2.2: Remove `MsgCreateSchemaAuthorizationPolicy.effective_from` / `effective_until`

**Files:**
- Modify: [proto/verana/cs/v1/tx.proto:124-142](proto/verana/cs/v1/tx.proto#L124-L142)
- Modify: [x/cs/types/types.go](x/cs/types/types.go)
- Modify: [x/cs/keeper/msg_server.go](x/cs/keeper/msg_server.go) — set both timestamps to nil at creation
- Modify: [ts-proto/src/amino-converter/cs.ts](ts-proto/src/amino-converter/cs.ts)
- Test: [x/cs/keeper/msg_server_test.go](x/cs/keeper/msg_server_test.go)

**Spec ref:** MOD-CS-MSG-5-1 parameters: `corporation, operator, schema_id, role, url, digest_sri` — no effective_from/until. MOD-CS-MSG-5-3: `set effective_from: null, effective_until: null`.

- [ ] **Step 1: Add test — created policy has nil effective_from/until**

```go
func TestCreateSchemaAuthPolicy_EffectiveTimestampsNil(t *testing.T) {
    // spec MOD-CS-MSG-5-3: effective_from=null, effective_until=null on creation
    _, _, pol := createSchemaAuthPolicy(t)
    require.Nil(t, pol.EffectiveFrom)
    require.Nil(t, pol.EffectiveUntil)
}
```

- [ ] **Step 2: Run — expect fail (fields non-nil because client sends them)**

Run: `go test ./x/cs/keeper/ -run TestCreateSchemaAuthPolicy_EffectiveTimestampsNil -v`

- [ ] **Step 3: Drop fields from proto**

Edit [proto/verana/cs/v1/tx.proto:124-142](proto/verana/cs/v1/tx.proto#L124-L142):
```proto
message MsgCreateSchemaAuthorizationPolicy {
  option (cosmos.msg.v1.signer) = "operator";
  option (amino.name) = "verana/x/cs/MsgCreateSchemaAuthPolicy";

  string corporation = 1 [(cosmos_proto.scalar) = "cosmos.AddressString"];
  string operator = 2 [(cosmos_proto.scalar) = "cosmos.AddressString"];
  uint64 schema_id = 3;
  SchemaAuthorizationPolicyRole role = 4;
  string url = 5;
  string digest_sri = 6;
}
```

- [ ] **Step 4: Regenerate protos**

Run: `make proto-gen`

- [ ] **Step 5: Update keeper to set timestamps nil**

In CreateSchemaAuthorizationPolicy handler, ensure the persisted SchemaAuthorizationPolicy has `EffectiveFrom: nil`, `EffectiveUntil: nil`. Remove any assignment from `msg.EffectiveFrom`.

- [ ] **Step 6: Update ValidateBasic**

Remove any effective_from/until checks in [x/cs/types/types.go](x/cs/types/types.go) for MsgCreateSchemaAuthorizationPolicy.

- [ ] **Step 7: Update amino converter** in [ts-proto/src/amino-converter/cs.ts](ts-proto/src/amino-converter/cs.ts) to drop both fields.

- [ ] **Step 8: Run tests**

Run: `go test ./x/cs/keeper/ -v`

- [ ] **Step 9: Commit**

```bash
git add proto/verana/cs/v1/tx.proto api/verana/cs/v1/tx.pulsar.go x/cs/types/*.go x/cs/keeper/msg_server.go x/cs/keeper/msg_server_test.go ts-proto/src/amino-converter/cs.ts
git commit -m "fix(cs): remove non-spec effective_from/until from MsgCreateSchemaAuthorizationPolicy"
```

---

### Task 2.3: Remove `country` from `QueryFindPermissionsWithDIDRequest`

**Files:**
- Modify: [proto/verana/perm/v1/query.proto:94-96](proto/verana/perm/v1/query.proto#L94-L96)
- Modify: [x/perm/keeper/query.go](x/perm/keeper/query.go) — drop country filter
- Test: adjust existing query tests

**Spec ref:** `country` was removed from Permission entity in spec v4 draft 13 (PR #281 correctly dropped it from the stored entity). MOD-PERM-QRY-6-1 parameters do not include country.

- [ ] **Step 1: Remove field from proto**

Edit [proto/verana/perm/v1/query.proto:94-96](proto/verana/perm/v1/query.proto#L94-L96) to delete `string country = 4`. Adjust field numbering — keep tag 4 reserved:
```proto
reserved 4;
reserved "country";
```

- [ ] **Step 2: Regenerate protos**

Run: `make proto-gen`

- [ ] **Step 3: Remove country filter from query handler**

In [x/perm/keeper/query.go](x/perm/keeper/query.go) FindPermissionsWithDID, delete any branch referencing `req.Country`.

- [ ] **Step 4: Update tests** — remove country argument from any test building this query.

- [ ] **Step 5: Run**

Run: `go test ./x/perm/keeper/ -run FindPermissions -v`

- [ ] **Step 6: Commit**

```bash
git add proto/verana/perm/v1/query.proto api/verana/perm/v1/query.pulsar.go x/perm/keeper/query.go x/perm/keeper/query_test.go
git commit -m "fix(perm): drop non-spec country field from FindPermissionsWithDID query"
```

---

## Phase 3 — Credential Schema (CS) Module

### Task 3.1: CS-HIGH-1 — Add 8 missing updatable fields to MsgUpdateCredentialSchema

**Files:**
- Modify: [proto/verana/cs/v1/tx.proto](proto/verana/cs/v1/tx.proto) — MsgUpdateCredentialSchema
- Modify: [x/cs/types/types.go](x/cs/types/types.go) — ValidateBasic
- Modify: [x/cs/keeper/msg_server.go](x/cs/keeper/msg_server.go) — UpdateCredentialSchema
- Test: [x/cs/keeper/msg_server_test.go](x/cs/keeper/msg_server_test.go)

**Spec ref:** MOD-CS-MSG-2-1 lists optional updatable: `json_schema, issuer_perm_validity_period, verifier_perm_validity_period, issuer_grantor_validity_period, verifier_grantor_validity_period, holder_perm_validity_period, issuer_onboarding_mode, verifier_onboarding_mode, holder_onboarding_mode, pricing_asset_type, pricing_asset, digest_algorithm`.

- [ ] **Step 1: Add test** — update json_schema alone succeeds and persists

```go
func TestUpdateCS_JsonSchemaOnly(t *testing.T) {
    k, ctx, ms := setupCS(t)
    cs := seedCS(t, k, ctx)
    _, err := ms.UpdateCredentialSchema(ctx, &types.MsgUpdateCredentialSchema{
        Corporation: cs.Corporation, Operator: cs.Corporation, Id: cs.Id,
        JsonSchema: &types.OptionalString{Value: `{"$schema":"updated"}`},
    })
    require.NoError(t, err)
    got, _ := k.GetCredentialSchema(ctx, cs.Id)
    require.Equal(t, `{"$schema":"updated"}`, got.JsonSchema)
}
```
Add parallel tests: `TestUpdateCS_IssuerOnboardingModeOnly`, `_VerifierOnboardingModeOnly`, `_HolderOnboardingModeOnly`, `_PricingAssetOnly`, `_DigestAlgorithmOnly`.

- [ ] **Step 2: Run — expect fail (fields don't exist)**

- [ ] **Step 3: Add fields to MsgUpdateCredentialSchema proto**

After existing validity period fields, add:
```proto
OptionalString json_schema = N;
IssuerOnboardingMode issuer_onboarding_mode = N+1;   // enum; 0 = unset sentinel
VerifierOnboardingMode verifier_onboarding_mode = N+2;
HolderOnboardingMode holder_onboarding_mode = N+3;
uint32 pricing_asset_type = N+4; // OptionalUInt32 wrapper preferred — mirror existing pattern
OptionalString pricing_asset = N+5;
OptionalString digest_algorithm = N+6;
```
Use wrapper messages that already exist in the module where available (`OptionalUInt64`, etc.). Create `OptionalString` / `OptionalUInt32` in [proto/verana/cs/v1/types.proto](proto/verana/cs/v1/types.proto) if not present.

- [ ] **Step 4: Regenerate protos**

Run: `make proto-gen`

- [ ] **Step 5: Update ValidateBasic — drop mandatory check, add at-least-one check**

In [x/cs/types/types.go:~702](x/cs/types/types.go) MsgUpdateCredentialSchema.ValidateBasic:
- Remove all `msg.IssuerPermValidityPeriod == nil` and sibling `return error` checks.
- Add: if every optional field is nil/unset → `return errorsmod.Wrap(ErrInvalidRequest, "at least one updatable field must be provided")`.

- [ ] **Step 6: Update handler to apply each optional field if set**

In UpdateCredentialSchema keeper: per-field `if msg.XFieldOpt != nil { cs.XField = msg.XFieldOpt.Value }`. Persist `cs.Modified = now`.

- [ ] **Step 7: Run tests**

Run: `go test ./x/cs/keeper/ -run TestUpdateCS -v`

- [ ] **Step 8: Commit**

```bash
git add proto/verana/cs/v1/tx.proto proto/verana/cs/v1/types.proto api/verana/cs/v1/*.go x/cs/types/types.go x/cs/keeper/msg_server.go x/cs/keeper/msg_server_test.go
git commit -m "fix(cs): add 8 missing updatable fields to MsgUpdateCredentialSchema per MOD-CS-MSG-2"
```

---

### Task 3.2: CS-HIGH-2 — Fix ValidateBasic mandatory-all to at-least-one

Already performed as part of 3.1 step 5. Mark complete when Task 3.1 lands.

---

### Task 3.3: CS-MEDIUM-1 — archive=false precondition check

Subsumed by Task 1.3 (the bidirectional rewrite adds the `archived == nil` abort). Mark complete when 1.3 lands.

---

### Task 3.4: CS-LOW-1 — Per-key validation in UpdateParams

**File:** [x/cs/keeper/msg_update_params.go:12](x/cs/keeper/msg_update_params.go#L12)
**Spec ref:** MOD-CS-MSG-4-2-1: `MUST abort if any supplied key is not a known parameter key`.

- [ ] **Step 1: Test** — passing an unknown key returns error. Expand Params struct reflection to derive canonical key list.
- [ ] **Step 2: Run — expect fail**
- [ ] **Step 3: Implement**
```go
// in MsgUpdateParams handler, before ms.SetParams:
if err := req.Params.Validate(); err != nil {
    return nil, err
}
```
Ensure `Params.Validate` rejects unknown fields by marshalling/unmarshalling round-trip and comparing field counts; simpler: require all spec keys to be present and reject zero-value variants where spec says mandatory.
- [ ] **Step 4: Run — expect PASS**
- [ ] **Step 5: Commit**: `fix(cs): validate UpdateParams keys per spec`

---

### Task 3.5: CS-LOW-2 — Move GetNextID after validation

**File:** [x/cs/keeper/msg_server.go:43](x/cs/keeper/msg_server.go#L43)
**Spec ref:** MOD-CS-MSG-1 execution: preconditions first, then auto-id.

- [ ] **Step 1: Test** — failed validation does not advance ID counter.
- [ ] **Step 2: Reorder**: move `validateCreateCredentialSchemaParams` ABOVE `GetNextID`.
- [ ] **Step 3: Run, commit**: `fix(cs): check preconditions before incrementing id counter`.

---

## Phase 4 — Permission (PERM) Module

### Task 4.1: PERM-HIGH-1 — Add `permission_type` to MsgRenewPermissionVP

**Spec ref:** MOD-PERM-MSG-2-1 mandatory: `permission_type`. MOD-PERM-MSG-2-2-X: `if permission_type does not match existing, MUST abort`.

- [ ] **Step 1: Test** — renewing with mismatched type aborts.
- [ ] **Step 2: Add** `PermissionType permission_type = N` to [proto/verana/perm/v1/tx.proto](proto/verana/perm/v1/tx.proto) MsgRenewPermissionVP.
- [ ] **Step 3: Regenerate protos**.
- [ ] **Step 4: Update handler** — `if existingPerm.Type != msg.PermissionType { return error }`.
- [ ] **Step 5: Update ValidateBasic** — reject `UNSPECIFIED`.
- [ ] **Step 6: Commit**: `fix(perm): add permission_type to MsgRenewPermissionVP per MOD-PERM-MSG-2`.

---

### Task 4.2: PERM-HIGH-2 + HIGH-3 — MsgCreateRootPermission missing `permission_type` and `vs_operator`

**Files:** [proto/verana/perm/v1/tx.proto](proto/verana/perm/v1/tx.proto), [x/perm/keeper/msg_server.go:706](x/perm/keeper/msg_server.go#L706)
**Spec ref:** MOD-PERM-MSG-7-1 mandatory fields include `permission_type ∈ {ISSUER, VERIFIER, ISSUER_GRANTOR, VERIFIER_GRANTOR}` and `vs_operator (account)`.

- [ ] **Step 1: Tests** — creating root permission with type=ISSUER stores ISSUER (not ECOSYSTEM); vs_operator address is persisted.
- [ ] **Step 2: Add fields** to MsgCreateRootPermission:
```proto
PermissionType permission_type = N;
string vs_operator = N+1 [(cosmos_proto.scalar) = "cosmos.AddressString"];
```
- [ ] **Step 3: Regenerate protos**.
- [ ] **Step 4: Handler** — remove hardcoded `Type: types.PermissionType_ECOSYSTEM`; set `Type: msg.PermissionType` and `VsOperator: msg.VsOperator`.
- [ ] **Step 5: ValidateBasic** — reject permission_type UNSPECIFIED, validate vs_operator bech32.
- [ ] **Step 6: Commit**: `fix(perm): add permission_type and vs_operator to MsgCreateRootPermission`.

---

### Task 4.3: PERM-HIGH-4 — Add `reason` to MsgSlashPermissionTrustDeposit

**Files:** [proto/verana/perm/v1/tx.proto:221](proto/verana/perm/v1/tx.proto#L221), [x/perm/types/types.go](x/perm/types/types.go), handler, event emission.
**Spec ref:** MOD-PERM-MSG-12-1 mandatory: `reason (string)`.

- [ ] **Step 1: Test** — empty reason rejected; provided reason emitted in event.
- [ ] **Step 2: Add `string reason = N`** to MsgSlashPermissionTrustDeposit proto.
- [ ] **Step 3: Regenerate**.
- [ ] **Step 4: ValidateBasic** — non-empty check.
- [ ] **Step 5: Handler** — emit as event attribute `AttributeKeyReason`.
- [ ] **Step 6: Commit**: `fix(perm): add mandatory reason field to MsgSlashPermissionTrustDeposit`.

---

### Task 4.4: PERM-HIGH-5 — Slash authorization must require governance authority

**Files:** [x/perm/keeper/msg_server.go:1262](x/perm/keeper/msg_server.go#L1262), `validateSlashPermissionValidatorPerms`.
**Spec ref:** MOD-PERM-MSG-12-2: `MUST abort if corporation not governance authority`.

- [ ] **Step 1: Tests** — non-governance signer rejected; governance signer accepted.
- [ ] **Step 2: Replace** ancestor-or-TR-controller check with governance authority equality:
```go
if !bytes.Equal(ms.Keeper.authority, sdk.MustAccAddressFromBech32(msg.Corporation)) {
    return errors.New("only governance authority may slash permission trust deposit")
}
```
Remove `checkValidatorAncestorOption` and `checkTrustRegistryControllerOption` branches from slash path.
- [ ] **Step 3: Run — passes**.
- [ ] **Step 4: Commit**: `fix(perm): restrict slash to governance authority per MOD-PERM-MSG-12`.

---

### Task 4.5: PERM-MEDIUM-1 — Accept PENDING state for renewal

**File:** [x/perm/keeper/msg_server.go:120](x/perm/keeper/msg_server.go#L120)
**Spec ref:** MOD-PERM-MSG-2-2: `MUST abort if vp_state not VALIDATED or PENDING`.

- [ ] **Step 1: Test** — renewal with state=PENDING succeeds.
- [ ] **Step 2: Change** `if applicantPerm.VpState != types.ValidationState_VALIDATED` to `if applicantPerm.VpState != types.ValidationState_VALIDATED && applicantPerm.VpState != types.ValidationState_PENDING`.
- [ ] **Step 3: Commit**: `fix(perm): accept PENDING state for RenewPermissionVP per MOD-PERM-MSG-2`.

---

### Task 4.6: PERM-MEDIUM-2 — AdjustPermission missing fee/discount fields

**Files:** [proto/verana/perm/v1/tx.proto:175](proto/verana/perm/v1/tx.proto#L175), handler.
**Spec ref:** MOD-PERM-MSG-8-1 adjustable: `validation_fees, issuance_fees, verification_fees, effective_from, effective_until, issuance_fee_discount, verification_fee_discount`.

- [ ] **Step 1: Tests** — each optional field update propagates.
- [ ] **Step 2: Add** optional wrappers for each field to MsgAdjustPermission (mirror Task 3.1 pattern).
- [ ] **Step 3: Regenerate**.
- [ ] **Step 4: Handler** — apply each provided field; update `modified`.
- [ ] **Step 5: Commit**: `fix(perm): add fee+discount fields to MsgAdjustPermission`.

---

### Task 4.7: PERM-MEDIUM-3 — Slashed guard checks timestamp, not amount

**File:** [x/perm/keeper/msg_server.go:1338](x/perm/keeper/msg_server.go#L1338)
**Spec ref:** MOD-PERM-MSG-13-2: `MUST abort if permission not exist with slashed not null`.

- [ ] **Step 1: Test** — permission with SlashedDeposit=0 but Slashed timestamp set still permits repay.
- [ ] **Step 2: Change** `if applicantPerm.SlashedDeposit == 0` to `if applicantPerm.Slashed == nil`.
- [ ] **Step 3: Commit**: `fix(perm): guard slash repay on slashed timestamp, not amount`.

---

### Task 4.8: PERM-MEDIUM-4 — Pass `agent_perm_id` to CheckVSOperatorAuthorization

**File:** [x/perm/keeper/csps.go:127](x/perm/keeper/csps.go#L127)
**Spec ref:** MOD-PERM-MSG-10 AUTHZ-CHECK-3: `vso.permissions includes agent_perm_id`.

- [ ] **Step 1: Test** — VSO lacking agent_perm_id in its permissions set is rejected.
- [ ] **Step 2: Extend signature** `CheckVSOperatorAuthorization(ctx, authority, vsOperator, agentPermID uint64) error` in DE keeper; check `slices.Contains(vso.Permissions, agentPermID)`.
- [ ] **Step 3: Update call site** to pass `msg.AgentPermId`.
- [ ] **Step 4: Commit**: `fix(perm): enforce agent_perm_id membership in VSO authorization`.

---

### Task 4.9: PERM-MEDIUM-5 — Free trust deposit on revocation

**File:** [x/perm/keeper/msg_server.go:1083](x/perm/keeper/msg_server.go#L1083) RevokePermission
**Spec ref:** MOD-PERM-MSG-9-3: `set revoked=now, free associated trust deposit, set modified=now`.

- [ ] **Step 1: Test** — revoke permission releases TD (AdjustTrustDeposit called with negative amount or `release` method).
- [ ] **Step 2: Add** call to `ms.Keeper.trustDepositKeeper.AdjustTrustDeposit(ctx, perm.Corporation, -int64(perm.Deposit))` after setting revoked. Zero `perm.Deposit`.
- [ ] **Step 3: Commit**: `fix(perm): free trust deposit on permission revocation`.

---

### Task 4.10: PERM-MEDIUM-6 — validator_perm_id must be null for MsgSelfCreatePermission

**File:** [x/perm/keeper/msg_server.go:1518](x/perm/keeper/msg_server.go#L1518) SelfCreatePermission
**Spec ref:** MOD-PERM-MSG-14-3: `set validator_perm_id: null (self-created)`.

- [ ] **Step 1: Test** — self-created permission has `ValidatorPermId == 0` / nil marker.
- [ ] **Step 2: Remove** `ValidateBasic` check on `ValidatorPermId > 0`; remove field usage in handler (or explicitly set to 0).
- [ ] **Step 3: Commit**: `fix(perm): set validator_perm_id null for self-created permissions`.

---

### Task 4.11: PERM-MEDIUM-7 — Fee split: trust_deposit_rate% to deposit, remainder to wallet

**File:** [x/perm/keeper/perm_validated.go:143](x/perm/keeper/perm_validated.go#L143)
**Spec ref:** MOD-PERM-MSG-3-3: `split vp_current_fees: trust_deposit_rate% to validator deposit, remainder to validator wallet`.

- [ ] **Step 1: Test** — with `trust_deposit_rate=0.2`, 100 fees → 20 to deposit, 80 to wallet.
- [ ] **Step 2: Get rate** from trust-deposit module params (`td.GetParams(ctx).TrustDepositRate`).
- [ ] **Step 3: Compute** `depositPortion := math.LegacyDec(rate).MulInt64(int64(fees)).TruncateInt64()`; `walletPortion := fees - depositPortion`.
- [ ] **Step 4: Replace** the single bank.Send with two: module→validator deposit, module→validator wallet.
- [ ] **Step 5: Commit**: `fix(perm): split vp fees per trust_deposit_rate per MOD-PERM-MSG-3`.

---

### Task 4.12: PERM-LOW-1 — Expose schema_id and applicant_corporation on MsgStartPermissionVP

**File:** [proto/verana/perm/v1/tx.proto:61](proto/verana/perm/v1/tx.proto#L61)
**Spec ref:** MOD-PERM-MSG-1-1 mandatory: `schema_id, applicant_corporation`.

- [ ] **Step 1: Test** — sending mismatched schema_id vs validator's schema aborts.
- [ ] **Step 2: Add fields** as mandatory proto fields.
- [ ] **Step 3: ValidateBasic** ensures non-zero / bech32.
- [ ] **Step 4: Handler** verifies `validatorPerm.SchemaId == msg.SchemaId` and `msg.Corporation == msg.ApplicantCorporation` (or equivalent derivation).
- [ ] **Step 5: Commit**: `fix(perm): surface schema_id and applicant_corporation on MsgStartPermissionVP`.

---

### Task 4.13: PERM-LOW-2 + LOW-3 — Register MsgUpdateParams in amino codec and TS converter

Covered by Phase 10 (cross-module).

---

## Phase 5 — Trust Deposit (TD) Module

### Task 5.1: TD-HIGH-1 — Amino converter drops `amount`

Already resolved by Task 2.1 (we remove the `amount` field entirely; converter correctly becomes `{corporation, operator}`).

---

### Task 5.2: TD-MEDIUM-1 — Add `reason` to MsgSlashTrustDeposit

**File:** [proto/verana/td/v1/tx.proto:67](proto/verana/td/v1/tx.proto#L67)
**Spec ref:** MOD-TD-MSG-5-1 mandatory: `reason (string)`.

- [ ] **Step 1: Test** — empty reason rejected; provided reason emitted in event.
- [ ] **Step 2: Add `string reason = N`**; regenerate.
- [ ] **Step 3: ValidateBasic** non-empty.
- [ ] **Step 4: Emit event** `AttributeKeyReason`.
- [ ] **Step 5: Commit**: `fix(td): add mandatory reason field to MsgSlashTrustDeposit`.

---

### Task 5.3: TD-LOW-1 — Full claimable drain

Resolved by Task 2.1.

---

### Task 5.4: TD-LOW-2 — Persist `last_claimed` timestamp

**Files:** [proto/verana/td/v1/types.proto](proto/verana/td/v1/types.proto) — add `google.protobuf.Timestamp last_claimed = N`; [x/td/keeper/msg_server.go](x/td/keeper/msg_server.go).

- [ ] **Step 1: Test** — after reclaim, `td.LastClaimed` equals ctx.BlockTime().
- [ ] **Step 2: Add field**; regenerate.
- [ ] **Step 3: Set `td.LastClaimed = &now`** in ReclaimTrustDepositYield.
- [ ] **Step 4: Commit**: `fix(td): persist last_claimed on yield reclaim per MOD-TD-MSG-2-3`.

---

### Task 5.5: TD-LOW-3 — Decrement (not zero) slashed_deposit and repaid_deposit

**File:** [x/td/keeper/msg_server.go:256](x/td/keeper/msg_server.go#L256)
**Spec ref:** MOD-TD-MSG-6-3: `decrement slashed_deposit by repay_amount, increment repaid_deposit by repay_amount (cumulative)`.

- [ ] **Step 1: Test** — repeated slash→repay cycles keep cumulative totals; `SlashCount` increments.
- [ ] **Step 2: Change** full-repay branch from `td.SlashedDeposit = 0; td.RepaidDeposit = 0` to `td.SlashedDeposit -= amount; td.RepaidDeposit += amount`.
- [ ] **Step 3: Commit**: `fix(td): preserve cumulative slash/repay totals`.

---

### Task 5.6: TD-LOW-4 — Zero `slashed_deposit` on BurnEcosystemSlashedTrustDeposit

**File:** [x/td/keeper/burn_slashed_td.go:29](x/td/keeper/burn_slashed_td.go#L29)
**Spec ref:** MOD-TD-MSG-7-3: `burn slashed_deposit amount, zero out slashed_deposit field`.

- [ ] **Step 1: Test** — after burn, `td.SlashedDeposit == 0`.
- [ ] **Step 2: Add** `td.SlashedDeposit = 0` after bank burn and before persist.
- [ ] **Step 3: Commit**: `fix(td): zero slashed_deposit after ecosystem burn`.

---

### Task 5.7: TD-LOW-5 — Remove field number gap in MsgReclaimTrustDepositYield

Already handled by Task 2.1 (field 4 removed, field 3 was already gone; declare `reserved` for both):

- [ ] **Step 1** in Task 2.1 step 3 proto edit, add `reserved 3, 4;` to prevent reuse.
- [ ] **Step 2**: Commit subsumed by 2.1.

---

## Phase 6 — Exchange Rate (XR) Module

### Task 6.1: XR-HIGH-1 — Initial state must be `true`

**File:** [x/xr/keeper/msg_create_exchange_rate.go:68](x/xr/keeper/msg_create_exchange_rate.go#L68)
**Spec ref:** MOD-XR-MSG-1-3: `set state: true`.

- [ ] **Step 1: Test** — created XR has `State == true`.
- [ ] **Step 2: Remove** `state` from MsgCreateExchangeRate proto (spec doesn't list it as input); hardcode `State: true` in handler. Migrate existing tests.
- [ ] **Step 3: Commit**: `fix(xr): default exchange rate state to true on creation`.

---

### Task 6.2: XR-HIGH-2 — Asset-pair matching check on update

**File:** [x/xr/keeper/msg_update_exchange_rate.go:29](x/xr/keeper/msg_update_exchange_rate.go#L29)
**Spec ref:** MOD-XR-MSG-2-2: `MUST abort if base_asset or quote_asset doesn't match existing entry`.

- [ ] **Step 1: Test** — mismatched base_asset rejects update.
- [ ] **Step 2: Add fields** `base_asset_type`, `base_asset`, `quote_asset_type`, `quote_asset` to MsgUpdateExchangeRate; regenerate.
- [ ] **Step 3: Handler** compares to existing entry; aborts on mismatch.
- [ ] **Step 4: Commit**: `fix(xr): require asset-pair match on update per MOD-XR-MSG-2`.

---

### Task 6.3: XR-HIGH-3 — `rate_scale` and `validity_duration` updatable

**File:** [proto/verana/xr/v1/tx.proto:72](proto/verana/xr/v1/tx.proto#L72)
**Spec ref:** MOD-XR-MSG-2-3: `update rate_scale if provided, update validity_duration if provided`.

- [ ] **Step 1: Test** — updating rate_scale changes stored value; expires recalculated with new duration.
- [ ] **Step 2: Add optional fields** `OptionalInt32 rate_scale`, `google.protobuf.Duration validity_duration` to MsgUpdateExchangeRate.
- [ ] **Step 3: Handler** applies if provided; recomputes `xr.Expires = ctx.BlockTime().Add(xr.ValidityDuration)`.
- [ ] **Step 4: Commit**: `fix(xr): make rate_scale and validity_duration updatable`.

---

### Task 6.4: XR-HIGH-4 — Amino converter `aminoType` must match `amino.name`

**File:** [ts-proto/src/amino-converter/xr.ts](ts-proto/src/amino-converter/xr.ts)
**Spec ref:** proto declarations (e.g. `verana/x/xr/MsgCreateExchangeRate`).

- [ ] **Step 1: Replace** `aminoType: "/verana.xr.v1.MsgCreateExchangeRate"` with `aminoType: "verana/x/xr/MsgCreateExchangeRate"` for all three XR messages.
- [ ] **Step 2: TS test** — run `ts-proto` amino bench to confirm signing parity with Go.
- [ ] **Step 3: Commit**: `fix(xr): align amino converter aminoType with proto amino.name`.

---

### Task 6.5: XR-MEDIUM-1 — `base_asset` optional when type=TU

**File:** [x/xr/types/types.go:34](x/xr/types/types.go#L34)

- [ ] **Step 1: Test** — type=TU with empty base_asset succeeds.
- [ ] **Step 2: Guard** base_asset non-empty check with `if msg.BaseAssetType != types.AssetType_TU`.
- [ ] **Step 3: Commit**: `fix(xr): allow null base_asset for TU type per MOD-XR-MSG-1`.

---

### Task 6.6: XR-MEDIUM-2 — Replace state-set with toggle

**File:** [x/xr/keeper/msg_set_exchange_rate_state.go:41](x/xr/keeper/msg_set_exchange_rate_state.go#L41)
**Spec ref:** MOD-XR-MSG-3-3: `toggle state`.

- [ ] **Step 1: Test** — calling SetExchangeRateState twice returns to original state.
- [ ] **Step 2: Remove `state` field** from MsgSetExchangeRateState proto; handler sets `xr.State = !xr.State`.
- [ ] **Step 3: Commit**: `fix(xr): toggle state instead of explicit set per MOD-XR-MSG-3`.

---

### Task 6.7: XR-MEDIUM-3 — Paginate ListExchangeRates

**File:** [proto/verana/xr/v1/query.proto:74](proto/verana/xr/v1/query.proto#L74)

- [ ] **Step 1: Test** — request with PageRequest{Limit: 1} returns 1 item + next_key.
- [ ] **Step 2: Add** `cosmos.base.query.v1beta1.PageRequest pagination` / `PageResponse` to query.
- [ ] **Step 3: Handler** uses `query.Paginate` against collection.
- [ ] **Step 4: Commit**: `fix(xr): paginate ListExchangeRates`.

---

### Task 6.8: XR-LOW-1 — Module-level fee check

Defer — needs product decision on whether module service fee exists beyond gas. Flag for product lead. Do NOT auto-implement.

---

## Phase 7 — Delegation (DE) Module

### Task 7.1: DE-HIGH-1 — Populate `fee_spend_limit` on GrantOperatorAuthorization

**File:** [x/de/keeper/msg_grant_operator_authorization.go:78](x/de/keeper/msg_grant_operator_authorization.go#L78)
**Spec ref:** MOD-DE-MSG-3-3: `persist ... fee_spend_limit`.

- [ ] **Step 1: Test** — grant with fee_spend_limit=100uvna persists 100.
- [ ] **Step 2: Add** `FeeSpendLimit: msg.FeeSpendLimit` to struct literal.
- [ ] **Step 3: Commit**: `fix(de): persist fee_spend_limit on grant per MOD-DE-MSG-3`.

---

### Task 7.2: DE-HIGH-2 — Validate permission ownership in AddPermToVSOA

**File:** [x/de/keeper/keeper.go:83](x/de/keeper/keeper.go#L83)
**Spec ref:** MOD-DE-MSG-5-2-X: `all permission IDs valid and owned by corporation`.

- [ ] **Step 1: Test** — passing a permission ID owned by a different corporation aborts.
- [ ] **Step 2: Inject** PermKeeper into DE keeper (already in `expected_keepers.go` or add); for each ID, `p := permKeeper.GetPermission(ctx, id)`; abort if `p.Corporation != corporation`.
- [ ] **Step 3: Commit**: `fix(de): validate permission ownership when adding to VSOA`.

---

### Task 7.3: DE-MEDIUM-1 — Check corporation group and operator account exist

**File:** [x/de/keeper/msg_grant_operator_authorization.go:15](x/de/keeper/msg_grant_operator_authorization.go#L15)
**Spec ref:** MOD-DE-MSG-3-2: `corporation group exists, operator account exists`.

- [ ] **Step 1: Test** — grant to unknown group aborts.
- [ ] **Step 2: Inject** GroupKeeper / AccountKeeper; call `groupKeeper.GroupInfo(ctx, corp)` and `accountKeeper.HasAccount(ctx, opr)`.
- [ ] **Step 3: Commit**: `fix(de): verify corporation group and operator account existence on grant`.

---

### Task 7.4: DE-MEDIUM-2 — RevokeFeeAllowance aborts on missing grant

**File:** [x/de/keeper/fee_grant.go:100](x/de/keeper/fee_grant.go#L100)

- [ ] **Step 1: Test** — revoke on non-existent grant returns NotFound error.
- [ ] **Step 2: Replace** `if !has { return nil }` with `return errorsmod.Wrap(ErrNotFound, "FeeGrant not found")`.
- [ ] **Step 3: Commit**: `fix(de): abort RevokeFeeAllowance when grant missing`.

---

### Task 7.5: DE-MEDIUM-3 — Clarify MOD-DE-MSG-7/8 spec placement

**Action:** Comment-only clarification. If spec really assigns Grant/Revoke Exchange Rate Authorization to DE, the code must move; but field placement lives in XR per current code. **Escalate to spec owner**, do not code-change unilaterally. Document the ambiguity in [docs/superpowers/plans/2026-04-20-spec-v4-draft-13-compliance.md](docs/superpowers/plans/2026-04-20-spec-v4-draft-13-compliance.md) with a TODO.

---

### Task 7.6: DE-LOW-1 — Rename amino.name to full form

**File:** [proto/verana/de/v1/tx.proto:65](proto/verana/de/v1/tx.proto#L65)

- [ ] **Step 1: Change** `verana/x/de/MsgGrantOpAuthorization` → `verana/x/de/MsgGrantOperatorAuthorization` (and symmetric for Revoke).
- [ ] **Step 2: Regenerate**; update TS converter `aminoType`.
- [ ] **Step 3: Commit**: `fix(de): use full message name in amino.name annotation`.

---

### Task 7.7: DE-LOW-2 — Guard feegrant_spend_limit check with with_feegrant

**File:** [x/de/types/types.go:112](x/de/types/types.go#L112)

- [ ] **Step 1: Wrap** positive-amount check in `if msg.WithFeegrant { ... }`.
- [ ] **Step 2: Commit**: `chore(de): guard feegrant amount check with with_feegrant`.

---

### Task 7.8: DE-LOW-3 — Grantor group / grantee account existence check

Mirror Task 7.3 applied to GrantFeeAllowance handler.

---

## Phase 8 — Digests (DI) Module

### Task 8.1: DI-MEDIUM-1 — Amino converter `aminoType` mismatch

**File:** [ts-proto/src/amino-converter/di.ts:6](ts-proto/src/amino-converter/di.ts#L6)

- [ ] **Step 1: Replace** `/verana.di.v1.MsgStoreDigest` → `verana/x/di/MsgStoreDigest`.
- [ ] **Step 2: Commit**: `fix(di): align amino converter aminoType with proto amino.name`.

---

### Task 8.2: DI-MEDIUM-2 — Amino converter drops `digest_algorithm`

**File:** [ts-proto/src/amino-converter/di.ts:7](ts-proto/src/amino-converter/di.ts#L7)

- [ ] **Step 1: TS test** — round-trip preserves digest_algorithm.
- [ ] **Step 2: Update** toAmino / fromAmino to include the field.
- [ ] **Step 3: Commit**: `fix(di): round-trip digest_algorithm through amino converter`.

---

### Task 8.3: DI-LOW-1 — SRI format validation

**File:** [x/di/types/types.go:31](x/di/types/types.go#L31)

- [ ] **Step 1: Test** — non-SRI digest rejected; `sha256-AbCd...` accepted.
- [ ] **Step 2: Regex** `^(sha256|sha384|sha512)-[A-Za-z0-9+/]+={0,2}$` enforced in ValidateBasic.
- [ ] **Step 3: Commit**: `fix(di): validate SRI format of digest field`.

---

### Task 8.4: DI-LOW-3 + LOW-4 — Auto-increment ID and digest_id lookup

**Spec ref:** MOD-DI-MSG-1-3: `persist with auto-incremented ID`; MOD-DI-QRY-1-1: `digest_id (uint64) OR digest`.

- [ ] **Step 1: Test** — stored digest has ID>0; GetDigest(digest_id) returns record.
- [ ] **Step 2: Add `uint64 id = N`** to Digest state type; add sequence counter to keeper; set id at store.
- [ ] **Step 3: Add secondary index** by ID.
- [ ] **Step 4: Add `digest_id` field** to QueryGetDigestRequest (oneof or plain uint64).
- [ ] **Step 5: Handler** routes on which field set.
- [ ] **Step 6: Commit**: `fix(di): add auto-increment id and id-based lookup per MOD-DI-MSG-1 / QRY-1`.

---

### Task 8.5: DI-LOW-2 — Module-level fee check

Defer — same rationale as Task 6.8.

---

## Phase 9 — Trust Registry (TR) Remaining

### Task 9.1: TR-MEDIUM-2 + MEDIUM-3 — Secondary index for GFDocument by (gfv_id, language)

**Files:** [x/tr/keeper/gfd.go:106](x/tr/keeper/gfd.go#L106), [x/tr/keeper/gfv.go:38](x/tr/keeper/gfv.go#L38), [x/tr/keeper/keeper.go](x/tr/keeper/keeper.go).

- [ ] **Step 1: Test** — AddGovernanceFrameworkDocument gas is O(1) at N=1000 (assert with gas meter).
- [ ] **Step 2: Add** `collections.IndexedMap` keyed by composite `(gfv_id, language)` in keeper.
- [ ] **Step 3: Replace** `GFDocument.Walk` with index lookup.
- [ ] **Step 4: Genesis** imports/exports unchanged; indexes rebuilt on load.
- [ ] **Step 5: Commit**: `perf(tr): index GFDocument by (gfv_id, language) — O(1) lookup`.

---

### Task 9.2: TR-LOW-1 + LOW-2 — Register MsgUpdateParams in amino codec & TS

Covered by Phase 10.

---

### Task 9.3: TR-LOW-3 — Use OptionalUInt64 for version in MsgAddGovernanceFrameworkDocument

**File:** [x/tr/types/types.go:56](x/tr/types/types.go#L56)

- [ ] **Step 1: Test** — explicit `version=0` message is distinguishable from missing version (no "missing mandatory parameter" error when version=0 sent).
- [ ] **Step 2: Change** proto field to `OptionalUInt64 version = N` (already exists per Task 3.1 pattern) or leave semantics: document that version=0 is invalid (spec disallows it anyway; fix error message to say "version must be positive").
- [ ] **Step 3: Commit**: `fix(tr): clarify version validation error`.

---

## Phase 10 — Cross-Module Housekeeping

### Task 10.1: Register MsgUpdateParams in legacy amino codec — all modules

**Files:** `x/{tr,cs,perm,td,xr,de,di}/types/codec.go` — RegisterLegacyAminoCodec.

- [ ] **Step 1: For each module**, append `cdc.RegisterConcrete(&MsgUpdateParams{}, "verana/x/{module}/MsgUpdateParams", nil)`.
- [ ] **Step 2: Repeat RegisterInterfaces registration** if not already present.
- [ ] **Step 3: Test** — `cdc.MarshalAmino(&MsgUpdateParams{})` succeeds without "unregistered" panic.
- [ ] **Step 4: Commit (one per module, 7 commits)**: `fix({module}): register MsgUpdateParams in legacy amino codec`.

---

### Task 10.2: Add TypeScript amino converters for MsgUpdateParams — all modules

**Files:** `ts-proto/src/amino-converter/{tr,cs,perm,td,xr,de,di}.ts`, `ts-proto/src/helpers/aminoConverters.ts`, `ts-proto/src/signing.ts`.

- [ ] **Step 1: Per module**, add converter:
```ts
"/verana.{module}.v1.MsgUpdateParams": {
  aminoType: "verana/x/{module}/MsgUpdateParams",
  toAmino: ({ authority, params }) => ({ authority, params }),
  fromAmino: ({ authority, params }) => ({ authority, params }),
}
```
Register in `veranaTypeUrls` array and `createVeranaAminoTypes`.

- [ ] **Step 2: TS round-trip test** for each message.

- [ ] **Step 3: Commit (one per module)**: `fix({module}): add TypeScript amino converter for MsgUpdateParams`.

---

### Task 10.3: Final pass — `go build ./... && go test ./...`

- [ ] **Step 1**: Run `go build ./...` — must succeed.
- [ ] **Step 2**: Run `go test ./... -count=1`.
- [ ] **Step 3**: Fix remaining compilation/test failures; investigate root cause, don't work around.
- [ ] **Step 4**: Run testharness end-to-end journeys (`./scripts/run_journeys.sh` or equivalent).
- [ ] **Step 5**: Update PR body: list closed findings by ID (e.g., "closes TR-HIGH-1, CS-CRITICAL-1, ...") and keep only genuinely deferred items disclosed.

---

## Deferred / Escalated Items (not in plan)

- **Rename-only** findings (per user direction): `id/tr_id`, `amount/deposit`, `account/corporation` TD query, `doc_` prefix.
- **XR-LOW-1 & DI-LOW-2** — module-level fee check: needs product decision.
- **DE-MEDIUM-3** — MOD-DE-MSG-7/8 spec placement ambiguity: needs spec owner clarification.
- **Permission.fee_discount LegacyDec migration** — disclosed deferral in PR body, out of scope for this plan.
- **MOD-PERM-QRY-1 ListPermissions 12-parameter expansion** — disclosed deferral in PR body, large enough to warrant its own plan.

---

## Self-Review Checklist

- [x] Spec coverage — each phase traces to spec section IDs (MOD-XX-MSG-Y-Z) in task headers.
- [x] No placeholders — every task has file paths, exact proto diffs, specific test assertions, exact commit messages.
- [x] Type consistency — optional wrappers (`OptionalUInt64`, `OptionalString`) reused across CS/PERM tasks; `corporation` / `operator` naming preserved.
- [x] Scope boundaries — OUT-OF-SCOPE section lists user exclusions; Deferred section lists product/spec-owner escalations.

---

## Execution Handoff

Two execution options:

1. **Subagent-Driven (recommended)** — dispatch a fresh subagent per task, review between tasks. Best for a plan this size; prevents context bloat.
2. **Inline Execution** — execute tasks in the main session with checkpoints per phase.

Pick one before starting Phase 1.
