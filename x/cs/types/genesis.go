package types

import "fmt"

// this line is used by starport scaffolding # genesis/types/import

// DefaultIndex is the default global index
const DefaultIndex uint64 = 1

// DefaultGenesis returns the default genesis state
func DefaultGenesis() *GenesisState {
	return &GenesisState{
		Params:            DefaultParams(),
		CredentialSchemas: []CredentialSchema{},
		SchemaCounter:     0, // Start counter at 0
	}
}

// Validate performs basic genesis state validation returning an error upon any
// failure.
func (gs GenesisState) Validate() error {
	// Validate parameters
	if err := gs.Params.Validate(); err != nil {
		return fmt.Errorf("invalid params: %w", err)
	}

	// Validate credential schemas
	seenCredentialSchemaIDs := make(map[uint64]bool)

	// Validate each credential schema and check for duplicates
	for i, cs := range gs.CredentialSchemas {
		// Check for mandatory fields
		if cs.Id == 0 {
			return fmt.Errorf("credential schema at index %d has invalid ID 0", i)
		}

		if cs.TrId == 0 {
			return fmt.Errorf("credential schema at index %d has invalid trust registry ID 0", i)
		}

		if cs.Created.IsZero() {
			return fmt.Errorf("credential schema at index %d has invalid creation time", i)
		}

		if cs.Modified.IsZero() {
			return fmt.Errorf("credential schema at index %d has invalid modified time", i)
		}

		if cs.JsonSchema == "" {
			return fmt.Errorf("credential schema at index %d has empty JSON schema", i)
		}

		// Check perm management modes are valid
		if cs.IssuerPermManagementMode <= CredentialSchemaPermManagementMode_MODE_UNSPECIFIED ||
			cs.IssuerPermManagementMode > CredentialSchemaPermManagementMode_ECOSYSTEM {
			return fmt.Errorf("credential schema at index %d has invalid issuer perm management mode: %d",
				i, cs.IssuerPermManagementMode)
		}

		if cs.VerifierPermManagementMode <= CredentialSchemaPermManagementMode_MODE_UNSPECIFIED ||
			cs.VerifierPermManagementMode > CredentialSchemaPermManagementMode_ECOSYSTEM {
			return fmt.Errorf("credential schema at index %d has invalid verifier perm management mode: %d",
				i, cs.VerifierPermManagementMode)
		}

		// Validate pricing_asset_type
		if cs.PricingAssetType <= PricingAssetType_PRICING_ASSET_TYPE_UNSPECIFIED ||
			cs.PricingAssetType > PricingAssetType_FIAT {
			return fmt.Errorf("credential schema at index %d has invalid pricing_asset_type: %d",
				i, cs.PricingAssetType)
		}

		// Validate pricing_asset
		if cs.PricingAsset == "" {
			return fmt.Errorf("credential schema at index %d has empty pricing_asset", i)
		}

		// Validate digest_algorithm
		if !ValidDigestAlgorithms[cs.DigestAlgorithm] {
			return fmt.Errorf("credential schema at index %d has invalid digest_algorithm: %s", i, cs.DigestAlgorithm)
		}

		// Validate JSON schema format (basic check)
		if err := validateJSONSchema(cs.JsonSchema); err != nil {
			return fmt.Errorf("credential schema at index %d has invalid JSON schema: %w", i, err)
		}

		// Check for duplicate schema IDs
		if seenCredentialSchemaIDs[cs.Id] {
			return fmt.Errorf("duplicate credential schema ID found in genesis state: %d", cs.Id)
		}
		seenCredentialSchemaIDs[cs.Id] = true

		// Additional validations for validity periods
		if cs.IssuerGrantorValidationValidityPeriod > DefaultCredentialSchemaIssuerGrantorValidationValidityPeriodMaxDays {
			return fmt.Errorf("credential schema at index %d has issuer grantor validation validity period exceeding maximum", i)
		}

		if cs.VerifierGrantorValidationValidityPeriod > DefaultCredentialSchemaVerifierGrantorValidationValidityPeriodMaxDays {
			return fmt.Errorf("credential schema at index %d has verifier grantor validation validity period exceeding maximum", i)
		}

		if cs.IssuerValidationValidityPeriod > DefaultCredentialSchemaIssuerValidationValidityPeriodMaxDays {
			return fmt.Errorf("credential schema at index %d has issuer validation validity period exceeding maximum", i)
		}

		if cs.VerifierValidationValidityPeriod > DefaultCredentialSchemaVerifierValidationValidityPeriodMaxDays {
			return fmt.Errorf("credential schema at index %d has verifier validation validity period exceeding maximum", i)
		}

		if cs.HolderValidationValidityPeriod > DefaultCredentialSchemaHolderValidationValidityPeriodMaxDays {
			return fmt.Errorf("credential schema at index %d has holder validation validity period exceeding maximum", i)
		}

		// Validate consistency with other fields
		if cs.Created.After(cs.Modified) {
			return fmt.Errorf("credential schema at index %d has creation time after modified time", i)
		}

		// Validate archive status consistency
		if cs.Archived != nil {
			if cs.Archived.Before(cs.Created) {
				return fmt.Errorf("credential schema at index %d has archive time before creation time", i)
			}
		}
	}

	return nil
}
