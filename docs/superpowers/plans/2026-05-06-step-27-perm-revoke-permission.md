# Step 27: PERM RevokePermission — Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Replace legacy `TestMsgRevokePermission` tests with fixture-based tests that verify full struct equality, the trust-deposit release call, event emission, the PERM module invariant, and every spec precondition — including the three authority options and the deposit-release path.

**Architecture:** Step 21 created `x/perm/keeper/fixture_test.go`; this step only adds `x/perm/keeper/msg_revoke_permission_test.go` and deletes the legacy test block. `RevokePermission` calls `trustDeposit.AdjustTrustDeposit` but makes no direct bank transfer — `f.RequireNoBalanceChange` is called on the corporation address in every test. The `MockTrustDepositKeeper.AdjustHistory` field (added in the Fixture) is used to assert the deposit-release call.

**Tech Stack:** Go 1.24, Cosmos SDK v0.50.x, testify/require, StatefulBankMock from testutil/keeper/bank.go

---

## Pre-flight

- [ ] **Confirm step 21 is merged.** `x/perm/keeper/fixture_test.go` must exist.

  ```bash
  ls x/perm/keeper/fixture_test.go
  ```
  Expected: file exists.

- [ ] **Create worktree.** Branch name: `test/step-27-perm-revoke-permission`.

- [ ] **Sanity check.**

  ```bash
  go build ./... && go test ./x/perm/keeper/... -count=1
  ```
  Expected: all pass.

---

## File Structure

- **Create:** `x/perm/keeper/msg_revoke_permission_test.go` — fixture-based tests.
- **Delete (in same PR):** the `TestMsgRevokePermission` function block from `x/perm/keeper/msg_server_test.go`.

---

## Task 1: Extend `MockTrustDepositKeeper` with call tracking for `AdjustTrustDeposit`

The existing `MockTrustDepositKeeper` in `testutil/keeper/trustregistry.go` is a no-op. To assert the deposit-release call, we need the Fixture's `TDKeeper` to record `AdjustTrustDeposit` invocations.

The `Fixture` struct (created in step 21) already holds a `*PermTrustDepositMock` (the stateful version wired in `PermissionKeeperWithStatefulBank`). Confirm the Fixture exposes `TDKeeper` with `AdjustHistory`. If it does not yet have `AdjustHistory`, add it.

- [ ] **Step 1.1: Confirm (or add) `AdjustHistory` to the perm-module trust-deposit mock.**

The mock used inside the PERM fixture must record calls:

```go
// In testutil/keeper/permission.go (or the perm fixture's inline mock):

type AdjustCall struct {
    Corporation string
    Amount      int64
    Reason      string
}

type PermTrustDepositMock struct {
    AdjustErr     error
    AdjustHistory []AdjustCall
    BurnErr       error
    BurnHistory   []BurnCall
    // Configurable rates (default: 0)
    DepositRate            math.LegacyDec
    UserAgentRewardRate    math.LegacyDec
    WalletAgentRewardRate  math.LegacyDec
}

type BurnCall struct {
    Corporation string
    Amount      uint64
}

func (m *PermTrustDepositMock) AdjustTrustDeposit(_ sdk.Context, corp string, amount int64, reason string) error {
    m.AdjustHistory = append(m.AdjustHistory, AdjustCall{Corporation: corp, Amount: amount, Reason: reason})
    return m.AdjustErr
}

func (m *PermTrustDepositMock) AdjustTrustDepositOnBehalf(_ sdk.Context, _ string, _ sdk.AccAddress, _ int64) error {
    return m.AdjustErr
}

func (m *PermTrustDepositMock) GetTrustDepositRate(_ sdk.Context) math.LegacyDec {
    if m.DepositRate.IsNil() {
        return math.LegacyZeroDec()
    }
    return m.DepositRate
}

func (m *PermTrustDepositMock) GetUserAgentRewardRate(_ sdk.Context) math.LegacyDec {
    if m.UserAgentRewardRate.IsNil() {
        return math.LegacyZeroDec()
    }
    return m.UserAgentRewardRate
}

func (m *PermTrustDepositMock) GetWalletUserAgentRewardRate(_ sdk.Context) math.LegacyDec {
    if m.WalletAgentRewardRate.IsNil() {
        return math.LegacyZeroDec()
    }
    return m.WalletAgentRewardRate
}

func (m *PermTrustDepositMock) BurnEcosystemSlashedTrustDeposit(_ sdk.Context, corp string, amount uint64) error {
    m.BurnHistory = append(m.BurnHistory, BurnCall{Corporation: corp, Amount: amount})
    return m.BurnErr
}
```

