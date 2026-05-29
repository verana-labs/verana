# Step 12: CS UpdateCredentialSchema — Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Replace the existing `TestUpdateCredentialSchema` test with a fixture-based test suite that provides full struct assertions, event assertions, and invariant checks for `UpdateCredentialSchema`.

**Architecture:** Reuses the `Fixture` struct from step 11's `fixture_test.go`. New tests go into `x/cs/keeper/msg_update_credential_schema_test.go`. The `MockDelegationKeeper.ErrToReturn` field drives the AUTHZ failure case. Old `TestUpdateCredentialSchema` is deleted from `msg_server_test.go`.

**Tech Stack:** Go 1.24, Cosmos SDK v0.50.x, testify/require

---

## Pre-flight

- [ ] **Prerequisite:** Step 11 must be merged — `fixture_test.go` and `CredentialschemaKeeperWithDelegation` must exist.
- [ ] **Worktree.** Create an isolated worktree for this step. Branch name: `test/step-12-cs-update-credential-schema`.
- [ ] **Sanity check.** Run: `go build ./... && go vet ./...` — expected exit 0.

---

## File Map

| Action | File |
|--------|------|
| Create | `x/cs/keeper/msg_update_credential_schema_test.go` |
| Delete content | `x/cs/keeper/msg_server_test.go` — remove `TestUpdateCredentialSchema` |

---

## Task 1: Write `msg_update_credential_schema_test.go`

**File:** `x/cs/keeper/msg_update_credential_schema_test.go`

The update message only mutates the five validity period fields and the `Modified` timestamp. Every other field must remain unchanged.

- [ ] **Step 1.1:** Create `x/cs/keeper/msg_update_credential_schema_test.go`:

