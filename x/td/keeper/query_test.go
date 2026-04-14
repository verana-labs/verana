package keeper_test

import (
	"testing"

	"cosmossdk.io/math"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/stretchr/testify/require"
	keepertest "github.com/verana-labs/verana/testutil/keeper"
	"github.com/verana-labs/verana/x/td/types"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func TestGetTrustDeposit(t *testing.T) {
	keeper, ctx := keepertest.TrustdepositKeeper(t)
	wctx := sdk.WrapSDKContext(ctx)

	// Create test account address
	testAddr := sdk.AccAddress([]byte("test_address")).String()

	// Test with non-existent trust deposit - should return NotFound error
	_, err := keeper.GetTrustDeposit(wctx, &types.QueryGetTrustDepositRequest{
		Corporation: testAddr,
	})
	require.Error(t, err)
	require.Contains(t, err.Error(), "trust deposit not found")
	require.Contains(t, status.Code(err).String(), codes.NotFound.String())

	// Create a trust deposit
	trustDeposit := types.TrustDeposit{
		Corporation: testAddr,
		Share:       math.LegacyNewDec(100),
		Deposit:     1000,
		Claimable:   50,
	}
	err = keeper.TrustDeposit.Set(ctx, testAddr, trustDeposit)
	require.NoError(t, err)

	// Test with existing trust deposit
	resp, err := keeper.GetTrustDeposit(wctx, &types.QueryGetTrustDepositRequest{
		Corporation: testAddr,
	})
	require.NoError(t, err)
	require.Equal(t, trustDeposit, resp.TrustDeposit)

	// Test with invalid corporation address
	_, err = keeper.GetTrustDeposit(wctx, &types.QueryGetTrustDepositRequest{
		Corporation: "invalid_address",
	})
	require.Error(t, err)
	require.Contains(t, err.Error(), "invalid corporation address")
	require.Contains(t, status.Code(err).String(), codes.InvalidArgument.String())
}
