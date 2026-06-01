package keeper

import (
	"context"
	"errors"

	"cosmossdk.io/collections"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/verana-labs/verana/x/de/types"
)

// GetOperatorAuthorization implements [MOD-DE-QRY-3]: fetch an
// OperatorAuthorization by id.
func (q queryServer) GetOperatorAuthorization(ctx context.Context, req *types.QueryGetOperatorAuthorizationRequest) (*types.QueryGetOperatorAuthorizationResponse, error) {
	if req == nil {
		return nil, status.Error(codes.InvalidArgument, "invalid request")
	}
	if req.Id == 0 {
		return nil, status.Error(codes.InvalidArgument, "id must be greater than 0")
	}

	oa, err := q.k.OperatorAuthorizations.Get(ctx, req.Id)
	if err != nil {
		if errors.Is(err, collections.ErrNotFound) {
			return nil, status.Errorf(codes.NotFound, "operator authorization %d not found", req.Id)
		}
		return nil, status.Error(codes.Internal, err.Error())
	}

	return &types.QueryGetOperatorAuthorizationResponse{OperatorAuthorization: oa}, nil
}

// GetVSOperatorAuthorization implements [MOD-DE-QRY-4]: fetch a
// VSOperatorAuthorization (including its records) by id.
func (q queryServer) GetVSOperatorAuthorization(ctx context.Context, req *types.QueryGetVSOperatorAuthorizationRequest) (*types.QueryGetVSOperatorAuthorizationResponse, error) {
	if req == nil {
		return nil, status.Error(codes.InvalidArgument, "invalid request")
	}
	if req.Id == 0 {
		return nil, status.Error(codes.InvalidArgument, "id must be greater than 0")
	}

	vsoa, err := q.k.VSOperatorAuthorizations.Get(ctx, req.Id)
	if err != nil {
		if errors.Is(err, collections.ErrNotFound) {
			return nil, status.Errorf(codes.NotFound, "vs operator authorization %d not found", req.Id)
		}
		return nil, status.Error(codes.Internal, err.Error())
	}

	return &types.QueryGetVSOperatorAuthorizationResponse{VsOperatorAuthorization: vsoa}, nil
}
