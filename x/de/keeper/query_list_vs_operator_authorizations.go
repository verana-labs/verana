package keeper

import (
	"context"

	"cosmossdk.io/collections"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/verana-labs/verana/x/de/types"
)

// ListVSOperatorAuthorizations implements [MOD-DE-QRY-2].
func (q queryServer) ListVSOperatorAuthorizations(ctx context.Context, req *types.QueryListVSOperatorAuthorizationsRequest) (*types.QueryListVSOperatorAuthorizationsResponse, error) {
	if req == nil {
		return nil, status.Error(codes.InvalidArgument, "invalid request")
	}

	// [MOD-DE-QRY-2-2] Validate response_max_size
	if req.ResponseMaxSize == 0 {
		req.ResponseMaxSize = 64
	}
	if req.ResponseMaxSize < 1 || req.ResponseMaxSize > 1024 {
		return nil, status.Error(codes.InvalidArgument, "response_max_size must be between 1 and 1,024")
	}

	// [MOD-DE-QRY-2-3] Walk through all VS operator authorizations and apply filters
	var results []types.VSOperatorAuthorization

	err := q.k.VSOperatorAuthorizations.Walk(ctx, nil, func(key collections.Pair[string, string], vsoa types.VSOperatorAuthorization) (bool, error) {
		// Filter by authority if specified
		if req.Authority != "" && vsoa.Authority != req.Authority {
			return false, nil
		}
		// Filter by vs_operator if specified
		if req.VsOperator != "" && vsoa.VsOperator != req.VsOperator {
			return false, nil
		}

		results = append(results, vsoa)
		return len(results) >= int(req.ResponseMaxSize), nil
	})
	if err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}

	return &types.QueryListVSOperatorAuthorizationsResponse{
		VsOperatorAuthorizations: results,
	}, nil
}
