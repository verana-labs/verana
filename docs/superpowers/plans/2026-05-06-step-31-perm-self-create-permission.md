# Step 31: PERM SelfCreatePermission — Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Replace legacy `TestMsgSelfCreatePermission` tests with fixture-based tests that verify full struct equality on the created permission, event emission, the PERM module invariant, and every spec precondition — including all `effective_from`/`effective_until` boundary conditions and the `cs.IssuerOnboardingMode`/`cs.VerifierOnboardingMode` OPEN checks.

**Architecture:** `SelfCreatePermission` creates a new permission with no bank transfers and no fees. All tests call `f.RequireNoBalanceChange`. The `AddPermToVSOACalls` field of `MockDelegationKeeper` is used to assert the VS operator authorization side-effect. Step 31 is the last PERM message test step — after all tasks pass, the `RequireInvariant` in the fixture must validate the complete PERM module invariant (sum of VpCurrentFees across PENDING perms == module balance; all VALIDATED perms have non-nil EffectiveFrom).

**Tech Stack:** Go 1.24, Cosmos SDK v0.50.x, testify/require, StatefulBankMock from testutil/keeper/bank.go

---

## Pre-flight

- [ ] **Confirm step 21 is merged.** `x/perm/keeper/fixture_test.go` must exist.

  ```bash
  ls x/perm/keeper/fixture_test.go
  ```
  Expected: file exists.

- [ ] **Create worktree.** Branch name: `test/step-31-perm-self-create-permission`.

- [ ] **Sanity check.**

  ```bash
  go build ./... && go test ./x/perm/keeper/... -count=1
  ```
  Expected: all pass.

---

## File Structure

- **Create:** `x/perm/keeper/msg_self_create_permission_test.go` — fixture-based tests.
- **Delete (in same PR):** the `TestMsgSelfCreatePermission` function block from `x/perm/keeper/msg_server_test.go`.

---

## Task 1: Write `x/perm/keeper/msg_self_create_permission_test.go`

**File:** `x/perm/keeper/msg_self_create_permission_test.go`

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
// Spec formula functions
// ============================================================================
// SelfCreatePermission has no fee transfers. The only "formula" is the expected
// state of the created permission.

// specSelfCreateExpectedPerm returns the expected Permission state after a
// successful SelfCreatePermission call, given the inputs.
// fees are only non-zero for ISSUER type.
func specSelfCreateExpectedPerm(
	validatorPermId, schemaId uint64,
	permType types.PermissionType,
	corp, vsOp, did string,
	effectiveFrom, effectiveUntil *time.Time,
	issuanceFees, verificationFees uint64,
	vsOperatorAuthzEnabled bool,
	createdAt time.Time,
) types.Permission {
	p := types.Permission{
		ValidatorPermId:        validatorPermId,
		SchemaId:               schemaId,
		Type:                   permType,
		Corporation:            corp,
		VsOperator:             vsOp,
		Did:                    did,
		EffectiveFrom:          effectiveFrom,
		EffectiveUntil:         effectiveUntil,
		Deposit:                0,
		VsOperatorAuthzEnabled: vsOperatorAuthzEnabled,
		Modified:               &createdAt,
		Created:                &createdAt,
	}
	if permType == types.PermissionType_ISSUER {
		p.IssuanceFees = issuanceFees
		p.VerificationFees = verificationFees
	}
	return p
}

// ============================================================================
// Setup helpers
// ============================================================================

// makeEcoPerm creates an ECOSYSTEM permission that can be used as a validator
// for SelfCreatePermission. Returns the permission ID.
func makeEcoPerm(
	t *testing.T,
	f *Fixture,
	now time.Time,
	corp string,
	schemaID uint64,
	effectiveFrom, effectiveUntil *time.Time,
) uint64 {
	t.Helper()
	perm := types.Permission{
		SchemaId:        schemaID,
		Type:            types.PermissionType_ECOSYSTEM,
		Corporation:     corp,
		ValidatorPermId: 0,
		EffectiveFrom:   effectiveFrom,
		EffectiveUntil:  effectiveUntil,
	}
	id, err := f.K.CreatePermission(f.Ctx, perm)
	require.NoError(t, err)
	return id
}

// ============================================================================
// TestMsgSelfCreatePermission
// ============================================================================

