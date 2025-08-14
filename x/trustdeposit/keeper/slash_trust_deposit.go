package keeper

import (
	"fmt"

	errorsmod "cosmossdk.io/errors"
	"cosmossdk.io/math"
	sdk "github.com/cosmos/cosmos-sdk/types"
	govtypes "github.com/cosmos/cosmos-sdk/x/gov/types/v1beta1"
	"github.com/verana-labs/verana/x/trustdeposit/types"
)

func NewTrustDepositHandler(k Keeper) govtypes.Handler {
	return func(ctx sdk.Context, content govtypes.Content) error {
		switch c := content.(type) {
		case *types.SlashTrustDepositProposal:
			return handleSlashTrustDepositProposal(ctx, k, c)
		default:
			return errorsmod.Wrapf(types.ErrorUnknownProposalType, "%T", c)
		}
	}
}

func handleSlashTrustDepositProposal(ctx sdk.Context, k Keeper, p *types.SlashTrustDepositProposal) error {
	return k.HandleSlashTrustDepositProposal(ctx, p)
}

func (k Keeper) HandleSlashTrustDepositProposal(ctx sdk.Context, p *types.SlashTrustDepositProposal) error {
	// [MOD-TD-MSG-5-2-1] Slash Trust Deposit basic checks
	if err := k.validateSlashTrustDepositProposal(ctx, p); err != nil {
		return err
	}

	// [MOD-TD-MSG-5-3] Slash Trust Deposit execution of the method
	return k.executeSlashTrustDepositProposal(ctx, p)
}

func (k Keeper) validateSlashTrustDepositProposal(ctx sdk.Context, p *types.SlashTrustDepositProposal) error {
	// Check if TrustDeposit entry exists for the account
	td, err := k.TrustDeposit.Get(ctx, p.Account)
	if err != nil {
		return types.ErrTrustDepositNotFound.Wrapf("account: %s", p.Account)
	}

	// Check if deposit is sufficient
	if math.NewIntFromUint64(td.Amount).LT(p.Amount) {
		return types.ErrInsufficientTrustDeposit.Wrapf("deposit: %d, required: %s", td.Amount, p.Amount.String())
	}

	return nil
}

func (k Keeper) executeSlashTrustDepositProposal(ctx sdk.Context, p *types.SlashTrustDepositProposal) error {
	// [MOD-TD-MSG-5-2-1] Basic checks
	if p.Amount.IsZero() || p.Amount.IsNegative() {
		return types.ErrInvalidAmount.Wrap("amount must be greater than 0")
	}

	// Check if TrustDeposit entry exists for the account
	td, err := k.TrustDeposit.Get(ctx, p.Account)
	if err != nil {
		return types.ErrTrustDepositNotFound.Wrapf("account: %s", p.Account)
	}

	// Check if deposit is sufficient
	if math.NewIntFromUint64(td.Amount).LT(p.Amount) {
		return types.ErrInsufficientTrustDeposit.Wrapf("deposit: %d, required: %s", td.Amount, p.Amount.String())
	}

	// [MOD-TD-MSG-5-3] Execute the slash
	now := ctx.BlockTime()

	// Get global variables for share calculation
	params := k.GetParams(ctx)
	shareValue := params.TrustDepositShareValue

	// Calculate share reduction
	shareReduction := math.LegacyNewDecFromInt(p.Amount).Quo(shareValue)

	// Update TrustDeposit entry
	td.Amount = td.Amount - p.Amount.Uint64()
	td.Share = td.Share - uint64(shareReduction.TruncateInt64())
	td.SlashedDeposit = td.SlashedDeposit + p.Amount.Uint64()
	td.LastSlashed = &now
	td.LastRepaidBy = ""
	td.SlashCount++

	// Burn the slashed amount
	burnCoins := sdk.NewCoins(sdk.NewCoin(types.BondDenom, p.Amount))
	if err := k.bankKeeper.BurnCoins(ctx, types.ModuleName, burnCoins); err != nil {
		return fmt.Errorf("failed to burn coins: %w", err)
	}

	// Save the updated TrustDeposit entry
	if err := k.TrustDeposit.Set(ctx, p.Account, td); err != nil {
		return fmt.Errorf("failed to save trust deposit: %w", err)
	}

	// Emit event
	ctx.EventManager().EmitEvent(
		sdk.NewEvent(
			"trust_deposit_slashed",
			sdk.NewAttribute("account", p.Account),
			sdk.NewAttribute("amount", p.Amount.String()),
		),
	)

	return nil
}

// SlashTrustDeposit slashes a trust deposit for an account
func (k Keeper) SlashTrustDeposit(ctx sdk.Context, account string, amount math.Int) error {
	// [MOD-TD-MSG-5-2-1] Basic checks
	if amount.IsZero() || amount.IsNegative() {
		return types.ErrInvalidAmount.Wrap("amount must be greater than 0")
	}

	// Check if TrustDeposit entry exists for the account
	td, err := k.TrustDeposit.Get(ctx, account)
	if err != nil {
		return types.ErrTrustDepositNotFound.Wrapf("account: %s", account)
	}

	// Check if deposit is sufficient
	if math.NewIntFromUint64(td.Amount).LT(amount) {
		return types.ErrInsufficientTrustDeposit.Wrapf("deposit: %d, required: %s", td.Amount, amount.String())
	}

	// [MOD-TD-MSG-5-3] Execute the slash
	now := ctx.BlockTime()

	// Get global variables for share calculation
	params := k.GetParams(ctx)
	shareValue := params.TrustDepositShareValue

	// Calculate share reduction
	shareReduction := math.LegacyNewDecFromInt(amount).Quo(shareValue)

	// Update TrustDeposit entry
	td.Amount = td.Amount - amount.Uint64()
	td.Share = td.Share - uint64(shareReduction.TruncateInt64())
	td.SlashedDeposit = td.SlashedDeposit + amount.Uint64()
	td.LastSlashed = &now
	td.LastRepaidBy = ""
	td.SlashCount++

	// Burn the slashed amount
	burnCoins := sdk.NewCoins(sdk.NewCoin(types.BondDenom, amount))
	if err := k.bankKeeper.BurnCoins(ctx, types.ModuleName, burnCoins); err != nil {
		return fmt.Errorf("failed to burn coins: %w", err)
	}

	// Save the updated TrustDeposit entry
	if err := k.TrustDeposit.Set(ctx, account, td); err != nil {
		return fmt.Errorf("failed to save trust deposit: %w", err)
	}

	// Emit event
	ctx.EventManager().EmitEvent(
		sdk.NewEvent(
			"trust_deposit_slashed",
			sdk.NewAttribute("account", account),
			sdk.NewAttribute("amount", amount.String()),
		),
	)

	return nil
}
