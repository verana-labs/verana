package keeper

import (
	"fmt"

	"cosmossdk.io/math"
	sdk "github.com/cosmos/cosmos-sdk/types"
	credentialschematypes "github.com/verana-labs/verana/x/cs/types"
	"github.com/verana-labs/verana/x/pp/types"
)

func (ms msgServer) validatePermissionChecks(ctx sdk.Context, msg *types.MsgStartParticipantOP) (types.Participant, error) {
	// Load validator perm
	validatorPerm, err := ms.Keeper.GetParticipantByID(ctx, msg.ValidatorParticipantId)
	if err != nil {
		return types.Participant{}, fmt.Errorf("validator perm not found: %w", err)
	}

	// [MOD-PERM-MSG-1-2-2] Load Participant entry validator_perm from validator_participant_id.
	// It MUST be an active permission else transaction MUST abort.
	if err := IsValidPermission(validatorPerm, ctx.BlockTime()); err != nil {
		return types.Participant{}, fmt.Errorf("validator perm is not valid (must be ACTIVE): %w", err)
	}

	// Load credential schema from validator_perm.schema_id. It MUST exist.
	cs, err := ms.credentialSchemaKeeper.GetCredentialSchemaById(ctx, validatorPerm.SchemaId)
	if err != nil {
		return types.Participant{}, fmt.Errorf("credential schema not found: %w", err)
	}

	// Validate perm type combinations per spec v4
	if err := validateParticipantRoleCombination(types.ParticipantRole(msg.Role), validatorPerm.Role, cs); err != nil {
		return types.Participant{}, err
	}

	return validatorPerm, nil
}

func (ms msgServer) validateAndCalculateFees(ctx sdk.Context, validatorPerm types.Participant) (uint64, uint64, error) {
	// Get global variables
	trustUnitPrice := ms.ecosystemKeeper.GetTrustUnitPrice(ctx)
	trustDepositRate := ms.trustDeposit.GetTrustDepositRate(ctx)

	// Compute validator_perm.validation_fees * trust_unit_price via arbitrary-precision
	// math.Int so large fee * price products cannot wrap uint64 silently.
	feesProduct := math.NewIntFromUint64(validatorPerm.ValidationFees).Mul(math.NewIntFromUint64(trustUnitPrice))
	if !feesProduct.IsUint64() {
		return 0, 0, fmt.Errorf("validation fees * trust_unit_price overflows uint64: %s", feesProduct.String())
	}
	validationFeesInDenom := feesProduct.Uint64()

	validationTrustDepositInDenom, err := ms.Keeper.validationTrustDepositInDenomAmount(validationFeesInDenom, trustDepositRate)
	if err != nil {
		return 0, 0, err
	}

	return validationFeesInDenom, validationTrustDepositInDenom, nil
}

func (k Keeper) validationTrustDepositInDenomAmount(validationFeesInDenom uint64, trustDepositRate math.LegacyDec) (uint64, error) {
	validationFeesInDenomDec := math.LegacyNewDecFromInt(math.NewIntFromUint64(validationFeesInDenom))
	tdInt := validationFeesInDenomDec.Mul(trustDepositRate).TruncateInt()
	if !tdInt.IsUint64() {
		return 0, fmt.Errorf("validation trust deposit overflows uint64: %s", tdInt.String())
	}
	return tdInt.Uint64(), nil
}

// [MOD-PERM-MSG-1-2-4] Overlap checks
// Find all permissions for (schema_id, type, validator_participant_id, authority) with op_state = VALIDATED or PENDING.
// If any found, abort — cannot have 2 active VPs in the same context.
func (ms msgServer) checkOverlap(ctx sdk.Context, schemaId uint64, permType types.ParticipantRole, validatorPermId uint64, corporationId uint64) error {
	var found bool
	err := ms.Participant.Walk(ctx, nil, func(key uint64, perm types.Participant) (bool, error) {
		if perm.SchemaId == schemaId &&
			perm.Role == permType &&
			perm.ValidatorParticipantId == validatorPermId &&
			perm.CorporationId == corporationId &&
			(perm.OpState == types.OnboardingState_PENDING || perm.OpState == types.OnboardingState_VALIDATED) {
			found = true
			return true, nil // stop walking
		}
		return false, nil
	})
	if err != nil {
		return fmt.Errorf("failed to check overlap: %w", err)
	}
	if found {
		return fmt.Errorf("an active validation process already exists for this (schema_id, type, validator_participant_id, authority) context")
	}
	return nil
}

