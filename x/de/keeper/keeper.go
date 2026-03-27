package keeper

import (
	"context"
	"fmt"
	"strconv"

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

// AddPermToVSOA adds a permission ID to the VSOperatorAuthorization for the
// given (authority, vsOperator) pair, creating the entry if it doesn't exist.
// It also checks mutual exclusivity: an OperatorAuthorization must NOT exist
// for the same pair. [MOD-DE-MSG-5 storage]
func (k Keeper) AddPermToVSOA(ctx context.Context, authority, vsOperator string, permID uint64) error {
	sdkCtx := sdk.UnwrapSDKContext(ctx)

	// Mutual exclusivity check
	oaKey := collections.Join(authority, vsOperator)
	hasOA, err := k.OperatorAuthorizations.Has(sdkCtx, oaKey)
	if err != nil {
		return fmt.Errorf("failed to check OperatorAuthorization: %w", err)
	}
	if hasOA {
		return types.ErrOperatorAuthzExistsMutex
	}

	vsKey := collections.Join(authority, vsOperator)
	vsoa, err := k.VSOperatorAuthorizations.Get(sdkCtx, vsKey)
	if err != nil {
		vsoa = types.VSOperatorAuthorization{
			Authority:   authority,
			VsOperator:  vsOperator,
			Permissions: []uint64{},
		}
	}

	// Avoid duplicates
	for _, pid := range vsoa.Permissions {
		if pid == permID {
			return nil // already present
		}
	}
	vsoa.Permissions = append(vsoa.Permissions, permID)

	if err := k.VSOperatorAuthorizations.Set(sdkCtx, vsKey, vsoa); err != nil {
		return fmt.Errorf("failed to set VSOperatorAuthorization: %w", err)
	}

	// Emit event
	sdkCtx.EventManager().EmitEvent(
		sdk.NewEvent(
			types.EventTypeGrantVSOperatorAuthorization,
			sdk.NewAttribute(types.AttributeKeyAuthority, authority),
			sdk.NewAttribute(types.AttributeKeyVsOperator, vsOperator),
			sdk.NewAttribute(types.AttributeKeyPermissionID, strconv.FormatUint(permID, 10)),
			sdk.NewAttribute(types.AttributeKeyTimestamp, sdkCtx.BlockTime().String()),
		),
	)

	return nil
}

// RemovePermFromVSOA removes a permission ID from the VSOperatorAuthorization.
// If no permissions remain, the entry is deleted. Returns the remaining permission IDs.
// [MOD-DE-MSG-6 storage]
func (k Keeper) RemovePermFromVSOA(ctx context.Context, authority, vsOperator string, permID uint64) ([]uint64, error) {
	sdkCtx := sdk.UnwrapSDKContext(ctx)

	vsKey := collections.Join(authority, vsOperator)
	vsoa, err := k.VSOperatorAuthorizations.Get(sdkCtx, vsKey)
	if err != nil {
		return nil, nil // doesn't exist, nothing to do
	}

	newPerms := make([]uint64, 0, len(vsoa.Permissions))
	for _, pid := range vsoa.Permissions {
		if pid != permID {
			newPerms = append(newPerms, pid)
		}
	}
	vsoa.Permissions = newPerms

	if len(vsoa.Permissions) == 0 {
		if err := k.VSOperatorAuthorizations.Remove(sdkCtx, vsKey); err != nil {
			return nil, fmt.Errorf("failed to remove VSOperatorAuthorization: %w", err)
		}
	} else {
		if err := k.VSOperatorAuthorizations.Set(sdkCtx, vsKey, vsoa); err != nil {
			return nil, fmt.Errorf("failed to update VSOperatorAuthorization: %w", err)
		}
	}

	// Emit event
	sdkCtx.EventManager().EmitEvent(
		sdk.NewEvent(
			types.EventTypeRevokeVSOperatorAuthorization,
			sdk.NewAttribute(types.AttributeKeyAuthority, authority),
			sdk.NewAttribute(types.AttributeKeyVsOperator, vsOperator),
			sdk.NewAttribute(types.AttributeKeyPermissionID, strconv.FormatUint(permID, 10)),
			sdk.NewAttribute(types.AttributeKeyTimestamp, sdkCtx.BlockTime().String()),
		),
	)

	return newPerms, nil
}

// HasOperatorAuthorization checks if an OperatorAuthorization exists for the given pair.
// Used by the perm module for mutual exclusivity checks in [MOD-DE-MSG-5].
func (k Keeper) HasOperatorAuthorization(ctx context.Context, authority, operator string) (bool, error) {
	sdkCtx := sdk.UnwrapSDKContext(ctx)
	key := collections.Join(authority, operator)
	return k.OperatorAuthorizations.Has(sdkCtx, key)
}

// CheckVSOperatorAuthorization checks if a VS operator is authorized to act on behalf of the authority.
// [AUTHZ-CHECK-3]
func (k Keeper) CheckVSOperatorAuthorization(ctx context.Context, authority, vsOperator string) error {
	sdkCtx := sdk.UnwrapSDKContext(ctx)
	vsKey := collections.Join(authority, vsOperator)
	has, err := k.VSOperatorAuthorizations.Has(sdkCtx, vsKey)
	if err != nil {
		return fmt.Errorf("failed to check VS operator authorization: %w", err)
	}
	if !has {
		return fmt.Errorf("VS operator %s is not authorized for authority %s", vsOperator, authority)
	}
	return nil
}

// GetVSOAPermissions returns the permission IDs for a VSOperatorAuthorization.
// Returns nil if the VSOA doesn't exist.
func (k Keeper) GetVSOAPermissions(ctx context.Context, authority, vsOperator string) ([]uint64, error) {
	sdkCtx := sdk.UnwrapSDKContext(ctx)
	vsKey := collections.Join(authority, vsOperator)
	vsoa, err := k.VSOperatorAuthorizations.Get(sdkCtx, vsKey)
	if err != nil {
		return nil, nil
	}
	return vsoa.Permissions, nil
}
