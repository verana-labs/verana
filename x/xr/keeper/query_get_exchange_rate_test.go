package keeper_test

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	cstypes "github.com/verana-labs/verana/x/cs/types"
	"github.com/verana-labs/verana/x/xr/keeper"
	"github.com/verana-labs/verana/x/xr/types"
)

func seedExchangeRate(t *testing.T, f *fixture, state bool, expiresFromNow time.Duration) types.ExchangeRate {
	t.Helper()
	ms := keeper.NewMsgServerImpl(f.keeper)
	authorityStr, err := f.addressCodec.BytesToString(f.keeper.GetAuthority())
	require.NoError(t, err)

	resp, err := ms.CreateExchangeRate(f.ctx, &types.MsgCreateExchangeRate{
		Authority:        authorityStr,
		BaseAssetType:    cstypes.PricingAssetType_COIN,
		BaseAsset:        "uverana",
		QuoteAssetType:   cstypes.PricingAssetType_FIAT,
		QuoteAsset:       "USD",
		Rate:             "500",
		RateScale:        2,
		ValidityDuration: 10 * time.Minute,
	})
	require.NoError(t, err)

	xr, err := f.keeper.ExchangeRates.Get(f.ctx, resp.Id)
	require.NoError(t, err)

	// Override state and expires for testing
	xr.State = state
	if expiresFromNow != 0 {
		xr.Expires = xr.Updated.Add(expiresFromNow)
	}
	err = f.keeper.ExchangeRates.Set(f.ctx, xr.Id, xr)
	require.NoError(t, err)

	return xr
}

func TestGetExchangeRate_ById(t *testing.T) {
	f := initFixture(t)
	qs := keeper.NewQueryServerImpl(f.keeper)

	xr := seedExchangeRate(t, f, true, 10*time.Minute)

	resp, err := qs.GetExchangeRate(f.ctx, &types.QueryGetExchangeRateRequest{
		Id: xr.Id,
	})
	require.NoError(t, err)
	require.Equal(t, xr.Id, resp.ExchangeRate.Id)
	require.Equal(t, "uverana", resp.ExchangeRate.BaseAsset)
}

func TestGetExchangeRate_ByPair(t *testing.T) {
	f := initFixture(t)
	qs := keeper.NewQueryServerImpl(f.keeper)

	xr := seedExchangeRate(t, f, true, 10*time.Minute)

	resp, err := qs.GetExchangeRate(f.ctx, &types.QueryGetExchangeRateRequest{
		BaseAssetType:  cstypes.PricingAssetType_COIN,
		BaseAsset:      "uverana",
		QuoteAssetType: cstypes.PricingAssetType_FIAT,
		QuoteAsset:     "USD",
	})
	require.NoError(t, err)
	require.Equal(t, xr.Id, resp.ExchangeRate.Id)
}

func TestGetExchangeRate_NotFound(t *testing.T) {
	f := initFixture(t)
	qs := keeper.NewQueryServerImpl(f.keeper)

	// By non-existent ID
	_, err := qs.GetExchangeRate(f.ctx, &types.QueryGetExchangeRateRequest{
		Id: 999,
	})
	require.Error(t, err)
	require.Contains(t, err.Error(), "not found")

	// By non-existent pair
	_, err = qs.GetExchangeRate(f.ctx, &types.QueryGetExchangeRateRequest{
		BaseAssetType:  cstypes.PricingAssetType_COIN,
		BaseAsset:      "uverana",
		QuoteAssetType: cstypes.PricingAssetType_FIAT,
		QuoteAsset:     "EUR",
	})
	require.Error(t, err)
	require.Contains(t, err.Error(), "not found")
}

func TestGetExchangeRate_StateFilter(t *testing.T) {
	f := initFixture(t)
	qs := keeper.NewQueryServerImpl(f.keeper)

	// Seed an inactive exchange rate
	seedExchangeRate(t, f, false, 10*time.Minute)

	// Filter for active should not find it
	_, err := qs.GetExchangeRate(f.ctx, &types.QueryGetExchangeRateRequest{
		Id:    1,
		State: types.StateFilter_STATE_FILTER_ACTIVE,
	})
	require.Error(t, err)
	require.Contains(t, err.Error(), "not found")

	// Filter for inactive should find it
	resp, err := qs.GetExchangeRate(f.ctx, &types.QueryGetExchangeRateRequest{
		Id:    1,
		State: types.StateFilter_STATE_FILTER_INACTIVE,
	})
	require.NoError(t, err)
	require.Equal(t, uint64(1), resp.ExchangeRate.Id)
}

func TestGetExchangeRate_NilRequest(t *testing.T) {
	f := initFixture(t)
	qs := keeper.NewQueryServerImpl(f.keeper)

	_, err := qs.GetExchangeRate(f.ctx, nil)
	require.Error(t, err)
	require.Contains(t, err.Error(), "invalid request")
}
