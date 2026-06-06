package types

import (
	"cosmossdk.io/errors"
	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"

	"github.com/verana-labs/verana/util/validation"
)

// ValidateBasic on MsgUpdateParams: authority must be present.
func (m *MsgUpdateParams) ValidateBasic() error {
	if m.Authority == "" {
		return errors.Wrap(sdkerrors.ErrInvalidAddress, "authority is required")
	}
	return m.Params.Validate()
}

// ValidateBasic on MsgAddGovernanceFrameworkDocument.
func (m *MsgAddGovernanceFrameworkDocument) ValidateBasic() error {
	if m.Corporation == "" {
		return errors.Wrap(sdkerrors.ErrInvalidAddress, "corporation is required")
	}
	if m.Operator == "" {
		return errors.Wrap(sdkerrors.ErrInvalidAddress, "operator is required")
	}
	if m.DocLanguage == "" {
		return errors.Wrap(ErrInvalidLanguage, "doc_language is required")
	}
	if !IsValidBCP47(m.DocLanguage) {
		return errors.Wrap(ErrInvalidLanguage, m.DocLanguage)
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
	if !validation.IsValidDigestSRI(m.DocDigestSri) {
		return errors.Wrap(ErrInvalidDigestSRI, m.DocDigestSri)
	}
	if m.Version < 1 {
		return errors.Wrap(ErrInvalidVersion, "version must be >= 1")
	}
	return nil
}

// ValidateBasic on MsgIncreaseActiveGovernanceFrameworkVersion.
func (m *MsgIncreaseActiveGovernanceFrameworkVersion) ValidateBasic() error {
	if m.Corporation == "" {
		return errors.Wrap(sdkerrors.ErrInvalidAddress, "corporation is required")
	}
	if m.Operator == "" {
		return errors.Wrap(sdkerrors.ErrInvalidAddress, "operator is required")
	}
	return nil
}
