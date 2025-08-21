package permission_test

import (
	"testing"
	"time"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/stretchr/testify/require"

	keepertest "github.com/verana-labs/verana-blockchain/testutil/keeper"
	"github.com/verana-labs/verana-blockchain/testutil/nullify"
	permission "github.com/verana-labs/verana-blockchain/x/perm/module"
	"github.com/verana-labs/verana-blockchain/x/perm/types"
)

func TestGenesis(t *testing.T) {
	// Test default genesis state
	genesisState := types.GenesisState{
		Params:             types.DefaultParams(),
		Permissions:        []types.Permission{},
		PermissionSessions: []types.PermissionSession{},
		NextPermissionId:   1,
	}

	k, _, _, ctx := keepertest.PermissionKeeper(t)
	permission.InitGenesis(ctx, k, genesisState)
	got := permission.ExportGenesis(ctx, k)
	require.NotNil(t, got)

	nullify.Fill(&genesisState)
	nullify.Fill(got)

	require.Equal(t, genesisState.Params, got.Params)
	require.ElementsMatch(t, genesisState.Permissions, got.Permissions)
	require.ElementsMatch(t, genesisState.PermissionSessions, got.PermissionSessions)
	require.Equal(t, genesisState.NextPermissionId, got.NextPermissionId)
}

func TestDeterministicGenesis(t *testing.T) {
	k, _, _, ctx := keepertest.PermissionKeeper(t)

	nowTime := time.Now()
	futureTime := nowTime.Add(24 * time.Hour)
	creatorAddr := sdk.AccAddress([]byte("test_creator")).String()

	// Create test permissions in random order
	perm2 := types.Permission{
		Id:              2,
		Type:            types.PermissionType_PERMISSION_TYPE_ISSUER,
		Did:             "did:example:67890",
		Grantee:         creatorAddr,
		Created:         &nowTime,
		CreatedBy:       creatorAddr,
		Modified:        &nowTime,
		SchemaId:        1,
		Country:         "CA",
		ValidatorPermId: 1,
		EffectiveFrom:   &nowTime,
		EffectiveUntil:  &futureTime,
	}

	perm1 := types.Permission{
		Id:             1,
		Type:           types.PermissionType_PERMISSION_TYPE_ECOSYSTEM,
		Did:            "did:example:12345",
		Grantee:        creatorAddr,
		Created:        &nowTime,
		CreatedBy:      creatorAddr,
		Modified:       &nowTime,
		SchemaId:       1,
		Country:        "US",
		EffectiveFrom:  &nowTime,
		EffectiveUntil: &futureTime,
	}

	// Insert in reverse order
	require.NoError(t, k.Permission.Set(ctx, perm2.Id, perm2))
	require.NoError(t, k.Permission.Set(ctx, perm1.Id, perm1))
	require.NoError(t, k.PermissionCounter.Set(ctx, 3))

	// Create test sessions in random order
	session2 := types.PermissionSession{
		Id:          "test-session-id-2",
		Controller:  creatorAddr,
		AgentPermId: 2,
		Created:     &nowTime,
		Modified:    &nowTime,
		Authz: []*types.SessionAuthz{
			{
				ExecutorPermId:    1,
				BeneficiaryPermId: 2,
			},
		},
	}

	session1 := types.PermissionSession{
		Id:          "test-session-id-1",
		Controller:  creatorAddr,
		AgentPermId: 1,
		Created:     &nowTime,
		Modified:    &nowTime,
		Authz: []*types.SessionAuthz{
			{
				ExecutorPermId:    1,
				BeneficiaryPermId: 2,
			},
		},
	}

	// Insert sessions in reverse order
	require.NoError(t, k.PermissionSession.Set(ctx, session2.Id, session2))
	require.NoError(t, k.PermissionSession.Set(ctx, session1.Id, session1))

	// Export genesis
	exportedGenesis1 := permission.ExportGenesis(ctx, k)

	// First export should have deterministic ordering
	require.Len(t, exportedGenesis1.Permissions, 2)
	require.Len(t, exportedGenesis1.PermissionSessions, 2)

	// Check if permissions are sorted by ID
	require.Equal(t, uint64(1), exportedGenesis1.Permissions[0].Id)
	require.Equal(t, uint64(2), exportedGenesis1.Permissions[1].Id)

	// Check if sessions are sorted by ID
	require.Equal(t, "test-session-id-1", exportedGenesis1.PermissionSessions[0].Id)
	require.Equal(t, "test-session-id-2", exportedGenesis1.PermissionSessions[1].Id)

	// Create a new keeper instance for the second test
	k2, _, _, ctx2 := keepertest.PermissionKeeper(t)

	// Insert in opposite order for the second test
	require.NoError(t, k2.Permission.Set(ctx2, perm1.Id, perm1))
	require.NoError(t, k2.Permission.Set(ctx2, perm2.Id, perm2))
	require.NoError(t, k2.PermissionCounter.Set(ctx2, 3))

	// Insert sessions in opposite order
	require.NoError(t, k2.PermissionSession.Set(ctx2, session1.Id, session1))
	require.NoError(t, k2.PermissionSession.Set(ctx2, session2.Id, session2))

	// Export genesis again
	exportedGenesis2 := permission.ExportGenesis(ctx2, k2)

	// Second export should have same deterministic ordering despite different insertion order
	require.Len(t, exportedGenesis2.Permissions, 2)
	require.Len(t, exportedGenesis2.PermissionSessions, 2)

	// Check if permissions are sorted by ID
	require.Equal(t, uint64(1), exportedGenesis2.Permissions[0].Id)
	require.Equal(t, uint64(2), exportedGenesis2.Permissions[1].Id)

	// Check if sessions are sorted by ID
	require.Equal(t, "test-session-id-1", exportedGenesis2.PermissionSessions[0].Id)
	require.Equal(t, "test-session-id-2", exportedGenesis2.PermissionSessions[1].Id)

	// The two exports should be identical despite different insertion orders
	nullify.Fill(exportedGenesis1)
	nullify.Fill(exportedGenesis2)
	require.Equal(t, exportedGenesis1, exportedGenesis2)
}

