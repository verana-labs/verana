package keeper_test

import (
	"testing"
	"time"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/stretchr/testify/require"

	cstypes "github.com/verana-labs/verana/x/cs/types"
	"github.com/verana-labs/verana/x/xr/keeper"
	"github.com/verana-labs/verana/x/xr/types"
)

func seedPriceExchangeRate(t *testing.T, f *fixture, rate string, rateScale uint32, state bool, expired bool) {
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
		Rate:             rate,
		RateScale:        rateScale,
		ValidityDuration: 10 * time.Minute,
	})
	require.NoError(t, err)

	xr, err := f.keeper.ExchangeRates.Get(f.ctx, resp.Id)
	require.NoError(t, err)

	xr.State = state
	if expired {
		// Set expires to the past
		sdkCtx := sdk.UnwrapSDKContext(f.ctx)
		xr.Expires = sdkCtx.BlockTime().Add(-1 * time.Minute)
	}
	err = f.keeper.ExchangeRates.Set(f.ctx, xr.Id, xr)
	require.NoError(t, err)
}

func TestGetPrice_SameAsset(t *testing.T) {
	f := initFixture(t)
	qs := keeper.NewQueryServerImpl(f.keeper)

	resp, err := qs.GetPrice(f.ctx, &types.QueryGetPriceRequest{
		BaseAssetType:  cstypes.PricingAssetType_COIN,
		BaseAsset:      "uverana",
		QuoteAssetType: cstypes.PricingAssetType_COIN,
		QuoteAsset:     "uverana",
		Amount:         "1000",
	})
	require.NoError(t, err)
	require.Equal(t, "1000", resp.Price)
}

func TestGetPrice_ValidConversion(t *testing.T) {
	f := initFixture(t)
	qs := keeper.NewQueryServerImpl(f.keeper)

	// Rate=500, RateScale=2 => price = floor(amount * 500 / 100) = floor(amount * 5)
	seedPriceExchangeRate(t, f, "500", 2, true, false)

	resp, err := qs.GetPrice(f.ctx, &types.QueryGetPriceRequest{
		BaseAssetType:  cstypes.PricingAssetType_COIN,
		BaseAsset:      "uverana",
		QuoteAssetType: cstypes.PricingAssetType_FIAT,
		QuoteAsset:     "USD",
		Amount:         "100",
	})
	require.NoError(t, err)
	// 100 * 500 / 100 = 500
	require.Equal(t, "500", resp.Price)

	// Test with amount that results in floor division
	resp, err = qs.GetPrice(f.ctx, &types.QueryGetPriceRequest{
		BaseAssetType:  cstypes.PricingAssetType_COIN,
		BaseAsset:      "uverana",
		QuoteAssetType: cstypes.PricingAssetType_FIAT,
		QuoteAsset:     "USD",
		Amount:         "3",
	})
	require.NoError(t, err)
	// floor(3 * 500 / 100) = floor(1500/100) = 15
	require.Equal(t, "15", resp.Price)
}

func TestGetPrice_ExpiredRate(t *testing.T) {
	f := initFixture(t)
	qs := keeper.NewQueryServerImpl(f.keeper)

	seedPriceExchangeRate(t, f, "500", 2, true, true)

	_, err := qs.GetPrice(f.ctx, &types.QueryGetPriceRequest{
		BaseAssetType:  cstypes.PricingAssetType_COIN,
		BaseAsset:      "uverana",
		QuoteAssetType: cstypes.PricingAssetType_FIAT,
		QuoteAsset:     "USD",
		Amount:         "100",
	})
	require.Error(t, err)
	require.Contains(t, err.Error(), "expired")
}

func TestGetPrice_NotActive(t *testing.T) {
	f := initFixture(t)
	qs := keeper.NewQueryServerImpl(f.keeper)

	seedPriceExchangeRate(t, f, "500", 2, false, false)

	_, err := qs.GetPrice(f.ctx, &types.QueryGetPriceRequest{
		BaseAssetType:  cstypes.PricingAssetType_COIN,
		BaseAsset:      "uverana",
		QuoteAssetType: cstypes.PricingAssetType_FIAT,
		QuoteAsset:     "USD",
		Amount:         "100",
	})
	require.Error(t, err)
	require.Contains(t, err.Error(), "not active")
}

func TestGetPrice_NotFound(t *testing.T) {
	f := initFixture(t)
	qs := keeper.NewQueryServerImpl(f.keeper)

	_, err := qs.GetPrice(f.ctx, &types.QueryGetPriceRequest{
		BaseAssetType:  cstypes.PricingAssetType_COIN,
		BaseAsset:      "uverana",
		QuoteAssetType: cstypes.PricingAssetType_FIAT,
		QuoteAsset:     "USD",
		Amount:         "100",
	})
	require.Error(t, err)
	require.Contains(t, err.Error(), "not found")
}
