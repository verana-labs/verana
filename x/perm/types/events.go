package types

const (
	EventTypeCreateRootPermission               = "create_root_permission"
	AttributeKeyRootPermissionID                = "root_permission_id"
	AttributeKeySchemaID                        = "schema_id"
	AttributeKeyTimestamp                       = "timestamp"
	EventTypeStartPermissionVP                  = "start_permission_vp"
	AttributeKeyPermissionID                    = "permission_id"
	AttributeKeyCreator                         = "creator"
	AttributeKeyAuthority                       = "authority"
	AttributeKeyCorporation                     = "corporation"
	AttributeKeyOperator                        = "operator"
	AttributeKeyFees                            = "fees"
	AttributeKeyDeposit                         = "deposit"
	EventTypeCreateOrUpdatePermissionSession    = "create_update_csps"
	AttributeKeySessionID                       = "session_id"
	AttributeKeyAgentPermID                     = "agent_perm_id"
	AttributeKeyIssuerPermID                    = "issuer_perm_id"
	AttributeKeyVerifierPermID                  = "verifier_perm_id"
	AttributeKeyWalletAgentPermID               = "wallet_agent_perm_id"
	EventTypeSlashPermissionTrustDeposit        = "slash_permission_trust_deposit"
	AttributeKeySlashedAmount                   = "slashed_amount"
	EventTypeRepayPermissionSlashedTrustDeposit = "repay_permission_slashed_trust_deposit"
	AttributeKeyRepaidAmount                    = "repaid_amount"
	EventTypeCreatePermission                   = "create_permission"
	AttributeKeyValidatorPermID                 = "validator_perm_id"
	AttributeKeyType                            = "type"

	EventTypeRenewPermissionVP    = "renew_permission_vp"
	AttributeKeyValidationFees    = "validation_fees"
	AttributeKeyValidationDeposit = "validation_deposit"

	EventTypeSetPermissionVPToValidated = "set_permission_vp_to_validated"
	AttributeKeyVpSummaryDigest         = "vp_summary_digest"
	AttributeKeyEffectiveFrom           = "effective_from"
	AttributeKeyEffectiveUntil          = "effective_until"
	AttributeKeyIssuanceFees            = "issuance_fees"
	AttributeKeyVerificationFees        = "verification_fees"
	AttributeKeyVpExp                   = "vp_exp"

	EventTypeCancelPermissionVPLastRequest = "cancel_permission_vp_last_request"

	EventTypeAdjustPermission     = "adjust_permission"
	AttributeKeyNewEffectiveUntil = "new_effective_until"

	EventTypeRevokePermission = "revoke_permission"
	AttributeKeyRevokedAt     = "revoked_at"

	// [MOD-PERM-MSG-15] Trigger Resolver
	EventTypeTriggerResolver = "trigger_resolver"
)
