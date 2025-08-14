package keeper

import (
	"errors"
	"fmt"

	"cosmossdk.io/math"
	credentialschematypes "github.com/verana-labs/verana/x/credentialschema/types"

	"cosmossdk.io/collections"
	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"

	"time"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/verana-labs/verana/x/permission/types"
)

func (ms msgServer) validateSessionAccess(ctx sdk.Context, msg *types.MsgCreateOrUpdatePermissionSession) error {
	existingSession, err := ms.PermissionSession.Get(ctx, msg.Id)
	if err != nil {
		if errors.Is(err, collections.ErrNotFound) {
			return nil // New session case
		}
		return sdkerrors.ErrInvalidRequest.Wrapf("failed to get session: %v", err)
	}

	// Only session controller can update
	if existingSession.Controller != msg.Creator {
		return sdkerrors.ErrUnauthorized.Wrap("only session controller can update")
	}

	// Check for duplicate authorization
	for _, authz := range existingSession.Authz {
		if authz.ExecutorPermId == msg.IssuerPermId &&
			authz.BeneficiaryPermId == msg.VerifierPermId &&
			authz.WalletAgentPermId == msg.WalletAgentPermId {
			return sdkerrors.ErrInvalidRequest.Wrap("authorization already exists")
		}
	}

	return nil
}

func (ms msgServer) processFees(
	ctx sdk.Context,
	creator string,
	permSet []types.Permission,
	isVerifier bool,
	trustUnitPrice uint64,
	trustDepositRate math.LegacyDec,
) error {
	creatorAddr, err := sdk.AccAddressFromBech32(creator)
	if err != nil {
		return fmt.Errorf("invalid creator address: %w", err)
	}

	// Get the executor perm (issuer or verifier)
	var executorPerm types.Permission
	if isVerifier {
		// For verification, use the verifier perm
		executorPerm, err = ms.Permission.Get(ctx, permSet[0].ValidatorPermId)
	} else {
		// For issuance, use the issuer perm
		executorPerm, err = ms.Permission.Get(ctx, permSet[0].Id)
	}
	if err != nil {
		return fmt.Errorf("failed to get executor perm: %w", err)
	}

	// Process each perm's fees
	for _, perm := range permSet {
		var fees uint64
		if isVerifier {
			fees = perm.VerificationFees
		} else {
			fees = perm.IssuanceFees
		}

		if fees > 0 {
			// Calculate fees in denom
			feesInDenom := fees * trustUnitPrice

			// Calculate trust deposit amount
			trustDepositAmount := uint64(math.LegacyNewDec(int64(feesInDenom)).Mul(trustDepositRate).TruncateInt64())

			// Calculate direct fees (the portion that goes directly to the grantee)
			directFeesAmount := feesInDenom - trustDepositAmount

			// 1. Transfer direct fees from creator to perm grantee
			if directFeesAmount > 0 {
				granteeAddr, err := sdk.AccAddressFromBech32(perm.Grantee)
				if err != nil {
					return fmt.Errorf("invalid grantee address: %w", err)
				}

				err = ms.bankKeeper.SendCoins(
					ctx,
					creatorAddr,
					granteeAddr,
					sdk.NewCoins(sdk.NewInt64Coin(types.BondDenom, int64(directFeesAmount))),
				)
				if err != nil {
					return fmt.Errorf("failed to transfer direct fees: %w", err)
				}
			}

			// 2. Increase trust deposit for the grantee
			if trustDepositAmount > 0 {
				// First transfer funds from creator to module account
				err = ms.bankKeeper.SendCoinsFromAccountToModule(
					ctx,
					creatorAddr,
					types.ModuleName,
					sdk.NewCoins(sdk.NewInt64Coin(types.BondDenom, int64(trustDepositAmount))),
				)
				if err != nil {
					return fmt.Errorf("failed to transfer trust deposit to module: %w", err)
				}

				// Then adjust grantee's trust deposit
				err = ms.trustDeposit.AdjustTrustDeposit(
					ctx,
					perm.Grantee,
					int64(trustDepositAmount),
				)
				if err != nil {
					return fmt.Errorf("failed to adjust grantee trust deposit: %w", err)
				}

				// Update grantee's perm deposit
				perm.Deposit += trustDepositAmount
				if err := ms.Keeper.UpdatePermission(ctx, perm); err != nil {
					return fmt.Errorf("failed to update grantee perm deposit: %w", err)
				}

				// 3. Increase trust deposit for the creator (executor)
				err = ms.trustDeposit.AdjustTrustDeposit(
					ctx,
					creator,
					int64(trustDepositAmount),
				)
				if err != nil {
					return fmt.Errorf("failed to adjust creator trust deposit: %w", err)
				}

				// Update executor's perm deposit
				executorPerm.Deposit += trustDepositAmount
				if err := ms.Keeper.UpdatePermission(ctx, executorPerm); err != nil {
					return fmt.Errorf("failed to update executor perm deposit: %w", err)
				}
			}
		}
	}

	return nil
}

func (ms msgServer) createOrUpdateSession(ctx sdk.Context, msg *types.MsgCreateOrUpdatePermissionSession, now time.Time) error {
	session := &types.PermissionSession{
		Id:          msg.Id,
		Controller:  msg.Creator,
		AgentPermId: msg.AgentPermId,
		Modified:    &now,
	}

	existingSession, err := ms.PermissionSession.Get(ctx, msg.Id)
	if err == nil {
		// Update existing session
		session = &existingSession
		session.Modified = &now
	} else if errors.Is(err, collections.ErrNotFound) {
		// New session
		session.Created = &now
	} else {
		return err
	}

	// Add new authorization
	session.Authz = append(session.Authz, &types.SessionAuthz{
		ExecutorPermId:    msg.IssuerPermId,
		BeneficiaryPermId: msg.VerifierPermId,
		WalletAgentPermId: msg.WalletAgentPermId,
	})

	return ms.PermissionSession.Set(ctx, msg.Id, *session)
}

