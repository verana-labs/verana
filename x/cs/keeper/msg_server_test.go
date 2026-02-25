package keeper_test

import (
	"context"
	"encoding/json"
	"fmt"
	"testing"

	"time"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/stretchr/testify/require"

	keepertest "github.com/verana-labs/verana/testutil/keeper"
	"github.com/verana-labs/verana/x/cs/keeper"
	"github.com/verana-labs/verana/x/cs/types"
)

func setupMsgServer(t testing.TB) (*keeper.Keeper, types.MsgServer, *keepertest.MockTrustRegistryKeeper, context.Context) {
	k, mockTrk, ctx := keepertest.CredentialschemaKeeper(t)
	return &k, keeper.NewMsgServerImpl(k), mockTrk, ctx
}

func TestMsgServerCreateCredentialSchema(t *testing.T) {
	k, ms, mockTrk, ctx := setupMsgServer(t)

	authority := sdk.AccAddress([]byte("test_authority______")).String()
	operator := sdk.AccAddress([]byte("test_operator_______")).String()
	validDid := "did:example:123456789abcdefghi"

	// First create a trust registry with authority as controller
	trID := mockTrk.CreateMockTrustRegistry(authority, validDid)

	// Schema with placeholder $id (will be replaced with canonical $id)
	validJsonSchemaWithPlaceholder := `{
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
        "expirationDate",
        "countryOfResidence"
      ]
    }
  }
}`

	// Schema with no $id field (will have canonical $id injected)
	validJsonSchemaNoId := `{
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
        "expirationDate",
        "countryOfResidence"
      ]
    }
  }
}`

	// Schema with wrong $id (will be replaced with canonical $id)
	validJsonSchemaWrongId := `{
  "$id": "lol-not-even-a-uri",
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
        "expirationDate",
        "countryOfResidence"
      ]
    }
  }
}`

	// Test basic JSON parsing
	var schemaDoc map[string]interface{}
	err := json.Unmarshal([]byte(validJsonSchemaWithPlaceholder), &schemaDoc)
	require.NoError(t, err, "JSON should be valid")

	// Test the meta-schema JSON parsing
	var metaDoc map[string]interface{}
	err = json.Unmarshal([]byte(types.JsonSchemaMetaSchema), &metaDoc)
	require.NoError(t, err, "Meta-schema JSON should be valid")

	testCases := []struct {
		name              string
		msg               *types.MsgCreateCredentialSchema
		isValid           bool
		expectIdInjection bool
	}{
		{
			name:              "Valid Create Credential Schema with placeholder $id",
			msg:               keeper.CreateMsgWithValidityPeriods(authority, operator, trID, validJsonSchemaWithPlaceholder, 365, 365, 180, 180, 180, 2, 2, 1, "tu", "sha256"),
			isValid:           true,
			expectIdInjection: true,
		},
		{
			name:              "Valid Create Credential Schema with no $id",
			msg:               keeper.CreateMsgWithValidityPeriods(authority, operator, trID, validJsonSchemaNoId, 365, 365, 180, 180, 180, 2, 2, 1, "tu", "sha256"),
			isValid:           true,
			expectIdInjection: true,
		},
		{
			name:              "Valid Create Credential Schema with wrong $id",
			msg:               keeper.CreateMsgWithValidityPeriods(authority, operator, trID, validJsonSchemaWrongId, 365, 365, 180, 180, 180, 2, 2, 1, "tu", "sha256"),
			isValid:           true,
			expectIdInjection: true,
		},
		{
			name:              "Non-existent Trust Registry",
			msg:               keeper.CreateMsgWithValidityPeriods(authority, operator, 999, validJsonSchemaWithPlaceholder, 365, 365, 180, 180, 180, 2, 2, 1, "tu", "sha256"),
			isValid:           false,
			expectIdInjection: false,
		},
		{
			name:              "Wrong Trust Registry Controller",
			msg:               keeper.CreateMsgWithValidityPeriods(sdk.AccAddress([]byte("wrong_authority_____")).String(), operator, trID, validJsonSchemaWithPlaceholder, 365, 365, 180, 180, 180, 2, 2, 1, "tu", "sha256"),
			isValid:           false,
			expectIdInjection: false,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// Test stateless validation first
			err := tc.msg.ValidateBasic()
			if tc.isValid {
				require.NoError(t, err, "ValidateBasic should pass for valid message")

				// Then test the message server
				resp, err := ms.CreateCredentialSchema(ctx, tc.msg)
				require.NoError(t, err)
				require.NotNil(t, resp)

				// Verify canonical $id injection if expected
				if tc.expectIdInjection {
					sdkCtx := sdk.UnwrapSDKContext(ctx)
					schema, err := k.CredentialSchema.Get(ctx, resp.Id)
					require.NoError(t, err)

					var schemaDoc map[string]interface{}
					err = json.Unmarshal([]byte(schema.JsonSchema), &schemaDoc)
					require.NoError(t, err)

					canonicalId, ok := schemaDoc["$id"].(string)
					require.True(t, ok, "$id field should be present")
					expectedId := fmt.Sprintf("vpr:verana:%s/cs/v1/js/%d", sdkCtx.ChainID(), resp.Id)
					require.Equal(t, expectedId, canonicalId, "Schema should have canonical $id")

					// Verify new fields
					require.Equal(t, types.PricingAssetType(tc.msg.PricingAssetType), schema.PricingAssetType)
					require.Equal(t, tc.msg.PricingAsset, schema.PricingAsset)
					require.Equal(t, tc.msg.DigestAlgorithm, schema.DigestAlgorithm)
				}
			} else {
				// For invalid cases, check if it fails in ValidateBasic OR message server
				resp, msgServerErr := ms.CreateCredentialSchema(ctx, tc.msg)

				if err == nil {
					require.Error(t, msgServerErr)
				}
				require.Nil(t, resp)
			}
		})
	}
}

