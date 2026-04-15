package keeper

import (
	"bytes"
	"context"
	"fmt"

	errorsmod "cosmossdk.io/errors"
	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/verana-labs/verana/x/xr/types"
)

func (ms msgServer) ToggleExchangeRateState(ctx context.Context, msg *types.MsgToggleExchangeRateState) (*types.MsgToggleExchangeRateStateResponse, error) {
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

	// Load ExchangeRate by id (must exist)
	xr, err := ms.ExchangeRates.Get(ctx, msg.Id)
	if err != nil {
		return nil, errorsmod.Wrapf(types.ErrExchangeRateNotFound, "exchange rate with id %d not found", msg.Id)
	}

	sdkCtx := sdk.UnwrapSDKContext(ctx)
	blockTime := sdkCtx.BlockTime()

	// Set state to the provided value
	xr.State = msg.State
	// Set updated to block time
	xr.Updated = blockTime

	// Save
	if err := ms.ExchangeRates.Set(ctx, msg.Id, xr); err != nil {
		return nil, errorsmod.Wrap(err, "failed to store exchange rate")
	}

	// Emit event
	sdkCtx.EventManager().EmitEvent(
		sdk.NewEvent(
			types.EventTypeToggleExchangeRateState,
			sdk.NewAttribute(types.AttributeKeyID, fmt.Sprintf("%d", msg.Id)),
			sdk.NewAttribute(types.AttributeKeyAuthority, msg.Authority),
			sdk.NewAttribute(types.AttributeKeyState, fmt.Sprintf("%t", msg.State)),
		),
	)

	return &types.MsgToggleExchangeRateStateResponse{}, nil
}