If the `Fixture` struct in `x/perm/keeper/fixture_test.go` exposes `TDKeeper *PermTrustDepositMock`, this is already wired. Otherwise update the fixture to expose it.

- [ ] **Step 1.2: Verify build.**

  ```bash
  go build ./x/perm/keeper/... && go build ./testutil/keeper/...
  ```
  Expected: no output, exit 0.

- [ ] **Step 1.3: Commit (if changes were needed).**

  ```bash
  git add testutil/keeper/permission.go x/perm/keeper/fixture_test.go
  git commit -m "test(perm): add AdjustHistory tracking to PermTrustDepositMock"
  ```

---

## Task 2: Write `x/perm/keeper/msg_revoke_permission_test.go`

**File:** `x/perm/keeper/msg_revoke_permission_test.go`

- [ ] **Step 2.1: Create the test file.**

```go
package keeper_test

import (
	"fmt"
	"testing"
	"time"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/stretchr/testify/require"

	cstypes "github.com/verana-labs/verana/x/cs/types"
	trtypes "github.com/verana-labs/verana/x/tr/types"
	"github.com/verana-labs/verana/x/perm/types"
)

// ============================================================================
// Spec formula functions — independent of the implementation
// ============================================================================

// specRevokeDepositAfter returns the perm.Deposit after a successful revoke.
// Per [MOD-PERM-MSG-9-3]: deposit is zeroed on revoke.
func specRevokeDepositAfter() uint64 { return 0 }

// specRevokeAdjustCallAmount returns the expected amount passed to
// AdjustTrustDeposit on revoke (negative — deposit is released).
func specRevokeAdjustCallAmount(depositBefore uint64) int64 { return -int64(depositBefore) }

// ============================================================================
// TestMsgRevokePermission
// ============================================================================

func TestMsgRevokePermission(t *testing.T) {

	// ------------------------------------------------------------------ //
	// Happy path 1: revoke own ECOSYSTEM perm (Option #3 — own authority) //
	// ------------------------------------------------------------------ //
	t.Run("MOD-PERM-MSG-9-2-2 option3: perm.Corporation == msg.Corporation", func(t *testing.T) {
		f := NewFixture(t)
		now := time.Date(2026, 3, 1, 0, 0, 0, 0, time.UTC)
		f.SetBlockTime(now)

		corp := sdk.AccAddress([]byte("corp_revoke_own__1")).String()
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
			Deposit:        0, // no deposit for ECOSYSTEM in this test
		}
		id, err := f.K.CreatePermission(f.Ctx, perm)
		require.NoError(t, err)

		_, err = f.MS.RevokePermission(f.Ctx, &types.MsgRevokePermission{
			Id:          id,
			Corporation: corp,
			Operator:    corp,
		})
		require.NoError(t, err)

		// Full struct assertion
		got, err := f.K.GetPermissionByID(f.Ctx, id)
		require.NoError(t, err)
		require.NotNil(t, got.Revoked)
		require.Equal(t, now, *got.Revoked)
		require.NotNil(t, got.Modified)
		require.Equal(t, now, *got.Modified)
		require.Equal(t, specRevokeDepositAfter(), got.Deposit)

		// No direct bank transfer
		f.RequireNoBalanceChange(corp)

		// Event assertion
		f.RequireEvent(types.EventTypeRevokePermission, map[string]string{
			types.AttributeKeyPermissionID: fmt.Sprintf("%d", id),
			types.AttributeKeyCorporation:  corp,
			types.AttributeKeyRevokedAt:    now.String(),
		})

		// Module invariant
		f.RequireInvariant()
	})

	// ------------------------------------------------------------------ //
	// Happy path 2: revoke with deposit > 0 — AdjustTrustDeposit called   //
	// ------------------------------------------------------------------ //
	t.Run("MOD-PERM-MSG-9-3: deposit > 0 — AdjustTrustDeposit called with negative amount", func(t *testing.T) {
		f := NewFixture(t)
		now := time.Date(2026, 3, 1, 0, 0, 0, 0, time.UTC)
		f.SetBlockTime(now)

		corp := sdk.AccAddress([]byte("corp_revoke_dep__1")).String()
		f.SetBalance(corp, types.BondDenom, 5000)

		schemaID := uint64(1)
		f.CSKeeper.CreateMockCredentialSchema(schemaID,
			cstypes.IssuerOnboardingMode_ISSUER_ONBOARDING_MODE_OPEN,
			cstypes.VerifierOnboardingMode_VERIFIER_ONBOARDING_MODE_OPEN)

		effFrom := now.Add(-24 * time.Hour)
		effUntil := now.Add(30 * 24 * time.Hour)
		depositBefore := uint64(2000)
		perm := types.Permission{
			SchemaId:       schemaID,
			Type:           types.PermissionType_ECOSYSTEM,
			Corporation:    corp,
			EffectiveFrom:  &effFrom,
			EffectiveUntil: &effUntil,
			Deposit:        depositBefore,
		}
		id, err := f.K.CreatePermission(f.Ctx, perm)
		require.NoError(t, err)

		_, err = f.MS.RevokePermission(f.Ctx, &types.MsgRevokePermission{
			Id:          id,
			Corporation: corp,
			Operator:    corp,
		})
		require.NoError(t, err)

		got, err := f.K.GetPermissionByID(f.Ctx, id)
		require.NoError(t, err)
		require.Equal(t, specRevokeDepositAfter(), got.Deposit)
		require.NotNil(t, got.Revoked)

		// Trust deposit mock must have been called with the correct release amount
		require.Len(t, f.TDKeeper.AdjustHistory, 1)
		require.Equal(t, corp, f.TDKeeper.AdjustHistory[0].Corporation)
		require.Equal(t, specRevokeAdjustCallAmount(depositBefore), f.TDKeeper.AdjustHistory[0].Amount)
		require.Equal(t, "perm_revoke_release_deposit", f.TDKeeper.AdjustHistory[0].Reason)

		f.RequireNoBalanceChange(corp)
		f.RequireInvariant()
	})

	// ------------------------------------------------------------------ //
	// Happy path 3: Option #1 — validator ancestor authorized              //
	// ------------------------------------------------------------------ //
	t.Run("MOD-PERM-MSG-9-2-2 option1: validator ancestor authorized", func(t *testing.T) {
		f := NewFixture(t)
		now := time.Date(2026, 3, 1, 0, 0, 0, 0, time.UTC)
		f.SetBlockTime(now)

		vpCorp := sdk.AccAddress([]byte("corp_vp_revoke___1")).String()
		subCorp := sdk.AccAddress([]byte("corp_sub_revoke__1")).String()
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
		subPerm := types.Permission{
			SchemaId:        schemaID,
			Type:            types.PermissionType_ISSUER,
			Corporation:     subCorp,
			ValidatorPermId: vpID,
			EffectiveFrom:   &subEffFrom,
			EffectiveUntil:  &subEffUntil,
		}
		subID, err := f.K.CreatePermission(f.Ctx, subPerm)
		require.NoError(t, err)

		// vpCorp (ancestor) revokes subCorp's perm
		_, err = f.MS.RevokePermission(f.Ctx, &types.MsgRevokePermission{
			Id:          subID,
			Corporation: vpCorp,
			Operator:    vpCorp,
		})
		require.NoError(t, err)

		got, err := f.K.GetPermissionByID(f.Ctx, subID)
		require.NoError(t, err)
		require.NotNil(t, got.Revoked)

		f.RequireNoBalanceChange(vpCorp)
		f.RequireNoBalanceChange(subCorp)
		f.RequireInvariant()
	})

	// ------------------------------------------------------------------ //
	// Happy path 4: Option #2 — TR controller authorized                  //
	// ------------------------------------------------------------------ //
	t.Run("MOD-PERM-MSG-9-2-2 option2: TR controller authorized", func(t *testing.T) {
		f := NewFixture(t)
		now := time.Date(2026, 3, 1, 0, 0, 0, 0, time.UTC)
		f.SetBlockTime(now)

		trCorp := sdk.AccAddress([]byte("corp_tr_revoke___1")).String()
		subCorp := sdk.AccAddress([]byte("corp_sub_revoktr1")).String()
		f.SetBalance(trCorp, types.BondDenom, 1000)
		f.SetBalance(subCorp, types.BondDenom, 1000)

		// Create TR and CS so the keeper can resolve the controller
		trID := f.TRKeeper.CreateMockTrustRegistry(trCorp, "did:example:tr1")

		schemaID := uint64(1)
		f.CSKeeper.CreateMockCredentialSchemaFull(cstypes.CredentialSchema{
			Id:                     schemaID,
			TrId:                   trID,
			IssuerOnboardingMode:   cstypes.IssuerOnboardingMode_ISSUER_ONBOARDING_MODE_OPEN,
			VerifierOnboardingMode: cstypes.VerifierOnboardingMode_VERIFIER_ONBOARDING_MODE_OPEN,
		})

		effFrom := now.Add(-24 * time.Hour)
		effUntil := now.Add(30 * 24 * time.Hour)
		perm := types.Permission{
			SchemaId:        schemaID,
			Type:            types.PermissionType_ISSUER,
			Corporation:     subCorp,
			ValidatorPermId: 0, // ECOSYSTEM-ish, no validator ancestor
			EffectiveFrom:   &effFrom,
			EffectiveUntil:  &effUntil,
		}
		permID, err := f.K.CreatePermission(f.Ctx, perm)
		require.NoError(t, err)

		// TR controller revokes
		_, err = f.MS.RevokePermission(f.Ctx, &types.MsgRevokePermission{
			Id:          permID,
			Corporation: trCorp,
			Operator:    trCorp,
		})
		require.NoError(t, err)

		got, err := f.K.GetPermissionByID(f.Ctx, permID)
		require.NoError(t, err)
		require.NotNil(t, got.Revoked)

		f.RequireNoBalanceChange(trCorp)
		f.RequireInvariant()
	})

	// ------------------------------------------------------------------ //
	// Negative: [MOD-PERM-MSG-9-2-1] operator authorization failure       //
	// ------------------------------------------------------------------ //
	t.Run("MOD-PERM-MSG-9-2-1: operator authorization failure", func(t *testing.T) {
		f := NewFixture(t)
		now := time.Date(2026, 3, 1, 0, 0, 0, 0, time.UTC)
		f.SetBlockTime(now)

		corp := sdk.AccAddress([]byte("corp_authz_rev___1")).String()
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

		_, err = f.MS.RevokePermission(f.Ctx, &types.MsgRevokePermission{
			Id:          id,
			Corporation: corp,
			Operator:    sdk.AccAddress([]byte("bad_operator_____1")).String(),
		})
		require.ErrorContains(t, err, "authorization check failed")
		f.RequireNoBalanceChange(corp)
	})

	// ------------------------------------------------------------------ //
	// Negative: [MOD-PERM-MSG-9-2-1a] perm not found                     //
	// ------------------------------------------------------------------ //
	t.Run("MOD-PERM-MSG-9-2-1a: perm not found", func(t *testing.T) {
		f := NewFixture(t)
		corp := sdk.AccAddress([]byte("corp_notfound_rev1")).String()
		f.SetBalance(corp, types.BondDenom, 1000)

		_, err := f.MS.RevokePermission(f.Ctx, &types.MsgRevokePermission{
			Id:          99999,
			Corporation: corp,
			Operator:    corp,
		})
		require.ErrorContains(t, err, "permission not found")
		f.RequireNoBalanceChange(corp)
	})

	// ------------------------------------------------------------------ //
	// Negative: [MOD-PERM-MSG-9-2-1b] perm already revoked (not active)  //
	// ------------------------------------------------------------------ //
	t.Run("MOD-PERM-MSG-9-2-1b: perm is not active", func(t *testing.T) {
		f := NewFixture(t)
		now := time.Date(2026, 3, 1, 0, 0, 0, 0, time.UTC)
		f.SetBlockTime(now)

		corp := sdk.AccAddress([]byte("corp_inactive_rv1")).String()
		f.SetBalance(corp, types.BondDenom, 1000)

		schemaID := uint64(1)
		f.CSKeeper.CreateMockCredentialSchema(schemaID,
			cstypes.IssuerOnboardingMode_ISSUER_ONBOARDING_MODE_OPEN,
			cstypes.VerifierOnboardingMode_VERIFIER_ONBOARDING_MODE_OPEN)

		effFrom := now.Add(-24 * time.Hour)
		effUntil := now.Add(30 * 24 * time.Hour)
		alreadyRevoked := now.Add(-1 * time.Hour)
		perm := types.Permission{
			SchemaId:       schemaID,
			Type:           types.PermissionType_ECOSYSTEM,
			Corporation:    corp,
			EffectiveFrom:  &effFrom,
			EffectiveUntil: &effUntil,
			Revoked:        &alreadyRevoked,
		}
		id, err := f.K.CreatePermission(f.Ctx, perm)
		require.NoError(t, err)

		_, err = f.MS.RevokePermission(f.Ctx, &types.MsgRevokePermission{
			Id:          id,
			Corporation: corp,
			Operator:    corp,
		})
		require.ErrorContains(t, err, "applicant permission is not active")
		f.RequireNoBalanceChange(corp)
	})

	// ------------------------------------------------------------------ //
	// Negative: [MOD-PERM-MSG-9-2-2] no authority option matches         //
	// ------------------------------------------------------------------ //
	t.Run("MOD-PERM-MSG-9-2-2: none of the three authority options match", func(t *testing.T) {
		f := NewFixture(t)
		now := time.Date(2026, 3, 1, 0, 0, 0, 0, time.UTC)
		f.SetBlockTime(now)

		corp := sdk.AccAddress([]byte("corp_auth_fail_rv1")).String()
		stranger := sdk.AccAddress([]byte("corp_stranger_rv1")).String()
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

		_, err = f.MS.RevokePermission(f.Ctx, &types.MsgRevokePermission{
			Id:          id,
			Corporation: stranger, // not the owner, not a validator ancestor, not TR controller
			Operator:    stranger,
		})
		require.ErrorContains(t, err, "authority is not authorized to revoke this permission")
		f.RequireNoBalanceChange(corp)
	})

	// ------------------------------------------------------------------ //
	// Edge: ISSUER perm — revokeVSOperatorAuthorization called            //
	// ------------------------------------------------------------------ //
	t.Run("edge: ISSUER perm with VsOperator — RemovePermFromVSOA called", func(t *testing.T) {
		f := NewFixture(t)
		now := time.Date(2026, 3, 1, 0, 0, 0, 0, time.UTC)
		f.SetBlockTime(now)

		corp := sdk.AccAddress([]byte("corp_vsoa_revoke1")).String()
		vsOp := sdk.AccAddress([]byte("vsop_revoke______1")).String()
		f.SetBalance(corp, types.BondDenom, 1000)

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
			VsOperator:     vsOp,
		}
		id, err := f.K.CreatePermission(f.Ctx, perm)
		require.NoError(t, err)

		_, err = f.MS.RevokePermission(f.Ctx, &types.MsgRevokePermission{
			Id:          id,
			Corporation: corp,
			Operator:    corp,
		})
		require.NoError(t, err)

		require.Len(t, f.DelKeeper.RemovePermFromVSOACalls, 1)
		require.Equal(t, corp, f.DelKeeper.RemovePermFromVSOACalls[0].Authority)
		require.Equal(t, vsOp, f.DelKeeper.RemovePermFromVSOACalls[0].VsOperator)
		require.Equal(t, id, f.DelKeeper.RemovePermFromVSOACalls[0].PermID)

		f.RequireNoBalanceChange(corp)
		f.RequireInvariant()
	})
}

// Compile-time import guard: trtypes must be importable even if only used in
// CreateMockTrustRegistry calls. Keep this blank identifier to prevent
// "imported and not used" compile errors if the happy-path-4 test is removed.
var _ = trtypes.TrustRegistry{}
```

