# Step 1: TD ReclaimTrustDepositYield — Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Replace the legacy `TestMsgReclaimTrustDepositYield` test with a fixture-based test that verifies exact balance deltas, full struct comparison, event emission, and the TD module invariant.

**Architecture:** Creates `x/td/keeper/fixture_test.go` with the shared `Fixture` struct used by all TD steps. Adds `TrustdepositKeeperWithStatefulBank` to `testutil/keeper/trustdeposit.go` so the fixture can wire a `StatefulBankMock` instead of the legacy no-op mock. The new `reclaim_trust_deposit_yield_test.go` uses spec formula functions defined at the top of the file; the old `TestMsgReclaimTrustDepositYield` test block is deleted in the same PR.

**Tech Stack:** Go 1.24, Cosmos SDK v0.50.x, testify/require, StatefulBankMock from testutil/keeper/bank.go

---

## Pre-flight

- [ ] **Worktree.** Create an isolated worktree for this step. Branch name: `test/step-1-td-reclaim-yield`.
- [ ] **Gate check.** Confirm step 0 is merged: `testutil/keeper/bank.go` must exist and `go build ./testutil/keeper/...` must pass.
- [ ] **Sanity check.**

  Run: `go build ./... && go vet ./...`
  Expected: no output, exit 0.

---

## File Structure

- **Create:** `x/td/keeper/fixture_test.go` — shared `Fixture` struct for all TD keeper tests.
- **Create:** `x/td/keeper/reclaim_trust_deposit_yield_test.go` — new fixture-based tests.
- **Modify:** `testutil/keeper/trustdeposit.go` — add `TrustdepositKeeperWithStatefulBank`.
- **Delete (in same PR):** the `TestMsgReclaimTrustDepositYield` and `TestMsgReclaimTrustDepositYieldEdgeCases` test functions from `x/td/keeper/msg_server_test.go`.

---

## Task 1: Add `TrustdepositKeeperWithStatefulBank` to testutil

**File:** `testutil/keeper/trustdeposit.go`

- [ ] **Step 1.1: Add the new constructor.**

Append to `testutil/keeper/trustdeposit.go`:

```go
// TrustdepositKeeperWithStatefulBank creates a TD keeper wired to a
// StatefulBankMock instead of the legacy no-op MockBankKeeper. Use this
// for all fixture-based TD tests (issue #292 test overhaul).
// The returned MockDelegationKeeper can be configured to return errors via ErrToReturn.
func TrustdepositKeeperWithStatefulBank(t testing.TB, bank *StatefulBankMock) (keeper.Keeper, sdk.Context, *MockDelegationKeeper) {
	t.Helper()
	storeKey := storetypes.NewKVStoreKey(types.StoreKey)

	db := dbm.NewMemDB()
	stateStore := store.NewCommitMultiStore(db, log.NewNopLogger(), metrics.NewNoOpMetrics())
	stateStore.MountStoreWithDB(storeKey, storetypes.StoreTypeIAVL, db)
	require.NoError(t, stateStore.LoadLatestVersion())

	registry := codectypes.NewInterfaceRegistry()
	cdc := codec.NewProtoCodec(registry)
	authority := authtypes.NewModuleAddress(govtypes.ModuleName)
	mintKeeper := NewMockMintKeeper()
	delegationKeeper := &MockDelegationKeeper{}

	k := keeper.NewKeeper(
		cdc,
		runtime.NewKVStoreService(storeKey),
		log.NewNopLogger(),
		authority.String(),
		bank,
		mintKeeper,
		delegationKeeper,
	)

	ctx := sdk.NewContext(stateStore, cmtproto.Header{}, false, log.NewNopLogger())

	if err := k.SetParams(ctx, types.DefaultParams()); err != nil {
		panic(err)
	}

	return k, ctx, delegationKeeper
}
```

- [ ] **Step 1.2: Verify build.**

  Run: `go build ./testutil/keeper/...`
  Expected: no output, exit 0.

- [ ] **Step 1.3: Commit.**

