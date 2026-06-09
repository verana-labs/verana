package trustdeposit

import (
	"fmt"

	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/verana-labs/verana/x/td/keeper"
	"github.com/verana-labs/verana/x/td/types"
)

// InitGenesis initializes the module's state from a provided genesis state.
func InitGenesis(ctx sdk.Context, k keeper.Keeper, genState types.GenesisState) {
	// Set module parameters
	if err := k.SetParams(ctx, genState.Params); err != nil {
		panic(fmt.Sprintf("failed to set params: %s", err))
	}

	// Initialize dust
	if genState.Dust != "" {
		if err := k.Dust.Set(ctx, genState.Dust); err != nil {
			panic(fmt.Sprintf("failed to set dust: %s", err))
		}
	}

	// Initialize trust deposits (keyed by corporation_id)
	for _, td := range genState.TrustDeposits {
		// Create trust deposit entry
		trustDeposit := types.TrustDeposit{
			CorporationId: td.CorporationId,
			Share:         td.Share,
			Deposit:       td.Deposit,
			Refunded:      td.Refunded,
		}

		// Store the trust deposit
		if err := k.TrustDeposit.Set(ctx, td.CorporationId, trustDeposit); err != nil {
			panic(fmt.Sprintf("failed to set trust deposit for corporation_id %d: %s", td.CorporationId, err))
		}
	}
}

// ExportGenesis returns the module's exported genesis.
func ExportGenesis(ctx sdk.Context, k keeper.Keeper) *types.GenesisState {
	genesis := types.DefaultGenesis()
	genesis.Params = k.GetParams(ctx)

	// Export trust deposits
	var trustDeposits []types.TrustDepositRecord

	// Use a callback to gather all trust deposits in deterministic order
	// The Walk function should iterate over keys in lexicographical order
	_ = k.TrustDeposit.Walk(ctx, nil, func(key uint64, value types.TrustDeposit) (bool, error) {
		trustDeposits = append(trustDeposits, types.TrustDepositRecord{
			CorporationId: value.CorporationId,
			Share:         value.Share,
			Deposit:       value.Deposit,
			Refunded:      value.Refunded,
		})
		return false, nil // Continue iteration
	})

	genesis.TrustDeposits = trustDeposits

	// Export dust
	dust, err := k.Dust.Get(ctx)
	if err == nil {
		genesis.Dust = dust
	}

	return genesis
}
