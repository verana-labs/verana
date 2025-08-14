package keeper

import (
	"errors"
	"fmt"
	"time"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/verana-labs/verana/x/diddirectory/types"
)

func (ms msgServer) validateRemoveDIDParams(ctx sdk.Context, msg *types.MsgRemoveDID) error {
	if msg.Did == "" {
		return errors.New("DID is required")
	}

	if !isValidDID(msg.Did) {
		return errors.New("invalid DID syntax")
	}

	// Get DID entry
	didEntry, err := ms.DIDDirectory.Get(ctx, msg.Did)
	if err != nil {
		return fmt.Errorf("DID not found: %w", err)
	}

	// Get grace period
	params := ms.GetParams(ctx)
	gracePeriod := time.Duration(params.DidDirectoryGracePeriod) * 24 * time.Hour

	// Check authorization
	now := ctx.BlockTime()
	if now.Before(didEntry.Exp.Add(gracePeriod)) {
		// Before grace period: only controller can remove
		if msg.Creator != didEntry.Controller {
			return errors.New("only the controller can remove this DID before grace period")
		}
	}

	return nil
}

func (ms msgServer) executeRemoveDID(ctx sdk.Context, msg *types.MsgRemoveDID) error {
	// Load DidDirectory entry dd
	didEntry, err := ms.DIDDirectory.Get(ctx, msg.Did)
	if err != nil {
		return fmt.Errorf("error retrieving DID: %w", err)
	}

	// Use [MOD-TD-MSG-1] to decrease by dd.deposit the trust deposit of dd.controller account
	if didEntry.Deposit > 0 {
		// Convert to signed integer for adjustment
		depositAmount := didEntry.Deposit

		// Use negative value to decrease deposit and increase claimable amount
		if err := ms.trustDeposit.AdjustTrustDeposit(ctx, didEntry.Controller, -depositAmount); err != nil {
			return fmt.Errorf("failed to adjust trust deposit for controller %s: %w", didEntry.Controller, err)
		}
	}

	// Remove entry from DidDirectory
	if err = ms.DIDDirectory.Remove(ctx, msg.Did); err != nil {
		return fmt.Errorf("failed to remove DID: %w", err)
	}

	// Emit event
	ctx.EventManager().EmitEvent(
		sdk.NewEvent(
			types.EventTypeRemoveDID,
			sdk.NewAttribute(types.AttributeKeyDID, msg.Did),
			sdk.NewAttribute(types.AttributeKeyController, didEntry.Controller),
			sdk.NewAttribute(types.AttributeKeyDeposit, fmt.Sprintf("%d", didEntry.Deposit)),
		),
	)

	return nil
}
