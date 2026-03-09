package keeper

import (
	"errors"
	"fmt"

	"cosmossdk.io/math"
	credentialschematypes "github.com/verana-labs/verana/x/cs/types"

	"time"

	"cosmossdk.io/collections"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/verana-labs/verana/x/perm/types"
)

// [MOD-PERM-MSG-10-2] Create or Update Permission Session precondition checks
func (ms msgServer) validateCreateOrUpdatePermissionSessionPreconditions(ctx sdk.Context, msg *types.MsgCreateOrUpdatePermissionSession, now time.Time) error {
	// if issuer_perm_id is null AND verifier_perm_id is null, MUST abort
	if msg.IssuerPermId == 0 && msg.VerifierPermId == 0 {
		return fmt.Errorf("at least one of issuer_perm_id or verifier_perm_id must be provided")
	}

	// id MUST be a valid uuid (already validated in ValidateBasic)
	// If an entry with id already exists, existing_entry.authority MUST equal authority AND existing_entry.vs_operator MUST equal operator
	if err := ms.validateSessionAccess(ctx, msg); err != nil {
		return err
	}

	var issuerPerm, verifierPerm types.Permission
	var hasIssuer, hasVerifier bool

	// if issuer_perm_id is not null
	if msg.IssuerPermId != 0 {
		var err error
		issuerPerm, err = ms.Permission.Get(ctx, msg.IssuerPermId)
		if err != nil {
			return fmt.Errorf("issuer permission not found: %w", err)
		}
		hasIssuer = true

		// if issuer_perm.type is not ISSUER, abort
		if issuerPerm.Type != types.PermissionType_ISSUER {
			return fmt.Errorf("issuer permission must be ISSUER type")
		}

		// if issuer_perm is not an active permission, abort
		if err := IsValidPermission(issuerPerm, issuerPerm.Country, now); err != nil {
			return fmt.Errorf("issuer permission is not valid: %w", err)
		}

		// if issuer_perm.vs_operator is not equal to operator, abort
		if issuerPerm.VsOperator != msg.Operator {
			return fmt.Errorf("issuer permission vs_operator does not match operator")
		}

		// if issuer_perm.authority is not equal to authority, abort
		if issuerPerm.Authority != msg.Authority {
			return fmt.Errorf("issuer permission authority does not match authority")
		}

		// if digest is present but not a valid digest SRI, abort
		// (already validated in ValidateBasic)
	}

	// if verifier_perm_id is not null
	if msg.VerifierPermId != 0 {
		var err error
		verifierPerm, err = ms.Permission.Get(ctx, msg.VerifierPermId)
		if err != nil {
			return fmt.Errorf("verifier permission not found: %w", err)
		}
		hasVerifier = true

		// if verifier_perm.type is not VERIFIER, abort
		if verifierPerm.Type != types.PermissionType_VERIFIER {
			return fmt.Errorf("verifier permission must be VERIFIER type")
		}

		// if verifier_perm is not an active permission, abort
		if err := IsValidPermission(verifierPerm, verifierPerm.Country, now); err != nil {
			return fmt.Errorf("verifier permission is not valid: %w", err)
		}

		// if verifier_perm.vs_operator is not equal to operator, abort
		if verifierPerm.VsOperator != msg.Operator {
			return fmt.Errorf("verifier permission vs_operator does not match operator")
		}

		// if verifier_perm.authority is not equal to authority, abort
		if verifierPerm.Authority != msg.Authority {
			return fmt.Errorf("verifier permission authority does not match authority")
		}

		// if digest is present but not a valid digest SRI, abort
		// (already validated in ValidateBasic)
	}

	// Define the primary permission: if verifier_perm is not null, perm = verifier_perm, else perm = issuer_perm
	var primaryPerm types.Permission
	if hasVerifier {
		primaryPerm = verifierPerm
	} else if hasIssuer {
		primaryPerm = issuerPerm
	}

	// [AUTHZ-CHECK-3] MUST pass for this (authority, operator, perm) tuple
	if ms.delegationKeeper != nil {
		if err := ms.delegationKeeper.CheckVSOperatorAuthorization(
			ctx,
			msg.Authority,
			msg.Operator,
		); err != nil {
			return fmt.Errorf("VS operator authorization check failed: %w", err)
		}
	}

	// Check that perm.vs_operator_authz_enabled is true
	if !primaryPerm.VsOperatorAuthzEnabled {
		return fmt.Errorf("VS operator authorization is not enabled for permission %d", primaryPerm.Id)
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

	// if agent_perm is not an active permission, abort
	if err := IsValidPermission(agentPerm, agentPerm.Country, now); err != nil {
		return fmt.Errorf("agent permission is not valid: %w", err)
	}

	// wallet_agent: Load wallet_agent_perm from wallet_agent_perm_id
	walletAgentPerm, err := ms.Permission.Get(ctx, msg.WalletAgentPermId)
	if err != nil {
		return fmt.Errorf("wallet agent permission not found: %w", err)
	}

	// if wallet_agent_perm.type is not ISSUER, abort
	if walletAgentPerm.Type != types.PermissionType_ISSUER {
		return fmt.Errorf("wallet agent permission must be ISSUER type")
	}

	// if wallet_agent_perm is not an active permission, abort
	if err := IsValidPermission(walletAgentPerm, walletAgentPerm.Country, now); err != nil {
		return fmt.Errorf("wallet agent permission is not valid: %w", err)
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
	// Apply discounts from executor permission (issuer or verifier)
	beneficiaryFees := uint64(0)
	isVerification := msg.VerifierPermId != 0
	const discountScale = 10000 // 10000 = 1.0 = 100% discount

	// Get executor permission's discount
	var executorDiscount uint64
	if isVerification {
		executorPerm, err := ms.Permission.Get(ctx, msg.VerifierPermId)
		if err != nil {
			return nil, 0, 0, fmt.Errorf("failed to get verifier permission: %w", err)
		}
		executorDiscount = executorPerm.VerificationFeeDiscount
	} else {
		executorPerm, err := ms.Permission.Get(ctx, msg.IssuerPermId)
		if err != nil {
			return nil, 0, 0, fmt.Errorf("failed to get issuer permission: %w", err)
		}
		executorDiscount = executorPerm.IssuanceFeeDiscount
	}

	for _, perm := range foundPermSet {
		var fees uint64
		if isVerification {
			fees = perm.VerificationFees
		} else {
			fees = perm.IssuanceFees
		}

		// Apply executor's discount: beneficiary_fee = perm.fee * (1 - discount/10000)
		if executorDiscount > 0 {
			fees = (fees * (discountScale - executorDiscount)) / discountScale
		}

		beneficiaryFees += fees
	}

	// Get global variables for calculations
	userAgentRewardRate := ms.trustDeposit.GetUserAgentRewardRate(ctx)
	walletUserAgentRewardRate := ms.trustDeposit.GetWalletUserAgentRewardRate(ctx)
	trustDepositRate := ms.trustDeposit.GetTrustDepositRate(ctx)
	trustUnitPrice := ms.trustRegistryKeeper.GetTrustUnitPrice(ctx)

	// Calculate trust_fees = beneficiary_fees * (1 + user_agent_reward_rate + wallet_user_agent_reward_rate + trust_deposit_rate) * trust_unit_price
	multiplier := math.LegacyOneDec().Add(userAgentRewardRate).Add(walletUserAgentRewardRate).Add(trustDepositRate)
	trustFees := uint64(math.LegacyNewDec(int64(beneficiaryFees)).Mul(multiplier).Mul(math.LegacyNewDec(int64(trustUnitPrice))).TruncateInt64())

	// authority account MUST have sufficient available balance
	authorityAddr, err := sdk.AccAddressFromBech32(msg.Authority)
	if err != nil {
		return nil, 0, 0, fmt.Errorf("invalid authority address: %w", err)
	}

	requiredAmount := sdk.NewInt64Coin(types.BondDenom, int64(trustFees))
	if !ms.bankKeeper.HasBalance(ctx, authorityAddr, requiredAmount) {
		return nil, 0, 0, fmt.Errorf("insufficient funds: required %s", requiredAmount)
	}

	return foundPermSet, beneficiaryFees, trustFees, nil
}

// [MOD-PERM-MSG-10-4] Create or Update Permission Session execution
func (ms msgServer) executeCreateOrUpdatePermissionSession(ctx sdk.Context, msg *types.MsgCreateOrUpdatePermissionSession, foundPermSet []types.Permission, beneficiaryFees, trustFees uint64, now time.Time) error {
	isVerification := msg.VerifierPermId != 0
	trustUnitPrice := ms.trustRegistryKeeper.GetTrustUnitPrice(ctx)
	trustDepositRate := ms.trustDeposit.GetTrustDepositRate(ctx)
	userAgentRewardRate := ms.trustDeposit.GetUserAgentRewardRate(ctx)
	walletUserAgentRewardRate := ms.trustDeposit.GetWalletUserAgentRewardRate(ctx)

	authorityAddr, err := sdk.AccAddressFromBech32(msg.Authority)
	if err != nil {
		return fmt.Errorf("invalid authority address: %w", err)
	}

	// Get payer permission for deposit updates
	var payerPerm types.Permission
	if isVerification {
		payerPerm, err = ms.Permission.Get(ctx, msg.VerifierPermId)
	} else {
		payerPerm, err = ms.Permission.Get(ctx, msg.IssuerPermId)
	}
	if err != nil {
		return fmt.Errorf("failed to get payer permission: %w", err)
	}

	// Initialize agent reward accumulators
	accumulatedUserAgentReward := math.LegacyZeroDec()
	accumulatedWalletAgentReward := math.LegacyZeroDec()

	// Get executor's discount
	var executorDiscount uint64
	if isVerification {
		executorDiscount = payerPerm.VerificationFeeDiscount
	} else {
		executorDiscount = payerPerm.IssuanceFeeDiscount
	}

	// Step 1: Process fee distribution to each beneficiary
	const discountScale = 10000
	for _, perm := range foundPermSet {
		var fees uint64
		if isVerification {
			fees = perm.VerificationFees
		} else {
			fees = perm.IssuanceFees
		}

		if fees > 0 {
			// Apply executor's discount: beneficiary_fee = perm.fee * (1 - discount/10000)
			if executorDiscount > 0 {
				fees = (fees * (discountScale - executorDiscount)) / discountScale
			}

			// Calculate fee_in_native_denom (using trust unit price for now - Case B: TU pricing)
			feeInNativeDenom := math.LegacyNewDec(int64(fees * trustUnitPrice))

			// Calculate trust deposit and direct account amounts
			payerTrustDeposit := uint64(feeInNativeDenom.Mul(trustDepositRate).TruncateInt64())
			payeeFeesToAccount := uint64(feeInNativeDenom.TruncateInt64()) - payerTrustDeposit

			// Accumulate agent rewards
			accumulatedUserAgentReward = accumulatedUserAgentReward.Add(feeInNativeDenom.Mul(userAgentRewardRate))
			accumulatedWalletAgentReward = accumulatedWalletAgentReward.Add(feeInNativeDenom.Mul(walletUserAgentRewardRate))

			// Transfer payee_fees_to_account to perm.authority
			if payeeFeesToAccount > 0 {
				granteeAddr, err := sdk.AccAddressFromBech32(perm.Authority)
				if err != nil {
					return fmt.Errorf("invalid grantee address: %w", err)
				}

				err = ms.bankKeeper.SendCoins(
					ctx,
					authorityAddr,
					granteeAddr,
					sdk.NewCoins(sdk.NewInt64Coin(types.BondDenom, int64(payeeFeesToAccount))),
				)
				if err != nil {
					return fmt.Errorf("failed to transfer direct fees: %w", err)
				}
			}

			// Increase trust deposit of perm.authority (payee) and perm.deposit
			if payerTrustDeposit > 0 {
				// Transfer to module account first
				err = ms.bankKeeper.SendCoinsFromAccountToModule(
					ctx,
					authorityAddr,
					types.ModuleName,
					sdk.NewCoins(sdk.NewInt64Coin(types.BondDenom, int64(payerTrustDeposit))),
				)
				if err != nil {
					return fmt.Errorf("failed to transfer trust deposit to module: %w", err)
				}

				// Increase trust deposit of perm.authority (payee)
				err = ms.trustDeposit.AdjustTrustDeposit(ctx, perm.Authority, int64(payerTrustDeposit))
				if err != nil {
					return fmt.Errorf("failed to adjust grantee trust deposit: %w", err)
				}

				// Increase perm.deposit
				perm.Deposit += payerTrustDeposit
				if err := ms.Keeper.UpdatePermission(ctx, perm); err != nil {
					return fmt.Errorf("failed to update grantee permission deposit: %w", err)
				}

				// Increase trust deposit of authority (payer) and payer_perm.deposit
				err = ms.trustDeposit.AdjustTrustDeposit(ctx, msg.Authority, int64(payerTrustDeposit))
				if err != nil {
					return fmt.Errorf("failed to adjust payer trust deposit: %w", err)
				}

				payerPerm.Deposit += payerTrustDeposit
				if err := ms.Keeper.UpdatePermission(ctx, payerPerm); err != nil {
					return fmt.Errorf("failed to update payer permission deposit: %w", err)
				}
			}
		}
	}

	// Step 2: Process agent rewards
	// User Agent Reward
	if accumulatedUserAgentReward.IsPositive() {
		agentPerm, err := ms.Permission.Get(ctx, msg.AgentPermId)
		if err != nil {
			return fmt.Errorf("failed to get agent permission: %w", err)
		}

		agentTrustDeposit := uint64(accumulatedUserAgentReward.Mul(trustDepositRate).TruncateInt64())
		agentFeesToAccount := uint64(accumulatedUserAgentReward.TruncateInt64()) - agentTrustDeposit

		// Transfer direct amount to agent_perm.authority
		if agentFeesToAccount > 0 {
			agentGranteeAddr, err := sdk.AccAddressFromBech32(agentPerm.Authority)
			if err != nil {
				return fmt.Errorf("invalid agent grantee address: %w", err)
			}

			err = ms.bankKeeper.SendCoins(
				ctx,
				authorityAddr,
				agentGranteeAddr,
				sdk.NewCoins(sdk.NewInt64Coin(types.BondDenom, int64(agentFeesToAccount))),
			)
			if err != nil {
				return fmt.Errorf("failed to transfer user agent reward: %w", err)
			}
		}

		// Increase trust deposit of agent_perm.authority and agent_perm.deposit
		if agentTrustDeposit > 0 {
			err = ms.bankKeeper.SendCoinsFromAccountToModule(
				ctx,
				authorityAddr,
				types.ModuleName,
				sdk.NewCoins(sdk.NewInt64Coin(types.BondDenom, int64(agentTrustDeposit))),
			)
			if err != nil {
				return fmt.Errorf("failed to transfer user agent trust deposit to module: %w", err)
			}

			err = ms.trustDeposit.AdjustTrustDeposit(ctx, agentPerm.Authority, int64(agentTrustDeposit))
			if err != nil {
				return fmt.Errorf("failed to adjust agent trust deposit: %w", err)
			}

			agentPerm.Deposit += agentTrustDeposit
			if err := ms.Keeper.UpdatePermission(ctx, agentPerm); err != nil {
				return fmt.Errorf("failed to update agent permission deposit: %w", err)
			}
		}
	}

	// Wallet Agent Reward
	if accumulatedWalletAgentReward.IsPositive() {
		walletAgentPerm, err := ms.Permission.Get(ctx, msg.WalletAgentPermId)
		if err != nil {
			return fmt.Errorf("failed to get wallet agent permission: %w", err)
		}

		walletAgentTrustDeposit := uint64(accumulatedWalletAgentReward.Mul(trustDepositRate).TruncateInt64())
		walletAgentFeesToAccount := uint64(accumulatedWalletAgentReward.TruncateInt64()) - walletAgentTrustDeposit

		// Transfer direct amount to wallet_agent_perm.authority
		if walletAgentFeesToAccount > 0 {
			walletAgentGranteeAddr, err := sdk.AccAddressFromBech32(walletAgentPerm.Authority)
			if err != nil {
				return fmt.Errorf("invalid wallet agent grantee address: %w", err)
			}

			err = ms.bankKeeper.SendCoins(
				ctx,
				authorityAddr,
				walletAgentGranteeAddr,
				sdk.NewCoins(sdk.NewInt64Coin(types.BondDenom, int64(walletAgentFeesToAccount))),
			)
			if err != nil {
				return fmt.Errorf("failed to transfer wallet user agent reward: %w", err)
			}
		}

		// Increase trust deposit of wallet_agent_perm.authority and wallet_agent_perm.deposit
		if walletAgentTrustDeposit > 0 {
			err = ms.bankKeeper.SendCoinsFromAccountToModule(
				ctx,
				authorityAddr,
				types.ModuleName,
				sdk.NewCoins(sdk.NewInt64Coin(types.BondDenom, int64(walletAgentTrustDeposit))),
			)
			if err != nil {
				return fmt.Errorf("failed to transfer wallet user agent trust deposit to module: %w", err)
			}

			err = ms.trustDeposit.AdjustTrustDeposit(ctx, walletAgentPerm.Authority, int64(walletAgentTrustDeposit))
			if err != nil {
				return fmt.Errorf("failed to adjust wallet agent trust deposit: %w", err)
			}

			walletAgentPerm.Deposit += walletAgentTrustDeposit
			if err := ms.Keeper.UpdatePermission(ctx, walletAgentPerm); err != nil {
				return fmt.Errorf("failed to update wallet agent permission deposit: %w", err)
			}
		}
	}

	// Step 3: Create or update session records
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

	// existing_entry.authority MUST be equal to authority
	if existingSession.Authority != msg.Authority {
		return fmt.Errorf("session authority does not match: expected %s, got %s", existingSession.Authority, msg.Authority)
	}

	// existing_entry.vs_operator MUST be equal to operator
	if existingSession.VsOperator != msg.Operator {
		return fmt.Errorf("session vs_operator does not match: expected %s, got %s", existingSession.VsOperator, msg.Operator)
	}

	return nil
}

func (ms msgServer) createOrUpdateSession(ctx sdk.Context, msg *types.MsgCreateOrUpdatePermissionSession, now time.Time) error {
	session := &types.PermissionSession{
		Id:          msg.Id,
		Authority:   msg.Authority,
		VsOperator:  msg.Operator,
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

	// Create PermissionSessionRecord
	record := &types.PermissionSessionRecord{
		Created:           &now,
		IssuerPermId:      msg.IssuerPermId,
		VerifierPermId:    msg.VerifierPermId,
		WalletAgentPermId: msg.WalletAgentPermId,
	}

	// Add the record to session.session_records
	session.SessionRecords = append(session.SessionRecords, record)

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

				// Add to set if valid and not already included
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

				// Add to set if valid and not already included
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
