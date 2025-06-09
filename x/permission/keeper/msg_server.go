package keeper

import (
	"context"
	"fmt"
	"strconv"
	"time"

	"cosmossdk.io/math"
	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"
	credentialschematypes "github.com/verana-labs/verana-blockchain/x/credentialschema/types"
	trustdeposittypes "github.com/verana-labs/verana-blockchain/x/trustdeposit/types"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/verana-labs/verana-blockchain/x/permission/types"
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
		return nil, fmt.Errorf("permission validation failed: %w", err)
	}

	// [MOD-PERM-MSG-1-2-3] Fee checks
	fees, deposit, err := ms.validateAndCalculateFees(ctx, msg.Creator, validatorPerm)
	if err != nil {
		return nil, fmt.Errorf("fee validation failed: %w", err)
	}

	// [MOD-PERM-MSG-1-3] Execute the permission VP creation
	permID, err := ms.executeStartPermissionVP(ctx, msg, validatorPerm, fees, deposit)
	if err != nil {
		return nil, fmt.Errorf("failed to execute permission VP: %w", err)
	}

	ctx.EventManager().EmitEvents(sdk.Events{
		sdk.NewEvent(
			types.EventTypeStartPermissionVP,
			sdk.NewAttribute(types.AttributeKeyPermissionID, strconv.FormatUint(permID, 10)),
			sdk.NewAttribute(types.AttributeKeyCreator, msg.Creator),
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
		return nil, fmt.Errorf("permission not found: %w", err)
	}

	// Verify creator is the grantee
	if applicantPerm.Grantee != msg.Creator {
		return nil, fmt.Errorf("creator is not the permission grantee")
	}

	// Get validator permission
	validatorPerm, err := ms.Keeper.GetPermissionByID(ctx, applicantPerm.ValidatorPermId)
	if err != nil {
		return nil, fmt.Errorf("validator permission not found: %w", err)
	}

	if err := IsValidPermission(validatorPerm, applicantPerm.Country, ctx.BlockTime()); err != nil {
		return nil, fmt.Errorf("validator permission is not valid: %w", err)
	}

	// [MOD-PERM-MSG-2-2-3] Fee checks
	validationFees, validationDeposit, err := ms.validateAndCalculateFees(ctx, msg.Creator, validatorPerm)
	if err != nil {
		return nil, fmt.Errorf("fee validation failed: %w", err)
	}

	// [MOD-PERM-MSG-2-3] Execution
	if err := ms.executeRenewPermissionVP(ctx, applicantPerm, validationFees, validationDeposit); err != nil {
		return nil, fmt.Errorf("failed to execute permission VP renewal: %w", err)
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

	// Update permission
	perm.VpState = types.ValidationState_VALIDATION_STATE_PENDING
	perm.VpLastStateChange = &now
	perm.Deposit += deposit
	perm.VpCurrentFees = fees
	perm.VpCurrentDeposit = deposit
	perm.Modified = &now

	// Store updated permission
	return ms.Keeper.UpdatePermission(ctx, perm)
}

func (ms msgServer) SetPermissionVPToValidated(goCtx context.Context, msg *types.MsgSetPermissionVPToValidated) (*types.MsgSetPermissionVPToValidatedResponse, error) {
	ctx := sdk.UnwrapSDKContext(goCtx)
	now := ctx.BlockTime()

	// [MOD-PERM-MSG-3-2-1] Basic checks
	applicantPerm, err := ms.Keeper.GetPermissionByID(ctx, msg.Id)
	if err != nil {
		return nil, fmt.Errorf("permission not found: %w", err)
	}

	// Check permission state - must be PENDING
	if applicantPerm.VpState != types.ValidationState_VALIDATION_STATE_PENDING {
		return nil, fmt.Errorf("permission must be in PENDING state to be validated")
	}

	// Check renewal-specific constraints
	if applicantPerm.EffectiveFrom != nil {
		if msg.ValidationFees != applicantPerm.ValidationFees {
			return nil, fmt.Errorf("validation fees cannot be changed during renewal")
		}
		if msg.IssuanceFees != applicantPerm.IssuanceFees {
			return nil, fmt.Errorf("issuance fees cannot be changed during renewal")
		}
		if msg.VerificationFees != applicantPerm.VerificationFees {
			return nil, fmt.Errorf("verification fees cannot be changed during renewal")
		}
		if msg.Country != applicantPerm.Country {
			return nil, fmt.Errorf("country cannot be changed during renewal")
		}
	}

	// Check summary digest SRI
	if applicantPerm.Type == types.PermissionType_PERMISSION_TYPE_HOLDER && msg.VpSummaryDigestSri != "" {
		return nil, fmt.Errorf("vp_summary_digest_sri must be null for HOLDER type")
	}

	// [MOD-PERM-MSG-3-2-2] Validator permission checks
	validatorPerm, err := ms.Keeper.GetPermissionByID(ctx, applicantPerm.ValidatorPermId)
	if err != nil {
		return nil, fmt.Errorf("validator permission not found: %w", err)
	}

	if validatorPerm.Grantee != msg.Creator {
		return nil, fmt.Errorf("creator is not the validator")
	}

	if err := IsValidPermission(validatorPerm, msg.Country, ctx.BlockTime()); err != nil {
		return nil, fmt.Errorf("validator permission is not valid: %w", err)
	}

	// Get validation period and calculate expiration
	cs, err := ms.credentialSchemaKeeper.GetCredentialSchemaById(ctx, applicantPerm.SchemaId)
	if err != nil {
		return nil, fmt.Errorf("credential schema not found: %w", err)
	}

	validityPeriod := getValidityPeriod(uint32(applicantPerm.Type), cs)
	vpExp := calculateVPExp(applicantPerm.VpExp, uint64(validityPeriod), now)

	// Special check for OPEN mode permissions to ensure proper fee handling
	// Even in OPEN mode, fees can still apply
	if applicantPerm.Type == types.PermissionType_PERMISSION_TYPE_ISSUER &&
		cs.IssuerPermManagementMode == credentialschematypes.CredentialSchemaPermManagementMode_OPEN {
		// Ensure validator is ECOSYSTEM for fee collection in OPEN mode
		if validatorPerm.Type != types.PermissionType_PERMISSION_TYPE_ECOSYSTEM {
			return nil, fmt.Errorf("validator must be ECOSYSTEM type for OPEN issuer permission")
		}
	} else if applicantPerm.Type == types.PermissionType_PERMISSION_TYPE_VERIFIER &&
		cs.VerifierPermManagementMode == credentialschematypes.CredentialSchemaPermManagementMode_OPEN {
		// Ensure validator is ECOSYSTEM for fee collection in OPEN mode
		if validatorPerm.Type != types.PermissionType_PERMISSION_TYPE_ECOSYSTEM {
			return nil, fmt.Errorf("validator must be ECOSYSTEM type for OPEN verifier permission")
		}
	}

	// Check effective_until if provided
	if msg.EffectiveUntil != nil {
		if applicantPerm.EffectiveUntil == nil {
			if !msg.EffectiveUntil.After(now) {
				return nil, fmt.Errorf("effective_until must be after current time")
			}
			if vpExp != nil && msg.EffectiveUntil.After(*vpExp) {
				return nil, fmt.Errorf("effective_until cannot be after validation expiration")
			}
		} else {
			if !msg.EffectiveUntil.After(*applicantPerm.EffectiveUntil) {
				return nil, fmt.Errorf("effective_until must be after current effective_until")
			}
			if vpExp != nil && msg.EffectiveUntil.After(*vpExp) {
				return nil, fmt.Errorf("effective_until cannot be after validation expiration")
			}
		}
	} else {
		msg.EffectiveUntil = vpExp
	}

	// [MOD-PERM-MSG-3-3] Execution
	if err := ms.executeSetPermissionVPToValidated(ctx, applicantPerm, msg, now, vpExp); err != nil {
		return nil, fmt.Errorf("failed to execute set to validated: %w", err)
	}

	return &types.MsgSetPermissionVPToValidatedResponse{}, nil
}

func (ms msgServer) RequestPermissionVPTermination(goCtx context.Context, msg *types.MsgRequestPermissionVPTermination) (*types.MsgRequestPermissionVPTerminationResponse, error) {
	ctx := sdk.UnwrapSDKContext(goCtx)
	now := ctx.BlockTime()

	// [MOD-PERM-MSG-4-2-1] Basic checks
	applicantPerm, err := ms.Keeper.GetPermissionByID(ctx, msg.Id)
	if err != nil {
		return nil, fmt.Errorf("permission not found: %w", err)
	}

	if applicantPerm.VpState != types.ValidationState_VALIDATION_STATE_VALIDATED {
		return nil, fmt.Errorf("permission must be in VALIDATED state")
	}

	// Check termination authorization
	if applicantPerm.VpExp != nil && now.After(*applicantPerm.VpExp) {
		// VP has expired - either party can terminate
		validatorPerm, err := ms.Keeper.GetPermissionByID(ctx, applicantPerm.ValidatorPermId)
		if err != nil {
			return nil, fmt.Errorf("validator permission not found: %w", err)
		}
		if msg.Creator != applicantPerm.Grantee && msg.Creator != validatorPerm.Grantee {
			return nil, fmt.Errorf("only grantee or validator can terminate expired VP")
		}
	} else {
		// VP not expired - only grantee can terminate
		if msg.Creator != applicantPerm.Grantee {
			return nil, fmt.Errorf("only grantee can terminate active VP")
		}
	}

	// [MOD-PERM-MSG-4-3] Execution
	err = ms.executeRequestPermissionVPTermination(ctx, applicantPerm, msg.Creator, now)
	if err != nil {
		return nil, fmt.Errorf("failed to execute termination request: %w", err)
	}

	return &types.MsgRequestPermissionVPTerminationResponse{}, nil
}

func (ms msgServer) executeRequestPermissionVPTermination(ctx sdk.Context, perm types.Permission, terminator string, now time.Time) error {
	// Update basic fields
	perm.Modified = &now
	perm.VpTermRequested = &now
	perm.VpLastStateChange = &now

	// Set state based on conditions
	if perm.Type != types.PermissionType_PERMISSION_TYPE_HOLDER || // not HOLDER
		(perm.VpExp != nil && now.After(*perm.VpExp)) { // expired
		// Immediate termination
		perm.VpState = types.ValidationState_VALIDATION_STATE_TERMINATED
		perm.Terminated = &now
		perm.TerminatedBy = terminator

		// Handle deposits
		if err := ms.handleTerminationDeposits(ctx, &perm); err != nil {
			return fmt.Errorf("failed to handle termination deposits: %w", err)
		}
	} else {
		// Request termination
		perm.VpState = types.ValidationState_VALIDATION_STATE_TERMINATION_REQUESTED
	}

	return ms.Keeper.UpdatePermission(ctx, perm)
}

func (ms msgServer) handleTerminationDeposits(ctx sdk.Context, perm *types.Permission) error {
	// Handle applicant's deposit
	if perm.Deposit > 0 {
		// Convert to signed integer for adjustment
		depositAmount := int64(perm.Deposit)

		// Use negative value to decrease deposit and increase claimable
		if err := ms.trustDeposit.AdjustTrustDeposit(ctx, perm.Grantee, -depositAmount); err != nil {
			return fmt.Errorf("failed to adjust applicant trust deposit: %w", err)
		}

		// Clear the deposit in the permission
		perm.Deposit = 0
	}

	// Handle validator's deposit
	if perm.VpValidatorDeposit > 0 {
		validatorPerm, err := ms.Keeper.GetPermissionByID(ctx, perm.ValidatorPermId)
		if err != nil {
			return fmt.Errorf("validator permission not found: %w", err)
		}

		// Convert to signed integer for adjustment
		validatorDepositAmount := int64(perm.VpValidatorDeposit)

		// Use negative value to decrease deposit and increase claimable
		if err := ms.trustDeposit.AdjustTrustDeposit(ctx, validatorPerm.Grantee, -validatorDepositAmount); err != nil {
			return fmt.Errorf("failed to adjust validator trust deposit: %w", err)
		}

		// Clear the validator deposit in the permission
		perm.VpValidatorDeposit = 0
	}
	return nil
}

func (ms msgServer) ConfirmPermissionVPTermination(goCtx context.Context, msg *types.MsgConfirmPermissionVPTermination) (*types.MsgConfirmPermissionVPTerminationResponse, error) {
	ctx := sdk.UnwrapSDKContext(goCtx)
	now := ctx.BlockTime()

	// Load applicant permission
	applicantPerm, err := ms.Keeper.GetPermissionByID(ctx, msg.Id)
	if err != nil {
		return nil, fmt.Errorf("permission not found: %w", err)
	}

	// Check permission state
	if applicantPerm.VpState != types.ValidationState_VALIDATION_STATE_TERMINATION_REQUESTED {
		return nil, fmt.Errorf("permission must be in TERMINATION_REQUESTED state")
	}

	// [MOD-PERM-MSG-5-2-2] Permission checks
	validatorPerm, err := ms.Keeper.GetPermissionByID(ctx, applicantPerm.ValidatorPermId)
	if err != nil {
		return nil, fmt.Errorf("validator permission not found: %w", err)
	}

	// Calculate timeout
	termRequestTimeout := applicantPerm.VpTermRequested.AddDate(0, 0, int(ms.Keeper.GetParams(ctx).ValidationTermRequestedTimeoutDays))
	timeoutReached := now.After(termRequestTimeout)

	// Check authorization
	if !timeoutReached {
		// Before timeout: only validator can confirm
		if msg.Creator != validatorPerm.Grantee {
			return nil, fmt.Errorf("only validator can confirm termination before timeout")
		}
	} else {
		// After timeout: either validator or applicant can confirm
		if msg.Creator != validatorPerm.Grantee && msg.Creator != applicantPerm.Grantee {
			return nil, fmt.Errorf("only validator or applicant can confirm termination after timeout")
		}
	}

	// [MOD-PERM-MSG-5-3] Execution
	if err := ms.executeConfirmPermissionVPTermination(ctx, applicantPerm, validatorPerm, msg.Creator, now); err != nil {
		return nil, fmt.Errorf("failed to execute termination confirmation: %w", err)
	}

	return &types.MsgConfirmPermissionVPTerminationResponse{}, nil
}

func (ms msgServer) executeConfirmPermissionVPTermination(ctx sdk.Context, applicantPerm, validatorPerm types.Permission, confirmer string, now time.Time) error {
	// Update basic fields
	applicantPerm.Modified = &now
	applicantPerm.VpState = types.ValidationState_VALIDATION_STATE_TERMINATED
	applicantPerm.VpLastStateChange = &now
	applicantPerm.Terminated = &now
	applicantPerm.TerminatedBy = confirmer

	// Handle applicant's deposit
	if applicantPerm.Deposit > 0 {
		// Convert to signed integer for adjustment
		depositAmount := int64(applicantPerm.Deposit)

		// Use negative value to decrease deposit and increase claimable
		err := ms.trustDeposit.AdjustTrustDeposit(ctx, applicantPerm.Grantee, -depositAmount)
		if err != nil {
			return fmt.Errorf("failed to adjust applicant trust deposit: %w", err)
		}

		// Clear the deposit in the permission
		applicantPerm.Deposit = 0
	}

	// Only return validator deposit if validator confirmed
	if confirmer == validatorPerm.Grantee && applicantPerm.VpValidatorDeposit > 0 {
		// Convert to signed integer for adjustment
		validatorDepositAmount := int64(applicantPerm.VpValidatorDeposit)

		// Use negative value to decrease deposit and increase claimable
		err := ms.trustDeposit.AdjustTrustDeposit(ctx, validatorPerm.Grantee, -validatorDepositAmount)
		if err != nil {
			return fmt.Errorf("failed to adjust validator trust deposit: %w", err)
		}

		// Clear the validator deposit in the permission
		applicantPerm.VpValidatorDeposit = 0
	}

	// Persist changes
	return ms.Keeper.UpdatePermission(ctx, applicantPerm)
}

func (ms msgServer) CancelPermissionVPLastRequest(goCtx context.Context, msg *types.MsgCancelPermissionVPLastRequest) (*types.MsgCancelPermissionVPLastRequestResponse, error) {
	ctx := sdk.UnwrapSDKContext(goCtx)

	// Load applicant permission
	applicantPerm, err := ms.Keeper.GetPermissionByID(ctx, msg.Id)
	if err != nil {
		return nil, fmt.Errorf("permission not found: %w", err)
	}

	// Check if creator is the grantee
	if applicantPerm.Grantee != msg.Creator {
		return nil, fmt.Errorf("creator is not the permission grantee")
	}

	// Check permission state
	if applicantPerm.VpState != types.ValidationState_VALIDATION_STATE_PENDING {
		return nil, fmt.Errorf("permission must be in PENDING state")
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
		perm.VpState = types.ValidationState_VALIDATION_STATE_TERMINATED
	} else {
		perm.VpState = types.ValidationState_VALIDATION_STATE_VALIDATED
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

	// Add this check inside CreateRootPermission after the schema check
	if msg.EffectiveFrom != nil && !msg.EffectiveFrom.After(now) {
		return nil, fmt.Errorf("effective_from must be in the future")
	}

	// Check credential schema exists
	_, err := ms.credentialSchemaKeeper.GetCredentialSchemaById(ctx, msg.SchemaId)
	if err != nil {
		return nil, fmt.Errorf("credential schema not found: %w", err)
	}

	// [MOD-PERM-MSG-7-2-2] Permission checks
	if err := ms.validateCreateRootPermissionAuthority(ctx, msg); err != nil {
		return nil, err
	}

	// [MOD-PERM-MSG-7-3] Execution
	id, err := ms.executeCreateRootPermission(ctx, msg, now)
	if err != nil {
		return nil, fmt.Errorf("failed to create root permission: %w", err)
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

	// Check if creator is the controller
	if tr.Controller != msg.Creator {
		return fmt.Errorf("creator is not the trust registry controller")
	}

	return nil
}

func (ms msgServer) executeCreateRootPermission(ctx sdk.Context, msg *types.MsgCreateRootPermission, now time.Time) (uint64, error) {
	// Create new permission
	perm := types.Permission{
		SchemaId:         msg.SchemaId,
		Type:             types.PermissionType_PERMISSION_TYPE_ECOSYSTEM,
		Did:              msg.Did,
		Grantee:          msg.Creator,
		Created:          &now,
		CreatedBy:        msg.Creator,
		Modified:         &now,
		EffectiveFrom:    msg.EffectiveFrom,
		EffectiveUntil:   msg.EffectiveUntil,
		Country:          msg.Country,
		ValidationFees:   msg.ValidationFees,
		IssuanceFees:     msg.IssuanceFees,
		VerificationFees: msg.VerificationFees,
		Deposit:          0,
	}

	// Store the permission
	id, err := ms.Keeper.CreatePermission(ctx, perm)
	if err != nil {
		return 0, fmt.Errorf("failed to create permission: %w", err)
	}

	return id, nil
}

func (ms msgServer) ExtendPermission(goCtx context.Context, msg *types.MsgExtendPermission) (*types.MsgExtendPermissionResponse, error) {
	ctx := sdk.UnwrapSDKContext(goCtx)
	now := ctx.BlockTime()

	// [MOD-PERM-MSG-8-2-1] Basic checks
	applicantPerm, err := ms.Keeper.GetPermissionByID(ctx, msg.Id)
	if err != nil {
		return nil, fmt.Errorf("permission not found: %w", err)
	}

	// Check effective_until is after current effective_until
	if applicantPerm.EffectiveUntil != nil && !msg.EffectiveUntil.After(*applicantPerm.EffectiveUntil) {
		return nil, fmt.Errorf("effective_until must be after current effective_until")
	}

	// Check effective_until is before or equal to vp_exp
	if applicantPerm.VpExp != nil && msg.EffectiveUntil.After(*applicantPerm.VpExp) {
		return nil, fmt.Errorf("effective_until cannot be after validation expiration")
	}

	// [MOD-PERM-MSG-8-2-2] Validator permission checks
	if err := ms.validateExtendPermissionAuthority(ctx, msg.Creator, applicantPerm); err != nil {
		return nil, err
	}

	// [MOD-PERM-MSG-8-3] Execution
	if err := ms.executeExtendPermission(ctx, applicantPerm, msg.Creator, msg.EffectiveUntil, now); err != nil {
		return nil, fmt.Errorf("failed to extend permission: %w", err)
	}

	return &types.MsgExtendPermissionResponse{}, nil
}

func (ms msgServer) validateExtendPermissionAuthority(ctx sdk.Context, creator string, perm types.Permission) error {
	if perm.ValidatorPermId == 0 {
		// For TRUST_REGISTRY type, creator must be the grantee
		if perm.Type == types.PermissionType_PERMISSION_TYPE_ECOSYSTEM {
			if perm.Grantee != creator {
				return fmt.Errorf("creator is not the permission grantee")
			}
		} else {
			return fmt.Errorf("invalid permission type for extension")
		}
	} else {
		// For other types, creator must be the validator
		validatorPerm, err := ms.Keeper.GetPermissionByID(ctx, perm.ValidatorPermId)
		if err != nil {
			return fmt.Errorf("validator permission not found: %w", err)
		}

		if err = IsValidPermission(validatorPerm, perm.Country, ctx.BlockTime()); err != nil {
			return fmt.Errorf("validator permission is not valid: %w", err)
		}

		if validatorPerm.Grantee != creator {
			return fmt.Errorf("creator is not the validator")
		}
	}
	return nil
}

func (ms msgServer) executeExtendPermission(ctx sdk.Context, perm types.Permission, creator string, effectiveUntil *time.Time, now time.Time) error {
	perm.EffectiveUntil = effectiveUntil
	perm.Extended = &now
	perm.Modified = &now
	perm.ExtendedBy = creator

	return ms.Keeper.UpdatePermission(ctx, perm)
}

// RevokePermission handles the MsgRevokePermission message
func (ms msgServer) RevokePermission(goCtx context.Context, msg *types.MsgRevokePermission) (*types.MsgRevokePermissionResponse, error) {
	ctx := sdk.UnwrapSDKContext(goCtx)
	now := ctx.BlockTime()

	// [MOD-PERM-MSG-9-2-1] Basic checks
	applicantPerm, err := ms.Keeper.GetPermissionByID(ctx, msg.Id)
	if err != nil {
		return nil, fmt.Errorf("permission not found: %w", err)
	}

	// [MOD-PERM-MSG-9-2-2] Validator permission checks
	validatorPerm, err := ms.Keeper.GetPermissionByID(ctx, applicantPerm.ValidatorPermId)
	if err != nil {
		return nil, fmt.Errorf("validator permission not found: %w", err)
	}

	if err := IsValidPermission(validatorPerm, applicantPerm.Country, ctx.BlockTime()); err != nil {
		return nil, fmt.Errorf("validator permission is not valid: %w", err)
	}

	if validatorPerm.Grantee != msg.Creator {
		return nil, fmt.Errorf("creator is not the validator")
	}

	// [MOD-PERM-MSG-9-3] Execution
	if err := ms.executeRevokePermission(ctx, applicantPerm, msg.Creator, now); err != nil {
		return nil, fmt.Errorf("failed to revoke permission: %w", err)
	}

	return &types.MsgRevokePermissionResponse{}, nil
}

func (ms msgServer) executeRevokePermission(ctx sdk.Context, perm types.Permission, creator string, now time.Time) error {
	perm.Revoked = &now
	perm.Modified = &now
	perm.RevokedBy = creator

	return ms.Keeper.UpdatePermission(ctx, perm)
}

type PermissionSet []types.Permission

func (ps PermissionSet) contains(id uint64) bool {
	for _, perm := range ps {
		if perm.Id == id {
			return true
		}
	}
	return false
}

func (ms msgServer) CreateOrUpdatePermissionSession(goCtx context.Context, msg *types.MsgCreateOrUpdatePermissionSession) (*types.MsgCreateOrUpdatePermissionSessionResponse, error) {
	ctx := sdk.UnwrapSDKContext(goCtx)
	now := ctx.BlockTime()

	// Validate session access
	if err := ms.validateSessionAccess(ctx, msg); err != nil {
		return nil, err
	}

	// Define variables for issuerPerm and verifierPerm
	var verifierPerm *types.Permission

	// Load and validate issuer permission if specified
	if msg.IssuerPermId != 0 {
		perm, err := ms.Permission.Get(ctx, msg.IssuerPermId)
		if err != nil {
			return nil, sdkerrors.ErrNotFound.Wrapf("issuer permission not found: %v", err)
		}

		if perm.Type != types.PermissionType_PERMISSION_TYPE_ISSUER {
			return nil, sdkerrors.ErrInvalidRequest.Wrap("issuer permission must be ISSUER type")
		}

		if perm.Revoked != nil || perm.Terminated != nil || perm.SlashedDeposit > 0 {
			return nil, sdkerrors.ErrInvalidRequest.Wrap("issuer permission is revoked, terminated, or slashed")
		}
	}

	// Load and validate verifier permission if specified
	if msg.VerifierPermId != 0 {
		perm, err := ms.Permission.Get(ctx, msg.VerifierPermId)
		if err != nil {
			return nil, sdkerrors.ErrNotFound.Wrapf("verifier permission not found: %v", err)
		}

		if perm.Type != types.PermissionType_PERMISSION_TYPE_VERIFIER {
			return nil, sdkerrors.ErrInvalidRequest.Wrap("verifier permission must be VERIFIER type")
		}

		if perm.Revoked != nil || perm.Terminated != nil || perm.SlashedDeposit > 0 {
			return nil, sdkerrors.ErrInvalidRequest.Wrap("verifier permission is revoked, terminated, or slashed")
		}

		verifierPerm = &perm
	}

	// Validate agent permission
	agentPerm, err := ms.Permission.Get(ctx, msg.AgentPermId)
	if err != nil {
		return nil, sdkerrors.ErrNotFound.Wrap("agent permission not found")
	}

	if agentPerm.Type != types.PermissionType_PERMISSION_TYPE_HOLDER {
		return nil, sdkerrors.ErrInvalidRequest.Wrap("agent permission must be HOLDER type")
	}

	if agentPerm.Revoked != nil || agentPerm.Terminated != nil || agentPerm.SlashedDeposit > 0 {
		return nil, sdkerrors.ErrInvalidRequest.Wrap("agent permission is revoked, terminated, or slashed")
	}

	// Validate wallet agent permission if provided
	if msg.WalletAgentPermId != 0 {
		perm, err := ms.Permission.Get(ctx, msg.WalletAgentPermId)
		if err != nil {
			return nil, sdkerrors.ErrNotFound.Wrap("wallet agent permission not found")
		}

		if perm.Type != types.PermissionType_PERMISSION_TYPE_HOLDER {
			return nil, sdkerrors.ErrInvalidRequest.Wrap("wallet agent permission must be HOLDER type")
		}

		if perm.Revoked != nil || perm.Terminated != nil || perm.SlashedDeposit > 0 {
			return nil, sdkerrors.ErrInvalidRequest.Wrap("wallet agent permission is revoked, terminated, or slashed")
		}

	}

	// Get beneficiary permissions set
	foundPermSet, err := ms.findBeneficiaries(ctx, msg.IssuerPermId, msg.VerifierPermId)
	if err != nil {
		return nil, fmt.Errorf("failed to find beneficiaries: %w", err)
	}

	// Calculate fees
	trustUnitPrice := ms.trustRegistryKeeper.GetTrustUnitPrice(ctx)
	trustDepositRate := ms.trustDeposit.GetTrustDepositRate(ctx)
	userAgentRewardRate := ms.trustDeposit.GetUserAgentRewardRate(ctx)
	walletUserAgentRewardRate := ms.trustDeposit.GetWalletUserAgentRewardRate(ctx)

	// Calculate beneficiary fees
	beneficiaryFees := uint64(0)
	for _, perm := range foundPermSet {
		if verifierPerm != nil {
			beneficiaryFees += perm.VerificationFees
		} else {
			beneficiaryFees += perm.IssuanceFees
		}
	}

	// Calculate total required funds
	totalFees := beneficiaryFees * trustUnitPrice
	trustFees := uint64(math.LegacyNewDec(int64(totalFees)).Mul(trustDepositRate).TruncateInt64())
	rewardRate := userAgentRewardRate.Add(walletUserAgentRewardRate)
	rewards := uint64(math.LegacyNewDec(int64(totalFees)).Mul(rewardRate).TruncateInt64())

	// Calculate required balance
	requiredAmount := sdk.NewInt64Coin(types.BondDenom, int64(totalFees+trustFees+rewards))

	// Validate sender has sufficient balance
	creatorAddr, err := sdk.AccAddressFromBech32(msg.Creator)
	if err != nil {
		return nil, fmt.Errorf("invalid creator address: %w", err)
	}

	if !ms.bankKeeper.HasBalance(ctx, creatorAddr, requiredAmount) {
		return nil, sdkerrors.ErrInsufficientFunds.Wrapf("insufficient funds: required %s", requiredAmount)
	}

	// Process fees
	if err := ms.processFees(ctx, msg.Creator, foundPermSet, verifierPerm != nil, trustUnitPrice, trustDepositRate); err != nil {
		return nil, fmt.Errorf("failed to process fees: %w", err)
	}

	// Create or update session
	if err := ms.createOrUpdateSession(ctx, msg, now); err != nil {
		return nil, fmt.Errorf("failed to create/update session: %w", err)
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

	// Load Permission entry applicant_perm from id
	applicantPerm, err := ms.Keeper.GetPermissionByID(ctx, msg.Id)
	if err != nil {
		return nil, fmt.Errorf("permission not found: %w", err)
	}

	// applicant_perm MUST be a valid permission
	//if applicantPerm.Revoked != nil || applicantPerm.Terminated != nil {
	//	return nil, fmt.Errorf("permission is not valid (revoked or terminated)")
	//}

	// amount MUST be lower or equal to applicant_perm.deposit
	if msg.Amount > applicantPerm.Deposit {
		return nil, fmt.Errorf("amount exceeds available deposit: %d > %d", msg.Amount, applicantPerm.Deposit)
	}

	// [MOD-PERM-MSG-12-2-2] Slash Permission Trust Deposit validator perms
	hasSlashingAuthority := false

	// Option #1: executed by validator
	if applicantPerm.ValidatorPermId != 0 {
		validatorPerm, err := ms.Keeper.GetPermissionByID(ctx, applicantPerm.ValidatorPermId)
		if err == nil && validatorPerm.Revoked == nil && validatorPerm.Terminated == nil {
			if validatorPerm.Grantee == msg.Creator {
				hasSlashingAuthority = true
			}
		}
	}

	// Option #2: executed by ecosystem controller
	if !hasSlashingAuthority {
		ecosystemPerm, err := ms.findEcosystemPermissionForSchema(ctx, applicantPerm.SchemaId)
		if err == nil {
			if ecosystemPerm.Grantee == msg.Creator {
				hasSlashingAuthority = true
			}
		}
	}

	if !hasSlashingAuthority {
		return nil, fmt.Errorf("creator does not have authority to slash this permission")
	}

	// [MOD-PERM-MSG-12-2-3] Slash Permission Trust Deposit fee checks
	// Account executing the method MUST have the required estimated transaction fees
	// This is handled by the SDK automatically

	// [MOD-PERM-MSG-12-3] Slash Permission Trust Deposit execution
	if err := ms.executeSlashPermissionTrustDeposit(ctx, applicantPerm, msg.Amount, msg.Creator); err != nil {
		return nil, fmt.Errorf("failed to slash permission trust deposit: %w", err)
	}

	// Emit event
	ctx.EventManager().EmitEvents(sdk.Events{
		sdk.NewEvent(
			types.EventTypeSlashPermissionTrustDeposit,
			sdk.NewAttribute(types.AttributeKeyPermissionID, strconv.FormatUint(msg.Id, 10)),
			sdk.NewAttribute(types.AttributeKeySlashedAmount, strconv.FormatUint(msg.Amount, 10)),
			sdk.NewAttribute(types.AttributeKeySlashedBy, msg.Creator),
			sdk.NewAttribute(types.AttributeKeyTimestamp, ctx.BlockTime().String()),
		),
	})

	return &types.MsgSlashPermissionTrustDepositResponse{}, nil
}

// executeSlashPermissionTrustDeposit performs the actual slashing execution
func (ms msgServer) executeSlashPermissionTrustDeposit(ctx sdk.Context, applicantPerm types.Permission, amount uint64, slashedBy string) error {
	now := ctx.BlockTime()

	// Update Permission entry applicant_perm
	applicantPerm.Slashed = &now
	applicantPerm.Modified = &now
	applicantPerm.SlashedDeposit = applicantPerm.SlashedDeposit + amount
	applicantPerm.SlashedBy = slashedBy

	// This functionality doesn't exist yet, so commenting out for now
	if err := ms.trustDeposit.BurnEcosystemSlashedTrustDeposit(ctx, applicantPerm.Grantee, amount); err != nil {
		return fmt.Errorf("failed to burn trust deposit: %w", err)
	}

	// Update permission
	if err := ms.Keeper.UpdatePermission(ctx, applicantPerm); err != nil {
		return fmt.Errorf("failed to update permission: %w", err)
	}

	return nil
}

// findEcosystemPermissionForSchema finds the ECOSYSTEM permission for a given schema
func (ms msgServer) findEcosystemPermissionForSchema(ctx sdk.Context, schemaId uint64) (types.Permission, error) {
	var ecosystemPerm types.Permission
	var found bool

	err := ms.Permission.Walk(ctx, nil, func(key uint64, perm types.Permission) (bool, error) {
		if perm.Type == types.PermissionType_PERMISSION_TYPE_ECOSYSTEM &&
			perm.SchemaId == schemaId &&
			perm.Revoked == nil &&
			perm.Terminated == nil {
			ecosystemPerm = perm
			found = true
			return true, nil // Stop iteration
		}
		return false, nil
	})

	if err != nil {
		return types.Permission{}, err
	}

	if !found {
		return types.Permission{}, fmt.Errorf("ecosystem permission not found for schema %d", schemaId)
	}

	return ecosystemPerm, nil
}

// RepayPermissionSlashedTrustDeposit handles the MsgRepayPermissionSlashedTrustDeposit message
func (ms msgServer) RepayPermissionSlashedTrustDeposit(goCtx context.Context, msg *types.MsgRepayPermissionSlashedTrustDeposit) (*types.MsgRepayPermissionSlashedTrustDepositResponse, error) {
	ctx := sdk.UnwrapSDKContext(goCtx)

	// Load Permission entry applicant_perm from id
	applicantPerm, err := ms.Keeper.GetPermissionByID(ctx, msg.Id)
	if err != nil {
		return nil, fmt.Errorf("permission not found: %w", err)
	}

	// Check if permission has been slashed
	if applicantPerm.SlashedDeposit == 0 {
		return nil, fmt.Errorf("permission has no slashed deposit to repay")
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
		return nil, fmt.Errorf("failed to repay permission slashed trust deposit: %w", err)
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

	// Update permission
	if err := ms.Keeper.UpdatePermission(ctx, applicantPerm); err != nil {
		return fmt.Errorf("failed to update permission: %w", err)
	}

	return nil
}

// CreatePermission handles the MsgCreatePermission message
func (ms msgServer) CreatePermission(goCtx context.Context, msg *types.MsgCreatePermission) (*types.MsgCreatePermissionResponse, error) {
	ctx := sdk.UnwrapSDKContext(goCtx)

	// type MUST be ISSUER or VERIFIER
	if msg.Type != types.PermissionType_PERMISSION_TYPE_ISSUER &&
		msg.Type != types.PermissionType_PERMISSION_TYPE_VERIFIER {
		return nil, fmt.Errorf("type must be ISSUER or VERIFIER")
	}

	// effective_from must be in the future
	now := ctx.BlockTime()
	if msg.EffectiveFrom != nil && !msg.EffectiveFrom.After(now) {
		return nil, fmt.Errorf("effective_from must be in the future")
	}

	// effective_until must be greater than effective_from
	if msg.EffectiveUntil != nil && msg.EffectiveFrom != nil {
		if !msg.EffectiveUntil.After(*msg.EffectiveFrom) {
			return nil, fmt.Errorf("effective_until must be greater than effective_from")
		}
	}

	// country validation
	if msg.Country != "" && !isValidCountryCode(msg.Country) {
		return nil, fmt.Errorf("invalid country code format")
	}

	// verification_fees must be >= 0 (uint64 is naturally >= 0)

	// Load credential schema
	cs, err := ms.credentialSchemaKeeper.GetCredentialSchemaById(ctx, msg.SchemaId)
	if err != nil {
		return nil, fmt.Errorf("credential schema not found: %w", err)
	}

	// [MOD-PERM-MSG-14-2-2] Create Permission permission checks
	if msg.Type == types.PermissionType_PERMISSION_TYPE_ISSUER {
		if cs.IssuerPermManagementMode != credentialschematypes.CredentialSchemaPermManagementMode_OPEN {
			return nil, fmt.Errorf("issuer permission management mode is not OPEN")
		}
	} else if msg.Type == types.PermissionType_PERMISSION_TYPE_VERIFIER {
		if cs.VerifierPermManagementMode != credentialschematypes.CredentialSchemaPermManagementMode_OPEN {
			return nil, fmt.Errorf("verifier permission management mode is not OPEN")
		}
	}

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

// executeCreatePermission performs the actual permission creation
func (ms msgServer) executeCreatePermission(ctx sdk.Context, msg *types.MsgCreatePermission, now time.Time) (uint64, error) {
	// Load credential schema
	cs, err := ms.credentialSchemaKeeper.GetCredentialSchemaById(ctx, msg.SchemaId)
	if err != nil {
		return 0, fmt.Errorf("credential schema not found: %w", err)
	}

	// Find the ecosystem permission for this schema
	ecosystemPerm, err := ms.findEcosystemPermission(ctx, cs)
	if err != nil {
		return 0, fmt.Errorf("failed to find ecosystem permission: %w", err)
	}

	// Create new Permission entry
	perm := types.Permission{
		SchemaId:         msg.SchemaId,
		Type:             msg.Type,
		Did:              msg.Did,
		Grantee:          msg.Creator,
		Created:          &now,
		CreatedBy:        msg.Creator,
		Modified:         &now,
		EffectiveFrom:    msg.EffectiveFrom,
		EffectiveUntil:   msg.EffectiveUntil,
		Country:          msg.Country,
		VerificationFees: msg.VerificationFees,
		VpState:          types.ValidationState_VALIDATION_STATE_VALIDATED,
		ValidatorPermId:  ecosystemPerm.Id, // Set the ecosystem permission as the validator
	}

	// Store the permission
	id, err := ms.Keeper.CreatePermission(ctx, perm)
	if err != nil {
		return 0, fmt.Errorf("failed to create permission: %w", err)
	}

	return id, nil
}

// findEcosystemPermission finds the ecosystem permission for a given credential schema
func (ms msgServer) findEcosystemPermission(ctx sdk.Context, cs credentialschematypes.CredentialSchema) (types.Permission, error) {
	var foundPerm types.Permission
	var found bool

	// Iterate through all permissions to find the ecosystem permission for this schema
	err := ms.Permission.Walk(ctx, nil, func(id uint64, perm types.Permission) (stop bool, err error) {
		if perm.SchemaId == cs.Id && perm.Type == types.PermissionType_PERMISSION_TYPE_ECOSYSTEM {
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
		return types.Permission{}, fmt.Errorf("ecosystem permission not found for schema %d", cs.Id)
	}

	return foundPerm, nil
}
