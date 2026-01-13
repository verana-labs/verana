package keeper

import (
	"context"
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

// =============================================================================
// ANCHOR-BASED POC QUERIES
// =============================================================================

// GetAnchor returns the anchor info for a given anchor_id
func (k Keeper) GetAnchor(goCtx context.Context, req *types.QueryGetAnchorRequest) (*types.QueryGetAnchorResponse, error) {
	if req == nil {
		return nil, status.Error(codes.InvalidArgument, "empty request")
	}

	ctx := sdk.UnwrapSDKContext(goCtx)

	if _, err := sdk.AccAddressFromBech32(req.AnchorId); err != nil {
		return nil, status.Error(codes.InvalidArgument, fmt.Sprintf("invalid anchor_id: %s", err))
	}

	anchor, err := k.Anchors.Get(ctx, req.AnchorId)
	if err != nil {
		return nil, status.Error(codes.NotFound, fmt.Sprintf("anchor not found: %s", req.AnchorId))
	}

	return &types.QueryGetAnchorResponse{
		Anchor: anchor,
	}, nil
}

// GetVerifiableService returns the VS info for a given operator account
func (k Keeper) GetVerifiableService(goCtx context.Context, req *types.QueryGetVerifiableServiceRequest) (*types.QueryGetVerifiableServiceResponse, error) {
	if req == nil {
		return nil, status.Error(codes.InvalidArgument, "empty request")
	}

	ctx := sdk.UnwrapSDKContext(goCtx)

	if _, err := sdk.AccAddressFromBech32(req.OperatorAccount); err != nil {
		return nil, status.Error(codes.InvalidArgument, fmt.Sprintf("invalid operator_account: %s", err))
	}

	vs, err := k.VerifiableServices.Get(ctx, req.OperatorAccount)
	if err != nil {
		return nil, status.Error(codes.NotFound, fmt.Sprintf("verifiable service not found for operator: %s", req.OperatorAccount))
	}

	return &types.QueryGetVerifiableServiceResponse{
		VerifiableService: vs,
	}, nil
}

// GetOperatorAllowance returns the allowance for a given anchor/operator pair
func (k Keeper) GetOperatorAllowance(goCtx context.Context, req *types.QueryGetOperatorAllowanceRequest) (*types.QueryGetOperatorAllowanceResponse, error) {
	if req == nil {
		return nil, status.Error(codes.InvalidArgument, "empty request")
	}

	ctx := sdk.UnwrapSDKContext(goCtx)

	if _, err := sdk.AccAddressFromBech32(req.AnchorId); err != nil {
		return nil, status.Error(codes.InvalidArgument, fmt.Sprintf("invalid anchor_id: %s", err))
	}
	if _, err := sdk.AccAddressFromBech32(req.OperatorAccount); err != nil {
		return nil, status.Error(codes.InvalidArgument, fmt.Sprintf("invalid operator_account: %s", err))
	}

	key := collections.Join(req.AnchorId, req.OperatorAccount)
	allowance, err := k.OperatorAllowances.Get(ctx, key)
	if err != nil {
		return nil, status.Error(codes.NotFound, fmt.Sprintf("allowance not found for anchor %s, operator %s", req.AnchorId, req.OperatorAccount))
	}

	return &types.QueryGetOperatorAllowanceResponse{
		OperatorAllowance: allowance,
	}, nil
}
