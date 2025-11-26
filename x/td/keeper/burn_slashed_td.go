package keeper

import (
	"fmt"

	"cosmossdk.io/math"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/verana-labs/verana/x/td/types"
)

func (k Keeper) BurnEcosystemSlashedTrustDeposit(ctx sdk.Context, account string, amount uint64) error {
	// [MOD-TD-MSG-7-2-1] Basic checks
	if account == "" {
		return fmt.Errorf("account cannot be empty")
	}

	if amount == 0 {
		return fmt.Errorf("amount must be greater than 0")
	}

	// Load existing TrustDeposit entry (must exist)
	td, err := k.TrustDeposit.Get(ctx, account)
	if err != nil {
		return fmt.Errorf("trust deposit entry not found for account %s: %w", account, err)
	}

	// amount MUST be lower or equal than td.amount
	if amount > td.Amount {
		return fmt.Errorf("amount exceeds available deposit: %d > %d", amount, td.Amount)
	}

	// [MOD-TD-MSG-7-3] Execution
	if err := k.executeBurnEcosystemSlashedTrustDeposit(ctx, account, td, amount); err != nil {
		return fmt.Errorf("failed to execute burn ecosystem slashed trust deposit: %w", err)
	}

	return nil
}

func (k Keeper) executeBurnEcosystemSlashedTrustDeposit(ctx sdk.Context, account string, td types.TrustDeposit, amount uint64) error {
	// Get trust deposit share value from params
	params := k.GetParams(ctx)
	trustDepositShareValue := params.TrustDepositShareValue

	// Check if share value is zero using LegacyDec method
	if trustDepositShareValue.IsZero() {
		return fmt.Errorf("trust deposit share value cannot be zero")
	}

	now := ctx.BlockTime()

	// Update existing TrustDeposit entry
	td.Amount = td.Amount - amount

	// Calculate and reduce shares - convert types properly
	amountInt := math.NewInt(int64(amount))
	amountDec := math.LegacyNewDecFromInt(amountInt)
	shareReduction := amountDec.Quo(trustDepositShareValue)

	if shareReduction.GT(td.Share) {
		return fmt.Errorf("share reduction exceeds available shares: %s > %s", shareReduction.String(), td.Share.String())
	}
	td.Share = td.Share.Sub(shareReduction) // Use Sub method

	// Update v2 slashing fields
	td.SlashedDeposit += amount
	td.LastSlashed = &now
	td.SlashCount += 1

	// Burn amount from TrustDeposit module account
	burnCoins := sdk.NewCoins(sdk.NewInt64Coin(types.BondDenom, int64(amount)))
	if err := k.bankKeeper.BurnCoins(ctx, types.ModuleName, burnCoins); err != nil {
		return fmt.Errorf("failed to burn coins from trust deposit module: %w", err)
	}

	// Save updated trust deposit entry
	if err := k.TrustDeposit.Set(ctx, account, td); err != nil {
		return fmt.Errorf("failed to update trust deposit entry: %w", err)
	}

	return nil
}
