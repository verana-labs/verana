package keeper_test

import (
	"testing"
	"time"

	"cosmossdk.io/collections"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/stretchr/testify/require"

	cstypes "github.com/verana-labs/verana/x/cs/types"
	"github.com/verana-labs/verana/x/xr/keeper"
	"github.com/verana-labs/verana/x/xr/types"
)

// createActiveExchangeRateWithValidity creates an active rate with a custom validity duration.
func createActiveExchangeRateWithValidity(t *testing.T, f *fixture, ms types.MsgServer, validity time.Duration) uint64 {
	t.Helper()
	authorityStr, err := f.addressCodec.BytesToString(f.keeper.GetAuthority())
	require.NoError(t, err)

	resp, err := ms.CreateExchangeRate(f.ctx, &types.MsgCreateExchangeRate{
		Authority:        authorityStr,
		BaseAssetType:    cstypes.PricingAssetType_COIN,
		BaseAsset:        "uverana",
		QuoteAssetType:   cstypes.PricingAssetType_FIAT,
		QuoteAsset:       "EUR",
		Rate:             "100",
		RateScale:        2,
		ValidityDuration: validity,
	})
	require.NoError(t, err)

	xr, err := f.keeper.ExchangeRates.Get(f.ctx, resp.Id)
	require.NoError(t, err)
	xr.State = true
	require.NoError(t, f.keeper.ExchangeRates.Set(f.ctx, resp.Id, xr))
	return resp.Id
}

func TestGrantExchangeRateAuthorization_Success(t *testing.T) {
	f := initFixture(t)
	ms := keeper.NewMsgServerImpl(f.keeper)

	id := createActiveExchangeRate(t, f, ms)
	authorityStr, err := f.addressCodec.BytesToString(f.keeper.GetAuthority())
	require.NoError(t, err)
	op := stdOperator()
	exp := sdk.UnwrapSDKContext(f.ctx).BlockTime().Add(time.Hour)
	minInterval := 5 * time.Minute

	_, err = ms.GrantExchangeRateAuthorization(f.ctx, &types.MsgGrantExchangeRateAuthorization{
		Authority:       authorityStr,
		XrId:            id,
		Operator:        op,
		Expiration:      &exp,
		MinInterval:     &minInterval,
		MaxDeviationBps: 500,
	})
	require.NoError(t, err)

	stored, err := f.keeper.ExchangeRateAuthorizations.Get(f.ctx, collections.Join(id, op))
	require.NoError(t, err)
	require.Equal(t, op, stored.Operator)
	require.Equal(t, id, stored.XrId)
	require.Equal(t, uint32(500), stored.MaxDeviationBps)
	require.Equal(t, minInterval, stored.MinInterval)
}

func TestGrantExchangeRateAuthorization_NonAuthority(t *testing.T) {
	f := initFixture(t)
	ms := keeper.NewMsgServerImpl(f.keeper)

	id := createActiveExchangeRate(t, f, ms)
	notGov := sdk.AccAddress([]byte("not_gov_authority___")).String()
	exp := sdk.UnwrapSDKContext(f.ctx).BlockTime().Add(time.Hour)

	_, err := ms.GrantExchangeRateAuthorization(f.ctx, &types.MsgGrantExchangeRateAuthorization{
		Authority:  notGov,
		XrId:       id,
		Operator:   stdOperator(),
		Expiration: &exp,
	})
	require.ErrorIs(t, err, types.ErrInvalidSigner)
}

func TestGrantExchangeRateAuthorization_ExpirationPast(t *testing.T) {
	f := initFixture(t)
	ms := keeper.NewMsgServerImpl(f.keeper)

	id := createActiveExchangeRate(t, f, ms)
	authorityStr, err := f.addressCodec.BytesToString(f.keeper.GetAuthority())
	require.NoError(t, err)
	exp := sdk.UnwrapSDKContext(f.ctx).BlockTime().Add(-time.Hour)

	_, err = ms.GrantExchangeRateAuthorization(f.ctx, &types.MsgGrantExchangeRateAuthorization{
		Authority:  authorityStr,
		XrId:       id,
		Operator:   stdOperator(),
		Expiration: &exp,
	})
	require.ErrorIs(t, err, types.ErrInvalidExpiration)
}

