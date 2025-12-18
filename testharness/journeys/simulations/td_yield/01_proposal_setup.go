package td_yield

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/verana-labs/verana/testharness/lib"

	govtypes "github.com/cosmos/cosmos-sdk/x/gov/types/v1"
	"github.com/ignite/cli/v28/ignite/pkg/cosmosclient"
)

// SetupFundingProposal sets up the continuous funding proposal for Yield Intermediate Pool
// This should be run once before running the simulations
func SetupFundingProposal(ctx context.Context, client cosmosclient.Client, fundingPercentage string) (uint64, error) {
	fmt.Println("\n" + strings.Repeat("=", 80))
	fmt.Println("TD Yield Simulation - Proposal Setup")
	fmt.Println(strings.Repeat("=", 80))
	fmt.Printf("\nSetting up continuous funding proposal with %s%% of block rewards\n", fundingPercentage)

	govModuleAddr, err := lib.GetGovModuleAddress(client, ctx)
	if err != nil {
		return 0, fmt.Errorf("failed to get governance module address: %v", err)
	}

	yipAddr, err := lib.GetYieldIntermediatePoolAddress(client, ctx)
	if err != nil {
		return 0, fmt.Errorf("failed to get Yield Intermediate Pool address: %v", err)
	}

	coolUserAccount := lib.GetAccount(client, lib.COOLUSER_NAME)
	title := "Continuous funding for Yield Intermediate Pool - TD Yield Simulation"
	summary := fmt.Sprintf("Creates continuous fund to send %s%% of community pool contributions to the Yield Intermediate Pool for distributing yield to trust deposit holders", fundingPercentage)

	proposalID, err := lib.SubmitContinuousFundProposal(
		client, ctx, coolUserAccount,
		govModuleAddr, yipAddr, fundingPercentage,
		title, summary,
	)
	if err != nil {
		return 0, fmt.Errorf("failed to submit continuous fund proposal: %v", err)
	}
	fmt.Printf("✅ Submitted continuous fund proposal with ID: %d\n", proposalID)

	err = lib.VoteOnGovProposal(client, ctx, coolUserAccount, proposalID, govtypes.OptionYes)
	if err != nil {
		return 0, fmt.Errorf("failed to vote on proposal: %v", err)
	}
	fmt.Printf("✅ Voted YES on proposal %d\n", proposalID)

	fmt.Println("⏳ Waiting for proposal to pass...")
	err = lib.WaitForProposalToPass(client, ctx, proposalID, 40)
	if err != nil {
		return 0, fmt.Errorf("proposal did not pass: %v", err)
	}
	fmt.Printf("✅ Proposal %d has passed and executed\n", proposalID)

	// Wait a bit for the proposal to execute
	time.Sleep(10 * time.Second)

	fmt.Println("\n" + strings.Repeat("=", 80))
	fmt.Println("✅ Proposal setup completed successfully!")
	fmt.Println(strings.Repeat("=", 80))

	return proposalID, nil
}
