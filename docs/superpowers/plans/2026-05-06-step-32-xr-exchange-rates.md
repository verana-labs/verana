# Step 32: XR Module — CreateExchangeRate, UpdateExchangeRate, SetExchangeRateState

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Replace the legacy `keeper_test.go` fixture in `x/xr/keeper/` with a proper `Fixture`-based test suite covering all three XR messages. Each message gets: (a) a happy-path test with full struct assertion + event assertion + invariant check, (b) one `t.Run` per spec precondition (error string + zero state + no event), and (c) relevant edge cases. No bank math is involved (XR has no fees), so there is no `StatefulBankMock` dependency — only a `MockDelegationKeeper`.

**Module invariant checked after every happy-path:** At most one active rate row per `(baseAssetType, baseAsset, quoteAssetType, quoteAsset)` pair. The `PairIndex` enforces this at write time, but the invariant verifier independently scans the collection and confirms it holds.

**Tech Stack:** Go 1.24, Cosmos SDK v0.50.x, `cosmossdk.io/collections`, `github.com/stretchr/testify/require`, `testing`.

---

## Pre-flight

- [ ] **Worktree.** Create an isolated git worktree.
- [ ] **Branch.** Branch name: `test/step-32-xr-exchange-rates`.
- [ ] **Sanity-check.** Run `go build ./... && go vet ./...` — exit 0.

---

## File Layout

| Action | Path |
|--------|------|
| Create | `x/xr/keeper/fixture_test.go` |
| Create | `x/xr/keeper/msg_create_exchange_rate_test.go` |
| Create | `x/xr/keeper/msg_update_exchange_rate_test.go` |
| Create | `x/xr/keeper/msg_set_exchange_rate_state_test.go` |
| Delete | `x/xr/keeper/keeper_test.go` (old fixture — replaced in same PR) |

---

## Spec formula functions

These are pure Go, no keeper imports, written at the top of each `_test.go` file that needs them.

```go
// specXRExpires returns the expected Expires timestamp per [MOD-XR-MSG-1]:
//   xr.Expires = blockTime.Add(msg.ValidityDuration)
func specXRExpires(blockTime time.Time, validityDuration time.Duration) time.Time {
    return blockTime.Add(validityDuration)
}

// specXRUpdateExpires returns the expected Expires timestamp after UpdateExchangeRate
// per [MOD-XR-MSG-2]: xr.Expires = now.Add(xr.ValidityDuration)
// where xr.ValidityDuration may have been updated by msg.ValidityDuration.
func specXRUpdateExpires(now time.Time, effectiveValidityDuration time.Duration) time.Time {
    return now.Add(effectiveValidityDuration)
}
```

---

## Task 1: `fixture_test.go`

**File:** `x/xr/keeper/fixture_test.go`

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

    "github.com/verana-labs/verana/x/xr/keeper"
    module "github.com/verana-labs/verana/x/xr/module"
    "github.com/verana-labs/verana/x/xr/types"
)

// MockDelegationKeeper is a controllable stub for types.DelegationKeeper.
// Set ErrToReturn to a non-nil error to simulate an AUTHZ failure.
type MockDelegationKeeper struct {
    ErrToReturn error
}

func (m *MockDelegationKeeper) CheckOperatorAuthorization(_ context.Context, _, _, _ string, _ time.Time) error {
    return m.ErrToReturn
}

// Fixture owns the full test environment for the XR keeper.
// Create one per subtest via NewFixture(t).
type Fixture struct {
    t         *testing.T
    K         keeper.Keeper
    MS        types.MsgServer
    Ctx       sdk.Context
    Authority string // governance module address (bech32)
    DelKeeper *MockDelegationKeeper
}

// NewFixture wires up a fresh in-memory XR keeper with default params.
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

// SetBlockTime returns a new Fixture whose Ctx has the given block time.
// Call this before executing messages that care about block time.
func (f *Fixture) SetBlockTime(ts time.Time) *Fixture {
    f.Ctx = f.Ctx.WithBlockTime(ts)
    return f
}

// AdvanceTime returns a new Fixture whose Ctx is advanced by d.
func (f *Fixture) AdvanceTime(d time.Duration) *Fixture {
    f.Ctx = f.Ctx.WithBlockTime(f.Ctx.BlockTime().Add(d))
    return f
}

// RequireExchangeRate asserts that the stored ExchangeRate with the given id
// matches want field-by-field (full require.Equal).
func (f *Fixture) RequireExchangeRate(id uint64, want types.ExchangeRate) {
    f.t.Helper()
    got, err := f.K.ExchangeRates.Get(f.Ctx, id)
    require.NoError(f.t, err, "exchange rate %d not found", id)
    require.Equal(f.t, want, got)
}

