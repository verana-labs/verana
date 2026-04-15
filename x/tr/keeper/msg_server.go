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
	now := ctx.BlockTime()

	// [MOD-TR-MSG-1-2-1] [AUTHZ-CHECK] Verify operator authorization
	if ms.delegationKeeper == nil {
		return nil, fmt.Errorf("delegation keeper is required for operator authorization")
	}
	if err := ms.delegationKeeper.CheckOperatorAuthorization(
		ctx,
		msg.Corporation,
		msg.Operator,
		"/verana.tr.v1.MsgCreateTrustRegistry",
		now,
	); err != nil {
		return nil, fmt.Errorf("authorization check failed: %w", err)
	}

	// [MOD-TR-MSG-1-3] Create New Trust Registry execution
	// Spec v4: MsgCreateTrustRegistry no longer bundles an initial governance
	// framework document — callers must invoke MsgAddGovernanceFrameworkDocument
	// separately if they need one.

	tr, gfv, err := ms.createTrustRegistryEntries(ctx, msg, now)
	if err != nil {
		return nil, err
	}

	if err := ms.persistEntries(ctx, tr, gfv); err != nil {
		return nil, err
	}

	ctx.EventManager().EmitEvents(sdk.Events{
		sdk.NewEvent(
			types.EventTypeCreateTrustRegistry,
			sdk.NewAttribute(types.AttributeKeyTrustRegistryID, strconv.FormatUint(tr.Id, 10)),
			sdk.NewAttribute(types.AttributeKeyDID, tr.Did),
			sdk.NewAttribute(types.AttributeKeyController, tr.Corporation),
			sdk.NewAttribute(types.AttributeKeyAka, tr.Aka),
			sdk.NewAttribute(types.AttributeKeyLanguage, tr.Language),
			sdk.NewAttribute(types.AttributeKeyTimestamp, now.String()),
		),
	})

	return &types.MsgCreateTrustRegistryResponse{}, nil
}

func (ms msgServer) AddGovernanceFrameworkDocument(goCtx context.Context, msg *types.MsgAddGovernanceFrameworkDocument) (*types.MsgAddGovernanceFrameworkDocumentResponse, error) {
	ctx := sdk.UnwrapSDKContext(goCtx)
	now := ctx.BlockTime()

	// [MOD-TR-MSG-2-2-1] [AUTHZ-CHECK] Verify operator authorization
	if ms.delegationKeeper == nil {
		return nil, fmt.Errorf("delegation keeper is required for operator authorization")
	}
	if err := ms.delegationKeeper.CheckOperatorAuthorization(
		ctx,
		msg.Corporation,
		msg.Operator,
		"/verana.tr.v1.MsgAddGovernanceFrameworkDocument",
		now,
	); err != nil {
		return nil, fmt.Errorf("authorization check failed: %w", err)
	}

	if err := ms.validateAddGovernanceFrameworkDocumentParams(ctx, msg); err != nil {
		return nil, err
	}

	if err := ms.executeAddGovernanceFrameworkDocument(ctx, msg); err != nil {
		return nil, err
	}

	ctx.EventManager().EmitEvents(sdk.Events{
		sdk.NewEvent(
			types.EventTypeAddGovernanceFrameworkDocument,
			sdk.NewAttribute(types.AttributeKeyGFVersionID, strconv.FormatUint(msg.TrId, 10)),
			sdk.NewAttribute(types.AttributeKeyVersion, strconv.FormatInt(int64(msg.Version), 10)),
			sdk.NewAttribute(types.AttributeKeyLanguage, msg.Language),
			sdk.NewAttribute(types.AttributeKeyDocURL, msg.Url),
			sdk.NewAttribute(types.AttributeKeyDigestSri, msg.DigestSri),
			sdk.NewAttribute(types.AttributeKeyController, msg.Corporation),
			sdk.NewAttribute(types.AttributeKeyTimestamp, ctx.BlockTime().String()),
		),
	})

	return &types.MsgAddGovernanceFrameworkDocumentResponse{}, nil
}

