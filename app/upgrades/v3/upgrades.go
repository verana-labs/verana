package v3

import (
	"context"

	upgradetypes "cosmossdk.io/x/upgrade/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/types/module"
	"github.com/verana-labs/verana/app/upgrades/types"
	//credentialschema "github.com/verana-labs/verana/x/credentialschema/module"
	//credentialschematypes "github.com/verana-labs/verana/x/credentialschema/types"
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

		//credentialschema.InitGenesis(ctx, keepers.GetCredentialSchemaKeeper(), *credentialschematypes.DefaultGenesis())
		return migrations, nil
	}
}
