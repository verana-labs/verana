# Step 17: DE GrantOperatorAuthorization — Fixture-Based Unit Tests

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Replace the existing table-driven `TestMsgServerGrantOperatorAuthorization` test (and related tests) in `x/de/keeper/msg_server_test.go` with a fixture-based test suite in `x/de/keeper/msg_grant_operator_authorization_test.go`. Tests follow the precondition-matrix pattern from the design spec (issue #292).

**Branch name:** `test/step-17-de-grant-operator-authz`

**Architecture:**
- New `x/de/keeper/fixture_test.go` — `Fixture` struct with assertion helpers (no bank mock needed; DE has no BankKeeper).
- New `x/de/keeper/msg_grant_operator_authorization_test.go` — precondition matrix tests + happy path with full struct assertion + event assertion + invariant.
- **Delete:** `TestMsgServerGrantOperatorAuthorization`, `TestMsgServerGrantOperatorAuthorization_WithFeegrant`, `TestMsgServerGrantOperatorAuthorization_UpdateExisting`, `TestGrantThenRevokeOperatorAuthorization_E2E`, `TestMutualExclusivity_OAAndVSOA`, `TestMultipleGranteesForSameAuthority`, `TestGrantRevokeReGrant_E2E`, `TestAuthorityIsolation`, `TestOperatorCannotSelfGrant`, `TestBootstrapFlow_GroupProposalOnboardsFirstOperator`, `TestGrantOperatorAuthorization_FeegrantFieldsStoredCorrectly` from `msg_server_test.go` in the same PR. The tests for `GrantFeeAllowance`, `RevokeFeeAllowance`, `CheckOperatorAuthorization`, `RevokeOperatorAuthorization`, and query handlers remain until their respective steps.

---

## Pre-flight

- [ ] **Worktree.** Create an isolated worktree. Branch name: `test/step-17-de-grant-operator-authz`.
- [ ] **Sanity check current tree builds and tests pass.**

  ```bash
  go build ./... && go test ./x/de/keeper/... -count=1
  ```
  Expected: all pass.

---

## File Structure

- **Create:** `x/de/keeper/fixture_test.go` — `Fixture` struct and all assertion helpers.
- **Create:** `x/de/keeper/msg_grant_operator_authorization_test.go` — `TestMsgGrantOperatorAuthorization`.
- **Modify:** `x/de/keeper/msg_server_test.go` — delete the GrantOperatorAuthorization-specific test functions listed in the Architecture section.

---

## Task 1: Create `x/de/keeper/fixture_test.go`

The DE keeper constructor signature is:
```go
keeper.NewKeeper(storeService, cdc, addressCodec, authority)
```
No bank keeper. No mock keepers needed for DE unit tests (DE has no external keeper dependencies at the keeper level).

The existing `initFixture` in `keeper_test.go` uses `moduletestutil.MakeTestEncodingConfig` and `testutil.DefaultContextWithDB`. The new `Fixture` wraps the same pattern and adds assertion helpers.

- [ ] **Step 1.1: Create `x/de/keeper/fixture_test.go`.**

```go
package keeper_test

import (
	"testing"
	"time"

	"cosmossdk.io/collections"
	"cosmossdk.io/core/address"
	storetypes "cosmossdk.io/store/types"
	addresscodec "github.com/cosmos/cosmos-sdk/codec/address"
	"github.com/cosmos/cosmos-sdk/runtime"
	sdktestutil "github.com/cosmos/cosmos-sdk/testutil"
	sdk "github.com/cosmos/cosmos-sdk/types"
	moduletestutil "github.com/cosmos/cosmos-sdk/types/module/testutil"
	authtypes "github.com/cosmos/cosmos-sdk/x/auth/types"
	"github.com/stretchr/testify/require"

	"github.com/verana-labs/verana/x/de/keeper"
	module "github.com/verana-labs/verana/x/de/module"
	"github.com/verana-labs/verana/x/de/types"
)

// Fixture is the per-test environment for DE keeper tests.
// It owns the keeper, msg server, and SDK context; exposes typed assertion helpers.
// The DE keeper has no BankKeeper dependency, so no StatefulBankMock is needed.
type Fixture struct {
	t    *testing.T
	K    keeper.Keeper
	MS   types.MsgServer
	Ctx  sdk.Context
	AddrCodec address.Codec
}

// NewFixture creates a fresh Fixture with an isolated in-memory store.
func NewFixture(t *testing.T) *Fixture {
	t.Helper()

	encCfg := moduletestutil.MakeTestEncodingConfig(module.AppModule{})
	addrCodec := addresscodec.NewBech32Codec(sdk.GetConfig().GetBech32AccountAddrPrefix())
	storeKey := storetypes.NewKVStoreKey(types.StoreKey)

	storeService := runtime.NewKVStoreService(storeKey)
	ctx := sdktestutil.DefaultContextWithDB(t, storeKey, storetypes.NewTransientStoreKey("transient_test")).Ctx

	authority := authtypes.NewModuleAddress(types.GovModuleName)

	k := keeper.NewKeeper(
		storeService,
		encCfg.Codec,
		addrCodec,
		authority,
	)

	if err := k.Params.Set(ctx, types.DefaultParams()); err != nil {
		t.Fatalf("failed to set params: %v", err)
	}

	return &Fixture{
		t:         t,
		K:         k,
		MS:        keeper.NewMsgServerImpl(k),
		Ctx:       ctx,
		AddrCodec: addrCodec,
	}
}

// SetBlockTime sets the block time on the context.
func (f *Fixture) SetBlockTime(ts time.Time) {
	f.Ctx = f.Ctx.WithBlockTime(ts)
}

// AdvanceTime advances the block time by d.
func (f *Fixture) AdvanceTime(d time.Duration) {
	f.Ctx = f.Ctx.WithBlockTime(f.Ctx.BlockTime().Add(d))
}

// RequireOperatorAuth asserts that an OperatorAuthorization exists for
// (corp, grantee) and that it deep-equals want.
func (f *Fixture) RequireOperatorAuth(corp, grantee string, want types.OperatorAuthorization) {
	f.t.Helper()
	key := collections.Join(corp, grantee)
	got, err := f.K.OperatorAuthorizations.Get(f.Ctx, key)
	require.NoError(f.t, err, "OperatorAuthorization(%s, %s) not found", corp, grantee)
	require.Equal(f.t, want, got)
}

// RequireNoOperatorAuth asserts that no OperatorAuthorization exists for (corp, grantee).
func (f *Fixture) RequireNoOperatorAuth(corp, grantee string) {
	f.t.Helper()
	key := collections.Join(corp, grantee)
	has, err := f.K.OperatorAuthorizations.Has(f.Ctx, key)
	require.NoError(f.t, err)
	require.False(f.t, has, "expected no OperatorAuthorization for (%s, %s) but one exists", corp, grantee)
}

// RequireEvent asserts that an event of the given type was emitted on f.Ctx
// and that it contains all key=value attributes in attrs.
func (f *Fixture) RequireEvent(eventType string, attrs map[string]string) {
	f.t.Helper()
	events := f.Ctx.EventManager().Events()
	for _, e := range events {
		if e.Type != eventType {
			continue
		}
		for wantKey, wantVal := range attrs {
			found := false
			for _, a := range e.Attributes {
				if a.Key == wantKey && a.Value == wantVal {
					found = true
					break
				}
			}
			require.True(f.t, found,
				"event %q missing attribute %s=%s", eventType, wantKey, wantVal)
		}
		return // event type found; all required attrs matched
	}
	require.Fail(f.t, "event not found", "event type %q was not emitted; got events: %v", eventType, events)
}

// RequireNoEvent asserts that no event of eventType was emitted.
func (f *Fixture) RequireNoEvent(eventType string) {
	f.t.Helper()
	for _, e := range f.Ctx.EventManager().Events() {
		require.NotEqual(f.t, eventType, e.Type,
			"expected event %q NOT to be emitted but it was", eventType)
	}
}

// RequireOperatorAuthCount asserts the total number of OperatorAuthorization entries.
func (f *Fixture) RequireOperatorAuthCount(n int) {
	f.t.Helper()
	count := 0
	err := f.K.OperatorAuthorizations.Walk(f.Ctx, nil, func(_ collections.Pair[string, string], _ types.OperatorAuthorization) (bool, error) {
		count++
		return false, nil
	})
	require.NoError(f.t, err)
	require.Equal(f.t, n, count, "expected %d OperatorAuthorization entries, got %d", n, count)
}

// RequireInvariant checks the DE module invariant:
// every FeeGrant entry references an existing OperatorAuthorization
// (i.e., no orphan fee grants after a grant or revoke operation).
func (f *Fixture) RequireInvariant() {
	f.t.Helper()
	err := f.K.FeeGrants.Walk(f.Ctx, nil, func(key collections.Pair[string, string], fg types.FeeGrant) (bool, error) {
		corp, grantee := key.K1(), key.K2()
		oaKey := collections.Join(corp, grantee)
		has, err := f.K.OperatorAuthorizations.Has(f.Ctx, oaKey)
		if err != nil {
			return true, err
		}
		require.True(f.t, has,
			"DE invariant violated: FeeGrant(%s, %s) exists but no OperatorAuthorization for the same pair",
			corp, grantee)
		_ = fg
		return false, nil
	})
	require.NoError(f.t, err)
}
```

- [ ] **Step 1.2: Verify compilation.**

  ```bash
  go build ./x/de/keeper/...
  ```
  Expected: no errors.

- [ ] **Step 1.3: Commit.**

  ```bash
  git add x/de/keeper/fixture_test.go
  git commit -m "test(de): add Fixture struct with assertion helpers"
  ```

---

## Task 2: Create `x/de/keeper/msg_grant_operator_authorization_test.go`

**Preconditions for `GrantOperatorAuthorization` (from implementation):**

| ID | Precondition | Expected error |
|----|-------------|----------------|
| PRE-1 | AUTHZ-CHECK: if operator != "", an OperatorAuthorization must exist for (corp, operator) that covers `MsgGrantOperatorAuthorization` and is not expired | `"operator authorization not found"` or `"has expired"` or `"does not include requested message type"` |
| PRE-2 | Self-escalation guard: operator != "" AND grantee == operator → rejected | `"cannot grant authorization to itself"` |
| PRE-3 | Expiration must be strictly in the future if set | `"expiration must be in the future"` |
| PRE-4 | If `authz_spend_limit` is non-empty AND `authz_spend_limit_period` is set, period must be > 0 | `"authz_spend_limit_period must be a positive duration"` |
| PRE-5 | No VSOperatorAuthorization must exist for (corp, grantee) | `"VSOperatorAuthorization already exists"` |

**Happy path execution side-effects:**
1. `OperatorAuthorizations.Set(corp, grantee, OperatorAuthorization{...})` — full struct stored.
2. If `WithFeegrant=false`: any existing FeeGrant for (corp, grantee) is revoked.
3. If `WithFeegrant=true`: a FeeGrant is created for (corp, grantee).
4. Event `grant_operator_authorization` emitted with attrs: `corporation`, `grantee`, `with_feegrant`, `timestamp`.

- [ ] **Step 2.1: Create `x/de/keeper/msg_grant_operator_authorization_test.go`.**

```go
package keeper_test

import (
	"testing"
	"time"

	"cosmossdk.io/collections"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/stretchr/testify/require"

	"github.com/verana-labs/verana/x/de/types"
)

// Test address helpers — fixed bytes so addresses are deterministic.
var (
	corpAddr    = sdk.AccAddress([]byte("corp________________")).String()
	granteeAddr = sdk.AccAddress([]byte("grantee_____________")).String()
	operatorAddr = sdk.AccAddress([]byte("operator____________")).String()
)

// validMsgTypes is the minimal set used across grant tests.
var validMsgTypes = []string{"/verana.tr.v1.MsgCreateTrustRegistry"}

// bootstrapOperator seeds an OperatorAuthorization for (corp, operator) so that
// the AUTHZ-CHECK in GrantOperatorAuthorization passes for subsequent calls.
func bootstrapOperator(f *Fixture, corp, operator string, msgTypes []string) {
	f.t.Helper()
	key := collections.Join(corp, operator)
	err := f.K.OperatorAuthorizations.Set(f.Ctx, key, types.OperatorAuthorization{
		Corporation: corp,
		Operator:    operator,
		MsgTypes:    msgTypes,
	})
	require.NoError(f.t, err)
}

// ----------------------------------------------------------------------------
// Happy path
// ----------------------------------------------------------------------------

func TestMsgGrantOperatorAuthorization_HappyPath_GroupProposal(t *testing.T) {
	// Group proposal (operator == "") — AUTHZ-CHECK is skipped.
	f := NewFixture(t)
	now := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	f.SetBlockTime(now)
	futureExp := now.Add(24 * time.Hour)

	msg := &types.MsgGrantOperatorAuthorization{
		Corporation: corpAddr,
		Operator:    "", // group proposal
		Grantee:     granteeAddr,
		MsgTypes:    validMsgTypes,
		Expiration:  &futureExp,
	}

	resp, err := f.MS.GrantOperatorAuthorization(f.Ctx, msg)
	require.NoError(t, err)
	require.NotNil(t, resp)

	// Full struct assertion
	f.RequireOperatorAuth(corpAddr, granteeAddr, types.OperatorAuthorization{
		Corporation: corpAddr,
		Operator:    granteeAddr, // stored as Operator field per implementation
		MsgTypes:    validMsgTypes,
		Expiration:  &futureExp,
	})

	// Event assertion
	f.RequireEvent(types.EventTypeGrantOperatorAuthorization, map[string]string{
		types.AttributeKeyCorporation:  corpAddr,
		types.AttributeKeyGrantee:      granteeAddr,
		types.AttributeKeyWithFeegrant: "false",
	})

	// Invariant: no FeeGrants created without WithFeegrant=true
	f.RequireInvariant()
}

func TestMsgGrantOperatorAuthorization_HappyPath_WithFeegrant(t *testing.T) {
	f := NewFixture(t)
	now := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	f.SetBlockTime(now)

	msg := &types.MsgGrantOperatorAuthorization{
		Corporation:    corpAddr,
		Operator:       "",
		Grantee:        granteeAddr,
		MsgTypes:       validMsgTypes,
		WithFeegrant:   true,
	}

	_, err := f.MS.GrantOperatorAuthorization(f.Ctx, msg)
	require.NoError(t, err)

	// OperatorAuthorization stored
	f.RequireOperatorAuth(corpAddr, granteeAddr, types.OperatorAuthorization{
		Corporation: corpAddr,
		Operator:    granteeAddr,
		MsgTypes:    validMsgTypes,
	})

	// FeeGrant created
	fgKey := collections.Join(corpAddr, granteeAddr)
	fg, err := f.K.FeeGrants.Get(f.Ctx, fgKey)
	require.NoError(t, err)
	require.Equal(t, corpAddr, fg.Grantor)
	require.Equal(t, granteeAddr, fg.Grantee)
	require.Equal(t, validMsgTypes, fg.MsgTypes)

	// Event
	f.RequireEvent(types.EventTypeGrantOperatorAuthorization, map[string]string{
		types.AttributeKeyCorporation:  corpAddr,
		types.AttributeKeyGrantee:      granteeAddr,
		types.AttributeKeyWithFeegrant: "true",
	})

	// Invariant: FeeGrant has a matching OperatorAuthorization
	f.RequireInvariant()
}

func TestMsgGrantOperatorAuthorization_HappyPath_UpdateExistingRevokesOldFeeGrant(t *testing.T) {
	// Re-granting with WithFeegrant=false must revoke the previously-created FeeGrant.
	f := NewFixture(t)
	f.SetBlockTime(time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC))

	// Initial grant with feegrant
	_, err := f.MS.GrantOperatorAuthorization(f.Ctx, &types.MsgGrantOperatorAuthorization{
		Corporation:  corpAddr,
		Grantee:      granteeAddr,
		MsgTypes:     validMsgTypes,
		WithFeegrant: true,
	})
	require.NoError(t, err)

	fgKey := collections.Join(corpAddr, granteeAddr)
	has, err := f.K.FeeGrants.Has(f.Ctx, fgKey)
	require.NoError(t, err)
	require.True(t, has, "FeeGrant must exist after initial grant with feegrant=true")

	// Re-grant with feegrant=false
	newMsgTypes := []string{"/verana.cs.v1.MsgCreateCredentialSchema"}
	_, err = f.MS.GrantOperatorAuthorization(f.Ctx, &types.MsgGrantOperatorAuthorization{
		Corporation:  corpAddr,
		Grantee:      granteeAddr,
		MsgTypes:     newMsgTypes,
		WithFeegrant: false,
	})
	require.NoError(t, err)

	// FeeGrant must be gone
	has, err = f.K.FeeGrants.Has(f.Ctx, fgKey)
	require.NoError(t, err)
	require.False(t, has, "FeeGrant must be revoked after re-grant with feegrant=false")

	// OA updated to new msg types
	f.RequireOperatorAuth(corpAddr, granteeAddr, types.OperatorAuthorization{
		Corporation: corpAddr,
		Operator:    granteeAddr,
		MsgTypes:    newMsgTypes,
	})

	f.RequireInvariant()
}

func TestMsgGrantOperatorAuthorization_HappyPath_ExistingOperatorGrantsNew(t *testing.T) {
	// An operator with proper authorization grants a new grantee.
	f := NewFixture(t)
	f.SetBlockTime(time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC))

	bootstrapOperator(f, corpAddr, operatorAddr, []string{"/verana.de.v1.MsgGrantOperatorAuthorization"})

	newGrantee := sdk.AccAddress([]byte("new_grantee_________")).String()
	msg := &types.MsgGrantOperatorAuthorization{
		Corporation: corpAddr,
		Operator:    operatorAddr,
		Grantee:     newGrantee,
		MsgTypes:    validMsgTypes,
	}

	_, err := f.MS.GrantOperatorAuthorization(f.Ctx, msg)
	require.NoError(t, err)

	f.RequireOperatorAuth(corpAddr, newGrantee, types.OperatorAuthorization{
		Corporation: corpAddr,
		Operator:    newGrantee,
		MsgTypes:    validMsgTypes,
	})

	f.RequireInvariant()
}

// ----------------------------------------------------------------------------
// Negative cases — one per spec precondition
// ----------------------------------------------------------------------------

// [PRE-1a] AUTHZ-CHECK fails: operator has no authorization
func TestMsgGrantOperatorAuthorization_FailsIfOperatorNotAuthorized(t *testing.T) {
	f := NewFixture(t)
	f.SetBlockTime(time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC))

	msg := &types.MsgGrantOperatorAuthorization{
		Corporation: corpAddr,
		Operator:    operatorAddr, // no OA seeded
		Grantee:     granteeAddr,
		MsgTypes:    validMsgTypes,
	}

	resp, err := f.MS.GrantOperatorAuthorization(f.Ctx, msg)
	require.Error(t, err)
	require.ErrorContains(t, err, "operator authorization not found")
	require.Nil(t, resp)

	// Zero state written
	f.RequireNoOperatorAuth(corpAddr, granteeAddr)
	f.RequireNoEvent(types.EventTypeGrantOperatorAuthorization)
}

// [PRE-1b] AUTHZ-CHECK fails: operator authorization expired
func TestMsgGrantOperatorAuthorization_FailsIfOperatorAuthzExpired(t *testing.T) {
	f := NewFixture(t)
	now := time.Date(2026, 1, 1, 12, 0, 0, 0, time.UTC)
	f.SetBlockTime(now)

	pastExp := now.Add(-1 * time.Hour)
	key := collections.Join(corpAddr, operatorAddr)
	err := f.K.OperatorAuthorizations.Set(f.Ctx, key, types.OperatorAuthorization{
		Corporation: corpAddr,
		Operator:    operatorAddr,
		MsgTypes:    []string{"/verana.de.v1.MsgGrantOperatorAuthorization"},
		Expiration:  &pastExp,
	})
	require.NoError(t, err)

	msg := &types.MsgGrantOperatorAuthorization{
		Corporation: corpAddr,
		Operator:    operatorAddr,
		Grantee:     granteeAddr,
		MsgTypes:    validMsgTypes,
	}

	resp, err := f.MS.GrantOperatorAuthorization(f.Ctx, msg)
	require.Error(t, err)
	require.ErrorContains(t, err, "operator authorization has expired")
	require.Nil(t, resp)
	f.RequireNoOperatorAuth(corpAddr, granteeAddr)
	f.RequireNoEvent(types.EventTypeGrantOperatorAuthorization)
}

// [PRE-1c] AUTHZ-CHECK fails: operator does not have the required msg type
func TestMsgGrantOperatorAuthorization_FailsIfOperatorWrongMsgType(t *testing.T) {
	f := NewFixture(t)
	f.SetBlockTime(time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC))

	// Operator has only MsgRevoke, not MsgGrant
	bootstrapOperator(f, corpAddr, operatorAddr, []string{"/verana.de.v1.MsgRevokeOperatorAuthorization"})

	msg := &types.MsgGrantOperatorAuthorization{
		Corporation: corpAddr,
		Operator:    operatorAddr,
		Grantee:     granteeAddr,
		MsgTypes:    validMsgTypes,
	}

	resp, err := f.MS.GrantOperatorAuthorization(f.Ctx, msg)
	require.Error(t, err)
	require.ErrorContains(t, err, "does not include requested message type")
	require.Nil(t, resp)
	f.RequireNoOperatorAuth(corpAddr, granteeAddr)
}

// [PRE-2] Self-escalation guard: operator cannot grant to itself
func TestMsgGrantOperatorAuthorization_FailsOnSelfGrant(t *testing.T) {
	f := NewFixture(t)
	f.SetBlockTime(time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC))

	bootstrapOperator(f, corpAddr, operatorAddr, []string{"/verana.de.v1.MsgGrantOperatorAuthorization"})

	msg := &types.MsgGrantOperatorAuthorization{
		Corporation: corpAddr,
		Operator:    operatorAddr,
		Grantee:     operatorAddr, // grantee == operator — self-escalation
		MsgTypes:    []string{"/verana.tr.v1.MsgCreateTrustRegistry", "/verana.de.v1.MsgGrantOperatorAuthorization"},
	}

	resp, err := f.MS.GrantOperatorAuthorization(f.Ctx, msg)
	require.Error(t, err)
	require.ErrorContains(t, err, "cannot grant authorization to itself")
	require.Nil(t, resp)

	// Existing OA for operatorAddr must be unchanged (no privilege escalation)
	got, err := f.K.OperatorAuthorizations.Get(f.Ctx, collections.Join(corpAddr, operatorAddr))
	require.NoError(t, err)
	require.Equal(t, []string{"/verana.de.v1.MsgGrantOperatorAuthorization"}, got.MsgTypes,
		"self-grant must not have mutated the existing authorization")
}

// [PRE-3] Expiration in the past
func TestMsgGrantOperatorAuthorization_FailsIfExpirationInPast(t *testing.T) {
	f := NewFixture(t)
	now := time.Date(2026, 1, 1, 12, 0, 0, 0, time.UTC)
	f.SetBlockTime(now)

	pastExp := now.Add(-1 * time.Second)
	msg := &types.MsgGrantOperatorAuthorization{
		Corporation: corpAddr,
		Operator:    "",
		Grantee:     granteeAddr,
		MsgTypes:    validMsgTypes,
		Expiration:  &pastExp,
	}

	resp, err := f.MS.GrantOperatorAuthorization(f.Ctx, msg)
	require.Error(t, err)
	require.ErrorContains(t, err, "expiration must be in the future")
	require.Nil(t, resp)
	f.RequireNoOperatorAuth(corpAddr, granteeAddr)
	f.RequireNoEvent(types.EventTypeGrantOperatorAuthorization)
}

// [PRE-3 boundary] Expiration exactly at block time → fails (!After(now) == true when equal)
func TestMsgGrantOperatorAuthorization_FailsIfExpirationExactlyAtBlockTime(t *testing.T) {
	f := NewFixture(t)
	now := time.Date(2026, 1, 1, 12, 0, 0, 0, time.UTC)
	f.SetBlockTime(now)

	exactNow := now
	msg := &types.MsgGrantOperatorAuthorization{
		Corporation: corpAddr,
		Operator:    "",
		Grantee:     granteeAddr,
		MsgTypes:    validMsgTypes,
		Expiration:  &exactNow,
	}

	_, err := f.MS.GrantOperatorAuthorization(f.Ctx, msg)
	require.Error(t, err)
	require.ErrorContains(t, err, "expiration must be in the future")
}

// [PRE-4] authz_spend_limit_period <= 0 when authz_spend_limit is set
func TestMsgGrantOperatorAuthorization_FailsIfSpendLimitPeriodNotPositive(t *testing.T) {
	f := NewFixture(t)
	f.SetBlockTime(time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC))

	zeroPeriod := time.Duration(0)
	negPeriod := -time.Hour
	spendLimit := sdk.NewCoins(sdk.NewInt64Coin("uvna", 1000))

	for name, period := range map[string]time.Duration{"zero": zeroPeriod, "negative": negPeriod} {
		t.Run(name, func(t *testing.T) {
			p := period
			msg := &types.MsgGrantOperatorAuthorization{
				Corporation:             corpAddr,
				Operator:                "",
				Grantee:                 granteeAddr,
				MsgTypes:                validMsgTypes,
				AuthzSpendLimit:         spendLimit,
				AuthzSpendLimitPeriod:   &p,
			}
			resp, err := f.MS.GrantOperatorAuthorization(f.Ctx, msg)
			require.Error(t, err)
			require.ErrorContains(t, err, "authz_spend_limit_period must be a positive duration")
			require.Nil(t, resp)
			f.RequireNoOperatorAuth(corpAddr, granteeAddr)
		})
	}
}

// [PRE-4 valid] authz_spend_limit_period ignored when authz_spend_limit is nil/empty
func TestMsgGrantOperatorAuthorization_SpendLimitPeriodIgnoredWithoutSpendLimit(t *testing.T) {
	f := NewFixture(t)
	f.SetBlockTime(time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC))

	negPeriod := -time.Hour
	msg := &types.MsgGrantOperatorAuthorization{
		Corporation:             corpAddr,
		Operator:                "",
		Grantee:                 granteeAddr,
		MsgTypes:                validMsgTypes,
		AuthzSpendLimitPeriod:   &negPeriod, // ignored because AuthzSpendLimit is nil
	}

	_, err := f.MS.GrantOperatorAuthorization(f.Ctx, msg)
	require.NoError(t, err, "negative period must be ignored when no spend limit is set")
	f.RequireOperatorAuthCount(1)
}

// [PRE-5] VSOperatorAuthorization already exists for (corp, grantee) — mutual exclusivity
func TestMsgGrantOperatorAuthorization_FailsIfVSOAExists(t *testing.T) {
	f := NewFixture(t)
	f.SetBlockTime(time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC))

	vsKey := collections.Join(corpAddr, granteeAddr)
	err := f.K.VSOperatorAuthorizations.Set(f.Ctx, vsKey, types.VSOperatorAuthorization{
		Corporation: corpAddr,
		VsOperator:  granteeAddr,
	})
	require.NoError(t, err)

	msg := &types.MsgGrantOperatorAuthorization{
		Corporation: corpAddr,
		Operator:    "",
		Grantee:     granteeAddr,
		MsgTypes:    validMsgTypes,
	}

	resp, err := f.MS.GrantOperatorAuthorization(f.Ctx, msg)
	require.Error(t, err)
	require.ErrorContains(t, err, "VSOperatorAuthorization already exists")
	require.Nil(t, resp)
	f.RequireNoOperatorAuth(corpAddr, granteeAddr)
	f.RequireNoEvent(types.EventTypeGrantOperatorAuthorization)
}

// ----------------------------------------------------------------------------
// Edge cases
// ----------------------------------------------------------------------------

// Re-grant resets usage ledger (can be verified by checking OperatorAuthorizationUsage is absent)
func TestMsgGrantOperatorAuthorization_ReGrantResetsUsageLedger(t *testing.T) {
	f := NewFixture(t)
	f.SetBlockTime(time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC))

	// Seed a usage record as if the operator had spent some limit
	usageKey := collections.Join(corpAddr, granteeAddr)
	err := f.K.OperatorAuthorizationUsage.Set(f.Ctx, usageKey, types.OperatorAuthorizationUsage{
		Corporation: corpAddr,
		Operator:    granteeAddr,
	})
	require.NoError(t, err)

	// Re-grant: should delete the usage record
	_, err = f.MS.GrantOperatorAuthorization(f.Ctx, &types.MsgGrantOperatorAuthorization{
		Corporation: corpAddr,
		Grantee:     granteeAddr,
		MsgTypes:    validMsgTypes,
	})
	require.NoError(t, err)

	has, err := f.K.OperatorAuthorizationUsage.Has(f.Ctx, usageKey)
	require.NoError(t, err)
	require.False(t, has, "usage ledger must be reset on re-grant")
}

// Authority isolation: grants for authority1 do not affect authority2
func TestMsgGrantOperatorAuthorization_AuthorityIsolation(t *testing.T) {
	f := NewFixture(t)
	f.SetBlockTime(time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC))

	corp1 := sdk.AccAddress([]byte("corp1_______________")).String()
	corp2 := sdk.AccAddress([]byte("corp2_______________")).String()

	_, err := f.MS.GrantOperatorAuthorization(f.Ctx, &types.MsgGrantOperatorAuthorization{
		Corporation: corp1,
		Grantee:     granteeAddr,
		MsgTypes:    validMsgTypes,
	})
	require.NoError(t, err)

	_, err = f.MS.GrantOperatorAuthorization(f.Ctx, &types.MsgGrantOperatorAuthorization{
		Corporation: corp2,
		Grantee:     granteeAddr,
		MsgTypes:    validMsgTypes,
	})
	require.NoError(t, err)

	f.RequireOperatorAuth(corp1, granteeAddr, types.OperatorAuthorization{
		Corporation: corp1, Operator: granteeAddr, MsgTypes: validMsgTypes,
	})
	f.RequireOperatorAuth(corp2, granteeAddr, types.OperatorAuthorization{
		Corporation: corp2, Operator: granteeAddr, MsgTypes: validMsgTypes,
	})
	f.RequireOperatorAuthCount(2)
	f.RequireInvariant()
}
```

- [ ] **Step 2.2: Run the new tests.**

  ```bash
  go test ./x/de/keeper/... -run TestMsgGrantOperatorAuthorization -v -count=1
  ```
  Expected: all tests PASS.

- [ ] **Step 2.3: Commit.**

  ```bash
  git add x/de/keeper/msg_grant_operator_authorization_test.go
  git commit -m "test(de): add fixture-based tests for GrantOperatorAuthorization"
  ```

---

## Task 3: Delete old GrantOperatorAuthorization tests from `msg_server_test.go`

The following test functions in `x/de/keeper/msg_server_test.go` are superseded by the new fixture-based tests and must be deleted in this PR:

- `TestMsgServerGrantOperatorAuthorization` (line ~369)
- `TestMsgServerGrantOperatorAuthorization_WithFeegrant` (line ~701)
- `TestMsgServerGrantOperatorAuthorization_UpdateExisting` (line ~745)
- `TestGrantThenRevokeOperatorAuthorization_E2E` (line ~1179)
- `TestMutualExclusivity_OAAndVSOA` (line ~1238)
- `TestMultipleGranteesForSameAuthority` (line ~1275)
- `TestGrantRevokeReGrant_E2E` (line ~1322)
- `TestAuthorityIsolation` (line ~1360)
- `TestOperatorCannotSelfGrant` (line ~1565)
- `TestBootstrapFlow_GroupProposalOnboardsFirstOperator` (line ~1621)
- `TestGrantOperatorAuthorization_FeegrantFieldsStoredCorrectly` (line ~1512)

**Do NOT delete:** `TestGrantFeeAllowance`, `TestGrantFeeAllowance_UpdateExisting`, `TestRevokeFeeAllowance`, `TestRevokeFeeAllowance_DoubleRevoke`, `TestMsgServerRevokeOperatorAuthorization`, `TestMsgServerRevokeOperatorAuthorization_AlsoRevokesFeeGrant`, `TestMsgServerRevokeOperatorAuthorization_NoFeeGrant`, `TestCheckOperatorAuthorization`, `TestCheckOperatorAuthorization_ExpirationBoundary`, `TestOperatorRevokesOwnAuthorization`, `TestAddPermToVSOA`, `TestRemovePermFromVSOA`, `TestQueryListVSOperatorAuthorizations`, `TestQueryListOperatorAuthorizations`, `TestMsgServer`.

- [ ] **Step 3.1: Delete the listed test functions from `msg_server_test.go`.**

  Edit the file to remove each function body and its surrounding blank lines.

- [ ] **Step 3.2: Run full DE keeper test suite.**

  ```bash
  go test ./x/de/keeper/... -v -count=1
  ```
  Expected: all remaining tests pass. Zero compilation errors.

- [ ] **Step 3.3: Commit.**

  ```bash
  git add x/de/keeper/msg_server_test.go
  git commit -m "test(de): delete superseded GrantOperatorAuthorization tests from msg_server_test.go"
  ```

---

## Task 4: Final quality pass

- [ ] **Step 4.1: Run full test suite.**

  ```bash
  go test ./... -count=1
  ```
  Expected: PASS. No regressions.

- [ ] **Step 4.2: Run vet and linter.**

  ```bash
  go vet ./x/de/keeper/...
  golangci-lint run ./x/de/keeper/...
  ```
  Expected: no findings. Fix any reported issues.

- [ ] **Step 4.3: Coverage check.**

  ```bash
  go test ./x/de/keeper/ -cover -count=1 -run .
  ```
  Report coverage. Target ≥95% when the module is fully migrated. For this PR, note the coverage delta in the PR description.

- [ ] **Step 4.4: Push and open PR.**

  ```bash
  git push -u origin test/step-17-de-grant-operator-authz
  gh pr create --title "test(de): fixture-based tests for GrantOperatorAuthorization (issue #292 step 17)" --body "$(cat <<'EOF'
  ## Summary
  - New `x/de/keeper/fixture_test.go` with `Fixture` struct, assertion helpers (`RequireOperatorAuth`, `RequireNoOperatorAuth`, `RequireEvent`, `RequireNoEvent`, `RequireOperatorAuthCount`, `RequireInvariant`)
  - New `x/de/keeper/msg_grant_operator_authorization_test.go` with full precondition matrix (PRE-1a/b/c, PRE-2, PRE-3 including boundary, PRE-4, PRE-5), happy paths (group proposal, with feegrant, re-grant revokes fee grant, existing operator), and edge cases
  - Delete 11 superseded test functions from `msg_server_test.go`
  - DE invariant: every FeeGrant row references an existing OperatorAuthorization

  ## Test plan
  - [ ] `go test ./x/de/keeper/... -v -count=1` passes
  - [ ] `go test ./... -count=1` passes (no regressions)
  - [ ] `go vet ./x/de/keeper/...` clean
  - [ ] `golangci-lint run ./x/de/keeper/...` clean
  - [ ] Coverage delta noted in PR description
  EOF
  )"
  ```

---

## "Done" Criteria — Step 17

- [ ] `x/de/keeper/fixture_test.go` exists with `Fixture`, `NewFixture`, `RequireOperatorAuth`, `RequireNoOperatorAuth`, `RequireEvent`, `RequireNoEvent`, `RequireOperatorAuthCount`, `RequireInvariant`.
- [ ] `x/de/keeper/msg_grant_operator_authorization_test.go` exists.
- [ ] Happy path (group proposal): full `OperatorAuthorization` struct equality assertion, event assertion, invariant check.
- [ ] Happy path (with feegrant): `FeeGrant` struct assertion, event assertion, invariant check.
- [ ] Happy path (re-grant revokes existing fee grant): state verified before and after.
- [ ] Happy path (operator grants new): AUTHZ-CHECK exercised via bootstrapped OA.
- [ ] One `t.Run` equivalent per precondition: error contains expected string, zero OA written, no event emitted.
- [ ] PRE-3 boundary: expiration exactly at block time → fails.
- [ ] PRE-4: both zero and negative `authz_spend_limit_period` → fails; period ignored without spend limit → passes.
- [ ] PRE-2 self-grant: existing OA is unchanged after failed self-grant attempt.
- [ ] Edge case: re-grant resets usage ledger.
- [ ] Edge case: authority isolation (two corps, same grantee).
- [ ] 11 superseded test functions deleted from `msg_server_test.go`.
- [ ] `go test ./... -count=1` passes.
- [ ] `go vet ./x/de/keeper/...` clean.

---

## Self-Review Notes

- **No `testutil/keeper/delegation.go` needed:** The DE keeper's `NewKeeper` takes `(storeService, cdc, addressCodec, authority)` — no bank keeper, no mock delegation keeper. The `initFixture` in `keeper_test.go` already demonstrates the correct pattern. The new `Fixture` wraps that same construction.
- **`Operator` field semantics:** The implementation stores `msg.Grantee` in `oa.Operator` (line 80 of `msg_grant_operator_authorization.go`). Tests reflect this: `RequireOperatorAuth` compares `want.Operator == granteeAddr`.
- **Invariant scope for Step 17:** The DE invariant checks that every `FeeGrant` row has a matching `OperatorAuthorization`. Step 18 (Revoke) will reuse the same `RequireInvariant`. The full module invariant (`RequireInvariant` covering all four collection types) is finalized when Step 20 (RevokeVsOperatorAuthorization) lands.
- **Deleted tests:** The 11 deleted functions all test `GrantOperatorAuthorization` behavior. Tests for `Revoke`, `FeeAllowance`, `CheckOperatorAuthorization`, query handlers, genesis, and `TestMsgServer` remain in their current files and are migrated in Steps 18+ or left to the per-module completion step.
- **`setupMsgServer` helper:** The `setupMsgServer` function in `msg_server_test.go` is still used by remaining test functions (Revoke, FeeAllowance, etc.). Do NOT delete it.
