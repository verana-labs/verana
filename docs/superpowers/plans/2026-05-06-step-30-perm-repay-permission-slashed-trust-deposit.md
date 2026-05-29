# Step 30: PERM RepayPermissionSlashedTrustDeposit — Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Replace legacy `TestMsgRepayPermissionSlashedTrustDeposit` tests with fixture-based tests that verify full struct equality (Deposit, RepaidDeposit, Repaid), the `AdjustTrustDeposit` mock call, event emission, the PERM module invariant, and every spec precondition — including the partial-repay vs. full-repay boundary.

**Architecture:** `RepayPermissionSlashedTrustDeposit` calls `trustDeposit.AdjustTrustDeposit` with a positive amount (repay goes into the trust deposit) but makes no direct bank transfer. The `StatefulBankMock.HasBalance` is called to check the payer's balance — the fixture sets a sufficient balance via `f.SetBalance`. `AdjustHistory` from the `PermTrustDepositMock` (step 27) is used to assert the call. The `Repaid` timestamp is only set when `repaid_deposit >= slashed_deposit`.

**Tech Stack:** Go 1.24, Cosmos SDK v0.50.x, testify/require, StatefulBankMock from testutil/keeper/bank.go

---

## Pre-flight

- [ ] **Confirm step 27 is merged** (so `PermTrustDepositMock.AdjustHistory` exists).

  ```bash
  grep -n "AdjustHistory" testutil/keeper/permission.go
  ```
  Expected: line found.

- [ ] **Confirm step 21 is merged.** `x/perm/keeper/fixture_test.go` must exist.

  ```bash
  ls x/perm/keeper/fixture_test.go
  ```
  Expected: file exists.

- [ ] **Create worktree.** Branch name: `test/step-30-perm-repay-slashed-deposit`.

- [ ] **Sanity check.**

  ```bash
  go build ./... && go test ./x/perm/keeper/... -count=1
  ```
  Expected: all pass.

---

## File Structure

- **Create:** `x/perm/keeper/msg_repay_permission_slashed_trust_deposit_test.go` — fixture-based tests.
- **Delete (in same PR):** the `TestMsgRepayPermissionSlashedTrustDeposit` function block from `x/perm/keeper/msg_server_test.go`.

---

## Task 1: Write `x/perm/keeper/msg_repay_permission_slashed_trust_deposit_test.go`

**File:** `x/perm/keeper/msg_repay_permission_slashed_trust_deposit_test.go`

- [ ] **Step 1.1: Create the test file.**

