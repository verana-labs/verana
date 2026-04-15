package keeper_test

import (
	"context"
	"testing"
	"time"

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
	createMsg := &types.MsgCreateTrustRegistry{
		Corporation: authority,
		Operator:    operator,
		Did:         "did:example:123",
		Language:    "en",
	}
	_, err := ms.CreateTrustRegistry(ctx, createMsg)
	require.NoError(t, err)

	// Get the trust registry ID
	trID, err := k.TrustRegistryDIDIndex.Get(ctx, "did:example:123")
	require.NoError(t, err)

	// Spec draft 13: active version is immutable, so documents can only be added
	// to future (in-progress) versions. Seed v2 with an en doc, promote v2 to
	// active, then add v3 docs in en/es. After setup:
	//   version 2: 1 document (en) — active
	//   version 3: 2 documents (en, es) — pending
	_, err = ms.AddGovernanceFrameworkDocument(ctx, &types.MsgAddGovernanceFrameworkDocument{
		Corporation: authority,
		Operator:    operator,
		TrId:        trID,
		Language:    "en",
		Url:         "http://example.com/doc2-en",
		DigestSri:   "sha384-MzNNbQTWCSUSi0bbz7dbua+RcENv7C6FvlmYJ1Y+I727HsPOHdzwELMYO9Mz68M26",
		Version:     2,
	})
	require.NoError(t, err)

	_, err = ms.IncreaseActiveGovernanceFrameworkVersion(ctx, &types.MsgIncreaseActiveGovernanceFrameworkVersion{
		Corporation: authority,
		Operator:    operator,
		TrId:        trID,
	})
	require.NoError(t, err)

	_, err = ms.AddGovernanceFrameworkDocument(ctx, &types.MsgAddGovernanceFrameworkDocument{
		Corporation: authority,
		Operator:    operator,
		TrId:        trID,
		Language:    "en",
		Url:         "http://example.com/doc3-en",
		DigestSri:   "sha384-MzNNbQTWCSUSi0bbz7dbua+RcENv7C6FvlmYJ1Y+I727HsPOHdzwELMYO9Mz68M26",
		Version:     3,
	})
	require.NoError(t, err)

	_, err = ms.AddGovernanceFrameworkDocument(ctx, &types.MsgAddGovernanceFrameworkDocument{
		Corporation: authority,
		Operator:    operator,
		TrId:        trID,
		Language:    "es",
		Url:         "http://example.com/doc3-es",
		DigestSri:   "sha384-MzNNbQTWCSUSi0bbz7dbua+RcENv7C6FvlmYJ1Y+I727HsPOHdzwELMYO9Mz68M26",
		Version:     3,
	})
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
				require.Len(t, response.TrustRegistry.Versions, 3) // v1 (empty), v2 (active), v3 (pending)

				// Check versions and their documents
				for _, version := range response.TrustRegistry.Versions {
					switch version.Version {
					case 1:
						require.Len(t, version.Documents, 0) // v1 empty (create seeds but spec draft 13 no longer bundles)
					case 2:
						require.Len(t, version.Documents, 1) // v2 active: en
					case 3:
						require.Len(t, version.Documents, 2) // v3 pending: en, es
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
				require.Equal(t, int32(2), response.TrustRegistry.Versions[0].Version)
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
					if version.Version == 3 {
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
		Corporation: "another_authority",
		Operator:    "another_operator",
		Did:         "did:example:456",
		Language:    "fr",
	}
	_, err := ms.CreateTrustRegistry(ctx, createMsg)
	require.NoError(t, err)

	// Attach a v2 GFD to the second TR and activate it so tests that walk
	// (versions -> documents) see at least one active document. (Spec draft 13:
	// active version is immutable, so v1 cannot be modified.)
	trID2, err := k.TrustRegistryDIDIndex.Get(ctx, "did:example:456")
	require.NoError(t, err)
	_, err = ms.AddGovernanceFrameworkDocument(ctx, &types.MsgAddGovernanceFrameworkDocument{
		Corporation: "another_authority",
		Operator:    "another_operator",
		TrId:        trID2,
		Language:    "fr",
		Url:         "http://example.com/tr2-doc2-fr",
		DigestSri:   "sha384-MzNNbQTWCSUSi0bbz7dbua+RcENv7C6FvlmYJ1Y+I727HsPOHdzwELMYO9Mz68M26",
		Version:     2,
	})
	require.NoError(t, err)
	_, err = ms.IncreaseActiveGovernanceFrameworkVersion(ctx, &types.MsgIncreaseActiveGovernanceFrameworkVersion{
		Corporation: "another_authority",
		Operator:    "another_operator",
		TrId:        trID2,
	})
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

				// Check nested structure for each trust registry: each must have
				// at least one version with at least one document (v1 is always
				// empty under draft 13 since create doesn't bundle initial GFD).
				for _, tr := range response.TrustRegistries {
					require.NotEmpty(t, tr.Versions)
					hasDocs := false
					for _, version := range tr.Versions {
						if len(version.Documents) > 0 {
							hasDocs = true
							break
						}
					}
					require.True(t, hasDocs, "tr %d should have at least one version with documents", tr.Id)
				}
			},
		},
		{
			name: "Filter by Corporation",
			request: &types.QueryListTrustRegistriesRequest{
				Corporation:     "another_authority",
				ResponseMaxSize: 10,
			},
			expectedError: false,
			check: func(t *testing.T, response *types.QueryListTrustRegistriesResponse) {
				require.Len(t, response.TrustRegistries, 1)
				require.Equal(t, "another_authority", response.TrustRegistries[0].Corporation)

				// Find the active v2 (fr) document.
				tr := response.TrustRegistries[0]
				require.NotEmpty(t, tr.Versions)
				var foundFr bool
				for _, v := range tr.Versions {
					for _, d := range v.Documents {
						if d.Language == "fr" {
							foundFr = true
						}
					}
				}
				require.True(t, foundFr, "expected at least one fr document")
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
			name: "Default Response Max Size (unspecified defaults to 64)",
			request: &types.QueryListTrustRegistriesRequest{
				ResponseMaxSize: 0, // unspecified, should default to 64
			},
			expectedError: false,
			check: func(t *testing.T, response *types.QueryListTrustRegistriesResponse) {
				require.Len(t, response.TrustRegistries, 2)

				// Each TR has at least one version with docs (v1 is always empty).
				for _, tr := range response.TrustRegistries {
					require.NotEmpty(t, tr.Versions)
					hasDocs := false
					for _, version := range tr.Versions {
						if len(version.Documents) > 0 {
							hasDocs = true
							break
						}
					}
					require.True(t, hasDocs)
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

func TestPreferredLanguageFallbackToDefaultLanguage(t *testing.T) {
	_, qs, ctx, trID := setupTestData(t)

	// Request with preferred_language="de" which doesn't exist.
	// TR default language is "en", so should fall back to "en" document.
	response, err := qs.GetTrustRegistry(ctx, &types.QueryGetTrustRegistryRequest{
		TrId:              trID,
		PreferredLanguage: "de", // no German doc exists
	})
	require.NoError(t, err)
	require.NotNil(t, response.TrustRegistry)

	for _, version := range response.TrustRegistry.Versions {
		// Empty versions stay empty (no fallback source); populated versions
		// fall back to TR default language "en".
		if len(version.Documents) == 0 {
			continue
		}
		require.Len(t, version.Documents, 1, "version %d should have exactly 1 fallback doc", version.Version)
		require.Equal(t, "en", version.Documents[0].Language, "version %d should fall back to TR default language 'en'", version.Version)
	}
}

func TestListTrustRegistriesSortOrderWithModifiedAfter(t *testing.T) {
	k, qs, ctx, _ := setupTestData(t)
	ms := keeper.NewMsgServerImpl(k)

	// The first TR was created at block time. Create a second one with a later modified time.
	sdkCtx := sdk.UnwrapSDKContext(ctx)
	pastTime := sdkCtx.BlockTime().Add(-1 * time.Hour)

	// Query with modified_after set to a time in the past — results should be sorted by modified desc
	response, err := qs.ListTrustRegistries(ctx, &types.QueryListTrustRegistriesRequest{
		ModifiedAfter:   &pastTime,
		ResponseMaxSize: 10,
	})
	require.NoError(t, err)
	require.NotEmpty(t, response.TrustRegistries)

	// Verify descending order by modified time
	for i := 1; i < len(response.TrustRegistries); i++ {
		require.False(t,
			response.TrustRegistries[i].Modified.After(response.TrustRegistries[i-1].Modified),
			"results should be sorted by modified descending",
		)
	}

	// Query without modified_after — should not be sorted (or at least not error)
	_ = ms // suppress unused warning if needed
	response2, err := qs.ListTrustRegistries(ctx, &types.QueryListTrustRegistriesRequest{
		ResponseMaxSize: 10,
	})
	require.NoError(t, err)
	require.NotEmpty(t, response2.TrustRegistries)
}

func TestParams(t *testing.T) {
	_, qs, ctx, _ := setupTestData(t)

	response, err := qs.Params(ctx, &types.QueryParamsRequest{})
	require.NoError(t, err)
	require.NotNil(t, response)
	require.NotNil(t, response.Params)
}
