package types

import (
	"fmt"
	"regexp"

	sdk "github.com/cosmos/cosmos-sdk/types"
	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"
	"github.com/google/uuid"
	"github.com/verana-labs/verana/util/validation"
)

// VSOA per-role permitted msg_types (spec v4-rc2 MOD-PP-MSG-1/7/14).
const (
	MsgSetParticipantOPToValidatedTypeURL      = "/verana.pp.v1.MsgSetParticipantOPToValidated"
	MsgCreateOrUpdateParticipantSessionTypeURL = "/verana.pp.v1.MsgCreateOrUpdateParticipantSession"
	// MsgTriggerResolverTypeURL is the HOLDER-permitted msg type. The resolver
	// trigger message is a spec placeholder not yet implemented in this codebase.
	MsgTriggerResolverTypeURL = "/verana.pp.v1.MsgTriggerResolver"
)

// vsoaPermittedMsgTypes returns the set of msg_types a vs_operator may be granted
// for the given participant role, or nil if the role cannot be granted VSOA.
func vsoaPermittedMsgTypes(role ParticipantRole) map[string]bool {
	switch role {
	case ParticipantRole_HOLDER:
		return map[string]bool{MsgTriggerResolverTypeURL: true}
	case ParticipantRole_ISSUER:
		return map[string]bool{
			MsgCreateOrUpdateParticipantSessionTypeURL: true,
			MsgSetParticipantOPToValidatedTypeURL:      true,
		}
	case ParticipantRole_VERIFIER:
		return map[string]bool{MsgCreateOrUpdateParticipantSessionTypeURL: true}
	case ParticipantRole_ISSUER_GRANTOR, ParticipantRole_VERIFIER_GRANTOR:
		return map[string]bool{MsgSetParticipantOPToValidatedTypeURL: true}
	default:
		return nil
	}
}

// validateVSOperatorAuthz validates the VSOA parameter block on a create message.
// If any vs_operator_authz_* parameter is set, vs_operator_authz_msg_types MUST be
// non-empty, vs_operator MUST be set, and every msg_type MUST be permitted for the
// role per the spec whitelist.
func validateVSOperatorAuthz(role ParticipantRole, vsOperator string, msgTypes []string, anyParamSet bool) error {
	if !anyParamSet {
		return nil
	}
	if len(msgTypes) == 0 {
		return fmt.Errorf("vs_operator_authz_msg_types is required when any vs_operator_authz_* param is set")
	}
	if vsOperator == "" {
		return fmt.Errorf("vs_operator is required when vs_operator_authz_* params are set")
	}
	allowed := vsoaPermittedMsgTypes(role)
	if allowed == nil {
		return fmt.Errorf("role %s cannot be granted vs_operator authorization", role.String())
	}
	for _, mt := range msgTypes {
		if !allowed[mt] {
			return fmt.Errorf("msg_type %s is not permitted for role %s", mt, role.String())
		}
	}
	return nil
}

func (msg *MsgStartParticipantOP) ValidateBasic() error {
	// [MOD-PP-MSG-1-2-1] authority (group): signature must be verified
	if _, err := sdk.AccAddressFromBech32(msg.Corporation); err != nil {
		return fmt.Errorf("invalid corporation address: %w", err)
	}

	// [MOD-PP-MSG-1-2-1] operator (account): signature must be verified
	if _, err := sdk.AccAddressFromBech32(msg.Operator); err != nil {
		return fmt.Errorf("invalid operator address: %w", err)
	}

	if msg.ValidatorParticipantId == 0 {
		return fmt.Errorf("validator participant ID cannot be 0")
	}

	// [MOD-PP-MSG-1-2-1] type MUST be a valid ParticipantRole:
	// ISSUER_GRANTOR, VERIFIER_GRANTOR, ISSUER, VERIFIER, HOLDER.
	// ECOSYSTEM (5) is explicitly excluded — root participants are only
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
		return fmt.Errorf("participant type must be one of ISSUER, VERIFIER, ISSUER_GRANTOR, VERIFIER_GRANTOR, HOLDER (got %s)", pt.String())
	}

	// [MOD-PP-MSG-1-1] did is required and MUST conform to DID Syntax
	if msg.Did == "" {
		return fmt.Errorf("did is required")
	}
	if !validation.IsValidDID(msg.Did) {
		return fmt.Errorf("invalid DID format")
	}

	// Validate vs_operator address if provided
	if msg.VsOperator != "" {
		if _, err := sdk.AccAddressFromBech32(msg.VsOperator); err != nil {
			return fmt.Errorf("invalid vs_operator address: %w", err)
		}
	}

	// [MOD-PP-MSG-1-2-1] VSOA params: per-role whitelist + presence rules.
	anyParam := len(msg.VsOperatorAuthzMsgTypes) > 0 ||
		len(msg.VsOperatorAuthzSpendLimit) > 0 ||
		msg.VsOperatorAuthzWithFeegrant ||
		len(msg.VsOperatorAuthzFeeSpendLimit) > 0 ||
		msg.VsOperatorAuthzPeriod != nil
	if err := validateVSOperatorAuthz(ParticipantRole(msg.Role), msg.VsOperator, msg.VsOperatorAuthzMsgTypes, anyParam); err != nil {
		return err
	}

	return nil
}

