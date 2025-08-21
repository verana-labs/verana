package v2

import (
	"context"

	upgradetypes "cosmossdk.io/x/upgrade/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/types/module"
	"github.com/verana-labs/verana-blockchain/app/upgrades/types"
	diddirectory "github.com/verana-labs/verana-blockchain/x/dd/module"
	didtypes "github.com/verana-labs/verana-blockchain/x/dd/types"
)

func CreateUpgradeHandler(
	mm *module.Manager,
	configurator module.Configurator,
	_ types.BaseAppParamManager,
	keepers types.AppKeepers,
) upgradetypes.UpgradeHandler {
	return func(context context.Context, plan upgradetypes.Plan, fromVM module.VersionMap) (module.VersionMap, error) {
		ctx := sdk.UnwrapSDKContext(context)
		migrations, err := mm.RunMigrations(ctx, configurator, fromVM)
		if err != nil {
			return nil, err
		}

		diddirectory.InitGenesis(ctx, keepers.GetDidDirectoryKeeper(), *didtypes.DefaultGenesis())
		return migrations, nil
	}
}
