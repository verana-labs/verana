package keeper

import (
	"context"

	"cosmossdk.io/collections"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/verana-labs/verana/x/de/types"
)

func (q queryServer) ListOperatorAuthorizations(ctx context.Context, req *types.QueryListOperatorAuthorizationsRequest) (*types.QueryListOperatorAuthorizationsResponse, error) {
	if req == nil {
		return nil, status.Error(codes.InvalidArgument, "invalid request")
	}

	// [MOD-DE-QRY-1-2] Validate response_max_size
	if req.ResponseMaxSize == 0 {
		req.ResponseMaxSize = 64
	}
	if req.ResponseMaxSize < 1 || req.ResponseMaxSize > 1024 {
		return nil, status.Error(codes.InvalidArgument, "response_max_size must be between 1 and 1,024")
	}

	// [MOD-DE-QRY-1-3] Walk through all operator authorizations and apply filters
	var results []types.OperatorAuthorization

	err := q.k.OperatorAuthorizations.Walk(ctx, nil, func(key collections.Pair[string, string], oa types.OperatorAuthorization) (bool, error) {
		// Filter by authority if specified
		if req.Authority != "" && oa.Authority != req.Authority {
			return false, nil
		}
		// Filter by operator if specified
		if req.Operator != "" && oa.Operator != req.Operator {
			return false, nil
		}

		results = append(results, oa)
		return len(results) >= int(req.ResponseMaxSize), nil
	})
	if err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}

	return &types.QueryListOperatorAuthorizationsResponse{
		OperatorAuthorizations: results,
	}, nil
}
