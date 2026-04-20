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

func TestSetExchangeRateState_HappyPath_Enable(t *testing.T) {
	f := initFixture(t)
	ms := keeper.NewMsgServerImpl(f.keeper)

	authorityStr, err := f.addressCodec.BytesToString(f.keeper.GetAuthority())
	require.NoError(t, err)

	// Create exchange rate (state=true on creation per spec [MOD-XR-MSG-1])
	id := createTestExchangeRate(t, f, ms, authorityStr)

	// Verify initial state is true
	xr, err := f.keeper.ExchangeRates.Get(f.ctx, id)
	require.NoError(t, err)
	require.True(t, xr.State)

	// [MOD-XR-MSG-3-3] toggle flips current state.
	_, err = ms.SetExchangeRateState(f.ctx, &types.MsgSetExchangeRateState{
		Authority: authorityStr,
		Id:        id,
	})
	require.NoError(t, err)

	xr, err = f.keeper.ExchangeRates.Get(f.ctx, id)
	require.NoError(t, err)
	require.False(t, xr.State)

	// Toggle again flips back.
	_, err = ms.SetExchangeRateState(f.ctx, &types.MsgSetExchangeRateState{
		Authority: authorityStr,
		Id:        id,
	})
	require.NoError(t, err)

	// Verify state is now true
	xr, err = f.keeper.ExchangeRates.Get(f.ctx, id)
	require.NoError(t, err)
	require.True(t, xr.State)
}

func TestSetExchangeRateState_HappyPath_Disable(t *testing.T) {
	f := initFixture(t)
	ms := keeper.NewMsgServerImpl(f.keeper)

	authorityStr, err := f.addressCodec.BytesToString(f.keeper.GetAuthority())
	require.NoError(t, err)

	// [MOD-XR-MSG-1-3] created with state=true.
	id := createTestExchangeRate(t, f, ms, authorityStr)

	// [MOD-XR-MSG-3-3] spec v4 draft 13: call toggles the stored state (true ↔ false).
	_, err = ms.SetExchangeRateState(f.ctx, &types.MsgSetExchangeRateState{
		Authority: authorityStr,
		Id:        id,
	})
	require.NoError(t, err)

	// After one toggle, state flipped to false.
	xr, err := f.keeper.ExchangeRates.Get(f.ctx, id)
	require.NoError(t, err)
	require.False(t, xr.State)

	// Second toggle flips it back to true.
	_, err = ms.SetExchangeRateState(f.ctx, &types.MsgSetExchangeRateState{
		Authority: authorityStr,
		Id:        id,
	})
	require.NoError(t, err)

	xr, err = f.keeper.ExchangeRates.Get(f.ctx, id)
	require.NoError(t, err)
	require.True(t, xr.State)
}

func TestSetExchangeRateState_InvalidAuthority(t *testing.T) {
	f := initFixture(t)
	ms := keeper.NewMsgServerImpl(f.keeper)

	authorityStr, err := f.addressCodec.BytesToString(f.keeper.GetAuthority())
	require.NoError(t, err)

	id := createTestExchangeRate(t, f, ms, authorityStr)

	nonGovAddr := sdk.AccAddress([]byte("not_gov_authority___")).String()
	_, err = ms.SetExchangeRateState(f.ctx, &types.MsgSetExchangeRateState{
		Authority: nonGovAddr,
		Id:        id,
		State:     true,
	})
	require.Error(t, err)
	require.Contains(t, err.Error(), "expected gov account as only signer")
}

func TestSetExchangeRateState_NotFound(t *testing.T) {
	f := initFixture(t)
	ms := keeper.NewMsgServerImpl(f.keeper)

	authorityStr, err := f.addressCodec.BytesToString(f.keeper.GetAuthority())
	require.NoError(t, err)

	_, err = ms.SetExchangeRateState(f.ctx, &types.MsgSetExchangeRateState{
		Authority: authorityStr,
		Id:        999,
		State:     true,
	})
	require.Error(t, err)
	require.Contains(t, err.Error(), "exchange rate not found")
}
