package keeper

import (
	"time"

	sdk "github.com/cosmos/cosmos-sdk/types"
)

// computeVSOAFeegrantExpiration computes the feegrant expiration for a VSOA
// by iterating all permissions in the VSOA and finding the farthest effective_until.
// Returns nil if any permission has no expiration (meaning unlimited feegrant).
// Implements the feegrant computation logic from [MOD-DE-MSG-5-4] and [MOD-DE-MSG-6-4].
func (ms msgServer) computeVSOAFeegrantExpiration(ctx sdk.Context, authority, vsOperator string) (*time.Time, error) {
	permIDs, err := ms.delegationKeeper.GetVSOAPermissions(ctx, authority, vsOperator)
	if err != nil {
		return nil, err
	}

	var maxExpire *time.Time
	for _, permID := range permIDs {
		perm, err := ms.Keeper.Permission.Get(ctx, permID)
		if err != nil {
			continue // permission may have been deleted
		}
		if perm.VsOperatorAuthzWithFeegrant {
			if perm.EffectiveUntil == nil {
				return nil, nil // no expiration — unlimited feegrant
			}
			if maxExpire == nil || perm.EffectiveUntil.After(*maxExpire) {
				effectiveUntil := *perm.EffectiveUntil
				maxExpire = &effectiveUntil
			}
		}
	}

	return maxExpire, nil
}
