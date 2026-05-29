package keeper

import (
	"context"
	"fmt"

	"cosmossdk.io/collections"
	"cosmossdk.io/core/store"
	"cosmossdk.io/log"
	"github.com/cosmos/cosmos-sdk/codec"
	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/verana-labs/verana/x/ec/types"
)

// Keeper holds MOD-ES state. GFV/GFD storage lives in x/gf; this keeper holds
// only the Ecosystem entity + the (did, corporation_id) consistency index
// + the per-module counter for ec ids.
type Keeper struct {
	cdc          codec.BinaryCodec
	storeService store.KVStoreService
	logger       log.Logger
	authority    string

	Schema             collections.Schema
	Params             collections.Item[types.Params]
	Ecosystem          collections.Map[uint64, types.Ecosystem]
	EcosystemByDIDCorp collections.Map[collections.Pair[string, uint64], uint64]
	Counter            collections.Map[string, uint64]

	delegationKeeper types.DelegationKeeper
	coKeeper         types.CorporationKeeper
	gfKeeper         types.GFKeeper
}

func NewKeeper(
	cdc codec.BinaryCodec,
	storeService store.KVStoreService,
	logger log.Logger,
	authority string,
	delegationKeeper types.DelegationKeeper,
	coKeeper types.CorporationKeeper,
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
		Ecosystem:    collections.NewMap(sb, types.EcosystemKey, "ecosystem", collections.Uint64Key, codec.CollValue[types.Ecosystem](cdc)),
		EcosystemByDIDCorp: collections.NewMap(sb, types.EcosystemByDIDCorpKey, "ecosystem_by_did_corp",
			collections.PairKeyCodec(collections.StringKey, collections.Uint64Key), collections.Uint64Value),
		Counter:          collections.NewMap(sb, types.CounterKey, "counter", collections.StringKey, collections.Uint64Value),
		delegationKeeper: delegationKeeper,
		coKeeper:         coKeeper,
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

// GetEcosystem is the read accessor consumed by x/cs and x/pp via their
// respective EcosystemKeeper interfaces (cs uses it to enforce the
// ec.CorporationId ownership chain; perm uses it for the same plus the
// schema-controller lookup).
func (k Keeper) GetEcosystem(ctx context.Context, id uint64) (types.Ecosystem, error) {
	return k.Ecosystem.Get(ctx, id)
}

func (k Keeper) Logger() log.Logger {
	return k.logger.With("module", fmt.Sprintf("x/%s", types.ModuleName))
}

// GetNextID allocates the next id for a given entity_type and persists the
// updated counter. Mirrors the MOD-CO pattern.
func (k Keeper) GetNextID(ctx context.Context, entityType string) (uint64, error) {
	cur, err := k.Counter.Get(ctx, entityType)
	if err != nil {
		cur = 0
	}
	next := cur + 1
	if err := k.Counter.Set(ctx, entityType, next); err != nil {
		return 0, fmt.Errorf("set counter: %w", err)
	}
	return next, nil
}

// GetTrustUnitPrice retained for cross-module consumers (CS/PERM) that need
// the configured price for fee math. Signature accepts sdk.Context so the
// existing PERM expected_keepers contract continues to match.
func (k Keeper) GetTrustUnitPrice(ctx sdk.Context) uint64 {
	return k.GetParams(ctx).TrustUnitPrice
}