func isValidCountryCode(code string) bool {
	// Basic check for ISO 3166-1 alpha-2 format
	match, _ := regexp.MatchString(`^[A-Z]{2}$`, code)
	return match
}

func (msg *MsgRenewParticipantOP) ValidateBasic() error {
	// [MOD-PP-MSG-2-2-1] authority (group): signature must be verified
	if _, err := sdk.AccAddressFromBech32(msg.Corporation); err != nil {
		return fmt.Errorf("invalid corporation address: %w", err)
	}

	// [MOD-PP-MSG-2-2-1] operator (account): signature must be verified
	if _, err := sdk.AccAddressFromBech32(msg.Operator); err != nil {
		return fmt.Errorf("invalid operator address: %w", err)
	}

	// Validate participant ID
	if msg.Id == 0 {
		return fmt.Errorf("participant ID cannot be 0")
	}

	return nil
}

// ValidateBasic for MsgSetParticipantOPToValidated
func (msg *MsgSetParticipantOPToValidated) ValidateBasic() error {
	// [MOD-PP-MSG-3-2-1] authority (group): signature must be verified
	if _, err := sdk.AccAddressFromBech32(msg.Corporation); err != nil {
		return fmt.Errorf("invalid corporation address: %w", err)
	}

	// [MOD-PP-MSG-3-2-1] operator (account): signature must be verified
	if _, err := sdk.AccAddressFromBech32(msg.Operator); err != nil {
		return fmt.Errorf("invalid operator address: %w", err)
	}

	// Validate participant ID
	if msg.Id == 0 {
		return fmt.Errorf("participant ID cannot be 0")
	}

	// Validate digest SRI format if provided (optional)
	if msg.OpSummaryDigest != "" && !validation.IsValidDigestSRI(msg.OpSummaryDigest) {
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

// ValidateBasic for MsgConfirmParticipantVPTermination
func (msg *MsgCancelParticipantOPLastRequest) ValidateBasic() error {
	// Validate authority address
	if _, err := sdk.AccAddressFromBech32(msg.Corporation); err != nil {
		return fmt.Errorf("invalid corporation address: %w", err)
	}

	// Validate operator address
	if _, err := sdk.AccAddressFromBech32(msg.Operator); err != nil {
		return fmt.Errorf("invalid operator address: %w", err)
	}

	// Validate participant ID
	if msg.Id == 0 {
		return fmt.Errorf("participant ID cannot be 0")
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
	if !validation.IsValidDID(msg.Did) {
		return fmt.Errorf("invalid DID format")
	}

	// [MOD-PP-MSG-7-2-1] VSOA params: msg_types MUST be a subset of
	// [SetParticipantOPToValidated]; vs_operator required if any param set.
	anyParam := len(msg.VsOperatorAuthzMsgTypes) > 0 ||
		len(msg.VsOperatorAuthzSpendLimit) > 0 ||
		msg.VsOperatorAuthzWithFeegrant ||
		len(msg.VsOperatorAuthzFeeSpendLimit) > 0 ||
		msg.VsOperatorAuthzPeriod != nil
	if anyParam {
		if len(msg.VsOperatorAuthzMsgTypes) == 0 {
			return fmt.Errorf("vs_operator_authz_msg_types is required when any vs_operator_authz_* param is set")
		}
		if msg.VsOperator == "" {
			return fmt.Errorf("vs_operator is required when vs_operator_authz_* params are set")
		}
		for _, mt := range msg.VsOperatorAuthzMsgTypes {
			if mt != MsgSetParticipantOPToValidatedTypeURL {
				return fmt.Errorf("msg_type %s is not permitted for root participant (only SetParticipantOPToValidated)", mt)
			}
		}
		// The record is active immediately with expiration = effective_until; the
		// spec requires record.expiration, so effective_until MUST be set.
		if msg.EffectiveUntil == nil {
			return fmt.Errorf("effective_until is required when vs_operator_authz params are set")
		}
	}
	if msg.VsOperator != "" {
		if _, err := sdk.AccAddressFromBech32(msg.VsOperator); err != nil {
			return fmt.Errorf("invalid vs_operator address: %w", err)
		}
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
		return fmt.Errorf("participant ID cannot be 0")
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

	// Validate participant ID
	if msg.Id == 0 {
		return fmt.Errorf("participant ID cannot be 0")
	}

	return nil
}

func (msg *MsgCreateOrUpdateParticipantSession) ValidateBasic() error {
	// [MOD-PP-MSG-10-2] authority (group): signature must be verified
	if _, err := sdk.AccAddressFromBech32(msg.Corporation); err != nil {
		return fmt.Errorf("invalid corporation address: %w", err)
	}

	// [MOD-PP-MSG-10-2] operator (account): signature must be verified
	if _, err := sdk.AccAddressFromBech32(msg.Operator); err != nil {
		return fmt.Errorf("invalid operator address: %w", err)
	}

	// Validate UUID format
	if _, err := uuid.Parse(msg.Id); err != nil {
		return sdkerrors.ErrInvalidRequest.Wrap("invalid session ID: must be valid UUID")
	}

	// if issuer_participant_id is null AND verifier_participant_id is null, MUST abort
	if msg.IssuerParticipantId == 0 && msg.VerifierParticipantId == 0 {
		return sdkerrors.ErrInvalidRequest.Wrap("at least one of issuer_participant_id or verifier_participant_id must be provided")
	}

	// agent_participant_id is mandatory
	if msg.AgentParticipantId == 0 {
		return sdkerrors.ErrInvalidRequest.Wrap("agent_participant_id is mandatory")
	}

	// wallet_agent_participant_id is mandatory
	if msg.WalletAgentParticipantId == 0 {
		return sdkerrors.ErrInvalidRequest.Wrap("wallet_agent_participant_id is mandatory")
	}

	// Validate digest SRI format if provided
	if msg.Digest != "" && !validation.IsValidDigestSRI(msg.Digest) {
		return sdkerrors.ErrInvalidRequest.Wrap("invalid digest format")
	}

	return nil
}

func (msg *MsgSlashParticipantTrustDeposit) ValidateBasic() error {
	// [MOD-PP-MSG-12-2-1] authority (group): signature must be verified
	if _, err := sdk.AccAddressFromBech32(msg.Corporation); err != nil {
		return fmt.Errorf("invalid corporation address: %w", err)
	}

	// [MOD-PP-MSG-12-2-1] operator (account): signature must be verified
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
	// [MOD-PP-MSG-12-1] reason is mandatory per spec v4 draft 13
	if msg.Reason == "" {
		return sdkerrors.ErrInvalidRequest.Wrap("reason is required")
	}
	return nil
}

func (msg *MsgRepayParticipantSlashedTrustDeposit) ValidateBasic() error {
	// [MOD-PP-MSG-13-2-1] authority (group): signature must be verified
	if _, err := sdk.AccAddressFromBech32(msg.Corporation); err != nil {
		return fmt.Errorf("invalid corporation address: %w", err)
	}

	// [MOD-PP-MSG-13-2-1] operator (account): signature must be verified
	if _, err := sdk.AccAddressFromBech32(msg.Operator); err != nil {
		return fmt.Errorf("invalid operator address: %w", err)
	}

	// [MOD-PP-MSG-13-2-1] id MUST be a valid uint64
	if msg.Id == 0 {
		return sdkerrors.ErrInvalidRequest.Wrap("id must be a valid uint64")
	}

	if msg.Amount == 0 {
		return sdkerrors.ErrInvalidRequest.Wrap("amount must be positive")
	}

	return nil
}

func (msg *MsgSelfCreateParticipant) ValidateBasic() error {
	// [MOD-PP-MSG-14-2-1] corporation (group): signature must be verified
	if _, err := sdk.AccAddressFromBech32(msg.Corporation); err != nil {
		return fmt.Errorf("invalid corporation address: %w", err)
	}

	// [MOD-PP-MSG-14-2-1] operator (account): signature must be verified
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
	if !validation.IsValidDID(msg.Did) {
		return sdkerrors.ErrInvalidRequest.Wrap("invalid DID syntax")
	}

	// Validate vs_operator address if provided
	if msg.VsOperator != "" {
		if _, err := sdk.AccAddressFromBech32(msg.VsOperator); err != nil {
			return fmt.Errorf("invalid vs_operator address: %w", err)
		}
	}

	// [MOD-PP-MSG-14-2-1] VSOA params: per-role whitelist + presence rules.
	anyParam := len(msg.VsOperatorAuthzMsgTypes) > 0 ||
		len(msg.VsOperatorAuthzSpendLimit) > 0 ||
		msg.VsOperatorAuthzWithFeegrant ||
		len(msg.VsOperatorAuthzFeeSpendLimit) > 0 ||
		msg.VsOperatorAuthzPeriod != nil
	if err := validateVSOperatorAuthz(msg.Role, msg.VsOperator, msg.VsOperatorAuthzMsgTypes, anyParam); err != nil {
		return err
	}
	// OPEN mode creates the record active immediately with expiration =
	// effective_until; the spec requires record.expiration, so it MUST be set.
	if anyParam && msg.EffectiveUntil == nil {
		return fmt.Errorf("effective_until is required when vs_operator_authz params are set")
	}

	return nil
}
