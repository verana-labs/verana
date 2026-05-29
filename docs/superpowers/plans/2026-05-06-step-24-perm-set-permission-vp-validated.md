# Step 24: PERM SetPermissionVPToValidated — Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Replace the legacy `TestSetPermissionVPToValidated` test with a fixture-based test that verifies the bank transfer from module escrow to validator account, full `Permission` struct mutations, event emission, and the PERM module invariant for every spec precondition in `[MOD-PERM-MSG-3]`.

**Architecture:** The `Fixture` struct already exists from step 21 (`x/perm/keeper/fixture_test.go`). This step only creates `x/perm/keeper/set_permission_vp_validated_test.go`. Read `x/perm/keeper/perm_validated.go` for the exact execution path before writing — specifically `executeSetPermissionVPToValidated`. The key bank transfer is `SendCoinsFromModuleToAccount(perm_module, validatorCorp, vp_current_fees)`, which means the **module account is debited** and the **validator corporation account is credited**. The applicant corporation's balance is NOT touched in this message. Pre-conditions are extensive; each gets its own `t.Run`.

**Tech Stack:** Go 1.24, Cosmos SDK v0.50.x, testify/require, StatefulBankMock from testutil/keeper/bank.go

---

## Pre-flight

- [ ] **Worktree.** Create an isolated worktree for this step. Branch name: `test/step-24-perm-set-permission-vp-validated`.
- [ ] **Gate check.** Confirm step 21 is merged: `x/perm/keeper/fixture_test.go` must exist and `go build ./x/perm/keeper/...` must pass.
- [ ] **Read the implementation.** Before writing tests, read:
  - `x/perm/keeper/perm_validated.go` — `executeSetPermissionVPToValidated`, `checkValidatedOverlap`, `getValidityPeriod`.
  - `x/perm/keeper/msg_server.go` lines 219–421 — all precondition checks.
- [ ] **Sanity check.**

  Run: `go build ./... && go vet ./...`
  Expected: no output, exit 0.

---

## File Structure

- **Create:** `x/perm/keeper/set_permission_vp_validated_test.go` — new fixture-based tests.
- **Fixture already exists:** `x/perm/keeper/fixture_test.go` (from step 21) — do not recreate.
- **Delete (in same PR):** the `TestSetPermissionVPToValidated` test function from `x/perm/keeper/msg_server_test.go`.

---

## Task 1: Write `set_permission_vp_validated_test.go`

**File:** `x/perm/keeper/set_permission_vp_validated_test.go`

Key execution path of `SetPermissionVPToValidated`:

1. Operator authorization check (`msg.Corporation` / `msg.Operator`).
2. Load `applicantPerm` by `msg.Id` — must exist.
3. `applicantPerm.VpState` must be `PENDING`.
4. If `applicantPerm.EffectiveFrom != nil` (renewal): `validation_fees`, `issuance_fees`, `verification_fees` must equal existing values.
5. `vp_summary_digest` must be null for HOLDER type.
6. Load CS — must exist.
7. Validate `issuance_fee_discount` and `verification_fee_discount` ranges.
8. Calculate `vp_exp` from validity period in CS.
9. Validate and resolve `effective_until` (> now, <= vp_exp; if renewal must be > existing effective_until).
10. Load `validatorPerm` — must exist and be active.
11. `validatorPerm.Corporation` must equal `msg.Corporation`.
12. Overlap check.
13. Execute: transfer `vp_current_fees` from module to `validatorPerm.Corporation`; increase `validatorPerm` trust deposit by `vp_current_deposit`; set fields; emit event.

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

// specValidatedFeesTransferred returns the amount transferred from module
// escrow to the validator corporation per [MOD-PERM-MSG-3-3]:
// amount = applicantPerm.VpCurrentFees.
func specValidatedFeesTransferred(vpCurrentFees uint64) uint64 {
	return vpCurrentFees
}

// specValidatedDepositTransferred returns the deposit increment applied to the
// validator's trust deposit per [MOD-PERM-MSG-3-3]:
// amount = applicantPerm.VpCurrentDeposit.
func specValidatedDepositTransferred(vpCurrentDeposit uint64) uint64 {
	return vpCurrentDeposit
}

// specVpExpFirstValidation calculates the vp_exp for a first-time validation:
// vp_exp = now + validityPeriodDays.
func specVpExpFirstValidation(now time.Time, validityPeriodDays int) time.Time {
	return now.AddDate(0, 0, validityPeriodDays)
}

