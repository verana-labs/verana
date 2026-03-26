package keeper

import (
	"bytes"
	"context"
	"fmt"

	errorsmod "cosmossdk.io/errors"
	sdk "github.com/cosmos/cosmos-sdk/types"

	cstypes "github.com/verana-labs/verana/x/cs/types"
	"github.com/verana-labs/verana/x/xr/types"
)

func (ms msgServer) CreateExchangeRate(ctx context.Context, msg *types.MsgCreateExchangeRate) (*types.MsgCreateExchangeRateResponse, error) {
	// Validate basic fields
	if err := msg.ValidateBasic(); err != nil {
		return nil, err
	}

	// Governance authority check
	authority, err := ms.addressCodec.StringToBytes(msg.Authority)
	if err != nil {
		return nil, errorsmod.Wrap(err, "invalid authority address")
	}

	if !bytes.Equal(ms.GetAuthority(), authority) {
		expectedAuthorityStr, _ := ms.addressCodec.BytesToString(ms.GetAuthority())
		return nil, errorsmod.Wrapf(types.ErrInvalidSigner, "invalid authority; expected %s, got %s", expectedAuthorityStr, msg.Authority)
	}

	// Check pair uniqueness
	pairKey := buildPairKey(msg.BaseAssetType, msg.BaseAsset, msg.QuoteAssetType, msg.QuoteAsset)
	_, err = ms.PairIndex.Get(ctx, pairKey)
	if err == nil {
		return nil, errorsmod.Wrapf(types.ErrDuplicatePair, "exchange rate pair already exists for %s", pairKey)
	}

	// Auto-generate ID
	id, err := ms.GetNextID(ctx, types.CounterKeyExchangeRate)
	if err != nil {
		return nil, errorsmod.Wrap(err, "failed to generate exchange rate ID")
	}

	sdkCtx := sdk.UnwrapSDKContext(ctx)
	blockTime := sdkCtx.BlockTime()

	// Create ExchangeRate entry
	xr := types.ExchangeRate{
		Id:               id,
		BaseAssetType:    msg.BaseAssetType,
		BaseAsset:        msg.BaseAsset,
		QuoteAssetType:   msg.QuoteAssetType,
		QuoteAsset:       msg.QuoteAsset,
		Rate:             msg.Rate,
		RateScale:        msg.RateScale,
		ValidityDuration: msg.ValidityDuration,
		Expires:          blockTime.Add(msg.ValidityDuration),
		State:            false,
		Updated:          blockTime,
	}

	// Store in collection
	if err := ms.ExchangeRates.Set(ctx, id, xr); err != nil {
		return nil, errorsmod.Wrap(err, "failed to store exchange rate")
	}

	// Store pair index for uniqueness
	if err := ms.PairIndex.Set(ctx, pairKey, id); err != nil {
		return nil, errorsmod.Wrap(err, "failed to store pair index")
	}

	// Emit event
	sdkCtx.EventManager().EmitEvent(
		sdk.NewEvent(
			types.EventTypeCreateExchangeRate,
			sdk.NewAttribute(types.AttributeKeyID, fmt.Sprintf("%d", id)),
			sdk.NewAttribute(types.AttributeKeyAuthority, msg.Authority),
			sdk.NewAttribute(types.AttributeKeyBaseAssetType, msg.BaseAssetType.String()),
			sdk.NewAttribute(types.AttributeKeyBaseAsset, msg.BaseAsset),
			sdk.NewAttribute(types.AttributeKeyQuoteAssetType, msg.QuoteAssetType.String()),
			sdk.NewAttribute(types.AttributeKeyQuoteAsset, msg.QuoteAsset),
		),
	)

	return &types.MsgCreateExchangeRateResponse{Id: id}, nil
}

func buildPairKey(baseType cstypes.PricingAssetType, baseAsset string, quoteType cstypes.PricingAssetType, quoteAsset string) string {
	return fmt.Sprintf("%d:%s:%d:%s", baseType, baseAsset, quoteType, quoteAsset)
}
