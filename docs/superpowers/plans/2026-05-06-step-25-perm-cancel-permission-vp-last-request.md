# Step 25: PERM CancelPermissionVPLastRequest — Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Replace the legacy `TestCancelPermissionVPLastRequest` test with a fixture-based test that verifies the refund bank transfer from module to corporation, the trust deposit release, VpState transitions, and the PERM module invariant for every spec precondition in `[MOD-PERM-MSG-6]`.

**Architecture:** The `Fixture` struct already exists from step 21 (`x/perm/keeper/fixture_test.go`). This step only creates `x/perm/keeper/cancel_permission_vp_last_request_test.go`. The key execution is in `executeCancelPermissionVPLastRequest` (lines 480–543 of `msg_server.go`): fees are refunded from module to applicant; deposit is released via `AdjustTrustDeposit`. The `VpState` transition depends on whether `VpExp` is nil — TERMINATED for first VP (never validated), VALIDATED for renewals. Both transitions are tested as separate happy paths.

**Tech Stack:** Go 1.24, Cosmos SDK v0.50.x, testify/require, StatefulBankMock from testutil/keeper/bank.go

---

## Pre-flight

- [ ] **Worktree.** Create an isolated worktree for this step. Branch name: `test/step-25-perm-cancel-permission-vp-last-request`.
- [ ] **Gate check.** Confirm step 21 is merged: `x/perm/keeper/fixture_test.go` must exist and `go build ./x/perm/keeper/...` must pass.
- [ ] **Read the implementation.** Before writing tests, read `x/perm/keeper/msg_server.go` lines 423–543 — `CancelPermissionVPLastRequest` and `executeCancelPermissionVPLastRequest`.
- [ ] **Sanity check.**

  Run: `go build ./... && go vet ./...`
  Expected: no output, exit 0.

---

## File Structure

- **Create:** `x/perm/keeper/cancel_permission_vp_last_request_test.go` — new fixture-based tests.
- **Fixture already exists:** `x/perm/keeper/fixture_test.go` (from step 21) — do not recreate.
- **Delete (in same PR):** the `TestCancelPermissionVPLastRequest` test function from `x/perm/keeper/msg_server_test.go`.

---

## Task 1: Write `cancel_permission_vp_last_request_test.go`

**File:** `x/perm/keeper/cancel_permission_vp_last_request_test.go`

Key execution path of `executeCancelPermissionVPLastRequest`:

1. Set `Modified = now`, `VpLastStateChange = now`.
2. If `perm.VpExp == nil`: `VpState = TERMINATED`; else `VpState = VALIDATED`.
3. If `VpCurrentFees > 0`: `SendCoinsFromModuleToAccount(perm_module, corporation, vp_current_fees)` — refund to applicant.
4. If `VpCurrentDeposit > 0`: `AdjustTrustDeposit(corporation, -vp_current_deposit, "perm_deactivate_release_deposit")`.
5. Set `VpCurrentFees = 0`, `VpCurrentDeposit = 0`.
6. Persist.

The precondition check `if applicantPerm.Slashed != nil && applicantPerm.Repaid == nil` must be tested — a permission with a slashed-but-unrepaid deposit cannot be cancelled.

- [ ] **Step 1.1: Create the test file.**

