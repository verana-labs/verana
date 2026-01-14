package keeper

import (
	"context"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/verana-labs/verana/x/dd/types"
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

func (ms msgServer) AddDID(goCtx context.Context, msg *types.MsgAddDID) (*types.MsgAddDIDResponse, error) {
	ctx := sdk.UnwrapSDKContext(goCtx)

	// Basic parameter validation
	if err := ms.validateAddDIDParams(ctx, msg); err != nil {
		return nil, err
	}

	// Fee checks
	if err := ms.checkSufficientFees(ctx, msg.Creator, msg.Years); err != nil {
		return nil, err
	}

	// Execute the addition
	if err := ms.executeAddDID(ctx, msg); err != nil {
		return nil, err
	}

	return &types.MsgAddDIDResponse{}, nil
}

func (ms msgServer) RenewDID(goCtx context.Context, msg *types.MsgRenewDID) (*types.MsgRenewDIDResponse, error) {
	ctx := sdk.UnwrapSDKContext(goCtx)

	// Basic parameter validation
	if err := ms.validateRenewDIDParams(ctx, msg); err != nil {
		return nil, err
	}

	// Fee checks
	if err := ms.checkSufficientFees(ctx, msg.Creator, msg.Years); err != nil {
		return nil, err
	}

	// Execute the renewal
	if err := ms.executeRenewDID(ctx, msg); err != nil {
		return nil, err
	}

	return &types.MsgRenewDIDResponse{}, nil
}

func (ms msgServer) RemoveDID(goCtx context.Context, msg *types.MsgRemoveDID) (*types.MsgRemoveDIDResponse, error) {
	ctx := sdk.UnwrapSDKContext(goCtx)

	if err := ms.validateRemoveDIDParams(ctx, msg); err != nil {
		return nil, err
	}

	if err := ms.checkSufficientFees(ctx, msg.Creator, 0); err != nil {
		return nil, err
	}

	if err := ms.executeRemoveDID(ctx, msg); err != nil {
		return nil, err
	}

	return &types.MsgRemoveDIDResponse{}, nil
}

func (ms msgServer) TouchDID(goCtx context.Context, msg *types.MsgTouchDID) (*types.MsgTouchDIDResponse, error) {
	ctx := sdk.UnwrapSDKContext(goCtx)

	if err := ms.validateTouchDIDParams(ctx, msg); err != nil {
		return nil, err
	}

	if err := ms.executeTouchDID(ctx, msg); err != nil {
		return nil, err
	}

	return &types.MsgTouchDIDResponse{}, nil
}

// adjustTrustDepositAnchorAware is an anchor-aware version of AdjustTrustDeposit.
// It checks if the account is an anchor or VS operator and routes to the appropriate method.
// Issue #185: Anchor-Based Trust Deposit Architecture
func (ms msgServer) adjustTrustDepositAnchorAware(ctx sdk.Context, account string, augend int64) error {
	// Check if account is an anchor
	if ms.trustDeposit.IsAnchor(ctx, account) {
		// Direct anchor operation - no operator limit check
		return ms.trustDeposit.AdjustAnchorTrustDeposit(ctx, account, augend, "")
	}

	// Check if account is a VS operator
	if anchorID, err := ms.trustDeposit.GetAnchorForOperator(ctx, account); err == nil && anchorID != "" {
		// Operator acting on behalf of anchor - with limit enforcement
		return ms.trustDeposit.AdjustAnchorTrustDeposit(ctx, anchorID, augend, account)
	}

	// Regular account operation (backward compatible)
	return ms.trustDeposit.AdjustTrustDeposit(ctx, account, augend)
}
