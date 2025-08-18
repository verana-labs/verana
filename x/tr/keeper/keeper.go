package keeper

import (
	"cosmossdk.io/log"
	"fmt"
	sdk "github.com/cosmos/cosmos-sdk/types"

	"cosmossdk.io/collections"
	"cosmossdk.io/core/address"
	corestore "cosmossdk.io/core/store"
	"github.com/cosmos/cosmos-sdk/codec"

	"github.com/verana-labs/verana/x/tr/types"
)

type Keeper struct {
	storeService corestore.KVStoreService
	cdc          codec.Codec
	addressCodec address.Codec
	// Address capable of executing a MsgUpdateParams message.
	// Typically, this should be the x/gov module account.
	logger    log.Logger
	authority []byte

	Schema                collections.Schema
	Params                collections.Item[types.Params]
	TrustRegistry         collections.Map[uint64, types.TrustRegistry]
	TrustRegistryDIDIndex collections.Map[string, uint64] // Index for DID lookups
	GFVersion             collections.Map[uint64, types.GovernanceFrameworkVersion]
	GFDocument            collections.Map[uint64, types.GovernanceFrameworkDocument]
	Counter               collections.Map[string, uint64]
	// module references
	//bankKeeper    types.BankKeeper
	trustDeposit types.TrustDepositKeeper
}

func NewKeeper(
	storeService corestore.KVStoreService,
	cdc codec.Codec,
	addressCodec address.Codec,
	logger log.Logger,
	authority []byte,
	trustDeposit types.TrustDepositKeeper,

) Keeper {
	if _, err := addressCodec.BytesToString(authority); err != nil {
		panic(fmt.Sprintf("invalid authority address %s: %s", authority, err))
	}

	sb := collections.NewSchemaBuilder(storeService)

	k := Keeper{
		storeService: storeService,
		cdc:          cdc,
		addressCodec: addressCodec,
		authority:    authority,

		Params:                collections.NewItem(sb, types.ParamsKey, "params", codec.CollValue[types.Params](cdc)),
		TrustRegistry:         collections.NewMap(sb, types.TrustRegistryKey, "trust_registry", collections.Uint64Key, codec.CollValue[types.TrustRegistry](cdc)),
		TrustRegistryDIDIndex: collections.NewMap(sb, types.TrustRegistryDIDIndex, "trust_registry_did_index", collections.StringKey, collections.Uint64Value),
		GFVersion:             collections.NewMap(sb, types.GovernanceFrameworkVersionKey, "gf_version", collections.Uint64Key, codec.CollValue[types.GovernanceFrameworkVersion](cdc)),
		GFDocument:            collections.NewMap(sb, types.GovernanceFrameworkDocumentKey, "gf_document", collections.Uint64Key, codec.CollValue[types.GovernanceFrameworkDocument](cdc)),
		Counter:               collections.NewMap(sb, types.CounterKey, "counter", collections.StringKey, collections.Uint64Value),
		trustDeposit:          trustDeposit,
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
