package keeper

import (
	"cosmossdk.io/math"
	"errors"
	"fmt"
	credentialschematypes "github.com/verana-labs/verana/x/cs/types"

	"cosmossdk.io/collections"
	"time"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/verana-labs/verana/x/perm/types"
)

// [MOD-PERM-MSG-10-2] Create or Update Permission Session precondition checks
func (ms msgServer) validateCreateOrUpdatePermissionSessionPreconditions(ctx sdk.Context, msg *types.MsgCreateOrUpdatePermissionSession, now time.Time) error {
	// if issuer_perm_id is null AND verifier_perm_id is null, MUST abort
	if msg.IssuerPermId == 0 && msg.VerifierPermId == 0 {
		return fmt.Errorf("at least one of issuer_perm_id or verifier_perm_id must be provided")
	}

	// Validate session access for updates
	if err := ms.validateSessionAccess(ctx, msg); err != nil {
		return err
	}

	// if issuer_perm_id is not null
	if msg.IssuerPermId != 0 {
		issuerPerm, err := ms.Permission.Get(ctx, msg.IssuerPermId)
		if err != nil {
			return fmt.Errorf("issuer permission not found: %w", err)
		}

		// if issuer_perm.type is not ISSUER, abort
		if issuerPerm.Type != types.PermissionType_ISSUER {
			return fmt.Errorf("issuer permission must be ISSUER type")
		}

		// if issuer_perm is not a valid permission, abort
		if err := IsValidPermission(issuerPerm, issuerPerm.Country, now); err != nil {
			return fmt.Errorf("issuer permission is not valid: %w", err)
		}
	}

	// if verifier_perm_id is not null
	if msg.VerifierPermId != 0 {
		verifierPerm, err := ms.Permission.Get(ctx, msg.VerifierPermId)
		if err != nil {
			return fmt.Errorf("verifier permission not found: %w", err)
		}

		// if verifier_perm.type is not VERIFIER, abort
		if verifierPerm.Type != types.PermissionType_VERIFIER {
			return fmt.Errorf("verifier permission must be VERIFIER type")
		}

		// if verifier_perm is not a valid permission, abort
		if err := IsValidPermission(verifierPerm, verifierPerm.Country, now); err != nil {
			return fmt.Errorf("verifier permission is not valid: %w", err)
		}
	}

	// agent: Load agent_perm from agent_perm_id
	agentPerm, err := ms.Permission.Get(ctx, msg.AgentPermId)
	if err != nil {
		return fmt.Errorf("agent permission not found: %w", err)
	}

	// if agent_perm.type is not ISSUER, abort
	if agentPerm.Type != types.PermissionType_ISSUER {
		return fmt.Errorf("agent permission must be ISSUER type")
	}

	// if agent_perm is not a valid permission, abort
	if err := IsValidPermission(agentPerm, agentPerm.Country, now); err != nil {
		return fmt.Errorf("agent permission is not valid: %w", err)
	}

	// wallet_agent: Load wallet_agent_perm from wallet_agent_perm_id
	if msg.WalletAgentPermId != 0 {
		walletAgentPerm, err := ms.Permission.Get(ctx, msg.WalletAgentPermId)
		if err != nil {
			return fmt.Errorf("wallet agent permission not found: %w", err)
		}

		// if wallet_agent_perm.type is not ISSUER, abort
		if walletAgentPerm.Type != types.PermissionType_ISSUER {
			return fmt.Errorf("wallet agent permission must be ISSUER type")
		}

		// if wallet_agent_perm is not a valid permission, abort
		if err := IsValidPermission(walletAgentPerm, walletAgentPerm.Country, now); err != nil {
			return fmt.Errorf("wallet agent permission is not valid: %w", err)
		}
	}

	return nil
}

