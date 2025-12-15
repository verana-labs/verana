package v092

import (
	"context"

	"cosmossdk.io/math"
	upgradetypes "cosmossdk.io/x/upgrade/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/types/module"
	authtypes "github.com/cosmos/cosmos-sdk/x/auth/types"
	protocolpooltypes "github.com/cosmos/cosmos-sdk/x/protocolpool/types"

	"github.com/verana-labs/verana/app/upgrades/types"
	tdtypes "github.com/verana-labs/verana/x/td/types"
)

// CreateUpgradeHandler creates the v0.9.2 upgrade handler.
//
// This upgrade:
//   - Ensures the YieldIntermediatePool (YIP) module account is initialized with
//     a dust balance of 1 uvna (transferred from an existing module account) to
//     avoid bank invariants breaking when it starts receiving funds from
//     continuous funding proposals, without changing total supply.
//   - Runs the standard module migrations via module manager.
func CreateUpgradeHandler(
	mm *module.Manager,
	configurator module.Configurator,
	_ types.BaseAppParamManager,
	keepers types.AppKeepers,
) upgradetypes.UpgradeHandler {
	return func(ctx context.Context, plan upgradetypes.Plan, fromVM module.VersionMap) (module.VersionMap, error) {
		sdkCtx := sdk.UnwrapSDKContext(ctx)

		bankKeeper := keepers.GetBankKeeper()
		tdKeeper := keepers.GetTrustDepositKeeper()

		// Resolve the YieldIntermediatePool module account address.
		yipAddr := authtypes.NewModuleAddress(tdtypes.YieldIntermediatePool)

		// If the YIP account has zero balance in BondDenom, transfer 1 uvna from
		// the protocol pool module account into YIP. This keeps total supply
		// unchanged and just redistributes existing funds.
		balance := bankKeeper.GetBalance(sdkCtx, yipAddr, tdtypes.BondDenom)
		if balance.Amount.IsZero() {
			coin := sdk.NewInt64Coin(tdtypes.BondDenom, 1)
			coins := sdk.NewCoins(coin)

			// Transfer 1 uvna from protocol pool module to YieldIntermediatePool.
			// This assumes the protocol pool has at least 1 uvna available; if not,
			// the transfer will fail and so will the upgrade.
			if err := bankKeeper.SendCoinsFromModuleToModule(
				sdkCtx,
				protocolpooltypes.ModuleName,
				tdtypes.YieldIntermediatePool,
				coins,
			); err != nil {
				return nil, err
			}
		}

		// Initialize newly added TrustDeposit params if they are unset on existing
		// chains. DefaultParams are only used at genesis, so older networks will
		// see zero values for new fields after proto decode.
		params := tdKeeper.GetParams(sdkCtx)

		// Backfill TrustDepositMaxYieldRate if it is zero.
		if params.TrustDepositMaxYieldRate.IsZero() {
			defaultMaxYield, err := math.LegacyNewDecFromStr(tdtypes.DefaultTrustDepositMaxYieldRate)
			if err != nil {
				return nil, err
			}
			params.TrustDepositMaxYieldRate = defaultMaxYield
		}

		// Backfill YieldIntermediatePool if it is empty: derive from module account name.
		if params.YieldIntermediatePool == "" {
			params.YieldIntermediatePool = authtypes.NewModuleAddress(tdtypes.YieldIntermediatePool).String()
		}

		if err := tdKeeper.SetParams(sdkCtx, params); err != nil {
			return nil, err
		}

		// Run standard module migrations.
		return mm.RunMigrations(sdkCtx, configurator, fromVM)
	}
}