```bash
git add testutil/keeper/trustdeposit.go
git commit -m "test(td): add TrustdepositKeeperWithStatefulBank constructor"
```

---

## Task 2: Create `x/td/keeper/fixture_test.go`

**File:** `x/td/keeper/fixture_test.go`

- [ ] **Step 2.1: Create the fixture file.**

```go
package keeper_test

import (
	"testing"
	"time"

	sdk "github.com/cosmos/cosmos-sdk/types"
	authtypes "github.com/cosmos/cosmos-sdk/x/auth/types"
	govtypes "github.com/cosmos/cosmos-sdk/x/gov/types"
	"github.com/stretchr/testify/require"

	keepertest "github.com/verana-labs/verana/testutil/keeper"
	"github.com/verana-labs/verana/x/td/keeper"
	"github.com/verana-labs/verana/x/td/types"
)

// Fixture is the shared test environment for all TD keeper tests.
// It owns the StatefulBankMock so balance assertions are independent
// of the implementation under test.
type Fixture struct {
	t         *testing.T
	K         keeper.Keeper
	MS        types.MsgServer
	Ctx       sdk.Context
	Bank      *keepertest.StatefulBankMock
	DelKeeper *keepertest.MockDelegationKeeper
}

// NewFixture creates a fresh TD test environment wired to a StatefulBankMock.
func NewFixture(t *testing.T) *Fixture {
	t.Helper()
	bank := keepertest.NewStatefulBankMock(map[string]sdk.AccAddress{
		types.ModuleName: authtypes.NewModuleAddress(types.ModuleName),
	})
	k, ctx, delKeeper := keepertest.TrustdepositKeeperWithStatefulBank(t, bank)
	return &Fixture{t: t, K: k, MS: keeper.NewMsgServerImpl(k), Ctx: ctx, Bank: bank, DelKeeper: delKeeper}
}

// SetBalance sets the balance for a bech32 address string (convenience wrapper).
// Converts the bech32 string to sdk.AccAddress before calling StatefulBankMock.SetBalance.
func (f *Fixture) SetBalance(addr, denom string, amount int64) {
	f.t.Helper()
	f.Bank.SetBalance(sdk.MustAccAddressFromBech32(addr), denom, amount)
}

// RequireBalanceDelta asserts that the balance of addr changed by exactly delta
// relative to the baseline captured by the most recent SetBalance call.
func (f *Fixture) RequireBalanceDelta(addr, denom string, delta int64) {
	f.t.Helper()
	f.Bank.RequireBalanceDelta(f.t, sdk.MustAccAddressFromBech32(addr), denom, delta)
}

// RequireNoBalanceChange asserts that no balance denomination for addr has changed
// relative to SetBalance baselines. Asserts delta == 0 for uvna (the only denom used in TD).
func (f *Fixture) RequireNoBalanceChange(addr string) {
	f.t.Helper()
	f.Bank.RequireBalanceDelta(f.t, sdk.MustAccAddressFromBech32(addr), types.BondDenom, 0)
}

// RequireTrustDeposit asserts that the stored TrustDeposit for addr
// equals want via full require.Equal (all fields compared).
func (f *Fixture) RequireTrustDeposit(addr string, want types.TrustDeposit) {
	f.t.Helper()
	got, err := f.K.TrustDeposit.Get(f.Ctx, addr)
	require.NoError(f.t, err)
	require.Equal(f.t, want, got)
}

// RequireEvent asserts that an event of the given type was emitted and
// that every key-value pair in attrs appears as an attribute on that event.
func (f *Fixture) RequireEvent(eventType string, attrs map[string]string) {
	f.t.Helper()
	events := f.Ctx.EventManager().Events()
	for _, e := range events {
		if e.Type == eventType {
			for k, v := range attrs {
				found := false
				for _, a := range e.Attributes {
					if a.Key == k && a.Value == v {
						found = true
						break
					}
				}
				require.True(f.t, found, "event %s missing attribute %s=%s", eventType, k, v)
			}
			return
		}
	}
	require.Fail(f.t, "event not found", "expected event type %s", eventType)
}

// RequireInvariant checks the TD module invariant:
// sum of all TrustDeposit.Deposit fields == TD module account balance.
func (f *Fixture) RequireInvariant() {
	f.t.Helper()
	var totalDeposit uint64
	_ = f.K.TrustDeposit.Walk(f.Ctx, nil, func(_ string, td types.TrustDeposit) (bool, error) {
		totalDeposit += td.Deposit
		return false, nil
	})
	moduleBalance := f.Bank.BalanceOf(authtypes.NewModuleAddress(types.ModuleName), types.BondDenom)
	require.Equal(f.t, int64(totalDeposit), moduleBalance,
		"TD invariant violated: totalDeposit(%d) != moduleBalance(%d)", totalDeposit, moduleBalance)
}

// SetBlockTime sets the context block time.
func (f *Fixture) SetBlockTime(t time.Time) { f.Ctx = f.Ctx.WithBlockTime(t) }

// AdvanceTime advances the context block time by d.
func (f *Fixture) AdvanceTime(d time.Duration) {
	f.Ctx = f.Ctx.WithBlockTime(f.Ctx.BlockTime().Add(d))
}

// GovAuthority returns the governance module address string, which is used as
// the keeper's authority for messages like SlashTrustDeposit.
// Shared across all TD keeper test files (same package).
func GovAuthority() string {
	return authtypes.NewModuleAddress(govtypes.ModuleName).String()
}
```

