# Step 33: DI Module — StoreDigest (MsgServer + ModuleCall)

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Replace the legacy `keeper_test.go` fixture in `x/di/keeper/` with a proper `Fixture`-based test suite. `StoreDigest` has two distinct entry points that must both be tested:

1. `MsgStoreDigest` via `MsgServer` — includes `[AUTHZ-CHECK]` via `DelegationKeeper`.
2. `StoreDigestModuleCall` via `Keeper` directly — no authz check; called by the perm module.

Both paths store the same `Digest` struct and emit the same event. The only behavioral difference is the presence/absence of the AUTHZ check and the handling of a nil `delegationKeeper`.

**Module invariant:** No duplicate digest hashes. The implementation enforces this by returning `ErrDigestAlreadyExists` if the key already exists. `RequireInvariant` scans the collection and verifies all stored keys are unique (the collection itself provides this, but an explicit walk confirms no collision was silently overwritten).

**Tech Stack:** Go 1.24, Cosmos SDK v0.50.x, `cosmossdk.io/collections`, `github.com/stretchr/testify/require`, `testing`.

---

## Pre-flight

- [ ] **Worktree.** Create an isolated git worktree.
- [ ] **Branch.** Branch name: `test/step-33-di-store-digest`.
- [ ] **Sanity-check.** Run `go build ./... && go vet ./...` — exit 0.

---

## File Layout

| Action | Path |
|--------|------|
| Create | `x/di/keeper/fixture_test.go` |
| Create | `x/di/keeper/msg_store_digest_test.go` |
| Delete | `x/di/keeper/keeper_test.go` (old fixture — replaced in same PR) |

---

## Task 1: `fixture_test.go`

**File:** `x/di/keeper/fixture_test.go`

- [ ] **Step 1.1: Create the file.**

