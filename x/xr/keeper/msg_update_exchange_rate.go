package keeper

import (
	"context"
	"fmt"

	errorsmod "cosmossdk.io/errors"
	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/verana-labs/verana/x/xr/types"
)

// UpdateExchangeRate implements [MOD-XR-MSG-2] Update Exchange Rate.
func (ms msgServer) UpdateExchangeRate(ctx context.Context, msg *types.MsgUpdateExchangeRate) (*types.MsgUpdateExchangeRateResponse, error) {
	// Validate basic fields
	if err := msg.ValidateBasic(); err != nil {
		return nil, err
	}

	sdkCtx := sdk.UnwrapSDKContext(ctx)
	now := sdkCtx.BlockTime()

	// [AUTHZ-CHECK] Verify operator authorization via DE module
	if ms.delegationKeeper != nil {
		if err := ms.delegationKeeper.CheckOperatorAuthorization(
			ctx,
			msg.Authority,
			msg.Operator,
			"/verana.xr.v1.MsgUpdateExchangeRate",
			now,
		); err != nil {
			return nil, fmt.Errorf("authorization check failed: %w", err)
		}
	}

	// Load ExchangeRate by id
	xr, err := ms.ExchangeRates.Get(ctx, msg.Id)
	if err != nil {
		return nil, errorsmod.Wrapf(types.ErrExchangeRateNotFound, "exchange rate with id %d not found", msg.Id)
	}

	// Check xr.state == true (active/enabled)
	if !xr.State {
		return nil, errorsmod.Wrapf(types.ErrExchangeRateNotActive, "exchange rate with id %d is not active", msg.Id)
	}

	// Update fields per spec
	xr.Rate = msg.Rate
	xr.Expires = now.Add(xr.ValidityDuration)
	xr.Updated = now

	// Save updated exchange rate
	if err := ms.ExchangeRates.Set(ctx, msg.Id, xr); err != nil {
		return nil, errorsmod.Wrap(err, "failed to store updated exchange rate")
	}

	// Emit event
	sdkCtx.EventManager().EmitEvent(
		sdk.NewEvent(
			types.EventTypeUpdateExchangeRate,
			sdk.NewAttribute(types.AttributeKeyID, fmt.Sprintf("%d", msg.Id)),
			sdk.NewAttribute(types.AttributeKeyAuthority, msg.Authority),
			sdk.NewAttribute(types.AttributeKeyOperator, msg.Operator),
			sdk.NewAttribute(types.AttributeKeyRate, msg.Rate),
		),
	)

	return &types.MsgUpdateExchangeRateResponse{}, nil
}
