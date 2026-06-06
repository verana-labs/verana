package keeper

import (
	"context"
	"fmt"

	errorsmod "cosmossdk.io/errors"
	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/verana-labs/verana/x/di/types"
)

// StoreDigest implements [MOD-DI-MSG-1] Store Digest.
func (ms msgServer) StoreDigest(goCtx context.Context, msg *types.MsgStoreDigest) (*types.MsgStoreDigestResponse, error) {
	ctx := sdk.UnwrapSDKContext(goCtx)
	now := ctx.BlockTime()

	// [MOD-DI-MSG-1-2-1] Basic checks — ValidateBasic already covers address
	// and digest-not-empty validation.

	// [MOD-DI-MSG-1-2-1] [AUTHZ-CHECK] Verify operator authorization
	if ms.delegationKeeper == nil {
		return nil, types.ErrDelegationKeeperNil
	}
	if err := ms.delegationKeeper.CheckOperatorAuthorization(
		ctx,
		msg.Authority,
		msg.Operator,
		"/verana.di.v1.MsgStoreDigest",
		now,
	); err != nil {
		return nil, fmt.Errorf("authorization check failed: %w", err)
	}

	// [AUTHZ-CHECK-5] Signing authority account MUST be a registered Corporation.
	if _, err := ms.coKeeper.ResolveCorporationByPolicyAddress(ctx, msg.Authority); err != nil {
		return nil, err
	}

	// [MOD-DI-MSG-1-3] Execution — prevent duplicate (would overwrite created timestamp)
	if has, _ := ms.Keeper.Digests.Has(ctx, msg.Digest); has {
		return nil, errorsmod.Wrap(types.ErrDigestAlreadyExists, msg.Digest)
	}

	// [MOD-DI-MSG-1-3] Execution — Create Digest record
	digest := types.Digest{
		Digest:          msg.Digest,
		Created:         now,
		DigestAlgorithm: msg.DigestAlgorithm,
	}

	if err := ms.Digests.Set(ctx, msg.Digest, digest); err != nil {
		return nil, fmt.Errorf("failed to store digest: %w", err)
	}

	// Emit events
	ctx.EventManager().EmitEvent(
		sdk.NewEvent(
			types.EventTypeStoreDigest,
			sdk.NewAttribute(types.AttributeKeyAuthority, msg.Authority),
			sdk.NewAttribute(types.AttributeKeyOperator, msg.Operator),
			sdk.NewAttribute(types.AttributeKeyDigest, msg.Digest),
			sdk.NewAttribute(types.AttributeKeyTimestamp, now.String()),
		),
	)

	return &types.MsgStoreDigestResponse{}, nil
}

// StoreDigestModuleCall is the module-call entry point for [MOD-DI-MSG-1].
// It can be called directly by the perm module (CreateOrUpdatePermissionSession)
// with no signer/AUTHZ checks.
func (k Keeper) StoreDigestModuleCall(ctx context.Context, authority, digest, digestAlgorithm string) error {
	if _, err := sdk.AccAddressFromBech32(authority); err != nil {
		return fmt.Errorf("invalid authority address: %w", err)
	}

	sdkCtx := sdk.UnwrapSDKContext(ctx)
	now := sdkCtx.BlockTime()

	if digest == "" {
		return types.ErrDigestEmpty
	}

	// prevent duplicate (would overwrite created timestamp)
	if has, _ := k.Digests.Has(ctx, digest); has {
		return errorsmod.Wrap(types.ErrDigestAlreadyExists, digest)
	}

	d := types.Digest{
		Digest:          digest,
		Created:         now,
		DigestAlgorithm: digestAlgorithm,
	}

	if err := k.Digests.Set(ctx, digest, d); err != nil {
		return fmt.Errorf("failed to store digest: %w", err)
	}

	// Emit events
	sdkCtx.EventManager().EmitEvent(
		sdk.NewEvent(
			types.EventTypeStoreDigest,
			sdk.NewAttribute(types.AttributeKeyAuthority, authority),
			sdk.NewAttribute(types.AttributeKeyDigest, digest),
			sdk.NewAttribute(types.AttributeKeyTimestamp, now.String()),
		),
	)

	return nil
}
