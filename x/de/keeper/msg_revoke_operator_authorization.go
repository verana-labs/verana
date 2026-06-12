package keeper

import (
	"context"
	"fmt"
	"strconv"

	"cosmossdk.io/collections"
	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/verana-labs/verana/x/de/types"
)

// RevokeOperatorAuthorization implements [MOD-DE-MSG-4].
func (ms msgServer) RevokeOperatorAuthorization(goCtx context.Context, msg *types.MsgRevokeOperatorAuthorization) (*types.MsgRevokeOperatorAuthorizationResponse, error) {
	ctx := sdk.UnwrapSDKContext(goCtx)
	now := ctx.BlockTime()

	// [MOD-DE-MSG-4-2] Basic checks (stateful).

	// [AUTHZ-CHECK-1] Verify operator authorization for this (corporation, operator) pair.
	if err := ms.CheckOperatorAuthorization(
		ctx,
		msg.Corporation,
		msg.Operator,
		"/verana.de.v1.MsgRevokeOperatorAuthorization",
		now,
	); err != nil {
		return nil, err
	}

	// [AUTHZ-CHECK-5] Signing corporation account MUST be a registered Corporation;
	// resolve it to co.id.
	co, err := ms.corporationKeeper().ResolveCorporationByPolicyAddress(ctx, msg.Corporation)
	if err != nil {
		return nil, err
	}

	// An OperatorAuthorization MUST exist for (co.id, grantee).
	existing, found, err := ms.getOperatorAuthorizationByCorpOp(ctx, co.Id, msg.Grantee)
	if err != nil {
		return nil, err
	}
	if !found {
		return nil, types.ErrOperatorAuthzNotFound
	}

	// [MOD-DE-MSG-4-4] Execution.

	// 1. Delete the OperatorAuthorization and its secondary index.
	if err := ms.OperatorAuthorizations.Remove(ctx, existing.Id); err != nil {
		return nil, fmt.Errorf("failed to remove OperatorAuthorization: %w", err)
	}
	if err := ms.OperatorAuthorizationByCorpOp.Remove(ctx, collections.Join(co.Id, msg.Grantee)); err != nil {
		return nil, fmt.Errorf("failed to remove OperatorAuthorization index: %w", err)
	}

	// 2. Revoke Fee Allowance for (co.id, grantee).
	if err := ms.RevokeFeeAllowance(ctx, co.Id, msg.Grantee); err != nil {
		return nil, fmt.Errorf("failed to revoke fee allowance: %w", err)
	}

	// 3. Emit events.
	ctx.EventManager().EmitEvent(
		sdk.NewEvent(
			types.EventTypeRevokeOperatorAuthorization,
			sdk.NewAttribute(types.AttributeKeyAuthzID, strconv.FormatUint(existing.Id, 10)),
			sdk.NewAttribute(types.AttributeKeyCorporationID, strconv.FormatUint(co.Id, 10)),
			sdk.NewAttribute(types.AttributeKeyGrantee, msg.Grantee),
			sdk.NewAttribute(types.AttributeKeyTimestamp, now.String()),
		),
	)

	return &types.MsgRevokeOperatorAuthorizationResponse{}, nil
}
