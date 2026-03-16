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

	// [MOD-PERM-MSG-1-2-2] Load Permission entry validator_perm from validator_perm_id.
	// It MUST be an active permission else transaction MUST abort.
	if err := IsValidPermission(validatorPerm, "", ctx.BlockTime()); err != nil {
		return types.Permission{}, fmt.Errorf("validator perm is not valid (must be ACTIVE): %w", err)
	}

	// Load credential schema from validator_perm.schema_id. It MUST exist.
	cs, err := ms.credentialSchemaKeeper.GetCredentialSchemaById(ctx, validatorPerm.SchemaId)
	if err != nil {
		return types.Permission{}, fmt.Errorf("credential schema not found: %w", err)
	}

	// Validate perm type combinations per spec v4
	if err := validatePermissionTypeCombination(types.PermissionType(msg.Type), validatorPerm.Type, cs); err != nil {
		return types.Permission{}, err
	}

	return validatorPerm, nil
}

func (ms msgServer) validateAndCalculateFees(ctx sdk.Context, validatorPerm types.Permission) (uint64, uint64, error) {
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

// [MOD-PERM-MSG-1-2-4] Overlap checks
// Find all permissions for (schema_id, type, validator_perm_id, authority) with vp_state = VALIDATED or PENDING.
// If any found, abort — cannot have 2 active VPs in the same context.
func (ms msgServer) checkOverlap(ctx sdk.Context, schemaId uint64, permType types.PermissionType, validatorPermId uint64, authority string) error {
	var found bool
	err := ms.Permission.Walk(ctx, nil, func(key uint64, perm types.Permission) (bool, error) {
		if perm.SchemaId == schemaId &&
			perm.Type == permType &&
			perm.ValidatorPermId == validatorPermId &&
			perm.Authority == authority &&
			(perm.VpState == types.ValidationState_PENDING || perm.VpState == types.ValidationState_VALIDATED) {
			found = true
			return true, nil // stop walking
		}
		return false, nil
	})
	if err != nil {
		return fmt.Errorf("failed to check overlap: %w", err)
	}
	if found {
		return fmt.Errorf("an active validation process already exists for this (schema_id, type, validator_perm_id, authority) context")
	}
	return nil
}

func (ms msgServer) executeStartPermissionVP(ctx sdk.Context, msg *types.MsgStartPermissionVP, validatorPerm types.Permission, fees, deposit uint64) (uint64, error) {
	validationFeesInDenom := fees
	validationTrustDepositInDenom := deposit

	// [MOD-PERM-MSG-1-3] Use [MOD-TD-MSG-1] to increase trust deposit
	if validationTrustDepositInDenom > 0 {
		if err := ms.trustDeposit.AdjustTrustDeposit(ctx, msg.Authority, int64(validationTrustDepositInDenom)); err != nil {
			return 0, fmt.Errorf("failed to increase trust deposit: %w", err)
		}
	}

	// Send validation fees to escrow account if greater than 0
	if validationFeesInDenom > 0 {
		senderAddr, err := sdk.AccAddressFromBech32(msg.Authority)
		if err != nil {
			return 0, fmt.Errorf("invalid authority address: %w", err)
		}

		err = ms.bankKeeper.SendCoinsFromAccountToModule(
			ctx,
			senderAddr,
			types.ModuleName,
			sdk.NewCoins(sdk.NewInt64Coin(types.BondDenom, int64(validationFeesInDenom))),
		)
		if err != nil {
			return 0, fmt.Errorf("failed to transfer validation fees to escrow: %w", err)
		}
	}

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

	// [MOD-PERM-MSG-1-3] Create and persist new permission entry
	applicantPerm := types.Permission{
		Authority:                    msg.Authority,                  // applicant_perm.authority
		Type:                         types.PermissionType(msg.Type), // applicant_perm.type
		SchemaId:                     validatorPerm.SchemaId,         // applicant_perm.schema_id = validator_perm.schema_id
		Did:                          msg.Did,
		VsOperator:                   msg.VsOperator,                        // applicant_perm.vs_operator
		Created:                      &now,                                  // applicant_perm.created: now
		CreatedBy:                    msg.Operator,                          // created_by: operator who signed the tx
		Modified:                     &now,                                  // applicant_perm.modified: now
		Deposit:                      validationTrustDepositInDenom,         // applicant_perm.deposit
		ValidationFees:               requestedValidationFees,               // applicant_perm.validation_fees
		IssuanceFees:                 requestedIssuanceFees,                 // applicant_perm.issuance_fees
		VerificationFees:             requestedVerificationFees,             // applicant_perm.verification_fees
		ValidatorPermId:              msg.ValidatorPermId,                   // applicant_perm.validator_perm_id
		VpLastStateChange:            &now,                                  // applicant_perm.vp_last_state_change: now
		VpState:                      types.ValidationState_PENDING,         // applicant_perm.vp_state: PENDING
		VpCurrentFees:                validationFeesInDenom,                 // applicant_perm.vp_current_fees
		VpCurrentDeposit:             validationTrustDepositInDenom,         // applicant_perm.vp_current_deposit
		VpSummaryDigestSri:           "",                                    // applicant_perm.vp_summary_digest: null
		VpTermRequested:              nil,                                   // not set
		VpValidatorDeposit:           0,                                     // applicant_perm.vp_validator_deposit: 0
		VsOperatorAuthzEnabled:       msg.VsOperatorAuthzEnabled,            // applicant_perm.vs_operator_authz_enabled
		VsOperatorAuthzSpendLimit:    msg.VsOperatorAuthzSpendLimit,         // applicant_perm.vs_operator_authz_spend_limit
		VsOperatorAuthzWithFeegrant:  msg.VsOperatorAuthzWithFeegrant,       // applicant_perm.vs_operator_authz_with_feegrant
		VsOperatorAuthzFeeSpendLimit: msg.VsOperatorAuthzFeeSpendLimit,      // applicant_perm.vs_operator_authz_fee_spend_limit
		VsOperatorAuthzSpendPeriod:   msg.VsOperatorAuthzSpendPeriod,        // applicant_perm.vs_operator_authz_spend_period
	}

	id, err := ms.Keeper.CreatePermission(ctx, applicantPerm)
	if err != nil {
		return 0, fmt.Errorf("failed to create perm: %w", err)
	}

	return id, nil
}

// validatePermissionTypeCombination validates permission type combinations per spec v4 [MOD-PERM-MSG-1-2-2]
func validatePermissionTypeCombination(requestedType, validatorType types.PermissionType, cs credentialschematypes.CredentialSchema) error {
	switch requestedType {
	case types.PermissionType_ISSUER:
		// if cs.issuer_perm_management_mode == GRANTOR: validator_perm.type MUST be ISSUER_GRANTOR
		if cs.IssuerPermManagementMode == credentialschematypes.CredentialSchemaPermManagementMode_GRANTOR_VALIDATION {
			if validatorType != types.PermissionType_ISSUER_GRANTOR {
				return fmt.Errorf("issuer perm requires ISSUER_GRANTOR validator when mode is GRANTOR_VALIDATION")
			}
		} else if cs.IssuerPermManagementMode == credentialschematypes.CredentialSchemaPermManagementMode_ECOSYSTEM {
			// if cs.issuer_perm_management_mode == ECOSYSTEM: validator_perm.type MUST be ECOSYSTEM
			if validatorType != types.PermissionType_ECOSYSTEM {
				return fmt.Errorf("issuer perm requires ECOSYSTEM validator when mode is ECOSYSTEM")
			}
		} else {
			// else MUST abort
			return fmt.Errorf("issuer perm not supported with current schema issuer_perm_management_mode")
		}

	case types.PermissionType_ISSUER_GRANTOR:
		// if cs.issuer_perm_management_mode == GRANTOR: validator_perm.type MUST be ECOSYSTEM
		if cs.IssuerPermManagementMode == credentialschematypes.CredentialSchemaPermManagementMode_GRANTOR_VALIDATION {
			if validatorType != types.PermissionType_ECOSYSTEM {
				return fmt.Errorf("issuer grantor perm requires ECOSYSTEM validator")
			}
		} else {
			// else abort
			return fmt.Errorf("issuer grantor perm not supported with current schema settings")
		}

	case types.PermissionType_VERIFIER:
		// if cs.verifier_perm_management_mode == GRANTOR: validator_perm.type MUST be VERIFIER_GRANTOR
		if cs.VerifierPermManagementMode == credentialschematypes.CredentialSchemaPermManagementMode_GRANTOR_VALIDATION {
			if validatorType != types.PermissionType_VERIFIER_GRANTOR {
				return fmt.Errorf("verifier perm requires VERIFIER_GRANTOR validator when mode is GRANTOR_VALIDATION")
			}
		} else if cs.VerifierPermManagementMode == credentialschematypes.CredentialSchemaPermManagementMode_ECOSYSTEM {
			// if cs.verifier_perm_management_mode == ECOSYSTEM: validator_perm.type MUST be ECOSYSTEM
			if validatorType != types.PermissionType_ECOSYSTEM {
				return fmt.Errorf("verifier perm requires ECOSYSTEM validator when mode is ECOSYSTEM")
			}
		} else {
			// else abort
			return fmt.Errorf("verifier perm not supported with current schema verifier_perm_management_mode")
		}

	case types.PermissionType_VERIFIER_GRANTOR:
		// if cs.verifier_perm_management_mode == GRANTOR: validator_perm.type MUST be ECOSYSTEM
		if cs.VerifierPermManagementMode == credentialschematypes.CredentialSchemaPermManagementMode_GRANTOR_VALIDATION {
			if validatorType != types.PermissionType_ECOSYSTEM {
				return fmt.Errorf("verifier grantor perm requires ECOSYSTEM validator")
			}
		} else {
			// else abort
			return fmt.Errorf("verifier grantor perm not supported with current schema settings")
		}

	case types.PermissionType_HOLDER:
		// if cs.verifier_perm_management_mode == GRANTOR or ECOSYSTEM: validator_perm.type MUST be ISSUER
		if cs.VerifierPermManagementMode == credentialschematypes.CredentialSchemaPermManagementMode_GRANTOR_VALIDATION ||
			cs.VerifierPermManagementMode == credentialschematypes.CredentialSchemaPermManagementMode_ECOSYSTEM {
			if validatorType != types.PermissionType_ISSUER {
				return fmt.Errorf("holder perm requires ISSUER validator")
			}
		} else {
			// else abort (spec v4: no OPEN mode fallback for HOLDER)
			return fmt.Errorf("holder perm not supported with current schema settings")
		}
	}

	return nil
}
