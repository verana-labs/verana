package keeper

import (
	"context"
	"fmt"
	"strconv"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/verana-labs/verana/x/tr/types"
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

func (ms msgServer) CreateTrustRegistry(goCtx context.Context, msg *types.MsgCreateTrustRegistry) (*types.MsgCreateTrustRegistryResponse, error) {
	ctx := sdk.UnwrapSDKContext(goCtx)

	// [MOD-TR-MSG-1-3] Create New Trust Registry execution
	now := ctx.BlockTime()

	// Determine the controller (group or creator)
	operator := msg.Creator
	controller := msg.Creator

	// If group is specified, validate operator authorization
	if msg.Group != "" {
		controller = msg.Group

		// Check if operator is authorized for this group
		if !ms.Keeper.trustDeposit.IsAuthorizedOperator(ctx, msg.Group, operator) {
			return nil, fmt.Errorf("operator %s is not authorized for group %s", operator, msg.Group)
		}
	}

	// Calculate trust deposit amount
	params := ms.Keeper.GetParams(ctx)
	trustDeposit := params.TrustRegistryTrustDeposit * params.TrustUnitPrice

	// Increase trust deposit from controller (group or creator)
	if err := ms.Keeper.trustDeposit.AdjustTrustDeposit(ctx, controller, int64(trustDeposit)); err != nil {
		return nil, fmt.Errorf("failed to adjust trust deposit: %w", err)
	}

	// Track operator usage if group is specified
	if msg.Group != "" {
		if err := ms.Keeper.trustDeposit.IncrementOperatorUsage(ctx, msg.Group, operator, trustDeposit); err != nil {
			return nil, fmt.Errorf("failed to increment operator usage: %w", err)
		}
	}

	tr, gfv, gfd, err := ms.createTrustRegistryEntries(ctx, msg, now)
	if err != nil {
		return nil, err
	}

	// Update trust deposit amount in the trust registry entry
	tr.Deposit = int64(trustDeposit)
	// Set controller to group if specified, otherwise creator
	tr.Controller = controller

	if err := ms.persistEntries(ctx, tr, gfv, gfd); err != nil {
		return nil, err
	}

	ctx.EventManager().EmitEvents(sdk.Events{
		sdk.NewEvent(
			types.EventTypeCreateTrustRegistry,
			sdk.NewAttribute(types.AttributeKeyTrustRegistryID, strconv.FormatUint(tr.Id, 10)),
			sdk.NewAttribute(types.AttributeKeyDID, tr.Did),
			sdk.NewAttribute(types.AttributeKeyController, tr.Controller),
			sdk.NewAttribute(types.AttributeKeyAka, tr.Aka),
			sdk.NewAttribute(types.AttributeKeyLanguage, tr.Language),
			sdk.NewAttribute(types.AttributeKeyDeposit, strconv.FormatUint(uint64(tr.Deposit), 10)),
			sdk.NewAttribute(types.AttributeKeyTimestamp, now.String()),
			sdk.NewAttribute("operator", operator),
		),
		sdk.NewEvent(
			types.EventTypeCreateGovernanceFrameworkVersion,
			sdk.NewAttribute(types.AttributeKeyGFVersionID, strconv.FormatUint(gfv.Id, 10)),
			sdk.NewAttribute(types.AttributeKeyTrustRegistryID, strconv.FormatUint(gfv.TrId, 10)),
			sdk.NewAttribute(types.AttributeKeyVersion, strconv.FormatUint(uint64(gfv.Version), 10)),
		),
		sdk.NewEvent(
			types.EventTypeCreateGovernanceFrameworkDocument,
			sdk.NewAttribute(types.AttributeKeyGFDocumentID, strconv.FormatUint(gfd.Id, 10)),
			sdk.NewAttribute(types.AttributeKeyGFVersionID, strconv.FormatUint(gfd.GfvId, 10)),
			sdk.NewAttribute(types.AttributeKeyDocURL, gfd.Url),
			sdk.NewAttribute(types.AttributeKeyDigestSri, gfd.DigestSri),
		),
	})

	return &types.MsgCreateTrustRegistryResponse{}, nil
}

func (ms msgServer) AddGovernanceFrameworkDocument(goCtx context.Context, msg *types.MsgAddGovernanceFrameworkDocument) (*types.MsgAddGovernanceFrameworkDocumentResponse, error) {
	ctx := sdk.UnwrapSDKContext(goCtx)

	if err := ms.validateAddGovernanceFrameworkDocumentParams(ctx, msg); err != nil {
		return nil, err
	}

	if err := ms.executeAddGovernanceFrameworkDocument(ctx, msg); err != nil {
		return nil, err
	}

	ctx.EventManager().EmitEvents(sdk.Events{
		sdk.NewEvent(
			types.EventTypeAddGovernanceFrameworkDocument,
			sdk.NewAttribute(types.AttributeKeyGFVersionID, strconv.FormatUint(msg.Id, 10)),
			sdk.NewAttribute(types.AttributeKeyVersion, strconv.FormatInt(int64(msg.Version), 10)),
			sdk.NewAttribute(types.AttributeKeyLanguage, msg.DocLanguage),
			sdk.NewAttribute(types.AttributeKeyDocURL, msg.DocUrl),
			sdk.NewAttribute(types.AttributeKeyDigestSri, msg.DocDigestSri),
			sdk.NewAttribute(types.AttributeKeyController, msg.Creator),
			sdk.NewAttribute(types.AttributeKeyTimestamp, ctx.BlockTime().String()),
		),
	})

	return &types.MsgAddGovernanceFrameworkDocumentResponse{}, nil
}

func (ms msgServer) IncreaseActiveGovernanceFrameworkVersion(goCtx context.Context, msg *types.MsgIncreaseActiveGovernanceFrameworkVersion) (*types.MsgIncreaseActiveGovernanceFrameworkVersionResponse, error) {
	ctx := sdk.UnwrapSDKContext(goCtx)

	// Validate parameters
	if err := ms.validateIncreaseActiveGovernanceFrameworkVersionParams(ctx, msg); err != nil {
		return nil, err
	}

	// Execute the increase
	if err := ms.executeIncreaseActiveGovernanceFrameworkVersion(ctx, msg); err != nil {
		return nil, err
	}

	ctx.EventManager().EmitEvents(sdk.Events{
		sdk.NewEvent(
			types.EventTypeIncreaseActiveGFVersion,
			sdk.NewAttribute(types.AttributeKeyTrustRegistryID, strconv.FormatUint(msg.Id, 10)),
			sdk.NewAttribute(types.AttributeKeyController, msg.Creator),
			sdk.NewAttribute(types.AttributeKeyTimestamp, ctx.BlockTime().String()),
		),
	})

	return &types.MsgIncreaseActiveGovernanceFrameworkVersionResponse{}, nil
}

func (ms msgServer) UpdateTrustRegistry(goCtx context.Context, msg *types.MsgUpdateTrustRegistry) (*types.MsgUpdateTrustRegistryResponse, error) {
	ctx := sdk.UnwrapSDKContext(goCtx)

	// Get trust registry
	tr, err := ms.TrustRegistry.Get(ctx, msg.Id)
	if err != nil {
		return nil, fmt.Errorf("trust registry not found: %w", err)
	}

	// Check controller
	if tr.Controller != msg.Creator {
		return nil, fmt.Errorf("only trust registry controller can update trust registry")
	}

	// Update fields
	tr.Did = msg.Did
	tr.Aka = msg.Aka
	tr.Modified = ctx.BlockTime()

	// Save updated trust registry
	if err := ms.TrustRegistry.Set(ctx, tr.Id, tr); err != nil {
		return nil, fmt.Errorf("failed to update trust registry: %w", err)
	}

	ctx.EventManager().EmitEvents(sdk.Events{
		sdk.NewEvent(
			types.EventTypeUpdateTrustRegistry,
			sdk.NewAttribute(types.AttributeKeyTrustRegistryID, strconv.FormatUint(msg.Id, 10)),
			sdk.NewAttribute(types.AttributeKeyController, msg.Creator),
			sdk.NewAttribute(types.AttributeKeyDID, msg.Did),
			sdk.NewAttribute(types.AttributeKeyAka, msg.Aka),
			sdk.NewAttribute(types.AttributeKeyTimestamp, ctx.BlockTime().String()),
		),
	})

	return &types.MsgUpdateTrustRegistryResponse{}, nil
}

func (ms msgServer) ArchiveTrustRegistry(goCtx context.Context, msg *types.MsgArchiveTrustRegistry) (*types.MsgArchiveTrustRegistryResponse, error) {
	ctx := sdk.UnwrapSDKContext(goCtx)

	// Get trust registry
	tr, err := ms.TrustRegistry.Get(ctx, msg.Id)
	if err != nil {
		return nil, fmt.Errorf("trust registry not found: %w", err)
	}

	// Check controller
	if tr.Controller != msg.Creator {
		return nil, fmt.Errorf("only trust registry controller can archive trust registry")
	}

	// Check archive state
	if msg.Archive {
		if tr.Archived != nil {
			return nil, fmt.Errorf("trust registry is already archived")
		}
	} else {
		if tr.Archived == nil {
			return nil, fmt.Errorf("trust registry is not archived")
		}
	}

	// Update archive state
	now := ctx.BlockTime()
	if msg.Archive {
		tr.Archived = &now
	} else {
		tr.Archived = nil
	}
	tr.Modified = now

	// Save updated trust registry
	if err := ms.TrustRegistry.Set(ctx, tr.Id, tr); err != nil {
		return nil, fmt.Errorf("failed to update trust registry: %w", err)
	}

	archiveStatus := "archived"
	if !msg.Archive {
		archiveStatus = "unarchived"
	}

	ctx.EventManager().EmitEvents(sdk.Events{
		sdk.NewEvent(
			types.EventTypeArchiveTrustRegistry,
			sdk.NewAttribute(types.AttributeKeyTrustRegistryID, strconv.FormatUint(msg.Id, 10)),
			sdk.NewAttribute(types.AttributeKeyController, msg.Creator),
			sdk.NewAttribute(types.AttributeKeyArchiveStatus, archiveStatus),
			sdk.NewAttribute(types.AttributeKeyTimestamp, now.String()),
		),
	})

	return &types.MsgArchiveTrustRegistryResponse{}, nil
}
