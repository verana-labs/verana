package v091

import (
	"context"

	upgradetypes "cosmossdk.io/x/upgrade/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/types/module"
	"github.com/verana-labs/verana/app/upgrades/types"
	tdkeeper "github.com/verana-labs/verana/x/td/keeper"
)

func CreateUpgradeHandler(
	mm *module.Manager,
	configurator module.Configurator,
	_ types.BaseAppParamManager,
	keepers types.AppKeepers,
) upgradetypes.UpgradeHandler {
	return func(context context.Context, plan upgradetypes.Plan, fromVM module.VersionMap) (module.VersionMap, error) {
		ctx := sdk.UnwrapSDKContext(context)

		// Migrate TrustDeposit Share field from uint64 to LegacyDec
		trustDepositKeeper := keepers.GetTrustDepositKeeper()
		migrator := tdkeeper.NewMigrator(trustDepositKeeper)
		if err := migrator.Migrate1to2(ctx); err != nil {
			return nil, err
		}

		// Run standard migrations
		return mm.RunMigrations(ctx, configurator, fromVM)
	}
}