func (ms msgServer) IncreaseActiveGovernanceFrameworkVersion(goCtx context.Context, msg *types.MsgIncreaseActiveGovernanceFrameworkVersion) (*types.MsgIncreaseActiveGovernanceFrameworkVersionResponse, error) {
	ctx := sdk.UnwrapSDKContext(goCtx)
	now := ctx.BlockTime()

	// [MOD-TR-MSG-3-2-1] [AUTHZ-CHECK] Verify operator authorization
	if ms.delegationKeeper == nil {
		return nil, fmt.Errorf("delegation keeper is required for operator authorization")
	}
	if err := ms.delegationKeeper.CheckOperatorAuthorization(
		ctx,
		msg.Corporation,
		msg.Operator,
		"/verana.tr.v1.MsgIncreaseActiveGovernanceFrameworkVersion",
		now,
	); err != nil {
		return nil, fmt.Errorf("authorization check failed: %w", err)
	}

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
			sdk.NewAttribute(types.AttributeKeyTrustRegistryID, strconv.FormatUint(msg.TrId, 10)),
			sdk.NewAttribute(types.AttributeKeyController, msg.Corporation),
			sdk.NewAttribute(types.AttributeKeyTimestamp, now.String()),
		),
	})

	return &types.MsgIncreaseActiveGovernanceFrameworkVersionResponse{}, nil
}

func (ms msgServer) UpdateTrustRegistry(goCtx context.Context, msg *types.MsgUpdateTrustRegistry) (*types.MsgUpdateTrustRegistryResponse, error) {
	ctx := sdk.UnwrapSDKContext(goCtx)
	now := ctx.BlockTime()

	// [MOD-TR-MSG-4-2-1] [AUTHZ-CHECK] Verify operator authorization
	if ms.delegationKeeper == nil {
		return nil, fmt.Errorf("delegation keeper is required for operator authorization")
	}
	if err := ms.delegationKeeper.CheckOperatorAuthorization(
		ctx,
		msg.Corporation,
		msg.Operator,
		"/verana.tr.v1.MsgUpdateTrustRegistry",
		now,
	); err != nil {
		return nil, fmt.Errorf("authorization check failed: %w", err)
	}

	// Get trust registry
	tr, err := ms.TrustRegistry.Get(ctx, msg.TrId)
	if err != nil {
		return nil, fmt.Errorf("trust registry not found: %w", err)
	}

	// Check corporation - corporation must match the trust registry corporation
	if tr.Corporation != msg.Corporation {
		return nil, fmt.Errorf("only trust registry corporation can update trust registry")
	}

	// Spec v4: MsgUpdateTrustRegistry only updates aka/language; DID is immutable.
	tr.Aka = msg.Aka
	if msg.Language != "" {
		tr.Language = msg.Language
	}
	tr.Modified = now

	// Save updated trust registry
	if err := ms.TrustRegistry.Set(ctx, tr.Id, tr); err != nil {
		return nil, fmt.Errorf("failed to update trust registry: %w", err)
	}

	ctx.EventManager().EmitEvents(sdk.Events{
		sdk.NewEvent(
			types.EventTypeUpdateTrustRegistry,
			sdk.NewAttribute(types.AttributeKeyTrustRegistryID, strconv.FormatUint(msg.TrId, 10)),
			sdk.NewAttribute(types.AttributeKeyController, msg.Corporation),
			sdk.NewAttribute(types.AttributeKeyAka, msg.Aka),
			sdk.NewAttribute(types.AttributeKeyTimestamp, now.String()),
		),
	})

	return &types.MsgUpdateTrustRegistryResponse{}, nil
}

func (ms msgServer) ArchiveTrustRegistry(goCtx context.Context, msg *types.MsgArchiveTrustRegistry) (*types.MsgArchiveTrustRegistryResponse, error) {
	ctx := sdk.UnwrapSDKContext(goCtx)
	now := ctx.BlockTime()

	// [MOD-TR-MSG-5-2-1] [AUTHZ-CHECK] Verify operator authorization
	if ms.delegationKeeper == nil {
		return nil, fmt.Errorf("delegation keeper is required for operator authorization")
	}
	if err := ms.delegationKeeper.CheckOperatorAuthorization(
		ctx,
		msg.Corporation,
		msg.Operator,
		"/verana.tr.v1.MsgArchiveTrustRegistry",
		now,
	); err != nil {
		return nil, fmt.Errorf("authorization check failed: %w", err)
	}

	// Get trust registry
	tr, err := ms.TrustRegistry.Get(ctx, msg.TrId)
	if err != nil {
		return nil, fmt.Errorf("trust registry not found: %w", err)
	}

	// Check corporation
	if tr.Corporation != msg.Corporation {
		return nil, fmt.Errorf("only trust registry corporation can archive trust registry")
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
			sdk.NewAttribute(types.AttributeKeyTrustRegistryID, strconv.FormatUint(msg.TrId, 10)),
			sdk.NewAttribute(types.AttributeKeyController, msg.Corporation),
			sdk.NewAttribute(types.AttributeKeyArchiveStatus, archiveStatus),
			sdk.NewAttribute(types.AttributeKeyTimestamp, now.String()),
		),
	})

	return &types.MsgArchiveTrustRegistryResponse{}, nil
}
