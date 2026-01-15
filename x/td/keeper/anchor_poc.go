package keeper

import (
	"context"
	"fmt"

	"cosmossdk.io/collections"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/verana-labs/verana/x/td/types"
)

// =============================================================================
// ANCHOR MANAGEMENT
// =============================================================================

// RegisterAnchor registers a group policy address as an Anchor.
// For POC, we skip the group policy validation - just validate address format.
func (k Keeper) RegisterAnchor(ctx context.Context, anchorID string, groupID uint64, metadata string) error {
	sdkCtx := sdk.UnwrapSDKContext(ctx)

	// 1. Validate anchor address format
	_, err := sdk.AccAddressFromBech32(anchorID)
	if err != nil {
		return fmt.Errorf("invalid anchor address: %w", err)
	}

	// 2. Check if anchor already exists
	exists, err := k.Anchors.Has(ctx, anchorID)
	if err != nil {
		return err
	}
	if exists {
		return fmt.Errorf("anchor already registered: %s", anchorID)
	}

	// 3. Create Anchor record
	anchor := types.Anchor{
		AnchorId: anchorID,
		GroupId:  groupID,
		Created:  sdkCtx.BlockTime(),
		Metadata: metadata,
	}

	// 4. Save
	if err := k.Anchors.Set(ctx, anchorID, anchor); err != nil {
		return fmt.Errorf("failed to save anchor: %w", err)
	}

	// 5. Emit event
	sdkCtx.EventManager().EmitEvent(
		sdk.NewEvent(
			"anchor_registered",
			sdk.NewAttribute("anchor_id", anchorID),
			sdk.NewAttribute("group_id", fmt.Sprintf("%d", groupID)),
		),
	)

	return nil
}

// IsAnchor checks if an address is registered as an Anchor.
func (k Keeper) IsAnchor(ctx context.Context, anchorID string) bool {
	exists, _ := k.Anchors.Has(ctx, anchorID)
	return exists
}

// GetAnchorInfo retrieves an Anchor by ID (internal keeper method).
func (k Keeper) GetAnchorInfo(ctx context.Context, anchorID string) (types.Anchor, error) {
	return k.Anchors.Get(ctx, anchorID)
}

// =============================================================================
// VERIFIABLE SERVICE MANAGEMENT
// =============================================================================

// RegisterVerifiableService registers a hot operator key for an Anchor.
func (k Keeper) RegisterVerifiableService(
	ctx context.Context,
	anchorID string,
	operatorAccount string,
	metadata string,
) error {
	sdkCtx := sdk.UnwrapSDKContext(ctx)

	// 1. Verify Anchor exists
	if !k.IsAnchor(ctx, anchorID) {
		return fmt.Errorf("anchor not found: %s", anchorID)
	}

	// 2. Validate operator account address
	_, err := sdk.AccAddressFromBech32(operatorAccount)
	if err != nil {
		return fmt.Errorf("invalid operator address: %w", err)
	}

	// 3. Check if operator already registered to any Anchor
	existingVS, err := k.VerifiableServices.Get(ctx, operatorAccount)
	if err == nil {
		if existingVS.AnchorId != anchorID {
			return fmt.Errorf("operator already registered to anchor %s", existingVS.AnchorId)
		}
		// Already registered to same anchor - update metadata
		existingVS.Metadata = metadata
		return k.VerifiableServices.Set(ctx, operatorAccount, existingVS)
	}

	// 4. Create VS record
	vs := types.VerifiableService{
		AnchorId:        anchorID,
		OperatorAccount: operatorAccount,
		Registered:      sdkCtx.BlockTime(),
		Active:          true,
		Metadata:        metadata,
	}

	// 5. Save
	if err := k.VerifiableServices.Set(ctx, operatorAccount, vs); err != nil {
		return fmt.Errorf("failed to save verifiable service: %w", err)
	}

	// 6. Emit event
	sdkCtx.EventManager().EmitEvent(
		sdk.NewEvent(
			"vs_registered",
			sdk.NewAttribute("anchor_id", anchorID),
			sdk.NewAttribute("operator_account", operatorAccount),
		),
	)

	return nil
}

