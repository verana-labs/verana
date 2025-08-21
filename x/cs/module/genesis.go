package credentialschema

import (
	"fmt"
	"sort"

	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/verana-labs/verana/x/cs/keeper"
	"github.com/verana-labs/verana/x/cs/types"
)

// InitGenesis initializes the module's state from a provided genesis state.
func InitGenesis(ctx sdk.Context, k keeper.Keeper, genState types.GenesisState) {
	// Set module parameters
	if err := k.SetParams(ctx, genState.Params); err != nil {
		panic(fmt.Sprintf("failed to set params: %s", err))
	}

	// Initialize counter - we'll update with the highest ID after importing
	maxID := uint64(0)

	// Initialize Credential Schemas - sorted by ID for deterministic import
	schemas := genState.CredentialSchemas
	sort.Slice(schemas, func(i, j int) bool {
		return schemas[i].Id < schemas[j].Id
	})

	for _, cs := range schemas {
		// Set credential schema
		if err := k.CredentialSchema.Set(ctx, cs.Id, cs); err != nil {
			panic(fmt.Sprintf("failed to set Credential Schema: %s", err))
		}
		// Track highest ID to set counter correctly
		if cs.Id > maxID {
			maxID = cs.Id
		}
	}

	// Set counter to the highest existing ID
	// This is the fix: Always set the counter, even if maxID is 0
	// This ensures the collections key exists for later retrieval
	err := k.Counter.Set(ctx, "cs", maxID)
	if err != nil {
		panic(fmt.Sprintf("failed to set counter: %s", err))
	}

	k.Logger().Info("Initialized Credential Schema module",
		"schemas_count", len(schemas),
		"highest_id", maxID)
}

// ExportGenesis returns the module's exported genesis.
func ExportGenesis(ctx sdk.Context, k keeper.Keeper) *types.GenesisState {
	genesis := types.DefaultGenesis()
	genesis.Params = k.GetParams(ctx)

	// Export all credential schemas in a deterministic order
	var credentialSchemas []types.CredentialSchema
	err := k.CredentialSchema.Walk(ctx, nil, func(key uint64, cs types.CredentialSchema) (bool, error) {
		credentialSchemas = append(credentialSchemas, cs)
		return false, nil
	})
	if err != nil {
		panic(fmt.Sprintf("failed to export Credential Schemas: %s", err))
	}

	// Sort by ID for deterministic export
	sort.Slice(credentialSchemas, func(i, j int) bool {
		return credentialSchemas[i].Id < credentialSchemas[j].Id
	})

	genesis.CredentialSchemas = credentialSchemas

	// Export the counter value - THIS IS THE MISSING PART
	counter, err := k.Counter.Get(ctx, "cs")
	if err != nil {
		// If there's no counter but we have schemas, use the highest ID
		if len(credentialSchemas) > 0 {
			highestID := uint64(0)
			for _, cs := range credentialSchemas {
				if cs.Id > highestID {
					highestID = cs.Id
				}
			}
			counter = highestID
		}
	}
	genesis.SchemaCounter = counter

	return genesis
}
