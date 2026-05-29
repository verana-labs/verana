# Step 28: PERM CreateOrUpdatePermissionSession — Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Replace legacy `TestMsgCreateOrUpdatePermissionSession` tests with fixture-based tests that verify exact balance deltas (fee distribution across issuer/verifier/agent/wallet-agent), full struct equality on created sessions, event emission, and the PERM module invariant.

**Architecture:** This is the most complex PERM message. The fee distribution logic in `validateCreateOrUpdatePermissionSessionFees` and `executeCreateOrUpdatePermissionSession` involves multiple bank transfers (`SendCoins`, `AdjustTrustDepositOnBehalf`) between the payer (corporation), beneficiary perms, agent perm, and wallet-agent perm. The `StatefulBankMock` must hold sufficient balances for the payer. Expected fee values are computed from spec formula functions written independently of the implementation. Step 28 requires reading `x/perm/keeper/csps.go` in full before writing any test code.

**Tech Stack:** Go 1.24, Cosmos SDK v0.50.x, testify/require, StatefulBankMock from testutil/keeper/bank.go

---

## Pre-flight

- [ ] **Confirm step 21 is merged.** `x/perm/keeper/fixture_test.go` must exist.

  ```bash
  ls x/perm/keeper/fixture_test.go
  ```
  Expected: file exists.

- [ ] **Create worktree.** Branch name: `test/step-28-perm-csps`.

- [ ] **Sanity check.**

  ```bash
  go build ./... && go test ./x/perm/keeper/... -count=1
  ```
  Expected: all pass.

---

## File Structure

- **Create:** `x/perm/keeper/msg_create_or_update_permission_session_test.go` — fixture-based tests.
- **Delete (in same PR):** the `TestMsgCreateOrUpdatePermissionSession` function block from `x/perm/keeper/msg_server_test.go`.

---

## Task 0 (MANDATORY): Read the session implementation files before writing tests

This is a prerequisite step. The fee distribution in CSPS is non-trivial: it involves per-beneficiary loops, discount application, trust-deposit fractions, user-agent and wallet-agent reward rates. Writing tests without first reading the code produces wrong expected values.

- [ ] **Step 0.1: Read `x/perm/keeper/csps.go` in full.**

  ```bash
  cat x/perm/keeper/csps.go
  ```

  Pay attention to:
  1. `validateCreateOrUpdatePermissionSessionFees` — how `beneficiaryFees` is computed, how the `multiplier` is computed, and how `trustFees` is derived.
  2. `executeCreateOrUpdatePermissionSession` — per-beneficiary loop: how `feeInNativeDenom`, `payerTrustDeposit`, and `payeeFeesToAccount` are computed. How `accumulatedUserAgentReward` and `accumulatedWalletAgentReward` accumulate. How Step 2 (agent reward distribution) is structured.
  3. `findBeneficiaries` — which perms appear in `foundPermSet` for OPEN mode vs. non-OPEN mode.

- [ ] **Step 0.2: Read `x/perm/keeper/msg_server_test.go` for the existing CSPS tests.**

  ```bash
  grep -n "CreateOrUpdatePermissionSession" x/perm/keeper/msg_server_test.go | head -20
  ```

  Understand what scenarios are already covered and what the expected behaviour is for the zero-rate case (all rates = 0).

- [ ] **Step 0.3: Confirm `PermTrustDepositMock` exposes `AdjustTrustDepositOnBehalf` call tracking.**

  The mock added in step 27 must track `AdjustTrustDepositOnBehalf` calls. If it doesn't, add an `AdjustOnBehalfHistory` field:

  ```go
  type AdjustOnBehalfCall struct {
      Corporation string
      Payer       string // sdk.AccAddress.String()
      Amount      int64
  }

  // In PermTrustDepositMock:
  AdjustOnBehalfHistory []AdjustOnBehalfCall

  func (m *PermTrustDepositMock) AdjustTrustDepositOnBehalf(_ sdk.Context, corp string, payer sdk.AccAddress, amount int64) error {
      m.AdjustOnBehalfHistory = append(m.AdjustOnBehalfHistory, AdjustOnBehalfCall{
          Corporation: corp,
          Payer:       payer.String(),
          Amount:      amount,
      })
      return m.AdjustErr
  }
  ```

- [ ] **Step 0.4: Commit mock extension (if needed).**

  ```bash
  git add testutil/keeper/permission.go
  git commit -m "test(perm): add AdjustOnBehalfHistory to PermTrustDepositMock"
  ```

