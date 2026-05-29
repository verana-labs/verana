# Step 15: CS IncreaseActiveSAPVersion — Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Add a fixture-based test suite for `IncreaseActiveSchemaAuthorizationPolicyVersion`, covering the happy path (sets `effective_from` to block time on the lowest-version pending policy) plus all failure cases derived from the implementation.

**Architecture:** Reuses the `Fixture` struct from step 11. New tests go into `x/cs/keeper/msg_increase_active_sap_version_test.go`. No old tests existed specifically for `IncreaseActiveSchemaAuthorizationPolicyVersion` in isolation — `msg_schema_authorization_policy_test.go` uses it only as a setup step inside revoke tests. Those callers remain; no file is deleted until step 16.

**Tech Stack:** Go 1.24, Cosmos SDK v0.50.x, testify/require

---

## Pre-flight

- [ ] **Prerequisite:** Steps 11–14 must be merged.
- [ ] **Worktree.** Branch name: `test/step-15-cs-increase-active-sap-version`.
- [ ] **Sanity check.** `go build ./... && go vet ./...` — exit 0.

---

## File Map

| Action | File |
|--------|------|
| Create | `x/cs/keeper/msg_increase_active_sap_version_test.go` |

No deletions in this step. The remaining tests in `msg_schema_authorization_policy_test.go` (`TestRevokeSchemaAuthorizationPolicy_*`) are removed in step 16.

---

## Task 1: Write `msg_increase_active_sap_version_test.go`

`IncreaseActiveSchemaAuthorizationPolicyVersion` activates the lowest-version non-revoked, non-yet-active (pending) SAP for a given `(schema_id, role)` by setting `effective_from` to the current block time.

**Pending definition (from implementation):**
- `EffectiveFrom == nil` (created but never activated), OR
- `EffectiveFrom` is in the future relative to block time

**Error conditions:**
- AUTHZ failure
- Schema not found
- Corporation mismatch
- No policies exist at all for `(schema_id, role)` → "no schema authorization policy exists"
- All policies are revoked or already active → "no future (non-active) policy version exists"

- [ ] **Step 1.1:** Create `x/cs/keeper/msg_increase_active_sap_version_test.go`:

