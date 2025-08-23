package types

import (
	"fmt"
	"regexp"
	"time"

	sdk "github.com/cosmos/cosmos-sdk/types"
	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"
	"github.com/google/uuid"
)

func (msg *MsgStartPermissionVP) ValidateBasic() error {
	if _, err := sdk.AccAddressFromBech32(msg.Creator); err != nil {
		return fmt.Errorf("invalid creator address: %w", err)
	}

	if msg.ValidatorPermId == 0 {
		return fmt.Errorf("validator perm ID cannot be 0")
	}

	if msg.Type == 0 || msg.Type > 6 {
		return fmt.Errorf("perm type must be between 1 and 6")
	}

	if msg.Country == "" {
		return fmt.Errorf("country must be specified")
	}

	if !isValidCountryCode(msg.Country) {
		return fmt.Errorf("invalid country code format")
	}

	if msg.Did != "" && !isValidDID(msg.Did) {
		return fmt.Errorf("invalid DID format")
	}

	return nil
}

func isValidCountryCode(code string) bool {
	// Basic check for ISO 3166-1 alpha-2 format
	match, _ := regexp.MatchString(`^[A-Z]{2}$`, code)
	return match
}

func isValidDID(did string) bool {
	// Basic DID validation regex
	match, _ := regexp.MatchString(`^did:[a-zA-Z0-9]+:[a-zA-Z0-9._-]+$`, did)
	return match
}

func (msg *MsgRenewPermissionVP) ValidateBasic() error {
	// Validate creator address
	if _, err := sdk.AccAddressFromBech32(msg.Creator); err != nil {
		return fmt.Errorf("invalid creator address: %w", err)
	}

	// Validate perm ID
	if msg.Id == 0 {
		return fmt.Errorf("perm ID cannot be 0")
	}

	return nil
}

// ValidateBasic for MsgSetPermissionVPToValidated
func (msg *MsgSetPermissionVPToValidated) ValidateBasic() error {
	// Validate creator address
	if _, err := sdk.AccAddressFromBech32(msg.Creator); err != nil {
		return fmt.Errorf("invalid creator address: %w", err)
	}

	// Validate perm ID
	if msg.Id == 0 {
		return fmt.Errorf("perm ID cannot be 0")
	}

	// Validate fees are non-negative
	if msg.ValidationFees < 0 {
		return fmt.Errorf("validation fees cannot be negative")
	}
	if msg.IssuanceFees < 0 {
		return fmt.Errorf("issuance fees cannot be negative")
	}
	if msg.VerificationFees < 0 {
		return fmt.Errorf("verification fees cannot be negative")
	}

	// Validate country code if provided
	if msg.Country != "" && !isValidCountryCode(msg.Country) {
		return fmt.Errorf("invalid country code format")
	}

	// Validate digest SRI format if provided (optional)
	if msg.VpSummaryDigestSri != "" && !isValidDigestSRI(msg.VpSummaryDigestSri) {
		return fmt.Errorf("invalid vp_summary_digest_sri format")
	}

	// NOTE: Do NOT validate effective_until against current time in stateless validation
	// This will be done in stateful validation where we have access to block time

	return nil
}

// Add this helper function for digest SRI validation
func isValidDigestSRI(digestSRI string) bool {
	// Validate digest SRI format: algorithm-hash
	// Example: sha384-MzNNbQTWCSUSi0bbz7dbua+RcENv7C6FvlmYJ1Y+I727HsPOHdzwELMYO9Mz68M26
	if digestSRI == "" {
		return true // Empty is valid (optional)
	}

	// Simple regex for digest SRI format validation
	matched, _ := regexp.MatchString(`^[a-z0-9]+-[A-Za-z0-9+/]+=*$`, digestSRI)
	return matched
}

// ValidateBasic for MsgRequestPermissionVPTermination
func (msg *MsgRequestPermissionVPTermination) ValidateBasic() error {
	// Validate creator address
	if _, err := sdk.AccAddressFromBech32(msg.Creator); err != nil {
		return fmt.Errorf("invalid creator address: %w", err)
	}

	// Validate perm ID
	if msg.Id == 0 {
		return fmt.Errorf("perm ID cannot be 0")
	}

	return nil
}

// ValidateBasic for MsgConfirmPermissionVPTermination
func (msg *MsgConfirmPermissionVPTermination) ValidateBasic() error {
	// Validate creator address
	if _, err := sdk.AccAddressFromBech32(msg.Creator); err != nil {
		return fmt.Errorf("invalid creator address: %w", err)
	}

	// Validate perm ID
	if msg.Id == 0 {
		return fmt.Errorf("perm ID cannot be 0")
	}

	return nil
}

// ValidateBasic for MsgConfirmPermissionVPTermination
func (msg *MsgCancelPermissionVPLastRequest) ValidateBasic() error {
	// Validate creator address
	if _, err := sdk.AccAddressFromBech32(msg.Creator); err != nil {
		return fmt.Errorf("invalid creator address: %w", err)
	}

	// Validate perm ID
	if msg.Id == 0 {
		return fmt.Errorf("perm ID cannot be 0")
	}

	return nil
}

