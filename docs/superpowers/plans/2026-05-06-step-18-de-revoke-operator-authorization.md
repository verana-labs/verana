# Step 18: DE RevokeOperatorAuthorization — Fixture-Based Unit Tests

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Replace the existing table-driven `TestMsgServerRevokeOperatorAuthorization` tests in `x/de/keeper/msg_server_test.go` with a fixture-based test suite in `x/de/keeper/msg_revoke_operator_authorization_test.go`. Assumes Step 17 is already merged (`x/de/keeper/fixture_test.go` exists).

**Branch name:** `test/step-18-de-revoke-operator-authz`

**Architecture:**
- **Requires:** `x/de/keeper/fixture_test.go` from Step 17 (must be merged before this branch is started).
- **Create:** `x/de/keeper/msg_revoke_operator_authorization_test.go` — precondition matrix + happy path.
- **Delete:** `TestMsgServerRevokeOperatorAuthorization`, `TestMsgServerRevokeOperatorAuthorization_AlsoRevokesFeeGrant`, `TestMsgServerRevokeOperatorAuthorization_NoFeeGrant`, `TestOperatorRevokesOwnAuthorization` from `msg_server_test.go` in the same PR.

---

## Pre-flight

- [ ] **Worktree.** Create an isolated worktree. Branch from the merged Step 17 branch (or `main` after Step 17 merges). Branch name: `test/step-18-de-revoke-operator-authz`.
- [ ] **Confirm Step 17 is in the base.** `x/de/keeper/fixture_test.go` must exist with `Fixture`, `NewFixture`, `RequireOperatorAuth`, `RequireNoOperatorAuth`, `RequireEvent`, `RequireNoEvent`, `RequireInvariant`.
- [ ] **Sanity check.**

  ```bash
  go build ./... && go test ./x/de/keeper/... -count=1
  ```
  Expected: PASS.

---

## File Structure

- **Create:** `x/de/keeper/msg_revoke_operator_authorization_test.go`
- **Modify:** `x/de/keeper/msg_server_test.go` — delete the four Revoke test functions listed above.

---

## Task 1: Create `x/de/keeper/msg_revoke_operator_authorization_test.go`

**Preconditions for `RevokeOperatorAuthorization` (from implementation):**

| ID | Precondition | Expected error |
|----|-------------|----------------|
| PRE-1 | AUTHZ-CHECK: if operator != "", an OperatorAuthorization must exist for (corp, operator) that covers `MsgRevokeOperatorAuthorization` and is not expired | `"operator authorization not found"` or `"has expired"` or `"does not include requested message type"` |
| PRE-2 | OperatorAuthorization for (corp, grantee) must exist | `"operator authorization not found for this corporation/grantee pair"` |

**Happy path execution side-effects:**
1. `OperatorAuthorizations.Remove(corp, grantee)` — entry deleted.
2. `OperatorAuthorizationUsage.Remove(corp, grantee)` — usage ledger cleared (no-op if not present).
3. `RevokeFeeAllowance(corp, grantee)` called — any FeeGrant for (corp, grantee) is removed (no-op if not present).
4. Event `revoke_operator_authorization` emitted with attrs: `corporation`, `grantee`, `timestamp`.

- [ ] **Step 1.1: Create `x/de/keeper/msg_revoke_operator_authorization_test.go`.**

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

// ----------------------------------------------------------------------------
// Happy paths
// ----------------------------------------------------------------------------

