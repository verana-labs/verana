package keeper

import (
	"fmt"

	"cosmossdk.io/math"
	sdk "github.com/cosmos/cosmos-sdk/types"
	credentialschematypes "github.com/verana-labs/verana/x/cs/types"
	"github.com/verana-labs/verana/x/perm/types"
)

func (ms msgServer) validatePermissionChecks(ctx sdk.Context, msg *types.MsgStartPermissionVP) (types.Permission, error) {
	// Load validator perm
	validatorPerm, err := ms.Keeper.GetPermissionByID(ctx, msg.ValidatorPermId)
	if err != nil {
		return types.Permission{}, fmt.Errorf("validator perm not found: %w", err)
	}

	// [MOD-PERM-MSG-1-2-2] Check if validator perm is valid AND country compatibility
	// Spec: "It MUST be a valid permission AND (validator_perm.country MUST be equal to country, or validator_perm.country MUST be null)"
	// Spec: "When starting a Permission VP, if parent (validator) is INACTIVE (not valid) then MUST abort."
	// IsValidPermission already checks both: validity (ACTIVE state, time, revoked, slashed, repaid) AND country compatibility
	if err := IsValidPermission(validatorPerm, msg.Country, ctx.BlockTime()); err != nil {
		return types.Permission{}, fmt.Errorf("validator perm is not valid (must be ACTIVE): %w", err)
	}

	// Load credential schema
	cs, err := ms.credentialSchemaKeeper.GetCredentialSchemaById(ctx, validatorPerm.SchemaId)
	if err != nil {
		return types.Permission{}, fmt.Errorf("credential schema not found: %w", err)
	}

	// Validate perm type combinations
	if err := validatePermissionTypeCombination(types.PermissionType(msg.Type), validatorPerm.Type, cs); err != nil {
		return types.Permission{}, err
	}

	return validatorPerm, nil
}

func (ms msgServer) validateAndCalculateFees(ctx sdk.Context, _ string, validatorPerm types.Permission) (uint64, uint64, error) {
	// Get global variables
	trustUnitPrice := ms.trustRegistryKeeper.GetTrustUnitPrice(ctx)
	trustDepositRate := ms.trustDeposit.GetTrustDepositRate(ctx)

	validationFeesInDenom := validatorPerm.ValidationFees * trustUnitPrice
	validationTrustDepositInDenom := ms.Keeper.validationTrustDepositInDenomAmount(validationFeesInDenom, trustDepositRate)

	return validationFeesInDenom, validationTrustDepositInDenom, nil
}

func (k Keeper) validationTrustDepositInDenomAmount(validationFeesInDenom uint64, trustDepositRate math.LegacyDec) uint64 {
	validationFeesInDenomDec := math.LegacyNewDec(int64(validationFeesInDenom))
	validationTrustDepositInDenom := validationFeesInDenomDec.Mul(trustDepositRate)
	return validationTrustDepositInDenom.TruncateInt().Uint64()
}

func (ms msgServer) executeStartPermissionVP(ctx sdk.Context, msg *types.MsgStartPermissionVP, validatorPerm types.Permission, fees, deposit uint64) (uint64, error) {
	// Calculate fees and deposits as per spec
	validationFeesInDenom := fees
	validationTrustDepositInDenom := deposit

	// Increment trust deposit if deposit is greater than 0
	if validationTrustDepositInDenom > 0 {
		if err := ms.trustDeposit.AdjustTrustDeposit(ctx, msg.Creator, int64(validationTrustDepositInDenom)); err != nil {
			return 0, fmt.Errorf("failed to increase trust deposit: %w", err)
		}
	}

	// Send validation fees to escrow account if greater than 0
	if validationFeesInDenom > 0 {
		senderAddr, err := sdk.AccAddressFromBech32(msg.Creator)
		if err != nil {
			return 0, fmt.Errorf("invalid creator address: %w", err)
		}

		// Transfer fees to validation escrow account
		err = ms.bankKeeper.SendCoinsFromAccountToModule(
			ctx,
			senderAddr,
			types.ModuleName, // Validation escrow account
			sdk.NewCoins(sdk.NewInt64Coin(types.BondDenom, int64(validationFeesInDenom))),
		)
		if err != nil {
			return 0, fmt.Errorf("failed to transfer validation fees to escrow: %w", err)
		}
	}

	// Create new perm entry as specified in spec
	now := ctx.BlockTime()

	// Extract requested fees from optional fields
	var requestedValidationFees uint64
	var requestedIssuanceFees uint64
	var requestedVerificationFees uint64

	if msg.ValidationFees != nil {
		requestedValidationFees = msg.ValidationFees.Value
	}
	if msg.IssuanceFees != nil {
		requestedIssuanceFees = msg.IssuanceFees.Value
	}
	if msg.VerificationFees != nil {
		requestedVerificationFees = msg.VerificationFees.Value
	}

	applicantPerm := types.Permission{
		Grantee:            msg.Creator,                    // applicant_perm.grantee: applicant's account
		Type:               types.PermissionType(msg.Type), // applicant_perm.type: type
		SchemaId:           validatorPerm.SchemaId,
		Did:                msg.Did,
		Created:            &now, // applicant_perm.created: now
		CreatedBy:          msg.Creator,
		Modified:           &now,                          // applicant_perm.modified: now
		Deposit:            validationTrustDepositInDenom, // applicant_perm.deposit: validation_trust_deposit_in_denom
		ValidationFees:     requestedValidationFees,       // applicant_perm.validation_fees: validation_fees (from request)
		IssuanceFees:       requestedIssuanceFees,         // applicant_perm.issuance_fees: issuance_fees (from request)
		VerificationFees:   requestedVerificationFees,     // applicant_perm.verification_fees: verification_fees (from request)
		Country:            msg.Country,
		ValidatorPermId:    msg.ValidatorPermId,           // applicant_perm.validator_perm_id: validator_perm_id
		VpLastStateChange:  &now,                          // applicant_perm.vp_last_state_change: now
		VpState:            types.ValidationState_PENDING, // applicant_perm.vp_state: PENDING
		VpCurrentFees:      validationFeesInDenom,         // applicant_perm.vp_current_fees: validation_fees_in_denom
		VpCurrentDeposit:   validationTrustDepositInDenom, // applicant_perm.vp_current_deposit: validation_trust_deposit_in_denom
		VpSummaryDigestSri: "",                            // applicant_perm.vp_summary_digest_sri: null
		VpTermRequested:    nil,                           // applicant_perm.vp_term_requested: null
		VpValidatorDeposit: 0,                             // applicant_perm.vp_validator_deposit: 0
	}

	// Store the perm
	id, err := ms.Keeper.CreatePermission(ctx, applicantPerm)
	if err != nil {
		return 0, fmt.Errorf("failed to create perm: %w", err)
	}

	return id, nil
}