```go
package keeper_test

import (
    "context"
    "testing"
    "time"

    storetypes "cosmossdk.io/store/types"
    addresscodec "github.com/cosmos/cosmos-sdk/codec/address"
    "github.com/cosmos/cosmos-sdk/runtime"
    sdktestutil "github.com/cosmos/cosmos-sdk/testutil"
    sdk "github.com/cosmos/cosmos-sdk/types"
    moduletestutil "github.com/cosmos/cosmos-sdk/types/module/testutil"
    authtypes "github.com/cosmos/cosmos-sdk/x/auth/types"
    "github.com/stretchr/testify/require"

    "github.com/verana-labs/verana/x/di/keeper"
    module "github.com/verana-labs/verana/x/di/module"
    "github.com/verana-labs/verana/x/di/types"
)

// MockDelegationKeeper is a controllable stub for types.DelegationKeeper.
// Set ErrToReturn to a non-nil error to simulate an AUTHZ failure.
type MockDelegationKeeper struct {
    ErrToReturn error
}

func (m *MockDelegationKeeper) CheckOperatorAuthorization(_ context.Context, _, _, _ string, _ time.Time) error {
    return m.ErrToReturn
}

// Fixture owns the full test environment for the DI keeper.
// Create one per subtest via NewFixture(t).
type Fixture struct {
    t         *testing.T
    K         keeper.Keeper
    MS        types.MsgServer
    Ctx       sdk.Context
    Authority string // governance module address (bech32)
    DelKeeper *MockDelegationKeeper
}

// NewFixture wires up a fresh in-memory DI keeper with default params and a
// MockDelegationKeeper so the MsgServer's AUTHZ path can be exercised.
func NewFixture(t *testing.T) *Fixture {
    t.Helper()

    encCfg := moduletestutil.MakeTestEncodingConfig(module.AppModule{})
    addressCodec := addresscodec.NewBech32Codec(sdk.GetConfig().GetBech32AccountAddrPrefix())
    storeKey := storetypes.NewKVStoreKey(types.StoreKey)
    storeService := runtime.NewKVStoreService(storeKey)
    ctx := sdktestutil.DefaultContextWithDB(t, storeKey, storetypes.NewTransientStoreKey("transient_test")).Ctx

    authority := authtypes.NewModuleAddress(types.GovModuleName)
    authorityStr, err := addressCodec.BytesToString(authority)
    require.NoError(t, err)

    delKeeper := &MockDelegationKeeper{}

    k := keeper.NewKeeper(
        storeService,
        encCfg.Codec,
        addressCodec,
        authority,
        delKeeper,
    )
    require.NoError(t, k.Params.Set(ctx, types.DefaultParams()))

    ms := keeper.NewMsgServerImpl(k)

    return &Fixture{
        t:         t,
        K:         k,
        MS:        ms,
        Ctx:       ctx,
        Authority: authorityStr,
        DelKeeper: delKeeper,
    }
}

// SetBlockTime returns the Fixture with its Ctx updated to the given block time.
func (f *Fixture) SetBlockTime(ts time.Time) *Fixture {
    f.Ctx = f.Ctx.WithBlockTime(ts)
    return f
}

// RequireDigest asserts that the stored Digest with the given key matches want
// field-by-field (full require.Equal).
func (f *Fixture) RequireDigest(digestKey string, want types.Digest) {
    f.t.Helper()
    got, err := f.K.Digests.Get(f.Ctx, digestKey)
    require.NoError(f.t, err, "digest %q not found", digestKey)
    require.Equal(f.t, want, got)
}

// RequireDigestCount asserts the total number of stored Digests.
func (f *Fixture) RequireDigestCount(n int) {
    f.t.Helper()
    count := 0
    err := f.K.Digests.Walk(f.Ctx, nil, func(_ string, _ types.Digest) (bool, error) {
        count++
        return false, nil
    })
    require.NoError(f.t, err)
    require.Equal(f.t, n, count, "unexpected Digest count")
}

// RequireEvent asserts that the SDK event of eventType was emitted with the
// expected attribute key-value pairs (subset — extra attributes are ignored).
func (f *Fixture) RequireEvent(eventType string, attrs map[string]string) {
    f.t.Helper()
    events := f.Ctx.EventManager().Events()
    for _, ev := range events {
        if ev.Type != eventType {
            continue
        }
        matched := 0
        for _, attr := range ev.Attributes {
            if want, ok := attrs[attr.Key]; ok && attr.Value == want {
                matched++
            }
        }
        if matched == len(attrs) {
            return
        }
    }
    require.Failf(f.t, "event not found",
        "expected event %q with attrs %v in %v", eventType, attrs, events)
}

// RequireNoEvent asserts that no event of eventType was emitted.
func (f *Fixture) RequireNoEvent(eventType string) {
    f.t.Helper()
    for _, ev := range f.Ctx.EventManager().Events() {
        require.NotEqualf(f.t, eventType, ev.Type,
            "expected no event of type %q but found one", eventType)
    }
}

// RequireInvariant checks the DI module invariant:
//   No duplicate digest hashes exist in the store.
//
// Since the collection is keyed by digest string, this is structurally
// guaranteed by the map. The walk here serves as an explicit confirmation
// that no key maps to a different digest value (i.e., the stored Digest.Digest
// matches its map key).
func (f *Fixture) RequireInvariant() {
    f.t.Helper()
    err := f.K.Digests.Walk(f.Ctx, nil, func(key string, d types.Digest) (bool, error) {
        require.Equalf(f.t, key, d.Digest,
            "invariant violated: map key %q does not match stored Digest.Digest %q", key, d.Digest)
        return false, nil
    })
    require.NoError(f.t, err)
}
```

- [ ] **Step 1.2: Verify build.**

  Run: `go build ./x/di/keeper/...`
  Expected: exit 0.

- [ ] **Step 1.3: Commit.**

```bash
git add x/di/keeper/fixture_test.go
git commit -m "test(di): scaffold Fixture for DI keeper tests"
```

---

## Task 2: `msg_store_digest_test.go`

**File:** `x/di/keeper/msg_store_digest_test.go`

This file tests both entry points in one file. Section A covers `MsgStoreDigest` (via `MsgServer`). Section B covers `StoreDigestModuleCall` (via `Keeper`).

