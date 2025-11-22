package v9

import (
	"context"
	"time"

	upgradetypes "cosmossdk.io/x/upgrade/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/types/module"
	"github.com/verana-labs/verana/app/upgrades/types"
)

func CreateUpgradeHandler(
	mm *module.Manager,
	configurator module.Configurator,
	_ types.BaseAppParamManager,
	keepers types.AppKeepers,
) upgradetypes.UpgradeHandler {
	return func(context context.Context, plan upgradetypes.Plan, fromVM module.VersionMap) (module.VersionMap, error) {
		ctx := sdk.UnwrapSDKContext(context)

		// Update governance parameters
		govKeeper := keepers.GetGovKeeper()
		params, err := govKeeper.Params.Get(ctx)
		if err != nil {
			return nil, err
		}

		// Set voting_period to 48 hours
		votingPeriod := time.Hour * 48
		params.VotingPeriod = &votingPeriod

		// Set expedited_voting_period to 20 minutes
		expeditedVotingPeriod := time.Minute * 20
		params.ExpeditedVotingPeriod = &expeditedVotingPeriod

		// Update params
		if err := govKeeper.Params.Set(ctx, params); err != nil {
			return nil, err
		}

		// Run standard migrations
		return mm.RunMigrations(ctx, configurator, fromVM)
	}
}
