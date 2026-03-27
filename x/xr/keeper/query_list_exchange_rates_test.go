package keeper_test

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	cstypes "github.com/verana-labs/verana/x/cs/types"
	"github.com/verana-labs/verana/x/xr/keeper"
	"github.com/verana-labs/verana/x/xr/types"
)

func seedMultipleExchangeRates(t *testing.T, f *fixture) {
	t.Helper()
	ms := keeper.NewMsgServerImpl(f.keeper)
	authorityStr, err := f.addressCodec.BytesToString(f.keeper.GetAuthority())
	require.NoError(t, err)

	// Rate 1: COIN/uverana -> FIAT/USD
	_, err = ms.CreateExchangeRate(f.ctx, &types.MsgCreateExchangeRate{
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

	// Rate 2: COIN/uverana -> FIAT/EUR
	_, err = ms.CreateExchangeRate(f.ctx, &types.MsgCreateExchangeRate{
		Authority:        authorityStr,
		BaseAssetType:    cstypes.PricingAssetType_COIN,
		BaseAsset:        "uverana",
		QuoteAssetType:   cstypes.PricingAssetType_FIAT,
		QuoteAsset:       "EUR",
		Rate:             "450",
		RateScale:        2,
		ValidityDuration: 10 * time.Minute,
	})
	require.NoError(t, err)

	// Rate 3: TU/TU -> FIAT/USD (activate it)
	resp, err := ms.CreateExchangeRate(f.ctx, &types.MsgCreateExchangeRate{
		Authority:        authorityStr,
		BaseAssetType:    cstypes.PricingAssetType_TU,
		BaseAsset:        "TU",
		QuoteAssetType:   cstypes.PricingAssetType_FIAT,
		QuoteAsset:       "USD",
		Rate:             "100",
		RateScale:        2,
		ValidityDuration: 10 * time.Minute,
	})
	require.NoError(t, err)

	// Set rate 3 to active
	xr, err := f.keeper.ExchangeRates.Get(f.ctx, resp.Id)
	require.NoError(t, err)
	xr.State = true
	err = f.keeper.ExchangeRates.Set(f.ctx, xr.Id, xr)
	require.NoError(t, err)
}

func TestListExchangeRates_All(t *testing.T) {
	f := initFixture(t)
	qs := keeper.NewQueryServerImpl(f.keeper)

	seedMultipleExchangeRates(t, f)

	resp, err := qs.ListExchangeRates(f.ctx, &types.QueryListExchangeRatesRequest{})
	require.NoError(t, err)
	require.Len(t, resp.ExchangeRates, 3)
}

func TestListExchangeRates_FilterByAssetType(t *testing.T) {
	f := initFixture(t)
	qs := keeper.NewQueryServerImpl(f.keeper)

	seedMultipleExchangeRates(t, f)

	// Filter by base_asset_type = TU
	resp, err := qs.ListExchangeRates(f.ctx, &types.QueryListExchangeRatesRequest{
		BaseAssetType: cstypes.PricingAssetType_TU,
	})
	require.NoError(t, err)
	require.Len(t, resp.ExchangeRates, 1)
	require.Equal(t, "TU", resp.ExchangeRates[0].BaseAsset)

	// Filter by quote_asset = EUR
	resp, err = qs.ListExchangeRates(f.ctx, &types.QueryListExchangeRatesRequest{
		QuoteAsset: "EUR",
	})
	require.NoError(t, err)
	require.Len(t, resp.ExchangeRates, 1)
	require.Equal(t, "EUR", resp.ExchangeRates[0].QuoteAsset)

	// Filter by state = active (only rate 3 is active)
	resp, err = qs.ListExchangeRates(f.ctx, &types.QueryListExchangeRatesRequest{
		State: types.StateFilter_STATE_FILTER_ACTIVE,
	})
	require.NoError(t, err)
	require.Len(t, resp.ExchangeRates, 1)
	require.Equal(t, "TU", resp.ExchangeRates[0].BaseAsset)
}

func TestListExchangeRates_EmptyResult(t *testing.T) {
	f := initFixture(t)
	qs := keeper.NewQueryServerImpl(f.keeper)

	resp, err := qs.ListExchangeRates(f.ctx, &types.QueryListExchangeRatesRequest{})
	require.NoError(t, err)
	require.Empty(t, resp.ExchangeRates)
}
