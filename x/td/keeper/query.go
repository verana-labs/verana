package keeper

import (
	"context"
	"fmt"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/verana-labs/verana/x/td/types"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

var _ types.QueryServer = Keeper{}

func (k Keeper) GetTrustDeposit(goCtx context.Context, req *types.QueryGetTrustDepositRequest) (*types.QueryGetTrustDepositResponse, error) {
	if req == nil {
		return nil, status.Error(codes.InvalidArgument, "empty request")
	}

	ctx := sdk.UnwrapSDKContext(goCtx)

	// [MOD-TD-QRY-1-2] Validate account address
	if _, err := sdk.AccAddressFromBech32(req.Account); err != nil {
		return nil, status.Error(codes.InvalidArgument, fmt.Sprintf("invalid account address: %s", err))
	}

	// [MOD-TD-QRY-1-3] Get trust deposit for account
	trustDeposit, err := k.TrustDeposit.Get(ctx, req.Account)
	if err != nil {
		// Per spec: if not found, return not found error instead of zero values
		return nil, status.Error(codes.NotFound, fmt.Sprintf("trust deposit not found for account %s", req.Account))
	}

	return &types.QueryGetTrustDepositResponse{
		TrustDeposit: trustDeposit,
	}, nil
}
