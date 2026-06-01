package keeper

import (
	"fmt"
	"strconv"
	"time"

	sdk "github.com/cosmos/cosmos-sdk/types"
	cstypes "github.com/verana-labs/verana/x/cs/types"
	"github.com/verana-labs/verana/x/pp/types"
)

func getValidityPeriod(participantType uint32, cs cstypes.CredentialSchema) uint32 {
	switch participantType {
	case 3: // ISSUER_GRANTOR
		return cs.IssuerGrantorValidationValidityPeriod
	case 4: // VERIFIER_GRANTOR
		return cs.VerifierGrantorValidationValidityPeriod
	case 1: // ISSUER
		return cs.IssuerValidationValidityPeriod
	case 2: // VERIFIER
		return cs.VerifierValidationValidityPeriod
	case 6: // HOLDER
		return cs.HolderValidationValidityPeriod
	default:
		return 0
	}
}

func calculateVPExp(currentVPExp *time.Time, validityPeriod uint64, now time.Time) *time.Time {
	if validityPeriod == 0 {
		return nil
	}

	var exp time.Time
	if currentVPExp == nil {
		exp = now.AddDate(0, 0, int(validityPeriod))
	} else {
		exp = currentVPExp.AddDate(0, 0, int(validityPeriod))
	}
	return &exp
}

// [MOD-PP-MSG-3-2-4] Overlap checks for SetParticipantOPToValidated
// Find all active participants (not revoked, not slashed, not repaid) for schema_id, type, validator_participant_id, authority.
// For each, check that time ranges don't overlap.
func (ms msgServer) checkValidatedOverlap(ctx sdk.Context, applicantParticipant types.Participant, effectiveUntil *time.Time) error {
	now := ctx.BlockTime()

	// Determine the effective_from and effective_until for the participant being validated
	participantEffectiveFrom := applicantParticipant.EffectiveFrom
	if participantEffectiveFrom == nil {
		// First time validation: effective_from will be set to now
		participantEffectiveFrom = &now
	}

	participantEffectiveUntil := effectiveUntil
	// If effectiveUntil is nil, it will be set to op_exp later, but for overlap check
	// a nil effective_until means never expires

	err := ms.Participant.Walk(ctx, nil, func(key uint64, participant types.Participant) (bool, error) {
		// Skip self
		if participant.Id == applicantParticipant.Id {
			return false, nil
		}

		// Match on schema_id, role, validator_participant_id, corporation
		if participant.SchemaId != applicantParticipant.SchemaId ||
			participant.Role != applicantParticipant.Role ||
			participant.ValidatorParticipantId != applicantParticipant.ValidatorParticipantId ||
			participant.CorporationId != applicantParticipant.CorporationId {
			return false, nil
		}

		// Skip non-active participants (revoked, slashed, repaid)
		if participant.Revoked != nil || participant.Slashed != nil || participant.Repaid != nil {
			return false, nil
		}

		// Skip participants without effective_from (not yet validated)
		if participant.EffectiveFrom == nil {
			return false, nil
		}

		// [MOD-PP-MSG-3-2-4] if p.effective_until is NULL (never expire), abort
		if participant.EffectiveUntil == nil {
			return true, fmt.Errorf("existing participant %d never expires, cannot create overlapping participant", participant.Id)
		}

		// if p.effective_until is greater than effective_from, abort
		if participant.EffectiveUntil.After(*participantEffectiveFrom) {
			return true, fmt.Errorf("existing participant %d overlaps: its effective_until is after this participant's effective_from", participant.Id)
		}

		// if p.effective_from is lower than effective_until, abort
		if participantEffectiveUntil != nil && participant.EffectiveFrom.Before(*participantEffectiveUntil) {
			return true, fmt.Errorf("existing participant %d overlaps: its effective_from is before this participant's effective_until", participant.Id)
		}

		return false, nil
	})
	if err != nil {
		return err
	}
	return nil
}

