package keeper

import (
	"context"
	"errors"
	"fmt"
	"sort"

	"cosmossdk.io/collections"

	"github.com/verana-labs/verana/x/xr/types"
)

// InitGenesis initializes the module's state from a provided genesis state.
func (k Keeper) InitGenesis(ctx context.Context, genState types.GenesisState) error {
	if err := k.Params.Set(ctx, genState.Params); err != nil {
		return err
	}

	for _, xr := range genState.ExchangeRates {
		if err := k.ExchangeRates.Set(ctx, xr.Id, xr); err != nil {
			return fmt.Errorf("failed to set exchange rate %d: %w", xr.Id, err)
		}

		// Rebuild PairIndex (derived index)
		pairKey := buildPairKey(xr.BaseAssetType, xr.BaseAsset, xr.QuoteAssetType, xr.QuoteAsset)
		if err := k.PairIndex.Set(ctx, pairKey, xr.Id); err != nil {
			return fmt.Errorf("failed to set pair index for exchange rate %d: %w", xr.Id, err)
		}
	}

	// Set Counter for exchange_rate
	if genState.NextExchangeRateId > 0 {
		if err := k.Counter.Set(ctx, types.CounterKeyExchangeRate, genState.NextExchangeRateId); err != nil {
			return fmt.Errorf("failed to set exchange rate counter: %w", err)
		}
	}

	for _, authz := range genState.ExchangeRateAuthorizations {
		key := collections.Join(authz.XrId, authz.Operator)
		if err := k.ExchangeRateAuthorizations.Set(ctx, key, authz); err != nil {
			return fmt.Errorf("failed to set exchange rate authorization (%d, %s): %w", authz.XrId, authz.Operator, err)
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

	var exchangeRates []types.ExchangeRate
	err = k.ExchangeRates.Walk(ctx, nil, func(id uint64, xr types.ExchangeRate) (bool, error) {
		exchangeRates = append(exchangeRates, xr)
		return false, nil
	})
	if err != nil {
		return nil, fmt.Errorf("failed to export exchange rates: %w", err)
	}

	// Sort by ID for deterministic output
	sort.Slice(exchangeRates, func(i, j int) bool {
		return exchangeRates[i].Id < exchangeRates[j].Id
	})
	genesis.ExchangeRates = exchangeRates

	// Export counter
	nextID, err := k.Counter.Get(ctx, types.CounterKeyExchangeRate)
	if err != nil && !errors.Is(err, collections.ErrNotFound) {
		return nil, fmt.Errorf("failed to get exchange rate counter: %w", err)
	}
	if errors.Is(err, collections.ErrNotFound) {
		nextID = 0
	}
	genesis.NextExchangeRateId = nextID

	var authorizations []types.ExchangeRateAuthorization
	err = k.ExchangeRateAuthorizations.Walk(ctx, nil, func(_ collections.Pair[uint64, string], authz types.ExchangeRateAuthorization) (bool, error) {
		authorizations = append(authorizations, authz)
		return false, nil
	})
	if err != nil {
		return nil, fmt.Errorf("failed to export exchange rate authorizations: %w", err)
	}
	// Sort by (xr_id, operator) for deterministic output.
	sort.Slice(authorizations, func(i, j int) bool {
		if authorizations[i].XrId != authorizations[j].XrId {
			return authorizations[i].XrId < authorizations[j].XrId
		}
		return authorizations[i].Operator < authorizations[j].Operator
	})
	genesis.ExchangeRateAuthorizations = authorizations

	return genesis, nil
}
