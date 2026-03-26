package keeper

import (
	"context"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	cstypes "github.com/verana-labs/verana/x/cs/types"
	"github.com/verana-labs/verana/x/xr/types"
)

func (q queryServer) ListExchangeRates(ctx context.Context, req *types.QueryListExchangeRatesRequest) (*types.QueryListExchangeRatesResponse, error) {
	if req == nil {
		return nil, status.Error(codes.InvalidArgument, "invalid request")
	}

	var results []types.ExchangeRate

	err := q.k.ExchangeRates.Walk(ctx, nil, func(key uint64, xr types.ExchangeRate) (bool, error) {
		// Filter by base_asset_type
		if req.BaseAssetType != cstypes.PricingAssetType_PRICING_ASSET_TYPE_UNSPECIFIED && xr.BaseAssetType != req.BaseAssetType {
			return false, nil
		}
		// Filter by base_asset
		if req.BaseAsset != "" && xr.BaseAsset != req.BaseAsset {
			return false, nil
		}
		// Filter by quote_asset_type
		if req.QuoteAssetType != cstypes.PricingAssetType_PRICING_ASSET_TYPE_UNSPECIFIED && xr.QuoteAssetType != req.QuoteAssetType {
			return false, nil
		}
		// Filter by quote_asset
		if req.QuoteAsset != "" && xr.QuoteAsset != req.QuoteAsset {
			return false, nil
		}
		// Filter by state
		if req.State == types.StateFilter_STATE_FILTER_ACTIVE && !xr.State {
			return false, nil
		}
		if req.State == types.StateFilter_STATE_FILTER_INACTIVE && xr.State {
			return false, nil
		}
		// Filter by expire: return only if expires > expire
		if req.Expire != nil && !xr.Expires.After(*req.Expire) {
			return false, nil
		}

		results = append(results, xr)
		return false, nil
	})
	if err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}

	return &types.QueryListExchangeRatesResponse{ExchangeRates: results}, nil
}
