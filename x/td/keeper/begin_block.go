package keeper

import (
	"context"

	"cosmossdk.io/math"
	sdk "github.com/cosmos/cosmos-sdk/types"
	authtypes "github.com/cosmos/cosmos-sdk/x/auth/types"
	protocolpooltypes "github.com/cosmos/cosmos-sdk/x/protocolpool/types"

	"github.com/verana-labs/verana/x/td/types"
)

// BeginBlocker processes yield distribution at the beginning of each block
func (k Keeper) BeginBlocker(ctx context.Context) error {
	sdkCtx := sdk.UnwrapSDKContext(ctx)
	params := k.GetParams(sdkCtx)

	// Skip if yield distribution is not configured
	if params.YieldIntermediatePool == "" || params.BlocksPerYear == 0 || params.TrustDepositMaxYieldRate.IsZero() {
		return nil
	}

	// Get yield intermediate pool address
	yieldIntermediatePoolAddr, err := sdk.AccAddressFromBech32(params.YieldIntermediatePool)
	if err != nil {
		// If invalid, try to derive from module name
		yieldIntermediatePoolAddr = authtypes.NewModuleAddress(types.YieldIntermediatePool)
	}

	// Get trust deposit module address
	trustDepositAddr := authtypes.NewModuleAddress(types.ModuleName)

	// Get current dust (defaults to zero if not set)
	dustStr, err := k.Dust.Get(ctx)
	dust := math.LegacyZeroDec()
	if err == nil && dustStr != "" {
		dust, _ = math.LegacyNewDecFromStr(dustStr)
	}

	// Get trust deposit balance before transfer
	trustDepositBalanceBefore := k.bankKeeper.GetBalance(ctx, trustDepositAddr, types.BondDenom)
	trustDepositBalanceDec := math.LegacyNewDecFromInt(trustDepositBalanceBefore.Amount)

	// Compute yield allowance
	// allowance = dust + trust_deposit * trust_deposit_max_yield_rate / blocks_per_year
	blocksPerYearDec := math.LegacyNewDec(int64(params.BlocksPerYear))
	perBlockYieldRate := params.TrustDepositMaxYieldRate.Quo(blocksPerYearDec)
	perBlockYield := trustDepositBalanceDec.Mul(perBlockYieldRate)
	allowance := dust.Add(perBlockYield)

	// Get yield intermediate pool balance
	yieldIntermediatePoolBalance := k.bankKeeper.GetBalance(ctx, yieldIntermediatePoolAddr, types.BondDenom)
	yieldIntermediatePoolBalanceDec := math.LegacyNewDecFromInt(yieldIntermediatePoolBalance.Amount)

	// Determine transfer amount: min(allowance, yield_intermediate_pool_balance)
	transferAmountDec := math.LegacyMinDec(allowance, yieldIntermediatePoolBalanceDec)
	transferAmount := transferAmountDec.TruncateInt()

	// Only proceed if there's something to transfer
	if !transferAmount.IsPositive() {
		return nil
	}

	// Transfer yield from yield intermediate pool to trust deposit module
	transferCoins := sdk.NewCoins(sdk.NewCoin(types.BondDenom, transferAmount))
	if err := k.bankKeeper.SendCoinsFromModuleToModule(
		ctx,
		types.YieldIntermediatePool,
		types.ModuleName,
		transferCoins,
	); err != nil {
		k.Logger().Error("failed to transfer yield from intermediate pool to trust deposit", "error", err)
		return err
	}

	// Get trust deposit balance after transfer
	trustDepositBalanceAfter := k.bankKeeper.GetBalance(ctx, trustDepositAddr, types.BondDenom)
	trustDepositBalanceAfterDec := math.LegacyNewDecFromInt(trustDepositBalanceAfter.Amount)

	// Adjust Trust Deposit Share Value
	// trust_deposit_share_value = (trust_deposit_after / trust_deposit_before) * trust_deposit_share_value
	if !trustDepositBalanceDec.IsZero() {
		shareValueMultiplier := trustDepositBalanceAfterDec.Quo(trustDepositBalanceDec)
		newShareValue := params.TrustDepositShareValue.Mul(shareValueMultiplier)
		params.TrustDepositShareValue = newShareValue
		if err := k.SetParams(ctx, params); err != nil {
			k.Logger().Error("failed to update trust deposit share value", "error", err)
			return err
		}
	}

	// Update dust: remainder from the transfer
	// dust = min(allowance, yield_intermediate_pool_balance).remainder()
	remainder := transferAmountDec.Sub(math.LegacyNewDecFromInt(transferAmount))
	if err := k.Dust.Set(ctx, remainder.String()); err != nil {
		k.Logger().Error("failed to update dust", "error", err)
		return err
	}

	// Return rest of YIP account to community pool (protocol pool)
	remainingBalance := yieldIntermediatePoolBalanceDec.Sub(transferAmountDec)
	if remainingBalance.IsPositive() {
		remainingCoins := sdk.NewCoins(sdk.NewCoin(types.BondDenom, remainingBalance.TruncateInt()))

		// Send remaining balance back to protocol pool
		// Note: ProtocolPoolKeeper interface would be needed, but for now we use bank keeper
		// to send to protocol pool module account
		protocolPoolAddr := authtypes.NewModuleAddress(protocolpooltypes.ModuleName)
		if err := k.bankKeeper.SendCoinsFromModuleToAccount(
			ctx,
			types.YieldIntermediatePool,
			protocolPoolAddr,
			remainingCoins,
		); err != nil {
			k.Logger().Error("failed to return excess to protocol pool", "error", err)
			// Don't fail the block if this fails, just log
		}
	}

	return nil
}