func (ms msgServer) executeSetParticipantVPToValidated(
	ctx sdk.Context,
	applicantParticipant types.Participant,
	validatorParticipant types.Participant,
	msg *types.MsgSetParticipantOPToValidated,
	now time.Time,
	vpExp *time.Time,
	effectiveUntil *time.Time,
) (*types.MsgSetParticipantOPToValidatedResponse, error) {

	// Guard: cannot validate a slashed participant that has not been repaid
	if applicantParticipant.Slashed != nil && applicantParticipant.Repaid == nil {
		return nil, fmt.Errorf("cannot validate a slashed participant that has not been repaid")
	}

	// Update Participant applicant_participant:
	applicantParticipant.Modified = &now
	applicantParticipant.OpState = types.OnboardingState_VALIDATED
	applicantParticipant.OpLastStateChange = &now
	applicantParticipant.OpSummaryDigest = msg.OpSummaryDigest
	applicantParticipant.OpExp = vpExp
	applicantParticipant.EffectiveUntil = effectiveUntil

	// if applicant_participant.effective_from IS NULL (first time method is called for this participant, not a renewal):
	if applicantParticipant.EffectiveFrom == nil {
		applicantParticipant.ValidationFees = msg.ValidationFees
		applicantParticipant.IssuanceFees = msg.IssuanceFees
		applicantParticipant.VerificationFees = msg.VerificationFees
		applicantParticipant.IssuanceFeeDiscount = msg.IssuanceFeeDiscount
		applicantParticipant.VerificationFeeDiscount = msg.VerificationFeeDiscount
		applicantParticipant.EffectiveFrom = &now
	}
	// Renewal case: discounts are already validated to match existing, so no need to set them again

	// [MOD-PP-MSG-3-3] Fees and Trust Deposits:
	// transfer the full amount applicant_participant.op_current_fees from escrow account to validator account
	validatorCorpAcct, err := ms.corpAccountFromID(ctx, validatorParticipant.CorporationId)
	if err != nil {
		return nil, err
	}
	if applicantParticipant.OpCurrentFees > 0 {
		validatorAddr, err := sdk.AccAddressFromBech32(validatorCorpAcct)
		if err != nil {
			return nil, fmt.Errorf("invalid validator address: %w", err)
		}

		vpCurrentFeesI64, err := uint64ToInt64(applicantParticipant.OpCurrentFees, "op_current_fees")
		if err != nil {
			return nil, err
		}
		err = ms.bankKeeper.SendCoinsFromModuleToAccount(
			ctx,
			types.ModuleName,
			validatorAddr,
			sdk.NewCoins(sdk.NewInt64Coin(types.BondDenom, vpCurrentFeesI64)),
		)
		if err != nil {
			return nil, fmt.Errorf("failed to transfer fees to validator: %w", err)
		}
	}

	// [MOD-PP-MSG-3-3] Increase validator participant trust deposit:
	// use [MOD-TD-MSG-1] to increase by applicant_participant.op_current_deposit
	if applicantParticipant.OpCurrentDeposit > 0 {
		vpCurrentDepositI64, err := uint64ToInt64(applicantParticipant.OpCurrentDeposit, "op_current_deposit")
		if err != nil {
			return nil, err
		}
		err = ms.trustDeposit.AdjustTrustDeposit(
			ctx,
			validatorCorpAcct,
			vpCurrentDepositI64,
			"participant_validated_deposit",
		)
		if err != nil {
			return nil, fmt.Errorf("failed to adjust validator trust deposit: %w", err)
		}

		// Set applicant_participant.op_validator_deposit to applicant_participant.op_validator_deposit + applicant_participant.op_current_deposit
		applicantParticipant.OpValidatorDeposit += applicantParticipant.OpCurrentDeposit
	}

	// set applicant_participant.op_current_fees to 0
	applicantParticipant.OpCurrentFees = 0
	// set applicant_participant.op_current_deposit to 0
	applicantParticipant.OpCurrentDeposit = 0

	// Persist the updated participant
	if err := ms.Keeper.UpdateParticipant(ctx, applicantParticipant); err != nil {
		return nil, fmt.Errorf("failed to update participant: %w", err)
	}

	// [MOD-PP-MSG-3-3] Activate any disabled VSOA record by syncing its expiration
	// to the participant's effective_until via [MOD-DE-MSG-9]. No-op if no record.
	if applicantParticipant.EffectiveUntil != nil {
		if err := ms.delegationKeeper.UpdateVSOperatorAuthorizationExpiration(ctx, applicantParticipant.Id, *applicantParticipant.EffectiveUntil); err != nil {
			return nil, fmt.Errorf("failed to update VS operator authorization expiration: %w", err)
		}
	}

	ctx.EventManager().EmitEvents(sdk.Events{
		sdk.NewEvent(
			types.EventTypeSetParticipantOPToValidated,
			sdk.NewAttribute(types.AttributeKeyParticipantID, strconv.FormatUint(msg.Id, 10)),
			sdk.NewAttribute(types.AttributeKeyCorporation, msg.Corporation),
			sdk.NewAttribute(types.AttributeKeyOperator, msg.Operator),
			sdk.NewAttribute(types.AttributeKeyValidatorParticipantID, strconv.FormatUint(applicantParticipant.ValidatorParticipantId, 10)),
			sdk.NewAttribute(types.AttributeKeyOpSummaryDigest, msg.OpSummaryDigest),
			sdk.NewAttribute(types.AttributeKeyEffectiveUntil, formatTimePtr(msg.EffectiveUntil)),
			sdk.NewAttribute(types.AttributeKeyValidationFees, strconv.FormatUint(msg.ValidationFees, 10)),
			sdk.NewAttribute(types.AttributeKeyIssuanceFees, strconv.FormatUint(msg.IssuanceFees, 10)),
			sdk.NewAttribute(types.AttributeKeyVerificationFees, strconv.FormatUint(msg.VerificationFees, 10)),
			sdk.NewAttribute(types.AttributeKeyOpExp, formatTimePtr(vpExp)),
			sdk.NewAttribute(types.AttributeKeyTimestamp, now.String()),
		),
	})

	return &types.MsgSetParticipantOPToValidatedResponse{}, nil
}

