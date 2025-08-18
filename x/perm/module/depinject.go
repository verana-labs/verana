package perm

import (
	"cosmossdk.io/core/address"
	"cosmossdk.io/core/appmodule"
	"cosmossdk.io/core/store"
	"cosmossdk.io/depinject"
	"cosmossdk.io/depinject/appconfig"
	"cosmossdk.io/log"
	"github.com/cosmos/cosmos-sdk/codec"
	authtypes "github.com/cosmos/cosmos-sdk/x/auth/types"

	"github.com/verana-labs/verana/x/perm/keeper"
	"github.com/verana-labs/verana/x/perm/types"
)

var _ depinject.OnePerModuleType = AppModule{}

// IsOnePerModuleType implements the depinject.OnePerModuleType interface.
func (AppModule) IsOnePerModuleType() {}

func init() {
	appconfig.Register(
		&types.Module{},
		appconfig.Provide(ProvideModule),
	)
}

type ModuleInputs struct {
	depinject.In

	Config       *types.Module
	StoreService store.KVStoreService
	Cdc          codec.Codec
	AddressCodec address.Codec
	Logger       log.Logger

	AuthKeeper             types.AuthKeeper
	BankKeeper             types.BankKeeper
	CredentialSchemaKeeper types.CredentialSchemaKeeper `optional:"true"`
	TrustRegistryKeeper    types.TrustRegistryKeeper    `optional:"true"`
	TrustDepositKeeper     types.TrustDepositKeeper     `optional:"true"`
}

type ModuleOutputs struct {
	depinject.Out

	PermKeeper keeper.Keeper
	Module     appmodule.AppModule
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
		in.Logger,
		authority,
		in.CredentialSchemaKeeper,
		in.TrustRegistryKeeper,
		in.TrustDepositKeeper,
		in.BankKeeper,
	)
	m := NewAppModule(in.Cdc, k, in.AuthKeeper, in.BankKeeper)

	return ModuleOutputs{PermKeeper: k, Module: m}
}