```go
package keeper_test

import (
    "errors"
    "fmt"
    "testing"
    "time"

    "github.com/stretchr/testify/require"

    "github.com/verana-labs/verana/x/cs/types"
)

// SAP activation event type string from the implementation.
const eventTypeIncreaseActiveSAP = "increase_active_schema_authorization_policy_version"

// ---------------------------------------------------------------------------
// Helpers
// ---------------------------------------------------------------------------

// createPendingSAP creates a CS and one pending SAP for the ISSUER role.
// Returns the schema ID, the SAP ID (from response), and the base block time.
func createPendingSAP(t *testing.T, f *Fixture, corp, oper string, trID uint64) (schemaID, sapID uint64, baseTime time.Time) {
    t.Helper()
    var createTime time.Time
    schemaID, createTime = createCSForSAP(t, f, corp, oper, trID)
    baseTime = createTime

    sapResp, err := f.MS.CreateSchemaAuthorizationPolicy(f.Ctx, makeSAPMsg(
        corp, oper, schemaID,
        types.SchemaAuthorizationPolicyRole_SCHEMA_AUTHORIZATION_POLICY_ROLE_ISSUER,
        "https://example.com/policy/v1", "sha256-v1"))
    require.NoError(t, err)
    sapID = sapResp.Id
    return
}

// makeIncreaseMsg builds a MsgIncreaseActiveSchemaAuthorizationPolicyVersion.
func makeIncreaseMsg(corp, oper string, schemaID uint64, role types.SchemaAuthorizationPolicyRole) *types.MsgIncreaseActiveSchemaAuthorizationPolicyVersion {
    return &types.MsgIncreaseActiveSchemaAuthorizationPolicyVersion{
        Corporation: corp,
        Operator:    oper,
        SchemaId:    schemaID,
        Role:        role,
    }
}

// ---------------------------------------------------------------------------
// Tests
// ---------------------------------------------------------------------------

func TestIncreaseActiveSAPVersion_HappyPath(t *testing.T) {
    // [MOD-CS-MSG-6] Pending policy gets effective_from set to block time.
    f := NewFixture(t)

    corp := csAddr("corp_inc_happy")
    oper := csAddr("oper_inc_happy")
    trID := f.TRKeeper.CreateMockTrustRegistry(corp, "did:example:inc-happy")

    schemaID, sapID, _ := createPendingSAP(t, f, corp, oper, trID)

    activateTime := f.Ctx.BlockTime().Add(time.Hour)
    f.SetBlockTime(activateTime)
    f.ResetEvents()

    resp, err := f.MS.IncreaseActiveSchemaAuthorizationPolicyVersion(f.Ctx,
        makeIncreaseMsg(corp, oper, schemaID, types.SchemaAuthorizationPolicyRole_SCHEMA_AUTHORIZATION_POLICY_ROLE_ISSUER))
    require.NoError(t, err)
    require.NotNil(t, resp)

    // Full SAP struct assertion — only effective_from should change.
    policy, err := f.K.SchemaAuthorizationPolicies.Get(f.Ctx, sapID)
    require.NoError(t, err)
    require.Equal(t, activateTime, *policy.EffectiveFrom, "EffectiveFrom must equal block time at activation")
    require.Nil(t, policy.EffectiveUntil, "EffectiveUntil must remain nil")
    require.False(t, policy.Revoked, "Revoked must remain false")
    require.Equal(t, uint32(1), policy.Version)

    // Event assertion
    f.RequireEvent(eventTypeIncreaseActiveSAP, map[string]string{
        "schema_id":         fmt.Sprintf("%d", schemaID),
        "role":              types.SchemaAuthorizationPolicyRole_SCHEMA_AUTHORIZATION_POLICY_ROLE_ISSUER.String(),
        "new_active_version": "1",
    })

    // Invariant check — active SAP version > 0 implies SAP exists at that version
    f.RequireInvariant()
}

func TestIncreaseActiveSAPVersion_ActivatesLowestVersionFirst(t *testing.T) {
    // [MOD-CS-MSG-6-2-2] When multiple pending policies exist, the lowest version is activated.
    f := NewFixture(t)

    corp := csAddr("corp_inc_lowest")
    oper := csAddr("oper_inc_lowest")
    trID := f.TRKeeper.CreateMockTrustRegistry(corp, "did:example:inc-lowest")

    schemaID, _ := createCSForSAP(t, f, corp, oper, trID)

    // Create three pending SAPs → versions 1, 2, 3
    resp1, err := f.MS.CreateSchemaAuthorizationPolicy(f.Ctx, makeSAPMsg(
        corp, oper, schemaID,
        types.SchemaAuthorizationPolicyRole_SCHEMA_AUTHORIZATION_POLICY_ROLE_ISSUER,
        "https://example.com/v1", "sha256-1"))
    require.NoError(t, err)

    resp2, err := f.MS.CreateSchemaAuthorizationPolicy(f.Ctx, makeSAPMsg(
        corp, oper, schemaID,
        types.SchemaAuthorizationPolicyRole_SCHEMA_AUTHORIZATION_POLICY_ROLE_ISSUER,
        "https://example.com/v2", "sha256-2"))
    require.NoError(t, err)

    resp3, err := f.MS.CreateSchemaAuthorizationPolicy(f.Ctx, makeSAPMsg(
        corp, oper, schemaID,
        types.SchemaAuthorizationPolicyRole_SCHEMA_AUTHORIZATION_POLICY_ROLE_ISSUER,
        "https://example.com/v3", "sha256-3"))
    require.NoError(t, err)

    activateTime := f.Ctx.BlockTime().Add(time.Hour)
    f.SetBlockTime(activateTime)

    // Activate — must pick version 1 (lowest)
    _, err = f.MS.IncreaseActiveSchemaAuthorizationPolicyVersion(f.Ctx,
        makeIncreaseMsg(corp, oper, schemaID, types.SchemaAuthorizationPolicyRole_SCHEMA_AUTHORIZATION_POLICY_ROLE_ISSUER))
    require.NoError(t, err)

    p1, err := f.K.SchemaAuthorizationPolicies.Get(f.Ctx, resp1.Id)
    require.NoError(t, err)
    require.NotNil(t, p1.EffectiveFrom, "version 1 must be activated")
    require.Equal(t, activateTime, *p1.EffectiveFrom)

    p2, err := f.K.SchemaAuthorizationPolicies.Get(f.Ctx, resp2.Id)
    require.NoError(t, err)
    require.Nil(t, p2.EffectiveFrom, "version 2 must remain pending")

    p3, err := f.K.SchemaAuthorizationPolicies.Get(f.Ctx, resp3.Id)
    require.NoError(t, err)
    require.Nil(t, p3.EffectiveFrom, "version 3 must remain pending")

    f.RequireInvariant()
}

func TestIncreaseActiveSAPVersion_AuthzFail(t *testing.T) {
    // AUTHZ-CHECK failure → no state change
    f := NewFixture(t)

    corp := csAddr("corp_inc_authz")
    oper := csAddr("oper_inc_authz")
    trID := f.TRKeeper.CreateMockTrustRegistry(corp, "did:example:inc-authz")

    schemaID, sapID, _ := createPendingSAP(t, f, corp, oper, trID)
    f.ResetEvents()

    f.DelKeeper.ErrToReturn = errors.New("not authorized for increase")

    _, err := f.MS.IncreaseActiveSchemaAuthorizationPolicyVersion(f.Ctx,
        makeIncreaseMsg(corp, oper, schemaID, types.SchemaAuthorizationPolicyRole_SCHEMA_AUTHORIZATION_POLICY_ROLE_ISSUER))
    require.ErrorContains(t, err, "authorization check failed")

    policy, err := f.K.SchemaAuthorizationPolicies.Get(f.Ctx, sapID)
    require.NoError(t, err)
    require.Nil(t, policy.EffectiveFrom, "EffectiveFrom must remain nil on authz failure")

    f.RequireNoEvent(eventTypeIncreaseActiveSAP)
}

func TestIncreaseActiveSAPVersion_SchemaNotFound(t *testing.T) {
    // The credential schema must exist
    f := NewFixture(t)
    f.SetBlockTime(time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC))

    corp := csAddr("corp_inc_nocs")
    oper := csAddr("oper_inc_nocs")

    _, err := f.MS.IncreaseActiveSchemaAuthorizationPolicyVersion(f.Ctx,
        makeIncreaseMsg(corp, oper, 9999, types.SchemaAuthorizationPolicyRole_SCHEMA_AUTHORIZATION_POLICY_ROLE_ISSUER))
    require.ErrorContains(t, err, "credential schema not found")

    f.RequireNoEvent(eventTypeIncreaseActiveSAP)
}

func TestIncreaseActiveSAPVersion_CorporationMismatch(t *testing.T) {
    // Corporation must match the TR corporation
    f := NewFixture(t)

    corp := csAddr("corp_inc_owner")
    wrongCorp := csAddr("corp_inc_impostr")
    oper := csAddr("oper_inc_mismatch")
    trID := f.TRKeeper.CreateMockTrustRegistry(corp, "did:example:inc-mismatch")

    schemaID, sapID, _ := createPendingSAP(t, f, corp, oper, trID)
    f.ResetEvents()

    _, err := f.MS.IncreaseActiveSchemaAuthorizationPolicyVersion(f.Ctx,
        makeIncreaseMsg(wrongCorp, oper, schemaID, types.SchemaAuthorizationPolicyRole_SCHEMA_AUTHORIZATION_POLICY_ROLE_ISSUER))
    require.ErrorContains(t, err, "corporation does not own the trust registry")

    policy, err := f.K.SchemaAuthorizationPolicies.Get(f.Ctx, sapID)
    require.NoError(t, err)
    require.Nil(t, policy.EffectiveFrom, "EffectiveFrom must remain nil on corp mismatch")

    f.RequireNoEvent(eventTypeIncreaseActiveSAP)
}

func TestIncreaseActiveSAPVersion_NoPolicyExists(t *testing.T) {
    // No policies for (schema_id, role) at all → error "no schema authorization policy exists"
    f := NewFixture(t)

    corp := csAddr("corp_inc_nopol")
    oper := csAddr("oper_inc_nopol")
    trID := f.TRKeeper.CreateMockTrustRegistry(corp, "did:example:inc-nopol")

    schemaID, _ := createCSForSAP(t, f, corp, oper, trID)

    // No SAPs created for ISSUER role
    _, err := f.MS.IncreaseActiveSchemaAuthorizationPolicyVersion(f.Ctx,
        makeIncreaseMsg(corp, oper, schemaID, types.SchemaAuthorizationPolicyRole_SCHEMA_AUTHORIZATION_POLICY_ROLE_ISSUER))
    require.ErrorContains(t, err, "no schema authorization policy exists")

    f.RequireNoEvent(eventTypeIncreaseActiveSAP)
}

func TestIncreaseActiveSAPVersion_NoPendingPolicy(t *testing.T) {
    // All existing policies are revoked or already active — no pending policy exists.
    f := NewFixture(t)

    corp := csAddr("corp_inc_nopend")
    oper := csAddr("oper_inc_nopend")
    trID := f.TRKeeper.CreateMockTrustRegistry(corp, "did:example:inc-nopend")

    schemaID, _, _ := createPendingSAP(t, f, corp, oper, trID)

    // Activate the only pending policy
    f.AdvanceTime(time.Hour)
    _, err := f.MS.IncreaseActiveSchemaAuthorizationPolicyVersion(f.Ctx,
        makeIncreaseMsg(corp, oper, schemaID, types.SchemaAuthorizationPolicyRole_SCHEMA_AUTHORIZATION_POLICY_ROLE_ISSUER))
    require.NoError(t, err)

    f.ResetEvents()
    f.AdvanceTime(time.Minute)

    // Now try to activate again — the policy is already active, no pending policies remain
    _, err = f.MS.IncreaseActiveSchemaAuthorizationPolicyVersion(f.Ctx,
        makeIncreaseMsg(corp, oper, schemaID, types.SchemaAuthorizationPolicyRole_SCHEMA_AUTHORIZATION_POLICY_ROLE_ISSUER))
    require.ErrorContains(t, err, "no future (non-active) policy version exists")

    f.RequireNoEvent(eventTypeIncreaseActiveSAP)
}
```

