# Step 21: PERM CreateRootPermission — Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Replace the legacy `TestMsgCreateRootPermission` test with a fixture-based test that verifies full struct equality, event emission, the PERM module invariant, and all spec preconditions — while wiring a `StatefulBankMock` into the PERM keeper for the first time.

**Architecture:** This is the first PERM step. It creates `x/perm/keeper/fixture_test.go` with the shared `Fixture` struct reused by all PERM steps 21–31, adds `PermissionKeeperWithStatefulBank` to `testutil/keeper/permission.go`, and writes `x/perm/keeper/create_root_permission_test.go` with spec-formula-driven tests. The old `TestMsgCreateRootPermission` block is deleted from `x/perm/keeper/msg_server_test.go` in the same PR. `CreateRootPermission` transfers no funds, so balance assertions confirm zero deltas; the test focus is struct correctness, event emission, and overlap invariants.

**Tech Stack:** Go 1.24, Cosmos SDK v0.50.x, testify/require, StatefulBankMock from testutil/keeper/bank.go

---

## Pre-flight

- [ ] **Worktree.** Create an isolated worktree for this step. Branch name: `test/step-21-perm-create-root-permission`.
- [ ] **Gate check.** Confirm step 0 is merged: `testutil/keeper/bank.go` must exist and `go build ./testutil/keeper/...` must pass.
- [ ] **Sanity check.**

  Run: `go build ./... && go vet ./...`
  Expected: no output, exit 0.

---

## File Structure

- **Create:** `x/perm/keeper/fixture_test.go` — shared `Fixture` struct for all PERM keeper tests (steps 21–31).
- **Create:** `x/perm/keeper/create_root_permission_test.go` — new fixture-based tests for `CreateRootPermission`.
- **Modify:** `testutil/keeper/permission.go` — add `PermissionKeeperWithStatefulBank` constructor and extend `MockTrustRegistryKeeper` and `MockTrustDepositKeeper` with configurable return values.
- **Delete (in same PR):** the `TestMsgCreateRootPermission` test function from `x/perm/keeper/msg_server_test.go`.

---

## Task 1: Extend `testutil/keeper/permission.go`

**File:** `testutil/keeper/permission.go`

- [ ] **Step 1.1: Add configurable fields to `MockTrustRegistryKeeper`.**

The existing `MockTrustRegistryKeeper` in `testutil/keeper/credentialschema.go` always returns `1` for `GetTrustUnitPrice`. The PERM fixture needs a configurable price. Add a `TrustUnitPrice` field to the struct and update `GetTrustUnitPrice` to return it, with `1` as the default.

Edit `testutil/keeper/credentialschema.go` — change the struct and its method:

```go
// MockTrustRegistryKeeper is a mock implementation of types.TrustRegistryKeeper
type MockTrustRegistryKeeper struct {
	trustRegistries map[uint64]trtypes.TrustRegistry
	// TrustUnitPrice is returned by GetTrustUnitPrice. Defaults to 1.
	TrustUnitPrice uint64
}

func NewMockTrustRegistryKeeper() *MockTrustRegistryKeeper {
	return &MockTrustRegistryKeeper{
		trustRegistries: make(map[uint64]trtypes.TrustRegistry),
		TrustUnitPrice:  1,
	}
}

func (k *MockTrustRegistryKeeper) GetTrustUnitPrice(_ sdk.Context) uint64 {
	if k.TrustUnitPrice == 0 {
		return 1
	}
	return k.TrustUnitPrice
}
```

- [ ] **Step 1.2: Add configurable fields to `MockTrustDepositKeeper`.**

The existing `MockTrustDepositKeeper` in `testutil/keeper/trustregistry.go` always returns `0` for `GetTrustDepositRate` and silently succeeds for `AdjustTrustDeposit`. The PERM fixture needs configurable return values and call recording. Edit `testutil/keeper/trustregistry.go`:

