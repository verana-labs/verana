package v093

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

// CreateUpgradeHandler creates the v0.9.3 upgrade handler.
//
// This upgrade includes:
//   - Bug fixes: #191 (effective_from mandatory), #193 (INACTIVE validator check), #196 (revoke future perms)
//   - Feature: #186 (optional validation/issuance/verification fees)
//
// The handler reuses v0.9.2 idempotent logic:
//   - Ensures YieldIntermediatePool (YIP) module account is initialized (noop if already done)
//   - Ensures TD params are set (noop if already set)
//   - Runs standard module migrations
//
// This makes the upgrade safe for both:
//   - testnet (v0.9.1-dev.1 → v0.9.3): Will initialize YIP and params
//   - devnet (v0.9.2 → v0.9.3): Will skip YIP/params (already done), just run migrations
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

		// Step 1: Ensure YieldIntermediatePool module account has balance.
		// This is idempotent: only transfers if YIP balance is zero.
		yipAddr := authtypes.NewModuleAddress(tdtypes.YieldIntermediatePool)
		balance := bankKeeper.GetBalance(sdkCtx, yipAddr, tdtypes.BondDenom)
		if balance.Amount.IsZero() {
			coin := sdk.NewInt64Coin(tdtypes.BondDenom, 1)
			coins := sdk.NewCoins(coin)

			// Transfer 1 uvna from protocol pool to YieldIntermediatePool.
			if err := bankKeeper.SendCoinsFromModuleToModule(
				sdkCtx,
				protocolpooltypes.ModuleName,
				tdtypes.YieldIntermediatePool,
				coins,
			); err != nil {
				return nil, err
			}
		}

		// Step 2: Initialize TD params if unset.
		// This is idempotent: only sets params if they have zero values.
		params := tdKeeper.GetParams(sdkCtx)

		// Check for nil/zero TrustDepositMaxYieldRate (protobuf zero-value on old chains).
		if params.TrustDepositMaxYieldRate == (math.LegacyDec{}) || params.TrustDepositMaxYieldRate.IsZero() {
			defaultMaxYield, err := math.LegacyNewDecFromStr(tdtypes.DefaultTrustDepositMaxYieldRate)
			if err != nil {
				return nil, err
			}
			params.TrustDepositMaxYieldRate = defaultMaxYield
		}

		// Backfill YieldIntermediatePool address if empty.
		if params.YieldIntermediatePool == "" {
			params.YieldIntermediatePool = authtypes.NewModuleAddress(tdtypes.YieldIntermediatePool).String()
		}

		if err := tdKeeper.SetParams(sdkCtx, params); err != nil {
			return nil, err
		}

		// Step 3: Run standard module migrations.
		return mm.RunMigrations(sdkCtx, configurator, fromVM)
	}
}