- [ ] **Step 2.1: Create the file.**

```go
package keeper_test

import (
    "fmt"
    "testing"
    "time"

    "github.com/stretchr/testify/require"

    "github.com/verana-labs/verana/x/di/types"
)

// ============================================================
// Section A: MsgStoreDigest (MsgServer path — AUTHZ-gated)
// ============================================================

func TestMsgStoreDigest(t *testing.T) {
    baseTime := time.Date(2025, 1, 1, 12, 0, 0, 0, time.UTC)

    operatorAddr := "verana1qyq2k3a9q7l7azyqdmxmhvsw87rfhzuq123" // any valid bech32 for mock purposes

    validMsg := func(authority, operator, digest string) *types.MsgStoreDigest {
        return &types.MsgStoreDigest{
            Authority:       authority,
            Operator:        operator,
            Digest:          digest,
            DigestAlgorithm: "sha2-256",
        }
    }

    // --- Happy path ---
    t.Run("MOD-DI-MSG-1: valid MsgStoreDigest stores digest and emits event", func(t *testing.T) {
        f := NewFixture(t).SetBlockTime(baseTime)

        digestStr := "sha256:abc123def456"
        msg := validMsg(f.Authority, operatorAddr, digestStr)
        _, err := f.MS.StoreDigest(f.Ctx, msg)
        require.NoError(t, err)

        want := types.Digest{
            Digest:          digestStr,
            Created:         baseTime,
            DigestAlgorithm: "sha2-256",
        }
        f.RequireDigest(digestStr, want)
        f.RequireDigestCount(1)
        f.RequireEvent(types.EventTypeStoreDigest, map[string]string{
            types.AttributeKeyAuthority: f.Authority,
            types.AttributeKeyOperator:  operatorAddr,
            types.AttributeKeyDigest:    digestStr,
        })
        f.RequireInvariant()
    })

    // --- Happy path: multiple distinct digests ---
    t.Run("MOD-DI-MSG-1: two distinct digests stored independently", func(t *testing.T) {
        f := NewFixture(t).SetBlockTime(baseTime)

        d1 := "sha256:aaa111"
        d2 := "sha256:bbb222"
        _, err := f.MS.StoreDigest(f.Ctx, validMsg(f.Authority, operatorAddr, d1))
        require.NoError(t, err)
        _, err = f.MS.StoreDigest(f.Ctx, validMsg(f.Authority, operatorAddr, d2))
        require.NoError(t, err)

        f.RequireDigestCount(2)
        f.RequireInvariant()
    })

    // --- Happy path: Created timestamp matches block time ---
    t.Run("MOD-DI-MSG-1: Created timestamp is block time", func(t *testing.T) {
        specificTime := time.Date(2025, 6, 15, 9, 30, 0, 0, time.UTC)
        f := NewFixture(t).SetBlockTime(specificTime)

        digestStr := "sha256:timestamp-test"
        _, err := f.MS.StoreDigest(f.Ctx, validMsg(f.Authority, operatorAddr, digestStr))
        require.NoError(t, err)

        got, err := f.K.Digests.Get(f.Ctx, digestStr)
        require.NoError(t, err)
        require.Equal(t, specificTime, got.Created,
            "Created must equal block time, not local time")
    })

    // --- Negative: AUTHZ fail ---
    t.Run("MOD-DI-MSG-1-2-1-NEG: authorization check failed", func(t *testing.T) {
        f := NewFixture(t).SetBlockTime(baseTime)
        f.DelKeeper.ErrToReturn = fmt.Errorf("operator not authorized")

        digestStr := "sha256:should-not-be-stored"
        _, err := f.MS.StoreDigest(f.Ctx, validMsg(f.Authority, operatorAddr, digestStr))
        require.ErrorContains(t, err, "authorization check failed")
        f.RequireDigestCount(0)
        f.RequireNoEvent(types.EventTypeStoreDigest)
    })

    // --- Negative: duplicate digest ---
    t.Run("MOD-DI-MSG-1-NEG: duplicate digest rejected", func(t *testing.T) {
        f := NewFixture(t).SetBlockTime(baseTime)

        digestStr := "sha256:dup-test"
        _, err := f.MS.StoreDigest(f.Ctx, validMsg(f.Authority, operatorAddr, digestStr))
        require.NoError(t, err)

        // Reset event manager so we can assert no new event
        f.Ctx = f.Ctx.WithEventManager(sdk.NewEventManager())

        _, err = f.MS.StoreDigest(f.Ctx, validMsg(f.Authority, operatorAddr, digestStr))
        require.ErrorContains(t, err, "digest already exists")
        f.RequireDigestCount(1) // still only one
        f.RequireNoEvent(types.EventTypeStoreDigest)
        f.RequireInvariant()
    })

    // --- Negative: nil delegation keeper ---
    t.Run("MOD-DI-MSG-1-NEG: nil delegation keeper returns ErrDelegationKeeperNil", func(t *testing.T) {
        f := NewFixture(t).SetBlockTime(baseTime)

        // Build a MsgServer backed by a keeper with nil delegation keeper.
        // We do this by calling keeper.NewMsgServerImpl with a keeper that has
        // delegationKeeper == nil. The DI keeper.NewKeeper accepts nil explicitly.
        import_addCodecEtc := "already imported"
        _ = import_addCodecEtc

        // Use the existing fixture's keeper but swap the MS to one with nil delKeeper.
        // keeper.NewKeeper accepts nil delegationKeeper — see keeper.go.
        kNilDel := keeper.NewKeeperForTest(
            f.K, // share same store
            nil, // delegationKeeper = nil
        )
        msNilDel := keeper.NewMsgServerImpl(kNilDel)

        digestStr := "sha256:nil-del-test"
        _, err := msNilDel.StoreDigest(f.Ctx, validMsg(f.Authority, "operator", digestStr))
        require.ErrorIs(t, err, types.ErrDelegationKeeperNil)
        f.RequireDigestCount(0)
    })
}

// ============================================================
// Section B: StoreDigestModuleCall (Keeper direct path — no AUTHZ)
// ============================================================

func TestStoreDigestModuleCall(t *testing.T) {
    baseTime := time.Date(2025, 1, 1, 12, 0, 0, 0, time.UTC)

    validAuthorityAddr := "verana1qyq2k3a9q7l7azyqdmxmhvsw87rfhzuq123"

    // --- Happy path ---
    t.Run("MOD-DI-MODULECALL: valid call stores digest and emits event", func(t *testing.T) {
        f := NewFixture(t).SetBlockTime(baseTime)

        digestStr := "sha256:module-call-test"
        err := f.K.StoreDigestModuleCall(f.Ctx, validAuthorityAddr, digestStr, "sha2-256")
        require.NoError(t, err)

        want := types.Digest{
            Digest:          digestStr,
            Created:         baseTime,
            DigestAlgorithm: "sha2-256",
        }
        f.RequireDigest(digestStr, want)
        f.RequireEvent(types.EventTypeStoreDigest, map[string]string{
            types.AttributeKeyAuthority: validAuthorityAddr,
            types.AttributeKeyDigest:    digestStr,
        })
        f.RequireInvariant()
    })

    // --- Happy path: module call bypasses AUTHZ even when DelKeeper would fail ---
    t.Run("MOD-DI-MODULECALL: no AUTHZ check — DelKeeper error is ignored", func(t *testing.T) {
        f := NewFixture(t).SetBlockTime(baseTime)
        f.DelKeeper.ErrToReturn = fmt.Errorf("would fail if AUTHZ were checked")

        digestStr := "sha256:bypass-authz"
        err := f.K.StoreDigestModuleCall(f.Ctx, validAuthorityAddr, digestStr, "sha2-256")
        require.NoError(t, err)
        f.RequireDigestCount(1)
        f.RequireInvariant()
    })

    // --- Negative: duplicate digest ---
    t.Run("MOD-DI-MODULECALL-NEG: duplicate digest rejected", func(t *testing.T) {
        f := NewFixture(t).SetBlockTime(baseTime)

        digestStr := "sha256:dup-module"
        err := f.K.StoreDigestModuleCall(f.Ctx, validAuthorityAddr, digestStr, "sha2-256")
        require.NoError(t, err)

        f.Ctx = f.Ctx.WithEventManager(sdk.NewEventManager())

        err = f.K.StoreDigestModuleCall(f.Ctx, validAuthorityAddr, digestStr, "sha2-256")
        require.ErrorContains(t, err, "digest already exists")
        f.RequireDigestCount(1)
        f.RequireNoEvent(types.EventTypeStoreDigest)
        f.RequireInvariant()
    })

    // --- Negative: empty digest ---
    t.Run("MOD-DI-MODULECALL-NEG: empty digest rejected", func(t *testing.T) {
        f := NewFixture(t).SetBlockTime(baseTime)

        err := f.K.StoreDigestModuleCall(f.Ctx, validAuthorityAddr, "", "sha2-256")
        require.ErrorIs(t, err, types.ErrDigestEmpty)
        f.RequireDigestCount(0)
        f.RequireNoEvent(types.EventTypeStoreDigest)
    })

    // --- Negative: invalid authority address ---
    t.Run("MOD-DI-MODULECALL-NEG: invalid authority address rejected", func(t *testing.T) {
        f := NewFixture(t).SetBlockTime(baseTime)

        err := f.K.StoreDigestModuleCall(f.Ctx, "not-a-bech32-address", "sha256:valid-digest", "sha2-256")
        require.ErrorContains(t, err, "invalid authority address")
        f.RequireDigestCount(0)
        f.RequireNoEvent(types.EventTypeStoreDigest)
    })

    // --- Edge: Created timestamp is block time (module call path) ---
    t.Run("MOD-DI-MODULECALL-EDGE: Created timestamp is block time", func(t *testing.T) {
        specificTime := time.Date(2025, 12, 31, 23, 59, 59, 0, time.UTC)
        f := NewFixture(t).SetBlockTime(specificTime)

        digestStr := "sha256:mc-timestamp"
        err := f.K.StoreDigestModuleCall(f.Ctx, validAuthorityAddr, digestStr, "sha2-256")
        require.NoError(t, err)

        got, err := f.K.Digests.Get(f.Ctx, digestStr)
        require.NoError(t, err)
        require.Equal(t, specificTime, got.Created)
    })

    // --- Edge: MsgServer duplicate is rejected even after ModuleCall stored the same key ---
    t.Run("MOD-DI-EDGE: MsgServer rejects digest already stored by ModuleCall", func(t *testing.T) {
        f := NewFixture(t).SetBlockTime(baseTime)
        operatorAddr := validAuthorityAddr

        digestStr := "sha256:cross-path-dup"
        // Store via ModuleCall first
        err := f.K.StoreDigestModuleCall(f.Ctx, f.Authority, digestStr, "sha2-256")
        require.NoError(t, err)

        f.Ctx = f.Ctx.WithEventManager(sdk.NewEventManager())

        // Now try to store same digest via MsgServer
        msg := &types.MsgStoreDigest{
            Authority:       f.Authority,
            Operator:        operatorAddr,
            Digest:          digestStr,
            DigestAlgorithm: "sha2-256",
        }
        _, err = f.MS.StoreDigest(f.Ctx, msg)
        require.ErrorContains(t, err, "digest already exists")
        f.RequireDigestCount(1)
        f.RequireInvariant()
    })
}
```

