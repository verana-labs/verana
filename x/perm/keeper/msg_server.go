package keeper

import (
	"context"
	"fmt"
	"strconv"
	"time"

	credentialschematypes "github.com/verana-labs/verana/x/cs/types"


	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/verana-labs/verana/x/perm/types"
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

// StartPermissionVP handles the MsgStartPermissionVP message
func (ms msgServer) StartPermissionVP(goCtx context.Context, msg *types.MsgStartPermissionVP) (*types.MsgStartPermissionVPResponse, error) {
	ctx := sdk.UnwrapSDKContext(goCtx)
	now := ctx.BlockTime()

	// [MOD-PERM-MSG-1-2-1] [AUTHZ-CHECK] Verify operator authorization
	if ms.delegationKeeper != nil {
		if err := ms.delegationKeeper.CheckOperatorAuthorization(
			ctx,
			msg.Authority,
			msg.Operator,
			"/verana.perm.v1.MsgStartPermissionVP",
			now,
		); err != nil {
			return nil, fmt.Errorf("authorization check failed: %w", err)
		}
	}

	// [MOD-PERM-MSG-1-2-2] Permission checks
	validatorPerm, err := ms.validatePermissionChecks(ctx, msg)
	if err != nil {
		return nil, fmt.Errorf("perm validation failed: %w", err)
	}

	// [MOD-PERM-MSG-1-2-4] Overlap checks
	if err := ms.checkOverlap(ctx, validatorPerm.SchemaId, msg.Type, msg.ValidatorPermId, msg.Authority); err != nil {
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
			types.EventTypeStartPermissionVP,
			sdk.NewAttribute(types.AttributeKeyPermissionID, strconv.FormatUint(permID, 10)),
			sdk.NewAttribute(types.AttributeKeyAuthority, msg.Authority),
			sdk.NewAttribute(types.AttributeKeyOperator, msg.Operator),
			sdk.NewAttribute(types.AttributeKeyValidatorPermID, strconv.FormatUint(msg.ValidatorPermId, 10)),
			sdk.NewAttribute(types.AttributeKeyType, types.PermissionType(msg.Type).String()),
			sdk.NewAttribute(types.AttributeKeyFees, strconv.FormatUint(fees, 10)),
			sdk.NewAttribute(types.AttributeKeyDeposit, strconv.FormatUint(deposit, 10)),
			sdk.NewAttribute(types.AttributeKeyTimestamp, now.String()),
		),
	})

	return &types.MsgStartPermissionVPResponse{
		PermissionId: permID,
	}, nil
}

func (ms msgServer) RenewPermissionVP(goCtx context.Context, msg *types.MsgRenewPermissionVP) (*types.MsgRenewPermissionVPResponse, error) {
	ctx := sdk.UnwrapSDKContext(goCtx)
	now := ctx.BlockTime()

	// [MOD-PERM-MSG-2-2-1] [AUTHZ-CHECK] Verify operator authorization
	if ms.delegationKeeper != nil {
		if err := ms.delegationKeeper.CheckOperatorAuthorization(
			ctx,
			msg.Authority,
			msg.Operator,
			"/verana.perm.v1.MsgRenewPermissionVP",
			now,
		); err != nil {
			return nil, fmt.Errorf("authorization check failed: %w", err)
		}
	}

	// [MOD-PERM-MSG-2-2-2] Permission checks
	applicantPerm, err := ms.Keeper.GetPermissionByID(ctx, msg.Id)
	if err != nil {
		return nil, fmt.Errorf("perm not found: %w", err)
	}

	// [MOD-PERM-MSG-2-2-2] authority MUST be applicant_perm.authority
	if applicantPerm.Authority != msg.Authority {
		return nil, fmt.Errorf("authority is not the perm authority")
	}

	// [MOD-PERM-MSG-2-2-2] applicant_perm.vp_state MUST be VALIDATED to allow renewal.
	// Renewing a PENDING perm would overwrite vp_current_fees/vp_current_deposit without
	// refunding the escrowed funds, causing permanent fund loss.
	// TERMINATED or TERMINATION_REQUESTED perms cannot be renewed either.
	if applicantPerm.VpState != types.ValidationState_VALIDATED {
		return nil, fmt.Errorf("perm vp_state must be VALIDATED to renew, current state: %s", applicantPerm.VpState.String())
	}

	// Get validator perm
	validatorPerm, err := ms.Keeper.GetPermissionByID(ctx, applicantPerm.ValidatorPermId)
	if err != nil {
		return nil, fmt.Errorf("validator perm not found: %w", err)
	}

	if err := IsValidPermission(validatorPerm, applicantPerm.Country, ctx.BlockTime()); err != nil {
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
			types.EventTypeRenewPermissionVP,
			sdk.NewAttribute(types.AttributeKeyPermissionID, strconv.FormatUint(msg.Id, 10)),
			sdk.NewAttribute(types.AttributeKeyAuthority, msg.Authority),
			sdk.NewAttribute(types.AttributeKeyOperator, msg.Operator),
			sdk.NewAttribute(types.AttributeKeyValidatorPermID, strconv.FormatUint(applicantPerm.ValidatorPermId, 10)),
			sdk.NewAttribute(types.AttributeKeyValidationFees, strconv.FormatUint(validationFees, 10)),
			sdk.NewAttribute(types.AttributeKeyValidationDeposit, strconv.FormatUint(validationDeposit, 10)),
			sdk.NewAttribute(types.AttributeKeyTimestamp, now.String()),
		),
	})

	return &types.MsgRenewPermissionVPResponse{}, nil
}

func (ms msgServer) executeRenewPermissionVP(ctx sdk.Context, perm types.Permission, fees, deposit uint64) error {
	// Increment trust deposit if deposit is greater than 0
	if deposit > 0 {
		if err := ms.trustDeposit.AdjustTrustDeposit(ctx, perm.Authority, int64(deposit)); err != nil {
			return fmt.Errorf("failed to increase trust deposit: %w", err)
		}
	}

	// Send validation fees to escrow account if greater than 0
	if fees > 0 {
		// Get grantee address
		granteeAddr, err := sdk.AccAddressFromBech32(perm.Authority)
		if err != nil {
			return fmt.Errorf("invalid grantee address: %w", err)
		}

		// Transfer fees to module escrow account
		err = ms.bankKeeper.SendCoinsFromAccountToModule(
			ctx,
			granteeAddr,
			types.ModuleName, // Using module name as the escrow account
			sdk.NewCoins(sdk.NewInt64Coin(types.BondDenom, int64(fees))),
		)
		if err != nil {
			return fmt.Errorf("failed to transfer validation fees to escrow: %w", err)
		}
	}

	now := ctx.BlockTime()

	// Update perm
	perm.VpState = types.ValidationState_PENDING
	perm.VpLastStateChange = &now
	perm.Deposit += deposit
	perm.VpCurrentFees = fees
	perm.VpCurrentDeposit = deposit
	perm.Modified = &now

	// Store updated perm
	return ms.Keeper.UpdatePermission(ctx, perm)
}