// [MOD-PERM-MSG-10-3] Create or Update Permission Session fee checks
func (ms msgServer) validateCreateOrUpdatePermissionSessionFees(ctx sdk.Context, msg *types.MsgCreateOrUpdatePermissionSession) ([]types.Permission, uint64, uint64, error) {
	// use "Find Beneficiaries" query method to get the set of beneficiary permission found_perm_set
	foundPermSet, err := ms.findBeneficiaries(ctx, msg.IssuerPermId, msg.VerifierPermId)
	if err != nil {
		return nil, 0, 0, fmt.Errorf("failed to find beneficiaries: %w", err)
	}

	// calculate the required beneficiary fees
	beneficiaryFees := uint64(0)
	verifierPerm := msg.VerifierPermId != 0

	for _, perm := range foundPermSet {
		if verifierPerm {
			// if verifier_perm is NOT null: iterate over permissions perm of found_perm_set and set beneficiary_fees = beneficiary_fees + perm.verification_fees
			beneficiaryFees += perm.VerificationFees
		} else {
			// if verifier_perm is null: iterate over permissions perm of found_perm_set and set beneficiary_fees = beneficiary_fees + perm.issuance_fees
			beneficiaryFees += perm.IssuanceFees
		}
	}

	// Get global variables for calculations
	userAgentRewardRate := ms.trustDeposit.GetUserAgentRewardRate(ctx)
	walletUserAgentRewardRate := ms.trustDeposit.GetWalletUserAgentRewardRate(ctx)
	trustDepositRate := ms.trustDeposit.GetTrustDepositRate(ctx)
	trustUnitPrice := ms.trustRegistryKeeper.GetTrustUnitPrice(ctx)

	// Calculate trust_fees = beneficiary_fees * (1 + (user_agent_reward_rate + wallet_user_agent_reward_rate)) * (1 + trust_deposit_rate) * trust_unit_price
	agentRewardRate := userAgentRewardRate.Add(walletUserAgentRewardRate)
	multiplier := math.LegacyOneDec().Add(agentRewardRate).Mul(math.LegacyOneDec().Add(trustDepositRate))
	trustFees := uint64(math.LegacyNewDec(int64(beneficiaryFees)).Mul(multiplier).Mul(math.LegacyNewDec(int64(trustUnitPrice))).TruncateInt64())

	// Account MUST have sufficient available balance
	creatorAddr, err := sdk.AccAddressFromBech32(msg.Creator)
	if err != nil {
		return nil, 0, 0, fmt.Errorf("invalid creator address: %w", err)
	}

	requiredAmount := sdk.NewInt64Coin(types.BondDenom, int64(trustFees))
	if !ms.bankKeeper.HasBalance(ctx, creatorAddr, requiredAmount) {
		return nil, 0, 0, fmt.Errorf("insufficient funds: required %s", requiredAmount)
	}

	return foundPermSet, beneficiaryFees, trustFees, nil
}

