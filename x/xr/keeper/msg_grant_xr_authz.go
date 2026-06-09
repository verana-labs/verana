package keeper

import (
	"bytes"
	"context"
	"fmt"

	"cosmossdk.io/collections"
	errorsmod "cosmossdk.io/errors"
	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/verana-labs/verana/x/xr/types"
)

// GrantExchangeRateAuthorization implements [MOD-XR-MSG-4] Grant Exchange Rate Authorization.
func (ms msgServer) GrantExchangeRateAuthorization(ctx context.Context, msg *types.MsgGrantExchangeRateAuthorization) (*types.MsgGrantExchangeRateAuthorizationResponse, error) {
	if err := msg.ValidateBasic(); err != nil {
		return nil, err
	}

	// [MOD-XR-MSG-4-2-1] authority MUST equal the gov module account.
	authority, err := ms.addressCodec.StringToBytes(msg.Authority)
	if err != nil {
		return nil, errorsmod.Wrap(err, "invalid authority address")
	}
	if !bytes.Equal(ms.GetAuthority(), authority) {
		expected, _ := ms.addressCodec.BytesToString(ms.GetAuthority())
		return nil, errorsmod.Wrapf(types.ErrInvalidSigner, "invalid authority; expected %s, got %s", expected, msg.Authority)
	}

	sdkCtx := sdk.UnwrapSDKContext(ctx)
	now := sdkCtx.BlockTime()

	// xr_id MUST refer to an existing ExchangeRate.
	if _, err := ms.ExchangeRates.Get(ctx, msg.XrId); err != nil {
		return nil, errorsmod.Wrapf(types.ErrExchangeRateNotFound, "exchange rate with id %d not found", msg.XrId)
	}

	// expiration MUST be in the future.
	if !msg.Expiration.After(now) {
		return nil, types.ErrInvalidExpiration
	}

	// Build the authorization record (zero min_interval / max_deviation_bps = unset).
	authz := types.ExchangeRateAuthorization{
		XrId:            msg.XrId,
		Operator:        msg.Operator,
		Expiration:      *msg.Expiration,
		MaxDeviationBps: msg.MaxDeviationBps,
	}
	if msg.MinInterval != nil {
		authz.MinInterval = *msg.MinInterval
	}

	key := collections.Join(msg.XrId, msg.Operator)
	if err := ms.ExchangeRateAuthorizations.Set(ctx, key, authz); err != nil {
		return nil, errorsmod.Wrap(err, "failed to store exchange rate authorization")
	}

	sdkCtx.EventManager().EmitEvent(
		sdk.NewEvent(
			types.EventTypeGrantExchangeRateAuthz,
			sdk.NewAttribute(types.AttributeKeyID, fmt.Sprintf("%d", msg.XrId)),
			sdk.NewAttribute(types.AttributeKeyOperator, msg.Operator),
		),
	)

	return &types.MsgGrantExchangeRateAuthorizationResponse{}, nil
}
