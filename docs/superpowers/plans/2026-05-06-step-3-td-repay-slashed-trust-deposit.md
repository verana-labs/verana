# Step 3: TD RepaySlashedTrustDeposit — Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Replace the legacy `TestMsgRepaySlashedTrustDeposit` and `TestMsgRepaySlashedTrustDepositAuthz` tests with a fixture-based test that verifies the two-phase bank flow (corporation sends → module burns), full struct comparison, event emission, and the TD module invariant.

**Architecture:** Uses the `Fixture` struct from step 1. `RepaySlashedTrustDeposit` performs two bank calls in sequence: `SendCoinsFromAccountToModule` (corporation → td module) and `BurnCoins` (td module burns the previously-slashed coins). After repay, `deposit += amount`, `slashed_deposit -= amount`, `repaid_deposit += amount`, `shares += amount/shareValue`. The invariant check must account for the net effect: the corporation's new coins enter the module and the old slashed coins are burned, keeping module balance at `totalDeposit` (since `slashedDeposit` is now zero again for a full repay).

**Tech Stack:** Go 1.24, Cosmos SDK v0.50.x, testify/require, StatefulBankMock from testutil/keeper/bank.go

---

## Pre-flight

- [ ] **Gate check.** Step 1 must be merged: `x/td/keeper/fixture_test.go` must exist.
- [ ] **Worktree.** Create an isolated worktree. Branch name: `test/step-3-td-repay-slashed-trust-deposit`.
- [ ] **Sanity check.**

  Run: `go build ./... && go vet ./...`
  Expected: no output, exit 0.

---

## File Structure

- **Create:** `x/td/keeper/repay_slashed_trust_deposit_test.go` — fixture-based tests.
- **Modify (delete block):** `x/td/keeper/msg_server_test.go` — remove `TestMsgRepaySlashedTrustDeposit` and `TestMsgRepaySlashedTrustDepositAuthz`.

---

## Task 1: Write `repay_slashed_trust_deposit_test.go`

**File:** `x/td/keeper/repay_slashed_trust_deposit_test.go`

Key implementation facts from `msg_server.go`:
- Authorization check: `delegationKeeper.CheckOperatorAuthorization(...)` → error if fails
- TrustDeposit must exist for `msg.Corporation`
- `msg.Deposit` MUST equal `td.SlashedDeposit` exactly (full repay required)
- Execution in order:
  1. `td.Deposit += msg.Deposit`
  2. `td.Share += msg.Deposit / shareValue`
  3. `td.SlashedDeposit -= msg.Deposit`
  4. `td.RepaidDeposit += msg.Deposit`
  5. `td.LastRepaid = &now`
  6. State saved
  7. `SendCoinsFromAccountToModule(corporation → td, msg.Deposit uvna)` — NEW coins from corporation
  8. `BurnCoins(td, msg.Deposit uvna)` — BURNS the OLD slashed coins that were locked in module
- Net bank effect: corporation loses `msg.Deposit`, module account is UNCHANGED (receives same amount it burns), `td.Deposit` increases by `msg.Deposit`

Invariant after full repay: `totalDeposit == moduleBalance` because:
- Before repay: module holds `originalDeposit + slashedAmount` (slashed not yet burned)
- After repay: corporation sends `slashedAmount` in → module now holds `originalDeposit + 2*slashedAmount`; then burns `slashedAmount` → module holds `originalDeposit + slashedAmount`; and `td.Deposit` = `originalDeposit + slashedAmount`. So invariant holds.

- [ ] **Step 1.1: Create the test file.**

