package keeper

import (
	"context"
	"fmt"
	mathstd "math"
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
	account := msg.Corporation

	// [MOD-TD-MSG-2-2] [AUTHZ-CHECK] Verify operator authorization
	if ms.Keeper.delegationKeeper == nil {
		return nil, fmt.Errorf("delegation keeper is required for operator authorization")
	}
	if err := ms.Keeper.delegationKeeper.CheckOperatorAuthorization(
		ctx,
		msg.Corporation,
		msg.Operator,
		"/verana.td.v1.MsgReclaimTrustDepositYield",
		ctx.BlockTime(),
	); err != nil {
		return nil, fmt.Errorf("authorization check failed: %w", err)
	}

	// [MOD-TD-MSG-2-2-1] Load TrustDeposit entry
	td, err := ms.Keeper.TrustDeposit.Get(ctx, account)
	if err != nil {
		return nil, fmt.Errorf("trust deposit not found for account: %s", account)
	}

	// [MOD-TD-MSG-2-2-1] Spec: abort if slashed_deposit > 0 AND
	// repaid_deposit < slashed_deposit (outstanding slash exists).
	// slashed_deposit is cumulative; equality with repaid_deposit means fully repaid.
	if td.SlashedDeposit > 0 && td.RepaidDeposit < td.SlashedDeposit {
		return nil, fmt.Errorf("deposit has been slashed and not repaid")
	}

	// [MOD-TD-MSG-2-2-1] / [MOD-TD-MSG-2-3] Spec: claimable_yield is computed,
	// not stored. Formula: td.share * trust_deposit_share_value - td.deposit.
	// The stored td.Claimable field is owned by MOD-TD-MSG-1 (Adjust); reclaim
	// MUST NOT consult or mutate it.
	params := ms.Keeper.GetParams(ctx)
	yieldDec := td.Share.Mul(params.TrustDepositShareValue).Sub(math.LegacyNewDec(int64(td.Deposit)))
	if !yieldDec.IsPositive() {
		return nil, fmt.Errorf("no claimable yield")
	}
	claimed := yieldDec.TruncateInt().Uint64()

	// [MOD-TD-MSG-2-3] Reduce shares by claimable_yield / trust_deposit_share_value.
	sharesToReduce := ms.Keeper.AmountToShare(claimed, params.TrustDepositShareValue)
	td.Share = td.Share.Sub(sharesToReduce)

	// Validate corporation address
	addr, err := sdk.AccAddressFromBech32(account)
	if err != nil {
		return nil, fmt.Errorf("invalid corporation address: %w", err)
	}

	// Save updated trust deposit BEFORE bank transfer to ensure atomicity —
	// if Set fails, no coins have been transferred yet.
	if err := ms.Keeper.TrustDeposit.Set(ctx, account, td); err != nil {
		return nil, fmt.Errorf("failed to update trust deposit: %w", err)
	}

	// [MOD-TD-MSG-2-3] Transfer yield from TrustDeposit module account to corporation
	if claimed > uint64(mathstd.MaxInt64) {
		return nil, fmt.Errorf("amount exceeds maximum coin value: %d", claimed)
	}
	coins := sdk.NewCoins(sdk.NewInt64Coin(types.BondDenom, int64(claimed)))
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
			sdk.NewAttribute(types.AttributeKeyClaimedYield, strconv.FormatUint(claimed, 10)),
			sdk.NewAttribute(types.AttributeKeySharesReduced, sharesToReduce.String()),
			sdk.NewAttribute(types.AttributeKeyTimestamp, ctx.BlockTime().String()),
		),
	})

	return &types.MsgReclaimTrustDepositYieldResponse{
		ClaimedAmount: claimed,
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

// SlashTrustDeposit handles governance slashing of trust deposits
func (ms msgServer) SlashTrustDeposit(goCtx context.Context, msg *types.MsgSlashTrustDeposit) (*types.MsgSlashTrustDepositResponse, error) {
	ctx := sdk.UnwrapSDKContext(goCtx)

	// [CRITICAL] Authority check - only governance can call this
	if ms.Keeper.authority != msg.Authority {
		return nil, fmt.Errorf("invalid authority; expected %s, got %s", ms.Keeper.authority, msg.Authority)
	}

	// [MOD-TD-MSG-5-2-1] Basic checks
	if msg.Deposit.IsZero() || msg.Deposit.IsNegative() {
		return nil, fmt.Errorf("deposit must be greater than 0")
	}

	// [BUG-H2] Guard against uint64 overflow before calling Uint64()
	if !msg.Deposit.IsUint64() {
		return nil, fmt.Errorf("deposit amount exceeds uint64")
	}

	// Check if TrustDeposit entry exists for the corporation
	td, err := ms.Keeper.TrustDeposit.Get(ctx, msg.Corporation)
	if err != nil {
		return nil, fmt.Errorf("trust deposit not found for corporation: %s", msg.Corporation)
	}

	// Check if deposit is sufficient
	if math.NewIntFromUint64(td.Deposit).LT(msg.Deposit) {
		return nil, fmt.Errorf("insufficient trust deposit: deposit=%d, required=%s", td.Deposit, msg.Deposit.String())
	}

	// [MOD-TD-MSG-5-3] Execute the slash
	now := ctx.BlockTime()

	// Get global variables for share calculation
	params := ms.Keeper.GetParams(ctx)
	shareValue := params.TrustDepositShareValue

	// Calculate share reduction
	shareReduction := math.LegacyNewDecFromInt(msg.Deposit).Quo(shareValue)

	// [MOD-TD-MSG-5-3] Update TrustDeposit entry
	slashAmt := msg.Deposit.Uint64()
	td.Deposit = td.Deposit - slashAmt
	td.Share = td.Share.Sub(shareReduction)
	td.SlashedDeposit = td.SlashedDeposit + slashAmt
	td.LastSlashed = &now
	td.SlashCount++

	// Save state BEFORE burning to ensure atomicity — if Set fails,
	// no coins are burned.
	if err := ms.Keeper.TrustDeposit.Set(ctx, msg.Corporation, td); err != nil {
		return nil, fmt.Errorf("failed to save trust deposit: %w", err)
	}

	// [MOD-TD-MSG-5-3] Burn `amount` from TrustDeposit account immediately.
	if slashAmt > uint64(mathstd.MaxInt64) {
		return nil, fmt.Errorf("slash amount exceeds int64: %d", slashAmt)
	}
	burnCoins := sdk.NewCoins(sdk.NewInt64Coin(types.BondDenom, int64(slashAmt)))
	if err := ms.Keeper.bankKeeper.BurnCoins(ctx, types.ModuleName, burnCoins); err != nil {
		return nil, fmt.Errorf("failed to burn slashed coins: %w", err)
	}

	// Emit event
	ctx.EventManager().EmitEvent(
		sdk.NewEvent(
			types.EventTypeSlashTrustDeposit,
			sdk.NewAttribute(types.AttributeKeyAccount, msg.Corporation),
			sdk.NewAttribute(types.AttributeKeyAmount, msg.Deposit.String()),
			sdk.NewAttribute(types.AttributeKeySlashCount, strconv.FormatUint(td.SlashCount, 10)),
			sdk.NewAttribute("reason", msg.Reason),
			sdk.NewAttribute(types.AttributeKeyTimestamp, ctx.BlockTime().String()),
		),
	)

	return &types.MsgSlashTrustDepositResponse{}, nil
}

func (ms msgServer) RepaySlashedTrustDeposit(goCtx context.Context, msg *types.MsgRepaySlashedTrustDeposit) (*types.MsgRepaySlashedTrustDepositResponse, error) {
	ctx := sdk.UnwrapSDKContext(goCtx)
	account := msg.Corporation

	// [MOD-TD-MSG-6-2-1] [AUTHZ-CHECK] Verify operator authorization
	if ms.Keeper.delegationKeeper == nil {
		return nil, fmt.Errorf("delegation keeper is required for operator authorization")
	}
	if err := ms.Keeper.delegationKeeper.CheckOperatorAuthorization(
		ctx,
		msg.Corporation,
		msg.Operator,
		"/verana.td.v1.MsgRepaySlashedTrustDeposit",
		ctx.BlockTime(),
	); err != nil {
		return nil, fmt.Errorf("authorization check failed: %w", err)
	}

	// [MOD-TD-MSG-6-2-1] Load TrustDeposit entry for corporation (must exist)
	td, err := ms.Keeper.TrustDeposit.Get(ctx, account)
	if err != nil {
		return nil, fmt.Errorf("trust deposit entry not found for corporation %s: %w", account, err)
	}

	// [MOD-TD-MSG-6-2-1] Spec: amount MUST be exactly equal to
	// (td.slashed_deposit - td.repaid_deposit). slashed_deposit is a
	// cumulative counter; outstanding = slashed_deposit - repaid_deposit.
	if td.SlashedDeposit < td.RepaidDeposit {
		return nil, fmt.Errorf("invariant violated: repaid_deposit (%d) > slashed_deposit (%d)", td.RepaidDeposit, td.SlashedDeposit)
	}
	outstanding := td.SlashedDeposit - td.RepaidDeposit
	if msg.Deposit != outstanding {
		return nil, fmt.Errorf("deposit must exactly equal outstanding slashed amount: expected %d, got %d", outstanding, msg.Deposit)
	}

	// Validate corporation address for bank transfer
	corporationAddr, err := sdk.AccAddressFromBech32(account)
	if err != nil {
		return nil, fmt.Errorf("invalid corporation address: %w", err)
	}

	// [MOD-TD-MSG-6-3] Execution
	params := ms.Keeper.GetParams(ctx)
	now := ctx.BlockTime()

	// td.deposit += amount
	td.Deposit += msg.Deposit

	// td.share += amount / GlobalVariables.trust_deposit_share_value
	shareIncrease := ms.Keeper.AmountToShare(msg.Deposit, params.TrustDepositShareValue)
	td.Share = td.Share.Add(shareIncrease)

	// [MOD-TD-MSG-6-3] Spec: slashed_deposit is cumulative — DO NOT decrement.
	// Only repaid_deposit grows. Outstanding is derived as slashed - repaid.
	td.RepaidDeposit += msg.Deposit
	td.LastRepaid = &now

	// Save updated trust deposit BEFORE bank transfer to ensure atomicity
	if err := ms.Keeper.TrustDeposit.Set(ctx, account, td); err != nil {
		return nil, fmt.Errorf("failed to update trust deposit: %w", err)
	}

	// [MOD-TD-MSG-6-3] "add `amount` to TrustDeposit account": corporation
	// transfers fresh coins in. No burn here — the burn happened at slash time
	// per MOD-TD-MSG-5-3.
	if msg.Deposit > uint64(mathstd.MaxInt64) {
		return nil, fmt.Errorf("repay amount exceeds maximum coin amount: %d", msg.Deposit)
	}
	transferCoins := sdk.NewCoins(sdk.NewInt64Coin(types.BondDenom, int64(msg.Deposit)))
	if err := ms.Keeper.bankKeeper.SendCoinsFromAccountToModule(
		ctx,
		corporationAddr,
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
			sdk.NewAttribute(types.AttributeKeyAmount, strconv.FormatUint(msg.Deposit, 10)),
			sdk.NewAttribute(types.AttributeKeyTimestamp, ctx.BlockTime().String()),
		),
	})

	return &types.MsgRepaySlashedTrustDepositResponse{}, nil
}
