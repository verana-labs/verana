package keeper

import (
	"fmt"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/verana-labs/verana/x/cs/types"
)

func (ms msgServer) validateCreateCredentialSchemaParams(ctx sdk.Context, msg *types.MsgCreateCredentialSchema) error {
	params := ms.GetParams(ctx)

	// Validate trust registry ownership
	tr, err := ms.trustRegistryKeeper.GetTrustRegistry(ctx, msg.TrId)
	if err != nil {
		return fmt.Errorf("trust registry not found: %w", err)
	}
	if tr.Controller != msg.Creator {
		return fmt.Errorf("creator is not the controller of the trust registry")
	}

	// Check schema size
	if uint64(len(msg.JsonSchema)) > params.CredentialSchemaSchemaMaxSize {
		return fmt.Errorf("schema size exceeds maximum allowed size of %d bytes", params.CredentialSchemaSchemaMaxSize)
	}

	// Validate validity periods against params
	if err := validateValidityPeriodsWithParams(msg, params); err != nil {
		return fmt.Errorf("invalid validity period: %w", err)
	}

	return nil
}

func validateValidityPeriodsWithParams(msg *types.MsgCreateCredentialSchema, params types.Params) error {
	// 0 is valid (never expires), only check if > 0 and exceeds max
	// For Create, fields are required (should not be nil), but we check anyway
	if msg.GetIssuerGrantorValidationValidityPeriod() != nil {
		val := msg.GetIssuerGrantorValidationValidityPeriod().GetValue()
		if val > 0 && val > params.CredentialSchemaIssuerGrantorValidationValidityPeriodMaxDays {
			return fmt.Errorf("issuer grantor validation validity period exceeds maximum of %d days",
				params.CredentialSchemaIssuerGrantorValidationValidityPeriodMaxDays)
		}
	}

	if msg.GetVerifierGrantorValidationValidityPeriod() != nil {
		val := msg.GetVerifierGrantorValidationValidityPeriod().GetValue()
		if val > 0 && val > params.CredentialSchemaVerifierGrantorValidationValidityPeriodMaxDays {
			return fmt.Errorf("verifier grantor validation validity period exceeds maximum of %d days",
				params.CredentialSchemaVerifierGrantorValidationValidityPeriodMaxDays)
		}
	}

	if msg.GetIssuerValidationValidityPeriod() != nil {
		val := msg.GetIssuerValidationValidityPeriod().GetValue()
		if val > 0 && val > params.CredentialSchemaIssuerValidationValidityPeriodMaxDays {
			return fmt.Errorf("issuer validation validity period exceeds maximum of %d days",
				params.CredentialSchemaIssuerValidationValidityPeriodMaxDays)
		}
	}

	if msg.GetVerifierValidationValidityPeriod() != nil {
		val := msg.GetVerifierValidationValidityPeriod().GetValue()
		if val > 0 && val > params.CredentialSchemaVerifierValidationValidityPeriodMaxDays {
			return fmt.Errorf("verifier validation validity period exceeds maximum of %d days",
				params.CredentialSchemaVerifierValidationValidityPeriodMaxDays)
		}
	}

	if msg.GetHolderValidationValidityPeriod() != nil {
		val := msg.GetHolderValidationValidityPeriod().GetValue()
		if val > 0 && val > params.CredentialSchemaHolderValidationValidityPeriodMaxDays {
			return fmt.Errorf("holder validation validity period exceeds maximum of %d days",
				params.CredentialSchemaHolderValidationValidityPeriodMaxDays)
		}
	}

	return nil
}

func (ms msgServer) executeCreateCredentialSchema(ctx sdk.Context, schemaID uint64, msg *types.MsgCreateCredentialSchema) error {
	// Get params using the getter method
	params := ms.GetParams(ctx)

	// Calculate trust deposit amount
	trustDepositAmount := params.CredentialSchemaTrustDeposit * ms.trustRegistryKeeper.GetTrustUnitPrice(ctx)

	// Increase trust deposit
	if err := ms.trustDeposit.AdjustTrustDeposit(ctx, msg.Creator, int64(trustDepositAmount)); err != nil {
		return fmt.Errorf("failed to adjust trust deposit: %w", err)
	}

	// Inject canonical $id into the JSON schema
	processedJsonSchema, err := types.InjectCanonicalID(msg.JsonSchema, ctx.ChainID(), schemaID)
	if err != nil {
		return fmt.Errorf("failed to process JSON schema: %w", err)
	}

	// Create the credential schema
	// Extract values from OptionalUInt32 wrappers (0 is valid - means never expires)
	var issuerGrantor, verifierGrantor, issuer, verifier, holder uint32
	if msg.GetIssuerGrantorValidationValidityPeriod() != nil {
		issuerGrantor = msg.GetIssuerGrantorValidationValidityPeriod().GetValue()
	}
	if msg.GetVerifierGrantorValidationValidityPeriod() != nil {
		verifierGrantor = msg.GetVerifierGrantorValidationValidityPeriod().GetValue()
	}
	if msg.GetIssuerValidationValidityPeriod() != nil {
		issuer = msg.GetIssuerValidationValidityPeriod().GetValue()
	}
	if msg.GetVerifierValidationValidityPeriod() != nil {
		verifier = msg.GetVerifierValidationValidityPeriod().GetValue()
	}
	if msg.GetHolderValidationValidityPeriod() != nil {
		holder = msg.GetHolderValidationValidityPeriod().GetValue()
	}

	credentialSchema := types.CredentialSchema{
		Id:                                      schemaID, // Use the generated ID
		TrId:                                    msg.TrId,
		Created:                                 ctx.BlockTime(),
		Modified:                                ctx.BlockTime(),
		Deposit:                                 trustDepositAmount,
		JsonSchema:                              processedJsonSchema, // Now includes chain ID replacement
		IssuerGrantorValidationValidityPeriod:   issuerGrantor,
		VerifierGrantorValidationValidityPeriod: verifierGrantor,
		IssuerValidationValidityPeriod:          issuer,
		VerifierValidationValidityPeriod:        verifier,
		HolderValidationValidityPeriod:          holder,
		IssuerPermManagementMode:                types.CredentialSchemaPermManagementMode(msg.IssuerPermManagementMode),
		VerifierPermManagementMode:              types.CredentialSchemaPermManagementMode(msg.VerifierPermManagementMode),
	}

	// Persist the credential schema using keeper method
	if err := ms.SetCredentialSchema(ctx, credentialSchema); err != nil {
		return fmt.Errorf("failed to persist credential schema: %w", err)
	}

	// Emit event
	ctx.EventManager().EmitEvent(
		sdk.NewEvent(
			types.EventTypeCreateCredentialSchema,
			sdk.NewAttribute(types.AttributeKeyId, fmt.Sprintf("%d", schemaID)),
			sdk.NewAttribute(types.AttributeKeyTrId, fmt.Sprintf("%d", msg.TrId)),
			sdk.NewAttribute(types.AttributeKeyCreator, msg.Creator),
			sdk.NewAttribute(types.AttributeKeyDeposit, fmt.Sprintf("%d", trustDepositAmount)),
		),
	)

	return nil
}