func TestGrantExchangeRateAuthorization_InvalidXrId(t *testing.T) {
	f := initFixture(t)
	ms := keeper.NewMsgServerImpl(f.keeper)

	authorityStr, err := f.addressCodec.BytesToString(f.keeper.GetAuthority())
	require.NoError(t, err)
	exp := sdk.UnwrapSDKContext(f.ctx).BlockTime().Add(time.Hour)

	_, err = ms.GrantExchangeRateAuthorization(f.ctx, &types.MsgGrantExchangeRateAuthorization{
		Authority:  authorityStr,
		XrId:       999,
		Operator:   stdOperator(),
		Expiration: &exp,
	})
	require.ErrorIs(t, err, types.ErrExchangeRateNotFound)
}

func TestGrantExchangeRateAuthorization_MaxDeviationOutOfRange(t *testing.T) {
	f := initFixture(t)
	ms := keeper.NewMsgServerImpl(f.keeper)

	id := createActiveExchangeRate(t, f, ms)
	authorityStr, err := f.addressCodec.BytesToString(f.keeper.GetAuthority())
	require.NoError(t, err)
	exp := sdk.UnwrapSDKContext(f.ctx).BlockTime().Add(time.Hour)

	_, err = ms.GrantExchangeRateAuthorization(f.ctx, &types.MsgGrantExchangeRateAuthorization{
		Authority:       authorityStr,
		XrId:            id,
		Operator:        stdOperator(),
		Expiration:      &exp,
		MaxDeviationBps: 10001,
	})
	require.ErrorIs(t, err, types.ErrInvalidMaxDeviation)
}

func TestRevokeExchangeRateAuthorization_Success(t *testing.T) {
	f := initFixture(t)
	ms := keeper.NewMsgServerImpl(f.keeper)

	id := createActiveExchangeRate(t, f, ms)
	authorityStr, err := f.addressCodec.BytesToString(f.keeper.GetAuthority())
	require.NoError(t, err)
	op := stdOperator()
	grantXRAuthz(t, f, ms, id, op, 0, 0)

	_, err = ms.RevokeExchangeRateAuthorization(f.ctx, &types.MsgRevokeExchangeRateAuthorization{
		Authority: authorityStr,
		XrId:      id,
		Operator:  op,
	})
	require.NoError(t, err)

	// Update must now fail (no authorization).
	_, err = ms.UpdateExchangeRate(f.ctx, &types.MsgUpdateExchangeRate{Operator: op, Id: id, Rate: "200"})
	require.ErrorIs(t, err, types.ErrAuthorizationNotFound)
}

func TestRevokeExchangeRateAuthorization_Missing(t *testing.T) {
	f := initFixture(t)
	ms := keeper.NewMsgServerImpl(f.keeper)

	id := createActiveExchangeRate(t, f, ms)
	authorityStr, err := f.addressCodec.BytesToString(f.keeper.GetAuthority())
	require.NoError(t, err)

	_, err = ms.RevokeExchangeRateAuthorization(f.ctx, &types.MsgRevokeExchangeRateAuthorization{
		Authority: authorityStr,
		XrId:      id,
		Operator:  stdOperator(),
	})
	require.ErrorIs(t, err, types.ErrAuthorizationNotFound)
}