func TestCanonicalIdInjection(t *testing.T) {
	k, ms, mockTrk, ctx := setupMsgServer(t)

	authority := sdk.AccAddress([]byte("test_authority______")).String()
	operator := sdk.AccAddress([]byte("test_operator_______")).String()
	validDid := "did:example:123456789abcdefghi"

	trID := mockTrk.CreateMockTrustRegistry(authority, validDid)

	testCases := []struct {
		name        string
		inputSchema string
		description string
	}{
		{
			name: "No $id field",
			inputSchema: `{
  "$schema": "https://json-schema.org/draft/2020-12/schema",
  "title": "NoIdSchema",
  "description": "Schema without $id field",
  "type": "object",
  "properties": {
    "name": { "type": "string" }
  },
  "required": ["name"],
  "additionalProperties": false
}`,
			description: "Schema without $id should have canonical $id injected",
		},
		{
			name: "Malformed $id field",
			inputSchema: `{
  "$schema": "https://json-schema.org/draft/2020-12/schema",
  "$id": "lol-not-even-a-uri",
  "title": "BadIdSchema",
  "description": "Schema with malformed $id",
  "type": "object",
  "properties": {
    "name": { "type": "string" }
  },
  "required": ["name"],
  "additionalProperties": false
}`,
			description: "Malformed $id should be replaced with canonical $id",
		},
		{
			name: "Wrong format $id",
			inputSchema: `{
  "$schema": "https://json-schema.org/draft/2020-12/schema",
  "$id": "https://totally.not/verana",
  "title": "WrongIdSchema",
  "description": "Schema with wrong $id format",
  "type": "object",
  "properties": {
    "name": { "type": "string" }
  },
  "required": ["name"],
  "additionalProperties": false
}`,
			description: "Wrong $id format should be replaced with canonical $id",
		},
		{
			name: "Correct placeholder $id",
			inputSchema: `{
  "$schema": "https://json-schema.org/draft/2020-12/schema",
  "$id": "vpr:verana:VPR_CHAIN_ID/cs/v1/js/VPR_CREDENTIAL_SCHEMA_ID",
  "title": "PlaceholderIdSchema",
  "description": "Schema with placeholder $id",
  "type": "object",
  "properties": {
    "name": { "type": "string" }
  },
  "required": ["name"],
  "additionalProperties": false
}`,
			description: "Placeholder $id should be replaced with canonical $id",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			createMsg := keeper.CreateMsgWithValidityPeriods(authority, operator, trID, tc.inputSchema, 365, 365, 180, 180, 180, 2, 2, 1, "tu", "sha256")
			resp, err := ms.CreateCredentialSchema(ctx, createMsg)
			require.NoError(t, err)
			require.NotNil(t, resp)

			sdkCtx := sdk.UnwrapSDKContext(ctx)

			storedSchema, err := k.CredentialSchema.Get(ctx, resp.Id)
			require.NoError(t, err)

			var schemaDoc map[string]interface{}
			err = json.Unmarshal([]byte(storedSchema.JsonSchema), &schemaDoc)
			require.NoError(t, err, "Stored JSON schema should be valid JSON")

			storedId, ok := schemaDoc["$id"].(string)
			require.True(t, ok, "$id field should be present in stored schema")
			expectedId := fmt.Sprintf("vpr:verana:%s/cs/v1/js/%d", sdkCtx.ChainID(), resp.Id)
			require.Equal(t, expectedId, storedId, "Stored schema should have canonical $id")

			require.Equal(t, "https://json-schema.org/draft/2020-12/schema", schemaDoc["$schema"], "$schema should be preserved")

			title, ok := schemaDoc["title"].(string)
			require.True(t, ok, "title should be preserved")
			require.NotEmpty(t, title, "title should not be empty")

			desc, ok := schemaDoc["description"].(string)
			require.True(t, ok, "description should be preserved")
			require.NotEmpty(t, desc, "description should not be empty")

			require.Equal(t, "object", schemaDoc["type"], "type should be preserved as 'object'")

			t.Logf("Test '%s': %s", tc.name, tc.description)
			t.Logf("  Expected canonical $id: %s", expectedId)
			t.Logf("  Actual stored $id: %s", storedId)
		})
	}
}

