package types_test

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"github.com/verana-labs/verana-blockchain/x/cs/types"
)

func TestGenesisState_Validate(t *testing.T) {
	// Create a valid JSON schema for testing
	validJsonSchema := `{
  "$id": "vpr:verana:mainnet/cs/v1/js/1",
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
		TrId:                                    100,
		Created:                                 now.Add(-24 * time.Hour),
		Modified:                                now.Add(-12 * time.Hour),
		JsonSchema:                              validJsonSchema,
		Deposit:                                 1000,
		IssuerGrantorValidationValidityPeriod:   365,
		VerifierGrantorValidationValidityPeriod: 365,
		IssuerValidationValidityPeriod:          180,
		VerifierValidationValidityPeriod:        180,
		HolderValidationValidityPeriod:          180,
		IssuerPermManagementMode:                types.CredentialSchemaPermManagementMode_GRANTOR_VALIDATION,
		VerifierPermManagementMode:              types.CredentialSchemaPermManagementMode_GRANTOR_VALIDATION,
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
						TrId:                       101,
						Created:                    now.Add(-10 * time.Hour),
						Modified:                   now.Add(-5 * time.Hour),
						JsonSchema:                 validJsonSchema,
						IssuerPermManagementMode:   types.CredentialSchemaPermManagementMode_OPEN,
						VerifierPermManagementMode: types.CredentialSchemaPermManagementMode_ECOSYSTEM,
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
						TrId:                       101,
						Created:                    now.Add(-10 * time.Hour),
						Modified:                   now.Add(-5 * time.Hour),
						JsonSchema:                 "", // Empty schema
						IssuerPermManagementMode:   types.CredentialSchemaPermManagementMode_OPEN,
						VerifierPermManagementMode: types.CredentialSchemaPermManagementMode_ECOSYSTEM,
					},
				},
			},
			valid: false,
		},
		{
			name: "invalid parameter values",
			genState: &types.GenesisState{
				Params: types.Params{
					CredentialSchemaTrustDeposit:                                   0, // Invalid - must be positive
					CredentialSchemaSchemaMaxSize:                                  1000,
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
