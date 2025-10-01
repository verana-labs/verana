package keeper

import (
	"fmt"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/verana-labs/verana/x/td/types"
	"strconv"
)

// AdjustTrustDeposit modifies the trust deposit for an account by the specified amount.
// If augend is positive, it increases the trust deposit.
// If augend is negative, it decreases the trust deposit and increases the claimable amount.
//
// The function follows the specification [MOD-TD-MSG-1] from the Verana blockchain specs.
//
// Parameters:
// - ctx: The SDK context
// - account: The account address as a Bech32 string
// - augend: The amount to adjust (positive for increase, negative for decrease)
//
// Returns:
// - error: If the operation fails
func (k Keeper) AdjustTrustDeposit(ctx sdk.Context, account string, augend int64) error {
	// Basic validation
	senderAcc, err := sdk.AccAddressFromBech32(account)
	if err != nil {
		return fmt.Errorf("invalid account address: %w", err)
	}
	if account == "" {
		return fmt.Errorf("account cannot be empty")
	}
	if augend == 0 {
		return fmt.Errorf("augend must be non-zero")
	}

	// Get global share value parameter
	params := k.GetParams(ctx)
	shareValue := params.TrustDepositShareValue

	// Load existing trust deposit if it exists
	td, err := k.TrustDeposit.Get(ctx, account)

	if err != nil {
		// If trust deposit doesn't exist and trying to decrease, abort
		if augend < 0 {
			return fmt.Errorf("cannot decrease non-existent trust deposit")
		}

		// Initialize new trust deposit - create entry for positive augend
		// Transfer augend from account to TrustDeposit module
		err := k.bankKeeper.SendCoinsFromAccountToModule(
			ctx,
			senderAcc,
			types.ModuleName,
			sdk.NewCoins(sdk.NewInt64Coin(types.BondDenom, augend)),
		)
		if err != nil {
			return fmt.Errorf("failed to transfer tokens: %w", err)
		}

		// Calculate augend_share = amount / GlobalVariables.trust_deposit_share_value
		augendShare := k.AmountToShare(uint64(augend), shareValue)

		td = types.TrustDeposit{
			Account:   account,
			Amount:    uint64(augend),
			Share:     augendShare,
			Claimable: 0,
			// v2 fields auto-initialize to zero values
		}

		// Save new trust deposit
		err = k.TrustDeposit.Set(ctx, account, td)
		if err != nil {
			return fmt.Errorf("failed to save trust deposit: %w", err)
		}

		// Emit event for new entry
		ctx.EventManager().EmitEvents(sdk.Events{
			sdk.NewEvent(
				types.EventTypeAdjustTrustDeposit,
				sdk.NewAttribute(types.AttributeKeyAccount, account),
				sdk.NewAttribute(types.AttributeKeyAugend, strconv.FormatInt(augend, 10)),
				sdk.NewAttribute(types.AttributeKeyAdjustmentType, "increase"),
				sdk.NewAttribute(types.AttributeKeyNewAmount, strconv.FormatUint(td.Amount, 10)),
				sdk.NewAttribute(types.AttributeKeyNewShare, strconv.FormatUint(td.Share, 10)),
				sdk.NewAttribute(types.AttributeKeyTimestamp, ctx.BlockTime().String()),
			),
		})

		return nil
	}

	// Trust deposit exists - check slashing status
	if td.SlashedDeposit > 0 && td.SlashedDeposit < td.RepaidDeposit {
		return fmt.Errorf("trust deposit has been slashed and not fully repaid")
	}

	// Convert uint fields to int64 for calculations
	amount := int64(td.Amount)
	claimable := int64(td.Claimable)
	share := int64(td.Share)

	if augend > 0 {
		// Handle positive adjustment (increase)
		if claimable > 0 {
			if claimable >= augend {
				// Can cover from claimable amount
				claimable -= augend
			} else {
				// Need to transfer additional funds
				neededDeposit := augend - claimable

				// Transfer tokens from account to module
				err := k.bankKeeper.SendCoinsFromAccountToModule(
					ctx,
					senderAcc,
					types.ModuleName,
					sdk.NewCoins(sdk.NewInt64Coin(types.BondDenom, neededDeposit)),
				)
				if err != nil {
					return fmt.Errorf("failed to transfer tokens: %w", err)
				}

				// Calculate missing_augend_share = (augend - td.claimable) / GlobalVariables.trust_deposit_share_value
				missingShare := k.AmountToShare(uint64(neededDeposit), shareValue)

				amount += neededDeposit
				share += int64(missingShare)
				claimable = 0
			}
		} else {
			// No claimable amount, need to transfer full amount
			err := k.bankKeeper.SendCoinsFromAccountToModule(
				ctx,
				senderAcc,
				types.ModuleName,
				sdk.NewCoins(sdk.NewInt64Coin(types.BondDenom, augend)),
			)
			if err != nil {
				return fmt.Errorf("failed to transfer tokens: %w", err)
			}

			// Calculate augend_share = augend / GlobalVariables.trust_deposit_share_value
			augendShare := k.AmountToShare(uint64(augend), shareValue)

			amount += augend
			share += int64(augendShare)
		}
	} else { // augend < 0
		// Handle negative adjustment (decrease)
		absAugend := -augend

		// if augend is negative and td.claimable - augend > td.amount transaction MUST abort
		if claimable+absAugend > amount {
			return fmt.Errorf("claimable after adjustment would exceed deposit: %d > %d", claimable+absAugend, amount)
		}

		// Since augend is negative, we add absAugend to claimable
		// This implements "set td.claimable to td.claimable - augend" when augend is negative
		claimable += absAugend
	}

	// Convert back to uint for storage and ensure no negative values
	if amount < 0 {
		return fmt.Errorf("amount cannot be negative after adjustment: %d", amount)
	}
	if claimable < 0 {
		return fmt.Errorf("claimable amount cannot be negative after adjustment: %d", claimable)
	}
	if share < 0 {
		return fmt.Errorf("share cannot be negative after adjustment: %d", share)
	}

	td.Amount = uint64(amount)
	td.Claimable = uint64(claimable)
	td.Share = uint64(share)

	// Save updated trust deposit
	err = k.TrustDeposit.Set(ctx, account, td)
	if err != nil {
		return fmt.Errorf("failed to save trust deposit: %w", err)
	}

	// Emit event for adjustment
	adjustmentType := "increase"
	if augend < 0 {
		adjustmentType = "decrease"
	}

	ctx.EventManager().EmitEvents(sdk.Events{
		sdk.NewEvent(
			types.EventTypeAdjustTrustDeposit,
			sdk.NewAttribute(types.AttributeKeyAccount, account),
			sdk.NewAttribute(types.AttributeKeyAugend, strconv.FormatInt(augend, 10)),
			sdk.NewAttribute(types.AttributeKeyAdjustmentType, adjustmentType),
			sdk.NewAttribute(types.AttributeKeyNewAmount, strconv.FormatUint(td.Amount, 10)),
			sdk.NewAttribute(types.AttributeKeyNewShare, strconv.FormatUint(td.Share, 10)),
			sdk.NewAttribute(types.AttributeKeyNewClaimable, strconv.FormatUint(td.Claimable, 10)),
			sdk.NewAttribute(types.AttributeKeyTimestamp, ctx.BlockTime().String()),
		),
	})

	return nil
}