// [MOD-DE-MSG-4] Happy path — group proposal (operator == ""), no fee grant present.
func TestMsgRevokeOperatorAuthorization_HappyPath_GroupProposal_NoFeeGrant(t *testing.T) {
	f := NewFixture(t)
	now := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	f.SetBlockTime(now)

	// Seed: grant OA without fee grant
	_, err := f.MS.GrantOperatorAuthorization(f.Ctx, &types.MsgGrantOperatorAuthorization{
		Corporation:  corpAddr,
		Operator:     "",
		Grantee:      granteeAddr,
		MsgTypes:     validMsgTypes,
		WithFeegrant: false,
	})
	require.NoError(t, err)
	f.RequireOperatorAuth(corpAddr, granteeAddr, types.OperatorAuthorization{
		Corporation: corpAddr,
		Operator:    granteeAddr,
		MsgTypes:    validMsgTypes,
	})

	// Revoke
	resp, err := f.MS.RevokeOperatorAuthorization(f.Ctx, &types.MsgRevokeOperatorAuthorization{
		Corporation: corpAddr,
		Operator:    "", // group proposal
		Grantee:     granteeAddr,
	})
	require.NoError(t, err)
	require.NotNil(t, resp)

	// State assertion: OA is gone
	f.RequireNoOperatorAuth(corpAddr, granteeAddr)

	// Usage ledger is also gone (was never set, but removal is a no-op — no error)
	usageKey := collections.Join(corpAddr, granteeAddr)
	has, err := f.K.OperatorAuthorizationUsage.Has(f.Ctx, usageKey)
	require.NoError(t, err)
	require.False(t, has)

	// Event assertion
	f.RequireEvent(types.EventTypeRevokeOperatorAuthorization, map[string]string{
		types.AttributeKeyCorporation: corpAddr,
		types.AttributeKeyGrantee:     granteeAddr,
	})

	// Invariant
	f.RequireInvariant()
}

// Happy path — group proposal, fee grant present and must be revoked.
func TestMsgRevokeOperatorAuthorization_HappyPath_AlsoRevokesFeeGrant(t *testing.T) {
	f := NewFixture(t)
	f.SetBlockTime(time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC))

	// Seed: grant OA WITH fee grant
	_, err := f.MS.GrantOperatorAuthorization(f.Ctx, &types.MsgGrantOperatorAuthorization{
		Corporation:  corpAddr,
		Grantee:      granteeAddr,
		MsgTypes:     validMsgTypes,
		WithFeegrant: true,
	})
	require.NoError(t, err)

	// Confirm FeeGrant exists before revoke
	fgKey := collections.Join(corpAddr, granteeAddr)
	hasFG, err := f.K.FeeGrants.Has(f.Ctx, fgKey)
	require.NoError(t, err)
	require.True(t, hasFG, "FeeGrant must exist after grant with feegrant=true")

	// Revoke
	_, err = f.MS.RevokeOperatorAuthorization(f.Ctx, &types.MsgRevokeOperatorAuthorization{
		Corporation: corpAddr,
		Operator:    "",
		Grantee:     granteeAddr,
	})
	require.NoError(t, err)

	// OA gone
	f.RequireNoOperatorAuth(corpAddr, granteeAddr)

	// FeeGrant also gone
	hasFG, err = f.K.FeeGrants.Has(f.Ctx, fgKey)
	require.NoError(t, err)
	require.False(t, hasFG, "FeeGrant must be revoked when OA is revoked")

	f.RequireEvent(types.EventTypeRevokeOperatorAuthorization, map[string]string{
		types.AttributeKeyCorporation: corpAddr,
		types.AttributeKeyGrantee:     granteeAddr,
	})

	// Invariant: no orphan FeeGrants
	f.RequireInvariant()
}