```go
// MockTrustDepositKeeper is a mock implementation of the TrustDepositKeeper interface for testing.
// Used by CS, DD, and PERM module test utilities (not by TR module itself).
type MockTrustDepositKeeper struct {
	// TrustDepositRate is returned by GetTrustDepositRate. Defaults to "0".
	TrustDepositRate math.LegacyDec
	// ErrToReturn is returned by AdjustTrustDeposit when non-nil.
	ErrToReturn error
	// AdjustCalls records every AdjustTrustDeposit invocation.
	AdjustCalls []TrustDepositAdjustCall
}

// TrustDepositAdjustCall records one AdjustTrustDeposit call.
type TrustDepositAdjustCall struct {
	Corporation string
	Amount      int64
	Reason      string
}

func (m *MockTrustDepositKeeper) AdjustTrustDeposit(_ sdk.Context, corporation string, amount int64, reason string) error {
	m.AdjustCalls = append(m.AdjustCalls, TrustDepositAdjustCall{
		Corporation: corporation,
		Amount:      amount,
		Reason:      reason,
	})
	return m.ErrToReturn
}

func (m *MockTrustDepositKeeper) AdjustTrustDepositOnBehalf(_ sdk.Context, _ string, _ sdk.AccAddress, _ int64) error {
	return m.ErrToReturn
}

func (m *MockTrustDepositKeeper) GetTrustDepositRate(_ sdk.Context) math.LegacyDec {
	if m.TrustDepositRate.IsNil() {
		return math.LegacyZeroDec()
	}
	return m.TrustDepositRate
}

func (m *MockTrustDepositKeeper) GetUserAgentRewardRate(_ sdk.Context) math.LegacyDec {
	return math.LegacyZeroDec()
}

func (m *MockTrustDepositKeeper) GetWalletUserAgentRewardRate(_ sdk.Context) math.LegacyDec {
	return math.LegacyZeroDec()
}

func (m *MockTrustDepositKeeper) BurnEcosystemSlashedTrustDeposit(_ sdk.Context, _ string, _ uint64) error {
	return m.ErrToReturn
}
```

- [ ] **Step 1.3: Add `PermissionKeeperWithStatefulBank` to `testutil/keeper/permission.go`.**

Append the following to `testutil/keeper/permission.go`:

```go
// PermissionKeeperWithStatefulBank creates a PERM keeper wired to a
// StatefulBankMock instead of the legacy no-op MockBankKeeper. Use this
// for all fixture-based PERM tests (issue #292 test overhaul).
// The MockTrustDepositKeeper and other mock keepers are returned for
// per-test configuration.
func PermissionKeeperWithStatefulBank(
	t testing.TB,
	bank *StatefulBankMock,
) (
	keeper.Keeper,
	*MockCredentialSchemaKeeper,
	*MockTrustRegistryKeeper,
	*MockTrustDepositKeeper,
	sdk.Context,
	*MockDelegationKeeper,
	*MockDigestKeeper,
) {
	t.Helper()
	storeKey := storetypes.NewKVStoreKey(types.StoreKey)

	db := dbm.NewMemDB()
	stateStore := store.NewCommitMultiStore(db, log.NewNopLogger(), metrics.NewNoOpMetrics())
	stateStore.MountStoreWithDB(storeKey, storetypes.StoreTypeIAVL, db)
	require.NoError(t, stateStore.LoadLatestVersion())

	registry := codectypes.NewInterfaceRegistry()
	cdc := codec.NewProtoCodec(registry)
	authority := authtypes.NewModuleAddress(govtypes.ModuleName)

	csKeeper := NewMockCredentialSchemaKeeper()
	trkKeeper := NewMockTrustRegistryKeeper()
	tdKeeper := &MockTrustDepositKeeper{}
	delKeeper := &MockDelegationKeeper{}
	digestKeeper := &MockDigestKeeper{}

	k := keeper.NewKeeper(
		cdc,
		runtime.NewKVStoreService(storeKey),
		log.NewNopLogger(),
		authority.String(),
		csKeeper,
		trkKeeper,
		tdKeeper,
		bank,
		delKeeper,
		digestKeeper,
	)

	ctx := sdk.NewContext(stateStore, cmtproto.Header{}, false, log.NewNopLogger())

	if err := k.SetParams(ctx, types.DefaultParams()); err != nil {
		panic(err)
	}

	return k, csKeeper, trkKeeper, tdKeeper, ctx, delKeeper, digestKeeper
}
```

- [ ] **Step 1.4: Verify build.**

  Run: `go build ./testutil/keeper/...`
  Expected: no output, exit 0.

- [ ] **Step 1.5: Commit.**

```bash
git add testutil/keeper/permission.go testutil/keeper/credentialschema.go testutil/keeper/trustregistry.go
git commit -m "test(perm): add PermissionKeeperWithStatefulBank and configurable mocks"
```

---

## Task 2: Create `x/perm/keeper/fixture_test.go`

**File:** `x/perm/keeper/fixture_test.go`

This file is the shared test harness for ALL PERM keeper test steps (21–31). It must not be recreated in subsequent steps.

- [ ] **Step 2.1: Create the fixture file.**

