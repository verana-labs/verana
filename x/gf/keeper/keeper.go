package keeper

import (
	"fmt"

	"cosmossdk.io/collections"
	"cosmossdk.io/core/store"
	"cosmossdk.io/log"
	"github.com/cosmos/cosmos-sdk/codec"
	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/verana-labs/verana/x/gf/types"
)

// corpKeeperRef is a stable container for the CorporationKeeper interface,
// indirected behind a pointer so all by-value copies of Keeper (held by msg
// server, query server, etc.) see the same instance. SetCorporationKeeper
// writes to this container after construction — required because MOD-CO and
// MOD-GF are mutually-dependent at the keeper layer (cycle break per #303).
type corpKeeperRef struct {
	K types.CorporationKeeper
}

// ecoKeeperRef mirrors corpKeeperRef for MOD-ES (cycle break per #305).
// x/ec depends on x/gf (for CreateInitialGFVersionForEcosystem and
// ListVersionsByEcosystem) AND x/gf depends on x/ec (for GetEcosystemView and
// SetEcosystemActiveVersion). Resolved by injecting at construction time
// in one direction (ec → gf via concrete keeper) and post-construction in
// the other (gf.SetEcosystemKeeper(EcAsGFEcosystemKeeper{ec})).
type ecoKeeperRef struct {
	K types.EcosystemKeeper
}

type Keeper struct {
	cdc          codec.BinaryCodec
	storeService store.KVStoreService
	logger       log.Logger
	authority    string

	Schema                 collections.Schema
	Params                 collections.Item[types.Params]
	GFVersion              collections.Map[uint64, types.GovernanceFrameworkVersion]
	GFDocument             collections.Map[uint64, types.GovernanceFrameworkDocument]
	GFVersionByEcosystem   collections.Map[collections.Pair[uint64, uint32], uint64]
	GFVersionByCorporation collections.Map[collections.Pair[uint64, uint32], uint64]
	Counter                collections.Map[string, uint64]

	delegationKeeper types.DelegationKeeper
	ecoRef           *ecoKeeperRef
	corpRef          *corpKeeperRef
}

func NewKeeper(
	cdc codec.BinaryCodec,
	storeService store.KVStoreService,
	logger log.Logger,
	authority string,
	delegationKeeper types.DelegationKeeper,
) Keeper {
	if _, err := sdk.AccAddressFromBech32(authority); err != nil {
		panic(fmt.Sprintf("invalid authority address: %s", authority))
	}

	sb := collections.NewSchemaBuilder(storeService)
	k := Keeper{
		cdc:          cdc,
		storeService: storeService,
		authority:    authority,
		logger:       logger,
		Params:       collections.NewItem(sb, types.ParamsKey, "params", codec.CollValue[types.Params](cdc)),
		GFVersion:    collections.NewMap(sb, types.GovernanceFrameworkVersionKey, "gf_version", collections.Uint64Key, codec.CollValue[types.GovernanceFrameworkVersion](cdc)),
		GFDocument:   collections.NewMap(sb, types.GovernanceFrameworkDocumentKey, "gf_document", collections.Uint64Key, codec.CollValue[types.GovernanceFrameworkDocument](cdc)),
		GFVersionByEcosystem:   collections.NewMap(sb, types.GFVersionByEcosystemKey, "gf_version_by_ecosystem", collections.PairKeyCodec(collections.Uint64Key, collections.Uint32Key), collections.Uint64Value),
		GFVersionByCorporation: collections.NewMap(sb, types.GFVersionByCorporationKey, "gf_version_by_corporation", collections.PairKeyCodec(collections.Uint64Key, collections.Uint32Key), collections.Uint64Value),
		Counter:                collections.NewMap(sb, types.CounterKey, "counter", collections.StringKey, collections.Uint64Value),

		delegationKeeper: delegationKeeper,
		ecoRef:           &ecoKeeperRef{K: StubEcosystemKeeper{}},
		corpRef:          &corpKeeperRef{K: StubCorporationKeeper{}},
	}

	schema, err := sb.Build()
	if err != nil {
		panic(err)
	}
	k.Schema = schema
	return k
}

// SetCorporationKeeper wires the real MOD-CO keeper after both keepers exist.
// The receiver is by-value because the inner *corpKeeperRef is shared by all
// keeper copies (msg server, query server, etc.). MUST be called once during
// app init before any handler runs.
func (k Keeper) SetCorporationKeeper(c types.CorporationKeeper) {
	k.corpRef.K = c
}

// SetEcosystemKeeper mirrors SetCorporationKeeper: wires the real MOD-ES
// keeper post-construction (cycle break for #305).
func (k Keeper) SetEcosystemKeeper(e types.EcosystemKeeper) {
	k.ecoRef.K = e
}

// corporationKeeper returns the wired CorporationKeeper (real or stub).
func (k Keeper) corporationKeeper() types.CorporationKeeper {
	return k.corpRef.K
}

// ecosystemKeeper returns the wired EcosystemKeeper (real or stub).
func (k Keeper) ecosystemKeeper() types.EcosystemKeeper {
	return k.ecoRef.K
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
