package keeper

import (
	"fmt"

	"cosmossdk.io/collections"
	"cosmossdk.io/math"

	"cosmossdk.io/core/store"
	"cosmossdk.io/log"
	"github.com/cosmos/cosmos-sdk/codec"
	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/verana-labs/verana/x/td/types"
)

type (
	Keeper struct {
		cdc          codec.BinaryCodec
		storeService store.KVStoreService
		logger       log.Logger

		// the address capable of executing a MsgUpdateParams message. Typically, this
		// should be the x/gov module account.
		authority string
		// state — keyed by corporation_id (uint64) per v4-rc3
		TrustDeposit collections.Map[uint64, types.TrustDeposit]
		Dust         collections.Item[string] // Accumulated fractional yield (stored as string)
		// external keeper
		bankKeeper       types.BankKeeper
		mintKeeper       types.MintKeeper
		delegationKeeper types.DelegationKeeper
		coKeeper         types.CorporationKeeper
	}
)

func NewKeeper(
	cdc codec.BinaryCodec,
	storeService store.KVStoreService,
	logger log.Logger,
	authority string,
	bankKeeper types.BankKeeper,
	mintKeeper types.MintKeeper,
	delegationKeeper types.DelegationKeeper,
	coKeeper types.CorporationKeeper,
) Keeper {
	sb := collections.NewSchemaBuilder(storeService)

	if _, err := sdk.AccAddressFromBech32(authority); err != nil {
		panic(fmt.Sprintf("invalid authority address: %s", authority))
	}

	return Keeper{
		cdc:              cdc,
		storeService:     storeService,
		authority:        authority,
		logger:           logger,
		TrustDeposit:     collections.NewMap(sb, types.TrustDepositKey, "trust_deposit", collections.Uint64Key, codec.CollValue[types.TrustDeposit](cdc)),
		Dust:             collections.NewItem(sb, types.DustKey, "dust", collections.StringValue),
		bankKeeper:       bankKeeper,
		mintKeeper:       mintKeeper,
		delegationKeeper: delegationKeeper,
		coKeeper:         coKeeper,
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


// GetTrustDepositMap returns the TrustDeposit collections.Map for migration purposes.
func (k Keeper) GetTrustDepositMap() collections.Map[uint64, types.TrustDeposit] {
	return k.TrustDeposit
}

// resolveCorporationID resolves a corporation policy_address (account) to its
// registered Corporation id. Trust deposits are keyed by corporation_id, but the
// account is still needed for bank transfers, so callers pass the account and use
// the returned id only as the storage key.
func (k Keeper) resolveCorporationID(ctx sdk.Context, account string) (uint64, error) {
	co, err := k.coKeeper.ResolveCorporationByPolicyAddress(ctx, account)
	if err != nil {
		return 0, err
	}
	return co.Id, nil
}

func (k Keeper) GetTrustDepositRate(ctx sdk.Context) math.LegacyDec {
	params := k.GetParams(ctx)
	return params.TrustDepositRate
}

func (k Keeper) GetUserAgentRewardRate(ctx sdk.Context) math.LegacyDec {
	params := k.GetParams(ctx)
	return params.UserAgentRewardRate
}

func (k Keeper) GetWalletUserAgentRewardRate(ctx sdk.Context) math.LegacyDec {
	params := k.GetParams(ctx)
	return params.WalletUserAgentRewardRate
}

func (k Keeper) GetTrustDepositShareValue(ctx sdk.Context) math.LegacyDec {
	params := k.GetParams(ctx)
	return params.TrustDepositShareValue
}
