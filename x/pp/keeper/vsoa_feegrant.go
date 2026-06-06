package keeper

import (
	"time"

	sdk "github.com/cosmos/cosmos-sdk/types"
)

// computeVSOAFeegrantExpiration computes the feegrant expiration for a VSOA
// by iterating all participants in the VSOA and finding the farthest effective_until.
// Returns nil if any participant has no expiration (meaning unlimited feegrant).
// Implements the feegrant computation logic from [MOD-DE-MSG-5-4] and [MOD-DE-MSG-6-4].
func (ms msgServer) computeVSOAFeegrantExpiration(ctx sdk.Context, authority, vsOperator string) (*time.Time, error) {
	participantIDs, err := ms.delegationKeeper.GetVSOAPermissions(ctx, authority, vsOperator)
	if err != nil {
		return nil, err
	}

	var maxExpire *time.Time
	for _, participantID := range participantIDs {
		participant, err := ms.Keeper.Participant.Get(ctx, participantID)
		if err != nil {
			continue // participant may have been deleted
		}
		if participant.VsOperatorAuthzWithFeegrant {
			if participant.EffectiveUntil == nil {
				return nil, nil // no expiration — unlimited feegrant
			}
			if maxExpire == nil || participant.EffectiveUntil.After(*maxExpire) {
				effectiveUntil := *participant.EffectiveUntil
				maxExpire = &effectiveUntil
			}
		}
	}

	return maxExpire, nil
}
