package keeper_test

import (
	"testing"
	"time"

	"cosmossdk.io/collections"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/stretchr/testify/require"

	"github.com/verana-labs/verana/x/de/keeper"
	"github.com/verana-labs/verana/x/de/types"
)

func setupMsgServer(t *testing.T) (*fixture, types.MsgServer, sdk.Context) {
	t.Helper()
	f := initFixture(t)
	ctx := sdk.UnwrapSDKContext(f.ctx)
	return f, keeper.NewMsgServerImpl(f.keeper), ctx
}

func acc(s string) string { return sdk.AccAddress([]byte(s)).String() }

// vpr msg types used in records / fee grants (must be VPR delegable).
const (
	mtEcosystem = "/verana.ec.v1.MsgCreateEcosystem"
	mtSchema    = "/verana.cs.v1.MsgCreateCredentialSchema"
	mtValidated = "/verana.pp.v1.MsgSetParticipantOPToValidated"
	mtCSPS      = "/verana.pp.v1.MsgCreateOrUpdateParticipantSession"
)

func TestMsgServer(t *testing.T) {
	_, ms, ctx := setupMsgServer(t)
	require.NotNil(t, ms)
	require.NotNil(t, ctx)
}

// ---------------------------------------------------------------------------
// [MOD-DE-MSG-1/2] Fee allowance (internal keeper methods, by corporation_id)
// ---------------------------------------------------------------------------

func TestGrantAndRevokeFeeAllowance(t *testing.T) {
	f, _, ctx := setupMsgServer(t)
	k := f.keeper
	grantee := acc("grantee_____________")
	corpID := uint64(7)
	msgTypes := []string{mtEcosystem}

	require.NoError(t, k.GrantFeeAllowance(ctx, corpID, grantee, msgTypes, nil, nil, nil))
	fg, err := k.FeeGrants.Get(ctx, collections.Join(corpID, grantee))
	require.NoError(t, err)
	require.Equal(t, corpID, fg.GrantorCorporationId)
	require.Equal(t, grantee, fg.Grantee)
	require.Equal(t, msgTypes, fg.MsgTypes)

	// Update in place with new msg types.
	require.NoError(t, k.GrantFeeAllowance(ctx, corpID, grantee, []string{mtSchema}, nil, nil, nil))
	fg, err = k.FeeGrants.Get(ctx, collections.Join(corpID, grantee))
	require.NoError(t, err)
	require.Equal(t, []string{mtSchema}, fg.MsgTypes)

	require.NoError(t, k.RevokeFeeAllowance(ctx, corpID, grantee))
	has, err := k.FeeGrants.Has(ctx, collections.Join(corpID, grantee))
	require.NoError(t, err)
	require.False(t, has)
}

func TestFeeAllowance_Validation(t *testing.T) {
	f, _, ctx := setupMsgServer(t)
	k := f.keeper
	grantee := acc("grantee_____________")

	// non-delegable msg type
	require.Error(t, k.GrantFeeAllowance(ctx, 1, grantee, []string{"/verana.de.v1.MsgUpdateParams"}, nil, nil, nil))
	// empty msg types
	require.Error(t, k.GrantFeeAllowance(ctx, 1, grantee, nil, nil, nil, nil))

	require.ErrorContains(t, k.RevokeFeeAllowance(ctx, 0, grantee), "grantor_corporation_id")
	require.ErrorContains(t, k.RevokeFeeAllowance(ctx, 1, ""), "grantee")
	// no-op when absent
	require.NoError(t, k.RevokeFeeAllowance(ctx, 1, acc("absent______________")))
}

// ---------------------------------------------------------------------------
// [MOD-DE-MSG-3/4] Grant / Revoke Operator Authorization (msg server)
// ---------------------------------------------------------------------------

