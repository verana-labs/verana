package keeper

import (
	"context"
	"fmt"
	"strconv"

	"cosmossdk.io/math"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/verana-labs/verana/x/td/types"
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
	account := msg.Authority

	// [MOD-TD-MSG-2-2] [AUTHZ-CHECK] Verify operator authorization
	if ms.Keeper.delegationKeeper != nil {
		if err := ms.Keeper.delegationKeeper.CheckOperatorAuthorization(
			ctx,
			msg.Authority,
			msg.Operator,
			"/verana.td.v1.MsgReclaimTrustDepositYield",
			ctx.BlockTime(),
		); err != nil {
			return nil, fmt.Errorf("authorization check failed: %w", err)
		}
	}

	// [MOD-TD-MSG-2-2-1] Load TrustDeposit entry
	td, err := ms.Keeper.TrustDeposit.Get(ctx, account)
	if err != nil {
		return nil, fmt.Errorf("trust deposit not found for account: %s", account)
	}

	// [MOD-TD-MSG-2-2-1] Check slashing condition - CRITICAL MISSING CHECK
	if td.SlashedDeposit > 0 && td.RepaidDeposit < td.SlashedDeposit {
		return nil, fmt.Errorf("deposit has been slashed and not repaid")
	}

	// Get share value
	params := ms.Keeper.GetParams(ctx)

	// [MOD-TD-MSG-2-2-1] Calculate claimable yield
	// claimable_yield = td.share * GlobalVariables.trust_deposit_share_value - td.deposit
	// Use decimal math to avoid uint64 overflow on large share * shareValue products
	depositValueDec := td.Share.Mul(params.TrustDepositShareValue)
	depositAmountDec := math.LegacyNewDecFromInt(math.NewIntFromUint64(td.Amount))
	claimableYieldDec := depositValueDec.Sub(depositAmountDec)

	if !claimableYieldDec.IsPositive() {
		return nil, fmt.Errorf("no claimable yield available")
	}

	claimableYield := claimableYieldDec.TruncateInt().Uint64()
	if claimableYield == 0 {
		return nil, fmt.Errorf("no claimable yield available")
	}

	// [MOD-TD-MSG-2-3] Calculate shares to reduce
	// td.share = td.share - claimable_yield / GlobalVariables.trust_deposit_share_value
	sharesToReduce := ms.Keeper.AmountToShare(claimableYield, params.TrustDepositShareValue)
	td.Share = td.Share.Sub(sharesToReduce)

	// Validate authority address
	addr, err := sdk.AccAddressFromBech32(account)
	if err != nil {
		return nil, fmt.Errorf("invalid authority address: %w", err)
	}

	// Save updated trust deposit BEFORE bank transfer to ensure atomicity —
	// if Set fails, no coins have been transferred yet.
	if err := ms.Keeper.TrustDeposit.Set(ctx, account, td); err != nil {
		return nil, fmt.Errorf("failed to update trust deposit: %w", err)
	}

	// [MOD-TD-MSG-2-3] Transfer yield from TrustDeposit account to authority account
	coins := sdk.NewCoins(sdk.NewInt64Coin(types.BondDenom, int64(claimableYield)))
	if err := ms.Keeper.bankKeeper.SendCoinsFromModuleToAccount(
		ctx,
		types.ModuleName,
		addr,
		coins,
	); err != nil {
		return nil, fmt.Errorf("failed to transfer yield: %w", err)
	}

	ctx.EventManager().EmitEvents(sdk.Events{
		sdk.NewEvent(
			types.EventTypeReclaimTrustDepositYield,
			sdk.NewAttribute(types.AttributeKeyAccount, account),
			sdk.NewAttribute(types.AttributeKeyClaimedYield, strconv.FormatUint(claimableYield, 10)),
			sdk.NewAttribute(types.AttributeKeySharesReduced, sharesToReduce.String()),
			sdk.NewAttribute(types.AttributeKeyTimestamp, ctx.BlockTime().String()),
		),
	})

	return &types.MsgReclaimTrustDepositYieldResponse{
		ClaimedAmount: claimableYield, // Or rename to ClaimedYield
	}, nil
}

// ShareToAmount converts share value to amount using decimal math
func (k Keeper) ShareToAmount(share math.LegacyDec, shareValue math.LegacyDec) uint64 {
	amountDec := share.Mul(shareValue)
	return amountDec.TruncateInt().Uint64()
}

