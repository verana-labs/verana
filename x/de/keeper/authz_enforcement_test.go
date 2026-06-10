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

// [AUTHZ-CHECK-1] step 2: a period-bearing operator authorization auto-renews on
// the first check after expiry instead of returning ErrAuthzExpired (#324).
func TestOperatorAuthzPeriodRenewal(t *testing.T) {
	f, ms, ctx := setupMsgServer(t)
	qs := keeper.NewQueryServerImpl(f.keeper)
	corporation := acc("corp________________")
	grantee := acc("grantee_____________")
	now := ctx.BlockTime()

	period := time.Hour
	exp := now.Add(time.Hour)
	spend := sdk.NewCoins(sdk.NewInt64Coin("uvna", 100))
	fee := sdk.NewCoins(sdk.NewInt64Coin("uvna", 50))

	_, err := ms.GrantOperatorAuthorization(ctx, &types.MsgGrantOperatorAuthorization{
		Corporation:           corporation,
		Grantee:               grantee,
		MsgTypes:              []string{mtEcosystem},
		AuthzSpendLimit:       spend,
		FeeSpendLimit:         fee,
		AuthzSpendLimitPeriod: &period,
		Expiration:            &exp,
	})
	require.NoError(t, err)

	list, err := qs.ListOperatorAuthorizations(ctx, &types.QueryListOperatorAuthorizationsRequest{Operator: grantee})
	require.NoError(t, err)
	require.Len(t, list.OperatorAuthorizations, 1)
	id := list.OperatorAuthorizations[0].Id
	// Seeded at grant time.
	require.Equal(t, spend, list.OperatorAuthorizations[0].RemainingSpend)
	require.Equal(t, fee, list.OperatorAuthorizations[0].RemainingFeeSpend)

	// Check well past expiry: must succeed (auto-renew), not abort.
	checkAt := now.Add(90 * time.Minute)
	require.NoError(t, f.keeper.CheckOperatorAuthorization(ctx, corporation, grantee, mtEcosystem, checkAt))

	got, err := qs.GetOperatorAuthorization(ctx, &types.QueryGetOperatorAuthorizationRequest{Id: id})
	require.NoError(t, err)
	require.True(t, got.OperatorAuthorization.Expiration.After(checkAt), "expiration rolled forward")
	require.Equal(t, spend, got.OperatorAuthorization.RemainingSpend, "remaining reset on renewal")
	require.Equal(t, fee, got.OperatorAuthorization.RemainingFeeSpend)
}

// A non-period authorization still aborts once expired.
func TestOperatorAuthzExpiresWithoutPeriod(t *testing.T) {
	f, ms, ctx := setupMsgServer(t)
	corporation := acc("corp________________")
	grantee := acc("grantee_____________")
	now := ctx.BlockTime()
	exp := now.Add(time.Hour)

	_, err := ms.GrantOperatorAuthorization(ctx, &types.MsgGrantOperatorAuthorization{
		Corporation: corporation, Grantee: grantee, MsgTypes: []string{mtEcosystem}, Expiration: &exp,
	})
	require.NoError(t, err)

	err = f.keeper.CheckOperatorAuthorization(ctx, corporation, grantee, mtEcosystem, now.Add(2*time.Hour))
	require.ErrorIs(t, err, types.ErrAuthzExpired)
}

// [MOD-DE-MSG-1] FeeGrant seeds remaining_spend at grant.
func TestFeeGrantSeedsRemaining(t *testing.T) {
	f, _, ctx := setupMsgServer(t)
	k := f.keeper
	grantee := acc("grantee_____________")
	corpID := uint64(7)
	spend := sdk.NewCoins(sdk.NewInt64Coin("uvna", 500))

	require.NoError(t, k.GrantFeeAllowance(ctx, corpID, grantee, []string{mtEcosystem}, nil, spend, nil))
	fg, err := k.FeeGrants.Get(ctx, collections.Join(corpID, grantee))
	require.NoError(t, err)
	require.Equal(t, spend, fg.RemainingSpend)
}

// [MOD-DE-MSG-5] VSOA record seeds remaining_spend/remaining_fee_spend at creation.
func TestVSOARecordSeedsRemaining(t *testing.T) {
	f, _, ctx := setupMsgServer(t)
	k := f.keeper
	corpID := uint64(1)
	vsOp := acc("vsop________________")
	spend := sdk.NewCoins(sdk.NewInt64Coin("uvna", 200))
	fee := sdk.NewCoins(sdk.NewInt64Coin("uvna", 80))

	require.NoError(t, k.GrantVSOperatorAuthorization(ctx, corpID, vsOp, types.ParticipantAuthorizationRecord{
		ParticipantId: 10, MsgTypes: []string{mtCSPS}, SpendLimit: spend, FeeSpendLimit: fee,
	}))

	vsoaID, err := k.VSOAByParticipant.Get(ctx, 10)
	require.NoError(t, err)
	vsoa, err := k.VSOperatorAuthorizations.Get(ctx, vsoaID)
	require.NoError(t, err)
	require.Len(t, vsoa.Records, 1)
	require.Equal(t, spend, vsoa.Records[0].RemainingSpend)
	require.Equal(t, fee, vsoa.Records[0].RemainingFeeSpend)
}
