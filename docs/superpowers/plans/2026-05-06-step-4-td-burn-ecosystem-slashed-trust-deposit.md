# Step 4: TD BurnEcosystemSlashedTrustDeposit — Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Replace the legacy `TestBurnEcosystemSlashedTrustDeposit` test with a fixture-based test that verifies exact balance deductions (BurnCoins reduces module account balance), full struct comparison, event emission, and the TD module invariant.

**Architecture:** Uses the `Fixture` struct from step 1. `BurnEcosystemSlashedTrustDeposit` is a **Keeper method**, not a MsgServer method — called as `f.K.BurnEcosystemSlashedTrustDeposit(f.Ctx, addr, amount)`. There is no delegation keeper check and no authority check. It operates on `td.Deposit` (NOT `td.SlashedDeposit`): it reduces `td.Deposit` and `td.Share`, then calls `BurnCoins` to destroy `amount` uvna from the module account. The module balance decreases by `amount`. The `SlashedDeposit` and `SlashCount` fields are intentionally NOT modified (ecosystem slashes track at the permission level).

**Tech Stack:** Go 1.24, Cosmos SDK v0.50.x, testify/require, StatefulBankMock from testutil/keeper/bank.go

---

## Pre-flight

- [ ] **Gate check.** Step 1 must be merged: `x/td/keeper/fixture_test.go` must exist.
- [ ] **Worktree.** Create an isolated worktree. Branch name: `test/step-4-td-burn-ecosystem-slashed-trust-deposit`.
- [ ] **Sanity check.**

  Run: `go build ./... && go vet ./...`
  Expected: no output, exit 0.

---

## File Structure

- **Create:** `x/td/keeper/burn_ecosystem_slashed_trust_deposit_test.go` — fixture-based tests.
- **Modify (delete block):** `x/td/keeper/msg_server_test.go` — remove `TestBurnEcosystemSlashedTrustDeposit`.

---

## Task 1: Write `burn_ecosystem_slashed_trust_deposit_test.go`

**File:** `x/td/keeper/burn_ecosystem_slashed_trust_deposit_test.go`

Key implementation facts from `burn_slashed_td.go`:
- `account == ""` → error
- `amount == 0` → error
- TrustDeposit must exist for `account`
- `amount > td.Deposit` → error ("amount exceeds available deposit")
  - Note: the guard checks against `td.Deposit`, not `td.SlashedDeposit`
- `trustDepositShareValue == 0` → error
- Execution:
  1. `td.Deposit -= amount`
  2. `td.Share -= amount / shareValue` (clamped to zero if rounding goes negative)
  3. State saved (BEFORE burn for atomicity)
  4. `BurnCoins(td, amount uvna)` — destroys coins from module account
- `SlashedDeposit`, `SlashCount`, `LastSlashed` are NOT modified
- Bank net effect: module balance decreases by `amount`; corporation balance unchanged

Invariant after burn: `totalDeposit == moduleBalance` because both decrease by `amount`.

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

// ============================================================================
// Spec formula functions — independent of the implementation
// ============================================================================

// specBurnDepositAfter returns td.Deposit after BurnEcosystemSlashedTrustDeposit
// per [MOD-TD-MSG-7-3]: td.deposit = td.deposit - amount.
func specBurnDepositAfter(depositBefore, amount uint64) uint64 {
	return depositBefore - amount
}

// specBurnShareAfter returns td.Share after the burn per [MOD-TD-MSG-7-3]:
// td.share = td.share - amount / shareValue.
// Clamped to zero if the result is negative (rounding edge case in implementation).
func specBurnShareAfter(shareBefore math.LegacyDec, amount uint64, shareValue math.LegacyDec) math.LegacyDec {
	shareReduction := math.LegacyNewDecFromInt(math.NewInt(int64(amount))).Quo(shareValue)
	result := shareBefore.Sub(shareReduction)
	if result.IsNegative() {
		return math.LegacyZeroDec()
	}
	return result
}

// ============================================================================
// TestBurnEcosystemSlashedTrustDeposit
// ============================================================================

