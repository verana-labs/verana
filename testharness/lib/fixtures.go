package lib

// GenerateSimpleSchema generates a simple schema with just a name property
func GenerateSimpleSchema(trustRegistryID string) string {
	return `{
  "$id": "vpr:verana:VPR_CHAIN_ID/cs/v1/js/VPR_CREDENTIAL_SCHEMA_ID",
  "$schema": "https://json-schema.org/draft/2020-12/schema",
  "title": "ExampleCredential",
  "description": "ExampleCredential using JsonSchema",
  "type": "object",
  "properties": {
    "credentialSubject": {
      "type": "object",
      "properties": {
        "id": {
          "type": "string",
          "format": "uri"
        },
        "firstName": {
          "type": "string",
          "minLength": 0,
          "maxLength": 256
        },
        "lastName": {
          "type": "string",
          "minLength": 1,
          "maxLength": 256
        },
        "expirationDate": {
          "type": "string",
          "format": "date"
        },
        "countryOfResidence": {
          "type": "string",
          "minLength": 2,
          "maxLength": 2
        }
      },
      "required": [
        "id",
        "lastName",
        "birthDate",
        "expirationDate",
        "countryOfResidence"
      ]
    }
  }
}`
}

type JourneyResult struct {
	// Journey 1 fields
	TrustRegistryID  string
	SchemaID         string
	RootPermissionID string
	DID              string

	// Journey 2 fields
	IssuerGrantorDID    string
	IssuerGrantorPermID string

	// Journey 3 fields
	IssuerDID    string
	IssuerPermID string

	// Journey 4 fields
	VerifierDID    string
	VerifierPermID string

	// Journey 5 fields
	HolderWalletDID     string
	HolderWalletPermID  string
	CredentialSessionID string

	// Journey 6 fields
	VerificationSessionID string

	// Journey 7 fields
	RenewalTimestamp string

	// Journey 8 fields
	TerminationTimestamp string

	// Journey 9 fields
	GfVersion     string
	GfDocumentURL string

	// Journey 10 fields
	DepositHolder        string
	DepositHolderAddress string
	InitialDepositAmount string
	FinalDepositAmount   string
	AmountReclaimed      string

	// Journey 12 fields
	RevocationTimestamp string

	// Journey 13 fields
	ExtensionTimestamp string
	NewEffectiveUntil  string
	PermissionID       string

	// Journey 15 fields
	FailedPermissionID string
}
