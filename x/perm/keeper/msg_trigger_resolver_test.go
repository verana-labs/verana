package keeper_test

import (
	"context"
	"reflect"
	"strconv"
	"testing"
	"time"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/stretchr/testify/require"

	"github.com/verana-labs/verana/x/perm/types"
)

// Test fixtures -------------------------------------------------------------

func tr_addr(seed string) string {
	b := make([]byte, 20)
	for i := 0; i < 20 && i < len(seed); i++ {
		b[i] = seed[i]
	}
	return sdk.AccAddress(b).String()
}

// trBlockTime is the canonical "now" used by the TriggerResolver test suite.
// Set on the sdk.Context via withBlockTime; chosen far enough from year 0001
// so test fixtures can subtract durations without underflowing timestamppb.
var trBlockTime = time.Date(2026, 1, 15, 12, 0, 0, 0, time.UTC)

func withBlockTime(ctx context.Context) (context.Context, sdk.Context) {
	sdkCtx := sdk.UnwrapSDKContext(ctx).WithBlockTime(trBlockTime)
	return sdk.WrapSDKContext(sdkCtx), sdkCtx
}

// activePerm returns a Permission populated with the minimum fields required
// to satisfy IsValidPermission(perm, now) for the given block time.
func activePerm(now time.Time) types.Permission {
	past := now.Add(-time.Hour)
	return types.Permission{
		SchemaId:      1,
		Type:          types.PermissionType_ISSUER,
		Did:           "did:example:abcdef",
		Created:       &now,
		Adjusted:      &now,
		Modified:      &now,
		VpState:       types.ValidationState_VALIDATED,
		EffectiveFrom: &past,
	}
}

// TestMsgTriggerResolver_ValidateBasic --------------------------------------

func TestMsgTriggerResolver_ValidateBasic(t *testing.T) {
	corp := tr_addr("corp_addr_______")
	op := tr_addr("op_addr_________")

	cases := []struct {
		name string
		msg  types.MsgTriggerResolver
		err  string
	}{
		{
			name: "ok",
			msg:  types.MsgTriggerResolver{Corporation: corp, Operator: op, Id: 1},
		},
		{
			name: "id=0",
			msg:  types.MsgTriggerResolver{Corporation: corp, Operator: op, Id: 0},
			err:  "perm ID cannot be 0",
		},
		{
			name: "bad corporation",
			msg:  types.MsgTriggerResolver{Corporation: "not-bech32", Operator: op, Id: 1},
			err:  "invalid corporation address",
		},
		{
			name: "bad operator",
			msg:  types.MsgTriggerResolver{Corporation: corp, Operator: "not-bech32", Id: 1},
			err:  "invalid operator address",
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			err := tc.msg.ValidateBasic()
			if tc.err == "" {
				require.NoError(t, err)
				return
			}
			require.Error(t, err)
			require.Contains(t, err.Error(), tc.err)
		})
	}
}

// TestMsgTriggerResolver_Handler --------------------------------------------

func TestMsgTriggerResolver_Path1Happy(t *testing.T) {
	k, ms, _, _, ctx, mockDel := setupMsgServerWithDelegation(t)
	ctx, sdkCtx := withBlockTime(ctx)
	now := sdkCtx.BlockTime()

	corp := tr_addr("corp_a__________")
	op := tr_addr("op_a____________")

	target := activePerm(now)
	target.Corporation = corp
	target.VsOperator = op
	target.VsOperatorAuthzEnabled = true

	id, err := k.CreatePermission(sdkCtx, target)
	require.NoError(t, err)

	mockDel.GetVSOAPermissionsResult = []uint64{id}

	resp, err := ms.TriggerResolver(ctx, &types.MsgTriggerResolver{
		Corporation: corp, Operator: op, Id: id,
	})
	require.NoError(t, err)
	require.NotNil(t, resp)
	requireEventEmittedExactlyOnce(t, sdkCtx, types.EventTypeTriggerResolver, id)
	requirePermUnchanged(t, sdkCtx, k, id, target)
}