func TestBurnEcosystemSlashedTrustDeposit(t *testing.T) {

	// --- Happy path: standard burn ---
	t.Run("MOD-TD-MSG-7: valid burn reduces deposit, share, and module balance", func(t *testing.T) {
		f := NewFixture(t)
		now := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
		f.SetBlockTime(now)

		corpAddr := sdk.AccAddress([]byte("corp_burn_ok_____1"))
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

		// Module holds the deposit
		f.SetBalance(modAddr, types.BondDenom, 1000)
		// Corporation balance irrelevant (no send to/from corp)
		f.SetBalance(corp, types.BondDenom, 50)

		burnAmount := uint64(300)
		expectedShare := specBurnShareAfter(td.Share, burnAmount, shareValue)

		err := f.K.BurnEcosystemSlashedTrustDeposit(f.Ctx, corp, burnAmount)
		require.NoError(t, err)

		// Module balance decreased by burned amount
		f.RequireBalanceDelta(modAddr, types.BondDenom, -int64(burnAmount))
		// Corporation balance unchanged
		f.RequireNoBalanceChange(corp)

		// Full state assertion — SlashedDeposit and SlashCount must NOT change
		f.RequireTrustDeposit(corp, types.TrustDeposit{
			Corporation:    corp,
			Share:          expectedShare,
			Deposit:        specBurnDepositAfter(td.Deposit, burnAmount),
			Claimable:      0,
			SlashedDeposit: 0, // untouched
			RepaidDeposit:  0, // untouched
			SlashCount:     0, // untouched
		})

		// Event assertion
		f.RequireEvent(types.EventTypeBurnEcosystemSlashedTrustDeposit, map[string]string{
			types.AttributeKeyAccount:   corp,
			types.AttributeKeyAmount:    "300",
			types.AttributeKeyNewAmount: "700",
		})

		// Invariant: totalDeposit == moduleBalance (both decreased by burnAmount)
		f.RequireInvariant()
	})

	// --- Happy path: burn with existing SlashedDeposit unchanged ---
	t.Run("MOD-TD-MSG-7: SlashedDeposit and SlashCount are not modified", func(t *testing.T) {
		f := NewFixture(t)
		f.SetBlockTime(time.Date(2026, 2, 1, 0, 0, 0, 0, time.UTC))

		corpAddr := sdk.AccAddress([]byte("corp_burn_slash__1"))
		corp := corpAddr.String()
		modAddr := authtypes.NewModuleAddress(types.ModuleName).String()

		shareValue := math.LegacyMustNewDecFromStr("1.0")
		params := types.DefaultParams()
		params.TrustDepositShareValue = shareValue
		require.NoError(t, f.K.SetParams(f.Ctx, params))

		priorSlash := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
		td := types.TrustDeposit{
			Corporation:    corp,
			Share:          math.LegacyNewDec(1000),
			Deposit:        1000,
			SlashedDeposit: 50,
			SlashCount:     2,
			LastSlashed:    &priorSlash,
		}
		require.NoError(t, f.K.TrustDeposit.Set(f.Ctx, corp, td))

		// Module holds 1000 deposit + 50 slashed (governance slash, not yet burned)
		f.SetBalance(modAddr, types.BondDenom, 1050)
		f.SetBalance(corp, types.BondDenom, 0)

		burnAmount := uint64(100)

		err := f.K.BurnEcosystemSlashedTrustDeposit(f.Ctx, corp, burnAmount)
		require.NoError(t, err)

		f.RequireBalanceDelta(modAddr, types.BondDenom, -100)

		// SlashedDeposit and SlashCount must be exactly unchanged
		got, err := f.K.TrustDeposit.Get(f.Ctx, corp)
		require.NoError(t, err)
		require.Equal(t, uint64(50), got.SlashedDeposit, "SlashedDeposit must not change on ecosystem burn")
		require.Equal(t, uint64(2), got.SlashCount, "SlashCount must not change on ecosystem burn")
		require.Equal(t, &priorSlash, got.LastSlashed, "LastSlashed must not change on ecosystem burn")
		require.Equal(t, specBurnDepositAfter(td.Deposit, burnAmount), got.Deposit)

		f.RequireEvent(types.EventTypeBurnEcosystemSlashedTrustDeposit, map[string]string{
			types.AttributeKeyAccount: corp,
			types.AttributeKeyAmount:  "100",
		})
	})

	// --- Happy path: burn entire deposit ---
	t.Run("MOD-TD-MSG-7: burn entire deposit reduces deposit and share to zero", func(t *testing.T) {
		f := NewFixture(t)
		f.SetBlockTime(time.Date(2026, 3, 1, 0, 0, 0, 0, time.UTC))

		corpAddr := sdk.AccAddress([]byte("corp_burn_full___1"))
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

		err := f.K.BurnEcosystemSlashedTrustDeposit(f.Ctx, corp, 500)
		require.NoError(t, err)

		f.RequireBalanceDelta(modAddr, types.BondDenom, -500)

		got, err := f.K.TrustDeposit.Get(f.Ctx, corp)
		require.NoError(t, err)
		require.Equal(t, uint64(0), got.Deposit)
		// Share clamped to zero (implementation clamps negative rounding to 0)
		require.True(t, got.Share.IsZero() || !got.Share.IsNegative(),
			"share must be zero or non-negative after full burn, got %s", got.Share)

		f.RequireInvariant()
	})

	// --- Negative cases ---

	t.Run("MOD-TD-MSG-7-2-1: fails if account is empty string", func(t *testing.T) {
		f := NewFixture(t)
		modAddr := authtypes.NewModuleAddress(types.ModuleName).String()
		f.SetBalance(modAddr, types.BondDenom, 100)

		err := f.K.BurnEcosystemSlashedTrustDeposit(f.Ctx, "", 100)
		require.ErrorContains(t, err, "account cannot be empty")
		f.RequireNoBalanceChange(modAddr)
	})

	t.Run("MOD-TD-MSG-7-2-1: fails if amount is zero", func(t *testing.T) {
		f := NewFixture(t)
		corp := sdk.AccAddress([]byte("corp_burn_zero___1")).String()
		modAddr := authtypes.NewModuleAddress(types.ModuleName).String()
		f.SetBalance(modAddr, types.BondDenom, 100)
		f.SetBalance(corp, types.BondDenom, 0)

		err := f.K.BurnEcosystemSlashedTrustDeposit(f.Ctx, corp, 0)
		require.ErrorContains(t, err, "deposit must be greater than 0")
		f.RequireNoBalanceChange(modAddr)
	})

	t.Run("MOD-TD-MSG-7-2-1: fails if trust deposit not found", func(t *testing.T) {
		f := NewFixture(t)
		corp := sdk.AccAddress([]byte("corp_burn_nf_____1")).String()
		modAddr := authtypes.NewModuleAddress(types.ModuleName).String()
		f.SetBalance(modAddr, types.BondDenom, 100)
		f.SetBalance(corp, types.BondDenom, 0)

		err := f.K.BurnEcosystemSlashedTrustDeposit(f.Ctx, corp, 100)
		require.ErrorContains(t, err, "trust deposit entry not found")
		f.RequireNoBalanceChange(modAddr)
	})

	t.Run("MOD-TD-MSG-7-2-1: fails if amount exceeds td.Deposit", func(t *testing.T) {
		f := NewFixture(t)
		corpAddr := sdk.AccAddress([]byte("corp_burn_exc____1"))
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

		err := f.K.BurnEcosystemSlashedTrustDeposit(f.Ctx, corp, 200)
		require.ErrorContains(t, err, "amount exceeds available deposit")
		f.RequireNoBalanceChange(modAddr)
		f.RequireNoBalanceChange(corp)
	})

	t.Run("MOD-TD-MSG-7: fails if share value is zero", func(t *testing.T) {
		f := NewFixture(t)
		corpAddr := sdk.AccAddress([]byte("corp_burn_sv0____1"))
		corp := corpAddr.String()
		modAddr := authtypes.NewModuleAddress(types.ModuleName).String()

		params := types.DefaultParams()
		params.TrustDepositShareValue = math.LegacyZeroDec()
		require.NoError(t, f.K.SetParams(f.Ctx, params))

		td := types.TrustDeposit{
			Corporation: corp,
			Share:       math.LegacyNewDec(100),
			Deposit:     100,
		}
		require.NoError(t, f.K.TrustDeposit.Set(f.Ctx, corp, td))

		f.SetBalance(modAddr, types.BondDenom, 100)
		f.SetBalance(corp, types.BondDenom, 0)

		err := f.K.BurnEcosystemSlashedTrustDeposit(f.Ctx, corp, 50)
		require.ErrorContains(t, err, "trust deposit share value cannot be zero")
		f.RequireNoBalanceChange(modAddr)
		f.RequireNoBalanceChange(corp)
	})

	// --- Edge cases ---

	t.Run("edge: burn minimum amount (1 uvna)", func(t *testing.T) {
		f := NewFixture(t)
		f.SetBlockTime(time.Date(2026, 4, 1, 0, 0, 0, 0, time.UTC))

		corpAddr := sdk.AccAddress([]byte("corp_burn_min____1"))
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

		f.SetBalance(modAddr, types.BondDenom, 1000)
		f.SetBalance(corp, types.BondDenom, 0)

		err := f.K.BurnEcosystemSlashedTrustDeposit(f.Ctx, corp, 1)
		require.NoError(t, err)

		f.RequireBalanceDelta(modAddr, types.BondDenom, -1)

		got, err := f.K.TrustDeposit.Get(f.Ctx, corp)
		require.NoError(t, err)
		require.Equal(t, uint64(999), got.Deposit)

		f.RequireInvariant()
	})

	t.Run("edge: non-unit share value — fractional share reduction", func(t *testing.T) {
		f := NewFixture(t)
		f.SetBlockTime(time.Date(2026, 5, 1, 0, 0, 0, 0, time.UTC))

		corpAddr := sdk.AccAddress([]byte("corp_burn_frac___1"))
		corp := corpAddr.String()
		modAddr := authtypes.NewModuleAddress(types.ModuleName).String()

		// shareValue = 1.5 means 300 amount → 200 shares reduced
		shareValue := math.LegacyMustNewDecFromStr("1.5")
		params := types.DefaultParams()
		params.TrustDepositShareValue = shareValue
		require.NoError(t, f.K.SetParams(f.Ctx, params))

		td := types.TrustDeposit{
			Corporation: corp,
			Share:       math.LegacyNewDec(1000),
			Deposit:     1500,
		}
		require.NoError(t, f.K.TrustDeposit.Set(f.Ctx, corp, td))

		f.SetBalance(modAddr, types.BondDenom, 1500)
		f.SetBalance(corp, types.BondDenom, 0)

		burnAmount := uint64(300)
		expectedShare := specBurnShareAfter(td.Share, burnAmount, shareValue) // 1000 - 300/1.5 = 1000 - 200 = 800

		err := f.K.BurnEcosystemSlashedTrustDeposit(f.Ctx, corp, burnAmount)
		require.NoError(t, err)

		f.RequireBalanceDelta(modAddr, types.BondDenom, -300)

		f.RequireTrustDeposit(corp, types.TrustDeposit{
			Corporation: corp,
			Share:       expectedShare,
			Deposit:     1200,
		})

		f.RequireInvariant()
	})
}
```

- [ ] **Step 1.2: Verify build.**

  Run: `go build ./x/td/keeper/...`
  Expected: no output, exit 0.

- [ ] **Step 1.3: Run new tests.**

  Run: `go test ./x/td/keeper/... -run TestBurnEcosystemSlashedTrustDeposit -v -count=1`
  Expected: all subtests PASS.

- [ ] **Step 1.4: Commit.**

```bash
git add x/td/keeper/burn_ecosystem_slashed_trust_deposit_test.go
git commit -m "test(td): add fixture-based BurnEcosystemSlashedTrustDeposit tests"
```

---

## Task 2: Delete old tests for this message

**File:** `x/td/keeper/msg_server_test.go`

- [ ] **Step 2.1: Remove `TestBurnEcosystemSlashedTrustDeposit` function block.**

Delete the entire `func TestBurnEcosystemSlashedTrustDeposit(t *testing.T)` block.

After this deletion, check whether any remaining tests in `msg_server_test.go` still use the following helpers:
- `govAuthority()` — keep if still referenced, remove if not
- `defaultTestParams()` — keep if still referenced, remove if not
- `setupMsgServerWithDelegation` — keep if still referenced, remove if not
- `setupMsgServer` — keep if still referenced (used by `TestMsgServer`, `TestAdjustTrustDeposit`, `TestUtilityFunctions`)

After steps 1–4, the remaining functions in `msg_server_test.go` should be only: `TestMsgServer`, `TestAdjustTrustDeposit`, `TestUtilityFunctions`, `TestAdjustTrustDepositOnBehalf`, `TestAdjustTrustDepositSlashedGuard` and their supporting helpers.

- [ ] **Step 2.2: Verify build and full test suite.**

  Run: `go test ./x/td/keeper/... -count=1`
  Expected: PASS.

- [ ] **Step 2.3: Commit.**

```bash
git add x/td/keeper/msg_server_test.go
git commit -m "test(td): delete legacy TestBurnEcosystemSlashedTrustDeposit"
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
git push -u origin test/step-4-td-burn-ecosystem-slashed-trust-deposit
gh pr create --title "test(td): fixture-based BurnEcosystemSlashedTrustDeposit tests (step 4)" --body "$(cat <<'EOF'
## Summary
- Adds `burn_ecosystem_slashed_trust_deposit_test.go` with spec formula functions and fixture-based tests
- Tests Keeper method (not MsgServer): no authority check, no delegation keeper check
- Covers 3 happy paths: standard burn, burn leaving SlashedDeposit/SlashCount unchanged, burn entire deposit
- Covers 5 negative cases: empty account, zero amount, TD not found, amount exceeds deposit, zero share value
- Covers 2 edge cases: minimum 1 uvna burn, non-unit share value fractional reduction
- Module balance decreases by burned amount (BurnCoins); corporation balance unchanged; TD invariant verified
- Deletes legacy `TestBurnEcosystemSlashedTrustDeposit` from `msg_server_test.go`

