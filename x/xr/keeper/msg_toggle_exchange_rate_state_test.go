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

// helper to create an exchange rate and return its id
func createTestExchangeRate(t *testing.T, f *fixture, ms types.MsgServer, authorityStr string) uint64 {
	t.Helper()
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
	return resp.Id
}

func TestToggleExchangeRateState_HappyPath_Enable(t *testing.T) {
	f := initFixture(t)
	ms := keeper.NewMsgServerImpl(f.keeper)

	authorityStr, err := f.addressCodec.BytesToString(f.keeper.GetAuthority())
	require.NoError(t, err)

	// Create exchange rate (state defaults to false)
	id := createTestExchangeRate(t, f, ms, authorityStr)

	// Verify initial state is false
	xr, err := f.keeper.ExchangeRates.Get(f.ctx, id)
	require.NoError(t, err)
	require.False(t, xr.State)

	// Toggle to enabled
	_, err = ms.ToggleExchangeRateState(f.ctx, &types.MsgToggleExchangeRateState{
		Authority: authorityStr,
		Id:        id,
		State:     true,
	})
	require.NoError(t, err)

	// Verify state is now true
	xr, err = f.keeper.ExchangeRates.Get(f.ctx, id)
	require.NoError(t, err)
	require.True(t, xr.State)
}

func TestToggleExchangeRateState_HappyPath_Disable(t *testing.T) {
	f := initFixture(t)
	ms := keeper.NewMsgServerImpl(f.keeper)

	authorityStr, err := f.addressCodec.BytesToString(f.keeper.GetAuthority())
	require.NoError(t, err)

	// Create exchange rate and enable it first
	id := createTestExchangeRate(t, f, ms, authorityStr)

	_, err = ms.ToggleExchangeRateState(f.ctx, &types.MsgToggleExchangeRateState{
		Authority: authorityStr,
		Id:        id,
		State:     true,
	})
	require.NoError(t, err)

	// Verify enabled
	xr, err := f.keeper.ExchangeRates.Get(f.ctx, id)
	require.NoError(t, err)
	require.True(t, xr.State)

	// Now disable
	_, err = ms.ToggleExchangeRateState(f.ctx, &types.MsgToggleExchangeRateState{
		Authority: authorityStr,
		Id:        id,
		State:     false,
	})
	require.NoError(t, err)

	// Verify state is now false
	xr, err = f.keeper.ExchangeRates.Get(f.ctx, id)
	require.NoError(t, err)
	require.False(t, xr.State)
}

func TestToggleExchangeRateState_InvalidAuthority(t *testing.T) {
	f := initFixture(t)
	ms := keeper.NewMsgServerImpl(f.keeper)

	authorityStr, err := f.addressCodec.BytesToString(f.keeper.GetAuthority())
	require.NoError(t, err)

	id := createTestExchangeRate(t, f, ms, authorityStr)

	nonGovAddr := sdk.AccAddress([]byte("not_gov_authority___")).String()
	_, err = ms.ToggleExchangeRateState(f.ctx, &types.MsgToggleExchangeRateState{
		Authority: nonGovAddr,
		Id:        id,
		State:     true,
	})
	require.Error(t, err)
	require.Contains(t, err.Error(), "expected gov account as only signer")
}

func TestToggleExchangeRateState_NotFound(t *testing.T) {
	f := initFixture(t)
	ms := keeper.NewMsgServerImpl(f.keeper)

	authorityStr, err := f.addressCodec.BytesToString(f.keeper.GetAuthority())
	require.NoError(t, err)

	_, err = ms.ToggleExchangeRateState(f.ctx, &types.MsgToggleExchangeRateState{
		Authority: authorityStr,
		Id:        999,
		State:     true,
	})
	require.Error(t, err)
	require.Contains(t, err.Error(), "exchange rate not found")
}
