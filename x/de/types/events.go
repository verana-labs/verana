package types

const (
	EventTypeGrantOperatorAuthorization = "grant_operator_authorization"
	EventTypeGrantFeeAllowance          = "grant_fee_allowance"
	EventTypeRevokeFeeAllowance         = "revoke_fee_allowance"

	AttributeKeyAuthority    = "authority"
	AttributeKeyOperator     = "operator"
	AttributeKeyGrantee      = "grantee"
	AttributeKeyWithFeegrant = "with_feegrant"
	AttributeKeyTimestamp    = "timestamp"
)
