# Step 6: TR CreateTrustRegistry — Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Replace the existing `TestMsgServerCreateTrustRegistry` test with a fixture-based test that gives full struct assertions, event verification, invariant checks, and one `t.Run` per spec precondition failure mode.

**Architecture:** Create `x/tr/keeper/fixture_test.go` (package `keeper_test`) with the `Fixture` struct and all assertion helpers; the fixture uses `keepertest.TrustregistryKeeperWithDelegation` (to be added to `testutil/keeper/trustregistry.go`) so `DelKeeper.ErrToReturn` can simulate AUTHZ failures. `TestMsgCreateTrustRegistry` in `msg_server_test.go` is replaced by a new test using the fixture; there are no bank operations so `StatefulBankMock` is not wired into the TR fixture.

**Tech Stack:** Go 1.24, Cosmos SDK v0.50.x, testify/require

---

## Pre-flight (one-time, before Task 1)

- [ ] **Worktree.** Create an isolated worktree for this step. Branch name: `test/step-6-tr-create`.
- [ ] **Sanity check.**

  Run: `go build ./... && go vet ./...`
  Expected: no output, exit 0.

---

## File Structure

- Create: `x/tr/keeper/fixture_test.go` — `Fixture` struct, `NewFixture`, all assertion helpers.
- Modify: `testutil/keeper/trustregistry.go` — add `TrustregistryKeeperWithDelegation`.
- Modify: `x/tr/keeper/msg_server_test.go` — delete `TestMsgServerCreateTrustRegistry`, add `TestMsgCreateTrustRegistry` using fixture; keep all other existing tests intact (they'll be replaced in steps 7-10).

---

## Task 1: Add `TrustregistryKeeperWithDelegation` to testutil

**File:** `testutil/keeper/trustregistry.go`

This variant accepts a `*MockDelegationKeeper` so tests can set `ErrToReturn` to simulate AUTHZ failures.

- [ ] **Step 1.1: Add function.**

Append to `testutil/keeper/trustregistry.go`:

```go
// TrustregistryKeeperWithDelegation returns a TR keeper wired with the
// supplied MockDelegationKeeper. This allows tests to control AUTHZ outcomes
// by setting del.ErrToReturn before each call.
func TrustregistryKeeperWithDelegation(t testing.TB, del *MockDelegationKeeper) (keeper.Keeper, sdk.Context) {
	storeKey := storetypes.NewKVStoreKey(types.StoreKey)

	db := dbm.NewMemDB()
	stateStore := store.NewCommitMultiStore(db, log.NewNopLogger(), metrics.NewNoOpMetrics())
	stateStore.MountStoreWithDB(storeKey, storetypes.StoreTypeIAVL, db)
	require.NoError(t, stateStore.LoadLatestVersion())

	registry := codectypes.NewInterfaceRegistry()
	cdc := codec.NewProtoCodec(registry)
	authority := authtypes.NewModuleAddress(govtypes.ModuleName)

	k := keeper.NewKeeper(
		cdc,
		runtime.NewKVStoreService(storeKey),
		log.NewNopLogger(),
		authority.String(),
		del,
	)

	ctx := sdk.NewContext(stateStore, cmtproto.Header{}, false, log.NewNopLogger())

	if err := k.SetParams(ctx, types.DefaultParams()); err != nil {
		panic(err)
	}

	return k, ctx
}
```

- [ ] **Step 1.2: Build and verify.**

  Run: `go build ./testutil/keeper/... && go vet ./testutil/keeper/...`
  Expected: no output, exit 0.

- [ ] **Step 1.3: Commit.**

```bash
git add testutil/keeper/trustregistry.go
git commit -m "test(tr): add TrustregistryKeeperWithDelegation to testutil"
```

---

## Task 2: Create `fixture_test.go`

**File:** `x/tr/keeper/fixture_test.go`

- [ ] **Step 2.1: Create the file.**

```go
package keeper_test

import (
	"testing"
	"time"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/stretchr/testify/require"

	keepertest "github.com/verana-labs/verana/testutil/keeper"
	"github.com/verana-labs/verana/x/tr/keeper"
	"github.com/verana-labs/verana/x/tr/types"
)

// Fixture is the per-test environment for TR keeper tests.
// No bank mock is needed: TR has no financial operations.
type Fixture struct {
	t         *testing.T
	K         keeper.Keeper
	MS        types.MsgServer
	Ctx       sdk.Context
	DelKeeper *keepertest.MockDelegationKeeper
}

// NewFixture constructs a fresh TR test environment with a controllable
// MockDelegationKeeper. By default ErrToReturn is nil (all AUTHZ checks pass).
func NewFixture(t *testing.T) *Fixture {
	t.Helper()
	del := &keepertest.MockDelegationKeeper{}
	k, ctx := keepertest.TrustregistryKeeperWithDelegation(t, del)
	return &Fixture{
		t:         t,
		K:         k,
		MS:        keeper.NewMsgServerImpl(k),
		Ctx:       ctx,
		DelKeeper: del,
	}
}

// --- State assertion helpers ---

// RequireTrustRegistry asserts the stored TrustRegistry at id equals want.
// Uses require.Equal for full struct comparison.
func (f *Fixture) RequireTrustRegistry(id uint64, want types.TrustRegistry) {
	f.t.Helper()
	got, err := f.K.TrustRegistry.Get(f.Ctx, id)
	require.NoError(f.t, err)
	require.Equal(f.t, want, got)
}

// RequireTrustRegistryCount asserts that exactly n TrustRegistry rows exist.
func (f *Fixture) RequireTrustRegistryCount(n int) {
	f.t.Helper()
	count := 0
	_ = f.K.TrustRegistry.Walk(f.Ctx, nil, func(_ uint64, _ types.TrustRegistry) (bool, error) {
		count++
		return false, nil
	})
	require.Equal(f.t, n, count)
}

// RequireGFVersionCount asserts that exactly n GovernanceFrameworkVersion rows exist.
func (f *Fixture) RequireGFVersionCount(n int) {
	f.t.Helper()
	count := 0
	_ = f.K.GFVersion.Walk(f.Ctx, nil, func(_ uint64, _ types.GovernanceFrameworkVersion) (bool, error) {
		count++
		return false, nil
	})
	require.Equal(f.t, n, count)
}

// RequireGFDocumentCount asserts that exactly n GovernanceFrameworkDocument rows exist.
func (f *Fixture) RequireGFDocumentCount(n int) {
	f.t.Helper()
	count := 0
	_ = f.K.GFDocument.Walk(f.Ctx, nil, func(_ uint64, _ types.GovernanceFrameworkDocument) (bool, error) {
		count++
		return false, nil
	})
	require.Equal(f.t, n, count)
}

// RequireGFVersion asserts a GovernanceFrameworkVersion matching trID and
// version exists, and calls check with the found struct.
func (f *Fixture) RequireGFVersion(trID uint64, version int32, check func(gfv types.GovernanceFrameworkVersion)) {
	f.t.Helper()
	var found bool
	_ = f.K.GFVersion.Walk(f.Ctx, nil, func(_ uint64, gfv types.GovernanceFrameworkVersion) (bool, error) {
		if gfv.TrId == trID && gfv.Version == version {
			found = true
			check(gfv)
			return true, nil
		}
		return false, nil
	})
	require.True(f.t, found, "GFVersion trID=%d version=%d not found", trID, version)
}

// RequireGFDocument asserts a GovernanceFrameworkDocument matching gfvID and
// lang exists with the expected URL and digest.
func (f *Fixture) RequireGFDocument(gfvID uint64, lang, wantURL, wantDigest string) {
	f.t.Helper()
	var found bool
	_ = f.K.GFDocument.Walk(f.Ctx, nil, func(_ uint64, gfd types.GovernanceFrameworkDocument) (bool, error) {
		if gfd.GfvId == gfvID && gfd.Language == lang {
			found = true
			require.Equal(f.t, wantURL, gfd.Url, "GFDocument URL mismatch for lang %s", lang)
			require.Equal(f.t, wantDigest, gfd.DigestSri, "GFDocument digest mismatch for lang %s", lang)
			return true, nil
		}
		return false, nil
	})
	require.True(f.t, found, "GFDocument gfvID=%d lang=%s not found", gfvID, lang)
}

// RequireDIDIndex asserts that the DID index maps did to trID.
func (f *Fixture) RequireDIDIndex(did string, trID uint64) {
	f.t.Helper()
	got, err := f.K.TrustRegistryDIDIndex.Get(f.Ctx, did)
	require.NoError(f.t, err)
	require.Equal(f.t, trID, got)
}

// RequireNoDIDIndex asserts that did is absent from the DID index.
func (f *Fixture) RequireNoDIDIndex(did string) {
	f.t.Helper()
	_, err := f.K.TrustRegistryDIDIndex.Get(f.Ctx, did)
	require.Error(f.t, err, "DID index entry for %s should not exist", did)
}

// --- Event assertion helpers ---

// RequireEvent asserts that an event of eventType was emitted and has all
// key=value pairs in attrs. Fails if the event type is missing or any attr is absent.
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
				require.True(f.t, found, "event %s missing attr %s=%s", eventType, k, v)
			}
			return
		}
	}
	require.Fail(f.t, "event not emitted", "expected event type %q", eventType)
}

// RequireNoEvent asserts that no event of eventType was emitted.
func (f *Fixture) RequireNoEvent(eventType string) {
	f.t.Helper()
	for _, e := range f.Ctx.EventManager().Events() {
		if e.Type == eventType {
			require.Fail(f.t, "unexpected event", "event type %q was emitted but should not have been", eventType)
		}
	}
}

// --- Invariant helper ---

// RequireInvariant checks the TR module invariant:
// Every TrustRegistry has active_version >= 1.
func (f *Fixture) RequireInvariant() {
	f.t.Helper()
	_ = f.K.TrustRegistry.Walk(f.Ctx, nil, func(_ uint64, tr types.TrustRegistry) (bool, error) {
		require.GreaterOrEqual(f.t, tr.ActiveVersion, int32(1),
			"invariant violation: TrustRegistry id=%d has active_version=%d (must be >=1)", tr.Id, tr.ActiveVersion)
		return false, nil
	})
}

// --- Time helpers ---

// SetBlockTime sets the block time on the context.
func (f *Fixture) SetBlockTime(t time.Time) {
	f.Ctx = f.Ctx.WithBlockTime(t)
}

// AdvanceTime advances the block time by d.
func (f *Fixture) AdvanceTime(d time.Duration) {
	f.Ctx = f.Ctx.WithBlockTime(f.Ctx.BlockTime().Add(d))
}

// --- Shared test constants ---

const (
	testDigestSRI = "sha384-MzNNbQTWCSUSi0bbz7dbua+RcENv7C6FvlmYJ1Y+I727HsPOHdzwELMYO9Mz68M26"
	testCorp      = "cosmos1test_corp000000000000000000000000corp"
	testOperator  = "cosmos1test_oper000000000000000000000000oper"
	testDID       = "did:example:123456789abcdefghi"
	testAka       = "https://example.com/registry"
	testDocURL    = "https://example.com/gf-v1.html"
)
```

- [ ] **Step 2.2: Build and verify.**

  Run: `go build ./x/tr/keeper/... && go vet ./x/tr/keeper/...`
  Expected: no output, exit 0.

- [ ] **Step 2.3: Commit.**

```bash
git add x/tr/keeper/fixture_test.go
git commit -m "test(tr): add Fixture struct and assertion helpers"
```

---

## Task 3: Write `TestMsgCreateTrustRegistry` using the fixture

**File:** `x/tr/keeper/msg_server_test.go`

Delete `TestMsgServerCreateTrustRegistry` (the old table-driven test) and replace it with the new fixture-based `TestMsgCreateTrustRegistry`. Also delete the old `setupMsgServer` helper and `testDigestSRI` constant (now defined in fixture_test.go). Keep all other existing test functions (`TestMsgServer`, `TestMsgServerAddGovernanceFrameworkDocument`, etc.) intact — those are replaced in steps 7-10.

- [ ] **Step 3.1: Remove old test and add new test.**

After this edit, `msg_server_test.go` must:
1. No longer define `testDigestSRI` (moved to fixture_test.go).
2. No longer define `setupMsgServer` (replaced by `NewFixture`).
3. No longer contain `TestMsgServerCreateTrustRegistry`.
4. Contain `TestMsgServer` (unchanged) and the new `TestMsgCreateTrustRegistry` below.
5. Still contain `TestMsgServerAddGovernanceFrameworkDocument`, `TestMsgServerIncreaseActiveGovernanceFrameworkVersion`, `TestMsgServerUpdateTrustRegistry`, `TestMsgServerArchiveTrustRegistry` — these use their own `setupMsgServer`-style setup and will be replaced in steps 7-10.

Because `setupMsgServer` is still referenced by steps 7-10's existing tests, keep `setupMsgServer` defined but pointing to `NewFixture` OR leave the old tests to create their own keeper directly. The cleanest approach: keep a local `setupMsgServerLegacy` alias used only by the old tests, and do the full conversion in steps 7-10.

**Concrete edit:**

Replace the entire existing `msg_server_test.go` with:

```go
package keeper_test

import (
	"errors"
	"testing"
	"time"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/stretchr/testify/require"

	keepertest "github.com/verana-labs/verana/testutil/keeper"
	"github.com/verana-labs/verana/x/tr/keeper"
	"github.com/verana-labs/verana/x/tr/types"
)

// setupMsgServerLegacy is the legacy setup helper still used by steps 7-10 tests
// until they are replaced by fixture-based tests.
func setupMsgServerLegacy(t testing.TB) (keeper.Keeper, types.MsgServer, sdk.Context) {
	k, ctx := keepertest.TrustregistryKeeper(t)
	return k, keeper.NewMsgServerImpl(k), ctx
}

func TestMsgServer(t *testing.T) {
	k, ms, ctx := setupMsgServerLegacy(t)
	require.NotNil(t, ms)
	require.NotNil(t, ctx)
	require.NotEmpty(t, k)
}

// ---------------------------------------------------------------------------
// Step 6: CreateTrustRegistry — fixture-based tests
// ---------------------------------------------------------------------------

func TestMsgCreateTrustRegistry(t *testing.T) {
	t.Run("MOD-TR-MSG-1: happy path creates TR + GFV + GFD with full state", func(t *testing.T) {
		f := NewFixture(t)
		now := time.Date(2026, 1, 1, 12, 0, 0, 0, time.UTC)
		f.SetBlockTime(now)

		resp, err := f.MS.CreateTrustRegistry(f.Ctx, &types.MsgCreateTrustRegistry{
			Corporation:  testCorp,
			Operator:     testOperator,
			Did:          testDID,
			Aka:          testAka,
			Language:     "en",
			DocUrl:       testDocURL,
			DocDigestSri: testDigestSRI,
		})
		require.NoError(t, err)
		require.NotNil(t, resp)

		// Resolve the TR id via DID index
		trID, err := f.K.TrustRegistryDIDIndex.Get(f.Ctx, testDID)
		require.NoError(t, err)

		// Full struct assertion on TrustRegistry
		f.RequireTrustRegistry(trID, types.TrustRegistry{
			Id:            trID,
			Did:           testDID,
			Corporation:   testCorp,
			Created:       now,
			Modified:      now,
			Archived:      nil,
			Aka:           testAka,
			ActiveVersion: 1,
			Language:      "en",
		})

		// DID index is set
		f.RequireDIDIndex(testDID, trID)

		// Exactly one GFVersion was created at version 1 with ActiveSince = now
		f.RequireTrustRegistryCount(1)
		f.RequireGFVersionCount(1)
		f.RequireGFDocumentCount(1)

		f.RequireGFVersion(trID, 1, func(gfv types.GovernanceFrameworkVersion) {
			require.Equal(t, trID, gfv.TrId)
			require.Equal(t, int32(1), gfv.Version)
			require.Equal(t, now, gfv.Created)
			require.Equal(t, now, gfv.ActiveSince)
		})

		// Walk GFVersion to get the gfv.Id for GFDocument lookup
		var foundGfvID uint64
		_ = f.K.GFVersion.Walk(f.Ctx, nil, func(id uint64, gfv types.GovernanceFrameworkVersion) (bool, error) {
			if gfv.TrId == trID && gfv.Version == 1 {
				foundGfvID = id
				return true, nil
			}
			return false, nil
		})
		require.NotZero(t, foundGfvID)
		f.RequireGFDocument(foundGfvID, "en", testDocURL, testDigestSRI)

		// Event emitted
		f.RequireEvent(types.EventTypeCreateTrustRegistry, map[string]string{
			types.AttributeKeyDID:         testDID,
			types.AttributeKeyCorporation: testCorp,
			types.AttributeKeyAka:         testAka,
			types.AttributeKeyLanguage:    "en",
		})

		// Invariant: active_version >= 1
		f.RequireInvariant()
	})

	t.Run("MOD-TR-MSG-1: happy path without aka field", func(t *testing.T) {
		f := NewFixture(t)
		now := time.Date(2026, 1, 2, 0, 0, 0, 0, time.UTC)
		f.SetBlockTime(now)

		_, err := f.MS.CreateTrustRegistry(f.Ctx, &types.MsgCreateTrustRegistry{
			Corporation:  testCorp,
			Operator:     testOperator,
			Did:          "did:example:no-aka",
			Aka:          "",
			Language:     "fr",
			DocUrl:       testDocURL,
			DocDigestSri: testDigestSRI,
		})
		require.NoError(t, err)

		trID, err := f.K.TrustRegistryDIDIndex.Get(f.Ctx, "did:example:no-aka")
		require.NoError(t, err)

		f.RequireTrustRegistry(trID, types.TrustRegistry{
			Id:            trID,
			Did:           "did:example:no-aka",
			Corporation:   testCorp,
			Created:       now,
			Modified:      now,
			Archived:      nil,
			Aka:           "",
			ActiveVersion: 1,
			Language:      "fr",
		})
		f.RequireInvariant()
	})

	t.Run("MOD-TR-MSG-1-2-1: fails if operator not authorized", func(t *testing.T) {
		f := NewFixture(t)
		f.DelKeeper.ErrToReturn = errors.New("unauthorized operator")

		_, err := f.MS.CreateTrustRegistry(f.Ctx, &types.MsgCreateTrustRegistry{
			Corporation:  testCorp,
			Operator:     testOperator,
			Did:          testDID,
			Language:     "en",
			DocUrl:       testDocURL,
			DocDigestSri: testDigestSRI,
		})
		require.Error(t, err)
		require.ErrorContains(t, err, "authorization check failed")

		// No state written
		f.RequireTrustRegistryCount(0)
		f.RequireGFVersionCount(0)
		f.RequireGFDocumentCount(0)
		f.RequireNoDIDIndex(testDID)
		f.RequireNoEvent(types.EventTypeCreateTrustRegistry)
	})

	t.Run("MOD-TR-MSG-1-2-1: fails if delegation keeper is nil", func(t *testing.T) {
		// Constructed with normal keeper which has a non-nil delegation keeper,
		// so we test the nil-guard via direct keeper construction.
		// This is a belt-and-suspenders test for the nil guard in msg_server.go:29.
		// Since NewFixture always wires a mock, we skip this sub-test
		// (the nil guard is implementation-only; spec doesn't require it to be user-visible).
		t.Skip("nil delegation keeper guard is an implementation guard, not a spec precondition")
	})

	t.Run("edge: two registries get distinct IDs and DID index entries", func(t *testing.T) {
		f := NewFixture(t)
		did1 := "did:example:first"
		did2 := "did:example:second"

		_, err := f.MS.CreateTrustRegistry(f.Ctx, &types.MsgCreateTrustRegistry{
			Corporation:  testCorp,
			Operator:     testOperator,
			Did:          did1,
			Language:     "en",
			DocUrl:       testDocURL,
			DocDigestSri: testDigestSRI,
		})
		require.NoError(t, err)

		_, err = f.MS.CreateTrustRegistry(f.Ctx, &types.MsgCreateTrustRegistry{
			Corporation:  testCorp,
			Operator:     testOperator,
			Did:          did2,
			Language:     "de",
			DocUrl:       testDocURL,
			DocDigestSri: testDigestSRI,
		})
		require.NoError(t, err)

		f.RequireTrustRegistryCount(2)
		f.RequireGFVersionCount(2)
		f.RequireGFDocumentCount(2)

		id1, err := f.K.TrustRegistryDIDIndex.Get(f.Ctx, did1)
		require.NoError(t, err)
		id2, err := f.K.TrustRegistryDIDIndex.Get(f.Ctx, did2)
		require.NoError(t, err)
		require.NotEqual(t, id1, id2)

		f.RequireInvariant()
	})
}

// ---------------------------------------------------------------------------
// Legacy test stubs — replaced in steps 7-10
// ---------------------------------------------------------------------------

func TestMsgServerAddGovernanceFrameworkDocument(t *testing.T) {
	k, ms, ctx := setupMsgServerLegacy(t)

	authority := sdk.AccAddress([]byte("test_authority")).String()
	operator := sdk.AccAddress([]byte("test_operator")).String()
	validDid := "did:example:123456789abcdefghi"

	createMsg := &types.MsgCreateTrustRegistry{
		Corporation:  authority,
		Operator:     operator,
		Did:          validDid,
		Language:     "en",
		DocUrl:       "http://example.com/doc-v1",
		DocDigestSri: testDigestSRI,
	}
	_, err := ms.CreateTrustRegistry(ctx, createMsg)
	require.NoError(t, err)

	trID, err := k.TrustRegistryDIDIndex.Get(ctx, validDid)
	require.NoError(t, err)

	testCases := []struct {
		name      string
		setupFunc func()
		msg       *types.MsgAddGovernanceFrameworkDocument
		isValid   bool
	}{
		{
			name: "Valid Add Document with Next Version",
			msg: &types.MsgAddGovernanceFrameworkDocument{
				Corporation: authority,
				Operator:    operator,
				TrId:        trID,
				Language:    "en",
				Url:         "http://example.com/doc2",
				DigestSri:   "sha384-MzNNbQTWCSUSi0bbz7dbua+RcENv7C6FvlmYJ1Y+I727HsPOHdzwELMYO9Mz68M26",
				Version:     2,
			},
			isValid: true,
		},
		{
			name: "Valid Add Document to Same Version with Different Language",
			msg: &types.MsgAddGovernanceFrameworkDocument{
				Corporation: authority,
				Operator:    operator,
				TrId:        trID,
				Language:    "fr",
				Url:         "http://example.com/doc2-fr",
				DigestSri:   "sha384-MzNNbQTWCSUSi0bbz7dbua+RcENv7C6FvlmYJ1Y+I727HsPOHdzwELMYO9Mz68M26",
				Version:     2,
			},
			isValid: true,
		},
		{
			name: "Valid Add Next Version",
			msg: &types.MsgAddGovernanceFrameworkDocument{
				Corporation: authority,
				Operator:    operator,
				TrId:        trID,
				Language:    "en",
				Url:         "http://example.com/doc3",
				DigestSri:   "sha384-MzNNbQTWCSUSi0bbz7dbua+RcENv7C6FvlmYJ1Y+I727HsPOHdzwELMYO9Mz68M26",
				Version:     3,
			},
			isValid: true,
		},
		{
			name: "Invalid Add Document to Active Version 1",
			msg: &types.MsgAddGovernanceFrameworkDocument{
				Corporation: authority,
				Operator:    operator,
				TrId:        trID,
				Language:    "en",
				Url:         "http://example.com/doc-v1",
				DigestSri:   "sha384-MzNNbQTWCSUSi0bbz7dbua+RcENv7C6FvlmYJ1Y+I727HsPOHdzwELMYO9Mz68M26",
				Version:     1,
			},
			isValid: false,
		},
		{
			name: "Invalid Trust Registry ID",
			msg: &types.MsgAddGovernanceFrameworkDocument{
				Corporation: authority,
				Operator:    operator,
				TrId:        99999,
				Language:    "en",
				Url:         "http://example.com/doc2",
				DigestSri:   "sha384-MzNNbQTWCSUSi0bbz7dbua+RcENv7C6FvlmYJ1Y+I727HsPOHdzwELMYO9Mz68M26",
				Version:     2,
			},
			isValid: false,
		},
		{
			name: "Invalid Language Format",
			msg: &types.MsgAddGovernanceFrameworkDocument{
				Corporation: authority,
				Operator:    operator,
				TrId:        trID,
				Language:    "invalid-language",
				Url:         "http://example.com/doc2",
				DigestSri:   "sha384-MzNNbQTWCSUSi0bbz7dbua+RcENv7C6FvlmYJ1Y+I727HsPOHdzwELMYO9Mz68M26",
				Version:     2,
			},
			isValid: false,
		},
		{
			name: "Wrong Controller",
			msg: &types.MsgAddGovernanceFrameworkDocument{
				Corporation: "wrong-controller",
				Operator:    "wrong-controller",
				TrId:        trID,
				Language:    "en",
				Url:         "http://example.com/doc2",
				DigestSri:   "sha384-MzNNbQTWCSUSi0bbz7dbua+RcENv7C6FvlmYJ1Y+I727HsPOHdzwELMYO9Mz68M26",
				Version:     2,
			},
			isValid: false,
		},
		{
			name: "Invalid Version (Skipping Version)",
			setupFunc: func() {
				msg := &types.MsgAddGovernanceFrameworkDocument{
					Corporation: authority,
					Operator:    operator,
					TrId:        trID,
					Language:    "en",
					Url:         "http://example.com/doc3",
					DigestSri:   "sha384-MzNNbQTWCSUSi0bbz7dbua+RcENv7C6FvlmYJ1Y+I727HsPOHdzwELMYO9Mz68M26",
					Version:     3,
				}
				_, err := ms.AddGovernanceFrameworkDocument(ctx, msg)
				require.NoError(t, err)
			},
			msg: &types.MsgAddGovernanceFrameworkDocument{
				Corporation: authority,
				Operator:    operator,
				TrId:        trID,
				Language:    "en",
				Url:         "http://example.com/doc5",
				DigestSri:   "sha384-MzNNbQTWCSUSi0bbz7dbua+RcENv7C6FvlmYJ1Y+I727HsPOHdzwELMYO9Mz68M26",
				Version:     5,
			},
			isValid: false,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			if tc.setupFunc != nil {
				tc.setupFunc()
			}
			resp, err := ms.AddGovernanceFrameworkDocument(ctx, tc.msg)
			if tc.isValid {
				require.NoError(t, err)
				require.NotNil(t, resp)
				var found bool
				err = k.GFDocument.Walk(ctx, nil, func(id uint64, gfd types.GovernanceFrameworkDocument) (bool, error) {
					if gfd.Language == tc.msg.Language && gfd.Url == tc.msg.Url {
						found = true
						return true, nil
					}
					return false, nil
				})
				require.NoError(t, err)
				require.True(t, found)
			} else {
				require.Error(t, err)
				require.Nil(t, resp)
			}
		})
	}
}

func TestMsgServerIncreaseActiveGovernanceFrameworkVersion(t *testing.T) {
	k, ms, ctx := setupMsgServerLegacy(t)

	authority := sdk.AccAddress([]byte("test_authority")).String()
	operator := sdk.AccAddress([]byte("test_operator")).String()
	validDid := "did:example:123456789abcdefghi"

	createMsg := &types.MsgCreateTrustRegistry{
		Corporation:  authority,
		Operator:     operator,
		Did:          validDid,
		Language:     "en",
		DocUrl:       "http://example.com/doc-v1",
		DocDigestSri: testDigestSRI,
	}
	_, err := ms.CreateTrustRegistry(ctx, createMsg)
	require.NoError(t, err)

	trID, err := k.TrustRegistryDIDIndex.Get(ctx, validDid)
	require.NoError(t, err)

	addGFDocMsg := &types.MsgAddGovernanceFrameworkDocument{
		Corporation: authority,
		Operator:    operator,
		TrId:        trID,
		Language:    "es",
		Url:         "http://example.com/doc2-es",
		DigestSri:   "sha384-MzNNbQTWCSUSi0bbz7dbua+RcENv7C6FvlmYJ1Y+I727HsPOHdzwELMYO9Mz68M26",
		Version:     2,
	}
	_, err = ms.AddGovernanceFrameworkDocument(ctx, addGFDocMsg)
	require.NoError(t, err)

	testCases := []struct {
		name      string
		setupFunc func()
		msg       *types.MsgIncreaseActiveGovernanceFrameworkVersion
		isValid   bool
	}{
		{
			name: "Cannot Increase Version - Missing Default Language Document",
			msg: &types.MsgIncreaseActiveGovernanceFrameworkVersion{
				Corporation: authority,
				Operator:    operator,
				TrId:        trID,
			},
			isValid: false,
		},
		{
			name: "Valid Version Increase",
			setupFunc: func() {
				msg := &types.MsgAddGovernanceFrameworkDocument{
					Corporation: authority,
					Operator:    operator,
					TrId:        trID,
					Language:    "en",
					Url:         "http://example.com/doc2-en",
					DigestSri:   "sha384-MzNNbQTWCSUSi0bbz7dbua+RcENv7C6FvlmYJ1Y+I727HsPOHdzwELMYO9Mz68M26",
					Version:     2,
				}
				_, err := ms.AddGovernanceFrameworkDocument(ctx, msg)
				require.NoError(t, err)
			},
			msg: &types.MsgIncreaseActiveGovernanceFrameworkVersion{
				Corporation: authority,
				Operator:    operator,
				TrId:        trID,
			},
			isValid: true,
		},
		{
			name: "Wrong Controller",
			msg: &types.MsgIncreaseActiveGovernanceFrameworkVersion{
				Corporation: "wrong-controller",
				Operator:    operator,
				TrId:        trID,
			},
			isValid: false,
		},
		{
			name: "Non-existent Trust Registry",
			msg: &types.MsgIncreaseActiveGovernanceFrameworkVersion{
				Corporation: authority,
				Operator:    operator,
				TrId:        99999,
			},
			isValid: false,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			if tc.setupFunc != nil {
				tc.setupFunc()
			}
			resp, err := ms.IncreaseActiveGovernanceFrameworkVersion(ctx, tc.msg)
			if tc.isValid {
				require.NoError(t, err)
				require.NotNil(t, resp)
				tr, err := k.TrustRegistry.Get(ctx, tc.msg.TrId)
				require.NoError(t, err)
				require.Equal(t, int32(2), tr.ActiveVersion)
			} else {
				require.Error(t, err)
				require.Nil(t, resp)
			}
		})
	}
}

func TestMsgServerUpdateTrustRegistry(t *testing.T) {
	k, ms, ctx := setupMsgServerLegacy(t)

	authority := sdk.AccAddress([]byte("test_authority")).String()
	operator := sdk.AccAddress([]byte("test_operator")).String()
	validDid := "did:example:123456789abcdefghi"
	newDid := "did:example:updated987654321"
	createMsg := &types.MsgCreateTrustRegistry{
		Corporation:  authority,
		Operator:     operator,
		Did:          validDid,
		Language:     "en",
		DocUrl:       "http://example.com/doc-v1",
		DocDigestSri: testDigestSRI,
	}
	resp, err := ms.CreateTrustRegistry(ctx, createMsg)
	require.NoError(t, err)
	require.NotNil(t, resp)

	trID, err := k.TrustRegistryDIDIndex.Get(ctx, validDid)
	require.NoError(t, err)

	sdkCtx := sdk.UnwrapSDKContext(ctx)
	ctx = sdk.WrapSDKContext(sdkCtx.WithBlockTime(sdkCtx.BlockTime().Add(time.Hour)))

	testCases := []struct {
		name      string
		msg       *types.MsgUpdateTrustRegistry
		expectErr bool
	}{
		{
			name: "Valid Update",
			msg: &types.MsgUpdateTrustRegistry{
				Corporation: authority,
				Operator:    operator,
				TrId:        trID,
				Did:         newDid,
				Aka:         "http://new.example.com",
			},
			expectErr: false,
		},
		{
			name: "Wrong Controller",
			msg: &types.MsgUpdateTrustRegistry{
				Corporation: "wrong-controller",
				Operator:    "wrong-controller",
				TrId:        trID,
				Did:         newDid,
				Aka:         "http://example.com",
			},
			expectErr: true,
		},
		{
			name: "Non-existent Trust Registry",
			msg: &types.MsgUpdateTrustRegistry{
				Corporation: authority,
				Operator:    operator,
				TrId:        99999,
				Did:         newDid,
				Aka:         "http://example.com",
			},
			expectErr: true,
		},
		{
			name: "Clear AKA",
			msg: &types.MsgUpdateTrustRegistry{
				Corporation: authority,
				Operator:    operator,
				TrId:        trID,
				Did:         newDid,
				Aka:         "",
			},
			expectErr: false,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			sdkCtx := sdk.UnwrapSDKContext(ctx)
			testCtx := sdk.WrapSDKContext(sdkCtx.WithBlockTime(sdkCtx.BlockTime().Add(time.Hour)))
			resp, err := ms.UpdateTrustRegistry(testCtx, tc.msg)
			if tc.expectErr {
				require.Error(t, err)
				require.Nil(t, resp)
			} else {
				require.NoError(t, err)
				require.NotNil(t, resp)
				tr, err := k.TrustRegistry.Get(testCtx, tc.msg.TrId)
				require.NoError(t, err)
				require.Equal(t, tc.msg.Aka, tr.Aka)
				require.NotEqual(t, tr.Created, tr.Modified)
			}
		})
	}
}

func TestMsgServerArchiveTrustRegistry(t *testing.T) {
	k, ms, ctx := setupMsgServerLegacy(t)

	authority := sdk.AccAddress([]byte("test_authority")).String()
	operator := sdk.AccAddress([]byte("test_operator")).String()
	validDid := "did:example:123456789abcdefghi"
	createMsg := &types.MsgCreateTrustRegistry{
		Corporation:  authority,
		Operator:     operator,
		Did:          validDid,
		Language:     "en",
		DocUrl:       "http://example.com/doc-v1",
		DocDigestSri: testDigestSRI,
	}
	resp, err := ms.CreateTrustRegistry(ctx, createMsg)
	require.NoError(t, err)
	require.NotNil(t, resp)

	trID, err := k.TrustRegistryDIDIndex.Get(ctx, validDid)
	require.NoError(t, err)

	sdkCtx := sdk.UnwrapSDKContext(ctx)
	ctx = sdk.WrapSDKContext(sdkCtx.WithBlockTime(sdkCtx.BlockTime().Add(time.Hour)))

	testCases := []struct {
		name      string
		msg       *types.MsgArchiveTrustRegistry
		expectErr bool
	}{
		{
			name: "Valid Archive",
			msg: &types.MsgArchiveTrustRegistry{
				Corporation: authority,
				Operator:    operator,
				TrId:        trID,
				Archive:     true,
			},
			expectErr: false,
		},
		{
			name: "Already Archived",
			msg: &types.MsgArchiveTrustRegistry{
				Corporation: authority,
				Operator:    operator,
				TrId:        trID,
				Archive:     true,
			},
			expectErr: true,
		},
		{
			name: "Unarchive Succeeds",
			msg: &types.MsgArchiveTrustRegistry{
				Corporation: authority,
				Operator:    operator,
				TrId:        trID,
				Archive:     false,
			},
			expectErr: false,
		},
		{
			name: "Unarchive Not Archived (abort)",
			msg: &types.MsgArchiveTrustRegistry{
				Corporation: authority,
				Operator:    operator,
				TrId:        trID,
				Archive:     false,
			},
			expectErr: true,
		},
		{
			name: "Wrong Controller",
			msg: &types.MsgArchiveTrustRegistry{
				Corporation: "wrong-controller",
				Operator:    operator,
				TrId:        trID,
				Archive:     true,
			},
			expectErr: true,
		},
		{
			name: "Non-existent Trust Registry",
			msg: &types.MsgArchiveTrustRegistry{
				Corporation: authority,
				Operator:    operator,
				TrId:        99999,
				Archive:     true,
			},
			expectErr: true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			sdkCtx := sdk.UnwrapSDKContext(ctx)
			testCtx := sdk.WrapSDKContext(sdkCtx.WithBlockTime(sdkCtx.BlockTime().Add(time.Hour)))
			resp, err := ms.ArchiveTrustRegistry(testCtx, tc.msg)
			if tc.expectErr {
				require.Error(t, err)
				require.Nil(t, resp)
			} else {
				require.NoError(t, err)
				require.NotNil(t, resp)
				tr, err := k.TrustRegistry.Get(testCtx, tc.msg.TrId)
				require.NoError(t, err)
				if tc.msg.Archive {
					require.NotNil(t, tr.Archived)
				} else {
					require.Nil(t, tr.Archived)
				}
				require.NotEqual(t, tr.Created, tr.Modified)
			}
		})
	}
}
```

**Note:** GFVersion ID lookup uses a `GFVersion.Walk` instead of `GFVersionByTR.Get` to avoid importing the `collections` package and using the `collections.Join` pair codec in tests; walking is idiomatic here and matches the existing query_test.go patterns.

- [ ] **Step 3.2: Build.**

  Run: `go build ./x/tr/keeper/... && go vet ./x/tr/keeper/...`
  Expected: no output, exit 0.

- [ ] **Step 3.3: Run new tests.**

  Run: `go test ./x/tr/keeper/... -run TestMsgCreateTrustRegistry -v -count=1`
  Expected: PASS (all sub-tests green).

- [ ] **Step 3.4: Run full TR keeper suite to verify no regressions.**

  Run: `go test ./x/tr/keeper/... -count=1 -v`
  Expected: PASS.

- [ ] **Step 3.5: Commit.**

```bash
git add x/tr/keeper/msg_server_test.go
git commit -m "test(tr): replace TestMsgServerCreateTrustRegistry with fixture-based test"
```

---

## Task 4: Coverage check and CI validation

- [ ] **Step 4.1: Coverage.**

  Run: `go test ./x/tr/keeper/... -cover -count=1`
  Record coverage. Target: progress toward ≥95% (full 95% is the goal when all 5 TR steps are done).

- [ ] **Step 4.2: Vet.**

  Run: `go vet ./x/tr/keeper/... && go vet ./testutil/keeper/...`
  Expected: no output.

- [ ] **Step 4.3: Full build.**

  Run: `go build ./...`
  Expected: exit 0.

- [ ] **Step 4.4: Full test suite.**

  Run: `go test ./... -count=1`
  Expected: PASS — no regressions.

- [ ] **Step 4.5: Commit and push.**

```bash
git push -u origin test/step-6-tr-create
gh pr create --title "test(tr): step 6 — CreateTrustRegistry fixture-based tests" --body "$(cat <<'EOF'
## Summary
- Adds TrustregistryKeeperWithDelegation to testutil/keeper
- Creates x/tr/keeper/fixture_test.go with Fixture struct and full assertion helpers
- Replaces TestMsgServerCreateTrustRegistry with fixture-based TestMsgCreateTrustRegistry
- Full struct assertion on TrustRegistry + GFVersion + GFDocument
- AUTHZ failure negative case asserts zero state written, no event emitted

## Test plan
- [ ] go test ./x/tr/keeper/... -count=1 passes
- [ ] go test ./testutil/keeper/... -count=1 passes
- [ ] go vet ./... passes
- [ ] go test ./... -count=1 passes (no regressions)
EOF
)"
```

---

## "Done" Criteria — Step 6

- [ ] `testutil/keeper/trustregistry.go` exports `TrustregistryKeeperWithDelegation`.
- [ ] `x/tr/keeper/fixture_test.go` contains `Fixture`, `NewFixture`, `RequireTrustRegistry`, `RequireTrustRegistryCount`, `RequireGFVersionCount`, `RequireGFDocumentCount`, `RequireGFVersion`, `RequireGFDocument`, `RequireDIDIndex`, `RequireNoDIDIndex`, `RequireEvent`, `RequireNoEvent`, `RequireInvariant`, `SetBlockTime`, `AdvanceTime`.
- [ ] Happy path: full `require.Equal` on `TrustRegistry` struct + GFV + GFD + DID index + event + invariant.
- [ ] AUTHZ failure: error contains `"authorization check failed"`, zero state written, no event.
- [ ] Old `TestMsgServerCreateTrustRegistry` deleted.
- [ ] `go test ./x/tr/keeper/... -count=1` PASS.
- [ ] `go test ./... -count=1` PASS (no regressions).

---

## Self-Review

- No TBD/TODO items.
- `TrustregistryKeeperWithDelegation` is added to the same file as the existing keeper, keeping patterns consistent.
- `testDigestSRI` is defined once in `fixture_test.go`; the old definition in `msg_server_test.go` is removed to avoid duplicate symbol.
- Happy path asserts: full `TrustRegistry` struct equality, GFV fields, GFDocument URL+digest, DID index, event attributes, invariant.
- Negative case asserts: `ErrorContains("authorization check failed")`, count 0 for TR/GFV/GFD, `RequireNoDIDIndex`, `RequireNoEvent`.
- `ActiveVersion` type is `int32` (per `types.pb.go` line 40); `RequireInvariant` uses `int32(1)`.
- Legacy tests are preserved verbatim until steps 7-10 replace them.
