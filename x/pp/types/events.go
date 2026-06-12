package types

const (
	EventTypeCreateRootParticipant               = "create_root_participant"
	AttributeKeyRootParticipantID                = "root_participant_id"
	AttributeKeySchemaID                         = "schema_id"
	AttributeKeyTimestamp                        = "timestamp"
	EventTypeStartParticipantOP                  = "start_participant_op"
	AttributeKeyParticipantID                    = "participant_id"
	AttributeKeyCreator                          = "creator"
	AttributeKeyAuthority                        = "authority"
	AttributeKeyCorporation                      = "corporation"
	AttributeKeyCorporationID                    = "corporation_id"
	AttributeKeyOperator                         = "operator"
	AttributeKeyFees                             = "fees"
	AttributeKeyDeposit                          = "deposit"
	EventTypeCreateOrUpdateParticipantSession    = "create_update_csps"
	AttributeKeySessionID                        = "session_id"
	AttributeKeyAgentParticipantID               = "agent_participant_id"
	AttributeKeyIssuerParticipantID              = "issuer_participant_id"
	AttributeKeyVerifierParticipantID            = "verifier_participant_id"
	AttributeKeyWalletAgentParticipantID         = "wallet_agent_participant_id"
	EventTypeSlashParticipantTrustDeposit        = "slash_participant_trust_deposit"
	AttributeKeySlashedAmount                    = "slashed_amount"
	EventTypeRepayParticipantSlashedTrustDeposit = "repay_participant_slashed_trust_deposit"
	AttributeKeyRepaidAmount                     = "repaid_amount"
	EventTypeCreateParticipant                   = "create_participant"
	AttributeKeyValidatorParticipantID           = "validator_participant_id"
	AttributeKeyRole                             = "role"

	EventTypeRenewParticipantOP   = "renew_participant_op"
	AttributeKeyValidationFees    = "validation_fees"
	AttributeKeyValidationDeposit = "validation_deposit"

	EventTypeSetParticipantOPToValidated = "set_participant_op_to_validated"
	AttributeKeyOpSummaryDigest          = "op_summary_digest"
	AttributeKeyEffectiveFrom            = "effective_from"
	AttributeKeyEffectiveUntil           = "effective_until"
	AttributeKeyIssuanceFees             = "issuance_fees"
	AttributeKeyVerificationFees         = "verification_fees"
	AttributeKeyOpExp                    = "op_exp"

	EventTypeCancelParticipantOPLastRequest = "cancel_participant_op_last_request"

	EventTypeSetParticipantEffectiveUntil = "set_participant_effective_until"
	AttributeKeyNewEffectiveUntil         = "new_effective_until"

	EventTypeRevokeParticipant = "revoke_participant"
	AttributeKeyRevokedAt      = "revoked_at"

	EventTypeTriggerResolver = "trigger_resolver"
)
