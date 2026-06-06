package types

import (
	store "cosmossdk.io/store/types"
	upgradetypes "cosmossdk.io/x/upgrade/types"
	tmproto "github.com/cometbft/cometbft/proto/tendermint/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/types/module"
	authkeeper "github.com/cosmos/cosmos-sdk/x/auth/keeper"
	bankkeeper "github.com/cosmos/cosmos-sdk/x/bank/keeper"
	govkeeper "github.com/cosmos/cosmos-sdk/x/gov/keeper"
	credentialschemakeeper "github.com/verana-labs/verana/x/cs/keeper"
	ecosystem "github.com/verana-labs/verana/x/ec/keeper"
	participant "github.com/verana-labs/verana/x/pp/keeper"
	trustdeposit "github.com/verana-labs/verana/x/td/keeper"
)

type BaseAppParamManager interface {
	GetConsensusParams(ctx sdk.Context) tmproto.ConsensusParams
	StoreConsensusParams(ctx sdk.Context, cp tmproto.ConsensusParams) error
}

type AppKeepers interface {
	GetEcosystemKeeper() ecosystem.Keeper
	GetParticipantKeeper() participant.Keeper
	GetTrustDepositKeeper() trustdeposit.Keeper
	GetCredentialSchemaKeeper() credentialschemakeeper.Keeper
	GetBankKeeper() bankkeeper.Keeper
	GetAccountKeeper() authkeeper.AccountKeeper
	GetGovKeeper() *govkeeper.Keeper
}

type Upgrade struct {
	UpgradeName          string
	CreateUpgradeHandler func(*module.Manager, module.Configurator, BaseAppParamManager, AppKeepers) upgradetypes.UpgradeHandler
	StoreUpgrades        store.StoreUpgrades
}