// RequireExchangeRateCount asserts the total number of stored ExchangeRates.
func (f *Fixture) RequireExchangeRateCount(n int) {
    f.t.Helper()
    count := 0
    err := f.K.ExchangeRates.Walk(f.Ctx, nil, func(_ uint64, _ types.ExchangeRate) (bool, error) {
        count++
        return false, nil
    })
    require.NoError(f.t, err)
    require.Equal(f.t, n, count, "unexpected ExchangeRate count")
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

// RequireInvariant checks the XR module invariant:
//   At most one active rate row per (baseAssetType, baseAsset, quoteAssetType, quoteAsset) pair.
//
// This is verified independently of PairIndex by scanning ExchangeRates and
// building a local uniqueness map.
func (f *Fixture) RequireInvariant() {
    f.t.Helper()
    seen := map[string]uint64{}
    err := f.K.ExchangeRates.Walk(f.Ctx, nil, func(id uint64, xr types.ExchangeRate) (bool, error) {
        // Only active rates must be unique per pair.
        if !xr.State {
            return false, nil
        }
        key := buildPairKeyForTest(xr.BaseAssetType, xr.BaseAsset, xr.QuoteAssetType, xr.QuoteAsset)
        if existing, dup := seen[key]; dup {
            require.Failf(f.t, "invariant violated",
                "two active ExchangeRates with same pair key %q: ids %d and %d", key, existing, id)
        }
        seen[key] = id
        return false, nil
    })
    require.NoError(f.t, err)
}

// buildPairKeyForTest replicates the keeper's internal buildPairKey formula.
// Kept in the test package so the test never calls keeper-internal logic.
func buildPairKeyForTest(
    baseType interface{ String() string },
    baseAsset string,
    quoteType interface{ String() string },
    quoteAsset string,
) string {
    return baseType.String() + ":" + baseAsset + ":" + quoteType.String() + ":" + quoteAsset
}
```

- [ ] **Step 1.2: Verify build.**

  Run: `go build ./x/xr/keeper/...`
  Expected: exit 0.

- [ ] **Step 1.3: Commit.**

```bash
git add x/xr/keeper/fixture_test.go
git commit -m "test(xr): scaffold Fixture for XR keeper tests"
```

---

## Task 2: `msg_create_exchange_rate_test.go`

**File:** `x/xr/keeper/msg_create_exchange_rate_test.go`

Spec formula: `xr.Expires = blockTime.Add(msg.ValidityDuration)`

- [ ] **Step 2.1: Create the file with helper function.**

```go
package keeper_test

import (
    "fmt"
    "testing"
    "time"

    "github.com/stretchr/testify/require"

    cstypes "github.com/verana-labs/verana/x/cs/types"
    "github.com/verana-labs/verana/x/xr/types"
)

// specXRExpires returns the expected Expires per [MOD-XR-MSG-1]:
//   xr.Expires = blockTime.Add(msg.ValidityDuration)
func specXRExpires(blockTime time.Time, validityDuration time.Duration) time.Time {
    return blockTime.Add(validityDuration)
}

func TestMsgCreateExchangeRate(t *testing.T) {
    baseTime := time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC)
    dur := 30 * 24 * time.Hour // 30 days — well within DefaultMaxValidityDuration (365d)

    validMsg := func(authority string) *types.MsgCreateExchangeRate {
        return &types.MsgCreateExchangeRate{
            Authority:        authority,
            BaseAssetType:    cstypes.PricingAssetType_TU,
            BaseAsset:        "uvna",
            QuoteAssetType:   cstypes.PricingAssetType_FIAT,
            QuoteAsset:       "USD",
            Rate:             "1.5",
            RateScale:        6,
            ValidityDuration: dur,
        }
    }

    // --- Happy path ---
    t.Run("MOD-XR-MSG-1: valid governance CreateExchangeRate", func(t *testing.T) {
        f := NewFixture(t).SetBlockTime(baseTime)

        resp, err := f.MS.CreateExchangeRate(f.Ctx, validMsg(f.Authority))
        require.NoError(t, err)
        require.NotZero(t, resp.Id)

        want := types.ExchangeRate{
            Id:               resp.Id,
            BaseAssetType:    cstypes.PricingAssetType_TU,
            BaseAsset:        "uvna",
            QuoteAssetType:   cstypes.PricingAssetType_FIAT,
            QuoteAsset:       "USD",
            Rate:             "1.5",
            RateScale:        6,
            ValidityDuration: dur,
            Expires:          specXRExpires(baseTime, dur),
            State:            true,
            Updated:          baseTime,
        }
        f.RequireExchangeRate(resp.Id, want)
        f.RequireEvent(types.EventTypeCreateExchangeRate, map[string]string{
            types.AttributeKeyID:           fmt.Sprintf("%d", resp.Id),
            types.AttributeKeyAuthority:    f.Authority,
            types.AttributeKeyBaseAsset:    "uvna",
            types.AttributeKeyQuoteAsset:   "USD",
        })
        f.RequireInvariant()
    })

    // --- Multiple rates, each with distinct pair ---
    t.Run("MOD-XR-MSG-1: two distinct pairs coexist", func(t *testing.T) {
        f := NewFixture(t).SetBlockTime(baseTime)

        msg1 := validMsg(f.Authority)
        resp1, err := f.MS.CreateExchangeRate(f.Ctx, msg1)
        require.NoError(t, err)

        msg2 := validMsg(f.Authority)
        msg2.BaseAsset = "uatom"
        resp2, err := f.MS.CreateExchangeRate(f.Ctx, msg2)
        require.NoError(t, err)

        require.NotEqual(t, resp1.Id, resp2.Id)
        f.RequireExchangeRateCount(2)
        f.RequireInvariant()
    })

    // --- Negative: invalid authority ---
    t.Run("MOD-XR-MSG-1-NEG: wrong authority rejected", func(t *testing.T) {
        f := NewFixture(t).SetBlockTime(baseTime)

        badAuthority := "verana1qyq2k3a9q7l7azyqdmxmhvsw87rfhzuq"
        msg := validMsg(badAuthority)
        _, err := f.MS.CreateExchangeRate(f.Ctx, msg)
        require.ErrorContains(t, err, "invalid authority")
        f.RequireExchangeRateCount(0)
        f.RequireNoEvent(types.EventTypeCreateExchangeRate)
    })

    // --- Negative: validity_duration exceeds max ---
    t.Run("MOD-XR-MSG-1-NEG: validity_duration > max_validity_duration rejected", func(t *testing.T) {
        f := NewFixture(t).SetBlockTime(baseTime)

        msg := validMsg(f.Authority)
        msg.ValidityDuration = 400 * 24 * time.Hour // > 365 days default max
        _, err := f.MS.CreateExchangeRate(f.Ctx, msg)
        require.ErrorContains(t, err, "exceeds max_validity_duration")
        f.RequireExchangeRateCount(0)
        f.RequireNoEvent(types.EventTypeCreateExchangeRate)
    })

    // --- Negative: duplicate pair ---
    t.Run("MOD-XR-MSG-1-NEG: duplicate pair rejected", func(t *testing.T) {
        f := NewFixture(t).SetBlockTime(baseTime)

        msg := validMsg(f.Authority)
        _, err := f.MS.CreateExchangeRate(f.Ctx, msg)
        require.NoError(t, err)

        // Same pair again
        _, err = f.MS.CreateExchangeRate(f.Ctx, validMsg(f.Authority))
        require.ErrorContains(t, err, "exchange rate pair already exists")
        f.RequireExchangeRateCount(1) // only the first succeeded
        f.RequireInvariant()
    })

    // --- Edge: minimum validity duration (exactly 1 minute) ---
    t.Run("MOD-XR-MSG-1-EDGE: minimum validity duration accepted", func(t *testing.T) {
        f := NewFixture(t).SetBlockTime(baseTime)

        msg := validMsg(f.Authority)
        msg.ValidityDuration = time.Minute
        resp, err := f.MS.CreateExchangeRate(f.Ctx, msg)
        require.NoError(t, err)
        require.NotZero(t, resp.Id)

        got, err := f.K.ExchangeRates.Get(f.Ctx, resp.Id)
        require.NoError(t, err)
        require.Equal(t, specXRExpires(baseTime, time.Minute), got.Expires)
    })

    // --- Edge: max allowed validity duration (exactly DefaultMaxValidityDuration) ---
    t.Run("MOD-XR-MSG-1-EDGE: max validity duration accepted", func(t *testing.T) {
        f := NewFixture(t).SetBlockTime(baseTime)

        msg := validMsg(f.Authority)
        msg.ValidityDuration = types.DefaultMaxValidityDuration
        resp, err := f.MS.CreateExchangeRate(f.Ctx, msg)
        require.NoError(t, err)
        require.NotZero(t, resp.Id)
        f.RequireInvariant()
    })
}
```

- [ ] **Step 2.2: Run.**

  Run: `go test ./x/xr/keeper/... -run TestMsgCreateExchangeRate -v`
  Expected: all subtests PASS.

- [ ] **Step 2.3: Commit.**

```bash
git add x/xr/keeper/msg_create_exchange_rate_test.go
git commit -m "test(xr): add fixture-based TestMsgCreateExchangeRate"
```

---

## Task 3: `msg_update_exchange_rate_test.go`

**File:** `x/xr/keeper/msg_update_exchange_rate_test.go`

Spec: operator via delegation; re-calculates Expires = now.Add(effectiveValidityDuration); operator is any valid address (AUTHZ check is mocked).

- [ ] **Step 3.1: Create the file.**

```go
package keeper_test

