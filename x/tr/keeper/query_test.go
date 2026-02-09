package keeper_test

import (
	"context"
	"testing"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/stretchr/testify/require"
	keepertest "github.com/verana-labs/verana/testutil/keeper"
	"github.com/verana-labs/verana/x/tr/keeper"
	"github.com/verana-labs/verana/x/tr/types"
)

func setupTestData(t *testing.T) (keeper.Keeper, types.QueryServer, context.Context, uint64) {
	k, ctx := keepertest.TrustregistryKeeper(t)
	qs := keeper.NewQueryServerImpl(k)
	ms := keeper.NewMsgServerImpl(k)

	// Create a trust registry
	authority := sdk.AccAddress([]byte("test_authority")).String()
	operator := sdk.AccAddress([]byte("test_operator")).String()
	creator := authority // For backward compatibility in test, creator refers to authority
	createMsg := &types.MsgCreateTrustRegistry{
		Authority:    authority,
		Operator:     operator,
		Did:          "did:example:123",
		Language:     "en",
		DocUrl:       "http://example.com/doc1",
		DocDigestSri: "sha384-MzNNbQTWCSUSi0bbz7dbua+RcENv7C6FvlmYJ1Y+I727HsPOHdzwELMYO9Mz68M26",
	}
	_, err := ms.CreateTrustRegistry(ctx, createMsg)
	require.NoError(t, err)

	// Get the trust registry ID
	trID, err := k.TrustRegistryDIDIndex.Get(ctx, "did:example:123")
	require.NoError(t, err)

	// Add documents in different languages for version 2
	addDocMsg := &types.MsgAddGovernanceFrameworkDocument{
		Creator:      creator,
		Id:           trID,
		DocLanguage:  "en",
		DocUrl:       "http://example.com/doc2-en",
		DocDigestSri: "sha384-MzNNbQTWCSUSi0bbz7dbua+RcENv7C6FvlmYJ1Y+I727HsPOHdzwELMYO9Mz68M26",
		Version:      2,
	}
	_, err = ms.AddGovernanceFrameworkDocument(ctx, addDocMsg)
	require.NoError(t, err)

	addDocMsg.DocLanguage = "es"
	addDocMsg.DocUrl = "http://example.com/doc2-es"
	_, err = ms.AddGovernanceFrameworkDocument(ctx, addDocMsg)
	require.NoError(t, err)

	return k, qs, ctx, trID
}

func TestGetTrustRegistry(t *testing.T) {
	_, qs, ctx, trID := setupTestData(t)

	testCases := []struct {
		name          string
		request       *types.QueryGetTrustRegistryRequest
		expectedError bool
		check         func(*testing.T, *types.QueryGetTrustRegistryResponse)
	}{
		{
			name: "Valid Request - All Documents",
			request: &types.QueryGetTrustRegistryRequest{
				TrId:         trID,
				ActiveGfOnly: false,
			},
			expectedError: false,
			check: func(t *testing.T, response *types.QueryGetTrustRegistryResponse) {
				require.NotNil(t, response.TrustRegistry)
				require.Equal(t, trID, response.TrustRegistry.Id)
				require.Len(t, response.TrustRegistry.Versions, 2) // Version 1 and 2

				// Check versions and their documents
				for _, version := range response.TrustRegistry.Versions {
					if version.Version == 1 {
						require.Len(t, version.Documents, 1) // Version 1 has 1 document
					} else if version.Version == 2 {
						require.Len(t, version.Documents, 2) // Version 2 has 2 documents (en, es)
					}
				}
			},
		},
		{
			name: "Valid Request - Active Only",
			request: &types.QueryGetTrustRegistryRequest{
				TrId:         trID,
				ActiveGfOnly: true,
			},
			expectedError: false,
			check: func(t *testing.T, response *types.QueryGetTrustRegistryResponse) {
				require.NotNil(t, response.TrustRegistry)
				require.Len(t, response.TrustRegistry.Versions, 1)
				require.Equal(t, int32(1), response.TrustRegistry.Versions[0].Version)
				require.Len(t, response.TrustRegistry.Versions[0].Documents, 1)
			},
		},
		{
			name: "Valid Request - Preferred Language",
			request: &types.QueryGetTrustRegistryRequest{
				TrId:              trID,
				PreferredLanguage: "es",
			},
			expectedError: false,
			check: func(t *testing.T, response *types.QueryGetTrustRegistryResponse) {
				require.NotNil(t, response.TrustRegistry)
				for _, version := range response.TrustRegistry.Versions {
					if version.Version == 2 {
						require.Len(t, version.Documents, 1) // Should only have Spanish document
						require.Equal(t, "es", version.Documents[0].Language)
					}
				}
			},
		},
		{
			name: "Invalid Trust Registry ID",
			request: &types.QueryGetTrustRegistryRequest{
				TrId: 99999,
			},
			expectedError: true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			response, err := qs.GetTrustRegistry(ctx, tc.request)
			if tc.expectedError {
				require.Error(t, err)
				return
			}
			require.NoError(t, err)
			require.NotNil(t, response)
			if tc.check != nil {
				tc.check(t, response)
			}
		})
	}
}