func TestGrantOperatorAuthorization_NewAndUpdate(t *testing.T) {
	f, ms, ctx := setupMsgServer(t)
	qs := keeper.NewQueryServerImpl(f.keeper)
	corporation := acc("corp________________")
	grantee := acc("grantee_____________")

	_, err := ms.GrantOperatorAuthorization(ctx, &types.MsgGrantOperatorAuthorization{
		Corporation: corporation, Grantee: grantee, MsgTypes: []string{mtEcosystem},
	})
	require.NoError(t, err)

	list, err := qs.ListOperatorAuthorizations(ctx, &types.QueryListOperatorAuthorizationsRequest{Operator: grantee})
	require.NoError(t, err)
	require.Len(t, list.OperatorAuthorizations, 1)
	firstID := list.OperatorAuthorizations[0].Id
	require.NotZero(t, firstID)
	require.Equal(t, grantee, list.OperatorAuthorizations[0].Operator)

	// Re-grant is an in-place update that preserves the id.
	_, err = ms.GrantOperatorAuthorization(ctx, &types.MsgGrantOperatorAuthorization{
		Corporation: corporation, Grantee: grantee, MsgTypes: []string{mtSchema},
	})
	require.NoError(t, err)
	list, err = qs.ListOperatorAuthorizations(ctx, &types.QueryListOperatorAuthorizationsRequest{Operator: grantee})
	require.NoError(t, err)
	require.Len(t, list.OperatorAuthorizations, 1)
	require.Equal(t, firstID, list.OperatorAuthorizations[0].Id)
	require.Equal(t, []string{mtSchema}, list.OperatorAuthorizations[0].MsgTypes)

	// [MOD-DE-QRY-3] get by id.
	got, err := qs.GetOperatorAuthorization(ctx, &types.QueryGetOperatorAuthorizationRequest{Id: firstID})
	require.NoError(t, err)
	require.Equal(t, grantee, got.OperatorAuthorization.Operator)
}

func TestGrantOperatorAuthorization_MutualExclusivity(t *testing.T) {
	f, ms, ctx := setupMsgServer(t)
	corporation := acc("corp________________")
	grantee := acc("grantee_____________")

	// Resolve the signing corporation to its co.id (same path MSG-3 uses).
	co, err := f.corpKeeper.ResolveCorporationByPolicyAddress(ctx, corporation)
	require.NoError(t, err)
	// A VSOperatorAuthorization for (co.id, grantee) blocks the operator grant.
	require.NoError(t, f.keeper.GrantVSOperatorAuthorization(ctx, co.Id, grantee,
		types.ParticipantAuthorizationRecord{ParticipantId: 1, MsgTypes: []string{mtValidated}}))

	_, err = ms.GrantOperatorAuthorization(ctx, &types.MsgGrantOperatorAuthorization{
		Corporation: corporation, Grantee: grantee, MsgTypes: []string{mtEcosystem},
	})
	require.ErrorIs(t, err, types.ErrVSOperatorAuthzExists)
}

func TestRevokeOperatorAuthorization(t *testing.T) {
	f, ms, ctx := setupMsgServer(t)
	qs := keeper.NewQueryServerImpl(f.keeper)
	corporation := acc("corp________________")
	grantee := acc("grantee_____________")

	_, err := ms.GrantOperatorAuthorization(ctx, &types.MsgGrantOperatorAuthorization{
		Corporation: corporation, Grantee: grantee, MsgTypes: []string{mtEcosystem},
	})
	require.NoError(t, err)

	_, err = ms.RevokeOperatorAuthorization(ctx, &types.MsgRevokeOperatorAuthorization{
		Corporation: corporation, Grantee: grantee,
	})
	require.NoError(t, err)
	list, err := qs.ListOperatorAuthorizations(ctx, &types.QueryListOperatorAuthorizationsRequest{Operator: grantee})
	require.NoError(t, err)
	require.Empty(t, list.OperatorAuthorizations)

	// Revoking a missing entry aborts.
	_, err = ms.RevokeOperatorAuthorization(ctx, &types.MsgRevokeOperatorAuthorization{
		Corporation: corporation, Grantee: grantee,
	})
	require.ErrorIs(t, err, types.ErrOperatorAuthzNotFound)
}

