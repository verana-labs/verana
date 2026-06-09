package keeper

import (
	"fmt"
	"strconv"

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
		return fmt.Errorf("deposit must be greater than 0")
	}

	// Resolve the account to its corporation_id (the storage key).
	corpID, err := k.resolveCorporationID(ctx, account)
	if err != nil {
		return err
	}

	// Load existing TrustDeposit entry (must exist)
	td, err := k.TrustDeposit.Get(ctx, corpID)
	if err != nil {
		return fmt.Errorf("trust deposit entry not found for account %s: %w", account, err)
	}

	// amount MUST be lower or equal than td.deposit
	if amount > td.Deposit {
		return fmt.Errorf("amount exceeds available deposit: %d > %d", amount, td.Deposit)
	}

	// [MOD-TD-MSG-7-3] Execution
	if err := k.executeBurnEcosystemSlashedTrustDeposit(ctx, corpID, account, td, amount); err != nil {
		return fmt.Errorf("failed to execute burn ecosystem slashed trust deposit: %w", err)
	}

	return nil
}

func (k Keeper) executeBurnEcosystemSlashedTrustDeposit(ctx sdk.Context, corpID uint64, account string, td types.TrustDeposit, amount uint64) error {
	// Get trust deposit share value from params
	params := k.GetParams(ctx)
	trustDepositShareValue := params.TrustDepositShareValue

	if trustDepositShareValue.IsZero() {
		return fmt.Errorf("trust deposit share value cannot be zero")
	}

	// [MOD-TD-MSG-7-3] td.deposit = td.deposit - amount
	td.Deposit = td.Deposit - amount

	// [MOD-TD-MSG-7-3] td.share = td.share - amount / GlobalVariables.trust_deposit_share_value
	shareReduction := math.LegacyNewDecFromInt(math.NewInt(int64(amount))).Quo(trustDepositShareValue)
	td.Share = td.Share.Sub(shareReduction)

	// Clamp share to zero if rounding pushes it slightly negative
	if td.Share.IsNegative() {
		td.Share = math.LegacyZeroDec()
	}

	// Note: td.SlashedDeposit/LastSlashed/SlashCount are NOT updated here.
	// Those fields are for network governance slashes (MOD-TD-MSG-5) only.
	// Ecosystem slashes track slashing at the permission level (perm.slashed_deposit).

	// Save updated trust deposit entry BEFORE burning coins to ensure atomicity —
	// if Set fails, no coins have been burned yet.
	if err := k.TrustDeposit.Set(ctx, corpID, td); err != nil {
		return fmt.Errorf("failed to update trust deposit entry: %w", err)
	}

	// [MOD-TD-MSG-7-3] Burn amount from TrustDeposit account
	burnCoins := sdk.NewCoins(sdk.NewInt64Coin(types.BondDenom, int64(amount)))
	if err := k.bankKeeper.BurnCoins(ctx, types.ModuleName, burnCoins); err != nil {
		return fmt.Errorf("failed to burn coins from trust deposit module: %w", err)
	}

	// Emit event for observability
	ctx.EventManager().EmitEvent(
		sdk.NewEvent(
			types.EventTypeBurnEcosystemSlashedTrustDeposit,
			sdk.NewAttribute(types.AttributeKeyCorporationID, strconv.FormatUint(corpID, 10)),
			sdk.NewAttribute(types.AttributeKeyAccount, account),
			sdk.NewAttribute(types.AttributeKeyAmount, strconv.FormatUint(amount, 10)),
			sdk.NewAttribute(types.AttributeKeyNewAmount, strconv.FormatUint(td.Deposit, 10)),
			sdk.NewAttribute(types.AttributeKeyNewShare, td.Share.String()),
			sdk.NewAttribute(types.AttributeKeyTimestamp, ctx.BlockTime().String()),
		),
	)

	return nil
}
