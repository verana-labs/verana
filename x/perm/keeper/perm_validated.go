package keeper

import (
	"cosmossdk.io/math"
	"fmt"
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

func (ms msgServer) executeSetPermissionVPToValidated(
	ctx sdk.Context,
	applicantPerm types.Permission,
	validatorPerm types.Permission,
	msg *types.MsgSetPermissionVPToValidated,
	now time.Time,
	vpExp *time.Time,
) (*types.MsgSetPermissionVPToValidatedResponse, error) {

	// Change value of provided effective_until if needed
	effectiveUntil := msg.EffectiveUntil
	if effectiveUntil == nil {
		// if provided effective_until is NULL: change value to vp_exp
		effectiveUntil = vpExp
	}

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
		applicantPerm.Country = msg.Country
		applicantPerm.EffectiveFrom = &now
	}

	// Fees and Trust Deposits:
	// transfer the full amount applicant_perm.vp_current_fees from escrow account to validator account
	if applicantPerm.VpCurrentFees > 0 {
		validatorAddr, err := sdk.AccAddressFromBech32(validatorPerm.Grantee)
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

		// Calculate validator_trust_deposit = applicant_perm.vp_current_fees * GlobalVariables.trust_deposit_rate
		trustDepositRate := ms.trustDeposit.GetTrustDepositRate(ctx)
		validatorTrustDeposit := ms.Keeper.validatorTrustDepositAmount(applicantPerm.VpCurrentFees, trustDepositRate)

		// Increase validator perm trust deposit: use [MOD-TD-MSG-1] to increase by validator_trust_deposit
		if validatorTrustDeposit > 0 {
			err = ms.trustDeposit.AdjustTrustDeposit(
				ctx,
				validatorPerm.Grantee,
				int64(validatorTrustDeposit),
			)
			if err != nil {
				return nil, fmt.Errorf("failed to adjust validator trust deposit: %w", err)
			}

			// Set applicant_perm.vp_validator_deposit to applicant_perm.vp_validator_deposit + validator_trust_deposit
			applicantPerm.VpValidatorDeposit += validatorTrustDeposit
		}
	}

	// set applicant_perm.vp_current_fees to 0
	applicantPerm.VpCurrentFees = 0
	// set applicant_perm.vp_current_deposit to 0
	applicantPerm.VpCurrentDeposit = 0

	// Persist the updated perm
	if err := ms.Keeper.UpdatePermission(ctx, applicantPerm); err != nil {
		return nil, fmt.Errorf("failed to update perm: %w", err)
	}

	return &types.MsgSetPermissionVPToValidatedResponse{}, nil
}

func (k Keeper) validatorTrustDepositAmount(vpCurrentFees uint64, trustDepositRate math.LegacyDec) uint64 {
	vpCurrentFeesDec := math.LegacyNewDec(int64(vpCurrentFees))
	validatorTrustDeposit := vpCurrentFeesDec.Mul(trustDepositRate)
	return validatorTrustDeposit.TruncateInt().Uint64()
}