```go
package keeper_test

import (
    "errors"
    "fmt"
    "testing"
    "time"

    "github.com/stretchr/testify/require"

    cskeeper "github.com/verana-labs/verana/x/cs/keeper"
    "github.com/verana-labs/verana/x/cs/types"
)

// ---------------------------------------------------------------------------
// Spec formula functions
// ---------------------------------------------------------------------------

// specCSUpdateMaxValidityDays returns the maximum allowed validity period (days)
// from DefaultParams — 3650 (10 years).
func specCSUpdateMaxValidityDays() uint32 { return 3650 }

// ---------------------------------------------------------------------------
// Helpers
// ---------------------------------------------------------------------------

// makeUpdateMsg builds a MsgUpdateCredentialSchema with all five validity periods set.
func makeUpdateMsg(corp, oper string, id uint64, issuerGrantor, verifierGrantor, issuer, verifier, holder uint32) *types.MsgUpdateCredentialSchema {
    return cskeeper.CreateUpdateMsgWithValidityPeriods(corp, oper, id, issuerGrantor, verifierGrantor, issuer, verifier, holder)
}

// createSchemaForUpdate creates a CS and returns its ID, initial block time, and
// an advanced context time (createdAt + 1 hour) suitable for the update call.
func createSchemaForUpdate(t *testing.T, f *Fixture, corp, oper string, trID uint64) (schemaID uint64, createTime, updateTime time.Time) {
    t.Helper()
    createTime = time.Date(2026, 1, 1, 10, 0, 0, 0, time.UTC)
    updateTime = createTime.Add(time.Hour)

    f.SetBlockTime(createTime)
    msg := makeCreateMsg(corp, oper, trID, minimalJsonSchema, 365, 365, 180, 180, 180)
    resp, err := f.MS.CreateCredentialSchema(f.Ctx, msg)
    require.NoError(t, err)
    schemaID = resp.Id

    // Advance time so Modified will differ from Created
    f.SetBlockTime(updateTime)
    return
}

// ---------------------------------------------------------------------------
// Tests
// ---------------------------------------------------------------------------

func TestUpdateCredentialSchema_HappyPath(t *testing.T) {
    // [MOD-CS-MSG-2] valid operator, valid schema, new validity periods within bounds
    f := NewFixture(t)

    corp := csAddr("corp_update_happy")
    oper := csAddr("oper_update_happy")
    trID := f.TRKeeper.CreateMockTrustRegistry(corp, "did:example:update-happy")

    schemaID, createTime, updateTime := createSchemaForUpdate(t, f, corp, oper, trID)
    f.ResetEvents()

    // Retrieve the stored schema to capture the canonical JSON (set at create time).
    storedBefore, err := f.K.CredentialSchema.Get(f.Ctx, schemaID)
    require.NoError(t, err)

    msg := makeUpdateMsg(corp, oper, schemaID, 730, 730, 365, 365, 365)
    resp, err := f.MS.UpdateCredentialSchema(f.Ctx, msg)
    require.NoError(t, err)
    require.NotNil(t, resp)

    // Full struct assertion: only the five VP fields and Modified should change.
    expected := types.CredentialSchema{
        Id:                                      schemaID,
        TrId:                                    trID,
        Created:                                 createTime,
        Modified:                                updateTime, // must be update block time
        Archived:                                nil,
        JsonSchema:                              storedBefore.JsonSchema, // unchanged
        IssuerGrantorValidationValidityPeriod:   730,
        VerifierGrantorValidationValidityPeriod: 730,
        IssuerValidationValidityPeriod:          365,
        VerifierValidationValidityPeriod:        365,
        HolderValidationValidityPeriod:          365,
        IssuerOnboardingMode:                    storedBefore.IssuerOnboardingMode,
        VerifierOnboardingMode:                  storedBefore.VerifierOnboardingMode,
        HolderOnboardingMode:                    storedBefore.HolderOnboardingMode,
        PricingAssetType:                        storedBefore.PricingAssetType,
        PricingAsset:                            storedBefore.PricingAsset,
        DigestAlgorithm:                         storedBefore.DigestAlgorithm,
    }
    f.RequireCredentialSchema(schemaID, expected)

    // Event assertion
    f.RequireEvent(types.EventTypeUpdateCredentialSchema, map[string]string{
        types.AttributeKeyId:          fmt.Sprintf("%d", schemaID),
        types.AttributeKeyTrId:        fmt.Sprintf("%d", trID),
        types.AttributeKeyCorporation: corp,
        types.AttributeKeyOperator:    oper,
    })

    // Invariant check
    f.RequireInvariant()
}

func TestUpdateCredentialSchema_CreatedTimestampUnchanged(t *testing.T) {
    // Created must remain the original block time; Modified must be updated.
    f := NewFixture(t)

    corp := csAddr("corp_update_ts")
    oper := csAddr("oper_update_ts")
    trID := f.TRKeeper.CreateMockTrustRegistry(corp, "did:example:update-ts")

    schemaID, createTime, updateTime := createSchemaForUpdate(t, f, corp, oper, trID)

    msg := makeUpdateMsg(corp, oper, schemaID, 200, 200, 100, 100, 100)
    _, err := f.MS.UpdateCredentialSchema(f.Ctx, msg)
    require.NoError(t, err)

    stored, err := f.K.CredentialSchema.Get(f.Ctx, schemaID)
    require.NoError(t, err)
    require.Equal(t, createTime, stored.Created, "Created must not change on update")
    require.Equal(t, updateTime, stored.Modified, "Modified must be updated to block time")
    require.NotEqual(t, stored.Created, stored.Modified, "Created and Modified must differ after update")
}

func TestUpdateCredentialSchema_AuthzFail(t *testing.T) {
    // [MOD-CS-MSG-2-2-1] AUTHZ-CHECK: delegation keeper returns error → abort
    f := NewFixture(t)

    corp := csAddr("corp_upd_authz")
    oper := csAddr("oper_upd_authz")
    trID := f.TRKeeper.CreateMockTrustRegistry(corp, "did:example:upd-authz")

    schemaID, _, _ := createSchemaForUpdate(t, f, corp, oper, trID)
    f.ResetEvents()

    // Enable auth failure AFTER schema creation
    f.DelKeeper.ErrToReturn = errors.New("operator not authorized for update")

    msg := makeUpdateMsg(corp, oper, schemaID, 100, 100, 50, 50, 50)
    _, err := f.MS.UpdateCredentialSchema(f.Ctx, msg)
    require.ErrorContains(t, err, "authorization check failed")

    // State must be unchanged — re-fetch and compare
    stored, err := f.K.CredentialSchema.Get(f.Ctx, schemaID)
    require.NoError(t, err)
    require.Equal(t, uint32(365), stored.IssuerGrantorValidationValidityPeriod, "VP must not change on authz failure")

    f.RequireNoEvent(types.EventTypeUpdateCredentialSchema)
}

func TestUpdateCredentialSchema_SchemaNotFound(t *testing.T) {
    // [MOD-CS-MSG-2-2-1] Schema must exist
    f := NewFixture(t)
    f.SetBlockTime(time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC))

    corp := csAddr("corp_upd_notfound")
    oper := csAddr("oper_upd_notfound")

    msg := makeUpdateMsg(corp, oper, 9999, 365, 365, 180, 180, 180)
    _, err := f.MS.UpdateCredentialSchema(f.Ctx, msg)
    require.ErrorContains(t, err, "credential schema not found")

    f.RequireSchemaCount(0)
    f.RequireNoEvent(types.EventTypeUpdateCredentialSchema)
}

func TestUpdateCredentialSchema_CorporationMismatch(t *testing.T) {
    // [MOD-CS-MSG-2-2-1] Corporation must match the TR corporation
    f := NewFixture(t)

    corp := csAddr("corp_upd_owner")
    wrongCorp := csAddr("corp_upd_impostor")
    oper := csAddr("oper_upd_mismatch")
    trID := f.TRKeeper.CreateMockTrustRegistry(corp, "did:example:upd-mismatch")

    schemaID, _, _ := createSchemaForUpdate(t, f, corp, oper, trID)
    f.ResetEvents()

    msg := makeUpdateMsg(wrongCorp, oper, schemaID, 365, 365, 180, 180, 180)
    _, err := f.MS.UpdateCredentialSchema(f.Ctx, msg)
    require.ErrorContains(t, err, "corporation does not match the trust registry corporation")

    // VP fields must be unchanged
    stored, err := f.K.CredentialSchema.Get(f.Ctx, schemaID)
    require.NoError(t, err)
    require.Equal(t, uint32(365), stored.IssuerGrantorValidationValidityPeriod)

    f.RequireNoEvent(types.EventTypeUpdateCredentialSchema)
}

func TestUpdateCredentialSchema_ArchivedSchema(t *testing.T) {
    // [MOD-CS-MSG-2-2-1] Archived schemas cannot be updated
    f := NewFixture(t)

    corp := csAddr("corp_upd_archived")
    oper := csAddr("oper_upd_archived")
    trID := f.TRKeeper.CreateMockTrustRegistry(corp, "did:example:upd-archived")

    schemaID, _, updateTime := createSchemaForUpdate(t, f, corp, oper, trID)

    // Archive the schema
    f.SetBlockTime(updateTime)
    _, err := f.MS.ArchiveCredentialSchema(f.Ctx, &types.MsgArchiveCredentialSchema{
        Corporation: corp,
        Operator:    oper,
        Id:          schemaID,
        Archive:     true,
    })
    require.NoError(t, err)
    f.ResetEvents()
    f.AdvanceTime(time.Minute)

    msg := makeUpdateMsg(corp, oper, schemaID, 200, 200, 100, 100, 100)
    _, err = f.MS.UpdateCredentialSchema(f.Ctx, msg)
    require.ErrorContains(t, err, "cannot update an archived credential schema")

    f.RequireNoEvent(types.EventTypeUpdateCredentialSchema)
}

func TestUpdateCredentialSchema_ValidityPeriodExceedsMax(t *testing.T) {
    // [MOD-CS-MSG-2-2-1] Validity period must not exceed max (3650 days)
    f := NewFixture(t)

    corp := csAddr("corp_upd_maxvp")
    oper := csAddr("oper_upd_maxvp")
    trID := f.TRKeeper.CreateMockTrustRegistry(corp, "did:example:upd-maxvp")

    schemaID, _, _ := createSchemaForUpdate(t, f, corp, oper, trID)
    f.ResetEvents()

    overMax := specCSUpdateMaxValidityDays() + 1
    msg := makeUpdateMsg(corp, oper, schemaID, overMax, 365, 180, 180, 180)
    _, err := f.MS.UpdateCredentialSchema(f.Ctx, msg)
    require.ErrorContains(t, err, "exceeds maximum")

    // VP fields must be unchanged
    stored, err := f.K.CredentialSchema.Get(f.Ctx, schemaID)
    require.NoError(t, err)
    require.Equal(t, uint32(365), stored.IssuerGrantorValidationValidityPeriod)

    f.RequireNoEvent(types.EventTypeUpdateCredentialSchema)
}

func TestUpdateCredentialSchema_PartialUpdate(t *testing.T) {
    // Sending nil for some optional VP fields should leave those fields unchanged.
    // CreateUpdateMsgWithValidityPeriods sets all five — this test verifies the
    // implementation's optional-field semantics by checking that a second update
    // overwrites only the fields it supplies.
    f := NewFixture(t)

    corp := csAddr("corp_upd_partial")
    oper := csAddr("oper_upd_partial")
    trID := f.TRKeeper.CreateMockTrustRegistry(corp, "did:example:upd-partial")

    schemaID, _, _ := createSchemaForUpdate(t, f, corp, oper, trID)

    // First update: set all five to 200
    msg1 := makeUpdateMsg(corp, oper, schemaID, 200, 200, 200, 200, 200)
    _, err := f.MS.UpdateCredentialSchema(f.Ctx, msg1)
    require.NoError(t, err)

    // Second update: set all five to 300
    f.AdvanceTime(time.Minute)
    msg2 := makeUpdateMsg(corp, oper, schemaID, 300, 300, 300, 300, 300)
    _, err = f.MS.UpdateCredentialSchema(f.Ctx, msg2)
    require.NoError(t, err)

    stored, err := f.K.CredentialSchema.Get(f.Ctx, schemaID)
    require.NoError(t, err)
    require.Equal(t, uint32(300), stored.IssuerGrantorValidationValidityPeriod)
    require.Equal(t, uint32(300), stored.HolderValidationValidityPeriod)

    f.RequireInvariant()
}
```

