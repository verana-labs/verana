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

func TestCreateExchangeRate_HappyPath(t *testing.T) {
	f := initFixture(t)
	ms := keeper.NewMsgServerImpl(f.keeper)

	authorityStr, err := f.addressCodec.BytesToString(f.keeper.GetAuthority())
	require.NoError(t, err)

	resp, err := ms.CreateExchangeRate(f.ctx, &types.MsgCreateExchangeRate{
		Authority:        authorityStr,
		BaseAssetType:    cstypes.PricingAssetType_COIN,
		BaseAsset:        "uverana",
		QuoteAssetType:   cstypes.PricingAssetType_FIAT,
		QuoteAsset:       "USD",
		Rate:             "100",
		RateScale:        2,
		ValidityDuration: 10 * time.Minute,
	})
	require.NoError(t, err)
	require.Equal(t, uint64(1), resp.Id)

	// Verify stored
	xr, err := f.keeper.ExchangeRates.Get(f.ctx, resp.Id)
	require.NoError(t, err)
	require.Equal(t, uint64(1), xr.Id)
	require.Equal(t, cstypes.PricingAssetType_COIN, xr.BaseAssetType)
	require.Equal(t, "uverana", xr.BaseAsset)
	require.Equal(t, cstypes.PricingAssetType_FIAT, xr.QuoteAssetType)
	require.Equal(t, "USD", xr.QuoteAsset)
	require.Equal(t, "100", xr.Rate)
	require.Equal(t, uint32(2), xr.RateScale)
	require.Equal(t, 10*time.Minute, xr.ValidityDuration)
	require.False(t, xr.State)
}

func TestCreateExchangeRate_InvalidAuthority(t *testing.T) {
	f := initFixture(t)
	ms := keeper.NewMsgServerImpl(f.keeper)

	nonGovAddr := sdk.AccAddress([]byte("not_gov_authority___")).String()
	_, err := ms.CreateExchangeRate(f.ctx, &types.MsgCreateExchangeRate{
		Authority:        nonGovAddr,
		BaseAssetType:    cstypes.PricingAssetType_COIN,
		BaseAsset:        "uverana",
		QuoteAssetType:   cstypes.PricingAssetType_FIAT,
		QuoteAsset:       "USD",
		Rate:             "100",
		RateScale:        2,
		ValidityDuration: 10 * time.Minute,
	})
	require.Error(t, err)
	require.Contains(t, err.Error(), "expected gov account as only signer")
}

func TestCreateExchangeRate_InvalidAssetType(t *testing.T) {
	f := initFixture(t)
	ms := keeper.NewMsgServerImpl(f.keeper)

	authorityStr, err := f.addressCodec.BytesToString(f.keeper.GetAuthority())
	require.NoError(t, err)

	_, err = ms.CreateExchangeRate(f.ctx, &types.MsgCreateExchangeRate{
		Authority:        authorityStr,
		BaseAssetType:    cstypes.PricingAssetType_PRICING_ASSET_TYPE_UNSPECIFIED,
		BaseAsset:        "uverana",
		QuoteAssetType:   cstypes.PricingAssetType_FIAT,
		QuoteAsset:       "USD",
		Rate:             "100",
		RateScale:        2,
		ValidityDuration: 10 * time.Minute,
	})
	require.Error(t, err)
	require.Contains(t, err.Error(), "invalid base_asset_type")
}

func TestCreateExchangeRate_TrustUnitMustBeTU(t *testing.T) {
	f := initFixture(t)
	ms := keeper.NewMsgServerImpl(f.keeper)

	authorityStr, err := f.addressCodec.BytesToString(f.keeper.GetAuthority())
	require.NoError(t, err)

	_, err = ms.CreateExchangeRate(f.ctx, &types.MsgCreateExchangeRate{
		Authority:        authorityStr,
		BaseAssetType:    cstypes.PricingAssetType_TU,
		BaseAsset:        "WRONG",
		QuoteAssetType:   cstypes.PricingAssetType_FIAT,
		QuoteAsset:       "USD",
		Rate:             "100",
		RateScale:        2,
		ValidityDuration: 10 * time.Minute,
	})
	require.Error(t, err)
	require.Contains(t, err.Error(), "must equal \"TU\"")
}

func TestCreateExchangeRate_FiatMustBeISO4217(t *testing.T) {
	f := initFixture(t)
	ms := keeper.NewMsgServerImpl(f.keeper)

	authorityStr, err := f.addressCodec.BytesToString(f.keeper.GetAuthority())
	require.NoError(t, err)

	// lowercase fiat code
	_, err = ms.CreateExchangeRate(f.ctx, &types.MsgCreateExchangeRate{
		Authority:        authorityStr,
		BaseAssetType:    cstypes.PricingAssetType_COIN,
		BaseAsset:        "uverana",
		QuoteAssetType:   cstypes.PricingAssetType_FIAT,
		QuoteAsset:       "usd",
		Rate:             "100",
		RateScale:        2,
		ValidityDuration: 10 * time.Minute,
	})
	require.Error(t, err)
	require.Contains(t, err.Error(), "ISO-4217")

	// too long fiat code
	_, err = ms.CreateExchangeRate(f.ctx, &types.MsgCreateExchangeRate{
		Authority:        authorityStr,
		BaseAssetType:    cstypes.PricingAssetType_COIN,
		BaseAsset:        "uverana",
		QuoteAssetType:   cstypes.PricingAssetType_FIAT,
		QuoteAsset:       "USDT",
		Rate:             "100",
		RateScale:        2,
		ValidityDuration: 10 * time.Minute,
	})
	require.Error(t, err)
	require.Contains(t, err.Error(), "ISO-4217")
}

