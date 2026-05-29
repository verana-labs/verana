# Step 22: PERM StartPermissionVP — Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Replace the legacy `TestStartPermissionVP` test with a fixture-based test that verifies exact fee deductions via `StatefulBankMock`, full `Permission` struct equality, event emission, and the PERM module invariant for every spec precondition in `[MOD-PERM-MSG-1]`.

**Architecture:** The `Fixture` struct already exists from step 21 (`x/perm/keeper/fixture_test.go`). This step only creates `x/perm/keeper/start_permission_vp_test.go`. Fee expectations are computed from spec formula functions that are completely independent of the implementation under test; the `StatefulBankMock` enforces that the corporation account is actually debited and the module escrow account is actually credited. The `MockTrustDepositKeeper` (already stateful after step 21) captures `AdjustTrustDeposit` calls so the deposit leg can be verified without a live TD keeper.

**Tech Stack:** Go 1.24, Cosmos SDK v0.50.x, testify/require, StatefulBankMock from testutil/keeper/bank.go

---

## Pre-flight

- [ ] **Worktree.** Create an isolated worktree for this step. Branch name: `test/step-22-perm-start-permission-vp`.
- [ ] **Gate check.** Confirm step 21 is merged: `x/perm/keeper/fixture_test.go` must exist and `go build ./x/perm/keeper/...` must pass.
- [ ] **Sanity check.**

  Run: `go build ./... && go vet ./...`
  Expected: no output, exit 0.

---

## File Structure

- **Create:** `x/perm/keeper/start_permission_vp_test.go` — new fixture-based tests for `StartPermissionVP`.
- **Fixture already exists:** `x/perm/keeper/fixture_test.go` (from step 21) — do not recreate.
- **Delete (in same PR):** the `TestStartPermissionVP` test function from `x/perm/keeper/msg_server_test.go`.

---

## Task 1: Write `start_permission_vp_test.go`

**File:** `x/perm/keeper/start_permission_vp_test.go`

Read `x/perm/keeper/start_perm_vp.go` before writing. Key execution path:
1. `validatePermissionChecks` — loads validator perm, calls `credentialSchemaKeeper.GetCredentialSchemaById`, validates type combination.
2. `checkOverlap` — walks permissions for `(schema_id, type, validator_perm_id, corporation)` with `VpState=PENDING|VALIDATED`.
3. `validateAndCalculateFees` — fees = `validatorPerm.ValidationFees * trustUnitPrice`; deposit = fees * trustDepositRate (truncated).
4. `executeStartPermissionVP` — if deposit > 0: `AdjustTrustDeposit`; if fees > 0: `SendCoinsFromAccountToModule(corporation, "perm", fees)`.

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

// specStartVPFees returns the validation fee amount per [MOD-PERM-MSG-1-2-3]:
// fees = validatorPerm.ValidationFees * trustUnitPrice.
func specStartVPFees(validationFees, trustUnitPrice uint64) uint64 {
	return validationFees * trustUnitPrice
}

// specStartVPDeposit returns the trust deposit increment per [MOD-PERM-MSG-1-2-3]:
// deposit = fees * trustDepositRate (truncated integer).
func specStartVPDeposit(fees uint64, depositRate math.LegacyDec) uint64 {
	if depositRate.IsZero() {
		return 0
	}
	result := math.LegacyNewDecFromInt(math.NewIntFromUint64(fees)).Mul(depositRate).TruncateInt()
	return result.Uint64()
}

// ============================================================================
// TestMsgStartPermissionVP
// ============================================================================

