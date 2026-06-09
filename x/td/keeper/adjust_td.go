package keeper

import (
	"fmt"
	"strconv"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/verana-labs/verana/x/td/types"
)

// AdjustTrustDeposit adjusts a corporation's trust deposit. `account` is the
// corporation policy_address: it is used for bank transfers and resolved to the
// corporation_id, which is the trust deposit record key.
func (k Keeper) AdjustTrustDeposit(ctx sdk.Context, account string, augend int64, reason string) error {
	// Basic validation
	if account == "" {
		return fmt.Errorf("account cannot be empty")
	}
	senderAcc, err := sdk.AccAddressFromBech32(account)
	if err != nil {
		return fmt.Errorf("invalid account address: %w", err)
	}
	if augend == 0 {
		return fmt.Errorf("augend must be non-zero")
	}

	// Resolve the account to its corporation_id (the storage key).
	corpID, err := k.resolveCorporationID(ctx, account)
	if err != nil {
		return err
	}

	// Get global share value parameter
	params := k.GetParams(ctx)
	shareValue := params.TrustDepositShareValue

	// Load existing trust deposit if it exists
	td, err := k.TrustDeposit.Get(ctx, corpID)

	if err != nil {
		// If trust deposit doesn't exist and trying to decrease, abort
		if augend < 0 {
			return fmt.Errorf("cannot decrease non-existent trust deposit")
		}

		// Initialize new trust deposit - create entry for positive augend
		augendShare := k.AmountToShare(uint64(augend), shareValue)

		td = types.TrustDeposit{
			CorporationId: corpID,
			Deposit:       uint64(augend),
			Share:         augendShare,
			Refunded:      0,
		}

		// Save new trust deposit BEFORE bank transfer
		err := k.TrustDeposit.Set(ctx, corpID, td)
		if err != nil {
			return fmt.Errorf("failed to save trust deposit: %w", err)
		}

		// Transfer augend from account to TrustDeposit module
		if err := k.bankKeeper.SendCoinsFromAccountToModule(
			ctx,
			senderAcc,
			types.ModuleName,
			sdk.NewCoins(sdk.NewInt64Coin(types.BondDenom, augend)),
		); err != nil {
			return fmt.Errorf("failed to transfer tokens: %w", err)
		}

		// Emit event for new entry
		ctx.EventManager().EmitEvents(sdk.Events{
			sdk.NewEvent(
				types.EventTypeAdjustTrustDeposit,
				sdk.NewAttribute(types.AttributeKeyCorporationID, strconv.FormatUint(corpID, 10)),
				sdk.NewAttribute(types.AttributeKeyAccount, account),
				sdk.NewAttribute(types.AttributeKeyAugend, strconv.FormatInt(augend, 10)),
				sdk.NewAttribute(types.AttributeKeyAdjustmentType, "increase"),
				sdk.NewAttribute(types.AttributeKeyNewAmount, strconv.FormatUint(td.Deposit, 10)),
				sdk.NewAttribute(types.AttributeKeyNewShare, td.Share.String()),
				sdk.NewAttribute(types.AttributeKeyReason, reason),
				sdk.NewAttribute(types.AttributeKeyTimestamp, ctx.BlockTime().String()),
			),
		})

		return nil
	}

	// Trust deposit exists - check slashing status
	if td.SlashedDeposit > 0 && td.RepaidDeposit < td.SlashedDeposit {
		return fmt.Errorf("trust deposit has been slashed and not fully repaid")
	}

	// Convert uint fields to int64 for calculations
	deposit := int64(td.Deposit)
	refunded := int64(td.Refunded)
	// share stays as td.Share (math.LegacyDec)

	// Track how much needs to be transferred from account to module
	var transferAmount int64

	if augend > 0 {
		// Handle positive adjustment (increase)
		if refunded > 0 {
			if refunded >= augend {
				// Can cover from refunded amount — no bank transfer needed
				refunded -= augend
			} else {
				// Need to transfer additional funds
				neededDeposit := augend - refunded
				transferAmount = neededDeposit

				// Calculate missing_augend_share = (augend - td.refunded) / GlobalVariables.trust_deposit_share_value
				// computed BEFORE refunded is zeroed (spec commit 90679a9).
				missingShare := k.AmountToShare(uint64(neededDeposit), shareValue)

				deposit += neededDeposit
				td.Share = td.Share.Add(missingShare)
				refunded = 0
			}
		} else {
			// No refunded amount, need to transfer full amount
			transferAmount = augend

			// Calculate augend_share = augend / GlobalVariables.trust_deposit_share_value
			augendShare := k.AmountToShare(uint64(augend), shareValue)

			deposit += augend
			td.Share = td.Share.Add(augendShare)
		}
	} else { // augend < 0
		// Handle negative adjustment (decrease)
		absAugend := -augend

		// if augend is negative and td.refunded - augend > td.deposit transaction MUST abort
		if refunded+absAugend > deposit {
			return fmt.Errorf("refunded after adjustment would exceed deposit: %d > %d", refunded+absAugend, deposit)
		}

		// Since augend is negative, we add absAugend to refunded
		// This implements "set td.refunded to td.refunded - augend" when augend is negative
		refunded += absAugend
	}

	// Convert back to uint for storage and ensure no negative values
	if deposit < 0 {
		return fmt.Errorf("deposit cannot be negative after adjustment: %d", deposit)
	}
	if refunded < 0 {
		return fmt.Errorf("refunded amount cannot be negative after adjustment: %d", refunded)
	}
	if td.Share.IsNegative() {
		return fmt.Errorf("share cannot be negative after adjustment: %s", td.Share.String())
	}

	td.Deposit = uint64(deposit)
	td.Refunded = uint64(refunded)

	// Save updated trust deposit BEFORE bank transfer
	err = k.TrustDeposit.Set(ctx, corpID, td)
	if err != nil {
		return fmt.Errorf("failed to save trust deposit: %w", err)
	}

	// Transfer tokens from account to module (if needed)
	if transferAmount > 0 {
		if err := k.bankKeeper.SendCoinsFromAccountToModule(
			ctx,
			senderAcc,
			types.ModuleName,
			sdk.NewCoins(sdk.NewInt64Coin(types.BondDenom, transferAmount)),
		); err != nil {
			return fmt.Errorf("failed to transfer tokens: %w", err)
		}
	}

	// Emit event for adjustment
	adjustmentType := "increase"
	if augend < 0 {
		adjustmentType = "decrease"
	}

	ctx.EventManager().EmitEvents(sdk.Events{
		sdk.NewEvent(
			types.EventTypeAdjustTrustDeposit,
			sdk.NewAttribute(types.AttributeKeyCorporationID, strconv.FormatUint(corpID, 10)),
			sdk.NewAttribute(types.AttributeKeyAccount, account),
			sdk.NewAttribute(types.AttributeKeyAugend, strconv.FormatInt(augend, 10)),
			sdk.NewAttribute(types.AttributeKeyAdjustmentType, adjustmentType),
			sdk.NewAttribute(types.AttributeKeyNewAmount, strconv.FormatUint(td.Deposit, 10)),
			sdk.NewAttribute(types.AttributeKeyNewShare, td.Share.String()),
			sdk.NewAttribute(types.AttributeKeyNewRefunded, strconv.FormatUint(td.Refunded, 10)),
			sdk.NewAttribute(types.AttributeKeyReason, reason),
			sdk.NewAttribute(types.AttributeKeyTimestamp, ctx.BlockTime().String()),
		),
	})

	return nil
}

