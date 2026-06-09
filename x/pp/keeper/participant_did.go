package keeper

import (
	"context"
	"fmt"

	errorsmod "cosmossdk.io/errors"

	"github.com/verana-labs/verana/x/pp/types"
)

// assertDIDCorporationConsistent enforces the per-Participant
// (did, corporation_id) consistency invariant required at create time by
// spec [MOD-PP-MSG-1-2-1], [MOD-PP-MSG-7-2-1] and [MOD-PP-MSG-14-2-1]:
// every existing Participant entry that shares the given did MUST belong to
// corporationID; otherwise the create MUST abort. This is NOT a DID-uniqueness
// check — the same did may be reused across multiple participants of the same
// corporation. x/pp has no (did) index, so the check walks the Participant map;
// creates are infrequent, so the linear scan is acceptable.
func (k Keeper) assertDIDCorporationConsistent(ctx context.Context, did string, corporationID uint64) error {
	if did == "" {
		return nil
	}
	var (
		conflict     bool
		conflictCorp uint64
	)
	err := k.Participant.Walk(ctx, nil, func(_ uint64, p types.Participant) (bool, error) {
		if p.Did == did && p.CorporationId != corporationID {
			conflict = true
			conflictCorp = p.CorporationId
			return true, nil // stop on first conflict
		}
		return false, nil
	})
	if err != nil {
		return fmt.Errorf("walk participants for did consistency: %w", err)
	}
	if conflict {
		return errorsmod.Wrapf(types.ErrDIDOwnershipConflict,
			"did %q is controlled by corporation %d, not %d", did, conflictCorp, corporationID)
	}
	return nil
}
