package trustregistry

import (
	"fmt"

	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/verana-labs/verana/x/tr/keeper"
	"github.com/verana-labs/verana/x/tr/types"
)

// InitGenesis initializes the module's state from a provided genesis state.
func InitGenesis(ctx sdk.Context, k keeper.Keeper, genState types.GenesisState) {
	// Set module parameters
	if err := k.SetParams(ctx, genState.Params); err != nil {
		panic(fmt.Sprintf("failed to set params: %s", err))
	}

	// Initialize trust registries
	for _, tr := range genState.TrustRegistries {
		// Set trust registry
		if err := k.TrustRegistry.Set(ctx, tr.Id, tr); err != nil {
			panic(fmt.Sprintf("failed to set trust registry: %s", err))
		}

		// Set DID index
		if err := k.TrustRegistryDIDIndex.Set(ctx, tr.Did, tr.Id); err != nil {
			panic(fmt.Sprintf("failed to set DID index: %s", err))
		}
	}

	// Initialize governance framework versions
	for _, gfv := range genState.GovernanceFrameworkVersions {
		if err := k.GFVersion.Set(ctx, gfv.Id, gfv); err != nil {
			panic(fmt.Sprintf("failed to set governance framework version: %s", err))
		}
	}

	// Initialize governance framework documents
	for _, gfd := range genState.GovernanceFrameworkDocuments {
		if err := k.GFDocument.Set(ctx, gfd.Id, gfd); err != nil {
			panic(fmt.Sprintf("failed to set governance framework document: %s", err))
		}
	}

	// Initialize counters
	for _, counter := range genState.Counters {
		if err := k.Counter.Set(ctx, counter.EntityType, counter.Value); err != nil {
			panic(fmt.Sprintf("failed to set counter for %s: %s", counter.EntityType, err))
		}
	}
}

// ExportGenesis returns the module's exported genesis.
func ExportGenesis(ctx sdk.Context, k keeper.Keeper) *types.GenesisState {
	// Initialize with default genesis and update params
	genesis := types.DefaultGenesis()
	genesis.Params = k.GetParams(ctx)

	// Export all trust registries
	trustRegistries := exportTrustRegistries(ctx, k)
	genesis.TrustRegistries = trustRegistries

	// Export all governance framework versions
	gfVersions := exportGovernanceFrameworkVersions(ctx, k)
	genesis.GovernanceFrameworkVersions = gfVersions

	// Export all governance framework documents
	gfDocuments := exportGovernanceFrameworkDocuments(ctx, k)
	genesis.GovernanceFrameworkDocuments = gfDocuments

	// Export all counters
	counters := exportCounters(ctx, k)
	genesis.Counters = counters

	return genesis
}

// Helper functions for exporting each collection

func exportTrustRegistries(ctx sdk.Context, k keeper.Keeper) []types.TrustRegistry {
	var trustRegistries []types.TrustRegistry
	err := k.TrustRegistry.Walk(ctx, nil, func(key uint64, tr types.TrustRegistry) (bool, error) {
		trustRegistries = append(trustRegistries, tr)
		return false, nil
	})
	if err != nil {
		panic(fmt.Sprintf("failed to export trust registries: %s", err))
	}
	return trustRegistries
}

func exportGovernanceFrameworkVersions(ctx sdk.Context, k keeper.Keeper) []types.GovernanceFrameworkVersion {
	var gfVersions []types.GovernanceFrameworkVersion
	err := k.GFVersion.Walk(ctx, nil, func(key uint64, gfv types.GovernanceFrameworkVersion) (bool, error) {
		gfVersions = append(gfVersions, gfv)
		return false, nil
	})
	if err != nil {
		panic(fmt.Sprintf("failed to export governance framework versions: %s", err))
	}
	return gfVersions
}

func exportGovernanceFrameworkDocuments(ctx sdk.Context, k keeper.Keeper) []types.GovernanceFrameworkDocument {
	var gfDocuments []types.GovernanceFrameworkDocument
	err := k.GFDocument.Walk(ctx, nil, func(key uint64, gfd types.GovernanceFrameworkDocument) (bool, error) {
		gfDocuments = append(gfDocuments, gfd)
		return false, nil
	})
	if err != nil {
		panic(fmt.Sprintf("failed to export governance framework documents: %s", err))
	}
	return gfDocuments
}

func exportCounters(ctx sdk.Context, k keeper.Keeper) []types.Counter {
	var counters []types.Counter

	// We need to export all the counters we use in the module
	// Known counter keys from the code are: "tr", "gfv", "gfd"
	counterKeys := []string{"tr", "gfv", "gfd"}

	for _, key := range counterKeys {
		value, err := k.Counter.Get(ctx, key)
		// If the counter doesn't exist, we don't need to include it
		if err == nil {
			counters = append(counters, types.Counter{
				EntityType: key,
				Value:      value,
			})
		}
	}

	return counters
}
