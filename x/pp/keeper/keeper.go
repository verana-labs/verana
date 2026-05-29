package keeper

import (
	"fmt"
	"time"

	"cosmossdk.io/collections"

	"cosmossdk.io/core/store"
	"cosmossdk.io/log"
	"github.com/cosmos/cosmos-sdk/codec"
	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/verana-labs/verana/x/pp/types"
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
		Participant        collections.Map[uint64, types.Participant]
		ParticipantCounter collections.Item[uint64]
		ParticipantSession collections.Map[string, types.ParticipantSession]

		// external keeper
		credentialSchemaKeeper types.CredentialSchemaKeeper
		ecosystemKeeper        types.EcosystemKeeper
		coKeeper               types.CorporationKeeper
		trustDeposit           types.TrustDepositKeeper
		bankKeeper             types.BankKeeper
		delegationKeeper       types.DelegationKeeper
		digestKeeper           types.DigestKeeper
	}
)

func NewKeeper(
	cdc codec.BinaryCodec,
	storeService store.KVStoreService,
	logger log.Logger,
	authority string,
	credentialSchemaKeeper types.CredentialSchemaKeeper,
	ecosystemKeeper types.EcosystemKeeper,
	coKeeper types.CorporationKeeper,
	trustDeposit types.TrustDepositKeeper,
	bankKeeper types.BankKeeper,
	delegationKeeper types.DelegationKeeper,
	digestKeeper types.DigestKeeper,
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
		Participant:            collections.NewMap(sb, types.ParticipantKey, "perm", collections.Uint64Key, codec.CollValue[types.Participant](cdc)),
		ParticipantCounter:     collections.NewItem(sb, types.ParticipantCounterKey, "participant_counter", collections.Uint64Value),
		ParticipantSession:     collections.NewMap(sb, types.ParticipantSessionKey, "permission_session", collections.StringKey, codec.CollValue[types.ParticipantSession](cdc)),
		credentialSchemaKeeper: credentialSchemaKeeper,
		ecosystemKeeper:        ecosystemKeeper,
		coKeeper:               coKeeper,
		trustDeposit:           trustDeposit,
		bankKeeper:             bankKeeper,
		delegationKeeper:       delegationKeeper,
		digestKeeper:           digestKeeper,
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

func (k Keeper) GetParticipantByID(ctx sdk.Context, id uint64) (types.Participant, error) {
	return k.Participant.Get(ctx, id)
}

// CreatePermission creates a new perm and returns its ID
func (k Keeper) CreatePermission(ctx sdk.Context, perm types.Participant) (uint64, error) {
	id, err := k.getNextPermissionID(ctx)
	if err != nil {
		return 0, err
	}
	perm.Id = id
	if err := k.Participant.Set(ctx, id, perm); err != nil {
		return 0, err
	}

	return id, nil
}

// getNextPermissionID gets the next available perm ID
func (k Keeper) getNextPermissionID(ctx sdk.Context) (uint64, error) {
	id, err := k.ParticipantCounter.Get(ctx)
	if err != nil {
		id = 0
	}

	nextID := id + 1
	err = k.ParticipantCounter.Set(ctx, nextID)
	if err != nil {
		return 0, fmt.Errorf("failed to set perm counter: %w", err)
	}

	return nextID, nil
}

func (k Keeper) UpdatePermission(ctx sdk.Context, perm types.Participant) error {
	return k.Participant.Set(ctx, perm.Id, perm)
}

// IsValidPermission checks if a perm is valid for a given time
// A valid perm (ACTIVE state):
// - Is currently effective (effective_from must be set and effective_from ≤ now < effective_until)
// - Is not revoked
// - Is not slashed
// - Is not repaid
// According to the spec, if validator permission is INACTIVE (not valid), it must abort.
// INACTIVE means: effective_from is null OR effective_from equals now() exactly (not before).
func IsValidPermission(perm types.Participant, checkTime time.Time) error {
	// Check if perm is repaid (REPAID state)
	if perm.Repaid != nil {
		return fmt.Errorf("perm is repaid since %v", perm.Repaid)
	}

	// Check if perm is slashed (SLASHED state)
	if perm.Slashed != nil {
		return fmt.Errorf("perm is slashed since %v", perm.Slashed)
	}

	// Check if perm is revoked (REVOKED state)
	// Spec: "else if `revoked` is lower than now(), => `perm_state` is `REVOKED`"
	// This means revoked < now(), so we check checkTime.After(*perm.Revoked)
	if perm.Revoked != nil && checkTime.After(*perm.Revoked) {
		return fmt.Errorf("perm is revoked since %v", perm.Revoked)
	}

	// Check if perm is expired (EXPIRED state)
	if perm.EffectiveUntil != nil && !checkTime.Before(*perm.EffectiveUntil) {
		return fmt.Errorf("perm expired: ended at %v", perm.EffectiveUntil)
	}

	// Check if perm is in FUTURE state (effective_from is after now)
	if perm.EffectiveFrom != nil && checkTime.Before(*perm.EffectiveFrom) {
		return fmt.Errorf("perm not yet effective: begins at %v", perm.EffectiveFrom)
	}

	// Check if perm is INACTIVE (effective_from is null OR equals now exactly)
	// For ACTIVE state, effective_from must be set and must be before or equal to now
	if perm.EffectiveFrom == nil {
		return fmt.Errorf("perm is INACTIVE: effective_from is null")
	}

	// At this point, effective_from is set and checkTime is not before it
	// This means effective_from <= now, which is required for ACTIVE state
	// The permission is valid (ACTIVE)

	return nil
}

func formatTimePtr(t *time.Time) string {
	if t == nil {
		return ""
	}
	return t.String()
}