// GetAnchorForOperator resolves operator account to its Anchor.
func (k Keeper) GetAnchorForOperator(ctx context.Context, operatorAccount string) (string, error) {
	vs, err := k.VerifiableServices.Get(ctx, operatorAccount)
	if err != nil {
		return "", fmt.Errorf("operator not registered as VS: %w", err)
	}
	if !vs.Active {
		return "", fmt.Errorf("verifiable service is inactive")
	}
	return vs.AnchorId, nil
}

// IsVerifiableService checks if an account is a registered VS operator.
func (k Keeper) IsVerifiableService(ctx context.Context, operatorAccount string) bool {
	vs, err := k.VerifiableServices.Get(ctx, operatorAccount)
	return err == nil && vs.Active
}

// =============================================================================
// OPERATOR ALLOWANCE MANAGEMENT
// =============================================================================

// SetOperatorAllowance sets spending limits for a VS operator.
func (k Keeper) SetOperatorAllowance(
	ctx context.Context,
	anchorID string,
	operatorAccount string,
	allowanceLimit uint64,
	resetPeriod uint64,
) error {
	sdkCtx := sdk.UnwrapSDKContext(ctx)

	// 1. Verify VS exists for this anchor
	vs, err := k.VerifiableServices.Get(ctx, operatorAccount)
	if err != nil {
		return fmt.Errorf("operator not registered: %w", err)
	}
	if vs.AnchorId != anchorID {
		return fmt.Errorf("operator belongs to different anchor")
	}

	// 2. Create/Update allowance
	key := collections.Join(anchorID, operatorAccount)

	allowance := types.OperatorAllowance{
		AnchorId:        anchorID,
		OperatorAccount: operatorAccount,
		AllowanceLimit:  allowanceLimit,
		Spent:           0, // Reset spent on new allowance
		ResetPeriod:     resetPeriod,
		LastReset:       sdkCtx.BlockTime(),
	}

	if err := k.OperatorAllowances.Set(ctx, key, allowance); err != nil {
		return fmt.Errorf("failed to save allowance: %w", err)
	}

	// 3. Emit event
	sdkCtx.EventManager().EmitEvent(
		sdk.NewEvent(
			"operator_allowance_set",
			sdk.NewAttribute("anchor_id", anchorID),
			sdk.NewAttribute("operator_account", operatorAccount),
			sdk.NewAttribute("allowance_limit", fmt.Sprintf("%d", allowanceLimit)),
			sdk.NewAttribute("reset_period", fmt.Sprintf("%d", resetPeriod)),
		),
	)

	return nil
}

// GetOperatorAllowanceInfo retrieves allowance for an operator (internal keeper method).
func (k Keeper) GetOperatorAllowanceInfo(ctx context.Context, anchorID, operatorAccount string) (types.OperatorAllowance, error) {
	key := collections.Join(anchorID, operatorAccount)
	return k.OperatorAllowances.Get(ctx, key)
}

// =============================================================================
// ANCHOR TRUST DEPOSIT OPERATIONS
// =============================================================================

