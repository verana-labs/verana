# Step 23: PERM RenewPermissionVP — Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Replace the legacy `TestRenewPermissionVP` test with a fixture-based test that verifies exact fee deductions via `StatefulBankMock`, full `Permission` struct mutations, event emission, and the PERM module invariant for every spec precondition in `[MOD-PERM-MSG-2]`.

**Architecture:** The `Fixture` struct already exists from step 21 (`x/perm/keeper/fixture_test.go`). This step only creates `x/perm/keeper/renew_permission_vp_test.go`. The key difference from `StartPermissionVP` is that `RenewPermissionVP` mutates an **existing** VALIDATED permission — it does not create a new one. The test must set up a fully VALIDATED applicant perm (with `EffectiveFrom` set) and a valid validator perm, then assert delta mutations on the stored permission. Fee formulas are identical to step 22; the same `specStartVPFees`/`specStartVPDeposit` functions are used (defined per-file to keep each test file self-contained).

**Tech Stack:** Go 1.24, Cosmos SDK v0.50.x, testify/require, StatefulBankMock from testutil/keeper/bank.go

---

## Pre-flight

- [ ] **Worktree.** Create an isolated worktree for this step. Branch name: `test/step-23-perm-renew-permission-vp`.
- [ ] **Gate check.** Confirm step 21 is merged: `x/perm/keeper/fixture_test.go` must exist and `go build ./x/perm/keeper/...` must pass.
- [ ] **Sanity check.**

  Run: `go build ./... && go vet ./...`
  Expected: no output, exit 0.

---

## File Structure

- **Create:** `x/perm/keeper/renew_permission_vp_test.go` — new fixture-based tests for `RenewPermissionVP`.
- **Fixture already exists:** `x/perm/keeper/fixture_test.go` (from step 21) — do not recreate.
- **Delete (in same PR):** the `TestRenewPermissionVP` test function from `x/perm/keeper/msg_server_test.go`.

---

## Task 1: Write `renew_permission_vp_test.go`

**File:** `x/perm/keeper/renew_permission_vp_test.go`

Read `x/perm/keeper/msg_server.go` lines 88–217 (the `RenewPermissionVP` and `executeRenewPermissionVP` functions) before writing. Key execution path:

1. Operator authorization check.
2. Load `applicantPerm` by `msg.Id` — must exist.
3. `applicantPerm.Corporation` must equal `msg.Corporation`.
4. `applicantPerm.VpState` must be `VALIDATED`.
5. `IsValidPermission(applicantPerm, now)` — must be active.
6. Load `validatorPerm` from `applicantPerm.ValidatorPermId` — must exist and be valid.
7. `validateAndCalculateFees(validatorPerm)` — same formula as StartVP.
8. `executeRenewPermissionVP`: if deposit > 0 call `AdjustTrustDeposit`; if fees > 0 call `SendCoinsFromAccountToModule`; mutate `perm.VpState=PENDING`, `perm.Deposit+=deposit`, `perm.VpCurrentFees=fees`, `perm.VpCurrentDeposit=deposit`.

The test must assert **delta mutations**, not absolute values, for the `Deposit` field because it was already set during the original `StartVP/Validate` cycle.

- [ ] **Step 1.1: Create the test file.**