func TestListTrustRegistries(t *testing.T) {
	k, qs, ctx, _ := setupTestData(t)

	// Create additional trust registry for testing
	ms := keeper.NewMsgServerImpl(k)
	createMsg := &types.MsgCreateTrustRegistry{
		Authority:    "another_authority",
		Operator:     "another_operator",
		Did:          "did:example:456",
		Language:     "fr",
		DocUrl:       "http://example.com/doc-fr",
		DocDigestSri: "sha384-MzNNbQTWCSUSi0bbz7dbua+RcENv7C6FvlmYJ1Y+I727HsPOHdzwELMYO9Mz68M26",
	}
	_, err := ms.CreateTrustRegistry(ctx, createMsg)
	require.NoError(t, err)

	testCases := []struct {
		name          string
		request       *types.QueryListTrustRegistriesRequest
		expectedError bool
		check         func(*testing.T, *types.QueryListTrustRegistriesResponse)
	}{
		{
			name: "List All",
			request: &types.QueryListTrustRegistriesRequest{
				ResponseMaxSize: 10,
			},
			expectedError: false,
			check: func(t *testing.T, response *types.QueryListTrustRegistriesResponse) {
				require.Len(t, response.TrustRegistries, 2)

				// Check nested structure for each trust registry
				for _, tr := range response.TrustRegistries {
					require.NotEmpty(t, tr.Versions)
					for _, version := range tr.Versions {
						require.NotEmpty(t, version.Documents)
					}
				}
			},
		},
		{
			name: "Filter by Controller",
			request: &types.QueryListTrustRegistriesRequest{
				Controller:      "another_authority",
				ResponseMaxSize: 10,
			},
			expectedError: false,
			check: func(t *testing.T, response *types.QueryListTrustRegistriesResponse) {
				require.Len(t, response.TrustRegistries, 1)
				require.Equal(t, "another_authority", response.TrustRegistries[0].Controller)

				// Check nested structure
				tr := response.TrustRegistries[0]
				require.NotEmpty(t, tr.Versions)
				require.Len(t, tr.Versions[0].Documents, 1) // Should have one document
				require.Equal(t, "fr", tr.Versions[0].Documents[0].Language)
			},
		},
		{
			name: "Invalid Response Max Size",
			request: &types.QueryListTrustRegistriesRequest{
				ResponseMaxSize: 1025, // More than maximum allowed
			},
			expectedError: true,
		},
		{
			name: "Default Response Max Size",
			request: &types.QueryListTrustRegistriesRequest{
				ResponseMaxSize: 10,
			},
			expectedError: false,
			check: func(t *testing.T, response *types.QueryListTrustRegistriesResponse) {
				require.Len(t, response.TrustRegistries, 2)

				// Verify nested structure exists for all trust registries
				for _, tr := range response.TrustRegistries {
					require.NotEmpty(t, tr.Versions)
					for _, version := range tr.Versions {
						require.NotEmpty(t, version.Documents)
					}
				}
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			response, err := qs.ListTrustRegistries(ctx, tc.request)
			if tc.expectedError {
				require.Error(t, err)
				return
			}
			require.NoError(t, err)
			require.NotNil(t, response)
			if tc.check != nil {
				tc.check(t, response)
			}
		})
	}
}

func TestParams(t *testing.T) {
	_, qs, ctx, _ := setupTestData(t)

	response, err := qs.Params(ctx, &types.QueryParamsRequest{})
	require.NoError(t, err)
	require.NotNil(t, response)
	require.NotNil(t, response.Params)
}
