package keeper_test

import (
	"fmt"
	"testing"
	"time"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/stretchr/testify/require"

	cstypes "github.com/verana-labs/verana/x/cs/types"
	"github.com/verana-labs/verana/x/xr/keeper"
	"github.com/verana-labs/verana/x/xr/types"
)

// createActiveExchangeRate is a test helper that creates an exchange rate and sets state=true.
func createActiveExchangeRate(t *testing.T, f *fixture, ms types.MsgServer) uint64 {
	t.Helper()

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

	// Activate the exchange rate (set state=true)
	xr, err := f.keeper.ExchangeRates.Get(f.ctx, resp.Id)
	require.NoError(t, err)
	xr.State = true
	require.NoError(t, f.keeper.ExchangeRates.Set(f.ctx, resp.Id, xr))

	return resp.Id
}

func TestUpdateExchangeRate_HappyPath(t *testing.T) {
	f := initFixture(t)
	ms := keeper.NewMsgServerImpl(f.keeper)

	id := createActiveExchangeRate(t, f, ms)

	authorityStr, err := f.addressCodec.BytesToString(f.keeper.GetAuthority())
	require.NoError(t, err)
	operatorAddr := sdk.AccAddress([]byte("operator_address____")).String()

	sdkCtx := sdk.UnwrapSDKContext(f.ctx)
	blockTime := sdkCtx.BlockTime()

	_, err = ms.UpdateExchangeRate(f.ctx, &types.MsgUpdateExchangeRate{
		Authority: authorityStr,
		Operator:  operatorAddr,
		Id:        id,
		Rate:      "200",
	})
	require.NoError(t, err)

	// Verify updated fields
	xr, err := f.keeper.ExchangeRates.Get(f.ctx, id)
	require.NoError(t, err)
	require.Equal(t, "200", xr.Rate)
	require.Equal(t, blockTime.Add(xr.ValidityDuration), xr.Expires)
	require.Equal(t, blockTime, xr.Updated)
	require.True(t, xr.State) // state should remain true
}

func TestUpdateExchangeRate_InvalidAuthority(t *testing.T) {
	f := initFixture(t)
	ms := keeper.NewMsgServerImpl(f.keeper)

	operatorAddr := sdk.AccAddress([]byte("operator_address____")).String()

	_, err := ms.UpdateExchangeRate(f.ctx, &types.MsgUpdateExchangeRate{
		Authority: "invalid",
		Operator:  operatorAddr,
		Id:        1,
		Rate:      "200",
	})
	require.Error(t, err)
	require.Contains(t, err.Error(), "invalid authority address")
}

func TestUpdateExchangeRate_NotFound(t *testing.T) {
	f := initFixture(t)
	ms := keeper.NewMsgServerImpl(f.keeper)

	authorityStr, err := f.addressCodec.BytesToString(f.keeper.GetAuthority())
	require.NoError(t, err)
	operatorAddr := sdk.AccAddress([]byte("operator_address____")).String()

	_, err = ms.UpdateExchangeRate(f.ctx, &types.MsgUpdateExchangeRate{
		Authority: authorityStr,
		Operator:  operatorAddr,
		Id:        999,
		Rate:      "200",
	})
	require.Error(t, err)
	require.Contains(t, err.Error(), "not found")
}

func TestUpdateExchangeRate_NotActive(t *testing.T) {
	f := initFixture(t)
	ms := keeper.NewMsgServerImpl(f.keeper)

	authorityStr, err := f.addressCodec.BytesToString(f.keeper.GetAuthority())
	require.NoError(t, err)

	// Create exchange rate (state=true on creation per spec [MOD-XR-MSG-1])
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

	// Disable the exchange rate so it is not active
	_, err = ms.SetExchangeRateState(f.ctx, &types.MsgSetExchangeRateState{
		Authority: authorityStr,
		Id:        resp.Id,
		State:     false,
	})
	require.NoError(t, err)

	operatorAddr := sdk.AccAddress([]byte("operator_address____")).String()

	_, err = ms.UpdateExchangeRate(f.ctx, &types.MsgUpdateExchangeRate{
		Authority: authorityStr,
		Operator:  operatorAddr,
		Id:        resp.Id,
		Rate:      "200",
	})
	require.Error(t, err)
	require.Contains(t, err.Error(), "not active")
}