**Note:** The `sdk.NewEventManager()` calls require `sdk "github.com/cosmos/cosmos-sdk/types"` in the import block.

**Note on `keeper.NewKeeperForTest`:** The `nil delegationKeeper` test case requires a way to create a Keeper with a nil delegation keeper. Looking at `keeper.NewKeeper`, it accepts `delegationKeeper types.DelegationKeeper` which can be nil (the original `keeper_test.go` passes `nil` explicitly). However, it shares state with the fixture's keeper store if we call it that way. A simpler approach: create a second `NewFixture` variant `NewFixtureNilDel(t)` that initializes with `nil` delegation keeper and call the MsgServer with the dedicated nil-del fixture. Adjust the test accordingly:

```go
// NewFixtureNilDel creates a Fixture whose Keeper has nil delegationKeeper.
// Use this only to test ErrDelegationKeeperNil behavior.
func NewFixtureNilDel(t *testing.T) *Fixture {
    t.Helper()

    encCfg := moduletestutil.MakeTestEncodingConfig(module.AppModule{})
    addressCodec := addresscodec.NewBech32Codec(sdk.GetConfig().GetBech32AccountAddrPrefix())
    storeKey := storetypes.NewKVStoreKey(types.StoreKey)
    storeService := runtime.NewKVStoreService(storeKey)
    ctx := sdktestutil.DefaultContextWithDB(t, storeKey, storetypes.NewTransientStoreKey("transient_test_nil")).Ctx

    authority := authtypes.NewModuleAddress(types.GovModuleName)
    authorityStr, err := addressCodec.BytesToString(authority)
    require.NoError(t, err)

    k := keeper.NewKeeper(
        storeService,
        encCfg.Codec,
        addressCodec,
        authority,
        nil, // nil delegation keeper — ErrDelegationKeeperNil path
    )
    require.NoError(t, k.Params.Set(ctx, types.DefaultParams()))

    ms := keeper.NewMsgServerImpl(k)

    return &Fixture{
        t:         t,
        K:         k,
        MS:        ms,
        Ctx:       ctx,
        Authority: authorityStr,
        DelKeeper: nil,
    }
}
```