// AmountToShare converts amount to share value using decimal math
func (k Keeper) AmountToShare(amount uint64, shareValue math.LegacyDec) math.LegacyDec {
	amountDec := math.LegacyNewDec(int64(amount))
	if shareValue.IsZero() {
		return math.LegacyZeroDec() // Prevent division by zero
	}
	return amountDec.Quo(shareValue)
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
	td.Share = td.Share.Sub(shareReduction)

	addr, err := sdk.AccAddressFromBech32(msg.Creator)
	if err != nil {
		return nil, fmt.Errorf("invalid creator address: %w", err)
	}

	// Save updated trust deposit BEFORE bank operations to ensure atomicity —
	// if Set fails, no coins have been transferred or burned yet.
	if err := ms.Keeper.TrustDeposit.Set(ctx, account, td); err != nil {
		return nil, fmt.Errorf("failed to update trust deposit: %w", err)
	}

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

	ctx.EventManager().EmitEvents(sdk.Events{
		sdk.NewEvent(
			types.EventTypeReclaimTrustDeposit,
			sdk.NewAttribute(types.AttributeKeyAccount, account),
			sdk.NewAttribute(types.AttributeKeyClaimedAmount, strconv.FormatUint(msg.Claimed, 10)),
			sdk.NewAttribute(types.AttributeKeyBurnedAmount, strconv.FormatUint(toBurn, 10)),
			sdk.NewAttribute(types.AttributeKeyTransferAmount, strconv.FormatUint(toTransfer, 10)),
			sdk.NewAttribute(types.AttributeKeySharesReduced, shareReduction.String()),
			sdk.NewAttribute(types.AttributeKeyTimestamp, ctx.BlockTime().String()),
		),
	})

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

// SlashTrustDeposit handles governance slashing of trust deposits
func (ms msgServer) SlashTrustDeposit(goCtx context.Context, msg *types.MsgSlashTrustDeposit) (*types.MsgSlashTrustDepositResponse, error) {
	ctx := sdk.UnwrapSDKContext(goCtx)

	// [CRITICAL] Authority check - only governance can call this
	if ms.Keeper.authority != msg.Authority {
		return nil, fmt.Errorf("invalid authority; expected %s, got %s", ms.Keeper.authority, msg.Authority)
	}

	// [MOD-TD-MSG-5-2-1] Basic checks
	if msg.Amount.IsZero() || msg.Amount.IsNegative() {
		return nil, fmt.Errorf("amount must be greater than 0")
	}

	// Check if TrustDeposit entry exists for the account
	td, err := ms.Keeper.TrustDeposit.Get(ctx, msg.Account)
	if err != nil {
		return nil, fmt.Errorf("trust deposit not found for account: %s", msg.Account)
	}

	// Check if deposit is sufficient
	if math.NewIntFromUint64(td.Amount).LT(msg.Amount) {
		return nil, fmt.Errorf("insufficient trust deposit: deposit=%d, required=%s", td.Amount, msg.Amount.String())
	}

	// [MOD-TD-MSG-5-3] Execute the slash
	now := ctx.BlockTime()

	// Get global variables for share calculation
	params := ms.Keeper.GetParams(ctx)
	shareValue := params.TrustDepositShareValue

	// Calculate share reduction
	shareReduction := math.LegacyNewDecFromInt(msg.Amount).Quo(shareValue)

	// [MOD-TD-MSG-5-3] Update TrustDeposit entry
	td.Amount = td.Amount - msg.Amount.Uint64()
	td.Share = td.Share.Sub(shareReduction)
	td.SlashedDeposit = td.SlashedDeposit + msg.Amount.Uint64()
	td.LastSlashed = &now
	td.SlashCount++

	// Save the updated TrustDeposit entry BEFORE burning coins to ensure atomicity —
	// if Set fails, no coins have been burned yet.
	if err := ms.Keeper.TrustDeposit.Set(ctx, msg.Account, td); err != nil {
		return nil, fmt.Errorf("failed to save trust deposit: %w", err)
	}

	// Burn the slashed amount from TrustDeposit account
	burnCoins := sdk.NewCoins(sdk.NewCoin(types.BondDenom, msg.Amount))
	if err := ms.Keeper.bankKeeper.BurnCoins(ctx, types.ModuleName, burnCoins); err != nil {
		return nil, fmt.Errorf("failed to burn coins: %w", err)
	}

	// Emit event
	ctx.EventManager().EmitEvent(
		sdk.NewEvent(
			types.EventTypeSlashTrustDeposit,
			sdk.NewAttribute(types.AttributeKeyAccount, msg.Account),
			sdk.NewAttribute(types.AttributeKeyAmount, msg.Amount.String()),
			sdk.NewAttribute(types.AttributeKeySlashCount, strconv.FormatUint(td.SlashCount, 10)),
			sdk.NewAttribute(types.AttributeKeyTimestamp, ctx.BlockTime().String()),
		),
	)

	return &types.MsgSlashTrustDepositResponse{}, nil
}

func (ms msgServer) RepaySlashedTrustDeposit(goCtx context.Context, msg *types.MsgRepaySlashedTrustDeposit) (*types.MsgRepaySlashedTrustDepositResponse, error) {
	ctx := sdk.UnwrapSDKContext(goCtx)
	account := msg.Authority

	// [MOD-TD-MSG-6-2-1] [AUTHZ-CHECK] Verify operator authorization
	if ms.Keeper.delegationKeeper != nil {
		if err := ms.Keeper.delegationKeeper.CheckOperatorAuthorization(
			ctx,
			msg.Authority,
			msg.Operator,
			"/verana.td.v1.MsgRepaySlashedTrustDeposit",
			ctx.BlockTime(),
		); err != nil {
			return nil, fmt.Errorf("authorization check failed: %w", err)
		}
	}

	// [MOD-TD-MSG-6-2-1] Load TrustDeposit entry for authority (must exist)
	td, err := ms.Keeper.TrustDeposit.Get(ctx, account)
	if err != nil {
		return nil, fmt.Errorf("trust deposit entry not found for account %s: %w", account, err)
	}

	// [MOD-TD-MSG-6-2-1] amount MUST be exactly equal to td.slashed_deposit - td.repaid_deposit
	outstandingSlash := td.SlashedDeposit - td.RepaidDeposit
	if msg.Amount != outstandingSlash {
		return nil, fmt.Errorf("amount must exactly equal outstanding slashed amount: expected %d, got %d", outstandingSlash, msg.Amount)
	}

	// Validate authority address for bank transfer
	authorityAddr, err := sdk.AccAddressFromBech32(account)
	if err != nil {
		return nil, fmt.Errorf("invalid authority address: %w", err)
	}

	// [MOD-TD-MSG-6-3] Execution
	params := ms.Keeper.GetParams(ctx)
	now := ctx.BlockTime()

	// Update trust deposit fields
	td.Amount += msg.Amount

	// td.share = td.share + amount / GlobalVariables.trust_deposit_share_value
	shareIncrease := ms.Keeper.AmountToShare(msg.Amount, params.TrustDepositShareValue)
	td.Share = td.Share.Add(shareIncrease)

	// td.repaid_deposit = td.repaid_deposit + amount
	td.RepaidDeposit += msg.Amount
	// td.last_repaid = now
	td.LastRepaid = &now

	// Save updated trust deposit BEFORE bank transfer to ensure atomicity
	if err := ms.Keeper.TrustDeposit.Set(ctx, account, td); err != nil {
		return nil, fmt.Errorf("failed to update trust deposit: %w", err)
	}

	// [MOD-TD-MSG-6-2-2] / [MOD-TD-MSG-6-3] Transfer amount from authority to TrustDeposit account
	transferCoins := sdk.NewCoins(sdk.NewInt64Coin(types.BondDenom, int64(msg.Amount)))
	if err := ms.Keeper.bankKeeper.SendCoinsFromAccountToModule(
		ctx,
		authorityAddr,
		types.ModuleName,
		transferCoins,
	); err != nil {
		return nil, fmt.Errorf("failed to transfer tokens: %w", err)
	}

	// Emit event
	ctx.EventManager().EmitEvents(sdk.Events{
		sdk.NewEvent(
			types.EventTypeRepaySlashedTrustDeposit,
			sdk.NewAttribute(types.AttributeKeyAccount, account),
			sdk.NewAttribute(types.AttributeKeyAmount, strconv.FormatUint(msg.Amount, 10)),
			sdk.NewAttribute(types.AttributeKeyTimestamp, ctx.BlockTime().String()),
		),
	})

	return &types.MsgRepaySlashedTrustDepositResponse{}, nil
}