// AdjustTrustDepositOnBehalf increases the trust deposit of `account` using funds from `funder`.
// Unlike AdjustTrustDeposit, this transfers coins directly from the funder to the TD module,
// bypassing the refunded recycling logic. This is used when a third party (e.g., a fee payer)
// funds another account's trust deposit increase during CSPS fee distribution.
//
// Only positive amounts are supported (increase only).
func (k Keeper) AdjustTrustDepositOnBehalf(ctx sdk.Context, account string, funder sdk.AccAddress, amount int64) error {
	if amount <= 0 {
		return fmt.Errorf("amount must be positive, got %d", amount)
	}
	if account == "" {
		return fmt.Errorf("account cannot be empty")
	}

	// Resolve the account to its corporation_id (the storage key).
	corpID, err := k.resolveCorporationID(ctx, account)
	if err != nil {
		return err
	}

	// Check if account has an existing TD with unrepaid slash
	td, err := k.TrustDeposit.Get(ctx, corpID)
	exists := err == nil
	if exists && td.SlashedDeposit > 0 && td.RepaidDeposit < td.SlashedDeposit {
		return fmt.Errorf("trust deposit has been slashed and not repaid")
	}

	// Get global share value parameter
	params := k.GetParams(ctx)
	shareValue := params.TrustDepositShareValue

	augendShare := k.AmountToShare(uint64(amount), shareValue)

	// Load or create trust deposit for account
	if !exists {
		td = types.TrustDeposit{
			CorporationId: corpID,
			Deposit:       uint64(amount),
			Share:         augendShare,
			Refunded:      0,
		}
	} else {
		td.Deposit += uint64(amount)
		td.Share = td.Share.Add(augendShare)
	}

	// Save trust deposit BEFORE bank transfer
	if err := k.TrustDeposit.Set(ctx, corpID, td); err != nil {
		return fmt.Errorf("failed to save trust deposit: %w", err)
	}

	// Transfer from funder to TD module
	if err := k.bankKeeper.SendCoinsFromAccountToModule(
		ctx,
		funder,
		types.ModuleName,
		sdk.NewCoins(sdk.NewInt64Coin(types.BondDenom, amount)),
	); err != nil {
		return fmt.Errorf("failed to transfer tokens from funder: %w", err)
	}

	ctx.EventManager().EmitEvents(sdk.Events{
		sdk.NewEvent(
			types.EventTypeAdjustTrustDeposit,
			sdk.NewAttribute(types.AttributeKeyCorporationID, strconv.FormatUint(corpID, 10)),
			sdk.NewAttribute(types.AttributeKeyAccount, account),
			sdk.NewAttribute(types.AttributeKeyAugend, strconv.FormatInt(amount, 10)),
			sdk.NewAttribute(types.AttributeKeyAdjustmentType, "increase_on_behalf"),
			sdk.NewAttribute(types.AttributeKeyNewAmount, strconv.FormatUint(td.Deposit, 10)),
			sdk.NewAttribute(types.AttributeKeyNewShare, td.Share.String()),
			sdk.NewAttribute(types.AttributeKeyTimestamp, ctx.BlockTime().String()),
		),
	})

	return nil
}