// [MOD-PERM-MSG-10-4] Create or Update Permission Session execution
func (ms msgServer) executeCreateOrUpdatePermissionSession(ctx sdk.Context, msg *types.MsgCreateOrUpdatePermissionSession, foundPermSet []types.Permission, beneficiaryFees, trustFees uint64, now time.Time) error {
	// Load all permissions as in basic checks (already done in precondition checks)

	verifierPerm := msg.VerifierPermId != 0
	trustUnitPrice := ms.trustRegistryKeeper.GetTrustUnitPrice(ctx)
	trustDepositRate := ms.trustDeposit.GetTrustDepositRate(ctx)

	creatorAddr, err := sdk.AccAddressFromBech32(msg.Creator)
	if err != nil {
		return fmt.Errorf("invalid creator address: %w", err)
	}

	// Get executor permission for deposit updates
	var executorPerm types.Permission
	if verifierPerm {
		executorPerm, err = ms.Permission.Get(ctx, msg.VerifierPermId)
	} else {
		executorPerm, err = ms.Permission.Get(ctx, msg.IssuerPermId)
	}
	if err != nil {
		return fmt.Errorf("failed to get executor permission: %w", err)
	}

	// Process fees for each permission in found_perm_set
	for _, perm := range foundPermSet {
		var fees uint64
		if verifierPerm {
			fees = perm.VerificationFees
		} else {
			fees = perm.IssuanceFees
		}

		if fees > 0 {
			// Calculate amounts
			totalFeesAmount := fees * trustUnitPrice
			trustDepositAmount := uint64(math.LegacyNewDec(int64(totalFeesAmount)).Mul(trustDepositRate).TruncateInt64())
			directFeesAmount := totalFeesAmount - trustDepositAmount

			// transfer direct fees to perm.grantee
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

			// use MOD-TD-MSG-1 to increase trust deposit of perm.grantee and increase perm.deposit
			if trustDepositAmount > 0 {
				// Transfer to module account first
				err = ms.bankKeeper.SendCoinsFromAccountToModule(
					ctx,
					creatorAddr,
					types.ModuleName,
					sdk.NewCoins(sdk.NewInt64Coin(types.BondDenom, int64(trustDepositAmount))),
				)
				if err != nil {
					return fmt.Errorf("failed to transfer trust deposit to module: %w", err)
				}

				// Increase trust deposit of perm.grantee
				err = ms.trustDeposit.AdjustTrustDeposit(ctx, perm.Grantee, int64(trustDepositAmount))
				if err != nil {
					return fmt.Errorf("failed to adjust grantee trust deposit: %w", err)
				}

				// Increase perm.deposit by the same value
				perm.Deposit += trustDepositAmount
				if err := ms.Keeper.UpdatePermission(ctx, perm); err != nil {
					return fmt.Errorf("failed to update grantee permission deposit: %w", err)
				}

				// use MOD-TD-MSG-1 to increase trust deposit of account executing the method and add to executor_perm.deposit
				err = ms.trustDeposit.AdjustTrustDeposit(ctx, msg.Creator, int64(trustDepositAmount))
				if err != nil {
					return fmt.Errorf("failed to adjust creator trust deposit: %w", err)
				}

				// Add the same amount to executor_perm.deposit
				executorPerm.Deposit += trustDepositAmount
				if err := ms.Keeper.UpdatePermission(ctx, executorPerm); err != nil {
					return fmt.Errorf("failed to update executor permission deposit: %w", err)
				}
			}
		}
	}

	// Create or update session
	if err := ms.createOrUpdateSession(ctx, msg, now); err != nil {
		return fmt.Errorf("failed to create/update session: %w", err)
	}

	return nil
}