```go
package keeper_test

import (
	"fmt"
	"testing"
	"time"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/stretchr/testify/require"

	cstypes "github.com/verana-labs/verana/x/cs/types"
	"github.com/verana-labs/verana/x/perm/types"
)

// ============================================================================
// Spec formula functions — independent of the implementation
// ============================================================================

// specRepayDepositAfter returns perm.Deposit after a repay.
// Per [MOD-PERM-MSG-13-3]: deposit += amount.
func specRepayDepositAfter(depositBefore, amount uint64) uint64 {
	return depositBefore + amount
}

// specRepayRepaidDepositAfter returns perm.RepaidDeposit after a repay.
// Per [MOD-PERM-MSG-13-3]: repaid_deposit += amount.
func specRepayRepaidDepositAfter(repaidBefore, amount uint64) uint64 {
	return repaidBefore + amount
}

// specRepayIsFullyRepaid returns whether this repay completes the obligation.
// Per [MOD-PERM-MSG-13-3]: fully repaid when repaid_deposit >= slashed_deposit.
func specRepayIsFullyRepaid(repaidBefore, amount, slashedDeposit uint64) bool {
	return specRepayRepaidDepositAfter(repaidBefore, amount) >= slashedDeposit
}

// ============================================================================
// TestMsgRepayPermissionSlashedTrustDeposit
// ============================================================================

func TestMsgRepayPermissionSlashedTrustDeposit(t *testing.T) {

	// helper builds a slashed permission ready for repayment.
	makeSlashedPerm := func(
		t *testing.T,
		f *Fixture,
		now time.Time,
		corp string,
		depositBefore, slashedDeposit, repaidBefore uint64,
	) uint64 {
		t.Helper()
		schemaID := uint64(1)
		f.CSKeeper.CreateMockCredentialSchema(schemaID,
			cstypes.IssuerOnboardingMode_ISSUER_ONBOARDING_MODE_OPEN,
			cstypes.VerifierOnboardingMode_VERIFIER_ONBOARDING_MODE_OPEN)

		effFrom := now.Add(-48 * time.Hour)
		effUntil := now.Add(90 * 24 * time.Hour)
		slashedAt := now.Add(-1 * time.Hour)
		perm := types.Permission{
			SchemaId:        schemaID,
			Type:            types.PermissionType_ISSUER,
			Corporation:     corp,
			ValidatorPermId: 0,
			EffectiveFrom:   &effFrom,
			EffectiveUntil:  &effUntil,
			Deposit:         depositBefore,
			SlashedDeposit:  slashedDeposit,
			RepaidDeposit:   repaidBefore,
			Slashed:         &slashedAt,
		}
		id, err := f.K.CreatePermission(f.Ctx, perm)
		require.NoError(t, err)
		return id
	}

	// ------------------------------------------------------------------ //
	// Happy path 1: partial repay — Repaid timestamp NOT set yet          //
	// ------------------------------------------------------------------ //
	t.Run("MOD-PERM-MSG-13-3: partial repay — Repaid nil after partial payment", func(t *testing.T) {
		f := NewFixture(t)
		now := time.Date(2026, 3, 1, 0, 0, 0, 0, time.UTC)
		f.SetBlockTime(now)

		corp := sdk.AccAddress([]byte("corp_repay_part__1")).String()
		f.SetBalance(corp, types.BondDenom, 10000)

		depositBefore := uint64(3000) // current deposit after slash
		slashedDeposit := uint64(5000)
		repaidBefore := uint64(0)
		repayAmount := uint64(2000) // partial — won't fully cover slashedDeposit

		id := makeSlashedPerm(t, f, now, corp, depositBefore, slashedDeposit, repaidBefore)

		_, err := f.MS.RepayPermissionSlashedTrustDeposit(f.Ctx, &types.MsgRepayPermissionSlashedTrustDeposit{
			Id:          id,
			Corporation: corp,
			Operator:    corp,
			Amount:      repayAmount,
		})
		require.NoError(t, err)

		// Full struct assertion
		got, err := f.K.GetPermissionByID(f.Ctx, id)
		require.NoError(t, err)
		require.Equal(t, specRepayDepositAfter(depositBefore, repayAmount), got.Deposit)
		require.Equal(t, specRepayRepaidDepositAfter(repaidBefore, repayAmount), got.RepaidDeposit)
		// Partial repay: repaid_deposit (2000) < slashed_deposit (5000) → Repaid must be nil
		require.Nil(t, got.Repaid, "Repaid must be nil on partial repay")
		require.NotNil(t, got.Modified)
		require.Equal(t, now, *got.Modified)

		// AdjustTrustDeposit called with positive amount
		require.Len(t, f.TDKeeper.AdjustHistory, 1)
		require.Equal(t, corp, f.TDKeeper.AdjustHistory[0].Corporation)
		require.Equal(t, int64(repayAmount), f.TDKeeper.AdjustHistory[0].Amount)
		require.Equal(t, "perm_repay_slashed_deposit", f.TDKeeper.AdjustHistory[0].Reason)

		// No direct bank transfer (TD handles the transfer internally)
		f.RequireNoBalanceChange(corp)

		// Event assertion
		f.RequireEvent(types.EventTypeRepayPermissionSlashedTrustDeposit, map[string]string{
			types.AttributeKeyPermissionID: fmt.Sprintf("%d", id),
			types.AttributeKeyRepaidAmount: fmt.Sprintf("%d", repayAmount),
			types.AttributeKeyCorporation:  corp,
		})

		// Module invariant
		f.RequireInvariant()
	})

	// ------------------------------------------------------------------ //
	// Happy path 2: full repay — Repaid timestamp IS set                  //
	// ------------------------------------------------------------------ //
	t.Run("MOD-PERM-MSG-13-3: full repay — Repaid timestamp set when repaid >= slashed", func(t *testing.T) {
		f := NewFixture(t)
		now := time.Date(2026, 3, 1, 0, 0, 0, 0, time.UTC)
		f.SetBlockTime(now)

		corp := sdk.AccAddress([]byte("corp_repay_full__1")).String()
		f.SetBalance(corp, types.BondDenom, 10000)

		depositBefore := uint64(1000)
		slashedDeposit := uint64(3000)
		repaidBefore := uint64(0)
		repayAmount := uint64(3000) // exactly covers slashedDeposit

		id := makeSlashedPerm(t, f, now, corp, depositBefore, slashedDeposit, repaidBefore)

		_, err := f.MS.RepayPermissionSlashedTrustDeposit(f.Ctx, &types.MsgRepayPermissionSlashedTrustDeposit{
			Id: id, Corporation: corp, Operator: corp, Amount: repayAmount,
		})
		require.NoError(t, err)

		got, err := f.K.GetPermissionByID(f.Ctx, id)
		require.NoError(t, err)
		require.Equal(t, specRepayDepositAfter(depositBefore, repayAmount), got.Deposit)
		require.Equal(t, specRepayRepaidDepositAfter(repaidBefore, repayAmount), got.RepaidDeposit)
		// Full repay: repaid_deposit (3000) >= slashed_deposit (3000) → Repaid must be set
		require.NotNil(t, got.Repaid, "Repaid must be set when fully repaid")
		require.Equal(t, now, *got.Repaid)

		require.Len(t, f.TDKeeper.AdjustHistory, 1)
		require.Equal(t, int64(repayAmount), f.TDKeeper.AdjustHistory[0].Amount)

		f.RequireNoBalanceChange(corp)
		f.RequireInvariant()
	})

	// ------------------------------------------------------------------ //
	// Happy path 3: second repay completes the obligation                 //
	// ------------------------------------------------------------------ //
	t.Run("MOD-PERM-MSG-13-3: two partial repays — second completes the obligation", func(t *testing.T) {
		f := NewFixture(t)
		now := time.Date(2026, 3, 1, 0, 0, 0, 0, time.UTC)
		f.SetBlockTime(now)

		corp := sdk.AccAddress([]byte("corp_repay_2step1")).String()
		f.SetBalance(corp, types.BondDenom, 20000)

		depositBefore := uint64(0)
		slashedDeposit := uint64(4000)
		repaidBefore := uint64(0)
		repay1 := uint64(1500) // partial
		repay2 := uint64(2500) // completes: 1500+2500 = 4000 >= 4000

		id := makeSlashedPerm(t, f, now, corp, depositBefore, slashedDeposit, repaidBefore)

		_, err := f.MS.RepayPermissionSlashedTrustDeposit(f.Ctx, &types.MsgRepayPermissionSlashedTrustDeposit{
			Id: id, Corporation: corp, Operator: corp, Amount: repay1,
		})
		require.NoError(t, err)

		mid, err := f.K.GetPermissionByID(f.Ctx, id)
		require.NoError(t, err)
		require.Nil(t, mid.Repaid) // not yet fully repaid

		_, err = f.MS.RepayPermissionSlashedTrustDeposit(f.Ctx, &types.MsgRepayPermissionSlashedTrustDeposit{
			Id: id, Corporation: corp, Operator: corp, Amount: repay2,
		})
		require.NoError(t, err)

		got, err := f.K.GetPermissionByID(f.Ctx, id)
		require.NoError(t, err)
		require.Equal(t, specRepayDepositAfter(specRepayDepositAfter(depositBefore, repay1), repay2), got.Deposit)
		require.Equal(t, specRepayRepaidDepositAfter(specRepayRepaidDepositAfter(repaidBefore, repay1), repay2), got.RepaidDeposit)
		require.NotNil(t, got.Repaid)

		require.Len(t, f.TDKeeper.AdjustHistory, 2)
		f.RequireInvariant()
	})

	// ------------------------------------------------------------------ //
	// Negative: [MOD-PERM-MSG-13-2-1] operator authorization failure      //
	// ------------------------------------------------------------------ //
	t.Run("MOD-PERM-MSG-13-2-1: operator authorization failure", func(t *testing.T) {
		f := NewFixture(t)
		now := time.Date(2026, 3, 1, 0, 0, 0, 0, time.UTC)
		f.SetBlockTime(now)

		corp := sdk.AccAddress([]byte("corp_authz_rpay_1")).String()
		f.SetBalance(corp, types.BondDenom, 5000)
		f.DelKeeper.ErrToReturn = fmt.Errorf("mock: not authorized")

		id := makeSlashedPerm(t, f, now, corp, 1000, 2000, 0)

		_, err := f.MS.RepayPermissionSlashedTrustDeposit(f.Ctx, &types.MsgRepayPermissionSlashedTrustDeposit{
			Id: id, Corporation: corp, Operator: sdk.AccAddress([]byte("bad_op___________1")).String(), Amount: 500,
		})
		require.ErrorContains(t, err, "authorization check failed")
		f.RequireNoBalanceChange(corp)
	})

	// ------------------------------------------------------------------ //
	// Negative: [MOD-PERM-MSG-13-2-1a] perm not found                   //
	// ------------------------------------------------------------------ //
	t.Run("MOD-PERM-MSG-13-2-1a: perm not found", func(t *testing.T) {
		f := NewFixture(t)
		corp := sdk.AccAddress([]byte("corp_notfound_rp1")).String()
		f.SetBalance(corp, types.BondDenom, 5000)

		_, err := f.MS.RepayPermissionSlashedTrustDeposit(f.Ctx, &types.MsgRepayPermissionSlashedTrustDeposit{
			Id: 99999, Corporation: corp, Operator: corp, Amount: 100,
		})
		require.ErrorContains(t, err, "perm not found")
		f.RequireNoBalanceChange(corp)
	})

	// ------------------------------------------------------------------ //
	// Negative: [MOD-PERM-MSG-13-2-1b] corporation mismatch              //
	// ------------------------------------------------------------------ //
	t.Run("MOD-PERM-MSG-13-2-1b: corporation mismatch", func(t *testing.T) {
		f := NewFixture(t)
		now := time.Date(2026, 3, 1, 0, 0, 0, 0, time.UTC)
		f.SetBlockTime(now)

		corp := sdk.AccAddress([]byte("corp_corp_rpay___1")).String()
		wrongCorp := sdk.AccAddress([]byte("corp_wrong_rpay_1")).String()
		f.SetBalance(wrongCorp, types.BondDenom, 5000)

		id := makeSlashedPerm(t, f, now, corp, 1000, 2000, 0)

		_, err := f.MS.RepayPermissionSlashedTrustDeposit(f.Ctx, &types.MsgRepayPermissionSlashedTrustDeposit{
			Id: id, Corporation: wrongCorp, Operator: wrongCorp, Amount: 500,
		})
		require.ErrorContains(t, err, "authority is not the owner of this permission")
		f.RequireNoBalanceChange(wrongCorp)
	})

	// ------------------------------------------------------------------ //
	// Negative: perm has no slashed timestamp                             //
	// ------------------------------------------------------------------ //
	t.Run("MOD-PERM-MSG-13-2: perm not slashed — no Slashed timestamp", func(t *testing.T) {
		f := NewFixture(t)
		now := time.Date(2026, 3, 1, 0, 0, 0, 0, time.UTC)
		f.SetBlockTime(now)

		corp := sdk.AccAddress([]byte("corp_noslash_rp_1")).String()
		f.SetBalance(corp, types.BondDenom, 5000)

		schemaID := uint64(1)
		f.CSKeeper.CreateMockCredentialSchema(schemaID,
			cstypes.IssuerOnboardingMode_ISSUER_ONBOARDING_MODE_OPEN,
			cstypes.VerifierOnboardingMode_VERIFIER_ONBOARDING_MODE_OPEN)

		effFrom := now.Add(-24 * time.Hour)
		effUntil := now.Add(90 * 24 * time.Hour)
		perm := types.Permission{
			SchemaId:       schemaID,
			Type:           types.PermissionType_ISSUER,
			Corporation:    corp,
			EffectiveFrom:  &effFrom,
			EffectiveUntil: &effUntil,
			Deposit:        1000,
			Slashed:        nil, // not slashed
		}
		id, err := f.K.CreatePermission(f.Ctx, perm)
		require.NoError(t, err)

		_, err = f.MS.RepayPermissionSlashedTrustDeposit(f.Ctx, &types.MsgRepayPermissionSlashedTrustDeposit{
			Id: id, Corporation: corp, Operator: corp, Amount: 100,
		})
		require.ErrorContains(t, err, "perm has no slashed timestamp")
		f.RequireNoBalanceChange(corp)
	})

	// ------------------------------------------------------------------ //
	// Negative: already fully repaid                                      //
	// ------------------------------------------------------------------ //
	t.Run("MOD-PERM-MSG-13-2: slashed deposit already fully repaid", func(t *testing.T) {
		f := NewFixture(t)
		now := time.Date(2026, 3, 1, 0, 0, 0, 0, time.UTC)
		f.SetBlockTime(now)

		corp := sdk.AccAddress([]byte("corp_fullrpd_rp_1")).String()
		f.SetBalance(corp, types.BondDenom, 5000)

		schemaID := uint64(1)
		f.CSKeeper.CreateMockCredentialSchema(schemaID,
			cstypes.IssuerOnboardingMode_ISSUER_ONBOARDING_MODE_OPEN,
			cstypes.VerifierOnboardingMode_VERIFIER_ONBOARDING_MODE_OPEN)

		effFrom := now.Add(-24 * time.Hour)
		effUntil := now.Add(90 * 24 * time.Hour)
		slashedAt := now.Add(-1 * time.Hour)
		repaidAt := now.Add(-30 * time.Minute)
		// repaid_deposit == slashed_deposit → already fully repaid
		perm := types.Permission{
			SchemaId:        schemaID,
			Type:            types.PermissionType_ISSUER,
			Corporation:     corp,
			EffectiveFrom:   &effFrom,
			EffectiveUntil:  &effUntil,
			Deposit:         2000,
			SlashedDeposit:  2000,
			RepaidDeposit:   2000, // already fully repaid
			Slashed:         &slashedAt,
			Repaid:          &repaidAt,
		}
		id, err := f.K.CreatePermission(f.Ctx, perm)
		require.NoError(t, err)

		_, err = f.MS.RepayPermissionSlashedTrustDeposit(f.Ctx, &types.MsgRepayPermissionSlashedTrustDeposit{
			Id: id, Corporation: corp, Operator: corp, Amount: 100,
		})
		require.ErrorContains(t, err, "slashed deposit already fully repaid")
		f.RequireNoBalanceChange(corp)
	})

	// ------------------------------------------------------------------ //
	// Negative: amount exceeds outstanding                                //
	// ------------------------------------------------------------------ //
	t.Run("MOD-PERM-MSG-13-2: amount exceeds outstanding slashed deposit", func(t *testing.T) {
		f := NewFixture(t)
		now := time.Date(2026, 3, 1, 0, 0, 0, 0, time.UTC)
		f.SetBlockTime(now)

		corp := sdk.AccAddress([]byte("corp_overrpay_rp1")).String()
		f.SetBalance(corp, types.BondDenom, 10000)

		// outstanding = slashed (3000) - repaid (1000) = 2000
		id := makeSlashedPerm(t, f, now, corp, 0, 3000, 1000)

		_, err := f.MS.RepayPermissionSlashedTrustDeposit(f.Ctx, &types.MsgRepayPermissionSlashedTrustDeposit{
			Id: id, Corporation: corp, Operator: corp, Amount: 5000, // > outstanding of 2000
		})
		require.ErrorContains(t, err, "exceeds outstanding slashed deposit")
		f.RequireNoBalanceChange(corp)
	})

	// ------------------------------------------------------------------ //
	// Negative: [MOD-PERM-MSG-13-2-2] insufficient balance               //
	// ------------------------------------------------------------------ //
	t.Run("MOD-PERM-MSG-13-2-2: insufficient balance", func(t *testing.T) {
		f := NewFixture(t)
		now := time.Date(2026, 3, 1, 0, 0, 0, 0, time.UTC)
		f.SetBlockTime(now)

		corp := sdk.AccAddress([]byte("corp_nofund_rp__1")).String()
		// Set balance to only 1 uvna — far less than repay amount
		f.SetBalance(corp, types.BondDenom, 1)

		id := makeSlashedPerm(t, f, now, corp, 0, 5000, 0)

		_, err := f.MS.RepayPermissionSlashedTrustDeposit(f.Ctx, &types.MsgRepayPermissionSlashedTrustDeposit{
			Id: id, Corporation: corp, Operator: corp, Amount: 1000,
		})
		require.ErrorContains(t, err, "insufficient funds")
		f.RequireNoBalanceChange(corp)
	})
}
```

