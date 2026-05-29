# Step 29: PERM SlashPermissionTrustDeposit — Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Replace legacy `TestMsgSlashPermissionTrustDeposit` tests with fixture-based tests that verify full struct equality (Slashed, SlashedDeposit, Deposit), the `BurnEcosystemSlashedTrustDeposit` mock call, event emission, the PERM module invariant, and every spec precondition.

**Architecture:** `SlashPermissionTrustDeposit` has no direct bank transfer — it calls `trustDeposit.BurnEcosystemSlashedTrustDeposit` (mocked). All tests call `f.RequireNoBalanceChange` on the corporation. `BurnHistory` from the `PermTrustDepositMock` (added in step 27) is used to assert the burn call. The two authority options (validator ancestor and TR controller) each have a dedicated happy-path subtest.

**Tech Stack:** Go 1.24, Cosmos SDK v0.50.x, testify/require, StatefulBankMock from testutil/keeper/bank.go

---

## Pre-flight

- [ ] **Confirm step 27 is merged** (so `PermTrustDepositMock.BurnHistory` exists).

  ```bash
  grep -n "BurnHistory" testutil/keeper/permission.go
  ```
  Expected: line found.

- [ ] **Confirm step 21 is merged.** `x/perm/keeper/fixture_test.go` must exist.

  ```bash
  ls x/perm/keeper/fixture_test.go
  ```
  Expected: file exists.

- [ ] **Create worktree.** Branch name: `test/step-29-perm-slash-trust-deposit`.

- [ ] **Sanity check.**

  ```bash
  go build ./... && go test ./x/perm/keeper/... -count=1
  ```
  Expected: all pass.

---

## File Structure

- **Create:** `x/perm/keeper/msg_slash_permission_trust_deposit_test.go` — fixture-based tests.
- **Delete (in same PR):** the `TestMsgSlashPermissionTrustDeposit` function block from `x/perm/keeper/msg_server_test.go`.

---

## Task 1: Write `x/perm/keeper/msg_slash_permission_trust_deposit_test.go`

**File:** `x/perm/keeper/msg_slash_permission_trust_deposit_test.go`

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

// specSlashDepositAfter returns perm.Deposit after a slash.
// Per [MOD-PERM-MSG-12-3]: deposit -= amount.
func specSlashDepositAfter(depositBefore, amount uint64) uint64 {
	return depositBefore - amount
}

// specSlashSlashedDepositAfter returns perm.SlashedDeposit after a slash.
// Per [MOD-PERM-MSG-12-3]: slashed_deposit += amount.
func specSlashSlashedDepositAfter(slashedBefore, amount uint64) uint64 {
	return slashedBefore + amount
}

// ============================================================================
// TestMsgSlashPermissionTrustDeposit
// ============================================================================

