package v9

import (
	"context"

	upgradetypes "cosmossdk.io/x/upgrade/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/types/module"
	"github.com/verana-labs/verana/app/upgrades/types"
)

// CreateUpgradeHandler creates an upgrade handler for v9 upgrade.
// This upgrade migrates the TrustDeposit module from version 1 to 2,
// converting Share field from uint64 to math.LegacyDec.
func CreateUpgradeHandler(
	mm *module.Manager,
	configurator module.Configurator,
	_ types.BaseAppParamManager,
	_ types.AppKeepers,
) upgradetypes.UpgradeHandler {
	return func(context context.Context, plan upgradetypes.Plan, fromVM module.VersionMap) (module.VersionMap, error) {
		ctx := sdk.UnwrapSDKContext(context)

		// Run migrations - this will automatically trigger the TrustDeposit migration
		// from version 1 to 2 (Migrate1to2) which converts Share from uint64 to LegacyDec
		migrations, err := mm.RunMigrations(ctx, configurator, fromVM)
		if err != nil {
			return nil, err
		}

		return migrations, nil
	}
}
