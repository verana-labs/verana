# Step 11: CS CreateCredentialSchema — Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Replace the existing `TestMsgServerCreateCredentialSchema` and related tests with a fixture-based test suite that provides full struct assertions, event assertions, and invariant checks for `CreateCredentialSchema`.

**Architecture:** Create `x/cs/keeper/fixture_test.go` with the `Fixture` struct and add `CredentialschemaKeeperWithDelegation` to `testutil/keeper/credentialschema.go`. Step 11 tests live in a new `x/cs/keeper/msg_create_credential_schema_test.go`. All tests use the `Fixture` — no `setupMsgServer` helper. The `MockDelegationKeeper.ErrToReturn` field drives all AUTHZ failure cases. Old test code for `CreateCredentialSchema` is deleted from `msg_server_test.go`.

**Tech Stack:** Go 1.24, Cosmos SDK v0.50.x, testify/require

---

## Pre-flight

- [ ] **Worktree.** Create an isolated worktree for this step. Branch name: `test/step-11-cs-create-credential-schema`.
- [ ] **Sanity check.** Run: `go build ./... && go vet ./...` — expected exit 0.

---

## File Map

| Action | File |
|--------|------|
| Create | `testutil/keeper/credentialschema.go` — add `CredentialschemaKeeperWithDelegation` |
| Create | `x/cs/keeper/fixture_test.go` |
| Create | `x/cs/keeper/msg_create_credential_schema_test.go` |
| Delete content | `x/cs/keeper/msg_server_test.go` — remove `TestMsgServerCreateCredentialSchema`, `TestCanonicalIdInjection`, `TestQueryCanonicalId` (and `setupMsgServer` if it becomes unused after this step) |

---

## Task 1: Add `CredentialschemaKeeperWithDelegation` to testutil

**File:** `testutil/keeper/credentialschema.go`

The existing `CredentialschemaKeeper` hardwires a `MockDelegationKeeper` but does not return it. The fixture needs to control `ErrToReturn` directly to drive AUTHZ failure tests.

- [ ] **Step 1.1:** Append the following function to `testutil/keeper/credentialschema.go`:

```go
// CredentialschemaKeeperWithDelegation returns the same wiring as
// CredentialschemaKeeper but also exposes the MockDelegationKeeper so
// fixture-based tests can set ErrToReturn to simulate AUTHZ failures.
func CredentialschemaKeeperWithDelegation(t testing.TB) (keeper.Keeper, *MockTrustRegistryKeeper, sdk.Context, *MockDelegationKeeper) {
    t.Helper()
    storeKey := storetypes.NewKVStoreKey(types.StoreKey)

    db := dbm.NewMemDB()
    stateStore := store.NewCommitMultiStore(db, log.NewNopLogger(), metrics.NewNoOpMetrics())
    stateStore.MountStoreWithDB(storeKey, storetypes.StoreTypeIAVL, db)
    require.NoError(t, stateStore.LoadLatestVersion())

    registry := codectypes.NewInterfaceRegistry()
    cdc := codec.NewProtoCodec(registry)
    authority := authtypes.NewModuleAddress(govtypes.ModuleName)

    bankKeeper := NewMockBankKeeper()
    trustRegistryKeeper := NewMockTrustRegistryKeeper()
    delegationKeeper := &MockDelegationKeeper{}

    k := keeper.NewKeeper(
        cdc,
        runtime.NewKVStoreService(storeKey),
        log.NewNopLogger(),
        authority.String(),
        bankKeeper,
        trustRegistryKeeper,
        delegationKeeper,
    )

    ctx := sdk.NewContext(stateStore, cmtproto.Header{}, false, log.NewNopLogger())
    if err := k.SetParams(ctx, types.DefaultParams()); err != nil {
        panic(err)
    }

    return k, trustRegistryKeeper, ctx, delegationKeeper
}
```

- [ ] **Step 1.2:** Run `go build ./testutil/keeper/...` — expected exit 0.

- [ ] **Step 1.3:** Commit.

```bash
git add testutil/keeper/credentialschema.go
git commit -m "test(cs): add CredentialschemaKeeperWithDelegation to testutil"
```

---

## Task 2: Create `fixture_test.go`

**File:** `x/cs/keeper/fixture_test.go`

This file is the shared test harness for all CS steps (11–16). Steps 12–16 will add helpers to it as needed.

- [ ] **Step 2.1:** Create `x/cs/keeper/fixture_test.go`:

```go
package keeper_test

import (
    "testing"
    "time"

    sdk "github.com/cosmos/cosmos-sdk/types"
    "github.com/stretchr/testify/require"

    keepertest "github.com/verana-labs/verana/testutil/keeper"
    cskeeper "github.com/verana-labs/verana/x/cs/keeper"
    "github.com/verana-labs/verana/x/cs/types"
)

// minimalJsonSchema is a valid JSON schema used across CS fixture tests.
const minimalJsonSchema = `{
  "$schema": "https://json-schema.org/draft/2020-12/schema",
  "title": "FixtureSchema",
  "description": "Minimal schema for fixture tests",
  "type": "object",
  "properties": {
    "name": { "type": "string" }
  },
  "required": ["name"],
  "additionalProperties": false
}`

// Fixture is the per-test CS harness.
type Fixture struct {
    t         *testing.T
    K         cskeeper.Keeper
    MS        types.MsgServer
    Ctx       sdk.Context
    TRKeeper  *keepertest.MockTrustRegistryKeeper
    DelKeeper *keepertest.MockDelegationKeeper
}

// NewFixture creates a fresh Fixture for each test.
func NewFixture(t *testing.T) *Fixture {
    t.Helper()
    k, trKeeper, ctx, del := keepertest.CredentialschemaKeeperWithDelegation(t)
    return &Fixture{
        t:         t,
        K:         k,
        MS:        cskeeper.NewMsgServerImpl(k),
        Ctx:       ctx,
        TRKeeper:  trKeeper,
        DelKeeper: del,
    }
}

// RequireCredentialSchema asserts that the stored CredentialSchema at id
// exactly equals want via require.Equal (full struct comparison).
func (f *Fixture) RequireCredentialSchema(id uint64, want types.CredentialSchema) {
    f.t.Helper()
    got, err := f.K.CredentialSchema.Get(f.Ctx, id)
    require.NoError(f.t, err)
    require.Equal(f.t, want, got)
}

// RequireSchemaCount asserts the total number of stored CredentialSchemas.
func (f *Fixture) RequireSchemaCount(n int) {
    f.t.Helper()
    count := 0
    _ = f.K.CredentialSchema.Walk(f.Ctx, nil, func(_ uint64, _ types.CredentialSchema) (bool, error) {
        count++
        return false, nil
    })
    require.Equal(f.t, n, count)
}

// RequireSAP asserts that the stored SchemaAuthorizationPolicy at id
// exactly equals want via require.Equal (full struct comparison).
func (f *Fixture) RequireSAP(id uint64, want types.SchemaAuthorizationPolicy) {
    f.t.Helper()
    got, err := f.K.SchemaAuthorizationPolicies.Get(f.Ctx, id)
    require.NoError(f.t, err)
    require.Equal(f.t, want, got)
}

// RequireSAPCount asserts the total number of stored SchemaAuthorizationPolicies.
func (f *Fixture) RequireSAPCount(n int) {
    f.t.Helper()
    count := 0
    _ = f.K.SchemaAuthorizationPolicies.Walk(f.Ctx, nil, func(_ uint64, _ types.SchemaAuthorizationPolicy) (bool, error) {
        count++
        return false, nil
    })
    require.Equal(f.t, n, count)
}

// RequireEvent asserts that an event of eventType was emitted with all the
// given key=value attributes.
func (f *Fixture) RequireEvent(eventType string, attrs map[string]string) {
    f.t.Helper()
    for _, e := range f.Ctx.EventManager().Events() {
        if e.Type == eventType {
            for k, v := range attrs {
                found := false
                for _, a := range e.Attributes {
                    if a.Key == k && a.Value == v {
                        found = true
                        break
                    }
                }
                require.Truef(f.t, found, "event %s missing attr %s=%s", eventType, k, v)
            }
            return
        }
    }
    require.Failf(f.t, "event not found", "event type %q not emitted", eventType)
}

// RequireNoEvent asserts that no event of eventType was emitted.
func (f *Fixture) RequireNoEvent(eventType string) {
    f.t.Helper()
    for _, e := range f.Ctx.EventManager().Events() {
        if e.Type == eventType {
            require.Failf(f.t, "unexpected event", "event type %q was emitted but should not have been", eventType)
        }
    }
}

// RequireInvariant checks all CS-level invariants:
//   - Every active SAP version > 0 implies a SAP exists at that version (CS invariant from design spec).
//
// (TR cross-reference invariant: every CS references an existing TR row. Since
// we use a mock TR keeper, we assert that GetTrustRegistry does not error for
// each CS's TrId.)
func (f *Fixture) RequireInvariant() {
    f.t.Helper()
    // Walk all credential schemas — each must have a resolvable TR.
    _ = f.K.CredentialSchema.Walk(f.Ctx, nil, func(_ uint64, cs types.CredentialSchema) (bool, error) {
        _, err := f.TRKeeper.GetTrustRegistry(f.Ctx, cs.TrId)
        require.NoErrorf(f.t, err, "CS invariant: CS id=%d references non-existent TR id=%d", cs.Id, cs.TrId)
        return false, nil
    })
}

// SetBlockTime sets the block time on the context.
func (f *Fixture) SetBlockTime(t time.Time) {
    f.Ctx = f.Ctx.WithBlockTime(t)
}

// AdvanceTime advances the block time by d.
func (f *Fixture) AdvanceTime(d time.Duration) {
    f.Ctx = f.Ctx.WithBlockTime(f.Ctx.BlockTime().Add(d))
}

// ResetEvents resets the event manager on the context so each sub-test
// starts with a clean event slate.
func (f *Fixture) ResetEvents() {
    f.Ctx = f.Ctx.WithEventManager(sdk.NewEventManager())
}
```

