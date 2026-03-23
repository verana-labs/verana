package types

const (
	EventTypeGrantOperatorAuthorization    = "grant_operator_authorization"
	EventTypeRevokeOperatorAuthorization   = "revoke_operator_authorization"
	EventTypeGrantFeeAllowance             = "grant_fee_allowance"
	EventTypeRevokeFeeAllowance            = "revoke_fee_allowance"
	EventTypeGrantVSOperatorAuthorization  = "grant_vs_operator_authorization"
	EventTypeRevokeVSOperatorAuthorization = "revoke_vs_operator_authorization"

	AttributeKeyAuthority    = "authority"
	AttributeKeyOperator     = "operator"
	AttributeKeyGrantee      = "grantee"
	AttributeKeyVsOperator   = "vs_operator"
	AttributeKeyPermissionID = "permission_id"
	AttributeKeyWithFeegrant = "with_feegrant"
	AttributeKeyTimestamp    = "timestamp"
)
