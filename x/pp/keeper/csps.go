package keeper

import (
	"errors"
	"fmt"

	"cosmossdk.io/math"
	credentialschematypes "github.com/verana-labs/verana/x/cs/types"

	"time"

	"cosmossdk.io/collections"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/verana-labs/verana/x/pp/types"
)

// maxInt64AsUint64 is the highest uint64 that still fits in a signed int64.
// Used to guard narrowing conversions of fee/deposit amounts before they
// reach bank/sdk helpers that take int64.
const maxInt64AsUint64 uint64 = 1<<63 - 1

// uint64ToInt64 narrows a uint64 to int64 with an overflow guard. Returns an
// error when x does not fit, so the caller can abort the transaction rather
// than silently wrap to a negative amount.
func uint64ToInt64(x uint64, field string) (int64, error) {
	if x > maxInt64AsUint64 {
		return 0, fmt.Errorf("%s overflows int64: %d", field, x)
	}
	return int64(x), nil
}

// [MOD-PERM-MSG-10-2] Create or Update Participant Session precondition checks
func (ms msgServer) validateCreateOrUpdateParticipantSessionPreconditions(ctx sdk.Context, msg *types.MsgCreateOrUpdateParticipantSession, now time.Time) error {
	// if issuer_perm_id is null AND verifier_perm_id is null, MUST abort
	if msg.IssuerParticipantId == 0 && msg.VerifierParticipantId == 0 {
		return fmt.Errorf("at least one of issuer_perm_id or verifier_perm_id must be provided")
	}

	// id MUST be a valid uuid (already validated in ValidateBasic)
	// If an entry with id already exists, existing_entry.authority MUST equal authority AND existing_entry.vs_operator MUST equal operator
	if err := ms.validateSessionAccess(ctx, msg); err != nil {
		return err
	}

	var issuerPerm, verifierPerm types.Participant
	var hasIssuer, hasVerifier bool

	// if issuer_perm_id is not null
	if msg.IssuerParticipantId != 0 {
		var err error
		issuerPerm, err = ms.Participant.Get(ctx, msg.IssuerParticipantId)
		if err != nil {
			return fmt.Errorf("issuer permission not found: %w", err)
		}
		hasIssuer = true

		// if issuer_perm.type is not ISSUER, abort
		if issuerPerm.Role != types.ParticipantRole_ISSUER {
			return fmt.Errorf("issuer permission must be ISSUER type")
		}

		// if issuer_perm is not an active permission, abort
		if err := IsValidPermission(issuerPerm, now); err != nil {
			return fmt.Errorf("issuer permission is not valid: %w", err)
		}

		// if issuer_perm.vs_operator is not equal to operator, abort
		if issuerPerm.VsOperator != msg.Operator {
			return fmt.Errorf("issuer permission vs_operator does not match operator")
		}

		// if issuer_perm.authority is not equal to authority, abort
		issuerCorpAcct, err := ms.corpAccountFromID(ctx, issuerPerm.CorporationId)
		if err != nil {
			return err
		}
		if issuerCorpAcct != msg.Corporation {
			return fmt.Errorf("issuer participant authority does not match authority")
		}

		// if digest is present but not a valid digest SRI, abort
		// (already validated in ValidateBasic)
	}

	// if verifier_perm_id is not null
	if msg.VerifierParticipantId != 0 {
		var err error
		verifierPerm, err = ms.Participant.Get(ctx, msg.VerifierParticipantId)
		if err != nil {
			return fmt.Errorf("verifier permission not found: %w", err)
		}
		hasVerifier = true

		// if verifier_perm.type is not VERIFIER, abort
		if verifierPerm.Role != types.ParticipantRole_VERIFIER {
			return fmt.Errorf("verifier permission must be VERIFIER type")
		}

		// if verifier_perm is not an active permission, abort
		if err := IsValidPermission(verifierPerm, now); err != nil {
			return fmt.Errorf("verifier permission is not valid: %w", err)
		}

		// if verifier_perm.vs_operator is not equal to operator, abort
		if verifierPerm.VsOperator != msg.Operator {
			return fmt.Errorf("verifier permission vs_operator does not match operator")
		}

		// if verifier_perm.authority is not equal to authority, abort
		verifierCorpAcct, err := ms.corpAccountFromID(ctx, verifierPerm.CorporationId)
		if err != nil {
			return err
		}
		if verifierCorpAcct != msg.Corporation {
			return fmt.Errorf("verifier participant authority does not match authority")
		}

		// if digest is present but not a valid digest SRI, abort
		// (already validated in ValidateBasic)
	}

	// Define the primary permission: if verifier_perm is not null, perm = verifier_perm, else perm = issuer_perm
	var primaryPerm types.Participant
	if hasVerifier {
		primaryPerm = verifierPerm
	} else if hasIssuer {
		primaryPerm = issuerPerm
	}

	// [AUTHZ-CHECK-3] MUST pass for this (authority, operator, perm) tuple
	if ms.delegationKeeper == nil {
		return fmt.Errorf("delegation keeper is required for VS operator authorization")
	}
	if err := ms.delegationKeeper.CheckVSOperatorAuthorization(
		ctx,
		msg.Corporation,
		msg.Operator,
	); err != nil {
		return fmt.Errorf("VS operator authorization check failed: %w", err)
	}

	// Check that perm.vs_operator_authz_enabled is true
	if !primaryPerm.VsOperatorAuthzEnabled {
		return fmt.Errorf("VS operator authorization is not enabled for permission %d", primaryPerm.Id)
	}

	// agent: Load agent_perm from agent_perm_id
	agentPerm, err := ms.Participant.Get(ctx, msg.AgentParticipantId)
	if err != nil {
		return fmt.Errorf("agent permission not found: %w", err)
	}

	// if agent_perm.type is not ISSUER, abort
	if agentPerm.Role != types.ParticipantRole_ISSUER {
		return fmt.Errorf("agent permission must be ISSUER type")
	}

	// if agent_perm is not an active permission, abort
	if err := IsValidPermission(agentPerm, now); err != nil {
		return fmt.Errorf("agent permission is not valid: %w", err)
	}

	// wallet_agent: Load wallet_agent_perm from wallet_agent_perm_id
	walletAgentPerm, err := ms.Participant.Get(ctx, msg.WalletAgentParticipantId)
	if err != nil {
		return fmt.Errorf("wallet agent permission not found: %w", err)
	}

	// if wallet_agent_perm.type is not ISSUER, abort
	if walletAgentPerm.Role != types.ParticipantRole_ISSUER {
		return fmt.Errorf("wallet agent permission must be ISSUER type")
	}

	// if wallet_agent_perm is not an active permission, abort
	if err := IsValidPermission(walletAgentPerm, now); err != nil {
		return fmt.Errorf("wallet agent permission is not valid: %w", err)
	}

	return nil
}