func (ms msgServer) validateSessionAccess(ctx sdk.Context, msg *types.MsgCreateOrUpdatePermissionSession) error {
	existingSession, err := ms.PermissionSession.Get(ctx, msg.Id)
	if err != nil {
		if errors.Is(err, collections.ErrNotFound) {
			return nil // New session case
		}
		return fmt.Errorf("failed to get session: %w", err)
	}

	// Only session controller can update
	if existingSession.Controller != msg.Creator {
		return fmt.Errorf("only session controller can update")
	}

	// Check for duplicate authorization
	for _, authz := range existingSession.Authz {
		if authz.ExecutorPermId == msg.IssuerPermId &&
			authz.BeneficiaryPermId == msg.VerifierPermId &&
			authz.WalletAgentPermId == msg.WalletAgentPermId {
			return fmt.Errorf("authorization already exists")
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

	// Add new authorization: add (issuer_perm_id, verifier_perm_id, wallet_agent_perm_id) to session.authz[]
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
			return nil, fmt.Errorf("issuer permission not found: %w", err)
		}
		schemaID = issuerPerm.SchemaId
	} else if verifierPermId != 0 {
		verifierPerm, err := ms.Permission.Get(ctx, verifierPermId)
		if err != nil {
			return nil, fmt.Errorf("verifier permission not found: %w", err)
		}
		schemaID = verifierPerm.SchemaId
	} else {
		return nil, fmt.Errorf("at least one of issuer_perm_id or verifier_perm_id must be provided")
	}

	// Get schema to check permission management mode
	cs, err := ms.credentialSchemaKeeper.GetCredentialSchemaById(ctx, schemaID)
	if err != nil {
		return nil, fmt.Errorf("credential schema not found: %w", err)
	}

	// Check if schema is configured with OPEN permission management mode
	isOpenMode := false
	if (issuerPermId != 0 && cs.IssuerPermManagementMode == credentialschematypes.CredentialSchemaPermManagementMode_OPEN) ||
		(verifierPermId != 0 && cs.VerifierPermManagementMode == credentialschematypes.CredentialSchemaPermManagementMode_OPEN) {
		isOpenMode = true
	}

	// For OPEN mode, find the ECOSYSTEM permission
	if isOpenMode {
		// Find ECOSYSTEM permission for this schema
		err = ms.Permission.Walk(ctx, nil, func(id uint64, perm types.Permission) (bool, error) {
			if perm.SchemaId == schemaID &&
				perm.Type == types.PermissionType_ECOSYSTEM &&
				perm.Revoked == nil && perm.SlashedDeposit == 0 {
				foundPerms = append(foundPerms, perm)
				return true, nil // Stop iteration once found
			}
			return false, nil
		})

		if err != nil {
			return nil, fmt.Errorf("failed to query ECOSYSTEM permission: %w", err)
		}

		return foundPerms, nil
	}

	// Process issuer permission hierarchy if provided (non-OPEN mode)
	if issuerPermId != 0 {
		issuerPerm, err := ms.Permission.Get(ctx, issuerPermId)
		if err != nil {
			return nil, fmt.Errorf("issuer permission not found: %w", err)
		}

		// Follow the validator chain up
		if issuerPerm.ValidatorPermId != 0 {
			currentPermID := issuerPerm.ValidatorPermId
			for currentPermID != 0 {
				currentPerm, err := ms.Permission.Get(ctx, currentPermID)
				if err != nil {
					return nil, fmt.Errorf("failed to get permission: %w", err)
				}

				// Add to set if valid and not already included (removed terminated check)
				if currentPerm.Revoked == nil && currentPerm.SlashedDeposit == 0 && !containsPerm(currentPermID) {
					foundPerms = append(foundPerms, currentPerm)
				}

				// Move up
				currentPermID = currentPerm.ValidatorPermId
			}
		}
	}

	// Process verifier permission hierarchy if provided
	if verifierPermId != 0 {
		// First add issuer permission to the set if provided
		if issuerPermId != 0 {
			issuerPerm, err := ms.Permission.Get(ctx, issuerPermId)
			if err == nil && issuerPerm.Revoked == nil && !containsPerm(issuerPermId) {
				foundPerms = append(foundPerms, issuerPerm)
			}
		}

		// Then process verifier's validator chain
		verifierPerm, err := ms.Permission.Get(ctx, verifierPermId)
		if err != nil {
			return nil, fmt.Errorf("verifier permission not found: %w", err)
		}

		if verifierPerm.ValidatorPermId != 0 {
			currentPermID := verifierPerm.ValidatorPermId
			for currentPermID != 0 {
				currentPerm, err := ms.Permission.Get(ctx, currentPermID)
				if err != nil {
					return nil, fmt.Errorf("failed to get permission: %w", err)
				}

				// Add to set if valid and not already included (removed terminated check)
				if currentPerm.Revoked == nil && currentPerm.SlashedDeposit == 0 && !containsPerm(currentPermID) {
					foundPerms = append(foundPerms, currentPerm)
				}

				// Move up
				currentPermID = currentPerm.ValidatorPermId
			}
		}
	}

	return foundPerms, nil
}
