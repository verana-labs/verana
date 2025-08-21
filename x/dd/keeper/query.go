package keeper

import (
	"context"
	"fmt"
	"sort"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/verana-labs/verana/x/dd/types"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

var _ types.QueryServer = Keeper{}

func (k Keeper) ListDIDs(ctx context.Context, req *types.QueryListDIDsRequest) (*types.QueryListDIDsResponse, error) {
	if req == nil {
		return nil, status.Error(codes.InvalidArgument, "empty request")
	}

	// Validate response_max_size
	if req.ResponseMaxSize == 0 {
		req.ResponseMaxSize = 64 // default value
	}
	if req.ResponseMaxSize > 1024 {
		return nil, status.Error(codes.InvalidArgument, "response_max_size must be between 1 and 1,024")
	}

	sdkCtx := sdk.UnwrapSDKContext(ctx)
	params := k.GetParams(ctx)

	var dids []types.DIDDirectory
	now := sdkCtx.BlockTime()
	gracePeriod := now.AddDate(0, 0, -int(params.DidDirectoryGracePeriod))

	err := k.DIDDirectory.Walk(sdkCtx, nil, func(key string, did types.DIDDirectory) (bool, error) {
		// Apply basic filters first
		if req.Account != "" && did.Controller != req.Account {
			return false, nil
		}
		if req.Changed != nil && !did.Modified.After(*req.Changed) {
			return false, nil
		}

		// Check expiration status
		isExpired := did.Exp.Before(now)
		isOverGrace := did.Exp.Before(gracePeriod)

		// Special filtering cases
		if req.OverGrace {
			// When over_grace is true, show only over grace period DIDs
			if !isOverGrace {
				return false, nil
			}
		} else if req.Expired {
			// When expired is true, show all expired DIDs (both normal expired and over grace)
			if !isExpired {
				return false, nil
			}
		}
		// When neither flag is set, show all DIDs (no filtering on expiration)

		dids = append(dids, did)
		return len(dids) >= int(req.ResponseMaxSize), nil
	})

	if err != nil {
		return nil, status.Error(codes.Internal, fmt.Sprintf("failed to list DIDs: %v", err))
	}

	// Sort by modified time ascending
	sort.Slice(dids, func(i, j int) bool {
		return dids[i].Modified.Before(dids[j].Modified)
	})

	return &types.QueryListDIDsResponse{
		Dids: dids,
	}, nil
}

func (k Keeper) GetDID(ctx context.Context, req *types.QueryGetDIDRequest) (*types.QueryGetDIDResponse, error) {
	if req == nil {
		return nil, status.Error(codes.InvalidArgument, "empty request")
	}

	// Check DID format
	if !isValidDID(req.Did) {
		return nil, status.Error(codes.InvalidArgument, "invalid DID syntax")
	}

	// Get the DID entry
	didEntry, err := k.DIDDirectory.Get(ctx, req.Did)
	if err != nil {
		return nil, status.Error(codes.NotFound, fmt.Sprintf("DID %s not found", req.Did))
	}

	return &types.QueryGetDIDResponse{
		DidEntry: didEntry,
	}, nil
}
