package types

import (
	store "cosmossdk.io/store/types"
	upgradetypes "cosmossdk.io/x/upgrade/types"
	tmproto "github.com/cometbft/cometbft/proto/tendermint/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/types/module"
	authkeeper "github.com/cosmos/cosmos-sdk/x/auth/keeper"
	bankkeeper "github.com/cosmos/cosmos-sdk/x/bank/keeper"
	credentialschemakeeper "github.com/verana-labs/verana/x/credentialschema/keeper"
	diddirectorykeeper "github.com/verana-labs/verana/x/diddirectory/keeper"
	permission "github.com/verana-labs/verana/x/permission/keeper"
	trustdeposit "github.com/verana-labs/verana/x/trustdeposit/keeper"
	trustregistry "github.com/verana-labs/verana/x/trustregistry/keeper"
)

type BaseAppParamManager interface {
	GetConsensusParams(ctx sdk.Context) tmproto.ConsensusParams
	StoreConsensusParams(ctx sdk.Context, cp tmproto.ConsensusParams) error
}

type AppKeepers interface {
	GetTrustRegistryKeeper() trustregistry.Keeper
	GetPermissionKeeper() permission.Keeper
	GetTrustDepositKeeper() trustdeposit.Keeper
	GetDidDirectoryKeeper() diddirectorykeeper.Keeper
	GetCredentialSchemaKeeper() credentialschemakeeper.Keeper
	GetBankKeeper() bankkeeper.Keeper
	GetAccountKeeper() authkeeper.AccountKeeper
}

type Upgrade struct {
	UpgradeName          string
	CreateUpgradeHandler func(*module.Manager, module.Configurator, BaseAppParamManager, AppKeepers) upgradetypes.UpgradeHandler
	StoreUpgrades        store.StoreUpgrades
}
