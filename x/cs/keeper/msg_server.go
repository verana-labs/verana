package keeper

import (
	"context"
	"errors"
	"fmt"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/verana-labs/verana-blockchain/x/cs/types"
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

	// Validate validity periods against params
	params := ms.GetParams(ctx)
	if err := ValidateValidityPeriods(params, msg); err != nil {
		return nil, fmt.Errorf("invalid validity period: %w", err)
	}

	// [MOD-CS-MSG-2-3] Update mutable fields only
	cs.IssuerGrantorValidationValidityPeriod = msg.IssuerGrantorValidationValidityPeriod
	cs.VerifierGrantorValidationValidityPeriod = msg.VerifierGrantorValidationValidityPeriod
	cs.IssuerValidationValidityPeriod = msg.IssuerValidationValidityPeriod
	cs.VerifierValidationValidityPeriod = msg.VerifierValidationValidityPeriod
	cs.HolderValidationValidityPeriod = msg.HolderValidationValidityPeriod
	cs.Modified = ctx.BlockTime()

	if err := ms.CredentialSchema.Set(ctx, cs.Id, cs); err != nil {
		return nil, fmt.Errorf("failed to update credential schema: %w", err)
	}

	return &types.MsgUpdateCredentialSchemaResponse{}, nil
}

// ValidateValidityPeriods checks if all validity periods are within allowed ranges
func ValidateValidityPeriods(
	params types.Params,
	msg *types.MsgUpdateCredentialSchema,
) error {
	if msg.IssuerGrantorValidationValidityPeriod > params.CredentialSchemaIssuerGrantorValidationValidityPeriodMaxDays {
		return errors.New("issuer grantor validation validity period exceeds maximum allowed days")
	}
	if msg.VerifierGrantorValidationValidityPeriod > params.CredentialSchemaVerifierGrantorValidationValidityPeriodMaxDays {
		return errors.New("verifier grantor validation validity period exceeds maximum allowed days")
	}
	if msg.IssuerValidationValidityPeriod > params.CredentialSchemaIssuerValidationValidityPeriodMaxDays {
		return errors.New("issuer validation validity period exceeds maximum allowed days")
	}
	if msg.VerifierValidationValidityPeriod > params.CredentialSchemaVerifierValidationValidityPeriodMaxDays {
		return errors.New("verifier validation validity period exceeds maximum allowed days")
	}
	if msg.HolderValidationValidityPeriod > params.CredentialSchemaHolderValidationValidityPeriodMaxDays {
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

	return &types.MsgArchiveCredentialSchemaResponse{}, nil
}
