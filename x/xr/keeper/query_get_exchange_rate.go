package keeper

import (
	"context"
	"errors"
	"fmt"

	"cosmossdk.io/collections"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	cstypes "github.com/verana-labs/verana/x/cs/types"
	"github.com/verana-labs/verana/x/xr/types"
)

func (q queryServer) GetExchangeRate(ctx context.Context, req *types.QueryGetExchangeRateRequest) (*types.QueryGetExchangeRateResponse, error) {
	if req == nil {
		return nil, status.Error(codes.InvalidArgument, "invalid request")
	}

	var xr types.ExchangeRate
	var err error

	if req.Id > 0 {
		// Lookup by ID
		xr, err = q.k.ExchangeRates.Get(ctx, req.Id)
		if err != nil {
			if errors.Is(err, collections.ErrNotFound) {
				return nil, status.Error(codes.NotFound, "exchange rate not found")
			}
			return nil, status.Error(codes.Internal, err.Error())
		}
	} else {
		// Lookup by asset pair
		if req.BaseAssetType == cstypes.PricingAssetType_PRICING_ASSET_TYPE_UNSPECIFIED ||
			req.BaseAsset == "" ||
			req.QuoteAssetType == cstypes.PricingAssetType_PRICING_ASSET_TYPE_UNSPECIFIED ||
			req.QuoteAsset == "" {
			return nil, status.Error(codes.InvalidArgument, "must provide id or all of base_asset_type, base_asset, quote_asset_type, quote_asset")
		}

		pairKey := fmt.Sprintf("%d:%s:%d:%s", req.BaseAssetType, req.BaseAsset, req.QuoteAssetType, req.QuoteAsset)
		id, err := q.k.PairIndex.Get(ctx, pairKey)
		if err != nil {
			if errors.Is(err, collections.ErrNotFound) {
				return nil, status.Error(codes.NotFound, "exchange rate not found")
			}
			return nil, status.Error(codes.Internal, err.Error())
		}

		xr, err = q.k.ExchangeRates.Get(ctx, id)
		if err != nil {
			return nil, status.Error(codes.Internal, err.Error())
		}
	}

	// Apply state filter
	if req.State == types.StateFilter_STATE_FILTER_ACTIVE && !xr.State {
		return nil, status.Error(codes.NotFound, "exchange rate not found")
	}
	if req.State == types.StateFilter_STATE_FILTER_INACTIVE && xr.State {
		return nil, status.Error(codes.NotFound, "exchange rate not found")
	}

	// Apply expire_ts filter: return only if expires > expire_ts
	if req.ExpireTs != nil && !xr.Expires.After(*req.ExpireTs) {
		return nil, status.Error(codes.NotFound, "exchange rate not found")
	}

	return &types.QueryGetExchangeRateResponse{ExchangeRate: xr}, nil
}
