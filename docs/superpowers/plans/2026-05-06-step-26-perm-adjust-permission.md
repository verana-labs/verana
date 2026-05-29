# Step 26: PERM AdjustPermission — Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Replace legacy `TestMsgAdjustPermission` tests with fixture-based tests that verify full struct equality, event emission, the PERM module invariant, and every spec precondition — including the three mutually exclusive authority paths.

**Architecture:** Step 21 created `x/perm/keeper/fixture_test.go` and wired `PermissionKeeperWithStatefulBank` in `testutil/keeper/permission.go`; this step only adds `x/perm/keeper/msg_adjust_permission_test.go` and deletes the legacy test block. `AdjustPermission` has no bank transfers, so all tests call `f.RequireNoBalanceChange` on the corporation address. The three authority paths (ECOSYSTEM perm, self-created via ECOSYSTEM validator, VP-managed perm) each have a dedicated happy-path subtest.

**Tech Stack:** Go 1.24, Cosmos SDK v0.50.x, testify/require, StatefulBankMock from testutil/keeper/bank.go

---

## Pre-flight

- [ ] **Confirm step 21 is merged.** `x/perm/keeper/fixture_test.go` must exist.

  ```bash
  ls x/perm/keeper/fixture_test.go
  ```
  Expected: file exists.

- [ ] **Create worktree.** Branch name: `test/step-26-perm-adjust-permission`.

- [ ] **Sanity check.**

  ```bash
  go build ./... && go test ./x/perm/keeper/... -count=1
  ```
  Expected: all pass.

---

## File Structure

- **Create:** `x/perm/keeper/msg_adjust_permission_test.go` — fixture-based tests.
- **Delete (in same PR):** the `TestMsgAdjustPermission` function block from `x/perm/keeper/msg_server_test.go`.

---

## Task 1: Write `x/perm/keeper/msg_adjust_permission_test.go`

**File:** `x/perm/keeper/msg_adjust_permission_test.go`

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

// specAdjustEffectiveUntil returns the effective_until set by AdjustPermission.
// The spec simply sets perm.effective_until = msg.effective_until.
func specAdjustEffectiveUntil(requested *time.Time) *time.Time {
	return requested
}

// ============================================================================
// TestMsgAdjustPermission
// ============================================================================