// ---------------------------------------------------------------------------
// [MOD-DE-MSG-5/6/9] VS Operator Authorization (module-call keeper methods)
// ---------------------------------------------------------------------------

func TestGrantVSOperatorAuthorization(t *testing.T) {
	f, _, ctx := setupMsgServer(t)
	k := f.keeper
	corpID := uint64(1)
	vsOp := acc("vsop________________")
	rec := func(pid uint64) types.ParticipantAuthorizationRecord {
		return types.ParticipantAuthorizationRecord{ParticipantId: pid, MsgTypes: []string{mtValidated}}
	}

	require.NoError(t, k.GrantVSOperatorAuthorization(ctx, corpID, vsOp, rec(10)))
	require.NoError(t, k.GrantVSOperatorAuthorization(ctx, corpID, vsOp, rec(11)))

	// Duplicate participant_id is rejected (global uniqueness).
	require.ErrorIs(t, k.GrantVSOperatorAuthorization(ctx, corpID, vsOp, rec(10)), types.ErrParticipantRecordExists)
	// Non-delegable msg type is rejected.
	require.Error(t, k.GrantVSOperatorAuthorization(ctx, corpID, vsOp,
		types.ParticipantAuthorizationRecord{ParticipantId: 12, MsgTypes: []string{"/verana.de.v1.MsgUpdateParams"}}))

	vsoaID, err := k.VSOAByParticipant.Get(ctx, 10)
	require.NoError(t, err)
	vsoa, err := k.VSOperatorAuthorizations.Get(ctx, vsoaID)
	require.NoError(t, err)
	require.Len(t, vsoa.Records, 2)
	require.Equal(t, corpID, vsoa.CorporationId)
}

func TestGrantVSOperatorAuthorization_MutualExclusivity(t *testing.T) {
	f, _, ctx := setupMsgServer(t)
	k := f.keeper
	corpID := uint64(1)
	vsOp := acc("vsop________________")
	// Seed an OperatorAuthorization for (corpID, vsOp).
	require.NoError(t, k.OperatorAuthorizations.Set(ctx, 99, types.OperatorAuthorization{Id: 99, CorporationId: corpID, Operator: vsOp, MsgTypes: []string{mtEcosystem}}))
	require.NoError(t, k.OperatorAuthorizationByCorpOp.Set(ctx, collections.Join(corpID, vsOp), 99))

	err := k.GrantVSOperatorAuthorization(ctx, corpID, vsOp,
		types.ParticipantAuthorizationRecord{ParticipantId: 1, MsgTypes: []string{mtValidated}})
	require.ErrorIs(t, err, types.ErrOperatorAuthzExistsMutex)
}

func TestGrantVSOperatorAuthorization_SingleCorp(t *testing.T) {
	f, _, ctx := setupMsgServer(t)
	k := f.keeper
	vsOp := acc("vsop________________")
	require.NoError(t, k.GrantVSOperatorAuthorization(ctx, 1, vsOp,
		types.ParticipantAuthorizationRecord{ParticipantId: 1, MsgTypes: []string{mtValidated}}))
	// A different corporation cannot also authorize the same vs_operator.
	err := k.GrantVSOperatorAuthorization(ctx, 2, vsOp,
		types.ParticipantAuthorizationRecord{ParticipantId: 2, MsgTypes: []string{mtValidated}})
	require.ErrorIs(t, err, types.ErrVSOAOtherCorporation)
}