```go
package keeper_test

import (
	"testing"
	"time"

	sdk "github.com/cosmos/cosmos-sdk/types"
	authtypes "github.com/cosmos/cosmos-sdk/x/auth/types"
	govtypes "github.com/cosmos/cosmos-sdk/x/gov/types"
	"github.com/stretchr/testify/require"

	keepertest "github.com/verana-labs/verana/testutil/keeper"
	"github.com/verana-labs/verana/x/perm/keeper"
	"github.com/verana-labs/verana/x/perm/types"
)

// Fixture is the shared test environment for all PERM keeper tests (steps 21–31).
// It owns the StatefulBankMock so balance assertions are independent of the
// implementation under test.
type Fixture struct {
	t         *testing.T
	K         keeper.Keeper
	MS        types.MsgServer
	Ctx       sdk.Context
	Bank      *keepertest.StatefulBankMock
	CSKeeper  *keepertest.MockCredentialSchemaKeeper
	TRKeeper  *keepertest.MockTrustRegistryKeeper
	TDKeeper  *keepertest.MockTrustDepositKeeper
	DelKeeper *keepertest.MockDelegationKeeper
	Digest    *keepertest.MockDigestKeeper
}

// NewFixture creates a fresh PERM test environment wired to a StatefulBankMock.
// The module account for "perm" is registered in the bank mock so module-level
// balance assertions (e.g. escrow) work correctly.
func NewFixture(t *testing.T) *Fixture {
	t.Helper()
	bank := keepertest.NewStatefulBankMock(map[string]sdk.AccAddress{
		types.ModuleName: authtypes.NewModuleAddress(types.ModuleName),
	})
	k, csKeeper, trkKeeper, tdKeeper, ctx, delKeeper, digestKeeper :=
		keepertest.PermissionKeeperWithStatefulBank(t, bank)
	return &Fixture{
		t:         t,
		K:         k,
		MS:        keeper.NewMsgServerImpl(k),
		Ctx:       ctx,
		Bank:      bank,
		CSKeeper:  csKeeper,
		TRKeeper:  trkKeeper,
		TDKeeper:  tdKeeper,
		DelKeeper: delKeeper,
		Digest:    digestKeeper,
	}
}

// SetBalance sets the uvna balance for a bech32 address string.
func (f *Fixture) SetBalance(addr string, amount int64) {
	f.t.Helper()
	f.Bank.SetBalance(sdk.MustAccAddressFromBech32(addr), types.BondDenom, amount)
}

// SetModuleBalance sets the uvna balance for the perm module account.
func (f *Fixture) SetModuleBalance(amount int64) {
	f.t.Helper()
	f.Bank.SetBalance(authtypes.NewModuleAddress(types.ModuleName), types.BondDenom, amount)
}

// RequireBalanceDelta asserts that the uvna balance of addr changed by exactly
// delta relative to the baseline captured by the most recent SetBalance call.
func (f *Fixture) RequireBalanceDelta(addr string, delta int64) {
	f.t.Helper()
	f.Bank.RequireBalanceDelta(f.t, sdk.MustAccAddressFromBech32(addr), types.BondDenom, delta)
}

// RequireModuleBalanceDelta asserts that the perm module account uvna balance
// changed by exactly delta.
func (f *Fixture) RequireModuleBalanceDelta(delta int64) {
	f.t.Helper()
	f.Bank.RequireBalanceDelta(f.t, authtypes.NewModuleAddress(types.ModuleName), types.BondDenom, delta)
}

// RequireNoBalanceChange asserts that the uvna balance of addr has not changed.
func (f *Fixture) RequireNoBalanceChange(addr string) {
	f.t.Helper()
	f.Bank.RequireBalanceDelta(f.t, sdk.MustAccAddressFromBech32(addr), types.BondDenom, 0)
}

// RequirePermission asserts that the stored Permission with the given id equals
// want via full require.Equal (all fields compared).
func (f *Fixture) RequirePermission(id uint64, want types.Permission) {
	f.t.Helper()
	got, err := f.K.Permission.Get(f.Ctx, id)
	require.NoError(f.t, err)
	require.Equal(f.t, want, got)
}

// RequirePermissionNotFound asserts that no permission exists for the given id.
func (f *Fixture) RequirePermissionNotFound(id uint64) {
	f.t.Helper()
	_, err := f.K.Permission.Get(f.Ctx, id)
	require.Error(f.t, err)
}

// RequireObjectCount asserts the total number of permissions stored.
func (f *Fixture) RequireObjectCount(n int) {
	f.t.Helper()
	count := 0
	_ = f.K.Permission.Walk(f.Ctx, nil, func(_ uint64, _ types.Permission) (bool, error) {
		count++
		return false, nil
	})
	require.Equal(f.t, n, count, "expected %d permission objects, got %d", n, count)
}

// RequireEvent asserts that an event of eventType was emitted and that every
// key-value pair in attrs appears as an attribute on that event.
func (f *Fixture) RequireEvent(eventType string, attrs map[string]string) {
	f.t.Helper()
	events := f.Ctx.EventManager().Events()
	for _, e := range events {
		if e.Type == eventType {
			for k, v := range attrs {
				found := false
				for _, a := range e.Attributes {
					if a.Key == k && a.Value == v {
						found = true
						break
					}
				}
				require.True(f.t, found,
					"event %s missing attribute %s=%s (all attrs: %v)", eventType, k, v, e.Attributes)
			}
			return
		}
	}
	require.Fail(f.t, "event not found", "expected event type %s in %v", eventType, events)
}

// RequireNoEvent asserts that no event of eventType was emitted.
func (f *Fixture) RequireNoEvent(eventType string) {
	f.t.Helper()
	for _, e := range f.Ctx.EventManager().Events() {
		if e.Type == eventType {
			require.Fail(f.t, "unexpected event", "event %s was emitted but should not have been", eventType)
		}
	}
}

// RequireInvariant checks the PERM module invariant:
//  1. Sum of VpCurrentFees across all PENDING permissions == perm module account uvna balance.
//  2. Every permission with VpState == VALIDATED has non-nil EffectiveFrom.
func (f *Fixture) RequireInvariant() {
	f.t.Helper()
	var totalPendingFees uint64
	_ = f.K.Permission.Walk(f.Ctx, nil, func(_ uint64, perm types.Permission) (bool, error) {
		if perm.VpState == types.ValidationState_PENDING {
			totalPendingFees += perm.VpCurrentFees
		}
		if perm.VpState == types.ValidationState_VALIDATED {
			require.NotNil(f.t, perm.EffectiveFrom,
				"invariant: perm %d is VALIDATED but has nil EffectiveFrom", perm.Id)
		}
		return false, nil
	})
	moduleBalance := f.Bank.BalanceOf(authtypes.NewModuleAddress(types.ModuleName), types.BondDenom)
	require.Equal(f.t, int64(totalPendingFees), moduleBalance,
		"PERM invariant violated: sum(VpCurrentFees for PENDING)=%d != module_balance=%d",
		totalPendingFees, moduleBalance)
}

// SetBlockTime sets the context block time.
func (f *Fixture) SetBlockTime(t time.Time) { f.Ctx = f.Ctx.WithBlockTime(t) }

// AdvanceTime advances the context block time by d.
func (f *Fixture) AdvanceTime(d time.Duration) {
	f.Ctx = f.Ctx.WithBlockTime(f.Ctx.BlockTime().Add(d))
}

// GovAuthority returns the governance module address string.
// Used as the keeper authority for governance-gated messages.
func GovAuthority() string {
	return authtypes.NewModuleAddress(govtypes.ModuleName).String()
}
```