Add this to `fixture_test.go` and update the nil-delegation-keeper subtest to use `NewFixtureNilDel`:

```go
t.Run("MOD-DI-MSG-1-NEG: nil delegation keeper returns ErrDelegationKeeperNil", func(t *testing.T) {
    f := NewFixtureNilDel(t).SetBlockTime(baseTime)
    digestStr := "sha256:nil-del-test"
    _, err := f.MS.StoreDigest(f.Ctx, &types.MsgStoreDigest{
        Authority:       f.Authority,
        Operator:        "verana1qyq2k3a9q7l7azyqdmxmhvsw87rfhzuq123",
        Digest:          digestStr,
        DigestAlgorithm: "sha2-256",
    })
    require.ErrorIs(t, err, types.ErrDelegationKeeperNil)
    f.RequireDigestCount(0)
})
```

- [ ] **Step 2.2: Run.**

  Run: `go test ./x/di/keeper/... -run 'TestMsgStoreDigest|TestStoreDigestModuleCall' -v`
  Expected: all subtests PASS.

- [ ] **Step 2.3: Commit.**

```bash
git add x/di/keeper/fixture_test.go x/di/keeper/msg_store_digest_test.go
git commit -m "test(di): add fixture-based TestMsgStoreDigest and TestStoreDigestModuleCall"
```

