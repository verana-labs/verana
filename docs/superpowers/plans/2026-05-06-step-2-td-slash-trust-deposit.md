# Step 2: TD SlashTrustDeposit — Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Replace the legacy `TestMsgSlashTrustDeposit` test with a fixture-based test that verifies exact state mutation, no bank transfer occurs (coins stay locked in module account), event emission, and the TD module invariant.

**Architecture:** Extends the `Fixture` struct created in step 1 (no changes needed — it already covers all required helpers). Creates a new `slash_trust_deposit_test.go` file with spec formula functions for slash math. `SlashTrustDeposit` does NOT move coins — it only updates bookkeeping fields; the invariant must hold because `Deposit` is decremented while the underlying module-account coins stay. Deletes `TestMsgSlashTrustDeposit` from `msg_server_test.go` in the same PR.

**Tech Stack:** Go 1.24, Cosmos SDK v0.50.x, testify/require, StatefulBankMock from testutil/keeper/bank.go

---

## Pre-flight

- [ ] **Gate check.** Step 1 must be merged: `x/td/keeper/fixture_test.go` must exist and `go build ./x/td/keeper/...` must pass.
- [ ] **Worktree.** Create an isolated worktree. Branch name: `test/step-2-td-slash-trust-deposit`.
- [ ] **Sanity check.**

  Run: `go build ./... && go vet ./...`
  Expected: no output, exit 0.

---

## File Structure

- **Create:** `x/td/keeper/slash_trust_deposit_test.go` — fixture-based tests for `SlashTrustDeposit`.
- **Modify (delete block):** `x/td/keeper/msg_server_test.go` — remove `TestMsgSlashTrustDeposit`.

---

## Task 1: Write `slash_trust_deposit_test.go`

**File:** `x/td/keeper/slash_trust_deposit_test.go`

Key implementation facts from `msg_server.go`:
- Authority check: `ms.Keeper.authority != msg.Authority` → error
- `msg.Deposit.IsZero() || msg.Deposit.IsNegative()` → error
- `!msg.Deposit.IsUint64()` → error (uint64 overflow guard)
- TrustDeposit must exist for `msg.Corporation`
- `td.Deposit` must be >= `msg.Deposit`
- Execution: `td.Deposit -= amount`, `td.SlashedDeposit += amount`, `td.Share -= amount/shareValue`, `td.LastSlashed = &now`, `td.SlashCount++`
- No bank transfer: coins remain in module account (slashed coins burned later by BurnEcosystem or returned by Repay)

Invariant note: After slash, `td.Deposit` decreases by `amount` but the module account balance is unchanged (no bank call). The invariant `totalDeposit == moduleBalance` therefore requires the module balance to be set to match `totalDeposit` in the test, OR we must understand that after a slash the module holds `totalDeposit + totalSlashedDeposit` coins. Re-reading the design: the invariant is `sum(deposit) == module balance` where deposit is only the un-slashed portion. After a slash, `deposit` drops by `amount` but the coins stay in the module. This means the invariant as defined (`sum deposit == module balance`) will FAIL after a slash if module balance is not also reduced. This is by design — the slashed coins are still in the module but are no longer tracked in `deposit`. The correct invariant for step 2 is:

`sum(deposit) + sum(slashed_deposit) == module balance`

This is a more complete invariant. We will implement a TD-specific `RequireFullInvariant` helper inline in this test file that checks the full accounting invariant (deposit + slashed == module balance), and the fixture's `RequireInvariant` (deposit-only) will match only after BurnEcosystem or Repay resolves the slashed coins.

For the slash test happy path, call `f.RequireFullTDInvariant()` defined locally in this file.

- [ ] **Step 1.1: Create the test file.**

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

// GovAuthority() is defined in fixture_test.go and shared across all TD test files.

// ============================================================================
// Spec formula functions — independent of the implementation
// ============================================================================

// specSlashDepositAfter returns td.Deposit after a slash per [MOD-TD-MSG-5-3].
func specSlashDepositAfter(depositBefore, amount uint64) uint64 {
	return depositBefore - amount
}

// specSlashSlashedDepositAfter returns td.SlashedDeposit after a slash.
func specSlashSlashedDepositAfter(slashedBefore, amount uint64) uint64 {
	return slashedBefore + amount
}

