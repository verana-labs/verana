package diddirectory_test

import (
	"fmt"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	keepertest "github.com/verana-labs/verana-blockchain/testutil/keeper"
	"github.com/verana-labs/verana-blockchain/testutil/nullify"
	diddirectory "github.com/verana-labs/verana-blockchain/x/dd/module"
	"github.com/verana-labs/verana-blockchain/x/dd/types"
)

func TestGenesis(t *testing.T) {
	// Setup basic test parameters
	_, _ = keepertest.DiddirectoryKeeper(t)
	now := time.Now().UTC()
	oneYearLater := now.AddDate(1, 0, 0)

	testCases := []struct {
		name        string
		setupState  func(*types.GenesisState)
		verifyState func(*testing.T, *types.GenesisState)
	}{
		{
			name: "default genesis state",
			setupState: func(gs *types.GenesisState) {
				// Use default genesis state
			},
			verifyState: func(t *testing.T, exported *types.GenesisState) {
				require.Equal(t, types.DefaultParams(), exported.Params)
				require.Empty(t, exported.DidDirectories)
			},
		},
		{
			name: "custom genesis with DIDs",
			setupState: func(gs *types.GenesisState) {
				// Add DIDs in a non-deterministic order to test determinism
				gs.DidDirectories = []types.DIDDirectory{
					{
						Did:        "did:example:zzz",
						Controller: "cosmos1controller1",
						Created:    now,
						Modified:   now,
						Exp:        oneYearLater,
						Deposit:    5000,
					},
					{
						Did:        "did:example:aaa",
						Controller: "cosmos1controller2",
						Created:    now,
						Modified:   now,
						Exp:        oneYearLater,
						Deposit:    6000,
					},
				}
			},
			verifyState: func(t *testing.T, exported *types.GenesisState) {
				// Verify DIDs are exported in deterministic order (alphabetical by DID)
				require.Len(t, exported.DidDirectories, 2)
				require.Equal(t, "did:example:aaa", exported.DidDirectories[0].Did)
				require.Equal(t, "did:example:zzz", exported.DidDirectories[1].Did)

				// Verify all fields are preserved
				require.Equal(t, "cosmos1controller2", exported.DidDirectories[0].Controller)
				require.Equal(t, int64(6000), exported.DidDirectories[0].Deposit)
				require.Equal(t, "cosmos1controller1", exported.DidDirectories[1].Controller)
				require.Equal(t, int64(5000), exported.DidDirectories[1].Deposit)
			},
		},
		{
			name: "many DIDs with varied attributes",
			setupState: func(gs *types.GenesisState) {
				// Create many DIDs with different attributes
				gs.DidDirectories = []types.DIDDirectory{}

				// Generate 10 DIDs with different attributes
				for i := 0; i < 10; i++ {
					didNum := 10 - i // Reverse order to test sorting
					gs.DidDirectories = append(gs.DidDirectories, types.DIDDirectory{
						Did:        fmt.Sprintf("did:example:%03d", didNum),
						Controller: fmt.Sprintf("cosmos1controller%d", didNum),
						Created:    now.AddDate(0, 0, -i), // Different creation times
						Modified:   now.AddDate(0, 0, -i),
						Exp:        oneYearLater.AddDate(0, i, 0), // Different expiration times
						Deposit:    int64(1000 * (i + 1)),         // Different deposits
					})
				}
			},
			verifyState: func(t *testing.T, exported *types.GenesisState) {
				// Verify DIDs are exported in deterministic order
				require.Len(t, exported.DidDirectories, 10)

				// Check they're sorted by DID
				for i := 0; i < 9; i++ {
					require.True(t, exported.DidDirectories[i].Did < exported.DidDirectories[i+1].Did)
				}

				// Spot check first and last entries
				require.Equal(t, "did:example:001", exported.DidDirectories[0].Did)
				require.Equal(t, "did:example:010", exported.DidDirectories[9].Did)
			},
		},
		{
			name: "changing params",
			setupState: func(gs *types.GenesisState) {
				// Set custom params
				gs.Params = types.Params{
					DidDirectoryTrustDeposit: 10,
					DidDirectoryGracePeriod:  60,
				}
			},
			verifyState: func(t *testing.T, exported *types.GenesisState) {
				// Verify params are preserved
				require.Equal(t, uint64(10), exported.Params.DidDirectoryTrustDeposit)
				require.Equal(t, uint64(60), exported.Params.DidDirectoryGracePeriod)
			},
		},
	}

	// Run all test cases
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// Create a clean keeper for each test
			k, ctx := keepertest.DiddirectoryKeeper(t)

			// Setup genesis state
			genesisState := types.DefaultGenesis()
			tc.setupState(genesisState)

			// Initialize genesis
			diddirectory.InitGenesis(ctx, k, *genesisState)

			// Export genesis
			exported := diddirectory.ExportGenesis(ctx, k)

			// Verify exported state
			tc.verifyState(t, exported)
		})
	}
}