// findBeneficiaries gets the set of permissions that should receive fees
func (ms msgServer) findBeneficiaries(ctx sdk.Context, issuerPermId, verifierPermId uint64) ([]types.Permission, error) {
	var foundPerms []types.Permission
	var schemaID uint64

	// Helper function to check if a perm is already in the slice
	containsPerm := func(id uint64) bool {
		for _, p := range foundPerms {
			if p.Id == id {
				return true
			}
		}
		return false
	}

	// Get schema ID from either issuer or verifier perm
	if issuerPermId != 0 {
		issuerPerm, err := ms.Permission.Get(ctx, issuerPermId)
		if err != nil {
			return nil, fmt.Errorf("issuer perm not found: %w", err)
		}
		schemaID = issuerPerm.SchemaId
	} else if verifierPermId != 0 {
		verifierPerm, err := ms.Permission.Get(ctx, verifierPermId)
		if err != nil {
			return nil, fmt.Errorf("verifier perm not found: %w", err)
		}
		schemaID = verifierPerm.SchemaId
	} else {
		return nil, fmt.Errorf("at least one of issuer_perm_id or verifier_perm_id must be provided")
	}

	// Get schema to check perm management mode
	cs, err := ms.credentialSchemaKeeper.GetCredentialSchemaById(ctx, schemaID)
	if err != nil {
		return nil, fmt.Errorf("credential schema not found: %w", err)
	}

	// Check if schema is configured with OPEN perm management mode
	isOpenMode := false

	if (issuerPermId != 0 && cs.IssuerPermManagementMode == credentialschematypes.CredentialSchemaPermManagementMode_OPEN) ||
		(verifierPermId != 0 && cs.VerifierPermManagementMode == credentialschematypes.CredentialSchemaPermManagementMode_OPEN) {
		isOpenMode = true
	}

	// For OPEN mode, find the ECOSYSTEM perm
	if isOpenMode {
		// Find ECOSYSTEM perm for this schema
		err = ms.Permission.Walk(ctx, nil, func(id uint64, perm types.Permission) (bool, error) {
			if perm.SchemaId == schemaID &&
				perm.Type == types.PermissionType_PERMISSION_TYPE_ECOSYSTEM &&
				perm.Revoked == nil && perm.Terminated == nil && perm.SlashedDeposit == 0 {
				foundPerms = append(foundPerms, perm)
				return true, nil // Stop iteration once found
			}
			return false, nil
		})

		if err != nil {
			return nil, fmt.Errorf("failed to query ECOSYSTEM perm: %w", err)
		}

		// For OPEN mode, we only return the ECOSYSTEM perm as the beneficiary
		return foundPerms, nil
	}

	// Process issuer perm hierarchy if provided (non-OPEN mode)
	if issuerPermId != 0 {
		issuerPerm, err := ms.Permission.Get(ctx, issuerPermId)
		if err != nil {
			return nil, fmt.Errorf("issuer perm not found: %w", err)
		}

		// Follow the validator chain up
		if issuerPerm.ValidatorPermId != 0 {
			currentPermID := issuerPerm.ValidatorPermId
			for currentPermID != 0 {
				currentPerm, err := ms.Permission.Get(ctx, currentPermID)
				if err != nil {
					return nil, fmt.Errorf("failed to get perm: %w", err)
				}

				// Add to set if valid and not already included
				if currentPerm.Revoked == nil && currentPerm.Terminated == nil && currentPerm.SlashedDeposit == 0 && !containsPerm(currentPermID) {
					foundPerms = append(foundPerms, currentPerm)
				}

				// Move up
				currentPermID = currentPerm.ValidatorPermId
			}
		}
	}

	// Process verifier perm hierarchy if provided
	if verifierPermId != 0 {
		// First add issuer perm to the set if provided
		if issuerPermId != 0 {
			issuerPerm, err := ms.Permission.Get(ctx, issuerPermId)
			if err == nil && issuerPerm.Revoked == nil && issuerPerm.Terminated == nil && !containsPerm(issuerPermId) {
				foundPerms = append(foundPerms, issuerPerm)
			}
		}

		// Then process verifier's validator chain
		verifierPerm, err := ms.Permission.Get(ctx, verifierPermId)
		if err != nil {
			return nil, fmt.Errorf("verifier perm not found: %w", err)
		}

		if verifierPerm.ValidatorPermId != 0 {
			currentPermID := verifierPerm.ValidatorPermId
			for currentPermID != 0 {
				currentPerm, err := ms.Permission.Get(ctx, currentPermID)
				if err != nil {
					return nil, fmt.Errorf("failed to get perm: %w", err)
				}

				// Add to set if valid and not already included
				if currentPerm.Revoked == nil && currentPerm.Terminated == nil && currentPerm.SlashedDeposit == 0 && !containsPerm(currentPermID) {
					foundPerms = append(foundPerms, currentPerm)
				}

				// Move up
				currentPermID = currentPerm.ValidatorPermId
			}
		}
	}

	return foundPerms, nil
}
