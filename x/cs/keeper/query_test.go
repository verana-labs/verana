package keeper_test

import (
	_ "context"
	"testing"
	"time"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/stretchr/testify/require"

	keepertest "github.com/verana-labs/verana-blockchain/testutil/keeper"
	"github.com/verana-labs/verana-blockchain/x/cs/types"
)

func TestQueries(t *testing.T) {
	k, _, ctx := keepertest.CredentialschemaKeeper(t)

	validJsonSchema := `{
        "$schema": "https://json-schema.org/draft/2020-12/schema",
        "$id": "/dtr/v1/cs/js/1",
        "type": "object",
        "$defs": {},
        "properties": {
            "name": {
                "type": "string"
            }
        },
        "required": ["name"],
        "additionalProperties": false
    }`

	// Create test schemas
	schema1 := types.CredentialSchema{
		Id:                                      1,
		TrId:                                    1,
		Created:                                 time.Now().Add(-24 * time.Hour),
		Modified:                                time.Now().Add(-24 * time.Hour),
		JsonSchema:                              validJsonSchema,
		IssuerGrantorValidationValidityPeriod:   365,
		VerifierGrantorValidationValidityPeriod: 365,
		IssuerValidationValidityPeriod:          180,
		VerifierValidationValidityPeriod:        180,
		HolderValidationValidityPeriod:          180,
		IssuerPermManagementMode:                2,
		VerifierPermManagementMode:              2,
	}

	schema2 := types.CredentialSchema{
		Id:                                      2,
		TrId:                                    1,
		Created:                                 time.Now(),
		Modified:                                time.Now(),
		JsonSchema:                              validJsonSchema,
		IssuerGrantorValidationValidityPeriod:   365,
		VerifierGrantorValidationValidityPeriod: 365,
		IssuerValidationValidityPeriod:          180,
		VerifierValidationValidityPeriod:        180,
		HolderValidationValidityPeriod:          180,
		IssuerPermManagementMode:                2,
		VerifierPermManagementMode:              2,
	}
	modifiedAfterTime := schema1.Created.Add(time.Hour)
	require.NoError(t, k.CredentialSchema.Set(ctx, schema1.Id, schema1))
	require.NoError(t, k.CredentialSchema.Set(ctx, schema2.Id, schema2))

	t.Run("ListCredentialSchemas", func(t *testing.T) {
		testCases := []struct {
			name          string
			request       *types.QueryListCredentialSchemasRequest
			expectedCount int
			expectErr     bool
		}{
			{
				name: "List All",
				request: &types.QueryListCredentialSchemasRequest{
					ResponseMaxSize: 64,
				},
				expectedCount: 2,
				expectErr:     false,
			},
			{
				name: "Filter By Trust Registry",
				request: &types.QueryListCredentialSchemasRequest{
					TrId:            1,
					ResponseMaxSize: 64,
				},
				expectedCount: 2,
				expectErr:     false,
			},
			{
				name: "Filter By Modified After",
				request: &types.QueryListCredentialSchemasRequest{
					ModifiedAfter:   &modifiedAfterTime,
					ResponseMaxSize: 64,
				},
				expectedCount: 1,
				expectErr:     false,
			},
			{
				name: "Invalid Max Size",
				request: &types.QueryListCredentialSchemasRequest{
					ResponseMaxSize: 2000, // > 1024
				},
				expectedCount: 0,
				expectErr:     true,
			},
		}

		for _, tc := range testCases {
			t.Run(tc.name, func(t *testing.T) {
				resp, err := k.ListCredentialSchemas(sdk.WrapSDKContext(ctx), tc.request)
				if tc.expectErr {
					require.Error(t, err)
					return
				}
				require.NoError(t, err)
				require.NotNil(t, resp)
				require.Len(t, resp.Schemas, tc.expectedCount)
			})
		}
	})

	t.Run("GetCredentialSchema", func(t *testing.T) {
		testCases := []struct {
			name      string
			request   *types.QueryGetCredentialSchemaRequest
			expectErr bool
		}{
			{
				name: "Get Existing Schema",
				request: &types.QueryGetCredentialSchemaRequest{
					Id: 1,
				},
				expectErr: false,
			},
			{
				name: "Get Non-existent Schema",
				request: &types.QueryGetCredentialSchemaRequest{
					Id: 999,
				},
				expectErr: true,
			},
		}

		for _, tc := range testCases {
			t.Run(tc.name, func(t *testing.T) {
				resp, err := k.GetCredentialSchema(sdk.WrapSDKContext(ctx), tc.request)
				if tc.expectErr {
					require.Error(t, err)
					return
				}
				require.NoError(t, err)
				require.NotNil(t, resp)
				require.Equal(t, tc.request.Id, resp.Schema.Id)
			})
		}
	})

	t.Run("RenderJsonSchema", func(t *testing.T) {
		testCases := []struct {
			name      string
			request   *types.QueryRenderJsonSchemaRequest
			expectErr bool
		}{
			{
				name: "Render Existing Schema",
				request: &types.QueryRenderJsonSchemaRequest{
					Id: 1,
				},
				expectErr: false,
			},
			{
				name: "Render Non-existent Schema",
				request: &types.QueryRenderJsonSchemaRequest{
					Id: 999,
				},
				expectErr: true,
			},
		}

		for _, tc := range testCases {
			t.Run(tc.name, func(t *testing.T) {
				resp, err := k.RenderJsonSchema(sdk.WrapSDKContext(ctx), tc.request)
				if tc.expectErr {
					require.Error(t, err)
					return
				}
				require.NoError(t, err)
				require.NotNil(t, resp)
				require.Equal(t, validJsonSchema, resp.Schema)
			})
		}
	})
}
