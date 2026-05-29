# Step 13: CS ArchiveCredentialSchema — Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Replace the existing `TestArchiveCredentialSchema` test with a fixture-based test suite that provides full struct assertions, event assertions, and invariant checks for the `ArchiveCredentialSchema` bidirectional toggle (archive=true and archive=false).

**Architecture:** Reuses the `Fixture` struct from step 11. New tests go into `x/cs/keeper/msg_archive_credential_schema_test.go`. `MockDelegationKeeper.ErrToReturn` drives AUTHZ failures. Old `TestArchiveCredentialSchema` and the now-unused `setupMsgServer` helper are deleted from `msg_server_test.go`.

**Tech Stack:** Go 1.24, Cosmos SDK v0.50.x, testify/require

---

## Pre-flight

- [ ] **Prerequisite:** Steps 11 and 12 must be merged.
- [ ] **Worktree.** Branch name: `test/step-13-cs-archive-credential-schema`.
- [ ] **Sanity check.** `go build ./... && go vet ./...` — exit 0.

---

## File Map

| Action | File |
|--------|------|
| Create | `x/cs/keeper/msg_archive_credential_schema_test.go` |
| Delete content | `x/cs/keeper/msg_server_test.go` — remove `TestArchiveCredentialSchema` and `setupMsgServer` (no remaining callers after this step) |

---

## Task 1: Write `msg_archive_credential_schema_test.go`

The `ArchiveCredentialSchema` message is a bidirectional toggle:
- `archive=true` on a non-archived CS sets `Archived` to block time
- `archive=false` on an archived CS sets `Archived` to nil
- `archive=true` on an already-archived CS → error "already archived"
- `archive=false` on a non-archived CS → error "not archived"

Both paths emit `EventTypeArchiveCredentialSchema` with `archive_status="archived"` or `archive_status="unarchived"`.

- [ ] **Step 1.1:** Create `x/cs/keeper/msg_archive_credential_schema_test.go`:

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

// ---------------------------------------------------------------------------
// Helpers
// ---------------------------------------------------------------------------

// createSchemaForArchive creates a CS at a fixed time and returns its ID and
// the block time used for creation. The fixture's block time is set to
// createTime after the call.
func createSchemaForArchive(t *testing.T, f *Fixture, corp, oper string, trID uint64) (schemaID uint64, createTime time.Time) {
    t.Helper()
    createTime = time.Date(2026, 2, 1, 10, 0, 0, 0, time.UTC)
    f.SetBlockTime(createTime)

    msg := makeCreateMsg(corp, oper, trID, minimalJsonSchema, 365, 365, 180, 180, 180)
    resp, err := f.MS.CreateCredentialSchema(f.Ctx, msg)
    require.NoError(t, err)
    schemaID = resp.Id
    return
}

// ---------------------------------------------------------------------------
// Tests
// ---------------------------------------------------------------------------

func TestArchiveCredentialSchema_Archive(t *testing.T) {
    // [MOD-CS-MSG-3] archive=true on a non-archived CS
    f := NewFixture(t)

    corp := csAddr("corp_archive_true")
    oper := csAddr("oper_archive_true")
    trID := f.TRKeeper.CreateMockTrustRegistry(corp, "did:example:archive-true")

    schemaID, createTime := createSchemaForArchive(t, f, corp, oper, trID)

    archiveTime := createTime.Add(2 * time.Hour)
    f.SetBlockTime(archiveTime)
    f.ResetEvents()

    // Retrieve pre-archive state for fields that should be unchanged.
    storedBefore, err := f.K.CredentialSchema.Get(f.Ctx, schemaID)
    require.NoError(t, err)

    resp, err := f.MS.ArchiveCredentialSchema(f.Ctx, &types.MsgArchiveCredentialSchema{
        Corporation: corp,
        Operator:    oper,
        Id:          schemaID,
        Archive:     true,
    })
    require.NoError(t, err)
    require.NotNil(t, resp)

    // Full struct assertion
    expected := types.CredentialSchema{
        Id:                                      schemaID,
        TrId:                                    trID,
        Created:                                 createTime,
        Modified:                                archiveTime, // set to block time
        Archived:                                &archiveTime,
        JsonSchema:                              storedBefore.JsonSchema,
        IssuerGrantorValidationValidityPeriod:   storedBefore.IssuerGrantorValidationValidityPeriod,
        VerifierGrantorValidationValidityPeriod: storedBefore.VerifierGrantorValidationValidityPeriod,
        IssuerValidationValidityPeriod:          storedBefore.IssuerValidationValidityPeriod,
        VerifierValidationValidityPeriod:        storedBefore.VerifierValidationValidityPeriod,
        HolderValidationValidityPeriod:          storedBefore.HolderValidationValidityPeriod,
        IssuerOnboardingMode:                    storedBefore.IssuerOnboardingMode,
        VerifierOnboardingMode:                  storedBefore.VerifierOnboardingMode,
        HolderOnboardingMode:                    storedBefore.HolderOnboardingMode,
        PricingAssetType:                        storedBefore.PricingAssetType,
        PricingAsset:                            storedBefore.PricingAsset,
        DigestAlgorithm:                         storedBefore.DigestAlgorithm,
    }
    f.RequireCredentialSchema(schemaID, expected)

    // Event assertion
    f.RequireEvent(types.EventTypeArchiveCredentialSchema, map[string]string{
        types.AttributeKeyId:            fmt.Sprintf("%d", schemaID),
        types.AttributeKeyTrId:          fmt.Sprintf("%d", trID),
        types.AttributeKeyCorporation:   corp,
        types.AttributeKeyOperator:      oper,
        types.AttributeKeyArchiveStatus: "archived",
    })

    // Invariant check
    f.RequireInvariant()
}