- [ ] **Step 2.2: Verify build.**

  Run: `go build ./x/td/keeper/...`
  Expected: no output, exit 0.

- [ ] **Step 2.3: Commit.**

```bash
git add x/td/keeper/fixture_test.go
git commit -m "test(td): add shared Fixture struct for TD keeper tests"
```

---

## Task 3: Write `reclaim_trust_deposit_yield_test.go`

**File:** `x/td/keeper/reclaim_trust_deposit_yield_test.go`

The spec formula for `ReclaimTrustDepositYield` is:
- `claimed = td.Claimable` (full drain; no partial claim)
- `sharesToReduce = claimed / shareValue`
- `td.Claimable = 0`
- `td.Share = td.Share - sharesToReduce`
- Bank: `SendCoinsFromModuleToAccount(td, corporation, claimed uvna)`

Note: `Deposit` is NOT decremented on yield reclaim — yield comes from the yield pool
allocation that was already added to `Claimable`, not from `Deposit`.

- [ ] **Step 3.1: Create the test file.**

```go
package keeper_test

import (
	"testing"
	"time"

	"cosmossdk.io/math"
	sdk "github.com/cosmos/cosmos-sdk/types"
	authtypes "github.com/cosmos/cosmos-sdk/x/auth/types"
	"github.com/stretchr/testify/require"

	"github.com/verana-labs/verana/x/td/types"
)

// GovAuthority is defined in fixture_test.go (shared across all TD test files).

// ============================================================================
// Spec formula functions — independent of the implementation
// ============================================================================

// specReclaimClaimed returns the amount transferred to the corporation on a
// successful ReclaimTrustDepositYield call per [MOD-TD-MSG-2-3]:
// the full claimable balance is drained.
func specReclaimClaimed(claimable uint64) uint64 {
	return claimable
}

// specReclaimShareReduction returns the share reduction applied to td.Share
// per [MOD-TD-MSG-2-3]: sharesToReduce = claimed / shareValue.
// Uses the same LegacyDec math the keeper uses.
func specReclaimShareReduction(claimed uint64, shareValue math.LegacyDec) math.LegacyDec {
	return math.LegacyNewDec(int64(claimed)).Quo(shareValue)
}

// specReclaimShareAfter returns the expected td.Share after reclaim.
func specReclaimShareAfter(shareBefore math.LegacyDec, claimed uint64, shareValue math.LegacyDec) math.LegacyDec {
	return shareBefore.Sub(specReclaimShareReduction(claimed, shareValue))
}

// ============================================================================
// TestMsgReclaimTrustDepositYield
// ============================================================================

func TestMsgReclaimTrustDepositYield(t *testing.T) {

	// --- Happy path: standard yield reclaim ---
	t.Run("MOD-TD-MSG-2: valid reclaim drains claimable and reduces shares", func(t *testing.T) {
		f := NewFixture(t)
		now := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
		f.SetBlockTime(now)

		corpAddr := sdk.AccAddress([]byte("corp_reclaim_ok__1"))
		corp := corpAddr.String()
		modAddr := authtypes.NewModuleAddress(types.ModuleName).String()

		shareValue := math.LegacyMustNewDecFromStr("1.5")
		params := types.DefaultParams()
		params.TrustDepositShareValue = shareValue
		require.NoError(t, f.K.SetParams(f.Ctx, params))

		// 500 uvna claimable yield; deposit stays separate
		td := types.TrustDeposit{
			Corporation: corp,
			Share:       math.LegacyNewDec(1000),
			Deposit:     1000,
			Claimable:   500,
		}
		require.NoError(t, f.K.TrustDeposit.Set(f.Ctx, corp, td))

		// Module account must hold at least the claimable amount
		f.SetBalance(modAddr, types.BondDenom, 1500) // 1000 deposit + 500 claimable
		f.SetBalance(corp, types.BondDenom, 0)

		claimed := specReclaimClaimed(td.Claimable)
		expectedShare := specReclaimShareAfter(td.Share, claimed, shareValue)

		resp, err := f.MS.ReclaimTrustDepositYield(f.Ctx, &types.MsgReclaimTrustDepositYield{
			Corporation: corp,
			Operator:    corp,
		})
		require.NoError(t, err)
		require.Equal(t, claimed, resp.ClaimedAmount)

		// Balance assertions: corp received exactly claimed; module lost exactly claimed
		f.RequireBalanceDelta(corp, types.BondDenom, int64(claimed))
		f.RequireBalanceDelta(modAddr, types.BondDenom, -int64(claimed))

		// Full struct assertion
		f.RequireTrustDeposit(corp, types.TrustDeposit{
			Corporation: corp,
			Share:       expectedShare,
			Deposit:     1000,  // deposit unchanged
			Claimable:   0,     // drained to zero
		})

		// Event assertion
		f.RequireEvent(types.EventTypeReclaimTrustDepositYield, map[string]string{
			types.AttributeKeyAccount:      corp,
			types.AttributeKeyClaimedYield: "500",
		})

		// Invariant: sum of deposit fields == module balance
		// After reclaim, module balance is 1500-500=1000 == deposit 1000
		f.RequireInvariant()
	})

	// --- Happy path: previously slashed but fully repaid (slashed_deposit == 0) ---
	t.Run("MOD-TD-MSG-2: slashed_deposit=0 (fully repaid) allows yield reclaim", func(t *testing.T) {
		f := NewFixture(t)
		corpAddr := sdk.AccAddress([]byte("corp_repaid_ok___1"))
		corp := corpAddr.String()
		modAddr := authtypes.NewModuleAddress(types.ModuleName).String()

		shareValue := math.LegacyMustNewDecFromStr("1.0")
		params := types.DefaultParams()
		params.TrustDepositShareValue = shareValue
		require.NoError(t, f.K.SetParams(f.Ctx, params))

		td := types.TrustDeposit{
			Corporation:    corp,
			Share:          math.LegacyNewDec(1000),
			Deposit:        1000,
			Claimable:      200,
			SlashedDeposit: 0,   // fully repaid
			RepaidDeposit:  300, // cumulative history
		}
		require.NoError(t, f.K.TrustDeposit.Set(f.Ctx, corp, td))

		f.SetBalance(modAddr, types.BondDenom, 1200)
		f.SetBalance(corp, types.BondDenom, 0)

		claimed := specReclaimClaimed(td.Claimable) // 200
		expectedShare := specReclaimShareAfter(td.Share, claimed, shareValue)

		resp, err := f.MS.ReclaimTrustDepositYield(f.Ctx, &types.MsgReclaimTrustDepositYield{
			Corporation: corp,
			Operator:    corp,
		})
		require.NoError(t, err)
		require.Equal(t, claimed, resp.ClaimedAmount)

		f.RequireBalanceDelta(corp, types.BondDenom, 200)
		f.RequireBalanceDelta(modAddr, types.BondDenom, -200)

		f.RequireTrustDeposit(corp, types.TrustDeposit{
			Corporation:    corp,
			Share:          expectedShare,
			Deposit:        1000,
			Claimable:      0,
			SlashedDeposit: 0,
			RepaidDeposit:  300,
		})

		f.RequireEvent(types.EventTypeReclaimTrustDepositYield, map[string]string{
			types.AttributeKeyAccount:      corp,
			types.AttributeKeyClaimedYield: "200",
		})

		f.RequireInvariant()
	})

	// --- Negative cases: one per precondition ---

	t.Run("MOD-TD-MSG-2-2: fails if authorization check fails", func(t *testing.T) {
		f := NewFixture(t)
		corpAddr := sdk.AccAddress([]byte("corp_authz_fail__1"))
		corp := corpAddr.String()

		require.NoError(t, f.K.SetParams(f.Ctx, types.DefaultParams()))
		td := types.TrustDeposit{
			Corporation: corp,
			Share:       math.LegacyNewDec(1000),
			Deposit:     1000,
			Claimable:   500,
		}
		require.NoError(t, f.K.TrustDeposit.Set(f.Ctx, corp, td))

		f.SetBalance(corp, types.BondDenom, 0)

		// Force the delegation keeper to reject
		f.DelKeeper.ErrToReturn = fmt.Errorf("mock: operator not authorized")

		_, err := f.MS.ReclaimTrustDepositYield(f.Ctx, &types.MsgReclaimTrustDepositYield{
			Corporation: corp,
			Operator:    sdk.AccAddress([]byte("bad_operator_____1")).String(),
		})
		require.ErrorContains(t, err, "authorization check failed")
		f.RequireNoBalanceChange(corp)
	})

	t.Run("MOD-TD-MSG-2-2-1: fails if trust deposit not found", func(t *testing.T) {
		f := NewFixture(t)
		corp := sdk.AccAddress([]byte("corp_notfound____1")).String()

		f.SetBalance(corp, types.BondDenom, 0)

		_, err := f.MS.ReclaimTrustDepositYield(f.Ctx, &types.MsgReclaimTrustDepositYield{
			Corporation: corp,
			Operator:    corp,
		})
		require.ErrorContains(t, err, "trust deposit not found")
		f.RequireNoBalanceChange(corp)
	})

	t.Run("MOD-TD-MSG-2-2-1: fails if slashed_deposit > 0 (unrepaid slash)", func(t *testing.T) {
		f := NewFixture(t)
		corpAddr := sdk.AccAddress([]byte("corp_slashed_blk_1"))
		corp := corpAddr.String()

		require.NoError(t, f.K.SetParams(f.Ctx, types.DefaultParams()))
		td := types.TrustDeposit{
			Corporation:    corp,
			Share:          math.LegacyNewDec(1000),
			Deposit:        1000,
			Claimable:      500,
			SlashedDeposit: 100, // outstanding slash — must block reclaim
		}
		require.NoError(t, f.K.TrustDeposit.Set(f.Ctx, corp, td))

		f.SetBalance(corp, types.BondDenom, 0)

		_, err := f.MS.ReclaimTrustDepositYield(f.Ctx, &types.MsgReclaimTrustDepositYield{
			Corporation: corp,
			Operator:    corp,
		})
		require.ErrorContains(t, err, "deposit has been slashed and not repaid")
		f.RequireNoBalanceChange(corp)
	})

	t.Run("MOD-TD-MSG-2-2-1: fails if claimable == 0", func(t *testing.T) {
		f := NewFixture(t)
		corpAddr := sdk.AccAddress([]byte("corp_no_yield____1"))
		corp := corpAddr.String()

		require.NoError(t, f.K.SetParams(f.Ctx, types.DefaultParams()))
		td := types.TrustDeposit{
			Corporation: corp,
			Share:       math.LegacyNewDec(1000),
			Deposit:     1000,
			Claimable:   0,
		}
		require.NoError(t, f.K.TrustDeposit.Set(f.Ctx, corp, td))

		f.SetBalance(corp, types.BondDenom, 0)

		_, err := f.MS.ReclaimTrustDepositYield(f.Ctx, &types.MsgReclaimTrustDepositYield{
			Corporation: corp,
			Operator:    corp,
		})
		require.ErrorContains(t, err, "no claimable yield")
		f.RequireNoBalanceChange(corp)
	})

	// --- Edge cases ---

	t.Run("edge: exact minimum claimable (1 uvna)", func(t *testing.T) {
		f := NewFixture(t)
		corpAddr := sdk.AccAddress([]byte("corp_min_claim___1"))
		corp := corpAddr.String()
		modAddr := authtypes.NewModuleAddress(types.ModuleName).String()

		shareValue := math.LegacyMustNewDecFromStr("1.0")
		params := types.DefaultParams()
		params.TrustDepositShareValue = shareValue
		require.NoError(t, f.K.SetParams(f.Ctx, params))

		td := types.TrustDeposit{
			Corporation: corp,
			Share:       math.LegacyNewDec(1001),
			Deposit:     1000,
			Claimable:   1,
		}
		require.NoError(t, f.K.TrustDeposit.Set(f.Ctx, corp, td))

		f.SetBalance(modAddr, types.BondDenom, 1001)
		f.SetBalance(corp, types.BondDenom, 0)

		resp, err := f.MS.ReclaimTrustDepositYield(f.Ctx, &types.MsgReclaimTrustDepositYield{
			Corporation: corp,
			Operator:    corp,
		})
		require.NoError(t, err)
		require.Equal(t, uint64(1), resp.ClaimedAmount)

		f.RequireBalanceDelta(corp, types.BondDenom, 1)
		f.RequireBalanceDelta(modAddr, types.BondDenom, -1)
		f.RequireInvariant()
	})
}
```

