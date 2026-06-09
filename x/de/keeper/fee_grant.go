package keeper

import (
	"context"
	"fmt"
	"strconv"
	"time"

	"cosmossdk.io/collections"
	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/verana-labs/verana/x/de/types"
)

// GrantFeeAllowance implements [MOD-DE-MSG-1]. It creates or updates a FeeGrant
// record keyed by the composite (grantor_corporation_id, grantee). This is an
// internal method called by GrantOperatorAuthorization and the MSG-5-5
// recompute subroutine.
func (k Keeper) GrantFeeAllowance(
	goCtx context.Context,
	grantorCorporationID uint64,
	grantee string,
	msgTypes []string,
	expiration *time.Time,
	spendLimit sdk.Coins,
	period *time.Duration,
) error {
	ctx := sdk.UnwrapSDKContext(goCtx)
	now := ctx.BlockTime()

	// [MOD-DE-MSG-1-2] Basic checks.

	// msg_types MUST be a list of VPR delegable messages only.
	if len(msgTypes) == 0 {
		return fmt.Errorf("msg_types must not be empty")
	}
	for _, mt := range msgTypes {
		if !types.VPRDelegableMsgTypes[mt] {
			return fmt.Errorf("%w: %s", types.ErrInvalidMsgType, mt)
		}
	}

	// expiration if specified MUST be in the future.
	if expiration != nil && !expiration.After(now) {
		return types.ErrExpirationInPast
	}

	// spend_limit if specified MUST be valid.
	if len(spendLimit) > 0 && !spendLimit.IsValid() {
		return types.ErrInvalidSpendLimit
	}

	// period if specified MUST be valid (positive).
	if period != nil && *period <= 0 {
		return fmt.Errorf("period must be a positive duration")
	}

	// [MOD-DE-MSG-1-4] Execution.
	key := collections.Join(grantorCorporationID, grantee)
	feeGrant := types.FeeGrant{
		GrantorCorporationId: grantorCorporationID,
		Grantee:              grantee,
		MsgTypes:             msgTypes,
		SpendLimit:           spendLimit,
		Expiration:           expiration,
		Period:               period,
	}
	if err := k.FeeGrants.Set(ctx, key, feeGrant); err != nil {
		return fmt.Errorf("failed to set FeeGrant: %w", err)
	}

	ctx.EventManager().EmitEvent(
		sdk.NewEvent(
			types.EventTypeGrantFeeAllowance,
			sdk.NewAttribute(types.AttributeKeyCorporationID, strconv.FormatUint(grantorCorporationID, 10)),
			sdk.NewAttribute(types.AttributeKeyGrantee, grantee),
			sdk.NewAttribute(types.AttributeKeyTimestamp, now.String()),
		),
	)
	return nil
}

// RevokeFeeAllowance implements [MOD-DE-MSG-2]. It removes the FeeGrant for the
// composite (grantor_corporation_id, grantee). This is an internal method called
// by GrantOperatorAuthorization, RevokeOperatorAuthorization, and the MSG-5-5
// recompute subroutine.
func (k Keeper) RevokeFeeAllowance(goCtx context.Context, grantorCorporationID uint64, grantee string) error {
	ctx := sdk.UnwrapSDKContext(goCtx)

	// [MOD-DE-MSG-2-2] Basic checks.
	if grantorCorporationID == 0 {
		return fmt.Errorf("grantor_corporation_id must be specified")
	}
	if grantee == "" {
		return fmt.Errorf("grantee must be specified")
	}

	// [MOD-DE-MSG-2-4] Execution: if FeeGrant exists, delete it, else do nothing.
	key := collections.Join(grantorCorporationID, grantee)
	has, err := k.FeeGrants.Has(ctx, key)
	if err != nil {
		return err
	}
	if !has {
		return nil
	}
	if err := k.FeeGrants.Remove(ctx, key); err != nil {
		return err
	}

	ctx.EventManager().EmitEvent(
		sdk.NewEvent(
			types.EventTypeRevokeFeeAllowance,
			sdk.NewAttribute(types.AttributeKeyCorporationID, strconv.FormatUint(grantorCorporationID, 10)),
			sdk.NewAttribute(types.AttributeKeyGrantee, grantee),
			sdk.NewAttribute(types.AttributeKeyTimestamp, ctx.BlockTime().String()),
		),
	)
	return nil
}