func (ms msgServer) SetPermissionVPToValidated(goCtx context.Context, msg *types.MsgSetPermissionVPToValidated) (*types.MsgSetPermissionVPToValidatedResponse, error) {
	ctx := sdk.UnwrapSDKContext(goCtx)
	now := ctx.BlockTime()

	// [MOD-PERM-MSG-3-2-1] [AUTHZ-CHECK] Verify operator authorization
	if ms.delegationKeeper != nil {
		if err := ms.delegationKeeper.CheckOperatorAuthorization(
			ctx,
			msg.Authority,
			msg.Operator,
			"/verana.perm.v1.MsgSetPermissionVPToValidated",
			now,
		); err != nil {
			return nil, fmt.Errorf("authorization check failed: %w", err)
		}
	}

	// [MOD-PERM-MSG-3-2-1] Basic checks
	// Load Permission entry applicant_perm from id. If no entry found, abort.
	applicantPerm, err := ms.Keeper.GetPermissionByID(ctx, msg.Id)
	if err != nil {
		return nil, fmt.Errorf("perm not found: %w", err)
	}

	// applicant_perm.vp_state MUST be equal to PENDING, else abort.
	if applicantPerm.VpState != types.ValidationState_PENDING {
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

	// vp_summary_digest_sri: MUST be null if validation.type is set to HOLDER
	if applicantPerm.Type == types.PermissionType_HOLDER && msg.VpSummaryDigestSri != "" {
		return nil, fmt.Errorf("vp_summary_digest_sri must be null for HOLDER type")
	}

	// Load CredentialSchema cs from applicant_perm.schema_id.
	cs, err := ms.credentialSchemaKeeper.GetCredentialSchemaById(ctx, applicantPerm.SchemaId)
	if err != nil {
		return nil, fmt.Errorf("credential schema not found: %w", err)
	}

	// [MOD-PERM-MSG-3-2-1] Validate issuance_fee_discount
	// Load validator_perm early for discount validation
	validatorPerm, err := ms.Keeper.GetPermissionByID(ctx, applicantPerm.ValidatorPermId)
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
			if cs.IssuerPermManagementMode == credentialschematypes.CredentialSchemaPermManagementMode_GRANTOR_VALIDATION {
				if applicantPerm.Type == types.PermissionType_ISSUER_GRANTOR {
					// ISSUER_GRANTOR: can set 0-1 (100% discount)
					// Already validated range above
				} else if applicantPerm.Type == types.PermissionType_ISSUER {
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
			} else if cs.IssuerPermManagementMode == credentialschematypes.CredentialSchemaPermManagementMode_ECOSYSTEM {
				if applicantPerm.Type == types.PermissionType_ISSUER {
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
			if cs.VerifierPermManagementMode == credentialschematypes.CredentialSchemaPermManagementMode_GRANTOR_VALIDATION {
				if applicantPerm.Type == types.PermissionType_VERIFIER_GRANTOR {
					// VERIFIER_GRANTOR: can set 0-1 (100% discount)
					// Already validated range above
				} else if applicantPerm.Type == types.PermissionType_VERIFIER {
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
			} else if cs.VerifierPermManagementMode == credentialschematypes.CredentialSchemaPermManagementMode_ECOSYSTEM {
				if applicantPerm.Type == types.PermissionType_VERIFIER {
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

	// [MOD-PERM-MSG-3-2-1] Calculate vp_exp
	validityPeriod := getValidityPeriod(uint32(applicantPerm.Type), cs)
	var vpExp *time.Time
	if validityPeriod == 0 {
		vpExp = nil
	} else if applicantPerm.VpExp == nil {
		exp := now.AddDate(0, 0, int(validityPeriod))
		vpExp = &exp
	} else {
		exp := applicantPerm.VpExp.AddDate(0, 0, int(validityPeriod))
		vpExp = &exp
	}

	// [MOD-PERM-MSG-3-2-1] Verify effective_until and resolve its final value
	effectiveUntil := msg.EffectiveUntil
	if effectiveUntil == nil {
		// if provided effective_until is NULL: change value to vp_exp
		effectiveUntil = vpExp
	} else if applicantPerm.EffectiveUntil == nil {
		// effective_until MUST be greater than current timestamp
		if !effectiveUntil.After(now) {
			return nil, fmt.Errorf("effective_until must be greater than current timestamp")
		}
		// if vp_exp is not null, effective_until MUST be lower or equal to vp_exp
		if vpExp != nil && effectiveUntil.After(*vpExp) {
			return nil, fmt.Errorf("effective_until must be lower or equal to vp_exp")
		}
	} else {
		// effective_until MUST be greater than applicant_perm.effective_until
		if !effectiveUntil.After(*applicantPerm.EffectiveUntil) {
			return nil, fmt.Errorf("effective_until must be greater than current effective_until")
		}
		// if vp_exp is not null, effective_until MUST be lower or equal to vp_exp
		if vpExp != nil && effectiveUntil.After(*vpExp) {
			return nil, fmt.Errorf("effective_until must be lower or equal to vp_exp")
		}
	}

	// [MOD-PERM-MSG-3-2-2] Validator perms
	// validator_perm MUST be an active permission
	if err := IsValidPermission(validatorPerm, "", now); err != nil {
		return nil, fmt.Errorf("validator perm is not valid: %w", err)
	}

	// authority running the method MUST be validator_perm.authority
	if validatorPerm.Authority != msg.Authority {
		return nil, fmt.Errorf("authority must be validator perm authority")
	}

	// [MOD-PERM-MSG-3-2-4] Overlap checks (use resolved effectiveUntil)
	if err := ms.checkValidatedOverlap(ctx, applicantPerm, effectiveUntil); err != nil {
		return nil, fmt.Errorf("overlap check failed: %w", err)
	}

	// [MOD-PERM-MSG-3-3] Execution
	return ms.executeSetPermissionVPToValidated(ctx, applicantPerm, validatorPerm, msg, now, vpExp, effectiveUntil)
}

func (ms msgServer) CancelPermissionVPLastRequest(goCtx context.Context, msg *types.MsgCancelPermissionVPLastRequest) (*types.MsgCancelPermissionVPLastRequestResponse, error) {
	ctx := sdk.UnwrapSDKContext(goCtx)
	now := ctx.BlockTime()

	// [MOD-PERM-MSG-6-2-1] [AUTHZ-CHECK] Verify operator authorization
	if ms.delegationKeeper != nil {
		if err := ms.delegationKeeper.CheckOperatorAuthorization(
			ctx,
			msg.Authority,
			msg.Operator,
			"/verana.perm.v1.MsgCancelPermissionVPLastRequest",
			now,
		); err != nil {
			return nil, fmt.Errorf("authorization check failed: %w", err)
		}
	}

	// [MOD-PERM-MSG-6-2-1] Load Permission entry applicant_perm from id
	applicantPerm, err := ms.Keeper.GetPermissionByID(ctx, msg.Id)
	if err != nil {
		return nil, fmt.Errorf("perm not found: %w", err)
	}

	// authority running the transaction MUST be applicant_perm.authority
	if applicantPerm.Authority != msg.Authority {
		return nil, fmt.Errorf("authority is not the perm authority")
	}

	// applicant_perm.vp_state MUST be PENDING
	if applicantPerm.VpState != types.ValidationState_PENDING {
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
			types.EventTypeCancelPermissionVPLastRequest,
			sdk.NewAttribute(types.AttributeKeyPermissionID, strconv.FormatUint(msg.Id, 10)),
			sdk.NewAttribute(types.AttributeKeyAuthority, msg.Authority),
			sdk.NewAttribute(types.AttributeKeyOperator, msg.Operator),
			sdk.NewAttribute(types.AttributeKeyTimestamp, now.String()),
		),
	})

	return &types.MsgCancelPermissionVPLastRequestResponse{}, nil
}

func (ms msgServer) executeCancelPermissionVPLastRequest(ctx sdk.Context, perm types.Permission) error {
	now := ctx.BlockTime()

	// Update basic fields
	perm.Modified = &now
	perm.VpLastStateChange = &now

	// Set state based on vp_exp
	if perm.VpExp == nil {
		perm.VpState = types.ValidationState_TERMINATED
	} else {
		perm.VpState = types.ValidationState_VALIDATED
	}

	// Handle current fees if any
	if perm.VpCurrentFees > 0 {
		// Transfer escrowed fees back to the applicant
		granteeAddr, err := sdk.AccAddressFromBech32(perm.Authority)
		if err != nil {
			return fmt.Errorf("invalid grantee address: %w", err)
		}

		// Transfer fees from module escrow account to applicant account
		err = ms.bankKeeper.SendCoinsFromModuleToAccount(
			ctx,
			types.ModuleName, // Module escrow account
			granteeAddr,      // Applicant account
			sdk.NewCoins(sdk.NewInt64Coin(types.BondDenom, int64(perm.VpCurrentFees))),
		)
		if err != nil {
			return fmt.Errorf("failed to refund fees: %w", err)
		}

		perm.VpCurrentFees = 0
	}

	// Handle current deposit if any
	if perm.VpCurrentDeposit > 0 {
		// Use AdjustTrustDeposit to reduce trust deposit with negative value
		// to move funds from deposit to claimable
		if err := ms.trustDeposit.AdjustTrustDeposit(
			ctx,
			perm.Authority,
			-int64(perm.VpCurrentDeposit), // Negative value to reduce deposit and increase claimable
		); err != nil {
			return fmt.Errorf("failed to adjust trust deposit: %w", err)
		}

		perm.VpCurrentDeposit = 0
	}

	// Persist changes
	return ms.Keeper.UpdatePermission(ctx, perm)
}

func (ms msgServer) CreateRootPermission(goCtx context.Context, msg *types.MsgCreateRootPermission) (*types.MsgCreateRootPermissionResponse, error) {
	ctx := sdk.UnwrapSDKContext(goCtx)
	now := ctx.BlockTime()

	// [MOD-PERM-MSG-7-2-1] [AUTHZ-CHECK] Verify operator authorization
	if ms.delegationKeeper != nil {
		if err := ms.delegationKeeper.CheckOperatorAuthorization(
			ctx,
			msg.Authority,
			msg.Operator,
			"/verana.perm.v1.MsgCreateRootPermission",
			now,
		); err != nil {
			return nil, fmt.Errorf("authorization check failed: %w", err)
		}
	}

	// [MOD-PERM-MSG-7-2-1] Create Root Permission basic checks
	if err := ms.validateCreateRootPermissionBasicChecks(ctx, msg, now); err != nil {
		return nil, err
	}

	// [MOD-PERM-MSG-7-2-2] Permission checks
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
			types.EventTypeCreateRootPermission,
			sdk.NewAttribute(types.AttributeKeyRootPermissionID, strconv.FormatUint(id, 10)),
			sdk.NewAttribute(types.AttributeKeySchemaID, strconv.FormatUint(msg.SchemaId, 10)),
			sdk.NewAttribute(types.AttributeKeyAuthority, msg.Authority),
			sdk.NewAttribute(types.AttributeKeyOperator, msg.Operator),
			sdk.NewAttribute(types.AttributeKeyEffectiveFrom, formatTimePtr(msg.EffectiveFrom)),
			sdk.NewAttribute(types.AttributeKeyEffectiveUntil, formatTimePtr(msg.EffectiveUntil)),
			sdk.NewAttribute(types.AttributeKeyValidationFees, strconv.FormatUint(msg.ValidationFees, 10)),
			sdk.NewAttribute(types.AttributeKeyIssuanceFees, strconv.FormatUint(msg.IssuanceFees, 10)),
			sdk.NewAttribute(types.AttributeKeyVerificationFees, strconv.FormatUint(msg.VerificationFees, 10)),
			sdk.NewAttribute(types.AttributeKeyTimestamp, now.String()),
		),
	})

	return &types.MsgCreateRootPermissionResponse{
		Id: id,
	}, nil
}

// [MOD-PERM-MSG-7-2-1] Create Root Permission basic checks
func (ms msgServer) validateCreateRootPermissionBasicChecks(ctx sdk.Context, msg *types.MsgCreateRootPermission, now time.Time) error {
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
func (ms msgServer) validateCreateRootPermissionAuthority(ctx sdk.Context, msg *types.MsgCreateRootPermission) error {
	// Get credential schema
	cs, err := ms.credentialSchemaKeeper.GetCredentialSchemaById(ctx, msg.SchemaId)
	if err != nil {
		return fmt.Errorf("credential schema not found: %w", err)
	}

	// Load trust registry
	tr, err := ms.trustRegistryKeeper.GetTrustRegistry(ctx, cs.TrId)
	if err != nil {
		return fmt.Errorf("trust registry not found: %w", err)
	}

	// authority executing the method MUST be tr.authority (controller)
	if tr.Controller != msg.Authority {
		return fmt.Errorf("authority is not the trust registry controller")
	}

	return nil
}

// [MOD-PERM-MSG-7-2-4] Create Root Permission overlap checks
// Find all active permissions (not revoked, not slashed, not repaid) for schema_id, ECOSYSTEM, authority.
// For ECOSYSTEM type permissions, validator_perm_id is NULL (0), so we don't check it.
func (ms msgServer) checkCreateRootPermissionOverlap(ctx sdk.Context, msg *types.MsgCreateRootPermission) error {
	err := ms.Permission.Walk(ctx, nil, func(key uint64, perm types.Permission) (bool, error) {
		// Match on schema_id, ECOSYSTEM type, and authority
		if perm.SchemaId != msg.SchemaId ||
			perm.Type != types.PermissionType_ECOSYSTEM ||
			perm.Authority != msg.Authority {
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

// [MOD-PERM-MSG-7-3] Create Root Permission execution
func (ms msgServer) executeCreateRootPermission(ctx sdk.Context, msg *types.MsgCreateRootPermission, now time.Time) (uint64, error) {
	// Create new perm
	perm := types.Permission{
		// perm.id: auto-incremented uint64 (handled by CreatePermission)
		SchemaId:         msg.SchemaId,
		Modified:         &now,
		Type:             types.PermissionType_ECOSYSTEM,
		Did:              msg.Did,
		Authority:        msg.Authority,
		Created:          &now,
		EffectiveFrom:    msg.EffectiveFrom,
		EffectiveUntil:   msg.EffectiveUntil,
		ValidationFees:   msg.ValidationFees,
		IssuanceFees:     msg.IssuanceFees,
		VerificationFees: msg.VerificationFees,
		Deposit:          0,
	}

	// Store the perm
	id, err := ms.Keeper.CreatePermission(ctx, perm)
	if err != nil {
		return 0, fmt.Errorf("failed to create perm: %w", err)
	}

	return id, nil
}

func (ms msgServer) AdjustPermission(goCtx context.Context, msg *types.MsgAdjustPermission) (*types.MsgAdjustPermissionResponse, error) {
	ctx := sdk.UnwrapSDKContext(goCtx)
	now := ctx.BlockTime()

	// [MOD-PERM-MSG-8-2-1] [AUTHZ-CHECK] Verify operator authorization
	if ms.delegationKeeper != nil {
		if err := ms.delegationKeeper.CheckOperatorAuthorization(
			ctx,
			msg.Authority,
			msg.Operator,
			"/verana.perm.v1.MsgAdjustPermission",
			now,
		); err != nil {
			return nil, fmt.Errorf("authorization check failed: %w", err)
		}
	}

	// [MOD-PERM-MSG-8-2-1] Adjust Permission basic checks
	applicantPerm, err := ms.validateAdjustPermissionBasicChecks(ctx, msg, now)
	if err != nil {
		return nil, err
	}

	// [MOD-PERM-MSG-8-2-2] Adjust Permission advanced checks
	if err := ms.validateAdjustPermissionAdvancedChecks(ctx, msg, applicantPerm, now); err != nil {
		return nil, err
	}

	// [MOD-PERM-MSG-8-2-4] Overlap checks
	if err := ms.checkAdjustPermissionOverlap(ctx, applicantPerm, msg.EffectiveUntil); err != nil {
		return nil, fmt.Errorf("overlap check failed: %w", err)
	}

	// [MOD-PERM-MSG-8-3] Adjust Permission execution
	if err := ms.executeAdjustPermission(ctx, applicantPerm, msg.Authority, msg.EffectiveUntil, now); err != nil {
		return nil, fmt.Errorf("failed to adjust perm: %w", err)
	}

	// [MOD-PERM-MSG-8-3] If applicant_perm.type is ISSUER or VERIFIER and vs_operator_authz_enabled:
	// Grant VS Operator Authorization
	if (applicantPerm.Type == types.PermissionType_ISSUER || applicantPerm.Type == types.PermissionType_VERIFIER) &&
		applicantPerm.VsOperatorAuthzEnabled {
		if err := ms.grantVSOperatorAuthorization(ctx, applicantPerm); err != nil {
			return nil, fmt.Errorf("failed to grant VS operator authorization: %w", err)
		}
	}

	ctx.EventManager().EmitEvents(sdk.Events{
		sdk.NewEvent(
			types.EventTypeAdjustPermission,
			sdk.NewAttribute(types.AttributeKeyPermissionID, strconv.FormatUint(msg.Id, 10)),
			sdk.NewAttribute(types.AttributeKeyAuthority, msg.Authority),
			sdk.NewAttribute(types.AttributeKeyOperator, msg.Operator),
			sdk.NewAttribute(types.AttributeKeyAdjustedBy, msg.Authority),
			sdk.NewAttribute(types.AttributeKeyNewEffectiveUntil, msg.EffectiveUntil.String()),
			sdk.NewAttribute(types.AttributeKeyTimestamp, now.String()),
		),
	})

	return &types.MsgAdjustPermissionResponse{}, nil
}

// [MOD-PERM-MSG-8-2-1] Adjust Permission basic checks
func (ms msgServer) validateAdjustPermissionBasicChecks(ctx sdk.Context, msg *types.MsgAdjustPermission, now time.Time) (types.Permission, error) {
	var applicantPerm types.Permission

	// id MUST be a valid uint64 (already validated in ValidateBasic)

	// Load Permission entry applicant_perm from id. If no entry found, abort
	perm, err := ms.Keeper.GetPermissionByID(ctx, msg.Id)
	if err != nil {
		return applicantPerm, fmt.Errorf("permission not found: %w", err)
	}
	applicantPerm = perm

	// applicant_perm MUST be a valid permission
	if err := IsValidPermission(applicantPerm, applicantPerm.Country, now); err != nil {
		return applicantPerm, fmt.Errorf("applicant permission is not valid: %w", err)
	}

	// [MOD-PERM-MSG-8-2-1] effective_until MUST be greater than now()
	if !msg.EffectiveUntil.After(now) {
		return applicantPerm, fmt.Errorf("effective_until must be greater than current timestamp")
	}

	return applicantPerm, nil
}

// [MOD-PERM-MSG-8-2-2] Adjust Permission advanced checks
func (ms msgServer) validateAdjustPermissionAdvancedChecks(ctx sdk.Context, msg *types.MsgAdjustPermission, applicantPerm types.Permission, now time.Time) error {
	// 1. ECOSYSTEM permissions
	if applicantPerm.ValidatorPermId == 0 && applicantPerm.Type == types.PermissionType_ECOSYSTEM {
		// applicant_perm.authority MUST be msg.Authority else MUST abort
		if applicantPerm.Authority != msg.Authority {
			return fmt.Errorf("authority is not the permission authority")
		}
		return nil
	}

	// For permissions with validator_perm_id, we need to distinguish between cases 2 and 3
	if applicantPerm.ValidatorPermId != 0 {
		// Load validator_perm from applicant_perm.validator_perm_id
		validatorPerm, err := ms.Keeper.GetPermissionByID(ctx, applicantPerm.ValidatorPermId)
		if err != nil {
			return fmt.Errorf("validator permission not found: %w", err)
		}

		// validator_perm MUST be a valid permission
		if err := IsValidPermission(validatorPerm, validatorPerm.Country, now); err != nil {
			return fmt.Errorf("validator permission is not valid: %w", err)
		}

		// 2. Self-created permissions (validator is ECOSYSTEM)
		if validatorPerm.Type == types.PermissionType_ECOSYSTEM {
			// applicant_perm.authority MUST be msg.Authority else MUST abort
			if applicantPerm.Authority != msg.Authority {
				return fmt.Errorf("authority is not the permission authority")
			}
			return nil
		}

		// 3. VP managed permissions
		// effective_until MUST be lower or equal to applicant_perm.vp_exp else MUST abort
		if applicantPerm.VpExp != nil && msg.EffectiveUntil.After(*applicantPerm.VpExp) {
			return fmt.Errorf("effective_until cannot be after validation expiration")
		}

		// validator_perm.authority MUST be msg.Authority else MUST abort
		if validatorPerm.Authority != msg.Authority {
			return fmt.Errorf("authority is not the validator permission authority")
		}
		return nil
	}

	return fmt.Errorf("invalid permission configuration for adjustment")
}

// [MOD-PERM-MSG-8-2-4] Overlap checks for AdjustPermission
// Walk all permissions for same (schema_id, type, validator_perm_id, authority),
// skipping self and inactive (revoked/slashed/repaid).
func (ms msgServer) checkAdjustPermissionOverlap(ctx sdk.Context, applicantPerm types.Permission, effectiveUntil *time.Time) error {
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

// [MOD-PERM-MSG-8-3] Adjust Permission execution
func (ms msgServer) executeAdjustPermission(ctx sdk.Context, perm types.Permission, authority string, effectiveUntil *time.Time, now time.Time) error {
	// set applicant_perm.effective_until to effective_until
	perm.EffectiveUntil = effectiveUntil

	// set applicant_perm.adjusted to now
	perm.Adjusted = &now

	// set applicant_perm.modified to now
	perm.Modified = &now

	// set applicant_perm.adjusted_by to msg.authority
	perm.AdjustedBy = authority

	return ms.Keeper.UpdatePermission(ctx, perm)
}

// RevokePermission handles the MsgRevokePermission message
func (ms msgServer) RevokePermission(goCtx context.Context, msg *types.MsgRevokePermission) (*types.MsgRevokePermissionResponse, error) {
	ctx := sdk.UnwrapSDKContext(goCtx)
	now := ctx.BlockTime()

	// [MOD-PERM-MSG-9-2-1] [AUTHZ-CHECK] Verify operator authorization
	if ms.delegationKeeper != nil {
		if err := ms.delegationKeeper.CheckOperatorAuthorization(
			ctx,
			msg.Authority,
			msg.Operator,
			"/verana.perm.v1.MsgRevokePermission",
			now,
		); err != nil {
			return nil, fmt.Errorf("authorization check failed: %w", err)
		}
	}

	// [MOD-PERM-MSG-9-2-1] Revoke Permission basic checks
	applicantPerm, err := ms.validateRevokePermissionBasicChecks(ctx, msg, now)
	if err != nil {
		return nil, err
	}

	// [MOD-PERM-MSG-9-2-2] Revoke Permission advanced checks
	if err := ms.validateRevokePermissionAdvancedChecks(ctx, msg, applicantPerm, now); err != nil {
		return nil, err
	}

	// [MOD-PERM-MSG-9-2-3] Revoke Permission fee checks
	// Account MUST have the required estimated transaction fees available
	// (This is handled by the SDK automatically during transaction processing)

	// [MOD-PERM-MSG-9-3] Revoke Permission execution
	if err := ms.executeRevokePermission(ctx, applicantPerm, msg.Authority, now); err != nil {
		return nil, fmt.Errorf("failed to revoke permission: %w", err)
	}

	// [MOD-PERM-MSG-9-3] If applicant_perm.type is ISSUER or VERIFIER:
	// Delete authorization for applicant_perm.vs_operator
	if applicantPerm.Type == types.PermissionType_ISSUER || applicantPerm.Type == types.PermissionType_VERIFIER {
		if err := ms.revokeVSOperatorAuthorization(ctx, applicantPerm); err != nil {
			return nil, fmt.Errorf("failed to revoke VS operator authorization: %w", err)
		}
	}

	ctx.EventManager().EmitEvents(sdk.Events{
		sdk.NewEvent(
			types.EventTypeRevokePermission,
			sdk.NewAttribute(types.AttributeKeyPermissionID, strconv.FormatUint(msg.Id, 10)),
			sdk.NewAttribute(types.AttributeKeyAuthority, msg.Authority),
			sdk.NewAttribute(types.AttributeKeyOperator, msg.Operator),
			sdk.NewAttribute(types.AttributeKeyRevokedBy, msg.Authority),
			sdk.NewAttribute(types.AttributeKeyRevokedAt, now.String()),
			sdk.NewAttribute(types.AttributeKeyTimestamp, now.String()),
		),
	})

	return &types.MsgRevokePermissionResponse{}, nil
}

// [MOD-PERM-MSG-9-2-1] Revoke Permission basic checks
func (ms msgServer) validateRevokePermissionBasicChecks(ctx sdk.Context, msg *types.MsgRevokePermission, now time.Time) (types.Permission, error) {
	var applicantPerm types.Permission

	// id MUST be a valid uint64 (already validated in ValidateBasic)

	// Load Permission entry applicant_perm from id. If no entry found, abort
	perm, err := ms.Keeper.GetPermissionByID(ctx, msg.Id)
	if err != nil {
		return applicantPerm, fmt.Errorf("permission not found: %w", err)
	}
	applicantPerm = perm

	// [MOD-PERM-MSG-9-2-1] applicant_perm MUST be a active permission
	if err := IsValidPermission(applicantPerm, applicantPerm.Country, now); err != nil {
		return applicantPerm, fmt.Errorf("applicant permission is not active: %w", err)
	}

	return applicantPerm, nil
}

// [MOD-PERM-MSG-9-2-2] Revoke Permission advanced checks
func (ms msgServer) validateRevokePermissionAdvancedChecks(ctx sdk.Context, msg *types.MsgRevokePermission, applicantPerm types.Permission, now time.Time) error {
	// Either Option #1, #2 or #3 MUST return true, else abort

	// Option #1: executed by a validator ancestor
	if ms.checkValidatorAncestorOption(ctx, msg.Authority, applicantPerm, now) {
		return nil
	}

	// Option #2: executed by TrustRegistry controller
	if ms.checkTrustRegistryControllerOption(ctx, msg.Authority, applicantPerm) {
		return nil
	}

	// Option #3: executed by applicant_perm.authority
	if applicantPerm.Authority == msg.Authority {
		return nil
	}

	return fmt.Errorf("authority is not authorized to revoke this permission")
}

// Option #1: executed by a validator ancestor
func (ms msgServer) checkValidatorAncestorOption(ctx sdk.Context, authority string, applicantPerm types.Permission, now time.Time) bool {
	// if applicant_perm.validator_perm_id is defined
	if applicantPerm.ValidatorPermId == 0 {
		return false
	}

	// set validator_perm = applicant_perm
	// while validator_perm.validator_perm_id is defined
	currentValidatorPermId := applicantPerm.ValidatorPermId

	for currentValidatorPermId != 0 {
		// load validator_perm from validator_perm.validator_perm_id
		validatorPerm, err := ms.Keeper.GetPermissionByID(ctx, currentValidatorPermId)
		if err != nil {
			return false
		}

		// if validator_perm is a active permission and validator_perm.authority is who is running the method
		if IsValidPermission(validatorPerm, validatorPerm.Country, now) == nil &&
			validatorPerm.Authority == authority {
			return true
		}

		// Move up to the next ancestor
		currentValidatorPermId = validatorPerm.ValidatorPermId
	}

	return false
}

// Option #2: executed by TrustRegistry controller
func (ms msgServer) checkTrustRegistryControllerOption(ctx sdk.Context, authority string, applicantPerm types.Permission) bool {
	// load CredentialSchema cs from applicant_perm.schema_id
	cs, err := ms.credentialSchemaKeeper.GetCredentialSchemaById(ctx, applicantPerm.SchemaId)
	if err != nil {
		return false
	}

	// load TrustRegistry tr from cs.tr_id
	tr, err := ms.trustRegistryKeeper.GetTrustRegistry(ctx, cs.TrId)
	if err != nil {
		return false
	}

	// if authority running the method is tr.controller, return true
	return tr.Controller == authority
}

// [MOD-PERM-MSG-9-3] Revoke Permission execution
func (ms msgServer) executeRevokePermission(ctx sdk.Context, perm types.Permission, authority string, now time.Time) error {
	// set applicant_perm.revoked to now
	perm.Revoked = &now

	// set applicant_perm.modified to now
	perm.Modified = &now

	// set applicant_perm.revoked_by to authority
	perm.RevokedBy = authority

	return ms.Keeper.UpdatePermission(ctx, perm)
}

// revokeVSOperatorAuthorization revokes VS operator authorization for ISSUER/VERIFIER permissions
func (ms msgServer) revokeVSOperatorAuthorization(ctx sdk.Context, perm types.Permission) error {
	if perm.VsOperator == "" {
		return nil // No VS operator configured
	}

	if ms.delegationKeeper != nil {
		if err := ms.delegationKeeper.RevokeVSOperatorAuthorization(
			ctx,
			perm.Authority,
			perm.VsOperator,
			perm.Id,
		); err != nil {
			return fmt.Errorf("failed to revoke VS operator authorization: %w", err)
		}
	}

	return nil
}

func (ms msgServer) CreateOrUpdatePermissionSession(goCtx context.Context, msg *types.MsgCreateOrUpdatePermissionSession) (*types.MsgCreateOrUpdatePermissionSessionResponse, error) {
	ctx := sdk.UnwrapSDKContext(goCtx)
	now := ctx.BlockTime()

	// [MOD-PERM-MSG-10-2] Create or Update Permission Session precondition checks
	if err := ms.validateCreateOrUpdatePermissionSessionPreconditions(ctx, msg, now); err != nil {
		return nil, err
	}

	// [MOD-PERM-MSG-10-3] Create or Update Permission Session fee checks
	foundPermSet, beneficiaryFees, trustFees, err := ms.validateCreateOrUpdatePermissionSessionFees(ctx, msg)
	if err != nil {
		return nil, err
	}

	// [MOD-PERM-MSG-10-4] Create or Update Permission Session execution
	if err := ms.executeCreateOrUpdatePermissionSession(ctx, msg, foundPermSet, beneficiaryFees, trustFees, now); err != nil {
		return nil, fmt.Errorf("failed to create/update permission session: %w", err)
	}

	// Emit events
	ctx.EventManager().EmitEvents(sdk.Events{
		sdk.NewEvent(
			types.EventTypeCreateOrUpdatePermissionSession,
			sdk.NewAttribute(types.AttributeKeyAuthority, msg.Authority),
			sdk.NewAttribute(types.AttributeKeyOperator, msg.Operator),
			sdk.NewAttribute(types.AttributeKeySessionID, msg.Id),
			sdk.NewAttribute(types.AttributeKeyIssuerPermID, strconv.FormatUint(msg.IssuerPermId, 10)),
			sdk.NewAttribute(types.AttributeKeyVerifierPermID, strconv.FormatUint(msg.VerifierPermId, 10)),
			sdk.NewAttribute(types.AttributeKeyAgentPermID, strconv.FormatUint(msg.AgentPermId, 10)),
			sdk.NewAttribute(types.AttributeKeyWalletAgentPermID, strconv.FormatUint(msg.WalletAgentPermId, 10)),
			sdk.NewAttribute(types.AttributeKeyTimestamp, now.String()),
		),
	})

	return &types.MsgCreateOrUpdatePermissionSessionResponse{
		Id: msg.Id,
	}, nil
}

// SlashPermissionTrustDeposit handles the MsgSlashPermissionTrustDeposit message
func (ms msgServer) SlashPermissionTrustDeposit(goCtx context.Context, msg *types.MsgSlashPermissionTrustDeposit) (*types.MsgSlashPermissionTrustDepositResponse, error) {
	ctx := sdk.UnwrapSDKContext(goCtx)
	now := ctx.BlockTime()

	// [MOD-PERM-MSG-12-2-1] [AUTHZ-CHECK] Verify operator authorization
	if ms.delegationKeeper != nil {
		if err := ms.delegationKeeper.CheckOperatorAuthorization(
			ctx,
			msg.Authority,
			msg.Operator,
			"/verana.perm.v1.MsgSlashPermissionTrustDeposit",
			now,
		); err != nil {
			return nil, fmt.Errorf("authorization check failed: %w", err)
		}
	}

	// [MOD-PERM-MSG-12-2-1] Slash Permission Trust Deposit basic checks
	applicantPerm, err := ms.validateSlashPermissionBasicChecks(ctx, msg)
	if err != nil {
		return nil, err
	}

	// [MOD-PERM-MSG-12-2-2] Slash Permission Trust Deposit validator perms
	if err := ms.validateSlashPermissionValidatorPerms(ctx, msg, applicantPerm, now); err != nil {
		return nil, err
	}

	// [MOD-PERM-MSG-12-2-3] Slash Permission Trust Deposit fee checks
	// Account MUST have the required estimated transaction fees available
	// (This is handled by the SDK automatically during transaction processing)

	// [MOD-PERM-MSG-12-3] Slash Permission Trust Deposit execution
	if err := ms.executeSlashPermissionTrustDeposit(ctx, applicantPerm, msg.Amount, msg.Authority, now); err != nil {
		return nil, fmt.Errorf("failed to slash permission trust deposit: %w", err)
	}

	// [MOD-PERM-MSG-12-3] If applicant_perm.type is ISSUER or VERIFIER:
	// Delete authorization for applicant_perm.vs_operator
	if applicantPerm.Type == types.PermissionType_ISSUER || applicantPerm.Type == types.PermissionType_VERIFIER {
		if err := ms.revokeVSOperatorAuthorization(ctx, applicantPerm); err != nil {
			return nil, fmt.Errorf("failed to revoke VS operator authorization: %w", err)
		}
	}

	// Emit event
	ctx.EventManager().EmitEvents(sdk.Events{
		sdk.NewEvent(
			types.EventTypeSlashPermissionTrustDeposit,
			sdk.NewAttribute(types.AttributeKeyPermissionID, strconv.FormatUint(msg.Id, 10)),
			sdk.NewAttribute(types.AttributeKeySlashedAmount, strconv.FormatUint(msg.Amount, 10)),
			sdk.NewAttribute(types.AttributeKeyAuthority, msg.Authority),
			sdk.NewAttribute(types.AttributeKeyOperator, msg.Operator),
			sdk.NewAttribute(types.AttributeKeySlashedBy, msg.Authority),
			sdk.NewAttribute(types.AttributeKeyTimestamp, now.String()),
		),
	})

	return &types.MsgSlashPermissionTrustDepositResponse{}, nil
}

// [MOD-PERM-MSG-12-2-1] Slash Permission Trust Deposit basic checks
func (ms msgServer) validateSlashPermissionBasicChecks(ctx sdk.Context, msg *types.MsgSlashPermissionTrustDeposit) (types.Permission, error) {
	var applicantPerm types.Permission

	// id MUST be a valid uint64 (already validated in ValidateBasic)

	// Load Permission entry applicant_perm from id. If no entry found, abort
	perm, err := ms.Keeper.GetPermissionByID(ctx, msg.Id)
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

// [MOD-PERM-MSG-12-2-2] Slash Permission Trust Deposit validator perms
func (ms msgServer) validateSlashPermissionValidatorPerms(ctx sdk.Context, msg *types.MsgSlashPermissionTrustDeposit, applicantPerm types.Permission, now time.Time) error {
	// Either Option #1, or #2 MUST return true, else abort

	// Option #1: executed by a validator ancestor
	if ms.checkValidatorAncestorOption(ctx, msg.Authority, applicantPerm, now) {
		return nil
	}

	// Option #2: executed by TrustRegistry controller
	if ms.checkTrustRegistryControllerOption(ctx, msg.Authority, applicantPerm) {
		return nil
	}

	return fmt.Errorf("authority is not authorized to slash this permission")
}

// [MOD-PERM-MSG-12-3] Slash Permission Trust Deposit execution
func (ms msgServer) executeSlashPermissionTrustDeposit(ctx sdk.Context, applicantPerm types.Permission, amount uint64, authority string, now time.Time) error {
	// Load Permission entry validator_perm from applicant_perm.validator_perm_id
	if applicantPerm.ValidatorPermId != 0 {
		_, err := ms.Keeper.GetPermissionByID(ctx, applicantPerm.ValidatorPermId)
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

	// set applicant_perm.slashed_by to authority executing the method
	applicantPerm.SlashedBy = authority

	// use MOD-TD-MSG-7 to burn the slashed amount from the trust deposit of applicant_perm.authority
	if err := ms.trustDeposit.BurnEcosystemSlashedTrustDeposit(ctx, applicantPerm.Authority, amount); err != nil {
		return fmt.Errorf("failed to burn trust deposit: %w", err)
	}

	// Update permission
	if err := ms.Keeper.UpdatePermission(ctx, applicantPerm); err != nil {
		return fmt.Errorf("failed to update permission: %w", err)
	}

	return nil
}

// RepayPermissionSlashedTrustDeposit handles the MsgRepayPermissionSlashedTrustDeposit message
func (ms msgServer) RepayPermissionSlashedTrustDeposit(goCtx context.Context, msg *types.MsgRepayPermissionSlashedTrustDeposit) (*types.MsgRepayPermissionSlashedTrustDepositResponse, error) {
	ctx := sdk.UnwrapSDKContext(goCtx)
	now := ctx.BlockTime()

	// [AUTHZ-CHECK]
	if ms.delegationKeeper != nil {
		if err := ms.delegationKeeper.CheckOperatorAuthorization(ctx, msg.Authority, msg.Operator, "/verana.perm.v1.MsgRepayPermissionSlashedTrustDeposit", now); err != nil {
			return nil, fmt.Errorf("authorization check failed: %w", err)
		}
	}

	// [MOD-PERM-MSG-13-2-1] Load Permission entry applicant_perm from id
	applicantPerm, err := ms.Keeper.GetPermissionByID(ctx, msg.Id)
	if err != nil {
		return nil, fmt.Errorf("perm not found: %w", err)
	}

	// [MOD-PERM-MSG-13-2-1] if applicant_perm.authority is not equal to authority, abort
	if applicantPerm.Authority != msg.Authority {
		return nil, fmt.Errorf("authority is not the owner of this permission")
	}

	// Check if perm has been slashed
	if applicantPerm.SlashedDeposit == 0 {
		return nil, fmt.Errorf("perm has no slashed deposit to repay")
	}

	// Check if already repaid
	if applicantPerm.RepaidDeposit >= applicantPerm.SlashedDeposit {
		return nil, fmt.Errorf("slashed deposit already fully repaid")
	}

	// [MOD-PERM-MSG-13-2-2] authority MUST have at least applicant_perm.slashed_deposit in its account balance
	authorityAddr, err := sdk.AccAddressFromBech32(msg.Authority)
	if err != nil {
		return nil, fmt.Errorf("invalid authority address: %w", err)
	}
	requiredAmount := sdk.NewInt64Coin(types.BondDenom, int64(applicantPerm.SlashedDeposit))
	if !ms.bankKeeper.HasBalance(ctx, authorityAddr, requiredAmount) {
		return nil, fmt.Errorf("insufficient funds to repay slashed deposit: required %d", applicantPerm.SlashedDeposit)
	}

	// [MOD-PERM-MSG-13-3] Execution
	// Use AdjustTrustDeposit to transfer applicant_perm.slashed_deposit to trust deposit of applicant_perm.authority
	if err := ms.trustDeposit.AdjustTrustDeposit(ctx, applicantPerm.Authority, int64(applicantPerm.SlashedDeposit)); err != nil {
		return nil, fmt.Errorf("failed to adjust trust deposit: %w", err)
	}

	// Update Permission entry
	applicantPerm.Repaid = &now
	applicantPerm.Modified = &now
	applicantPerm.RepaidDeposit = applicantPerm.SlashedDeposit

	if err := ms.Keeper.UpdatePermission(ctx, applicantPerm); err != nil {
		return nil, fmt.Errorf("failed to update perm: %w", err)
	}

	// Emit event
	ctx.EventManager().EmitEvents(sdk.Events{
		sdk.NewEvent(
			types.EventTypeRepayPermissionSlashedTrustDeposit,
			sdk.NewAttribute(types.AttributeKeyPermissionID, strconv.FormatUint(msg.Id, 10)),
			sdk.NewAttribute(types.AttributeKeyRepaidAmount, strconv.FormatUint(applicantPerm.SlashedDeposit, 10)),
			sdk.NewAttribute(types.AttributeKeyAuthority, msg.Authority),
			sdk.NewAttribute(types.AttributeKeyOperator, msg.Operator),
			sdk.NewAttribute(types.AttributeKeyTimestamp, ctx.BlockTime().String()),
		),
	})

	return &types.MsgRepayPermissionSlashedTrustDepositResponse{}, nil
}

// CreatePermission handles the MsgCreatePermission message
func (ms msgServer) CreatePermission(goCtx context.Context, msg *types.MsgCreatePermission) (*types.MsgCreatePermissionResponse, error) {
	ctx := sdk.UnwrapSDKContext(goCtx)
	now := ctx.BlockTime()

	// [AUTHZ-CHECK]
	if ms.delegationKeeper != nil {
		if err := ms.delegationKeeper.CheckOperatorAuthorization(ctx, msg.Authority, msg.Operator, "/verana.perm.v1.MsgCreatePermission", now); err != nil {
			return nil, fmt.Errorf("authorization check failed: %w", err)
		}
	}

	// [MOD-PERM-MSG-14-2-1] Load validator_perm from validator_perm_id
	validatorPerm, err := ms.Keeper.GetPermissionByID(ctx, msg.ValidatorPermId)
	if err != nil {
		return nil, fmt.Errorf("validator permission not found: %w", err)
	}

	// validator_perm MUST be an ECOSYSTEM active permission or future permission
	if validatorPerm.Type != types.PermissionType_ECOSYSTEM {
		return nil, fmt.Errorf("validator_perm_id must reference an ECOSYSTEM permission")
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

	// [MOD-PERM-MSG-14-2-1] effective_from checks
	if msg.EffectiveFrom != nil {
		if !msg.EffectiveFrom.After(now) {
			return nil, fmt.Errorf("effective_from must be in the future")
		}
		// MUST be >= validator_perm.effective_from
		if validatorPerm.EffectiveFrom != nil && msg.EffectiveFrom.Before(*validatorPerm.EffectiveFrom) {
			return nil, fmt.Errorf("effective_from must be >= validator_perm.effective_from")
		}
		// if validator_perm.effective_until is not null, MUST be < validator_perm.effective_until
		if validatorPerm.EffectiveUntil != nil && !msg.EffectiveFrom.Before(*validatorPerm.EffectiveUntil) {
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
		if msg.EffectiveFrom != nil && !msg.EffectiveUntil.After(*msg.EffectiveFrom) {
			return nil, fmt.Errorf("effective_until must be greater than effective_from")
		}
		// if validator_perm.effective_until is not null, MUST be <= validator_perm.effective_until
		if validatorPerm.EffectiveUntil != nil && msg.EffectiveUntil.After(*validatorPerm.EffectiveUntil) {
			return nil, fmt.Errorf("effective_until must be <= validator_perm.effective_until")
		}
	}

	// verification_fees: If specified, MUST be >= 0 and MUST be a ISSUER permission
	if msg.VerificationFees > 0 && msg.Type != types.PermissionType_ISSUER {
		return nil, fmt.Errorf("verification_fees can only be specified for ISSUER permissions")
	}
	// validation_fees: If specified, MUST be >= 0 and MUST be a ISSUER permission
	if msg.ValidationFees > 0 && msg.Type != types.PermissionType_ISSUER {
		return nil, fmt.Errorf("validation_fees can only be specified for ISSUER permissions")
	}

	// [MOD-PERM-MSG-14-2-2] Permission checks
	cs, err := ms.credentialSchemaKeeper.GetCredentialSchemaById(ctx, validatorPerm.SchemaId)
	if err != nil {
		return nil, fmt.Errorf("credential schema not found: %w", err)
	}

	if msg.Type == types.PermissionType_ISSUER {
		if cs.IssuerPermManagementMode != credentialschematypes.CredentialSchemaPermManagementMode_OPEN {
			return nil, fmt.Errorf("issuer permission management mode is not OPEN")
		}
	}
	if msg.Type == types.PermissionType_VERIFIER {
		if cs.VerifierPermManagementMode != credentialschematypes.CredentialSchemaPermManagementMode_OPEN {
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
	if err := ms.checkCreatePermissionOverlap(ctx, validatorPerm.SchemaId, msg); err != nil {
		return nil, err
	}

	// [MOD-PERM-MSG-14-3] Execution
	perm := types.Permission{
		ValidatorPermId:              msg.ValidatorPermId,
		SchemaId:                     validatorPerm.SchemaId,
		Modified:                     &now,
		Type:                         msg.Type,
		Did:                          msg.Did,
		Authority:                    msg.Authority,
		VsOperator:                   msg.VsOperator,
		Created:                      &now,
		CreatedBy:                    msg.Authority,
		EffectiveFrom:                msg.EffectiveFrom,
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
	if msg.Type == types.PermissionType_ISSUER {
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
			types.EventTypeCreatePermission,
			sdk.NewAttribute(types.AttributeKeyPermissionID, strconv.FormatUint(id, 10)),
			sdk.NewAttribute(types.AttributeKeySchemaID, strconv.FormatUint(validatorPerm.SchemaId, 10)),
			sdk.NewAttribute(types.AttributeKeyAuthority, msg.Authority),
			sdk.NewAttribute(types.AttributeKeyOperator, msg.Operator),
			sdk.NewAttribute(types.AttributeKeyType, msg.Type.String()),
			sdk.NewAttribute(types.AttributeKeyEffectiveFrom, formatTimePtr(msg.EffectiveFrom)),
			sdk.NewAttribute(types.AttributeKeyEffectiveUntil, formatTimePtr(msg.EffectiveUntil)),
			sdk.NewAttribute(types.AttributeKeyTimestamp, now.String()),
		),
	})

	return &types.MsgCreatePermissionResponse{
		Id: id,
	}, nil
}

// [MOD-PERM-MSG-14-2-4] Overlap checks for CreatePermission
func (ms msgServer) checkCreatePermissionOverlap(ctx sdk.Context, schemaId uint64, msg *types.MsgCreatePermission) error {
	// Find all active permissions (not revoked, not slashed, not repaid)
	// for same cs.id, type, validator_perm_id, authority
	var overlaps []types.Permission
	err := ms.Permission.Walk(ctx, nil, func(id uint64, p types.Permission) (stop bool, err error) {
		if p.SchemaId == schemaId &&
			p.Type == msg.Type &&
			p.ValidatorPermId == msg.ValidatorPermId &&
			p.Authority == msg.Authority &&
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
		if msg.EffectiveFrom != nil && p.EffectiveUntil.After(*msg.EffectiveFrom) {
			return fmt.Errorf("existing permission %d overlaps: its effective_until is after your effective_from", p.Id)
		}
		// if p.effective_from is lower than effective_until, abort
		if msg.EffectiveUntil != nil && p.EffectiveFrom != nil && p.EffectiveFrom.Before(*msg.EffectiveUntil) {
			return fmt.Errorf("existing permission %d overlaps: its effective_from is before your effective_until", p.Id)
		}
	}

	return nil
}