func (ms msgServer) executeStartPermissionVP(ctx sdk.Context, msg *types.MsgStartParticipantOP, validatorPerm types.Participant, fees, deposit uint64) (uint64, error) {
	validationFeesInDenom := fees
	validationTrustDepositInDenom := deposit

	// [MOD-PERM-MSG-1-3] Use [MOD-TD-MSG-1] to increase trust deposit
	if validationTrustDepositInDenom > 0 {
		tdI64, err := uint64ToInt64(validationTrustDepositInDenom, "validation_trust_deposit")
		if err != nil {
			return 0, err
		}
		if err := ms.trustDeposit.AdjustTrustDeposit(ctx, msg.Corporation, tdI64, "start_perm_vp_deposit"); err != nil {
			return 0, fmt.Errorf("failed to increase trust deposit: %w", err)
		}
	}

	// Send validation fees to escrow account if greater than 0
	if validationFeesInDenom > 0 {
		senderAddr, err := sdk.AccAddressFromBech32(msg.Corporation)
		if err != nil {
			return 0, fmt.Errorf("invalid authority address: %w", err)
		}

		feesI64, err := uint64ToInt64(validationFeesInDenom, "validation_fees")
		if err != nil {
			return 0, err
		}
		err = ms.bankKeeper.SendCoinsFromAccountToModule(
			ctx,
			senderAddr,
			types.ModuleName,
			sdk.NewCoins(sdk.NewInt64Coin(types.BondDenom, feesI64)),
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

	// Resolve the signing corporation account (policy_address) to its uint64 id.
	corporationId, err := ms.corpIDFromAccount(ctx, msg.Corporation)
	if err != nil {
		return 0, err
	}

	// [MOD-PP-MSG-1-3] Create and persist new participant entry
	applicantPerm := types.Participant{
		CorporationId:                corporationId,                   // applicant_participant.corporation_id
		Role:                         types.ParticipantRole(msg.Role), // applicant_participant.role
		SchemaId:                     validatorPerm.SchemaId,          // applicant_perm.schema_id = validator_perm.schema_id
		Did:                          msg.Did,
		VsOperator:                   msg.VsOperator,                   // applicant_perm.vs_operator
		Created:                      &now,                             // applicant_perm.created: now
		Modified:                     &now,                             // applicant_perm.modified: now
		Deposit:                      validationTrustDepositInDenom,    // applicant_perm.deposit
		ValidationFees:               requestedValidationFees,          // applicant_perm.validation_fees
		IssuanceFees:                 requestedIssuanceFees,            // applicant_perm.issuance_fees
		VerificationFees:             requestedVerificationFees,        // applicant_perm.verification_fees
		ValidatorParticipantId:       msg.ValidatorParticipantId,       // applicant_perm.validator_participant_id
		OpLastStateChange:            &now,                             // applicant_perm.op_last_state_change: now
		OpState:                      types.OnboardingState_PENDING,    // applicant_perm.op_state: PENDING
		OpCurrentFees:                validationFeesInDenom,            // applicant_perm.op_current_fees
		OpCurrentDeposit:             validationTrustDepositInDenom,    // applicant_perm.op_current_deposit
		OpSummaryDigest:              "",                               // applicant_perm.op_summary_digest: null
		OpValidatorDeposit:           0,                                // applicant_perm.op_validator_deposit: 0
		VsOperatorAuthzEnabled:       msg.VsOperatorAuthzEnabled,       // applicant_perm.vs_operator_authz_enabled
		VsOperatorAuthzSpendLimit:    msg.VsOperatorAuthzSpendLimit,    // applicant_perm.vs_operator_authz_spend_limit
		VsOperatorAuthzWithFeegrant:  msg.VsOperatorAuthzWithFeegrant,  // applicant_perm.vs_operator_authz_with_feegrant
		VsOperatorAuthzFeeSpendLimit: msg.VsOperatorAuthzFeeSpendLimit, // applicant_perm.vs_operator_authz_fee_spend_limit
		VsOperatorAuthzSpendPeriod:   msg.VsOperatorAuthzSpendPeriod,   // applicant_perm.vs_operator_authz_spend_period
	}

	id, err := ms.Keeper.CreatePermission(ctx, applicantPerm)
	if err != nil {
		return 0, fmt.Errorf("failed to create perm: %w", err)
	}

	return id, nil
}

// validateParticipantRoleCombination validates permission type combinations per spec v4 [MOD-PERM-MSG-1-2-2]
func validateParticipantRoleCombination(requestedType, validatorType types.ParticipantRole, cs credentialschematypes.CredentialSchema) error {
	switch requestedType {
	case types.ParticipantRole_ISSUER:
		// if cs.issuer_perm_management_mode == GRANTOR: validator_perm.type MUST be ISSUER_GRANTOR
		if cs.IssuerOnboardingMode == credentialschematypes.IssuerOnboardingMode_ISSUER_ONBOARDING_MODE_GRANTOR_VALIDATION_PROCESS {
			if validatorType != types.ParticipantRole_ISSUER_GRANTOR {
				return fmt.Errorf("issuer perm requires ISSUER_GRANTOR validator when mode is GRANTOR_VALIDATION")
			}
		} else if cs.IssuerOnboardingMode == credentialschematypes.IssuerOnboardingMode_ISSUER_ONBOARDING_MODE_ECOSYSTEM_VALIDATION_PROCESS {
			// if cs.issuer_perm_management_mode == ECOSYSTEM: validator_perm.type MUST be ECOSYSTEM
			if validatorType != types.ParticipantRole_ECOSYSTEM {
				return fmt.Errorf("issuer perm requires ECOSYSTEM validator when mode is ECOSYSTEM")
			}
		} else {
			// else MUST abort
			return fmt.Errorf("issuer perm not supported with current schema issuer_perm_management_mode")
		}

	case types.ParticipantRole_ISSUER_GRANTOR:
		// if cs.issuer_perm_management_mode == GRANTOR: validator_perm.type MUST be ECOSYSTEM
		if cs.IssuerOnboardingMode == credentialschematypes.IssuerOnboardingMode_ISSUER_ONBOARDING_MODE_GRANTOR_VALIDATION_PROCESS {
			if validatorType != types.ParticipantRole_ECOSYSTEM {
				return fmt.Errorf("issuer grantor perm requires ECOSYSTEM validator")
			}
		} else {
			// else abort
			return fmt.Errorf("issuer grantor perm not supported with current schema settings")
		}

	case types.ParticipantRole_VERIFIER:
		// if cs.verifier_perm_management_mode == GRANTOR: validator_perm.type MUST be VERIFIER_GRANTOR
		if cs.VerifierOnboardingMode == credentialschematypes.VerifierOnboardingMode_VERIFIER_ONBOARDING_MODE_GRANTOR_VALIDATION_PROCESS {
			if validatorType != types.ParticipantRole_VERIFIER_GRANTOR {
				return fmt.Errorf("verifier perm requires VERIFIER_GRANTOR validator when mode is GRANTOR_VALIDATION")
			}
		} else if cs.VerifierOnboardingMode == credentialschematypes.VerifierOnboardingMode_VERIFIER_ONBOARDING_MODE_ECOSYSTEM_VALIDATION_PROCESS {
			// if cs.verifier_perm_management_mode == ECOSYSTEM: validator_perm.type MUST be ECOSYSTEM
			if validatorType != types.ParticipantRole_ECOSYSTEM {
				return fmt.Errorf("verifier perm requires ECOSYSTEM validator when mode is ECOSYSTEM")
			}
		} else {
			// else abort
			return fmt.Errorf("verifier perm not supported with current schema verifier_perm_management_mode")
		}

	case types.ParticipantRole_VERIFIER_GRANTOR:
		// if cs.verifier_perm_management_mode == GRANTOR: validator_perm.type MUST be ECOSYSTEM
		if cs.VerifierOnboardingMode == credentialschematypes.VerifierOnboardingMode_VERIFIER_ONBOARDING_MODE_GRANTOR_VALIDATION_PROCESS {
			if validatorType != types.ParticipantRole_ECOSYSTEM {
				return fmt.Errorf("verifier grantor perm requires ECOSYSTEM validator")
			}
		} else {
			// else abort
			return fmt.Errorf("verifier grantor perm not supported with current schema settings")
		}

	case types.ParticipantRole_HOLDER:
		// if cs.verifier_perm_management_mode == GRANTOR or ECOSYSTEM: validator_perm.type MUST be ISSUER
		if cs.VerifierOnboardingMode == credentialschematypes.VerifierOnboardingMode_VERIFIER_ONBOARDING_MODE_GRANTOR_VALIDATION_PROCESS ||
			cs.VerifierOnboardingMode == credentialschematypes.VerifierOnboardingMode_VERIFIER_ONBOARDING_MODE_ECOSYSTEM_VALIDATION_PROCESS {
			if validatorType != types.ParticipantRole_ISSUER {
				return fmt.Errorf("holder perm requires ISSUER validator")
			}
		} else {
			// else abort (spec v4: no OPEN mode fallback for HOLDER)
			return fmt.Errorf("holder perm not supported with current schema settings")
		}
	}

	return nil
}
