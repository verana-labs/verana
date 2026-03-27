package keeper

import (
	"fmt"
	"strconv"
	"time"

	sdk "github.com/cosmos/cosmos-sdk/types"
	cstypes "github.com/verana-labs/verana/x/cs/types"
	"github.com/verana-labs/verana/x/perm/types"
)

func getValidityPeriod(permType uint32, cs cstypes.CredentialSchema) uint32 {
	switch permType {
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

// [MOD-PERM-MSG-3-2-4] Overlap checks for SetPermissionVPToValidated
// Find all active permissions (not revoked, not slashed, not repaid) for schema_id, type, validator_perm_id, authority.
// For each, check that time ranges don't overlap.
func (ms msgServer) checkValidatedOverlap(ctx sdk.Context, applicantPerm types.Permission, effectiveUntil *time.Time) error {
	now := ctx.BlockTime()

	// Determine the effective_from and effective_until for the permission being validated
	permEffectiveFrom := applicantPerm.EffectiveFrom
	if permEffectiveFrom == nil {
		// First time validation: effective_from will be set to now
		permEffectiveFrom = &now
	}

	permEffectiveUntil := effectiveUntil
	// If effectiveUntil is nil, it will be set to vp_exp later, but for overlap check
	// a nil effective_until means never expires

	err := ms.Permission.Walk(ctx, nil, func(key uint64, perm types.Permission) (bool, error) {
		// Skip self
		if perm.Id == applicantPerm.Id {
			return false, nil
		}

		// Match on schema_id, type, validator_perm_id, authority
		if perm.SchemaId != applicantPerm.SchemaId ||
			perm.Type != applicantPerm.Type ||
			perm.ValidatorPermId != applicantPerm.ValidatorPermId ||
			perm.Authority != applicantPerm.Authority {
			return false, nil
		}

		// Skip non-active permissions (revoked, slashed, repaid)
		if perm.Revoked != nil || perm.Slashed != nil || perm.Repaid != nil {
			return false, nil
		}

		// Skip permissions without effective_from (not yet validated)
		if perm.EffectiveFrom == nil {
			return false, nil
		}

		// [MOD-PERM-MSG-3-2-4] if p.effective_until is NULL (never expire), abort
		if perm.EffectiveUntil == nil {
			return true, fmt.Errorf("existing permission %d never expires, cannot create overlapping permission", perm.Id)
		}

		// if p.effective_until is greater than effective_from, abort
		if perm.EffectiveUntil.After(*permEffectiveFrom) {
			return true, fmt.Errorf("existing permission %d overlaps: its effective_until is after this permission's effective_from", perm.Id)
		}

		// if p.effective_from is lower than effective_until, abort
		if permEffectiveUntil != nil && perm.EffectiveFrom.Before(*permEffectiveUntil) {
			return true, fmt.Errorf("existing permission %d overlaps: its effective_from is before this permission's effective_until", perm.Id)
		}

		return false, nil
	})
	if err != nil {
		return err
	}
	return nil
}

func (ms msgServer) executeSetPermissionVPToValidated(
	ctx sdk.Context,
	applicantPerm types.Permission,
	validatorPerm types.Permission,
	msg *types.MsgSetPermissionVPToValidated,
	now time.Time,
	vpExp *time.Time,
	effectiveUntil *time.Time,
) (*types.MsgSetPermissionVPToValidatedResponse, error) {

	// Update Permission applicant_perm:
	applicantPerm.Modified = &now
	applicantPerm.VpState = types.ValidationState_VALIDATED
	applicantPerm.VpLastStateChange = &now
	applicantPerm.VpSummaryDigestSri = msg.VpSummaryDigestSri
	applicantPerm.VpExp = vpExp
	applicantPerm.EffectiveUntil = effectiveUntil

	// if applicant_perm.effective_from IS NULL (first time method is called for this perm, not a renewal):
	if applicantPerm.EffectiveFrom == nil {
		applicantPerm.ValidationFees = msg.ValidationFees
		applicantPerm.IssuanceFees = msg.IssuanceFees
		applicantPerm.VerificationFees = msg.VerificationFees
		applicantPerm.IssuanceFeeDiscount = msg.IssuanceFeeDiscount
		applicantPerm.VerificationFeeDiscount = msg.VerificationFeeDiscount
		applicantPerm.EffectiveFrom = &now
	}
	// Renewal case: discounts are already validated to match existing, so no need to set them again

	// [MOD-PERM-MSG-3-3] Fees and Trust Deposits:
	// transfer the full amount applicant_perm.vp_current_fees from escrow account to validator account
	if applicantPerm.VpCurrentFees > 0 {
		validatorAddr, err := sdk.AccAddressFromBech32(validatorPerm.Authority)
		if err != nil {
			return nil, fmt.Errorf("invalid validator address: %w", err)
		}

		err = ms.bankKeeper.SendCoinsFromModuleToAccount(
			ctx,
			types.ModuleName,
			validatorAddr,
			sdk.NewCoins(sdk.NewInt64Coin(types.BondDenom, int64(applicantPerm.VpCurrentFees))),
		)
		if err != nil {
			return nil, fmt.Errorf("failed to transfer fees to validator: %w", err)
		}
	}

	// [MOD-PERM-MSG-3-3] Increase validator perm trust deposit:
	// use [MOD-TD-MSG-1] to increase by applicant_perm.vp_current_deposit
	if applicantPerm.VpCurrentDeposit > 0 {
		err := ms.trustDeposit.AdjustTrustDeposit(
			ctx,
			validatorPerm.Authority,
			int64(applicantPerm.VpCurrentDeposit),
		)
		if err != nil {
			return nil, fmt.Errorf("failed to adjust validator trust deposit: %w", err)
		}

		// Set applicant_perm.vp_validator_deposit to applicant_perm.vp_validator_deposit + applicant_perm.vp_current_deposit
		applicantPerm.VpValidatorDeposit += applicantPerm.VpCurrentDeposit
	}

	// set applicant_perm.vp_current_fees to 0
	applicantPerm.VpCurrentFees = 0
	// set applicant_perm.vp_current_deposit to 0
	applicantPerm.VpCurrentDeposit = 0

	// Persist the updated perm
	if err := ms.Keeper.UpdatePermission(ctx, applicantPerm); err != nil {
		return nil, fmt.Errorf("failed to update perm: %w", err)
	}

	// [MOD-PERM-MSG-3-3] If applicant_perm.type is ISSUER or VERIFIER and vs_operator_authz_enabled:
	// Grant VS Operator Authorization
	if (applicantPerm.Type == types.PermissionType_ISSUER || applicantPerm.Type == types.PermissionType_VERIFIER) &&
		applicantPerm.VsOperatorAuthzEnabled {
		if err := ms.grantVSOperatorAuthorization(ctx, applicantPerm); err != nil {
			return nil, fmt.Errorf("failed to grant VS operator authorization: %w", err)
		}
	}

	ctx.EventManager().EmitEvents(sdk.Events{
		sdk.NewEvent(
			types.EventTypeSetPermissionVPToValidated,
			sdk.NewAttribute(types.AttributeKeyPermissionID, strconv.FormatUint(msg.Id, 10)),
			sdk.NewAttribute(types.AttributeKeyAuthority, msg.Authority),
			sdk.NewAttribute(types.AttributeKeyOperator, msg.Operator),
			sdk.NewAttribute(types.AttributeKeyValidatorPermID, strconv.FormatUint(applicantPerm.ValidatorPermId, 10)),
			sdk.NewAttribute(types.AttributeKeyVpSummaryDigestSri, msg.VpSummaryDigestSri),
			sdk.NewAttribute(types.AttributeKeyEffectiveUntil, formatTimePtr(msg.EffectiveUntil)),
			sdk.NewAttribute(types.AttributeKeyValidationFees, strconv.FormatUint(msg.ValidationFees, 10)),
			sdk.NewAttribute(types.AttributeKeyIssuanceFees, strconv.FormatUint(msg.IssuanceFees, 10)),
			sdk.NewAttribute(types.AttributeKeyVerificationFees, strconv.FormatUint(msg.VerificationFees, 10)),
			sdk.NewAttribute(types.AttributeKeyVpExp, formatTimePtr(vpExp)),
			sdk.NewAttribute(types.AttributeKeyTimestamp, now.String()),
		),
	})

	return &types.MsgSetPermissionVPToValidatedResponse{}, nil
}

// grantVSOperatorAuthorization implements [MOD-DE-MSG-5] orchestration.
// Called by: SetPermissionVPToValidated, AdjustPermission, SelfCreatePermission.
// VSOA storage is in DE module; this method handles the business logic.
func (ms msgServer) grantVSOperatorAuthorization(ctx sdk.Context, perm types.Permission) error {
	if perm.VsOperator == "" {
		return nil
	}
	if ms.delegationKeeper == nil {
		return fmt.Errorf("delegation keeper is required for VS operator authorization")
	}

	// [MOD-DE-MSG-5-2] Basic checks: authority and vs_operator already validated by caller

	// Add permission to VSOA (DE handles mutual exclusivity check internally)
	if err := ms.delegationKeeper.AddPermToVSOA(ctx, perm.Authority, perm.VsOperator, perm.Id); err != nil {
		return fmt.Errorf("failed to grant VS operator authorization: %w", err)
	}

	// [MOD-DE-MSG-5-4] Handle feegrant
	if perm.VsOperatorAuthzWithFeegrant {
		expiration, err := ms.computeVSOAFeegrantExpiration(ctx, perm.Authority, perm.VsOperator)
		if err != nil {
			return fmt.Errorf("failed to compute feegrant expiration: %w", err)
		}

		// Only grant if expiration is nil (no limit) or in the future
		if expiration == nil || expiration.After(ctx.BlockTime()) {
			msgTypes := []string{"/verana.perm.v1.MsgCreateOrUpdatePermissionSession"}
			if err := ms.delegationKeeper.GrantFeeAllowance(ctx, perm.Authority, perm.VsOperator, msgTypes, expiration, nil, nil); err != nil {
				return fmt.Errorf("failed to grant fee allowance: %w", err)
			}
		}
	}

	return nil
}