func TestQueryCanonicalId(t *testing.T) {
	k, ms, mockTrk, ctx := setupMsgServer(t)

	authority := sdk.AccAddress([]byte("test_authority______")).String()
	operator := sdk.AccAddress([]byte("test_operator_______")).String()
	validDid := "did:example:123456789abcdefghi"

	trID := mockTrk.CreateMockTrustRegistry(authority, validDid)

	schemaNoId := `{
  "$schema": "https://json-schema.org/draft/2020-12/schema",
  "title": "TestSchema",
  "description": "Test schema",
  "type": "object",
  "properties": {
    "name": { "type": "string" }
  },
  "required": ["name"],
  "additionalProperties": false
}`

	createMsg := keeper.CreateMsgWithValidityPeriods(authority, operator, trID, schemaNoId, 365, 365, 180, 180, 180, 2, 2, 1, "tu", "sha256")
	resp, err := ms.CreateCredentialSchema(ctx, createMsg)
	require.NoError(t, err)
	require.NotNil(t, resp)

	sdkCtx := sdk.UnwrapSDKContext(ctx)

	t.Run("GetCredentialSchema returns canonical $id", func(t *testing.T) {
		queryResp, err := k.GetCredentialSchema(ctx, &types.QueryGetCredentialSchemaRequest{Id: resp.Id})
		require.NoError(t, err)
		require.NotNil(t, queryResp)

		var schemaDoc map[string]interface{}
		err = json.Unmarshal([]byte(queryResp.Schema.JsonSchema), &schemaDoc)
		require.NoError(t, err)

		storedId, ok := schemaDoc["$id"].(string)
		require.True(t, ok, "Query should return schema with $id")
		expectedId := fmt.Sprintf("vpr:verana:%s/cs/v1/js/%d", sdkCtx.ChainID(), resp.Id)
		require.Equal(t, expectedId, storedId, "Query response should have canonical $id")
	})

	t.Run("RenderJsonSchema returns canonical $id", func(t *testing.T) {
		renderResp, err := k.RenderJsonSchema(ctx, &types.QueryRenderJsonSchemaRequest{Id: resp.Id})
		require.NoError(t, err)
		require.NotNil(t, renderResp)

		var schemaDoc map[string]interface{}
		err = json.Unmarshal([]byte(renderResp.Schema), &schemaDoc)
		require.NoError(t, err)

		storedId, ok := schemaDoc["$id"].(string)
		require.True(t, ok, "Render query should return schema with $id")
		expectedId := fmt.Sprintf("vpr:verana:%s/cs/v1/js/%d", sdkCtx.ChainID(), resp.Id)
		require.Equal(t, expectedId, storedId, "Render response should have canonical $id")
	})

	t.Run("ListCredentialSchemas returns canonical $id", func(t *testing.T) {
		listResp, err := k.ListCredentialSchemas(ctx, &types.QueryListCredentialSchemasRequest{ResponseMaxSize: 64})
		require.NoError(t, err)
		require.NotNil(t, listResp)
		require.GreaterOrEqual(t, len(listResp.Schemas), 1, "Should have at least one schema")

		var found bool
		for _, schema := range listResp.Schemas {
			if schema.Id == resp.Id {
				found = true
				var schemaDoc map[string]interface{}
				err = json.Unmarshal([]byte(schema.JsonSchema), &schemaDoc)
				require.NoError(t, err)

				storedId, ok := schemaDoc["$id"].(string)
				require.True(t, ok, "List query should return schema with $id")
				expectedId := fmt.Sprintf("vpr:verana:%s/cs/v1/js/%d", sdkCtx.ChainID(), resp.Id)
				require.Equal(t, expectedId, storedId, "List response should have canonical $id")
			}
		}
		require.True(t, found, "Should find our created schema in list")
	})
}

