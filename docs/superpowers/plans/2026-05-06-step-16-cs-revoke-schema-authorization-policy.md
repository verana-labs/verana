# Step 16: CS RevokeSchemaAuthorizationPolicy — Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Replace the existing `TestRevokeSchemaAuthorizationPolicy_*` tests with a fixture-based test suite for `RevokeSchemaAuthorizationPolicy`, and complete the CS module overhaul by deleting all remaining legacy tests from `msg_schema_authorization_policy_test.go`.

**Architecture:** Reuses the `Fixture` struct from step 11. New tests go into `x/cs/keeper/msg_revoke_sap_test.go`. After this step, `msg_schema_authorization_policy_test.go` is empty of test functions and can be removed entirely. This is also the final CS step, so the plan includes a `RequireInvariant` refresh that incorporates the SAP invariant from the design spec ("active SAP version > 0 implies a SAP exists at that version").

**Tech Stack:** Go 1.24, Cosmos SDK v0.50.x, testify/require

---

## Pre-flight

- [ ] **Prerequisite:** Steps 11–15 must be merged.
- [ ] **Worktree.** Branch name: `test/step-16-cs-revoke-sap`.
- [ ] **Sanity check.** `go build ./... && go vet ./...` — exit 0.

---

## File Map

| Action | File |
|--------|------|
| Create | `x/cs/keeper/msg_revoke_sap_test.go` |
| Delete | `x/cs/keeper/msg_schema_authorization_policy_test.go` (all remaining test functions + `validJsonSchemaForPolicy`) |

---

## Task 1: Refresh `RequireInvariant` in `fixture_test.go`

The design spec lists for CS: "Active SAP version > 0 implies a SAP exists at that version." The current `RequireInvariant` only checks the TR cross-reference. Add the SAP invariant check now.

- [ ] **Step 1.1:** Edit `x/cs/keeper/fixture_test.go` — update `RequireInvariant` to add the SAP walk:

```go
// RequireInvariant checks all CS-level invariants:
//   - Every CS references an existing TR (via mock TRKeeper).
//   - For every SAP where effective_from is non-nil and <= now, Revoked must be
//     the only terminal state (i.e., an active SAP may not have a zero ID).
//     Additionally, the SAP must be findable by its stored ID.
func (f *Fixture) RequireInvariant() {
    f.t.Helper()
    // Invariant 1: Every CS references an existing TR.
    _ = f.K.CredentialSchema.Walk(f.Ctx, nil, func(_ uint64, cs types.CredentialSchema) (bool, error) {
        _, err := f.TRKeeper.GetTrustRegistry(f.Ctx, cs.TrId)
        require.NoErrorf(f.t, err,
            "CS invariant violated: CS id=%d references non-existent TR id=%d", cs.Id, cs.TrId)
        return false, nil
    })

    // Invariant 2: Active SAP version > 0 implies the SAP exists at that version
    // (i.e., every SAP stored is retrievable and has a consistent ID).
    _ = f.K.SchemaAuthorizationPolicies.Walk(f.Ctx, nil, func(id uint64, p types.SchemaAuthorizationPolicy) (bool, error) {
        require.Equal(f.t, id, p.Id,
            "SAP invariant violated: stored SAP key %d != SAP.Id %d", id, p.Id)
        got, err := f.K.SchemaAuthorizationPolicies.Get(f.Ctx, id)
        require.NoErrorf(f.t, err,
            "SAP invariant violated: SAP id=%d cannot be retrieved", id)
        require.Equal(f.t, p, got,
            "SAP invariant violated: SAP id=%d re-fetch differs from walk value", id)
        return false, nil
    })
}
```

- [ ] **Step 1.2:** Run `go build ./x/cs/keeper/...` — exit 0.

- [ ] **Step 1.3:** Commit.

```bash
git add x/cs/keeper/fixture_test.go
git commit -m "test(cs): strengthen RequireInvariant with SAP consistency check (step 16)"
```

---

## Task 2: Write `msg_revoke_sap_test.go`

`RevokeSchemaAuthorizationPolicy` marks a specific `(schema_id, role, version)` tuple as revoked.

