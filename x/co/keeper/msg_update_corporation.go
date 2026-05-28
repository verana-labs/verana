package keeper

import (
	"context"
	"fmt"

	"cosmossdk.io/errors"
	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/verana-labs/verana/x/co/types"
)

// UpdateCorporation implements MOD-CO-MSG-2.
//
// Rotates the DID of an existing Corporation. The corporation field on the msg
// is the policy_address (= AUTHZ-CHECK-5 subject); the operator must hold a
// delegation from that Corporation for THIS msg type. Language is immutable.
func (ms msgServer) UpdateCorporation(goCtx context.Context, msg *types.MsgUpdateCorporation) (*types.MsgUpdateCorporationResponse, error) {
	if err := msg.ValidateBasic(); err != nil {
		return nil, err
	}
	ctx := sdk.UnwrapSDKContext(goCtx)

	// AUTHZ-CHECK-5: resolve policy_address → corporation_id.
	coID, err := ms.CorporationByPolicyAddr.Get(ctx, msg.Corporation)
	if err != nil {
		return nil, errors.Wrap(types.ErrCorporationNotRegistered, msg.Corporation)
	}
	co, err := ms.Corporation.Get(ctx, coID)
	if err != nil {
		// Should never happen: the reverse index just resolved this id but the
		// primary table doesn't have it. This is a store-inconsistency, not a
		// user-facing not-found.
		return nil, fmt.Errorf("store inconsistency: corporation id %d resolved by index but missing from primary: %w", coID, err)
	}

	// AUTHZ-CHECK-1: operator must hold a delegation from the Corporation for
	// this msg type. msg.Corporation is the policy_address — DE's check accepts
	// the address form.
	if err := ms.delegationKeeper.CheckOperatorAuthorization(ctx, msg.Corporation, msg.Operator, sdk.MsgTypeURL(msg), ctx.BlockTime()); err != nil {
		return nil, err
	}

	if msg.Did == co.Did {
		// No-op: nothing to mutate, no event.
		return &types.MsgUpdateCorporationResponse{}, nil
	}

	// DID uniqueness gate: the new did must not be bound to a different Corporation.
	if has, err := ms.CorporationByDID.Has(ctx, msg.Did); err != nil {
		return nil, fmt.Errorf("check did uniqueness: %w", err)
	} else if has {
		return nil, errors.Wrap(types.ErrDIDAlreadyExists, msg.Did)
	}

	// Swap the DID index (remove old, add new) and update the Corporation row.
	if err := ms.CorporationByDID.Remove(ctx, co.Did); err != nil {
		return nil, fmt.Errorf("remove old did index: %w", err)
	}
	if err := ms.CorporationByDID.Set(ctx, msg.Did, co.Id); err != nil {
		return nil, fmt.Errorf("persist new did index: %w", err)
	}
	co.Did = msg.Did
	co.Modified = ctx.BlockTime()
	if err := ms.Corporation.Set(ctx, co.Id, co); err != nil {
		return nil, fmt.Errorf("persist corporation: %w", err)
	}

	ctx.EventManager().EmitEvent(sdk.NewEvent(
		types.EventTypeUpdateCorporation,
		sdk.NewAttribute(types.AttributeKeyCorporationID, fmt.Sprintf("%d", co.Id)),
		sdk.NewAttribute(types.AttributeKeyPolicyAddress, co.PolicyAddress),
		sdk.NewAttribute(types.AttributeKeyDID, msg.Did),
	))

	return &types.MsgUpdateCorporationResponse{}, nil
}