```go
package keeper_test

import (
	"fmt"
	"strconv"
	"testing"
	"time"

	authtypes "github.com/cosmos/cosmos-sdk/x/auth/types"
	"github.com/stretchr/testify/require"

	cstypes "github.com/verana-labs/verana/x/cs/types"
	"github.com/verana-labs/verana/x/perm/types"
)

// ============================================================================
// Spec formula functions — independent of the implementation
// ============================================================================

// specCancelRefund returns the fees to be refunded to the corporation on
// cancellation per [MOD-PERM-MSG-6-3]: refund = vp_current_fees.
func specCancelRefund(vpCurrentFees uint64) uint64 {
	return vpCurrentFees
}

// specCancelDepositRelease returns the deposit amount to release per [MOD-PERM-MSG-6-3]:
// release = vp_current_deposit (passed as negative to AdjustTrustDeposit).
func specCancelDepositRelease(vpCurrentDeposit uint64) uint64 {
	return vpCurrentDeposit
}

// specCancelVpStateNoExp returns the expected VpState when vp_exp is nil
// (first VP, never validated): TERMINATED.
func specCancelVpStateNoExp() types.ValidationState {
	return types.ValidationState_TERMINATED
}

// specCancelVpStateWithExp returns the expected VpState when vp_exp is not nil
// (renewal, previously validated): VALIDATED.
func specCancelVpStateWithExp() types.ValidationState {
	return types.ValidationState_VALIDATED
}

// ============================================================================
// Test helpers
// ============================================================================

// makePendingPermForCancel builds a PENDING permission for CancelPermissionVPLastRequest tests.
// If vpExp is nil, the perm has never been validated (VpState will go to TERMINATED on cancel).
// If vpExp is non-nil, the perm is a renewal (VpState will go to VALIDATED on cancel).
func makePendingPermForCancel(
	f *Fixture,
	applicantCorp string,
	vpCurrentFees uint64,
	vpCurrentDeposit uint64,
	vpExp *time.Time,
	slashedAt *time.Time,
	repaidAt *time.Time,
) uint64 {
	f.t.Helper()
	now := f.Ctx.BlockTime()
	perm := types.Permission{
		SchemaId:         1,
		Type:             types.PermissionType_ISSUER,
		Corporation:      applicantCorp,
		ValidatorPermId:  0,
		Created:          &now,
		Modified:         &now,
		VpState:          types.ValidationState_PENDING,
		VpLastStateChange: &now,
		VpCurrentFees:    vpCurrentFees,
		VpCurrentDeposit: vpCurrentDeposit,
		Deposit:          vpCurrentDeposit,
		VpExp:            vpExp,
		Slashed:          slashedAt,
		Repaid:           repaidAt,
	}
	id, err := f.K.CreatePermission(f.Ctx, perm)
	require.NoError(f.t, err)
	return id
}

// ============================================================================
// TestMsgCancelPermissionVPLastRequest
// ============================================================================

func TestMsgCancelPermissionVPLastRequest(t *testing.T) {
	baseTime := time.Date(2026, 5, 1, 12, 0, 0, 0, time.UTC)

	setup := func(t *testing.T) (*Fixture, string) {
		t.Helper()
		f := NewFixture(t)
		f.SetBlockTime(baseTime)

		corp := "cosmos1qqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqnrql8a"
		trID := f.TRKeeper.CreateMockTrustRegistry(corp, "did:example:tr1")
		f.CSKeeper.CreateMockCredentialSchemaFull(cstypes.CredentialSchema{
			Id: 1, TrId: trID,
			IssuerOnboardingMode: cstypes.IssuerOnboardingMode_ISSUER_ONBOARDING_MODE_ECOSYSTEM_VALIDATION_PROCESS,
		})
		return f, corp
	}

	// -----------------------------------------------------------------------
	// Happy path 1: cancel first VP (vp_exp == nil) → VpState = TERMINATED,
	// fees refunded, deposit released
	// -----------------------------------------------------------------------
	t.Run("MOD-PERM-MSG-6: cancel first VP refunds fees and sets TERMINATED", func(t *testing.T) {
		f, corp := setup(t)
		modAddr := authtypes.NewModuleAddress(types.ModuleName).String()

		vpCurrentFees := uint64(30)
		vpCurrentDeposit := uint64(6)

		// Pre-fund module with escrowed fees
		f.SetModuleBalance(int64(vpCurrentFees))
		f.SetBalance(corp, 0)

		// perm with no vp_exp → first VP
		permID := makePendingPermForCancel(f, corp, vpCurrentFees, vpCurrentDeposit, nil, nil, nil)

		refund := specCancelRefund(vpCurrentFees)         // 30
		depositRelease := specCancelDepositRelease(vpCurrentDeposit) // 6
		expectedVpState := specCancelVpStateNoExp()        // TERMINATED

		msg := &types.MsgCancelPermissionVPLastRequest{
			Id:          permID,
			Corporation: corp,
			Operator:    corp,
		}
		_, err := f.MS.CancelPermissionVPLastRequest(f.Ctx, msg)
		require.NoError(t, err)

		// Bank: module loses fees, applicant gains fees (refund)
		f.RequireBalanceDelta(modAddr, -int64(refund))
		f.RequireBalanceDelta(corp, int64(refund))

		// Permission state
		stored, err := f.K.Permission.Get(f.Ctx, permID)
		require.NoError(t, err)
		require.Equal(t, expectedVpState, stored.VpState)
		require.Equal(t, uint64(0), stored.VpCurrentFees)
		require.Equal(t, uint64(0), stored.VpCurrentDeposit)
		require.NotNil(t, stored.Modified)
		require.NotNil(t, stored.VpLastStateChange)

		// Deposit release: AdjustTrustDeposit called with -depositRelease
		require.NotEmpty(t, f.TDKeeper.AdjustCalls)
		require.Equal(t, corp, f.TDKeeper.AdjustCalls[0].Corporation)
		require.Equal(t, -int64(depositRelease), f.TDKeeper.AdjustCalls[0].Amount)
		require.Equal(t, "perm_deactivate_release_deposit", f.TDKeeper.AdjustCalls[0].Reason)

		// Event
		f.RequireEvent(types.EventTypeCancelPermissionVPLastRequest, map[string]string{
			types.AttributeKeyPermissionID: strconv.FormatUint(permID, 10),
			types.AttributeKeyCorporation:  corp,
			types.AttributeKeyOperator:     corp,
		})

		// Invariant: module balance = sum(VpCurrentFees for PENDING) = 0
		f.RequireInvariant()
	})

	// -----------------------------------------------------------------------
	// Happy path 2: cancel renewal (vp_exp != nil) → VpState = VALIDATED
	// -----------------------------------------------------------------------
	t.Run("MOD-PERM-MSG-6: cancel renewal VP sets VALIDATED state", func(t *testing.T) {
		f, corp := setup(t)
		modAddr := authtypes.NewModuleAddress(types.ModuleName).String()

		vpCurrentFees := uint64(15)
		vpCurrentDeposit := uint64(3)
		vpExp := baseTime.Add(200 * 24 * time.Hour)

		f.SetModuleBalance(int64(vpCurrentFees))
		f.SetBalance(corp, 0)

		permID := makePendingPermForCancel(f, corp, vpCurrentFees, vpCurrentDeposit, &vpExp, nil, nil)

		expectedVpState := specCancelVpStateWithExp() // VALIDATED

		msg := &types.MsgCancelPermissionVPLastRequest{
			Id:          permID,
			Corporation: corp,
			Operator:    corp,
		}
		_, err := f.MS.CancelPermissionVPLastRequest(f.Ctx, msg)
		require.NoError(t, err)

		f.RequireBalanceDelta(modAddr, -int64(vpCurrentFees))
		f.RequireBalanceDelta(corp, int64(vpCurrentFees))

		stored, err := f.K.Permission.Get(f.Ctx, permID)
		require.NoError(t, err)
		require.Equal(t, expectedVpState, stored.VpState) // VALIDATED not TERMINATED
		require.Equal(t, uint64(0), stored.VpCurrentFees)
		require.Equal(t, uint64(0), stored.VpCurrentDeposit)

		f.RequireInvariant()
	})

	// -----------------------------------------------------------------------
	// Happy path 3: zero fees and zero deposit — no bank transfer, no adjust call
	// -----------------------------------------------------------------------
	t.Run("MOD-PERM-MSG-6: zero fees and deposit — no bank transfer", func(t *testing.T) {
		f, corp := setup(t)

		f.SetModuleBalance(0)
		f.SetBalance(corp, 0)

		permID := makePendingPermForCancel(f, corp, 0, 0, nil, nil, nil)

		_, err := f.MS.CancelPermissionVPLastRequest(f.Ctx, &types.MsgCancelPermissionVPLastRequest{
			Id: permID, Corporation: corp, Operator: corp,
		})
		require.NoError(t, err)

		// No bank movement
		f.RequireModuleBalanceDelta(0)
		f.RequireNoBalanceChange(corp)

		// No deposit adjust call
		require.Empty(t, f.TDKeeper.AdjustCalls)

		stored, err := f.K.Permission.Get(f.Ctx, permID)
		require.NoError(t, err)
		require.Equal(t, types.ValidationState_TERMINATED, stored.VpState)

		f.RequireInvariant()
	})

	// -----------------------------------------------------------------------
	// Negative: [MOD-PERM-MSG-6-2-1] operator authorization failure
	// -----------------------------------------------------------------------
	t.Run("MOD-PERM-MSG-6-2-1: fails if operator authorization fails", func(t *testing.T) {
		f, corp := setup(t)
		f.SetModuleBalance(10)
		f.SetBalance(corp, 0)
		f.DelKeeper.ErrToReturn = fmt.Errorf("mock: unauthorized")

		permID := makePendingPermForCancel(f, corp, 10, 2, nil, nil, nil)

		_, err := f.MS.CancelPermissionVPLastRequest(f.Ctx, &types.MsgCancelPermissionVPLastRequest{
			Id: permID, Corporation: corp, Operator: "cosmos1badoperator00000000000000000000aaaaaa",
		})
		require.ErrorContains(t, err, "authorization check failed")
		f.RequireNoBalanceChange(corp)
		f.RequireModuleBalanceDelta(0)
	})

	// -----------------------------------------------------------------------
	// Negative: [MOD-PERM-MSG-6-2-1a] perm not found
	// -----------------------------------------------------------------------
	t.Run("MOD-PERM-MSG-6-2-1a: fails if perm id does not exist", func(t *testing.T) {
		f, corp := setup(t)
		f.SetBalance(corp, 0)

		_, err := f.MS.CancelPermissionVPLastRequest(f.Ctx, &types.MsgCancelPermissionVPLastRequest{
			Id: 9999, Corporation: corp, Operator: corp,
		})
		require.ErrorContains(t, err, "perm not found")
		f.RequireNoBalanceChange(corp)
	})

	// -----------------------------------------------------------------------
	// Negative: [MOD-PERM-MSG-6-2-1b] corporation mismatch
	// -----------------------------------------------------------------------
	t.Run("MOD-PERM-MSG-6-2-1b: fails if corporation does not match perm.Corporation", func(t *testing.T) {
		f, corp := setup(t)
		f.SetBalance(corp, 0)

		permID := makePendingPermForCancel(f, corp, 10, 0, nil, nil, nil)

		wrongCorp := "cosmos1qqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqpjdpjx"
		f.SetBalance(wrongCorp, 0)

		_, err := f.MS.CancelPermissionVPLastRequest(f.Ctx, &types.MsgCancelPermissionVPLastRequest{
			Id: permID, Corporation: wrongCorp, Operator: wrongCorp,
		})
		require.ErrorContains(t, err, "authority is not the perm authority")
		f.RequireNoBalanceChange(wrongCorp)
	})

	// -----------------------------------------------------------------------
	// Negative: [MOD-PERM-MSG-6-2-1c] perm.VpState is not PENDING
	// -----------------------------------------------------------------------
	t.Run("MOD-PERM-MSG-6-2-1c: fails if perm.VpState is not PENDING", func(t *testing.T) {
		f, corp := setup(t)
		f.SetBalance(corp, 0)

		// Create a VALIDATED perm instead of PENDING
		now := f.Ctx.BlockTime()
		past := now.Add(-48 * time.Hour)
		future := now.Add(90 * 24 * time.Hour)
		validatedPerm := types.Permission{
			SchemaId: 1, Type: types.PermissionType_ISSUER, Corporation: corp,
			Created: &now, Modified: &now, VpState: types.ValidationState_VALIDATED,
			EffectiveFrom: &past, EffectiveUntil: &future,
		}
		validatedPermID, err := f.K.CreatePermission(f.Ctx, validatedPerm)
		require.NoError(t, err)

		_, err = f.MS.CancelPermissionVPLastRequest(f.Ctx, &types.MsgCancelPermissionVPLastRequest{
			Id: validatedPermID, Corporation: corp, Operator: corp,
		})
		require.ErrorContains(t, err, "perm must be in PENDING state")
		f.RequireNoBalanceChange(corp)
	})

	// -----------------------------------------------------------------------
	// Negative: [MOD-PERM-MSG-6-2-1d] deposit slashed and not repaid
	// -----------------------------------------------------------------------
	t.Run("MOD-PERM-MSG-6-2-1d: fails if deposit is slashed and not repaid", func(t *testing.T) {
		f, corp := setup(t)
		f.SetModuleBalance(10)
		f.SetBalance(corp, 0)

		slashedAt := baseTime.Add(-24 * time.Hour)
		// slashed is set, repaid is nil → abort
		permID := makePendingPermForCancel(f, corp, 10, 2, nil, &slashedAt, nil)

		_, err := f.MS.CancelPermissionVPLastRequest(f.Ctx, &types.MsgCancelPermissionVPLastRequest{
			Id: permID, Corporation: corp, Operator: corp,
		})
		require.ErrorContains(t, err, "permission deposit has been slashed and not repaid")
		f.RequireNoBalanceChange(corp)
		f.RequireModuleBalanceDelta(0)
	})

	// -----------------------------------------------------------------------
	// Edge: slashed AND repaid — cancellation is allowed
	// -----------------------------------------------------------------------
	t.Run("edge: slashed and fully repaid allows cancellation", func(t *testing.T) {
		f, corp := setup(t)
		modAddr := authtypes.NewModuleAddress(types.ModuleName).String()

		vpCurrentFees := uint64(20)
		f.SetModuleBalance(int64(vpCurrentFees))
		f.SetBalance(corp, 0)

		slashedAt := baseTime.Add(-48 * time.Hour)
		repaidAt := baseTime.Add(-24 * time.Hour)
		// slashed != nil AND repaid != nil → allowed
		permID := makePendingPermForCancel(f, corp, vpCurrentFees, 0, nil, &slashedAt, &repaidAt)

		_, err := f.MS.CancelPermissionVPLastRequest(f.Ctx, &types.MsgCancelPermissionVPLastRequest{
			Id: permID, Corporation: corp, Operator: corp,
		})
		require.NoError(t, err)

		f.RequireBalanceDelta(modAddr, -int64(vpCurrentFees))
		f.RequireBalanceDelta(corp, int64(vpCurrentFees))

		stored, err := f.K.Permission.Get(f.Ctx, permID)
		require.NoError(t, err)
		require.Equal(t, types.ValidationState_TERMINATED, stored.VpState)
		require.Equal(t, uint64(0), stored.VpCurrentFees)

		f.RequireInvariant()
	})

	// -----------------------------------------------------------------------
	// Edge: multiple pending perms — only target perm is affected
	// -----------------------------------------------------------------------
	t.Run("edge: only target perm is cancelled, other perms unaffected", func(t *testing.T) {
		f, corp := setup(t)
		modAddr := authtypes.NewModuleAddress(types.ModuleName).String()

		// Create two pending perms
		fees1 := uint64(10)
		fees2 := uint64(20)
		f.SetModuleBalance(int64(fees1 + fees2))
		f.SetBalance(corp, 0)

		corp2 := "cosmos1qqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqpjdpjx"
		f.SetBalance(corp2, 0)

		permID1 := makePendingPermForCancel(f, corp, fees1, 0, nil, nil, nil)
		permID2 := makePendingPermForCancel(f, corp2, fees2, 0, nil, nil, nil)

		// Cancel only perm1
		_, err := f.MS.CancelPermissionVPLastRequest(f.Ctx, &types.MsgCancelPermissionVPLastRequest{
			Id: permID1, Corporation: corp, Operator: corp,
		})
		require.NoError(t, err)

		// Corp gets fees1 back; module loses fees1
		f.RequireBalanceDelta(corp, int64(fees1))
		f.RequireBalanceDelta(modAddr, -int64(fees1))

		// Corp2 and perm2 unaffected
		f.RequireNoBalanceChange(corp2)
		stored2, err := f.K.Permission.Get(f.Ctx, permID2)
		require.NoError(t, err)
		require.Equal(t, types.ValidationState_PENDING, stored2.VpState)
		require.Equal(t, fees2, stored2.VpCurrentFees)

		// Invariant: remaining PENDING fees = fees2 = module_balance (30 - 10 = 20)
		f.RequireInvariant()
	})
}
```