// Happy path — operator revokes another operator's authorization (not a self-revoke for escalation — just a valid revoke).
func TestMsgRevokeOperatorAuthorization_HappyPath_OperatorRevokesDifferentGrantee(t *testing.T) {
	f := NewFixture(t)
	f.SetBlockTime(time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC))

	// Bootstrap: seed operator's own OA with MsgRevoke permission
	bootstrapOperator(f, corpAddr, operatorAddr, []string{"/verana.de.v1.MsgRevokeOperatorAuthorization"})

	// Seed: target grantee's OA
	targetGrantee := sdk.AccAddress([]byte("target_grantee______")).String()
	targetKey := collections.Join(corpAddr, targetGrantee)
	err := f.K.OperatorAuthorizations.Set(f.Ctx, targetKey, types.OperatorAuthorization{
		Corporation: corpAddr,
		Operator:    targetGrantee,
		MsgTypes:    validMsgTypes,
	})
	require.NoError(t, err)

	// Revoke targetGrantee's OA using operatorAddr
	_, err = f.MS.RevokeOperatorAuthorization(f.Ctx, &types.MsgRevokeOperatorAuthorization{
		Corporation: corpAddr,
		Operator:    operatorAddr,
		Grantee:     targetGrantee,
	})
	require.NoError(t, err)

	f.RequireNoOperatorAuth(corpAddr, targetGrantee)
	// Operator's own OA still exists
	f.RequireOperatorAuth(corpAddr, operatorAddr, types.OperatorAuthorization{
		Corporation: corpAddr,
		Operator:    operatorAddr,
		MsgTypes:    []string{"/verana.de.v1.MsgRevokeOperatorAuthorization"},
	})

	f.RequireInvariant()
}

// Happy path — operator revokes its own authorization (self-revoke is allowed).
func TestMsgRevokeOperatorAuthorization_HappyPath_OperatorSelfRevoke(t *testing.T) {
	f := NewFixture(t)
	f.SetBlockTime(time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC))

	bootstrapOperator(f, corpAddr, operatorAddr, []string{
		"/verana.de.v1.MsgRevokeOperatorAuthorization",
	})

	// Operator revokes itself
	_, err := f.MS.RevokeOperatorAuthorization(f.Ctx, &types.MsgRevokeOperatorAuthorization{
		Corporation: corpAddr,
		Operator:    operatorAddr,
		Grantee:     operatorAddr, // self-revoke
	})
	require.NoError(t, err)

	f.RequireNoOperatorAuth(corpAddr, operatorAddr)
	f.RequireInvariant()
}

// Happy path — revoke with usage ledger present; usage ledger is cleared.
func TestMsgRevokeOperatorAuthorization_HappyPath_ClearsUsageLedger(t *testing.T) {
	f := NewFixture(t)
	f.SetBlockTime(time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC))

	// Seed OA
	_, err := f.MS.GrantOperatorAuthorization(f.Ctx, &types.MsgGrantOperatorAuthorization{
		Corporation: corpAddr,
		Grantee:     granteeAddr,
		MsgTypes:    validMsgTypes,
	})
	require.NoError(t, err)

	// Manually seed usage ledger
	usageKey := collections.Join(corpAddr, granteeAddr)
	err = f.K.OperatorAuthorizationUsage.Set(f.Ctx, usageKey, types.OperatorAuthorizationUsage{
		Corporation: corpAddr,
		Operator:    granteeAddr,
	})
	require.NoError(t, err)

	// Revoke
	_, err = f.MS.RevokeOperatorAuthorization(f.Ctx, &types.MsgRevokeOperatorAuthorization{
		Corporation: corpAddr,
		Grantee:     granteeAddr,
	})
	require.NoError(t, err)

	// Usage ledger cleared
	has, err := f.K.OperatorAuthorizationUsage.Has(f.Ctx, usageKey)
	require.NoError(t, err)
	require.False(t, has, "usage ledger must be cleared on revoke")

	f.RequireInvariant()
}

// ----------------------------------------------------------------------------
// Negative cases — one per spec precondition
// ----------------------------------------------------------------------------

// [PRE-1a] AUTHZ-CHECK fails: operator has no authorization
func TestMsgRevokeOperatorAuthorization_FailsIfOperatorNotAuthorized(t *testing.T) {
	f := NewFixture(t)
	f.SetBlockTime(time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC))

	// Seed the grantee's OA that we want to revoke
	targetKey := collections.Join(corpAddr, granteeAddr)
	err := f.K.OperatorAuthorizations.Set(f.Ctx, targetKey, types.OperatorAuthorization{
		Corporation: corpAddr,
		Operator:    granteeAddr,
		MsgTypes:    validMsgTypes,
	})
	require.NoError(t, err)

	resp, err := f.MS.RevokeOperatorAuthorization(f.Ctx, &types.MsgRevokeOperatorAuthorization{
		Corporation: corpAddr,
		Operator:    operatorAddr, // no OA seeded for operatorAddr
		Grantee:     granteeAddr,
	})
	require.Error(t, err)
	require.ErrorContains(t, err, "operator authorization not found")
	require.Nil(t, resp)

	// Target OA must be untouched
	f.RequireOperatorAuth(corpAddr, granteeAddr, types.OperatorAuthorization{
		Corporation: corpAddr,
		Operator:    granteeAddr,
		MsgTypes:    validMsgTypes,
	})
	f.RequireNoEvent(types.EventTypeRevokeOperatorAuthorization)
}