```go
package keeper_test

import (
	"fmt"
	"strconv"
	"testing"
	"time"

	"cosmossdk.io/math"
	authtypes "github.com/cosmos/cosmos-sdk/x/auth/types"
	"github.com/stretchr/testify/require"

	cstypes "github.com/verana-labs/verana/x/cs/types"
	"github.com/verana-labs/verana/x/perm/types"
)

// ============================================================================
// Spec formula functions — independent of the implementation
// ============================================================================

// specRenewVPFees returns the validation fee for a renewal per [MOD-PERM-MSG-2-2-3]:
// fees = validatorPerm.ValidationFees * trustUnitPrice.
// (Identical formula to StartVP; redefined here for self-contained file.)
func specRenewVPFees(validationFees, trustUnitPrice uint64) uint64 {
	return validationFees * trustUnitPrice
}

// specRenewVPDeposit returns the trust deposit increment for a renewal:
// deposit = fees * trustDepositRate (truncated integer).
func specRenewVPDeposit(fees uint64, depositRate math.LegacyDec) uint64 {
	if depositRate.IsZero() {
		return 0
	}
	result := math.LegacyNewDecFromInt(math.NewIntFromUint64(fees)).Mul(depositRate).TruncateInt()
	return result.Uint64()
}

// specRenewVPDepositAfter returns the expected perm.Deposit after a renewal:
// depositAfter = depositBefore + newDeposit.
func specRenewVPDepositAfter(depositBefore, newDeposit uint64) uint64 {
	return depositBefore + newDeposit
}

// ============================================================================
// Test helpers
// ============================================================================

// makeValidatedPerm creates a VALIDATED ISSUER permission with the given parameters,
// wired to a specific validator perm. It simulates the state after a successful
// StartPermissionVP + SetPermissionVPToValidated cycle without calling those messages.
func makeValidatedPerm(
	f *Fixture,
	schemaID uint64,
	applicantCorp string,
	validatorPermID uint64,
	existingDeposit uint64,
	effectiveFrom time.Time,
	effectiveUntil time.Time,
) uint64 {
	f.t.Helper()
	now := f.Ctx.BlockTime()
	vpExp := effectiveUntil.Add(90 * 24 * time.Hour) // vp_exp beyond effective_until
	perm := types.Permission{
		SchemaId:         schemaID,
		Type:             types.PermissionType_ISSUER,
		Corporation:      applicantCorp,
		ValidatorPermId:  validatorPermID,
		Created:          &now,
		Modified:         &now,
		VpState:          types.ValidationState_VALIDATED,
		VpLastStateChange: &now,
		EffectiveFrom:    &effectiveFrom,
		EffectiveUntil:   &effectiveUntil,
		VpExp:            &vpExp,
		Deposit:          existingDeposit,
		VpCurrentFees:    0,
		VpCurrentDeposit: 0,
		ValidationFees:   5,
		IssuanceFees:     2,
		VerificationFees: 1,
	}
	id, err := f.K.CreatePermission(f.Ctx, perm)
	require.NoError(f.t, err)
	return id
}

// ============================================================================
// TestMsgRenewPermissionVP
// ============================================================================

func TestMsgRenewPermissionVP(t *testing.T) {
	baseTime := time.Date(2026, 3, 1, 12, 0, 0, 0, time.UTC)
	past := baseTime.Add(-48 * time.Hour)
	// Permission is active (effective_from < now, effective_until > now)
	effectiveFrom := baseTime.Add(-24 * time.Hour)
	effectiveUntil := baseTime.Add(180 * 24 * time.Hour)
	validatorUntil := baseTime.Add(365 * 24 * time.Hour)

	setupBase := func(t *testing.T) (*Fixture, uint64, uint64, string, string) {
		t.Helper()
		f := NewFixture(t)
		f.SetBlockTime(baseTime)

		corp := "cosmos1qqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqnrql8a"
		applicantCorp := "cosmos1qqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqpjdpjx"

		trID := f.TRKeeper.CreateMockTrustRegistry(corp, "did:example:tr1")
		f.CSKeeper.CreateMockCredentialSchemaFull(cstypes.CredentialSchema{
			Id:                   1,
			TrId:                 trID,
			IssuerOnboardingMode: cstypes.IssuerOnboardingMode_ISSUER_ONBOARDING_MODE_ECOSYSTEM_VALIDATION_PROCESS,
		})

		now := f.Ctx.BlockTime()
		validatorPerm := types.Permission{
			SchemaId: 1, Type: types.PermissionType_ECOSYSTEM, Corporation: corp,
			Created: &now, Modified: &now, VpState: types.ValidationState_VALIDATED,
			EffectiveFrom: &past, EffectiveUntil: &validatorUntil,
			ValidationFees: 8, // fees = 8 * trustUnitPrice
		}
		validatorPermID, err := f.K.CreatePermission(f.Ctx, validatorPerm)
		require.NoError(t, err)

		applicantPermID := makeValidatedPerm(f, 1, applicantCorp, validatorPermID, 40, effectiveFrom, effectiveUntil)
		return f, validatorPermID, applicantPermID, corp, applicantCorp
	}

	// -----------------------------------------------------------------------
	// Happy path: renewal with zero fees (trustUnitPrice=1, validationFees=0)
	// -----------------------------------------------------------------------
	t.Run("MOD-PERM-MSG-2: valid renewal with zero fees", func(t *testing.T) {
		f, _, applicantPermID, _, applicantCorp := setupBase(t)

		// Override validator perm to have ValidationFees=0 so no bank transfer
		// Do this by reading and re-writing the stored validator perm.
		vp, err := f.K.Permission.Get(f.Ctx, 1) // first perm is validator
		require.NoError(t, err)
		vp.ValidationFees = 0
		require.NoError(t, f.K.UpdatePermission(f.Ctx, vp))

		beforePerm, err := f.K.Permission.Get(f.Ctx, applicantPermID)
		require.NoError(t, err)
		depositBefore := beforePerm.Deposit

		f.SetBalance(applicantCorp, 0)
		f.SetModuleBalance(0)

		fees := specRenewVPFees(0, f.TRKeeper.TrustUnitPrice)           // 0
		deposit := specRenewVPDeposit(fees, f.TDKeeper.TrustDepositRate) // 0

		msg := &types.MsgRenewPermissionVP{
			Id:          applicantPermID,
			Corporation: applicantCorp,
			Operator:    applicantCorp,
		}
		_, err = f.MS.RenewPermissionVP(f.Ctx, msg)
		require.NoError(t, err)

		// No bank transfer when fees == 0
		f.RequireNoBalanceChange(applicantCorp)
		f.RequireModuleBalanceDelta(0)

		// Permission mutations
		stored, err := f.K.Permission.Get(f.Ctx, applicantPermID)
		require.NoError(t, err)
		require.Equal(t, types.ValidationState_PENDING, stored.VpState)
		require.Equal(t, fees, stored.VpCurrentFees)
		require.Equal(t, deposit, stored.VpCurrentDeposit)
		require.Equal(t, specRenewVPDepositAfter(depositBefore, deposit), stored.Deposit)
		require.NotNil(t, stored.Modified)

		// Event
		f.RequireEvent(types.EventTypeRenewPermissionVP, map[string]string{
			types.AttributeKeyPermissionID:    strconv.FormatUint(applicantPermID, 10),
			types.AttributeKeyCorporation:     applicantCorp,
			types.AttributeKeyValidationFees:  "0",
			types.AttributeKeyValidationDeposit: "0",
		})

		// Invariant
		f.RequireInvariant()
	})

	// -----------------------------------------------------------------------
	// Happy path: renewal with non-zero fees and deposit
	// -----------------------------------------------------------------------
	t.Run("MOD-PERM-MSG-2: valid renewal with fees and deposit", func(t *testing.T) {
		f, _, applicantPermID, _, applicantCorp := setupBase(t)
		modAddr := authtypes.NewModuleAddress(types.ModuleName).String()

		// trustUnitPrice=1 (default), depositRate=0.25
		f.TDKeeper.TrustDepositRate = math.LegacyMustNewDecFromStr("0.25")

		beforePerm, err := f.K.Permission.Get(f.Ctx, applicantPermID)
		require.NoError(t, err)
		depositBefore := beforePerm.Deposit // 40

		fees := specRenewVPFees(8, f.TRKeeper.TrustUnitPrice)           // 8 * 1 = 8
		deposit := specRenewVPDeposit(fees, f.TDKeeper.TrustDepositRate) // 8 * 0.25 = 2

		f.SetBalance(applicantCorp, int64(fees)+100)
		f.SetModuleBalance(0)

		msg := &types.MsgRenewPermissionVP{
			Id:          applicantPermID,
			Corporation: applicantCorp,
			Operator:    applicantCorp,
		}
		_, err = f.MS.RenewPermissionVP(f.Ctx, msg)
		require.NoError(t, err)

		// Bank: applicant pays fees to module
		f.RequireBalanceDelta(applicantCorp, -int64(fees))
		f.RequireBalanceDelta(modAddr, int64(fees))

		// Permission state mutations
		stored, err := f.K.Permission.Get(f.Ctx, applicantPermID)
		require.NoError(t, err)
		require.Equal(t, types.ValidationState_PENDING, stored.VpState)
		require.Equal(t, fees, stored.VpCurrentFees)
		require.Equal(t, deposit, stored.VpCurrentDeposit)
		require.Equal(t, specRenewVPDepositAfter(depositBefore, deposit), stored.Deposit) // 40 + 2 = 42

		// EffectiveFrom must remain unchanged (renewal keeps the original value)
		require.Equal(t, beforePerm.EffectiveFrom, stored.EffectiveFrom)

		// Deposit call
		require.NotEmpty(t, f.TDKeeper.AdjustCalls)
		require.Equal(t, int64(deposit), f.TDKeeper.AdjustCalls[0].Amount)
		require.Equal(t, applicantCorp, f.TDKeeper.AdjustCalls[0].Corporation)

		// Event
		f.RequireEvent(types.EventTypeRenewPermissionVP, map[string]string{
			types.AttributeKeyPermissionID:    strconv.FormatUint(applicantPermID, 10),
			types.AttributeKeyCorporation:     applicantCorp,
			types.AttributeKeyValidationFees:  strconv.FormatUint(fees, 10),
			types.AttributeKeyValidationDeposit: strconv.FormatUint(deposit, 10),
		})

		// Invariant
		f.RequireInvariant()
	})

	// -----------------------------------------------------------------------
	// Negative: [MOD-PERM-MSG-2-2-1] operator authorization failure
	// -----------------------------------------------------------------------
	t.Run("MOD-PERM-MSG-2-2-1: fails if operator authorization fails", func(t *testing.T) {
		f, _, applicantPermID, _, applicantCorp := setupBase(t)
		f.DelKeeper.ErrToReturn = fmt.Errorf("mock: unauthorized operator")
		f.SetBalance(applicantCorp, 1000)

		_, err := f.MS.RenewPermissionVP(f.Ctx, &types.MsgRenewPermissionVP{
			Id: applicantPermID, Corporation: applicantCorp,
			Operator: "cosmos1badoperator00000000000000000000aaaaaa",
		})
		require.ErrorContains(t, err, "authorization check failed")
		f.RequireNoBalanceChange(applicantCorp)
	})

	// -----------------------------------------------------------------------
	// Negative: [MOD-PERM-MSG-2-2-2a] perm not found
	// -----------------------------------------------------------------------
	t.Run("MOD-PERM-MSG-2-2-2a: fails if perm id does not exist", func(t *testing.T) {
		f, _, _, _, applicantCorp := setupBase(t)
		f.SetBalance(applicantCorp, 1000)

		_, err := f.MS.RenewPermissionVP(f.Ctx, &types.MsgRenewPermissionVP{
			Id: 9999, Corporation: applicantCorp, Operator: applicantCorp,
		})
		require.ErrorContains(t, err, "perm not found")
		f.RequireNoBalanceChange(applicantCorp)
	})

	// -----------------------------------------------------------------------
	// Negative: [MOD-PERM-MSG-2-2-2b] corporation mismatch
	// -----------------------------------------------------------------------
	t.Run("MOD-PERM-MSG-2-2-2b: fails if corporation is not perm.Corporation", func(t *testing.T) {
		f, _, applicantPermID, _, _ := setupBase(t)

		wrongCorp := "cosmos1qqqqqqqqqqqqqqqqqqqqqqqqqqqqqqzz5n5fp"
		f.SetBalance(wrongCorp, 1000)

		_, err := f.MS.RenewPermissionVP(f.Ctx, &types.MsgRenewPermissionVP{
			Id: applicantPermID, Corporation: wrongCorp, Operator: wrongCorp,
		})
		require.ErrorContains(t, err, "authority is not the perm authority")
		f.RequireNoBalanceChange(wrongCorp)
	})

	// -----------------------------------------------------------------------
	// Negative: [MOD-PERM-MSG-2-2-2c] perm.VpState is not VALIDATED
	// -----------------------------------------------------------------------
	t.Run("MOD-PERM-MSG-2-2-2c: fails if perm.VpState is PENDING (not VALIDATED)", func(t *testing.T) {
		f, _, applicantPermID, _, applicantCorp := setupBase(t)

		// Force perm to PENDING state
		perm, err := f.K.Permission.Get(f.Ctx, applicantPermID)
		require.NoError(t, err)
		perm.VpState = types.ValidationState_PENDING
		require.NoError(t, f.K.UpdatePermission(f.Ctx, perm))

		f.SetBalance(applicantCorp, 1000)

		_, err = f.MS.RenewPermissionVP(f.Ctx, &types.MsgRenewPermissionVP{
			Id: applicantPermID, Corporation: applicantCorp, Operator: applicantCorp,
		})
		require.ErrorContains(t, err, "perm vp_state must be VALIDATED to renew")
		f.RequireNoBalanceChange(applicantCorp)
	})

	// -----------------------------------------------------------------------
	// Negative: [MOD-PERM-MSG-2-2-2d] applicant perm not active (expired)
	// -----------------------------------------------------------------------
	t.Run("MOD-PERM-MSG-2-2-2d: fails if applicant perm is expired", func(t *testing.T) {
		f := NewFixture(t)
		f.SetBlockTime(baseTime)

		corp := "cosmos1qqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqnrql8a"
		applicantCorp := "cosmos1qqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqpjdpjx"

		trID := f.TRKeeper.CreateMockTrustRegistry(corp, "did:example:tr1")
		f.CSKeeper.CreateMockCredentialSchemaFull(cstypes.CredentialSchema{
			Id: 1, TrId: trID,
			IssuerOnboardingMode: cstypes.IssuerOnboardingMode_ISSUER_ONBOARDING_MODE_ECOSYSTEM_VALIDATION_PROCESS,
		})

		now := f.Ctx.BlockTime()
		vp := types.Permission{
			SchemaId: 1, Type: types.PermissionType_ECOSYSTEM, Corporation: corp,
			Created: &now, Modified: &now, VpState: types.ValidationState_VALIDATED,
			EffectiveFrom: &past, EffectiveUntil: &validatorUntil, ValidationFees: 5,
		}
		validatorPermID, _ := f.K.CreatePermission(f.Ctx, vp)

		// Applicant perm is already expired (effective_until in the past)
		expiredUntil := baseTime.Add(-1 * time.Hour)
		applicantPermID := makeValidatedPerm(f, 1, applicantCorp, validatorPermID, 10, effectiveFrom, expiredUntil)

		f.SetBalance(applicantCorp, 1000)

		_, err := f.MS.RenewPermissionVP(f.Ctx, &types.MsgRenewPermissionVP{
			Id: applicantPermID, Corporation: applicantCorp, Operator: applicantCorp,
		})
		require.ErrorContains(t, err, "applicant perm is not active")
		f.RequireNoBalanceChange(applicantCorp)
	})

	// -----------------------------------------------------------------------
	// Negative: [MOD-PERM-MSG-2-2-2e] validator perm not found
	// -----------------------------------------------------------------------
	t.Run("MOD-PERM-MSG-2-2-2e: fails if validator perm does not exist", func(t *testing.T) {
		f := NewFixture(t)
		f.SetBlockTime(baseTime)

		corp := "cosmos1qqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqnrql8a"
		applicantCorp := "cosmos1qqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqpjdpjx"

		trID := f.TRKeeper.CreateMockTrustRegistry(corp, "did:example:tr1")
		f.CSKeeper.CreateMockCredentialSchemaFull(cstypes.CredentialSchema{
			Id: 1, TrId: trID,
			IssuerOnboardingMode: cstypes.IssuerOnboardingMode_ISSUER_ONBOARDING_MODE_ECOSYSTEM_VALIDATION_PROCESS,
		})

		now := f.Ctx.BlockTime()
		// Create applicant perm pointing to a non-existent validator perm id
		perm := types.Permission{
			SchemaId: 1, Type: types.PermissionType_ISSUER, Corporation: applicantCorp,
			ValidatorPermId: 9999, // does not exist
			Created: &now, Modified: &now, VpState: types.ValidationState_VALIDATED,
			EffectiveFrom: &effectiveFrom, EffectiveUntil: &effectiveUntil,
		}
		applicantPermID, _ := f.K.CreatePermission(f.Ctx, perm)
		f.SetBalance(applicantCorp, 1000)

		_, err := f.MS.RenewPermissionVP(f.Ctx, &types.MsgRenewPermissionVP{
			Id: applicantPermID, Corporation: applicantCorp, Operator: applicantCorp,
		})
		require.ErrorContains(t, err, "validator perm not found")
		f.RequireNoBalanceChange(applicantCorp)
	})

	// -----------------------------------------------------------------------
	// Negative: [MOD-PERM-MSG-2-2-2f] validator perm not active (revoked)
	// -----------------------------------------------------------------------
	t.Run("MOD-PERM-MSG-2-2-2f: fails if validator perm is revoked", func(t *testing.T) {
		f, validatorPermID, applicantPermID, _, applicantCorp := setupBase(t)

		// Revoke the validator perm
		vp, err := f.K.Permission.Get(f.Ctx, validatorPermID)
		require.NoError(t, err)
		revokedAt := f.Ctx.BlockTime().Add(-1 * time.Hour)
		vp.Revoked = &revokedAt
		require.NoError(t, f.K.UpdatePermission(f.Ctx, vp))

		f.SetBalance(applicantCorp, 1000)

		_, err = f.MS.RenewPermissionVP(f.Ctx, &types.MsgRenewPermissionVP{
			Id: applicantPermID, Corporation: applicantCorp, Operator: applicantCorp,
		})
		require.ErrorContains(t, err, "validator perm is not valid")
		f.RequireNoBalanceChange(applicantCorp)
	})

	// -----------------------------------------------------------------------
	// Negative: insufficient funds for fees (StatefulBankMock enforces deduction)
	// -----------------------------------------------------------------------
	t.Run("bank: fails if corporation has insufficient balance for renewal fees", func(t *testing.T) {
		f, _, applicantPermID, _, applicantCorp := setupBase(t)
		modAddr := authtypes.NewModuleAddress(types.ModuleName).String()

		// Configure fees to 100 (8 * 100/8 simplification: set unitPrice high)
		f.TRKeeper.TrustUnitPrice = 50 // fees = 8 * 50 = 400

		fees := specRenewVPFees(8, 50) // 400
		// Set balance lower than required fees
		f.SetBalance(applicantCorp, int64(fees)-1) // 399, insufficient
		f.SetModuleBalance(0)

		_, err := f.MS.RenewPermissionVP(f.Ctx, &types.MsgRenewPermissionVP{
			Id: applicantPermID, Corporation: applicantCorp, Operator: applicantCorp,
		})
		require.ErrorContains(t, err, "failed to execute perm VP renewal")
		f.RequireNoBalanceChange(applicantCorp)
		f.RequireModuleBalanceDelta(0)
	})
}
```

