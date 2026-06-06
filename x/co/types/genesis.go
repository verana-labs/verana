package types

import (
	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"

	"github.com/verana-labs/verana/util/validation"
)

func DefaultGenesis() *GenesisState {
	return &GenesisState{
		Params:       DefaultParams(),
		Corporations: nil,
	}
}

// Validate performs basic genesis state validation. Enforces invariants the
// runtime keeper relies on: unique id, unique policy_address, unique did,
// non-zero & monotone timestamps, and valid language tag.
func (gs GenesisState) Validate() error {
	if err := gs.Params.Validate(); err != nil {
		return err
	}
	ids := map[uint64]struct{}{}
	addrs := map[string]struct{}{}
	dids := map[string]struct{}{}
	for _, co := range gs.Corporations {
		if co.Id == 0 {
			return ErrCorporationNotFound.Wrap("corporation id must be > 0")
		}
		if _, dup := ids[co.Id]; dup {
			return ErrCorporationNotFound.Wrapf("duplicate corporation id %d", co.Id)
		}
		ids[co.Id] = struct{}{}

		if co.PolicyAddress == "" {
			return sdkerrors.ErrInvalidAddress.Wrap("policy_address is required")
		}
		if _, dup := addrs[co.PolicyAddress]; dup {
			return ErrPolicyAddressAlreadyBound.Wrap(co.PolicyAddress)
		}
		addrs[co.PolicyAddress] = struct{}{}

		if co.Did == "" || !validation.IsValidDID(co.Did) {
			return ErrInvalidDID.Wrap(co.Did)
		}
		if _, dup := dids[co.Did]; dup {
			return ErrDIDAlreadyExists.Wrap(co.Did)
		}
		dids[co.Did] = struct{}{}

		if !IsValidBCP47(co.Language) {
			return ErrInvalidLanguage.Wrap(co.Language)
		}

		if co.Created.IsZero() {
			return ErrInvalidTimestamp.Wrapf("corporation %d: created is required", co.Id)
		}
		if co.Modified.IsZero() {
			return ErrInvalidTimestamp.Wrapf("corporation %d: modified is required", co.Id)
		}
		if co.Modified.Before(co.Created) {
			return ErrInvalidTimestamp.Wrapf("corporation %d: modified (%s) is before created (%s)", co.Id, co.Modified, co.Created)
		}
	}
	return nil
}
