package types

const (
	EventTypeCreateTrustRegistry               = "create_trust_registry"
	EventTypeCreateGovernanceFrameworkVersion  = "create_governance_framework_version"
	EventTypeCreateGovernanceFrameworkDocument = "create_governance_framework_document"
	EventTypeAddGovernanceFrameworkDocument    = "add_governance_framework_document"
	EventTypeIncreaseActiveGFVersion           = "increase_active_gf_version"
	EventTypeUpdateTrustRegistry               = "update_trust_registry"
	EventTypeArchiveTrustRegistry              = "archive_trust_registry"

	AttributeKeyTrustRegistryID = "trust_registry_id"
	AttributeKeyDID             = "did"
	AttributeKeyController      = "controller"
	AttributeKeyAka             = "aka"
	AttributeKeyLanguage        = "language"
	AttributeKeyTimestamp       = "timestamp"
	AttributeKeyGFVersionID     = "gf_version_id"
	AttributeKeyVersion         = "version"
	AttributeKeyGFDocumentID    = "gf_document_id"
	AttributeKeyDocURL          = "doc_url"
	AttributeKeyDigestSri       = "digest_sri"
	AttributeKeyDeposit         = "deposit"
	AttributeKeyArchiveStatus   = "archive_status"
)