func TestMsgTriggerResolver_Path1Happy_WithFeegrantFlag(t *testing.T) {
	k, ms, _, _, ctx, mockDel := setupMsgServerWithDelegation(t)
	ctx, sdkCtx := withBlockTime(ctx)
	now := sdkCtx.BlockTime()

	corp := tr_addr("corp_b__________")
	op := tr_addr("op_b____________")

	target := activePerm(now)
	target.Corporation = corp
	target.VsOperator = op
	target.VsOperatorAuthzEnabled = true
	target.VsOperatorAuthzWithFeegrant = true

	id, err := k.CreatePermission(sdkCtx, target)
	require.NoError(t, err)
	mockDel.GetVSOAPermissionsResult = []uint64{id}

	_, err = ms.TriggerResolver(ctx, &types.MsgTriggerResolver{
		Corporation: corp, Operator: op, Id: id,
	})
	require.NoError(t, err)
}

func TestMsgTriggerResolver_Path2a_AncestorVSOperator(t *testing.T) {
	k, ms, _, _, ctx, mockDel := setupMsgServerWithDelegation(t)
	ctx, sdkCtx := withBlockTime(ctx)
	now := sdkCtx.BlockTime()

	corpA := tr_addr("corp_anc________")
	corpT := tr_addr("corp_target_____") // different from corpA: kills Path 1 corp match
	opA := tr_addr("op_anc__________")

	ancestor := activePerm(now)
	ancestor.Corporation = corpA
	ancestor.VsOperator = opA
	ancestor.VsOperatorAuthzEnabled = true
	ancID, err := k.CreatePermission(sdkCtx, ancestor)
	require.NoError(t, err)

	target := activePerm(now)
	target.Corporation = corpT // not corpA — Path 1 corp check fails
	target.ValidatorPermId = ancID
	tgtID, err := k.CreatePermission(sdkCtx, target)
	require.NoError(t, err)

	// VSOA is keyed (corpA, opA) — return ancestor's id.
	mockDel.GetVSOAPermissionsResult = []uint64{ancID}

	_, err = ms.TriggerResolver(ctx, &types.MsgTriggerResolver{
		Corporation: corpA, Operator: opA, Id: tgtID,
	})
	require.NoError(t, err)
}

func TestMsgTriggerResolver_Path2b_AncestorOperatorAuthorization(t *testing.T) {
	k, ms, _, _, ctx, mockDel := setupMsgServerWithDelegation(t)
	ctx, sdkCtx := withBlockTime(ctx)
	now := sdkCtx.BlockTime()

	corpA := tr_addr("corp_anc________")
	corpT := tr_addr("corp_target_____")
	opCaller := tr_addr("op_caller_______")
	otherVS := tr_addr("op_other_vs_____") // ancestor.vs_operator != opCaller, kills 2a

	ancestor := activePerm(now)
	ancestor.Corporation = corpA
	ancestor.VsOperator = otherVS
	ancID, err := k.CreatePermission(sdkCtx, ancestor)
	require.NoError(t, err)

	target := activePerm(now)
	target.Corporation = corpT
	target.ValidatorPermId = ancID
	tgtID, err := k.CreatePermission(sdkCtx, target)
	require.NoError(t, err)

	// Mock allows all by default; CheckOperatorAuthorization returns nil → 2b passes.
	mockDel.ErrToReturn = nil

	_, err = ms.TriggerResolver(ctx, &types.MsgTriggerResolver{
		Corporation: corpA, Operator: opCaller, Id: tgtID,
	})
	require.NoError(t, err)
}

func TestMsgTriggerResolver_WalkSkipsInactiveAncestor(t *testing.T) {
	k, ms, _, _, ctx, _ := setupMsgServerWithDelegation(t)
	ctx, sdkCtx := withBlockTime(ctx)
	now := sdkCtx.BlockTime()
	future := now.Add(2 * time.Hour)

	corp := tr_addr("corp_anc________")
	op := tr_addr("op_caller_______")
	corpT := tr_addr("corp_target_____")

	// grandparent (active, will authorize via 2b)
	gp := activePerm(now)
	gp.Corporation = corp
	gpID, err := k.CreatePermission(sdkCtx, gp)
	require.NoError(t, err)

	// parent (inactive: not yet effective)
	parent := activePerm(now)
	parent.Corporation = corp
	parent.EffectiveFrom = &future
	parent.ValidatorPermId = gpID
	parentID, err := k.CreatePermission(sdkCtx, parent)
	require.NoError(t, err)

	// target
	target := activePerm(now)
	target.Corporation = corpT
	target.ValidatorPermId = parentID
	tgtID, err := k.CreatePermission(sdkCtx, target)
	require.NoError(t, err)

	_, err = ms.TriggerResolver(ctx, &types.MsgTriggerResolver{
		Corporation: corp, Operator: op, Id: tgtID,
	})
	require.NoError(t, err, "walk must skip inactive parent and authorize via active grandparent")
}

