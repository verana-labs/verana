package keeper

import (
	"fmt"

	"cosmossdk.io/math"
	sdk "github.com/cosmos/cosmos-sdk/types"
	credentialschematypes "github.com/verana-labs/verana/x/cs/types"
	detypes "github.com/verana-labs/verana/x/de/types"
	"github.com/verana-labs/verana/x/pp/types"
)

func (ms msgServer) validateParticipantChecks(ctx sdk.Context, msg *types.MsgStartParticipantOP) (types.Participant, error) {
	// Load validator participant
	validatorParticipant, err := ms.Keeper.GetParticipantByID(ctx, msg.ValidatorParticipantId)
	if err != nil {
		return types.Participant{}, fmt.Errorf("validator participant not found: %w", err)
	}

	// [MOD-PP-MSG-1-2-2] Load Participant entry validator_participant from validator_participant_id.
	// It MUST be an active participant else transaction MUST abort.
	if err := IsValidParticipant(validatorParticipant, ctx.BlockTime()); err != nil {
		return types.Participant{}, fmt.Errorf("validator participant is not valid (must be ACTIVE): %w", err)
	}

	// Load credential schema from validator_participant.schema_id. It MUST exist.
	cs, err := ms.credentialSchemaKeeper.GetCredentialSchemaById(ctx, validatorParticipant.SchemaId)
	if err != nil {
		return types.Participant{}, fmt.Errorf("credential schema not found: %w", err)
	}

	// Validate participant type combinations per spec v4
	if err := validateParticipantRoleCombination(types.ParticipantRole(msg.Role), validatorParticipant.Role, cs); err != nil {
		return types.Participant{}, err
	}

	return validatorParticipant, nil
}