// [PRE-1b] AUTHZ-CHECK fails: operator authorization expired
func TestMsgRevokeOperatorAuthorization_FailsIfOperatorAuthzExpired(t *testing.T) {
	f := NewFixture(t)
	now := time.Date(2026, 1, 1, 12, 0, 0, 0, time.UTC)
	f.SetBlockTime(now)

	// Seed expired operator OA
	pastExp := now.Add(-1 * time.Hour)
	opKey := collections.Join(corpAddr, operatorAddr)
	err := f.K.OperatorAuthorizations.Set(f.Ctx, opKey, types.OperatorAuthorization{
		Corporation: corpAddr,
		Operator:    operatorAddr,
		MsgTypes:    []string{"/verana.de.v1.MsgRevokeOperatorAuthorization"},
		Expiration:  &pastExp,
	})
	require.NoError(t, err)

	// Seed the grantee's OA that we want to revoke
	targetKey := collections.Join(corpAddr, granteeAddr)
	err = f.K.OperatorAuthorizations.Set(f.Ctx, targetKey, types.OperatorAuthorization{
		Corporation: corpAddr,
		Operator:    granteeAddr,
		MsgTypes:    validMsgTypes,
	})
	require.NoError(t, err)

	resp, err := f.MS.RevokeOperatorAuthorization(f.Ctx, &types.MsgRevokeOperatorAuthorization{
		Corporation: corpAddr,
		Operator:    operatorAddr,
		Grantee:     granteeAddr,
	})
	require.Error(t, err)
	require.ErrorContains(t, err, "operator authorization has expired")
	require.Nil(t, resp)

	// Target OA still exists
	f.RequireOperatorAuth(corpAddr, granteeAddr, types.OperatorAuthorization{
		Corporation: corpAddr,
		Operator:    granteeAddr,
		MsgTypes:    validMsgTypes,
	})
	f.RequireNoEvent(types.EventTypeRevokeOperatorAuthorization)
}

// [PRE-1c] AUTHZ-CHECK fails: operator does not have MsgRevokeOperatorAuthorization
func TestMsgRevokeOperatorAuthorization_FailsIfOperatorWrongMsgType(t *testing.T) {
	f := NewFixture(t)
	f.SetBlockTime(time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC))

	// Operator has only MsgGrant, not MsgRevoke
	bootstrapOperator(f, corpAddr, operatorAddr, []string{"/verana.de.v1.MsgGrantOperatorAuthorization"})

	targetKey := collections.Join(corpAddr, granteeAddr)
	err := f.K.OperatorAuthorizations.Set(f.Ctx, targetKey, types.OperatorAuthorization{
		Corporation: corpAddr,
		Operator:    granteeAddr,
		MsgTypes:    validMsgTypes,
	})
	require.NoError(t, err)

	resp, err := f.MS.RevokeOperatorAuthorization(f.Ctx, &types.MsgRevokeOperatorAuthorization{
		Corporation: corpAddr,
		Operator:    operatorAddr,
		Grantee:     granteeAddr,
	})
	require.Error(t, err)
	require.ErrorContains(t, err, "does not include requested message type")
	require.Nil(t, resp)
	f.RequireNoEvent(types.EventTypeRevokeOperatorAuthorization)
}