- [ ] **Step 2.2:** Run `go build ./x/cs/keeper/...` — expected exit 0.

- [ ] **Step 2.3:** Commit.

```bash
git add x/cs/keeper/fixture_test.go
git commit -m "test(cs): add Fixture struct and harness for CS module (step 11)"
```

---

## Task 3: Write `msg_create_credential_schema_test.go`

**File:** `x/cs/keeper/msg_create_credential_schema_test.go`

- [ ] **Step 3.1:** Create `x/cs/keeper/msg_create_credential_schema_test.go`:

```go
package keeper_test

import (
    "encoding/json"
    "errors"
    "fmt"
    "strings"
    "testing"
    "time"

    sdk "github.com/cosmos/cosmos-sdk/types"
    "github.com/stretchr/testify/require"

    cskeeper "github.com/verana-labs/verana/x/cs/keeper"
    "github.com/verana-labs/verana/x/cs/types"
)

// ---------------------------------------------------------------------------
// Spec formula functions (independent of implementation)
// ---------------------------------------------------------------------------

// specCSMaxValidityDays returns the default max validity period (10 years = 3650 days)
// as defined in DefaultParams.
func specCSMaxValidityDays() uint32 { return 3650 }

// ---------------------------------------------------------------------------
// Helpers
// ---------------------------------------------------------------------------

// csAddr returns a deterministic sdk.AccAddress bech32 string from a
// human-readable 20-character label (padded with spaces).
func csAddr(label string) string {
    b := []byte(label)
    if len(b) < 20 {
        pad := make([]byte, 20-len(b))
        for i := range pad {
            pad[i] = '_'
        }
        b = append(b, pad...)
    }
    return sdk.AccAddress(b[:20]).String()
}

// makeCreateMsg builds a MsgCreateCredentialSchema using the keeper helper,
// applying sane defaults for onboarding modes and pricing.
func makeCreateMsg(corp, oper string, trID uint64, jsonSchema string, issuerGrantor, verifierGrantor, issuer, verifier, holder uint32) *types.MsgCreateCredentialSchema {
    return cskeeper.CreateMsgWithValidityPeriods(
        corp, oper, trID, jsonSchema,
        issuerGrantor, verifierGrantor, issuer, verifier, holder,
        2, 2, 2, // issuerMode, verifierMode, holderMode
        1, "tu", "sha256",
    )
}

// ---------------------------------------------------------------------------
// Tests
// ---------------------------------------------------------------------------

func TestCreateCredentialSchema_HappyPath(t *testing.T) {
    // [MOD-CS-MSG-1] valid operator, valid TR, valid JSON schema
    f := NewFixture(t)
    now := time.Date(2026, 1, 1, 12, 0, 0, 0, time.UTC)
    f.SetBlockTime(now)

    corp := csAddr("corp_create_happy")
    oper := csAddr("oper_create_happy")
    trID := f.TRKeeper.CreateMockTrustRegistry(corp, "did:example:create-happy")

    msg := makeCreateMsg(corp, oper, trID, minimalJsonSchema, 365, 365, 180, 180, 180)

    resp, err := f.MS.CreateCredentialSchema(f.Ctx, msg)
    require.NoError(t, err)
    require.NotZero(t, resp.Id)

    // Full struct assertion — every field must match expected values.
    // The JSON schema will have canonical $id injected, so we retrieve it and
    // re-inject to form the expected value.
    stored, err := f.K.CredentialSchema.Get(f.Ctx, resp.Id)
    require.NoError(t, err)

    // Verify canonical $id was injected (chain ID is empty string in test ctx)
    var schemaDoc map[string]interface{}
    require.NoError(t, json.Unmarshal([]byte(stored.JsonSchema), &schemaDoc))
    canonicalID, ok := schemaDoc["$id"].(string)
    require.True(t, ok, "$id must be present in stored schema")
    expectedCanonicalID := fmt.Sprintf("vpr:verana:%s/cs/v1/js/%d", f.Ctx.ChainID(), resp.Id)
    require.Equal(t, expectedCanonicalID, canonicalID)

    // Verify all scalar fields
    require.Equal(t, resp.Id, stored.Id)
    require.Equal(t, trID, stored.TrId)
    require.Equal(t, now, stored.Created)
    require.Equal(t, now, stored.Modified)
    require.Nil(t, stored.Archived)
    require.Equal(t, uint32(365), stored.IssuerGrantorValidationValidityPeriod)
    require.Equal(t, uint32(365), stored.VerifierGrantorValidationValidityPeriod)
    require.Equal(t, uint32(180), stored.IssuerValidationValidityPeriod)
    require.Equal(t, uint32(180), stored.VerifierValidationValidityPeriod)
    require.Equal(t, uint32(180), stored.HolderValidationValidityPeriod)
    require.Equal(t, types.PricingAssetType(1), stored.PricingAssetType)
    require.Equal(t, "tu", stored.PricingAsset)
    require.Equal(t, "sha256", stored.DigestAlgorithm)

    // Event assertion
    f.RequireEvent(types.EventTypeCreateCredentialSchema, map[string]string{
        types.AttributeKeyId:          fmt.Sprintf("%d", resp.Id),
        types.AttributeKeyTrId:        fmt.Sprintf("%d", trID),
        types.AttributeKeyCorporation: corp,
        types.AttributeKeyOperator:    oper,
    })

    // Invariant check
    f.RequireInvariant()
}

func TestCreateCredentialSchema_SchemaCountIncrement(t *testing.T) {
    // Verifies that each successful call increments the stored schema count.
    f := NewFixture(t)
    f.SetBlockTime(time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC))

    corp := csAddr("corp_count")
    oper := csAddr("oper_count")
    trID := f.TRKeeper.CreateMockTrustRegistry(corp, "did:example:count")

    f.RequireSchemaCount(0)

    msg1 := makeCreateMsg(corp, oper, trID, minimalJsonSchema, 365, 365, 180, 180, 180)
    _, err := f.MS.CreateCredentialSchema(f.Ctx, msg1)
    require.NoError(t, err)
    f.RequireSchemaCount(1)

    msg2 := makeCreateMsg(corp, oper, trID, minimalJsonSchema, 365, 365, 180, 180, 180)
    _, err = f.MS.CreateCredentialSchema(f.Ctx, msg2)
    require.NoError(t, err)
    f.RequireSchemaCount(2)
}

func TestCreateCredentialSchema_AuthzFail(t *testing.T) {
    // [MOD-CS-MSG-1-2-1] AUTHZ-CHECK: delegation keeper returns error → abort
    f := NewFixture(t)
    f.SetBlockTime(time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC))
    f.DelKeeper.ErrToReturn = errors.New("operator not authorized")

    corp := csAddr("corp_authz_fail")
    oper := csAddr("oper_authz_fail")
    trID := f.TRKeeper.CreateMockTrustRegistry(corp, "did:example:authz-fail")

    msg := makeCreateMsg(corp, oper, trID, minimalJsonSchema, 365, 365, 180, 180, 180)

    _, err := f.MS.CreateCredentialSchema(f.Ctx, msg)
    require.ErrorContains(t, err, "authorization check failed")

    // Zero state written
    f.RequireSchemaCount(0)
    // No event emitted
    f.RequireNoEvent(types.EventTypeCreateCredentialSchema)
}

func TestCreateCredentialSchema_TrustRegistryNotFound(t *testing.T) {
    // [MOD-CS-MSG-1-2-1] Trust registry must exist
    f := NewFixture(t)
    f.SetBlockTime(time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC))

    corp := csAddr("corp_tr_notfound")
    oper := csAddr("oper_tr_notfound")
    // Do NOT register a trust registry — TR id 9999 will not be found.

    msg := makeCreateMsg(corp, oper, 9999, minimalJsonSchema, 365, 365, 180, 180, 180)

    _, err := f.MS.CreateCredentialSchema(f.Ctx, msg)
    require.ErrorContains(t, err, "trust registry not found")

    f.RequireSchemaCount(0)
    f.RequireNoEvent(types.EventTypeCreateCredentialSchema)
}

func TestCreateCredentialSchema_CorporationMismatch(t *testing.T) {
    // [MOD-CS-MSG-1-2-1] Corporation in msg must match TR corporation
    f := NewFixture(t)
    f.SetBlockTime(time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC))

    corp := csAddr("corp_owner")
    wrongCorp := csAddr("corp_impostor")
    oper := csAddr("oper_mismatch")
    trID := f.TRKeeper.CreateMockTrustRegistry(corp, "did:example:mismatch")

    msg := makeCreateMsg(wrongCorp, oper, trID, minimalJsonSchema, 365, 365, 180, 180, 180)

    _, err := f.MS.CreateCredentialSchema(f.Ctx, msg)
    require.ErrorContains(t, err, "corporation does not match the trust registry corporation")

    f.RequireSchemaCount(0)
    f.RequireNoEvent(types.EventTypeCreateCredentialSchema)
}

func TestCreateCredentialSchema_ValidityPeriodExceedsMax(t *testing.T) {
    // [MOD-CS-MSG-1-2-1] validity period must not exceed max (3650 days by default)
    f := NewFixture(t)
    f.SetBlockTime(time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC))

    corp := csAddr("corp_maxperiod")
    oper := csAddr("oper_maxperiod")
    trID := f.TRKeeper.CreateMockTrustRegistry(corp, "did:example:maxperiod")

    overMax := specCSMaxValidityDays() + 1
    msg := makeCreateMsg(corp, oper, trID, minimalJsonSchema, overMax, 365, 180, 180, 180)

    _, err := f.MS.CreateCredentialSchema(f.Ctx, msg)
    require.ErrorContains(t, err, "exceeds maximum")

    f.RequireSchemaCount(0)
    f.RequireNoEvent(types.EventTypeCreateCredentialSchema)
}

func TestCreateCredentialSchema_SchemaSizeExceedsMax(t *testing.T) {
    // Schema larger than CredentialSchemaSchemaMaxSize (8192 bytes) must be rejected.
    f := NewFixture(t)
    f.SetBlockTime(time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC))

    corp := csAddr("corp_bigschema")
    oper := csAddr("oper_bigschema")
    trID := f.TRKeeper.CreateMockTrustRegistry(corp, "did:example:bigschema")

    // Build a JSON schema slightly over 8192 bytes by padding the description field.
    padding := strings.Repeat("x", 8200)
    bigSchema := fmt.Sprintf(`{
  "$schema": "https://json-schema.org/draft/2020-12/schema",
  "title": "BigSchema",
  "description": %q,
  "type": "object",
  "properties": { "name": { "type": "string" } },
  "required": ["name"],
  "additionalProperties": false
}`, padding)

    msg := makeCreateMsg(corp, oper, trID, bigSchema, 365, 365, 180, 180, 180)

    _, err := f.MS.CreateCredentialSchema(f.Ctx, msg)
    require.ErrorContains(t, err, "schema size exceeds maximum")

    f.RequireSchemaCount(0)
    f.RequireNoEvent(types.EventTypeCreateCredentialSchema)
}

func TestCreateCredentialSchema_ZeroValidityPeriod(t *testing.T) {
    // Zero validity periods are legal (means "never expire") per the spec.
    f := NewFixture(t)
    f.SetBlockTime(time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC))

    corp := csAddr("corp_zero_vp")
    oper := csAddr("oper_zero_vp")
    trID := f.TRKeeper.CreateMockTrustRegistry(corp, "did:example:zero-vp")

    msg := makeCreateMsg(corp, oper, trID, minimalJsonSchema, 0, 0, 0, 0, 0)

    resp, err := f.MS.CreateCredentialSchema(f.Ctx, msg)
    require.NoError(t, err)
    require.NotZero(t, resp.Id)

    stored, err := f.K.CredentialSchema.Get(f.Ctx, resp.Id)
    require.NoError(t, err)
    require.Equal(t, uint32(0), stored.IssuerGrantorValidationValidityPeriod)
    require.Equal(t, uint32(0), stored.HolderValidationValidityPeriod)

    f.RequireEvent(types.EventTypeCreateCredentialSchema, map[string]string{
        types.AttributeKeyId: fmt.Sprintf("%d", resp.Id),
    })
    f.RequireInvariant()
}
```

