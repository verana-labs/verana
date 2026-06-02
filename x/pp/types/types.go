package types

import (
	"fmt"
	"regexp"

	sdk "github.com/cosmos/cosmos-sdk/types"
	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"
	"github.com/google/uuid"
)

func (msg *MsgStartParticipantOP) ValidateBasic() error {
	// [MOD-PERM-MSG-1-2-1] authority (group): signature must be verified
	if _, err := sdk.AccAddressFromBech32(msg.Corporation); err != nil {
		return fmt.Errorf("invalid corporation address: %w", err)
	}

	// [MOD-PERM-MSG-1-2-1] operator (account): signature must be verified
	if _, err := sdk.AccAddressFromBech32(msg.Operator); err != nil {
		return fmt.Errorf("invalid operator address: %w", err)
	}

	if msg.ValidatorParticipantId == 0 {
		return fmt.Errorf("validator perm ID cannot be 0")
	}

	// [MOD-PERM-MSG-1-2-1] type MUST be a valid ParticipantRole:
	// ISSUER_GRANTOR, VERIFIER_GRANTOR, ISSUER, VERIFIER, HOLDER.
	// ECOSYSTEM (5) is explicitly excluded — root permissions are only
	// created via MsgCreateRootParticipant, never via StartParticipantOP.
	pt := ParticipantRole(msg.Role)
	switch pt {
	case ParticipantRole_ISSUER,
		ParticipantRole_VERIFIER,
		ParticipantRole_ISSUER_GRANTOR,
		ParticipantRole_VERIFIER_GRANTOR,
		ParticipantRole_HOLDER:
		// ok
	default:
		return fmt.Errorf("perm type must be one of ISSUER, VERIFIER, ISSUER_GRANTOR, VERIFIER_GRANTOR, HOLDER (got %s)", pt.String())
	}

	// [MOD-PERM-MSG-1-1] did is required and MUST conform to DID Syntax
	if msg.Did == "" {
		return fmt.Errorf("did is required")
	}
	if !isValidDID(msg.Did) {
		return fmt.Errorf("invalid DID format")
	}

	// [MOD-PERM-MSG-1-2-1] vs_operator_authz_enabled: if true, vs_operator MUST NOT be null
	if msg.VsOperatorAuthzEnabled && msg.VsOperator == "" {
		return fmt.Errorf("vs_operator is required when vs_operator_authz_enabled is true")
	}

	// [MOD-PERM-MSG-1-2-1] vs_operator_authz_with_feegrant: if true, vs_operator MUST NOT be null
	if msg.VsOperatorAuthzWithFeegrant && msg.VsOperator == "" {
		return fmt.Errorf("vs_operator is required when vs_operator_authz_with_feegrant is true")
	}

	// [MOD-PERM-MSG-1-2-1] vs_operator_authz_spend_period: if not null, vs_operator MUST NOT be null
	if msg.VsOperatorAuthzSpendPeriod != nil && msg.VsOperator == "" {
		return fmt.Errorf("vs_operator is required when vs_operator_authz_spend_period is set")
	}

	// Validate vs_operator address if provided
	if msg.VsOperator != "" {
		if _, err := sdk.AccAddressFromBech32(msg.VsOperator); err != nil {
			return fmt.Errorf("invalid vs_operator address: %w", err)
		}
	}

	return nil
}

func isValidCountryCode(code string) bool {
	// Basic check for ISO 3166-1 alpha-2 format
	match, _ := regexp.MatchString(`^[A-Z]{2}$`, code)
	return match
}

func isValidDID(did string) bool {
	// DID validation regex following W3C DID specification
	// Format: did:<method-name>:<method-specific-id>
	// Method-specific-id can contain alphanumeric, dots, underscores, hyphens, colons, and slashes
	match, _ := regexp.MatchString(`^did:[a-zA-Z0-9]+:[a-zA-Z0-9._:/-]+$`, did)
	return match
}

func (msg *MsgRenewParticipantOP) ValidateBasic() error {
	// [MOD-PERM-MSG-2-2-1] authority (group): signature must be verified
	if _, err := sdk.AccAddressFromBech32(msg.Corporation); err != nil {
		return fmt.Errorf("invalid corporation address: %w", err)
	}

	// [MOD-PERM-MSG-2-2-1] operator (account): signature must be verified
	if _, err := sdk.AccAddressFromBech32(msg.Operator); err != nil {
		return fmt.Errorf("invalid operator address: %w", err)
	}

	// Validate perm ID
	if msg.Id == 0 {
		return fmt.Errorf("perm ID cannot be 0")
	}

	return nil
}

