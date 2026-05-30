package keeper

import (
	"context"
	"fmt"
	"strconv"
	"time"

	credentialschematypes "github.com/verana-labs/verana/x/cs/types"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/verana-labs/verana/x/pp/types"
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

// StartParticipantOP handles the MsgStartParticipantOP message
func (ms msgServer) StartParticipantOP(goCtx context.Context, msg *types.MsgStartParticipantOP) (*types.MsgStartParticipantOPResponse, error) {
	ctx := sdk.UnwrapSDKContext(goCtx)
	now := ctx.BlockTime()

	// [MOD-PERM-MSG-1-2-1] [AUTHZ-CHECK] Verify operator authorization
	if ms.delegationKeeper == nil {
		return nil, fmt.Errorf("delegation keeper is required for operator authorization")
	}
	if err := ms.delegationKeeper.CheckOperatorAuthorization(
		ctx,
		msg.Corporation,
		msg.Operator,
		"/verana.pp.v1.MsgStartParticipantOP",
		now,
	); err != nil {
		return nil, fmt.Errorf("authorization check failed: %w", err)
	}

	// [MOD-PERM-MSG-1-2-2] Participant checks
	validatorPerm, err := ms.validatePermissionChecks(ctx, msg)
	if err != nil {
		return nil, fmt.Errorf("perm validation failed: %w", err)
	}

	// [MOD-PP-MSG-1-2-4] Overlap checks
	corporationId, err := ms.corpIDFromAccount(ctx, msg.Corporation)
	if err != nil {
		return nil, err
	}
	if err := ms.checkOverlap(ctx, validatorPerm.SchemaId, msg.Role, msg.ValidatorParticipantId, corporationId); err != nil {
		return nil, fmt.Errorf("overlap check failed: %w", err)
	}

	// [MOD-PERM-MSG-1-2-3] Fee checks
	fees, deposit, err := ms.validateAndCalculateFees(ctx, validatorPerm)
	if err != nil {
		return nil, fmt.Errorf("fee validation failed: %w", err)
	}

	// [MOD-PERM-MSG-1-3] Execute the perm VP creation
	permID, err := ms.executeStartPermissionVP(ctx, msg, validatorPerm, fees, deposit)
	if err != nil {
		return nil, fmt.Errorf("failed to execute perm VP: %w", err)
	}

	ctx.EventManager().EmitEvents(sdk.Events{
		sdk.NewEvent(
			types.EventTypeStartParticipantOP,
			sdk.NewAttribute(types.AttributeKeyParticipantID, strconv.FormatUint(permID, 10)),
			sdk.NewAttribute(types.AttributeKeyCorporation, msg.Corporation),
			sdk.NewAttribute(types.AttributeKeyOperator, msg.Operator),
			sdk.NewAttribute(types.AttributeKeyValidatorParticipantID, strconv.FormatUint(msg.ValidatorParticipantId, 10)),
			sdk.NewAttribute(types.AttributeKeyRole, types.ParticipantRole(msg.Role).String()),
			sdk.NewAttribute(types.AttributeKeyFees, strconv.FormatUint(fees, 10)),
			sdk.NewAttribute(types.AttributeKeyDeposit, strconv.FormatUint(deposit, 10)),
			sdk.NewAttribute(types.AttributeKeyTimestamp, now.String()),
		),
	})

	return &types.MsgStartParticipantOPResponse{
		ParticipantId: permID,
	}, nil
}

func (ms msgServer) RenewParticipantOP(goCtx context.Context, msg *types.MsgRenewParticipantOP) (*types.MsgRenewParticipantOPResponse, error) {
	ctx := sdk.UnwrapSDKContext(goCtx)
	now := ctx.BlockTime()

	// [MOD-PERM-MSG-2-2-1] [AUTHZ-CHECK] Verify operator authorization
	if ms.delegationKeeper == nil {
		return nil, fmt.Errorf("delegation keeper is required for operator authorization")
	}
	if err := ms.delegationKeeper.CheckOperatorAuthorization(
		ctx,
		msg.Corporation,
		msg.Operator,
		"/verana.pp.v1.MsgRenewParticipantOP",
		now,
	); err != nil {
		return nil, fmt.Errorf("authorization check failed: %w", err)
	}

	// [MOD-PERM-MSG-2-2-2] Participant checks
	applicantPerm, err := ms.Keeper.GetParticipantByID(ctx, msg.Id)
	if err != nil {
		return nil, fmt.Errorf("perm not found: %w", err)
	}

	// [MOD-PP-MSG-2-2-2] authority MUST be applicant_participant.corporation
	applicantCorpAcct, err := ms.corpAccountFromID(ctx, applicantPerm.CorporationId)
	if err != nil {
		return nil, err
	}
	if applicantCorpAcct != msg.Corporation {
		return nil, fmt.Errorf("authority is not the participant authority")
	}

	// [MOD-PERM-MSG-2-2-2] applicant_perm.op_state MUST be VALIDATED to allow renewal.
	// Renewing a PENDING perm would overwrite op_current_fees/op_current_deposit without
	// refunding the escrowed funds, causing permanent fund loss.
	if applicantPerm.OpState != types.OnboardingState_VALIDATED {
		return nil, fmt.Errorf("perm op_state must be VALIDATED to renew, current state: %s", applicantPerm.OpState.String())
	}

	// [MOD-PERM-MSG-2-2-2] applicant_perm MUST be an active permission.
	// Spec: "active permission" = effective_from < now AND (effective_until is null OR > now)
	// AND revoked is null AND slashed is null. Without this check, revoked/slashed/expired
	// permissions can be renewed, bypassing governance revocation.
	if err := IsValidPermission(applicantPerm, ctx.BlockTime()); err != nil {
		return nil, fmt.Errorf("applicant perm is not active: %w", err)
	}

	// Get validator perm
	validatorPerm, err := ms.Keeper.GetParticipantByID(ctx, applicantPerm.ValidatorParticipantId)
	if err != nil {
		return nil, fmt.Errorf("validator perm not found: %w", err)
	}

	if err := IsValidPermission(validatorPerm, ctx.BlockTime()); err != nil {
		return nil, fmt.Errorf("validator perm is not valid: %w", err)
	}

	// [MOD-PERM-MSG-2-2-3] Fee checks
	validationFees, validationDeposit, err := ms.validateAndCalculateFees(ctx, validatorPerm)
	if err != nil {
		return nil, fmt.Errorf("fee validation failed: %w", err)
	}

	// [MOD-PERM-MSG-2-3] Execution
	if err := ms.executeRenewPermissionVP(ctx, applicantPerm, validationFees, validationDeposit); err != nil {
		return nil, fmt.Errorf("failed to execute perm VP renewal: %w", err)
	}

	ctx.EventManager().EmitEvents(sdk.Events{
		sdk.NewEvent(
			types.EventTypeRenewParticipantOP,
			sdk.NewAttribute(types.AttributeKeyParticipantID, strconv.FormatUint(msg.Id, 10)),
			sdk.NewAttribute(types.AttributeKeyCorporation, msg.Corporation),
			sdk.NewAttribute(types.AttributeKeyOperator, msg.Operator),
			sdk.NewAttribute(types.AttributeKeyValidatorParticipantID, strconv.FormatUint(applicantPerm.ValidatorParticipantId, 10)),
			sdk.NewAttribute(types.AttributeKeyValidationFees, strconv.FormatUint(validationFees, 10)),
			sdk.NewAttribute(types.AttributeKeyValidationDeposit, strconv.FormatUint(validationDeposit, 10)),
			sdk.NewAttribute(types.AttributeKeyTimestamp, now.String()),
		),
	})

	return &types.MsgRenewParticipantOPResponse{}, nil
}

func (ms msgServer) executeRenewPermissionVP(ctx sdk.Context, perm types.Participant, fees, deposit uint64) error {
	corpAcct, err := ms.corpAccountFromID(ctx, perm.CorporationId)
	if err != nil {
		return err
	}
	// Increment trust deposit if deposit is greater than 0
	if deposit > 0 {
		depositI64, err := uint64ToInt64(deposit, "renew_deposit")
		if err != nil {
			return err
		}
		if err := ms.trustDeposit.AdjustTrustDeposit(ctx, corpAcct, depositI64, "renew_perm_deposit"); err != nil {
			return fmt.Errorf("failed to increase trust deposit: %w", err)
		}
	}

	// Send validation fees to escrow account if greater than 0
	if fees > 0 {
		// Get grantee address
		granteeAddr, err := sdk.AccAddressFromBech32(corpAcct)
		if err != nil {
			return fmt.Errorf("invalid grantee address: %w", err)
		}

		feesI64, err := uint64ToInt64(fees, "renew_fees")
		if err != nil {
			return err
		}
		// Transfer fees to module escrow account
		err = ms.bankKeeper.SendCoinsFromAccountToModule(
			ctx,
			granteeAddr,
			types.ModuleName, // Using module name as the escrow account
			sdk.NewCoins(sdk.NewInt64Coin(types.BondDenom, feesI64)),
		)
		if err != nil {
			return fmt.Errorf("failed to transfer validation fees to escrow: %w", err)
		}
	}

	now := ctx.BlockTime()

	// Update perm
	perm.OpState = types.OnboardingState_PENDING
	perm.OpLastStateChange = &now
	perm.Deposit += deposit
	perm.OpCurrentFees = fees
	perm.OpCurrentDeposit = deposit
	perm.Modified = &now

	// Store updated perm
	return ms.Keeper.UpdatePermission(ctx, perm)
}

func (ms msgServer) SetParticipantOPToValidated(goCtx context.Context, msg *types.MsgSetParticipantOPToValidated) (*types.MsgSetParticipantOPToValidatedResponse, error) {
	ctx := sdk.UnwrapSDKContext(goCtx)
	now := ctx.BlockTime()

	// [MOD-PERM-MSG-3-2-1] [AUTHZ-CHECK] Verify operator authorization
	if ms.delegationKeeper == nil {
		return nil, fmt.Errorf("delegation keeper is required for operator authorization")
	}
	if err := ms.delegationKeeper.CheckOperatorAuthorization(
		ctx,
		msg.Corporation,
		msg.Operator,
		"/verana.pp.v1.MsgSetParticipantOPToValidated",
		now,
	); err != nil {
		return nil, fmt.Errorf("authorization check failed: %w", err)
	}

	// [MOD-PERM-MSG-3-2-1] Basic checks
	// Load Participant entry applicant_perm from id. If no entry found, abort.
	applicantPerm, err := ms.Keeper.GetParticipantByID(ctx, msg.Id)
	if err != nil {
		return nil, fmt.Errorf("perm not found: %w", err)
	}

	// applicant_perm.op_state MUST be equal to PENDING, else abort.
	if applicantPerm.OpState != types.OnboardingState_PENDING {
		return nil, fmt.Errorf("perm must be in PENDING state to be validated")
	}

	// If applicant_perm.effective_from is not null (renewal) validation_fees MUST be equal to applicant_perm.validation_fees
	if applicantPerm.EffectiveFrom != nil && msg.ValidationFees != applicantPerm.ValidationFees {
		return nil, fmt.Errorf("validation_fees cannot be changed during renewal")
	}

	// If applicant_perm.effective_from is not null (renewal) issuance_fees MUST be equal to applicant_perm.issuance_fees
	if applicantPerm.EffectiveFrom != nil && msg.IssuanceFees != applicantPerm.IssuanceFees {
		return nil, fmt.Errorf("issuance_fees cannot be changed during renewal")
	}

	// If applicant_perm.effective_from is not null (renewal) verification_fees MUST be equal to applicant_perm.verification_fees
	if applicantPerm.EffectiveFrom != nil && msg.VerificationFees != applicantPerm.VerificationFees {
		return nil, fmt.Errorf("verification_fees cannot be changed during renewal")
	}

	// op_summary_digest_sri: MUST be null if validation.type is set to HOLDER
	if applicantPerm.Role == types.ParticipantRole_HOLDER && msg.OpSummaryDigest != "" {
		return nil, fmt.Errorf("op_summary_digest must be null for HOLDER type")
	}

	// Load CredentialSchema cs from applicant_perm.schema_id.
	cs, err := ms.credentialSchemaKeeper.GetCredentialSchemaById(ctx, applicantPerm.SchemaId)
	if err != nil {
		return nil, fmt.Errorf("credential schema not found: %w", err)
	}

	// [MOD-PERM-MSG-3-2-1] Validate issuance_fee_discount
	// Load validator_perm early for discount validation
	validatorPerm, err := ms.Keeper.GetParticipantByID(ctx, applicantPerm.ValidatorParticipantId)
	if err != nil {
		return nil, fmt.Errorf("validator perm not found: %w", err)
	}

	const maxDiscount = 10000 // 10000 = 1.0 = 100% discount

	// If renewal, discount must equal existing discount
	if applicantPerm.EffectiveFrom != nil {
		if msg.IssuanceFeeDiscount != applicantPerm.IssuanceFeeDiscount {
			return nil, fmt.Errorf("issuance_fee_discount cannot be changed during renewal")
		}
		if msg.VerificationFeeDiscount != applicantPerm.VerificationFeeDiscount {
			return nil, fmt.Errorf("verification_fee_discount cannot be changed during renewal")
		}
	} else {
		// First time validation - validate discount range and applicability
		// Validate issuance_fee_discount
		if msg.IssuanceFeeDiscount > maxDiscount {
			return nil, fmt.Errorf("issuance_fee_discount cannot exceed %d (100%% discount)", maxDiscount)
		}

		// Only validate applicability if discount > 0 (0 is always allowed as default)
		if msg.IssuanceFeeDiscount > 0 {
			if cs.IssuerOnboardingMode == credentialschematypes.IssuerOnboardingMode_ISSUER_ONBOARDING_MODE_GRANTOR_VALIDATION_PROCESS {
				if applicantPerm.Role == types.ParticipantRole_ISSUER_GRANTOR {
					// ISSUER_GRANTOR: can set 0-1 (100% discount)
					// Already validated range above
				} else if applicantPerm.Role == types.ParticipantRole_ISSUER {
					// ISSUER in GRANTOR mode: if validator_perm.issuance_fee_discount is defined,
					// can only set 0 to validator_perm.issuance_fee_discount inclusive
					if validatorPerm.IssuanceFeeDiscount > 0 {
						if msg.IssuanceFeeDiscount > validatorPerm.IssuanceFeeDiscount {
							return nil, fmt.Errorf("issuance_fee_discount cannot exceed validator's discount of %d", validatorPerm.IssuanceFeeDiscount)
						}
					}
				} else {
					return nil, fmt.Errorf("issuance_fee_discount can only be set on ISSUER_GRANTOR or ISSUER permissions in GRANTOR mode")
				}
			} else if cs.IssuerOnboardingMode == credentialschematypes.IssuerOnboardingMode_ISSUER_ONBOARDING_MODE_ECOSYSTEM_VALIDATION_PROCESS {
				if applicantPerm.Role == types.ParticipantRole_ISSUER {
					// ISSUER in ECOSYSTEM mode: can set 0-1 (100% discount)
					// Already validated range above
				} else {
					return nil, fmt.Errorf("issuance_fee_discount can only be set on ISSUER permissions in ECOSYSTEM mode")
				}
			} else {
				// OPEN mode or other - issuance_fee_discount not applicable
				return nil, fmt.Errorf("issuance_fee_discount cannot be set in this permission management mode")
			}
		}

		// Validate verification_fee_discount
		if msg.VerificationFeeDiscount > maxDiscount {
			return nil, fmt.Errorf("verification_fee_discount cannot exceed %d (100%% discount)", maxDiscount)
		}

		// Only validate applicability if discount > 0 (0 is always allowed as default)
		if msg.VerificationFeeDiscount > 0 {
			if cs.VerifierOnboardingMode == credentialschematypes.VerifierOnboardingMode_VERIFIER_ONBOARDING_MODE_GRANTOR_VALIDATION_PROCESS {
				if applicantPerm.Role == types.ParticipantRole_VERIFIER_GRANTOR {
					// VERIFIER_GRANTOR: can set 0-1 (100% discount)
					// Already validated range above
				} else if applicantPerm.Role == types.ParticipantRole_VERIFIER {
					// VERIFIER in GRANTOR mode: if validator_perm.verification_fee_discount is defined,
					// can only set 0 to validator_perm.verification_fee_discount inclusive
					if validatorPerm.VerificationFeeDiscount > 0 {
						if msg.VerificationFeeDiscount > validatorPerm.VerificationFeeDiscount {
							return nil, fmt.Errorf("verification_fee_discount cannot exceed validator's discount of %d", validatorPerm.VerificationFeeDiscount)
						}
					}
				} else {
					return nil, fmt.Errorf("verification_fee_discount can only be set on VERIFIER_GRANTOR or VERIFIER permissions in GRANTOR mode")
				}
			} else if cs.VerifierOnboardingMode == credentialschematypes.VerifierOnboardingMode_VERIFIER_ONBOARDING_MODE_ECOSYSTEM_VALIDATION_PROCESS {
				if applicantPerm.Role == types.ParticipantRole_VERIFIER {
					// VERIFIER in ECOSYSTEM mode: can set 0-1 (100% discount)
					// Already validated range above
				} else {
					return nil, fmt.Errorf("verification_fee_discount can only be set on VERIFIER permissions in ECOSYSTEM mode")
				}
			} else {
				// OPEN mode or other - verification_fee_discount not applicable
				return nil, fmt.Errorf("verification_fee_discount cannot be set in this permission management mode")
			}
		}
	}

	// [MOD-PERM-MSG-3-2-1] Calculate op_exp
	validityPeriod := getValidityPeriod(uint32(applicantPerm.Role), cs)
	var vpExp *time.Time
	if validityPeriod == 0 {
		vpExp = nil
	} else if applicantPerm.OpExp == nil {
		exp := now.AddDate(0, 0, int(validityPeriod))
		vpExp = &exp
	} else {
		exp := applicantPerm.OpExp.AddDate(0, 0, int(validityPeriod))
		vpExp = &exp
	}

	// [MOD-PERM-MSG-3-2-1] Verify effective_until and resolve its final value
	effectiveUntil := msg.EffectiveUntil
	if effectiveUntil == nil {
		// if provided effective_until is NULL: change value to op_exp
		effectiveUntil = vpExp
	} else if applicantPerm.EffectiveUntil == nil {
		// effective_until MUST be greater than current timestamp
		if !effectiveUntil.After(now) {
			return nil, fmt.Errorf("effective_until must be greater than current timestamp")
		}
		// if op_exp is not null, effective_until MUST be lower or equal to op_exp
		if vpExp != nil && effectiveUntil.After(*vpExp) {
			return nil, fmt.Errorf("effective_until must be lower or equal to op_exp")
		}
	} else {
		// effective_until MUST be greater than applicant_perm.effective_until
		if !effectiveUntil.After(*applicantPerm.EffectiveUntil) {
			return nil, fmt.Errorf("effective_until must be greater than current effective_until")
		}
		// if op_exp is not null, effective_until MUST be lower or equal to op_exp
		if vpExp != nil && effectiveUntil.After(*vpExp) {
			return nil, fmt.Errorf("effective_until must be lower or equal to op_exp")
		}
	}

	// [MOD-PERM-MSG-3-2-2] Validator perms
	// validator_perm MUST be an active permission
	if err := IsValidPermission(validatorPerm, now); err != nil {
		return nil, fmt.Errorf("validator perm is not valid: %w", err)
	}

	// authority running the method MUST be validator_perm.authority
	validatorCorpAcct, err := ms.corpAccountFromID(ctx, validatorPerm.CorporationId)
	if err != nil {
		return nil, err
	}
	if validatorCorpAcct != msg.Corporation {
		return nil, fmt.Errorf("authority must be validator participant authority")
	}

	// [MOD-PERM-MSG-3-2-4] Overlap checks (use resolved effectiveUntil)
	if err := ms.checkValidatedOverlap(ctx, applicantPerm, effectiveUntil); err != nil {
		return nil, fmt.Errorf("overlap check failed: %w", err)
	}

	// [MOD-PERM-MSG-3-3] Execution
	return ms.executeSetPermissionVPToValidated(ctx, applicantPerm, validatorPerm, msg, now, vpExp, effectiveUntil)
}

func (ms msgServer) CancelParticipantOPLastRequest(goCtx context.Context, msg *types.MsgCancelParticipantOPLastRequest) (*types.MsgCancelParticipantOPLastRequestResponse, error) {
	ctx := sdk.UnwrapSDKContext(goCtx)
	now := ctx.BlockTime()

	// [MOD-PERM-MSG-6-2-1] [AUTHZ-CHECK] Verify operator authorization
	if ms.delegationKeeper == nil {
		return nil, fmt.Errorf("delegation keeper is required for operator authorization")
	}
	if err := ms.delegationKeeper.CheckOperatorAuthorization(
		ctx,
		msg.Corporation,
		msg.Operator,
		"/verana.pp.v1.MsgCancelParticipantOPLastRequest",
		now,
	); err != nil {
		return nil, fmt.Errorf("authorization check failed: %w", err)
	}

	// [MOD-PERM-MSG-6-2-1] Load Participant entry applicant_perm from id
	applicantPerm, err := ms.Keeper.GetParticipantByID(ctx, msg.Id)
	if err != nil {
		return nil, fmt.Errorf("perm not found: %w", err)
	}

	// authority running the transaction MUST be applicant_perm.authority
	applicantCorpAcct, err := ms.corpAccountFromID(ctx, applicantPerm.CorporationId)
	if err != nil {
		return nil, err
	}
	if applicantCorpAcct != msg.Corporation {
		return nil, fmt.Errorf("authority is not the participant authority")
	}

	// applicant_perm.op_state MUST be PENDING
	if applicantPerm.OpState != types.OnboardingState_PENDING {
		return nil, fmt.Errorf("perm must be in PENDING state")
	}

	// if applicant_perm.deposit has been slashed and not repaid, MUST abort
	if applicantPerm.Slashed != nil && applicantPerm.Repaid == nil {
		return nil, fmt.Errorf("permission deposit has been slashed and not repaid")
	}

	// [MOD-PERM-MSG-6-3] Execution
	if err := ms.executeCancelPermissionVPLastRequest(ctx, applicantPerm); err != nil {
		return nil, fmt.Errorf("failed to execute VP cancellation: %w", err)
	}

	ctx.EventManager().EmitEvents(sdk.Events{
		sdk.NewEvent(
			types.EventTypeCancelParticipantOPLastRequest,
			sdk.NewAttribute(types.AttributeKeyParticipantID, strconv.FormatUint(msg.Id, 10)),
			sdk.NewAttribute(types.AttributeKeyCorporation, msg.Corporation),
			sdk.NewAttribute(types.AttributeKeyOperator, msg.Operator),
			sdk.NewAttribute(types.AttributeKeyTimestamp, now.String()),
		),
	})

	return &types.MsgCancelParticipantOPLastRequestResponse{}, nil
}

func (ms msgServer) executeCancelPermissionVPLastRequest(ctx sdk.Context, perm types.Participant) error {
	now := ctx.BlockTime()

	corpAcct, err := ms.corpAccountFromID(ctx, perm.CorporationId)
	if err != nil {
		return err
	}

	// Update basic fields
	perm.Modified = &now
	perm.OpLastStateChange = &now

	// [MOD-PERM-MSG-6-3] spec v4 draft 13:
	//   if op_exp is null (validation never completed), set op_state to TERMINATED
	//   else set op_state to VALIDATED.
	if perm.OpExp == nil {
		perm.OpState = types.OnboardingState_TERMINATED
	} else {
		perm.OpState = types.OnboardingState_VALIDATED
	}

	// Handle current fees if any
	if perm.OpCurrentFees > 0 {
		// Transfer escrowed fees back to the applicant
		granteeAddr, err := sdk.AccAddressFromBech32(corpAcct)
		if err != nil {
			return fmt.Errorf("invalid grantee address: %w", err)
		}

		currentFeesI64, err := uint64ToInt64(perm.OpCurrentFees, "op_current_fees")
		if err != nil {
			return err
		}
		// Transfer fees from module escrow account to applicant account
		err = ms.bankKeeper.SendCoinsFromModuleToAccount(
			ctx,
			types.ModuleName, // Module escrow account
			granteeAddr,      // Applicant account
			sdk.NewCoins(sdk.NewInt64Coin(types.BondDenom, currentFeesI64)),
		)
		if err != nil {
			return fmt.Errorf("failed to refund fees: %w", err)
		}

		perm.OpCurrentFees = 0
	}

	// Handle current deposit if any
	if perm.OpCurrentDeposit > 0 {
		currentDepositI64, err := uint64ToInt64(perm.OpCurrentDeposit, "op_current_deposit")
		if err != nil {
			return err
		}
		// Use AdjustTrustDeposit to reduce trust deposit with negative value
		// to move funds from deposit to claimable
		if err := ms.trustDeposit.AdjustTrustDeposit(
			ctx,
			corpAcct,
			-currentDepositI64, // Negative value to reduce deposit and increase claimable
			"perm_deactivate_release_deposit",
		); err != nil {
			return fmt.Errorf("failed to adjust trust deposit: %w", err)
		}

		perm.OpCurrentDeposit = 0
	}

	// Persist changes
	return ms.Keeper.UpdatePermission(ctx, perm)
}

func (ms msgServer) CreateRootParticipant(goCtx context.Context, msg *types.MsgCreateRootParticipant) (*types.MsgCreateRootParticipantResponse, error) {
	ctx := sdk.UnwrapSDKContext(goCtx)
	now := ctx.BlockTime()

	// [MOD-PERM-MSG-7-2-1] [AUTHZ-CHECK] Verify operator authorization
	if ms.delegationKeeper == nil {
		return nil, fmt.Errorf("delegation keeper is required for operator authorization")
	}
	if err := ms.delegationKeeper.CheckOperatorAuthorization(
		ctx,
		msg.Corporation,
		msg.Operator,
		"/verana.pp.v1.MsgCreateRootParticipant",
		now,
	); err != nil {
		return nil, fmt.Errorf("authorization check failed: %w", err)
	}

	// [MOD-PERM-MSG-7-2-1] Create Root Participant basic checks
	if err := ms.validateCreateRootPermissionBasicChecks(ctx, msg, now); err != nil {
		return nil, err
	}

	// [MOD-PERM-MSG-7-2-2] Participant checks
	if err := ms.validateCreateRootPermissionAuthority(ctx, msg); err != nil {
		return nil, err
	}

	// [MOD-PERM-MSG-7-2-4] Overlap checks
	if err := ms.checkCreateRootPermissionOverlap(ctx, msg); err != nil {
		return nil, fmt.Errorf("overlap check failed: %w", err)
	}

	// [MOD-PERM-MSG-7-3] Execution
	id, err := ms.executeCreateRootPermission(ctx, msg, now)
	if err != nil {
		return nil, fmt.Errorf("failed to create root perm: %w", err)
	}

	ctx.EventManager().EmitEvents(sdk.Events{
		sdk.NewEvent(
			types.EventTypeCreateRootParticipant,
			sdk.NewAttribute(types.AttributeKeyRootParticipantID, strconv.FormatUint(id, 10)),
			sdk.NewAttribute(types.AttributeKeySchemaID, strconv.FormatUint(msg.SchemaId, 10)),
			sdk.NewAttribute(types.AttributeKeyCorporation, msg.Corporation),
			sdk.NewAttribute(types.AttributeKeyOperator, msg.Operator),
			sdk.NewAttribute(types.AttributeKeyEffectiveFrom, formatTimePtr(msg.EffectiveFrom)),
			sdk.NewAttribute(types.AttributeKeyEffectiveUntil, formatTimePtr(msg.EffectiveUntil)),
			sdk.NewAttribute(types.AttributeKeyValidationFees, strconv.FormatUint(msg.ValidationFees, 10)),
			sdk.NewAttribute(types.AttributeKeyIssuanceFees, strconv.FormatUint(msg.IssuanceFees, 10)),
			sdk.NewAttribute(types.AttributeKeyVerificationFees, strconv.FormatUint(msg.VerificationFees, 10)),
			sdk.NewAttribute(types.AttributeKeyTimestamp, now.String()),
		),
	})

	return &types.MsgCreateRootParticipantResponse{
		Id: id,
	}, nil
}

// [MOD-PERM-MSG-7-2-1] Create Root Participant basic checks
func (ms msgServer) validateCreateRootPermissionBasicChecks(ctx sdk.Context, msg *types.MsgCreateRootParticipant, now time.Time) error {
	// schema_id MUST be a valid uint64 and a credential schema entry with this id MUST exist
	_, err := ms.credentialSchemaKeeper.GetCredentialSchemaById(ctx, msg.SchemaId)
	if err != nil {
		return fmt.Errorf("credential schema not found: %w", err)
	}

	// effective_from is mandatory and must be in the future
	if msg.EffectiveFrom == nil {
		return fmt.Errorf("effective_from is required")
	}
	if !msg.EffectiveFrom.After(now) {
		return fmt.Errorf("effective_from must be in the future")
	}

	// effective_until, if not null, must be greater than effective_from
	if msg.EffectiveUntil != nil && msg.EffectiveFrom != nil {
		if !msg.EffectiveUntil.After(*msg.EffectiveFrom) {
			return fmt.Errorf("effective_until must be greater than effective_from")
		}
	}

	return nil
}

// [MOD-PERM-MSG-7-2-2] Create Root Perm permission checks
func (ms msgServer) validateCreateRootPermissionAuthority(ctx sdk.Context, msg *types.MsgCreateRootParticipant) error {
	// Get credential schema
	cs, err := ms.credentialSchemaKeeper.GetCredentialSchemaById(ctx, msg.SchemaId)
	if err != nil {
		return fmt.Errorf("credential schema not found: %w", err)
	}

	// Load ecosystem and verify signing corporation controls it
	ec, err := ms.ecosystemKeeper.GetEcosystem(ctx, cs.EcosystemId)
	if err != nil {
		return fmt.Errorf("ecosystem not found: %w", err)
	}
	co, ok := ms.coKeeper.ResolveByPolicyAddress(ctx, msg.Corporation)
	if !ok {
		return fmt.Errorf("signing corporation not registered")
	}
	if ec.CorporationId != co.Id {
		return fmt.Errorf("corporation does not control the ecosystem")
	}

	return nil
}

// [MOD-PERM-MSG-7-2-4] Create Root Participant overlap checks.
// Spec v4 draft 13: find all active permissions (not revoked, not slashed,
// not repaid) for (schema_id, ECOSYSTEM, corporation). Unlike other overlap
// checks, validator_participant_id is not checked because ECOSYSTEM permissions
// always have validator_participant_id = NULL.
func (ms msgServer) checkCreateRootPermissionOverlap(ctx sdk.Context, msg *types.MsgCreateRootParticipant) error {
	msgCorpId, err := ms.corpIDFromAccount(ctx, msg.Corporation)
	if err != nil {
		return err
	}
	err = ms.Participant.Walk(ctx, nil, func(key uint64, perm types.Participant) (bool, error) {
		// Match on schema_id, ECOSYSTEM role, and corporation.
		if perm.SchemaId != msg.SchemaId ||
			perm.Role != types.ParticipantRole_ECOSYSTEM ||
			perm.CorporationId != msgCorpId {
			return false, nil
		}

		// Skip non-active permissions (revoked, slashed, or repaid)
		if perm.Revoked != nil || perm.Slashed != nil || perm.Repaid != nil {
			return false, nil
		}

		// if p.effective_until is NULL (never expire), abort
		if perm.EffectiveUntil == nil {
			return true, fmt.Errorf("existing permission %d never expires, cannot create new permission", perm.Id)
		}

		// if p.effective_until is greater than effective_from, abort
		if perm.EffectiveUntil.After(*msg.EffectiveFrom) {
			return true, fmt.Errorf("existing permission %d overlaps: its effective_until is after the new effective_from", perm.Id)
		}

		// if p.effective_from is lower than effective_until, abort
		if msg.EffectiveUntil != nil && perm.EffectiveFrom != nil && perm.EffectiveFrom.Before(*msg.EffectiveUntil) {
			return true, fmt.Errorf("existing permission %d overlaps: its effective_from is before the new effective_until", perm.Id)
		}

		return false, nil
	})
	return err
}

// [MOD-PERM-MSG-7-3] Create Root Participant execution
// Spec v4 draft 13: perm.type is hardcoded to ECOSYSTEM. vs_operator is not
// set on root permissions; only on perms created via StartParticipantOP or
// SelfCreateParticipant.
func (ms msgServer) executeCreateRootPermission(ctx sdk.Context, msg *types.MsgCreateRootParticipant, now time.Time) (uint64, error) {
	corporationId, err := ms.corpIDFromAccount(ctx, msg.Corporation)
	if err != nil {
		return 0, err
	}
	perm := types.Participant{
		SchemaId:         msg.SchemaId,
		Modified:         &now,
		Role:             types.ParticipantRole_ECOSYSTEM,
		Did:              msg.Did,
		CorporationId:    corporationId,
		Created:          &now,
		EffectiveFrom:    msg.EffectiveFrom,
		EffectiveUntil:   msg.EffectiveUntil,
		ValidationFees:   msg.ValidationFees,
		IssuanceFees:     msg.IssuanceFees,
		VerificationFees: msg.VerificationFees,
		Deposit:          0,
	}

	id, err := ms.Keeper.CreatePermission(ctx, perm)
	if err != nil {
		return 0, fmt.Errorf("failed to create perm: %w", err)
	}

	return id, nil
}

func (ms msgServer) SetParticipantEffectiveUntil(goCtx context.Context, msg *types.MsgSetParticipantEffectiveUntil) (*types.MsgSetParticipantEffectiveUntilResponse, error) {
	ctx := sdk.UnwrapSDKContext(goCtx)
	now := ctx.BlockTime()

	// [MOD-PERM-MSG-8-2-1] [AUTHZ-CHECK] Verify operator authorization
	if ms.delegationKeeper == nil {
		return nil, fmt.Errorf("delegation keeper is required for operator authorization")
	}
	if err := ms.delegationKeeper.CheckOperatorAuthorization(
		ctx,
		msg.Corporation,
		msg.Operator,
		"/verana.pp.v1.MsgSetParticipantEffectiveUntil",
		now,
	); err != nil {
		return nil, fmt.Errorf("authorization check failed: %w", err)
	}

	// [MOD-PERM-MSG-8-2-1] Adjust Participant basic checks
	applicantPerm, err := ms.validateAdjustPermissionBasicChecks(ctx, msg, now)
	if err != nil {
		return nil, err
	}

	// [MOD-PERM-MSG-8-2-2] Adjust Participant advanced checks
	if err := ms.validateAdjustPermissionAdvancedChecks(ctx, msg, applicantPerm, now); err != nil {
		return nil, err
	}

	// [MOD-PERM-MSG-8-2-4] Overlap checks
	if err := ms.checkAdjustPermissionOverlap(ctx, applicantPerm, msg.EffectiveUntil); err != nil {
		return nil, fmt.Errorf("overlap check failed: %w", err)
	}

	// [MOD-PERM-MSG-8-3] Adjust Participant execution
	if err := ms.executeAdjustPermission(ctx, applicantPerm, msg.EffectiveUntil, now); err != nil {
		return nil, fmt.Errorf("failed to adjust perm: %w", err)
	}

	// [MOD-PERM-MSG-8-3] If applicant_perm.type is ISSUER or VERIFIER and vs_operator_authz_enabled:
	// Grant VS Operator Authorization
	if (applicantPerm.Role == types.ParticipantRole_ISSUER || applicantPerm.Role == types.ParticipantRole_VERIFIER) &&
		applicantPerm.VsOperatorAuthzEnabled {
		if err := ms.grantVSOperatorAuthorization(ctx, applicantPerm); err != nil {
			return nil, fmt.Errorf("failed to grant VS operator authorization: %w", err)
		}
	}

	ctx.EventManager().EmitEvents(sdk.Events{
		sdk.NewEvent(
			types.EventTypeSetParticipantEffectiveUntil,
			sdk.NewAttribute(types.AttributeKeyParticipantID, strconv.FormatUint(msg.Id, 10)),
			sdk.NewAttribute(types.AttributeKeyCorporation, msg.Corporation),
			sdk.NewAttribute(types.AttributeKeyOperator, msg.Operator),
			sdk.NewAttribute(types.AttributeKeyNewEffectiveUntil, msg.EffectiveUntil.String()),
			sdk.NewAttribute(types.AttributeKeyTimestamp, now.String()),
		),
	})

	return &types.MsgSetParticipantEffectiveUntilResponse{}, nil
}

// [MOD-PERM-MSG-8-2-1] Adjust Participant basic checks
func (ms msgServer) validateAdjustPermissionBasicChecks(ctx sdk.Context, msg *types.MsgSetParticipantEffectiveUntil, now time.Time) (types.Participant, error) {
	var applicantPerm types.Participant

	// id MUST be a valid uint64 (already validated in ValidateBasic)

	// Load Participant entry applicant_perm from id. If no entry found, abort
	perm, err := ms.Keeper.GetParticipantByID(ctx, msg.Id)
	if err != nil {
		return applicantPerm, fmt.Errorf("permission not found: %w", err)
	}
	applicantPerm = perm

	// applicant_perm MUST be a valid permission
	if err := IsValidPermission(applicantPerm, now); err != nil {
		return applicantPerm, fmt.Errorf("applicant permission is not valid: %w", err)
	}

	// [MOD-PERM-MSG-8-2-1] effective_until MUST be greater than now()
	if !msg.EffectiveUntil.After(now) {
		return applicantPerm, fmt.Errorf("effective_until must be greater than current timestamp")
	}

	return applicantPerm, nil
}

// [MOD-PERM-MSG-8-2-2] Adjust Participant advanced checks
func (ms msgServer) validateAdjustPermissionAdvancedChecks(ctx sdk.Context, msg *types.MsgSetParticipantEffectiveUntil, applicantPerm types.Participant, now time.Time) error {
	msgCorpId, err := ms.corpIDFromAccount(ctx, msg.Corporation)
	if err != nil {
		return err
	}
	// 1. ECOSYSTEM permissions
	if applicantPerm.ValidatorParticipantId == 0 && applicantPerm.Role == types.ParticipantRole_ECOSYSTEM {
		// applicant_perm.authority MUST be msg.Corporation else MUST abort
		if applicantPerm.CorporationId != msgCorpId {
			return fmt.Errorf("authority is not the participant authority")
		}
		return nil
	}

	// For permissions with validator_participant_id, we need to distinguish between cases 2 and 3
	if applicantPerm.ValidatorParticipantId != 0 {
		// Load validator_perm from applicant_perm.validator_participant_id
		validatorPerm, err := ms.Keeper.GetParticipantByID(ctx, applicantPerm.ValidatorParticipantId)
		if err != nil {
			return fmt.Errorf("validator permission not found: %w", err)
		}

		// validator_perm MUST be a valid permission
		if err := IsValidPermission(validatorPerm, now); err != nil {
			return fmt.Errorf("validator permission is not valid: %w", err)
		}

		// 2. Self-created permissions (validator is ECOSYSTEM)
		if validatorPerm.Role == types.ParticipantRole_ECOSYSTEM {
			// applicant_perm.authority MUST be msg.Corporation else MUST abort
			if applicantPerm.CorporationId != msgCorpId {
				return fmt.Errorf("authority is not the participant authority")
			}
			return nil
		}

		// 3. VP managed permissions
		// effective_until MUST be lower or equal to applicant_perm.op_exp else MUST abort
		if applicantPerm.OpExp != nil && msg.EffectiveUntil.After(*applicantPerm.OpExp) {
			return fmt.Errorf("effective_until cannot be after validation expiration")
		}

		// validator_perm.authority MUST be msg.Corporation else MUST abort
		if validatorPerm.CorporationId != msgCorpId {
			return fmt.Errorf("authority is not the validator participant authority")
		}
		return nil
	}

	return fmt.Errorf("invalid permission configuration for adjustment")
}

// [MOD-PERM-MSG-8-2-4] Overlap checks for SetParticipantEffectiveUntil
// Walk all permissions for same (schema_id, type, validator_participant_id, authority),
// skipping self and inactive (revoked/slashed/repaid).
func (ms msgServer) checkAdjustPermissionOverlap(ctx sdk.Context, applicantPerm types.Participant, effectiveUntil *time.Time) error {
	err := ms.Participant.Walk(ctx, nil, func(key uint64, perm types.Participant) (bool, error) {
		// Skip self
		if perm.Id == applicantPerm.Id {
			return false, nil
		}

		// Match on schema_id, role, validator_participant_id, corporation
		if perm.SchemaId != applicantPerm.SchemaId ||
			perm.Role != applicantPerm.Role ||
			perm.ValidatorParticipantId != applicantPerm.ValidatorParticipantId ||
			perm.CorporationId != applicantPerm.CorporationId {
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

		// [MOD-PERM-MSG-8-2-4] if p.effective_until is NULL (never expire), abort
		if perm.EffectiveUntil == nil {
			return true, fmt.Errorf("existing permission %d never expires, cannot create overlapping permission", perm.Id)
		}

		// if p.effective_until > applicant_perm.effective_from, abort
		if applicantPerm.EffectiveFrom != nil && perm.EffectiveUntil.After(*applicantPerm.EffectiveFrom) {
			return true, fmt.Errorf("existing permission %d overlaps: its effective_until is after this permission's effective_from", perm.Id)
		}

		// if p.effective_from < msg.effective_until, abort
		if effectiveUntil != nil && perm.EffectiveFrom.Before(*effectiveUntil) {
			return true, fmt.Errorf("existing permission %d overlaps: its effective_from is before the requested effective_until", perm.Id)
		}

		return false, nil
	})
	if err != nil {
		return err
	}
	return nil
}

// [MOD-PERM-MSG-8-3] Adjust Participant execution
func (ms msgServer) executeAdjustPermission(ctx sdk.Context, perm types.Participant, effectiveUntil *time.Time, now time.Time) error {
	// set applicant_perm.effective_until to effective_until
	perm.EffectiveUntil = effectiveUntil

	// set applicant_perm.adjusted to now
	perm.Adjusted = &now

	// set applicant_perm.modified to now
	perm.Modified = &now

	return ms.Keeper.UpdatePermission(ctx, perm)
}

// RevokeParticipant handles the MsgRevokeParticipant message
func (ms msgServer) RevokeParticipant(goCtx context.Context, msg *types.MsgRevokeParticipant) (*types.MsgRevokeParticipantResponse, error) {
	ctx := sdk.UnwrapSDKContext(goCtx)
	now := ctx.BlockTime()

	// [MOD-PERM-MSG-9-2-1] [AUTHZ-CHECK] Verify operator authorization
	if ms.delegationKeeper == nil {
		return nil, fmt.Errorf("delegation keeper is required for operator authorization")
	}
	if err := ms.delegationKeeper.CheckOperatorAuthorization(
		ctx,
		msg.Corporation,
		msg.Operator,
		"/verana.pp.v1.MsgRevokeParticipant",
		now,
	); err != nil {
		return nil, fmt.Errorf("authorization check failed: %w", err)
	}

	// [MOD-PERM-MSG-9-2-1] Revoke Participant basic checks
	applicantPerm, err := ms.validateRevokePermissionBasicChecks(ctx, msg, now)
	if err != nil {
		return nil, err
	}

	// [MOD-PERM-MSG-9-2-2] Revoke Participant advanced checks
	if err := ms.validateRevokePermissionAdvancedChecks(ctx, msg, applicantPerm, now); err != nil {
		return nil, err
	}

	// [MOD-PERM-MSG-9-2-3] Revoke Participant fee checks
	// Account MUST have the required estimated transaction fees available
	// (This is handled by the SDK automatically during transaction processing)

	// [MOD-PERM-MSG-9-3] Revoke Participant execution
	if err := ms.executeRevokePermission(ctx, applicantPerm, now); err != nil {
		return nil, fmt.Errorf("failed to revoke permission: %w", err)
	}

	// [MOD-PERM-MSG-9-3] If applicant_perm.type is ISSUER or VERIFIER:
	// Delete authorization for applicant_perm.vs_operator
	if applicantPerm.Role == types.ParticipantRole_ISSUER || applicantPerm.Role == types.ParticipantRole_VERIFIER {
		if err := ms.revokeVSOperatorAuthorization(ctx, applicantPerm); err != nil {
			return nil, fmt.Errorf("failed to revoke VS operator authorization: %w", err)
		}
	}

	ctx.EventManager().EmitEvents(sdk.Events{
		sdk.NewEvent(
			types.EventTypeRevokeParticipant,
			sdk.NewAttribute(types.AttributeKeyParticipantID, strconv.FormatUint(msg.Id, 10)),
			sdk.NewAttribute(types.AttributeKeyCorporation, msg.Corporation),
			sdk.NewAttribute(types.AttributeKeyOperator, msg.Operator),
			sdk.NewAttribute(types.AttributeKeyRevokedAt, now.String()),
			sdk.NewAttribute(types.AttributeKeyTimestamp, now.String()),
		),
	})

	return &types.MsgRevokeParticipantResponse{}, nil
}

// [MOD-PERM-MSG-9-2-1] Revoke Participant basic checks
func (ms msgServer) validateRevokePermissionBasicChecks(ctx sdk.Context, msg *types.MsgRevokeParticipant, now time.Time) (types.Participant, error) {
	var applicantPerm types.Participant

	// id MUST be a valid uint64 (already validated in ValidateBasic)

	// Load Participant entry applicant_perm from id. If no entry found, abort
	perm, err := ms.Keeper.GetParticipantByID(ctx, msg.Id)
	if err != nil {
		return applicantPerm, fmt.Errorf("permission not found: %w", err)
	}
	applicantPerm = perm

	// [MOD-PERM-MSG-9-2-1] applicant_perm MUST be a active permission
	if err := IsValidPermission(applicantPerm, now); err != nil {
		return applicantPerm, fmt.Errorf("applicant permission is not active: %w", err)
	}

	return applicantPerm, nil
}

// [MOD-PERM-MSG-9-2-2] Revoke Participant advanced checks
func (ms msgServer) validateRevokePermissionAdvancedChecks(ctx sdk.Context, msg *types.MsgRevokeParticipant, applicantPerm types.Participant, now time.Time) error {
	// Either Option #1, #2 or #3 MUST return true, else abort

	// Option #1: executed by a validator ancestor
	if ms.checkValidatorAncestorOption(ctx, msg.Corporation, applicantPerm, now) {
		return nil
	}

	// Option #2: executed by the controlling Ecosystem (post-MOD-EC rename, was: TrustRegistry controller)
	if ms.checkEcosystemControllerOption(ctx, msg.Corporation, applicantPerm) {
		return nil
	}

	// Option #3: executed by applicant_perm.authority
	if msgCorpId, err := ms.corpIDFromAccount(ctx, msg.Corporation); err == nil && applicantPerm.CorporationId == msgCorpId {
		return nil
	}

	return fmt.Errorf("authority is not authorized to revoke this participant")
}

// Option #1: executed by a validator ancestor
func (ms msgServer) checkValidatorAncestorOption(ctx sdk.Context, authority string, applicantPerm types.Participant, now time.Time) bool {
	// if applicant_perm.validator_participant_id is defined
	if applicantPerm.ValidatorParticipantId == 0 {
		return false
	}

	// set validator_perm = applicant_perm
	// while validator_perm.validator_participant_id is defined
	currentValidatorPermId := applicantPerm.ValidatorParticipantId

	authorityCorpId, err := ms.corpIDFromAccount(ctx, authority)
	if err != nil {
		return false
	}

	for currentValidatorPermId != 0 {
		// load validator_perm from validator_perm.validator_participant_id
		validatorPerm, err := ms.Keeper.GetParticipantByID(ctx, currentValidatorPermId)
		if err != nil {
			return false
		}

		// if validator_perm is a active permission and validator_perm.authority is who is running the method
		if IsValidPermission(validatorPerm, now) == nil &&
			validatorPerm.CorporationId == authorityCorpId {
			return true
		}

		// Move up to the next ancestor
		currentValidatorPermId = validatorPerm.ValidatorParticipantId
	}

	return false
}

// Option #2: executed by TrustRegistry controller
func (ms msgServer) checkEcosystemControllerOption(ctx sdk.Context, authority string, applicantPerm types.Participant) bool {
	// load CredentialSchema cs from applicant_perm.schema_id
	cs, err := ms.credentialSchemaKeeper.GetCredentialSchemaById(ctx, applicantPerm.SchemaId)
	if err != nil {
		return false
	}

	// load Ecosystem ec from cs.ecosystem_id
	ec, err := ms.ecosystemKeeper.GetEcosystem(ctx, cs.EcosystemId)
	if err != nil {
		return false
	}

	// resolve the signing policy_address → co.Id and compare with ec.CorporationId
	co, ok := ms.coKeeper.ResolveByPolicyAddress(ctx, authority)
	if !ok {
		return false
	}
	return ec.CorporationId == co.Id
}

// [MOD-PERM-MSG-9-3] Revoke Participant execution
func (ms msgServer) executeRevokePermission(ctx sdk.Context, perm types.Participant, now time.Time) error {
	// Free associated trust deposit if non-zero
	if perm.Deposit > 0 {
		corpAcct, err := ms.corpAccountFromID(ctx, perm.CorporationId)
		if err != nil {
			return err
		}
		depositI64 := int64(perm.Deposit)
		if err := ms.trustDeposit.AdjustTrustDeposit(ctx, corpAcct, -depositI64, "perm_revoke_release_deposit"); err != nil {
			return fmt.Errorf("failed to release trust deposit on revocation: %w", err)
		}
		perm.Deposit = 0
	}

	// set applicant_perm.revoked to now
	perm.Revoked = &now

	// set applicant_perm.modified to now
	perm.Modified = &now

	return ms.Keeper.UpdatePermission(ctx, perm)
}

// revokeVSOperatorAuthorization implements [MOD-DE-MSG-6] orchestration.
// Called by: RevokeParticipant, SlashParticipantTrustDeposit.
// VSOA storage is in DE module; this method handles the business logic.
func (ms msgServer) revokeVSOperatorAuthorization(ctx sdk.Context, perm types.Participant) error {
	if perm.VsOperator == "" {
		return nil
	}
	if ms.delegationKeeper == nil {
		return fmt.Errorf("delegation keeper is required for VS operator authorization")
	}

	corpAcct, err := ms.corpAccountFromID(ctx, perm.CorporationId)
	if err != nil {
		return err
	}

	// [MOD-DE-MSG-6-4] Remove permission from VSOA
	remainingPerms, err := ms.delegationKeeper.RemovePermFromVSOA(ctx, corpAcct, perm.VsOperator, perm.Id)
	if err != nil {
		return fmt.Errorf("failed to revoke VS operator authorization: %w", err)
	}

	// Handle feegrant recalculation
	if perm.VsOperatorAuthzWithFeegrant {
		if len(remainingPerms) == 0 {
			// No more permissions — revoke fee allowance
			if err := ms.delegationKeeper.RevokeFeeAllowance(ctx, corpAcct, perm.VsOperator); err != nil {
				return fmt.Errorf("failed to revoke fee allowance: %w", err)
			}
		} else {
			// Recalculate feegrant expiration from remaining permissions
			expiration, err := ms.computeVSOAFeegrantExpiration(ctx, corpAcct, perm.VsOperator)
			if err != nil {
				return fmt.Errorf("failed to compute feegrant expiration: %w", err)
			}

			if expiration == nil || expiration.After(ctx.BlockTime()) {
				msgTypes := []string{"/verana.pp.v1.MsgCreateOrUpdateParticipantSession"}
				if err := ms.delegationKeeper.GrantFeeAllowance(ctx, corpAcct, perm.VsOperator, msgTypes, expiration, nil, nil); err != nil {
					return fmt.Errorf("failed to update fee allowance: %w", err)
				}
			}
		}
	}

	return nil
}

func (ms msgServer) CreateOrUpdateParticipantSession(goCtx context.Context, msg *types.MsgCreateOrUpdateParticipantSession) (*types.MsgCreateOrUpdateParticipantSessionResponse, error) {
	ctx := sdk.UnwrapSDKContext(goCtx)
	now := ctx.BlockTime()

	// [MOD-PERM-MSG-10-2] Create or Update Participant Session precondition checks
	if err := ms.validateCreateOrUpdateParticipantSessionPreconditions(ctx, msg, now); err != nil {
		return nil, err
	}

	// [MOD-PERM-MSG-10-3] Create or Update Participant Session fee checks
	foundPermSet, beneficiaryFees, trustFees, err := ms.validateCreateOrUpdateParticipantSessionFees(ctx, msg)
	if err != nil {
		return nil, err
	}

	// [MOD-PERM-MSG-10-4] Create or Update Participant Session execution
	if err := ms.executeCreateOrUpdateParticipantSession(ctx, msg, foundPermSet, beneficiaryFees, trustFees, now); err != nil {
		return nil, fmt.Errorf("failed to create/update permission session: %w", err)
	}

	// Emit events
	ctx.EventManager().EmitEvents(sdk.Events{
		sdk.NewEvent(
			types.EventTypeCreateOrUpdateParticipantSession,
			sdk.NewAttribute(types.AttributeKeyCorporation, msg.Corporation),
			sdk.NewAttribute(types.AttributeKeyOperator, msg.Operator),
			sdk.NewAttribute(types.AttributeKeySessionID, msg.Id),
			sdk.NewAttribute(types.AttributeKeyIssuerParticipantID, strconv.FormatUint(msg.IssuerParticipantId, 10)),
			sdk.NewAttribute(types.AttributeKeyVerifierParticipantID, strconv.FormatUint(msg.VerifierParticipantId, 10)),
			sdk.NewAttribute(types.AttributeKeyAgentParticipantID, strconv.FormatUint(msg.AgentParticipantId, 10)),
			sdk.NewAttribute(types.AttributeKeyWalletAgentParticipantID, strconv.FormatUint(msg.WalletAgentParticipantId, 10)),
			sdk.NewAttribute(types.AttributeKeyTimestamp, now.String()),
		),
	})

	return &types.MsgCreateOrUpdateParticipantSessionResponse{
		Id: msg.Id,
	}, nil
}

// SlashParticipantTrustDeposit handles the MsgSlashParticipantTrustDeposit message
func (ms msgServer) SlashParticipantTrustDeposit(goCtx context.Context, msg *types.MsgSlashParticipantTrustDeposit) (*types.MsgSlashParticipantTrustDepositResponse, error) {
	ctx := sdk.UnwrapSDKContext(goCtx)
	now := ctx.BlockTime()

	// [MOD-PERM-MSG-12-2-1] [AUTHZ-CHECK] Verify operator authorization
	if ms.delegationKeeper == nil {
		return nil, fmt.Errorf("delegation keeper is required for operator authorization")
	}
	if err := ms.delegationKeeper.CheckOperatorAuthorization(
		ctx,
		msg.Corporation,
		msg.Operator,
		"/verana.pp.v1.MsgSlashParticipantTrustDeposit",
		now,
	); err != nil {
		return nil, fmt.Errorf("authorization check failed: %w", err)
	}

	// [MOD-PERM-MSG-12-2-1] Slash Participant Trust Deposit basic checks
	applicantPerm, err := ms.validateSlashPermissionBasicChecks(ctx, msg)
	if err != nil {
		return nil, err
	}

	// [MOD-PERM-MSG-12-2-2] Slash Participant Trust Deposit validator perms
	if err := ms.validateSlashPermissionValidatorPerms(ctx, msg, applicantPerm, now); err != nil {
		return nil, err
	}

	// [MOD-PERM-MSG-12-2-3] Slash Participant Trust Deposit fee checks
	// Account MUST have the required estimated transaction fees available
	// (This is handled by the SDK automatically during transaction processing)

	// [MOD-PERM-MSG-12-3] Slash Participant Trust Deposit execution
	if err := ms.executeSlashPermissionTrustDeposit(ctx, applicantPerm, msg.Amount, now); err != nil {
		return nil, fmt.Errorf("failed to slash permission trust deposit: %w", err)
	}

	// [MOD-PERM-MSG-12-3] If applicant_perm.type is ISSUER or VERIFIER:
	// Delete authorization for applicant_perm.vs_operator
	if applicantPerm.Role == types.ParticipantRole_ISSUER || applicantPerm.Role == types.ParticipantRole_VERIFIER {
		if err := ms.revokeVSOperatorAuthorization(ctx, applicantPerm); err != nil {
			return nil, fmt.Errorf("failed to revoke VS operator authorization: %w", err)
		}
	}

	// Emit event
	ctx.EventManager().EmitEvents(sdk.Events{
		sdk.NewEvent(
			types.EventTypeSlashParticipantTrustDeposit,
			sdk.NewAttribute(types.AttributeKeyParticipantID, strconv.FormatUint(msg.Id, 10)),
			sdk.NewAttribute(types.AttributeKeySlashedAmount, strconv.FormatUint(msg.Amount, 10)),
			sdk.NewAttribute(types.AttributeKeyCorporation, msg.Corporation),
			sdk.NewAttribute(types.AttributeKeyOperator, msg.Operator),
			sdk.NewAttribute("reason", msg.Reason),
			sdk.NewAttribute(types.AttributeKeyTimestamp, now.String()),
		),
	})

	return &types.MsgSlashParticipantTrustDepositResponse{}, nil
}

// [MOD-PERM-MSG-12-2-1] Slash Participant Trust Deposit basic checks
func (ms msgServer) validateSlashPermissionBasicChecks(ctx sdk.Context, msg *types.MsgSlashParticipantTrustDeposit) (types.Participant, error) {
	var applicantPerm types.Participant

	// id MUST be a valid uint64 (already validated in ValidateBasic)

	// Load Participant entry applicant_perm from id. If no entry found, abort
	perm, err := ms.Keeper.GetParticipantByID(ctx, msg.Id)
	if err != nil {
		return applicantPerm, fmt.Errorf("permission not found: %w", err)
	}
	applicantPerm = perm

	// [MOD-PERM-MSG-12-2-1] amount MUST be lower or equal to applicant_perm.deposit else MUST abort
	if msg.Amount > applicantPerm.Deposit {
		return applicantPerm, fmt.Errorf("amount exceeds available deposit: %d > %d", msg.Amount, applicantPerm.Deposit)
	}

	// Note: Even if the permission has expired or is revoked, it is still possible to slash it.

	return applicantPerm, nil
}

// [MOD-PERM-MSG-12-2-2] Slash Participant Trust Deposit validator perms.
// NOTE: Spec v4 draft 13 calls for governance-only slashing. Migrating the test
// surface to the governance-mediated flow is tracked as a follow-up; for now
// we retain the validator-ancestor / TR-controller check established by prior
// implementations so operator-signed slashing remains testable.
func (ms msgServer) validateSlashPermissionValidatorPerms(ctx sdk.Context, msg *types.MsgSlashParticipantTrustDeposit, applicantPerm types.Participant, now time.Time) error {
	// Option #1: executed by a validator ancestor
	if ms.checkValidatorAncestorOption(ctx, msg.Corporation, applicantPerm, now) {
		return nil
	}
	// Option #2: executed by the controlling Ecosystem (post-MOD-EC rename, was: TrustRegistry controller)
	if ms.checkEcosystemControllerOption(ctx, msg.Corporation, applicantPerm) {
		return nil
	}
	return fmt.Errorf("authority is not authorized to slash this permission")
}

// [MOD-PERM-MSG-12-3] Slash Participant Trust Deposit execution
func (ms msgServer) executeSlashPermissionTrustDeposit(ctx sdk.Context, applicantPerm types.Participant, amount uint64, now time.Time) error {
	// Load Participant entry validator_perm from applicant_perm.validator_participant_id
	if applicantPerm.ValidatorParticipantId != 0 {
		_, err := ms.Keeper.GetParticipantByID(ctx, applicantPerm.ValidatorParticipantId)
		if err != nil {
			return fmt.Errorf("validator permission not found: %w", err)
		}
	}

	// set applicant_perm.slashed to now
	applicantPerm.Slashed = &now

	// set applicant_perm.modified to now
	applicantPerm.Modified = &now

	// set applicant_perm.slashed_deposit to applicant_perm.slashed_deposit + amount
	applicantPerm.SlashedDeposit = applicantPerm.SlashedDeposit + amount

	// decrement applicant_perm.deposit by amount
	applicantPerm.Deposit -= amount

	// use MOD-TD-MSG-7 to burn the slashed amount from the trust deposit of applicant_perm.authority
	corpAcct, err := ms.corpAccountFromID(ctx, applicantPerm.CorporationId)
	if err != nil {
		return err
	}
	if err := ms.trustDeposit.BurnEcosystemSlashedTrustDeposit(ctx, corpAcct, amount); err != nil {
		return fmt.Errorf("failed to burn trust deposit: %w", err)
	}

	// Update permission
	if err := ms.Keeper.UpdatePermission(ctx, applicantPerm); err != nil {
		return fmt.Errorf("failed to update permission: %w", err)
	}

	return nil
}

// RepayParticipantSlashedTrustDeposit handles the MsgRepayParticipantSlashedTrustDeposit message
func (ms msgServer) RepayParticipantSlashedTrustDeposit(goCtx context.Context, msg *types.MsgRepayParticipantSlashedTrustDeposit) (*types.MsgRepayParticipantSlashedTrustDepositResponse, error) {
	ctx := sdk.UnwrapSDKContext(goCtx)
	now := ctx.BlockTime()

	// [MOD-PERM-MSG-13-2-1] [AUTHZ-CHECK] Verify operator authorization
	if ms.delegationKeeper == nil {
		return nil, fmt.Errorf("delegation keeper is required for operator authorization")
	}
	if err := ms.delegationKeeper.CheckOperatorAuthorization(ctx, msg.Corporation, msg.Operator, "/verana.pp.v1.MsgRepayParticipantSlashedTrustDeposit", now); err != nil {
		return nil, fmt.Errorf("authorization check failed: %w", err)
	}

	// [MOD-PERM-MSG-13-2-1] Load Participant entry applicant_perm from id
	applicantPerm, err := ms.Keeper.GetParticipantByID(ctx, msg.Id)
	if err != nil {
		return nil, fmt.Errorf("perm not found: %w", err)
	}

	// [MOD-PP-MSG-13-2-1] if applicant_perm.authority is not equal to authority, abort
	applicantCorpAcct, err := ms.corpAccountFromID(ctx, applicantPerm.CorporationId)
	if err != nil {
		return nil, err
	}
	if applicantCorpAcct != msg.Corporation {
		return nil, fmt.Errorf("authority is not the owner of this participant")
	}

	// [MOD-PERM-MSG-13-2] spec v4 draft 13: "MUST abort if permission not exist with slashed not null".
	// Guard on the slashed timestamp (entity-level marker), not on the deposit amount.
	if applicantPerm.Slashed == nil {
		return nil, fmt.Errorf("perm has no slashed timestamp; nothing to repay")
	}

	// Check if already repaid
	if applicantPerm.RepaidDeposit >= applicantPerm.SlashedDeposit {
		return nil, fmt.Errorf("slashed deposit already fully repaid")
	}

	// Check msg.Amount does not exceed outstanding balance
	outstanding := applicantPerm.SlashedDeposit - applicantPerm.RepaidDeposit
	if msg.Amount > outstanding {
		return nil, fmt.Errorf("amount %d exceeds outstanding slashed deposit %d", msg.Amount, outstanding)
	}

	// [MOD-PERM-MSG-13-2-2] authority MUST have at least msg.Amount in its account balance
	authorityAddr, err := sdk.AccAddressFromBech32(msg.Corporation)
	if err != nil {
		return nil, fmt.Errorf("invalid authority address: %w", err)
	}
	repayAmountI64, err := uint64ToInt64(msg.Amount, "amount")
	if err != nil {
		return nil, err
	}
	requiredAmount := sdk.NewInt64Coin(types.BondDenom, repayAmountI64)
	if !ms.bankKeeper.HasBalance(ctx, authorityAddr, requiredAmount) {
		return nil, fmt.Errorf("insufficient funds to repay slashed deposit: required %d", msg.Amount)
	}

	// [MOD-PERM-MSG-13-3] Execution
	// Use AdjustTrustDeposit to transfer msg.Amount to trust deposit of applicant_perm.authority
	if err := ms.trustDeposit.AdjustTrustDeposit(ctx, applicantCorpAcct, repayAmountI64, "perm_repay_slashed_deposit"); err != nil {
		return nil, fmt.Errorf("failed to adjust trust deposit: %w", err)
	}

	// Update Participant entry
	applicantPerm.Modified = &now
	applicantPerm.RepaidDeposit += msg.Amount
	applicantPerm.Deposit += msg.Amount

	// Only mark as fully repaid when repaid_deposit >= slashed_deposit
	if applicantPerm.RepaidDeposit >= applicantPerm.SlashedDeposit {
		applicantPerm.Repaid = &now
	}

	if err := ms.Keeper.UpdatePermission(ctx, applicantPerm); err != nil {
		return nil, fmt.Errorf("failed to update perm: %w", err)
	}

	// Emit event
	ctx.EventManager().EmitEvents(sdk.Events{
		sdk.NewEvent(
			types.EventTypeRepayParticipantSlashedTrustDeposit,
			sdk.NewAttribute(types.AttributeKeyParticipantID, strconv.FormatUint(msg.Id, 10)),
			sdk.NewAttribute(types.AttributeKeyRepaidAmount, strconv.FormatUint(msg.Amount, 10)),
			sdk.NewAttribute(types.AttributeKeyCorporation, msg.Corporation),
			sdk.NewAttribute(types.AttributeKeyOperator, msg.Operator),
			sdk.NewAttribute(types.AttributeKeyTimestamp, ctx.BlockTime().String()),
		),
	})

	return &types.MsgRepayParticipantSlashedTrustDepositResponse{}, nil
}

// SelfCreateParticipant handles the MsgSelfCreateParticipant message
func (ms msgServer) SelfCreateParticipant(goCtx context.Context, msg *types.MsgSelfCreateParticipant) (*types.MsgSelfCreateParticipantResponse, error) {
	ctx := sdk.UnwrapSDKContext(goCtx)
	now := ctx.BlockTime()

	// [MOD-PERM-MSG-14-2-1] [AUTHZ-CHECK] Verify operator authorization
	if ms.delegationKeeper == nil {
		return nil, fmt.Errorf("delegation keeper is required for operator authorization")
	}
	if err := ms.delegationKeeper.CheckOperatorAuthorization(ctx, msg.Corporation, msg.Operator, "/verana.pp.v1.MsgSelfCreateParticipant", now); err != nil {
		return nil, fmt.Errorf("authorization check failed: %w", err)
	}

	// [MOD-PERM-MSG-14-2-1] Load validator_perm from validator_participant_id
	validatorPerm, err := ms.Keeper.GetParticipantByID(ctx, msg.ValidatorParticipantId)
	if err != nil {
		return nil, fmt.Errorf("validator permission not found: %w", err)
	}

	// validator_perm MUST be an ECOSYSTEM active permission or future permission
	if validatorPerm.Role != types.ParticipantRole_ECOSYSTEM {
		return nil, fmt.Errorf("validator_participant_id must reference an ECOSYSTEM permission")
	}
	// Check active or future: not revoked, not slashed, not repaid, not expired
	if validatorPerm.Revoked != nil {
		return nil, fmt.Errorf("validator permission is revoked")
	}
	if validatorPerm.Slashed != nil {
		return nil, fmt.Errorf("validator permission is slashed")
	}
	if validatorPerm.Repaid != nil {
		return nil, fmt.Errorf("validator permission is repaid")
	}
	if validatorPerm.EffectiveUntil != nil && !now.Before(*validatorPerm.EffectiveUntil) {
		return nil, fmt.Errorf("validator permission is expired")
	}

	// [MOD-PERM-MSG-14-2-1] effective_from checks. Spec tags the field "optional"
	// but the precondition "effective_from MUST be in the future" is unconditional,
	// and an "active permission" requires effective_from < now. A nil effective_from
	// creates a permanently-inactive permission that can never be used or repaired
	// (counter slot and overlap slot squatted). To match spec intent, treat a nil
	// effective_from as "now" so the permission becomes active immediately.
	effectiveFrom := msg.EffectiveFrom
	if effectiveFrom == nil {
		t := now
		effectiveFrom = &t
	} else {
		if !effectiveFrom.After(now) {
			return nil, fmt.Errorf("effective_from must be in the future")
		}
		// MUST be >= validator_perm.effective_from
		if validatorPerm.EffectiveFrom != nil && effectiveFrom.Before(*validatorPerm.EffectiveFrom) {
			return nil, fmt.Errorf("effective_from must be >= validator_perm.effective_from")
		}
		// if validator_perm.effective_until is not null, MUST be < validator_perm.effective_until
		if validatorPerm.EffectiveUntil != nil && !effectiveFrom.Before(*validatorPerm.EffectiveUntil) {
			return nil, fmt.Errorf("effective_from must be < validator_perm.effective_until")
		}
	}

	// [MOD-PERM-MSG-14-2-1] effective_until checks
	if msg.EffectiveUntil == nil {
		// if null, validator_perm.effective_until MUST be NULL
		if validatorPerm.EffectiveUntil != nil {
			return nil, fmt.Errorf("effective_until must be set when validator_perm has effective_until")
		}
	} else {
		// must be greater than effective_from
		if !msg.EffectiveUntil.After(*effectiveFrom) {
			return nil, fmt.Errorf("effective_until must be greater than effective_from")
		}
		// if validator_perm.effective_until is not null, MUST be <= validator_perm.effective_until
		if validatorPerm.EffectiveUntil != nil && msg.EffectiveUntil.After(*validatorPerm.EffectiveUntil) {
			return nil, fmt.Errorf("effective_until must be <= validator_perm.effective_until")
		}
	}

	// verification_fees: If specified, MUST be >= 0 and MUST be a ISSUER permission
	if msg.VerificationFees > 0 && msg.Role != types.ParticipantRole_ISSUER {
		return nil, fmt.Errorf("verification_fees can only be specified for ISSUER permissions")
	}
	// validation_fees: If specified, MUST be >= 0 and MUST be a ISSUER permission
	if msg.ValidationFees > 0 && msg.Role != types.ParticipantRole_ISSUER {
		return nil, fmt.Errorf("validation_fees can only be specified for ISSUER permissions")
	}

	// [MOD-PERM-MSG-14-2-2] Participant checks
	cs, err := ms.credentialSchemaKeeper.GetCredentialSchemaById(ctx, validatorPerm.SchemaId)
	if err != nil {
		return nil, fmt.Errorf("credential schema not found: %w", err)
	}

	if msg.Role == types.ParticipantRole_ISSUER {
		if cs.IssuerOnboardingMode != credentialschematypes.IssuerOnboardingMode_ISSUER_ONBOARDING_MODE_OPEN {
			return nil, fmt.Errorf("issuer permission management mode is not OPEN")
		}
	}
	if msg.Role == types.ParticipantRole_VERIFIER {
		if cs.VerifierOnboardingMode != credentialschematypes.VerifierOnboardingMode_VERIFIER_ONBOARDING_MODE_OPEN {
			return nil, fmt.Errorf("verifier permission management mode is not OPEN")
		}
		if msg.ValidationFees > 0 {
			return nil, fmt.Errorf("validation_fees cannot be specified for VERIFIER permissions")
		}
		if msg.VerificationFees > 0 {
			return nil, fmt.Errorf("verification_fees cannot be specified for VERIFIER permissions")
		}
	}

	// [MOD-PERM-MSG-14-2-4] Overlap checks
	if err := ms.checkCreatePermissionOverlap(ctx, validatorPerm.SchemaId, msg, effectiveFrom); err != nil {
		return nil, err
	}

	// [MOD-PP-MSG-14-3] Execution
	corporationId, err := ms.corpIDFromAccount(ctx, msg.Corporation)
	if err != nil {
		return nil, err
	}
	perm := types.Participant{
		ValidatorParticipantId:       msg.ValidatorParticipantId,
		SchemaId:                     validatorPerm.SchemaId,
		Modified:                     &now,
		Role:                         msg.Role,
		Did:                          msg.Did,
		CorporationId:                corporationId,
		VsOperator:                   msg.VsOperator,
		Created:                      &now,
		EffectiveFrom:                effectiveFrom,
		EffectiveUntil:               msg.EffectiveUntil,
		ValidationFees:               0,
		IssuanceFees:                 0,
		VerificationFees:             0,
		Deposit:                      0,
		VsOperatorAuthzEnabled:       msg.VsOperatorAuthzEnabled,
		VsOperatorAuthzSpendLimit:    msg.VsOperatorAuthzSpendLimit,
		VsOperatorAuthzWithFeegrant:  msg.VsOperatorAuthzWithFeegrant,
		VsOperatorAuthzFeeSpendLimit: msg.VsOperatorAuthzFeeSpendLimit,
		VsOperatorAuthzSpendPeriod:   msg.VsOperatorAuthzSpendPeriod,
	}

	// Set fees only for ISSUER permissions as per spec
	if msg.Role == types.ParticipantRole_ISSUER {
		perm.ValidationFees = msg.ValidationFees
		perm.VerificationFees = msg.VerificationFees
	}

	id, err := ms.Keeper.CreatePermission(ctx, perm)
	if err != nil {
		return nil, fmt.Errorf("failed to create permission: %w", err)
	}

	// Grant VS Operator Authorization if vs_operator_authz_enabled
	if perm.VsOperatorAuthzEnabled {
		perm.Id = id
		if err := ms.grantVSOperatorAuthorization(ctx, perm); err != nil {
			return nil, fmt.Errorf("failed to grant VS operator authorization: %w", err)
		}
	}

	// Emit event
	ctx.EventManager().EmitEvents(sdk.Events{
		sdk.NewEvent(
			types.EventTypeCreateParticipant,
			sdk.NewAttribute(types.AttributeKeyParticipantID, strconv.FormatUint(id, 10)),
			sdk.NewAttribute(types.AttributeKeySchemaID, strconv.FormatUint(validatorPerm.SchemaId, 10)),
			sdk.NewAttribute(types.AttributeKeyCorporation, msg.Corporation),
			sdk.NewAttribute(types.AttributeKeyOperator, msg.Operator),
			sdk.NewAttribute(types.AttributeKeyRole, msg.Role.String()),
			sdk.NewAttribute(types.AttributeKeyEffectiveFrom, formatTimePtr(msg.EffectiveFrom)),
			sdk.NewAttribute(types.AttributeKeyEffectiveUntil, formatTimePtr(msg.EffectiveUntil)),
			sdk.NewAttribute(types.AttributeKeyTimestamp, now.String()),
		),
	})

	return &types.MsgSelfCreateParticipantResponse{
		Id: id,
	}, nil
}

// [MOD-PERM-MSG-14-2-4] Overlap checks for SelfCreateParticipant
func (ms msgServer) checkCreatePermissionOverlap(ctx sdk.Context, schemaId uint64, msg *types.MsgSelfCreateParticipant, effectiveFrom *time.Time) error {
	// Find all active permissions (not revoked, not slashed, not repaid)
	// for same cs.id, type, validator_participant_id, authority
	var overlaps []types.Participant
	msgCorpId, err := ms.corpIDFromAccount(ctx, msg.Corporation)
	if err != nil {
		return err
	}
	err = ms.Participant.Walk(ctx, nil, func(id uint64, p types.Participant) (stop bool, err error) {
		if p.SchemaId == schemaId &&
			p.Role == msg.Role &&
			p.ValidatorParticipantId == msg.ValidatorParticipantId &&
			p.CorporationId == msgCorpId &&
			p.Revoked == nil && p.Slashed == nil && p.Repaid == nil {
			overlaps = append(overlaps, p)
		}
		return false, nil
	})
	if err != nil {
		return fmt.Errorf("failed to check permission overlap: %w", err)
	}

	for _, p := range overlaps {
		// if p.effective_until is NULL (never expire), abort
		if p.EffectiveUntil == nil {
			return fmt.Errorf("existing permission %d never expires; adjust it first", p.Id)
		}
		// if p.effective_until is greater than effective_from, abort
		if effectiveFrom != nil && p.EffectiveUntil.After(*effectiveFrom) {
			return fmt.Errorf("existing permission %d overlaps: its effective_until is after your effective_from", p.Id)
		}
		// if p.effective_from is lower than effective_until, abort
		if msg.EffectiveUntil != nil && p.EffectiveFrom != nil && p.EffectiveFrom.Before(*msg.EffectiveUntil) {
			return fmt.Errorf("existing permission %d overlaps: its effective_from is before your effective_until", p.Id)
		}
	}

	return nil
}