- [ ] **Step 1.2:** Run `go build ./x/cs/keeper/...` — exit 0.
- [ ] **Step 1.3:** Run `go test ./x/cs/keeper/... -run TestIncreaseActiveSAPVersion -v` — all pass.

- [ ] **Step 1.4:** Commit.

```bash
git add x/cs/keeper/msg_increase_active_sap_version_test.go
git commit -m "test(cs): add fixture-based IncreaseActiveSAPVersion tests (step 15)"
```

---

## Task 2: Final validation

- [ ] **Step 2.1:** `go test ./x/cs/keeper/... -v -count=1` — all pass (new + remaining legacy).
- [ ] **Step 2.2:** `go test ./x/cs/keeper/... -cover -count=1` — coverage delta noted.
- [ ] **Step 2.3:** `go vet ./x/cs/keeper/...` — clean.
- [ ] **Step 2.4:** `golangci-lint run ./x/cs/keeper/...` — clean.
- [ ] **Step 2.5:** `go build ./... && go test ./... -count=1` — no regressions.

- [ ] **Step 2.6:** Push and open PR.

```bash
git push -u origin test/step-15-cs-increase-active-sap-version
gh pr create \
  --title "test(cs): fixture-based IncreaseActiveSAPVersion tests (issue #292 step 15)" \
  --body "$(cat <<'EOF'
## Summary
- Add x/cs/keeper/msg_increase_active_sap_version_test.go
  - Happy path: effective_from set to block time, correct event emitted, invariant passes
  - Lowest-version-first activation: three pending SAPs → only version 1 gets activated
  - Negative: AUTHZ fail, schema not found, corp mismatch, no policy at all, no pending policy remaining

## Test plan
- [ ] go test ./x/cs/keeper/... -v passes
- [ ] go vet ./x/cs/keeper/... clean
- [ ] golangci-lint run ./x/cs/keeper/... clean
- [ ] go test ./... passes (no regressions)
EOF
)"
```

---

## "Done" Criteria

- [ ] Happy path: `EffectiveFrom` = block time, `EffectiveUntil` = nil, `Revoked` = false, `Version` = 1, event with `new_active_version`, invariant passes
- [ ] Lowest-version ordering: three SAPs — only version 1 gets `EffectiveFrom` set, versions 2 and 3 remain pending
- [ ] Negative cases: AUTHZ fail (EffectiveFrom unchanged), schema not found, corp mismatch, no policy exists, no pending policy remains
- [ ] On errors: no state mutation, no event emitted
- [ ] `go test`, `go vet`, `golangci-lint` all pass
