# Step 14: CS CreateSchemaAuthorizationPolicy — Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Replace the existing `TestCreateSchemaAuthorizationPolicy_*` tests with a fixture-based test suite that provides full struct assertions, event assertions, and invariant checks for `CreateSchemaAuthorizationPolicy`.

**Architecture:** Reuses the `Fixture` struct from step 11. New tests go into `x/cs/keeper/msg_create_sap_test.go`. The SAP event constants come directly from the implementation (`"create_schema_authorization_policy"`). Old tests are deleted from `msg_schema_authorization_policy_test.go`.

**Tech Stack:** Go 1.24, Cosmos SDK v0.50.x, testify/require

---

## Pre-flight

- [ ] **Prerequisite:** Steps 11, 12, 13 must be merged.
- [ ] **Worktree.** Branch name: `test/step-14-cs-create-sap`.
- [ ] **Sanity check.** `go build ./... && go vet ./...` — exit 0.

---

## File Map

| Action | File |
|--------|------|
| Create | `x/cs/keeper/msg_create_sap_test.go` |
| Delete content | `x/cs/keeper/msg_schema_authorization_policy_test.go` — remove `TestCreateSchemaAuthorizationPolicy_HappyPath`, `TestCreateSchemaAuthorizationPolicy_VersionIncrement`, `TestCreateSchemaAuthorizationPolicy_SchemaNotFound`, `TestCreateSchemaAuthorizationPolicy_WrongCorporation` |

---

## Task 1: Write `msg_create_sap_test.go`

`CreateSchemaAuthorizationPolicy` creates a new SAP entry for a given `(schema_id, role)` pair. Per spec v4 draft 13:
- `effective_from` and `effective_until` are `nil` at creation (pending, not yet active)
- `version` auto-increments per `(schema_id, role)` — 1 for the first, 2 for the second, etc.
- `revoked` = false at creation
- The SAP event is emitted with raw string `"create_schema_authorization_policy"` (no constant in `types/events.go` — use the inline string from the implementation)

- [ ] **Step 1.1:** Create `x/cs/keeper/msg_create_sap_test.go`:

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

// SAP event type strings — taken directly from the implementation since they
// are not exported via types/events.go.
const (
    eventTypeCreateSAP = "create_schema_authorization_policy"
)

// ---------------------------------------------------------------------------
// Helpers
// ---------------------------------------------------------------------------

// createCSForSAP creates a credential schema and returns its ID.
// The fixture block time is set to baseTime.
func createCSForSAP(t *testing.T, f *Fixture, corp, oper string, trID uint64) (schemaID uint64, baseTime time.Time) {
    t.Helper()
    baseTime = time.Date(2026, 3, 1, 9, 0, 0, 0, time.UTC)
    f.SetBlockTime(baseTime)

    msg := makeCreateMsg(corp, oper, trID, minimalJsonSchema, 365, 365, 180, 180, 180)
    resp, err := f.MS.CreateCredentialSchema(f.Ctx, msg)
    require.NoError(t, err)
    schemaID = resp.Id

    // Advance time so SAP.Created differs from CS.Created
    f.AdvanceTime(time.Minute)
    return
}

// makeSAPMsg builds a MsgCreateSchemaAuthorizationPolicy.
func makeSAPMsg(corp, oper string, schemaID uint64, role types.SchemaAuthorizationPolicyRole, url, digestSri string) *types.MsgCreateSchemaAuthorizationPolicy {
    return &types.MsgCreateSchemaAuthorizationPolicy{
        Corporation: corp,
        Operator:    oper,
        SchemaId:    schemaID,
        Role:        role,
        Url:         url,
        DigestSri:   digestSri,
    }
}

// ---------------------------------------------------------------------------
// Tests
// ---------------------------------------------------------------------------

func TestCreateSAP_HappyPath(t *testing.T) {
    // [MOD-CS-MSG-5] valid operator, existing schema, matching corporation
    f := NewFixture(t)

    corp := csAddr("corp_sap_happy")
    oper := csAddr("oper_sap_happy")
    trID := f.TRKeeper.CreateMockTrustRegistry(corp, "did:example:sap-happy")

    schemaID, _ := createCSForSAP(t, f, corp, oper, trID)
    f.ResetEvents()

    sapTime := f.Ctx.BlockTime()

    msg := makeSAPMsg(corp, oper, schemaID,
        types.SchemaAuthorizationPolicyRole_SCHEMA_AUTHORIZATION_POLICY_ROLE_ISSUER,
        "https://example.com/policy/v1", "sha256-abc123")

    resp, err := f.MS.CreateSchemaAuthorizationPolicy(f.Ctx, msg)
    require.NoError(t, err)
    require.NotZero(t, resp.Id)

    // Full struct assertion — [MOD-CS-MSG-5-3]: effective_from and effective_until null at creation.
    expected := types.SchemaAuthorizationPolicy{
        Id:             resp.Id,
        SchemaId:       schemaID,
        Role:           types.SchemaAuthorizationPolicyRole_SCHEMA_AUTHORIZATION_POLICY_ROLE_ISSUER,
        Url:            "https://example.com/policy/v1",
        DigestSri:      "sha256-abc123",
        EffectiveFrom:  nil,
        EffectiveUntil: nil,
        Revoked:        false,
        Created:        sapTime,
        Version:        1,
    }
    f.RequireSAP(resp.Id, expected)

    // Event assertion
    f.RequireEvent(eventTypeCreateSAP, map[string]string{
        "id":        fmt.Sprintf("%d", resp.Id),
        "schema_id": fmt.Sprintf("%d", schemaID),
        "role":      types.SchemaAuthorizationPolicyRole_SCHEMA_AUTHORIZATION_POLICY_ROLE_ISSUER.String(),
        "version":   "1",
    })

    // Invariant check
    f.RequireInvariant()
    f.RequireSAPCount(1)
}