- [ ] **Step 2.2: Verify build.**

  ```bash
  go build ./x/perm/keeper/...
  ```
  Expected: no output, exit 0.

- [ ] **Step 2.3: Run new tests.**

  ```bash
  go test ./x/perm/keeper/... -run TestMsgRevokePermission -v -count=1
  ```
  Expected: all subtests PASS.

- [ ] **Step 2.4: Commit.**

  ```bash
  git add x/perm/keeper/msg_revoke_permission_test.go
  git commit -m "test(perm): add fixture-based RevokePermission tests"
  ```

---

## Task 3: Delete old tests for `RevokePermission`

**File:** `x/perm/keeper/msg_server_test.go`

- [ ] **Step 3.1: Delete the `TestMsgRevokePermission` function block.**

  Remove the entire `func TestMsgRevokePermission(t *testing.T)` block from `x/perm/keeper/msg_server_test.go`. Keep all other test functions intact.

- [ ] **Step 3.2: Verify build and test suite.**

  ```bash
  go test ./x/perm/keeper/... -count=1
  ```
  Expected: all pass.

- [ ] **Step 3.3: Commit.**

  ```bash
  git add x/perm/keeper/msg_server_test.go
  git commit -m "test(perm): delete legacy TestMsgRevokePermission"
  ```

