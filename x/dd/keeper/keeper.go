package keeper

import (
	"fmt"

	"cosmossdk.io/collections"
	"cosmossdk.io/core/address"
	corestore "cosmossdk.io/core/store"
	"cosmossdk.io/log"
	"github.com/cosmos/cosmos-sdk/codec"
	"github.com/verana-labs/verana/x/dd/types"
)

type Keeper struct {
	storeService corestore.KVStoreService
	cdc          codec.Codec
	addressCodec address.Codec
	logger       log.Logger

	// Address capable of executing a MsgUpdateParams message.
	// Typically, this should be the x/gov module account.
	authority []byte

	Schema              collections.Schema
	DIDDirectory        collections.Map[string, types.DIDDirectory]
	trustDeposit        types.TrustDepositKeeper
	trustRegistryKeeper types.TrustRegistryKeeper
}

func NewKeeper(
	storeService corestore.KVStoreService,
	cdc codec.Codec,
	addressCodec address.Codec,
	logger log.Logger,
	authority []byte,
	trustDeposit types.TrustDepositKeeper,
	trustRegistryKeeper types.TrustRegistryKeeper,
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

		DIDDirectory:        collections.NewMap(sb, types.DIDDirectoryKey, "did_directory", collections.StringKey, codec.CollValue[types.DIDDirectory](cdc)),
		trustDeposit:        trustDeposit,
		trustRegistryKeeper: trustRegistryKeeper,
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

func (k Keeper) Logger() log.Logger {
	return k.logger.With("module", fmt.Sprintf("x/%s", types.ModuleName))
}