// [PRE-2] OperatorAuthorization for (corp, grantee) does not exist
func TestMsgRevokeOperatorAuthorization_FailsIfGranteeOANotFound(t *testing.T) {
	f := NewFixture(t)
	f.SetBlockTime(time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC))

	nonExistent := sdk.AccAddress([]byte("non_existent________")).String()

	resp, err := f.MS.RevokeOperatorAuthorization(f.Ctx, &types.MsgRevokeOperatorAuthorization{
		Corporation: corpAddr,
		Operator:    "", // group proposal
		Grantee:     nonExistent,
	})
	require.Error(t, err)
	require.ErrorContains(t, err, "operator authorization not found for this corporation/grantee pair")
	require.Nil(t, resp)
	f.RequireNoEvent(types.EventTypeRevokeOperatorAuthorization)
}

// ----------------------------------------------------------------------------
// Edge cases
// ----------------------------------------------------------------------------

// Revoke-revoke idempotency: second revoke must fail because OA no longer exists.
func TestMsgRevokeOperatorAuthorization_DoubleRevokeFails(t *testing.T) {
	f := NewFixture(t)
	f.SetBlockTime(time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC))

	// Grant then revoke
	_, err := f.MS.GrantOperatorAuthorization(f.Ctx, &types.MsgGrantOperatorAuthorization{
		Corporation: corpAddr,
		Grantee:     granteeAddr,
		MsgTypes:    validMsgTypes,
	})
	require.NoError(t, err)

	_, err = f.MS.RevokeOperatorAuthorization(f.Ctx, &types.MsgRevokeOperatorAuthorization{
		Corporation: corpAddr,
		Grantee:     granteeAddr,
	})
	require.NoError(t, err)
	f.RequireNoOperatorAuth(corpAddr, granteeAddr)

	// Second revoke must fail
	_, err = f.MS.RevokeOperatorAuthorization(f.Ctx, &types.MsgRevokeOperatorAuthorization{
		Corporation: corpAddr,
		Grantee:     granteeAddr,
	})
	require.Error(t, err)
	require.ErrorContains(t, err, "operator authorization not found for this corporation/grantee pair")
}

// Authority isolation: revoking corp1/grantee does not affect corp2/grantee.
func TestMsgRevokeOperatorAuthorization_AuthorityIsolation(t *testing.T) {
	f := NewFixture(t)
	f.SetBlockTime(time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC))

	corp1 := sdk.AccAddress([]byte("corp1_______________")).String()
	corp2 := sdk.AccAddress([]byte("corp2_______________")).String()

	// Grant for both corporations
	for _, corp := range []string{corp1, corp2} {
		_, err := f.MS.GrantOperatorAuthorization(f.Ctx, &types.MsgGrantOperatorAuthorization{
			Corporation: corp,
			Grantee:     granteeAddr,
			MsgTypes:    validMsgTypes,
		})
		require.NoError(t, err)
	}

	// Revoke only corp1
	_, err := f.MS.RevokeOperatorAuthorization(f.Ctx, &types.MsgRevokeOperatorAuthorization{
		Corporation: corp1,
		Grantee:     granteeAddr,
	})
	require.NoError(t, err)

	// corp1/grantee is gone
	f.RequireNoOperatorAuth(corp1, granteeAddr)

	// corp2/grantee is untouched
	f.RequireOperatorAuth(corp2, granteeAddr, types.OperatorAuthorization{
		Corporation: corp2,
		Operator:    granteeAddr,
		MsgTypes:    validMsgTypes,
	})

	// corp1 cannot revoke corp2's grant
	_, err = f.MS.RevokeOperatorAuthorization(f.Ctx, &types.MsgRevokeOperatorAuthorization{
		Corporation: corp1,
		Grantee:     granteeAddr,
	})
	require.Error(t, err)
	require.ErrorContains(t, err, "operator authorization not found for this corporation/grantee pair")

	f.RequireInvariant()
}

