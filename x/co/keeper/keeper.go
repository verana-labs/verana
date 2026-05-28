package keeper

import (
	"fmt"

	"cosmossdk.io/collections"
	"cosmossdk.io/core/store"
	"cosmossdk.io/log"
	"github.com/cosmos/cosmos-sdk/codec"
	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/verana-labs/verana/x/co/types"
)

type Keeper struct {
	cdc          codec.BinaryCodec
	storeService store.KVStoreService
	logger       log.Logger
	authority    string

	Schema                    collections.Schema
	Params                    collections.Item[types.Params]
	Corporation               collections.Map[uint64, types.Corporation]
	CorporationByPolicyAddr   collections.Map[string, uint64]
	CorporationByDID          collections.Map[string, uint64]
	Counter                   collections.Map[string, uint64]

	delegationKeeper types.DelegationKeeper
	groupKeeper      types.GroupKeeper
	gfKeeper         types.GFKeeper
}

func NewKeeper(
	cdc codec.BinaryCodec,
	storeService store.KVStoreService,
	logger log.Logger,
	authority string,
	delegationKeeper types.DelegationKeeper,
	groupKeeper types.GroupKeeper,
	gfKeeper types.GFKeeper,
) Keeper {
	if _, err := sdk.AccAddressFromBech32(authority); err != nil {
		panic(fmt.Sprintf("invalid authority address: %s", authority))
	}

	sb := collections.NewSchemaBuilder(storeService)
	k := Keeper{
		cdc:          cdc,
		storeService: storeService,
		logger:       logger,
		authority:    authority,
		Params:       collections.NewItem(sb, types.ParamsKey, "params", codec.CollValue[types.Params](cdc)),
		Corporation:  collections.NewMap(sb, types.CorporationKey, "corporation", collections.Uint64Key, codec.CollValue[types.Corporation](cdc)),
		CorporationByPolicyAddr: collections.NewMap(sb, types.CorporationByPolicyAddressKey, "corporation_by_policy_addr", collections.StringKey, collections.Uint64Value),
		CorporationByDID:        collections.NewMap(sb, types.CorporationByDIDKey, "corporation_by_did", collections.StringKey, collections.Uint64Value),
		Counter:                 collections.NewMap(sb, types.CounterKey, "counter", collections.StringKey, collections.Uint64Value),

		delegationKeeper: delegationKeeper,
		groupKeeper:      groupKeeper,
		gfKeeper:         gfKeeper,
	}

	schema, err := sb.Build()
	if err != nil {
		panic(err)
	}
	k.Schema = schema
	return k
}

func (k Keeper) GetAuthority() string { return k.authority }

func (k Keeper) Logger() log.Logger {
	return k.logger.With("module", "x/"+types.ModuleName)
}

func (k Keeper) GetNextID(ctx sdk.Context, entityType string) (uint64, error) {
	current, err := k.Counter.Get(ctx, entityType)
	if err != nil {
		current = 0
	}
	next := current + 1
	if err := k.Counter.Set(ctx, entityType, next); err != nil {
		return 0, fmt.Errorf("failed to set counter: %w", err)
	}
	return next, nil
}
