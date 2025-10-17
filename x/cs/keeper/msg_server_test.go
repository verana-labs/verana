package keeper_test

import (
	"context"
	"encoding/json"
	"testing"

	"time"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/stretchr/testify/require"

	keepertest "github.com/verana-labs/verana/testutil/keeper"
	"github.com/verana-labs/verana/x/cs/keeper"
	"github.com/verana-labs/verana/x/cs/types"
)

func setupMsgServer(t testing.TB) (keeper.Keeper, types.MsgServer, *keepertest.MockTrustRegistryKeeper, context.Context) {
	k, mockTrk, ctx := keepertest.CredentialschemaKeeper(t)
	return k, keeper.NewMsgServerImpl(k), mockTrk, ctx
}

func TestMsgServerCreateCredentialSchema(t *testing.T) {
	_, ms, mockTrk, ctx := setupMsgServer(t)

	creator := sdk.AccAddress([]byte("test_creator")).String()
	validDid := "did:example:123456789abcdefghi"

	// First create a trust registry
	trID := mockTrk.CreateMockTrustRegistry(creator, validDid)

	validJsonSchema := `{
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

	// Test basic JSON parsing
	var schemaDoc map[string]interface{}
	err := json.Unmarshal([]byte(validJsonSchema), &schemaDoc)
	require.NoError(t, err, "JSON should be valid")

	// Test the meta-schema JSON parsing
	var metaDoc map[string]interface{}
	err = json.Unmarshal([]byte(types.JsonSchemaMetaSchema), &metaDoc)
	require.NoError(t, err, "Meta-schema JSON should be valid")

	testCases := []struct {
		name    string
		msg     *types.MsgCreateCredentialSchema
		isValid bool
	}{
		{
			name:    "Valid Create Credential Schema",
			msg:     keeper.CreateMsgWithValidityPeriods(creator, trID, validJsonSchema, 365, 365, 180, 180, 180, 2, 2),
			isValid: true,
		},
		{
			name:    "Non-existent Trust Registry",
			msg:     keeper.CreateMsgWithValidityPeriods(creator, 999, validJsonSchema, 365, 365, 180, 180, 180, 2, 2),
			isValid: false,
		},
		{
			name:    "Wrong Trust Registry Controller",
			msg:     keeper.CreateMsgWithValidityPeriods(sdk.AccAddress([]byte("wrong_creator")).String(), trID, validJsonSchema, 365, 365, 180, 180, 180, 2, 2),
			isValid: false,
		},
	}

	//var expectedID uint64 = 1 // Track expected auto-generated ID

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
				// ... rest of assertions
			} else {
				// For invalid cases, check if it fails in ValidateBasic OR message server
				resp, msgServerErr := ms.CreateCredentialSchema(ctx, tc.msg)

				// Should fail in either ValidateBasic OR message server
				if err == nil {
					// If ValidateBasic passes, message server should fail
					require.Error(t, msgServerErr)
				}
				require.Nil(t, resp)
			}
		})
	}
}

func TestUpdateCredentialSchema(t *testing.T) {
	k, ms, mockTrk, ctx := setupMsgServer(t)

	creator := sdk.AccAddress([]byte("test_creator")).String()
	validDid := "did:example:123456789abcdefghi"

	// First create a trust registry
	trID := mockTrk.CreateMockTrustRegistry(creator, validDid)

	// Create a valid credential schema
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
	createMsg := &types.MsgCreateCredentialSchema{
		Creator:    creator,
		TrId:       trID,
		JsonSchema: validJsonSchema,
	}

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
			msg:     keeper.CreateUpdateMsgWithValidityPeriods(creator, schemaID.Id, 365, 365, 180, 180, 180),
			expPass: true,
		},
		{
			name:          "non-existent schema",
			msg:           keeper.CreateUpdateMsgWithValidityPeriods(creator, 999, 365, 365, 180, 180, 180),
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
			msg:           keeper.CreateUpdateMsgWithValidityPeriods(creator, schemaID.Id, 99999, 365, 180, 180, 180),
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

				// Verify changes
				schema, err := k.CredentialSchema.Get(ctx, tc.msg.Id)
				require.NoError(t, err)
				require.Equal(t, tc.msg.GetIssuerGrantorValidationValidityPeriod(), schema.IssuerGrantorValidationValidityPeriod)
				require.Equal(t, tc.msg.GetVerifierGrantorValidationValidityPeriod(), schema.VerifierGrantorValidationValidityPeriod)
				require.Equal(t, tc.msg.GetIssuerValidationValidityPeriod(), schema.IssuerValidationValidityPeriod)
				require.Equal(t, tc.msg.GetVerifierValidationValidityPeriod(), schema.VerifierValidationValidityPeriod)
				require.Equal(t, tc.msg.GetHolderValidationValidityPeriod(), schema.HolderValidationValidityPeriod)
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

	creator := sdk.AccAddress([]byte("test_creator")).String()
	validDid := "did:example:123456789abcdefghi"

	// First create a trust registry
	trID := mockTrk.CreateMockTrustRegistry(creator, validDid)

	// Create a valid credential schema
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
	createMsg := &types.MsgCreateCredentialSchema{
		Creator:    creator,
		TrId:       trID,
		JsonSchema: validJsonSchema,
	}

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
				Creator: creator,
				Id:      schemaID.Id,
				Archive: true,
			},
			expPass: true,
		},
		{
			name: "valid unarchive",
			msg: &types.MsgArchiveCredentialSchema{
				Creator: creator,
				Id:      schemaID.Id,
				Archive: false,
			},
			expPass: true,
		},
		{
			name: "non-existent schema",
			msg: &types.MsgArchiveCredentialSchema{
				Creator: creator,
				Id:      999, // Non-existent schema ID
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
				Creator: creator,
				Id:      schemaID.Id,
				Archive: true,
			},
			setupFn: func() {
				// Archive first
				_, err := ms.ArchiveCredentialSchema(ctx, &types.MsgArchiveCredentialSchema{
					Creator: creator,
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
				Creator: creator,
				Id:      schemaID.Id,
				Archive: false,
			},
			setupFn: func() {
				// Unarchive first
				_, err := ms.ArchiveCredentialSchema(ctx, &types.MsgArchiveCredentialSchema{
					Creator: creator,
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

				// Verify changes
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