Note: the `fmt` import is needed for the authz failure test. Add it to the import block.

- [ ] **Step 3.2: Confirm `fmt` import is present.**

The file uses `fmt.Errorf` for the authz failure test case. Ensure the import block reads:

```go
import (
	"fmt"
	"testing"
	"time"

	"cosmossdk.io/math"
	sdk "github.com/cosmos/cosmos-sdk/types"
	authtypes "github.com/cosmos/cosmos-sdk/x/auth/types"
	"github.com/stretchr/testify/require"

	"github.com/verana-labs/verana/x/td/types"
)
```

(`GovAuthority()` is defined in `fixture_test.go` in the same package — no `govtypes` import needed here.)

- [ ] **Step 3.3: Verify build.**

  Run: `go build ./x/td/keeper/...`
  Expected: no output, exit 0.

- [ ] **Step 3.4: Run new tests.**

  Run: `go test ./x/td/keeper/... -run TestMsgReclaimTrustDepositYield -v -count=1`
  Expected: all subtests PASS.

---

## Task 4: Delete old tests for this message

**File:** `x/td/keeper/msg_server_test.go`

- [ ] **Step 4.1: Delete `TestMsgReclaimTrustDepositYield` test function.**

Remove the entire `func TestMsgReclaimTrustDepositYield(t *testing.T)` block (lines ~31–153 in the current file). Also remove the `TestMsgReclaimTrustDepositYieldEdgeCases` function block.

