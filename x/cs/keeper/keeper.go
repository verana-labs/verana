package keeper

import (
	"cosmossdk.io/log"
	"fmt"
	sdk "github.com/cosmos/cosmos-sdk/types"

	"cosmossdk.io/collections"
	"cosmossdk.io/core/address"
	corestore "cosmossdk.io/core/store"
	"github.com/cosmos/cosmos-sdk/codec"

	"github.com/verana-labs/verana/x/cs/types"
)

type Keeper struct {
	storeService corestore.KVStoreService
	cdc          codec.Codec
	addressCodec address.Codec
	logger       log.Logger
	// Address capable of executing a MsgUpdateParams message.
	// Typically, this should be the x/gov module account.
	authority           []byte
	bankKeeper          types.BankKeeper
	trustRegistryKeeper types.TrustRegistryKeeper

	Schema           collections.Schema
	CredentialSchema collections.Map[uint64, types.CredentialSchema]
	Counter          collections.Map[string, uint64]
	trustDeposit     types.TrustDepositKeeper
}

func NewKeeper(
	storeService corestore.KVStoreService,
	cdc codec.Codec,
	addressCodec address.Codec,
	logger log.Logger,
	authority []byte,
	bankKeeper types.BankKeeper,
	trustRegistryKeeper types.TrustRegistryKeeper,
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
		logger:       logger,
		authority:    authority,

		//Params:              collections.NewItem(sb, types.ParamsKey, "params", codec.CollValue[types.Params](cdc)),
		bankKeeper:          bankKeeper,
		trustRegistryKeeper: trustRegistryKeeper,

		// Initialize collections
		CredentialSchema: collections.NewMap(
			sb,
			types.CredentialSchemaKey,
			"credential_schema",
			collections.Uint64Key,
			codec.CollValue[types.CredentialSchema](cdc),
		),
		Counter: collections.NewMap(
			sb,
			types.CounterKey,
			"counter",
			collections.StringKey,
			collections.Uint64Value,
		),
		trustDeposit: trustDeposit,
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

// GetCredentialSchemaById GetCredentialSchema returns a credential schema by ID
func (k Keeper) GetCredentialSchemaById(ctx sdk.Context, id uint64) (types.CredentialSchema, error) {
	return k.CredentialSchema.Get(ctx, id)
}

// SetCredentialSchema sets a credential schema
func (k Keeper) SetCredentialSchema(ctx sdk.Context, schema types.CredentialSchema) error {
	return k.CredentialSchema.Set(ctx, schema.Id, schema)
}

// DeleteCredentialSchema deletes a credential schema
func (k Keeper) DeleteCredentialSchema(ctx sdk.Context, id uint64) error {
	return k.CredentialSchema.Remove(ctx, id)
}

// IterateCredentialSchemas iterates over all credential schemas
func (k Keeper) IterateCredentialSchemas(ctx sdk.Context, fn func(schema types.CredentialSchema) (stop bool)) error {
	return k.CredentialSchema.Walk(ctx, nil, func(key uint64, value types.CredentialSchema) (bool, error) {
		return fn(value), nil
	})
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
