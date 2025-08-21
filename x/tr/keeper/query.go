package keeper

import (
	"context"
	"errors"
	"sort"

	"cosmossdk.io/collections"
	"github.com/verana-labs/verana-blockchain/x/tr/types"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

var _ types.QueryServer = queryServer{}

func NewQueryServerImpl(k Keeper) types.QueryServer {
	return queryServer{k}
}

type queryServer struct {
	k Keeper
}

func (qs queryServer) GetTrustRegistry(ctx context.Context, req *types.QueryGetTrustRegistryRequest) (*types.QueryGetTrustRegistryResponse, error) {
	if req.TrId == 0 {
		return nil, status.Error(codes.InvalidArgument, "trust registry ID is required")
	}

	// Direct lookup by ID
	tr, err := qs.k.TrustRegistry.Get(ctx, req.TrId)
	if err != nil {
		if errors.Is(err, collections.ErrNotFound) {
			return nil, status.Error(codes.NotFound, "trust registry not found")
		}
		return nil, status.Error(codes.Internal, err.Error())
	}

	// Get versions with nested documents
	trWithVersions, err := qs.getTrustRegistryWithVersions(ctx, tr, req.ActiveGfOnly, req.PreferredLanguage)
	if err != nil {
		return nil, err
	}

	return &types.QueryGetTrustRegistryResponse{
		TrustRegistry: trWithVersions,
	}, nil
}

func (qs queryServer) getTrustRegistryWithVersions(ctx context.Context, tr types.TrustRegistry, activeOnly bool, preferredLang string) (*types.TrustRegistryWithVersions, error) {
	var versionsWithDocs []types.GovernanceFrameworkVersionWithDocs

	// Fetch all versions for this trust registry
	err := qs.k.GFVersion.Walk(ctx, nil, func(id uint64, gfv types.GovernanceFrameworkVersion) (bool, error) {
		if gfv.TrId == tr.Id {
			if !activeOnly || gfv.Version == tr.ActiveVersion {
				var docs []types.GovernanceFrameworkDocument

				// Fetch documents for this version
				err := qs.k.GFDocument.Walk(ctx, nil, func(docId uint64, gfd types.GovernanceFrameworkDocument) (bool, error) {
					if gfd.GfvId == gfv.Id {
						if preferredLang == "" || gfd.Language == preferredLang {
							docs = append(docs, gfd)
						}
					}
					return false, nil
				})
				if err != nil {
					return true, err
				}

				// If we have a preferred language but didn't find a matching document,
				// include the first document as fallback
				if preferredLang != "" && len(docs) == 0 {
					err := qs.k.GFDocument.Walk(ctx, nil, func(docId uint64, gfd types.GovernanceFrameworkDocument) (bool, error) {
						if gfd.GfvId == gfv.Id {
							docs = append(docs, gfd)
							return true, nil
						}
						return false, nil
					})
					if err != nil {
						return true, err
					}
				}

				versionWithDocs := types.GovernanceFrameworkVersionWithDocs{
					Id:          gfv.Id,
					TrId:        gfv.TrId,
					Created:     gfv.Created,
					Version:     gfv.Version,
					ActiveSince: gfv.ActiveSince,
					Documents:   docs,
				}
				versionsWithDocs = append(versionsWithDocs, versionWithDocs)
			}
		}
		return false, nil
	})
	if err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}

	return &types.TrustRegistryWithVersions{
		Id:            tr.Id,
		Did:           tr.Did,
		Controller:    tr.Controller,
		Created:       tr.Created,
		Modified:      tr.Modified,
		Archived:      tr.Archived,
		Deposit:       tr.Deposit,
		Aka:           tr.Aka,
		ActiveVersion: tr.ActiveVersion,
		Language:      tr.Language,
		Versions:      versionsWithDocs,
	}, nil
}

func (qs queryServer) ListTrustRegistries(ctx context.Context, req *types.QueryListTrustRegistriesRequest) (*types.QueryListTrustRegistriesResponse, error) {
	// Validate response_max_size
	if req.ResponseMaxSize < 1 || req.ResponseMaxSize > 1024 {
		return nil, status.Error(codes.InvalidArgument, "response_max_size must be between 1 and 1,024")
	}

	var registriesWithVersions []types.TrustRegistryWithVersions

	// Collect all matching trust registries with their nested versions and documents
	err := qs.k.TrustRegistry.Walk(ctx, nil, func(key uint64, tr types.TrustRegistry) (bool, error) {
		// Apply filters
		if req.Controller != "" && tr.Controller != req.Controller {
			return false, nil
		}
		if req.ModifiedAfter != nil && !tr.Modified.After(*req.ModifiedAfter) {
			return false, nil
		}

		// Get versions with nested documents for this trust registry
		trWithVersions, err := qs.getTrustRegistryWithVersions(ctx, tr, req.ActiveGfOnly, req.PreferredLanguage)
		if err != nil {
			return true, err
		}

		registriesWithVersions = append(registriesWithVersions, *trWithVersions)
		return len(registriesWithVersions) >= int(req.ResponseMaxSize), nil
	})
	if err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}

	// Sort by modified time ascending
	sort.Slice(registriesWithVersions, func(i, j int) bool {
		return registriesWithVersions[i].Modified.Before(registriesWithVersions[j].Modified)
	})

	return &types.QueryListTrustRegistriesResponse{
		TrustRegistries: registriesWithVersions,
	}, nil
}

// Params defines the handler for the Query/Params RPC method.
func (qs queryServer) Params(ctx context.Context, req *types.QueryParamsRequest) (*types.QueryParamsResponse, error) {
	params, err := qs.k.Params.Get(ctx)
	if err != nil {
		if errors.Is(err, collections.ErrNotFound) {
			return &types.QueryParamsResponse{Params: types.Params{}}, nil
		}
		return nil, status.Error(codes.Internal, err.Error())
	}

	return &types.QueryParamsResponse{Params: params}, nil
}
