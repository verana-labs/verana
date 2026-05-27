package keeper

import (
	"context"
	"sort"

	"cosmossdk.io/collections"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/verana-labs/verana/x/gf/types"
)

func (q querier) GetGovernanceFrameworkVersion(goCtx context.Context, req *types.QueryGetGovernanceFrameworkVersionRequest) (*types.QueryGetGovernanceFrameworkVersionResponse, error) {
	if req == nil {
		return nil, status.Error(codes.InvalidArgument, "empty request")
	}
	gfv, err := q.GFVersion.Get(goCtx, req.Id)
	if err != nil {
		return nil, status.Errorf(codes.NotFound, "gfv %d not found", req.Id)
	}
	docs, err := q.collectDocs(goCtx, gfv.Id, req.PreferredLanguage)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "collect docs: %v", err)
	}
	return &types.QueryGetGovernanceFrameworkVersionResponse{
		Version: types.GovernanceFrameworkVersionWithDocs{
			Id:            gfv.Id,
			EcosystemId:   gfv.EcosystemId,
			CorporationId: gfv.CorporationId,
			Created:       gfv.Created,
			Version:       gfv.Version,
			ActiveSince:   gfv.ActiveSince,
			Documents:     docs,
		},
	}, nil
}

func (q querier) ListGovernanceFrameworkVersions(goCtx context.Context, req *types.QueryListGovernanceFrameworkVersionsRequest) (*types.QueryListGovernanceFrameworkVersionsResponse, error) {
	if req == nil {
		return nil, status.Error(codes.InvalidArgument, "empty request")
	}
	hasEco := req.EcosystemId > 0
	hasCorp := req.CorporationId > 0
	if hasEco == hasCorp {
		return nil, status.Error(codes.InvalidArgument, "exactly one of ecosystem_id or corporation_id must be set")
	}
	if req.ResponseMaxSize == 0 {
		req.ResponseMaxSize = 64
	}
	if req.ResponseMaxSize > 1024 {
		return nil, status.Error(codes.InvalidArgument, "response_max_size must be <= 1024")
	}

	// Spec MOD-GF-QRY-2-1: active_only returns "only the entry corresponding to
	// the subject's active_version" — resolve subject's active_version once.
	var subjectActiveVersion uint32
	if req.ActiveOnly {
		if hasEco {
			eco, ok := q.ecosystemKeeper().GetEcosystemView(goCtx, req.EcosystemId)
			if !ok {
				return nil, status.Errorf(codes.NotFound, "ecosystem %d not found", req.EcosystemId)
			}
			subjectActiveVersion = eco.ActiveVersion
		} else {
			coView, ok := q.corporationKeeper().GetByID(goCtx, req.CorporationId)
			if !ok {
				return nil, status.Errorf(codes.NotFound, "corporation %d not found", req.CorporationId)
			}
			subjectActiveVersion = coView.ActiveVersion
		}
	}

	var gfvIDs []uint64
	if hasEco {
		iter, err := q.GFVersionByEcosystem.Iterate(goCtx, collections.NewPrefixedPairRange[uint64, uint32](req.EcosystemId))
		if err != nil {
			return nil, status.Errorf(codes.Internal, "iterate: %v", err)
		}
		defer iter.Close()
		for ; iter.Valid(); iter.Next() {
			id, err := iter.Value()
			if err != nil {
				return nil, status.Errorf(codes.Internal, "iter value: %v", err)
			}
			gfvIDs = append(gfvIDs, id)
		}
	} else {
		iter, err := q.GFVersionByCorporation.Iterate(goCtx, collections.NewPrefixedPairRange[uint64, uint32](req.CorporationId))
		if err != nil {
			return nil, status.Errorf(codes.Internal, "iterate: %v", err)
		}
		defer iter.Close()
		for ; iter.Valid(); iter.Next() {
			id, err := iter.Value()
			if err != nil {
				return nil, status.Errorf(codes.Internal, "iter value: %v", err)
			}
			gfvIDs = append(gfvIDs, id)
		}
	}

	versions := make([]types.GovernanceFrameworkVersionWithDocs, 0, len(gfvIDs))
	for _, id := range gfvIDs {
		gfv, err := q.GFVersion.Get(goCtx, id)
		if err != nil {
			return nil, status.Errorf(codes.Internal, "fetch gfv %d: %v", id, err)
		}
		if req.ActiveOnly && gfv.Version != subjectActiveVersion {
			continue
		}
		docs, err := q.collectDocs(goCtx, gfv.Id, req.PreferredLanguage)
		if err != nil {
			return nil, status.Errorf(codes.Internal, "collect docs: %v", err)
		}
		versions = append(versions, types.GovernanceFrameworkVersionWithDocs{
			Id:            gfv.Id,
			EcosystemId:   gfv.EcosystemId,
			CorporationId: gfv.CorporationId,
			Created:       gfv.Created,
			Version:       gfv.Version,
			ActiveSince:   gfv.ActiveSince,
			Documents:     docs,
		})
	}
	// Spec MOD-GF-QRY-2-3: order by ascending version.
	sort.Slice(versions, func(i, j int) bool { return versions[i].Version < versions[j].Version })
	if uint32(len(versions)) > req.ResponseMaxSize {
		versions = versions[:req.ResponseMaxSize]
	}
	return &types.QueryListGovernanceFrameworkVersionsResponse{Versions: versions}, nil
}

func (q querier) collectDocs(ctx context.Context, gfvID uint64, preferredLang string) ([]types.GovernanceFrameworkDocument, error) {
	var out []types.GovernanceFrameworkDocument
	var preferred *types.GovernanceFrameworkDocument
	if err := q.GFDocument.Walk(ctx, nil, func(_ uint64, d types.GovernanceFrameworkDocument) (bool, error) {
		if d.GfvId != gfvID {
			return false, nil
		}
		if preferredLang != "" {
			if d.Language == preferredLang && preferred == nil {
				cp := d
				preferred = &cp
			}
			return false, nil
		}
		out = append(out, d)
		return false, nil
	}); err != nil {
		return nil, err
	}
	if preferredLang != "" {
		if preferred != nil {
			return []types.GovernanceFrameworkDocument{*preferred}, nil
		}
		// Fall back to all docs if preferred language not present (spec QRY-1-3 says "preferring").
		_ = q.GFDocument.Walk(ctx, nil, func(_ uint64, d types.GovernanceFrameworkDocument) (bool, error) {
			if d.GfvId == gfvID {
				out = append(out, d)
			}
			return false, nil
		})
		return out, nil
	}
	return out, nil
}
