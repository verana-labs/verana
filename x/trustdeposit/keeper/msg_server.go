package keeper

import (
	"context"
	"cosmossdk.io/math"
	"fmt"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/verana-labs/verana/x/trustdeposit/types"
)

type msgServer struct {
	Keeper
}

// NewMsgServerImpl returns an implementation of the MsgServer interface
// for the provided Keeper.
func NewMsgServerImpl(keeper Keeper) types.MsgServer {
	return &msgServer{Keeper: keeper}
}

var _ types.MsgServer = msgServer{}

func (ms msgServer) ReclaimTrustDepositYield(goCtx context.Context, msg *types.MsgReclaimTrustDepositYield) (*types.MsgReclaimTrustDepositYieldResponse, error) {
	ctx := sdk.UnwrapSDKContext(goCtx)
	account := msg.Creator

	// [MOD-TD-MSG-2-2-1] Load TrustDeposit entry
	td, err := ms.Keeper.TrustDeposit.Get(ctx, account)
	if err != nil {
		return nil, fmt.Errorf("trust deposit not found for account: %s", account)
	}

	// [MOD-TD-MSG-2-2-1] Check slashing condition - CRITICAL MISSING CHECK
	if td.SlashedDeposit > 0 && td.SlashedDeposit < td.RepaidDeposit {
		return nil, fmt.Errorf("deposit has been slashed and not repaid")
	}

	// Get share value
	params := ms.Keeper.GetParams(ctx)

	// [MOD-TD-MSG-2-2-1] Calculate claimable yield
	// claimable_yield = td.share * GlobalVariables.trust_deposit_share_value - td.deposit
	depositValue := ms.Keeper.ShareToAmount(td.Share, params.TrustDepositShareValue)
	if depositValue <= td.Amount { // td.Amount maps to spec's td.deposit
		return nil, fmt.Errorf("no claimable yield available") // Updated error message
	}

	claimableYield := depositValue - td.Amount

	// [MOD-TD-MSG-2-3] Calculate shares to reduce
	// td.share = td.share - claimable_yield / GlobalVariables.trust_deposit_share_value
	sharesToReduce := ms.Keeper.AmountToShare(claimableYield, params.TrustDepositShareValue)
	td.Share -= sharesToReduce

	addr, _ := sdk.AccAddressFromBech32(account)

	// [MOD-TD-MSG-2-3] Transfer yield from TrustDeposit account to account
	coins := sdk.NewCoins(sdk.NewInt64Coin(types.BondDenom, int64(claimableYield)))
	if err := ms.Keeper.bankKeeper.SendCoinsFromModuleToAccount(
		ctx,
		types.ModuleName,
		addr,
		coins,
	); err != nil {
		return nil, fmt.Errorf("failed to transfer yield: %w", err)
	}

	// Save updated trust deposit
	if err := ms.Keeper.TrustDeposit.Set(ctx, account, td); err != nil {
		return nil, fmt.Errorf("failed to update trust deposit: %w", err)
	}

	return &types.MsgReclaimTrustDepositYieldResponse{
		ClaimedAmount: claimableYield, // Or rename to ClaimedYield
	}, nil
}

// ShareToAmount converts share value to amount using decimal math
func (k Keeper) ShareToAmount(share uint64, shareValue math.LegacyDec) uint64 {
	shareDec := math.LegacyNewDec(int64(share))
	amountDec := shareDec.Mul(shareValue)
	return amountDec.TruncateInt().Uint64()
}

// AmountToShare converts amount to share value using decimal math
func (k Keeper) AmountToShare(amount uint64, shareValue math.LegacyDec) uint64 {
	amountDec := math.LegacyNewDec(int64(amount))
	if shareValue.IsZero() {
		return 0 // Prevent division by zero
	}
	shareDec := amountDec.Quo(shareValue)
	return shareDec.TruncateInt().Uint64()
}