func TestMsgTriggerResolver_WalkNoMatch(t *testing.T) {
	k, ms, _, _, ctx, mockDel := setupMsgServerWithDelegation(t)
	ctx, sdkCtx := withBlockTime(ctx)
	now := sdkCtx.BlockTime()

	corpA := tr_addr("corp_anc________")
	corpT := tr_addr("corp_target_____")
	corpX := tr_addr("corp_caller_____") // matches no perm in chain
	op := tr_addr("op_caller_______")

	ancestor := activePerm(now)
	ancestor.Corporation = corpA
	ancID, err := k.CreatePermission(sdkCtx, ancestor)
	require.NoError(t, err)

	target := activePerm(now)
	target.Corporation = corpT
	target.ValidatorPermId = ancID
	tgtID, err := k.CreatePermission(sdkCtx, target)
	require.NoError(t, err)

	// Mock OK by default — but no corp in the chain matches corpX, so neither path passes.
	mockDel.ErrToReturn = nil

	_, err = ms.TriggerResolver(ctx, &types.MsgTriggerResolver{
		Corporation: corpX, Operator: op, Id: tgtID,
	})
	require.Error(t, err)
	require.ErrorIs(t, err, types.ErrTriggerResolverUnauthorized)
}

func TestMsgTriggerResolver_SelfCycleRejected(t *testing.T) {
	k, ms, _, _, ctx, mockDel := setupMsgServerWithDelegation(t)
	ctx, sdkCtx := withBlockTime(ctx)
	now := sdkCtx.BlockTime()

	corp := tr_addr("corp_self_______")
	op := tr_addr("op_self_________")
	corpCaller := tr_addr("corp_caller_____") // != target.corp, kills Path 1

	target := activePerm(now)
	target.Corporation = corp
	target.VsOperator = op
	target.VsOperatorAuthzEnabled = true
	tgtID, err := k.CreatePermission(sdkCtx, target)
	require.NoError(t, err)

	// Force validator_perm_id to point at itself (self-cycle).
	target.Id = tgtID
	target.ValidatorPermId = tgtID
	require.NoError(t, k.UpdatePermission(sdkCtx, target))

	mockDel.GetVSOAPermissionsResult = []uint64{tgtID}

	// Caller has corpCaller (no Path 1, no ancestor match by corp). Even though
	// the walk visits target itself via the cycle, msg.corporation != target.corporation,
	// so neither 2a nor 2b passes; cycle detection then aborts the walk.
	_, err = ms.TriggerResolver(ctx, &types.MsgTriggerResolver{
		Corporation: corpCaller, Operator: op, Id: tgtID,
	})
	require.Error(t, err)
	require.ErrorIs(t, err, types.ErrTriggerResolverUnauthorized)
}

func TestMsgTriggerResolver_RejectInactivePerm(t *testing.T) {
	k, ms, _, _, ctx, mockDel := setupMsgServerWithDelegation(t)
	ctx, sdkCtx := withBlockTime(ctx)
	now := sdkCtx.BlockTime()
	past := now.Add(-time.Hour)

	corp := tr_addr("corp_a__________")
	op := tr_addr("op_a____________")
	mockDel.ErrToReturn = nil

	cases := []struct {
		name  string
		patch func(*types.Permission)
	}{
		{"revoked", func(p *types.Permission) { rev := past; p.Revoked = &rev }},
		{"slashed", func(p *types.Permission) { sl := now; p.Slashed = &sl }},
		{"repaid", func(p *types.Permission) { rp := now; p.Repaid = &rp }},
		{"not_yet_effective", func(p *types.Permission) {
			fut := now.Add(time.Hour)
			p.EffectiveFrom = &fut
		}},
		{"expired", func(p *types.Permission) {
			ended := now.Add(-time.Minute)
			p.EffectiveUntil = &ended
		}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			perm := activePerm(now)
			perm.Corporation = corp
			perm.VsOperator = op
			perm.VsOperatorAuthzEnabled = true
			tc.patch(&perm)
			id, err := k.CreatePermission(sdkCtx, perm)
			require.NoError(t, err)
			mockDel.GetVSOAPermissionsResult = []uint64{id}

			_, err = ms.TriggerResolver(ctx, &types.MsgTriggerResolver{
				Corporation: corp, Operator: op, Id: id,
			})
			require.Error(t, err)
			require.ErrorIs(t, err, types.ErrPermissionNotActive)
		})
	}
}