// specSlashSlashCountAfter returns td.SlashCount after a slash.
func specSlashSlashCountAfter(countBefore uint64) uint64 {
	return countBefore + 1
}

// specSlashShareAfter returns td.Share after a slash per [MOD-TD-MSG-5-3]:
// shareReduction = amount / shareValue; share = shareBefore - shareReduction.
func specSlashShareAfter(shareBefore math.LegacyDec, amount uint64, shareValue math.LegacyDec) math.LegacyDec {
	shareReduction := math.LegacyNewDecFromInt(math.NewInt(int64(amount))).Quo(shareValue)
	return shareBefore.Sub(shareReduction)
}

// ============================================================================
// Full TD accounting invariant for slash tests:
// sum(deposit) + sum(slashed_deposit) == module account balance.
// After a slash, coins stay in the module account but move from deposit to
// slashed_deposit tracking; the module balance is therefore unchanged.
// ============================================================================

func requireFullTDInvariant(t *testing.T, f *Fixture) {
	t.Helper()
	var totalDeposit, totalSlashed uint64
	_ = f.K.TrustDeposit.Walk(f.Ctx, nil, func(_ string, td types.TrustDeposit) (bool, error) {
		totalDeposit += td.Deposit
		totalSlashed += td.SlashedDeposit
		return false, nil
	})
	moduleBalance := f.Bank.BalanceOf(authtypes.NewModuleAddress(types.ModuleName), types.BondDenom)
	require.Equal(t, int64(totalDeposit+totalSlashed), moduleBalance,
		"TD full invariant violated: deposit(%d)+slashed(%d)=%d != moduleBalance(%d)",
		totalDeposit, totalSlashed, totalDeposit+totalSlashed, moduleBalance)
}

// GovAuthority() is defined in fixture_test.go (shared across all TD test files in this package).
// Use GovAuthority() — do NOT redefine it here.
// Replace all GovAuthority() calls below with GovAuthority() when implementing.

// ============================================================================
// TestMsgSlashTrustDeposit
// ============================================================================

