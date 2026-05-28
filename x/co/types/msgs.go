package types

import (
	"cosmossdk.io/errors"
	sdk "github.com/cosmos/cosmos-sdk/types"
	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"
)

// ValidateBasic on MsgUpdateParams: authority must be a valid bech32 address.
func (m *MsgUpdateParams) ValidateBasic() error {
	if m.Authority == "" {
		return errors.Wrap(sdkerrors.ErrInvalidAddress, "authority is required")
	}
	if _, err := sdk.AccAddressFromBech32(m.Authority); err != nil {
		return errors.Wrapf(sdkerrors.ErrInvalidAddress, "authority: %s", err)
	}
	return m.Params.Validate()
}

// ValidateBasic on MsgCreateCorporation — MOD-CO-MSG-1.
func (m *MsgCreateCorporation) ValidateBasic() error {
	if m.Signer == "" {
		return errors.Wrap(sdkerrors.ErrInvalidAddress, "signer is required")
	}
	if _, err := sdk.AccAddressFromBech32(m.Signer); err != nil {
		return errors.Wrapf(sdkerrors.ErrInvalidAddress, "signer: %s", err)
	}
	if len(m.Members) == 0 {
		return errors.Wrap(ErrInvalidMembers, "members must contain at least one entry")
	}
	for i, mem := range m.Members {
		if mem.Address == "" {
			return errors.Wrapf(ErrInvalidMembers, "members[%d].address is required", i)
		}
		if _, err := sdk.AccAddressFromBech32(mem.Address); err != nil {
			return errors.Wrapf(sdkerrors.ErrInvalidAddress, "members[%d].address: %s", i, err)
		}
		if mem.Weight == "" {
			return errors.Wrapf(ErrInvalidMembers, "members[%d].weight is required", i)
		}
	}
	if m.DecisionPolicy == nil {
		return errors.Wrap(ErrInvalidDecisionPolicy, "decision_policy is required")
	}
	if m.Did == "" {
		return errors.Wrap(ErrInvalidDID, "did is required")
	}
	if !IsValidDID(m.Did) {
		return errors.Wrap(ErrInvalidDID, m.Did)
	}
	if m.Language == "" {
		return errors.Wrap(ErrInvalidLanguage, "language is required")
	}
	if !IsValidBCP47(m.Language) {
		return errors.Wrap(ErrInvalidLanguage, m.Language)
	}
	if m.DocUrl == "" {
		return errors.Wrap(ErrInvalidURL, "doc_url is required")
	}
	if !IsValidURL(m.DocUrl) {
		return errors.Wrap(ErrInvalidURL, m.DocUrl)
	}
	if m.DocDigestSri == "" {
		return errors.Wrap(ErrInvalidDigestSRI, "doc_digest_sri is required")
	}
	if !IsValidDigestSRI(m.DocDigestSri) {
		return errors.Wrap(ErrInvalidDigestSRI, m.DocDigestSri)
	}
	return nil
}

// ValidateBasic on MsgUpdateCorporation — MOD-CO-MSG-2.
func (m *MsgUpdateCorporation) ValidateBasic() error {
	if m.Corporation == "" {
		return errors.Wrap(sdkerrors.ErrInvalidAddress, "corporation is required")
	}
	if _, err := sdk.AccAddressFromBech32(m.Corporation); err != nil {
		return errors.Wrapf(sdkerrors.ErrInvalidAddress, "corporation: %s", err)
	}
	if m.Operator == "" {
		return errors.Wrap(sdkerrors.ErrInvalidAddress, "operator is required")
	}
	if _, err := sdk.AccAddressFromBech32(m.Operator); err != nil {
		return errors.Wrapf(sdkerrors.ErrInvalidAddress, "operator: %s", err)
	}
	if m.Did == "" {
		return errors.Wrap(ErrInvalidDID, "did is required")
	}
	if !IsValidDID(m.Did) {
		return errors.Wrap(ErrInvalidDID, m.Did)
	}
	return nil
}