func TestUpdateExchangeRate_AuthzExpired(t *testing.T) {
	f := initFixture(t)
	ms := keeper.NewMsgServerImpl(f.keeper)

	id := createActiveExchangeRateWithValidity(t, f, ms, 3*time.Hour)
	op := stdOperator()
	grantXRAuthz(t, f, ms, id, op, 0, 0) // expiration = now + 1h

	// Advance block time past the authorization expiration (now + 2h).
	blockTime := sdk.UnwrapSDKContext(f.ctx).BlockTime()
	ctx2 := sdk.UnwrapSDKContext(f.ctx).WithBlockTime(blockTime.Add(2 * time.Hour))

	_, err := ms.UpdateExchangeRate(ctx2, &types.MsgUpdateExchangeRate{Operator: op, Id: id, Rate: "200"})
	require.ErrorIs(t, err, types.ErrAuthorizationExpired)
}

func TestUpdateExchangeRate_MinIntervalAntiSpam(t *testing.T) {
	f := initFixture(t)
	ms := keeper.NewMsgServerImpl(f.keeper)

	id := createActiveExchangeRate(t, f, ms) // Updated = blockTime
	op := stdOperator()
	grantXRAuthz(t, f, ms, id, op, 10*time.Minute, 0)

	// Update at the same block time: 0 elapsed < 10m min_interval -> rejected.
	_, err := ms.UpdateExchangeRate(f.ctx, &types.MsgUpdateExchangeRate{Operator: op, Id: id, Rate: "200"})
	require.ErrorIs(t, err, types.ErrUpdateTooSoon)
}

func TestUpdateExchangeRate_MinIntervalElapsed(t *testing.T) {
	f := initFixture(t)
	ms := keeper.NewMsgServerImpl(f.keeper)

	id := createActiveExchangeRateWithValidity(t, f, ms, 3*time.Hour)
	op := stdOperator()
	grantXRAuthz(t, f, ms, id, op, 10*time.Minute, 0)

	// Advance 11 minutes (> min_interval, < validity and authz expiration).
	blockTime := sdk.UnwrapSDKContext(f.ctx).BlockTime()
	ctx2 := sdk.UnwrapSDKContext(f.ctx).WithBlockTime(blockTime.Add(11 * time.Minute))

	_, err := ms.UpdateExchangeRate(ctx2, &types.MsgUpdateExchangeRate{Operator: op, Id: id, Rate: "200"})
	require.NoError(t, err)
}

func TestUpdateExchangeRate_MaxDeviationCircuitBreaker(t *testing.T) {
	f := initFixture(t)
	ms := keeper.NewMsgServerImpl(f.keeper)

	id := createActiveExchangeRate(t, f, ms) // rate = 100
	op := stdOperator()
	grantXRAuthz(t, f, ms, id, op, 0, 1000) // 10% max deviation

	// 100 -> 200 is +100% > 10% -> rejected.
	_, err := ms.UpdateExchangeRate(f.ctx, &types.MsgUpdateExchangeRate{Operator: op, Id: id, Rate: "200"})
	require.ErrorIs(t, err, types.ErrRateDeviationExceeded)

	// 100 -> 105 is +5% <= 10% -> allowed.
	_, err = ms.UpdateExchangeRate(f.ctx, &types.MsgUpdateExchangeRate{Operator: op, Id: id, Rate: "105"})
	require.NoError(t, err)
}

func TestGetExchangeRate_IncludesAuthorizations(t *testing.T) {
	f := initFixture(t)
	ms := keeper.NewMsgServerImpl(f.keeper)
	qs := keeper.NewQueryServerImpl(f.keeper)

	id := createActiveExchangeRate(t, f, ms)
	op := stdOperator()
	grantXRAuthz(t, f, ms, id, op, 0, 0)

	resp, err := qs.GetExchangeRate(f.ctx, &types.QueryGetExchangeRateRequest{Id: id})
	require.NoError(t, err)
	require.Len(t, resp.Authorizations, 1)
	require.Equal(t, op, resp.Authorizations[0].Operator)
	require.Equal(t, id, resp.Authorizations[0].XrId)
}
