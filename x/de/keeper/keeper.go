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

	// permKeeper is used by [MOD-DE-MSG-5] and [MOD-DE-MSG-6] to load
	// permission data for VS operator authorization management.
	// This is a pointer-to-interface so that SetPermKeeper (called after
	// depinject) propagates to all copies of the Keeper (value type).
	permKeeper *types.PermKeeper

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

	// Allocate the permKeeper slot on the heap so all value-copies of
	// this Keeper share the same pointer. SetPermKeeper writes into it
	// after depinject completes, and every copy (including the one held
	// by AppModule / msgServer) sees the update.
	var pkSlot types.PermKeeper

	k := Keeper{
		storeService: storeService,
		cdc:          cdc,
		addressCodec: addressCodec,
		authority:    authority,
		permKeeper:   &pkSlot,

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

// SetPermKeeper sets the permission keeper after initialization.
// This is done post-depinject to avoid a cyclic dependency
// (TR → DE → Perm → TR). The value is stored behind a pointer so
// the mutation propagates to all value-copies of the Keeper.
func (k *Keeper) SetPermKeeper(pk types.PermKeeper) {
	*k.permKeeper = pk
}
