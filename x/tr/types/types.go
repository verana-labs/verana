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
	// Check mandatory parameters
	if msg.Did == "" || msg.Language == "" || msg.DocUrl == "" || msg.DocDigestSri == "" {
		return fmt.Errorf("missing mandatory parameter")
	}

	// Validate authority address (group account)
	if _, err := sdk.AccAddressFromBech32(msg.Authority); err != nil {
		return sdkerrors.ErrInvalidAddress.Wrapf("invalid authority address: %s", err)
	}

	// Validate operator address (signer authorized by authority)
	if _, err := sdk.AccAddressFromBech32(msg.Operator); err != nil {
		return sdkerrors.ErrInvalidAddress.Wrapf("invalid operator address: %s", err)
	}

	// DID syntax validation - must conform to DID-CORE spec
	if !isValidDID(msg.Did) {
		return fmt.Errorf("invalid DID syntax")
	}

	// Validate AKA URI if present - must be a valid URI
	if msg.Aka != "" && !isValidURI(msg.Aka) {
		return fmt.Errorf("invalid AKA URI")
	}

	// Validate language tag (RFC1766, max 17 chars)
	if !isValidLanguageTagForCreateTrustRegistry(msg.Language) {
		return fmt.Errorf("invalid language tag (must conform to RFC 1766 and be 2 characters long)")
	}

	// Validate URL
	if !isValidURL(msg.DocUrl) {
		return fmt.Errorf("invalid document URL")
	}

	// Validate document digest sri
	if !isValidDigestSRI(msg.DocDigestSri) {
		return fmt.Errorf("invalid document digest sri")
	}

	return nil
}

func isValidLanguageTagForCreateTrustRegistry(lang string) bool {
	// RFC1766 primary tag must be exactly 2 letters
	if len(lang) > 17 || len(lang) < 2 {
		return false
	}
	// Must be lowercase letters only
	match, _ := regexp.MatchString(`^[a-z]{2}$`, lang[:2]) // Check only the first two characters
	return match
}

func (msg *MsgAddGovernanceFrameworkDocument) ValidateBasic() error {
	if msg.Id == 0 || msg.DocLanguage == "" || msg.DocUrl == "" || msg.DocDigestSri == "" || msg.Version == 0 {
		return fmt.Errorf("missing mandatory parameter")
	}

	if _, err := sdk.AccAddressFromBech32(msg.Creator); err != nil {
		return sdkerrors.ErrInvalidAddress.Wrapf("invalid creator address: %s", err)
	}

	// Language tag validation (RFC1766)
	if !isValidLanguageTag(msg.DocLanguage) {
		return fmt.Errorf("invalid language tag (must conform to rfc1766)")
	}

	// Validate URL and hash
	if _, err := url.Parse(msg.DocUrl); err != nil {
		return fmt.Errorf("invalid document URL")
	}

	// Validate document digest sri
	if !isValidDigestSRI(msg.DocDigestSri) {
		return fmt.Errorf("invalid document digest sri")
	}

	return nil
}

func (msg *MsgIncreaseActiveGovernanceFrameworkVersion) ValidateBasic() error {
	if msg.Creator == "" {
		return fmt.Errorf("creator address is required")
	}

	if _, err := sdk.AccAddressFromBech32(msg.Creator); err != nil {
		return sdkerrors.ErrInvalidAddress.Wrapf("invalid creator address: %s", err)
	}

	if msg.Id == 0 {
		return fmt.Errorf("trust registry id is required")
	}

	return nil
}

func (msg *MsgUpdateTrustRegistry) ValidateBasic() error {
	if msg.Creator == "" {
		return fmt.Errorf("creator address is required")
	}

	if _, err := sdk.AccAddressFromBech32(msg.Creator); err != nil {
		return sdkerrors.ErrInvalidAddress.Wrapf("invalid creator address: %s", err)
	}

	if msg.Id == 0 {
		return fmt.Errorf("trust registry id is required")
	}

	if msg.Did == "" {
		return fmt.Errorf("did is required")
	}

	if !isValidDID(msg.Did) {
		return fmt.Errorf("invalid did")
	}

	return nil
}

func (msg *MsgArchiveTrustRegistry) ValidateBasic() error {
	if msg.Creator == "" {
		return fmt.Errorf("creator address is required")
	}

	if msg.Id == 0 {
		return fmt.Errorf("trust registry id is required")
	}

	if _, err := sdk.AccAddressFromBech32(msg.Creator); err != nil {
		return sdkerrors.ErrInvalidAddress.Wrapf("invalid creator address: %s", err)
	}

	return nil
}

// Helper functions
func isValidDID(did string) bool {
	// DID validation regex following W3C DID specification
	// Format: did:<method-name>:<method-specific-id>
	// Method-specific-id can contain alphanumeric, dots, underscores, hyphens, colons, and slashes
	didRegex := regexp.MustCompile(`^did:[a-zA-Z0-9]+:[a-zA-Z0-9._:/-]+$`)
	return didRegex.MatchString(did)
}

func isValidLanguageTag(lang string) bool {
	// RFC1766 primary tag must be exactly 2 letters
	if len(lang) != 2 {
		return false
	}
	// Must be lowercase letters only
	match, _ := regexp.MatchString(`^[a-z]{2}$`, lang)
	return match
}

func isValidURI(uri string) bool {
	_, err := url.ParseRequestURI(uri)
	return err == nil
}

func isValidURL(urlStr string) bool {
	_, err := url.ParseRequestURI(urlStr)
	return err == nil
}

func isValidDigestSRI(digestSRI string) bool {
	// sha256-[base64], sha384-[base64], or sha512-[base64]
	sriRegex := regexp.MustCompile(`^(sha256|sha384|sha512)-[A-Za-z0-9+/]+[=]{0,2}$`)
	return sriRegex.MatchString(digestSRI)
}