// ============================================================================
// Test helpers
// ============================================================================

// makePendingPerm creates a PENDING permission ready for SetPermissionVPToValidated.
// validatorPermID is the perm that will validate it.
// vpCurrentFees and vpCurrentDeposit represent already-escrowed amounts.
func makePendingPerm(
	f *Fixture,
	schemaID uint64,
	applicantCorp string,
	validatorPermID uint64,
	permType types.PermissionType,
	vpCurrentFees uint64,
	vpCurrentDeposit uint64,
) uint64 {
	f.t.Helper()
	now := f.Ctx.BlockTime()
	perm := types.Permission{
		SchemaId:         schemaID,
		Type:             permType,
		Corporation:      applicantCorp,
		ValidatorPermId:  validatorPermID,
		Created:          &now,
		Modified:         &now,
		VpState:          types.ValidationState_PENDING,
		VpLastStateChange: &now,
		VpCurrentFees:    vpCurrentFees,
		VpCurrentDeposit: vpCurrentDeposit,
		Deposit:          vpCurrentDeposit, // set during StartVP
	}
	id, err := f.K.CreatePermission(f.Ctx, perm)
	require.NoError(f.t, err)
	return id
}

// makeRenewalPendingPerm creates a PENDING permission that represents a renewal:
// EffectiveFrom is already set (from a prior validation).
func makeRenewalPendingPerm(
	f *Fixture,
	schemaID uint64,
	applicantCorp string,
	validatorPermID uint64,
	effectiveFrom time.Time,
	vpExp time.Time,
	vpCurrentFees uint64,
	vpCurrentDeposit uint64,
	existingValidationFees uint64,
	existingIssuanceFees uint64,
	existingVerificationFees uint64,
) uint64 {
	f.t.Helper()
	now := f.Ctx.BlockTime()
	perm := types.Permission{
		SchemaId:         schemaID,
		Type:             types.PermissionType_ISSUER,
		Corporation:      applicantCorp,
		ValidatorPermId:  validatorPermID,
		Created:          &now,
		Modified:         &now,
		VpState:          types.ValidationState_PENDING,
		VpLastStateChange: &now,
		VpCurrentFees:    vpCurrentFees,
		VpCurrentDeposit: vpCurrentDeposit,
		Deposit:          vpCurrentDeposit,
		EffectiveFrom:    &effectiveFrom, // already set = renewal
		VpExp:            &vpExp,
		ValidationFees:   existingValidationFees,
		IssuanceFees:     existingIssuanceFees,
		VerificationFees: existingVerificationFees,
	}
	id, err := f.K.CreatePermission(f.Ctx, perm)
	require.NoError(f.t, err)
	return id
}

// ============================================================================
// TestMsgSetPermissionVPToValidated
// ============================================================================

