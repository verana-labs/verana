package keeper

import (
	"context"
	"fmt"

	"cosmossdk.io/collections"
	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/verana-labs/verana/x/de/types"
)

// RevokeOperatorAuthorization implements [MOD-DE-MSG-4].
func (ms msgServer) RevokeOperatorAuthorization(goCtx context.Context, msg *types.MsgRevokeOperatorAuthorization) (*types.MsgRevokeOperatorAuthorizationResponse, error) {
	ctx := sdk.UnwrapSDKContext(goCtx)
	now := ctx.BlockTime()

	// [MOD-DE-MSG-4-2] Basic checks (stateful)

	// [AUTHZ-CHECK-1] Verify operator authorization for this (authority, operator) pair
	if err := ms.CheckOperatorAuthorization(
		ctx,
		msg.Authority,
		msg.Operator,
		"/verana.de.v1.MsgRevokeOperatorAuthorization",
		now,
	); err != nil {
		return nil, err
	}

	// An Authorization entry MUST exist for this (authority, grantee)
	oaKey := collections.Join(msg.Authority, msg.Grantee)
	hasOA, err := ms.OperatorAuthorizations.Has(ctx, oaKey)
	if err != nil {
		return nil, fmt.Errorf("failed to check OperatorAuthorization: %w", err)
	}
	if !hasOA {
		return nil, types.ErrOperatorAuthzNotFound
	}

	// [MOD-DE-MSG-4-4] Execution

	// 1. Delete Authorization entry for this (authority, grantee)
	if err := ms.OperatorAuthorizations.Remove(ctx, oaKey); err != nil {
		return nil, fmt.Errorf("failed to remove OperatorAuthorization: %w", err)
	}

	// 2. Revoke Fee Allowance (authority, grantee)
	if err := ms.RevokeFeeAllowance(ctx, msg.Authority, msg.Grantee); err != nil {
		return nil, fmt.Errorf("failed to revoke fee allowance: %w", err)
	}

	// 3. Emit events
	ctx.EventManager().EmitEvent(
		sdk.NewEvent(
			types.EventTypeRevokeOperatorAuthorization,
			sdk.NewAttribute(types.AttributeKeyAuthority, msg.Authority),
			sdk.NewAttribute(types.AttributeKeyGrantee, msg.Grantee),
			sdk.NewAttribute(types.AttributeKeyTimestamp, now.String()),
		),
	)

	return &types.MsgRevokeOperatorAuthorizationResponse{}, nil
}
