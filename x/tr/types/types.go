package types

import (
	"fmt"
	"net/url"
	"regexp"

	sdk "github.com/cosmos/cosmos-sdk/types"
	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"
)

// ValidateBasic performs stateless validation of MsgCreateTrustRegistry
// [MOD-TR-MSG-1-2-1] Create New Trust Registry basic checks
func (msg *MsgCreateTrustRegistry) ValidateBasic() error {
	if msg.Did == "" || msg.Language == "" || msg.DocUrl == "" || msg.DocDigestSri == "" {
		return fmt.Errorf("missing mandatory parameter")
	}

	if _, err := sdk.AccAddressFromBech32(msg.Corporation); err != nil {
		return sdkerrors.ErrInvalidAddress.Wrapf("invalid corporation address: %s", err)
	}

	if _, err := sdk.AccAddressFromBech32(msg.Operator); err != nil {
		return sdkerrors.ErrInvalidAddress.Wrapf("invalid operator address: %s", err)
	}

	if !isValidDID(msg.Did) {
		return fmt.Errorf("invalid DID syntax")
	}

	if msg.Aka != "" && !isValidURI(msg.Aka) {
		return fmt.Errorf("invalid AKA URI")
	}

	if !isValidBCP47(msg.Language) {
		return fmt.Errorf("invalid language tag (must be a valid BCP 47 tag)")
	}

	// [MOD-TR-MSG-1-2-1] doc_url must be a valid http/https URL.
	u, err := url.ParseRequestURI(msg.DocUrl)
	if err != nil || (u.Scheme != "http" && u.Scheme != "https") {
		return fmt.Errorf("doc_url must be a valid http/https URL")
	}

	// [MOD-TR-MSG-1-2-1] doc_digest_sri must be a valid SRI string.
	if !isValidDigestSRI(msg.DocDigestSri) {
		return fmt.Errorf("invalid doc_digest_sri")
	}

	return nil
}

// ValidateBasic performs stateless validation of MsgAddGovernanceFrameworkDocument
// [MOD-TR-MSG-2-2-1] Add Governance Framework Document basic checks
func (msg *MsgAddGovernanceFrameworkDocument) ValidateBasic() error {
	if msg.TrId == 0 || msg.Language == "" || msg.Url == "" || msg.DigestSri == "" || msg.Version == 0 {
		return fmt.Errorf("missing mandatory parameter")
	}

	if _, err := sdk.AccAddressFromBech32(msg.Corporation); err != nil {
		return sdkerrors.ErrInvalidAddress.Wrapf("invalid corporation address: %s", err)
	}

	if _, err := sdk.AccAddressFromBech32(msg.Operator); err != nil {
		return sdkerrors.ErrInvalidAddress.Wrapf("invalid operator address: %s", err)
	}

	if !isValidBCP47(msg.Language) {
		return fmt.Errorf("invalid language tag (must be a valid BCP 47 tag)")
	}

	u, err := url.ParseRequestURI(msg.Url)
	if err != nil || (u.Scheme != "http" && u.Scheme != "https") {
		return fmt.Errorf("url must be a valid http/https URL")
	}

	if !isValidDigestSRI(msg.DigestSri) {
		return fmt.Errorf("invalid document digest sri")
	}

	return nil
}

// ValidateBasic performs stateless validation of MsgIncreaseActiveGovernanceFrameworkVersion
// [MOD-TR-MSG-3-2-1] Increase Active Governance Framework Version basic checks
func (msg *MsgIncreaseActiveGovernanceFrameworkVersion) ValidateBasic() error {
	if msg.Corporation == "" {
		return fmt.Errorf("corporation address is required")
	}
	if _, err := sdk.AccAddressFromBech32(msg.Corporation); err != nil {
		return sdkerrors.ErrInvalidAddress.Wrapf("invalid corporation address: %s", err)
	}

	if _, err := sdk.AccAddressFromBech32(msg.Operator); err != nil {
		return sdkerrors.ErrInvalidAddress.Wrapf("invalid operator address: %s", err)
	}

	if msg.TrId == 0 {
		return fmt.Errorf("trust registry id is required")
	}

	return nil
}

// ValidateBasic performs stateless validation of MsgUpdateTrustRegistry
// [MOD-TR-MSG-4-2-1] Update Trust Registry basic checks
func (msg *MsgUpdateTrustRegistry) ValidateBasic() error {
	if msg.Corporation == "" {
		return fmt.Errorf("corporation address is required")
	}
	if _, err := sdk.AccAddressFromBech32(msg.Corporation); err != nil {
		return sdkerrors.ErrInvalidAddress.Wrapf("invalid corporation address: %s", err)
	}

	if msg.Operator == "" {
		return fmt.Errorf("operator address is required")
	}
	if _, err := sdk.AccAddressFromBech32(msg.Operator); err != nil {
		return sdkerrors.ErrInvalidAddress.Wrapf("invalid operator address: %s", err)
	}

	if msg.TrId == 0 {
		return fmt.Errorf("trust registry id is required")
	}

	// [MOD-TR-MSG-4-1] did is mandatory per spec draft 13.
	if msg.Did == "" {
		return fmt.Errorf("did is required")
	}
	if !isValidDID(msg.Did) {
		return fmt.Errorf("invalid DID format")
	}

	if msg.Aka != "" {
		if !isValidURI(msg.Aka) {
			return fmt.Errorf("aka must be a valid URI")
		}
	}

	return nil
}

// ValidateBasic performs stateless validation of MsgArchiveTrustRegistry
// [MOD-TR-MSG-5-2-1] Archive Trust Registry basic checks
func (msg *MsgArchiveTrustRegistry) ValidateBasic() error {
	if msg.Corporation == "" {
		return fmt.Errorf("corporation address is required")
	}
	if _, err := sdk.AccAddressFromBech32(msg.Corporation); err != nil {
		return sdkerrors.ErrInvalidAddress.Wrapf("invalid corporation address: %s", err)
	}

	if _, err := sdk.AccAddressFromBech32(msg.Operator); err != nil {
		return sdkerrors.ErrInvalidAddress.Wrapf("invalid operator address: %s", err)
	}

	if msg.TrId == 0 {
		return fmt.Errorf("trust registry id is required")
	}

	return nil
}

func isValidDID(did string) bool {
	// Simplified but correct DID syntax: did:method:method-specific-id
	re := regexp.MustCompile(`^did:[a-zA-Z0-9-]+:[a-zA-Z0-9._:/-]+(#[^\s]*)?(\?[^\s]*)?$`)
	return re.MatchString(did)
}

func isValidURI(uri string) bool {
	_, err := url.ParseRequestURI(uri)
	return err == nil
}

func isValidDigestSRI(digestSRI string) bool {
	sriRegex := regexp.MustCompile(`^(sha256|sha384|sha512)-[A-Za-z0-9+/]+[=]{0,2}$`)
	return sriRegex.MatchString(digestSRI)
}
