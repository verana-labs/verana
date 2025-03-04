package keeper

import (
	"context"
	"fmt"
	"strconv"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/verana-labs/verana-blockchain/x/trustregistry/types"
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

	// Calculate trust deposit amount
	params := ms.Keeper.GetParams(ctx)
	trustDeposit := params.TrustRegistryTrustDeposit * params.TrustUnitPrice

	// Increase trust deposit
	if err := ms.Keeper.trustDeposit.AdjustTrustDeposit(ctx, msg.Creator, int64(trustDeposit)); err != nil {
		return nil, fmt.Errorf("failed to adjust trust deposit: %w", err)
	}

	tr, gfv, gfd, err := ms.createTrustRegistryEntries(ctx, msg, now)
	if err != nil {
		return nil, err
	}

	// Update trust deposit amount in the trust registry entry
	tr.Deposit = int64(trustDeposit)

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

	return &types.MsgArchiveTrustRegistryResponse{}, nil
}