func TestGenesisImportExport(t *testing.T) {
	k, _, _, ctx := keepertest.PermissionKeeper(t)

	// Create some test data
	nowTime := time.Now()
	futureTime := nowTime.Add(24 * time.Hour)
	creatorAddr := sdk.AccAddress([]byte("test_creator")).String()

	// Create test permissions
	perm1 := types.Permission{
		Id:             1,
		Type:           types.PermissionType_PERMISSION_TYPE_ECOSYSTEM,
		Did:            "did:example:12345",
		Grantee:        creatorAddr,
		Created:        &nowTime,
		CreatedBy:      creatorAddr,
		Modified:       &nowTime,
		SchemaId:       1,
		Country:        "US",
		EffectiveFrom:  &nowTime,
		EffectiveUntil: &futureTime,
	}

	perm2 := types.Permission{
		Id:              2,
		Type:            types.PermissionType_PERMISSION_TYPE_ISSUER,
		Did:             "did:example:67890",
		Grantee:         creatorAddr,
		Created:         &nowTime,
		CreatedBy:       creatorAddr,
		Modified:        &nowTime,
		SchemaId:        1,
		Country:         "CA",
		EffectiveFrom:   &nowTime,
		EffectiveUntil:  &futureTime,
		ValidatorPermId: 1,
	}

	perm3 := types.Permission{
		Id:              3,
		Type:            types.PermissionType_PERMISSION_TYPE_VERIFIER,
		Did:             "did:example:verifier",
		Grantee:         creatorAddr,
		Created:         &nowTime,
		CreatedBy:       creatorAddr,
		Modified:        &nowTime,
		SchemaId:        1,
		Country:         "UK",
		EffectiveFrom:   &nowTime,
		EffectiveUntil:  &futureTime,
		ValidatorPermId: 1,
	}

	require.NoError(t, k.Permission.Set(ctx, perm1.Id, perm1))
	require.NoError(t, k.Permission.Set(ctx, perm2.Id, perm2))
	require.NoError(t, k.Permission.Set(ctx, perm3.Id, perm3))
	require.NoError(t, k.PermissionCounter.Set(ctx, 4))

	// Create test perm sessions
	session1 := types.PermissionSession{
		Id:          "test-session-id-1",
		Controller:  creatorAddr,
		AgentPermId: 2,
		Created:     &nowTime,
		Modified:    &nowTime,
		Authz: []*types.SessionAuthz{
			{
				ExecutorPermId:    1,
				BeneficiaryPermId: 2,
			},
		},
	}

	session2 := types.PermissionSession{
		Id:          "test-session-id-2",
		Controller:  creatorAddr,
		AgentPermId: 3,
		Created:     &nowTime,
		Modified:    &nowTime,
		Authz: []*types.SessionAuthz{
			{
				ExecutorPermId:    1,
				BeneficiaryPermId: 3,
				WalletAgentPermId: 2,
			},
		},
	}

	require.NoError(t, k.PermissionSession.Set(ctx, session1.Id, session1))
	require.NoError(t, k.PermissionSession.Set(ctx, session2.Id, session2))

	// Export genesis state
	genesisState := permission.ExportGenesis(ctx, k)

	// Verify exported data
	require.Equal(t, uint64(4), genesisState.NextPermissionId)
	require.Len(t, genesisState.Permissions, 3)
	require.Len(t, genesisState.PermissionSessions, 2)

	// Create a new keeper instance
	k2, _, _, ctx2 := keepertest.PermissionKeeper(t)

	// Initialize with the exported genesis state
	permission.InitGenesis(ctx2, k2, *genesisState)

	// Verify all data was imported correctly
	perm1Get, err := k2.Permission.Get(ctx2, 1)
	require.NoError(t, err)
	require.Equal(t, perm1.Id, perm1Get.Id)
	require.Equal(t, perm1.Did, perm1Get.Did)
	require.Equal(t, perm1.Type, perm1Get.Type)

	perm2Get, err := k2.Permission.Get(ctx2, 2)
	require.NoError(t, err)
	require.Equal(t, perm2.Id, perm2Get.Id)
	require.Equal(t, perm2.ValidatorPermId, perm2Get.ValidatorPermId)

	perm3Get, err := k2.Permission.Get(ctx2, 3)
	require.NoError(t, err)
	require.Equal(t, perm3.Id, perm3Get.Id)
	require.Equal(t, perm3.Country, perm3Get.Country)

	counter, err := k2.PermissionCounter.Get(ctx2)
	require.NoError(t, err)
	require.Equal(t, uint64(4), counter)

	session1Get, err := k2.PermissionSession.Get(ctx2, "test-session-id-1")
	require.NoError(t, err)
	require.Equal(t, session1.Id, session1Get.Id)
	require.Equal(t, session1.AgentPermId, session1Get.AgentPermId)

	session2Get, err := k2.PermissionSession.Get(ctx2, "test-session-id-2")
	require.NoError(t, err)
	require.Equal(t, session2.Id, session2Get.Id)
	require.Equal(t, session2.Authz[0].WalletAgentPermId, session2Get.Authz[0].WalletAgentPermId)

	// Export from the new keeper and verify it matches the original export
	exportedState2 := permission.ExportGenesis(ctx2, k2)

	// Both states should be identical
	nullify.Fill(genesisState)
	nullify.Fill(exportedState2)
	require.Equal(t, genesisState, exportedState2)
}