// Grant → Revoke → Re-grant cycle works correctly.
func TestMsgRevokeOperatorAuthorization_GrantRevokeReGrant(t *testing.T) {
	f := NewFixture(t)
	f.SetBlockTime(time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC))

	msgTypes1 := []string{"/verana.tr.v1.MsgCreateTrustRegistry"}
	msgTypes2 := []string{"/verana.cs.v1.MsgCreateCredentialSchema"}

	// Grant
	_, err := f.MS.GrantOperatorAuthorization(f.Ctx, &types.MsgGrantOperatorAuthorization{
		Corporation: corpAddr,
		Grantee:     granteeAddr,
		MsgTypes:    msgTypes1,
	})
	require.NoError(t, err)

	// Revoke
	_, err = f.MS.RevokeOperatorAuthorization(f.Ctx, &types.MsgRevokeOperatorAuthorization{
		Corporation: corpAddr,
		Grantee:     granteeAddr,
	})
	require.NoError(t, err)
	f.RequireNoOperatorAuth(corpAddr, granteeAddr)

	// Re-grant with new msg types
	_, err = f.MS.GrantOperatorAuthorization(f.Ctx, &types.MsgGrantOperatorAuthorization{
		Corporation: corpAddr,
		Grantee:     granteeAddr,
		MsgTypes:    msgTypes2,
	})
	require.NoError(t, err)

	f.RequireOperatorAuth(corpAddr, granteeAddr, types.OperatorAuthorization{
		Corporation: corpAddr,
		Operator:    granteeAddr,
		MsgTypes:    msgTypes2,
	})

	f.RequireInvariant()
}

// Multiple grantees: revoking one does not affect others under the same corp.
func TestMsgRevokeOperatorAuthorization_MultipleGranteesIsolation(t *testing.T) {
	f := NewFixture(t)
	f.SetBlockTime(time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC))

	grantee1 := sdk.AccAddress([]byte("grantee1____________")).String()
	grantee2 := sdk.AccAddress([]byte("grantee2____________")).String()
	grantee3 := sdk.AccAddress([]byte("grantee3____________")).String()

	for _, g := range []string{grantee1, grantee2, grantee3} {
		_, err := f.MS.GrantOperatorAuthorization(f.Ctx, &types.MsgGrantOperatorAuthorization{
			Corporation: corpAddr,
			Grantee:     g,
			MsgTypes:    validMsgTypes,
		})
		require.NoError(t, err)
	}
	f.RequireOperatorAuthCount(3)

	// Revoke grantee2
	_, err := f.MS.RevokeOperatorAuthorization(f.Ctx, &types.MsgRevokeOperatorAuthorization{
		Corporation: corpAddr,
		Grantee:     grantee2,
	})
	require.NoError(t, err)

	f.RequireOperatorAuthCount(2)
	f.RequireNoOperatorAuth(corpAddr, grantee2)
	f.RequireOperatorAuth(corpAddr, grantee1, types.OperatorAuthorization{
		Corporation: corpAddr, Operator: grantee1, MsgTypes: validMsgTypes,
	})
	f.RequireOperatorAuth(corpAddr, grantee3, types.OperatorAuthorization{
		Corporation: corpAddr, Operator: grantee3, MsgTypes: validMsgTypes,
	})

	f.RequireInvariant()
}

