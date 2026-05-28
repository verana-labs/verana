package keeper

import (
	"context"
	"sort"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/verana-labs/verana/x/co/types"
)

// GetCorporation implements MOD-CO-QRY-1.
func (q querier) GetCorporation(goCtx context.Context, req *types.QueryGetCorporationRequest) (*types.QueryGetCorporationResponse, error) {
	if req == nil {
		return nil, status.Error(codes.InvalidArgument, "empty request")
	}
	if req.CorporationId == 0 {
		return nil, status.Error(codes.InvalidArgument, "corporation_id must be > 0")
	}
	co, err := q.Corporation.Get(goCtx, req.CorporationId)
	if err != nil {
		return nil, status.Errorf(codes.NotFound, "corporation %d not found", req.CorporationId)
	}
	versions, err := q.gfKeeper.ListVersionsByCorporation(goCtx, co.Id, co.ActiveVersion, req.ActiveGfOnly, req.PreferredLanguage)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "list gf versions: %v", err)
	}
	return &types.QueryGetCorporationResponse{
		Corporation: types.CorporationWithGF{
			Id:            co.Id,
			PolicyAddress: co.PolicyAddress,
			Did:           co.Did,
			Created:       co.Created,
			Modified:      co.Modified,
			Language:      co.Language,
			ActiveVersion: co.ActiveVersion,
			Versions:      versions,
		},
	}, nil
}

// ListCorporations implements MOD-CO-QRY-2. Ordered by `modified` descending.
func (q querier) ListCorporations(goCtx context.Context, req *types.QueryListCorporationsRequest) (*types.QueryListCorporationsResponse, error) {
	if req == nil {
		return nil, status.Error(codes.InvalidArgument, "empty request")
	}
	if req.ResponseMaxSize == 0 {
		req.ResponseMaxSize = 64
	}
	if req.ResponseMaxSize > 1024 {
		return nil, status.Error(codes.InvalidArgument, "response_max_size must be <= 1024")
	}

	var corps []types.Corporation
	if err := q.Corporation.Walk(goCtx, nil, func(_ uint64, co types.Corporation) (bool, error) {
		if req.ModifiedAfter != nil && !co.Modified.After(*req.ModifiedAfter) {
			return false, nil
		}
		corps = append(corps, co)
		return false, nil
	}); err != nil {
		return nil, status.Errorf(codes.Internal, "walk corporations: %v", err)
	}

	sort.Slice(corps, func(i, j int) bool { return corps[i].Modified.After(corps[j].Modified) })
	if uint32(len(corps)) > req.ResponseMaxSize {
		corps = corps[:req.ResponseMaxSize]
	}

	out := make([]types.CorporationWithGF, 0, len(corps))
	for _, co := range corps {
		versions, err := q.gfKeeper.ListVersionsByCorporation(goCtx, co.Id, co.ActiveVersion, req.ActiveGfOnly, req.PreferredLanguage)
		if err != nil {
			return nil, status.Errorf(codes.Internal, "list gf versions for co %d: %v", co.Id, err)
		}
		out = append(out, types.CorporationWithGF{
			Id:            co.Id,
			PolicyAddress: co.PolicyAddress,
			Did:           co.Did,
			Created:       co.Created,
			Modified:      co.Modified,
			Language:      co.Language,
			ActiveVersion: co.ActiveVersion,
			Versions:      versions,
		})
	}
	return &types.QueryListCorporationsResponse{Corporations: out}, nil
}