- [ ] **Step 1.2: Verify build.**

  ```bash
  go build ./x/perm/keeper/...
  ```
  Expected: no output, exit 0.

- [ ] **Step 1.3: Run new tests.**

  ```bash
  go test ./x/perm/keeper/... -run TestMsgRepayPermissionSlashedTrustDeposit -v -count=1
  ```
  Expected: all subtests PASS.

- [ ] **Step 1.4: Commit.**

  ```bash
  git add x/perm/keeper/msg_repay_permission_slashed_trust_deposit_test.go
  git commit -m "test(perm): add fixture-based RepayPermissionSlashedTrustDeposit tests"
  ```

---

## Task 2: Delete old tests for `RepayPermissionSlashedTrustDeposit`

**File:** `x/perm/keeper/msg_server_test.go`

- [ ] **Step 2.1: Delete the `TestMsgRepayPermissionSlashedTrustDeposit` function block.**

  Remove the entire `func TestMsgRepayPermissionSlashedTrustDeposit(t *testing.T)` block. Keep all other test functions intact.

- [ ] **Step 2.2: Verify build and test suite.**

  ```bash
  go test ./x/perm/keeper/... -count=1
  ```
  Expected: all pass.

- [ ] **Step 2.3: Commit.**

  ```bash
  git add x/perm/keeper/msg_server_test.go
  git commit -m "test(perm): delete legacy TestMsgRepayPermissionSlashedTrustDeposit"
  ```