func TestMsgSlashPermissionTrustDeposit(t *testing.T) {

	// ------------------------------------------------------------------ //
	// Happy path 1: partial slash — validator ancestor authorized          //
	// ------------------------------------------------------------------ //
	t.Run("MOD-PERM-MSG-12-2-2 option1: validator ancestor authorized — partial slash", func(t *testing.T) {
		f := NewFixture(t)
		now := time.Date(2026, 3, 1, 0, 0, 0, 0, time.UTC)
		f.SetBlockTime(now)

		vpCorp := sdk.AccAddress([]byte("corp_vp_slash____1")).String()
		subCorp := sdk.AccAddress([]byte("corp_sub_slash___1")).String()
		f.SetBalance(vpCorp, types.BondDenom, 1000)
		f.SetBalance(subCorp, types.BondDenom, 1000)

		schemaID := uint64(1)
		f.CSKeeper.CreateMockCredentialSchema(schemaID,
			cstypes.IssuerOnboardingMode_ISSUER_ONBOARDING_MODE_OPEN,
			cstypes.VerifierOnboardingMode_VERIFIER_ONBOARDING_MODE_OPEN)

		vpEffFrom := now.Add(-48 * time.Hour)
		vpEffUntil := now.Add(180 * 24 * time.Hour)
		vpPerm := types.Permission{
			SchemaId:        schemaID,
			Type:            types.PermissionType_ISSUER,
			Corporation:     vpCorp,
			ValidatorPermId: 0,
			EffectiveFrom:   &vpEffFrom,
			EffectiveUntil:  &vpEffUntil,
		}
		vpID, err := f.K.CreatePermission(f.Ctx, vpPerm)
		require.NoError(t, err)

		subEffFrom := now.Add(-24 * time.Hour)
		subEffUntil := now.Add(30 * 24 * time.Hour)
		depositBefore := uint64(5000)
		subPerm := types.Permission{
			SchemaId:        schemaID,
			Type:            types.PermissionType_ISSUER,
			Corporation:     subCorp,
			ValidatorPermId: vpID,
			EffectiveFrom:   &subEffFrom,
			EffectiveUntil:  &subEffUntil,
			Deposit:         depositBefore,
		}
		subID, err := f.K.CreatePermission(f.Ctx, subPerm)
		require.NoError(t, err)

		slashAmount := uint64(2000)
		_, err = f.MS.SlashPermissionTrustDeposit(f.Ctx, &types.MsgSlashPermissionTrustDeposit{
			Id:          subID,
			Corporation: vpCorp,
			Operator:    vpCorp,
			Amount:      slashAmount,
			Reason:      "test slash reason",
		})
		require.NoError(t, err)

		// Full struct assertion
		got, err := f.K.GetPermissionByID(f.Ctx, subID)
		require.NoError(t, err)
		require.Equal(t, specSlashDepositAfter(depositBefore, slashAmount), got.Deposit)
		require.Equal(t, specSlashSlashedDepositAfter(0, slashAmount), got.SlashedDeposit)
		require.NotNil(t, got.Slashed)
		require.Equal(t, now, *got.Slashed)
		require.NotNil(t, got.Modified)
		require.Equal(t, now, *got.Modified)

		// BurnEcosystemSlashedTrustDeposit must have been called
		require.Len(t, f.TDKeeper.BurnHistory, 1)
		require.Equal(t, subCorp, f.TDKeeper.BurnHistory[0].Corporation)
		require.Equal(t, slashAmount, f.TDKeeper.BurnHistory[0].Amount)

		// No direct bank transfer
		f.RequireNoBalanceChange(vpCorp)
		f.RequireNoBalanceChange(subCorp)

		// Event assertion
		f.RequireEvent(types.EventTypeSlashPermissionTrustDeposit, map[string]string{
			types.AttributeKeyPermissionID: fmt.Sprintf("%d", subID),
			types.AttributeKeySlashedAmount: fmt.Sprintf("%d", slashAmount),
			types.AttributeKeyCorporation:  vpCorp,
		})

		// Module invariant
		f.RequireInvariant()
	})

	// ------------------------------------------------------------------ //
	// Happy path 2: full slash (amount == deposit)                        //
	// ------------------------------------------------------------------ //
	t.Run("MOD-PERM-MSG-12-3: full slash (amount == deposit)", func(t *testing.T) {
		f := NewFixture(t)
		now := time.Date(2026, 3, 1, 0, 0, 0, 0, time.UTC)
		f.SetBlockTime(now)

		vpCorp := sdk.AccAddress([]byte("corp_vp_fullslsh1")).String()
		subCorp := sdk.AccAddress([]byte("corp_sub_fullslsh1")).String()
		f.SetBalance(vpCorp, types.BondDenom, 1000)

		schemaID := uint64(1)
		f.CSKeeper.CreateMockCredentialSchema(schemaID,
			cstypes.IssuerOnboardingMode_ISSUER_ONBOARDING_MODE_OPEN,
			cstypes.VerifierOnboardingMode_VERIFIER_ONBOARDING_MODE_OPEN)

		vpEffFrom := now.Add(-48 * time.Hour)
		vpEffUntil := now.Add(180 * 24 * time.Hour)
		vpPerm := types.Permission{
			SchemaId:       schemaID,
			Type:           types.PermissionType_ISSUER,
			Corporation:    vpCorp,
			EffectiveFrom:  &vpEffFrom,
			EffectiveUntil: &vpEffUntil,
		}
		vpID, err := f.K.CreatePermission(f.Ctx, vpPerm)
		require.NoError(t, err)

		subEffFrom := now.Add(-24 * time.Hour)
		subEffUntil := now.Add(30 * 24 * time.Hour)
		depositBefore := uint64(3000)
		subPerm := types.Permission{
			SchemaId:        schemaID,
			Type:            types.PermissionType_ISSUER,
			Corporation:     subCorp,
			ValidatorPermId: vpID,
			EffectiveFrom:   &subEffFrom,
			EffectiveUntil:  &subEffUntil,
			Deposit:         depositBefore,
		}
		subID, err := f.K.CreatePermission(f.Ctx, subPerm)
		require.NoError(t, err)

		// Slash the full deposit
		_, err = f.MS.SlashPermissionTrustDeposit(f.Ctx, &types.MsgSlashPermissionTrustDeposit{
			Id: subID, Corporation: vpCorp, Operator: vpCorp,
			Amount: depositBefore, Reason: "full slash",
		})
		require.NoError(t, err)

		got, err := f.K.GetPermissionByID(f.Ctx, subID)
		require.NoError(t, err)
		require.Equal(t, uint64(0), got.Deposit)
		require.Equal(t, depositBefore, got.SlashedDeposit)
		require.NotNil(t, got.Slashed)

		require.Len(t, f.TDKeeper.BurnHistory, 1)
		require.Equal(t, depositBefore, f.TDKeeper.BurnHistory[0].Amount)

		f.RequireNoBalanceChange(vpCorp)
		f.RequireInvariant()
	})

	// ------------------------------------------------------------------ //
	// Happy path 3: TR controller authorized (Option #2)                  //
	// ------------------------------------------------------------------ //
	t.Run("MOD-PERM-MSG-12-2-2 option2: TR controller authorized", func(t *testing.T) {
		f := NewFixture(t)
		now := time.Date(2026, 3, 1, 0, 0, 0, 0, time.UTC)
		f.SetBlockTime(now)

		trCorp := sdk.AccAddress([]byte("corp_tr_slash____1")).String()
		subCorp := sdk.AccAddress([]byte("corp_sub_slashtr1")).String()
		f.SetBalance(trCorp, types.BondDenom, 1000)

		trID := f.TRKeeper.CreateMockTrustRegistry(trCorp, "did:example:tr-slash1")

		schemaID := uint64(1)
		f.CSKeeper.CreateMockCredentialSchemaFull(cstypes.CredentialSchema{
			Id:                     schemaID,
			TrId:                   trID,
			IssuerOnboardingMode:   cstypes.IssuerOnboardingMode_ISSUER_ONBOARDING_MODE_OPEN,
			VerifierOnboardingMode: cstypes.VerifierOnboardingMode_VERIFIER_ONBOARDING_MODE_OPEN,
		})

		effFrom := now.Add(-24 * time.Hour)
		effUntil := now.Add(30 * 24 * time.Hour)
		depositBefore := uint64(1000)
		perm := types.Permission{
			SchemaId:        schemaID,
			Type:            types.PermissionType_ISSUER,
			Corporation:     subCorp,
			ValidatorPermId: 0,
			EffectiveFrom:   &effFrom,
			EffectiveUntil:  &effUntil,
			Deposit:         depositBefore,
		}
		permID, err := f.K.CreatePermission(f.Ctx, perm)
		require.NoError(t, err)

		slashAmount := uint64(500)
		_, err = f.MS.SlashPermissionTrustDeposit(f.Ctx, &types.MsgSlashPermissionTrustDeposit{
			Id: permID, Corporation: trCorp, Operator: trCorp,
			Amount: slashAmount, Reason: "tr controller slash",
		})
		require.NoError(t, err)

		got, err := f.K.GetPermissionByID(f.Ctx, permID)
		require.NoError(t, err)
		require.Equal(t, specSlashDepositAfter(depositBefore, slashAmount), got.Deposit)
		require.Equal(t, specSlashSlashedDepositAfter(0, slashAmount), got.SlashedDeposit)
		require.NotNil(t, got.Slashed)

		f.RequireNoBalanceChange(trCorp)
		f.RequireInvariant()
	})

	// ------------------------------------------------------------------ //
	// Negative: [MOD-PERM-MSG-12-2-1] operator authorization failure      //
	// ------------------------------------------------------------------ //
	t.Run("MOD-PERM-MSG-12-2-1: operator authorization failure", func(t *testing.T) {
		f := NewFixture(t)
		now := time.Date(2026, 3, 1, 0, 0, 0, 0, time.UTC)
		f.SetBlockTime(now)

		corp := sdk.AccAddress([]byte("corp_authz_sls___1")).String()
		f.SetBalance(corp, types.BondDenom, 1000)
		f.DelKeeper.ErrToReturn = fmt.Errorf("mock: not authorized")

		schemaID := uint64(1)
		f.CSKeeper.CreateMockCredentialSchema(schemaID,
			cstypes.IssuerOnboardingMode_ISSUER_ONBOARDING_MODE_OPEN,
			cstypes.VerifierOnboardingMode_VERIFIER_ONBOARDING_MODE_OPEN)

		effFrom := now.Add(-24 * time.Hour)
		effUntil := now.Add(30 * 24 * time.Hour)
		perm := types.Permission{
			SchemaId:       schemaID,
			Type:           types.PermissionType_ISSUER,
			Corporation:    corp,
			EffectiveFrom:  &effFrom,
			EffectiveUntil: &effUntil,
			Deposit:        1000,
		}
		id, err := f.K.CreatePermission(f.Ctx, perm)
		require.NoError(t, err)

		_, err = f.MS.SlashPermissionTrustDeposit(f.Ctx, &types.MsgSlashPermissionTrustDeposit{
			Id: id, Corporation: corp, Operator: sdk.AccAddress([]byte("bad_op___________1")).String(),
			Amount: 100, Reason: "test",
		})
		require.ErrorContains(t, err, "authorization check failed")
		f.RequireNoBalanceChange(corp)
	})

	// ------------------------------------------------------------------ //
	// Negative: [MOD-PERM-MSG-12-2-1a] perm not found                   //
	// ------------------------------------------------------------------ //
	t.Run("MOD-PERM-MSG-12-2-1a: permission not found", func(t *testing.T) {
		f := NewFixture(t)
		corp := sdk.AccAddress([]byte("corp_notfound_sl1")).String()
		f.SetBalance(corp, types.BondDenom, 1000)

		_, err := f.MS.SlashPermissionTrustDeposit(f.Ctx, &types.MsgSlashPermissionTrustDeposit{
			Id: 99999, Corporation: corp, Operator: corp, Amount: 100, Reason: "test",
		})
		require.ErrorContains(t, err, "permission not found")
		f.RequireNoBalanceChange(corp)
	})

	// ------------------------------------------------------------------ //
	// Negative: [MOD-PERM-MSG-12-2-1b] amount > deposit                  //
	// ------------------------------------------------------------------ //
	t.Run("MOD-PERM-MSG-12-2-1b: amount exceeds available deposit", func(t *testing.T) {
		f := NewFixture(t)
		now := time.Date(2026, 3, 1, 0, 0, 0, 0, time.UTC)
		f.SetBlockTime(now)

		vpCorp := sdk.AccAddress([]byte("corp_vp_overslsh1")).String()
		subCorp := sdk.AccAddress([]byte("corp_sub_ovrslsh1")).String()
		f.SetBalance(vpCorp, types.BondDenom, 1000)

		schemaID := uint64(1)
		f.CSKeeper.CreateMockCredentialSchema(schemaID,
			cstypes.IssuerOnboardingMode_ISSUER_ONBOARDING_MODE_OPEN,
			cstypes.VerifierOnboardingMode_VERIFIER_ONBOARDING_MODE_OPEN)

		vpEffFrom := now.Add(-48 * time.Hour)
		vpEffUntil := now.Add(180 * 24 * time.Hour)
		vpPerm := types.Permission{
			SchemaId:       schemaID,
			Type:           types.PermissionType_ISSUER,
			Corporation:    vpCorp,
			EffectiveFrom:  &vpEffFrom,
			EffectiveUntil: &vpEffUntil,
		}
		vpID, err := f.K.CreatePermission(f.Ctx, vpPerm)
		require.NoError(t, err)

		subEffFrom := now.Add(-24 * time.Hour)
		subEffUntil := now.Add(30 * 24 * time.Hour)
		subPerm := types.Permission{
			SchemaId:        schemaID,
			Type:            types.PermissionType_ISSUER,
			Corporation:     subCorp,
			ValidatorPermId: vpID,
			EffectiveFrom:   &subEffFrom,
			EffectiveUntil:  &subEffUntil,
			Deposit:         500,
		}
		subID, err := f.K.CreatePermission(f.Ctx, subPerm)
		require.NoError(t, err)

		_, err = f.MS.SlashPermissionTrustDeposit(f.Ctx, &types.MsgSlashPermissionTrustDeposit{
			Id: subID, Corporation: vpCorp, Operator: vpCorp,
			Amount: 1000, // > deposit of 500
			Reason: "over-slash",
		})
		require.ErrorContains(t, err, "amount exceeds available deposit")
		f.RequireNoBalanceChange(vpCorp)
	})

	// ------------------------------------------------------------------ //
	// Negative: [MOD-PERM-MSG-12-2-2] no authority option matches        //
	// ------------------------------------------------------------------ //
	t.Run("MOD-PERM-MSG-12-2-2: authority not authorized to slash", func(t *testing.T) {
		f := NewFixture(t)
		now := time.Date(2026, 3, 1, 0, 0, 0, 0, time.UTC)
		f.SetBlockTime(now)

		corp := sdk.AccAddress([]byte("corp_slash_noauth1")).String()
		stranger := sdk.AccAddress([]byte("corp_slash_strgr1")).String()
		f.SetBalance(stranger, types.BondDenom, 1000)

		schemaID := uint64(1)
		f.CSKeeper.CreateMockCredentialSchema(schemaID,
			cstypes.IssuerOnboardingMode_ISSUER_ONBOARDING_MODE_OPEN,
			cstypes.VerifierOnboardingMode_VERIFIER_ONBOARDING_MODE_OPEN)

		effFrom := now.Add(-24 * time.Hour)
		effUntil := now.Add(30 * 24 * time.Hour)
		perm := types.Permission{
			SchemaId:        schemaID,
			Type:            types.PermissionType_ISSUER,
			Corporation:     corp,
			ValidatorPermId: 0,
			EffectiveFrom:   &effFrom,
			EffectiveUntil:  &effUntil,
			Deposit:         1000,
		}
		id, err := f.K.CreatePermission(f.Ctx, perm)
		require.NoError(t, err)

		_, err = f.MS.SlashPermissionTrustDeposit(f.Ctx, &types.MsgSlashPermissionTrustDeposit{
			Id: id, Corporation: stranger, Operator: stranger, Amount: 100, Reason: "stranger slash",
		})
		require.ErrorContains(t, err, "authority is not authorized to slash this permission")
		f.RequireNoBalanceChange(stranger)
	})

	// ------------------------------------------------------------------ //
	// Edge: ISSUER perm with VsOperator — RemovePermFromVSOA called       //
	// ------------------------------------------------------------------ //
	t.Run("edge: ISSUER perm with VsOperator — RemovePermFromVSOA called on slash", func(t *testing.T) {
		f := NewFixture(t)
		now := time.Date(2026, 3, 1, 0, 0, 0, 0, time.UTC)
		f.SetBlockTime(now)

		vpCorp := sdk.AccAddress([]byte("corp_vp_vsoa_sl_1")).String()
		subCorp := sdk.AccAddress([]byte("corp_sub_vsoa_sl1")).String()
		vsOp := sdk.AccAddress([]byte("vsop_slash_______1")).String()
		f.SetBalance(vpCorp, types.BondDenom, 1000)

		schemaID := uint64(1)
		f.CSKeeper.CreateMockCredentialSchema(schemaID,
			cstypes.IssuerOnboardingMode_ISSUER_ONBOARDING_MODE_OPEN,
			cstypes.VerifierOnboardingMode_VERIFIER_ONBOARDING_MODE_OPEN)

		vpEffFrom := now.Add(-48 * time.Hour)
		vpEffUntil := now.Add(180 * 24 * time.Hour)
		vpPerm := types.Permission{
			SchemaId:       schemaID,
			Type:           types.PermissionType_ISSUER,
			Corporation:    vpCorp,
			EffectiveFrom:  &vpEffFrom,
			EffectiveUntil: &vpEffUntil,
		}
		vpID, err := f.K.CreatePermission(f.Ctx, vpPerm)
		require.NoError(t, err)

		subEffFrom := now.Add(-24 * time.Hour)
		subEffUntil := now.Add(30 * 24 * time.Hour)
		subPerm := types.Permission{
			SchemaId:        schemaID,
			Type:            types.PermissionType_ISSUER,
			Corporation:     subCorp,
			ValidatorPermId: vpID,
			EffectiveFrom:   &subEffFrom,
			EffectiveUntil:  &subEffUntil,
			Deposit:         2000,
			VsOperator:      vsOp,
		}
		subID, err := f.K.CreatePermission(f.Ctx, subPerm)
		require.NoError(t, err)

		_, err = f.MS.SlashPermissionTrustDeposit(f.Ctx, &types.MsgSlashPermissionTrustDeposit{
			Id: subID, Corporation: vpCorp, Operator: vpCorp, Amount: 500, Reason: "vsoa edge",
		})
		require.NoError(t, err)

		require.Len(t, f.DelKeeper.RemovePermFromVSOACalls, 1)
		require.Equal(t, subCorp, f.DelKeeper.RemovePermFromVSOACalls[0].Authority)
		require.Equal(t, vsOp, f.DelKeeper.RemovePermFromVSOACalls[0].VsOperator)

		f.RequireNoBalanceChange(vpCorp)
		f.RequireInvariant()
	})

	// ------------------------------------------------------------------ //
	// Edge: cumulative slash — second slash accumulates SlashedDeposit    //
	// ------------------------------------------------------------------ //
	t.Run("edge: cumulative slashes accumulate SlashedDeposit", func(t *testing.T) {
		f := NewFixture(t)
		now := time.Date(2026, 3, 1, 0, 0, 0, 0, time.UTC)
		f.SetBlockTime(now)

		vpCorp := sdk.AccAddress([]byte("corp_vp_cumslsh_1")).String()
		subCorp := sdk.AccAddress([]byte("corp_sub_cumslsh1")).String()
		f.SetBalance(vpCorp, types.BondDenom, 1000)

		schemaID := uint64(1)
		f.CSKeeper.CreateMockCredentialSchema(schemaID,
			cstypes.IssuerOnboardingMode_ISSUER_ONBOARDING_MODE_OPEN,
			cstypes.VerifierOnboardingMode_VERIFIER_ONBOARDING_MODE_OPEN)

		vpEffFrom := now.Add(-48 * time.Hour)
		vpEffUntil := now.Add(180 * 24 * time.Hour)
		vpPerm := types.Permission{
			SchemaId:       schemaID,
			Type:           types.PermissionType_ISSUER,
			Corporation:    vpCorp,
			EffectiveFrom:  &vpEffFrom,
			EffectiveUntil: &vpEffUntil,
		}
		vpID, err := f.K.CreatePermission(f.Ctx, vpPerm)
		require.NoError(t, err)

		subEffFrom := now.Add(-24 * time.Hour)
		subEffUntil := now.Add(30 * 24 * time.Hour)
		depositBefore := uint64(6000)
		subPerm := types.Permission{
			SchemaId:        schemaID,
			Type:            types.PermissionType_ISSUER,
			Corporation:     subCorp,
			ValidatorPermId: vpID,
			EffectiveFrom:   &subEffFrom,
			EffectiveUntil:  &subEffUntil,
			Deposit:         depositBefore,
		}
		subID, err := f.K.CreatePermission(f.Ctx, subPerm)
		require.NoError(t, err)

		// First slash: 1000
		slash1 := uint64(1000)
		_, err = f.MS.SlashPermissionTrustDeposit(f.Ctx, &types.MsgSlashPermissionTrustDeposit{
			Id: subID, Corporation: vpCorp, Operator: vpCorp, Amount: slash1, Reason: "first",
		})
		require.NoError(t, err)

		// Second slash: 500
		slash2 := uint64(500)
		_, err = f.MS.SlashPermissionTrustDeposit(f.Ctx, &types.MsgSlashPermissionTrustDeposit{
			Id: subID, Corporation: vpCorp, Operator: vpCorp, Amount: slash2, Reason: "second",
		})
		require.NoError(t, err)

		got, err := f.K.GetPermissionByID(f.Ctx, subID)
		require.NoError(t, err)
		require.Equal(t, specSlashDepositAfter(specSlashDepositAfter(depositBefore, slash1), slash2), got.Deposit)
		require.Equal(t, specSlashSlashedDepositAfter(specSlashSlashedDepositAfter(0, slash1), slash2), got.SlashedDeposit)
		require.Equal(t, 2, len(f.TDKeeper.BurnHistory))

		f.RequireInvariant()
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
  go test ./x/perm/keeper/... -run TestMsgSlashPermissionTrustDeposit -v -count=1
  ```
  Expected: all subtests PASS.

- [ ] **Step 1.4: Commit.**

  ```bash
  git add x/perm/keeper/msg_slash_permission_trust_deposit_test.go
  git commit -m "test(perm): add fixture-based SlashPermissionTrustDeposit tests"
  ```

---

## Task 2: Delete old tests for `SlashPermissionTrustDeposit`

**File:** `x/perm/keeper/msg_server_test.go`

- [ ] **Step 2.1: Delete the `TestMsgSlashPermissionTrustDeposit` function block.**

  Remove the entire `func TestMsgSlashPermissionTrustDeposit(t *testing.T)` block. Keep all other test functions intact.

- [ ] **Step 2.2: Verify build and test suite.**

  ```bash
  go test ./x/perm/keeper/... -count=1
  ```
  Expected: all pass.

- [ ] **Step 2.3: Commit.**

  ```bash
  git add x/perm/keeper/msg_server_test.go
  git commit -m "test(perm): delete legacy TestMsgSlashPermissionTrustDeposit"
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
  git push -u origin test/step-29-perm-slash-trust-deposit
  gh pr create --title "test(perm): fixture-based SlashPermissionTrustDeposit tests (step 29)" --body "$(cat <<'EOF'
  ## Summary
  - Adds fixture-based TestMsgSlashPermissionTrustDeposit in msg_slash_permission_trust_deposit_test.go
  - Covers both authority options: validator ancestor and TR controller
  - Partial slash, full slash, and cumulative slash (two consecutive slashes) tested
  - BurnHistory assertions verify BurnEcosystemSlashedTrustDeposit call parameters
  - ISSUER VsOperator edge case: RemovePermFromVSOACalls asserted
  - All tests call RequireNoBalanceChange (no direct bank transfer)
  - Spec formula functions specSlashDepositAfter and specSlashSlashedDepositAfter

  ## Test plan
  - [ ] `go test ./x/perm/keeper/... -race -count=1` passes
  - [ ] `go test ./x/perm/keeper/ -cover` reports ≥95%
  - [ ] `golangci-lint run ./x/perm/keeper/...` clean
  - [ ] `go test ./...` passes — no regressions
  EOF
  )"
  ```

---

## "Done" Criteria — Step 29

- [ ] `x/perm/keeper/msg_slash_permission_trust_deposit_test.go` exists.
- [ ] Spec formulas `specSlashDepositAfter` and `specSlashSlashedDepositAfter` defined.
- [ ] Three happy paths: validator ancestor, TR controller, cumulative.
- [ ] Every happy path: full struct assertion (Slashed, SlashedDeposit, Deposit, Modified) + BurnHistory + event + invariant + `RequireNoBalanceChange`.
- [ ] Five negative subtests covering every precondition.
- [ ] ISSUER VsOperator edge case tested.
- [ ] Legacy `TestMsgSlashPermissionTrustDeposit` deleted from `msg_server_test.go`.
- [ ] `go test ./x/perm/keeper/... -race -count=1` passes.
- [ ] Coverage ≥95% on `x/perm/keeper/`.