```go
package keeper_test

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

// ============================================================================
// Spec formula functions — independent of the implementation
// ============================================================================

// specRepayDepositAfter returns td.Deposit after repay per [MOD-TD-MSG-6-3].
func specRepayDepositAfter(depositBefore, repayAmount uint64) uint64 {
	return depositBefore + repayAmount
}

// specRepaySlashedDepositAfter returns td.SlashedDeposit after repay.
// Per spec v4 draft 13, slashed_deposit is decremented on each repay
// (it tracks the outstanding balance, not cumulative history).
func specRepaySlashedDepositAfter(slashedBefore, repayAmount uint64) uint64 {
	return slashedBefore - repayAmount
}

// specRepayRepaidDepositAfter returns td.RepaidDeposit after repay.
// repaid_deposit is cumulative across all repays.
func specRepayRepaidDepositAfter(repaidBefore, repayAmount uint64) uint64 {
	return repaidBefore + repayAmount
}

// specRepayShareAfter returns td.Share after repay per [MOD-TD-MSG-6-3]:
// shareIncrease = repayAmount / shareValue; share = shareBefore + shareIncrease.
func specRepayShareAfter(shareBefore math.LegacyDec, repayAmount uint64, shareValue math.LegacyDec) math.LegacyDec {
	shareIncrease := math.LegacyNewDec(int64(repayAmount)).Quo(shareValue)
	return shareBefore.Add(shareIncrease)
}

// ============================================================================
// TestMsgRepaySlashedTrustDeposit
// ============================================================================

func TestMsgRepaySlashedTrustDeposit(t *testing.T) {

	// --- Happy path: standard full repay ---
	t.Run("MOD-TD-MSG-6: valid repay sends new coins and burns old slashed coins", func(t *testing.T) {
		f := NewFixture(t)
		now := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
		f.SetBlockTime(now)

		corpAddr := sdk.AccAddress([]byte("corp_repay_ok____1"))
		corp := corpAddr.String()
		modAddr := authtypes.NewModuleAddress(types.ModuleName).String()

		shareValue := math.LegacyMustNewDecFromStr("1.0")
		params := types.DefaultParams()
		params.TrustDepositShareValue = shareValue
		require.NoError(t, f.K.SetParams(f.Ctx, params))

		// TD was slashed 300 uvna previously; deposit currently 700; slashed_deposit 300
		td := types.TrustDeposit{
			Corporation:    corp,
			Share:          math.LegacyNewDec(700),
			Deposit:        700,
			SlashedDeposit: 300,
			RepaidDeposit:  0,
			SlashCount:     1,
		}
		require.NoError(t, f.K.TrustDeposit.Set(f.Ctx, corp, td))

		// Module currently holds 700 deposit + 300 slashed (locked) = 1000 uvna total
		f.SetBalance(modAddr, types.BondDenom, 1000)
		// Corporation must have enough to repay
		f.SetBalance(corp, types.BondDenom, 500)

		repayAmount := uint64(300)
		expectedShare := specRepayShareAfter(td.Share, repayAmount, shareValue)

		_, err := f.MS.RepaySlashedTrustDeposit(f.Ctx, &types.MsgRepaySlashedTrustDeposit{
			Corporation: corp,
			Operator:    corp,
			Deposit:     repayAmount,
		})
		require.NoError(t, err)

		// Bank flow assertions:
		// - corporation sent 300 to module: corp balance -300
		// - module received 300 then burned 300: net module balance unchanged (1000 -> 1000)
		f.RequireBalanceDelta(corp, types.BondDenom, -int64(repayAmount))
		f.RequireBalanceDelta(modAddr, types.BondDenom, 0)

		// Full state assertion
		f.RequireTrustDeposit(corp, types.TrustDeposit{
			Corporation:    corp,
			Share:          expectedShare,
			Deposit:        specRepayDepositAfter(td.Deposit, repayAmount),
			SlashedDeposit: specRepaySlashedDepositAfter(td.SlashedDeposit, repayAmount),
			RepaidDeposit:  specRepayRepaidDepositAfter(td.RepaidDeposit, repayAmount),
			SlashCount:     1,
			LastRepaid:     &now,
		})

		// Event assertion
		f.RequireEvent(types.EventTypeRepaySlashedTrustDeposit, map[string]string{
			types.AttributeKeyAccount: corp,
			types.AttributeKeyAmount:  "300",
		})

		// Invariant: totalDeposit == moduleBalance
		// After repay: deposit=1000, slashed=0. Module balance = 1000 (1000+300-300).
		f.RequireInvariant()
	})

	// --- Happy path: repay with cumulative history ---
	t.Run("MOD-TD-MSG-6: repay with prior repaid_deposit accumulates cumulative total", func(t *testing.T) {
		f := NewFixture(t)
		now := time.Date(2026, 2, 1, 0, 0, 0, 0, time.UTC)
		f.SetBlockTime(now)

		corpAddr := sdk.AccAddress([]byte("corp_repay_cum___1"))
		corp := corpAddr.String()
		modAddr := authtypes.NewModuleAddress(types.ModuleName).String()

		shareValue := math.LegacyMustNewDecFromStr("1.0")
		params := types.DefaultParams()
		params.TrustDepositShareValue = shareValue
		require.NoError(t, f.K.SetParams(f.Ctx, params))

		// Corporation previously repaid 200; now owes 300 more outstanding
		td := types.TrustDeposit{
			Corporation:    corp,
			Share:          math.LegacyNewDec(800),
			Deposit:        800,
			SlashedDeposit: 300,
			RepaidDeposit:  200, // prior cumulative repay
			SlashCount:     2,
		}
		require.NoError(t, f.K.TrustDeposit.Set(f.Ctx, corp, td))

		// Module: 800 deposit + 300 slashed = 1100
		f.SetBalance(modAddr, types.BondDenom, 1100)
		f.SetBalance(corp, types.BondDenom, 400)

		repayAmount := uint64(300)
		expectedShare := specRepayShareAfter(td.Share, repayAmount, shareValue)

		_, err := f.MS.RepaySlashedTrustDeposit(f.Ctx, &types.MsgRepaySlashedTrustDeposit{
			Corporation: corp,
			Operator:    corp,
			Deposit:     repayAmount,
		})
		require.NoError(t, err)

		f.RequireBalanceDelta(corp, types.BondDenom, -300)
		f.RequireBalanceDelta(modAddr, types.BondDenom, 0)

		f.RequireTrustDeposit(corp, types.TrustDeposit{
			Corporation:    corp,
			Share:          expectedShare,
			Deposit:        1100, // 800+300
			SlashedDeposit: 0,    // fully resolved
			RepaidDeposit:  500,  // 200+300 cumulative
			SlashCount:     2,
			LastRepaid:     &now,
		})

		f.RequireInvariant()
	})

	// --- Negative cases ---

	t.Run("MOD-TD-MSG-6-2-1: fails if authorization check fails", func(t *testing.T) {
		f := NewFixture(t)
		corpAddr := sdk.AccAddress([]byte("corp_repay_auth__1"))
		corp := corpAddr.String()
		modAddr := authtypes.NewModuleAddress(types.ModuleName).String()

		require.NoError(t, f.K.SetParams(f.Ctx, types.DefaultParams()))
		td := types.TrustDeposit{
			Corporation:    corp,
			Share:          math.LegacyNewDec(700),
			Deposit:        700,
			SlashedDeposit: 300,
		}
		require.NoError(t, f.K.TrustDeposit.Set(f.Ctx, corp, td))

		f.SetBalance(modAddr, types.BondDenom, 1000)
		f.SetBalance(corp, types.BondDenom, 400)

		// Force delegation keeper to reject
		f.DelKeeper.ErrToReturn = fmt.Errorf("mock: operator not authorized")

		_, err := f.MS.RepaySlashedTrustDeposit(f.Ctx, &types.MsgRepaySlashedTrustDeposit{
			Corporation: corp,
			Operator:    sdk.AccAddress([]byte("bad_operator_____1")).String(),
			Deposit:     300,
		})
		require.ErrorContains(t, err, "authorization check failed")
		f.RequireNoBalanceChange(corp)
		f.RequireNoBalanceChange(modAddr)
	})

	t.Run("MOD-TD-MSG-6-2-1: fails if trust deposit not found", func(t *testing.T) {
		f := NewFixture(t)
		corp := sdk.AccAddress([]byte("corp_repay_nf____1")).String()
		f.SetBalance(corp, types.BondDenom, 0)

		_, err := f.MS.RepaySlashedTrustDeposit(f.Ctx, &types.MsgRepaySlashedTrustDeposit{
			Corporation: corp,
			Operator:    corp,
			Deposit:     100,
		})
		require.ErrorContains(t, err, "trust deposit entry not found")
		f.RequireNoBalanceChange(corp)
	})

	t.Run("MOD-TD-MSG-6-2-1: fails if msg.Deposit != td.SlashedDeposit", func(t *testing.T) {
		f := NewFixture(t)
		corpAddr := sdk.AccAddress([]byte("corp_repay_mism__1"))
		corp := corpAddr.String()
		modAddr := authtypes.NewModuleAddress(types.ModuleName).String()

		require.NoError(t, f.K.SetParams(f.Ctx, types.DefaultParams()))
		td := types.TrustDeposit{
			Corporation:    corp,
			Share:          math.LegacyNewDec(700),
			Deposit:        700,
			SlashedDeposit: 300,
		}
		require.NoError(t, f.K.TrustDeposit.Set(f.Ctx, corp, td))

		f.SetBalance(modAddr, types.BondDenom, 1000)
		f.SetBalance(corp, types.BondDenom, 400)

		// Send partial amount (200 instead of required 300)
		_, err := f.MS.RepaySlashedTrustDeposit(f.Ctx, &types.MsgRepaySlashedTrustDeposit{
			Corporation: corp,
			Operator:    corp,
			Deposit:     200,
		})
		require.ErrorContains(t, err, "deposit must exactly equal outstanding slashed amount")
		f.RequireNoBalanceChange(corp)
		f.RequireNoBalanceChange(modAddr)
	})

	t.Run("MOD-TD-MSG-6: fails if corporation has insufficient funds for repay", func(t *testing.T) {
		f := NewFixture(t)
		corpAddr := sdk.AccAddress([]byte("corp_repay_insuf_1"))
		corp := corpAddr.String()
		modAddr := authtypes.NewModuleAddress(types.ModuleName).String()

		require.NoError(t, f.K.SetParams(f.Ctx, types.DefaultParams()))
		td := types.TrustDeposit{
			Corporation:    corp,
			Share:          math.LegacyNewDec(700),
			Deposit:        700,
			SlashedDeposit: 300,
		}
		require.NoError(t, f.K.TrustDeposit.Set(f.Ctx, corp, td))

		f.SetBalance(modAddr, types.BondDenom, 1000)
		// Corporation only has 50 uvna but needs 300 for repay
		f.SetBalance(corp, types.BondDenom, 50)

		_, err := f.MS.RepaySlashedTrustDeposit(f.Ctx, &types.MsgRepaySlashedTrustDeposit{
			Corporation: corp,
			Operator:    corp,
			Deposit:     300,
		})
		require.Error(t, err)
		// StatefulBankMock returns ErrInsufficientFunds wrapped in "failed to transfer tokens"
		require.ErrorContains(t, err, "failed to transfer tokens")
		f.RequireNoBalanceChange(corp)
		f.RequireNoBalanceChange(modAddr)
	})

	// --- Edge cases ---

	t.Run("edge: repay with authorized but different operator", func(t *testing.T) {
		f := NewFixture(t)
		corpAddr := sdk.AccAddress([]byte("corp_repay_op____1"))
		corp := corpAddr.String()
		modAddr := authtypes.NewModuleAddress(types.ModuleName).String()
		operator := sdk.AccAddress([]byte("authorized_op____1")).String()

		require.NoError(t, f.K.SetParams(f.Ctx, types.DefaultParams()))
		td := types.TrustDeposit{
			Corporation:    corp,
			Share:          math.LegacyNewDec(700),
			Deposit:        700,
			SlashedDeposit: 100,
		}
		require.NoError(t, f.K.TrustDeposit.Set(f.Ctx, corp, td))

		f.SetBalance(modAddr, types.BondDenom, 800)
		f.SetBalance(corp, types.BondDenom, 200)

		// DelKeeper returns nil by default — authorized
		f.DelKeeper.ErrToReturn = nil

		_, err := f.MS.RepaySlashedTrustDeposit(f.Ctx, &types.MsgRepaySlashedTrustDeposit{
			Corporation: corp,
			Operator:    operator,
			Deposit:     100,
		})
		require.NoError(t, err)
		f.RequireBalanceDelta(corp, types.BondDenom, -100)
		f.RequireInvariant()
	})
}
```