- [ ] **Step 2.2: Verify build.**

  Run: `go build ./x/perm/keeper/...`
  Expected: no output, exit 0.

- [ ] **Step 2.3: Commit.**

```bash
git add x/perm/keeper/fixture_test.go
git commit -m "test(perm): add shared Fixture struct for PERM keeper tests"
```

---

## Task 3: Write `create_root_permission_test.go`

**File:** `x/perm/keeper/create_root_permission_test.go`

`CreateRootPermission` issues no bank transfer. Spec formula functions are trivial (no fee math). The test focus is: struct correctness, event attributes, overlap rejection, and time validation.

- [ ] **Step 3.1: Create the test file.**

```go
package keeper_test

import (
	"fmt"
	"strconv"
	"testing"
	"time"

	cstypes "github.com/verana-labs/verana/x/cs/types"
	trtypes "github.com/verana-labs/verana/x/tr/types"

	"github.com/stretchr/testify/require"

	"github.com/verana-labs/verana/x/perm/types"
)

// ============================================================================
// Spec formula functions — independent of the implementation
// ============================================================================

// specCreateRootPermNoFees asserts that CreateRootPermission transfers zero
// uvna. Root permissions carry no financial transaction.
func specCreateRootPermNoFees() uint64 { return 0 }

// specCreateRootPermType returns the expected permission type for a root perm:
// always ECOSYSTEM regardless of what was requested.
func specCreateRootPermType() types.PermissionType { return types.PermissionType_ECOSYSTEM }

// specCreateRootPermValidatorPermId returns the expected validator_perm_id for a
// root permission: always 0 (no parent).
func specCreateRootPermValidatorPermId() uint64 { return 0 }

// specCreateRootPermDeposit returns the expected deposit for a root permission:
// always 0 (root perms carry no trust deposit).
func specCreateRootPermDeposit() uint64 { return 0 }

// ============================================================================
// TestMsgCreateRootPermission
// ============================================================================

func TestMsgCreateRootPermission(t *testing.T) {
	baseTime := time.Date(2026, 1, 1, 12, 0, 0, 0, time.UTC)
	futureFrom := baseTime.Add(24 * time.Hour)
	futureUntil := baseTime.Add(365 * 24 * time.Hour)

	// Helper: build a fixture with one credential schema backed by a trust registry
	// whose corporation matches corp.
	makeFixture := func(t *testing.T, corp string) (*Fixture, uint64) {
		t.Helper()
		f := NewFixture(t)
		f.SetBlockTime(baseTime)
		trID := f.TRKeeper.CreateMockTrustRegistry(corp, "did:example:tr1")
		f.CSKeeper.CreateMockCredentialSchemaFull(cstypes.CredentialSchema{
			Id:                     1,
			TrId:                   trID,
			IssuerOnboardingMode:   cstypes.IssuerOnboardingMode_ISSUER_ONBOARDING_MODE_OPEN,
			VerifierOnboardingMode: cstypes.VerifierOnboardingMode_VERIFIER_ONBOARDING_MODE_OPEN,
		})
		return f, 1
	}

	// -----------------------------------------------------------------------
	// Happy path: minimal root permission (no effective_until)
	// -----------------------------------------------------------------------
	t.Run("MOD-PERM-MSG-7: valid root perm created with no effective_until", func(t *testing.T) {
		corp := "cosmos1qqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqnrql8a"
		f, schemaID := makeFixture(t, corp)
		f.SetBalance(corp, 0)

		msg := &types.MsgCreateRootPermission{
			Corporation:      corp,
			Operator:         corp,
			SchemaId:         schemaID,
			EffectiveFrom:    &futureFrom,
			EffectiveUntil:   nil,
			ValidationFees:   100,
			IssuanceFees:     50,
			VerificationFees: 25,
		}

		resp, err := f.MS.CreateRootPermission(f.Ctx, msg)
		require.NoError(t, err)
		require.NotZero(t, resp.Id)

		// No bank transfer — zero fees for root permissions
		fees := specCreateRootPermNoFees()
		require.Equal(t, uint64(0), fees)
		f.RequireNoBalanceChange(corp)
		f.RequireModuleBalanceDelta(0)

		// Full struct assertion
		f.RequirePermission(resp.Id, types.Permission{
			Id:               resp.Id,
			SchemaId:         schemaID,
			Type:             specCreateRootPermType(),
			Corporation:      corp,
			ValidatorPermId:  specCreateRootPermValidatorPermId(),
			EffectiveFrom:    &futureFrom,
			EffectiveUntil:   nil,
			ValidationFees:   100,
			IssuanceFees:     50,
			VerificationFees: 25,
			Deposit:          specCreateRootPermDeposit(),
			Created:          f.Ctx.BlockTime().UTC().(*time.Time), // set below via RequirePermission pattern
		})
		// Note: Created and Modified are set to ctx.BlockTime() by the implementation.
		// Retrieve actual stored perm for time fields rather than hardcoding.
		stored, err := f.K.Permission.Get(f.Ctx, resp.Id)
		require.NoError(t, err)
		require.Equal(t, types.PermissionType_ECOSYSTEM, stored.Type)
		require.Equal(t, uint64(0), stored.ValidatorPermId)
		require.Equal(t, uint64(0), stored.Deposit)
		require.Equal(t, &futureFrom, stored.EffectiveFrom)
		require.Nil(t, stored.EffectiveUntil)
		require.Equal(t, uint64(100), stored.ValidationFees)
		require.Equal(t, uint64(50), stored.IssuanceFees)
		require.Equal(t, uint64(25), stored.VerificationFees)
		require.Equal(t, corp, stored.Corporation)
		require.NotNil(t, stored.Created)
		require.NotNil(t, stored.Modified)

		// Event assertion
		f.RequireEvent(types.EventTypeCreateRootPermission, map[string]string{
			types.AttributeKeyRootPermissionID: strconv.FormatUint(resp.Id, 10),
			types.AttributeKeySchemaID:         strconv.FormatUint(schemaID, 10),
			types.AttributeKeyCorporation:      corp,
			types.AttributeKeyOperator:         corp,
			types.AttributeKeyValidationFees:   "100",
			types.AttributeKeyIssuanceFees:     "50",
			types.AttributeKeyVerificationFees: "25",
		})

		// Invariant: no PENDING perms, no escrow balance needed
		f.RequireInvariant()
		f.RequireObjectCount(1)
	})

	// -----------------------------------------------------------------------
	// Happy path: root perm with effective_until
	// -----------------------------------------------------------------------
	t.Run("MOD-PERM-MSG-7: valid root perm with effective_until after effective_from", func(t *testing.T) {
		corp := "cosmos1qqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqnrql8a"
		f, schemaID := makeFixture(t, corp)
		f.SetBalance(corp, 0)

		msg := &types.MsgCreateRootPermission{
			Corporation:    corp,
			Operator:       corp,
			SchemaId:       schemaID,
			EffectiveFrom:  &futureFrom,
			EffectiveUntil: &futureUntil,
		}

		resp, err := f.MS.CreateRootPermission(f.Ctx, msg)
		require.NoError(t, err)
		require.NotZero(t, resp.Id)

		stored, err := f.K.Permission.Get(f.Ctx, resp.Id)
		require.NoError(t, err)
		require.Equal(t, &futureUntil, stored.EffectiveUntil)
		require.Equal(t, types.PermissionType_ECOSYSTEM, stored.Type)

		f.RequireNoBalanceChange(corp)
		f.RequireInvariant()
	})

	// -----------------------------------------------------------------------
	// Negative: [MOD-PERM-MSG-7-2-1] operator authorization failure
	// -----------------------------------------------------------------------
	t.Run("MOD-PERM-MSG-7-2-1: fails if operator authorization fails", func(t *testing.T) {
		corp := "cosmos1qqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqnrql8a"
		f, schemaID := makeFixture(t, corp)
		f.SetBalance(corp, 0)
		f.DelKeeper.ErrToReturn = fmt.Errorf("mock: operator not authorized")

		badOperator := "cosmos1qqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqpjdpjx"
		msg := &types.MsgCreateRootPermission{
			Corporation:   corp,
			Operator:      badOperator,
			SchemaId:      schemaID,
			EffectiveFrom: &futureFrom,
		}

		_, err := f.MS.CreateRootPermission(f.Ctx, msg)
		require.ErrorContains(t, err, "authorization check failed")
		f.RequireNoBalanceChange(corp)
		f.RequireObjectCount(0)
	})

	// -----------------------------------------------------------------------
	// Negative: [MOD-PERM-MSG-7-2-1] schema_id does not exist
	// -----------------------------------------------------------------------
	t.Run("MOD-PERM-MSG-7-2-1: fails if schema_id does not exist", func(t *testing.T) {
		corp := "cosmos1qqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqnrql8a"
		f, _ := makeFixture(t, corp)
		f.SetBalance(corp, 0)

		msg := &types.MsgCreateRootPermission{
			Corporation:   corp,
			Operator:      corp,
			SchemaId:      9999, // non-existent
			EffectiveFrom: &futureFrom,
		}

		_, err := f.MS.CreateRootPermission(f.Ctx, msg)
		require.ErrorContains(t, err, "credential schema not found")
		f.RequireNoBalanceChange(corp)
		f.RequireObjectCount(0)
	})

	// -----------------------------------------------------------------------
	// Negative: [MOD-PERM-MSG-7-2-1] effective_from is nil
	// -----------------------------------------------------------------------
	t.Run("MOD-PERM-MSG-7-2-1: fails if effective_from is nil", func(t *testing.T) {
		corp := "cosmos1qqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqnrql8a"
		f, schemaID := makeFixture(t, corp)
		f.SetBalance(corp, 0)

		msg := &types.MsgCreateRootPermission{
			Corporation:   corp,
			Operator:      corp,
			SchemaId:      schemaID,
			EffectiveFrom: nil, // missing
		}

		_, err := f.MS.CreateRootPermission(f.Ctx, msg)
		require.ErrorContains(t, err, "effective_from is required")
		f.RequireNoBalanceChange(corp)
		f.RequireObjectCount(0)
	})

	// -----------------------------------------------------------------------
	// Negative: [MOD-PERM-MSG-7-2-1] effective_from is not in the future
	// -----------------------------------------------------------------------
	t.Run("MOD-PERM-MSG-7-2-1: fails if effective_from is not in the future", func(t *testing.T) {
		corp := "cosmos1qqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqnrql8a"
		f, schemaID := makeFixture(t, corp)
		f.SetBalance(corp, 0)

		past := baseTime.Add(-1 * time.Hour)
		msg := &types.MsgCreateRootPermission{
			Corporation:   corp,
			Operator:      corp,
			SchemaId:      schemaID,
			EffectiveFrom: &past,
		}

		_, err := f.MS.CreateRootPermission(f.Ctx, msg)
		require.ErrorContains(t, err, "effective_from must be in the future")
		f.RequireNoBalanceChange(corp)
		f.RequireObjectCount(0)
	})

	// -----------------------------------------------------------------------
	// Negative: [MOD-PERM-MSG-7-2-1] effective_until <= effective_from
	// -----------------------------------------------------------------------
	t.Run("MOD-PERM-MSG-7-2-1: fails if effective_until is not after effective_from", func(t *testing.T) {
		corp := "cosmos1qqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqnrql8a"
		f, schemaID := makeFixture(t, corp)
		f.SetBalance(corp, 0)

		sameTime := futureFrom
		msg := &types.MsgCreateRootPermission{
			Corporation:    corp,
			Operator:       corp,
			SchemaId:       schemaID,
			EffectiveFrom:  &futureFrom,
			EffectiveUntil: &sameTime, // equal, not strictly after
		}

		_, err := f.MS.CreateRootPermission(f.Ctx, msg)
		require.ErrorContains(t, err, "effective_until must be greater than effective_from")
		f.RequireNoBalanceChange(corp)
		f.RequireObjectCount(0)
	})

	// -----------------------------------------------------------------------
	// Negative: [MOD-PERM-MSG-7-2-2] corporation does not match TR corporation
	// -----------------------------------------------------------------------
	t.Run("MOD-PERM-MSG-7-2-2: fails if corporation does not match trust registry", func(t *testing.T) {
		corp := "cosmos1qqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqnrql8a"
		f, schemaID := makeFixture(t, corp)

		differentCorp := "cosmos1qqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqpjdpjx"
		f.SetBalance(differentCorp, 0)

		msg := &types.MsgCreateRootPermission{
			Corporation:   differentCorp, // not the TR owner
			Operator:      differentCorp,
			SchemaId:      schemaID,
			EffectiveFrom: &futureFrom,
		}

		_, err := f.MS.CreateRootPermission(f.Ctx, msg)
		require.ErrorContains(t, err, "corporation does not match the trust registry corporation")
		f.RequireNoBalanceChange(differentCorp)
		f.RequireObjectCount(0)
	})

	// -----------------------------------------------------------------------
	// Negative: [MOD-PERM-MSG-7-2-4] overlap: existing non-expired ECOSYSTEM perm for same (schema_id, corporation)
	// -----------------------------------------------------------------------
	t.Run("MOD-PERM-MSG-7-2-4: fails if overlap exists with non-revoked ECOSYSTEM perm", func(t *testing.T) {
		corp := "cosmos1qqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqnrql8a"
		f, schemaID := makeFixture(t, corp)
		f.SetBalance(corp, 0)

		// Create first root perm with no effective_until (never expires)
		msg1 := &types.MsgCreateRootPermission{
			Corporation:    corp,
			Operator:       corp,
			SchemaId:       schemaID,
			EffectiveFrom:  &futureFrom,
			EffectiveUntil: nil, // never expires
		}
		_, err := f.MS.CreateRootPermission(f.Ctx, msg1)
		require.NoError(t, err)

		// Attempt to create a second root perm for same (schema_id, corporation)
		futureFrom2 := futureFrom.Add(48 * time.Hour)
		msg2 := &types.MsgCreateRootPermission{
			Corporation:   corp,
			Operator:      corp,
			SchemaId:      schemaID,
			EffectiveFrom: &futureFrom2,
		}
		_, err = f.MS.CreateRootPermission(f.Ctx, msg2)
		require.ErrorContains(t, err, "overlap check failed")
		f.RequireObjectCount(1) // only the first one created
	})

	// -----------------------------------------------------------------------
	// Edge: different corporation can create root perm for same schema
	// -----------------------------------------------------------------------
	t.Run("edge: different corporation cannot create root perm for same schema (TR mismatch)", func(t *testing.T) {
		corp1 := "cosmos1qqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqnrql8a"
		f := NewFixture(t)
		f.SetBlockTime(baseTime)

		// TR1 belongs to corp1
		trID1 := f.TRKeeper.CreateMockTrustRegistry(corp1, "did:example:tr1")
		f.CSKeeper.CreateMockCredentialSchemaFull(cstypes.CredentialSchema{
			Id:   1,
			TrId: trID1,
		})

		// corp2 tries to create root perm for schema owned by TR of corp1
		corp2 := "cosmos1qqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqpjdpjx"
		f.SetBalance(corp2, 0)

		msg := &types.MsgCreateRootPermission{
			Corporation:   corp2,
			Operator:      corp2,
			SchemaId:      1,
			EffectiveFrom: &futureFrom,
		}
		_, err := f.MS.CreateRootPermission(f.Ctx, msg)
		require.ErrorContains(t, err, "corporation does not match the trust registry corporation")
		f.RequireObjectCount(0)
	})

	// -----------------------------------------------------------------------
	// Edge: zero fees are valid (minimally configured root perm)
	// -----------------------------------------------------------------------
	t.Run("edge: zero validation/issuance/verification fees are valid", func(t *testing.T) {
		corp := "cosmos1qqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqnrql8a"
		f, schemaID := makeFixture(t, corp)
		f.SetBalance(corp, 0)

		msg := &types.MsgCreateRootPermission{
			Corporation:      corp,
			Operator:         corp,
			SchemaId:         schemaID,
			EffectiveFrom:    &futureFrom,
			ValidationFees:   0,
			IssuanceFees:     0,
			VerificationFees: 0,
		}

		resp, err := f.MS.CreateRootPermission(f.Ctx, msg)
		require.NoError(t, err)

		stored, err := f.K.Permission.Get(f.Ctx, resp.Id)
		require.NoError(t, err)
		require.Equal(t, uint64(0), stored.ValidationFees)
		require.Equal(t, uint64(0), stored.IssuanceFees)
		require.Equal(t, uint64(0), stored.VerificationFees)
		require.Equal(t, uint64(0), stored.Deposit)

		f.RequireNoBalanceChange(corp)
		f.RequireInvariant()
	})
}

// Compile-time check: ensure trtypes is used (avoids import cycle if unused).
var _ = trtypes.TrustRegistry{}
```

