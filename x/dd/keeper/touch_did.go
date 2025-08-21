package keeper

import (
	"errors"
	"fmt"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/verana-labs/verana/x/dd/types"
)

func (ms msgServer) validateTouchDIDParams(ctx sdk.Context, msg *types.MsgTouchDID) error {
	if msg.Did == "" {
		return errors.New("DID is required")
	}

	if !isValidDID(msg.Did) {
		return errors.New("invalid DID syntax")
	}

	// Check if DID exists
	_, err := ms.DIDDirectory.Get(ctx, msg.Did)
	if err != nil {
		return fmt.Errorf("DID not found: %w", err)
	}

	return nil
}

func (ms msgServer) executeTouchDID(ctx sdk.Context, msg *types.MsgTouchDID) error {
	// Get current DID entry
	didEntry, err := ms.DIDDirectory.Get(ctx, msg.Did)
	if err != nil {
		return fmt.Errorf("error retrieving DID: %w", err)
	}

	// Update modified time
	didEntry.Modified = ctx.BlockTime()

	// Save updated entry
	if err = ms.DIDDirectory.Set(ctx, msg.Did, didEntry); err != nil {
		return fmt.Errorf("failed to update DID: %w", err)
	}
	// Emit event
	ctx.EventManager().EmitEvent(
		sdk.NewEvent(
			types.EventTouchDID,
			sdk.NewAttribute(types.AttributeKeyDID, msg.Did),
			sdk.NewAttribute(types.AttributeKeyController, msg.Creator),
		),
	)
	return nil
}
