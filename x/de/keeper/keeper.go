package keeper

import (
	"fmt"

	"cosmossdk.io/collections"
	"cosmossdk.io/core/address"
	corestore "cosmossdk.io/core/store"
	"github.com/cosmos/cosmos-sdk/codec"

	"github.com/verana-labs/verana/x/de/types"
)

type Keeper struct {
	storeService corestore.KVStoreService
	cdc          codec.Codec
	addressCodec address.Codec
	// Address capable of executing a MsgUpdateParams message.
	// Typically, this should be the x/gov module account.
	authority []byte

	Schema                   collections.Schema
	Params                   collections.Item[types.Params]
	OperatorAuthorizations   collections.Map[collections.Pair[string, string], types.OperatorAuthorization]
	FeeGrants                collections.Map[collections.Pair[string, string], types.FeeGrant]
	VSOperatorAuthorizations collections.Map[collections.Pair[string, string], types.VSOperatorAuthorization]
}

func NewKeeper(
	storeService corestore.KVStoreService,
	cdc codec.Codec,
	addressCodec address.Codec,
	authority []byte,

) Keeper {
	if _, err := addressCodec.BytesToString(authority); err != nil {
		panic(fmt.Sprintf("invalid authority address %s: %s", authority, err))
	}

	sb := collections.NewSchemaBuilder(storeService)

	pairKeyCodec := collections.PairKeyCodec(collections.StringKey, collections.StringKey)

	k := Keeper{
		storeService: storeService,
		cdc:          cdc,
		addressCodec: addressCodec,
		authority:    authority,

		Params: collections.NewItem(sb, types.ParamsKey, "params", codec.CollValue[types.Params](cdc)),
		OperatorAuthorizations: collections.NewMap(sb, types.OperatorAuthorizationKey, "operator_authorization",
			pairKeyCodec, codec.CollValue[types.OperatorAuthorization](cdc)),
		FeeGrants: collections.NewMap(sb, types.FeeGrantKey, "fee_grant",
			pairKeyCodec, codec.CollValue[types.FeeGrant](cdc)),
		VSOperatorAuthorizations: collections.NewMap(sb, types.VSOperatorAuthorizationKey, "vs_operator_authorization",
			pairKeyCodec, codec.CollValue[types.VSOperatorAuthorization](cdc)),
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
