package di

import (
	"cosmossdk.io/core/address"
	"cosmossdk.io/core/appmodule"
	"cosmossdk.io/core/store"
	"cosmossdk.io/depinject"
	"cosmossdk.io/depinject/appconfig"
	"github.com/cosmos/cosmos-sdk/codec"
	authtypes "github.com/cosmos/cosmos-sdk/x/auth/types"

	cokeeper "github.com/verana-labs/verana/x/co/keeper"
	"github.com/verana-labs/verana/x/di/keeper"
	"github.com/verana-labs/verana/x/di/types"
)

var _ depinject.OnePerModuleType = AppModule{}

// IsOnePerModuleType implements the depinject.OnePerModuleType interface.
func (AppModule) IsOnePerModuleType() {}

func init() {
	appconfig.Register(
		&types.Module{},
		appconfig.Provide(
			ProvideModule,
			ProvideCorporationKeeperForDI,
		),
	)
}

// ProvideCorporationKeeperForDI supplies ditypes.CorporationKeeper for MOD-DI
// AUTHZ-CHECK-5, sourced from the concrete x/co keeper.
func ProvideCorporationKeeperForDI(co cokeeper.Keeper) types.CorporationKeeper {
	return keeper.NewCoAsDICorporationKeeper(co)
}

type ModuleInputs struct {
	depinject.In

	Config       *types.Module
	StoreService store.KVStoreService
	Cdc          codec.Codec
	AddressCodec address.Codec

	AuthKeeper        types.AuthKeeper
	BankKeeper        types.BankKeeper
	DelegationKeeper  types.DelegationKeeper
	CorporationKeeper types.CorporationKeeper
}

type ModuleOutputs struct {
	depinject.Out

	DiKeeper keeper.Keeper
	Module   appmodule.AppModule
}

func ProvideModule(in ModuleInputs) ModuleOutputs {
	// default to governance authority if not provided
	authority := authtypes.NewModuleAddress(types.GovModuleName)
	if in.Config.Authority != "" {
		authority = authtypes.NewModuleAddressOrBech32Address(in.Config.Authority)
	}
	k := keeper.NewKeeper(
		in.StoreService,
		in.Cdc,
		in.AddressCodec,
		authority,
		in.DelegationKeeper,
		in.CorporationKeeper,
	)
	m := NewAppModule(in.Cdc, k, in.AuthKeeper, in.BankKeeper)

	return ModuleOutputs{DiKeeper: k, Module: m}
}
