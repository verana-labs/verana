package keeper

import (
	"context"
	"fmt"

	"github.com/verana-labs/verana/x/di/types"
)

// InitGenesis initializes the module's state from a provided genesis state.
func (k Keeper) InitGenesis(ctx context.Context, genState types.GenesisState) error {
	if err := k.Params.Set(ctx, genState.Params); err != nil {
		return err
	}

	for _, digest := range genState.Digests {
		if err := k.Digests.Set(ctx, digest.Digest, digest); err != nil {
			return fmt.Errorf("failed to set digest %s: %w", digest.Digest, err)
		}
	}

	return nil
}

// ExportGenesis returns the module's exported genesis.
func (k Keeper) ExportGenesis(ctx context.Context) (*types.GenesisState, error) {
	var err error

	genesis := types.DefaultGenesis()
	genesis.Params, err = k.Params.Get(ctx)
	if err != nil {
		return nil, err
	}

	var digests []types.Digest
	err = k.Digests.Walk(ctx, nil, func(key string, digest types.Digest) (bool, error) {
		digests = append(digests, digest)
		return false, nil
	})
	if err != nil {
		return nil, fmt.Errorf("failed to export digests: %w", err)
	}
	genesis.Digests = digests

	return genesis, nil
}