---

## Task 4: Final pass

- [ ] **Step 4.1: Run full PERM keeper test suite.**

  ```bash
  go test ./x/perm/keeper/... -v -count=1
  ```
  Expected: PASS.

- [ ] **Step 4.2: Race detector.**

  ```bash
  go test ./x/perm/keeper/... -race -count=1
  ```
  Expected: PASS, no DATA RACE.

- [ ] **Step 4.3: Vet and lint.**

  ```bash
  go vet ./x/perm/keeper/... && golangci-lint run ./x/perm/keeper/...
  ```
  Expected: no output.

- [ ] **Step 4.4: Coverage check.**

  ```bash
  go test ./x/perm/keeper/ -cover -count=1
  ```
  Expected: coverage ≥95%.

- [ ] **Step 4.5: Full repo sanity.**

  ```bash
  go build ./... && go test ./... -count=1
  ```
  Expected: PASS.

- [ ] **Step 4.6: Push and open PR.**

  ```bash
  git push -u origin test/step-27-perm-revoke-permission
  gh pr create --title "test(perm): fixture-based RevokePermission tests (step 27)" --body "$(cat <<'EOF'
  ## Summary
  - Extends PermTrustDepositMock with AdjustHistory and BurnHistory call tracking
  - Adds fixture-based TestMsgRevokePermission in msg_revoke_permission_test.go
  - Covers all three authority options (own corp, validator ancestor, TR controller)
  - Deposit release path: asserts AdjustTrustDeposit called with correct negative amount
  - ISSUER perm side-effect: asserts RemovePermFromVSOA called on delegation keeper
  - All tests call RequireNoBalanceChange (no direct bank transfer in RevokePermission)
  - Deletes legacy TestMsgRevokePermission from msg_server_test.go

  ## Test plan
  - [ ] `go test ./x/perm/keeper/... -race -count=1` passes
  - [ ] `go test ./x/perm/keeper/ -cover` reports ≥95%
  - [ ] `golangci-lint run ./x/perm/keeper/...` clean
  - [ ] `go test ./...` passes — no regressions
  EOF
  )"
  ```

---

## "Done" Criteria — Step 27

- [ ] `x/perm/keeper/msg_revoke_permission_test.go` exists with `TestMsgRevokePermission`.
- [ ] `PermTrustDepositMock` exposes `AdjustHistory []AdjustCall`.
- [ ] Four happy paths: own corp, validator ancestor, TR controller, with-deposit.
- [ ] Every happy path: full struct assertion (Revoked, Modified, Deposit==0) + event + invariant + `RequireNoBalanceChange`.
- [ ] Deposit path: `TDKeeper.AdjustHistory[0].Amount == -depositBefore` asserted.
- [ ] Five negative subtests covering every precondition.
- [ ] ISSUER edge case: `RemovePermFromVSOACalls` length and content asserted.
- [ ] Legacy `TestMsgRevokePermission` deleted from `msg_server_test.go`.
- [ ] `go test ./x/perm/keeper/... -race -count=1` passes.
- [ ] Coverage ≥95% on `x/perm/keeper/`.