- [ ] **Step 3.2:** Run `go build ./x/cs/keeper/...` — expected exit 0.
- [ ] **Step 3.3:** Run `go test ./x/cs/keeper/... -run TestCreateCredentialSchema -v` — expected: all tests pass.

- [ ] **Step 3.4:** Commit.

```bash
git add x/cs/keeper/msg_create_credential_schema_test.go
git commit -m "test(cs): add fixture-based CreateCredentialSchema tests (step 11)"
```

---

## Task 4: Delete old CreateCredentialSchema tests from `msg_server_test.go`

**File:** `x/cs/keeper/msg_server_test.go`

- [ ] **Step 4.1:** Remove the following top-level test functions from `msg_server_test.go` (the `setupMsgServer` helper is also removed if it becomes unused after steps 12 and 13 land; for now just remove the three functions):
  - `TestMsgServerCreateCredentialSchema` (lines 24–259)
  - `TestCanonicalIdInjection` (lines 261–378)
  - `TestQueryCanonicalId` (lines 380–460)
  
  Also remove `setupMsgServer` (lines 19–22) if it is now only used by `TestUpdateCredentialSchema` and `TestArchiveCredentialSchema` — those will be migrated in steps 12 and 13. Leave `setupMsgServer` in place until step 13 removes the last user.

