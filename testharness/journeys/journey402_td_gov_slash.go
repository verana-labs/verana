package journeys

import (
	"context"
	"fmt"
	"strconv"

	"cosmossdk.io/math"
	authtypes "github.com/cosmos/cosmos-sdk/x/auth/types"
	"github.com/ignite/cli/v28/ignite/pkg/cosmosclient"

	permtypes "github.com/verana-labs/verana/x/pp/types"

	"github.com/verana-labs/verana/testharness/lib"
)

// RunTdGovSlashJourney implements Journey 402: TD governance slash (MOD-TD-MSG-5).
// Slashes a corporation's trust deposit by corporation_id via a governance proposal
// (td.MsgSlashTrustDeposit, signer = gov authority). To guarantee a slashable
// (non-released) balance, the operator first starts a fresh participant VP, which
// locks a trust deposit that persists (it is not cancelled). Depends on Journey
// 302 (corp + operator) and 304 (root permission used as validator).
func RunTdGovSlashJourney(ctx context.Context, client cosmosclient.Client) error {
	fmt.Println("Starting Journey 402: TD Governance Slash (MsgSlashTrustDeposit by corporation_id)")

	setup302 := lib.LoadJourneyResult("journey302")
	setup304 := lib.LoadJourneyResult("journey304")
	policyAddr := setup302.GroupPolicyAddr
	operatorAddr := setup302.OperatorAddr
	operatorAccount := lib.GetAccount(client, permOperatorName)
	adminAccount := lib.GetAccount(client, permGroupAdminName)
	member1Account := lib.GetAccount(client, permGroupMember1Name)
	govAuthority := authtypes.NewModuleAddress("gov").String()
	cooluser := lib.GetAccount(client, lib.COOLUSER_NAME)

	rootPermID, err := strconv.ParseUint(setup304.PermissionID, 10, 64)
	if err != nil {
		return fmt.Errorf("step 0 failed: parse root permission id: %w", err)
	}

	// Step 0: grant operator authz for MsgStartParticipantOP (replaced by later grants)
	// Step 1: Ensure the corporation has a (non-released) trust deposit to slash.
	// If it does not already have one, lock a fresh one by starting a participant
	// VP: StartParticipantOP charges validation fees to escrow plus a trust
	// deposit (validator fees * trust_unit_price * trust_deposit_rate). We use
	// VERIFIER_GRANTOR (a role the rest of the flow never uses against this
	// validator, so no overlap) and never cancel it, so the deposit persists.
	fmt.Println("\n--- Step 1: Ensure corporation has a trust deposit ---")
	tdBefore, err := queryTrustDepositByAddr(client, ctx, policyAddr)
	if err != nil || tdBefore == nil || tdBefore.Deposit == 0 {
		fmt.Println("  No existing deposit; locking one via a participant VP")
		if err := lib.GrantOperatorAuthorizationViaGroup(
			client, ctx, adminAccount, member1Account,
			policyAddr, operatorAddr, operatorAddr,
			[]string{"/verana.pp.v1.MsgStartParticipantOP"},
		); err != nil {
			return fmt.Errorf("step 1 failed: grant start-op authz: %w", err)
		}
		waitForTx("grant start-op authz")

		lib.SendFunds(client, ctx, lib.COOLUSER_ADDRESS, policyAddr, math.NewInt(200000000))
		waitForTx("fund policy for VP")

		if _, err := lib.StartPermissionVPWithAuthority(
			client, ctx, operatorAccount, policyAddr,
			permtypes.ParticipantRole_VERIFIER_GRANTOR,
			rootPermID,
			"did:example:td-gov-slash-deposit",
		); err != nil {
			return fmt.Errorf("step 1 failed: start VP: %w", err)
		}
		waitForTx("start vp deposit")

		tdBefore, err = queryTrustDepositByAddr(client, ctx, policyAddr)
		if err != nil {
			return fmt.Errorf("step 1 failed: could not query TD for %s: %w", policyAddr, err)
		}
	}

	// Step 2: confirm a slashable deposit exists
	fmt.Println("\n--- Step 2: Corporation trust deposit ---")
	fmt.Printf("  Deposit: %d, SlashedDeposit: %d\n", tdBefore.Deposit, tdBefore.SlashedDeposit)
	if tdBefore.Deposit == 0 {
		return fmt.Errorf("step 2 failed: corporation has no trust deposit to slash")
	}

	slashAmount := tdBefore.Deposit / 10
	if slashAmount == 0 {
		slashAmount = 1
	}
	fmt.Printf("  Slashing amount: %d\n", slashAmount)

	// Step 3: Submit + pass the gov slash proposal (helper resolves account -> corporation_id)
	fmt.Println("\n--- Step 3: Submit + pass MsgSlashTrustDeposit gov proposal ---")
	proposalID, err := lib.SubmitSlashTrustDepositProposal(
		client, ctx, cooluser, govAuthority, policyAddr, slashAmount,
		"Slash corporation trust deposit", "Governance slash for coverage journey",
	)
	if err != nil {
		return fmt.Errorf("step 3 failed: submit slash proposal: %w", err)
	}
	if err := voteAndPassGovProposal(client, ctx, proposalID); err != nil {
		return fmt.Errorf("step 3 failed: %w", err)
	}
	fmt.Println("OK Step 3: slash proposal passed")

	// Step 4: Verify slashed_deposit increased
	fmt.Println("\n--- Step 4: Verify trust deposit was slashed ---")
	tdAfter, err := queryTrustDepositByAddr(client, ctx, policyAddr)
	if err != nil {
		return fmt.Errorf("step 4 failed: could not re-query TD: %w", err)
	}
	fmt.Printf("  Deposit: %d -> %d, SlashedDeposit: %d -> %d\n",
		tdBefore.Deposit, tdAfter.Deposit, tdBefore.SlashedDeposit, tdAfter.SlashedDeposit)
	if tdAfter.SlashedDeposit <= tdBefore.SlashedDeposit {
		return fmt.Errorf("step 4 failed: slashed_deposit did not increase (%d -> %d)", tdBefore.SlashedDeposit, tdAfter.SlashedDeposit)
	}
	fmt.Println("OK Step 4: trust deposit slashed correctly")

	fmt.Println("\n========================================")
	fmt.Println("Journey 402 completed successfully!")
	fmt.Println("TD governance slash by corporation_id validated")
	fmt.Println("========================================")
	return nil
}