Keep all other test functions (`TestMsgServer`, `TestAdjustTrustDeposit`, `TestUtilityFunctions`, `TestMsgSlashTrustDeposit`, `TestMsgRepaySlashedTrustDeposit`, `TestMsgRepaySlashedTrustDepositAuthz`, `TestBurnEcosystemSlashedTrustDeposit`, `TestAdjustTrustDepositOnBehalf`, `TestAdjustTrustDepositSlashedGuard`) intact — they will be migrated in steps 2–4.

Also remove `govAuthority()` and `defaultTestParams()` helpers from `msg_server_test.go` only if they are no longer referenced by remaining tests. If they are still used, keep them.

- [ ] **Step 4.2: Verify build and full test suite.**

  Run: `go test ./x/td/keeper/... -count=1 -v`
  Expected: no compilation errors, all remaining tests PASS. The deleted tests must not appear.

- [ ] **Step 4.3: Commit.**

```bash
git add x/td/keeper/reclaim_trust_deposit_yield_test.go
git add x/td/keeper/msg_server_test.go
git commit -m "test(td): add fixture-based ReclaimTrustDepositYield tests, delete legacy"
```

---

## Task 5: Final pass

- [ ] **Step 5.1: Run the full TD keeper test suite.**

  Run: `go test ./x/td/keeper/... -v -count=1`
  Expected: PASS.

