package types

import (
	"net/url"

	"cosmossdk.io/errors"
	sdk "github.com/cosmos/cosmos-sdk/types"
	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"
	"github.com/verana-labs/verana/util/validation"
)

// MsgCreateEcosystem.ValidateBasic implements MOD-ES-MSG-1-2-1 stateless
// preconditions: signer bech32, operator bech32, did syntax, BCP-47 language,
// http(s) doc_url, SRI-formatted doc_digest_sri.
func (m *MsgCreateEcosystem) ValidateBasic() error {
	if _, err := sdk.AccAddressFromBech32(m.Corporation); err != nil {
		return errors.Wrapf(sdkerrors.ErrInvalidAddress, "corporation: %s", err)
	}
	if _, err := sdk.AccAddressFromBech32(m.Operator); err != nil {
		return errors.Wrapf(sdkerrors.ErrInvalidAddress, "operator: %s", err)
	}
	if m.Did == "" {
		return errors.Wrap(ErrInvalidDID, "did is required")
	}
	if !validation.IsValidDID(m.Did) {
		return errors.Wrap(ErrInvalidDID, "did syntax invalid")
	}
	if m.Language == "" {
		return errors.Wrap(ErrInvalidLanguage, "language is required")
	}
	if !isValidBCP47(m.Language) {
		return errors.Wrap(ErrInvalidLanguage, "language must be BCP 47")
	}
	if m.DocUrl == "" {
		return errors.Wrap(ErrInvalidURL, "doc_url is required")
	}
	if !isValidHTTPURL(m.DocUrl) {
		return errors.Wrap(ErrInvalidURL, "doc_url must be http/https URL")
	}
	if m.DocDigestSri == "" {
		return errors.Wrap(ErrInvalidDigestSRI, "doc_digest_sri is required")
	}
	if !validation.IsValidDigestSRI(m.DocDigestSri) {
		return errors.Wrap(ErrInvalidDigestSRI, "doc_digest_sri must be a valid SRI")
	}
	return nil
}

// MsgUpdateEcosystem.ValidateBasic implements MOD-ES-MSG-2-2-1 stateless
// preconditions: signer bech32, operator bech32, id>0, did syntax.
func (m *MsgUpdateEcosystem) ValidateBasic() error {
	if _, err := sdk.AccAddressFromBech32(m.Corporation); err != nil {
		return errors.Wrapf(sdkerrors.ErrInvalidAddress, "corporation: %s", err)
	}
	if _, err := sdk.AccAddressFromBech32(m.Operator); err != nil {
		return errors.Wrapf(sdkerrors.ErrInvalidAddress, "operator: %s", err)
	}
	if m.Id == 0 {
		return errors.Wrap(ErrInvalidSubject, "id is required")
	}
	if m.Did == "" {
		return errors.Wrap(ErrInvalidDID, "did is required")
	}
	if !validation.IsValidDID(m.Did) {
		return errors.Wrap(ErrInvalidDID, "did syntax invalid")
	}
	return nil
}

// MsgArchiveEcosystem.ValidateBasic implements MOD-ES-MSG-3-2-1 stateless
// preconditions: signer bech32, operator bech32, id>0. `archive` is a proto3
// bool — its presence/absence cannot be distinguished on the wire, so the
// idempotency-abort branch in the keeper is what catches "false on
// un-archived" submissions.
func (m *MsgArchiveEcosystem) ValidateBasic() error {
	if _, err := sdk.AccAddressFromBech32(m.Corporation); err != nil {
		return errors.Wrapf(sdkerrors.ErrInvalidAddress, "corporation: %s", err)
	}
	if _, err := sdk.AccAddressFromBech32(m.Operator); err != nil {
		return errors.Wrapf(sdkerrors.ErrInvalidAddress, "operator: %s", err)
	}
	if m.Id == 0 {
		return errors.Wrap(ErrInvalidSubject, "id is required")
	}
	return nil
}

// --- shared validators -----------------------------------------------------

func isValidHTTPURL(s string) bool {
	u, err := url.ParseRequestURI(s)
	if err != nil {
		return false
	}
	return u.Scheme == "http" || u.Scheme == "https"
}