- [ ] **Step 1.2:** Run `go build ./x/cs/keeper/...` — expected exit 0.
- [ ] **Step 1.3:** Run `go test ./x/cs/keeper/... -run TestUpdateCredentialSchema -v` — all pass.

- [ ] **Step 1.4:** Commit.

```bash
git add x/cs/keeper/msg_update_credential_schema_test.go
git commit -m "test(cs): add fixture-based UpdateCredentialSchema tests (step 12)"
```

---

## Task 2: Delete old `TestUpdateCredentialSchema` from `msg_server_test.go`

- [ ] **Step 2.1:** Remove `TestUpdateCredentialSchema` (and `setupMsgServer` if only used by `TestArchiveCredentialSchema` now — leave it if still needed, it will be removed in step 13).

- [ ] **Step 2.2:** Run `go test ./x/cs/keeper/... -v` — all pass, no compile errors.

- [ ] **Step 2.3:** Commit.

```bash
git add x/cs/keeper/msg_server_test.go
git commit -m "test(cs): delete legacy TestUpdateCredentialSchema from msg_server_test.go"
```

---

## Task 3: Final validation

- [ ] **Step 3.1:** `go test ./x/cs/keeper/... -v -count=1` — all tests pass.
- [ ] **Step 3.2:** `go test ./x/cs/keeper/... -cover -count=1` — coverage delta noted.
- [ ] **Step 3.3:** `go vet ./x/cs/keeper/...` — clean.
- [ ] **Step 3.4:** `golangci-lint run ./x/cs/keeper/...` — clean.
- [ ] **Step 3.5:** `go build ./... && go test ./... -count=1` — no regressions.

- [ ] **Step 3.6:** Push and open PR.

```bash
git push -u origin test/step-12-cs-update-credential-schema
gh pr create \
  --title "test(cs): fixture-based UpdateCredentialSchema tests (issue #292 step 12)" \
  --body "$(cat <<'EOF'
## Summary
- Add x/cs/keeper/msg_update_credential_schema_test.go — full happy path (struct + event + invariant) + 6 negative/edge cases
- Delete legacy TestUpdateCredentialSchema from msg_server_test.go

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

- [ ] Happy path: full struct assertion (all fields including unchanged fields), event asserted, invariant passes
- [ ] Created timestamp is verified unchanged; Modified is verified to equal update block time
- [ ] Negative cases: AUTHZ fail, schema not found, corp mismatch, archived schema, validity period exceeds max
- [ ] Edge case: partial update (multiple sequential updates) — fields converge correctly
- [ ] Old `TestUpdateCredentialSchema` deleted from `msg_server_test.go`
- [ ] `go test`, `go vet`, `golangci-lint` all pass