import (
    "fmt"
    "testing"
    "time"

    "github.com/stretchr/testify/require"

    authtypes "github.com/cosmos/cosmos-sdk/x/auth/types"
    cstypes "github.com/verana-labs/verana/x/cs/types"
    "github.com/verana-labs/verana/x/xr/types"
)

// specXRUpdateExpires returns the expected Expires after UpdateExchangeRate per [MOD-XR-MSG-2]:
//   xr.Expires = now.Add(effectiveValidityDuration)
// where effectiveValidityDuration is the stored ValidityDuration after any update.
func specXRUpdateExpires(now time.Time, effectiveValidityDuration time.Duration) time.Time {
    return now.Add(effectiveValidityDuration)
}

func TestMsgUpdateExchangeRate(t *testing.T) {
    baseTime := time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC)
    dur := 30 * 24 * time.Hour

    operatorAddr := authtypes.NewModuleAddress("test-operator").String()

    // seedXR creates one active, non-expired XR and returns its id.
    seedXR := func(f *Fixture) uint64 {
        f.t.Helper()
        resp, err := f.MS.CreateExchangeRate(f.Ctx, &types.MsgCreateExchangeRate{
            Authority:        f.Authority,
            BaseAssetType:    cstypes.PricingAssetType_TU,
            BaseAsset:        "uvna",
            QuoteAssetType:   cstypes.PricingAssetType_FIAT,
            QuoteAsset:       "USD",
            Rate:             "1.5",
            RateScale:        6,
            ValidityDuration: dur,
        })
        require.NoError(f.t, err)
        return resp.Id
    }

    // --- Happy path: update rate only ---
    t.Run("MOD-XR-MSG-2: update rate only", func(t *testing.T) {
        f := NewFixture(t).SetBlockTime(baseTime)
        id := seedXR(f)

        // Advance time to simulate later update
        updateTime := baseTime.Add(time.Hour)
        f.SetBlockTime(updateTime)

        msg := &types.MsgUpdateExchangeRate{
            Authority: f.Authority,
            Operator:  operatorAddr,
            Id:        id,
            Rate:      "2.0",
        }
        _, err := f.MS.UpdateExchangeRate(f.Ctx, msg)
        require.NoError(t, err)

        got, err := f.K.ExchangeRates.Get(f.Ctx, id)
        require.NoError(t, err)
        require.Equal(t, "2.0", got.Rate)
        require.Equal(t, uint32(6), got.RateScale) // unchanged
        require.Equal(t, dur, got.ValidityDuration) // unchanged
        require.Equal(t, specXRUpdateExpires(updateTime, dur), got.Expires)
        require.Equal(t, updateTime, got.Updated)

        f.RequireEvent(types.EventTypeUpdateExchangeRate, map[string]string{
            types.AttributeKeyID:        fmt.Sprintf("%d", id),
            types.AttributeKeyAuthority: f.Authority,
            types.AttributeKeyRate:      "2.0",
        })
        f.RequireInvariant()
    })

    // --- Happy path: update rate + rate_scale ---
    t.Run("MOD-XR-MSG-2: update rate and rate_scale", func(t *testing.T) {
        f := NewFixture(t).SetBlockTime(baseTime)
        id := seedXR(f)
        updateTime := baseTime.Add(2 * time.Hour)
        f.SetBlockTime(updateTime)

        newRateScale := uint32(8)
        msg := &types.MsgUpdateExchangeRate{
            Authority: f.Authority,
            Operator:  operatorAddr,
            Id:        id,
            Rate:      "3.0",
            RateScale: newRateScale,
        }
        _, err := f.MS.UpdateExchangeRate(f.Ctx, msg)
        require.NoError(t, err)

        got, err := f.K.ExchangeRates.Get(f.Ctx, id)
        require.NoError(t, err)
        require.Equal(t, "3.0", got.Rate)
        require.Equal(t, newRateScale, got.RateScale)
        require.Equal(t, specXRUpdateExpires(updateTime, dur), got.Expires)
        f.RequireInvariant()
    })

    // --- Happy path: update rate + validity_duration ---
    t.Run("MOD-XR-MSG-2: update rate and validity_duration", func(t *testing.T) {
        f := NewFixture(t).SetBlockTime(baseTime)
        id := seedXR(f)
        updateTime := baseTime.Add(3 * time.Hour)
        f.SetBlockTime(updateTime)

        newDur := 15 * 24 * time.Hour // 15 days
        msg := &types.MsgUpdateExchangeRate{
            Authority:        f.Authority,
            Operator:         operatorAddr,
            Id:               id,
            Rate:             "4.0",
            ValidityDuration: &newDur,
        }
        _, err := f.MS.UpdateExchangeRate(f.Ctx, msg)
        require.NoError(t, err)

        got, err := f.K.ExchangeRates.Get(f.Ctx, id)
        require.NoError(t, err)
        require.Equal(t, newDur, got.ValidityDuration)
        require.Equal(t, specXRUpdateExpires(updateTime, newDur), got.Expires)
        f.RequireInvariant()
    })

    // --- Negative: AUTHZ fail ---
    t.Run("MOD-XR-MSG-2-NEG: authorization check failed", func(t *testing.T) {
        f := NewFixture(t).SetBlockTime(baseTime)
        id := seedXR(f)

        f.DelKeeper.ErrToReturn = fmt.Errorf("not authorized")
        msg := &types.MsgUpdateExchangeRate{
            Authority: f.Authority,
            Operator:  operatorAddr,
            Id:        id,
            Rate:      "9.9",
        }
        _, err := f.MS.UpdateExchangeRate(f.Ctx, msg)
        require.ErrorContains(t, err, "authorization check failed")

        // Rate must be unchanged
        got, err := f.K.ExchangeRates.Get(f.Ctx, id)
        require.NoError(t, err)
        require.Equal(t, "1.5", got.Rate)
        f.RequireNoEvent(types.EventTypeUpdateExchangeRate)
    })

    // --- Negative: XR not found ---
    t.Run("MOD-XR-MSG-2-NEG: exchange rate not found", func(t *testing.T) {
        f := NewFixture(t).SetBlockTime(baseTime)
        msg := &types.MsgUpdateExchangeRate{
            Authority: f.Authority,
            Operator:  operatorAddr,
            Id:        999, // does not exist
            Rate:      "1.0",
        }
        _, err := f.MS.UpdateExchangeRate(f.Ctx, msg)
        require.ErrorContains(t, err, "exchange rate with id 999 not found")
        f.RequireNoEvent(types.EventTypeUpdateExchangeRate)
    })

    // --- Negative: XR not active (state == false) ---
    t.Run("MOD-XR-MSG-2-NEG: exchange rate not active", func(t *testing.T) {
        f := NewFixture(t).SetBlockTime(baseTime)
        id := seedXR(f)

        // Disable the rate via SetExchangeRateState
        _, err := f.MS.SetExchangeRateState(f.Ctx, &types.MsgSetExchangeRateState{
            Authority: f.Authority,
            Id:        id,
        })
        require.NoError(t, err) // state toggled from true → false

        // Reset the event manager so we test UpdateExchangeRate in isolation
        f.Ctx = f.Ctx.WithEventManager(sdk.NewEventManager())

        msg := &types.MsgUpdateExchangeRate{
            Authority: f.Authority,
            Operator:  operatorAddr,
            Id:        id,
            Rate:      "9.9",
        }
        _, err = f.MS.UpdateExchangeRate(f.Ctx, msg)
        require.ErrorContains(t, err, "not active")
        f.RequireNoEvent(types.EventTypeUpdateExchangeRate)
    })

    // --- Negative: XR expired ---
    t.Run("MOD-XR-MSG-2-NEG: exchange rate expired", func(t *testing.T) {
        f := NewFixture(t).SetBlockTime(baseTime)
        id := seedXR(f)

        // Advance past expiry
        f.SetBlockTime(baseTime.Add(dur + time.Second))

        msg := &types.MsgUpdateExchangeRate{
            Authority: f.Authority,
            Operator:  operatorAddr,
            Id:        id,
            Rate:      "9.9",
        }
        _, err := f.MS.UpdateExchangeRate(f.Ctx, msg)
        require.ErrorContains(t, err, "expired")
        f.RequireNoEvent(types.EventTypeUpdateExchangeRate)
    })
}
```

Note: the `sdk.NewEventManager()` import requires adding `sdk "github.com/cosmos/cosmos-sdk/types"` to the import block of this file.

- [ ] **Step 3.2: Run.**

  Run: `go test ./x/xr/keeper/... -run TestMsgUpdateExchangeRate -v`
  Expected: all subtests PASS.

- [ ] **Step 3.3: Commit.**

```bash
git add x/xr/keeper/msg_update_exchange_rate_test.go
git commit -m "test(xr): add fixture-based TestMsgUpdateExchangeRate"
```

---

## Task 4: `msg_set_exchange_rate_state_test.go`

**File:** `x/xr/keeper/msg_set_exchange_rate_state_test.go`

Spec: governance-only, toggles `xr.State = !xr.State`, sets `xr.Updated = blockTime`.

- [ ] **Step 4.1: Create the file.**

```go
package keeper_test