---

## Task 3: Delete old `keeper_test.go`

- [ ] **Step 3.1: Delete the file.**

```bash
git rm x/di/keeper/keeper_test.go
```

- [ ] **Step 3.2: Verify full suite still builds and passes.**

  Run: `go test ./x/di/keeper/... -v -count=1`
  Expected: PASS.

- [ ] **Step 3.3: Commit.**

```bash
git commit -m "test(di): remove old keeper_test.go (replaced by fixture-based tests)"
```

---

## Task 4: Final pass — vet, lint, coverage

- [ ] **Step 4.1: Run full test suite.**

  Run: `go test ./x/di/keeper/... -v -count=1`
  Expected: PASS.

- [ ] **Step 4.2: Run with race detector.**

  Run: `go test ./x/di/keeper/... -race -count=1`
  Expected: PASS, no DATA RACE.

- [ ] **Step 4.3: Run vet.**

  Run: `go vet ./x/di/keeper/...`
  Expected: no output.

- [ ] **Step 4.4: Run linter.**

  Run: `golangci-lint run ./x/di/keeper/...`
  Expected: no findings. Fix any reported issues then re-run.

- [ ] **Step 4.5: Coverage check.**

  Run: `go test ./x/di/keeper/... -cover -count=1`
  Expected: coverage ≥95% on keeper files. If below, run:
  `go test ./x/di/keeper/ -coverprofile=/tmp/di.cov && go tool cover -func=/tmp/di.cov`
  to find uncovered lines and add tests.

