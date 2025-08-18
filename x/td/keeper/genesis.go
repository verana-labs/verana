package keeper

import (
	"context"
	"fmt"

	"github.com/verana-labs/verana/x/td/types"
)

// InitGenesis initializes the module's state from a provided genesis state.
func (k Keeper) InitGenesis(ctx context.Context, genState types.GenesisState) error {
	if err := k.SetParams(ctx, genState.Params); err != nil {
		panic(fmt.Sprintf("failed to set params: %s", err))
	}

	// Initialize trust deposits
	for _, td := range genState.TrustDeposits {
		// Create trust deposit entry
		trustDeposit := types.TrustDeposit{
			Account:   td.Account,
			Share:     td.Share,
			Amount:    td.Amount,
			Claimable: td.Claimable,
		}

		// Store the trust deposit
		if err := k.TrustDeposit.Set(ctx, td.Account, trustDeposit); err != nil {
			panic(fmt.Sprintf("failed to set trust deposit for account %s: %s", td.Account, err))
		}
	}
	return nil
}

// ExportGenesis returns the module's exported genesis.
func (k Keeper) ExportGenesis(ctx context.Context) (*types.GenesisState, error) {
	genesis := types.DefaultGenesis()
	genesis.Params = k.GetParams(ctx)

	// Export trust deposits
	var trustDeposits []types.TrustDepositRecord

	// Use a callback to gather all trust deposits in deterministic order
	// The Walk function should iterate over keys in lexicographical order
	_ = k.TrustDeposit.Walk(ctx, nil, func(key string, value types.TrustDeposit) (bool, error) {
		trustDeposits = append(trustDeposits, types.TrustDepositRecord{
			Account:   value.Account,
			Share:     value.Share,
			Amount:    value.Amount,
			Claimable: value.Claimable,
		})
		return false, nil // Continue iteration
	})

	genesis.TrustDeposits = trustDeposits

	return genesis, nil
}