func TestArchiveCredentialSchema_Unarchive(t *testing.T) {
    // [MOD-CS-MSG-3-3] archive=false on an archived CS sets Archived to nil
    f := NewFixture(t)

    corp := csAddr("corp_unarchive")
    oper := csAddr("oper_unarchive")
    trID := f.TRKeeper.CreateMockTrustRegistry(corp, "did:example:unarchive")

    schemaID, createTime := createSchemaForArchive(t, f, corp, oper, trID)

    archiveTime := createTime.Add(time.Hour)
    f.SetBlockTime(archiveTime)

    // Archive first
    _, err := f.MS.ArchiveCredentialSchema(f.Ctx, &types.MsgArchiveCredentialSchema{
        Corporation: corp,
        Operator:    oper,
        Id:          schemaID,
        Archive:     true,
    })
    require.NoError(t, err)

    unarchiveTime := archiveTime.Add(time.Hour)
    f.SetBlockTime(unarchiveTime)
    f.ResetEvents()

    storedBefore, err := f.K.CredentialSchema.Get(f.Ctx, schemaID)
    require.NoError(t, err)
    require.NotNil(t, storedBefore.Archived, "schema should be archived before unarchive call")

    resp, err := f.MS.ArchiveCredentialSchema(f.Ctx, &types.MsgArchiveCredentialSchema{
        Corporation: corp,
        Operator:    oper,
        Id:          schemaID,
        Archive:     false,
    })
    require.NoError(t, err)
    require.NotNil(t, resp)

    // Full struct assertion — Archived must be nil, Modified updated
    expected := types.CredentialSchema{
        Id:                                      schemaID,
        TrId:                                    trID,
        Created:                                 createTime,
        Modified:                                unarchiveTime,
        Archived:                                nil,
        JsonSchema:                              storedBefore.JsonSchema,
        IssuerGrantorValidationValidityPeriod:   storedBefore.IssuerGrantorValidationValidityPeriod,
        VerifierGrantorValidationValidityPeriod: storedBefore.VerifierGrantorValidationValidityPeriod,
        IssuerValidationValidityPeriod:          storedBefore.IssuerValidationValidityPeriod,
        VerifierValidationValidityPeriod:        storedBefore.VerifierValidationValidityPeriod,
        HolderValidationValidityPeriod:          storedBefore.HolderValidationValidityPeriod,
        IssuerOnboardingMode:                    storedBefore.IssuerOnboardingMode,
        VerifierOnboardingMode:                  storedBefore.VerifierOnboardingMode,
        HolderOnboardingMode:                    storedBefore.HolderOnboardingMode,
        PricingAssetType:                        storedBefore.PricingAssetType,
        PricingAsset:                            storedBefore.PricingAsset,
        DigestAlgorithm:                         storedBefore.DigestAlgorithm,
    }
    f.RequireCredentialSchema(schemaID, expected)

    // Event assertion — archive_status must be "unarchived"
    f.RequireEvent(types.EventTypeArchiveCredentialSchema, map[string]string{
        types.AttributeKeyId:            fmt.Sprintf("%d", schemaID),
        types.AttributeKeyArchiveStatus: "unarchived",
    })

    f.RequireInvariant()
}