// TestGenesisPanics checks that InitGenesis panics properly when invalid data is provided
func TestGenesisPanics(t *testing.T) {
	k, ctx := keepertest.DiddirectoryKeeper(t)

	// Test with duplicate DIDs - should panic
	duplicateDIDState := types.DefaultGenesis()
	now := time.Now().UTC()
	oneYearLater := now.AddDate(1, 0, 0)

	// Add duplicate DIDs
	duplicateDIDState.DidDirectories = []types.DIDDirectory{
		{
			Did:        "did:example:same",
			Controller: "cosmos1controller1",
			Created:    now,
			Modified:   now,
			Exp:        oneYearLater,
			Deposit:    5000,
		},
		{
			Did:        "did:example:same", // Duplicate DID
			Controller: "cosmos1controller2",
			Created:    now,
			Modified:   now,
			Exp:        oneYearLater,
			Deposit:    6000,
		},
	}

	// This should panic
	require.Panics(t, func() {
		diddirectory.InitGenesis(ctx, k, *duplicateDIDState)
	})
}

// TestImportExportEquivalence verifies that export after import produces the same state
func TestImportExportEquivalence(t *testing.T) {
	// Create a keeper with some initial state
	k, ctx := keepertest.DiddirectoryKeeper(t)
	now := time.Now().UTC()
	oneYearLater := now.AddDate(1, 0, 0)

	// Create a genesis state with some DIDs
	initialState := types.DefaultGenesis()
	initialState.DidDirectories = []types.DIDDirectory{
		{
			Did:        "did:example:first",
			Controller: "cosmos1controller1",
			Created:    now,
			Modified:   now,
			Exp:        oneYearLater,
			Deposit:    5000,
		},
		{
			Did:        "did:example:second",
			Controller: "cosmos1controller2",
			Created:    now,
			Modified:   now,
			Exp:        oneYearLater,
			Deposit:    6000,
		},
	}

	// Initialize with this state
	diddirectory.InitGenesis(ctx, k, *initialState)

	// Export the state
	exportedState := diddirectory.ExportGenesis(ctx, k)

	// Verify that the exported state matches the initial state
	// We need to account for deterministic ordering of DIDs
	require.Len(t, exportedState.DidDirectories, len(initialState.DidDirectories))

	// Create maps to check DIDs by ID
	initialMap := make(map[string]types.DIDDirectory)
	for _, did := range initialState.DidDirectories {
		initialMap[did.Did] = did
	}

	exportedMap := make(map[string]types.DIDDirectory)
	for _, did := range exportedState.DidDirectories {
		exportedMap[did.Did] = did
	}

	// Check that all DIDs are preserved
	for didID, initialDID := range initialMap {
		exportedDID, exists := exportedMap[didID]
		require.True(t, exists)

		// Compare all fields
		require.Equal(t, initialDID.Controller, exportedDID.Controller)
		require.Equal(t, initialDID.Deposit, exportedDID.Deposit)

		// Use with nullify to ignore slight timestamp differences that might exist
		nullifiedInitial := nullify.Fill(initialDID)
		nullifiedExported := nullify.Fill(exportedDID)
		require.Equal(t, nullifiedInitial, nullifiedExported)
	}

	// Also verify params
	require.Equal(t, initialState.Params, exportedState.Params)

	// Now create a new keeper and import the exported state
	newKeeper, newCtx := keepertest.DiddirectoryKeeper(t)
	diddirectory.InitGenesis(newCtx, newKeeper, *exportedState)

	// Export again and verify it's the same
	reExportedState := diddirectory.ExportGenesis(newCtx, newKeeper)

	// Use nullify to ignore non-deterministic fields like timestamps with nanosecond precision
	nullifiedExported := nullify.Fill(exportedState)
	nullifiedReExported := nullify.Fill(reExportedState)

	require.Equal(t, nullifiedExported, nullifiedReExported)
}
