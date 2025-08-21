package keeper_test

import (
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/stretchr/testify/require"
	keepertest "github.com/verana-labs/verana-blockchain/testutil/keeper"
	"github.com/verana-labs/verana-blockchain/x/td/types"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"testing"
)

func TestGetTrustDeposit(t *testing.T) {
	keeper, ctx := keepertest.TrustdepositKeeper(t)
	wctx := sdk.WrapSDKContext(ctx)

	// Create test account address
	testAddr := sdk.AccAddress([]byte("test_address")).String()

	// Test with non-existent trust deposit - should return NotFound error
	_, err := keeper.GetTrustDeposit(wctx, &types.QueryGetTrustDepositRequest{
		Account: testAddr,
	})
	require.Error(t, err)
	require.Contains(t, err.Error(), "trust deposit not found")
	require.Contains(t, status.Code(err).String(), codes.NotFound.String())

	// Create a trust deposit
	trustDeposit := types.TrustDeposit{
		Account:   testAddr,
		Share:     100,
		Amount:    1000,
		Claimable: 50,
	}
	err = keeper.TrustDeposit.Set(ctx, testAddr, trustDeposit)
	require.NoError(t, err)

	// Test with existing trust deposit
	resp, err := keeper.GetTrustDeposit(wctx, &types.QueryGetTrustDepositRequest{
		Account: testAddr,
	})
	require.NoError(t, err)
	require.Equal(t, trustDeposit, resp.TrustDeposit)

	// Test with invalid account address
	_, err = keeper.GetTrustDeposit(wctx, &types.QueryGetTrustDepositRequest{
		Account: "invalid_address",
	})
	require.Error(t, err)
	require.Contains(t, err.Error(), "invalid account address")
	require.Contains(t, status.Code(err).String(), codes.InvalidArgument.String())
}