func TestMsgAdjustPermission(t *testing.T) {

	// ------------------------------------------------------------------ //
	// Happy path 1: ECOSYSTEM permission (ValidatorPermId == 0)            //
	// ------------------------------------------------------------------ //
	t.Run("MOD-PERM-MSG-8-2-2: ECOSYSTEM perm — corporation must match", func(t *testing.T) {
		f := NewFixture(t)
		now := time.Date(2026, 3, 1, 0, 0, 0, 0, time.UTC)
		f.SetBlockTime(now)

		corp := sdk.AccAddress([]byte("corp_eco_adjust__1")).String()
		f.SetBalance(corp, types.BondDenom, 1000)

		// Create ECOSYSTEM permission
		schemaID := uint64(1)
		f.CSKeeper.CreateMockCredentialSchema(schemaID,
			cstypes.IssuerOnboardingMode_ISSUER_ONBOARDING_MODE_OPEN,
			cstypes.VerifierOnboardingMode_VERIFIER_ONBOARDING_MODE_OPEN)

		effFrom := now.Add(-24 * time.Hour)
		effUntil := now.Add(30 * 24 * time.Hour)
		perm := types.Permission{
			SchemaId:        schemaID,
			Type:            types.PermissionType_ECOSYSTEM,
			Corporation:     corp,
			ValidatorPermId: 0,
			EffectiveFrom:   &effFrom,
			EffectiveUntil:  &effUntil,
		}
		id, err := f.K.CreatePermission(f.Ctx, perm)
		require.NoError(t, err)

		newUntil := now.Add(60 * 24 * time.Hour)
		_, err = f.MS.AdjustPermission(f.Ctx, &types.MsgAdjustPermission{
			Id:             id,
			Corporation:    corp,
			Operator:       corp,
			EffectiveUntil: &newUntil,
		})
		require.NoError(t, err)

		// Full struct assertion — only EffectiveUntil, Adjusted, Modified change
		got, err := f.K.GetPermissionByID(f.Ctx, id)
		require.NoError(t, err)
		require.Equal(t, specAdjustEffectiveUntil(&newUntil), got.EffectiveUntil)
		require.NotNil(t, got.Adjusted)
		require.NotNil(t, got.Modified)
		require.Equal(t, now, *got.Adjusted)
		require.Equal(t, now, *got.Modified)

		// No bank transfers on AdjustPermission
		f.RequireNoBalanceChange(corp)

		// Event assertion
		f.RequireEvent(types.EventTypeAdjustPermission, map[string]string{
			types.AttributeKeyPermissionID: fmt.Sprintf("%d", id),
			types.AttributeKeyCorporation:  corp,
		})

		// Module invariant
		f.RequireInvariant()
	})

	// ------------------------------------------------------------------ //
	// Happy path 2: Self-created perm (validator.Type == ECOSYSTEM)        //
	// ------------------------------------------------------------------ //
	t.Run("MOD-PERM-MSG-8-2-2: self-created perm via ECOSYSTEM validator", func(t *testing.T) {
		f := NewFixture(t)
		now := time.Date(2026, 3, 1, 0, 0, 0, 0, time.UTC)
		f.SetBlockTime(now)

		corp := sdk.AccAddress([]byte("corp_self_adjust_1")).String()
		f.SetBalance(corp, types.BondDenom, 1000)

		schemaID := uint64(1)
		f.CSKeeper.CreateMockCredentialSchema(schemaID,
			cstypes.IssuerOnboardingMode_ISSUER_ONBOARDING_MODE_OPEN,
			cstypes.VerifierOnboardingMode_VERIFIER_ONBOARDING_MODE_OPEN)

		// Create parent ECOSYSTEM perm
		ecoEffFrom := now.Add(-48 * time.Hour)
		ecoEffUntil := now.Add(365 * 24 * time.Hour)
		ecoPerm := types.Permission{
			SchemaId:        schemaID,
			Type:            types.PermissionType_ECOSYSTEM,
			Corporation:     corp,
			ValidatorPermId: 0,
			EffectiveFrom:   &ecoEffFrom,
			EffectiveUntil:  &ecoEffUntil,
		}
		ecoID, err := f.K.CreatePermission(f.Ctx, ecoPerm)
		require.NoError(t, err)

		// Create child ISSUER perm referencing the ECOSYSTEM perm
		issuerEffFrom := now.Add(-24 * time.Hour)
		issuerEffUntil := now.Add(30 * 24 * time.Hour)
		issuerPerm := types.Permission{
			SchemaId:        schemaID,
			Type:            types.PermissionType_ISSUER,
			Corporation:     corp,
			ValidatorPermId: ecoID,
			EffectiveFrom:   &issuerEffFrom,
			EffectiveUntil:  &issuerEffUntil,
		}
		issuerID, err := f.K.CreatePermission(f.Ctx, issuerPerm)
		require.NoError(t, err)

		newUntil := now.Add(60 * 24 * time.Hour)
		_, err = f.MS.AdjustPermission(f.Ctx, &types.MsgAdjustPermission{
			Id:             issuerID,
			Corporation:    corp,
			Operator:       corp,
			EffectiveUntil: &newUntil,
		})
		require.NoError(t, err)

		got, err := f.K.GetPermissionByID(f.Ctx, issuerID)
		require.NoError(t, err)
		require.Equal(t, &newUntil, got.EffectiveUntil)
		require.NotNil(t, got.Adjusted)
		require.Equal(t, now, *got.Adjusted)

		f.RequireNoBalanceChange(corp)
		f.RequireEvent(types.EventTypeAdjustPermission, map[string]string{
			types.AttributeKeyPermissionID: fmt.Sprintf("%d", issuerID),
		})
		f.RequireInvariant()
	})

	// ------------------------------------------------------------------ //
	// Happy path 3: VP-managed perm (validator is not ECOSYSTEM type)      //
	// ------------------------------------------------------------------ //
	t.Run("MOD-PERM-MSG-8-2-2: VP-managed perm — effective_until <= vpExp, corp == validator.Corporation", func(t *testing.T) {
		f := NewFixture(t)
		now := time.Date(2026, 3, 1, 0, 0, 0, 0, time.UTC)
		f.SetBlockTime(now)

		vpCorp := sdk.AccAddress([]byte("corp_vp_adjust___1")).String()
		subCorp := sdk.AccAddress([]byte("corp_sub_adjust__1")).String()
		f.SetBalance(vpCorp, types.BondDenom, 1000)
		f.SetBalance(subCorp, types.BondDenom, 1000)

		schemaID := uint64(1)
		f.CSKeeper.CreateMockCredentialSchema(schemaID,
			cstypes.IssuerOnboardingMode_ISSUER_ONBOARDING_MODE_OPEN,
			cstypes.VerifierOnboardingMode_VERIFIER_ONBOARDING_MODE_OPEN)

		// Create an ISSUER validator perm (non-ECOSYSTEM) owned by vpCorp
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

		// VpExp caps how far subCorp can extend its effective_until
		vpExp := now.Add(90 * 24 * time.Hour)

		// Create child perm for subCorp, validated by vpCorp's perm
		subEffFrom := now.Add(-24 * time.Hour)
		subEffUntil := now.Add(30 * 24 * time.Hour)
		subPerm := types.Permission{
			SchemaId:        schemaID,
			Type:            types.PermissionType_ISSUER,
			Corporation:     subCorp,
			ValidatorPermId: vpID,
			EffectiveFrom:   &subEffFrom,
			EffectiveUntil:  &subEffUntil,
			VpExp:           &vpExp,
		}
		subID, err := f.K.CreatePermission(f.Ctx, subPerm)
		require.NoError(t, err)

		// Adjust within vpExp — caller is vpCorp (the validator authority)
		newUntil := now.Add(60 * 24 * time.Hour) // < vpExp (90 days)
		_, err = f.MS.AdjustPermission(f.Ctx, &types.MsgAdjustPermission{
			Id:             subID,
			Corporation:    vpCorp,
			Operator:       vpCorp,
			EffectiveUntil: &newUntil,
		})
		require.NoError(t, err)

		got, err := f.K.GetPermissionByID(f.Ctx, subID)
		require.NoError(t, err)
		require.Equal(t, &newUntil, got.EffectiveUntil)
		require.NotNil(t, got.Adjusted)

		f.RequireNoBalanceChange(vpCorp)
		f.RequireNoBalanceChange(subCorp)
		f.RequireInvariant()
	})

	// ------------------------------------------------------------------ //
	// Negative: [MOD-PERM-MSG-8-2-1] operator authorization failure        //
	// ------------------------------------------------------------------ //
	t.Run("MOD-PERM-MSG-8-2-1: operator authorization failure", func(t *testing.T) {
		f := NewFixture(t)
		now := time.Date(2026, 3, 1, 0, 0, 0, 0, time.UTC)
		f.SetBlockTime(now)

		corp := sdk.AccAddress([]byte("corp_authz_adj___1")).String()
		f.SetBalance(corp, types.BondDenom, 1000)

		f.DelKeeper.ErrToReturn = fmt.Errorf("mock: operator not authorized")

		schemaID := uint64(1)
		f.CSKeeper.CreateMockCredentialSchema(schemaID,
			cstypes.IssuerOnboardingMode_ISSUER_ONBOARDING_MODE_OPEN,
			cstypes.VerifierOnboardingMode_VERIFIER_ONBOARDING_MODE_OPEN)

		effFrom := now.Add(-24 * time.Hour)
		effUntil := now.Add(30 * 24 * time.Hour)
		perm := types.Permission{
			SchemaId:       schemaID,
			Type:           types.PermissionType_ECOSYSTEM,
			Corporation:    corp,
			EffectiveFrom:  &effFrom,
			EffectiveUntil: &effUntil,
		}
		id, err := f.K.CreatePermission(f.Ctx, perm)
		require.NoError(t, err)

		newUntil := now.Add(60 * 24 * time.Hour)
		_, err = f.MS.AdjustPermission(f.Ctx, &types.MsgAdjustPermission{
			Id:             id,
			Corporation:    corp,
			Operator:       sdk.AccAddress([]byte("bad_operator_____1")).String(),
			EffectiveUntil: &newUntil,
		})
		require.ErrorContains(t, err, "authorization check failed")
		f.RequireNoBalanceChange(corp)
	})

	// ------------------------------------------------------------------ //
	// Negative: [MOD-PERM-MSG-8-2-1a] permission not found                //
	// ------------------------------------------------------------------ //
	t.Run("MOD-PERM-MSG-8-2-1a: permission not found", func(t *testing.T) {
		f := NewFixture(t)
		now := time.Date(2026, 3, 1, 0, 0, 0, 0, time.UTC)
		f.SetBlockTime(now)

		corp := sdk.AccAddress([]byte("corp_notfound_adj1")).String()
		f.SetBalance(corp, types.BondDenom, 1000)

		newUntil := now.Add(60 * 24 * time.Hour)
		_, err := f.MS.AdjustPermission(f.Ctx, &types.MsgAdjustPermission{
			Id:             99999,
			Corporation:    corp,
			Operator:       corp,
			EffectiveUntil: &newUntil,
		})
		require.ErrorContains(t, err, "permission not found")
		f.RequireNoBalanceChange(corp)
	})

	// ------------------------------------------------------------------ //
	// Negative: [MOD-PERM-MSG-8-2-1b] perm is revoked (not active)        //
	// ------------------------------------------------------------------ //
	t.Run("MOD-PERM-MSG-8-2-1b: perm is revoked (not valid)", func(t *testing.T) {
		f := NewFixture(t)
		now := time.Date(2026, 3, 1, 0, 0, 0, 0, time.UTC)
		f.SetBlockTime(now)

		corp := sdk.AccAddress([]byte("corp_revoked_adj_1")).String()
		f.SetBalance(corp, types.BondDenom, 1000)

		schemaID := uint64(1)
		f.CSKeeper.CreateMockCredentialSchema(schemaID,
			cstypes.IssuerOnboardingMode_ISSUER_ONBOARDING_MODE_OPEN,
			cstypes.VerifierOnboardingMode_VERIFIER_ONBOARDING_MODE_OPEN)

		effFrom := now.Add(-24 * time.Hour)
		effUntil := now.Add(30 * 24 * time.Hour)
		revokedAt := now.Add(-1 * time.Hour)
		perm := types.Permission{
			SchemaId:       schemaID,
			Type:           types.PermissionType_ECOSYSTEM,
			Corporation:    corp,
			EffectiveFrom:  &effFrom,
			EffectiveUntil: &effUntil,
			Revoked:        &revokedAt,
		}
		id, err := f.K.CreatePermission(f.Ctx, perm)
		require.NoError(t, err)

		newUntil := now.Add(60 * 24 * time.Hour)
		_, err = f.MS.AdjustPermission(f.Ctx, &types.MsgAdjustPermission{
			Id:             id,
			Corporation:    corp,
			Operator:       corp,
			EffectiveUntil: &newUntil,
		})
		require.ErrorContains(t, err, "applicant permission is not valid")
		f.RequireNoBalanceChange(corp)
	})

	// ------------------------------------------------------------------ //
	// Negative: [MOD-PERM-MSG-8-2-1c] effective_until <= now              //
	// ------------------------------------------------------------------ //
	t.Run("MOD-PERM-MSG-8-2-1c: effective_until is not in the future", func(t *testing.T) {
		f := NewFixture(t)
		now := time.Date(2026, 3, 1, 0, 0, 0, 0, time.UTC)
		f.SetBlockTime(now)

		corp := sdk.AccAddress([]byte("corp_pastunt_adj_1")).String()
		f.SetBalance(corp, types.BondDenom, 1000)

		schemaID := uint64(1)
		f.CSKeeper.CreateMockCredentialSchema(schemaID,
			cstypes.IssuerOnboardingMode_ISSUER_ONBOARDING_MODE_OPEN,
			cstypes.VerifierOnboardingMode_VERIFIER_ONBOARDING_MODE_OPEN)

		effFrom := now.Add(-24 * time.Hour)
		effUntil := now.Add(30 * 24 * time.Hour)
		perm := types.Permission{
			SchemaId:       schemaID,
			Type:           types.PermissionType_ECOSYSTEM,
			Corporation:    corp,
			EffectiveFrom:  &effFrom,
			EffectiveUntil: &effUntil,
		}
		id, err := f.K.CreatePermission(f.Ctx, perm)
		require.NoError(t, err)

		pastUntil := now.Add(-1 * time.Hour) // in the past
		_, err = f.MS.AdjustPermission(f.Ctx, &types.MsgAdjustPermission{
			Id:             id,
			Corporation:    corp,
			Operator:       corp,
			EffectiveUntil: &pastUntil,
		})
		require.ErrorContains(t, err, "effective_until must be greater than current timestamp")
		f.RequireNoBalanceChange(corp)
	})

	// ------------------------------------------------------------------ //
	// Negative: [MOD-PERM-MSG-8-2-2] ECOSYSTEM — wrong corporation        //
	// ------------------------------------------------------------------ //
	t.Run("MOD-PERM-MSG-8-2-2: ECOSYSTEM perm — wrong corporation", func(t *testing.T) {
		f := NewFixture(t)
		now := time.Date(2026, 3, 1, 0, 0, 0, 0, time.UTC)
		f.SetBlockTime(now)

		corp := sdk.AccAddress([]byte("corp_eco_adj_ok__1")).String()
		wrongCorp := sdk.AccAddress([]byte("corp_eco_adj_bad1")).String()
		f.SetBalance(corp, types.BondDenom, 1000)

		schemaID := uint64(1)
		f.CSKeeper.CreateMockCredentialSchema(schemaID,
			cstypes.IssuerOnboardingMode_ISSUER_ONBOARDING_MODE_OPEN,
			cstypes.VerifierOnboardingMode_VERIFIER_ONBOARDING_MODE_OPEN)

		effFrom := now.Add(-24 * time.Hour)
		effUntil := now.Add(30 * 24 * time.Hour)
		perm := types.Permission{
			SchemaId:        schemaID,
			Type:            types.PermissionType_ECOSYSTEM,
			Corporation:     corp,
			ValidatorPermId: 0,
			EffectiveFrom:   &effFrom,
			EffectiveUntil:  &effUntil,
		}
		id, err := f.K.CreatePermission(f.Ctx, perm)
		require.NoError(t, err)

		newUntil := now.Add(60 * 24 * time.Hour)
		_, err = f.MS.AdjustPermission(f.Ctx, &types.MsgAdjustPermission{
			Id:             id,
			Corporation:    wrongCorp,
			Operator:       wrongCorp,
			EffectiveUntil: &newUntil,
		})
		require.ErrorContains(t, err, "authority is not the permission authority")
		f.RequireNoBalanceChange(corp)
	})

	// ------------------------------------------------------------------ //
	// Negative: [MOD-PERM-MSG-8-2-2] VP-managed — effective_until > vpExp //
	// ------------------------------------------------------------------ //
	t.Run("MOD-PERM-MSG-8-2-2: VP-managed perm — effective_until exceeds vp_exp", func(t *testing.T) {
		f := NewFixture(t)
		now := time.Date(2026, 3, 1, 0, 0, 0, 0, time.UTC)
		f.SetBlockTime(now)

		vpCorp := sdk.AccAddress([]byte("corp_vp_adj_exp__1")).String()
		subCorp := sdk.AccAddress([]byte("corp_sub_adj_exp1")).String()
		f.SetBalance(vpCorp, types.BondDenom, 1000)

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

		vpExp := now.Add(90 * 24 * time.Hour)
		subEffFrom := now.Add(-24 * time.Hour)
		subEffUntil := now.Add(30 * 24 * time.Hour)
		subPerm := types.Permission{
			SchemaId:        schemaID,
			Type:            types.PermissionType_ISSUER,
			Corporation:     subCorp,
			ValidatorPermId: vpID,
			EffectiveFrom:   &subEffFrom,
			EffectiveUntil:  &subEffUntil,
			VpExp:           &vpExp,
		}
		subID, err := f.K.CreatePermission(f.Ctx, subPerm)
		require.NoError(t, err)

		// Request effective_until beyond vpExp
		beyondVpExp := vpExp.Add(24 * time.Hour)
		_, err = f.MS.AdjustPermission(f.Ctx, &types.MsgAdjustPermission{
			Id:             subID,
			Corporation:    vpCorp,
			Operator:       vpCorp,
			EffectiveUntil: &beyondVpExp,
		})
		require.ErrorContains(t, err, "effective_until cannot be after validation expiration")
		f.RequireNoBalanceChange(vpCorp)
	})

	// ------------------------------------------------------------------ //
	// Negative: [MOD-PERM-MSG-8-2-4] overlap check                       //
	// ------------------------------------------------------------------ //
	t.Run("MOD-PERM-MSG-8-2-4: overlap check rejects conflicting adjustment", func(t *testing.T) {
		f := NewFixture(t)
		now := time.Date(2026, 3, 1, 0, 0, 0, 0, time.UTC)
		f.SetBlockTime(now)

		corp := sdk.AccAddress([]byte("corp_overlap_adj1")).String()
		f.SetBalance(corp, types.BondDenom, 1000)

		schemaID := uint64(1)
		f.CSKeeper.CreateMockCredentialSchema(schemaID,
			cstypes.IssuerOnboardingMode_ISSUER_ONBOARDING_MODE_OPEN,
			cstypes.VerifierOnboardingMode_VERIFIER_ONBOARDING_MODE_OPEN)

		// Create two ECOSYSTEM perms for the same corp/schema; perm2 starts after perm1 ends
		perm1EffFrom := now.Add(-48 * time.Hour)
		perm1EffUntil := now.Add(10 * 24 * time.Hour)
		perm1 := types.Permission{
			SchemaId:        schemaID,
			Type:            types.PermissionType_ECOSYSTEM,
			Corporation:     corp,
			ValidatorPermId: 0,
			EffectiveFrom:   &perm1EffFrom,
			EffectiveUntil:  &perm1EffUntil,
		}
		perm1ID, err := f.K.CreatePermission(f.Ctx, perm1)
		require.NoError(t, err)

		perm2EffFrom := now.Add(11 * 24 * time.Hour) // starts after perm1 ends
		perm2EffUntil := now.Add(40 * 24 * time.Hour)
		perm2 := types.Permission{
			SchemaId:        schemaID,
			Type:            types.PermissionType_ECOSYSTEM,
			Corporation:     corp,
			ValidatorPermId: 0,
			EffectiveFrom:   &perm2EffFrom,
			EffectiveUntil:  &perm2EffUntil,
		}
		_, err = f.K.CreatePermission(f.Ctx, perm2)
		require.NoError(t, err)

		// Try to extend perm1 so it overlaps with perm2's range
		overlappingUntil := now.Add(20 * 24 * time.Hour) // perm2 starts at day 11, this goes to day 20
		_, err = f.MS.AdjustPermission(f.Ctx, &types.MsgAdjustPermission{
			Id:             perm1ID,
			Corporation:    corp,
			Operator:       corp,
			EffectiveUntil: &overlappingUntil,
		})
		require.ErrorContains(t, err, "overlap check failed")
		f.RequireNoBalanceChange(corp)
	})

	// ------------------------------------------------------------------ //
	// Edge: ISSUER perm with VsOperatorAuthzEnabled triggers grant call    //
	// ------------------------------------------------------------------ //
	t.Run("edge: ISSUER perm with VsOperatorAuthzEnabled triggers grantVSOperatorAuthorization", func(t *testing.T) {
		f := NewFixture(t)
		now := time.Date(2026, 3, 1, 0, 0, 0, 0, time.UTC)
		f.SetBlockTime(now)

		corp := sdk.AccAddress([]byte("corp_vsoa_adj____1")).String()
		vsOp := sdk.AccAddress([]byte("vsop_adj_________1")).String()
		f.SetBalance(corp, types.BondDenom, 1000)

		schemaID := uint64(1)
		f.CSKeeper.CreateMockCredentialSchema(schemaID,
			cstypes.IssuerOnboardingMode_ISSUER_ONBOARDING_MODE_OPEN,
			cstypes.VerifierOnboardingMode_VERIFIER_ONBOARDING_MODE_OPEN)

		effFrom := now.Add(-24 * time.Hour)
		effUntil := now.Add(30 * 24 * time.Hour)
		perm := types.Permission{
			SchemaId:               schemaID,
			Type:                   types.PermissionType_ISSUER,
			Corporation:            corp,
			ValidatorPermId:        0,
			EffectiveFrom:          &effFrom,
			EffectiveUntil:         &effUntil,
			VsOperator:             vsOp,
			VsOperatorAuthzEnabled: true,
		}
		id, err := f.K.CreatePermission(f.Ctx, perm)
		require.NoError(t, err)

		newUntil := now.Add(60 * 24 * time.Hour)
		_, err = f.MS.AdjustPermission(f.Ctx, &types.MsgAdjustPermission{
			Id:             id,
			Corporation:    corp,
			Operator:       corp,
			EffectiveUntil: &newUntil,
		})
		require.NoError(t, err)

		// delegationKeeper.AddPermToVSOA should have been called
		require.Len(t, f.DelKeeper.AddPermToVSOACalls, 1)
		require.Equal(t, corp, f.DelKeeper.AddPermToVSOACalls[0].Authority)

		f.RequireNoBalanceChange(corp)
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
  go test ./x/perm/keeper/... -run TestMsgAdjustPermission -v -count=1
  ```
  Expected: all subtests PASS.

- [ ] **Step 1.4: Commit.**

  ```bash
  git add x/perm/keeper/msg_adjust_permission_test.go
  git commit -m "test(perm): add fixture-based AdjustPermission tests"
  ```

---

## Task 2: Delete old tests for `AdjustPermission`

**File:** `x/perm/keeper/msg_server_test.go`

- [ ] **Step 2.1: Delete the `TestMsgAdjustPermission` function block.**

  Remove the entire `func TestMsgAdjustPermission(t *testing.T)` block from `x/perm/keeper/msg_server_test.go`. Keep all other test functions intact.

- [ ] **Step 2.2: Verify build and test suite.**

  ```bash
  go test ./x/perm/keeper/... -count=1
  ```
  Expected: no compilation errors, all remaining tests PASS.

- [ ] **Step 2.3: Commit.**

  ```bash
  git add x/perm/keeper/msg_server_test.go
  git commit -m "test(perm): delete legacy TestMsgAdjustPermission"
  ```

---

## Task 3: Final pass

- [ ] **Step 3.1: Run full PERM keeper test suite.**

  ```bash
  go test ./x/perm/keeper/... -v -count=1
  ```
  Expected: PASS.

- [ ] **Step 3.2: Race detector.**

  ```bash
  go test ./x/perm/keeper/... -race -count=1
  ```
  Expected: PASS, no DATA RACE.

- [ ] **Step 3.3: Vet and lint.**

  ```bash
  go vet ./x/perm/keeper/... && golangci-lint run ./x/perm/keeper/...
  ```
  Expected: no output.

- [ ] **Step 3.4: Coverage check.**

  ```bash
  go test ./x/perm/keeper/ -cover -count=1
  ```
  Expected: coverage ≥95%.

- [ ] **Step 3.5: Full repo sanity.**

  ```bash
  go build ./... && go test ./... -count=1
  ```
  Expected: PASS.

- [ ] **Step 3.6: Push and open PR.**

  ```bash
  git push -u origin test/step-26-perm-adjust-permission
  gh pr create --title "test(perm): fixture-based AdjustPermission tests (step 26)" --body "$(cat <<'EOF'
  ## Summary
  - Adds fixture-based `TestMsgAdjustPermission` in `msg_adjust_permission_test.go`
  - Covers all three authority paths: ECOSYSTEM, self-created via ECOSYSTEM validator, VP-managed
  - All 8 preconditions covered with independent t.Run blocks
  - No bank transfers — all tests call RequireNoBalanceChange
  - VsOperatorAuthzEnabled side-effect tested via DelKeeper.AddPermToVSOACalls
  - Deletes legacy TestMsgAdjustPermission from msg_server_test.go

  ## Test plan
  - [ ] `go test ./x/perm/keeper/... -race -count=1` passes
  - [ ] `go test ./x/perm/keeper/ -cover` reports ≥95%
  - [ ] `golangci-lint run ./x/perm/keeper/...` clean
  - [ ] `go test ./...` passes — no regressions
  EOF
  )"
  ```

---

## "Done" Criteria — Step 26

- [ ] `x/perm/keeper/msg_adjust_permission_test.go` exists with `TestMsgAdjustPermission`.
- [ ] Three happy paths: ECOSYSTEM, self-created, VP-managed — each with full struct assertion + event + invariant + `RequireNoBalanceChange`.
- [ ] Six negative subtests: authz failure, perm not found, perm revoked, effective_until in past, wrong corporation (ECOSYSTEM), effective_until > vpExp (VP-managed), overlap.
- [ ] One edge case: `VsOperatorAuthzEnabled` triggers `AddPermToVSOACalls`.
- [ ] Legacy `TestMsgAdjustPermission` deleted from `msg_server_test.go`.
- [ ] `go test ./x/perm/keeper/... -race -count=1` passes.
- [ ] Coverage ≥95% on `x/perm/keeper/`.
