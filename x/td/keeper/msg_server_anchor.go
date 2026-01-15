package keeper

import (
	"context"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/verana-labs/verana/x/td/types"
)

// RegisterAnchor handles MsgRegisterAnchor.
func (ms msgServer) RegisterAnchor(goCtx context.Context, msg *types.MsgRegisterAnchor) (*types.MsgRegisterAnchorResponse, error) {
	ctx := sdk.UnwrapSDKContext(goCtx)

	// For POC: creator must be the anchor itself (group policy account)
	// In production, this would verify via x/group membership or x/authz
	if msg.Creator != msg.AnchorId {
		return nil, types.ErrUnauthorized.Wrap("creator must be the anchor (group policy account)")
	}

	if err := ms.Keeper.RegisterAnchor(ctx, msg.AnchorId, msg.GroupId, msg.Metadata); err != nil {
		return nil, err
	}

	return &types.MsgRegisterAnchorResponse{}, nil
}

// RegisterVerifiableService handles MsgRegisterVerifiableService.
func (ms msgServer) RegisterVerifiableService(goCtx context.Context, msg *types.MsgRegisterVerifiableService) (*types.MsgRegisterVerifiableServiceResponse, error) {
	ctx := sdk.UnwrapSDKContext(goCtx)

	// For POC: creator must be the anchor itself
	// In production, this would verify via x/authz grant from anchor
	if msg.Creator != msg.AnchorId {
		return nil, types.ErrUnauthorized.Wrap("creator must be the anchor")
	}

	if err := ms.Keeper.RegisterVerifiableService(ctx, msg.AnchorId, msg.OperatorAccount, msg.Metadata); err != nil {
		return nil, err
	}

	return &types.MsgRegisterVerifiableServiceResponse{}, nil
}

// SetOperatorAllowance handles MsgSetOperatorAllowance.
func (ms msgServer) SetOperatorAllowance(goCtx context.Context, msg *types.MsgSetOperatorAllowance) (*types.MsgSetOperatorAllowanceResponse, error) {
	ctx := sdk.UnwrapSDKContext(goCtx)

	// For POC: creator must be the anchor itself
	// In production, this would verify via x/authz grant from anchor
	if msg.Creator != msg.AnchorId {
		return nil, types.ErrUnauthorized.Wrap("creator must be the anchor")
	}

	if err := ms.Keeper.SetOperatorAllowance(ctx, msg.AnchorId, msg.OperatorAccount, msg.AllowanceLimit, msg.ResetPeriod); err != nil {
		return nil, err
	}

	return &types.MsgSetOperatorAllowanceResponse{}, nil
}