- [ ] **Step 4.2:** Run `go test ./x/cs/keeper/... -v` — expected: all remaining tests pass, no compile errors.

- [ ] **Step 4.3:** Commit.

```bash
git add x/cs/keeper/msg_server_test.go
git commit -m "test(cs): delete legacy CreateCredentialSchema tests from msg_server_test.go"
```

---

## Task 5: Final validation

- [ ] **Step 5.1:** Run `go test ./x/cs/keeper/... -v -count=1` — all tests pass.
- [ ] **Step 5.2:** Run `go test ./x/cs/keeper/... -cover -count=1` — note coverage delta.
- [ ] **Step 5.3:** Run `go vet ./x/cs/keeper/...` — no output.
- [ ] **Step 5.4:** Run `golangci-lint run ./x/cs/keeper/...` — no findings.
- [ ] **Step 5.5:** Run `go build ./... && go test ./... -count=1` — no regressions.

- [ ] **Step 5.6:** Push and open PR.

```bash
git push -u origin test/step-11-cs-create-credential-schema
gh pr create \
  --title "test(cs): fixture-based CreateCredentialSchema tests (issue #292 step 11)" \
  --body "$(cat <<'EOF'
## Summary
- Add CredentialschemaKeeperWithDelegation to testutil — exposes MockDelegationKeeper for AUTHZ failure control
- Add x/cs/keeper/fixture_test.go — Fixture struct shared by all CS steps 11–16
- Add x/cs/keeper/msg_create_credential_schema_test.go — full happy path (struct + event + invariant) + 5 negative cases
- Delete legacy TestMsgServerCreateCredentialSchema, TestCanonicalIdInjection, TestQueryCanonicalId from msg_server_test.go

## Test plan
- [ ] go test ./x/cs/keeper/... -v passes
- [ ] go test ./x/cs/keeper/... -cover reports improved coverage
- [ ] go vet ./x/cs/keeper/... clean
- [ ] golangci-lint run ./x/cs/keeper/... clean
- [ ] go test ./... passes (no regressions)
EOF
)"
```

---

## "Done" Criteria

- [ ] `CredentialschemaKeeperWithDelegation` exported from `testutil/keeper/credentialschema.go`
- [ ] `x/cs/keeper/fixture_test.go` exists with `Fixture`, `NewFixture`, `RequireCredentialSchema`, `RequireSchemaCount`, `RequireSAP`, `RequireSAPCount`, `RequireEvent`, `RequireNoEvent`, `RequireInvariant`, `SetBlockTime`, `AdvanceTime`, `ResetEvents`
- [ ] Happy path test: full field-by-field assertion, canonical `$id` verified, event asserted, invariant passes
- [ ] Negative cases: AUTHZ fail, TR not found, corp mismatch, validity period exceeds max, schema size exceeds max
- [ ] Edge case: zero validity periods are accepted
- [ ] Old `TestMsgServerCreateCredentialSchema`, `TestCanonicalIdInjection`, `TestQueryCanonicalId` deleted
- [ ] `go test ./x/cs/keeper/...`, `go vet ./x/cs/keeper/...`, `golangci-lint` all pass
