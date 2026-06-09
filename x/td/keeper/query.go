package keeper

import (
	"context"
	"errors"
	"fmt"

	"cosmossdk.io/collections"
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

	// [MOD-TD-QRY-1-2] Validate corporation_id
	if req.CorporationId == 0 {
		return nil, status.Error(codes.InvalidArgument, "corporation_id must be greater than 0")
	}

	// [MOD-TD-QRY-1-3] Get trust deposit for corporation_id
	trustDeposit, err := k.TrustDeposit.Get(ctx, req.CorporationId)
	if err != nil {
		if errors.Is(err, collections.ErrNotFound) {
			return nil, status.Error(codes.NotFound, fmt.Sprintf("trust deposit not found for corporation_id %d", req.CorporationId))
		}
		return nil, status.Error(codes.Internal, fmt.Sprintf("failed to get trust deposit: %s", err))
	}

	return &types.QueryGetTrustDepositResponse{
		TrustDeposit: trustDeposit,
	}, nil
}
