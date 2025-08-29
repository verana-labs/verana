package keeper

import (
	"context"
	"fmt"
	"strconv"
	"time"

	credentialschematypes "github.com/verana-labs/verana/x/cs/types"
	trustdeposittypes "github.com/verana-labs/verana/x/td/types"

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

	// [MOD-PERM-MSG-1-2-2] Permission checks
	validatorPerm, err := ms.validatePermissionChecks(ctx, msg)
	if err != nil {
		return nil, fmt.Errorf("perm validation failed: %w", err)
	}

	// [MOD-PERM-MSG-1-2-3] Fee checks
	fees, deposit, err := ms.validateAndCalculateFees(ctx, msg.Creator, validatorPerm)
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
			sdk.NewAttribute(types.AttributeKeyCreator, msg.Creator),
			sdk.NewAttribute(types.AttributeKeyValidatorPermID, strconv.FormatUint(msg.ValidatorPermId, 10)),
			sdk.NewAttribute(types.AttributeKeyType, types.PermissionType(msg.Type).String()),
			sdk.NewAttribute(types.AttributeKeyCountry, msg.Country),
			sdk.NewAttribute(types.AttributeKeyFees, strconv.FormatUint(fees, 10)),
			sdk.NewAttribute(types.AttributeKeyDeposit, strconv.FormatUint(deposit, 10)),
		),
	})

	return &types.MsgStartPermissionVPResponse{
		PermissionId: permID,
	}, nil
}

