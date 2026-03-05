package keeper

import (
	"context"
	"fmt"
	"time"

	"cosmossdk.io/collections"
	"cosmossdk.io/core/address"
	corestore "cosmossdk.io/core/store"
	"github.com/cosmos/cosmos-sdk/codec"
	sdk "github.com/cosmos/cosmos-sdk/types"

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

// GrantVSOperatorAuthorization grants a VS operator the authorization to call
// CreateOrUpdatePermissionSession on behalf of the authority for a given permission.
// TODO(MOD-DE-MSG-5): Implement full VS operator authorization logic.
func (k Keeper) GrantVSOperatorAuthorization(
	ctx context.Context,
	authority string,
	vsOperator string,
	permissionID uint64,
	spendLimit sdk.Coins,
	withFeegrant bool,
	feeSpendLimit sdk.Coins,
	spendPeriod *time.Duration,
) error {
	sdkCtx := sdk.UnwrapSDKContext(ctx)

	// Store the VS operator authorization
	vsKey := collections.Join(authority, vsOperator)
	vsAuth := types.VSOperatorAuthorization{
		Authority:  authority,
		VsOperator: vsOperator,
	}
	if err := k.VSOperatorAuthorizations.Set(sdkCtx, vsKey, vsAuth); err != nil {
		return fmt.Errorf("failed to store VS operator authorization: %w", err)
	}

	return nil
}

// RevokeVSOperatorAuthorization removes a VS operator's authorization for a given permission.
func (k Keeper) RevokeVSOperatorAuthorization(
	ctx context.Context,
	authority string,
	vsOperator string,
	permissionID uint64,
) error {
	sdkCtx := sdk.UnwrapSDKContext(ctx)

	// Remove the VS operator authorization
	vsKey := collections.Join(authority, vsOperator)
	has, err := k.VSOperatorAuthorizations.Has(sdkCtx, vsKey)
	if err != nil {
		return fmt.Errorf("failed to check VS operator authorization: %w", err)
	}
	if !has {
		return nil // Already revoked or never existed
	}
	if err := k.VSOperatorAuthorizations.Remove(sdkCtx, vsKey); err != nil {
		return fmt.Errorf("failed to remove VS operator authorization: %w", err)
	}

	// Revoke associated fee allowance if any
	if err := k.RevokeFeeAllowance(sdkCtx, authority, vsOperator); err != nil {
		return fmt.Errorf("failed to revoke fee allowance: %w", err)
	}

	return nil
}