- [ ] **Step 1.2: Verify build.**

  Run: `go build ./x/perm/keeper/...`
  Expected: no output, exit 0.

- [ ] **Step 1.3: Run new tests.**

  Run: `go test ./x/perm/keeper/... -run TestMsgCancelPermissionVPLastRequest -v -count=1`
  Expected: all subtests PASS.

---

## Task 2: Delete old `TestCancelPermissionVPLastRequest` from `msg_server_test.go`

**File:** `x/perm/keeper/msg_server_test.go`

- [ ] **Step 2.1: Delete the old test function.**

Find and remove the entire `func TestCancelPermissionVPLastRequest(t *testing.T)` block. Do not touch any other test functions.

- [ ] **Step 2.2: Verify build and full test suite.**

  Run: `go test ./x/perm/keeper/... -count=1`
  Expected: no compilation errors, all remaining tests PASS.

- [ ] **Step 2.3: Commit.**

```bash
git add x/perm/keeper/cancel_permission_vp_last_request_test.go
git add x/perm/keeper/msg_server_test.go
git commit -m "test(perm): add fixture-based CancelPermissionVPLastRequest tests, delete legacy"
```

---

## Task 3: Final pass

- [ ] **Step 3.1: Run the full PERM keeper test suite.**

  Run: `go test ./x/perm/keeper/... -v -count=1`
  Expected: PASS.