- [ ] **Step 5.2: Run race detector.**

  Run: `go test ./x/td/keeper/... -race -count=1`
  Expected: PASS, no DATA RACE reports.

- [ ] **Step 5.3: Run vet and lint.**

  Run: `go vet ./x/td/keeper/... && golangci-lint run ./x/td/keeper/...`
  Expected: no output.

- [ ] **Step 5.4: Coverage check.**

  Run: `go test ./x/td/keeper/ -cover -count=1`
  Expected: coverage ≥95%. If below, run `go test ./x/td/keeper/ -coverprofile=/tmp/td.cov && go tool cover -func=/tmp/td.cov | grep -v "_test.go"` to find uncovered lines.

- [ ] **Step 5.5: Full repo sanity.**

  Run: `go build ./... && go test ./... -count=1`
  Expected: PASS.

- [ ] **Step 5.6: Push and open PR.**

```bash
git push -u origin test/step-1-td-reclaim-yield
gh pr create --title "test(td): fixture-based ReclaimTrustDepositYield tests (step 1)" --body "$(cat <<'EOF'
## Summary
- Adds `TrustdepositKeeperWithStatefulBank` to `testutil/keeper/trustdeposit.go`
- Creates `x/td/keeper/fixture_test.go` with the shared `Fixture` struct (used by steps 1–4)
- Replaces legacy `TestMsgReclaimTrustDepositYield` with fixture-based test in `reclaim_trust_deposit_yield_test.go`
- Spec formula functions computed independently of the implementation
- All preconditions tested: authz failure, TD not found, unrepaid slash, no claimable yield
- Invariant (totalDeposit == module balance) checked on every happy path

## Test plan
- [ ] `go test ./x/td/keeper/... -race -count=1` passes
- [ ] `go test ./x/td/keeper/ -cover` reports ≥95%
- [ ] `golangci-lint run ./x/td/keeper/...` clean
- [ ] `go test ./...` (full repo) passes — no regressions
EOF
)"
```

---

## "Done" Criteria — Step 1

- [ ] `testutil/keeper/trustdeposit.go` exports `TrustdepositKeeperWithStatefulBank`.
- [ ] `x/td/keeper/fixture_test.go` exists with `Fixture`, `NewFixture`, and all assertion helpers.
- [ ] `x/td/keeper/reclaim_trust_deposit_yield_test.go` covers: 2 happy paths, 4 negative cases, 1 edge case.
- [ ] Every happy path: balance delta (corp +claimed, module -claimed) + full struct + event + invariant.
- [ ] Every negative case: `ErrorContains` + `RequireNoBalanceChange`.
- [ ] `TestMsgReclaimTrustDepositYield` and `TestMsgReclaimTrustDepositYieldEdgeCases` deleted from `msg_server_test.go`.
- [ ] `go test ./x/td/keeper/... -race -count=1` passes.
- [ ] Coverage ≥95% on `x/td/keeper/`.