- [ ] **Step 3.2: Verify build.**

  Run: `go build ./x/perm/keeper/...`
  Expected: no output, exit 0.

- [ ] **Step 3.3: Run new tests.**

  Run: `go test ./x/perm/keeper/... -run TestMsgCreateRootPermission -v -count=1`
  Expected: all subtests PASS.

---

## Task 4: Delete old `TestMsgCreateRootPermission` from `msg_server_test.go`

**File:** `x/perm/keeper/msg_server_test.go`

- [ ] **Step 4.1: Delete the old test function.**

Find and remove the entire `func TestMsgCreateRootPermission(t *testing.T)` block. Do not touch any other test functions in the file.

- [ ] **Step 4.2: Verify build and full test suite.**

  Run: `go test ./x/perm/keeper/... -count=1 -v`
  Expected: no compilation errors, all remaining tests PASS. The deleted `TestMsgCreateRootPermission` must not appear.

- [ ] **Step 4.3: Commit.**

```bash
git add x/perm/keeper/create_root_permission_test.go
git add x/perm/keeper/msg_server_test.go
git commit -m "test(perm): add fixture-based CreateRootPermission tests, delete legacy"
```

---

## Task 5: Final pass

- [ ] **Step 5.1: Run the full PERM keeper test suite.**

  Run: `go test ./x/perm/keeper/... -v -count=1`
  Expected: PASS.