func TestMsgSetPermissionVPToValidated(t *testing.T) {
	baseTime := time.Date(2026, 4, 1, 12, 0, 0, 0, time.UTC)
	past := baseTime.Add(-72 * time.Hour)
	validatorUntil := baseTime.Add(400 * 24 * time.Hour)
	futureEffUntil := baseTime.Add(90 * 24 * time.Hour)

	// setupWithPendingPerm creates a fixture with a validator perm + PENDING applicant perm.
	// validatorCorp owns the validator perm and is the msg.Corporation for SetPermissionVPToValidated.
	// applicantCorp owns the applicant perm.
	setupWithPendingPerm := func(t *testing.T, vpCurrentFees uint64, vpCurrentDeposit uint64) (
		*Fixture, uint64, uint64, string, string,
	) {
		t.Helper()
		f := NewFixture(t)
		f.SetBlockTime(baseTime)

		validatorCorp := "cosmos1qqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqnrql8a"
		applicantCorp := "cosmos1qqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqpjdpjx"

		trID := f.TRKeeper.CreateMockTrustRegistry(validatorCorp, "did:example:tr1")
		f.CSKeeper.CreateMockCredentialSchemaFull(cstypes.CredentialSchema{
			Id:   1,
			TrId: trID,
			IssuerOnboardingMode: cstypes.IssuerOnboardingMode_ISSUER_ONBOARDING_MODE_ECOSYSTEM_VALIDATION_PROCESS,
			IssuerValidationValidityPeriod: 180, // 180 days validity
		})

		now := f.Ctx.BlockTime()
		validatorPerm := types.Permission{
			SchemaId: 1, Type: types.PermissionType_ECOSYSTEM, Corporation: validatorCorp,
			Created: &now, Modified: &now, VpState: types.ValidationState_VALIDATED,
			EffectiveFrom: &past, EffectiveUntil: &validatorUntil,
		}
		validatorPermID, err := f.K.CreatePermission(f.Ctx, validatorPerm)
		require.NoError(t, err)

		applicantPermID := makePendingPerm(f, 1, applicantCorp, validatorPermID,
			types.PermissionType_ISSUER, vpCurrentFees, vpCurrentDeposit)

		return f, validatorPermID, applicantPermID, validatorCorp, applicantCorp
	}

	// -----------------------------------------------------------------------
	// Happy path: first-time validation with fees and deposit transfer
	// -----------------------------------------------------------------------
	t.Run("MOD-PERM-MSG-3: first-time validation transfers fees from module to validator", func(t *testing.T) {
		vpCurrentFees := uint64(30)
		vpCurrentDeposit := uint64(6)
		f, _, applicantPermID, validatorCorp, _ := setupWithPendingPerm(t, vpCurrentFees, vpCurrentDeposit)
		modAddr := authtypes.NewModuleAddress(types.ModuleName).String()

		// Pre-fund module account with escrowed fees (simulating StartVP having run)
		f.SetModuleBalance(int64(vpCurrentFees))
		f.SetBalance(validatorCorp, 0)

		feesTransferred := specValidatedFeesTransferred(vpCurrentFees) // 30
		depositTransferred := specValidatedDepositTransferred(vpCurrentDeposit) // 6

		vpExp := specVpExpFirstValidation(baseTime, 180)

		msg := &types.MsgSetPermissionVPToValidated{
			Id:               applicantPermID,
			Corporation:      validatorCorp,
			Operator:         validatorCorp,
			ValidationFees:   5,
			IssuanceFees:     2,
			VerificationFees: 1,
			EffectiveUntil:   &futureEffUntil,
		}

		_, err := f.MS.SetPermissionVPToValidated(f.Ctx, msg)
		require.NoError(t, err)

		// Bank: module loses fees, validator gains fees
		f.RequireBalanceDelta(modAddr, -int64(feesTransferred))
		f.RequireBalanceDelta(validatorCorp, int64(feesTransferred))

		// Permission mutations
		stored, err := f.K.Permission.Get(f.Ctx, applicantPermID)
		require.NoError(t, err)
		require.Equal(t, types.ValidationState_VALIDATED, stored.VpState)
		require.Equal(t, uint64(0), stored.VpCurrentFees)    // cleared
		require.Equal(t, uint64(0), stored.VpCurrentDeposit) // cleared
		require.NotNil(t, stored.EffectiveFrom)              // set to now (first validation)
		require.Equal(t, &futureEffUntil, stored.EffectiveUntil)
		require.NotNil(t, stored.VpExp)
		// vp_exp should equal now + 180 days (within 1 second tolerance)
		require.InDelta(t, vpExp.Unix(), stored.VpExp.Unix(), 1)
		require.Equal(t, uint64(5), stored.ValidationFees)
		require.Equal(t, uint64(2), stored.IssuanceFees)
		require.Equal(t, uint64(1), stored.VerificationFees)

		// Trust deposit: AdjustTrustDeposit called with deposit amount for validator
		require.NotEmpty(t, f.TDKeeper.AdjustCalls)
		require.Equal(t, validatorCorp, f.TDKeeper.AdjustCalls[0].Corporation)
		require.Equal(t, int64(depositTransferred), f.TDKeeper.AdjustCalls[0].Amount)
		require.Equal(t, "perm_validated_deposit", f.TDKeeper.AdjustCalls[0].Reason)

		// Event
		f.RequireEvent(types.EventTypeSetPermissionVPToValidated, map[string]string{
			types.AttributeKeyPermissionID:   strconv.FormatUint(applicantPermID, 10),
			types.AttributeKeyCorporation:    validatorCorp,
			types.AttributeKeyValidationFees: "5",
			types.AttributeKeyIssuanceFees:   "2",
			types.AttributeKeyVerificationFees: "1",
		})

		// Invariant: VpCurrentFees is 0 for validated perm, module balance == 0
		f.RequireInvariant()
	})

	// -----------------------------------------------------------------------
	// Happy path: zero fees — no bank transfer occurs
	// -----------------------------------------------------------------------
	t.Run("MOD-PERM-MSG-3: zero fees — no bank transfer", func(t *testing.T) {
		f, _, applicantPermID, validatorCorp, _ := setupWithPendingPerm(t, 0, 0)

		f.SetModuleBalance(0)
		f.SetBalance(validatorCorp, 0)

		msg := &types.MsgSetPermissionVPToValidated{
			Id: applicantPermID, Corporation: validatorCorp, Operator: validatorCorp,
			EffectiveUntil: &futureEffUntil,
		}
		_, err := f.MS.SetPermissionVPToValidated(f.Ctx, msg)
		require.NoError(t, err)

		f.RequireModuleBalanceDelta(0)
		f.RequireNoBalanceChange(validatorCorp)
		f.RequireInvariant()
	})

	// -----------------------------------------------------------------------
	// Happy path: renewal — effective_from already set, fees cannot change
	// -----------------------------------------------------------------------
	t.Run("MOD-PERM-MSG-3: renewal validation — fees unchanged, effective_until extended", func(t *testing.T) {
		f := NewFixture(t)
		f.SetBlockTime(baseTime)

		validatorCorp := "cosmos1qqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqnrql8a"
		applicantCorp := "cosmos1qqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqpjdpjx"

		trID := f.TRKeeper.CreateMockTrustRegistry(validatorCorp, "did:example:tr1")
		f.CSKeeper.CreateMockCredentialSchemaFull(cstypes.CredentialSchema{
			Id: 1, TrId: trID,
			IssuerOnboardingMode:           cstypes.IssuerOnboardingMode_ISSUER_ONBOARDING_MODE_ECOSYSTEM_VALIDATION_PROCESS,
			IssuerValidationValidityPeriod: 180,
		})

		now := f.Ctx.BlockTime()
		vpExp := now.Add(200 * 24 * time.Hour)
		vp := types.Permission{
			SchemaId: 1, Type: types.PermissionType_ECOSYSTEM, Corporation: validatorCorp,
			Created: &now, Modified: &now, VpState: types.ValidationState_VALIDATED,
			EffectiveFrom: &past, EffectiveUntil: &validatorUntil,
		}
		validatorPermID, _ := f.K.CreatePermission(f.Ctx, vp)

		existingEffUntil := baseTime.Add(60 * 24 * time.Hour)
		applicantPermID := makeRenewalPendingPerm(f, 1, applicantCorp, validatorPermID,
			baseTime.Add(-24*time.Hour), vpExp, 15, 3, 5, 2, 1)

		modAddr := authtypes.NewModuleAddress(types.ModuleName).String()
		f.SetModuleBalance(15)
		f.SetBalance(validatorCorp, 0)

		newEffUntil := existingEffUntil.Add(90 * 24 * time.Hour)

		msg := &types.MsgSetPermissionVPToValidated{
			Id: applicantPermID, Corporation: validatorCorp, Operator: validatorCorp,
			// Must match existing fee values for renewal
			ValidationFees:   5,
			IssuanceFees:     2,
			VerificationFees: 1,
			EffectiveUntil:   &newEffUntil,
		}

		_, err := f.MS.SetPermissionVPToValidated(f.Ctx, msg)
		require.NoError(t, err)

		f.RequireBalanceDelta(modAddr, -15)
		f.RequireBalanceDelta(validatorCorp, 15)

		stored, err := f.K.Permission.Get(f.Ctx, applicantPermID)
		require.NoError(t, err)
		require.Equal(t, types.ValidationState_VALIDATED, stored.VpState)
		require.Equal(t, uint64(0), stored.VpCurrentFees)
		require.NotNil(t, stored.EffectiveFrom) // unchanged from before
		require.Equal(t, &newEffUntil, stored.EffectiveUntil)

		f.RequireInvariant()
	})

	// -----------------------------------------------------------------------
	// Negative: [MOD-PERM-MSG-3-2-1] operator authorization failure
	// -----------------------------------------------------------------------
	t.Run("MOD-PERM-MSG-3-2-1: fails if operator authorization fails", func(t *testing.T) {
		f, _, applicantPermID, validatorCorp, _ := setupWithPendingPerm(t, 10, 2)
		f.SetModuleBalance(10)
		f.SetBalance(validatorCorp, 0)
		f.DelKeeper.ErrToReturn = fmt.Errorf("mock: unauthorized")

		_, err := f.MS.SetPermissionVPToValidated(f.Ctx, &types.MsgSetPermissionVPToValidated{
			Id: applicantPermID, Corporation: validatorCorp, Operator: "cosmos1badoperator00000000000000000000aaaaaa",
			EffectiveUntil: &futureEffUntil,
		})
		require.ErrorContains(t, err, "authorization check failed")
		f.RequireNoBalanceChange(validatorCorp)
		f.RequireModuleBalanceDelta(0)
	})

	// -----------------------------------------------------------------------
	// Negative: [MOD-PERM-MSG-3-2-1a] perm not found
	// -----------------------------------------------------------------------
	t.Run("MOD-PERM-MSG-3-2-1a: fails if perm id does not exist", func(t *testing.T) {
		f, _, _, validatorCorp, _ := setupWithPendingPerm(t, 0, 0)
		f.SetBalance(validatorCorp, 0)

		_, err := f.MS.SetPermissionVPToValidated(f.Ctx, &types.MsgSetPermissionVPToValidated{
			Id: 9999, Corporation: validatorCorp, Operator: validatorCorp,
			EffectiveUntil: &futureEffUntil,
		})
		require.ErrorContains(t, err, "perm not found")
	})

	// -----------------------------------------------------------------------
	// Negative: [MOD-PERM-MSG-3-2-1b] perm.VpState is not PENDING
	// -----------------------------------------------------------------------
	t.Run("MOD-PERM-MSG-3-2-1b: fails if perm.VpState is not PENDING", func(t *testing.T) {
		f, _, applicantPermID, validatorCorp, _ := setupWithPendingPerm(t, 0, 0)
		f.SetBalance(validatorCorp, 0)

		// Force perm to VALIDATED
		perm, err := f.K.Permission.Get(f.Ctx, applicantPermID)
		require.NoError(t, err)
		perm.VpState = types.ValidationState_VALIDATED
		require.NoError(t, f.K.UpdatePermission(f.Ctx, perm))

		_, err = f.MS.SetPermissionVPToValidated(f.Ctx, &types.MsgSetPermissionVPToValidated{
			Id: applicantPermID, Corporation: validatorCorp, Operator: validatorCorp,
			EffectiveUntil: &futureEffUntil,
		})
		require.ErrorContains(t, err, "perm must be in PENDING state to be validated")
		f.RequireNoBalanceChange(validatorCorp)
	})

	// -----------------------------------------------------------------------
	// Negative: [MOD-PERM-MSG-3-2-1c] renewal: validation_fees changed
	// -----------------------------------------------------------------------
	t.Run("MOD-PERM-MSG-3-2-1c: fails if validation_fees changed on renewal", func(t *testing.T) {
		f := NewFixture(t)
		f.SetBlockTime(baseTime)

		validatorCorp := "cosmos1qqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqnrql8a"
		applicantCorp := "cosmos1qqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqpjdpjx"

		trID := f.TRKeeper.CreateMockTrustRegistry(validatorCorp, "did:example:tr1")
		f.CSKeeper.CreateMockCredentialSchemaFull(cstypes.CredentialSchema{
			Id: 1, TrId: trID,
			IssuerOnboardingMode:           cstypes.IssuerOnboardingMode_ISSUER_ONBOARDING_MODE_ECOSYSTEM_VALIDATION_PROCESS,
			IssuerValidationValidityPeriod: 180,
		})

		now := f.Ctx.BlockTime()
		vpExpiry := now.Add(200 * 24 * time.Hour)
		vp := types.Permission{
			SchemaId: 1, Type: types.PermissionType_ECOSYSTEM, Corporation: validatorCorp,
			Created: &now, Modified: &now, VpState: types.ValidationState_VALIDATED,
			EffectiveFrom: &past, EffectiveUntil: &validatorUntil,
		}
		validatorPermID, _ := f.K.CreatePermission(f.Ctx, vp)

		existingEffFrom := baseTime.Add(-24 * time.Hour)
		// Renewal perm with existing ValidationFees=5
		applicantPermID := makeRenewalPendingPerm(f, 1, applicantCorp, validatorPermID,
			existingEffFrom, vpExpiry, 0, 0, 5, 2, 1)
		newEffUntil := baseTime.Add(120 * 24 * time.Hour)

		// Try to change validation_fees to 99 (different from existing 5)
		_, err := f.MS.SetPermissionVPToValidated(f.Ctx, &types.MsgSetPermissionVPToValidated{
			Id: applicantPermID, Corporation: validatorCorp, Operator: validatorCorp,
			ValidationFees: 99, IssuanceFees: 2, VerificationFees: 1,
			EffectiveUntil: &newEffUntil,
		})
		require.ErrorContains(t, err, "validation_fees cannot be changed during renewal")
	})

	// -----------------------------------------------------------------------
	// Negative: [MOD-PERM-MSG-3-2-1d] vp_summary_digest set for HOLDER type
	// -----------------------------------------------------------------------
	t.Run("MOD-PERM-MSG-3-2-1d: fails if vp_summary_digest is set for HOLDER type", func(t *testing.T) {
		f := NewFixture(t)
		f.SetBlockTime(baseTime)

		validatorCorp := "cosmos1qqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqnrql8a"
		applicantCorp := "cosmos1qqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqpjdpjx"

		trID := f.TRKeeper.CreateMockTrustRegistry(validatorCorp, "did:example:tr1")
		f.CSKeeper.CreateMockCredentialSchemaFull(cstypes.CredentialSchema{
			Id: 1, TrId: trID,
			// HOLDER needs ISSUER validator; use ECOSYSTEM_VALIDATION for verifier
			VerifierOnboardingMode: cstypes.VerifierOnboardingMode_VERIFIER_ONBOARDING_MODE_ECOSYSTEM_VALIDATION_PROCESS,
		})

		now := f.Ctx.BlockTime()
		// For HOLDER perm, need ISSUER validator
		issuerPerm := types.Permission{
			SchemaId: 1, Type: types.PermissionType_ISSUER, Corporation: validatorCorp,
			Created: &now, Modified: &now, VpState: types.ValidationState_VALIDATED,
			EffectiveFrom: &past, EffectiveUntil: &validatorUntil,
		}
		issuerPermID, _ := f.K.CreatePermission(f.Ctx, issuerPerm)

		holderPermID := makePendingPerm(f, 1, applicantCorp, issuerPermID,
			types.PermissionType_HOLDER, 0, 0)

		_, err := f.MS.SetPermissionVPToValidated(f.Ctx, &types.MsgSetPermissionVPToValidated{
			Id: holderPermID, Corporation: validatorCorp, Operator: validatorCorp,
			VpSummaryDigest: "sha256:abc123", // must be null for HOLDER
			EffectiveUntil:  &futureEffUntil,
		})
		require.ErrorContains(t, err, "vp_summary_digest must be null for HOLDER type")
	})

	// -----------------------------------------------------------------------
	// Negative: [MOD-PERM-MSG-3-2-1e] credential schema not found
	// -----------------------------------------------------------------------
	t.Run("MOD-PERM-MSG-3-2-1e: fails if credential schema does not exist", func(t *testing.T) {
		f := NewFixture(t)
		f.SetBlockTime(baseTime)

		validatorCorp := "cosmos1qqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqnrql8a"
		applicantCorp := "cosmos1qqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqpjdpjx"

		now := f.Ctx.BlockTime()
		vp := types.Permission{
			SchemaId: 99, // schema 99 does not exist in mock
			Type: types.PermissionType_ECOSYSTEM, Corporation: validatorCorp,
			Created: &now, Modified: &now, VpState: types.ValidationState_VALIDATED,
			EffectiveFrom: &past, EffectiveUntil: &validatorUntil,
		}
		validatorPermID, _ := f.K.CreatePermission(f.Ctx, vp)

		pendingID := makePendingPerm(f, 99, applicantCorp, validatorPermID,
			types.PermissionType_ISSUER, 0, 0)

		_, err := f.MS.SetPermissionVPToValidated(f.Ctx, &types.MsgSetPermissionVPToValidated{
			Id: pendingID, Corporation: validatorCorp, Operator: validatorCorp,
			EffectiveUntil: &futureEffUntil,
		})
		require.ErrorContains(t, err, "credential schema not found")
	})

	// -----------------------------------------------------------------------
	// Negative: [MOD-PERM-MSG-3-2-1f] issuance_fee_discount > 10000
	// -----------------------------------------------------------------------
	t.Run("MOD-PERM-MSG-3-2-1f: fails if issuance_fee_discount exceeds 10000", func(t *testing.T) {
		f, _, applicantPermID, validatorCorp, _ := setupWithPendingPerm(t, 0, 0)
		f.SetBalance(validatorCorp, 0)

		_, err := f.MS.SetPermissionVPToValidated(f.Ctx, &types.MsgSetPermissionVPToValidated{
			Id: applicantPermID, Corporation: validatorCorp, Operator: validatorCorp,
			IssuanceFeeDiscount: 10001, // exceeds max of 10000
			EffectiveUntil:      &futureEffUntil,
		})
		require.ErrorContains(t, err, "issuance_fee_discount cannot exceed")
	})

	// -----------------------------------------------------------------------
	// Negative: [MOD-PERM-MSG-3-2-1g] effective_until constraints: not after now
	// -----------------------------------------------------------------------
	t.Run("MOD-PERM-MSG-3-2-1g: fails if effective_until is not after current timestamp", func(t *testing.T) {
		f, _, applicantPermID, validatorCorp, _ := setupWithPendingPerm(t, 0, 0)
		f.SetBalance(validatorCorp, 0)

		pastEffUntil := baseTime.Add(-1 * time.Hour) // in the past

		_, err := f.MS.SetPermissionVPToValidated(f.Ctx, &types.MsgSetPermissionVPToValidated{
			Id: applicantPermID, Corporation: validatorCorp, Operator: validatorCorp,
			EffectiveUntil: &pastEffUntil,
		})
		require.ErrorContains(t, err, "effective_until must be greater than current timestamp")
	})

	// -----------------------------------------------------------------------
	// Negative: [MOD-PERM-MSG-3-2-2] validator perm not active (revoked)
	// -----------------------------------------------------------------------
	t.Run("MOD-PERM-MSG-3-2-2: fails if validator perm is revoked", func(t *testing.T) {
		f, validatorPermID, applicantPermID, validatorCorp, _ := setupWithPendingPerm(t, 0, 0)
		f.SetBalance(validatorCorp, 0)

		// Revoke the validator perm
		vp, err := f.K.Permission.Get(f.Ctx, validatorPermID)
		require.NoError(t, err)
		revokedAt := baseTime.Add(-1 * time.Hour)
		vp.Revoked = &revokedAt
		require.NoError(t, f.K.UpdatePermission(f.Ctx, vp))

		_, err = f.MS.SetPermissionVPToValidated(f.Ctx, &types.MsgSetPermissionVPToValidated{
			Id: applicantPermID, Corporation: validatorCorp, Operator: validatorCorp,
			EffectiveUntil: &futureEffUntil,
		})
		require.ErrorContains(t, err, "validator perm is not valid")
	})

	// -----------------------------------------------------------------------
	// Negative: [MOD-PERM-MSG-3-2-2] corporation must be validator perm authority
	// -----------------------------------------------------------------------
	t.Run("MOD-PERM-MSG-3-2-2: fails if corporation is not validator perm authority", func(t *testing.T) {
		f, _, applicantPermID, _, _ := setupWithPendingPerm(t, 0, 0)

		wrongCorp := "cosmos1qqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqzz5n5fp"
		f.SetBalance(wrongCorp, 0)

		_, err := f.MS.SetPermissionVPToValidated(f.Ctx, &types.MsgSetPermissionVPToValidated{
			Id: applicantPermID, Corporation: wrongCorp, Operator: wrongCorp,
			EffectiveUntil: &futureEffUntil,
		})
		require.ErrorContains(t, err, "authority must be validator perm authority")
	})

	// -----------------------------------------------------------------------
	// Negative: [MOD-PERM-MSG-3-2-4] overlap check: validated overlap
	// -----------------------------------------------------------------------
	t.Run("MOD-PERM-MSG-3-2-4: fails if validated overlap exists", func(t *testing.T) {
		f := NewFixture(t)
		f.SetBlockTime(baseTime)

		validatorCorp := "cosmos1qqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqnrql8a"
		applicantCorp := "cosmos1qqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqpjdpjx"

		trID := f.TRKeeper.CreateMockTrustRegistry(validatorCorp, "did:example:tr1")
		f.CSKeeper.CreateMockCredentialSchemaFull(cstypes.CredentialSchema{
			Id: 1, TrId: trID,
			IssuerOnboardingMode:           cstypes.IssuerOnboardingMode_ISSUER_ONBOARDING_MODE_ECOSYSTEM_VALIDATION_PROCESS,
			IssuerValidationValidityPeriod: 365,
		})

		now := f.Ctx.BlockTime()
		vp := types.Permission{
			SchemaId: 1, Type: types.PermissionType_ECOSYSTEM, Corporation: validatorCorp,
			Created: &now, Modified: &now, VpState: types.ValidationState_VALIDATED,
			EffectiveFrom: &past, EffectiveUntil: &validatorUntil,
		}
		validatorPermID, _ := f.K.CreatePermission(f.Ctx, vp)

		// Create an existing VALIDATED perm with non-expiring effective_until
		neverExpires := baseTime.Add(500 * 24 * time.Hour)
		existingPerm := types.Permission{
			SchemaId: 1, Type: types.PermissionType_ISSUER, Corporation: applicantCorp,
			ValidatorPermId: validatorPermID,
			Created: &now, Modified: &now, VpState: types.ValidationState_VALIDATED,
			EffectiveFrom: &past, EffectiveUntil: &neverExpires,
		}
		_, _ = f.K.CreatePermission(f.Ctx, existingPerm)

		// New PENDING perm for same context
		pendingID := makePendingPerm(f, 1, applicantCorp, validatorPermID,
			types.PermissionType_ISSUER, 0, 0)
		f.SetModuleBalance(0)
		f.SetBalance(validatorCorp, 0)

		// Attempt validation — should fail due to overlap
		overlappingUntil := neverExpires.Add(-30 * 24 * time.Hour) // before neverExpires but overlaps
		_, err := f.MS.SetPermissionVPToValidated(f.Ctx, &types.MsgSetPermissionVPToValidated{
			Id: pendingID, Corporation: validatorCorp, Operator: validatorCorp,
			EffectiveUntil: &overlappingUntil,
		})
		require.ErrorContains(t, err, "overlap check failed")
	})
}
```

- [ ] **Step 1.2: Verify build.**

  Run: `go build ./x/perm/keeper/...`
  Expected: no output, exit 0.

- [ ] **Step 1.3: Run new tests.**

  Run: `go test ./x/perm/keeper/... -run TestMsgSetPermissionVPToValidated -v -count=1`
  Expected: all subtests PASS.

---

## Task 2: Delete old `TestSetPermissionVPToValidated` from `msg_server_test.go`

**File:** `x/perm/keeper/msg_server_test.go`

- [ ] **Step 2.1: Delete the old test function.**

Find and remove the entire `func TestSetPermissionVPToValidated(t *testing.T)` block. Do not touch any other test functions.

- [ ] **Step 2.2: Verify build and full test suite.**

  Run: `go test ./x/perm/keeper/... -count=1`
  Expected: no compilation errors, all remaining tests PASS.

- [ ] **Step 2.3: Commit.**

```bash
git add x/perm/keeper/set_permission_vp_validated_test.go
git add x/perm/keeper/msg_server_test.go
git commit -m "test(perm): add fixture-based SetPermissionVPToValidated tests, delete legacy"
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
git push -u origin test/step-24-perm-set-permission-vp-validated
gh pr create --title "test(perm): fixture-based SetPermissionVPToValidated tests (step 24)" --body "$(cat <<'EOF'
## Summary
- Creates x/perm/keeper/set_permission_vp_validated_test.go with fixture-based tests
- Bank transfer verified: module loses VpCurrentFees, validator gains VpCurrentFees
- AdjustTrustDeposit call verified for VpCurrentDeposit to validator corp
- Full Permission mutations checked: VpState=VALIDATED, VpCurrentFees=0, VpCurrentDeposit=0, EffectiveFrom set, VpExp set
- Renewal path: fees immutable, effective_until extended
- All 9 preconditions tested with one t.Run each

## Test plan
- [ ] go test ./x/perm/keeper/... -race -count=1 passes
- [ ] go test ./x/perm/keeper/ -cover reports ≥85%
- [ ] golangci-lint run ./x/perm/keeper/... clean
- [ ] go test ./... (full repo) passes — no regressions
EOF
)"
```

---

## "Done" Criteria — Step 24

- [ ] `x/perm/keeper/set_permission_vp_validated_test.go` exists with 3 happy paths, 9 negative cases.
- [ ] Happy path: `RequireBalanceDelta(modAddr, -fees)` + `RequireBalanceDelta(validatorCorp, +fees)`.
- [ ] Happy path: `VpCurrentFees=0`, `VpCurrentDeposit=0`, `VpState=VALIDATED` on stored perm.
- [ ] Happy path: `EffectiveFrom` set to block time on first validation; unchanged on renewal.
- [ ] Happy path: `AdjustCalls[0]` verified for deposit to validator.
- [ ] Every happy path: `RequireInvariant()` passes.
- [ ] Every negative case: `ErrorContains` asserts specific spec-tagged error message.
- [ ] Legacy `TestSetPermissionVPToValidated` deleted from `msg_server_test.go`.
- [ ] `go test ./x/perm/keeper/... -race -count=1` passes.