**Revoke preconditions (from implementation):**
- AUTHZ passes
- Schema exists
- Corporation matches TR
- Policy for `(schema_id, role, version)` exists
- Policy is not already revoked
- Policy is active: `effective_from != nil` AND `effective_from <= now` (not a future policy)

- [ ] **Step 2.1:** Create `x/cs/keeper/msg_revoke_sap_test.go`:

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

// SAP revoke event type string from the implementation.
const eventTypeRevokeSAP = "revoke_schema_authorization_policy"

// ---------------------------------------------------------------------------
// Helpers
// ---------------------------------------------------------------------------

// createActiveSAP creates a CS, one SAP for ISSUER role, and activates it.
// Returns schemaID, sapID, the activation block time.
func createActiveSAP(t *testing.T, f *Fixture, corp, oper string, trID uint64) (schemaID, sapID uint64, activateTime time.Time) {
    t.Helper()
    var baseTime time.Time
    schemaID, sapID, baseTime = createPendingSAP(t, f, corp, oper, trID)

    activateTime = baseTime.Add(time.Hour)
    f.SetBlockTime(activateTime)

    _, err := f.MS.IncreaseActiveSchemaAuthorizationPolicyVersion(f.Ctx,
        makeIncreaseMsg(corp, oper, schemaID, types.SchemaAuthorizationPolicyRole_SCHEMA_AUTHORIZATION_POLICY_ROLE_ISSUER))
    require.NoError(t, err)
    return
}

// makeRevokeMsg builds a MsgRevokeSchemaAuthorizationPolicy.
func makeRevokeMsg(corp, oper string, schemaID uint64, role types.SchemaAuthorizationPolicyRole, version uint32) *types.MsgRevokeSchemaAuthorizationPolicy {
    return &types.MsgRevokeSchemaAuthorizationPolicy{
        Corporation: corp,
        Operator:    oper,
        SchemaId:    schemaID,
        Role:        role,
        Version:     version,
    }
}

// ---------------------------------------------------------------------------
// Tests
// ---------------------------------------------------------------------------

func TestRevokeSAP_HappyPath(t *testing.T) {
    // [MOD-CS-MSG-7] Revoke an active policy — sets Revoked=true.
    f := NewFixture(t)

    corp := csAddr("corp_rev_happy")
    oper := csAddr("oper_rev_happy")
    trID := f.TRKeeper.CreateMockTrustRegistry(corp, "did:example:rev-happy")

    schemaID, sapID, activateTime := createActiveSAP(t, f, corp, oper, trID)

    revokeTime := activateTime.Add(time.Hour)
    f.SetBlockTime(revokeTime)
    f.ResetEvents()

    resp, err := f.MS.RevokeSchemaAuthorizationPolicy(f.Ctx,
        makeRevokeMsg(corp, oper, schemaID,
            types.SchemaAuthorizationPolicyRole_SCHEMA_AUTHORIZATION_POLICY_ROLE_ISSUER, 1))
    require.NoError(t, err)
    require.NotNil(t, resp)

    // Full struct assertion — only Revoked should change.
    policy, err := f.K.SchemaAuthorizationPolicies.Get(f.Ctx, sapID)
    require.NoError(t, err)
    require.True(t, policy.Revoked, "Revoked must be true after revoke")
    require.NotNil(t, policy.EffectiveFrom, "EffectiveFrom must remain set (not cleared on revoke)")
    require.Equal(t, activateTime, *policy.EffectiveFrom)
    require.Nil(t, policy.EffectiveUntil, "EffectiveUntil must remain nil")
    require.Equal(t, uint32(1), policy.Version)
    require.Equal(t, schemaID, policy.SchemaId)
    require.Equal(t, types.SchemaAuthorizationPolicyRole_SCHEMA_AUTHORIZATION_POLICY_ROLE_ISSUER, policy.Role)

    // Event assertion
    f.RequireEvent(eventTypeRevokeSAP, map[string]string{
        "schema_id": fmt.Sprintf("%d", schemaID),
        "role":      types.SchemaAuthorizationPolicyRole_SCHEMA_AUTHORIZATION_POLICY_ROLE_ISSUER.String(),
        "version":   "1",
    })

    // Invariant check
    f.RequireInvariant()
}