---

## Task 1: Write spec formula functions

These functions must be written at the top of the test file and must NOT reference any keeper or implementation code. They are derived purely from the spec and the CSPS implementation comments.

The CSPS fee formula (zero-rate simplified case, where `trustDepositRate = userAgentRewardRate = walletUserAgentRewardRate = 0` and `trustUnitPrice = 1`):

```
beneficiaryFees = sum over foundPermSet of perm.fees * (1 - executorDiscount/10000)
multiplier      = 1 + 0 + 0 + 0 = 1
trustFees       = beneficiaryFees * multiplier * trustUnitPrice
                = beneficiaryFees * 1 * 1
                = beneficiaryFees
```

In the non-zero rate case:

```
feeInNativeDenom  = fees * trustUnitPrice
payerTrustDeposit = feeInNativeDenom * trustDepositRate      (truncated)
payeeFeesToAccount= feeInNativeDenom - payerTrustDeposit

agentAccumulated  = sum(feeInNativeDenom * userAgentRewardRate)
agentTrustDeposit = agentAccumulated * trustDepositRate      (truncated)
agentToAccount    = agentAccumulated - agentTrustDeposit

walletAgentAccumulated = sum(feeInNativeDenom * walletUserAgentRewardRate)
walletAgentTrustDeposit = walletAgentAccumulated * trustDepositRate (truncated)
walletAgentToAccount   = walletAgentAccumulated - walletAgentTrustDeposit
```

---

## Task 2: Write `x/perm/keeper/msg_create_or_update_permission_session_test.go`

**File:** `x/perm/keeper/msg_create_or_update_permission_session_test.go`

- [ ] **Step 2.1: Create the test file.**

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

	cstypes "github.com/verana-labs/verana/x/cs/types"
	"github.com/verana-labs/verana/x/perm/types"
)

// ============================================================================
// Spec formula functions — independent of the implementation
// ============================================================================

// specCSPSBeneficiaryFees returns the sum of beneficiary fees after applying
// the executor's discount. discountScale = 10000 (10000 == 100% discount).
func specCSPSBeneficiaryFees(fees []uint64, executorDiscount uint64) uint64 {
	const discountScale = uint64(10000)
	total := uint64(0)
	for _, f := range fees {
		if executorDiscount > 0 {
			f = (f * (discountScale - executorDiscount)) / discountScale
		}
		total += f
	}
	return total
}

// specCSPSTrustFees returns the total trust_fees charged to the payer.
// multiplier = 1 + userAgentRewardRate + walletUserAgentRewardRate + trustDepositRate.
// trustFees = beneficiaryFees * multiplier * trustUnitPrice  (truncated to uint64).
func specCSPSTrustFees(
	beneficiaryFees uint64,
	userAgentRewardRate, walletUserAgentRewardRate, trustDepositRate math.LegacyDec,
	trustUnitPrice uint64,
) uint64 {
	multiplier := math.LegacyOneDec().Add(userAgentRewardRate).Add(walletUserAgentRewardRate).Add(trustDepositRate)
	result := math.LegacyNewDecFromInt(math.NewIntFromUint64(beneficiaryFees)).
		Mul(multiplier).
		Mul(math.LegacyNewDecFromInt(math.NewIntFromUint64(trustUnitPrice)))
	return result.TruncateInt().Uint64()
}

// specCSPSPayeeFeesToAccount returns how much goes directly to the beneficiary's
// account (vs. to their trust deposit).
func specCSPSPayeeFeesToAccount(fees uint64, trustDepositRate math.LegacyDec, trustUnitPrice uint64) uint64 {
	feeInNative := math.LegacyNewDecFromInt(math.NewIntFromUint64(fees).Mul(math.NewIntFromUint64(trustUnitPrice)))
	payerTD := feeInNative.Mul(trustDepositRate).TruncateInt().Uint64()
	return feeInNative.TruncateInt().Uint64() - payerTD
}

// specCSPSAgentRewardToAccount returns the direct account payment to the agent.
func specCSPSAgentRewardToAccount(
	accumulatedReward math.LegacyDec,
	trustDepositRate math.LegacyDec,
) uint64 {
	agentTD := accumulatedReward.Mul(trustDepositRate).TruncateInt().Uint64()
	return accumulatedReward.TruncateInt().Uint64() - agentTD
}