func TestArchiveCredentialSchema_AuthzFail(t *testing.T) {
    // [MOD-CS-MSG-3-2-1] AUTHZ-CHECK failure → abort, no state change
    f := NewFixture(t)

    corp := csAddr("corp_arch_authz")
    oper := csAddr("oper_arch_authz")
    trID := f.TRKeeper.CreateMockTrustRegistry(corp, "did:example:arch-authz")

    schemaID, _ := createSchemaForArchive(t, f, corp, oper, trID)
    f.ResetEvents()

    f.DelKeeper.ErrToReturn = errors.New("unauthorized for archive")

    _, err := f.MS.ArchiveCredentialSchema(f.Ctx, &types.MsgArchiveCredentialSchema{
        Corporation: corp,
        Operator:    oper,
        Id:          schemaID,
        Archive:     true,
    })
    require.ErrorContains(t, err, "authorization check failed")

    stored, err := f.K.CredentialSchema.Get(f.Ctx, schemaID)
    require.NoError(t, err)
    require.Nil(t, stored.Archived, "Archived must remain nil on authz failure")

    f.RequireNoEvent(types.EventTypeArchiveCredentialSchema)
}

func TestArchiveCredentialSchema_SchemaNotFound(t *testing.T) {
    // [MOD-CS-MSG-3-2-1] Schema must exist
    f := NewFixture(t)
    f.SetBlockTime(time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC))

    corp := csAddr("corp_arch_notfnd")
    oper := csAddr("oper_arch_notfnd")

    _, err := f.MS.ArchiveCredentialSchema(f.Ctx, &types.MsgArchiveCredentialSchema{
        Corporation: corp,
        Operator:    oper,
        Id:          9999,
        Archive:     true,
    })
    require.ErrorContains(t, err, "credential schema not found")

    f.RequireSchemaCount(0)
    f.RequireNoEvent(types.EventTypeArchiveCredentialSchema)
}

func TestArchiveCredentialSchema_CorporationMismatch(t *testing.T) {
    // [MOD-CS-MSG-3-2-1] Corporation must match the TR corporation
    f := NewFixture(t)

    corp := csAddr("corp_arch_owner")
    wrongCorp := csAddr("corp_arch_impos")
    oper := csAddr("oper_arch_mismatch")
    trID := f.TRKeeper.CreateMockTrustRegistry(corp, "did:example:arch-mismatch")

    schemaID, _ := createSchemaForArchive(t, f, corp, oper, trID)
    f.ResetEvents()

    _, err := f.MS.ArchiveCredentialSchema(f.Ctx, &types.MsgArchiveCredentialSchema{
        Corporation: wrongCorp,
        Operator:    oper,
        Id:          schemaID,
        Archive:     true,
    })
    require.ErrorContains(t, err, "corporation does not match the trust registry corporation")

    stored, err := f.K.CredentialSchema.Get(f.Ctx, schemaID)
    require.NoError(t, err)
    require.Nil(t, stored.Archived, "Archived must remain nil on corp mismatch")

    f.RequireNoEvent(types.EventTypeArchiveCredentialSchema)
}

func TestArchiveCredentialSchema_AlreadyArchived(t *testing.T) {
    // [MOD-CS-MSG-3] archive=true on an already-archived CS → error
    f := NewFixture(t)

    corp := csAddr("corp_dbl_archive")
    oper := csAddr("oper_dbl_archive")
    trID := f.TRKeeper.CreateMockTrustRegistry(corp, "did:example:dbl-archive")

    schemaID, createTime := createSchemaForArchive(t, f, corp, oper, trID)

    archiveTime := createTime.Add(time.Hour)
    f.SetBlockTime(archiveTime)
    _, err := f.MS.ArchiveCredentialSchema(f.Ctx, &types.MsgArchiveCredentialSchema{
        Corporation: corp,
        Operator:    oper,
        Id:          schemaID,
        Archive:     true,
    })
    require.NoError(t, err)

    f.AdvanceTime(time.Minute)
    f.ResetEvents()

    // Second archive attempt must fail
    _, err = f.MS.ArchiveCredentialSchema(f.Ctx, &types.MsgArchiveCredentialSchema{
        Corporation: corp,
        Operator:    oper,
        Id:          schemaID,
        Archive:     true,
    })
    require.ErrorContains(t, err, "already archived")

    // Archived timestamp must remain the original archiveTime
    stored, err := f.K.CredentialSchema.Get(f.Ctx, schemaID)
    require.NoError(t, err)
    require.NotNil(t, stored.Archived)
    require.Equal(t, archiveTime, *stored.Archived, "Archived timestamp must not change on duplicate archive")

    f.RequireNoEvent(types.EventTypeArchiveCredentialSchema)
}

