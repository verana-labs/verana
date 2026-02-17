package keeper

import (
	"fmt"

	"cosmossdk.io/collections"
	"cosmossdk.io/core/store"
	"cosmossdk.io/log"
	"github.com/cosmos/cosmos-sdk/codec"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/verana-labs/verana/x/tr/types"
)

type (
	Keeper struct {
		cdc          codec.BinaryCodec
		storeService store.KVStoreService
		logger       log.Logger

		// the address capable of executing a MsgUpdateParams message. Typically, this
		// should be the x/gov module account.
		authority string
		// state management
		Schema                collections.Schema
		Params                collections.Item[types.Params]
		TrustRegistry         collections.Map[uint64, types.TrustRegistry]
		TrustRegistryDIDIndex collections.Map[string, uint64] // Index for DID lookups
		GFVersion             collections.Map[uint64, types.GovernanceFrameworkVersion]
		GFDocument            collections.Map[uint64, types.GovernanceFrameworkDocument]
		Counter               collections.Map[string, uint64]
		// module references
		//bankKeeper    types.BankKeeper
		trustDeposit     types.TrustDepositKeeper
		delegationKeeper types.DelegationKeeper
	}
)

func NewKeeper(
	cdc codec.BinaryCodec,
	storeService store.KVStoreService,
	logger log.Logger,
	authority string,
	trustDeposit types.TrustDepositKeeper,
	delegationKeeper types.DelegationKeeper,
) Keeper {
	if _, err := sdk.AccAddressFromBech32(authority); err != nil {
		panic(fmt.Sprintf("invalid authority address: %s", authority))
	}

	sb := collections.NewSchemaBuilder(storeService)
	k := Keeper{
		cdc: cdc,
		//addressCodec:  addressCodec,
		storeService:          storeService,
		authority:             authority,
		logger:                logger,
		Params:                collections.NewItem(sb, types.ParamsKey, "params", codec.CollValue[types.Params](cdc)),
		TrustRegistry:         collections.NewMap(sb, types.TrustRegistryKey, "trust_registry", collections.Uint64Key, codec.CollValue[types.TrustRegistry](cdc)),
		TrustRegistryDIDIndex: collections.NewMap(sb, types.TrustRegistryDIDIndex, "trust_registry_did_index", collections.StringKey, collections.Uint64Value),
		GFVersion:             collections.NewMap(sb, types.GovernanceFrameworkVersionKey, "gf_version", collections.Uint64Key, codec.CollValue[types.GovernanceFrameworkVersion](cdc)),
		GFDocument:            collections.NewMap(sb, types.GovernanceFrameworkDocumentKey, "gf_document", collections.Uint64Key, codec.CollValue[types.GovernanceFrameworkDocument](cdc)),
		Counter:               collections.NewMap(sb, types.CounterKey, "counter", collections.StringKey, collections.Uint64Value),
		trustDeposit:          trustDeposit,
		delegationKeeper:      delegationKeeper,
	}

	schema, err := sb.Build()
	if err != nil {
		panic(err)
	}
	k.Schema = schema

	return k

}

// GetAuthority returns the module's authority.
func (k Keeper) GetAuthority() string {
	return k.authority
}

// Logger returns a module-specific logger.
func (k Keeper) Logger() log.Logger {
	return k.logger.With("module", fmt.Sprintf("x/%s", types.ModuleName))
}

func (k Keeper) GetTrustRegistryByDID(ctx sdk.Context, did string) (types.TrustRegistry, error) {
	// Get ID from DID index
	id, err := k.TrustRegistryDIDIndex.Get(ctx, did)
	if err != nil {
		return types.TrustRegistry{}, fmt.Errorf("trust registry with DID %s not found: %w", did, err)
	}

	// Get Trust Registry using ID
	return k.TrustRegistry.Get(ctx, id)
}

func (k Keeper) GetTrustRegistry(ctx sdk.Context, id uint64) (types.TrustRegistry, error) {
	return k.TrustRegistry.Get(ctx, id)
}

func (k Keeper) GetNextID(ctx sdk.Context, entityType string) (uint64, error) {
	currentID, err := k.Counter.Get(ctx, entityType)
	if err != nil {
		currentID = 0
	}

	nextID := currentID + 1
	err = k.Counter.Set(ctx, entityType, nextID)
	if err != nil {
		return 0, fmt.Errorf("failed to set counter: %w", err)
	}

	return nextID, nil
}

func (k Keeper) GetTrustUnitPrice(ctx sdk.Context) uint64 {
	params := k.GetParams(ctx)
	return params.TrustUnitPrice
}