- [ ] **Step 1.2: Verify build.**

  Run: `go build ./x/td/keeper/...`
  Expected: no output, exit 0.

- [ ] **Step 1.3: Run new tests.**

  Run: `go test ./x/td/keeper/... -run TestMsgRepaySlashedTrustDeposit -v -count=1`
  Expected: all subtests PASS.

- [ ] **Step 1.4: Commit.**

```bash
git add x/td/keeper/repay_slashed_trust_deposit_test.go
git commit -m "test(td): add fixture-based RepaySlashedTrustDeposit tests"
```

---

## Task 2: Delete old tests for this message

**File:** `x/td/keeper/msg_server_test.go`

- [ ] **Step 2.1: Remove legacy repay test functions.**

Delete both:
- `func TestMsgRepaySlashedTrustDeposit(t *testing.T)` block
- `func TestMsgRepaySlashedTrustDepositAuthz(t *testing.T)` block

Also remove `setupMsgServerWithDelegation` helper if it is no longer referenced by any remaining test in the file. If it is still referenced by `TestBurnEcosystemSlashedTrustDeposit` or other tests, keep it.

- [ ] **Step 2.2: Verify build and full test suite.**

  Run: `go test ./x/td/keeper/... -count=1`
  Expected: PASS.

- [ ] **Step 2.3: Commit.**

