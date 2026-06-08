package types

// DONTCOVER

import (
	sdkerrors "cosmossdk.io/errors"
)

// x/pp module sentinel errors
var (
	ErrInvalidSigner = sdkerrors.Register(ModuleName, 1100, "expected gov account as only signer for proposal message")
	ErrSample        = sdkerrors.Register(ModuleName, 1101, "sample error")
	// ErrDIDOwnershipConflict is returned when a create would introduce a
	// Participant whose did is already controlled by a different corporation,
	// violating the per-Participant (did, corporation_id) consistency invariant
	// (spec MOD-PP-MSG-1-2-1 / 7-2-1 / 14-2-1).
	ErrDIDOwnershipConflict = sdkerrors.Register(ModuleName, 1102, "did is controlled by a different corporation")
)
