package keeper

import (
	"cosmossdk.io/collections"
	"cosmossdk.io/core/address"
	corestore "cosmossdk.io/core/store"
	"cosmossdk.io/log"
	"cosmossdk.io/math"
	"fmt"
	"github.com/cosmos/cosmos-sdk/codec"
	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/verana-labs/verana/x/td/types"
)

type Keeper struct {
	storeService corestore.KVStoreService
	cdc          codec.Codec
	addressCodec address.Codec
	// Address capable of executing a MsgUpdateParams message.
	// Typically, this should be the x/gov module account.
	logger    log.Logger
	authority []byte

	Schema       collections.Schema
	TrustDeposit collections.Map[string, types.TrustDeposit]
	// external keeper
	bankKeeper types.BankKeeper
}

func NewKeeper(
	storeService corestore.KVStoreService,
	cdc codec.Codec,
	addressCodec address.Codec,
	logger log.Logger,
	authority []byte,
	bankKeeper types.BankKeeper,

) Keeper {
	if _, err := addressCodec.BytesToString(authority); err != nil {
		panic(fmt.Sprintf("invalid authority address %s: %s", authority, err))
	}

	sb := collections.NewSchemaBuilder(storeService)

	k := Keeper{
		storeService: storeService,
		cdc:          cdc,
		addressCodec: addressCodec,
		authority:    authority,

		logger:       logger,
		TrustDeposit: collections.NewMap(sb, types.TrustDepositKey, "trust_deposit", collections.StringKey, codec.CollValue[types.TrustDeposit](cdc)),
		bankKeeper:   bankKeeper,
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

// Logger returns a module-specific logger.
func (k Keeper) Logger() log.Logger {
	return k.logger.With("module", fmt.Sprintf("x/%s", types.ModuleName))
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