## Test plan
- [ ] `go test ./x/td/keeper/... -race -count=1` passes
- [ ] `go test ./x/td/keeper/ -cover` reports ≥95%
- [ ] `golangci-lint run ./x/td/keeper/...` clean
- [ ] `go test ./...` (full repo) passes — no regressions
EOF
)"
```

---

## "Done" Criteria — Step 4

- [ ] `x/td/keeper/burn_ecosystem_slashed_trust_deposit_test.go` covers: 3 happy paths, 5 negative cases, 2 edge cases.
- [ ] Every happy path: module balance delta `-amount` + corporation no-change + full struct (confirming `SlashedDeposit`/`SlashCount` unchanged) + event + invariant.
- [ ] Every negative case: `ErrorContains` + `RequireNoBalanceChange` on module address.
- [ ] `TestBurnEcosystemSlashedTrustDeposit` deleted from `msg_server_test.go`.
- [ ] `go test ./x/td/keeper/... -race -count=1` passes.
- [ ] Coverage ≥95% on `x/td/keeper/`.

---

## Notes

- `BurnEcosystemSlashedTrustDeposit` operates on `td.Deposit` (ecosystem-level permission slash), not on `td.SlashedDeposit` (governance-level slash from `SlashTrustDeposit`). These are distinct accounting tracks. Verify the test assertions reflect this: `SlashedDeposit` must be unchanged in happy paths.
- The `amount > td.Deposit` check (not `amount > td.SlashedDeposit`) is confirmed in `burn_slashed_td.go:29`. Test the error case accordingly.
- After all four steps (1–4) are complete, the only remaining content in `msg_server_test.go` will be non-financial keeper utility tests (`TestAdjustTrustDeposit`, `TestUtilityFunctions`, `TestAdjustTrustDepositOnBehalf`, `TestAdjustTrustDepositSlashedGuard`, `TestMsgServer`). These are not part of the financial overhaul and remain as-is unless a subsequent step targets them.
