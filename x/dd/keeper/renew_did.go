package keeper

import (
	"errors"
	"fmt"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/verana-labs/verana/x/dd/types"
)

func (ms msgServer) validateRenewDIDParams(ctx sdk.Context, msg *types.MsgRenewDID) error {
	if msg.Did == "" {
		return errors.New("DID is required")
	}

	// Validate DID format
	if !isValidDID(msg.Did) {
		return errors.New("invalid DID syntax")
	}

	// Get existing DID entry
	didEntry, err := ms.DIDDirectory.Get(ctx, msg.Did)
	if err != nil {
		return fmt.Errorf("DID not found: %w", err)
	}

	// Check if caller is the controller
	if didEntry.Controller != msg.Creator {
		return errors.New("only the controller can renew a DID")
	}

	// Validate years (1-31)
	years := msg.Years
	if years == 0 {
		years = 1
	}
	if years > 31 {
		return errors.New("years must be between 1 and 31")
	}

	return nil
}

func (ms msgServer) executeRenewDID(ctx sdk.Context, msg *types.MsgRenewDID) error {
	params := ms.GetParams(ctx)
	trustUnitPrice := ms.trustRegistryKeeper.GetTrustUnitPrice(ctx)

	// Get existing DID entry
	didEntry, err := ms.DIDDirectory.Get(ctx, msg.Did)
	if err != nil {
		return fmt.Errorf("error retrieving DID: %w", err)
	}

	years := msg.Years
	if years == 0 {
		years = 1
	}

	now := ctx.BlockTime()
	// Add years to current expiration
	newExpiration := didEntry.Exp.AddDate(int(years), 0, 0)

	// Calculate additional deposit
	additionalDeposit := params.DidDirectoryTrustDeposit * trustUnitPrice * uint64(years)

	// Increase trust deposit
	if err := ms.trustDeposit.AdjustTrustDeposit(ctx, msg.Creator, int64(additionalDeposit)); err != nil {
		return fmt.Errorf("failed to adjust trust deposit: %w", err)
	}

	// Update DID entry
	didEntry.Modified = now
	didEntry.Exp = newExpiration
	didEntry.Deposit += int64(additionalDeposit)

	// Store the updated DID entry
	if err = ms.DIDDirectory.Set(ctx, msg.Did, didEntry); err != nil {
		return fmt.Errorf("failed to update DID: %w", err)
	}

	// Emit event
	ctx.EventManager().EmitEvent(
		sdk.NewEvent(
			types.EventTypeRenewDID,
			sdk.NewAttribute(types.AttributeKeyDID, msg.Did),
			sdk.NewAttribute(types.AttributeKeyController, msg.Creator),
			sdk.NewAttribute(types.AttributeKeyDeposit, fmt.Sprintf("%d", additionalDeposit)),
			sdk.NewAttribute(types.AttributeKeyYears, fmt.Sprintf("%d", years)),
		),
	)

	return nil
}
