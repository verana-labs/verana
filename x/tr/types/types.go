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
	if msg.Did == "" || msg.Language == "" {
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

	if !isValidLanguageTagForCreateTrustRegistry(msg.Language) {
		return fmt.Errorf("invalid language tag (must conform to RFC 1766 and be 2 characters long)")
	}

	return nil
}

func isValidLanguageTagForCreateTrustRegistry(lang string) bool {
	if len(lang) > 17 || len(lang) < 2 {
		return false
	}
	match, _ := regexp.MatchString(`^[a-z]{2}$`, lang[:2])
	return match
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

	if !isValidLanguageTag(msg.Language) {
		return fmt.Errorf("invalid language tag (must conform to rfc1766)")
	}

	if _, err := url.Parse(msg.Url); err != nil {
		return fmt.Errorf("invalid document URL")
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
	didRegex := regexp.MustCompile(`^did:[a-zA-Z0-9]+:[a-zA-Z0-9._:/-]+$`)
	return didRegex.MatchString(did)
}

func isValidLanguageTag(lang string) bool {
	if len(lang) != 2 {
		return false
	}
	match, _ := regexp.MatchString(`^[a-z]{2}$`, lang)
	return match
}

func isValidURI(uri string) bool {
	_, err := url.ParseRequestURI(uri)
	return err == nil
}

func isValidDigestSRI(digestSRI string) bool {
	sriRegex := regexp.MustCompile(`^(sha256|sha384|sha512)-[A-Za-z0-9+/]+[=]{0,2}$`)
	return sriRegex.MatchString(digestSRI)
}