- [ ] **Step 1.2: Verify build.**

  Run: `go build ./x/perm/keeper/...`
  Expected: no output, exit 0.

- [ ] **Step 1.3: Run new tests.**

  Run: `go test ./x/perm/keeper/... -run TestMsgRenewPermissionVP -v -count=1`
  Expected: all subtests PASS.

---

## Task 2: Delete old `TestRenewPermissionVP` from `msg_server_test.go`

**File:** `x/perm/keeper/msg_server_test.go`

- [ ] **Step 2.1: Delete the old test function.**

Find and remove the entire `func TestRenewPermissionVP(t *testing.T)` block from `msg_server_test.go`. Do not touch any other test functions.

- [ ] **Step 2.2: Verify build and full test suite.**

  Run: `go test ./x/perm/keeper/... -count=1`
  Expected: no compilation errors, all remaining tests PASS.

- [ ] **Step 2.3: Commit.**

```bash
git add x/perm/keeper/renew_permission_vp_test.go
git add x/perm/keeper/msg_server_test.go
git commit -m "test(perm): add fixture-based RenewPermissionVP tests, delete legacy"
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
git push -u origin test/step-23-perm-renew-permission-vp
gh pr create --title "test(perm): fixture-based RenewPermissionVP tests (step 23)" --body "$(cat <<'EOF'
## Summary
- Creates x/perm/keeper/renew_permission_vp_test.go with fixture-based tests
- specRenewVPFees and specRenewVPDeposit computed independently of implementation
- StatefulBankMock enforces actual fee deduction from corporation balance
- Deposit delta mutation verified (depositAfter == depositBefore + newDeposit)
- AdjustTrustDeposit calls verified on MockTrustDepositKeeper
- All preconditions tested: authz, not found, corp mismatch, wrong VpState, expired perm, validator not found, revoked validator, insufficient funds

## Test plan
- [ ] go test ./x/perm/keeper/... -race -count=1 passes
- [ ] go test ./x/perm/keeper/ -cover reports ≥85%
- [ ] golangci-lint run ./x/perm/keeper/... clean
- [ ] go test ./... (full repo) passes — no regressions
EOF
)"
```

---

## "Done" Criteria — Step 23

- [ ] `x/perm/keeper/renew_permission_vp_test.go` exists with 2 happy paths, 7 negative cases.
- [ ] Happy path with fees > 0: `RequireBalanceDelta(applicant, -fees)` + `RequireModuleBalanceDelta(+fees)`.
- [ ] Happy path: `Deposit` field mutation verified: `depositAfter == depositBefore + newDeposit`.
- [ ] Happy path: `VpCurrentFees`, `VpCurrentDeposit`, `VpState=PENDING` verified on stored perm.
- [ ] Happy path: `AdjustCalls[0].Amount == int64(deposit)` verified on `MockTrustDepositKeeper`.
- [ ] Every happy path: `RequireInvariant()` passes.
- [ ] Every negative case: `ErrorContains` + `RequireNoBalanceChange`.
- [ ] Legacy `TestRenewPermissionVP` deleted from `msg_server_test.go`.
- [ ] `go test ./x/perm/keeper/... -race -count=1` passes.