// Bootstrap flow: group proposal onboards first operator; first operator onboards second;
// group revokes second; first operator still valid.
func TestMsgRevokeOperatorAuthorization_BootstrapFlowGroupRevoke(t *testing.T) {
	f := NewFixture(t)
	f.SetBlockTime(time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC))

	groupAccount := sdk.AccAddress([]byte("group_policy________")).String()
	firstOp := sdk.AccAddress([]byte("first_op____________")).String()
	secondOp := sdk.AccAddress([]byte("second_op___________")).String()

	deMsgTypes := []string{
		"/verana.de.v1.MsgGrantOperatorAuthorization",
		"/verana.de.v1.MsgRevokeOperatorAuthorization",
	}
	trMsgTypes := []string{"/verana.tr.v1.MsgCreateTrustRegistry"}

	// Step 1: group proposal onboards firstOp
	_, err := f.MS.GrantOperatorAuthorization(f.Ctx, &types.MsgGrantOperatorAuthorization{
		Corporation: groupAccount,
		Operator:    "",
		Grantee:     firstOp,
		MsgTypes:    append(deMsgTypes, trMsgTypes...),
	})
	require.NoError(t, err)

	// Step 2: firstOp onboards secondOp with limited permissions
	_, err = f.MS.GrantOperatorAuthorization(f.Ctx, &types.MsgGrantOperatorAuthorization{
		Corporation: groupAccount,
		Operator:    firstOp,
		Grantee:     secondOp,
		MsgTypes:    trMsgTypes,
	})
	require.NoError(t, err)

	f.RequireOperatorAuthCount(2)

	// Step 3: group revokes secondOp directly (group proposal, operator == "")
	_, err = f.MS.RevokeOperatorAuthorization(f.Ctx, &types.MsgRevokeOperatorAuthorization{
		Corporation: groupAccount,
		Operator:    "",
		Grantee:     secondOp,
	})
	require.NoError(t, err)

	f.RequireNoOperatorAuth(groupAccount, secondOp)
	f.RequireOperatorAuthCount(1)

	// firstOp still valid
	f.RequireOperatorAuth(groupAccount, firstOp, types.OperatorAuthorization{
		Corporation: groupAccount,
		Operator:    firstOp,
		MsgTypes:    append(deMsgTypes, trMsgTypes...),
	})

	f.RequireInvariant()
}
```

- [ ] **Step 1.2: Run the new tests.**

  ```bash
  go test ./x/de/keeper/... -run TestMsgRevokeOperatorAuthorization -v -count=1
  ```
  Expected: all tests PASS.

- [ ] **Step 1.3: Commit.**

  ```bash
  git add x/de/keeper/msg_revoke_operator_authorization_test.go
  git commit -m "test(de): add fixture-based tests for RevokeOperatorAuthorization"
  ```

---

## Task 2: Delete old RevokeOperatorAuthorization tests from `msg_server_test.go`

The following test functions are superseded by the new fixture-based tests and must be deleted:

- `TestMsgServerRevokeOperatorAuthorization` (line ~782)
- `TestMsgServerRevokeOperatorAuthorization_AlsoRevokesFeeGrant` (line ~937)
- `TestMsgServerRevokeOperatorAuthorization_NoFeeGrant` (line ~983)
- `TestOperatorRevokesOwnAuthorization` (line ~1417)

**Do NOT delete** any other functions — the fee allowance tests, CheckOperatorAuthorization tests, AddPermToVSOA, RemovePermFromVSOA, query handler tests, genesis tests, and `TestMsgServer` all remain.

- [ ] **Step 2.1: Delete the four listed functions from `msg_server_test.go`.**

- [ ] **Step 2.2: Run full DE keeper test suite.**

  ```bash
  go test ./x/de/keeper/... -v -count=1
  ```
  Expected: all remaining tests pass.

- [ ] **Step 2.3: Commit.**

  ```bash
  git add x/de/keeper/msg_server_test.go
  git commit -m "test(de): delete superseded RevokeOperatorAuthorization tests from msg_server_test.go"
  ```

---

## Task 3: Final quality pass

- [ ] **Step 3.1: Run full test suite.**

  ```bash
  go test ./... -count=1
  ```
  Expected: PASS. No regressions.

- [ ] **Step 3.2: Run vet and linter.**

  ```bash
  go vet ./x/de/keeper/...
  golangci-lint run ./x/de/keeper/...
  ```
  Expected: no findings.

- [ ] **Step 3.3: Coverage check.**

  ```bash
  go test ./x/de/keeper/ -cover -count=1 -run .
  ```
  Note coverage delta in PR description.

- [ ] **Step 3.4: Push and open PR.**

  ```bash
  git push -u origin test/step-18-de-revoke-operator-authz
  gh pr create --title "test(de): fixture-based tests for RevokeOperatorAuthorization (issue #292 step 18)" --body "$(cat <<'EOF'
  ## Summary
  - New `x/de/keeper/msg_revoke_operator_authorization_test.go` with full precondition matrix (PRE-1a/b/c, PRE-2), happy paths (group proposal no-feegrant, group proposal with-feegrant cascade, operator revokes other, self-revoke, usage-ledger cleared), and edge cases (double revoke, authority isolation, grant-revoke-regrant, multiple grantees, bootstrap flow)
  - Delete 4 superseded test functions from `msg_server_test.go`
  - Reuses `Fixture` from Step 17; `RequireInvariant` verifies no orphan FeeGrants after every revoke

  ## Test plan
  - [ ] `go test ./x/de/keeper/... -v -count=1` passes
  - [ ] `go test ./... -count=1` passes (no regressions)
  - [ ] `go vet ./x/de/keeper/...` clean
  - [ ] `golangci-lint run ./x/de/keeper/...` clean
  - [ ] Coverage delta noted
  EOF
  )"
  ```

---

## "Done" Criteria — Step 18

- [ ] `x/de/keeper/msg_revoke_operator_authorization_test.go` exists.
- [ ] Happy path (group proposal, no fee grant): OA removed, usage cleared, event emitted, invariant passes.
- [ ] Happy path (with fee grant): FeeGrant also removed on revoke, invariant passes.
- [ ] Happy path (operator revokes other): AUTHZ-CHECK exercised.
- [ ] Happy path (self-revoke): operator can revoke its own OA.
- [ ] Happy path (usage ledger cleared): manually seeded usage record is removed.
- [ ] One test per PRE-1a/1b/1c: error string, target OA untouched, no event.
- [ ] PRE-2: non-existent grantee → `"operator authorization not found for this corporation/grantee pair"`.
- [ ] Edge: double revoke → second fails.
- [ ] Edge: authority isolation (corp1 revoke does not affect corp2).
- [ ] Edge: grant-revoke-re-grant cycle verifies new msg types.
- [ ] Edge: multiple grantees — revoking one leaves others intact, count verified.
- [ ] Bootstrap flow: group revokes secondOp; firstOp still valid; count is 1.
- [ ] 4 superseded test functions deleted from `msg_server_test.go`.
- [ ] `go test ./... -count=1` passes.
- [ ] `go vet ./x/de/keeper/...` clean.

---

## Self-Review Notes

- **Dependency on Step 17:** This plan assumes `x/de/keeper/fixture_test.go` exists (created in Step 17). The `Fixture`, `NewFixture`, `RequireOperatorAuth`, `RequireNoOperatorAuth`, `RequireEvent`, `RequireNoEvent`, `RequireOperatorAuthCount`, `RequireInvariant`, and `bootstrapOperator` helpers are all reused without re-declaration.
- **`corpAddr`, `granteeAddr`, `operatorAddr`:** These `var` declarations exist in `msg_grant_operator_authorization_test.go` (same package `keeper_test`). They are shared across both test files without re-declaration.
- **`validMsgTypes`:** Declared in Step 17's test file; accessible here via same package.
- **Invariant after revoke:** After a successful revoke, no FeeGrant should reference the now-deleted OA. `RequireInvariant` in every happy path confirms this. The invariant is: "for every FeeGrant(corp, grantee), an OperatorAuthorization(corp, grantee) must exist."
- **`bootstrapOperator` helper:** Declared in `msg_grant_operator_authorization_test.go`; accessible here.
- **`setupMsgServer` function:** Still present in `msg_server_test.go` and used by remaining test functions (`TestGrantFeeAllowance`, etc.). Do NOT delete it when deleting the four Revoke test functions.
- **Event timing:** `f.RequireEvent` checks `f.Ctx.EventManager().Events()`. Since each `NewFixture(t)` call creates a fresh context with a fresh event manager, there is no cross-test event pollution.
- **Out of scope:** Steps 19 (GrantVsOperatorAuthorization), 20 (RevokeVsOperatorAuthorization) are separate. Once Step 20 lands, `RequireInvariant` can be extended to also check VSOperatorAuthorization entries.
