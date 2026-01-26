package keeper

import (
	"fmt"
	"time"

	"cosmossdk.io/collections"
	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/verana-labs/verana/x/td/types"
)

// OperatorAllowanceKey returns the composite key for an operator allowance
func OperatorAllowanceKeyFromGroupOperator(group, operator string) string {
	return fmt.Sprintf("%s/%s", group, operator)
}

// SetOperatorAllowance creates or updates an operator allowance record
func (k Keeper) SetOperatorAllowance(ctx sdk.Context, allowance types.OperatorAllowance) error {
	key := OperatorAllowanceKeyFromGroupOperator(allowance.Group, allowance.Operator)
	return k.OperatorAllowance.Set(ctx, key, allowance)
}

// GetOperatorAllowance retrieves an operator allowance record
func (k Keeper) GetOperatorAllowance(ctx sdk.Context, group, operator string) (types.OperatorAllowance, error) {
	key := OperatorAllowanceKeyFromGroupOperator(group, operator)
	return k.OperatorAllowance.Get(ctx, key)
}

// HasOperatorAllowance checks if an operator allowance record exists
func (k Keeper) HasOperatorAllowance(ctx sdk.Context, group, operator string) bool {
	key := OperatorAllowanceKeyFromGroupOperator(group, operator)
	has, _ := k.OperatorAllowance.Has(ctx, key)
	return has
}

// RemoveOperatorAllowance deletes an operator allowance record
func (k Keeper) RemoveOperatorAllowance(ctx sdk.Context, group, operator string) error {
	key := OperatorAllowanceKeyFromGroupOperator(group, operator)
	return k.OperatorAllowance.Remove(ctx, key)
}

// IsAuthorizedOperator checks if an operator is authorized and active for a group
func (k Keeper) IsAuthorizedOperator(ctx sdk.Context, group, operator string) bool {
	allowance, err := k.GetOperatorAllowance(ctx, group, operator)
	if err != nil {
		return false
	}
	return allowance.Active
}

// IncrementOperatorUsage increments an operator's usage and checks against allowance
// This automatically resets usage if the reset period has elapsed
func (k Keeper) IncrementOperatorUsage(ctx sdk.Context, group, operator string, amount uint64) error {
	allowance, err := k.GetOperatorAllowance(ctx, group, operator)
	if err != nil {
		return fmt.Errorf("operator not found: %w", err)
	}

	if !allowance.Active {
		return fmt.Errorf("operator is not active")
	}

	currentTime := ctx.BlockTime()

	// Check if reset period has elapsed
	if allowance.ResetPeriodSeconds > 0 && allowance.LastResetAt != nil {
		timeSinceReset := currentTime.Sub(*allowance.LastResetAt)
		resetPeriod := time.Duration(allowance.ResetPeriodSeconds) * time.Second

		if timeSinceReset >= resetPeriod {
			// Reset usage
			allowance.Usage = 0
			allowance.LastResetAt = &currentTime
		}
	}

	// Check if usage would exceed allowance
	if allowance.Usage+amount > allowance.Allowance {
		return fmt.Errorf("usage would exceed allowance: current=%d, requested=%d, limit=%d",
			allowance.Usage, amount, allowance.Allowance)
	}

	// Update usage
	allowance.Usage += amount
	allowance.LastUsageAt = &currentTime

	return k.SetOperatorAllowance(ctx, allowance)
}

// ResetOperatorUsage manually resets an operator's usage counter
func (k Keeper) ResetOperatorUsage(ctx sdk.Context, group, operator string) error {
	allowance, err := k.GetOperatorAllowance(ctx, group, operator)
	if err != nil {
		return err
	}

	currentTime := ctx.BlockTime()
	allowance.Usage = 0
	allowance.LastResetAt = &currentTime

	return k.SetOperatorAllowance(ctx, allowance)
}

// GetAllOperatorAllowances returns all operator allowances (for genesis export)
func (k Keeper) GetAllOperatorAllowances(ctx sdk.Context) ([]types.OperatorAllowance, error) {
	var allowances []types.OperatorAllowance

	err := k.OperatorAllowance.Walk(ctx, nil, func(key string, value types.OperatorAllowance) (stop bool, err error) {
		allowances = append(allowances, value)
		return false, nil
	})
	if err != nil {
		return nil, err
	}

	return allowances, nil
}

// GetOperatorAllowancesByGroup returns all operator allowances for a specific group
func (k Keeper) GetOperatorAllowancesByGroup(ctx sdk.Context, group string) ([]types.OperatorAllowance, error) {
	var allowances []types.OperatorAllowance
	prefix := collections.NewPrefixedPairRange[string, string](group)

	err := k.OperatorAllowance.Walk(ctx, nil, func(key string, value types.OperatorAllowance) (stop bool, err error) {
		if value.Group == group {
			allowances = append(allowances, value)
		}
		return false, nil
	})
	if err != nil {
		return nil, err
	}
	_ = prefix // Suppress unused variable warning

	return allowances, nil
}