func TestMsgSelfCreatePermission(t *testing.T) {

	// ------------------------------------------------------------------ //
	// Happy path 1: ISSUER self-create under OPEN schema                  //
	// ------------------------------------------------------------------ //
	t.Run("MOD-PERM-MSG-14-2-2: ISSUER self-create on OPEN schema", func(t *testing.T) {
		f := NewFixture(t)
		now := time.Date(2026, 3, 1, 0, 0, 0, 0, time.UTC)
		f.SetBlockTime(now)

		ecoCorp := sdk.AccAddress([]byte("corp_eco_scp_____1")).String()
		newCorp := sdk.AccAddress([]byte("corp_new_scp_____1")).String()
		f.SetBalance(newCorp, types.BondDenom, 1000)

		schemaID := uint64(1)
		f.CSKeeper.CreateMockCredentialSchema(schemaID,
			cstypes.IssuerOnboardingMode_ISSUER_ONBOARDING_MODE_OPEN,
			cstypes.VerifierOnboardingMode_VERIFIER_ONBOARDING_MODE_OPEN)

		ecoEffFrom := now.Add(-48 * time.Hour)
		ecoEffUntil := now.Add(365 * 24 * time.Hour)
		ecoID := makeEcoPerm(t, f, now, ecoCorp, schemaID, &ecoEffFrom, &ecoEffUntil)

		// effective_from in future; effective_until within ecoEffUntil
		newEffFrom := now.Add(1 * time.Hour)
		newEffUntil := now.Add(90 * 24 * time.Hour)

		issuanceFees := uint64(500)
		verificationFees := uint64(200)

		resp, err := f.MS.SelfCreatePermission(f.Ctx, &types.MsgSelfCreatePermission{
			ValidatorPermId:  ecoID,
			Type:             types.PermissionType_ISSUER,
			Corporation:      newCorp,
			Operator:         newCorp,
			Did:              "did:example:newissuer1",
			EffectiveFrom:    &newEffFrom,
			EffectiveUntil:   &newEffUntil,
			IssuanceFees:     issuanceFees,
			VerificationFees: verificationFees,
		})
		require.NoError(t, err)
		require.NotZero(t, resp.Id)

		// Full struct assertion
		got, err := f.K.GetPermissionByID(f.Ctx, resp.Id)
		require.NoError(t, err)

		expected := specSelfCreateExpectedPerm(
			ecoID, schemaID, types.PermissionType_ISSUER,
			newCorp, "", "did:example:newissuer1",
			&newEffFrom, &newEffUntil,
			issuanceFees, verificationFees,
			false, now,
		)
		expected.Id = resp.Id
		require.Equal(t, expected, got)

		// No bank transfers
		f.RequireNoBalanceChange(newCorp)

		// Event assertion
		f.RequireEvent(types.EventTypeCreatePermission, map[string]string{
			types.AttributeKeyPermissionID: fmt.Sprintf("%d", resp.Id),
			types.AttributeKeyCorporation:  newCorp,
			types.AttributeKeySchemaID:     fmt.Sprintf("%d", schemaID),
			types.AttributeKeyType:         types.PermissionType_ISSUER.String(),
		})

		// Module invariant
		f.RequireInvariant()
	})

	// ------------------------------------------------------------------ //
	// Happy path 2: VERIFIER self-create under OPEN schema                //
	// ------------------------------------------------------------------ //
	t.Run("MOD-PERM-MSG-14-2-2: VERIFIER self-create on OPEN schema", func(t *testing.T) {
		f := NewFixture(t)
		now := time.Date(2026, 3, 1, 0, 0, 0, 0, time.UTC)
		f.SetBlockTime(now)

		ecoCorp := sdk.AccAddress([]byte("corp_eco_scp_vrf1")).String()
		newCorp := sdk.AccAddress([]byte("corp_new_scp_vrf1")).String()
		f.SetBalance(newCorp, types.BondDenom, 1000)

		schemaID := uint64(1)
		f.CSKeeper.CreateMockCredentialSchema(schemaID,
			cstypes.IssuerOnboardingMode_ISSUER_ONBOARDING_MODE_OPEN,
			cstypes.VerifierOnboardingMode_VERIFIER_ONBOARDING_MODE_OPEN)

		ecoEffFrom := now.Add(-48 * time.Hour)
		ecoEffUntil := now.Add(365 * 24 * time.Hour)
		ecoID := makeEcoPerm(t, f, now, ecoCorp, schemaID, &ecoEffFrom, &ecoEffUntil)

		newEffFrom := now.Add(1 * time.Hour)
		newEffUntil := now.Add(60 * 24 * time.Hour)

		resp, err := f.MS.SelfCreatePermission(f.Ctx, &types.MsgSelfCreatePermission{
			ValidatorPermId: ecoID,
			Type:            types.PermissionType_VERIFIER,
			Corporation:     newCorp,
			Operator:        newCorp,
			Did:             "did:example:newverifier1",
			EffectiveFrom:   &newEffFrom,
			EffectiveUntil:  &newEffUntil,
			// VERIFIER must not have fees set
			IssuanceFees:     0,
			VerificationFees: 0,
		})
		require.NoError(t, err)

		got, err := f.K.GetPermissionByID(f.Ctx, resp.Id)
		require.NoError(t, err)
		require.Equal(t, types.PermissionType_VERIFIER, got.Type)
		require.Equal(t, uint64(0), got.VerificationFees) // VERIFIER never has fees
		require.Equal(t, uint64(0), got.Deposit)
		require.Nil(t, got.Revoked)
		require.NotNil(t, got.Created)

		f.RequireNoBalanceChange(newCorp)
		f.RequireInvariant()
	})

	// ------------------------------------------------------------------ //
	// Happy path 3: nil effective_from defaults to now                    //
	// ------------------------------------------------------------------ //
	t.Run("MOD-PERM-MSG-14-2-1d: nil effective_from defaults to now (active immediately)", func(t *testing.T) {
		f := NewFixture(t)
		now := time.Date(2026, 3, 1, 0, 0, 0, 0, time.UTC)
		f.SetBlockTime(now)

		ecoCorp := sdk.AccAddress([]byte("corp_eco_scp_nil1")).String()
		newCorp := sdk.AccAddress([]byte("corp_new_scp_nil1")).String()
		f.SetBalance(newCorp, types.BondDenom, 1000)

		schemaID := uint64(1)
		f.CSKeeper.CreateMockCredentialSchema(schemaID,
			cstypes.IssuerOnboardingMode_ISSUER_ONBOARDING_MODE_OPEN,
			cstypes.VerifierOnboardingMode_VERIFIER_ONBOARDING_MODE_OPEN)

		ecoEffFrom := now.Add(-48 * time.Hour)
		ecoEffUntil := now.Add(365 * 24 * time.Hour)
		ecoID := makeEcoPerm(t, f, now, ecoCorp, schemaID, &ecoEffFrom, &ecoEffUntil)

		newEffUntil := now.Add(60 * 24 * time.Hour)
		resp, err := f.MS.SelfCreatePermission(f.Ctx, &types.MsgSelfCreatePermission{
			ValidatorPermId: ecoID,
			Type:            types.PermissionType_ISSUER,
			Corporation:     newCorp,
			Operator:        newCorp,
			EffectiveFrom:   nil, // should default to now
			EffectiveUntil:  &newEffUntil,
		})
		require.NoError(t, err)

		got, err := f.K.GetPermissionByID(f.Ctx, resp.Id)
		require.NoError(t, err)
		require.NotNil(t, got.EffectiveFrom)
		require.Equal(t, now, *got.EffectiveFrom, "nil effective_from must default to block time")

		f.RequireNoBalanceChange(newCorp)
		f.RequireInvariant()
	})

	// ------------------------------------------------------------------ //
	// Happy path 4: VsOperatorAuthzEnabled triggers grant call            //
	// ------------------------------------------------------------------ //
	t.Run("edge: VsOperatorAuthzEnabled triggers AddPermToVSOA", func(t *testing.T) {
		f := NewFixture(t)
		now := time.Date(2026, 3, 1, 0, 0, 0, 0, time.UTC)
		f.SetBlockTime(now)

		ecoCorp := sdk.AccAddress([]byte("corp_eco_scp_vso1")).String()
		newCorp := sdk.AccAddress([]byte("corp_new_scp_vso1")).String()
		vsOp := sdk.AccAddress([]byte("vsop_scp_________1")).String()
		f.SetBalance(newCorp, types.BondDenom, 1000)

		schemaID := uint64(1)
		f.CSKeeper.CreateMockCredentialSchema(schemaID,
			cstypes.IssuerOnboardingMode_ISSUER_ONBOARDING_MODE_OPEN,
			cstypes.VerifierOnboardingMode_VERIFIER_ONBOARDING_MODE_OPEN)

		ecoEffFrom := now.Add(-48 * time.Hour)
		ecoEffUntil := now.Add(365 * 24 * time.Hour)
		ecoID := makeEcoPerm(t, f, now, ecoCorp, schemaID, &ecoEffFrom, &ecoEffUntil)

		newEffFrom := now.Add(1 * time.Hour)
		newEffUntil := now.Add(90 * 24 * time.Hour)

		resp, err := f.MS.SelfCreatePermission(f.Ctx, &types.MsgSelfCreatePermission{
			ValidatorPermId:        ecoID,
			Type:                   types.PermissionType_ISSUER,
			Corporation:            newCorp,
			Operator:               newCorp,
			VsOperator:             vsOp,
			VsOperatorAuthzEnabled: true,
			EffectiveFrom:          &newEffFrom,
			EffectiveUntil:         &newEffUntil,
		})
		require.NoError(t, err)

		require.Len(t, f.DelKeeper.AddPermToVSOACalls, 1)
		require.Equal(t, newCorp, f.DelKeeper.AddPermToVSOACalls[0].Authority)
		require.Equal(t, vsOp, f.DelKeeper.AddPermToVSOACalls[0].VsOperator)
		require.Equal(t, resp.Id, f.DelKeeper.AddPermToVSOACalls[0].PermID)

		f.RequireNoBalanceChange(newCorp)
		f.RequireInvariant()
	})

	// ------------------------------------------------------------------ //
	// Negative: [MOD-PERM-MSG-14-2-1] operator authorization failure      //
	// ------------------------------------------------------------------ //
	t.Run("MOD-PERM-MSG-14-2-1: operator authorization failure", func(t *testing.T) {
		f := NewFixture(t)
		now := time.Date(2026, 3, 1, 0, 0, 0, 0, time.UTC)
		f.SetBlockTime(now)

		corp := sdk.AccAddress([]byte("corp_authz_scp___1")).String()
		f.SetBalance(corp, types.BondDenom, 1000)
		f.DelKeeper.ErrToReturn = fmt.Errorf("mock: not authorized")

		schemaID := uint64(1)
		f.CSKeeper.CreateMockCredentialSchema(schemaID,
			cstypes.IssuerOnboardingMode_ISSUER_ONBOARDING_MODE_OPEN,
			cstypes.VerifierOnboardingMode_VERIFIER_ONBOARDING_MODE_OPEN)

		ecoEffFrom := now.Add(-48 * time.Hour)
		ecoEffUntil := now.Add(365 * 24 * time.Hour)
		ecoID := makeEcoPerm(t, f, now, corp, schemaID, &ecoEffFrom, &ecoEffUntil)

		newEffFrom := now.Add(1 * time.Hour)
		newEffUntil := now.Add(60 * 24 * time.Hour)
		_, err := f.MS.SelfCreatePermission(f.Ctx, &types.MsgSelfCreatePermission{
			ValidatorPermId: ecoID, Type: types.PermissionType_ISSUER,
			Corporation: corp, Operator: sdk.AccAddress([]byte("bad_op___________1")).String(),
			EffectiveFrom: &newEffFrom, EffectiveUntil: &newEffUntil,
		})
		require.ErrorContains(t, err, "authorization check failed")
		f.RequireNoBalanceChange(corp)
	})

	// ------------------------------------------------------------------ //
	// Negative: [MOD-PERM-MSG-14-2-1a] validator perm not found           //
	// ------------------------------------------------------------------ //
	t.Run("MOD-PERM-MSG-14-2-1a: validator perm not found", func(t *testing.T) {
		f := NewFixture(t)
		now := time.Date(2026, 3, 1, 0, 0, 0, 0, time.UTC)
		f.SetBlockTime(now)

		corp := sdk.AccAddress([]byte("corp_novalidtr_s1")).String()
		f.SetBalance(corp, types.BondDenom, 1000)

		newEffFrom := now.Add(1 * time.Hour)
		newEffUntil := now.Add(60 * 24 * time.Hour)
		_, err := f.MS.SelfCreatePermission(f.Ctx, &types.MsgSelfCreatePermission{
			ValidatorPermId: 99999, Type: types.PermissionType_ISSUER,
			Corporation: corp, Operator: corp,
			EffectiveFrom: &newEffFrom, EffectiveUntil: &newEffUntil,
		})
		require.ErrorContains(t, err, "validator permission not found")
		f.RequireNoBalanceChange(corp)
	})

	// ------------------------------------------------------------------ //
	// Negative: [MOD-PERM-MSG-14-2-1b] validator perm not ECOSYSTEM type  //
	// ------------------------------------------------------------------ //
	t.Run("MOD-PERM-MSG-14-2-1b: validator perm is not ECOSYSTEM type", func(t *testing.T) {
		f := NewFixture(t)
		now := time.Date(2026, 3, 1, 0, 0, 0, 0, time.UTC)
		f.SetBlockTime(now)

		corp := sdk.AccAddress([]byte("corp_noteco_scp_1")).String()
		f.SetBalance(corp, types.BondDenom, 1000)

		schemaID := uint64(1)
		f.CSKeeper.CreateMockCredentialSchema(schemaID,
			cstypes.IssuerOnboardingMode_ISSUER_ONBOARDING_MODE_OPEN,
			cstypes.VerifierOnboardingMode_VERIFIER_ONBOARDING_MODE_OPEN)

		// Create an ISSUER perm instead of ECOSYSTEM
		effFrom := now.Add(-24 * time.Hour)
		effUntil := now.Add(365 * 24 * time.Hour)
		issuerPerm := types.Permission{
			SchemaId: schemaID, Type: types.PermissionType_ISSUER,
			Corporation: corp, EffectiveFrom: &effFrom, EffectiveUntil: &effUntil,
		}
		issuerID, err := f.K.CreatePermission(f.Ctx, issuerPerm)
		require.NoError(t, err)

		newEffFrom := now.Add(1 * time.Hour)
		newEffUntil := now.Add(60 * 24 * time.Hour)
		_, err = f.MS.SelfCreatePermission(f.Ctx, &types.MsgSelfCreatePermission{
			ValidatorPermId: issuerID, Type: types.PermissionType_ISSUER,
			Corporation: corp, Operator: corp,
			EffectiveFrom: &newEffFrom, EffectiveUntil: &newEffUntil,
		})
		require.ErrorContains(t, err, "ECOSYSTEM")
		f.RequireNoBalanceChange(corp)
	})

	// ------------------------------------------------------------------ //
	// Negative: [MOD-PERM-MSG-14-2-1c] validator perm is revoked          //
	// ------------------------------------------------------------------ //
	t.Run("MOD-PERM-MSG-14-2-1c: validator perm is revoked", func(t *testing.T) {
		f := NewFixture(t)
		now := time.Date(2026, 3, 1, 0, 0, 0, 0, time.UTC)
		f.SetBlockTime(now)

		corp := sdk.AccAddress([]byte("corp_revokedvld_1")).String()
		f.SetBalance(corp, types.BondDenom, 1000)

		schemaID := uint64(1)
		f.CSKeeper.CreateMockCredentialSchema(schemaID,
			cstypes.IssuerOnboardingMode_ISSUER_ONBOARDING_MODE_OPEN,
			cstypes.VerifierOnboardingMode_VERIFIER_ONBOARDING_MODE_OPEN)

		effFrom := now.Add(-48 * time.Hour)
		effUntil := now.Add(365 * 24 * time.Hour)
		revokedAt := now.Add(-1 * time.Hour)
		ecoPerm := types.Permission{
			SchemaId:       schemaID,
			Type:           types.PermissionType_ECOSYSTEM,
			Corporation:    corp,
			EffectiveFrom:  &effFrom,
			EffectiveUntil: &effUntil,
			Revoked:        &revokedAt,
		}
		ecoID, err := f.K.CreatePermission(f.Ctx, ecoPerm)
		require.NoError(t, err)

		newEffFrom := now.Add(1 * time.Hour)
		newEffUntil := now.Add(60 * 24 * time.Hour)
		_, err = f.MS.SelfCreatePermission(f.Ctx, &types.MsgSelfCreatePermission{
			ValidatorPermId: ecoID, Type: types.PermissionType_ISSUER,
			Corporation: corp, Operator: corp,
			EffectiveFrom: &newEffFrom, EffectiveUntil: &newEffUntil,
		})
		require.ErrorContains(t, err, "validator permission is revoked")
		f.RequireNoBalanceChange(corp)
	})

	// ------------------------------------------------------------------ //
	// Negative: [MOD-PERM-MSG-14-2-1d] effective_from in the past         //
	// ------------------------------------------------------------------ //
	t.Run("MOD-PERM-MSG-14-2-1d: effective_from is in the past", func(t *testing.T) {
		f := NewFixture(t)
		now := time.Date(2026, 3, 1, 0, 0, 0, 0, time.UTC)
		f.SetBlockTime(now)

		corp := sdk.AccAddress([]byte("corp_pastefrom_s1")).String()
		f.SetBalance(corp, types.BondDenom, 1000)

		schemaID := uint64(1)
		f.CSKeeper.CreateMockCredentialSchema(schemaID,
			cstypes.IssuerOnboardingMode_ISSUER_ONBOARDING_MODE_OPEN,
			cstypes.VerifierOnboardingMode_VERIFIER_ONBOARDING_MODE_OPEN)

		ecoEffFrom := now.Add(-48 * time.Hour)
		ecoEffUntil := now.Add(365 * 24 * time.Hour)
		ecoID := makeEcoPerm(t, f, now, corp, schemaID, &ecoEffFrom, &ecoEffUntil)

		pastEffFrom := now.Add(-1 * time.Hour) // must be in future
		newEffUntil := now.Add(60 * 24 * time.Hour)
		_, err := f.MS.SelfCreatePermission(f.Ctx, &types.MsgSelfCreatePermission{
			ValidatorPermId: ecoID, Type: types.PermissionType_ISSUER,
			Corporation: corp, Operator: corp,
			EffectiveFrom: &pastEffFrom, EffectiveUntil: &newEffUntil,
		})
		require.ErrorContains(t, err, "effective_from must be in the future")
		f.RequireNoBalanceChange(corp)
	})

	// ------------------------------------------------------------------ //
	// Negative: [MOD-PERM-MSG-14-2-1e] effective_until <= effective_from  //
	// ------------------------------------------------------------------ //
	t.Run("MOD-PERM-MSG-14-2-1e: effective_until is not after effective_from", func(t *testing.T) {
		f := NewFixture(t)
		now := time.Date(2026, 3, 1, 0, 0, 0, 0, time.UTC)
		f.SetBlockTime(now)

		corp := sdk.AccAddress([]byte("corp_baduntil_s_1")).String()
		f.SetBalance(corp, types.BondDenom, 1000)

		schemaID := uint64(1)
		f.CSKeeper.CreateMockCredentialSchema(schemaID,
			cstypes.IssuerOnboardingMode_ISSUER_ONBOARDING_MODE_OPEN,
			cstypes.VerifierOnboardingMode_VERIFIER_ONBOARDING_MODE_OPEN)

		ecoEffFrom := now.Add(-48 * time.Hour)
		ecoEffUntil := now.Add(365 * 24 * time.Hour)
		ecoID := makeEcoPerm(t, f, now, corp, schemaID, &ecoEffFrom, &ecoEffUntil)

		newEffFrom := now.Add(10 * time.Hour)
		badEffUntil := now.Add(5 * time.Hour) // before effective_from
		_, err := f.MS.SelfCreatePermission(f.Ctx, &types.MsgSelfCreatePermission{
			ValidatorPermId: ecoID, Type: types.PermissionType_ISSUER,
			Corporation: corp, Operator: corp,
			EffectiveFrom: &newEffFrom, EffectiveUntil: &badEffUntil,
		})
		require.ErrorContains(t, err, "effective_until must be greater than effective_from")
		f.RequireNoBalanceChange(corp)
	})

	// ------------------------------------------------------------------ //
	// Negative: [MOD-PERM-MSG-14-2-1f] verification_fees on non-ISSUER   //
	// ------------------------------------------------------------------ //
	t.Run("MOD-PERM-MSG-14-2-1f: verification_fees on VERIFIER type", func(t *testing.T) {
		f := NewFixture(t)
		now := time.Date(2026, 3, 1, 0, 0, 0, 0, time.UTC)
		f.SetBlockTime(now)

		corp := sdk.AccAddress([]byte("corp_vrffees_scp1")).String()
		f.SetBalance(corp, types.BondDenom, 1000)

		schemaID := uint64(1)
		f.CSKeeper.CreateMockCredentialSchema(schemaID,
			cstypes.IssuerOnboardingMode_ISSUER_ONBOARDING_MODE_OPEN,
			cstypes.VerifierOnboardingMode_VERIFIER_ONBOARDING_MODE_OPEN)

		ecoEffFrom := now.Add(-48 * time.Hour)
		ecoEffUntil := now.Add(365 * 24 * time.Hour)
		ecoID := makeEcoPerm(t, f, now, corp, schemaID, &ecoEffFrom, &ecoEffUntil)

		newEffFrom := now.Add(1 * time.Hour)
		newEffUntil := now.Add(60 * 24 * time.Hour)
		_, err := f.MS.SelfCreatePermission(f.Ctx, &types.MsgSelfCreatePermission{
			ValidatorPermId:  ecoID,
			Type:             types.PermissionType_VERIFIER,
			Corporation:      corp,
			Operator:         corp,
			EffectiveFrom:    &newEffFrom,
			EffectiveUntil:   &newEffUntil,
			VerificationFees: 100, // invalid for VERIFIER
		})
		require.ErrorContains(t, err, "verification_fees cannot be specified for VERIFIER permissions")
		f.RequireNoBalanceChange(corp)
	})

	// ------------------------------------------------------------------ //
	// Negative: [MOD-PERM-MSG-14-2-2] ISSUER — schema not OPEN            //
	// ------------------------------------------------------------------ //
	t.Run("MOD-PERM-MSG-14-2-2: ISSUER — IssuerOnboardingMode is not OPEN", func(t *testing.T) {
		f := NewFixture(t)
		now := time.Date(2026, 3, 1, 0, 0, 0, 0, time.UTC)
		f.SetBlockTime(now)

		corp := sdk.AccAddress([]byte("corp_notopen_scp1")).String()
		f.SetBalance(corp, types.BondDenom, 1000)

		schemaID := uint64(1)
		// Set to a non-OPEN mode (e.g., MANAGED or the zero value)
		f.CSKeeper.CreateMockCredentialSchema(schemaID,
			cstypes.IssuerOnboardingMode_ISSUER_ONBOARDING_MODE_UNSPECIFIED,
			cstypes.VerifierOnboardingMode_VERIFIER_ONBOARDING_MODE_OPEN)

		ecoEffFrom := now.Add(-48 * time.Hour)
		ecoEffUntil := now.Add(365 * 24 * time.Hour)
		ecoID := makeEcoPerm(t, f, now, corp, schemaID, &ecoEffFrom, &ecoEffUntil)

		newEffFrom := now.Add(1 * time.Hour)
		newEffUntil := now.Add(60 * 24 * time.Hour)
		_, err := f.MS.SelfCreatePermission(f.Ctx, &types.MsgSelfCreatePermission{
			ValidatorPermId: ecoID, Type: types.PermissionType_ISSUER,
			Corporation: corp, Operator: corp,
			EffectiveFrom: &newEffFrom, EffectiveUntil: &newEffUntil,
		})
		require.ErrorContains(t, err, "issuer permission management mode is not OPEN")
		f.RequireNoBalanceChange(corp)
	})

	// ------------------------------------------------------------------ //
	// Negative: [MOD-PERM-MSG-14-2-2] VERIFIER — schema not OPEN         //
	// ------------------------------------------------------------------ //
	t.Run("MOD-PERM-MSG-14-2-2: VERIFIER — VerifierOnboardingMode is not OPEN", func(t *testing.T) {
		f := NewFixture(t)
		now := time.Date(2026, 3, 1, 0, 0, 0, 0, time.UTC)
		f.SetBlockTime(now)

		corp := sdk.AccAddress([]byte("corp_notopenV_s1")).String()
		f.SetBalance(corp, types.BondDenom, 1000)

		schemaID := uint64(1)
		f.CSKeeper.CreateMockCredentialSchema(schemaID,
			cstypes.IssuerOnboardingMode_ISSUER_ONBOARDING_MODE_OPEN,
			cstypes.VerifierOnboardingMode_VERIFIER_ONBOARDING_MODE_UNSPECIFIED)

		ecoEffFrom := now.Add(-48 * time.Hour)
		ecoEffUntil := now.Add(365 * 24 * time.Hour)
		ecoID := makeEcoPerm(t, f, now, corp, schemaID, &ecoEffFrom, &ecoEffUntil)

		newEffFrom := now.Add(1 * time.Hour)
		newEffUntil := now.Add(60 * 24 * time.Hour)
		_, err := f.MS.SelfCreatePermission(f.Ctx, &types.MsgSelfCreatePermission{
			ValidatorPermId: ecoID, Type: types.PermissionType_VERIFIER,
			Corporation: corp, Operator: corp,
			EffectiveFrom: &newEffFrom, EffectiveUntil: &newEffUntil,
		})
		require.ErrorContains(t, err, "verifier permission management mode is not OPEN")
		f.RequireNoBalanceChange(corp)
	})

	// ------------------------------------------------------------------ //
	// Negative: [MOD-PERM-MSG-14-2-4] overlap check                      //
	// ------------------------------------------------------------------ //
	t.Run("MOD-PERM-MSG-14-2-4: overlap — existing perm never expires", func(t *testing.T) {
		f := NewFixture(t)
		now := time.Date(2026, 3, 1, 0, 0, 0, 0, time.UTC)
		f.SetBlockTime(now)

		corp := sdk.AccAddress([]byte("corp_overlap_scp1")).String()
		f.SetBalance(corp, types.BondDenom, 1000)

		schemaID := uint64(1)
		f.CSKeeper.CreateMockCredentialSchema(schemaID,
			cstypes.IssuerOnboardingMode_ISSUER_ONBOARDING_MODE_OPEN,
			cstypes.VerifierOnboardingMode_VERIFIER_ONBOARDING_MODE_OPEN)

		ecoEffFrom := now.Add(-48 * time.Hour)
		// nil effective_until → never expires
		ecoID := makeEcoPerm(t, f, now, corp, schemaID, &ecoEffFrom, nil)

		// Create an existing ISSUER perm under this eco with nil effective_until
		existingEffFrom := now.Add(-24 * time.Hour)
		existingPerm := types.Permission{
			SchemaId:        schemaID,
			Type:            types.PermissionType_ISSUER,
			Corporation:     corp,
			ValidatorPermId: ecoID,
			EffectiveFrom:   &existingEffFrom,
			EffectiveUntil:  nil, // never expires
		}
		_, err := f.K.CreatePermission(f.Ctx, existingPerm)
		require.NoError(t, err)

		// Try to self-create a second ISSUER perm under the same eco → overlap
		newEffFrom := now.Add(1 * time.Hour)
		newEffUntil := now.Add(60 * 24 * time.Hour)
		_, err = f.MS.SelfCreatePermission(f.Ctx, &types.MsgSelfCreatePermission{
			ValidatorPermId: ecoID, Type: types.PermissionType_ISSUER,
			Corporation: corp, Operator: corp,
			EffectiveFrom: &newEffFrom, EffectiveUntil: &newEffUntil,
		})
		require.ErrorContains(t, err, "never expires")
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
  go test ./x/perm/keeper/... -run TestMsgSelfCreatePermission -v -count=1
  ```
  Expected: all subtests PASS.

- [ ] **Step 1.4: Commit.**

  ```bash
  git add x/perm/keeper/msg_self_create_permission_test.go
  git commit -m "test(perm): add fixture-based SelfCreatePermission tests"
  ```

---

## Task 2: Delete old tests for `SelfCreatePermission`

**File:** `x/perm/keeper/msg_server_test.go`

- [ ] **Step 2.1: Delete the `TestMsgSelfCreatePermission` function block.**

  Remove the entire `func TestMsgSelfCreatePermission(t *testing.T)` block. Keep all other test functions intact.

- [ ] **Step 2.2: Verify build and test suite.**

  ```bash
  go test ./x/perm/keeper/... -count=1
  ```
  Expected: all pass.

- [ ] **Step 2.3: Commit.**

  ```bash
  git add x/perm/keeper/msg_server_test.go
  git commit -m "test(perm): delete legacy TestMsgSelfCreatePermission"
  ```

---

## Task 3: Final pass (post-last-PERM-message)

Step 31 is the last PERM message migration. After this step, the full PERM keeper coverage check is meaningful.

- [ ] **Step 3.1: Run full PERM keeper test suite.**

  ```bash
  go test ./x/perm/keeper/... -v -count=1
  ```
  Expected: PASS.

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
  Expected: coverage ≥95%. If below, run coverage profile to identify gaps:
  ```bash
  go test ./x/perm/keeper/ -coverprofile=/tmp/perm.cov && go tool cover -func=/tmp/perm.cov | grep -v "_test.go"
  ```

- [ ] **Step 3.5: Full repo sanity.**

  ```bash
  go build ./... && go test ./... -count=1
  ```

- [ ] **Step 3.6: Push and open PR.**

  ```bash
  git push -u origin test/step-31-perm-self-create-permission
  gh pr create --title "test(perm): fixture-based SelfCreatePermission tests (step 31)" --body "$(cat <<'EOF'
  ## Summary
  - Adds fixture-based TestMsgSelfCreatePermission in msg_self_create_permission_test.go
  - Covers ISSUER and VERIFIER creation, nil effective_from default, VsOperatorAuthzEnabled side-effect
  - All 11 spec preconditions covered with independent t.Run blocks
  - No bank transfers — all tests call RequireNoBalanceChange
  - specSelfCreateExpectedPerm formula function asserts full Permission struct equality
  - Deletes legacy TestMsgSelfCreatePermission from msg_server_test.go
  - Final PERM message migration — triggers full 95%+ coverage gate

  ## Test plan
  - [ ] `go test ./x/perm/keeper/... -race -count=1` passes
  - [ ] `go test ./x/perm/keeper/ -cover` reports ≥95%
  - [ ] `golangci-lint run ./x/perm/keeper/...` clean
  - [ ] `go test ./...` passes — no regressions
  EOF
  )"
  ```

---

## "Done" Criteria — Step 31

- [ ] `x/perm/keeper/msg_self_create_permission_test.go` exists.
- [ ] `specSelfCreateExpectedPerm` formula function defined and used.
- [ ] Four happy paths: ISSUER, VERIFIER, nil effective_from, VsOperatorAuthzEnabled.
- [ ] Every happy path: full struct assertion via `specSelfCreateExpectedPerm` + event + invariant + `RequireNoBalanceChange`.
- [ ] Nine negative subtests covering every precondition (authz, validator not found, not ECOSYSTEM, revoked, expired, effective_from past, effective_until bad, fees on VERIFIER, schema not OPEN, overlap).
- [ ] VsOperatorAuthzEnabled side-effect: `AddPermToVSOACalls` length and content asserted.
- [ ] Legacy `TestMsgSelfCreatePermission` deleted from `msg_server_test.go`.
- [ ] `go test ./x/perm/keeper/... -race -count=1` passes.
- [ ] Coverage ≥95% on `x/perm/keeper/`.
