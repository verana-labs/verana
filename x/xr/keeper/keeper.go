package keeper

import (
	"context"
	"fmt"

	"cosmossdk.io/collections"
	"cosmossdk.io/core/address"
	corestore "cosmossdk.io/core/store"
	"github.com/cosmos/cosmos-sdk/codec"

	"github.com/verana-labs/verana/x/xr/types"
)

type Keeper struct {
	storeService corestore.KVStoreService
	cdc          codec.Codec
	addressCodec address.Codec
	// Address capable of executing a MsgUpdateParams message.
	// Typically, this should be the x/gov module account.
	authority []byte

	// module references
	delegationKeeper types.DelegationKeeper

	Schema        collections.Schema
	Params        collections.Item[types.Params]
	ExchangeRates collections.Map[uint64, types.ExchangeRate]
	Counter       collections.Map[string, uint64]
	// PairIndex maps "baseType:baseAsset:quoteType:quoteAsset" -> exchange rate id for uniqueness checks
	PairIndex collections.Map[string, uint64]
}

func NewKeeper(
	storeService corestore.KVStoreService,
	cdc codec.Codec,
	addressCodec address.Codec,
	authority []byte,
	delegationKeeper types.DelegationKeeper,
) Keeper {
	if _, err := addressCodec.BytesToString(authority); err != nil {
		panic(fmt.Sprintf("invalid authority address %s: %s", authority, err))
	}

	sb := collections.NewSchemaBuilder(storeService)

	k := Keeper{
		storeService:     storeService,
		cdc:              cdc,
		addressCodec:     addressCodec,
		authority:        authority,
		delegationKeeper: delegationKeeper,

		Params: collections.NewItem(sb, types.ParamsKey, "params", codec.CollValue[types.Params](cdc)),
		ExchangeRates: collections.NewMap(
			sb,
			types.ExchangeRateKey,
			"exchange_rate",
			collections.Uint64Key,
			codec.CollValue[types.ExchangeRate](cdc),
		),
		Counter: collections.NewMap(
			sb,
			types.CounterKey,
			"counter",
			collections.StringKey,
			collections.Uint64Value,
		),
		PairIndex: collections.NewMap(
			sb,
			types.ExchangeRatePairIndexKey,
			"exchange_rate_pair_index",
			collections.StringKey,
			collections.Uint64Value,
		),
	}

	schema, err := sb.Build()
	if err != nil {
		panic(err)
	}
	k.Schema = schema

	return k
}

// GetAuthority returns the module's authority.
func (k Keeper) GetAuthority() []byte {
	return k.authority
}

// GetNextID returns the next auto-incremented ID for the given entity type.
func (k Keeper) GetNextID(ctx context.Context, entityType string) (uint64, error) {
	currentID, err := k.Counter.Get(ctx, entityType)
	if err != nil {
		currentID = 0
	}

	nextID := currentID + 1
	if err := k.Counter.Set(ctx, entityType, nextID); err != nil {
		return 0, fmt.Errorf("failed to set counter: %w", err)
	}

	return nextID, nil
}
