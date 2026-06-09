package types

const (
	EventTypeGrantOperatorAuthorization    = "grant_operator_authorization"
	EventTypeRevokeOperatorAuthorization   = "revoke_operator_authorization"
	EventTypeGrantFeeAllowance             = "grant_fee_allowance"
	EventTypeRevokeFeeAllowance            = "revoke_fee_allowance"
	EventTypeGrantVSOperatorAuthorization  = "grant_vs_operator_authorization"
	EventTypeRevokeVSOperatorAuthorization = "revoke_vs_operator_authorization"
	EventTypeUpdateVSOperatorAuthorization = "update_vs_operator_authorization"

	AttributeKeyCorporation   = "corporation"
	AttributeKeyCorporationID = "corporation_id"
	AttributeKeyOperator      = "operator"
	AttributeKeyGrantee       = "grantee"
	AttributeKeyWithFeegrant  = "with_feegrant"
	AttributeKeyTimestamp     = "timestamp"
	AttributeKeyVsOperator    = "vs_operator"
	AttributeKeyPermissionID  = "permission_id"
	AttributeKeyAuthzID       = "authz_id"
	AttributeKeyVsoaID        = "vsoa_id"
	AttributeKeyParticipantID = "participant_id"
)
