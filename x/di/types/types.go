package types

import (
	"fmt"

	errorsmod "cosmossdk.io/errors"
	sdk "github.com/cosmos/cosmos-sdk/types"
	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"
	"github.com/verana-labs/verana/util/validation"
)

// allowedDigestAlgorithms is the set of accepted hash algorithm identifiers.
var allowedDigestAlgorithms = map[string]struct{}{
	"sha2-256": {},
	"sha2-512": {},
	"sha3-256": {},
}

// ValidateBasic performs stateless validation on MsgStoreDigest.
func (msg *MsgStoreDigest) ValidateBasic() error {
	// authority (corporation) is mandatory
	if _, err := sdk.AccAddressFromBech32(msg.Authority); err != nil {
		return fmt.Errorf("invalid authority address: %w", err)
	}

	// operator is mandatory
	if _, err := sdk.AccAddressFromBech32(msg.Operator); err != nil {
		return fmt.Errorf("invalid operator address: %w", err)
	}

	// digest must not be empty
	if msg.Digest == "" {
		return ErrDigestEmpty
	}

	// digest must not exceed maximum length
	if len(msg.Digest) > 256 {
		return errorsmod.Wrap(sdkerrors.ErrInvalidRequest, "digest exceeds maximum length of 256 bytes")
	}

	// [MOD-DI-MSG-1-1] digest must be a valid SRI string per spec v4 draft 13.
	if !validation.IsValidDigestSRI(msg.Digest) {
		return errorsmod.Wrap(sdkerrors.ErrInvalidRequest, "digest must be a valid SRI string (e.g. sha256-<base64>)")
	}

	// digest_algorithm is mandatory and must be a known algorithm
	if msg.DigestAlgorithm == "" {
		return errorsmod.Wrap(sdkerrors.ErrInvalidRequest, "digest_algorithm is required")
	}
	if _, ok := allowedDigestAlgorithms[msg.DigestAlgorithm]; !ok {
		return errorsmod.Wrap(sdkerrors.ErrInvalidRequest, "digest_algorithm must be one of: sha2-256, sha2-512, sha3-256")
	}

	return nil
}