func TestMsgTriggerResolver_RejectEmptyDID(t *testing.T) {
	k, ms, _, _, ctx, mockDel := setupMsgServerWithDelegation(t)
	ctx, sdkCtx := withBlockTime(ctx)
	now := sdkCtx.BlockTime()

	corp := tr_addr("corp_a__________")
	op := tr_addr("op_a____________")

	perm := activePerm(now)
	perm.Did = "" // empty
	perm.Corporation = corp
	perm.VsOperator = op
	perm.VsOperatorAuthzEnabled = true
	id, err := k.CreatePermission(sdkCtx, perm)
	require.NoError(t, err)
	mockDel.GetVSOAPermissionsResult = []uint64{id}

	_, err = ms.TriggerResolver(ctx, &types.MsgTriggerResolver{
		Corporation: corp, Operator: op, Id: id,
	})
	require.Error(t, err)
	require.ErrorIs(t, err, types.ErrPermissionDIDEmpty)
}

func TestMsgTriggerResolver_NonexistentPerm(t *testing.T) {
	_, ms, _, _, ctx, _ := setupMsgServerWithDelegation(t)

	corp := tr_addr("corp_a__________")
	op := tr_addr("op_a____________")
	_, err := ms.TriggerResolver(ctx, &types.MsgTriggerResolver{
		Corporation: corp, Operator: op, Id: 9999,
	})
	require.Error(t, err)
	require.Contains(t, err.Error(), "not found")
}

func TestMsgTriggerResolver_DelegationKeeperFailureRejects(t *testing.T) {
	k, ms, _, _, ctx, mockDel := setupMsgServerWithDelegation(t)
	ctx, sdkCtx := withBlockTime(ctx)
	now := sdkCtx.BlockTime()

	corp := tr_addr("corp_a__________")
	op := tr_addr("op_a____________")

	target := activePerm(now)
	target.Corporation = corp
	target.VsOperator = op
	target.VsOperatorAuthzEnabled = true
	id, err := k.CreatePermission(sdkCtx, target)
	require.NoError(t, err)

	mockDel.GetVSOAPermissionsResult = []uint64{id}
	// All delegation calls fail → both Path 1 (VSOA check) and Path 2b
	// (CheckOperatorAuthorization) return errors.
	mockDel.ErrToReturn = errSentinel("delegation denied")

	_, err = ms.TriggerResolver(ctx, &types.MsgTriggerResolver{
		Corporation: corp, Operator: op, Id: id,
	})
	require.Error(t, err)
	require.ErrorIs(t, err, types.ErrTriggerResolverUnauthorized)
}

// Helpers -------------------------------------------------------------------

type errSentinel string

func (e errSentinel) Error() string { return string(e) }

func requireEventEmittedExactlyOnce(t *testing.T, ctx sdk.Context, eventType string, expectedPermID uint64) {
	t.Helper()
	count := 0
	for _, e := range ctx.EventManager().Events() {
		if e.Type != eventType {
			continue
		}
		count++
		var got string
		for _, a := range e.Attributes {
			if a.Key == types.AttributeKeyPermissionID {
				got = a.Value
			}
		}
		require.Equal(t, strconv.FormatUint(expectedPermID, 10), got)
	}
	require.Equal(t, 1, count, "expected exactly one %s event", eventType)
}

func requirePermUnchanged(t *testing.T, ctx sdk.Context, k interface {
	GetPermissionByID(sdk.Context, uint64) (types.Permission, error)
}, id uint64, before types.Permission) {
	t.Helper()
	after, err := k.GetPermissionByID(ctx, id)
	require.NoError(t, err)
	// CreatePermission stamps Id; align before with after's Id for comparison.
	before.Id = after.Id
	require.True(t, reflect.DeepEqual(before, after), "permission record must not be mutated by TriggerResolver:\nbefore=%+v\nafter=%+v", before, after)
}