```bash
git add x/td/keeper/msg_server_test.go
git commit -m "test(td): delete legacy TestMsgRepaySlashedTrustDeposit and Authz tests"
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
git push -u origin test/step-3-td-repay-slashed-trust-deposit
gh pr create --title "test(td): fixture-based RepaySlashedTrustDeposit tests (step 3)" --body "$(cat <<'EOF'
## Summary
- Adds `repay_slashed_trust_deposit_test.go` with spec formula functions and fixture-based tests
- Covers 2 happy paths: standard full repay and repay with prior cumulative repaid_deposit
- Covers 5 negative cases: authz failure, TD not found, wrong repay amount, insufficient corporation funds, different operator
- Two-phase bank flow verified: corporation -repayAmount, module unchanged (receives then burns)
- Standard TD invariant (totalDeposit == moduleBalance) passes after full repay
- Deletes legacy `TestMsgRepaySlashedTrustDeposit` and `TestMsgRepaySlashedTrustDepositAuthz`

## Test plan
- [ ] `go test ./x/td/keeper/... -race -count=1` passes
- [ ] `go test ./x/td/keeper/ -cover` reports ≥95%
- [ ] `golangci-lint run ./x/td/keeper/...` clean
- [ ] `go test ./...` (full repo) passes — no regressions
EOF
)"
```

---

## "Done" Criteria — Step 3

- [ ] `x/td/keeper/repay_slashed_trust_deposit_test.go` covers: 2 happy paths, 5 negative cases, 1 edge case.
- [ ] Every happy path: corporation balance -repayAmount + module balance unchanged (send+burn cancel) + full struct (all fields including `LastRepaid`) + event + invariant.
- [ ] Every negative case: `ErrorContains` + `RequireNoBalanceChange` on both corporation and module addresses.
- [ ] `TestMsgRepaySlashedTrustDeposit` and `TestMsgRepaySlashedTrustDepositAuthz` deleted from `msg_server_test.go`.
- [ ] `go test ./x/td/keeper/... -race -count=1` passes.
- [ ] Coverage ≥95% on `x/td/keeper/`.