func TestCreateSAP_VersionIncrement(t *testing.T) {
    // Creating two SAPs for the same (schema_id, role) must yield versions 1 and 2.
    // A SAP for a different role resets the version counter for that role.
    f := NewFixture(t)

    corp := csAddr("corp_sap_version")
    oper := csAddr("oper_sap_version")
    trID := f.TRKeeper.CreateMockTrustRegistry(corp, "did:example:sap-version")

    schemaID, _ := createCSForSAP(t, f, corp, oper, trID)

    // First ISSUER SAP → version 1
    resp1, err := f.MS.CreateSchemaAuthorizationPolicy(f.Ctx, makeSAPMsg(
        corp, oper, schemaID,
        types.SchemaAuthorizationPolicyRole_SCHEMA_AUTHORIZATION_POLICY_ROLE_ISSUER,
        "https://example.com/policy/v1", "sha256-v1"))
    require.NoError(t, err)

    // Second ISSUER SAP → version 2
    resp2, err := f.MS.CreateSchemaAuthorizationPolicy(f.Ctx, makeSAPMsg(
        corp, oper, schemaID,
        types.SchemaAuthorizationPolicyRole_SCHEMA_AUTHORIZATION_POLICY_ROLE_ISSUER,
        "https://example.com/policy/v2", "sha256-v2"))
    require.NoError(t, err)

    // VERIFIER SAP for same schema → version 1 (independent counter per role)
    resp3, err := f.MS.CreateSchemaAuthorizationPolicy(f.Ctx, makeSAPMsg(
        corp, oper, schemaID,
        types.SchemaAuthorizationPolicyRole_SCHEMA_AUTHORIZATION_POLICY_ROLE_VERIFIER,
        "https://example.com/policy/v1-ver", "sha256-ver"))
    require.NoError(t, err)

    p1, err := f.K.SchemaAuthorizationPolicies.Get(f.Ctx, resp1.Id)
    require.NoError(t, err)
    require.Equal(t, uint32(1), p1.Version)

    p2, err := f.K.SchemaAuthorizationPolicies.Get(f.Ctx, resp2.Id)
    require.NoError(t, err)
    require.Equal(t, uint32(2), p2.Version)

    p3, err := f.K.SchemaAuthorizationPolicies.Get(f.Ctx, resp3.Id)
    require.NoError(t, err)
    require.Equal(t, uint32(1), p3.Version, "VERIFIER version must start at 1 independently of ISSUER versions")

    f.RequireSAPCount(3)
    f.RequireInvariant()
}

func TestCreateSAP_PendingStateAtCreation(t *testing.T) {
    // Verify that effective_from and effective_until are nil after creation.
    // This is a direct assertion of [MOD-CS-MSG-5-3].
    f := NewFixture(t)

    corp := csAddr("corp_sap_pending")
    oper := csAddr("oper_sap_pending")
    trID := f.TRKeeper.CreateMockTrustRegistry(corp, "did:example:sap-pending")

    schemaID, _ := createCSForSAP(t, f, corp, oper, trID)

    resp, err := f.MS.CreateSchemaAuthorizationPolicy(f.Ctx, makeSAPMsg(
        corp, oper, schemaID,
        types.SchemaAuthorizationPolicyRole_SCHEMA_AUTHORIZATION_POLICY_ROLE_ISSUER,
        "https://example.com/pending", "sha256-pend"))
    require.NoError(t, err)

    policy, err := f.K.SchemaAuthorizationPolicies.Get(f.Ctx, resp.Id)
    require.NoError(t, err)
    require.Nil(t, policy.EffectiveFrom, "EffectiveFrom must be nil (pending) at creation")
    require.Nil(t, policy.EffectiveUntil, "EffectiveUntil must be nil at creation")
    require.False(t, policy.Revoked, "Revoked must be false at creation")
}