// ValidateBasic for MsgSetParticipantOPToValidated
func (msg *MsgSetParticipantOPToValidated) ValidateBasic() error {
	// [MOD-PERM-MSG-3-2-1] authority (group): signature must be verified
	if _, err := sdk.AccAddressFromBech32(msg.Corporation); err != nil {
		return fmt.Errorf("invalid corporation address: %w", err)
	}

	// [MOD-PERM-MSG-3-2-1] operator (account): signature must be verified
	if _, err := sdk.AccAddressFromBech32(msg.Operator); err != nil {
		return fmt.Errorf("invalid operator address: %w", err)
	}

	// Validate perm ID
	if msg.Id == 0 {
		return fmt.Errorf("perm ID cannot be 0")
	}

	// Validate digest SRI format if provided (optional)
	if msg.OpSummaryDigest != "" && !isValidDigestSRI(msg.OpSummaryDigest) {
		return fmt.Errorf("invalid op_summary_digest format")
	}

	// Validate discount fields (scaled: 0 = 0.0, 10000 = 1.0, range 0-10000)
	const maxDiscount = 10000 // 10000 = 100% discount = 1.0
	if msg.IssuanceFeeDiscount > maxDiscount {
		return fmt.Errorf("issuance_fee_discount cannot exceed %d (100%% discount)", maxDiscount)
	}
	if msg.VerificationFeeDiscount > maxDiscount {
		return fmt.Errorf("verification_fee_discount cannot exceed %d (100%% discount)", maxDiscount)
	}

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

// ValidateBasic for MsgConfirmPermissionVPTermination
func (msg *MsgCancelParticipantOPLastRequest) ValidateBasic() error {
	// Validate authority address
	if _, err := sdk.AccAddressFromBech32(msg.Corporation); err != nil {
		return fmt.Errorf("invalid corporation address: %w", err)
	}

	// Validate operator address
	if _, err := sdk.AccAddressFromBech32(msg.Operator); err != nil {
		return fmt.Errorf("invalid operator address: %w", err)
	}

	// Validate perm ID
	if msg.Id == 0 {
		return fmt.Errorf("perm ID cannot be 0")
	}

	return nil
}

func (msg *MsgCreateRootParticipant) ValidateBasic() error {
	// Validate authority address
	if _, err := sdk.AccAddressFromBech32(msg.Corporation); err != nil {
		return fmt.Errorf("invalid corporation address: %w", err)
	}

	// Validate operator address
	if _, err := sdk.AccAddressFromBech32(msg.Operator); err != nil {
		return fmt.Errorf("invalid operator address: %w", err)
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

	// Note: Time-based validations are moved to the main function
	// to use blockchain time instead of system time

	return nil
}

func (msg *MsgSetParticipantEffectiveUntil) ValidateBasic() error {
	// Validate authority address
	if _, err := sdk.AccAddressFromBech32(msg.Corporation); err != nil {
		return fmt.Errorf("invalid corporation address: %w", err)
	}

	// Validate operator address
	if _, err := sdk.AccAddressFromBech32(msg.Operator); err != nil {
		return fmt.Errorf("invalid operator address: %w", err)
	}

	// if a mandatory parameter is not present, transaction MUST abort
	// id MUST be a valid uint64
	if msg.Id == 0 {
		return fmt.Errorf("permission ID cannot be 0")
	}

	// effective_until is mandatory according to spec
	if msg.EffectiveUntil == nil {
		return fmt.Errorf("effective_until is required")
	}

	return nil
}

func (msg *MsgRevokeParticipant) ValidateBasic() error {
	// Validate authority address
	if _, err := sdk.AccAddressFromBech32(msg.Corporation); err != nil {
		return fmt.Errorf("invalid corporation address: %w", err)
	}

	// Validate operator address
	if _, err := sdk.AccAddressFromBech32(msg.Operator); err != nil {
		return fmt.Errorf("invalid operator address: %w", err)
	}

	// Validate perm ID
	if msg.Id == 0 {
		return fmt.Errorf("perm ID cannot be 0")
	}

	return nil
}

func (msg *MsgCreateOrUpdateParticipantSession) ValidateBasic() error {
	// [MOD-PERM-MSG-10-2] authority (group): signature must be verified
	if _, err := sdk.AccAddressFromBech32(msg.Corporation); err != nil {
		return fmt.Errorf("invalid corporation address: %w", err)
	}

	// [MOD-PERM-MSG-10-2] operator (account): signature must be verified
	if _, err := sdk.AccAddressFromBech32(msg.Operator); err != nil {
		return fmt.Errorf("invalid operator address: %w", err)
	}

	// Validate UUID format
	if _, err := uuid.Parse(msg.Id); err != nil {
		return sdkerrors.ErrInvalidRequest.Wrap("invalid session ID: must be valid UUID")
	}

	// if issuer_perm_id is null AND verifier_perm_id is null, MUST abort
	if msg.IssuerParticipantId == 0 && msg.VerifierParticipantId == 0 {
		return sdkerrors.ErrInvalidRequest.Wrap("at least one of issuer_perm_id or verifier_perm_id must be provided")
	}

	// agent_perm_id is mandatory
	if msg.AgentParticipantId == 0 {
		return sdkerrors.ErrInvalidRequest.Wrap("agent_perm_id is mandatory")
	}

	// wallet_agent_perm_id is mandatory
	if msg.WalletAgentParticipantId == 0 {
		return sdkerrors.ErrInvalidRequest.Wrap("wallet_agent_perm_id is mandatory")
	}

	// Validate digest SRI format if provided
	if msg.Digest != "" && !isValidDigestSRI(msg.Digest) {
		return sdkerrors.ErrInvalidRequest.Wrap("invalid digest format")
	}

	return nil
}

func (msg *MsgSlashParticipantTrustDeposit) ValidateBasic() error {
	// [MOD-PERM-MSG-12-2-1] authority (group): signature must be verified
	if _, err := sdk.AccAddressFromBech32(msg.Corporation); err != nil {
		return fmt.Errorf("invalid corporation address: %w", err)
	}

	// [MOD-PERM-MSG-12-2-1] operator (account): signature must be verified
	if _, err := sdk.AccAddressFromBech32(msg.Operator); err != nil {
		return fmt.Errorf("invalid operator address: %w", err)
	}

	// if a mandatory parameter is not present, transaction MUST abort
	// id MUST be a valid uint64
	if msg.Id == 0 {
		return sdkerrors.ErrInvalidRequest.Wrap("id must be a valid uint64")
	}

	if msg.Amount == 0 {
		return sdkerrors.ErrInvalidRequest.Wrap("amount must be greater than 0")
	}
	// [MOD-PERM-MSG-12-1] reason is mandatory per spec v4 draft 13
	if msg.Reason == "" {
		return sdkerrors.ErrInvalidRequest.Wrap("reason is required")
	}
	return nil
}

func (msg *MsgRepayParticipantSlashedTrustDeposit) ValidateBasic() error {
	// [MOD-PERM-MSG-13-2-1] authority (group): signature must be verified
	if _, err := sdk.AccAddressFromBech32(msg.Corporation); err != nil {
		return fmt.Errorf("invalid corporation address: %w", err)
	}

	// [MOD-PERM-MSG-13-2-1] operator (account): signature must be verified
	if _, err := sdk.AccAddressFromBech32(msg.Operator); err != nil {
		return fmt.Errorf("invalid operator address: %w", err)
	}

	// [MOD-PERM-MSG-13-2-1] id MUST be a valid uint64
	if msg.Id == 0 {
		return sdkerrors.ErrInvalidRequest.Wrap("id must be a valid uint64")
	}

	if msg.Amount == 0 {
		return sdkerrors.ErrInvalidRequest.Wrap("amount must be positive")
	}

	return nil
}

func (msg *MsgSelfCreateParticipant) ValidateBasic() error {
	// [MOD-PERM-MSG-14-2-1] corporation (group): signature must be verified
	if _, err := sdk.AccAddressFromBech32(msg.Corporation); err != nil {
		return fmt.Errorf("invalid corporation address: %w", err)
	}

	// [MOD-PERM-MSG-14-2-1] operator (account): signature must be verified
	if _, err := sdk.AccAddressFromBech32(msg.Operator); err != nil {
		return fmt.Errorf("invalid operator address: %w", err)
	}

	// type (ParticipantRole) (mandatory): MUST be ISSUER or VERIFIER, else abort
	if msg.Role != ParticipantRole_ISSUER && msg.Role != ParticipantRole_VERIFIER {
		return sdkerrors.ErrInvalidRequest.Wrap("type must be ISSUER or VERIFIER")
	}

	// validator_participant_id (mandatory)
	if msg.ValidatorParticipantId == 0 {
		return sdkerrors.ErrInvalidRequest.Wrap("validator_participant_id is mandatory")
	}

	// did MUST conform to DID Syntax
	if msg.Did == "" {
		return sdkerrors.ErrInvalidRequest.Wrap("did is mandatory")
	}
	if !isValidDID(msg.Did) {
		return sdkerrors.ErrInvalidRequest.Wrap("invalid DID syntax")
	}

	// vs_operator_authz_enabled: if true, vs_operator MUST NOT be null
	if msg.VsOperatorAuthzEnabled && msg.VsOperator == "" {
		return fmt.Errorf("vs_operator is required when vs_operator_authz_enabled is true")
	}

	// vs_operator_authz_with_feegrant: if true, vs_operator MUST NOT be null
	if msg.VsOperatorAuthzWithFeegrant && msg.VsOperator == "" {
		return fmt.Errorf("vs_operator is required when vs_operator_authz_with_feegrant is true")
	}

	// vs_operator_authz_spend_period: if not null, vs_operator MUST NOT be null
	if msg.VsOperatorAuthzSpendPeriod != nil && msg.VsOperator == "" {
		return fmt.Errorf("vs_operator is required when vs_operator_authz_spend_period is set")
	}

	// Validate vs_operator address if provided
	if msg.VsOperator != "" {
		if _, err := sdk.AccAddressFromBech32(msg.VsOperator); err != nil {
			return fmt.Errorf("invalid vs_operator address: %w", err)
		}
	}

	return nil
}