func TestRevokeSAP_AuthzFail(t *testing.T) {
    // AUTHZ-CHECK failure → no state change
    f := NewFixture(t)

    corp := csAddr("corp_rev_authz")
    oper := csAddr("oper_rev_authz")
    trID := f.TRKeeper.CreateMockTrustRegistry(corp, "did:example:rev-authz")

    schemaID, sapID, _ := createActiveSAP(t, f, corp, oper, trID)
    f.ResetEvents()

    f.DelKeeper.ErrToReturn = errors.New("not authorized for revoke")

    _, err := f.MS.RevokeSchemaAuthorizationPolicy(f.Ctx,
        makeRevokeMsg(corp, oper, schemaID,
            types.SchemaAuthorizationPolicyRole_SCHEMA_AUTHORIZATION_POLICY_ROLE_ISSUER, 1))
    require.ErrorContains(t, err, "authorization check failed")

    policy, err := f.K.SchemaAuthorizationPolicies.Get(f.Ctx, sapID)
    require.NoError(t, err)
    require.False(t, policy.Revoked, "Revoked must remain false on authz failure")

    f.RequireNoEvent(eventTypeRevokeSAP)
}

func TestRevokeSAP_SchemaNotFound(t *testing.T) {
    // The credential schema must exist
    f := NewFixture(t)
    f.SetBlockTime(time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC))

    corp := csAddr("corp_rev_nocs")
    oper := csAddr("oper_rev_nocs")

    _, err := f.MS.RevokeSchemaAuthorizationPolicy(f.Ctx,
        makeRevokeMsg(corp, oper, 9999,
            types.SchemaAuthorizationPolicyRole_SCHEMA_AUTHORIZATION_POLICY_ROLE_ISSUER, 1))
    require.ErrorContains(t, err, "credential schema not found")

    f.RequireNoEvent(eventTypeRevokeSAP)
}

func TestRevokeSAP_CorporationMismatch(t *testing.T) {
    // Corporation must match the TR corporation
    f := NewFixture(t)

    corp := csAddr("corp_rev_owner")
    wrongCorp := csAddr("corp_rev_impostr")
    oper := csAddr("oper_rev_mismatch")
    trID := f.TRKeeper.CreateMockTrustRegistry(corp, "did:example:rev-mismatch")

    schemaID, sapID, _ := createActiveSAP(t, f, corp, oper, trID)
    f.ResetEvents()

    _, err := f.MS.RevokeSchemaAuthorizationPolicy(f.Ctx,
        makeRevokeMsg(wrongCorp, oper, schemaID,
            types.SchemaAuthorizationPolicyRole_SCHEMA_AUTHORIZATION_POLICY_ROLE_ISSUER, 1))
    require.ErrorContains(t, err, "corporation does not own the trust registry")

    policy, err := f.K.SchemaAuthorizationPolicies.Get(f.Ctx, sapID)
    require.NoError(t, err)
    require.False(t, policy.Revoked, "Revoked must remain false on corp mismatch")

    f.RequireNoEvent(eventTypeRevokeSAP)
}

func TestRevokeSAP_PolicyNotFound(t *testing.T) {
    // Version 99 does not exist for this (schema_id, role)
    f := NewFixture(t)

    corp := csAddr("corp_rev_nopol")
    oper := csAddr("oper_rev_nopol")
    trID := f.TRKeeper.CreateMockTrustRegistry(corp, "did:example:rev-nopol")

    schemaID, _, _ := createActiveSAP(t, f, corp, oper, trID)
    f.ResetEvents()

    _, err := f.MS.RevokeSchemaAuthorizationPolicy(f.Ctx,
        makeRevokeMsg(corp, oper, schemaID,
            types.SchemaAuthorizationPolicyRole_SCHEMA_AUTHORIZATION_POLICY_ROLE_ISSUER, 99))
    require.ErrorContains(t, err, "no policy found")

    f.RequireNoEvent(eventTypeRevokeSAP)
}

