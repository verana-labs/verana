package keeper

import (
	"context"

	"errors"

	"cosmossdk.io/collections"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/verana-labs/verana/x/di/types"
)

// GetDigest implements [MOD-DI-QRY-1] Get Digest.
func (q queryServer) GetDigest(ctx context.Context, req *types.QueryGetDigestRequest) (*types.QueryGetDigestResponse, error) {
	if req == nil {
		return nil, status.Error(codes.InvalidArgument, "invalid request")
	}

	// [MOD-DI-QRY-1-2] digest MUST not be empty.
	if req.Digest == "" {
		return nil, status.Error(codes.InvalidArgument, "digest must not be empty")
	}

	// [MOD-DI-QRY-1-3] Return found Digest entry matching digest.
	stored, err := q.k.Digests.Get(ctx, req.Digest)
	if err != nil {
		if errors.Is(err, collections.ErrNotFound) {
			return nil, status.Error(codes.NotFound, "digest not found")
		}
		return nil, status.Error(codes.Internal, err.Error())
	}

	return &types.QueryGetDigestResponse{
		Digest: &types.DigestInfo{
			Digest:  stored.Digest,
			Created: stored.Created,
		},
	}, nil
}
