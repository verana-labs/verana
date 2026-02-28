package keeper

import (
	"fmt"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/verana-labs/verana/x/cs/types"
)

func (ms msgServer) validateCreateCredentialSchemaParams(ctx sdk.Context, msg *types.MsgCreateCredentialSchema) error {
	params := ms.GetParams(ctx)

	// Validate trust registry ownership - authority must be the controller
	tr, err := ms.trustRegistryKeeper.GetTrustRegistry(ctx, msg.TrId)
	if err != nil {
		return fmt.Errorf("trust registry not found: %w", err)
	}
	if tr.Controller != msg.Authority {
		return fmt.Errorf("authority is not the controller of the trust registry")
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
	// [MOD-CS-MSG-1-2-1] All validity period fields are mandatory
	// Must be between 0 (never expire) and max_days

	// Check mandatory fields are present
	if msg.GetIssuerGrantorValidationValidityPeriod() == nil {
		return fmt.Errorf("issuer_grantor_validation_validity_period is mandatory")
	}
	if msg.GetVerifierGrantorValidationValidityPeriod() == nil {
		return fmt.Errorf("verifier_grantor_validation_validity_period is mandatory")
	}
	if msg.GetIssuerValidationValidityPeriod() == nil {
		return fmt.Errorf("issuer_validation_validity_period is mandatory")
	}
	if msg.GetVerifierValidationValidityPeriod() == nil {
		return fmt.Errorf("verifier_validation_validity_period is mandatory")
	}
	if msg.GetHolderValidationValidityPeriod() == nil {
		return fmt.Errorf("holder_validation_validity_period is mandatory")
	}

	// Validate ranges: must be between 0 (never expire) and max_days
	val := msg.GetIssuerGrantorValidationValidityPeriod().GetValue()
	if val > 0 && val > params.CredentialSchemaIssuerGrantorValidationValidityPeriodMaxDays {
		return fmt.Errorf("issuer grantor validation validity period exceeds maximum of %d days",
			params.CredentialSchemaIssuerGrantorValidationValidityPeriodMaxDays)
	}

	val = msg.GetVerifierGrantorValidationValidityPeriod().GetValue()
	if val > 0 && val > params.CredentialSchemaVerifierGrantorValidationValidityPeriodMaxDays {
		return fmt.Errorf("verifier grantor validation validity period exceeds maximum of %d days",
			params.CredentialSchemaVerifierGrantorValidationValidityPeriodMaxDays)
	}

	val = msg.GetIssuerValidationValidityPeriod().GetValue()
	if val > 0 && val > params.CredentialSchemaIssuerValidationValidityPeriodMaxDays {
		return fmt.Errorf("issuer validation validity period exceeds maximum of %d days",
			params.CredentialSchemaIssuerValidationValidityPeriodMaxDays)
	}

	val = msg.GetVerifierValidationValidityPeriod().GetValue()
	if val > 0 && val > params.CredentialSchemaVerifierValidationValidityPeriodMaxDays {
		return fmt.Errorf("verifier validation validity period exceeds maximum of %d days",
			params.CredentialSchemaVerifierValidationValidityPeriodMaxDays)
	}

	val = msg.GetHolderValidationValidityPeriod().GetValue()
	if val > 0 && val > params.CredentialSchemaHolderValidationValidityPeriodMaxDays {
		return fmt.Errorf("holder validation validity period exceeds maximum of %d days",
			params.CredentialSchemaHolderValidationValidityPeriodMaxDays)
	}

	return nil
}

func (ms msgServer) executeCreateCredentialSchema(ctx sdk.Context, schemaID uint64, msg *types.MsgCreateCredentialSchema) error {
	// Inject canonical $id into the JSON schema
	processedJsonSchema, err := types.InjectCanonicalID(msg.JsonSchema, ctx.ChainID(), schemaID)
	if err != nil {
		return fmt.Errorf("failed to process JSON schema: %w", err)
	}

	// [MOD-CS-MSG-1-3] Create the credential schema
	credentialSchema := types.CredentialSchema{
		Id:                                      schemaID,
		TrId:                                    msg.TrId,
		Created:                                 ctx.BlockTime(),
		Modified:                                ctx.BlockTime(),
		JsonSchema:                              processedJsonSchema,
		IssuerGrantorValidationValidityPeriod:   msg.GetIssuerGrantorValidationValidityPeriod().GetValue(),
		VerifierGrantorValidationValidityPeriod: msg.GetVerifierGrantorValidationValidityPeriod().GetValue(),
		IssuerValidationValidityPeriod:          msg.GetIssuerValidationValidityPeriod().GetValue(),
		VerifierValidationValidityPeriod:        msg.GetVerifierValidationValidityPeriod().GetValue(),
		HolderValidationValidityPeriod:          msg.GetHolderValidationValidityPeriod().GetValue(),
		IssuerPermManagementMode:                types.CredentialSchemaPermManagementMode(msg.IssuerPermManagementMode),
		VerifierPermManagementMode:              types.CredentialSchemaPermManagementMode(msg.VerifierPermManagementMode),
		PricingAssetType:                        types.PricingAssetType(msg.PricingAssetType),
		PricingAsset:                            msg.PricingAsset,
		DigestAlgorithm:                         msg.DigestAlgorithm,
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
			sdk.NewAttribute(types.AttributeKeyAuthority, msg.Authority),
			sdk.NewAttribute(types.AttributeKeyOperator, msg.Operator),
		),
	)

	return nil
}