// AdjustAnchorTrustDeposit is the Anchor-aware version of AdjustTrustDeposit.
// This is called by other modules (dd, perm) when operating on behalf of an Anchor.
//
// Positive augend: Trust deposit increases (e.g., from DID registration)
// Negative augend: Trust deposit decreases (debited for operations)
//
// When operatorAccount is provided, allowance limits are enforced for debits.
func (k Keeper) AdjustAnchorTrustDeposit(
	ctx sdk.Context,
	anchorID string,
	augend int64,
	operatorAccount string,
) error {
	// 1. Verify Anchor exists
	if !k.IsAnchor(ctx, anchorID) {
		return fmt.Errorf("anchor not found: %s", anchorID)
	}

	// 2. Get existing trust deposit or create new one
	td, err := k.TrustDeposit.Get(ctx, anchorID)
	if err != nil {
		// Create new trust deposit for this anchor
		td = types.TrustDeposit{
			Account:   anchorID,
			Amount:    0,
			Share:     k.AmountToShare(0, k.GetParams(ctx).TrustDepositShareValue),
			Claimable: 0,
		}
	}

	// 3. For debits (negative augend), check operator allowance
	if augend < 0 && operatorAccount != "" {
		// Validate operator belongs to this anchor
		resolvedAnchor, err := k.GetAnchorForOperator(ctx, operatorAccount)
		if err != nil {
			return fmt.Errorf("operator not authorized: %w", err)
		}
		if resolvedAnchor != anchorID {
			return fmt.Errorf("operator belongs to different anchor: expected %s, got %s", anchorID, resolvedAnchor)
		}

		// Check and update allowance
		if err := k.checkAndUpdateAllowance(ctx, anchorID, operatorAccount, uint64(-augend)); err != nil {
			return err
		}
	}

	// 4. Apply adjustment
	params := k.GetParams(ctx)
	if augend >= 0 {
		// Positive adjustment (accumulation from operations)
		td.Amount += uint64(augend)
		shareIncrease := k.AmountToShare(uint64(augend), params.TrustDepositShareValue)
		td.Share = td.Share.Add(shareIncrease)
	} else {
		// Negative adjustment (debit)
		debitAmount := uint64(-augend)
		if td.Amount < debitAmount {
			return fmt.Errorf("insufficient trust deposit: have %d, need %d", td.Amount, debitAmount)
		}
		td.Amount -= debitAmount
		shareReduction := k.AmountToShare(debitAmount, params.TrustDepositShareValue)
		td.Share = td.Share.Sub(shareReduction)
	}

	// 5. Save
	if err := k.TrustDeposit.Set(ctx, anchorID, td); err != nil {
		return fmt.Errorf("failed to save trust deposit: %w", err)
	}

	// 6. Emit event
	ctx.EventManager().EmitEvent(
		sdk.NewEvent(
			"anchor_trust_deposit_adjusted",
			sdk.NewAttribute("anchor_id", anchorID),
			sdk.NewAttribute("adjustment", fmt.Sprintf("%d", augend)),
			sdk.NewAttribute("operator", operatorAccount),
			sdk.NewAttribute("new_amount", fmt.Sprintf("%d", td.Amount)),
		),
	)

	return nil
}

// DebitAnchorTrustDeposit debits from an Anchor's trust deposit.
// This is a convenience wrapper around AdjustAnchorTrustDeposit for debits.
func (k Keeper) DebitAnchorTrustDeposit(
	ctx sdk.Context,
	anchorID string,
	amount uint64,
	operatorAccount string,
	reason string,
) error {
	return k.AdjustAnchorTrustDeposit(ctx, anchorID, -int64(amount), operatorAccount)
}

// checkAndUpdateAllowance validates and updates operator spending allowance.
func (k Keeper) checkAndUpdateAllowance(ctx sdk.Context, anchorID, operatorAccount string, amount uint64) error {
	key := collections.Join(anchorID, operatorAccount)

	allowance, err := k.OperatorAllowances.Get(ctx, key)
	if err != nil {
		return fmt.Errorf("no allowance found for operator %s", operatorAccount)
	}

	// Check if reset period has passed
	elapsed := ctx.BlockTime().Sub(allowance.LastReset)
	if elapsed.Seconds() >= float64(allowance.ResetPeriod) {
		// Reset spent amount
		allowance.Spent = 0
		allowance.LastReset = ctx.BlockTime()
	}

	// Check if spend would exceed allowance
	if allowance.Spent+amount > allowance.AllowanceLimit {
		return fmt.Errorf("spend amount %d exceeds remaining allowance %d (limit: %d, spent: %d)",
			amount, allowance.AllowanceLimit-allowance.Spent, allowance.AllowanceLimit, allowance.Spent)
	}

	// Update spent
	allowance.Spent += amount

	// Save
	return k.OperatorAllowances.Set(ctx, key, allowance)
}