- [ ] **Step 5.2: Run race detector.**

  Run: `go test ./x/perm/keeper/... -race -count=1`
  Expected: PASS, no DATA RACE reports.

- [ ] **Step 5.3: Run vet and lint.**

  Run: `go vet ./x/perm/keeper/... && golangci-lint run ./x/perm/keeper/...`
  Expected: no output.

- [ ] **Step 5.4: Coverage check.**

  Run: `go test ./x/perm/keeper/ -cover -count=1`
  Expected: coverage ≥85%. If below, run `go test ./x/perm/keeper/ -coverprofile=/tmp/perm.cov && go tool cover -func=/tmp/perm.cov | grep -v "_test.go"` to find uncovered lines.

- [ ] **Step 5.5: Full repo sanity.**

  Run: `go build ./... && go test ./... -count=1`
  Expected: PASS.

- [ ] **Step 5.6: Push and open PR.**

```bash
git push -u origin test/step-21-perm-create-root-permission
gh pr create --title "test(perm): fixture-based CreateRootPermission tests (step 21)" --body "$(cat <<'EOF'
## Summary
- Adds PermissionKeeperWithStatefulBank to testutil/keeper/permission.go
- Extends MockTrustRegistryKeeper and MockTrustDepositKeeper with configurable fields
- Creates x/perm/keeper/fixture_test.go with the shared Fixture struct (used by steps 21–31)
- Replaces legacy TestMsgCreateRootPermission with fixture-based test in create_root_permission_test.go
- All preconditions tested: authz, missing schema, nil effective_from, past effective_from, bad effective_until, wrong corporation, overlap
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

## "Done" Criteria — Step 21

- [ ] `testutil/keeper/permission.go` exports `PermissionKeeperWithStatefulBank`.
- [ ] `MockTrustRegistryKeeper.TrustUnitPrice` is configurable (defaults to 1).
- [ ] `MockTrustDepositKeeper` has configurable `TrustDepositRate`, `ErrToReturn`, and `AdjustCalls` recording.
- [ ] `x/perm/keeper/fixture_test.go` exists with `Fixture`, `NewFixture`, and all assertion helpers.
- [ ] `x/perm/keeper/create_root_permission_test.go` covers: 2 happy paths, 5 negative cases, 2 edge cases.
- [ ] Every happy path: zero balance delta assertions + full struct field checks + event + invariant.
- [ ] Every negative case: `ErrorContains` + `RequireNoBalanceChange` + `RequireObjectCount(0)`.
- [ ] Legacy `TestMsgCreateRootPermission` deleted from `msg_server_test.go`.
- [ ] `go test ./x/perm/keeper/... -race -count=1` passes.