- [ ] **Step 3.2: Run race detector.**

  Run: `go test ./x/perm/keeper/... -race -count=1`
  Expected: PASS, no DATA RACE reports.

- [ ] **Step 3.3: Run vet and lint.**

  Run: `go vet ./x/perm/keeper/... && golangci-lint run ./x/perm/keeper/...`
  Expected: no output.

- [ ] **Step 3.4: Full repo sanity.**

  Run: `go build ./... && go test ./... -count=1`
  Expected: PASS.

- [ ] **Step 3.5: Push and open PR.**

```bash
git push -u origin test/step-25-perm-cancel-permission-vp-last-request
gh pr create --title "test(perm): fixture-based CancelPermissionVPLastRequest tests (step 25)" --body "$(cat <<'EOF'
## Summary
- Creates x/perm/keeper/cancel_permission_vp_last_request_test.go with fixture-based tests
- specCancelRefund and specCancelDepositRelease computed independently of implementation
- Bank refund verified: module loses VpCurrentFees, corporation gains VpCurrentFees
- AdjustTrustDeposit release call verified: -VpCurrentDeposit to corporation
- VpState transitions tested: TERMINATED (nil vp_exp) and VALIDATED (non-nil vp_exp)
- All 4 preconditions tested + 2 edge cases (repaid slash allowed; partial cancellation invariant)
- Invariant enforced after every happy path

## Test plan
- [ ] go test ./x/perm/keeper/... -race -count=1 passes
- [ ] go test ./x/perm/keeper/ -cover reports ≥85%
- [ ] golangci-lint run ./x/perm/keeper/... clean
- [ ] go test ./... (full repo) passes — no regressions
EOF
)"
```

---

## "Done" Criteria — Step 25

- [ ] `x/perm/keeper/cancel_permission_vp_last_request_test.go` exists with 3 happy paths, 4 negative cases, 2 edge cases.
- [ ] Happy path 1 (nil vp_exp): `VpState=TERMINATED`, refund delta verified, `AdjustCalls[0].Amount == -vpCurrentDeposit`.
- [ ] Happy path 2 (non-nil vp_exp): `VpState=VALIDATED`.
- [ ] Happy path 3 (zero fees/deposit): no bank transfer, no `AdjustCalls`.
- [ ] Every happy path: `RequireInvariant()` passes (sum(VpCurrentFees for PENDING) == module balance).
- [ ] Slashed-not-repaid negative case: `ErrorContains("permission deposit has been slashed and not repaid")`.
- [ ] Edge case multi-perm: only target perm affected, invariant holds with remaining perm's fees.
- [ ] Legacy `TestCancelPermissionVPLastRequest` deleted from `msg_server_test.go`.
- [ ] `go test ./x/perm/keeper/... -race -count=1` passes.