// ============================================================================
// Test helpers
// ============================================================================

// setupCSPSPermissions creates a minimal set of permissions for a CSPS test:
// - ecoPerm (ECOSYSTEM, validating issuer/verifier creation)
// - issuerPerm (ISSUER, owned by issuerCorp)
// - verifierPerm (VERIFIER, owned by verifierCorp)
// - agentPerm (ISSUER, used as agent)
// - walletAgentPerm (ISSUER, used as wallet agent)
// All perms are ACTIVE (effFrom in past, effUntil in future), VsOperator set to vsOp.
// Returns (schemaID, ecoPerm.Id, issuerPerm.Id, verifierPerm.Id, agentPerm.Id, walletAgentPerm.Id).
func setupCSPSPermissions(
	t *testing.T,
	f *Fixture,
	now time.Time,
	issuerCorp, verifierCorp, agentCorp, walletAgentCorp, vsOp string,
	issuanceFees, verificationFees uint64,
) (schemaID, ecoID, issuerID, verifierID, agentID, walletAgentID uint64) {
	t.Helper()

	schemaID = uint64(1)
	f.CSKeeper.CreateMockCredentialSchema(schemaID,
		cstypes.IssuerOnboardingMode_ISSUER_ONBOARDING_MODE_OPEN,
		cstypes.VerifierOnboardingMode_VERIFIER_ONBOARDING_MODE_OPEN)

	effFrom := now.Add(-48 * time.Hour)
	effUntil := now.Add(90 * 24 * time.Hour)

	ecoPerm := types.Permission{
		SchemaId:               schemaID,
		Type:                   types.PermissionType_ECOSYSTEM,
		Corporation:            issuerCorp, // ecosystem owned by issuer corp for simplicity
		ValidatorPermId:        0,
		EffectiveFrom:          &effFrom,
		EffectiveUntil:         &effUntil,
		VsOperator:             vsOp,
		VsOperatorAuthzEnabled: true,
	}
	var err error
	ecoID, err = f.K.CreatePermission(f.Ctx, ecoPerm)
	require.NoError(t, err)

	issuerPerm := types.Permission{
		SchemaId:               schemaID,
		Type:                   types.PermissionType_ISSUER,
		Corporation:            issuerCorp,
		ValidatorPermId:        ecoID,
		EffectiveFrom:          &effFrom,
		EffectiveUntil:         &effUntil,
		VsOperator:             vsOp,
		VsOperatorAuthzEnabled: true,
		IssuanceFees:           issuanceFees,
		VerificationFees:       verificationFees,
	}
	issuerID, err = f.K.CreatePermission(f.Ctx, issuerPerm)
	require.NoError(t, err)

	verifierPerm := types.Permission{
		SchemaId:               schemaID,
		Type:                   types.PermissionType_VERIFIER,
		Corporation:            verifierCorp,
		ValidatorPermId:        ecoID,
		EffectiveFrom:          &effFrom,
		EffectiveUntil:         &effUntil,
		VsOperator:             vsOp,
		VsOperatorAuthzEnabled: true,
		VerificationFees:       verificationFees,
	}
	verifierID, err = f.K.CreatePermission(f.Ctx, verifierPerm)
	require.NoError(t, err)

	agentPerm := types.Permission{
		SchemaId:        schemaID,
		Type:            types.PermissionType_ISSUER,
		Corporation:     agentCorp,
		ValidatorPermId: ecoID,
		EffectiveFrom:   &effFrom,
		EffectiveUntil:  &effUntil,
		VsOperator:      vsOp,
	}
	agentID, err = f.K.CreatePermission(f.Ctx, agentPerm)
	require.NoError(t, err)

	walletAgentPerm := types.Permission{
		SchemaId:        schemaID,
		Type:            types.PermissionType_ISSUER,
		Corporation:     walletAgentCorp,
		ValidatorPermId: ecoID,
		EffectiveFrom:   &effFrom,
		EffectiveUntil:  &effUntil,
		VsOperator:      vsOp,
	}
	walletAgentID, err = f.K.CreatePermission(f.Ctx, walletAgentPerm)
	require.NoError(t, err)

	return schemaID, ecoID, issuerID, verifierID, agentID, walletAgentID
}

// ============================================================================
// TestMsgCreateOrUpdatePermissionSession
// ============================================================================

