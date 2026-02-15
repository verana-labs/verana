package keeper

import (
	"context"
	"fmt"

	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/verana-labs/verana/x/de/types"
)

// GrantOperatorAuthorization implements [MOD-DE-MSG-3].
func (ms msgServer) GrantOperatorAuthorization(goCtx context.Context, msg *types.MsgGrantOperatorAuthorization) (*types.MsgGrantOperatorAuthorizationResponse, error) {
	ctx := sdk.UnwrapSDKContext(goCtx)
	now := ctx.BlockTime()

	// [MOD-DE-MSG-3-2] Basic checks (stateful)

	// TODO: [AUTHZ-CHECK] for (authority, operator) pair - skipped for now

	// Expiration must be in the future if specified
	if msg.Expiration != nil && !msg.Expiration.After(now) {
		return nil, types.ErrExpirationInPast
	}

	// Check mutual exclusivity: VSOperatorAuthorization must NOT exist for
	// this authority/grantee pair.
	vsKey := CompositeKey(msg.Authority, msg.Grantee)
	hasVSOA, err := ms.VSOperatorAuthorizations.Has(ctx, vsKey)
	if err != nil {
		return nil, fmt.Errorf("failed to check VSOperatorAuthorization: %w", err)
	}
	if hasVSOA {
		return nil, types.ErrVSOperatorAuthzExists
	}

	// [MOD-DE-MSG-3-4] Execution

	// 1. Create or update OperatorAuthorization
	oaKey := CompositeKey(msg.Authority, msg.Grantee)
	oa := types.OperatorAuthorization{
		Authority:  msg.Authority,
		Operator:   msg.Grantee,
		MsgTypes:   msg.MsgTypes,
		SpendLimit: msg.AuthzSpendLimit,
		Expiration: msg.Expiration,
		Period:     msg.AuthzSpendLimitPeriod,
	}
	if err := ms.OperatorAuthorizations.Set(ctx, oaKey, oa); err != nil {
		return nil, fmt.Errorf("failed to set OperatorAuthorization: %w", err)
	}

	// 2. Handle fee grant
	if !msg.WithFeegrant {
		// Revoke any existing fee grant
		if err := ms.RevokeFeeAllowance(ctx, msg.Authority, msg.Grantee); err != nil {
			return nil, fmt.Errorf("failed to revoke fee allowance: %w", err)
		}
	} else {
		// Grant fee allowance
		if err := ms.GrantFeeAllowance(
			ctx,
			msg.Authority,
			msg.Grantee,
			msg.MsgTypes,
			msg.Expiration,
			msg.FeegrantSpendLimit,
			msg.FeegrantSpendLimitPeriod,
		); err != nil {
			return nil, fmt.Errorf("failed to grant fee allowance: %w", err)
		}
	}

	// 3. Emit events
	ctx.EventManager().EmitEvents(sdk.Events{
		sdk.NewEvent(
			types.EventTypeGrantOperatorAuthorization,
			sdk.NewAttribute(types.AttributeKeyAuthority, msg.Authority),
			sdk.NewAttribute(types.AttributeKeyGrantee, msg.Grantee),
			sdk.NewAttribute(types.AttributeKeyWithFeegrant, fmt.Sprintf("%t", msg.WithFeegrant)),
			sdk.NewAttribute(types.AttributeKeyTimestamp, now.String()),
		),
	})

	return &types.MsgGrantOperatorAuthorizationResponse{}, nil
}
