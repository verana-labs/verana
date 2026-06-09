package v091

import (
	"context"

	upgradetypes "cosmossdk.io/x/upgrade/types"
	"github.com/cosmos/cosmos-sdk/types/module"
	"github.com/verana-labs/verana/app/upgrades/types"
)

func CreateUpgradeHandler(
	mm *module.Manager,
	configurator module.Configurator,
	_ types.BaseAppParamManager,
	_ types.AppKeepers,
) upgradetypes.UpgradeHandler {
	return func(context context.Context, plan upgradetypes.Plan, fromVM module.VersionMap) (module.VersionMap, error) {
		// The original TrustDeposit Share uint64->LegacyDec migration was removed
		// with the v4-rc3 trust-deposit rekey (chain resets on the version bump).
		return mm.RunMigrations(context, configurator, fromVM)
	}
}
