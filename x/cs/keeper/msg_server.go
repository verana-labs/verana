package keeper

import (
	"context"
	"errors"
	"fmt"
	"strconv"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/verana-labs/verana/x/cs/types"
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

func (ms msgServer) CreateCredentialSchema(goCtx context.Context, msg *types.MsgCreateCredentialSchema) (*types.MsgCreateCredentialSchemaResponse, error) {
	ctx := sdk.UnwrapSDKContext(goCtx)
	now := ctx.BlockTime()

	// [MOD-CS-MSG-1-2-1] [AUTHZ-CHECK] Verify operator authorization
	if ms.delegationKeeper != nil {
		if err := ms.delegationKeeper.CheckOperatorAuthorization(
			ctx,
			msg.Authority,
			msg.Operator,
			"/verana.cs.v1.MsgCreateCredentialSchema",
			now,
		); err != nil {
			return nil, fmt.Errorf("authorization check failed: %w", err)
		}
	}

	// Generate next ID
	nextID, err := ms.GetNextID(ctx, "cs")
	if err != nil {
		return nil, fmt.Errorf("failed to generate schema ID: %w", err)
	}

	// [MOD-CS-MSG-1-2-1] Basic checks
	if err := ms.validateCreateCredentialSchemaParams(ctx, msg); err != nil {
		return nil, err
	}

	// [MOD-CS-MSG-1-3] Execution
	if err := ms.executeCreateCredentialSchema(ctx, nextID, msg); err != nil {
		return nil, err
	}

	ctx.EventManager().EmitEvents(sdk.Events{
		sdk.NewEvent(
			types.EventTypeCreateCredentialSchema,
			sdk.NewAttribute(types.AttributeKeyId, strconv.FormatUint(nextID, 10)),
			sdk.NewAttribute(types.AttributeKeyTrId, strconv.FormatUint(msg.TrId, 10)),
			sdk.NewAttribute(types.AttributeKeyAuthority, msg.Authority),
			sdk.NewAttribute(types.AttributeKeyOperator, msg.Operator),
			sdk.NewAttribute(types.AttributeKeyTimestamp, ctx.BlockTime().String()),
		),
	})

	return &types.MsgCreateCredentialSchemaResponse{
		Id: nextID,
	}, nil
}

func (ms msgServer) UpdateCredentialSchema(goCtx context.Context, msg *types.MsgUpdateCredentialSchema) (*types.MsgUpdateCredentialSchemaResponse, error) {
	ctx := sdk.UnwrapSDKContext(goCtx)

	// Get credential schema
	cs, err := ms.CredentialSchema.Get(ctx, msg.Id)
	if err != nil {
		return nil, fmt.Errorf("credential schema not found: %w", err)
	}

	// Check trust registry controller
	tr, err := ms.trustRegistryKeeper.GetTrustRegistry(ctx, cs.TrId)
	if err != nil {
		return nil, fmt.Errorf("trust registry not found: %w", err)
	}
	if tr.Controller != msg.Creator {
		return nil, fmt.Errorf("creator is not the controller of the trust registry")
	}

	// Validate validity periods against params (only for fields that are set)
	params := ms.GetParams(ctx)
	if err := ValidateValidityPeriods(params, msg); err != nil {
		return nil, fmt.Errorf("invalid validity period: %w", err)
	}

	// [MOD-CS-MSG-2-3] Update mutable fields
	cs.IssuerGrantorValidationValidityPeriod = msg.GetIssuerGrantorValidationValidityPeriod().GetValue()
	cs.VerifierGrantorValidationValidityPeriod = msg.GetVerifierGrantorValidationValidityPeriod().GetValue()
	cs.IssuerValidationValidityPeriod = msg.GetIssuerValidationValidityPeriod().GetValue()
	cs.VerifierValidationValidityPeriod = msg.GetVerifierValidationValidityPeriod().GetValue()
	cs.HolderValidationValidityPeriod = msg.GetHolderValidationValidityPeriod().GetValue()
	cs.Modified = ctx.BlockTime()

	if err := ms.CredentialSchema.Set(ctx, cs.Id, cs); err != nil {
		return nil, fmt.Errorf("failed to update credential schema: %w", err)
	}

	ctx.EventManager().EmitEvents(sdk.Events{
		sdk.NewEvent(
			types.EventTypeUpdateCredentialSchema,
			sdk.NewAttribute(types.AttributeKeyId, strconv.FormatUint(msg.Id, 10)),
			sdk.NewAttribute(types.AttributeKeyTrId, strconv.FormatUint(cs.TrId, 10)),
			sdk.NewAttribute(types.AttributeKeyCreator, msg.Creator),
			sdk.NewAttribute(types.AttributeKeyIssuerGrantorValidationValidityPeriod, strconv.FormatUint(uint64(msg.GetIssuerGrantorValidationValidityPeriod().GetValue()), 10)),
			sdk.NewAttribute(types.AttributeKeyVerifierGrantorValidationValidityPeriod, strconv.FormatUint(uint64(msg.GetVerifierGrantorValidationValidityPeriod().GetValue()), 10)),
			sdk.NewAttribute(types.AttributeKeyIssuerValidationValidityPeriod, strconv.FormatUint(uint64(msg.GetIssuerValidationValidityPeriod().GetValue()), 10)),
			sdk.NewAttribute(types.AttributeKeyVerifierValidationValidityPeriod, strconv.FormatUint(uint64(msg.GetVerifierValidationValidityPeriod().GetValue()), 10)),
			sdk.NewAttribute(types.AttributeKeyHolderValidationValidityPeriod, strconv.FormatUint(uint64(msg.GetHolderValidationValidityPeriod().GetValue()), 10)),
			sdk.NewAttribute(types.AttributeKeyTimestamp, ctx.BlockTime().String()),
		),
	})

	return &types.MsgUpdateCredentialSchemaResponse{}, nil
}

// ValidateValidityPeriods checks if all validity periods are within allowed ranges
func ValidateValidityPeriods(
	params types.Params,
	msg *types.MsgUpdateCredentialSchema,
) error {
	val := msg.GetIssuerGrantorValidationValidityPeriod().GetValue()
	if val > 0 && val > params.CredentialSchemaIssuerGrantorValidationValidityPeriodMaxDays {
		return errors.New("issuer grantor validation validity period exceeds maximum allowed days")
	}

	val = msg.GetVerifierGrantorValidationValidityPeriod().GetValue()
	if val > 0 && val > params.CredentialSchemaVerifierGrantorValidationValidityPeriodMaxDays {
		return errors.New("verifier grantor validation validity period exceeds maximum allowed days")
	}

	val = msg.GetIssuerValidationValidityPeriod().GetValue()
	if val > 0 && val > params.CredentialSchemaIssuerValidationValidityPeriodMaxDays {
		return errors.New("issuer validation validity period exceeds maximum allowed days")
	}

	val = msg.GetVerifierValidationValidityPeriod().GetValue()
	if val > 0 && val > params.CredentialSchemaVerifierValidationValidityPeriodMaxDays {
		return errors.New("verifier validation validity period exceeds maximum allowed days")
	}

	val = msg.GetHolderValidationValidityPeriod().GetValue()
	if val > 0 && val > params.CredentialSchemaHolderValidationValidityPeriodMaxDays {
		return errors.New("holder validation validity period exceeds maximum allowed days")
	}

	return nil
}

func (ms msgServer) ArchiveCredentialSchema(goCtx context.Context, msg *types.MsgArchiveCredentialSchema) (*types.MsgArchiveCredentialSchemaResponse, error) {
	ctx := sdk.UnwrapSDKContext(goCtx)

	// Get credential schema
	cs, err := ms.CredentialSchema.Get(ctx, msg.Id)
	if err != nil {
		return nil, fmt.Errorf("credential schema not found: %w", err)
	}

	// Check trust registry controller
	tr, err := ms.trustRegistryKeeper.GetTrustRegistry(ctx, cs.TrId)
	if err != nil {
		return nil, fmt.Errorf("trust registry not found: %w", err)
	}
	if tr.Controller != msg.Creator {
		return nil, fmt.Errorf("only trust registry controller can archive credential schema")
	}

	// Check archive state
	if msg.Archive {
		if cs.Archived != nil {
			return nil, fmt.Errorf("credential schema is already archived")
		}
	} else {
		if cs.Archived == nil {
			return nil, fmt.Errorf("credential schema is not archived")
		}
	}

	// Update archive state
	now := ctx.BlockTime()
	if msg.Archive {
		cs.Archived = &now
	} else {
		cs.Archived = nil
	}
	cs.Modified = now

	// Save updated credential schema
	if err := ms.CredentialSchema.Set(ctx, cs.Id, cs); err != nil {
		return nil, fmt.Errorf("failed to update credential schema: %w", err)
	}

	// Determine archive status string
	archiveStatus := "archived"
	if !msg.Archive {
		archiveStatus = "unarchived"
	}

	ctx.EventManager().EmitEvents(sdk.Events{
		sdk.NewEvent(
			types.EventTypeArchiveCredentialSchema,
			sdk.NewAttribute(types.AttributeKeyId, strconv.FormatUint(msg.Id, 10)),
			sdk.NewAttribute(types.AttributeKeyTrId, strconv.FormatUint(cs.TrId, 10)),
			sdk.NewAttribute(types.AttributeKeyCreator, msg.Creator),
			sdk.NewAttribute(types.AttributeKeyArchiveStatus, archiveStatus),
			sdk.NewAttribute(types.AttributeKeyTimestamp, now.String()),
		),
	})

	return &types.MsgArchiveCredentialSchemaResponse{}, nil
}