func TestRevokeVSOperatorAuthorization(t *testing.T) {
	f, _, ctx := setupMsgServer(t)
	k := f.keeper
	corpID := uint64(1)
	vsOp := acc("vsop________________")
	mt := []string{mtValidated}
	require.NoError(t, k.GrantVSOperatorAuthorization(ctx, corpID, vsOp, types.ParticipantAuthorizationRecord{ParticipantId: 10, MsgTypes: mt}))
	require.NoError(t, k.GrantVSOperatorAuthorization(ctx, corpID, vsOp, types.ParticipantAuthorizationRecord{ParticipantId: 11, MsgTypes: mt}))
	vsoaID, _ := k.VSOAByParticipant.Get(ctx, 10)

	// Removing one record leaves the VSOA in place.
	require.NoError(t, k.RevokeVSOperatorAuthorization(ctx, 10))
	vsoa, err := k.VSOperatorAuthorizations.Get(ctx, vsoaID)
	require.NoError(t, err)
	require.Len(t, vsoa.Records, 1)

	// Removing the last record deletes the VSOA.
	require.NoError(t, k.RevokeVSOperatorAuthorization(ctx, 11))
	_, err = k.VSOperatorAuthorizations.Get(ctx, vsoaID)
	require.Error(t, err)

	// No-op when no record exists.
	require.NoError(t, k.RevokeVSOperatorAuthorization(ctx, 999))

	// Re-granting the same pair mints a fresh vsoa id.
	require.NoError(t, k.GrantVSOperatorAuthorization(ctx, corpID, vsOp, types.ParticipantAuthorizationRecord{ParticipantId: 12, MsgTypes: mt}))
	newID, _ := k.VSOAByParticipant.Get(ctx, 12)
	require.NotEqual(t, vsoaID, newID)
}

func TestUpdateVSOperatorAuthorizationExpiration(t *testing.T) {
	f, _, ctx := setupMsgServer(t)
	k := f.keeper
	vsOp := acc("vsop________________")
	now := ctx.BlockTime()
	require.NoError(t, k.GrantVSOperatorAuthorization(ctx, 1, vsOp,
		types.ParticipantAuthorizationRecord{ParticipantId: 10, MsgTypes: []string{mtValidated}, Expiration: &now}))

	future := now.Add(48 * time.Hour)
	require.NoError(t, k.UpdateVSOperatorAuthorizationExpiration(ctx, 10, future))
	vsoaID, _ := k.VSOAByParticipant.Get(ctx, 10)
	vsoa, _ := k.VSOperatorAuthorizations.Get(ctx, vsoaID)
	require.NotNil(t, vsoa.Records[0].Expiration)
	require.True(t, vsoa.Records[0].Expiration.Equal(future))

	// No-op when no record exists.
	require.NoError(t, k.UpdateVSOperatorAuthorizationExpiration(ctx, 999, future))
}

// [MOD-DE-MSG-5-5] Recompute drives the chain-level fee allowance from records.
func TestRecomputeFeeAllowance(t *testing.T) {
	f, _, ctx := setupMsgServer(t)
	k := f.keeper
	corpID := uint64(1)
	vsOp := acc("vsop________________")
	now := ctx.BlockTime()
	future := now.Add(24 * time.Hour)

	// A with_feegrant record with a future expiration grants the allowance.
	require.NoError(t, k.GrantVSOperatorAuthorization(ctx, corpID, vsOp,
		types.ParticipantAuthorizationRecord{ParticipantId: 10, MsgTypes: []string{mtCSPS}, WithFeegrant: true, Expiration: &future}))
	has, err := k.FeeGrants.Has(ctx, collections.Join(corpID, vsOp))
	require.NoError(t, err)
	require.True(t, has)

	// Revoking it removes the allowance.
	require.NoError(t, k.RevokeVSOperatorAuthorization(ctx, 10))
	has, err = k.FeeGrants.Has(ctx, collections.Join(corpID, vsOp))
	require.NoError(t, err)
	require.False(t, has)
}

func TestQueriesNotFound(t *testing.T) {
	f, _, ctx := setupMsgServer(t)
	qs := keeper.NewQueryServerImpl(f.keeper)
	_, err := qs.GetOperatorAuthorization(ctx, &types.QueryGetOperatorAuthorizationRequest{Id: 999})
	require.Error(t, err)
	_, err = qs.GetVSOperatorAuthorization(ctx, &types.QueryGetVSOperatorAuthorizationRequest{Id: 999})
	require.Error(t, err)
}