func TestCreateSAP_AuthzFail(t *testing.T) {
    // AUTHZ-CHECK failure → no SAP written
    f := NewFixture(t)

    corp := csAddr("corp_sap_authz")
    oper := csAddr("oper_sap_authz")
    trID := f.TRKeeper.CreateMockTrustRegistry(corp, "did:example:sap-authz")

    schemaID, _ := createCSForSAP(t, f, corp, oper, trID)
    f.ResetEvents()

    f.DelKeeper.ErrToReturn = errors.New("not authorized to create SAP")

    _, err := f.MS.CreateSchemaAuthorizationPolicy(f.Ctx, makeSAPMsg(
        corp, oper, schemaID,
        types.SchemaAuthorizationPolicyRole_SCHEMA_AUTHORIZATION_POLICY_ROLE_ISSUER,
        "https://example.com/policy", "sha256-abc"))
    require.ErrorContains(t, err, "authorization check failed")

    f.RequireSAPCount(0)
    f.RequireNoEvent(eventTypeCreateSAP)
}

func TestCreateSAP_SchemaNotFound(t *testing.T) {
    // The credential schema must exist
    f := NewFixture(t)
    f.SetBlockTime(time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC))

    corp := csAddr("corp_sap_nocs")
    oper := csAddr("oper_sap_nocs")

    _, err := f.MS.CreateSchemaAuthorizationPolicy(f.Ctx, makeSAPMsg(
        corp, oper, 9999,
        types.SchemaAuthorizationPolicyRole_SCHEMA_AUTHORIZATION_POLICY_ROLE_ISSUER,
        "https://example.com/policy", "sha256-abc"))
    require.ErrorContains(t, err, "credential schema not found")

    f.RequireSAPCount(0)
    f.RequireNoEvent(eventTypeCreateSAP)
}

func TestCreateSAP_CorporationMismatch(t *testing.T) {
    // The corporation in the message must match the TR corporation
    f := NewFixture(t)

    corp := csAddr("corp_sap_owner")
    wrongCorp := csAddr("corp_sap_impostr")
    oper := csAddr("oper_sap_mismatch")
    trID := f.TRKeeper.CreateMockTrustRegistry(corp, "did:example:sap-mismatch")

    schemaID, _ := createCSForSAP(t, f, corp, oper, trID)
    f.ResetEvents()

    _, err := f.MS.CreateSchemaAuthorizationPolicy(f.Ctx, makeSAPMsg(
        wrongCorp, oper, schemaID,
        types.SchemaAuthorizationPolicyRole_SCHEMA_AUTHORIZATION_POLICY_ROLE_ISSUER,
        "https://example.com/policy", "sha256-abc"))
    require.ErrorContains(t, err, "corporation does not own the trust registry")

    f.RequireSAPCount(0)
    f.RequireNoEvent(eventTypeCreateSAP)
}
```

- [ ] **Step 1.2:** Run `go build ./x/cs/keeper/...` — exit 0.
- [ ] **Step 1.3:** Run `go test ./x/cs/keeper/... -run TestCreateSAP -v` — all pass.

- [ ] **Step 1.4:** Commit.

```bash
git add x/cs/keeper/msg_create_sap_test.go
git commit -m "test(cs): add fixture-based CreateSchemaAuthorizationPolicy tests (step 14)"
```

---

## Task 2: Delete old CreateSAP tests from `msg_schema_authorization_policy_test.go`

- [ ] **Step 2.1:** Remove the following functions from `msg_schema_authorization_policy_test.go`:
  - `TestCreateSchemaAuthorizationPolicy_HappyPath`
  - `TestCreateSchemaAuthorizationPolicy_VersionIncrement`
  - `TestCreateSchemaAuthorizationPolicy_SchemaNotFound`
  - `TestCreateSchemaAuthorizationPolicy_WrongCorporation`
  
  Also remove the `validJsonSchemaForPolicy` constant if it is no longer referenced in the file.

- [ ] **Step 2.2:** Run `go test ./x/cs/keeper/... -v` — all pass.

- [ ] **Step 2.3:** Commit.

```bash
git add x/cs/keeper/msg_schema_authorization_policy_test.go
git commit -m "test(cs): delete legacy CreateSchemaAuthorizationPolicy tests"
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
git push -u origin test/step-14-cs-create-sap
gh pr create \
  --title "test(cs): fixture-based CreateSchemaAuthorizationPolicy tests (issue #292 step 14)" \
  --body "$(cat <<'EOF'
## Summary
- Add x/cs/keeper/msg_create_sap_test.go — full happy path (struct + event + invariant) + version increment verification + 3 negative cases
- Asserts [MOD-CS-MSG-5-3]: effective_from and effective_until are nil at creation
- Delete legacy TestCreateSchemaAuthorizationPolicy_* from msg_schema_authorization_policy_test.go

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

- [ ] Happy path: full struct assertion including `EffectiveFrom=nil`, `EffectiveUntil=nil`, `Revoked=false`, `Version=1`, event asserted, invariant passes
- [ ] Version increment: second SAP for same `(schema_id, role)` has version 2; SAP for different role starts at version 1
- [ ] Pending state explicitly tested: `EffectiveFrom` and `EffectiveUntil` are nil at creation
- [ ] Negative cases: AUTHZ fail, schema not found, corp mismatch
- [ ] On errors: zero SAPs written, no event emitted
- [ ] Old `TestCreateSchemaAuthorizationPolicy_*` tests deleted
- [ ] `go test`, `go vet`, `golangci-lint` all pass
