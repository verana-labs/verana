package keeper_test

import (
	"fmt"
	"testing"
	"time"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/stretchr/testify/require"

	keepertest "github.com/verana-labs/verana/testutil/keeper"
	"github.com/verana-labs/verana/x/pp/keeper"
	"github.com/verana-labs/verana/x/pp/types"
)

// helper: an active HOLDER participant.
func activeParticipant(corpID uint64, vsOperator, did string, validatorID uint64, from time.Time) types.Participant {
	return types.Participant{
		SchemaId:               1,
		Role:                   types.ParticipantRole_HOLDER,
		Did:                    did,
		CorporationId:          corpID,
		VsOperator:             vsOperator,
		ValidatorParticipantId: validatorID,
		EffectiveFrom:          &from,
	}
}

func setupTriggerResolver(t testing.TB) (
	keeper.Keeper, types.MsgServer, sdk.Context,
	*keepertest.MockParticipantEcosystemKeeper, *keepertest.MockDelegationKeeper,
) {
	t.Helper()
	k, ms, _, trk, ctx, del := setupMsgServerWithDelegation(t)
	sdkCtx := sdk.UnwrapSDKContext(ctx).WithBlockTime(time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC))
	return k, ms, sdkCtx, trk, del
}

// TestTriggerResolver_Path1 — vs_operator of the target participant.
func TestTriggerResolver_Path1(t *testing.T) {
	k, ms, ctx, trk, del := setupTriggerResolver(t)
	corporation := sdk.AccAddress([]byte("corp_trigger________")).String()
	operator := sdk.AccAddress([]byte("op_trigger__________")).String()
	corpID := trk.RegisterCorp(corporation)
	past := ctx.BlockTime().Add(-time.Hour)

	pid, err := k.CreateParticipant(ctx, activeParticipant(corpID, operator, "did:example:holder", 0, past))
	require.NoError(t, err)

	del.ErrToReturn = nil // CHECK-3 passes
	_, err = ms.TriggerResolver(sdk.WrapSDKContext(ctx), &types.MsgTriggerResolver{
		Corporation: corporation, Operator: operator, Id: pid,
	})
	require.NoError(t, err)
}

// TestTriggerResolver_Path2 — ancestor validator's corp operator (AUTHZ-CHECK-1).
func TestTriggerResolver_Path2(t *testing.T) {
	k, ms, ctx, trk, del := setupTriggerResolver(t)
	corporation := sdk.AccAddress([]byte("corp_trigger________")).String()
	operator := sdk.AccAddress([]byte("op_trigger__________")).String()
	corpID := trk.RegisterCorp(corporation)
	past := ctx.BlockTime().Add(-time.Hour)

	// Active ancestor in the same corporation.
	ancestorID, err := k.CreateParticipant(ctx, activeParticipant(corpID, "other_vsop", "did:example:anc", 0, past))
	require.NoError(t, err)
	// Target: operator is NOT its vs_operator (Path 1 fails), validator points to ancestor.
	pid, err := k.CreateParticipant(ctx, activeParticipant(corpID, "someone_else", "did:example:holder", ancestorID, past))
	require.NoError(t, err)

	del.ErrToReturn = nil // CHECK-1 passes for the ancestor's corp operator
	_, err = ms.TriggerResolver(sdk.WrapSDKContext(ctx), &types.MsgTriggerResolver{
		Corporation: corporation, Operator: operator, Id: pid,
	})
	require.NoError(t, err)
}

// TestTriggerResolver_Unauthorized — no path matches.
func TestTriggerResolver_Unauthorized(t *testing.T) {
	k, ms, ctx, trk, del := setupTriggerResolver(t)
	corporation := sdk.AccAddress([]byte("corp_trigger________")).String()
	operator := sdk.AccAddress([]byte("op_trigger__________")).String()
	corpID := trk.RegisterCorp(corporation)
	past := ctx.BlockTime().Add(-time.Hour)

	// operator is not the vs_operator and there is no ancestor.
	pid, err := k.CreateParticipant(ctx, activeParticipant(corpID, "someone_else", "did:example:holder", 0, past))
	require.NoError(t, err)

	del.ErrToReturn = fmt.Errorf("denied")
	_, err = ms.TriggerResolver(sdk.WrapSDKContext(ctx), &types.MsgTriggerResolver{
		Corporation: corporation, Operator: operator, Id: pid,
	})
	require.ErrorContains(t, err, "authorization check failed")
}

// TestTriggerResolver_EmptyDidAllowed — the spec (MOD-PP-MSG-15-2-1) does not
// require a non-empty did, so an active participant with no did still triggers.
func TestTriggerResolver_EmptyDidAllowed(t *testing.T) {
	k, ms, ctx, trk, del := setupTriggerResolver(t)
	corporation := sdk.AccAddress([]byte("corp_trigger________")).String()
	operator := sdk.AccAddress([]byte("op_trigger__________")).String()
	corpID := trk.RegisterCorp(corporation)
	past := ctx.BlockTime().Add(-time.Hour)

	pid, err := k.CreateParticipant(ctx, activeParticipant(corpID, operator, "", 0, past))
	require.NoError(t, err)

	del.ErrToReturn = nil // Path 1 (operator == vs_operator) passes
	_, err = ms.TriggerResolver(sdk.WrapSDKContext(ctx), &types.MsgTriggerResolver{
		Corporation: corporation, Operator: operator, Id: pid,
	})
	require.NoError(t, err)
}

// TestTriggerResolver_RejectInactive — participant must be active.
func TestTriggerResolver_RejectInactive(t *testing.T) {
	k, ms, ctx, trk, del := setupTriggerResolver(t)
	corporation := sdk.AccAddress([]byte("corp_trigger________")).String()
	operator := sdk.AccAddress([]byte("op_trigger__________")).String()
	corpID := trk.RegisterCorp(corporation)
	future := ctx.BlockTime().Add(time.Hour)

	// effective_from in the future => not active.
	pid, err := k.CreateParticipant(ctx, activeParticipant(corpID, operator, "did:example:holder", 0, future))
	require.NoError(t, err)

	del.ErrToReturn = nil
	_, err = ms.TriggerResolver(sdk.WrapSDKContext(ctx), &types.MsgTriggerResolver{
		Corporation: corporation, Operator: operator, Id: pid,
	})
	require.ErrorContains(t, err, "not active")
}

// TestTriggerResolver_RejectNotFound — unknown participant id.
func TestTriggerResolver_RejectNotFound(t *testing.T) {
	_, ms, ctx, trk, del := setupTriggerResolver(t)
	corporation := sdk.AccAddress([]byte("corp_trigger________")).String()
	operator := sdk.AccAddress([]byte("op_trigger__________")).String()
	trk.RegisterCorp(corporation)

	del.ErrToReturn = nil
	_, err := ms.TriggerResolver(sdk.WrapSDKContext(ctx), &types.MsgTriggerResolver{
		Corporation: corporation, Operator: operator, Id: 999,
	})
	require.ErrorContains(t, err, "not found")
}
