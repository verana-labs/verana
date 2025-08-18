package keeper

import (
	"context"
	"fmt"
	"sort"

	"github.com/verana-labs/verana/x/dd/types"
)

// InitGenesis initializes the module's state from a provided genesis state.
func (k Keeper) InitGenesis(ctx context.Context, genState types.GenesisState) error {
	// Validate genesis state first
	if err := genState.Validate(); err != nil {
		panic(fmt.Sprintf("invalid dd genesis state: %s", err))
	}

	// Set module parameters
	if err := k.SetParams(ctx, genState.Params); err != nil {
		panic(fmt.Sprintf("failed to set parameters: %s", err))
	}

	// Initialize DID directories in deterministic order
	// Sort them by DID to ensure deterministic initialization
	didDirectories := genState.DidDirectories
	sort.SliceStable(didDirectories, func(i, j int) bool {
		return didDirectories[i].Did < didDirectories[j].Did
	})

	// Track DIDs to prevent duplicates during import
	seenDIDs := make(map[string]bool)

	// Set all DID directory entries
	for _, dd := range didDirectories {
		// Double-check for duplicates (should already be caught by Validate)
		if seenDIDs[dd.Did] {
			panic(fmt.Sprintf("duplicate DID in genesis state: %s", dd.Did))
		}
		seenDIDs[dd.Did] = true

		// Set DID directory
		if err := k.DIDDirectory.Set(ctx, dd.Did, dd); err != nil {
			panic(fmt.Sprintf("failed to set DID directory: %s", err))
		}
	}

	k.Logger().Info("Initialized DID directory genesis state",
		"params", genState.Params,
		"did_count", len(didDirectories))

	return nil
}

// ExportGenesis returns the module's exported genesis.
func (k Keeper) ExportGenesis(ctx context.Context) (*types.GenesisState, error) {
	genesis := types.DefaultGenesis()
	genesis.Params = k.GetParams(ctx)

	// Export all DID directories
	var didDirectories []types.DIDDirectory

	// Use collections.Walk to get all DID entries
	err := k.DIDDirectory.Walk(ctx, nil, func(key string, did types.DIDDirectory) (bool, error) {
		didDirectories = append(didDirectories, did)
		return false, nil
	})

	if err != nil {
		panic(fmt.Sprintf("failed to export DID directory: %s", err))
	}

	// Sort DID directories by DID to ensure deterministic export order
	sort.SliceStable(didDirectories, func(i, j int) bool {
		return didDirectories[i].Did < didDirectories[j].Did
	})

	genesis.DidDirectories = didDirectories

	// Perform a final sanitization to ensure deterministic ordering
	return types.SanitizeGenesisState(genesis), nil
}