func TestCreateExchangeRate_IdenticalPair(t *testing.T) {
	f := initFixture(t)
	ms := keeper.NewMsgServerImpl(f.keeper)

	authorityStr, err := f.addressCodec.BytesToString(f.keeper.GetAuthority())
	require.NoError(t, err)

	_, err = ms.CreateExchangeRate(f.ctx, &types.MsgCreateExchangeRate{
		Authority:        authorityStr,
		BaseAssetType:    cstypes.PricingAssetType_FIAT,
		BaseAsset:        "USD",
		QuoteAssetType:   cstypes.PricingAssetType_FIAT,
		QuoteAsset:       "USD",
		Rate:             "100",
		RateScale:        2,
		ValidityDuration: 10 * time.Minute,
	})
	require.Error(t, err)
	require.Contains(t, err.Error(), "identical")
}

func TestCreateExchangeRate_DuplicatePair(t *testing.T) {
	f := initFixture(t)
	ms := keeper.NewMsgServerImpl(f.keeper)

	authorityStr, err := f.addressCodec.BytesToString(f.keeper.GetAuthority())
	require.NoError(t, err)

	msg := &types.MsgCreateExchangeRate{
		Authority:        authorityStr,
		BaseAssetType:    cstypes.PricingAssetType_COIN,
		BaseAsset:        "uverana",
		QuoteAssetType:   cstypes.PricingAssetType_FIAT,
		QuoteAsset:       "USD",
		Rate:             "100",
		RateScale:        2,
		ValidityDuration: 10 * time.Minute,
	}

	// First creation should succeed
	_, err = ms.CreateExchangeRate(f.ctx, msg)
	require.NoError(t, err)

	// Duplicate should fail
	_, err = ms.CreateExchangeRate(f.ctx, msg)
	require.Error(t, err)
	require.Contains(t, err.Error(), "already exists")
}

func TestCreateExchangeRate_InvalidRate(t *testing.T) {
	f := initFixture(t)
	ms := keeper.NewMsgServerImpl(f.keeper)

	authorityStr, err := f.addressCodec.BytesToString(f.keeper.GetAuthority())
	require.NoError(t, err)

	tests := []struct {
		name string
		rate string
		errMsg string
	}{
		{"zero rate", "0", "strictly greater than 0"},
		{"negative rate", "-1", "strictly greater than 0"},
		{"non-numeric rate", "abc", "unsigned integer"},
		{"empty rate", "", "unsigned integer"},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			_, err := ms.CreateExchangeRate(f.ctx, &types.MsgCreateExchangeRate{
				Authority:        authorityStr,
				BaseAssetType:    cstypes.PricingAssetType_COIN,
				BaseAsset:        "uverana",
				QuoteAssetType:   cstypes.PricingAssetType_FIAT,
				QuoteAsset:       "USD",
				Rate:             tc.rate,
				RateScale:        2,
				ValidityDuration: 10 * time.Minute,
			})
			require.Error(t, err)
			require.Contains(t, err.Error(), tc.errMsg)
		})
	}
}

func TestCreateExchangeRate_InvalidRateScale(t *testing.T) {
	f := initFixture(t)
	ms := keeper.NewMsgServerImpl(f.keeper)

	authorityStr, err := f.addressCodec.BytesToString(f.keeper.GetAuthority())
	require.NoError(t, err)

	_, err = ms.CreateExchangeRate(f.ctx, &types.MsgCreateExchangeRate{
		Authority:        authorityStr,
		BaseAssetType:    cstypes.PricingAssetType_COIN,
		BaseAsset:        "uverana",
		QuoteAssetType:   cstypes.PricingAssetType_FIAT,
		QuoteAsset:       "USD",
		Rate:             "100",
		RateScale:        19,
		ValidityDuration: 10 * time.Minute,
	})
	require.Error(t, err)
	require.Contains(t, err.Error(), "rate_scale must be <= 18")
}

func TestCreateExchangeRate_InvalidDuration(t *testing.T) {
	f := initFixture(t)
	ms := keeper.NewMsgServerImpl(f.keeper)

	authorityStr, err := f.addressCodec.BytesToString(f.keeper.GetAuthority())
	require.NoError(t, err)

	_, err = ms.CreateExchangeRate(f.ctx, &types.MsgCreateExchangeRate{
		Authority:        authorityStr,
		BaseAssetType:    cstypes.PricingAssetType_COIN,
		BaseAsset:        "uverana",
		QuoteAssetType:   cstypes.PricingAssetType_FIAT,
		QuoteAsset:       "USD",
		Rate:             "100",
		RateScale:        2,
		ValidityDuration: 30 * time.Second,
	})
	require.Error(t, err)
	require.Contains(t, err.Error(), "validity_duration must be >= 1 minute")
}