func TestUpdateCredentialSchema(t *testing.T) {
	k, ms, mockTrk, ctx := setupMsgServer(t)

	authority := sdk.AccAddress([]byte("test_authority______")).String()
	operator := sdk.AccAddress([]byte("test_operator_______")).String()
	validDid := "did:example:123456789abcdefghi"

	trID := mockTrk.CreateMockTrustRegistry(authority, validDid)

	validJsonSchema := `{
        "$schema": "https://json-schema.org/draft/2020-12/schema",
        "$id": "/vpr/v1/cs/js/1",
        "type": "object",
        "properties": {
            "name": {
                "type": "string"
            }
        },
        "required": ["name"],
        "additionalProperties": false
    }`
	createMsg := keeper.CreateMsgWithValidityPeriods(authority, operator, trID, validJsonSchema, 365, 365, 180, 180, 180, 2, 2, 1, "tu", "sha256")

	schemaID, err := ms.CreateCredentialSchema(ctx, createMsg)
	require.NoError(t, err)

	sdkCtx := sdk.UnwrapSDKContext(ctx)
	ctx = sdk.WrapSDKContext(sdkCtx.WithBlockTime(sdkCtx.BlockTime().Add(time.Hour)))

	testCases := []struct {
		name          string
		msg           *types.MsgUpdateCredentialSchema
		expPass       bool
		errorContains string
	}{
		{
			name:    "valid update",
			msg:     keeper.CreateUpdateMsgWithValidityPeriods(authority, schemaID.Id, 365, 365, 180, 180, 180),
			expPass: true,
		},
		{
			name:          "non-existent schema",
			msg:           keeper.CreateUpdateMsgWithValidityPeriods(authority, 999, 365, 365, 180, 180, 180),
			expPass:       false,
			errorContains: "credential schema not found",
		},
		{
			name:          "unauthorized update - not controller",
			msg:           keeper.CreateUpdateMsgWithValidityPeriods("verana1unauthorized", schemaID.Id, 365, 365, 180, 180, 180),
			expPass:       false,
			errorContains: "creator is not the controller",
		},
		{
			name:          "invalid validity period - exceeds maximum",
			msg:           keeper.CreateUpdateMsgWithValidityPeriods(authority, schemaID.Id, 99999, 365, 180, 180, 180),
			expPass:       false,
			errorContains: "exceeds maximum",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			resp, err := ms.UpdateCredentialSchema(ctx, tc.msg)
			if tc.expPass {
				require.NoError(t, err)
				require.NotNil(t, resp)

				schema, err := k.CredentialSchema.Get(ctx, tc.msg.Id)
				require.NoError(t, err)
				if tc.msg.GetIssuerGrantorValidationValidityPeriod() != nil {
					require.Equal(t, tc.msg.GetIssuerGrantorValidationValidityPeriod().GetValue(), schema.IssuerGrantorValidationValidityPeriod)
				}
				if tc.msg.GetVerifierGrantorValidationValidityPeriod() != nil {
					require.Equal(t, tc.msg.GetVerifierGrantorValidationValidityPeriod().GetValue(), schema.VerifierGrantorValidationValidityPeriod)
				}
				if tc.msg.GetIssuerValidationValidityPeriod() != nil {
					require.Equal(t, tc.msg.GetIssuerValidationValidityPeriod().GetValue(), schema.IssuerValidationValidityPeriod)
				}
				if tc.msg.GetVerifierValidationValidityPeriod() != nil {
					require.Equal(t, tc.msg.GetVerifierValidationValidityPeriod().GetValue(), schema.VerifierValidationValidityPeriod)
				}
				if tc.msg.GetHolderValidationValidityPeriod() != nil {
					require.Equal(t, tc.msg.GetHolderValidationValidityPeriod().GetValue(), schema.HolderValidationValidityPeriod)
				}
				require.NotEqual(t, schema.Created, schema.Modified)
			} else {
				require.Error(t, err)
				if tc.errorContains != "" {
					require.Contains(t, err.Error(), tc.errorContains)
				}
				require.Nil(t, resp)
			}
		})
	}
}

