package types

// Event types and attribute keys for credential schema module
const (
	// EventTypeCreateCredentialSchema is the event type for creating a credential schema
	EventTypeCreateCredentialSchema = "create_credential_schema"

	// Attribute keys
	AttributeKeyId      = "credential_schema_id"
	AttributeKeyTrId    = "trust_registry_id"
	AttributeKeyCreator = "creator"
	AttributeKeyDeposit = "deposit"
)