func TestMsgSlashTrustDeposit(t *testing.T) {

	// --- Happy path: standard slash ---
	t.Run("MOD-TD-MSG-5: valid slash reduces deposit and increments slash fields", func(t *testing.T) {
		f := NewFixture(t)
		now := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
		f.SetBlockTime(now)

		corpAddr := sdk.AccAddress([]byte("corp_slash_ok____1"))
		corp := corpAddr.String()
		modAddr := authtypes.NewModuleAddress(types.ModuleName).String()

		shareValue := math.LegacyMustNewDecFromStr("1.0")
		params := types.DefaultParams()
		params.TrustDepositShareValue = shareValue
		require.NoError(t, f.K.SetParams(f.Ctx, params))

		td := types.TrustDeposit{
			Corporation: corp,
			Share:       math.LegacyNewDec(1000),
			Deposit:     1000,
		}
		require.NoError(t, f.K.TrustDeposit.Set(f.Ctx, corp, td))

		// Module account holds the deposit coins (no claimable yet)
		f.SetBalance(modAddr, types.BondDenom, 1000)
		// Corporation has no balance; slash does not send coins anywhere
		f.SetBalance(corp, types.BondDenom, 0)

		slashAmount := uint64(300)

		_, err := f.MS.SlashTrustDeposit(f.Ctx, &types.MsgSlashTrustDeposit{
			Authority:   GovAuthority(),
			Corporation: corp,
			Deposit:     math.NewInt(int64(slashAmount)),
			Reason:      "test governance slash",
		})
		require.NoError(t, err)

		// No bank transfer: module balance must be unchanged
		f.RequireBalanceDelta(modAddr, types.BondDenom, 0)
		f.RequireNoBalanceChange(corp)

		// Full state assertion
		expectedShare := specSlashShareAfter(td.Share, slashAmount, shareValue)
		f.RequireTrustDeposit(corp, types.TrustDeposit{
			Corporation:    corp,
			Share:          expectedShare,
			Deposit:        specSlashDepositAfter(td.Deposit, slashAmount),
			SlashedDeposit: specSlashSlashedDepositAfter(td.SlashedDeposit, slashAmount),
			SlashCount:     specSlashSlashCountAfter(td.SlashCount),
			LastSlashed:    &now,
		})

		// Event assertion
		f.RequireEvent(types.EventTypeSlashTrustDeposit, map[string]string{
			types.AttributeKeyAccount:    corp,
			types.AttributeKeyAmount:     "300",
			types.AttributeKeySlashCount: "1",
		})

		// Full accounting invariant (deposit+slashed == module balance)
		requireFullTDInvariant(t, f)
	})

	// --- Happy path: second slash accumulates ---
	t.Run("MOD-TD-MSG-5: second slash accumulates SlashedDeposit and SlashCount", func(t *testing.T) {
		f := NewFixture(t)
		now := time.Date(2026, 2, 1, 0, 0, 0, 0, time.UTC)
		f.SetBlockTime(now)

		corpAddr := sdk.AccAddress([]byte("corp_slash_2nd___1"))
		corp := corpAddr.String()
		modAddr := authtypes.NewModuleAddress(types.ModuleName).String()

		shareValue := math.LegacyMustNewDecFromStr("1.0")
		params := types.DefaultParams()
		params.TrustDepositShareValue = shareValue
		require.NoError(t, f.K.SetParams(f.Ctx, params))

		// Prior slash already recorded
		priorSlash := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
		td := types.TrustDeposit{
			Corporation:    corp,
			Share:          math.LegacyNewDec(900),
			Deposit:        900,
			SlashedDeposit: 100,
			SlashCount:     1,
			LastSlashed:    &priorSlash,
		}
		require.NoError(t, f.K.TrustDeposit.Set(f.Ctx, corp, td))

		f.SetBalance(modAddr, types.BondDenom, 1000) // 900 deposit + 100 slashed
		f.SetBalance(corp, types.BondDenom, 0)

		slashAmount := uint64(200)

		_, err := f.MS.SlashTrustDeposit(f.Ctx, &types.MsgSlashTrustDeposit{
			Authority:   GovAuthority(),
			Corporation: corp,
			Deposit:     math.NewInt(int64(slashAmount)),
			Reason:      "second slash",
		})
		require.NoError(t, err)

		f.RequireBalanceDelta(modAddr, types.BondDenom, 0)

		expectedShare := specSlashShareAfter(td.Share, slashAmount, shareValue)
		f.RequireTrustDeposit(corp, types.TrustDeposit{
			Corporation:    corp,
			Share:          expectedShare,
			Deposit:        700,
			SlashedDeposit: 300, // 100 + 200
			SlashCount:     2,
			LastSlashed:    &now,
		})

		f.RequireEvent(types.EventTypeSlashTrustDeposit, map[string]string{
			types.AttributeKeySlashCount: "2",
		})

		requireFullTDInvariant(t, f)
	})

	// --- Happy path: slash exact full deposit ---
	t.Run("MOD-TD-MSG-5: slash entire deposit reduces deposit to zero", func(t *testing.T) {
		f := NewFixture(t)
		now := time.Date(2026, 3, 1, 0, 0, 0, 0, time.UTC)
		f.SetBlockTime(now)

		corpAddr := sdk.AccAddress([]byte("corp_slash_full__1"))
		corp := corpAddr.String()
		modAddr := authtypes.NewModuleAddress(types.ModuleName).String()

		shareValue := math.LegacyMustNewDecFromStr("1.0")
		params := types.DefaultParams()
		params.TrustDepositShareValue = shareValue
		require.NoError(t, f.K.SetParams(f.Ctx, params))

		td := types.TrustDeposit{
			Corporation: corp,
			Share:       math.LegacyNewDec(500),
			Deposit:     500,
		}
		require.NoError(t, f.K.TrustDeposit.Set(f.Ctx, corp, td))

		f.SetBalance(modAddr, types.BondDenom, 500)
		f.SetBalance(corp, types.BondDenom, 0)

		_, err := f.MS.SlashTrustDeposit(f.Ctx, &types.MsgSlashTrustDeposit{
			Authority:   GovAuthority(),
			Corporation: corp,
			Deposit:     math.NewInt(500),
			Reason:      "full slash",
		})
		require.NoError(t, err)

		f.RequireTrustDeposit(corp, types.TrustDeposit{
			Corporation:    corp,
			Share:          math.LegacyZeroDec(),
			Deposit:        0,
			SlashedDeposit: 500,
			SlashCount:     1,
			LastSlashed:    &now,
		})

		requireFullTDInvariant(t, f)
	})

	// --- Negative cases ---

	t.Run("MOD-TD-MSG-5: fails if authority is wrong", func(t *testing.T) {
		f := NewFixture(t)
		corp := sdk.AccAddress([]byte("corp_bad_auth____1")).String()
		f.SetBalance(corp, types.BondDenom, 0)

		_, err := f.MS.SlashTrustDeposit(f.Ctx, &types.MsgSlashTrustDeposit{
			Authority:   "verana1invalidauthority",
			Corporation: corp,
			Deposit:     math.NewInt(100),
			Reason:      "test",
		})
		require.ErrorContains(t, err, "invalid authority")
		f.RequireNoBalanceChange(corp)
	})

	t.Run("MOD-TD-MSG-5-2-1: fails if deposit is zero", func(t *testing.T) {
		f := NewFixture(t)
		corp := sdk.AccAddress([]byte("corp_zero_dep____1")).String()
		f.SetBalance(corp, types.BondDenom, 0)

		_, err := f.MS.SlashTrustDeposit(f.Ctx, &types.MsgSlashTrustDeposit{
			Authority:   GovAuthority(),
			Corporation: corp,
			Deposit:     math.NewInt(0),
			Reason:      "test",
		})
		require.ErrorContains(t, err, "deposit must be greater than 0")
		f.RequireNoBalanceChange(corp)
	})

	t.Run("MOD-TD-MSG-5-2-1: fails if deposit is negative", func(t *testing.T) {
		f := NewFixture(t)
		corp := sdk.AccAddress([]byte("corp_neg_dep_____1")).String()
		f.SetBalance(corp, types.BondDenom, 0)

		_, err := f.MS.SlashTrustDeposit(f.Ctx, &types.MsgSlashTrustDeposit{
			Authority:   GovAuthority(),
			Corporation: corp,
			Deposit:     math.NewInt(-100),
			Reason:      "test",
		})
		require.ErrorContains(t, err, "deposit must be greater than 0")
		f.RequireNoBalanceChange(corp)
	})

	t.Run("MOD-TD-MSG-5-2-1: fails if trust deposit not found", func(t *testing.T) {
		f := NewFixture(t)
		corp := sdk.AccAddress([]byte("corp_notfound_sl_1")).String()
		f.SetBalance(corp, types.BondDenom, 0)

		_, err := f.MS.SlashTrustDeposit(f.Ctx, &types.MsgSlashTrustDeposit{
			Authority:   GovAuthority(),
			Corporation: corp,
			Deposit:     math.NewInt(100),
			Reason:      "test",
		})
		require.ErrorContains(t, err, "trust deposit not found")
		f.RequireNoBalanceChange(corp)
	})

	t.Run("MOD-TD-MSG-5-2-1: fails if deposit exceeds td.Deposit", func(t *testing.T) {
		f := NewFixture(t)
		corpAddr := sdk.AccAddress([]byte("corp_insuf_dep___1"))
		corp := corpAddr.String()
		modAddr := authtypes.NewModuleAddress(types.ModuleName).String()

		require.NoError(t, f.K.SetParams(f.Ctx, types.DefaultParams()))
		td := types.TrustDeposit{
			Corporation: corp,
			Share:       math.LegacyNewDec(100),
			Deposit:     100,
		}
		require.NoError(t, f.K.TrustDeposit.Set(f.Ctx, corp, td))

		f.SetBalance(modAddr, types.BondDenom, 100)
		f.SetBalance(corp, types.BondDenom, 0)

		_, err := f.MS.SlashTrustDeposit(f.Ctx, &types.MsgSlashTrustDeposit{
			Authority:   GovAuthority(),
			Corporation: corp,
			Deposit:     math.NewInt(200),
			Reason:      "test",
		})
		require.ErrorContains(t, err, "insufficient trust deposit")
		f.RequireNoBalanceChange(corp)
		f.RequireNoBalanceChange(modAddr)
	})
}
```

- [ ] **Step 1.2: Verify build.**

  Run: `go build ./x/td/keeper/...`
  Expected: no output, exit 0.

- [ ] **Step 1.3: Run new tests.**

  Run: `go test ./x/td/keeper/... -run TestMsgSlashTrustDeposit -v -count=1`
  Expected: all subtests PASS.

- [ ] **Step 1.4: Commit.**

```bash
git add x/td/keeper/slash_trust_deposit_test.go
git commit -m "test(td): add fixture-based SlashTrustDeposit tests"
```

---

## Task 2: Delete old tests for this message

**File:** `x/td/keeper/msg_server_test.go`

- [ ] **Step 2.1: Remove `TestMsgSlashTrustDeposit` function block.**

Delete the entire `func TestMsgSlashTrustDeposit(t *testing.T)` block from `msg_server_test.go`.

Keep `TestMsgServer`, `TestAdjustTrustDeposit`, `TestUtilityFunctions`, `TestMsgRepaySlashedTrustDeposit`, `TestMsgRepaySlashedTrustDepositAuthz`, `TestBurnEcosystemSlashedTrustDeposit`, `TestAdjustTrustDepositOnBehalf`, `TestAdjustTrustDepositSlashedGuard` intact.

- [ ] **Step 2.2: Verify build and full test suite.**

  Run: `go test ./x/td/keeper/... -count=1`
  Expected: PASS.

- [ ] **Step 2.3: Commit.**

```bash
git add x/td/keeper/msg_server_test.go
git commit -m "test(td): delete legacy TestMsgSlashTrustDeposit"
```

---

## Task 3: Final pass

- [ ] **Step 3.1: Run full TD keeper test suite.**

  Run: `go test ./x/td/keeper/... -v -count=1`
  Expected: PASS.

- [ ] **Step 3.2: Run race detector.**

  Run: `go test ./x/td/keeper/... -race -count=1`
  Expected: PASS, no DATA RACE reports.

- [ ] **Step 3.3: Run vet and lint.**

  Run: `go vet ./x/td/keeper/... && golangci-lint run ./x/td/keeper/...`
  Expected: no output.

- [ ] **Step 3.4: Coverage check.**

  Run: `go test ./x/td/keeper/ -cover -count=1`
  Expected: coverage ≥95%.

- [ ] **Step 3.5: Full repo sanity.**

  Run: `go build ./... && go test ./... -count=1`
  Expected: PASS.

- [ ] **Step 3.6: Push and open PR.**

```bash
git push -u origin test/step-2-td-slash-trust-deposit
gh pr create --title "test(td): fixture-based SlashTrustDeposit tests (step 2)" --body "$(cat <<'EOF'
## Summary
- Adds `slash_trust_deposit_test.go` with spec formula functions and fixture-based tests
- Covers 3 happy paths (standard slash, second slash accumulation, full-deposit slash)
- Covers 5 negative cases: wrong authority, zero deposit, negative deposit, TD not found, insufficient deposit
- No bank transfer on slash (coins stay in module); full accounting invariant (deposit+slashed==module balance) verified
- Deletes legacy `TestMsgSlashTrustDeposit` from `msg_server_test.go`

## Test plan
- [ ] `go test ./x/td/keeper/... -race -count=1` passes
- [ ] `go test ./x/td/keeper/ -cover` reports ≥95%
- [ ] `golangci-lint run ./x/td/keeper/...` clean
- [ ] `go test ./...` (full repo) passes — no regressions
EOF
)"
```

---

## "Done" Criteria — Step 2

- [ ] `x/td/keeper/slash_trust_deposit_test.go` covers: 3 happy paths, 5 negative cases.
- [ ] Every happy path: no balance change (no bank transfer on slash) + full struct (all fields including `LastSlashed`, `SlashCount`) + event + full accounting invariant (deposit+slashed==module balance).
- [ ] Every negative case: `ErrorContains` + `RequireNoBalanceChange`.
- [ ] `TestMsgSlashTrustDeposit` deleted from `msg_server_test.go`.
- [ ] `go test ./x/td/keeper/... -race -count=1` passes.
- [ ] Coverage ≥95% on `x/td/keeper/`.