func TestGenesisValidation(t *testing.T) {
	nowTime := time.Now()
	creatorAddr := sdk.AccAddress([]byte("test_creator")).String()

	testCases := []struct {
		name         string
		genesisState types.GenesisState
		expectedErr  string
	}{
		{
			name: "duplicate perm IDs",
			genesisState: types.GenesisState{
				Params: types.DefaultParams(),
				Permissions: []types.Permission{
					{
						Id:       1,
						Type:     types.PermissionType_PERMISSION_TYPE_ISSUER,
						Grantee:  creatorAddr,
						Created:  &nowTime,
						Modified: &nowTime,
					},
					{
						Id:       1, // Duplicate ID
						Type:     types.PermissionType_PERMISSION_TYPE_VERIFIER,
						Grantee:  creatorAddr,
						Created:  &nowTime,
						Modified: &nowTime,
					},
				},
				NextPermissionId: 2,
			},
			expectedErr: "duplicate perm ID found: 1",
		},
		{
			name: "next perm ID too low",
			genesisState: types.GenesisState{
				Params: types.DefaultParams(),
				Permissions: []types.Permission{
					{
						Id:       5,
						Type:     types.PermissionType_PERMISSION_TYPE_ISSUER,
						Grantee:  creatorAddr,
						Created:  &nowTime,
						Modified: &nowTime,
					},
				},
				NextPermissionId: 3, // Should be > 5
			},
			expectedErr: "next_permission_id (3) must be greater than the maximum perm ID (5)",
		},
		{
			name: "missing required perm field",
			genesisState: types.GenesisState{
				Params: types.DefaultParams(),
				Permissions: []types.Permission{
					{
						Id:   1,
						Type: types.PermissionType_PERMISSION_TYPE_ISSUER,
						// Missing Grantee field
						Created:  &nowTime,
						Modified: &nowTime,
					},
				},
				NextPermissionId: 2,
			},
			expectedErr: "grantee cannot be empty for perm ID 1",
		},
		{
			name: "invalid validator reference",
			genesisState: types.GenesisState{
				Params: types.DefaultParams(),
				Permissions: []types.Permission{
					{
						Id:              2,
						Type:            types.PermissionType_PERMISSION_TYPE_ISSUER,
						Grantee:         creatorAddr,
						Created:         &nowTime,
						Modified:        &nowTime,
						ValidatorPermId: 999, // Non-existent validator
					},
				},
				NextPermissionId: 3,
			},
			expectedErr: "validator perm ID 999 not found for perm ID 2",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// Validate the genesis state
			err := tc.genesisState.Validate()
			require.Error(t, err)
			require.Contains(t, err.Error(), tc.expectedErr)
		})
	}
}
