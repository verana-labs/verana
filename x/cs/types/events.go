package types

// Event types and attribute keys for credential schema module
const (
	// EventTypeCreateCredentialSchema is the event type for creating a credential schema
	EventTypeCreateCredentialSchema  = "create_credential_schema"
	EventTypeUpdateCredentialSchema  = "update_credential_schema"
	EventTypeArchiveCredentialSchema = "archive_credential_schema"

	// Attribute keys
	AttributeKeyId                                      = "credential_schema_id"
	AttributeKeyTrId                                    = "trust_registry_id"
	AttributeKeyAuthority                               = "authority"
	AttributeKeyOperator                                = "operator"
	AttributeKeyTimestamp                               = "timestamp"
	AttributeKeyArchiveStatus                           = "archive_status"
	AttributeKeyIssuerGrantorValidationValidityPeriod   = "issuer_grantor_validation_validity_period"
	AttributeKeyVerifierGrantorValidationValidityPeriod = "verifier_grantor_validation_validity_period"
	AttributeKeyIssuerValidationValidityPeriod          = "issuer_validation_validity_period"
	AttributeKeyVerifierValidationValidityPeriod        = "verifier_validation_validity_period"
	AttributeKeyHolderValidationValidityPeriod          = "holder_validation_validity_period"
	AttributeKeyPricingAssetType                        = "pricing_asset_type"
	AttributeKeyPricingAsset                            = "pricing_asset"
	AttributeKeyDigestAlgorithm                         = "digest_algorithm"
)