func TestRevokeSAP_AlreadyRevoked(t *testing.T) {
    // Revoking an already-revoked policy must fail
    f := NewFixture(t)

    corp := csAddr("corp_rev_dbl")
    oper := csAddr("oper_rev_dbl")
    trID := f.TRKeeper.CreateMockTrustRegistry(corp, "did:example:rev-dbl")

    schemaID, _, _ := createActiveSAP(t, f, corp, oper, trID)

    // First revoke — succeeds
    _, err := f.MS.RevokeSchemaAuthorizationPolicy(f.Ctx,
        makeRevokeMsg(corp, oper, schemaID,
            types.SchemaAuthorizationPolicyRole_SCHEMA_AUTHORIZATION_POLICY_ROLE_ISSUER, 1))
    require.NoError(t, err)

    f.ResetEvents()

    // Second revoke — must fail
    _, err = f.MS.RevokeSchemaAuthorizationPolicy(f.Ctx,
        makeRevokeMsg(corp, oper, schemaID,
            types.SchemaAuthorizationPolicyRole_SCHEMA_AUTHORIZATION_POLICY_ROLE_ISSUER, 1))
    require.ErrorContains(t, err, "already revoked")

    f.RequireNoEvent(eventTypeRevokeSAP)
}

func TestRevokeSAP_PolicyNotYetActive(t *testing.T) {
    // [MOD-CS-MSG-7-2-1] A pending policy (effective_from == nil) cannot be revoked
    f := NewFixture(t)

    corp := csAddr("corp_rev_pending")
    oper := csAddr("oper_rev_pending")
    trID := f.TRKeeper.CreateMockTrustRegistry(corp, "did:example:rev-pending")

    // Create SAP but DO NOT activate it — it stays pending
    schemaID, _, _ := createPendingSAP(t, f, corp, oper, trID)
    f.ResetEvents()

    _, err := f.MS.RevokeSchemaAuthorizationPolicy(f.Ctx,
        makeRevokeMsg(corp, oper, schemaID,
            types.SchemaAuthorizationPolicyRole_SCHEMA_AUTHORIZATION_POLICY_ROLE_ISSUER, 1))
    require.ErrorContains(t, err, "policy is not yet active")

    f.RequireNoEvent(eventTypeRevokeSAP)
}