- [ ] **Step 4.6: Full repo build check.**

  Run: `go build ./... && go test ./... -count=1`
  Expected: PASS, no regressions.

- [ ] **Step 4.7: Push and open PR.**

```bash
git push -u origin test/step-33-di-store-digest
gh pr create --title "test(di): fixture-based tests for StoreDigest (step 33)" --body "$(cat <<'EOF'
## Summary
- New `fixture_test.go` in `x/di/keeper/` with Fixture struct, MockDelegationKeeper, NewFixtureNilDel, RequireDigest, RequireDigestCount, RequireEvent, RequireNoEvent, RequireInvariant
- `TestMsgStoreDigest` (MsgServer path): happy paths (single digest, multiple digests, Created timestamp), negatives (AUTHZ fail, duplicate digest, nil delegation keeper)
- `TestStoreDigestModuleCall` (Keeper direct path): happy paths (valid call, AUTHZ bypass), negatives (duplicate digest, empty digest, invalid authority address), edges (Created timestamp, cross-path duplicate rejection)
- Old `keeper_test.go` deleted in this PR

## Test plan
- [ ] `go test ./x/di/keeper/... -race -count=1` passes
- [ ] `go test ./x/di/keeper/... -cover` reports ≥95%
- [ ] `golangci-lint run ./x/di/keeper/...` clean
- [ ] `go test ./...` (full repo) passes — no regressions
EOF
)"
```

---

## "Done" Criteria — Step 33

- [ ] `fixture_test.go` exists with `Fixture`, `MockDelegationKeeper`, `NewFixtureNilDel`, all helper methods, `RequireInvariant`.
- [ ] `TestMsgStoreDigest`: happy path (full struct assertion + event + invariant), multiple distinct digests, Created = block time, AUTHZ fail, duplicate digest, nil delegation keeper.
- [ ] `TestStoreDigestModuleCall`: happy path, AUTHZ bypass, duplicate, empty digest, invalid authority, Created = block time edge, cross-path duplicate rejection.
- [ ] Old `keeper_test.go` deleted in the same PR.
- [ ] `go test ./x/di/keeper/... -race -count=1` passes.
- [ ] Coverage ≥95% on `x/di/keeper/`.
- [ ] `go test ./...`, `go vet ./...`, `golangci-lint run` all pass.

---

## Self-Review Notes

- **Two entry points:** `StoreDigest` (MsgServer) and `StoreDigestModuleCall` (Keeper) both store the same struct. The key behavioral difference is AUTHZ: MsgServer checks it; ModuleCall does not. Both paths guard against duplicate digests and empty digests.
- **nil delegationKeeper:** `keeper.NewKeeper` accepts `nil` (see original `keeper_test.go:44: nil`). `NewFixtureNilDel` replicates this pattern. The `ErrDelegationKeeperNil` test validates that the keeper surfaces a clear error rather than a panic or nil pointer dereference.
- **Invariant verification:** The DI invariant (no duplicate hashes) is structurally enforced by the `collections.Map` which keys on the digest string. `RequireInvariant` additionally confirms `Digest.Digest == map key` — a useful sanity check if the struct's Digest field ever diverges from its storage key.
- **Event manager reset:** Tests that use setup operations (e.g., a prior StoreDigest to create a duplicate target) call `f.Ctx = f.Ctx.WithEventManager(sdk.NewEventManager())` before the message under test to isolate event assertions.
- **No bank mock needed:** DI module has no bank operations. Fixture does not embed `StatefulBankMock`.
- **Cross-path duplicate:** The edge test `MOD-DI-EDGE: MsgServer rejects digest already stored by ModuleCall` confirms the two entry points share the same underlying collection, so a digest stored by the perm module's `StoreDigestModuleCall` cannot be accidentally overwritten by a subsequent `MsgStoreDigest`.