// [MOD-PERM-MSG-10-3] Create or Update Participant Session fee checks
func (ms msgServer) validateCreateOrUpdateParticipantSessionFees(ctx sdk.Context, msg *types.MsgCreateOrUpdateParticipantSession) ([]types.Participant, uint64, uint64, error) {
	// use "Find Beneficiaries" query method to get the set of beneficiary permission found_perm_set
	foundPermSet, err := ms.findBeneficiaries(ctx, msg.IssuerParticipantId, msg.VerifierParticipantId)
	if err != nil {
		return nil, 0, 0, fmt.Errorf("failed to find beneficiaries: %w", err)
	}

	// calculate the required beneficiary fees
	// Apply discounts from executor permission (issuer or verifier)
	beneficiaryFees := uint64(0)
	isVerification := msg.VerifierParticipantId != 0
	const discountScale = 10000 // 10000 = 1.0 = 100% discount

	// Get executor permission's discount
	var executorDiscount uint64
	if isVerification {
		executorPerm, err := ms.Participant.Get(ctx, msg.VerifierParticipantId)
		if err != nil {
			return nil, 0, 0, fmt.Errorf("failed to get verifier permission: %w", err)
		}
		executorDiscount = executorPerm.VerificationFeeDiscount
	} else {
		executorPerm, err := ms.Participant.Get(ctx, msg.IssuerParticipantId)
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
	trustUnitPrice := ms.ecosystemKeeper.GetTrustUnitPrice(ctx)

	// Calculate trust_fees = beneficiary_fees * (1 + user_agent_reward_rate + wallet_user_agent_reward_rate + trust_deposit_rate) * trust_unit_price
	//
	// Use math.Int arbitrary-precision arithmetic throughout: naive int64(fees)
	// would wrap for values >= 2^63, and uint64 * uint64 multiplications can
	// overflow silently before any int64 cast. Convert uint64 inputs via
	// math.NewIntFromUint64, multiply through LegacyDec, then bounds-check
	// before narrowing back to uint64/int64.
	multiplier := math.LegacyOneDec().Add(userAgentRewardRate).Add(walletUserAgentRewardRate).Add(trustDepositRate)
	trustFeesDec := math.LegacyNewDecFromInt(math.NewIntFromUint64(beneficiaryFees)).
		Mul(multiplier).
		Mul(math.LegacyNewDecFromInt(math.NewIntFromUint64(trustUnitPrice)))
	trustFeesInt := trustFeesDec.TruncateInt()
	if !trustFeesInt.IsUint64() {
		return nil, 0, 0, fmt.Errorf("trust fees overflow uint64: %s", trustFeesInt.String())
	}
	trustFees := trustFeesInt.Uint64()

	// authority account MUST have sufficient available balance
	authorityAddr, err := sdk.AccAddressFromBech32(msg.Corporation)
	if err != nil {
		return nil, 0, 0, fmt.Errorf("invalid authority address: %w", err)
	}

	trustFeesI64, err := uint64ToInt64(trustFees, "trust_fees")
	if err != nil {
		return nil, 0, 0, err
	}
	requiredAmount := sdk.NewInt64Coin(types.BondDenom, trustFeesI64)
	if !ms.bankKeeper.HasBalance(ctx, authorityAddr, requiredAmount) {
		return nil, 0, 0, fmt.Errorf("insufficient funds: required %s", requiredAmount)
	}

	return foundPermSet, beneficiaryFees, trustFees, nil
}

// [MOD-PERM-MSG-10-4] Create or Update Participant Session execution
func (ms msgServer) executeCreateOrUpdateParticipantSession(ctx sdk.Context, msg *types.MsgCreateOrUpdateParticipantSession, foundPermSet []types.Participant, beneficiaryFees, trustFees uint64, now time.Time) error {
	isVerification := msg.VerifierParticipantId != 0
	trustUnitPrice := ms.ecosystemKeeper.GetTrustUnitPrice(ctx)
	trustDepositRate := ms.trustDeposit.GetTrustDepositRate(ctx)
	userAgentRewardRate := ms.trustDeposit.GetUserAgentRewardRate(ctx)
	walletUserAgentRewardRate := ms.trustDeposit.GetWalletUserAgentRewardRate(ctx)

	authorityAddr, err := sdk.AccAddressFromBech32(msg.Corporation)
	if err != nil {
		return fmt.Errorf("invalid authority address: %w", err)
	}

	// Get payer permission for deposit updates
	var payerPerm types.Participant
	if isVerification {
		payerPerm, err = ms.Participant.Get(ctx, msg.VerifierParticipantId)
	} else {
		payerPerm, err = ms.Participant.Get(ctx, msg.IssuerParticipantId)
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
			permCorpAcct, err := ms.corpAccountFromID(ctx, perm.CorporationId)
			if err != nil {
				return err
			}
			// Apply executor's discount: beneficiary_fee = perm.fee * (1 - discount/10000)
			if executorDiscount > 0 {
				fees = (fees * (discountScale - executorDiscount)) / discountScale
			}

			// Calculate fee_in_native_denom (using trust unit price for now - Case B: TU pricing).
			//
			// Safety: compute fees * trustUnitPrice via math.Int (arbitrary precision)
			// to avoid uint64*uint64 overflow, then lift into LegacyDec for the rate math.
			feeInNativeDenom := math.LegacyNewDecFromInt(
				math.NewIntFromUint64(fees).Mul(math.NewIntFromUint64(trustUnitPrice)),
			)

			// Calculate trust deposit and direct account amounts
			payerTrustDepositInt := feeInNativeDenom.Mul(trustDepositRate).TruncateInt()
			if !payerTrustDepositInt.IsUint64() {
				return fmt.Errorf("payer trust deposit overflows uint64: %s", payerTrustDepositInt.String())
			}
			payerTrustDeposit := payerTrustDepositInt.Uint64()
			feeNativeInt := feeInNativeDenom.TruncateInt()
			if !feeNativeInt.IsUint64() {
				return fmt.Errorf("fee in native denom overflows uint64: %s", feeNativeInt.String())
			}
			payeeFeesToAccount := feeNativeInt.Uint64() - payerTrustDeposit

			// Accumulate agent rewards
			accumulatedUserAgentReward = accumulatedUserAgentReward.Add(feeInNativeDenom.Mul(userAgentRewardRate))
			accumulatedWalletAgentReward = accumulatedWalletAgentReward.Add(feeInNativeDenom.Mul(walletUserAgentRewardRate))

			// Transfer payee_fees_to_account to perm.authority
			if payeeFeesToAccount > 0 {
				granteeAddr, err := sdk.AccAddressFromBech32(permCorpAcct)
				if err != nil {
					return fmt.Errorf("invalid grantee address: %w", err)
				}

				payeeFeesI64, err := uint64ToInt64(payeeFeesToAccount, "payee_fees_to_account")
				if err != nil {
					return err
				}
				err = ms.bankKeeper.SendCoins(
					ctx,
					authorityAddr,
					granteeAddr,
					sdk.NewCoins(sdk.NewInt64Coin(types.BondDenom, payeeFeesI64)),
				)
				if err != nil {
					return fmt.Errorf("failed to transfer direct fees: %w", err)
				}
			}

			// Increase trust deposit of perm.authority (payee) and perm.deposit
			if payerTrustDeposit > 0 {
				payerTDI64, err := uint64ToInt64(payerTrustDeposit, "payer_trust_deposit")
				if err != nil {
					return err
				}
				// Increase beneficiary's TD funded by payer (transfers from payer to TD module directly)
				err = ms.trustDeposit.AdjustTrustDepositOnBehalf(ctx, permCorpAcct, authorityAddr, payerTDI64)
				if err != nil {
					return fmt.Errorf("failed to adjust grantee trust deposit: %w", err)
				}

				// Increase perm.deposit
				perm.Deposit += payerTrustDeposit
				if err := ms.Keeper.UpdatePermission(ctx, perm); err != nil {
					return fmt.Errorf("failed to update grantee permission deposit: %w", err)
				}

				// Increase payer's own TD (standard self-funded adjustment)
				err = ms.trustDeposit.AdjustTrustDeposit(ctx, msg.Corporation, payerTDI64, "csps_payer_trust_deposit")
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
		agentPerm, err := ms.Participant.Get(ctx, msg.AgentParticipantId)
		if err != nil {
			return fmt.Errorf("failed to get agent permission: %w", err)
		}

		agentTrustDepositInt := accumulatedUserAgentReward.Mul(trustDepositRate).TruncateInt()
		if !agentTrustDepositInt.IsUint64() {
			return fmt.Errorf("agent trust deposit overflows uint64: %s", agentTrustDepositInt.String())
		}
		agentTrustDeposit := agentTrustDepositInt.Uint64()
		agentAccumInt := accumulatedUserAgentReward.TruncateInt()
		if !agentAccumInt.IsUint64() {
			return fmt.Errorf("agent accumulated reward overflows uint64: %s", agentAccumInt.String())
		}
		agentFeesToAccount := agentAccumInt.Uint64() - agentTrustDeposit

		// Transfer direct amount to agent_perm.authority
		agentCorpAcct, err := ms.corpAccountFromID(ctx, agentPerm.CorporationId)
		if err != nil {
			return err
		}
		if agentFeesToAccount > 0 {
			agentGranteeAddr, err := sdk.AccAddressFromBech32(agentCorpAcct)
			if err != nil {
				return fmt.Errorf("invalid agent grantee address: %w", err)
			}

			agentFeesI64, err := uint64ToInt64(agentFeesToAccount, "agent_fees_to_account")
			if err != nil {
				return err
			}
			err = ms.bankKeeper.SendCoins(
				ctx,
				authorityAddr,
				agentGranteeAddr,
				sdk.NewCoins(sdk.NewInt64Coin(types.BondDenom, agentFeesI64)),
			)
			if err != nil {
				return fmt.Errorf("failed to transfer user agent reward: %w", err)
			}
		}

		// Increase trust deposit of agent_perm.authority and agent_perm.deposit
		if agentTrustDeposit > 0 {
			agentTDI64, err := uint64ToInt64(agentTrustDeposit, "agent_trust_deposit")
			if err != nil {
				return err
			}
			// Increase agent's TD funded by payer (transfers from payer to TD module directly)
			err = ms.trustDeposit.AdjustTrustDepositOnBehalf(ctx, agentCorpAcct, authorityAddr, agentTDI64)
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
		walletAgentPerm, err := ms.Participant.Get(ctx, msg.WalletAgentParticipantId)
		if err != nil {
			return fmt.Errorf("failed to get wallet agent permission: %w", err)
		}

		walletAgentTDInt := accumulatedWalletAgentReward.Mul(trustDepositRate).TruncateInt()
		if !walletAgentTDInt.IsUint64() {
			return fmt.Errorf("wallet agent trust deposit overflows uint64: %s", walletAgentTDInt.String())
		}
		walletAgentTrustDeposit := walletAgentTDInt.Uint64()
		walletAccumInt := accumulatedWalletAgentReward.TruncateInt()
		if !walletAccumInt.IsUint64() {
			return fmt.Errorf("wallet agent accumulated reward overflows uint64: %s", walletAccumInt.String())
		}
		walletAgentFeesToAccount := walletAccumInt.Uint64() - walletAgentTrustDeposit

		// Transfer direct amount to wallet_agent_perm.authority
		walletAgentCorpAcct, err := ms.corpAccountFromID(ctx, walletAgentPerm.CorporationId)
		if err != nil {
			return err
		}
		if walletAgentFeesToAccount > 0 {
			walletAgentGranteeAddr, err := sdk.AccAddressFromBech32(walletAgentCorpAcct)
			if err != nil {
				return fmt.Errorf("invalid wallet agent grantee address: %w", err)
			}

			walletAgentFeesI64, err := uint64ToInt64(walletAgentFeesToAccount, "wallet_agent_fees_to_account")
			if err != nil {
				return err
			}
			err = ms.bankKeeper.SendCoins(
				ctx,
				authorityAddr,
				walletAgentGranteeAddr,
				sdk.NewCoins(sdk.NewInt64Coin(types.BondDenom, walletAgentFeesI64)),
			)
			if err != nil {
				return fmt.Errorf("failed to transfer wallet user agent reward: %w", err)
			}
		}

		// Increase trust deposit of wallet_agent_perm.authority and wallet_agent_perm.deposit
		if walletAgentTrustDeposit > 0 {
			// Increase wallet agent's TD funded by payer (transfers from payer to TD module directly)
			walletAgentTDI64, err := uint64ToInt64(walletAgentTrustDeposit, "wallet_agent_trust_deposit")
			if err != nil {
				return err
			}
			err = ms.trustDeposit.AdjustTrustDepositOnBehalf(ctx, walletAgentCorpAcct, authorityAddr, walletAgentTDI64)
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

	// [MOD-PERM-MSG-10] If the current transaction is for issuance of a
	// credential, persist the digest SRI by calling [MOD-DI-MSG-1] keeper-to-
	// keeper. Spec explicitly lets perm invoke DI with no signer/AUTHZ check.
	// We scope this to the issuance path (IssuerParticipantId != 0) and only fire
	// when the caller supplied a non-empty digest.
	if msg.Digest != "" && msg.IssuerParticipantId != 0 {
		if ms.digestKeeper == nil {
			return fmt.Errorf("digest keeper is required but not set")
		}
		if err := ms.digestKeeper.StoreDigestModuleCall(ctx, msg.Corporation, msg.Digest, "sha2-256"); err != nil {
			return fmt.Errorf("failed to persist credential digest: %w", err)
		}
	}

	return nil
}

func (ms msgServer) validateSessionAccess(ctx sdk.Context, msg *types.MsgCreateOrUpdateParticipantSession) error {
	existingSession, err := ms.ParticipantSession.Get(ctx, msg.Id)
	if err != nil {
		if errors.Is(err, collections.ErrNotFound) {
			return nil // New session case
		}
		return fmt.Errorf("failed to get session: %w", err)
	}

	// existing_entry.corporation MUST be equal to corporation
	msgCorpId, err := ms.corpIDFromAccount(ctx, msg.Corporation)
	if err != nil {
		return err
	}
	if existingSession.CorporationId != msgCorpId {
		return fmt.Errorf("session corporation does not match: expected %d, got %s", existingSession.CorporationId, msg.Corporation)
	}

	// existing_entry.vs_operator MUST be equal to operator
	if existingSession.VsOperator != msg.Operator {
		return fmt.Errorf("session vs_operator does not match: expected %s, got %s", existingSession.VsOperator, msg.Operator)
	}

	return nil
}

func (ms msgServer) createOrUpdateSession(ctx sdk.Context, msg *types.MsgCreateOrUpdateParticipantSession, now time.Time) error {
	corporationId, err := ms.corpIDFromAccount(ctx, msg.Corporation)
	if err != nil {
		return err
	}
	session := &types.ParticipantSession{
		Id:            msg.Id,
		CorporationId: corporationId,
		VsOperator:    msg.Operator,
		Modified:      &now,
	}

	existingSession, err := ms.ParticipantSession.Get(ctx, msg.Id)
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

	// Create ParticipantSessionRecord with its own uint64 id (sequential within
	// the session). agent_participant_id now lives on the record per spec v4-rc2.
	record := &types.ParticipantSessionRecord{
		Id:                       uint64(len(session.SessionRecords) + 1),
		Created:                  &now,
		IssuerParticipantId:      msg.IssuerParticipantId,
		VerifierParticipantId:    msg.VerifierParticipantId,
		WalletAgentParticipantId: msg.WalletAgentParticipantId,
		AgentParticipantId:       msg.AgentParticipantId,
	}

	// Add the record to session.session_records
	session.SessionRecords = append(session.SessionRecords, record)

	return ms.ParticipantSession.Set(ctx, msg.Id, *session)
}

// findBeneficiaries gets the set of permissions that should receive fees
func (ms msgServer) findBeneficiaries(ctx sdk.Context, issuerPermId, verifierPermId uint64) ([]types.Participant, error) {
	var foundPerms []types.Participant
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
		issuerPerm, err := ms.Participant.Get(ctx, issuerPermId)
		if err != nil {
			return nil, fmt.Errorf("issuer permission not found: %w", err)
		}
		schemaID = issuerPerm.SchemaId
	} else if verifierPermId != 0 {
		verifierPerm, err := ms.Participant.Get(ctx, verifierPermId)
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
	if (issuerPermId != 0 && cs.IssuerOnboardingMode == credentialschematypes.IssuerOnboardingMode_ISSUER_ONBOARDING_MODE_OPEN) ||
		(verifierPermId != 0 && cs.VerifierOnboardingMode == credentialschematypes.VerifierOnboardingMode_VERIFIER_ONBOARDING_MODE_OPEN) {
		isOpenMode = true
	}

	// For OPEN mode, find the ECOSYSTEM permission
	if isOpenMode {
		// Find ECOSYSTEM permission for this schema
		err = ms.Participant.Walk(ctx, nil, func(id uint64, perm types.Participant) (bool, error) {
			if perm.SchemaId == schemaID &&
				perm.Role == types.ParticipantRole_ECOSYSTEM &&
				perm.Revoked == nil && perm.Slashed == nil {
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
		issuerPerm, err := ms.Participant.Get(ctx, issuerPermId)
		if err != nil {
			return nil, fmt.Errorf("issuer permission not found: %w", err)
		}

		// Follow the validator chain up
		if issuerPerm.ValidatorParticipantId != 0 {
			currentPermID := issuerPerm.ValidatorParticipantId
			for currentPermID != 0 {
				currentPerm, err := ms.Participant.Get(ctx, currentPermID)
				if err != nil {
					return nil, fmt.Errorf("failed to get permission: %w", err)
				}

				// Add to set if valid and not already included
				if currentPerm.Revoked == nil && currentPerm.Slashed == nil && !containsPerm(currentPermID) {
					foundPerms = append(foundPerms, currentPerm)
				}

				// Move up
				currentPermID = currentPerm.ValidatorParticipantId
			}
		}
	}

	// Process verifier permission hierarchy if provided
	if verifierPermId != 0 {
		// First add issuer permission to the set if provided
		if issuerPermId != 0 {
			issuerPerm, err := ms.Participant.Get(ctx, issuerPermId)
			if err == nil && issuerPerm.Revoked == nil && !containsPerm(issuerPermId) {
				foundPerms = append(foundPerms, issuerPerm)
			}
		}

		// Then process verifier's validator chain
		verifierPerm, err := ms.Participant.Get(ctx, verifierPermId)
		if err != nil {
			return nil, fmt.Errorf("verifier permission not found: %w", err)
		}

		if verifierPerm.ValidatorParticipantId != 0 {
			currentPermID := verifierPerm.ValidatorParticipantId
			for currentPermID != 0 {
				currentPerm, err := ms.Participant.Get(ctx, currentPermID)
				if err != nil {
					return nil, fmt.Errorf("failed to get permission: %w", err)
				}

				// Add to set if valid and not already included
				if currentPerm.Revoked == nil && currentPerm.Slashed == nil && !containsPerm(currentPermID) {
					foundPerms = append(foundPerms, currentPerm)
				}

				// Move up
				currentPermID = currentPerm.ValidatorParticipantId
			}
		}
	}

	return foundPerms, nil
}