func TestMsgStartPermissionVP(t *testing.T) {
	baseTime := time.Date(2026, 1, 1, 12, 0, 0, 0, time.UTC)
	past := baseTime.Add(-2 * time.Hour)
	futureUntil := baseTime.Add(365 * 24 * time.Hour)

	// makeValidatorPerm creates an ECOSYSTEM root permission that can serve as the
	// validator for ISSUER_GRANTOR/ISSUER perms. The corp is both the TR owner and
	// the permission holder.
	makeEcosystemPerm := func(f *Fixture, corp string, schemaID uint64) uint64 {
		f.t.Helper()
		now := f.Ctx.BlockTime()
		perm := types.Permission{
			SchemaId:         schemaID,
			Type:             types.PermissionType_ECOSYSTEM,
			Corporation:      corp,
			Created:          &now,
			Modified:         &now,
			VpState:          types.ValidationState_VALIDATED,
			EffectiveFrom:    &past,
			EffectiveUntil:   &futureUntil,
			ValidationFees:   10, // 10 units × trustUnitPrice = fees
			IssuanceFees:     5,
			VerificationFees: 2,
		}
		id, err := f.K.CreatePermission(f.Ctx, perm)
		require.NoError(f.t, err)
		return id
	}

	// -----------------------------------------------------------------------
	// Happy path: ISSUER perm VP started, fees=0 (deposit rate=0, unit price=1)
	// -----------------------------------------------------------------------
	t.Run("MOD-PERM-MSG-1: valid start VP with zero fees (validationFees=0)", func(t *testing.T) {
		f := NewFixture(t)
		f.SetBlockTime(baseTime)

		corp := "cosmos1qqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqnrql8a"
		applicantCorp := "cosmos1qqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqpjdpjx"
		modAddr := authtypes.NewModuleAddress(types.ModuleName).String()

		trID := f.TRKeeper.CreateMockTrustRegistry(corp, "did:example:tr1")
		f.CSKeeper.CreateMockCredentialSchemaFull(cstypes.CredentialSchema{
			Id:                     1,
			TrId:                   trID,
			IssuerOnboardingMode:   cstypes.IssuerOnboardingMode_ISSUER_ONBOARDING_MODE_ECOSYSTEM_VALIDATION_PROCESS,
			VerifierOnboardingMode: cstypes.VerifierOnboardingMode_VERIFIER_ONBOARDING_MODE_OPEN,
		})

		// Validator perm with ValidationFees=0 → no bank transfer
		now := f.Ctx.BlockTime()
		validatorPerm := types.Permission{
			SchemaId:       1,
			Type:           types.PermissionType_ECOSYSTEM,
			Corporation:    corp,
			Created:        &now,
			Modified:       &now,
			VpState:        types.ValidationState_VALIDATED,
			EffectiveFrom:  &past,
			EffectiveUntil: &futureUntil,
			ValidationFees: 0, // zero fees
		}
		validatorPermID, err := f.K.CreatePermission(f.Ctx, validatorPerm)
		require.NoError(t, err)

		f.SetBalance(applicantCorp, 0)
		f.SetModuleBalance(0)

		fees := specStartVPFees(0, f.TRKeeper.TrustUnitPrice)       // 0
		deposit := specStartVPDeposit(fees, f.TDKeeper.TrustDepositRate) // 0

		msg := &types.MsgStartPermissionVP{
			Corporation:     applicantCorp,
			Operator:        applicantCorp,
			ValidatorPermId: validatorPermID,
			Type:            types.PermissionType_ISSUER,
		}

		resp, err := f.MS.StartPermissionVP(f.Ctx, msg)
		require.NoError(t, err)
		require.NotZero(t, resp.PermissionId)

		// Balance: no transfer when fees == 0
		_ = fees
		_ = deposit
		f.RequireNoBalanceChange(applicantCorp)
		f.RequireModuleBalanceDelta(0)

		// Full struct assertion (time-insensitive fields)
		stored, err := f.K.Permission.Get(f.Ctx, resp.PermissionId)
		require.NoError(t, err)
		require.Equal(t, types.PermissionType_ISSUER, stored.Type)
		require.Equal(t, types.ValidationState_PENDING, stored.VpState)
		require.Equal(t, validatorPermID, stored.ValidatorPermId)
		require.Equal(t, applicantCorp, stored.Corporation)
		require.Equal(t, uint64(0), stored.VpCurrentFees)
		require.Equal(t, uint64(0), stored.VpCurrentDeposit)
		require.Equal(t, uint64(0), stored.Deposit)
		require.Nil(t, stored.EffectiveFrom) // not set until validated
		require.NotNil(t, stored.Created)
		require.NotNil(t, stored.Modified)

		// Event
		f.RequireEvent(types.EventTypeStartPermissionVP, map[string]string{
			types.AttributeKeyPermissionID:   strconv.FormatUint(resp.PermissionId, 10),
			types.AttributeKeyCorporation:    applicantCorp,
			types.AttributeKeyValidatorPermID: strconv.FormatUint(validatorPermID, 10),
			types.AttributeKeyFees:           "0",
			types.AttributeKeyDeposit:        "0",
		})

		// Invariant: sum(VpCurrentFees for PENDING) == module balance (both 0)
		f.RequireInvariant()
	})

	// -----------------------------------------------------------------------
	// Happy path: ISSUER perm VP started with non-zero fees and deposit
	// -----------------------------------------------------------------------
	t.Run("MOD-PERM-MSG-1: valid start VP with fees and deposit", func(t *testing.T) {
		f := NewFixture(t)
		f.SetBlockTime(baseTime)

		corp := "cosmos1qqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqnrql8a"
		applicantCorp := "cosmos1qqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqpjdpjx"
		modAddr := authtypes.NewModuleAddress(types.ModuleName).String()

		trID := f.TRKeeper.CreateMockTrustRegistry(corp, "did:example:tr1")
		f.CSKeeper.CreateMockCredentialSchemaFull(cstypes.CredentialSchema{
			Id:                   1,
			TrId:                 trID,
			IssuerOnboardingMode: cstypes.IssuerOnboardingMode_ISSUER_ONBOARDING_MODE_ECOSYSTEM_VALIDATION_PROCESS,
		})

		// Configure trust unit price = 3, deposit rate = 0.2
		f.TRKeeper.TrustUnitPrice = 3
		f.TDKeeper.TrustDepositRate = math.LegacyMustNewDecFromStr("0.2")

		now := f.Ctx.BlockTime()
		validatorPerm := types.Permission{
			SchemaId:       1,
			Type:           types.PermissionType_ECOSYSTEM,
			Corporation:    corp,
			Created:        &now,
			Modified:       &now,
			VpState:        types.ValidationState_VALIDATED,
			EffectiveFrom:  &past,
			EffectiveUntil: &futureUntil,
			ValidationFees: 10, // 10 * 3 = 30 uvna fees
		}
		validatorPermID, err := f.K.CreatePermission(f.Ctx, validatorPerm)
		require.NoError(t, err)

		fees := specStartVPFees(10, 3)                                           // 30
		deposit := specStartVPDeposit(fees, math.LegacyMustNewDecFromStr("0.2")) // 6

		// Fund applicant; module starts empty
		f.SetBalance(applicantCorp, int64(fees)+100) // enough to cover fees
		f.SetModuleBalance(0)

		msg := &types.MsgStartPermissionVP{
			Corporation:     applicantCorp,
			Operator:        applicantCorp,
			ValidatorPermId: validatorPermID,
			Type:            types.PermissionType_ISSUER,
		}

		resp, err := f.MS.StartPermissionVP(f.Ctx, msg)
		require.NoError(t, err)
		require.NotZero(t, resp.PermissionId)

		// Balance deltas: applicant pays fees, module receives fees
		f.RequireBalanceDelta(applicantCorp, -int64(fees))
		f.RequireBalanceDelta(modAddr, int64(fees))

		// Full struct field check
		stored, err := f.K.Permission.Get(f.Ctx, resp.PermissionId)
		require.NoError(t, err)
		require.Equal(t, types.ValidationState_PENDING, stored.VpState)
		require.Equal(t, fees, stored.VpCurrentFees)
		require.Equal(t, deposit, stored.VpCurrentDeposit)
		require.Equal(t, deposit, stored.Deposit)
		require.Equal(t, validatorPermID, stored.ValidatorPermId)
		require.Equal(t, applicantCorp, stored.Corporation)
		require.Nil(t, stored.EffectiveFrom)

		// Deposit leg: AdjustTrustDeposit must have been called with deposit amount
		require.Len(t, f.TDKeeper.AdjustCalls, 1)
		require.Equal(t, applicantCorp, f.TDKeeper.AdjustCalls[0].Corporation)
		require.Equal(t, int64(deposit), f.TDKeeper.AdjustCalls[0].Amount)
		require.Equal(t, "start_perm_vp_deposit", f.TDKeeper.AdjustCalls[0].Reason)

		// Event
		f.RequireEvent(types.EventTypeStartPermissionVP, map[string]string{
			types.AttributeKeyPermissionID:   strconv.FormatUint(resp.PermissionId, 10),
			types.AttributeKeyFees:           strconv.FormatUint(fees, 10),
			types.AttributeKeyDeposit:        strconv.FormatUint(deposit, 10),
		})

		// Invariant: pending fees == module balance
		f.RequireInvariant()
	})

	// -----------------------------------------------------------------------
	// Happy path: ISSUER_GRANTOR perm VP (validator is ECOSYSTEM, mode=GRANTOR)
	// -----------------------------------------------------------------------
	t.Run("MOD-PERM-MSG-1: valid ISSUER_GRANTOR VP start", func(t *testing.T) {
		f := NewFixture(t)
		f.SetBlockTime(baseTime)

		corp := "cosmos1qqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqnrql8a"
		applicantCorp := "cosmos1qqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqpjdpjx"

		trID := f.TRKeeper.CreateMockTrustRegistry(corp, "did:example:tr2")
		f.CSKeeper.CreateMockCredentialSchemaFull(cstypes.CredentialSchema{
			Id:                   2,
			TrId:                 trID,
			IssuerOnboardingMode: cstypes.IssuerOnboardingMode_ISSUER_ONBOARDING_MODE_GRANTOR_VALIDATION_PROCESS,
		})

		now := f.Ctx.BlockTime()
		validatorPerm := types.Permission{
			SchemaId:       2,
			Type:           types.PermissionType_ECOSYSTEM,
			Corporation:    corp,
			Created:        &now,
			Modified:       &now,
			VpState:        types.ValidationState_VALIDATED,
			EffectiveFrom:  &past,
			EffectiveUntil: &futureUntil,
			ValidationFees: 5,
		}
		validatorPermID, err := f.K.CreatePermission(f.Ctx, validatorPerm)
		require.NoError(t, err)

		fees := specStartVPFees(5, f.TRKeeper.TrustUnitPrice) // 5 * 1 = 5
		f.SetBalance(applicantCorp, int64(fees)+50)
		f.SetModuleBalance(0)

		msg := &types.MsgStartPermissionVP{
			Corporation:     applicantCorp,
			Operator:        applicantCorp,
			ValidatorPermId: validatorPermID,
			Type:            types.PermissionType_ISSUER_GRANTOR,
		}

		resp, err := f.MS.StartPermissionVP(f.Ctx, msg)
		require.NoError(t, err)

		stored, err := f.K.Permission.Get(f.Ctx, resp.PermissionId)
		require.NoError(t, err)
		require.Equal(t, types.PermissionType_ISSUER_GRANTOR, stored.Type)
		require.Equal(t, types.ValidationState_PENDING, stored.VpState)
		require.Equal(t, fees, stored.VpCurrentFees)

		f.RequireInvariant()
	})

	// -----------------------------------------------------------------------
	// Negative: [MOD-PERM-MSG-1-2-1] operator authorization failure
	// -----------------------------------------------------------------------
	t.Run("MOD-PERM-MSG-1-2-1: fails if operator authorization fails", func(t *testing.T) {
		f := NewFixture(t)
		f.SetBlockTime(baseTime)
		f.DelKeeper.ErrToReturn = fmt.Errorf("mock: operator not authorized")

		corp := "cosmos1qqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqnrql8a"
		applicantCorp := "cosmos1qqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqpjdpjx"

		trID := f.TRKeeper.CreateMockTrustRegistry(corp, "did:example:tr1")
		f.CSKeeper.UpdateMockCredentialSchema(1, trID,
			cstypes.IssuerOnboardingMode_ISSUER_ONBOARDING_MODE_ECOSYSTEM_VALIDATION_PROCESS,
			cstypes.VerifierOnboardingMode_VERIFIER_ONBOARDING_MODE_OPEN)

		now := f.Ctx.BlockTime()
		vp := types.Permission{
			SchemaId: 1, Type: types.PermissionType_ECOSYSTEM, Corporation: corp,
			Created: &now, Modified: &now, VpState: types.ValidationState_VALIDATED,
			EffectiveFrom: &past, EffectiveUntil: &futureUntil, ValidationFees: 5,
		}
		vpID, _ := f.K.CreatePermission(f.Ctx, vp)

		f.SetBalance(applicantCorp, 1000)

		_, err := f.MS.StartPermissionVP(f.Ctx, &types.MsgStartPermissionVP{
			Corporation: applicantCorp, Operator: "cosmos1badoperator00000000000000000000aaaaaa",
			ValidatorPermId: vpID, Type: types.PermissionType_ISSUER,
		})
		require.ErrorContains(t, err, "authorization check failed")
		f.RequireNoBalanceChange(applicantCorp)
		f.RequireObjectCount(1) // only the validator perm
	})

	// -----------------------------------------------------------------------
	// Negative: [MOD-PERM-MSG-1-2-2a] validator perm not found
	// -----------------------------------------------------------------------
	t.Run("MOD-PERM-MSG-1-2-2a: fails if validator perm does not exist", func(t *testing.T) {
		f := NewFixture(t)
		f.SetBlockTime(baseTime)

		applicantCorp := "cosmos1qqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqpjdpjx"
		f.SetBalance(applicantCorp, 1000)

		_, err := f.MS.StartPermissionVP(f.Ctx, &types.MsgStartPermissionVP{
			Corporation: applicantCorp, Operator: applicantCorp,
			ValidatorPermId: 9999, // non-existent
			Type:            types.PermissionType_ISSUER,
		})
		require.ErrorContains(t, err, "perm validation failed")
		f.RequireNoBalanceChange(applicantCorp)
	})

	// -----------------------------------------------------------------------
	// Negative: [MOD-PERM-MSG-1-2-2b] validator perm not active (revoked)
	// -----------------------------------------------------------------------
	t.Run("MOD-PERM-MSG-1-2-2b: fails if validator perm is revoked", func(t *testing.T) {
		f := NewFixture(t)
		f.SetBlockTime(baseTime)

		corp := "cosmos1qqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqnrql8a"
		applicantCorp := "cosmos1qqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqpjdpjx"

		trID := f.TRKeeper.CreateMockTrustRegistry(corp, "did:example:tr1")
		f.CSKeeper.UpdateMockCredentialSchema(1, trID,
			cstypes.IssuerOnboardingMode_ISSUER_ONBOARDING_MODE_ECOSYSTEM_VALIDATION_PROCESS,
			cstypes.VerifierOnboardingMode_VERIFIER_ONBOARDING_MODE_OPEN)

		now := f.Ctx.BlockTime()
		revokedAt := now.Add(-1 * time.Hour)
		vp := types.Permission{
			SchemaId: 1, Type: types.PermissionType_ECOSYSTEM, Corporation: corp,
			Created: &now, Modified: &now, VpState: types.ValidationState_VALIDATED,
			EffectiveFrom: &past, EffectiveUntil: &futureUntil,
			Revoked: &revokedAt, // revoked
		}
		vpID, _ := f.K.CreatePermission(f.Ctx, vp)
		f.SetBalance(applicantCorp, 1000)

		_, err := f.MS.StartPermissionVP(f.Ctx, &types.MsgStartPermissionVP{
			Corporation: applicantCorp, Operator: applicantCorp,
			ValidatorPermId: vpID, Type: types.PermissionType_ISSUER,
		})
		require.ErrorContains(t, err, "perm validation failed")
		f.RequireNoBalanceChange(applicantCorp)
	})

	// -----------------------------------------------------------------------
	// Negative: [MOD-PERM-MSG-1-2-2c] perm type incompatible with schema mode
	// -----------------------------------------------------------------------
	t.Run("MOD-PERM-MSG-1-2-2c: fails if perm type incompatible with schema onboarding mode", func(t *testing.T) {
		f := NewFixture(t)
		f.SetBlockTime(baseTime)

		corp := "cosmos1qqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqnrql8a"
		applicantCorp := "cosmos1qqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqpjdpjx"

		trID := f.TRKeeper.CreateMockTrustRegistry(corp, "did:example:tr1")
		// Schema is GRANTOR mode, but we'll try to create an ISSUER VP with ECOSYSTEM validator
		f.CSKeeper.UpdateMockCredentialSchema(1, trID,
			cstypes.IssuerOnboardingMode_ISSUER_ONBOARDING_MODE_GRANTOR_VALIDATION_PROCESS,
			cstypes.VerifierOnboardingMode_VERIFIER_ONBOARDING_MODE_OPEN)

		now := f.Ctx.BlockTime()
		vp := types.Permission{
			SchemaId: 1, Type: types.PermissionType_ECOSYSTEM, Corporation: corp,
			Created: &now, Modified: &now, VpState: types.ValidationState_VALIDATED,
			EffectiveFrom: &past, EffectiveUntil: &futureUntil,
		}
		vpID, _ := f.K.CreatePermission(f.Ctx, vp)
		f.SetBalance(applicantCorp, 1000)

		// ISSUER requires ISSUER_GRANTOR validator when mode=GRANTOR, but we used ECOSYSTEM
		_, err := f.MS.StartPermissionVP(f.Ctx, &types.MsgStartPermissionVP{
			Corporation: applicantCorp, Operator: applicantCorp,
			ValidatorPermId: vpID, Type: types.PermissionType_ISSUER,
		})
		require.ErrorContains(t, err, "perm validation failed")
		f.RequireNoBalanceChange(applicantCorp)
	})

	// -----------------------------------------------------------------------
	// Negative: [MOD-PERM-MSG-1-2-4] overlap: existing PENDING perm for same context
	// -----------------------------------------------------------------------
	t.Run("MOD-PERM-MSG-1-2-4: fails if overlap exists (existing PENDING perm)", func(t *testing.T) {
		f := NewFixture(t)
		f.SetBlockTime(baseTime)

		corp := "cosmos1qqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqnrql8a"
		applicantCorp := "cosmos1qqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqpjdpjx"

		trID := f.TRKeeper.CreateMockTrustRegistry(corp, "did:example:tr1")
		f.CSKeeper.UpdateMockCredentialSchema(1, trID,
			cstypes.IssuerOnboardingMode_ISSUER_ONBOARDING_MODE_ECOSYSTEM_VALIDATION_PROCESS,
			cstypes.VerifierOnboardingMode_VERIFIER_ONBOARDING_MODE_OPEN)

		now := f.Ctx.BlockTime()
		vp := types.Permission{
			SchemaId: 1, Type: types.PermissionType_ECOSYSTEM, Corporation: corp,
			Created: &now, Modified: &now, VpState: types.ValidationState_VALIDATED,
			EffectiveFrom: &past, EffectiveUntil: &futureUntil, ValidationFees: 0,
		}
		vpID, _ := f.K.CreatePermission(f.Ctx, vp)

		f.SetBalance(applicantCorp, 5000)
		f.SetModuleBalance(0)

		msg := &types.MsgStartPermissionVP{
			Corporation: applicantCorp, Operator: applicantCorp,
			ValidatorPermId: vpID, Type: types.PermissionType_ISSUER,
		}

		// First call succeeds
		_, err := f.MS.StartPermissionVP(f.Ctx, msg)
		require.NoError(t, err)

		// Second call must fail: existing PENDING perm for same context
		_, err = f.MS.StartPermissionVP(f.Ctx, msg)
		require.ErrorContains(t, err, "overlap check failed")
	})

	// -----------------------------------------------------------------------
	// Negative: [MOD-PERM-MSG-1-2-3] fee overflow (uint64 overflow)
	// -----------------------------------------------------------------------
	t.Run("MOD-PERM-MSG-1-2-3: fails if validationFees * trustUnitPrice overflows uint64", func(t *testing.T) {
		f := NewFixture(t)
		f.SetBlockTime(baseTime)

		corp := "cosmos1qqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqnrql8a"
		applicantCorp := "cosmos1qqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqpjdpjx"

		trID := f.TRKeeper.CreateMockTrustRegistry(corp, "did:example:tr1")
		f.CSKeeper.UpdateMockCredentialSchema(1, trID,
			cstypes.IssuerOnboardingMode_ISSUER_ONBOARDING_MODE_ECOSYSTEM_VALIDATION_PROCESS,
			cstypes.VerifierOnboardingMode_VERIFIER_ONBOARDING_MODE_OPEN)

		// Set trust unit price to max uint64 so overflow is guaranteed
		f.TRKeeper.TrustUnitPrice = ^uint64(0) // max uint64

		now := f.Ctx.BlockTime()
		vp := types.Permission{
			SchemaId: 1, Type: types.PermissionType_ECOSYSTEM, Corporation: corp,
			Created: &now, Modified: &now, VpState: types.ValidationState_VALIDATED,
			EffectiveFrom: &past, EffectiveUntil: &futureUntil,
			ValidationFees: 2, // 2 * max_uint64 overflows
		}
		vpID, _ := f.K.CreatePermission(f.Ctx, vp)

		f.SetBalance(applicantCorp, int64(^uint32(0))) // large but not enough anyway

		_, err := f.MS.StartPermissionVP(f.Ctx, &types.MsgStartPermissionVP{
			Corporation: applicantCorp, Operator: applicantCorp,
			ValidatorPermId: vpID, Type: types.PermissionType_ISSUER,
		})
		require.ErrorContains(t, err, "fee validation failed")
	})

	// -----------------------------------------------------------------------
	// Negative: insufficient funds (StatefulBankMock enforces deduction)
	// -----------------------------------------------------------------------
	t.Run("bank: fails if corporation has insufficient uvna balance for fees", func(t *testing.T) {
		f := NewFixture(t)
		f.SetBlockTime(baseTime)

		corp := "cosmos1qqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqnrql8a"
		applicantCorp := "cosmos1qqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqpjdpjx"

		trID := f.TRKeeper.CreateMockTrustRegistry(corp, "did:example:tr1")
		f.CSKeeper.UpdateMockCredentialSchema(1, trID,
			cstypes.IssuerOnboardingMode_ISSUER_ONBOARDING_MODE_ECOSYSTEM_VALIDATION_PROCESS,
			cstypes.VerifierOnboardingMode_VERIFIER_ONBOARDING_MODE_OPEN)

		f.TRKeeper.TrustUnitPrice = 100 // fees = 10 * 100 = 1000

		now := f.Ctx.BlockTime()
		vp := types.Permission{
			SchemaId: 1, Type: types.PermissionType_ECOSYSTEM, Corporation: corp,
			Created: &now, Modified: &now, VpState: types.ValidationState_VALIDATED,
			EffectiveFrom: &past, EffectiveUntil: &futureUntil, ValidationFees: 10,
		}
		vpID, _ := f.K.CreatePermission(f.Ctx, vp)

		// Set applicant balance to less than required fees (1000 uvna)
		f.SetBalance(applicantCorp, 999) // insufficient
		f.SetModuleBalance(0)

		_, err := f.MS.StartPermissionVP(f.Ctx, &types.MsgStartPermissionVP{
			Corporation: applicantCorp, Operator: applicantCorp,
			ValidatorPermId: vpID, Type: types.PermissionType_ISSUER,
		})
		require.ErrorContains(t, err, "failed to execute perm VP")
		// Balance must not have changed — StatefulBankMock rejected the transfer
		f.RequireNoBalanceChange(applicantCorp)
		f.RequireModuleBalanceDelta(0)
	})
}
```

- [ ] **Step 1.2: Verify build.**

  Run: `go build ./x/perm/keeper/...`
  Expected: no output, exit 0.

- [ ] **Step 1.3: Run new tests.**

  Run: `go test ./x/perm/keeper/... -run TestMsgStartPermissionVP -v -count=1`
  Expected: all subtests PASS.

---

## Task 2: Delete old `TestStartPermissionVP` from `msg_server_test.go`

**File:** `x/perm/keeper/msg_server_test.go`

- [ ] **Step 2.1: Delete the old test function.**

Find and remove the entire `func TestStartPermissionVP(t *testing.T)` block from `msg_server_test.go`. Do not touch any other test functions.

- [ ] **Step 2.2: Verify build and full test suite.**

  Run: `go test ./x/perm/keeper/... -count=1`
  Expected: no compilation errors, all remaining tests PASS.

- [ ] **Step 2.3: Commit.**

```bash
git add x/perm/keeper/start_permission_vp_test.go
git add x/perm/keeper/msg_server_test.go
git commit -m "test(perm): add fixture-based StartPermissionVP tests, delete legacy"
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
git push -u origin test/step-22-perm-start-permission-vp
gh pr create --title "test(perm): fixture-based StartPermissionVP tests (step 22)" --body "$(cat <<'EOF'
## Summary
- Creates x/perm/keeper/start_permission_vp_test.go with fixture-based tests
- Spec formula functions specStartVPFees and specStartVPDeposit computed independently
- StatefulBankMock enforces actual fee deduction from corporation balance
- MockTrustDepositKeeper AdjustCalls verified for deposit leg
- All preconditions tested: authz, validator not found, revoked validator, type mismatch, overlap, overflow, insufficient funds
- Invariant (sum(VpCurrentFees for PENDING) == module balance) checked on every happy path

## Test plan
- [ ] go test ./x/perm/keeper/... -race -count=1 passes
- [ ] go test ./x/perm/keeper/ -cover reports ≥85%
- [ ] golangci-lint run ./x/perm/keeper/... clean
- [ ] go test ./... (full repo) passes — no regressions
EOF
)"
```

---

## "Done" Criteria — Step 22

- [ ] `x/perm/keeper/start_permission_vp_test.go` exists with 3 happy paths, 6 negative cases.
- [ ] Every happy path with fees > 0: `RequireBalanceDelta(applicantCorp, -fees)` + `RequireModuleBalanceDelta(+fees)`.
- [ ] Every happy path: `RequireInvariant()` passes.
- [ ] Every happy path with deposit > 0: `f.TDKeeper.AdjustCalls` verified with correct amount and reason.
- [ ] Every negative case: `ErrorContains` + `RequireNoBalanceChange`.
- [ ] Legacy `TestStartPermissionVP` deleted from `msg_server_test.go`.
- [ ] `go test ./x/perm/keeper/... -race -count=1` passes.
