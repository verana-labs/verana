package credentialschema_test

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	keepertest "github.com/verana-labs/verana/testutil/keeper"
	"github.com/verana-labs/verana/testutil/nullify"
	credentialschema "github.com/verana-labs/verana/x/credentialschema/module"
	"github.com/verana-labs/verana/x/credentialschema/types"
)

func TestGenesisImportExport(t *testing.T) {
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

	// Create test schemas with valid values
	now := time.Now().UTC()
	schemaA := types.CredentialSchema{
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

	schemaB := types.CredentialSchema{
		Id:                                      2,
		TrId:                                    101,
		Created:                                 now.Add(-10 * time.Hour),
		Modified:                                now.Add(-5 * time.Hour),
		JsonSchema:                              validJsonSchema,
		Deposit:                                 1500,
		IssuerGrantorValidationValidityPeriod:   180,
		VerifierGrantorValidationValidityPeriod: 180,
		IssuerValidationValidityPeriod:          90,
		VerifierValidationValidityPeriod:        90,
		HolderValidationValidityPeriod:          90,
		IssuerPermManagementMode:                types.CredentialSchemaPermManagementMode_OPEN,
		VerifierPermManagementMode:              types.CredentialSchemaPermManagementMode_ECOSYSTEM,
	}

	// Create a test genesis state with multiple schemas
	genesisState := types.GenesisState{
		Params:            types.DefaultParams(),
		CredentialSchemas: []types.CredentialSchema{schemaB, schemaA}, // Deliberately out of order
	}

	// Setup the module
	k, _, ctx := keepertest.CredentialschemaKeeper(t)

	// Test import
	credentialschema.InitGenesis(ctx, k, genesisState)

	// Verify schemas were imported correctly
	schemaFromState1, err := k.CredentialSchema.Get(ctx, 1)
	require.NoError(t, err)
	require.Equal(t, schemaA.Id, schemaFromState1.Id)
	require.Equal(t, schemaA.TrId, schemaFromState1.TrId)
	require.Equal(t, schemaA.IssuerPermManagementMode, schemaFromState1.IssuerPermManagementMode)

	schemaFromState2, err := k.CredentialSchema.Get(ctx, 2)
	require.NoError(t, err)
	require.Equal(t, schemaB.Id, schemaFromState2.Id)
	require.Equal(t, schemaB.TrId, schemaFromState2.TrId)
	require.Equal(t, schemaB.IssuerPermManagementMode, schemaFromState2.IssuerPermManagementMode)

	// Verify counter was set correctly (to highest ID)
	counterId, err := k.Counter.Get(ctx, "cs")
	require.NoError(t, err)
	require.Equal(t, uint64(2), counterId)

	// Test export
	exportedGenesis := credentialschema.ExportGenesis(ctx, k)
	require.NotNil(t, exportedGenesis)

	// Verify schemas are exported in deterministic order (by ID)
	require.Len(t, exportedGenesis.CredentialSchemas, 2)
	require.Equal(t, uint64(1), exportedGenesis.CredentialSchemas[0].Id)
	require.Equal(t, uint64(2), exportedGenesis.CredentialSchemas[1].Id)

	// Verify all parameters match
	require.Equal(t, genesisState.Params, exportedGenesis.Params)

	// Use nullify to ignore irrelevant fields and verify the exported data matches the imported data
	nullify.Fill(&genesisState)
	nullify.Fill(exportedGenesis)

	require.Equal(t, genesisState.Params, exportedGenesis.Params)
	require.ElementsMatch(t, genesisState.CredentialSchemas, exportedGenesis.CredentialSchemas)
}

func TestGenesisValidation(t *testing.T) {
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
		name         string
		genesisState types.GenesisState
		valid        bool
		errorMsg     string
	}{
		{
			name:         "default is valid",
			genesisState: *types.DefaultGenesis(),
			valid:        true,
		},
		{
			name: "valid genesis state with schemas",
			genesisState: types.GenesisState{
				Params:            types.DefaultParams(),
				CredentialSchemas: []types.CredentialSchema{validSchema},
			},
			valid: true,
		},
		{
			name: "duplicate schema ID",
			genesisState: types.GenesisState{
				Params: types.DefaultParams(),
				CredentialSchemas: []types.CredentialSchema{
					validSchema,
					{
						Id:                         1, // Same ID
						TrId:                       101,
						Created:                    now.Add(-10 * time.Hour),
						Modified:                   now.Add(-5 * time.Hour),
						JsonSchema:                 validJsonSchema,
						IssuerPermManagementMode:   types.CredentialSchemaPermManagementMode_OPEN,
						VerifierPermManagementMode: types.CredentialSchemaPermManagementMode_ECOSYSTEM,
					},
				},
			},
			valid:    false,
			errorMsg: "duplicate credential schema ID",
		},
		{
			name: "invalid schema ID (zero)",
			genesisState: types.GenesisState{
				Params: types.DefaultParams(),
				CredentialSchemas: []types.CredentialSchema{
					{
						Id:                         0, // Invalid ID
						TrId:                       101,
						Created:                    now.Add(-10 * time.Hour),
						Modified:                   now.Add(-5 * time.Hour),
						JsonSchema:                 validJsonSchema,
						IssuerPermManagementMode:   types.CredentialSchemaPermManagementMode_OPEN,
						VerifierPermManagementMode: types.CredentialSchemaPermManagementMode_ECOSYSTEM,
					},
				},
			},
			valid:    false,
			errorMsg: "invalid ID 0",
		},
		{
			name: "invalid trust registry ID (zero)",
			genesisState: types.GenesisState{
				Params: types.DefaultParams(),
				CredentialSchemas: []types.CredentialSchema{
					{
						Id:                         3,
						TrId:                       0, // Invalid TR ID
						Created:                    now.Add(-10 * time.Hour),
						Modified:                   now.Add(-5 * time.Hour),
						JsonSchema:                 validJsonSchema,
						IssuerPermManagementMode:   types.CredentialSchemaPermManagementMode_OPEN,
						VerifierPermManagementMode: types.CredentialSchemaPermManagementMode_ECOSYSTEM,
					},
				},
			},
			valid:    false,
			errorMsg: "invalid trust registry ID 0",
		},
		{
			name: "invalid empty JSON schema",
			genesisState: types.GenesisState{
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
			valid:    false,
			errorMsg: "empty JSON schema",
		},
		{
			name: "invalid perm management mode",
			genesisState: types.GenesisState{
				Params: types.DefaultParams(),
				CredentialSchemas: []types.CredentialSchema{
					{
						Id:                         3,
						TrId:                       101,
						Created:                    now.Add(-10 * time.Hour),
						Modified:                   now.Add(-5 * time.Hour),
						JsonSchema:                 validJsonSchema,
						IssuerPermManagementMode:   types.CredentialSchemaPermManagementMode_MODE_UNSPECIFIED, // Invalid mode
						VerifierPermManagementMode: types.CredentialSchemaPermManagementMode_ECOSYSTEM,
					},
				},
			},
			valid:    false,
			errorMsg: "invalid issuer perm management mode",
		},
		{
			name: "zero creation time",
			genesisState: types.GenesisState{
				Params: types.DefaultParams(),
				CredentialSchemas: []types.CredentialSchema{
					{
						Id:                         3,
						TrId:                       101,
						Created:                    time.Time{}, // Zero time
						Modified:                   now.Add(-5 * time.Hour),
						JsonSchema:                 validJsonSchema,
						IssuerPermManagementMode:   types.CredentialSchemaPermManagementMode_OPEN,
						VerifierPermManagementMode: types.CredentialSchemaPermManagementMode_ECOSYSTEM,
					},
				},
			},
			valid:    false,
			errorMsg: "invalid creation time",
		},
		{
			name: "creation time after modified time (inconsistent timestamps)",
			genesisState: types.GenesisState{
				Params: types.DefaultParams(),
				CredentialSchemas: []types.CredentialSchema{
					{
						Id:                         3,
						TrId:                       101,
						Created:                    now, // More recent than modified
						Modified:                   now.Add(-5 * time.Hour),
						JsonSchema:                 validJsonSchema,
						IssuerPermManagementMode:   types.CredentialSchemaPermManagementMode_OPEN,
						VerifierPermManagementMode: types.CredentialSchemaPermManagementMode_ECOSYSTEM,
					},
				},
			},
			valid:    false,
			errorMsg: "creation time after modified time",
		},
		{
			name: "exceeding max validity period days",
			genesisState: types.GenesisState{
				Params: types.DefaultParams(),
				CredentialSchemas: []types.CredentialSchema{
					{
						Id:                                      3,
						TrId:                                    101,
						Created:                                 now.Add(-10 * time.Hour),
						Modified:                                now.Add(-5 * time.Hour),
						JsonSchema:                              validJsonSchema,
						IssuerGrantorValidationValidityPeriod:   9999, // Exceeds max
						VerifierGrantorValidationValidityPeriod: 365,
						IssuerValidationValidityPeriod:          180,
						VerifierValidationValidityPeriod:        180,
						HolderValidationValidityPeriod:          180,
						IssuerPermManagementMode:                types.CredentialSchemaPermManagementMode_OPEN,
						VerifierPermManagementMode:              types.CredentialSchemaPermManagementMode_ECOSYSTEM,
					},
				},
			},
			valid:    false,
			errorMsg: "issuer grantor validation validity period exceeding maximum",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			err := tc.genesisState.Validate()
			if tc.valid {
				require.NoError(t, err)
			} else {
				require.Error(t, err)
				if tc.errorMsg != "" {
					require.Contains(t, err.Error(), tc.errorMsg)
				}
			}
		})
	}
}
