package keeper

import (
	"context"
	"errors"
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

	Schema collections.Schema
	Params collections.Item[types.Params]

	// OperatorAuthorization: keyed by its own uint64 id; (corporation_id,
	// operator) is a unique secondary index. Runtime spend balances live
	// on-object (remaining_spend / remaining_fee_spend).
	OperatorAuthorizations        collections.Map[uint64, types.OperatorAuthorization]
	OperatorAuthorizationByCorpOp collections.Map[collections.Pair[uint64, string], uint64]
	OperatorAuthorizationSeq      collections.Sequence

	// FeeGrant: composite key (grantor_corporation_id, grantee).
	FeeGrants collections.Map[collections.Pair[uint64, string], types.FeeGrant]

	// VSOperatorAuthorization: keyed by its own uint64 id; (corporation_id,
	// vs_operator) is a unique secondary index; participant_id -> vsoa.id is a
	// tertiary index for global participant_id uniqueness and MSG-6 / MSG-9.
	VSOperatorAuthorizations collections.Map[uint64, types.VSOperatorAuthorization]
	VSOAByCorpOp             collections.Map[collections.Pair[uint64, string], uint64]
	VSOAByParticipant        collections.Map[uint64, uint64]
	VSOASeq                  collections.Sequence

	// corpRef backs AUTHZ-CHECK-5; wired post-construction via
	// SetCorporationKeeper to break the MOD-DE ↔ MOD-CO cycle (#308).
	corpRef *corpKeeperRef
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

	corpOpKeyCodec := collections.PairKeyCodec(collections.Uint64Key, collections.StringKey)

	k := Keeper{
		storeService: storeService,
		cdc:          cdc,
		addressCodec: addressCodec,
		authority:    authority,
		corpRef:      &corpKeeperRef{K: StubCorporationKeeper{}},

		Params: collections.NewItem(sb, types.ParamsKey, "params", codec.CollValue[types.Params](cdc)),

		OperatorAuthorizations: collections.NewMap(sb, types.OperatorAuthorizationKey, "operator_authorization",
			collections.Uint64Key, codec.CollValue[types.OperatorAuthorization](cdc)),
		OperatorAuthorizationByCorpOp: collections.NewMap(sb, types.OperatorAuthorizationByCorpOpKey, "operator_authorization_by_corp_op",
			corpOpKeyCodec, collections.Uint64Value),
		OperatorAuthorizationSeq: collections.NewSequence(sb, types.OperatorAuthorizationSeqKey, "operator_authorization_seq"),

		FeeGrants: collections.NewMap(sb, types.FeeGrantKey, "fee_grant",
			corpOpKeyCodec, codec.CollValue[types.FeeGrant](cdc)),

		VSOperatorAuthorizations: collections.NewMap(sb, types.VSOperatorAuthorizationKey, "vs_operator_authorization",
			collections.Uint64Key, codec.CollValue[types.VSOperatorAuthorization](cdc)),
		VSOAByCorpOp: collections.NewMap(sb, types.VSOAByCorpOpKey, "vsoa_by_corp_op",
			corpOpKeyCodec, collections.Uint64Value),
		VSOAByParticipant: collections.NewMap(sb, types.VSOAByParticipantKey, "vsoa_by_participant",
			collections.Uint64Key, collections.Uint64Value),
		VSOASeq: collections.NewSequence(sb, types.VSOASeqKey, "vsoa_seq"),
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

// SetCorporationKeeper wires the real MOD-CO keeper after both keepers exist.
// The receiver is by-value because the inner *corpKeeperRef is shared by all
// keeper copies (msg server, query server, etc.). MUST be called once during
// app init before any handler runs (cycle break for #308).
func (k Keeper) SetCorporationKeeper(c types.CorporationKeeper) {
	k.corpRef.K = c
}

// corporationKeeper returns the wired CorporationKeeper (real or stub) used by
// AUTHZ-CHECK-5 on delegable MOD-DE messages.
func (k Keeper) corporationKeeper() types.CorporationKeeper {
	return k.corpRef.K
}

// nextOperatorAuthorizationID returns a fresh, 1-based OperatorAuthorization id.
func (k Keeper) nextOperatorAuthorizationID(ctx context.Context) (uint64, error) {
	n, err := k.OperatorAuthorizationSeq.Next(ctx)
	if err != nil {
		return 0, err
	}
	return n + 1, nil
}

// nextVSOAID returns a fresh, 1-based VSOperatorAuthorization id.
func (k Keeper) nextVSOAID(ctx context.Context) (uint64, error) {
	n, err := k.VSOASeq.Next(ctx)
	if err != nil {
		return 0, err
	}
	return n + 1, nil
}

// getOperatorAuthorizationByCorpOp loads the OperatorAuthorization indexed by
// the (corporation_id, operator) secondary index. Returns found=false when no
// entry exists.
func (k Keeper) getOperatorAuthorizationByCorpOp(ctx context.Context, corporationID uint64, operator string) (types.OperatorAuthorization, bool, error) {
	id, err := k.OperatorAuthorizationByCorpOp.Get(ctx, collections.Join(corporationID, operator))
	if err != nil {
		if errors.Is(err, collections.ErrNotFound) {
			return types.OperatorAuthorization{}, false, nil
		}
		return types.OperatorAuthorization{}, false, err
	}
	oa, err := k.OperatorAuthorizations.Get(ctx, id)
	if err != nil {
		return types.OperatorAuthorization{}, false, err
	}
	return oa, true, nil
}

// getVSOAByCorpOp loads the VSOperatorAuthorization indexed by the
// (corporation_id, vs_operator) secondary index. Returns found=false when no
// entry exists.
func (k Keeper) getVSOAByCorpOp(ctx context.Context, corporationID uint64, vsOperator string) (types.VSOperatorAuthorization, bool, error) {
	id, err := k.VSOAByCorpOp.Get(ctx, collections.Join(corporationID, vsOperator))
	if err != nil {
		if errors.Is(err, collections.ErrNotFound) {
			return types.VSOperatorAuthorization{}, false, nil
		}
		return types.VSOperatorAuthorization{}, false, err
	}
	vsoa, err := k.VSOperatorAuthorizations.Get(ctx, id)
	if err != nil {
		return types.VSOperatorAuthorization{}, false, err
	}
	return vsoa, true, nil
}