func (msg *MsgCreateRootPermission) ValidateBasic() error {
	// Validate creator address
	if _, err := sdk.AccAddressFromBech32(msg.Creator); err != nil {
		return fmt.Errorf("invalid creator address: %w", err)
	}

	// if a mandatory parameter is not present, transaction MUST abort
	if msg.SchemaId == 0 {
		return fmt.Errorf("schema ID cannot be 0")
	}

	if msg.Did == "" {
		return fmt.Errorf("DID is required")
	}

	// did, if specified, MUST conform to the DID Syntax
	if !isValidDID(msg.Did) {
		return fmt.Errorf("invalid DID format")
	}

	// validation_fees MUST be >= 0
	if msg.ValidationFees < 0 {
		return fmt.Errorf("validation fees cannot be negative")
	}

	// issuance_fees MUST be >= 0
	if msg.IssuanceFees < 0 {
		return fmt.Errorf("issuance fees cannot be negative")
	}

	// verification_fees MUST be >= 0
	if msg.VerificationFees < 0 {
		return fmt.Errorf("verification fees cannot be negative")
	}

	// country if not null, MUST be a valid alpha-2 code (ISO 3166)
	if msg.Country != "" && !isValidCountryCode(msg.Country) {
		return fmt.Errorf("invalid country code format")
	}

	// Note: Time-based validations are moved to the main function
	// to use blockchain time instead of system time

	return nil
}

func (msg *MsgExtendPermission) ValidateBasic() error {
	// Validate creator address
	if _, err := sdk.AccAddressFromBech32(msg.Creator); err != nil {
		return fmt.Errorf("invalid creator address: %w", err)
	}

	// Validate perm ID
	if msg.Id == 0 {
		return fmt.Errorf("perm ID cannot be 0")
	}

	// Validate effective_until is in the future
	if msg.EffectiveUntil != nil && !msg.EffectiveUntil.After(time.Now()) {
		return fmt.Errorf("effective_until must be in the future")
	}

	return nil
}

func (msg *MsgRevokePermission) ValidateBasic() error {
	// Validate creator address
	if _, err := sdk.AccAddressFromBech32(msg.Creator); err != nil {
		return fmt.Errorf("invalid creator address: %w", err)
	}

	// Validate perm ID
	if msg.Id == 0 {
		return fmt.Errorf("perm ID cannot be 0")
	}

	return nil
}

func (msg *MsgCreateOrUpdatePermissionSession) ValidateBasic() error {
	if _, err := sdk.AccAddressFromBech32(msg.Creator); err != nil {
		return sdkerrors.ErrInvalidAddress.Wrapf("invalid creator address: %s", err)
	}

	// Validate UUID format
	if _, err := uuid.Parse(msg.Id); err != nil {
		return sdkerrors.ErrInvalidRequest.Wrap("invalid session ID: must be valid UUID")
	}

	// At least one of issuer or verifier must be provided
	if msg.IssuerPermId == 0 && msg.VerifierPermId == 0 {
		return sdkerrors.ErrInvalidRequest.Wrap("at least one of issuer_perm_id or verifier_perm_id must be provided")
	}

	// Agent perm ID is required
	if msg.AgentPermId == 0 {
		return sdkerrors.ErrInvalidRequest.Wrap("agent perm ID required")
	}

	return nil
}

func (msg *MsgSlashPermissionTrustDeposit) ValidateBasic() error {
	if _, err := sdk.AccAddressFromBech32(msg.Creator); err != nil {
		return sdkerrors.ErrInvalidAddress.Wrapf("invalid creator address: %s", err)
	}

	// [MOD-PERM-MSG-12-2-1] Slash Permission Trust Deposit basic checks
	if msg.Id == 0 {
		return sdkerrors.ErrInvalidRequest.Wrap("id must be a valid uint64")
	}

	if msg.Amount == 0 {
		return sdkerrors.ErrInvalidRequest.Wrap("amount must be greater than 0")
	}
	return nil
}

func (msg *MsgRepayPermissionSlashedTrustDeposit) ValidateBasic() error {
	if _, err := sdk.AccAddressFromBech32(msg.Creator); err != nil {
		return sdkerrors.ErrInvalidAddress.Wrapf("invalid creator address: %s", err)
	}
	// [MOD-PERM-MSG-13-2-1] Repay Permission Slashed Trust Deposit basic checks
	if msg.Id == 0 {
		return sdkerrors.ErrInvalidRequest.Wrap("id must be a valid uint64")
	}
	return nil
}

func (msg *MsgCreatePermission) ValidateBasic() error {
	if _, err := sdk.AccAddressFromBech32(msg.Creator); err != nil {
		return sdkerrors.ErrInvalidAddress.Wrapf("invalid creator address: %s", err)
	}
	// [MOD-PERM-MSG-14-2-1] Create Permission basic checks
	if msg.SchemaId == 0 {
		return sdkerrors.ErrInvalidRequest.Wrap("schema_id must be a valid uint64")
	}
	// type MUST be ISSUER or VERIFIER
	if msg.Type != PermissionType_ISSUER &&
		msg.Type != PermissionType_VERIFIER {
		return sdkerrors.ErrInvalidRequest.Wrap("type must be ISSUER or VERIFIER")
	}

	// did MUST conform to DID Syntax
	if msg.Did == "" {
		return sdkerrors.ErrInvalidRequest.Wrap("did is mandatory")
	}
	if !isValidDID(msg.Did) {
		return sdkerrors.ErrInvalidRequest.Wrap("invalid DID syntax")
	}
	return nil
}