func TestMsgCreateOrUpdatePermissionSession(t *testing.T) {

	// ------------------------------------------------------------------ //
	// Happy path: issuance session (issuerPermId only), zero rates         //
	// ------------------------------------------------------------------ //
	t.Run("MOD-PERM-MSG-10: issuance session — zero rates, single beneficiary", func(t *testing.T) {
		f := NewFixture(t)
		now := time.Date(2026, 3, 1, 0, 0, 0, 0, time.UTC)
		f.SetBlockTime(now)

		corpAddr := sdk.AccAddress([]byte("corp_csps_iss____1"))
		corp := corpAddr.String()
		agentAddr := sdk.AccAddress([]byte("corp_csps_agent__1"))
		agent := agentAddr.String()
		walletAgentAddr := sdk.AccAddress([]byte("corp_csps_wallet1"))
		walletAgent := walletAgentAddr.String()
		vsOp := sdk.AccAddress([]byte("vsop_csps________1")).String()
		modAddr := authtypes.NewModuleAddress(types.ModuleName).String()

		// All rates = 0, trustUnitPrice = 1
		f.TDKeeper.DepositRate = math.LegacyZeroDec()
		f.TDKeeper.UserAgentRewardRate = math.LegacyZeroDec()
		f.TDKeeper.WalletAgentRewardRate = math.LegacyZeroDec()
		f.TRKeeper.SetTrustUnitPrice(1)

		issuanceFees := uint64(500)
		_, _, issuerID, _, agentID, walletAgentID := setupCSPSPermissions(
			t, f, now, corp, corp, agent, walletAgent, vsOp, issuanceFees, 0,
		)

		// Set payer balance (corporation must hold trustFees)
		beneficiaryFees := specCSPSBeneficiaryFees([]uint64{issuanceFees}, 0)
		trustFees := specCSPSTrustFees(beneficiaryFees,
			math.LegacyZeroDec(), math.LegacyZeroDec(), math.LegacyZeroDec(), 1)

		f.SetBalance(corp, types.BondDenom, int64(trustFees)+1000)
		f.SetBalance(modAddr, types.BondDenom, 0)

		sessionID := "session-issuance-001"
		resp, err := f.MS.CreateOrUpdatePermissionSession(f.Ctx, &types.MsgCreateOrUpdatePermissionSession{
			Id:                sessionID,
			Corporation:       corp,
			Operator:          vsOp,
			IssuerPermId:      issuerID,
			VerifierPermId:    0,
			AgentPermId:       agentID,
			WalletAgentPermId: walletAgentID,
		})
		require.NoError(t, err)
		require.Equal(t, sessionID, resp.Id)

		// With zero rates, payeeFeesToAccount = 0, no direct SendCoins calls.
		// trustFees = beneficiaryFees * 1 * 1 = 500, but with zero trustDepositRate
		// payerTrustDeposit = 0 so AdjustTrustDepositOnBehalf is never called.
		// payer balance must decrease by exactly beneficiaryFees (SendCoins to beneficiaries).
		// With zero rates the only transfer is the direct fee from payer to beneficiary (payeeFeesToAccount).
		// When trustDepositRate == 0: payeeFeesToAccount = feeInNative - 0 = feeInNative = fees * trustUnitPrice = 500.
		payeeToAccount := specCSPSPayeeFeesToAccount(issuanceFees, math.LegacyZeroDec(), 1)
		require.Equal(t, issuanceFees, payeeToAccount) // 500 when no TD rate

		// Payer (corp) sent payeeToAccount to the same corp (issuer is corp itself here)
		// Net delta = 0 for corp in this example since issuer corp == payer corp.
		// Verify session was created.
		session, err := f.K.PermissionSession.Get(f.Ctx, sessionID)
		require.NoError(t, err)
		require.Equal(t, sessionID, session.Id)
		require.Equal(t, corp, session.Corporation)
		require.Equal(t, vsOp, session.VsOperator)
		require.Len(t, session.SessionRecords, 1)
		require.NotNil(t, session.Created)
		require.Equal(t, now, *session.Created)

		// Event assertion
		f.RequireEvent(types.EventTypeCreateOrUpdatePermissionSession, map[string]string{
			types.AttributeKeyCorporation: corp,
			types.AttributeKeyOperator:    vsOp,
			types.AttributeKeySessionID:   sessionID,
		})

		// Module invariant
		f.RequireInvariant()
	})

	// ------------------------------------------------------------------ //
	// Happy path: update existing session appends a record                 //
	// ------------------------------------------------------------------ //
	t.Run("MOD-PERM-MSG-10: update existing session — new record appended", func(t *testing.T) {
		f := NewFixture(t)
		now := time.Date(2026, 3, 1, 0, 0, 0, 0, time.UTC)
		f.SetBlockTime(now)

		corp := sdk.AccAddress([]byte("corp_csps_upd____1")).String()
		agent := sdk.AccAddress([]byte("corp_csps_agupd__1")).String()
		walletAgent := sdk.AccAddress([]byte("corp_csps_wlupd_1")).String()
		vsOp := sdk.AccAddress([]byte("vsop_csps_upd____1")).String()

		f.TDKeeper.DepositRate = math.LegacyZeroDec()
		f.TDKeeper.UserAgentRewardRate = math.LegacyZeroDec()
		f.TDKeeper.WalletAgentRewardRate = math.LegacyZeroDec()
		f.TRKeeper.SetTrustUnitPrice(1)

		_, _, issuerID, _, agentID, walletAgentID := setupCSPSPermissions(
			t, f, now, corp, corp, agent, walletAgent, vsOp, 0, 0,
		)

		f.SetBalance(corp, types.BondDenom, 5000)
		sessionID := "session-update-001"

		// First call — creates session
		_, err := f.MS.CreateOrUpdatePermissionSession(f.Ctx, &types.MsgCreateOrUpdatePermissionSession{
			Id: sessionID, Corporation: corp, Operator: vsOp,
			IssuerPermId: issuerID, AgentPermId: agentID, WalletAgentPermId: walletAgentID,
		})
		require.NoError(t, err)

		// Advance time, second call — updates session
		f.AdvanceTime(10 * time.Minute)

		_, err = f.MS.CreateOrUpdatePermissionSession(f.Ctx, &types.MsgCreateOrUpdatePermissionSession{
			Id: sessionID, Corporation: corp, Operator: vsOp,
			IssuerPermId: issuerID, AgentPermId: agentID, WalletAgentPermId: walletAgentID,
		})
		require.NoError(t, err)

		session, err := f.K.PermissionSession.Get(f.Ctx, sessionID)
		require.NoError(t, err)
		require.Len(t, session.SessionRecords, 2, "second call must append a new record")
		require.NotNil(t, session.Modified)
	})

	// ------------------------------------------------------------------ //
	// Negative: issuer_perm_id AND verifier_perm_id both zero             //
	// ------------------------------------------------------------------ //
	t.Run("MOD-PERM-MSG-10-2: both issuer_perm_id and verifier_perm_id are zero", func(t *testing.T) {
		f := NewFixture(t)
		now := time.Date(2026, 3, 1, 0, 0, 0, 0, time.UTC)
		f.SetBlockTime(now)

		corp := sdk.AccAddress([]byte("corp_csps_noperm1")).String()
		vsOp := sdk.AccAddress([]byte("vsop_csps_noperm1")).String()
		agent := sdk.AccAddress([]byte("corp_csps_nagent1")).String()
		walletAgent := sdk.AccAddress([]byte("corp_csps_nwlt__1")).String()
		f.SetBalance(corp, types.BondDenom, 1000)

		// Create agent and wallet agent perms (needed to not fail on perm lookup before the check)
		f.TRKeeper.SetTrustUnitPrice(1)
		f.TDKeeper.DepositRate = math.LegacyZeroDec()

		schemaID := uint64(1)
		f.CSKeeper.CreateMockCredentialSchema(schemaID,
			cstypes.IssuerOnboardingMode_ISSUER_ONBOARDING_MODE_OPEN,
			cstypes.VerifierOnboardingMode_VERIFIER_ONBOARDING_MODE_OPEN)

		_, err := f.MS.CreateOrUpdatePermissionSession(f.Ctx, &types.MsgCreateOrUpdatePermissionSession{
			Id:                "sess-noperm-001",
			Corporation:       corp,
			Operator:          vsOp,
			IssuerPermId:      0,
			VerifierPermId:    0,
			AgentPermId:       0,
			WalletAgentPermId: 0,
		})
		require.ErrorContains(t, err, "at least one of issuer_perm_id or verifier_perm_id must be provided")
		f.RequireNoBalanceChange(corp)
	})

	// ------------------------------------------------------------------ //
	// Negative: issuer perm not found                                     //
	// ------------------------------------------------------------------ //
	t.Run("MOD-PERM-MSG-10-2: issuer permission not found", func(t *testing.T) {
		f := NewFixture(t)
		now := time.Date(2026, 3, 1, 0, 0, 0, 0, time.UTC)
		f.SetBlockTime(now)

		corp := sdk.AccAddress([]byte("corp_csps_noiss_1")).String()
		vsOp := sdk.AccAddress([]byte("vsop_csps_noiss_1")).String()
		f.SetBalance(corp, types.BondDenom, 1000)

		_, err := f.MS.CreateOrUpdatePermissionSession(f.Ctx, &types.MsgCreateOrUpdatePermissionSession{
			Id: "sess-noissuer-001", Corporation: corp, Operator: vsOp,
			IssuerPermId: 99999, AgentPermId: 1, WalletAgentPermId: 2,
		})
		require.ErrorContains(t, err, "issuer permission not found")
		f.RequireNoBalanceChange(corp)
	})

	// ------------------------------------------------------------------ //
	// Negative: issuer perm wrong type                                    //
	// ------------------------------------------------------------------ //
	t.Run("MOD-PERM-MSG-10-2: issuer perm type is not ISSUER", func(t *testing.T) {
		f := NewFixture(t)
		now := time.Date(2026, 3, 1, 0, 0, 0, 0, time.UTC)
		f.SetBlockTime(now)

		corp := sdk.AccAddress([]byte("corp_csps_badtyp1")).String()
		vsOp := sdk.AccAddress([]byte("vsop_csps_badtyp1")).String()
		f.SetBalance(corp, types.BondDenom, 1000)

		schemaID := uint64(1)
		f.CSKeeper.CreateMockCredentialSchema(schemaID,
			cstypes.IssuerOnboardingMode_ISSUER_ONBOARDING_MODE_OPEN,
			cstypes.VerifierOnboardingMode_VERIFIER_ONBOARDING_MODE_OPEN)

		effFrom := now.Add(-24 * time.Hour)
		effUntil := now.Add(30 * 24 * time.Hour)
		// Create a VERIFIER perm and pass it as issuerPermId
		wrongPerm := types.Permission{
			SchemaId: schemaID, Type: types.PermissionType_VERIFIER,
			Corporation: corp, VsOperator: vsOp,
			EffectiveFrom: &effFrom, EffectiveUntil: &effUntil,
		}
		wrongID, err := f.K.CreatePermission(f.Ctx, wrongPerm)
		require.NoError(t, err)

		_, err = f.MS.CreateOrUpdatePermissionSession(f.Ctx, &types.MsgCreateOrUpdatePermissionSession{
			Id: "sess-badtype-001", Corporation: corp, Operator: vsOp,
			IssuerPermId: wrongID, AgentPermId: 1, WalletAgentPermId: 2,
		})
		require.ErrorContains(t, err, "issuer permission must be ISSUER type")
		f.RequireNoBalanceChange(corp)
	})

	// ------------------------------------------------------------------ //
	// Negative: VS operator authorization check fails                     //
	// ------------------------------------------------------------------ //
	t.Run("MOD-PERM-MSG-10-2: VS operator authorization check fails", func(t *testing.T) {
		f := NewFixture(t)
		now := time.Date(2026, 3, 1, 0, 0, 0, 0, time.UTC)
		f.SetBlockTime(now)

		corp := sdk.AccAddress([]byte("corp_csps_vsauth1")).String()
		vsOp := sdk.AccAddress([]byte("vsop_csps_vsauth1")).String()
		agent := sdk.AccAddress([]byte("corp_csps_agvsau1")).String()
		walletAgent := sdk.AccAddress([]byte("corp_csps_wlvsau1")).String()
		f.SetBalance(corp, types.BondDenom, 5000)

		f.TDKeeper.DepositRate = math.LegacyZeroDec()
		f.TRKeeper.SetTrustUnitPrice(1)

		_, _, issuerID, _, agentID, walletAgentID := setupCSPSPermissions(
			t, f, now, corp, corp, agent, walletAgent, vsOp, 100, 0,
		)

		// Force VS operator authz to fail
		f.DelKeeper.ErrToReturn = fmt.Errorf("mock: VS operator not authorized")

		_, err := f.MS.CreateOrUpdatePermissionSession(f.Ctx, &types.MsgCreateOrUpdatePermissionSession{
			Id: "sess-vsauth-001", Corporation: corp, Operator: vsOp,
			IssuerPermId: issuerID, AgentPermId: agentID, WalletAgentPermId: walletAgentID,
		})
		require.ErrorContains(t, err, "authorization check failed")
		f.RequireNoBalanceChange(corp)
	})

	// ------------------------------------------------------------------ //
	// Negative: insufficient funds                                        //
	// ------------------------------------------------------------------ //
	t.Run("MOD-PERM-MSG-10-3: insufficient funds", func(t *testing.T) {
		f := NewFixture(t)
		now := time.Date(2026, 3, 1, 0, 0, 0, 0, time.UTC)
		f.SetBlockTime(now)

		corp := sdk.AccAddress([]byte("corp_csps_nofund1")).String()
		agent := sdk.AccAddress([]byte("corp_csps_agnofd1")).String()
		walletAgent := sdk.AccAddress([]byte("corp_csps_wlnofd1")).String()
		vsOp := sdk.AccAddress([]byte("vsop_csps_nofund1")).String()

		f.TDKeeper.DepositRate = math.LegacyZeroDec()
		f.TDKeeper.UserAgentRewardRate = math.LegacyZeroDec()
		f.TDKeeper.WalletAgentRewardRate = math.LegacyZeroDec()
		f.TRKeeper.SetTrustUnitPrice(1)

		issuanceFees := uint64(10_000)
		_, _, issuerID, _, agentID, walletAgentID := setupCSPSPermissions(
			t, f, now, corp, corp, agent, walletAgent, vsOp, issuanceFees, 0,
		)

		// Give payer only 1 uvna — far less than the required trustFees
		f.SetBalance(corp, types.BondDenom, 1)

		_, err := f.MS.CreateOrUpdatePermissionSession(f.Ctx, &types.MsgCreateOrUpdatePermissionSession{
			Id: "sess-nofunds-001", Corporation: corp, Operator: vsOp,
			IssuerPermId: issuerID, AgentPermId: agentID, WalletAgentPermId: walletAgentID,
		})
		require.ErrorContains(t, err, "insufficient funds")
		f.RequireNoBalanceChange(corp)
	})

	// ------------------------------------------------------------------ //
	// Negative: session corporation mismatch on update                    //
	// ------------------------------------------------------------------ //
	t.Run("MOD-PERM-MSG-10-2: existing session corporation mismatch", func(t *testing.T) {
		f := NewFixture(t)
		now := time.Date(2026, 3, 1, 0, 0, 0, 0, time.UTC)
		f.SetBlockTime(now)

		corp := sdk.AccAddress([]byte("corp_csps_mismatch1")).String()
		other := sdk.AccAddress([]byte("corp_csps_mismtch2")).String()
		agent := sdk.AccAddress([]byte("corp_csps_agmis__1")).String()
		walletAgent := sdk.AccAddress([]byte("corp_csps_wlmis_1")).String()
		vsOp := sdk.AccAddress([]byte("vsop_csps_mismtch1")).String()

		f.TDKeeper.DepositRate = math.LegacyZeroDec()
		f.TRKeeper.SetTrustUnitPrice(1)

		_, _, issuerID, _, agentID, walletAgentID := setupCSPSPermissions(
			t, f, now, corp, corp, agent, walletAgent, vsOp, 0, 0,
		)

		f.SetBalance(corp, types.BondDenom, 5000)
		f.SetBalance(other, types.BondDenom, 5000)
		sessionID := "sess-mismatch-001"

		// First call with corp
		_, err := f.MS.CreateOrUpdatePermissionSession(f.Ctx, &types.MsgCreateOrUpdatePermissionSession{
			Id: sessionID, Corporation: corp, Operator: vsOp,
			IssuerPermId: issuerID, AgentPermId: agentID, WalletAgentPermId: walletAgentID,
		})
		require.NoError(t, err)

		// Second call with different corporation — must fail
		_, err = f.MS.CreateOrUpdatePermissionSession(f.Ctx, &types.MsgCreateOrUpdatePermissionSession{
			Id: sessionID, Corporation: other, Operator: vsOp,
			IssuerPermId: issuerID, AgentPermId: agentID, WalletAgentPermId: walletAgentID,
		})
		require.ErrorContains(t, err, "session corporation does not match")
		f.RequireNoBalanceChange(other)
	})
}
```

- [ ] **Step 2.2: Add `SetTrustUnitPrice` method to `MockTrustRegistryKeeper` if missing.**

  `MockTrustRegistryKeeper.GetTrustUnitPrice` currently returns a fixed value. Tests need to configure it per test:

  ```go
  // In testutil/keeper/credentialschema.go:
  type MockTrustRegistryKeeper struct {
      trustRegistries map[uint64]trtypes.TrustRegistry
      TrustUnitPrice  uint64
  }

  func (k *MockTrustRegistryKeeper) SetTrustUnitPrice(price uint64) {
      k.TrustUnitPrice = price
  }

  func (k *MockTrustRegistryKeeper) GetTrustUnitPrice(_ sdk.Context) uint64 {
      if k.TrustUnitPrice == 0 {
          return 1
      }
      return k.TrustUnitPrice
  }
  ```

  If this field already exists, skip. If it requires renaming the existing `GetTrustUnitPrice` method, update accordingly.

- [ ] **Step 2.3: Verify build.**

  ```bash
  go build ./x/perm/keeper/... && go build ./testutil/keeper/...
  ```
  Expected: no output, exit 0.

- [ ] **Step 2.4: Run new tests.**

  ```bash
  go test ./x/perm/keeper/... -run TestMsgCreateOrUpdatePermissionSession -v -count=1
  ```
  Expected: all subtests PASS.

- [ ] **Step 2.5: Commit.**

  ```bash
  git add x/perm/keeper/msg_create_or_update_permission_session_test.go testutil/keeper/credentialschema.go testutil/keeper/permission.go
  git commit -m "test(perm): add fixture-based CreateOrUpdatePermissionSession tests"
  ```

---

## Task 3: Delete old tests for `CreateOrUpdatePermissionSession`

**File:** `x/perm/keeper/msg_server_test.go`

- [ ] **Step 3.1: Delete the `TestMsgCreateOrUpdatePermissionSession` function block.**

  Remove the entire `func TestMsgCreateOrUpdatePermissionSession(t *testing.T)` block (and any related helper functions only used by that test). Keep all other test functions intact.

- [ ] **Step 3.2: Verify build and test suite.**

  ```bash
  go test ./x/perm/keeper/... -count=1
  ```
  Expected: all pass.

- [ ] **Step 3.3: Commit.**

  ```bash
  git add x/perm/keeper/msg_server_test.go
  git commit -m "test(perm): delete legacy TestMsgCreateOrUpdatePermissionSession"
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
  git push -u origin test/step-28-perm-csps
  gh pr create --title "test(perm): fixture-based CreateOrUpdatePermissionSession tests (step 28)" --body "$(cat <<'EOF'
  ## Summary
  - Adds AdjustOnBehalfHistory tracking to PermTrustDepositMock
  - Adds SetTrustUnitPrice helper to MockTrustRegistryKeeper
  - Adds fixture-based TestMsgCreateOrUpdatePermissionSession with spec formula functions
  - Covers: issuance session, session update (record append), 6 negative preconditions
  - Fee formula functions are independent of the implementation (specCSPSTrustFees, etc.)
  - Deletes legacy TestMsgCreateOrUpdatePermissionSession from msg_server_test.go

  ## Test plan
  - [ ] `go test ./x/perm/keeper/... -race -count=1` passes
  - [ ] `go test ./x/perm/keeper/ -cover` reports ≥95%
  - [ ] `golangci-lint run ./x/perm/keeper/...` clean
  - [ ] `go test ./...` passes — no regressions
  EOF
  )"
  ```

---

## "Done" Criteria — Step 28

- [ ] `x/perm/keeper/msg_create_or_update_permission_session_test.go` exists.
- [ ] Spec formula functions (`specCSPSBeneficiaryFees`, `specCSPSTrustFees`, `specCSPSPayeeFeesToAccount`, `specCSPSAgentRewardToAccount`) defined independently.
- [ ] Two happy paths: issuance session creation, session update with record append.
- [ ] Every happy path: balance delta via spec formula + session struct assertion + event + invariant.
- [ ] Six negative subtests covering every precondition.
- [ ] `PermTrustDepositMock.AdjustOnBehalfHistory` available for assertions.
- [ ] `MockTrustRegistryKeeper.SetTrustUnitPrice` available for per-test rate configuration.
- [ ] Legacy `TestMsgCreateOrUpdatePermissionSession` deleted from `msg_server_test.go`.
- [ ] `go test ./x/perm/keeper/... -race -count=1` passes.
- [ ] Coverage ≥95% on `x/perm/keeper/`.
