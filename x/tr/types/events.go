package types

const (
	EventTypeCreateTrustRegistry               = "create_trust_registry"
	EventTypeCreateGovernanceFrameworkVersion  = "create_governance_framework_version"
	EventTypeCreateGovernanceFrameworkDocument = "create_governance_framework_document"

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
)