---

## Task 3: Final pass

- [ ] **Step 3.1: Run full PERM keeper test suite.**

  ```bash
  go test ./x/perm/keeper/... -v -count=1
  ```

- [ ] **Step 3.2: Race detector.**

  ```bash
  go test ./x/perm/keeper/... -race -count=1
  ```

- [ ] **Step 3.3: Vet and lint.**

  ```bash
  go vet ./x/perm/keeper/... && golangci-lint run ./x/perm/keeper/...
  ```

- [ ] **Step 3.4: Coverage check.**

  ```bash
  go test ./x/perm/keeper/ -cover -count=1
  ```
  Expected: coverage ≥95%.

- [ ] **Step 3.5: Full repo sanity.**

  ```bash
  go build ./... && go test ./... -count=1
  ```

- [ ] **Step 3.6: Push and open PR.**

  ```bash
  git push -u origin test/step-30-perm-repay-slashed-deposit
  gh pr create --title "test(perm): fixture-based RepayPermissionSlashedTrustDeposit tests (step 30)" --body "$(cat <<'EOF'
  ## Summary
  - Adds fixture-based TestMsgRepayPermissionSlashedTrustDeposit
  - Covers partial repay (Repaid nil), full repay (Repaid set), two-step repay completion
  - AdjustHistory assertions verify AdjustTrustDeposit call with correct positive amount
  - All tests call RequireNoBalanceChange (no direct bank transfer in RepayPermissionSlashedTrustDeposit)
  - Spec formula functions: specRepayDepositAfter, specRepayRepaidDepositAfter, specRepayIsFullyRepaid
  - 7 negative subtests covering every precondition and edge case
  - Deletes legacy TestMsgRepayPermissionSlashedTrustDeposit from msg_server_test.go

  ## Test plan
  - [ ] `go test ./x/perm/keeper/... -race -count=1` passes
  - [ ] `go test ./x/perm/keeper/ -cover` reports ≥95%
  - [ ] `golangci-lint run ./x/perm/keeper/...` clean
  - [ ] `go test ./...` passes — no regressions
  EOF
  )"
  ```

---

## "Done" Criteria — Step 30

- [ ] `x/perm/keeper/msg_repay_permission_slashed_trust_deposit_test.go` exists.
- [ ] Spec formulas `specRepayDepositAfter`, `specRepayRepaidDepositAfter`, `specRepayIsFullyRepaid` defined.
- [ ] Three happy paths: partial repay, full repay, two-step repay.
- [ ] Every happy path: full struct assertion (Deposit, RepaidDeposit, Repaid) + AdjustHistory + event + invariant + `RequireNoBalanceChange`.
- [ ] Seven negative subtests covering every precondition.
- [ ] Legacy `TestMsgRepayPermissionSlashedTrustDeposit` deleted from `msg_server_test.go`.
- [ ] `go test ./x/perm/keeper/... -race -count=1` passes.
- [ ] Coverage ≥95% on `x/perm/keeper/`.
