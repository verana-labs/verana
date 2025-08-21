package keeper

import (
	"fmt"
	"time"

	"cosmossdk.io/collections"

	"cosmossdk.io/core/store"
	"cosmossdk.io/log"
	"github.com/cosmos/cosmos-sdk/codec"
	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/verana-labs/verana/x/perm/types"
)

type (
	Keeper struct {
		cdc          codec.BinaryCodec
		storeService store.KVStoreService
		logger       log.Logger

		// the address capable of executing a MsgUpdateParams message. Typically, this
		// should be the x/gov module account.
		authority string
		// state
		Permission        collections.Map[uint64, types.Permission]
		PermissionCounter collections.Item[uint64]
		PermissionSession collections.Map[string, types.PermissionSession]

		// external keeper
		credentialSchemaKeeper types.CredentialSchemaKeeper
		trustRegistryKeeper    types.TrustRegistryKeeper
		trustDeposit           types.TrustDepositKeeper
		bankKeeper             types.BankKeeper
	}
)

func NewKeeper(
	cdc codec.BinaryCodec,
	storeService store.KVStoreService,
	logger log.Logger,
	authority string,
	credentialSchemaKeeper types.CredentialSchemaKeeper,
	trustRegistryKeeper types.TrustRegistryKeeper,
	trustDeposit types.TrustDepositKeeper,
	bankKeeper types.BankKeeper,
) Keeper {
	sb := collections.NewSchemaBuilder(storeService)

	if _, err := sdk.AccAddressFromBech32(authority); err != nil {
		panic(fmt.Sprintf("invalid authority address: %s", authority))
	}

	return Keeper{
		cdc:                    cdc,
		storeService:           storeService,
		authority:              authority,
		logger:                 logger,
		Permission:             collections.NewMap(sb, types.PermissionKey, "perm", collections.Uint64Key, codec.CollValue[types.Permission](cdc)),
		PermissionCounter:      collections.NewItem(sb, types.PermissionCounterKey, "permission_counter", collections.Uint64Value),
		PermissionSession:      collections.NewMap(sb, types.PermissionSessionKey, "permission_session", collections.StringKey, codec.CollValue[types.PermissionSession](cdc)),
		credentialSchemaKeeper: credentialSchemaKeeper,
		trustRegistryKeeper:    trustRegistryKeeper,
		trustDeposit:           trustDeposit,
		bankKeeper:             bankKeeper,
	}
}

// GetAuthority returns the module's authority.
func (k Keeper) GetAuthority() string {
	return k.authority
}

// Logger returns a module-specific logger.
func (k Keeper) Logger() log.Logger {
	return k.logger.With("module", fmt.Sprintf("x/%s", types.ModuleName))
}

func (k Keeper) GetPermissionByID(ctx sdk.Context, id uint64) (types.Permission, error) {
	return k.Permission.Get(ctx, id)
}

// CreatePermission creates a new perm and returns its ID
func (k Keeper) CreatePermission(ctx sdk.Context, perm types.Permission) (uint64, error) {
	id, err := k.getNextPermissionID(ctx)
	if err != nil {
		return 0, err
	}
	perm.Id = id
	if err := k.Permission.Set(ctx, id, perm); err != nil {
		return 0, err
	}

	return id, nil
}

// getNextPermissionID gets the next available perm ID
func (k Keeper) getNextPermissionID(ctx sdk.Context) (uint64, error) {
	id, err := k.PermissionCounter.Get(ctx)
	if err != nil {
		id = 0
	}

	nextID := id + 1
	err = k.PermissionCounter.Set(ctx, nextID)
	if err != nil {
		return 0, fmt.Errorf("failed to set perm counter: %w", err)
	}

	return nextID, nil
}

func (k Keeper) UpdatePermission(ctx sdk.Context, perm types.Permission) error {
	return k.Permission.Set(ctx, perm.Id, perm)
}

// IsValidPermission checks if a perm is valid for a given country code and time
// A valid perm:
// - Has a matching country (perm country is null or matches the provided country)
// - Is currently effective (effective_from ≤ now < effective_until)
// - Is not revoked or terminated
// - Is not slashed
func IsValidPermission(perm types.Permission, country string, checkTime time.Time) error {
	// Check country compatibility
	if perm.Country != "" && perm.Country != country {
		return fmt.Errorf("perm country mismatch: perm has %s, requested %s",
			perm.Country, country)
	}

	// Check if perm is effective at the given time
	if perm.EffectiveFrom != nil && checkTime.Before(*perm.EffectiveFrom) {
		return fmt.Errorf("perm not yet effective: begins at %v", perm.EffectiveFrom)
	}

	if perm.EffectiveUntil != nil && !checkTime.Before(*perm.EffectiveUntil) {
		return fmt.Errorf("perm expired: ended at %v", perm.EffectiveUntil)
	}

	// Check if perm is revoked
	if perm.Revoked != nil {
		return fmt.Errorf("perm is revoked since %v", perm.Revoked)
	}

	// Check if perm is terminated
	if perm.Terminated != nil {
		return fmt.Errorf("perm is terminated since %v", perm.Terminated)
	}

	// Check if perm is slashed
	if perm.SlashedDeposit > 0 {
		return fmt.Errorf("perm is slashed with amount %d", perm.SlashedDeposit)
	}

	return nil
}