import (
    "fmt"
    "testing"
    "time"

    "github.com/stretchr/testify/require"

    cstypes "github.com/verana-labs/verana/x/cs/types"
    "github.com/verana-labs/verana/x/xr/types"
)

func TestMsgSetExchangeRateState(t *testing.T) {
    baseTime := time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC)
    dur := 30 * 24 * time.Hour

    seedXR := func(f *Fixture) uint64 {
        f.t.Helper()
        resp, err := f.MS.CreateExchangeRate(f.Ctx, &types.MsgCreateExchangeRate{
            Authority:        f.Authority,
            BaseAssetType:    cstypes.PricingAssetType_TU,
            BaseAsset:        "uvna",
            QuoteAssetType:   cstypes.PricingAssetType_FIAT,
            QuoteAsset:       "USD",
            Rate:             "1.5",
            RateScale:        6,
            ValidityDuration: dur,
        })
        require.NoError(f.t, err)
        return resp.Id
    }

    // --- Happy path: active → inactive ---
    t.Run("MOD-XR-MSG-3: toggle active XR to inactive", func(t *testing.T) {
        f := NewFixture(t).SetBlockTime(baseTime)
        id := seedXR(f)

        toggleTime := baseTime.Add(time.Hour)
        f.SetBlockTime(toggleTime)
        f.Ctx = f.Ctx.WithEventManager(sdk.NewEventManager())

        _, err := f.MS.SetExchangeRateState(f.Ctx, &types.MsgSetExchangeRateState{
            Authority: f.Authority,
            Id:        id,
        })
        require.NoError(t, err)

        got, err := f.K.ExchangeRates.Get(f.Ctx, id)
        require.NoError(t, err)
        require.False(t, got.State, "state should be toggled to false")
        require.Equal(t, toggleTime, got.Updated)

        f.RequireEvent(types.EventTypeSetExchangeRateState, map[string]string{
            types.AttributeKeyID:        fmt.Sprintf("%d", id),
            types.AttributeKeyAuthority: f.Authority,
            types.AttributeKeyState:     "false",
        })
        // Invariant: inactive rates are excluded from uniqueness check, still valid.
        f.RequireInvariant()
    })

    // --- Happy path: inactive → active ---
    t.Run("MOD-XR-MSG-3: toggle inactive XR back to active", func(t *testing.T) {
        f := NewFixture(t).SetBlockTime(baseTime)
        id := seedXR(f)

        // First toggle: active → inactive
        f.SetBlockTime(baseTime.Add(time.Hour))
        _, err := f.MS.SetExchangeRateState(f.Ctx, &types.MsgSetExchangeRateState{
            Authority: f.Authority,
            Id:        id,
        })
        require.NoError(t, err)

        // Second toggle: inactive → active
        reactivateTime := baseTime.Add(2 * time.Hour)
        f.SetBlockTime(reactivateTime)
        f.Ctx = f.Ctx.WithEventManager(sdk.NewEventManager())

        _, err = f.MS.SetExchangeRateState(f.Ctx, &types.MsgSetExchangeRateState{
            Authority: f.Authority,
            Id:        id,
        })
        require.NoError(t, err)

        got, err := f.K.ExchangeRates.Get(f.Ctx, id)
        require.NoError(t, err)
        require.True(t, got.State, "state should be toggled back to true")
        require.Equal(t, reactivateTime, got.Updated)

        f.RequireEvent(types.EventTypeSetExchangeRateState, map[string]string{
            types.AttributeKeyState: "true",
        })
        f.RequireInvariant()
    })

    // --- Negative: wrong authority ---
    t.Run("MOD-XR-MSG-3-NEG: wrong authority rejected", func(t *testing.T) {
        f := NewFixture(t).SetBlockTime(baseTime)
        id := seedXR(f)

        badAuthority := "verana1qyq2k3a9q7l7azyqdmxmhvsw87rfhzuq"
        f.Ctx = f.Ctx.WithEventManager(sdk.NewEventManager())

        _, err := f.MS.SetExchangeRateState(f.Ctx, &types.MsgSetExchangeRateState{
            Authority: badAuthority,
            Id:        id,
        })
        require.ErrorContains(t, err, "invalid authority")

        // State must be unchanged
        got, err := f.K.ExchangeRates.Get(f.Ctx, id)
        require.NoError(t, err)
        require.True(t, got.State, "state must not have been toggled")
        f.RequireNoEvent(types.EventTypeSetExchangeRateState)
    })

    // --- Negative: XR not found ---
    t.Run("MOD-XR-MSG-3-NEG: exchange rate not found", func(t *testing.T) {
        f := NewFixture(t).SetBlockTime(baseTime)
        f.Ctx = f.Ctx.WithEventManager(sdk.NewEventManager())

        _, err := f.MS.SetExchangeRateState(f.Ctx, &types.MsgSetExchangeRateState{
            Authority: f.Authority,
            Id:        999, // does not exist
        })
        require.ErrorContains(t, err, "exchange rate with id 999 not found")
        f.RequireNoEvent(types.EventTypeSetExchangeRateState)
    })

    // --- Edge: toggle does not affect other XRs ---
    t.Run("MOD-XR-MSG-3-EDGE: toggle one XR leaves other XR unchanged", func(t *testing.T) {
        f := NewFixture(t).SetBlockTime(baseTime)
        id1 := seedXR(f)

        // Second rate with different pair
        resp2, err := f.MS.CreateExchangeRate(f.Ctx, &types.MsgCreateExchangeRate{
            Authority:        f.Authority,
            BaseAssetType:    cstypes.PricingAssetType_TU,
            BaseAsset:        "uatom",
            QuoteAssetType:   cstypes.PricingAssetType_FIAT,
            QuoteAsset:       "EUR",
            Rate:             "0.9",
            RateScale:        6,
            ValidityDuration: dur,
        })
        require.NoError(t, err)
        id2 := resp2.Id

        // Toggle only id1
        f.SetBlockTime(baseTime.Add(time.Hour))
        f.Ctx = f.Ctx.WithEventManager(sdk.NewEventManager())
        _, err = f.MS.SetExchangeRateState(f.Ctx, &types.MsgSetExchangeRateState{
            Authority: f.Authority,
            Id:        id1,
        })
        require.NoError(t, err)

        got1, err := f.K.ExchangeRates.Get(f.Ctx, id1)
        require.NoError(t, err)
        require.False(t, got1.State)

        got2, err := f.K.ExchangeRates.Get(f.Ctx, id2)
        require.NoError(t, err)
        require.True(t, got2.State, "id2 state should be unaffected")

        f.RequireInvariant()
    })
}
```

Note: `sdk.NewEventManager()` requires adding `sdk "github.com/cosmos/cosmos-sdk/types"` to this file's imports.

- [ ] **Step 4.2: Run.**

  Run: `go test ./x/xr/keeper/... -run TestMsgSetExchangeRateState -v`
  Expected: all subtests PASS.

- [ ] **Step 4.3: Commit.**

```bash
git add x/xr/keeper/msg_set_exchange_rate_state_test.go
git commit -m "test(xr): add fixture-based TestMsgSetExchangeRateState"
```

---

## Task 5: Delete old `keeper_test.go`

- [ ] **Step 5.1: Delete the file.**

```bash
git rm x/xr/keeper/keeper_test.go
```

- [ ] **Step 5.2: Verify full suite still builds and passes.**

  Run: `go test ./x/xr/keeper/... -v -count=1`
  Expected: PASS.

- [ ] **Step 5.3: Commit.**

```bash
git commit -m "test(xr): remove old keeper_test.go (replaced by fixture-based tests)"
```

---

## Task 6: Final pass — vet, lint, coverage

- [ ] **Step 6.1: Run full test suite.**

  Run: `go test ./x/xr/keeper/... -v -count=1`
  Expected: PASS.

- [ ] **Step 6.2: Run with race detector.**

  Run: `go test ./x/xr/keeper/... -race -count=1`
  Expected: PASS, no DATA RACE.

- [ ] **Step 6.3: Run vet.**

  Run: `go vet ./x/xr/keeper/...`
  Expected: no output.

- [ ] **Step 6.4: Run linter.**

  Run: `golangci-lint run ./x/xr/keeper/...`
  Expected: no findings. Fix any reported issues then re-run.

- [ ] **Step 6.5: Coverage check.**

  Run: `go test ./x/xr/keeper/... -cover -count=1`
  Expected: coverage ≥95% on keeper files. If below, run:
  `go test ./x/xr/keeper/ -coverprofile=/tmp/xr.cov && go tool cover -func=/tmp/xr.cov`
  to find uncovered lines and add tests.

- [ ] **Step 6.6: Full repo build check.**

  Run: `go build ./... && go test ./... -count=1`
  Expected: PASS, no regressions.

- [ ] **Step 6.7: Push and open PR.**

```bash
git push -u origin test/step-32-xr-exchange-rates
gh pr create --title "test(xr): fixture-based tests for all 3 XR messages (step 32)" --body "$(cat <<'EOF'
## Summary
- New `fixture_test.go` in `x/xr/keeper/` with Fixture struct, MockDelegationKeeper, RequireExchangeRate, RequireEvent, RequireNoEvent, RequireInvariant, RequireExchangeRateCount
- `TestMsgCreateExchangeRate`: happy path (full struct assertion + event + invariant), two-pair coexistence, negative (bad authority, duration > max, duplicate pair), edge (min/max duration)
- `TestMsgUpdateExchangeRate`: happy paths (rate only, rate+scale, rate+duration), negatives (AUTHZ fail, not found, not active, expired)
- `TestMsgSetExchangeRateState`: happy paths (active→inactive, inactive→active), negatives (bad authority, not found), edge (other XR unaffected)
- Old `keeper_test.go` deleted in this PR
- Spec formula functions: specXRExpires, specXRUpdateExpires — independent of implementation

