package keeper

import (
	"context"
	"time"

	"cosmossdk.io/collections"
	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/verana-labs/verana/x/de/types"
)

// GrantFeeAllowance creates or updates a FeeGrant record for the given
// grantor/grantee pair. This is an internal method (MOD-DE-MSG-1).
func (k Keeper) GrantFeeAllowance(
	ctx context.Context,
	grantor string,
	grantee string,
	msgTypes []string,
	expiration *time.Time,
	spendLimit sdk.Coins,
	period *time.Duration,
) error {
	key := collections.Join(grantor, grantee)
	feeGrant := types.FeeGrant{
		Grantor:    grantor,
		Grantee:    grantee,
		MsgTypes:   msgTypes,
		SpendLimit: spendLimit,
		Expiration: expiration,
		Period:     period,
	}
	return k.FeeGrants.Set(ctx, key, feeGrant)
}

// RevokeFeeAllowance removes a FeeGrant record for the given grantor/grantee
// pair. This is an internal method (MOD-DE-MSG-2). It is a no-op if no grant
// exists.
func (k Keeper) RevokeFeeAllowance(ctx context.Context, grantor string, grantee string) error {
	key := collections.Join(grantor, grantee)
	has, err := k.FeeGrants.Has(ctx, key)
	if err != nil {
		return err
	}
	if !has {
		return nil
	}
	return k.FeeGrants.Remove(ctx, key)
}
