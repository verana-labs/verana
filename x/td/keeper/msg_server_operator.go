package keeper

import (
	"context"
	"fmt"
	"strconv"

	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/verana-labs/verana/x/td/types"
)

// AddOperator adds an operator to a group with specified allowance
func (ms msgServer) AddOperator(goCtx context.Context, msg *types.MsgAddOperator) (*types.MsgAddOperatorResponse, error) {
	ctx := sdk.UnwrapSDKContext(goCtx)

	// Validate creator is a valid address (should be a group policy address)
	_, err := sdk.AccAddressFromBech32(msg.Creator)
	if err != nil {
		return nil, fmt.Errorf("invalid creator address: %w", err)
	}

	// Validate operator is a valid address
	_, err = sdk.AccAddressFromBech32(msg.Operator)
	if err != nil {
		return nil, fmt.Errorf("invalid operator address: %w", err)
	}

	// Check if operator already exists for this group
	if ms.Keeper.HasOperatorAllowance(ctx, msg.Creator, msg.Operator) {
		return nil, fmt.Errorf("operator already exists for this group")
	}

	currentTime := ctx.BlockTime()

	allowance := types.OperatorAllowance{
		Group:              msg.Creator,
		Operator:           msg.Operator,
		Allowance:          msg.Allowance,
		Usage:              0,
		ResetPeriodSeconds: msg.ResetPeriodSeconds,
		LastResetAt:        &currentTime,
		LastUsageAt:        nil,
		Active:             true,
	}

	if err := ms.Keeper.SetOperatorAllowance(ctx, allowance); err != nil {
		return nil, fmt.Errorf("failed to set operator allowance: %w", err)
	}

	ctx.EventManager().EmitEvent(
		sdk.NewEvent(
			"operator_added",
			sdk.NewAttribute("group", msg.Creator),
			sdk.NewAttribute("operator", msg.Operator),
			sdk.NewAttribute("allowance", strconv.FormatUint(msg.Allowance, 10)),
		),
	)

	return &types.MsgAddOperatorResponse{}, nil
}

// RemoveOperator removes an operator from a group
func (ms msgServer) RemoveOperator(goCtx context.Context, msg *types.MsgRemoveOperator) (*types.MsgRemoveOperatorResponse, error) {
	ctx := sdk.UnwrapSDKContext(goCtx)

	// Validate addresses
	_, err := sdk.AccAddressFromBech32(msg.Creator)
	if err != nil {
		return nil, fmt.Errorf("invalid creator address: %w", err)
	}

	_, err = sdk.AccAddressFromBech32(msg.Operator)
	if err != nil {
		return nil, fmt.Errorf("invalid operator address: %w", err)
	}

	// Check if operator exists
	if !ms.Keeper.HasOperatorAllowance(ctx, msg.Creator, msg.Operator) {
		return nil, fmt.Errorf("operator not found for this group")
	}

	if err := ms.Keeper.RemoveOperatorAllowance(ctx, msg.Creator, msg.Operator); err != nil {
		return nil, fmt.Errorf("failed to remove operator: %w", err)
	}

	ctx.EventManager().EmitEvent(
		sdk.NewEvent(
			"operator_removed",
			sdk.NewAttribute("group", msg.Creator),
			sdk.NewAttribute("operator", msg.Operator),
		),
	)

	return &types.MsgRemoveOperatorResponse{}, nil
}

// UpdateOperatorAllowance updates an operator's allowance configuration
func (ms msgServer) UpdateOperatorAllowance(goCtx context.Context, msg *types.MsgUpdateOperatorAllowance) (*types.MsgUpdateOperatorAllowanceResponse, error) {
	ctx := sdk.UnwrapSDKContext(goCtx)

	// Validate addresses
	_, err := sdk.AccAddressFromBech32(msg.Creator)
	if err != nil {
		return nil, fmt.Errorf("invalid creator address: %w", err)
	}

	_, err = sdk.AccAddressFromBech32(msg.Operator)
	if err != nil {
		return nil, fmt.Errorf("invalid operator address: %w", err)
	}

	// Get existing operator
	allowance, err := ms.Keeper.GetOperatorAllowance(ctx, msg.Creator, msg.Operator)
	if err != nil {
		return nil, fmt.Errorf("operator not found: %w", err)
	}

	// Update fields
	allowance.Allowance = msg.Allowance
	allowance.ResetPeriodSeconds = msg.ResetPeriodSeconds

	if err := ms.Keeper.SetOperatorAllowance(ctx, allowance); err != nil {
		return nil, fmt.Errorf("failed to update operator: %w", err)
	}

	ctx.EventManager().EmitEvent(
		sdk.NewEvent(
			"operator_allowance_updated",
			sdk.NewAttribute("group", msg.Creator),
			sdk.NewAttribute("operator", msg.Operator),
			sdk.NewAttribute("new_allowance", strconv.FormatUint(msg.Allowance, 10)),
		),
	)

	return &types.MsgUpdateOperatorAllowanceResponse{}, nil
}

// ResetOperatorUsage manually resets an operator's usage counter
func (ms msgServer) ResetOperatorUsage(goCtx context.Context, msg *types.MsgResetOperatorUsage) (*types.MsgResetOperatorUsageResponse, error) {
	ctx := sdk.UnwrapSDKContext(goCtx)

	// Validate addresses
	_, err := sdk.AccAddressFromBech32(msg.Creator)
	if err != nil {
		return nil, fmt.Errorf("invalid creator address: %w", err)
	}

	_, err = sdk.AccAddressFromBech32(msg.Operator)
	if err != nil {
		return nil, fmt.Errorf("invalid operator address: %w", err)
	}

	if err := ms.Keeper.ResetOperatorUsage(ctx, msg.Creator, msg.Operator); err != nil {
		return nil, fmt.Errorf("failed to reset usage: %w", err)
	}

	ctx.EventManager().EmitEvent(
		sdk.NewEvent(
			"operator_usage_reset",
			sdk.NewAttribute("group", msg.Creator),
			sdk.NewAttribute("operator", msg.Operator),
		),
	)

	return &types.MsgResetOperatorUsageResponse{}, nil
}