## Test plan
- [ ] `go test ./x/xr/keeper/... -race -count=1` passes
- [ ] `go test ./x/xr/keeper/... -cover` reports ≥95%
- [ ] `golangci-lint run ./x/xr/keeper/...` clean
- [ ] `go test ./...` (full repo) passes — no regressions
EOF
)"
```

---

## "Done" Criteria — Step 32

- [ ] `fixture_test.go` exists with `Fixture`, `MockDelegationKeeper`, all helper methods, `RequireInvariant`.
- [ ] `TestMsgCreateExchangeRate`: happy path full struct assertion + event + invariant, negatives for bad authority / duration > max / duplicate pair, edge min/max duration.
- [ ] `TestMsgUpdateExchangeRate`: happy paths for rate / rate+scale / rate+duration updates, negatives for AUTHZ fail / not found / not active / expired.
- [ ] `TestMsgSetExchangeRateState`: happy paths for toggle active↔inactive, negatives for bad authority / not found, edge isolation across multiple XRs.
- [ ] Old `keeper_test.go` deleted in the same PR.
- [ ] `go test ./x/xr/keeper/... -race -count=1` passes.
- [ ] Coverage ≥95% on `x/xr/keeper/`.
- [ ] `go test ./...`, `go vet ./...`, `golangci-lint run` all pass.

---

## Self-Review Notes

- **Spec formula independence:** `specXRExpires` and `specXRUpdateExpires` are pure functions with no keeper imports. They encode the spec formula directly. If the implementation uses a different calculation, the test breaks — that is the purpose.
- **No StatefulBankMock needed:** XR module has no bank operations. Fixture does not embed one.
- **MockDelegationKeeper.ErrToReturn:** Reset to nil between subtests by virtue of `NewFixture(t)` creating a fresh instance per subtest. Each `t.Run` that tests AUTHZ failure sets it on its own fixture.
- **Event manager reset:** When a test needs to inspect events from a single operation (e.g., after a setup operation already emitted events), `f.Ctx = f.Ctx.WithEventManager(sdk.NewEventManager())` is called before the message under test.
- **buildPairKeyForTest:** Replicates the pair key formula from the keeper without calling the keeper's internal function directly. If the keeper changes its key format, this test helper must be updated separately, which is intentional (spec says the invariant must be independently verified).
- **Invariant scope:** XR invariant checks only active rates (State == true). Inactive rates may share a pair key — the spec only says at most one *active* rate per pair.