func (ms msgServer) validateAndCalculateFees(ctx sdk.Context, validatorParticipant types.Participant) (uint64, uint64, error) {
	// Get global variables
	trustUnitPrice := ms.ecosystemKeeper.GetTrustUnitPrice(ctx)
	trustDepositRate := ms.trustDeposit.GetTrustDepositRate(ctx)

	// Compute validator_participant.validation_fees * trust_unit_price via arbitrary-precision
	// math.Int so large fee * price products cannot wrap uint64 silently.
	feesProduct := math.NewIntFromUint64(validatorParticipant.ValidationFees).Mul(math.NewIntFromUint64(trustUnitPrice))
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

// [MOD-PP-MSG-1-2-4] Overlap checks
// Find all participants for (schema_id, type, validator_participant_id, authority) with op_state = VALIDATED or PENDING.
// If any found, abort — cannot have 2 active VPs in the same context.
func (ms msgServer) checkOverlap(ctx sdk.Context, schemaId uint64, participantType types.ParticipantRole, validatorParticipantId uint64, corporationId uint64) error {
	var found bool
	err := ms.Participant.Walk(ctx, nil, func(key uint64, participant types.Participant) (bool, error) {
		if participant.SchemaId == schemaId &&
			participant.Role == participantType &&
			participant.ValidatorParticipantId == validatorParticipantId &&
			participant.CorporationId == corporationId &&
			(participant.OpState == types.OnboardingState_PENDING || participant.OpState == types.OnboardingState_VALIDATED) {
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

func (ms msgServer) executeStartParticipantVP(ctx sdk.Context, msg *types.MsgStartParticipantOP, validatorParticipant types.Participant, fees, deposit uint64) (uint64, error) {
	validationFeesInDenom := fees
	validationTrustDepositInDenom := deposit

	// [MOD-PP-MSG-1-3] Use [MOD-TD-MSG-1] to increase trust deposit
	if validationTrustDepositInDenom > 0 {
		tdI64, err := uint64ToInt64(validationTrustDepositInDenom, "validation_trust_deposit")
		if err != nil {
			return 0, err
		}
		if err := ms.trustDeposit.AdjustTrustDeposit(ctx, msg.Corporation, tdI64, "start_participant_vp_deposit"); err != nil {
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
	applicantParticipant := types.Participant{
		CorporationId:                corporationId,                   // applicant_participant.corporation_id
		Role:                         types.ParticipantRole(msg.Role), // applicant_participant.role
		SchemaId:                     validatorParticipant.SchemaId,   // applicant_participant.schema_id = validator_participant.schema_id
		Did:                          msg.Did,
		VsOperator:                   msg.VsOperator,                   // applicant_participant.vs_operator
		Created:                      &now,                             // applicant_participant.created: now
		Modified:                     &now,                             // applicant_participant.modified: now
		Deposit:                      validationTrustDepositInDenom,    // applicant_participant.deposit
		ValidationFees:               requestedValidationFees,          // applicant_participant.validation_fees
		IssuanceFees:                 requestedIssuanceFees,            // applicant_participant.issuance_fees
		VerificationFees:             requestedVerificationFees,        // applicant_participant.verification_fees
		ValidatorParticipantId:       msg.ValidatorParticipantId,       // applicant_participant.validator_participant_id
		OpLastStateChange:            &now,                             // applicant_participant.op_last_state_change: now
		OpState:                      types.OnboardingState_PENDING,    // applicant_participant.op_state: PENDING
		OpCurrentFees:                validationFeesInDenom,            // applicant_participant.op_current_fees
		OpCurrentDeposit:             validationTrustDepositInDenom,    // applicant_participant.op_current_deposit
		OpSummaryDigest:              "",                               // applicant_participant.op_summary_digest: null
		OpValidatorDeposit:           0,                                // applicant_participant.op_validator_deposit: 0
	}

	id, err := ms.Keeper.CreateParticipant(ctx, applicantParticipant)
	if err != nil {
		return 0, fmt.Errorf("failed to create participant: %w", err)
	}

	// [MOD-PP-MSG-1-3] If VSOA params provided, create a DISABLED record (expiration
	// = now) via [MOD-DE-MSG-5]; it is activated at validation time by [MOD-DE-MSG-9].
	if len(msg.VsOperatorAuthzMsgTypes) > 0 {
		record := detypes.ParticipantAuthorizationRecord{
			ParticipantId: id,
			MsgTypes:      msg.VsOperatorAuthzMsgTypes,
			SpendLimit:    msg.VsOperatorAuthzSpendLimit,
			FeeSpendLimit: msg.VsOperatorAuthzFeeSpendLimit,
			WithFeegrant:  msg.VsOperatorAuthzWithFeegrant,
			Period:        msg.VsOperatorAuthzPeriod,
			Expiration:    &now, // disabled until validation
		}
		if err := ms.delegationKeeper.GrantVSOperatorAuthorization(ctx, corporationId, msg.VsOperator, record); err != nil {
			return 0, fmt.Errorf("failed to grant VS operator authorization: %w", err)
		}
	}

	return id, nil
}

// validateParticipantRoleCombination validates participant type combinations per spec v4 [MOD-PP-MSG-1-2-2]
func validateParticipantRoleCombination(requestedType, validatorType types.ParticipantRole, cs credentialschematypes.CredentialSchema) error {
	switch requestedType {
	case types.ParticipantRole_ISSUER:
		// if cs.issuer_participant_management_mode == GRANTOR: validator_participant.type MUST be ISSUER_GRANTOR
		if cs.IssuerOnboardingMode == credentialschematypes.IssuerOnboardingMode_ISSUER_ONBOARDING_MODE_GRANTOR_VALIDATION_PROCESS {
			if validatorType != types.ParticipantRole_ISSUER_GRANTOR {
				return fmt.Errorf("issuer participant requires ISSUER_GRANTOR validator when mode is GRANTOR_VALIDATION")
			}
		} else if cs.IssuerOnboardingMode == credentialschematypes.IssuerOnboardingMode_ISSUER_ONBOARDING_MODE_ECOSYSTEM_VALIDATION_PROCESS {
			// if cs.issuer_participant_management_mode == ECOSYSTEM: validator_participant.type MUST be ECOSYSTEM
			if validatorType != types.ParticipantRole_ECOSYSTEM {
				return fmt.Errorf("issuer participant requires ECOSYSTEM validator when mode is ECOSYSTEM")
			}
		} else {
			// else MUST abort
			return fmt.Errorf("issuer participant not supported with current schema issuer_participant_management_mode")
		}

	case types.ParticipantRole_ISSUER_GRANTOR:
		// if cs.issuer_participant_management_mode == GRANTOR: validator_participant.type MUST be ECOSYSTEM
		if cs.IssuerOnboardingMode == credentialschematypes.IssuerOnboardingMode_ISSUER_ONBOARDING_MODE_GRANTOR_VALIDATION_PROCESS {
			if validatorType != types.ParticipantRole_ECOSYSTEM {
				return fmt.Errorf("issuer grantor participant requires ECOSYSTEM validator")
			}
		} else {
			// else abort
			return fmt.Errorf("issuer grantor participant not supported with current schema settings")
		}

	case types.ParticipantRole_VERIFIER:
		// if cs.verifier_participant_management_mode == GRANTOR: validator_participant.type MUST be VERIFIER_GRANTOR
		if cs.VerifierOnboardingMode == credentialschematypes.VerifierOnboardingMode_VERIFIER_ONBOARDING_MODE_GRANTOR_VALIDATION_PROCESS {
			if validatorType != types.ParticipantRole_VERIFIER_GRANTOR {
				return fmt.Errorf("verifier participant requires VERIFIER_GRANTOR validator when mode is GRANTOR_VALIDATION")
			}
		} else if cs.VerifierOnboardingMode == credentialschematypes.VerifierOnboardingMode_VERIFIER_ONBOARDING_MODE_ECOSYSTEM_VALIDATION_PROCESS {
			// if cs.verifier_participant_management_mode == ECOSYSTEM: validator_participant.type MUST be ECOSYSTEM
			if validatorType != types.ParticipantRole_ECOSYSTEM {
				return fmt.Errorf("verifier participant requires ECOSYSTEM validator when mode is ECOSYSTEM")
			}
		} else {
			// else abort
			return fmt.Errorf("verifier participant not supported with current schema verifier_participant_management_mode")
		}

	case types.ParticipantRole_VERIFIER_GRANTOR:
		// if cs.verifier_participant_management_mode == GRANTOR: validator_participant.type MUST be ECOSYSTEM
		if cs.VerifierOnboardingMode == credentialschematypes.VerifierOnboardingMode_VERIFIER_ONBOARDING_MODE_GRANTOR_VALIDATION_PROCESS {
			if validatorType != types.ParticipantRole_ECOSYSTEM {
				return fmt.Errorf("verifier grantor participant requires ECOSYSTEM validator")
			}
		} else {
			// else abort
			return fmt.Errorf("verifier grantor participant not supported with current schema settings")
		}

	case types.ParticipantRole_HOLDER:
		// if cs.verifier_participant_management_mode == GRANTOR or ECOSYSTEM: validator_participant.type MUST be ISSUER
		if cs.VerifierOnboardingMode == credentialschematypes.VerifierOnboardingMode_VERIFIER_ONBOARDING_MODE_GRANTOR_VALIDATION_PROCESS ||
			cs.VerifierOnboardingMode == credentialschematypes.VerifierOnboardingMode_VERIFIER_ONBOARDING_MODE_ECOSYSTEM_VALIDATION_PROCESS {
			if validatorType != types.ParticipantRole_ISSUER {
				return fmt.Errorf("holder participant requires ISSUER validator")
			}
		} else {
			// else abort (spec v4: no OPEN mode fallback for HOLDER)
			return fmt.Errorf("holder participant not supported with current schema settings")
		}
	}

	return nil
}