func (ms msgServer) RenewPermissionVP(goCtx context.Context, msg *types.MsgRenewPermissionVP) (*types.MsgRenewPermissionVPResponse, error) {
	ctx := sdk.UnwrapSDKContext(goCtx)

	// [MOD-PERM-MSG-2-2-2] Permission checks
	applicantPerm, err := ms.Keeper.GetPermissionByID(ctx, msg.Id)
	if err != nil {
		return nil, fmt.Errorf("perm not found: %w", err)
	}

	// Verify creator is the grantee
	if applicantPerm.Grantee != msg.Creator {
		return nil, fmt.Errorf("creator is not the perm grantee")
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
	validationFees, validationDeposit, err := ms.validateAndCalculateFees(ctx, msg.Creator, validatorPerm)
	if err != nil {
		return nil, fmt.Errorf("fee validation failed: %w", err)
	}

	// [MOD-PERM-MSG-2-3] Execution
	if err := ms.executeRenewPermissionVP(ctx, applicantPerm, validationFees, validationDeposit); err != nil {
		return nil, fmt.Errorf("failed to execute perm VP renewal: %w", err)
	}

	return &types.MsgRenewPermissionVPResponse{}, nil
}

func (ms msgServer) executeRenewPermissionVP(ctx sdk.Context, perm types.Permission, fees, deposit uint64) error {
	// Increment trust deposit if deposit is greater than 0
	if deposit > 0 {
		if err := ms.trustDeposit.AdjustTrustDeposit(ctx, perm.Grantee, int64(deposit)); err != nil {
			return fmt.Errorf("failed to increase trust deposit: %w", err)
		}
	}

	// Send validation fees to escrow account if greater than 0
	if fees > 0 {
		// Get grantee address
		granteeAddr, err := sdk.AccAddressFromBech32(perm.Grantee)
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

	// country: If applicant_perm.effective_from is not null (renewal) country MUST be equal to applicant_perm.country
	if applicantPerm.EffectiveFrom != nil && msg.Country != applicantPerm.Country {
		return nil, fmt.Errorf("country cannot be changed during renewal")
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

	// Calculate vp_exp
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

	// Verify effective_until
	if msg.EffectiveUntil != nil {
		if applicantPerm.EffectiveUntil == nil {
			// effective_until MUST be greater than current timestamp
			if !msg.EffectiveUntil.After(now) {
				return nil, fmt.Errorf("effective_until must be greater than current timestamp")
			}
			// if vp_exp is not null, lower or equal to vp_exp
			if vpExp != nil && msg.EffectiveUntil.After(*vpExp) {
				return nil, fmt.Errorf("effective_until must be lower or equal to vp_exp")
			}
		} else {
			// effective_until MUST be greater than applicant_perm.effective_until
			if !msg.EffectiveUntil.After(*applicantPerm.EffectiveUntil) {
				return nil, fmt.Errorf("effective_until must be greater than current effective_until")
			}
			// if vp_exp is not null, lower or equal to vp_exp
			if vpExp != nil && msg.EffectiveUntil.After(*vpExp) {
				return nil, fmt.Errorf("effective_until must be lower or equal to vp_exp")
			}
		}
	}

	// [MOD-PERM-MSG-3-2-2] Validator perms
	// load validator_perm from applicant_perm.validator_perm_id
	validatorPerm, err := ms.Keeper.GetPermissionByID(ctx, applicantPerm.ValidatorPermId)
	if err != nil {
		return nil, fmt.Errorf("validator perm not found: %w", err)
	}
	// TODO: check for valid perm

	// account running the method MUST be validator_perm.grantee
	if validatorPerm.Grantee != msg.Creator {
		return nil, fmt.Errorf("account running method must be validator grantee")
	}

	// [MOD-PERM-MSG-3-3] Execution
	return ms.executeSetPermissionVPToValidated(ctx, applicantPerm, validatorPerm, msg, now, vpExp)
}

func (ms msgServer) CancelPermissionVPLastRequest(goCtx context.Context, msg *types.MsgCancelPermissionVPLastRequest) (*types.MsgCancelPermissionVPLastRequestResponse, error) {
	ctx := sdk.UnwrapSDKContext(goCtx)

	// Load applicant perm
	applicantPerm, err := ms.Keeper.GetPermissionByID(ctx, msg.Id)
	if err != nil {
		return nil, fmt.Errorf("perm not found: %w", err)
	}

	// Check if creator is the grantee
	if applicantPerm.Grantee != msg.Creator {
		return nil, fmt.Errorf("creator is not the perm grantee")
	}

	// Check perm state
	if applicantPerm.VpState != types.ValidationState_PENDING {
		return nil, fmt.Errorf("perm must be in PENDING state")
	}

	// [MOD-PERM-MSG-6-3] Execution
	if err := ms.executeCancelPermissionVPLastRequest(ctx, applicantPerm); err != nil {
		return nil, fmt.Errorf("failed to execute VP cancellation: %w", err)
	}

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
		granteeAddr, err := sdk.AccAddressFromBech32(perm.Grantee)
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
			perm.Grantee,
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

	// [MOD-PERM-MSG-7-2-1] Create Root Permission basic checks
	if err := ms.validateCreateRootPermissionBasicChecks(ctx, msg, now); err != nil {
		return nil, err
	}

	// [MOD-PERM-MSG-7-2-2] Permission checks
	if err := ms.validateCreateRootPermissionAuthority(ctx, msg); err != nil {
		return nil, err
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

	// effective_from must be in the future
	if msg.EffectiveFrom != nil && !msg.EffectiveFrom.After(now) {
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

	// account executing the method MUST be the controller of tr
	if tr.Controller != msg.Creator {
		return fmt.Errorf("creator is not the trust registry controller")
	}

	return nil
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
		Grantee:          msg.Creator,
		Created:          &now,
		CreatedBy:        msg.Creator,
		EffectiveFrom:    msg.EffectiveFrom,
		EffectiveUntil:   msg.EffectiveUntil,
		Country:          msg.Country,
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

func (ms msgServer) ExtendPermission(goCtx context.Context, msg *types.MsgExtendPermission) (*types.MsgExtendPermissionResponse, error) {
	ctx := sdk.UnwrapSDKContext(goCtx)
	now := ctx.BlockTime()

	// [MOD-PERM-MSG-8-2-1] Extend Permission basic checks
	applicantPerm, err := ms.validateExtendPermissionBasicChecks(ctx, msg, now)
	if err != nil {
		return nil, err
	}

	// [MOD-PERM-MSG-8-2-2] Extend Permission advanced checks
	if err := ms.validateExtendPermissionAdvancedChecks(ctx, msg, applicantPerm, now); err != nil {
		return nil, err
	}

	// [MOD-PERM-MSG-8-3] Extend Permission execution
	if err := ms.executeExtendPermission(ctx, applicantPerm, msg.Creator, msg.EffectiveUntil, now); err != nil {
		return nil, fmt.Errorf("failed to extend perm: %w", err)
	}

	return &types.MsgExtendPermissionResponse{}, nil
}

// [MOD-PERM-MSG-8-2-1] Extend Permission basic checks
func (ms msgServer) validateExtendPermissionBasicChecks(ctx sdk.Context, msg *types.MsgExtendPermission, now time.Time) (types.Permission, error) {
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

	// effective_until MUST be greater than applicant_perm.effective_until else MUST abort
	if applicantPerm.EffectiveUntil == nil {
		return applicantPerm, fmt.Errorf("cannot extend permission with no current effective_until")
	}
	if !msg.EffectiveUntil.After(*applicantPerm.EffectiveUntil) {
		return applicantPerm, fmt.Errorf("effective_until must be greater than current effective_until")
	}

	return applicantPerm, nil
}

// [MOD-PERM-MSG-8-2-2] Extend Permission advanced checks
func (ms msgServer) validateExtendPermissionAdvancedChecks(ctx sdk.Context, msg *types.MsgExtendPermission, applicantPerm types.Permission, now time.Time) error {
	// 1. ECOSYSTEM permissions
	if applicantPerm.ValidatorPermId == 0 && applicantPerm.Type == types.PermissionType_ECOSYSTEM {
		// account running the method MUST be applicant_perm.grantee
		if applicantPerm.Grantee != msg.Creator {
			return fmt.Errorf("creator is not the permission grantee")
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

		// 2. Self-created permissions
		if validatorPerm.Type == types.PermissionType_ECOSYSTEM {
			// account running the method MUST be applicant_perm.grantee
			if applicantPerm.Grantee != msg.Creator {
				return fmt.Errorf("creator is not the permission grantee")
			}
			return nil
		}

		// 3. VP managed permissions
		// effective_until MUST be lower or equal to applicant_perm.vp_exp else MUST abort
		if applicantPerm.VpExp != nil && msg.EffectiveUntil.After(*applicantPerm.VpExp) {
			return fmt.Errorf("effective_until cannot be after validation expiration")
		}

		// account running the method MUST be validator_perm.grantee
		if validatorPerm.Grantee != msg.Creator {
			return fmt.Errorf("creator is not the validator permission grantee")
		}
		return nil
	}

	return fmt.Errorf("invalid permission configuration for extension")
}

// [MOD-PERM-MSG-8-3] Extend Permission execution
func (ms msgServer) executeExtendPermission(ctx sdk.Context, perm types.Permission, creator string, effectiveUntil *time.Time, now time.Time) error {
	// set applicant_perm.effective_until to effective_until
	perm.EffectiveUntil = effectiveUntil

	// set applicant_perm.extended to now
	perm.Extended = &now

	// set applicant_perm.modified to now
	perm.Modified = &now

	// set applicant_perm.extended_by to account executing the method
	perm.ExtendedBy = creator

	return ms.Keeper.UpdatePermission(ctx, perm)
}

// RevokePermission handles the MsgRevokePermission message
func (ms msgServer) RevokePermission(goCtx context.Context, msg *types.MsgRevokePermission) (*types.MsgRevokePermissionResponse, error) {
	ctx := sdk.UnwrapSDKContext(goCtx)
	now := ctx.BlockTime()

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
	if err := ms.executeRevokePermission(ctx, applicantPerm, msg.Creator, now); err != nil {
		return nil, fmt.Errorf("failed to revoke permission: %w", err)
	}

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

	// applicant_perm MUST be a valid permission
	if err := IsValidPermission(applicantPerm, applicantPerm.Country, now); err != nil {
		return applicantPerm, fmt.Errorf("applicant permission is not valid: %w", err)
	}

	return applicantPerm, nil
}

// [MOD-PERM-MSG-9-2-2] Revoke Permission advanced checks
func (ms msgServer) validateRevokePermissionAdvancedChecks(ctx sdk.Context, msg *types.MsgRevokePermission, applicantPerm types.Permission, now time.Time) error {
	// Either Option #1, #2 or #3 MUST return true, else abort

	// Option #1: executed by a validator ancestor
	if ms.checkValidatorAncestorOption(ctx, msg.Creator, applicantPerm, now) {
		return nil
	}

	// Option #2: executed by TrustRegistry controller
	if ms.checkTrustRegistryControllerOption(ctx, msg.Creator, applicantPerm) {
		return nil
	}

	// Option #3: executed by applicant_perm.grantee
	if applicantPerm.Grantee == msg.Creator {
		return nil
	}

	return fmt.Errorf("creator is not authorized to revoke this permission")
}

// Option #1: executed by a validator ancestor
func (ms msgServer) checkValidatorAncestorOption(ctx sdk.Context, creator string, applicantPerm types.Permission, now time.Time) bool {
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

		// if validator_perm is a valid permission and validator_perm.grantee is who is running the method
		if IsValidPermission(validatorPerm, validatorPerm.Country, now) == nil &&
			validatorPerm.Grantee == creator {
			return true
		}

		// Move up to the next ancestor
		currentValidatorPermId = validatorPerm.ValidatorPermId
	}

	return false
}

// Option #2: executed by TrustRegistry controller
func (ms msgServer) checkTrustRegistryControllerOption(ctx sdk.Context, creator string, applicantPerm types.Permission) bool {
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

	// if account running the method is tr.controller, return true
	return tr.Controller == creator
}

// [MOD-PERM-MSG-9-3] Revoke Permission execution
func (ms msgServer) executeRevokePermission(ctx sdk.Context, perm types.Permission, creator string, now time.Time) error {
	// set applicant_perm.revoked to now
	perm.Revoked = &now

	// set applicant_perm.modified to now
	perm.Modified = &now

	// set applicant_perm.revoked_by to account executing the method
	perm.RevokedBy = creator

	return ms.Keeper.UpdatePermission(ctx, perm)
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
	if err := ms.executeSlashPermissionTrustDeposit(ctx, applicantPerm, msg.Amount, msg.Creator, now); err != nil {
		return nil, fmt.Errorf("failed to slash permission trust deposit: %w", err)
	}

	// Emit event
	ctx.EventManager().EmitEvents(sdk.Events{
		sdk.NewEvent(
			types.EventTypeSlashPermissionTrustDeposit,
			sdk.NewAttribute(types.AttributeKeyPermissionID, strconv.FormatUint(msg.Id, 10)),
			sdk.NewAttribute(types.AttributeKeySlashedAmount, strconv.FormatUint(msg.Amount, 10)),
			sdk.NewAttribute(types.AttributeKeySlashedBy, msg.Creator),
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

	// amount MUST be lower or equal to applicant_perm.deposit else MUST abort
	if msg.Amount > applicantPerm.Deposit {
		return applicantPerm, fmt.Errorf("amount exceeds available deposit: %d > %d", msg.Amount, applicantPerm.Deposit)
	}

	return applicantPerm, nil
}

// [MOD-PERM-MSG-12-2-2] Slash Permission Trust Deposit validator perms
func (ms msgServer) validateSlashPermissionValidatorPerms(ctx sdk.Context, msg *types.MsgSlashPermissionTrustDeposit, applicantPerm types.Permission, now time.Time) error {
	// Either Option #1, or #2 MUST return true, else abort

	// Option #1: executed by a validator ancestor
	if ms.checkValidatorAncestorOption(ctx, msg.Creator, applicantPerm, now) {
		return nil
	}

	// Option #2: executed by TrustRegistry controller
	if ms.checkTrustRegistryControllerOption(ctx, msg.Creator, applicantPerm) {
		return nil
	}

	return fmt.Errorf("creator does not have authority to slash this permission")
}

// [MOD-PERM-MSG-12-3] Slash Permission Trust Deposit execution
func (ms msgServer) executeSlashPermissionTrustDeposit(ctx sdk.Context, applicantPerm types.Permission, amount uint64, creator string, now time.Time) error {
	// Load Permission entry applicant_perm from id (already loaded)

	// Load Permission entry validator_perm from applicant_perm.validator_perm_id
	if applicantPerm.ValidatorPermId != 0 {
		_, err := ms.Keeper.GetPermissionByID(ctx, applicantPerm.ValidatorPermId)
		if err != nil {
			return fmt.Errorf("validator permission not found: %w", err)
		}
		// Note: validator_perm is loaded but not used per spec
	}

	// set applicant_perm.slashed to now
	applicantPerm.Slashed = &now

	// set applicant_perm.modified to now
	applicantPerm.Modified = &now

	// set applicant_perm.slashed_deposit to applicant_perm.slashed_deposit + amount
	applicantPerm.SlashedDeposit = applicantPerm.SlashedDeposit + amount

	// set applicant_perm.slashed_by to account executing the method
	applicantPerm.SlashedBy = creator

	// use MOD-TD-MSG-7 to burn the slashed amount from the trust deposit of applicant_perm.grantee
	if err := ms.trustDeposit.BurnEcosystemSlashedTrustDeposit(ctx, applicantPerm.Grantee, amount); err != nil {
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

	// Load Permission entry applicant_perm from id
	applicantPerm, err := ms.Keeper.GetPermissionByID(ctx, msg.Id)
	if err != nil {
		return nil, fmt.Errorf("perm not found: %w", err)
	}

	// Check if perm has been slashed
	if applicantPerm.SlashedDeposit == 0 {
		return nil, fmt.Errorf("perm has no slashed deposit to repay")
	}

	// Check if already repaid
	if applicantPerm.RepaidDeposit >= applicantPerm.SlashedDeposit {
		return nil, fmt.Errorf("slashed deposit already fully repaid")
	}

	// Calculate amount to repay (remaining slashed amount)
	amountToRepay := applicantPerm.SlashedDeposit - applicantPerm.RepaidDeposit

	// [MOD-PERM-MSG-13-2-3] Repay Permission Slashed Trust Deposit fee checks
	// Account must have transaction fees + slashed_deposit amount
	senderAddr, err := sdk.AccAddressFromBech32(msg.Creator)
	if err != nil {
		return nil, fmt.Errorf("invalid creator address: %w", err)
	}

	// Check if sender has sufficient balance for repayment
	requiredAmount := sdk.NewCoins(sdk.NewInt64Coin(types.BondDenom, int64(amountToRepay)))
	if !ms.bankKeeper.HasBalance(ctx, senderAddr, requiredAmount[0]) {
		return nil, fmt.Errorf("insufficient funds to repay slashed deposit: required %d", amountToRepay)
	}

	// [MOD-PERM-MSG-13-3] Repay Permission Slashed Trust Deposit execution
	if err := ms.executeRepayPermissionSlashedTrustDeposit(ctx, applicantPerm, amountToRepay, msg.Creator); err != nil {
		return nil, fmt.Errorf("failed to repay perm slashed trust deposit: %w", err)
	}

	// Emit event
	ctx.EventManager().EmitEvents(sdk.Events{
		sdk.NewEvent(
			types.EventTypeRepayPermissionSlashedTrustDeposit,
			sdk.NewAttribute(types.AttributeKeyPermissionID, strconv.FormatUint(msg.Id, 10)),
			sdk.NewAttribute(types.AttributeKeyRepaidAmount, strconv.FormatUint(amountToRepay, 10)),
			sdk.NewAttribute(types.AttributeKeyRepaidBy, msg.Creator),
			sdk.NewAttribute(types.AttributeKeyTimestamp, ctx.BlockTime().String()),
		),
	})

	return &types.MsgRepayPermissionSlashedTrustDepositResponse{}, nil
}

// executeRepayPermissionSlashedTrustDeposit performs the actual repayment execution
func (ms msgServer) executeRepayPermissionSlashedTrustDeposit(ctx sdk.Context, applicantPerm types.Permission, amount uint64, repaidBy string) error {
	now := ctx.BlockTime()

	// Transfer repayment amount from repayer to trust deposit module
	senderAddr, err := sdk.AccAddressFromBech32(repaidBy)
	if err != nil {
		return fmt.Errorf("invalid repaid_by address: %w", err)
	}

	// Transfer tokens from repayer to trust deposit module
	if err := ms.bankKeeper.SendCoinsFromAccountToModule(
		ctx,
		senderAddr,
		trustdeposittypes.ModuleName, //to the trust deposit module
		sdk.NewCoins(sdk.NewInt64Coin(types.BondDenom, int64(amount))),
	); err != nil {
		return fmt.Errorf("failed to transfer repayment: %w", err)
	}

	// Update Permission entry applicant_perm
	applicantPerm.Repaid = &now
	applicantPerm.Modified = &now
	applicantPerm.RepaidDeposit = amount
	applicantPerm.RepaidBy = repaidBy

	// Use AdjustTrustDeposit to transfer amount to trust deposit of applicant_perm.grantee
	if err := ms.trustDeposit.AdjustTrustDeposit(ctx, applicantPerm.Grantee, int64(amount)); err != nil {
		return fmt.Errorf("failed to adjust trust deposit: %w", err)
	}

	// Update perm
	if err := ms.Keeper.UpdatePermission(ctx, applicantPerm); err != nil {
		return fmt.Errorf("failed to update perm: %w", err)
	}

	return nil
}

// CreatePermission handles the MsgCreatePermission message
func (ms msgServer) CreatePermission(goCtx context.Context, msg *types.MsgCreatePermission) (*types.MsgCreatePermissionResponse, error) {
	ctx := sdk.UnwrapSDKContext(goCtx)
	now := ctx.BlockTime()

	// [MOD-PERM-MSG-14-2-1] Create Permission basic checks
	if err := ms.validateCreatePermissionBasicChecks(ctx, msg, now); err != nil {
		return nil, err
	}

	// [MOD-PERM-MSG-14-2-2] Create Permission permission checks
	if err := ms.validateCreatePermissionPermissionChecks(ctx, msg); err != nil {
		return nil, err
	}

	// [MOD-PERM-MSG-14-2-3] Create Permission fee checks
	// Account MUST have the required estimated transaction fees available
	// (This is handled by the SDK automatically during transaction processing)

	// [MOD-PERM-MSG-14-3] Create Permission execution
	permissionId, err := ms.executeCreatePermission(ctx, msg, now)
	if err != nil {
		return nil, fmt.Errorf("failed to create permission: %w", err)
	}

	// Emit event
	ctx.EventManager().EmitEvents(sdk.Events{
		sdk.NewEvent(
			types.EventTypeCreatePermission,
			sdk.NewAttribute(types.AttributeKeyPermissionID, strconv.FormatUint(permissionId, 10)),
			sdk.NewAttribute(types.AttributeKeySchemaID, strconv.FormatUint(msg.SchemaId, 10)),
			sdk.NewAttribute(types.AttributeKeyCreator, msg.Creator),
			sdk.NewAttribute(types.AttributeKeyTimestamp, now.String()),
		),
	})

	return &types.MsgCreatePermissionResponse{
		Id: permissionId,
	}, nil
}

// [MOD-PERM-MSG-14-2-1] Create Permission basic checks
func (ms msgServer) validateCreatePermissionBasicChecks(ctx sdk.Context, msg *types.MsgCreatePermission, now time.Time) error {
	// schema_id MUST be a valid uint64 and a credential schema entry with this id MUST exist
	_, err := ms.credentialSchemaKeeper.GetCredentialSchemaById(ctx, msg.SchemaId)
	if err != nil {
		return fmt.Errorf("credential schema not found: %w", err)
	}

	// type (PermissionType) (mandatory): MUST be ISSUER or VERIFIER, else abort
	// (already validated in ValidateBasic)

	// did, if specified, MUST conform to the DID Syntax
	// (already validated in ValidateBasic)

	// effective_from must be in the future
	if msg.EffectiveFrom != nil && !msg.EffectiveFrom.After(now) {
		return fmt.Errorf("effective_from must be in the future")
	}

	// effective_until, if not null, must be greater than effective_from
	if msg.EffectiveUntil != nil && msg.EffectiveFrom != nil {
		if !msg.EffectiveUntil.After(*msg.EffectiveFrom) {
			return fmt.Errorf("effective_until must be greater than effective_from")
		}
	}

	// country if not null, MUST be a valid alpha-2 code (ISO 3166)
	if msg.Country != "" && !isValidCountryCode(msg.Country) {
		return fmt.Errorf("invalid country code format")
	}

	// verification_fees (number) (optional): If specified, MUST be >= 0 and MUST be a ISSUER permission
	if msg.VerificationFees > 0 && msg.Type != types.PermissionType_ISSUER {
		return fmt.Errorf("verification_fees can only be specified for ISSUER permissions")
	}

	// validation_fees (number) (optional): If specified, MUST be >= 0 and MUST be a ISSUER permission
	if msg.ValidationFees > 0 && msg.Type != types.PermissionType_ISSUER {
		return fmt.Errorf("validation_fees can only be specified for ISSUER permissions")
	}

	return nil
}

// [MOD-PERM-MSG-14-2-2] Create Permission permission checks
func (ms msgServer) validateCreatePermissionPermissionChecks(ctx sdk.Context, msg *types.MsgCreatePermission) error {
	// Load credential schema
	cs, err := ms.credentialSchemaKeeper.GetCredentialSchemaById(ctx, msg.SchemaId)
	if err != nil {
		return fmt.Errorf("credential schema not found: %w", err)
	}

	// if type is equal to ISSUER: if cs.issuer_perm_management_mode is not equal to OPEN, MUST abort
	if msg.Type == types.PermissionType_ISSUER {
		if cs.IssuerPermManagementMode != credentialschematypes.CredentialSchemaPermManagementMode_OPEN {
			return fmt.Errorf("issuer permission management mode is not OPEN")
		}
	}

	// if type is equal to VERIFIER: if cs.verifier_perm_management_mode is not equal to OPEN, MUST abort
	if msg.Type == types.PermissionType_VERIFIER {
		if cs.VerifierPermManagementMode != credentialschematypes.CredentialSchemaPermManagementMode_OPEN {
			return fmt.Errorf("verifier permission management mode is not OPEN")
		}

		// if type is equal to VERIFIER and validation_fees is specified and different than 0, MUST abort
		if msg.ValidationFees > 0 {
			return fmt.Errorf("validation_fees cannot be specified for VERIFIER permissions")
		}

		// if type is equal to VERIFIER and verification_fees is specified and different than 0, MUST abort
		if msg.VerificationFees > 0 {
			return fmt.Errorf("verification_fees cannot be specified for VERIFIER permissions")
		}
	}

	return nil
}

// [MOD-PERM-MSG-14-3] Create Permission execution
func (ms msgServer) executeCreatePermission(ctx sdk.Context, msg *types.MsgCreatePermission, now time.Time) (uint64, error) {
	// load the root permission of the schema (the one of type ECOSYSTEM) to ecosystem_perm_id
	ecosystemPerm, err := ms.findEcosystemPermission(ctx, msg.SchemaId)
	if err != nil {
		return 0, fmt.Errorf("failed to find ecosystem permission: %w", err)
	}

	// A new entry Permission perm MUST be created
	perm := types.Permission{
		// perm.id: auto-incremented uint64 (handled by CreatePermission)
		SchemaId:         msg.SchemaId,       // perm.schema_id: schema_id
		Modified:         &now,               // perm.modified: now
		Type:             msg.Type,           // perm.type: type
		Did:              msg.Did,            // perm.did: did
		Grantee:          msg.Creator,        // perm.grantee: account executing the method
		Created:          &now,               // perm.created: now
		CreatedBy:        msg.Creator,        // perm.created_by: account executing the method
		EffectiveFrom:    msg.EffectiveFrom,  // perm.effective_from: effective_from
		EffectiveUntil:   msg.EffectiveUntil, // perm.effective_until: effective_until
		Country:          msg.Country,        // perm.country: country
		ValidationFees:   0,                  // perm.validation_fees: validation_fees if specified and type is ISSUER, else 0
		IssuanceFees:     0,                  // perm.issuance_fees: 0
		VerificationFees: 0,                  // perm.verification_fees: verification_fees if specified and type is ISSUER, else 0
		Deposit:          0,                  // perm.deposit: 0
		ValidatorPermId:  ecosystemPerm.Id,   // perm.validator_perm_id: ecosystem_perm_id
	}

	// Set fees only for ISSUER permissions as per spec
	if msg.Type == types.PermissionType_ISSUER {
		perm.ValidationFees = msg.ValidationFees     // validation_fees if specified and type is ISSUER, else 0
		perm.VerificationFees = msg.VerificationFees // verification_fees if specified and type is ISSUER, else 0
	}

	// Store the permission
	id, err := ms.Keeper.CreatePermission(ctx, perm)
	if err != nil {
		return 0, fmt.Errorf("failed to create permission: %w", err)
	}

	return id, nil
}

// findEcosystemPermission finds the ecosystem permission for a given schema
func (ms msgServer) findEcosystemPermission(ctx sdk.Context, schemaId uint64) (types.Permission, error) {
	var foundPerm types.Permission
	var found bool

	// Iterate through all permissions to find the ecosystem permission for this schema
	err := ms.Permission.Walk(ctx, nil, func(id uint64, perm types.Permission) (stop bool, err error) {
		if perm.SchemaId == schemaId && perm.Type == types.PermissionType_ECOSYSTEM {
			foundPerm = perm
			found = true
			return true, nil
		}
		return false, nil
	})

	if err != nil {
		return types.Permission{}, fmt.Errorf("failed to iterate permissions: %w", err)
	}

	if !found {
		return types.Permission{}, fmt.Errorf("ecosystem permission not found for schema %d", schemaId)
	}

	return foundPerm, nil
}
