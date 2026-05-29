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

	// [MOD-CS-MSG-1-2-1] [AUTHZ-CHECK] Verify operator authorization
	if ms.delegationKeeper == nil {
		return nil, fmt.Errorf("delegation keeper is required for operator authorization")
	}
	if err := ms.delegationKeeper.CheckOperatorAuthorization(
		ctx,
		msg.Corporation,
		msg.Operator,
		"/verana.cs.v1.MsgCreateCredentialSchema",
		ctx.BlockTime(),
	); err != nil {
		return nil, fmt.Errorf("authorization check failed: %w", err)
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

	return &types.MsgCreateCredentialSchemaResponse{
		Id: nextID,
	}, nil
}

func (ms msgServer) UpdateCredentialSchema(goCtx context.Context, msg *types.MsgUpdateCredentialSchema) (*types.MsgUpdateCredentialSchemaResponse, error) {
	ctx := sdk.UnwrapSDKContext(goCtx)
	now := ctx.BlockTime()

	// [MOD-CS-MSG-2-2-1] [AUTHZ-CHECK] Verify operator authorization
	if ms.delegationKeeper == nil {
		return nil, fmt.Errorf("delegation keeper is required for operator authorization")
	}
	if err := ms.delegationKeeper.CheckOperatorAuthorization(
		ctx,
		msg.Corporation,
		msg.Operator,
		"/verana.cs.v1.MsgUpdateCredentialSchema",
		now,
	); err != nil {
		return nil, fmt.Errorf("authorization check failed: %w", err)
	}

	// Get credential schema
	cs, err := ms.CredentialSchema.Get(ctx, msg.Id)
	if err != nil {
		return nil, fmt.Errorf("credential schema not found: %w", err)
	}

	// [MOD-CS-MSG-2-2-1] Archived schemas cannot be updated
	if cs.Archived != nil {
		return nil, fmt.Errorf("cannot update an archived credential schema")
	}

	// [MOD-CS-MSG-2-2-1] Check ecosystem authority
	if err := ms.checkSchemaOwnership(ctx, cs, msg.Corporation); err != nil {
		return nil, err
	}

	// Validate validity periods against params
	params := ms.GetParams(ctx)
	if err := ValidateValidityPeriods(params, msg); err != nil {
		return nil, fmt.Errorf("invalid validity period: %w", err)
	}

	// [MOD-CS-MSG-2-3] Update mutable fields (only overwrite if the field is explicitly provided)
	if msg.GetIssuerGrantorValidationValidityPeriod() != nil {
		cs.IssuerGrantorValidationValidityPeriod = msg.GetIssuerGrantorValidationValidityPeriod().GetValue()
	}
	if msg.GetVerifierGrantorValidationValidityPeriod() != nil {
		cs.VerifierGrantorValidationValidityPeriod = msg.GetVerifierGrantorValidationValidityPeriod().GetValue()
	}
	if msg.GetIssuerValidationValidityPeriod() != nil {
		cs.IssuerValidationValidityPeriod = msg.GetIssuerValidationValidityPeriod().GetValue()
	}
	if msg.GetVerifierValidationValidityPeriod() != nil {
		cs.VerifierValidationValidityPeriod = msg.GetVerifierValidationValidityPeriod().GetValue()
	}
	if msg.GetHolderValidationValidityPeriod() != nil {
		cs.HolderValidationValidityPeriod = msg.GetHolderValidationValidityPeriod().GetValue()
	}
	cs.Modified = now

	if err := ms.CredentialSchema.Set(ctx, cs.Id, cs); err != nil {
		return nil, fmt.Errorf("failed to update credential schema: %w", err)
	}

	ctx.EventManager().EmitEvents(sdk.Events{
		sdk.NewEvent(
			types.EventTypeUpdateCredentialSchema,
			sdk.NewAttribute(types.AttributeKeyId, strconv.FormatUint(msg.Id, 10)),
			sdk.NewAttribute(types.AttributeKeyEcosystemId, strconv.FormatUint(cs.EcosystemId, 10)),
			sdk.NewAttribute(types.AttributeKeyCorporation, msg.Corporation),
			sdk.NewAttribute(types.AttributeKeyOperator, msg.Operator),
			sdk.NewAttribute(types.AttributeKeyIssuerGrantorValidationValidityPeriod, strconv.FormatUint(uint64(cs.IssuerGrantorValidationValidityPeriod), 10)),
			sdk.NewAttribute(types.AttributeKeyVerifierGrantorValidationValidityPeriod, strconv.FormatUint(uint64(cs.VerifierGrantorValidationValidityPeriod), 10)),
			sdk.NewAttribute(types.AttributeKeyIssuerValidationValidityPeriod, strconv.FormatUint(uint64(cs.IssuerValidationValidityPeriod), 10)),
			sdk.NewAttribute(types.AttributeKeyVerifierValidationValidityPeriod, strconv.FormatUint(uint64(cs.VerifierValidationValidityPeriod), 10)),
			sdk.NewAttribute(types.AttributeKeyHolderValidationValidityPeriod, strconv.FormatUint(uint64(cs.HolderValidationValidityPeriod), 10)),
			sdk.NewAttribute(types.AttributeKeyTimestamp, now.String()),
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
	now := ctx.BlockTime()

	// [MOD-CS-MSG-3-2-1] [AUTHZ-CHECK] Verify operator authorization
	if ms.delegationKeeper == nil {
		return nil, fmt.Errorf("delegation keeper is required for operator authorization")
	}
	if err := ms.delegationKeeper.CheckOperatorAuthorization(
		ctx,
		msg.Corporation,
		msg.Operator,
		"/verana.cs.v1.MsgArchiveCredentialSchema",
		now,
	); err != nil {
		return nil, fmt.Errorf("authorization check failed: %w", err)
	}

	// Get credential schema
	cs, err := ms.CredentialSchema.Get(ctx, msg.Id)
	if err != nil {
		return nil, fmt.Errorf("credential schema not found: %w", err)
	}

	// [MOD-CS-MSG-3-2-1] Check ecosystem authority
	if err := ms.checkSchemaOwnership(ctx, cs, msg.Corporation); err != nil {
		return nil, err
	}

	// [MOD-CS-MSG-3] Spec v4 draft 13: archive is a bidirectional toggle.
	// MOD-CS-MSG-3-3: if archive is false, set cs.archived to null.
	archiveStatus := "archived"
	if msg.Archive {
		if cs.Archived != nil {
			return nil, fmt.Errorf("credential schema is already archived")
		}
		cs.Archived = &now
	} else {
		if cs.Archived == nil {
			return nil, fmt.Errorf("credential schema is not archived")
		}
		cs.Archived = nil
		archiveStatus = "unarchived"
	}

	// [MOD-CS-MSG-3-3] Update modified timestamp
	cs.Modified = now

	if err := ms.CredentialSchema.Set(ctx, cs.Id, cs); err != nil {
		return nil, fmt.Errorf("failed to update credential schema: %w", err)
	}

	ctx.EventManager().EmitEvents(sdk.Events{
		sdk.NewEvent(
			types.EventTypeArchiveCredentialSchema,
			sdk.NewAttribute(types.AttributeKeyId, strconv.FormatUint(msg.Id, 10)),
			sdk.NewAttribute(types.AttributeKeyEcosystemId, strconv.FormatUint(cs.EcosystemId, 10)),
			sdk.NewAttribute(types.AttributeKeyCorporation, msg.Corporation),
			sdk.NewAttribute(types.AttributeKeyOperator, msg.Operator),
			sdk.NewAttribute(types.AttributeKeyArchiveStatus, archiveStatus),
			sdk.NewAttribute(types.AttributeKeyTimestamp, now.String()),
		),
	})

	return &types.MsgArchiveCredentialSchemaResponse{}, nil
}