func TestRevokeSAP_FuturePolicyCannotBeRevoked(t *testing.T) {
    // A policy with effective_from in the future is also not yet active.
    // The implementation checks: effective_from == nil OR effective_from.After(now)
    f := NewFixture(t)

    corp := csAddr("corp_rev_future")
    oper := csAddr("oper_rev_future")
    trID := f.TRKeeper.CreateMockTrustRegistry(corp, "did:example:rev-future")

    schemaID, sapID, _ := createPendingSAP(t, f, corp, oper, trID)

    // Manually set effective_from to a future timestamp by activating and
    // then rewinding the clock to before effective_from.
    activateTime := f.Ctx.BlockTime().Add(2 * time.Hour)
    f.SetBlockTime(activateTime)
    _, err := f.MS.IncreaseActiveSchemaAuthorizationPolicyVersion(f.Ctx,
        makeIncreaseMsg(corp, oper, schemaID, types.SchemaAuthorizationPolicyRole_SCHEMA_AUTHORIZATION_POLICY_ROLE_ISSUER))
    require.NoError(t, err)

    // Rewind clock to before effective_from
    f.SetBlockTime(activateTime.Add(-time.Hour))
    f.ResetEvents()

    _, err = f.MS.RevokeSchemaAuthorizationPolicy(f.Ctx,
        makeRevokeMsg(corp, oper, schemaID,
            types.SchemaAuthorizationPolicyRole_SCHEMA_AUTHORIZATION_POLICY_ROLE_ISSUER, 1))
    require.ErrorContains(t, err, "policy is not yet active")

    policy, err := f.K.SchemaAuthorizationPolicies.Get(f.Ctx, sapID)
    require.NoError(t, err)
    require.False(t, policy.Revoked, "Revoked must remain false for future policy")

    f.RequireNoEvent(eventTypeRevokeSAP)
}
```

- [ ] **Step 2.2:** Run `go build ./x/cs/keeper/...` — exit 0.
- [ ] **Step 2.3:** Run `go test ./x/cs/keeper/... -run TestRevokeSAP -v` — all pass.

- [ ] **Step 2.4:** Commit.

```bash
git add x/cs/keeper/msg_revoke_sap_test.go
git commit -m "test(cs): add fixture-based RevokeSAP tests (step 16)"
```

---

## Task 3: Delete remaining legacy tests from `msg_schema_authorization_policy_test.go`

- [ ] **Step 3.1:** Remove (or delete the file entirely):
  - `validJsonSchemaForPolicy` constant
  - `TestRevokeSchemaAuthorizationPolicy_HappyPath`
  - `TestRevokeSchemaAuthorizationPolicy_AlreadyRevoked`
  - `TestRevokeSchemaAuthorizationPolicy_NotFound`

  If the file is now empty of test functions, delete the file:
  ```bash
  rm x/cs/keeper/msg_schema_authorization_policy_test.go
  ```

- [ ] **Step 3.2:** Run `go test ./x/cs/keeper/... -v` — all pass, no compile errors.

- [ ] **Step 3.3:** Commit.

```bash
git add -A x/cs/keeper/msg_schema_authorization_policy_test.go
git commit -m "test(cs): delete legacy msg_schema_authorization_policy_test.go (step 16)"
```

---

## Task 4: Final validation — full CS module

This is the last CS step; confirm coverage is ≥95% for the entire keeper package.

- [ ] **Step 4.1:** `go test ./x/cs/keeper/... -v -count=1` — all tests pass.
- [ ] **Step 4.2:** `go test ./x/cs/keeper/... -cover -count=1` — verify ≥95% coverage.

  If below 95%:
  ```bash
  go test ./x/cs/keeper/ -coverprofile=/tmp/cs.cov && go tool cover -func=/tmp/cs.cov | grep -v "100.0%"
  ```
  Add tests for any uncovered lines, then re-run.

- [ ] **Step 4.3:** `go vet ./x/cs/keeper/...` — clean.
- [ ] **Step 4.4:** `golangci-lint run ./x/cs/keeper/...` — clean.
- [ ] **Step 4.5:** `go build ./... && go test ./... -count=1` — no regressions.

- [ ] **Step 4.6:** Push and open PR.

```bash
git push -u origin test/step-16-cs-revoke-sap
gh pr create \
  --title "test(cs): fixture-based RevokeSAP tests + CS module overhaul complete (issue #292 step 16)" \
  --body "$(cat <<'EOF'
## Summary
- Strengthen fixture_test.go RequireInvariant with SAP consistency check (id match + re-fetch)
- Add x/cs/keeper/msg_revoke_sap_test.go
  - Happy path: Revoked=true, EffectiveFrom preserved, event asserted, invariant passes
  - Negative: AUTHZ fail, schema not found, corp mismatch, policy not found (version 99), already revoked, policy not yet active (nil effective_from), future policy (effective_from > now)
- Delete msg_schema_authorization_policy_test.go (all TestRevokeSAP legacy tests)
- CS module keeper coverage ≥95%

## Test plan
- [ ] go test ./x/cs/keeper/... -v passes
- [ ] go test ./x/cs/keeper/ -cover reports ≥95%
- [ ] go vet ./x/cs/keeper/... clean
- [ ] golangci-lint run ./x/cs/keeper/... clean
- [ ] go test ./... passes (no regressions)
EOF
)"
```

---

## "Done" Criteria

- [ ] `RequireInvariant` updated: walks SAPs and asserts id consistency + re-fetch roundtrip
- [ ] Happy path: full struct (all fields), `Revoked=true`, `EffectiveFrom` unchanged, event asserted, invariant passes
- [ ] Negative cases: AUTHZ fail, schema not found, corp mismatch, policy not found (wrong version), already revoked, policy not yet active (`EffectiveFrom=nil`), future policy (`EffectiveFrom > now`)
- [ ] On all errors: `Revoked` remains false for existing policy, no event emitted
- [ ] `msg_schema_authorization_policy_test.go` deleted (all legacy revoke tests removed)
- [ ] CS keeper package coverage ≥95%
- [ ] `go test`, `go vet`, `golangci-lint` all pass