func (ms msgServer) ReclaimTrustDeposit(goCtx context.Context, msg *types.MsgReclaimTrustDeposit) (*types.MsgReclaimTrustDepositResponse, error) {
	ctx := sdk.UnwrapSDKContext(goCtx)

	// Basic validations
	if msg.Claimed == 0 {
		return nil, fmt.Errorf("claimed amount must be greater than 0")
	}

	// Get account running the method
	account := msg.Creator

	// Load TrustDeposit entry
	td, err := ms.Keeper.TrustDeposit.Get(ctx, account)
	if err != nil {
		return nil, fmt.Errorf("trust deposit not found for account: %s", account)
	}

	// Check if claimed amount is within claimable balance
	if td.Claimable < msg.Claimed {
		return nil, fmt.Errorf("claimed amount exceeds claimable balance")
	}

	// Get module params for calculations
	params := ms.Keeper.GetParams(ctx)

	// Calculate required minimum deposit using decimal math
	requiredMinDeposit := ms.Keeper.ShareToAmount(td.Share, params.TrustDepositShareValue)

	if td.Amount < msg.Claimed {
		return nil, fmt.Errorf("amount less than claimed")
	}

	if requiredMinDeposit < (td.Amount - msg.Claimed) {
		return nil, fmt.Errorf("insufficient required minimum deposit")
	}

	// Calculate burn amount and transfer amount using decimal math
	toBurn := ms.Keeper.CalculateBurnAmount(msg.Claimed, params.TrustDepositReclaimBurnRate)
	toTransfer := msg.Claimed - toBurn

	// Calculate share reduction using decimal math
	shareReduction := ms.Keeper.AmountToShare(msg.Claimed, params.TrustDepositShareValue)

	// Update trust deposit
	td.Claimable -= msg.Claimed
	td.Amount -= msg.Claimed
	td.Share -= shareReduction

	addr, _ := sdk.AccAddressFromBech32(msg.Creator)
	// Transfer claimable amount minus burn to the account
	if toTransfer > 0 {
		transferCoins := sdk.NewCoins(sdk.NewInt64Coin(types.BondDenom, int64(toTransfer)))
		if err := ms.Keeper.bankKeeper.SendCoinsFromModuleToAccount(
			ctx,
			types.ModuleName,
			addr,
			transferCoins,
		); err != nil {
			return nil, fmt.Errorf("failed to transfer coins: %w", err)
		}
	}

	// Burn the calculated amount
	if toBurn > 0 {
		burnCoins := sdk.NewCoins(sdk.NewInt64Coin(types.BondDenom, int64(toBurn)))
		if err := ms.Keeper.bankKeeper.BurnCoins(
			ctx,
			types.ModuleName,
			burnCoins,
		); err != nil {
			return nil, fmt.Errorf("failed to burn coins: %w", err)
		}
	}

	// Save updated trust deposit
	if err := ms.Keeper.TrustDeposit.Set(ctx, account, td); err != nil {
		return nil, fmt.Errorf("failed to update trust deposit: %w", err)
	}

	return &types.MsgReclaimTrustDepositResponse{
		BurnedAmount:  toBurn,
		ClaimedAmount: toTransfer,
	}, nil
}

// CalculateBurnAmount applies burn rate to claimed amount using decimal math
func (k Keeper) CalculateBurnAmount(claimed uint64, burnRate math.LegacyDec) uint64 {
	claimedDec := math.LegacyNewDec(int64(claimed))
	burnAmountDec := claimedDec.Mul(burnRate)
	return burnAmountDec.TruncateInt().Uint64()
}

func (ms msgServer) RepaySlashedTrustDeposit(goCtx context.Context, msg *types.MsgRepaySlashedTrustDeposit) (*types.MsgRepaySlashedTrustDepositResponse, error) {
	ctx := sdk.UnwrapSDKContext(goCtx)

	// Load TrustDeposit entry (must exist)
	td, err := ms.Keeper.TrustDeposit.Get(ctx, msg.Account)
	if err != nil {
		return nil, fmt.Errorf("trust deposit entry not found for account %s: %w", msg.Account, err)
	}

	// Check that amount exactly equals slashed_deposit - repaid_deposit
	outstandingSlash := td.SlashedDeposit - td.RepaidDeposit
	if msg.Amount != outstandingSlash {
		return nil, fmt.Errorf("amount must exactly equal outstanding slashed amount: expected %d, got %d", outstandingSlash, msg.Amount)
	}

	// [MOD-TD-MSG-6-2-2] Fee checks validation
	creatorAddr, err := sdk.AccAddressFromBech32(msg.Creator)
	if err != nil {
		return nil, fmt.Errorf("invalid creator address: %w", err)
	}

	// Transfer amount from creator to trust deposit module
	transferCoins := sdk.NewCoins(sdk.NewInt64Coin(types.BondDenom, int64(msg.Amount)))
	if err := ms.Keeper.bankKeeper.SendCoinsFromAccountToModule(
		ctx,
		creatorAddr,
		types.ModuleName,
		transferCoins,
	); err != nil {
		return nil, fmt.Errorf("failed to transfer tokens: %w", err)
	}

	// [MOD-TD-MSG-6-3] Execution
	params := ms.Keeper.GetParams(ctx)
	now := ctx.BlockTime()

	// Update trust deposit fields
	td.Amount += msg.Amount

	// Calculate share increase using decimal math
	shareIncrease := ms.Keeper.AmountToShare(msg.Amount, params.TrustDepositShareValue)
	td.Share += shareIncrease

	// Update repayment tracking
	td.RepaidDeposit += msg.Amount
	td.LastRepaid = &now
	td.LastRepaidBy = msg.Creator

	// Save updated trust deposit
	if err := ms.Keeper.TrustDeposit.Set(ctx, msg.Account, td); err != nil {
		return nil, fmt.Errorf("failed to update trust deposit: %w", err)
	}

	// Emit event
	ctx.EventManager().EmitEvent(
		sdk.NewEvent(
			types.EventTypeRepaySlashedTrustDeposit,
			sdk.NewAttribute(types.AttributeKeyAccount, msg.Account),
			sdk.NewAttribute(types.AttributeKeyAmount, fmt.Sprintf("%d", msg.Amount)),
			sdk.NewAttribute(types.AttributeKeyRepaidBy, msg.Creator),
		),
	)

	return &types.MsgRepaySlashedTrustDepositResponse{}, nil
}