func TestArchiveCredentialSchema_UnarchiveNotArchived(t *testing.T) {
    // [MOD-CS-MSG-3-2-1] archive=false on a non-archived CS → error "not archived"
    f := NewFixture(t)

    corp := csAddr("corp_unarch_none")
    oper := csAddr("oper_unarch_none")
    trID := f.TRKeeper.CreateMockTrustRegistry(corp, "did:example:unarch-none")

    schemaID, _ := createSchemaForArchive(t, f, corp, oper, trID)
    f.ResetEvents()

    _, err := f.MS.ArchiveCredentialSchema(f.Ctx, &types.MsgArchiveCredentialSchema{
        Corporation: corp,
        Operator:    oper,
        Id:          schemaID,
        Archive:     false,
    })
    require.ErrorContains(t, err, "not archived")

    stored, err := f.K.CredentialSchema.Get(f.Ctx, schemaID)
    require.NoError(t, err)
    require.Nil(t, stored.Archived, "Archived must remain nil")

    f.RequireNoEvent(types.EventTypeArchiveCredentialSchema)
}
```

- [ ] **Step 1.2:** Run `go build ./x/cs/keeper/...` — exit 0.
- [ ] **Step 1.3:** Run `go test ./x/cs/keeper/... -run TestArchiveCredentialSchema -v` — all pass.

- [ ] **Step 1.4:** Commit.

```bash
git add x/cs/keeper/msg_archive_credential_schema_test.go
git commit -m "test(cs): add fixture-based ArchiveCredentialSchema tests (step 13)"
```

---

## Task 2: Delete `TestArchiveCredentialSchema` and `setupMsgServer` from `msg_server_test.go`

- [ ] **Step 2.1:** Remove `TestArchiveCredentialSchema` from `msg_server_test.go`. Also remove `setupMsgServer` (it should have no remaining callers at this point).

- [ ] **Step 2.2:** Run `go test ./x/cs/keeper/... -v` — all pass, no compile errors.

- [ ] **Step 2.3:** Commit.

```bash
git add x/cs/keeper/msg_server_test.go
git commit -m "test(cs): delete legacy TestArchiveCredentialSchema and setupMsgServer"
```

---

## Task 3: Final validation

- [ ] **Step 3.1:** `go test ./x/cs/keeper/... -v -count=1` — all pass.
- [ ] **Step 3.2:** `go test ./x/cs/keeper/... -cover -count=1` — coverage delta noted.
- [ ] **Step 3.3:** `go vet ./x/cs/keeper/...` — clean.
- [ ] **Step 3.4:** `golangci-lint run ./x/cs/keeper/...` — clean.
- [ ] **Step 3.5:** `go build ./... && go test ./... -count=1` — no regressions.

- [ ] **Step 3.6:** Push and open PR.

```bash
git push -u origin test/step-13-cs-archive-credential-schema
gh pr create \
  --title "test(cs): fixture-based ArchiveCredentialSchema tests (issue #292 step 13)" \
  --body "$(cat <<'EOF'
## Summary
- Add x/cs/keeper/msg_archive_credential_schema_test.go
  - Happy path archive (archive=true): full struct assertion, Archived timestamp verified, archive_status="archived" event
  - Happy path unarchive (archive=false): full struct assertion, Archived=nil verified, archive_status="unarchived" event
  - Negative: AUTHZ fail, schema not found, corp mismatch, already archived, not archived (archive=false on live schema)
- Delete TestArchiveCredentialSchema and setupMsgServer from msg_server_test.go

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

- [ ] Happy path archive: full struct (all fields including unchanged ones), `Archived` = block time, `Modified` = block time, event with `archive_status="archived"`, invariant passes
- [ ] Happy path unarchive: full struct, `Archived` = nil, `Modified` = unarchive block time, event with `archive_status="unarchived"`, invariant passes
- [ ] Negative cases: AUTHZ fail, schema not found, corp mismatch, already archived, not archived (archive=false on live schema)
- [ ] On errors: `Archived` field is unchanged, no event emitted
- [ ] `TestArchiveCredentialSchema` and `setupMsgServer` deleted from `msg_server_test.go`
- [ ] `go test`, `go vet`, `golangci-lint` all pass