func validatePermissionTypeCombination(requestedType, validatorType types.PermissionType, cs credentialschematypes.CredentialSchema) error {
	switch requestedType {
	case types.PermissionType_ISSUER:
		if cs.IssuerPermManagementMode == credentialschematypes.CredentialSchemaPermManagementMode_GRANTOR_VALIDATION {
			if validatorType != types.PermissionType_ISSUER_GRANTOR {
				return fmt.Errorf("issuer perm requires ISSUER_GRANTOR validator")
			}
		} else if cs.IssuerPermManagementMode == credentialschematypes.CredentialSchemaPermManagementMode_ECOSYSTEM {
			if validatorType != types.PermissionType_ECOSYSTEM {
				return fmt.Errorf("issuer perm requires ECOSYSTEM validator")
			}
		} else if cs.IssuerPermManagementMode == credentialschematypes.CredentialSchemaPermManagementMode_OPEN {
			// Mode is OPEN which means anyone can issue credential of this schema
			// But formal perm creation is still needed when payment is required
			// Check if validator has the correct type for fee collection
			if validatorType != types.PermissionType_ECOSYSTEM {
				return fmt.Errorf("open issuance still requires ECOSYSTEM validator for fee collection")
			}
		} else {
			return fmt.Errorf("issuer perm not supported with current schema settings")
		}

	case types.PermissionType_ISSUER_GRANTOR:
		if cs.IssuerPermManagementMode == credentialschematypes.CredentialSchemaPermManagementMode_GRANTOR_VALIDATION {
			if validatorType != types.PermissionType_ECOSYSTEM {
				return fmt.Errorf("issuer grantor perm requires ECOSYSTEM validator")
			}
		} else {
			return fmt.Errorf("issuer grantor perm not supported with current schema settings")
		}

	case types.PermissionType_VERIFIER:
		if cs.VerifierPermManagementMode == credentialschematypes.CredentialSchemaPermManagementMode_GRANTOR_VALIDATION {
			if validatorType != types.PermissionType_VERIFIER_GRANTOR {
				return fmt.Errorf("verifier perm requires VERIFIER_GRANTOR validator")
			}
		} else if cs.VerifierPermManagementMode == credentialschematypes.CredentialSchemaPermManagementMode_ECOSYSTEM {
			if validatorType != types.PermissionType_ECOSYSTEM {
				return fmt.Errorf("verifier perm requires ECOSYSTEM validator")
			}
		} else if cs.VerifierPermManagementMode == credentialschematypes.CredentialSchemaPermManagementMode_OPEN {
			// Mode is OPEN which means anyone can verify credentials of this schema
			// This doesn't imply no payment is necessary - formal perm might be
			// required when payment is needed
			// Check if validator has the correct type for fee collection
			if validatorType != types.PermissionType_ECOSYSTEM {
				return fmt.Errorf("open verification still requires ECOSYSTEM validator for fee collection")
			}
		} else {
			return fmt.Errorf("verifier perm not supported with current schema settings")
		}

	case types.PermissionType_VERIFIER_GRANTOR:
		if cs.VerifierPermManagementMode == credentialschematypes.CredentialSchemaPermManagementMode_GRANTOR_VALIDATION {
			if validatorType != types.PermissionType_ECOSYSTEM {
				return fmt.Errorf("verifier grantor perm requires ECOSYSTEM validator")
			}
		} else {
			return fmt.Errorf("verifier grantor perm not supported with current schema settings")
		}

	case types.PermissionType_HOLDER:
		if cs.VerifierPermManagementMode == credentialschematypes.CredentialSchemaPermManagementMode_GRANTOR_VALIDATION ||
			cs.VerifierPermManagementMode == credentialschematypes.CredentialSchemaPermManagementMode_ECOSYSTEM {
			if validatorType != types.PermissionType_ISSUER {
				return fmt.Errorf("holder perm requires ISSUER validator")
			}
		} else if cs.VerifierPermManagementMode == credentialschematypes.CredentialSchemaPermManagementMode_OPEN {
			// Even in OPEN mode, holder permissions might require validation from an ISSUER
			if validatorType != types.PermissionType_ISSUER {
				return fmt.Errorf("holder perm requires ISSUER validator even in OPEN verification mode")
			}
		} else {
			return fmt.Errorf("holder perm not supported with current schema settings")
		}
	}

	return nil
}
