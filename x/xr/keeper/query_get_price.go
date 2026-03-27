package keeper

import (
	"context"
	"errors"
	"fmt"

	"cosmossdk.io/collections"
	"cosmossdk.io/math"
	sdk "github.com/cosmos/cosmos-sdk/types"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	cstypes "github.com/verana-labs/verana/x/cs/types"
	"github.com/verana-labs/verana/x/xr/types"
)

func (q queryServer) GetPrice(ctx context.Context, req *types.QueryGetPriceRequest) (*types.QueryGetPriceResponse, error) {
	if req == nil {
		return nil, status.Error(codes.InvalidArgument, "invalid request")
	}

	if req.Amount == "" {
		return nil, status.Error(codes.InvalidArgument, "amount is required")
	}

	price, err := q.k.GetPrice(ctx, req.BaseAssetType, req.BaseAsset, req.QuoteAssetType, req.QuoteAsset, req.Amount)
	if err != nil {
		return nil, err
	}

	return &types.QueryGetPriceResponse{Price: price}, nil
}

// GetPrice computes the converted price using an exchange rate.
// This method can be called by other modules.
func (k Keeper) GetPrice(ctx context.Context, baseAssetType cstypes.PricingAssetType, baseAsset string, quoteAssetType cstypes.PricingAssetType, quoteAsset string, amount string) (string, error) {
	// Parse amount
	amountInt, ok := math.NewIntFromString(amount)
	if !ok {
		return "", status.Error(codes.InvalidArgument, "invalid amount: must be a valid integer")
	}

	// Same asset pair: return amount directly
	if baseAssetType == quoteAssetType && baseAsset == quoteAsset {
		return amountInt.String(), nil
	}

	// Look up exchange rate by pair
	pairKey := fmt.Sprintf("%d:%s:%d:%s", baseAssetType, baseAsset, quoteAssetType, quoteAsset)
	id, err := k.PairIndex.Get(ctx, pairKey)
	if err != nil {
		if errors.Is(err, collections.ErrNotFound) {
			return "", status.Error(codes.NotFound, "exchange rate not found")
		}
		return "", status.Error(codes.Internal, err.Error())
	}

	xr, err := k.ExchangeRates.Get(ctx, id)
	if err != nil {
		return "", status.Error(codes.Internal, err.Error())
	}

	// Check active
	if !xr.State {
		return "", status.Error(codes.FailedPrecondition, "exchange rate is not active")
	}

	// Check not expired
	sdkCtx := sdk.UnwrapSDKContext(ctx)
	blockTime := sdkCtx.BlockTime()
	if !xr.Expires.After(blockTime) {
		return "", status.Error(codes.FailedPrecondition, "exchange rate is expired")
	}

	// Compute price = floor(amount * rate / 10^rate_scale) using integer arithmetic
	rate, ok := math.NewIntFromString(xr.Rate)
	if !ok {
		return "", status.Error(codes.Internal, "invalid stored rate")
	}

	// 10^rate_scale
	divisor := math.NewInt(1)
	ten := math.NewInt(10)
	for i := uint32(0); i < xr.RateScale; i++ {
		divisor = divisor.Mul(ten)
	}

	// price = floor(amount * rate / divisor)
	price := amountInt.Mul(rate).Quo(divisor)

	return price.String(), nil
}