func TestArchiveCredentialSchema(t *testing.T) {
	k, ms, mockTrk, ctx := setupMsgServer(t)

	authority := sdk.AccAddress([]byte("test_authority______")).String()
	operator := sdk.AccAddress([]byte("test_operator_______")).String()
	validDid := "did:example:123456789abcdefghi"

	trID := mockTrk.CreateMockTrustRegistry(authority, validDid)

	validJsonSchema := `{
        "$schema": "https://json-schema.org/draft/2020-12/schema",
        "$id": "/vpr/v1/cs/js/1",
        "type": "object",
        "properties": {
            "name": {
                "type": "string"
            }
        },
        "required": ["name"],
        "additionalProperties": false
    }`
	createMsg := keeper.CreateMsgWithValidityPeriods(authority, operator, trID, validJsonSchema, 365, 365, 180, 180, 180, 2, 2, 1, "tu", "sha256")

	schemaID, err := ms.CreateCredentialSchema(ctx, createMsg)
	require.NoError(t, err)

	sdkCtx := sdk.UnwrapSDKContext(ctx)
	ctx = sdk.WrapSDKContext(sdkCtx.WithBlockTime(sdkCtx.BlockTime().Add(time.Hour)))

	testCases := []struct {
		name          string
		msg           *types.MsgArchiveCredentialSchema
		setupFn       func()
		expPass       bool
		errorContains string
	}{
		{
			name: "valid archive",
			msg: &types.MsgArchiveCredentialSchema{
				Creator: authority,
				Id:      schemaID.Id,
				Archive: true,
			},
			expPass: true,
		},
		{
			name: "valid unarchive",
			msg: &types.MsgArchiveCredentialSchema{
				Creator: authority,
				Id:      schemaID.Id,
				Archive: false,
			},
			expPass: true,
		},
		{
			name: "non-existent schema",
			msg: &types.MsgArchiveCredentialSchema{
				Creator: authority,
				Id:      999,
				Archive: true,
			},
			expPass:       false,
			errorContains: "credential schema not found",
		},
		{
			name: "unauthorized archive - not controller",
			msg: &types.MsgArchiveCredentialSchema{
				Creator: "verana1unauthorized",
				Id:      schemaID.Id,
				Archive: true,
			},
			expPass:       false,
			errorContains: "only trust registry controller can archive credential schema",
		},
		{
			name: "already archived",
			msg: &types.MsgArchiveCredentialSchema{
				Creator: authority,
				Id:      schemaID.Id,
				Archive: true,
			},
			setupFn: func() {
				_, err := ms.ArchiveCredentialSchema(ctx, &types.MsgArchiveCredentialSchema{
					Creator: authority,
					Id:      schemaID.Id,
					Archive: true,
				})
				require.NoError(t, err)
			},
			expPass:       false,
			errorContains: "already archived",
		},
		{
			name: "already unarchived",
			msg: &types.MsgArchiveCredentialSchema{
				Creator: authority,
				Id:      schemaID.Id,
				Archive: false,
			},
			setupFn: func() {
				_, err := ms.ArchiveCredentialSchema(ctx, &types.MsgArchiveCredentialSchema{
					Creator: authority,
					Id:      schemaID.Id,
					Archive: false,
				})
				require.NoError(t, err)
			},
			expPass:       false,
			errorContains: "not archived",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			if tc.setupFn != nil {
				tc.setupFn()
			}

			resp, err := ms.ArchiveCredentialSchema(ctx, tc.msg)
			if tc.expPass {
				require.NoError(t, err)
				require.NotNil(t, resp)

				schema, err := k.CredentialSchema.Get(ctx, tc.msg.Id)
				require.NoError(t, err)
				if tc.msg.Archive {
					require.NotNil(t, schema.Archived)
				} else {
					require.Nil(t, schema.Archived)
				}
				require.NotEqual(t, schema.Created, schema.Modified)
			} else {
				require.Error(t, err)
				if tc.errorContains != "" {
					require.Contains(t, err.Error(), tc.errorContains)
				}
				require.Nil(t, resp)
			}
		})
	}
}
