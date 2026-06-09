package journeys

import (
	"context"
	"fmt"

	authtypes "github.com/cosmos/cosmos-sdk/x/auth/types"
	"github.com/ignite/cli/v28/ignite/pkg/cosmosclient"

	"github.com/verana-labs/verana/testharness/lib"
)

// RunTdGovSlashJourney implements Journey 402: TD governance slash (MOD-TD-MSG-5).
// Slashes a corporation's trust deposit by corporation_id via a governance proposal
// (td.MsgSlashTrustDeposit, signer = gov authority). The harness helper resolves the
// corporation policy_address to its corporation_id. Depends on Journey 301 (corp) and
// the perm flows (304/307) that build the corp's account-level trust deposit.
func RunTdGovSlashJourney(ctx context.Context, client cosmosclient.Client) error {
	fmt.Println("Starting Journey 402: TD Governance Slash (MsgSlashTrustDeposit by corporation_id)")

	setup301 := lib.LoadJourneyResult("journey301")
	policyAddr := setup301.GroupPolicyAddr
	govAuthority := authtypes.NewModuleAddress("gov").String()
	cooluser := lib.GetAccount(client, lib.COOLUSER_NAME)

	// Step 1: Read the corporation's trust deposit
	fmt.Println("\n--- Step 1: Query corporation trust deposit ---")
	tdBefore, err := queryTrustDepositByAddr(client, ctx, policyAddr)
	if err != nil {
		return fmt.Errorf("step 1 failed: could not query TD for %s: %w", policyAddr, err)
	}
	fmt.Printf("  Deposit: %d, SlashedDeposit: %d\n", tdBefore.Deposit, tdBefore.SlashedDeposit)
	if tdBefore.Deposit == 0 {
		return fmt.Errorf("step 1 failed: corporation has no trust deposit to slash")
	}

	slashAmount := tdBefore.Deposit / 10
	if slashAmount == 0 {
		slashAmount = 1
	}
	fmt.Printf("  Slashing amount: %d\n", slashAmount)

	// Step 2: Submit + pass the gov slash proposal (helper resolves account -> corporation_id)
	fmt.Println("\n--- Step 2: Submit + pass MsgSlashTrustDeposit gov proposal ---")
	proposalID, err := lib.SubmitSlashTrustDepositProposal(
		client, ctx, cooluser, govAuthority, policyAddr, slashAmount,
		"Slash corporation trust deposit", "Governance slash for coverage journey",
	)
	if err != nil {
		return fmt.Errorf("step 2 failed: submit slash proposal: %w", err)
	}
	if err := voteAndPassGovProposal(client, ctx, proposalID); err != nil {
		return fmt.Errorf("step 2 failed: %w", err)
	}
	fmt.Println("OK Step 2: slash proposal passed")

	// Step 3: Verify slashed_deposit increased
	fmt.Println("\n--- Step 3: Verify trust deposit was slashed ---")
	tdAfter, err := queryTrustDepositByAddr(client, ctx, policyAddr)
	if err != nil {
		return fmt.Errorf("step 3 failed: could not re-query TD: %w", err)
	}
	fmt.Printf("  Deposit: %d -> %d, SlashedDeposit: %d -> %d\n",
		tdBefore.Deposit, tdAfter.Deposit, tdBefore.SlashedDeposit, tdAfter.SlashedDeposit)
	if tdAfter.SlashedDeposit <= tdBefore.SlashedDeposit {
		return fmt.Errorf("step 3 failed: slashed_deposit did not increase (%d -> %d)", tdBefore.SlashedDeposit, tdAfter.SlashedDeposit)
	}
	if tdAfter.Deposit != tdBefore.Deposit-slashAmount {
		return fmt.Errorf("step 3 failed: deposit expected %d, got %d", tdBefore.Deposit-slashAmount, tdAfter.Deposit)
	}
	fmt.Println("OK Step 3: trust deposit slashed correctly")

	fmt.Println("\n========================================")
	fmt.Println("Journey 402 completed successfully!")
	fmt.Println("TD governance slash by corporation_id validated")
	fmt.Println("========================================")
	return nil
}