func TestUpdateExchangeRate_InvalidRate(t *testing.T) {
	f := initFixture(t)
	ms := keeper.NewMsgServerImpl(f.keeper)

	authorityStr, err := f.addressCodec.BytesToString(f.keeper.GetAuthority())
	require.NoError(t, err)
	operatorAddr := sdk.AccAddress([]byte("operator_address____")).String()

	tests := []struct {
		name   string
		rate   string
		errMsg string
	}{
		{"zero rate", "0", "strictly greater than 0"},
		{"negative rate", "-1", "strictly greater than 0"},
		{"non-numeric rate", "abc", "unsigned integer"},
		{"empty rate", "", "unsigned integer"},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			_, err := ms.UpdateExchangeRate(f.ctx, &types.MsgUpdateExchangeRate{
				Authority: authorityStr,
				Operator:  operatorAddr,
				Id:        1,
				Rate:      tc.rate,
			})
			require.Error(t, err)
			require.Contains(t, err.Error(), tc.errMsg)
		})
	}
}

func TestUpdateExchangeRate_RateScale(t *testing.T) {
	f := initFixture(t)
	ms := keeper.NewMsgServerImpl(f.keeper)

	id := createActiveExchangeRate(t, f, ms)

	authorityStr, err := f.addressCodec.BytesToString(f.keeper.GetAuthority())
	require.NoError(t, err)
	operatorAddr := sdk.AccAddress([]byte("operator_address____")).String()

	// Updating with rate_scale=2 should change xr.RateScale to 2
	_, err = ms.UpdateExchangeRate(f.ctx, &types.MsgUpdateExchangeRate{
		Authority: authorityStr,
		Operator:  operatorAddr,
		Id:        id,
		Rate:      "200",
		RateScale: 2,
	})
	require.NoError(t, err)
	xr, err := f.keeper.ExchangeRates.Get(f.ctx, id)
	require.NoError(t, err)
	require.Equal(t, uint32(2), xr.RateScale)

	// Updating with rate_scale=0 should keep existing xr.RateScale
	_, err = ms.UpdateExchangeRate(f.ctx, &types.MsgUpdateExchangeRate{
		Authority: authorityStr,
		Operator:  operatorAddr,
		Id:        id,
		Rate:      "300",
		RateScale: 0,
	})
	require.NoError(t, err)
	xr, err = f.keeper.ExchangeRates.Get(f.ctx, id)
	require.NoError(t, err)
	require.Equal(t, uint32(2), xr.RateScale) // unchanged
}

func TestUpdateExchangeRate_ValidityDuration(t *testing.T) {
	f := initFixture(t)
	ms := keeper.NewMsgServerImpl(f.keeper)

	id := createActiveExchangeRate(t, f, ms)

	authorityStr, err := f.addressCodec.BytesToString(f.keeper.GetAuthority())
	require.NoError(t, err)
	operatorAddr := sdk.AccAddress([]byte("operator_address____")).String()

	sdkCtx := sdk.UnwrapSDKContext(f.ctx)
	blockTime := sdkCtx.BlockTime()

	newDuration := 30 * time.Minute
	_, err = ms.UpdateExchangeRate(f.ctx, &types.MsgUpdateExchangeRate{
		Authority:        authorityStr,
		Operator:         operatorAddr,
		Id:               id,
		Rate:             "200",
		ValidityDuration: &newDuration,
	})
	require.NoError(t, err)

	xr, err := f.keeper.ExchangeRates.Get(f.ctx, id)
	require.NoError(t, err)
	require.Equal(t, newDuration, xr.ValidityDuration)
	require.Equal(t, blockTime.Add(newDuration), xr.Expires)
}

func TestUpdateExchangeRate_AuthzFailure(t *testing.T) {
	f := initFixture(t)
	ms := keeper.NewMsgServerImpl(f.keeper)

	id := createActiveExchangeRate(t, f, ms)

	authorityStr, err := f.addressCodec.BytesToString(f.keeper.GetAuthority())
	require.NoError(t, err)
	operatorAddr := sdk.AccAddress([]byte("operator_address____")).String()

	// Set the mock to return an error
	f.delegationKeeper.ErrToReturn = fmt.Errorf("operator not authorized")

	_, err = ms.UpdateExchangeRate(f.ctx, &types.MsgUpdateExchangeRate{
		Authority: authorityStr,
		Operator:  operatorAddr,
		Id:        id,
		Rate:      "200",
	})
	require.Error(t, err)
	require.Contains(t, err.Error(), "authorization check failed")

	// Reset mock and verify the exchange rate was NOT modified
	f.delegationKeeper.ErrToReturn = nil
	xr, err := f.keeper.ExchangeRates.Get(f.ctx, id)
	require.NoError(t, err)
	require.Equal(t, "100", xr.Rate) // rate should be unchanged
}
