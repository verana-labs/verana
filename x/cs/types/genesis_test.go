package types_test

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"github.com/verana-labs/verana/x/cs/types"
)

func TestGenesisState_Validate(t *testing.T) {
	// Create a valid JSON schema for testing
	validJsonSchema := `{
  "$id": "vpr:verana:VPR_CHAIN_ID:cs:VPR_CREDENTIAL_SCHEMA_ID",
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

	// Setup valid schema for testing
	now := time.Now().UTC()
	validSchema := types.CredentialSchema{
		Id:                                      1,
		EcosystemId:                                    100,
		Created:                                 now.Add(-24 * time.Hour),
		Modified:                                now.Add(-12 * time.Hour),
		JsonSchema:                              validJsonSchema,
		IssuerGrantorValidationValidityPeriod:   365,
		VerifierGrantorValidationValidityPeriod: 365,
		IssuerValidationValidityPeriod:          180,
		VerifierValidationValidityPeriod:        180,
		HolderValidationValidityPeriod:          180,
		IssuerOnboardingMode:                    types.IssuerOnboardingMode_ISSUER_ONBOARDING_MODE_GRANTOR_VALIDATION_PROCESS,
		VerifierOnboardingMode:                  types.VerifierOnboardingMode_VERIFIER_ONBOARDING_MODE_GRANTOR_VALIDATION_PROCESS,
		HolderOnboardingMode:                    types.HolderOnboardingMode_HOLDER_ONBOARDING_MODE_UNSPECIFIED,
		PricingAssetType:                        types.PricingAssetType_TU,
		PricingAsset:                            "tu",
		DigestAlgorithm:                         "sha256",
	}

	tests := []struct {
		name     string
		genState *types.GenesisState
		valid    bool
	}{
		{
			name:     "default is valid",
			genState: types.DefaultGenesis(),
			valid:    true,
		},
		{
			name: "valid genesis state with schema",
			genState: &types.GenesisState{
				Params:            types.DefaultParams(),
				CredentialSchemas: []types.CredentialSchema{validSchema},
			},
			valid: true,
		},
		{
			name: "duplicate schema ID",
			genState: &types.GenesisState{
				Params: types.DefaultParams(),
				CredentialSchemas: []types.CredentialSchema{
					validSchema,
					{
						Id:                         1, // Duplicate ID
						EcosystemId:                       101,
						Created:                    now.Add(-10 * time.Hour),
						Modified:                   now.Add(-5 * time.Hour),
						JsonSchema:                 validJsonSchema,
						IssuerOnboardingMode:       types.IssuerOnboardingMode_ISSUER_ONBOARDING_MODE_OPEN,
						VerifierOnboardingMode:     types.VerifierOnboardingMode_VERIFIER_ONBOARDING_MODE_ECOSYSTEM_VALIDATION_PROCESS,
						PricingAssetType:           types.PricingAssetType_TU,
						PricingAsset:               "tu",
						DigestAlgorithm:            "sha256",
					},
				},
			},
			valid: false,
		},
		{
			name: "missing JSON schema",
			genState: &types.GenesisState{
				Params: types.DefaultParams(),
				CredentialSchemas: []types.CredentialSchema{
					{
						Id:                         3,
						EcosystemId:                       101,
						Created:                    now.Add(-10 * time.Hour),
						Modified:                   now.Add(-5 * time.Hour),
						JsonSchema:                 "", // Empty schema
						IssuerOnboardingMode:       types.IssuerOnboardingMode_ISSUER_ONBOARDING_MODE_OPEN,
						VerifierOnboardingMode:     types.VerifierOnboardingMode_VERIFIER_ONBOARDING_MODE_ECOSYSTEM_VALIDATION_PROCESS,
						PricingAssetType:           types.PricingAssetType_TU,
						PricingAsset:               "tu",
						DigestAlgorithm:            "sha256",
					},
				},
			},
			valid: false,
		},
		{
			name: "invalid parameter values",
			genState: &types.GenesisState{
				Params: types.Params{
					CredentialSchemaSchemaMaxSize:                                  0, // Invalid - must be positive
					CredentialSchemaIssuerGrantorValidationValidityPeriodMaxDays:   365,
					CredentialSchemaVerifierGrantorValidationValidityPeriodMaxDays: 365,
					CredentialSchemaIssuerValidationValidityPeriodMaxDays:          180,
					CredentialSchemaVerifierValidationValidityPeriodMaxDays:        180,
					CredentialSchemaHolderValidationValidityPeriodMaxDays:          180,
				},
				CredentialSchemas: []types.CredentialSchema{validSchema},
			},
			valid: false,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			err := tc.genState.Validate()
			if tc.valid {
				require.NoError(t, err)
			} else {
				require.Error(t, err)
			}
		})
	}
}
