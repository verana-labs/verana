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
		// state
		TrustDeposit collections.Map[string, types.TrustDeposit]
		// external keeper
		bankKeeper types.BankKeeper
	}
)

func NewKeeper(
	cdc codec.BinaryCodec,
	storeService store.KVStoreService,
	logger log.Logger,
	authority string,
	bankKeeper types.BankKeeper,

) Keeper {
	sb := collections.NewSchemaBuilder(storeService)

	if _, err := sdk.AccAddressFromBech32(authority); err != nil {
		panic(fmt.Sprintf("invalid authority address: %s", authority))
	}

	return Keeper{
		cdc:          cdc,
		storeService: storeService,
		authority:    authority,
		logger:       logger,
		TrustDeposit: collections.NewMap(sb, types.TrustDepositKey, "trust_deposit", collections.StringKey, codec.CollValue[types.TrustDeposit](cdc)),
		bankKeeper:   bankKeeper,
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

// GetStoreService returns the store service for migration purposes.
func (k Keeper) GetStoreService() store.KVStoreService {
	return k.storeService
}

// GetCodec returns the codec for migration purposes.
func (k Keeper) GetCodec() codec.BinaryCodec {
	return k.cdc
}

// GetLogger returns the logger for migration purposes.
func (k Keeper) GetLogger() interface {
	Info(msg string, keyvals ...interface{})
} {
	return k.Logger()
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
