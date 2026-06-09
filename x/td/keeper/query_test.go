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

	const corpID = uint64(1)

	// Test with non-existent trust deposit - should return NotFound error
	_, err := keeper.GetTrustDeposit(wctx, &types.QueryGetTrustDepositRequest{
		CorporationId: corpID,
	})
	require.Error(t, err)
	require.Contains(t, err.Error(), "trust deposit not found")
	require.Contains(t, status.Code(err).String(), codes.NotFound.String())

	// Create a trust deposit (keyed by corporation_id)
	trustDeposit := types.TrustDeposit{
		CorporationId: corpID,
		Share:         math.LegacyNewDec(100),
		Deposit:       1000,
		Refunded:      50,
	}
	err = keeper.TrustDeposit.Set(ctx, corpID, trustDeposit)
	require.NoError(t, err)

	// Test with existing trust deposit
	resp, err := keeper.GetTrustDeposit(wctx, &types.QueryGetTrustDepositRequest{
		CorporationId: corpID,
	})
	require.NoError(t, err)
	require.Equal(t, trustDeposit, resp.TrustDeposit)

	// Test with invalid corporation_id (zero)
	_, err = keeper.GetTrustDeposit(wctx, &types.QueryGetTrustDepositRequest{
		CorporationId: 0,
	})
	require.Error(t, err)
	require.Contains(t, err.Error(), "corporation_id must be greater than 0")
	require.Contains(t, status.Code(err).String(), codes.InvalidArgument.String())
}
