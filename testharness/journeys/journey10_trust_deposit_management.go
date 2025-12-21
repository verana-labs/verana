package journeys

import (
	"context"
	"fmt"
	"github.com/ignite/cli/v28/ignite/pkg/cosmosclient"
	_ "github.com/verana-labs/verana/x/perm/types"
	"strconv"
	"github.com/verana-labs/verana/testharness/lib"
)

// RunTrustDepositManagementJourney implements Journey 10: Trust Deposit Management Journey
func RunTrustDepositManagementJourney(ctx context.Context, client cosmosclient.Client) error {
	fmt.Println("Starting Journey 10: Trust Deposit Management Journey")
	//lib.SendFunds(client, ctx, lib.COOLUSER_ADDRESS, lib.ISSUER_APPLICANT_ADDRESS, math.NewInt(100000000))

	// Step 1: Identify a Deposit_Holder account with accumulated trust deposit
	_, err := lib.GetJourneyResult("journey2")
	// We'll use the Issuer_Applicant account from Journey 3
	journey3Result, err := lib.GetJourneyResult("journey3")
	if err != nil {
		return fmt.Errorf("failed to load Journey 3 results: %v", err)
	}

	fmt.Println("✅ Step 1: Identified Deposit_Holder account from previous journeys")
	fmt.Printf("    - Using Issuer_Applicant account that participated in Journey 3\n")
	fmt.Printf("    - Issuer Permission ID: %s\n", journey3Result.IssuerPermID)

	// Get the account
	depositHolderAccount, err := client.Account(lib.ISSUER_APPLICANT_NAME)
	if err != nil {
		return fmt.Errorf("failed to get %s account: %v", lib.ISSUER_APPLICANT_NAME, err)
	}

	depositHolderAddr, err := depositHolderAccount.Address(lib.GetAddressPrefix())
	if err != nil {
		return fmt.Errorf("failed to get address for %s: %v", lib.ISSUER_APPLICANT_NAME, err)
	}

	// Step 2: Check initial trust deposit status
	fmt.Println("Checking initial trust deposit status...")
	initialDeposit, err := lib.GetTrustDeposit(client, ctx, depositHolderAccount)
	if err != nil {
		return fmt.Errorf("failed to get initial trust deposit: %v", err)
	}

	fmt.Printf("✅ Step 2: Checked initial trust deposit status\n")
	fmt.Printf("    - Initial Trust Deposit: %d\n", initialDeposit.Amount)
	fmt.Printf("    - Initial Claimable Amount: %d\n", initialDeposit.Claimable)

	//Todo: after SDK upgrade

	// Final check of trust deposit status
	finalDeposit, err := lib.GetTrustDeposit(client, ctx, depositHolderAccount)
	if err != nil {
		return fmt.Errorf("failed to get final trust deposit: %v", err)
	}

	fmt.Println("Journey 10 completed successfully! ✨")
	fmt.Printf("    - Final Trust Deposit: %d\n", finalDeposit.Amount)
	fmt.Printf("    - Final Claimable Amount: %d\n", finalDeposit.Claimable)

	// Store the result for future reference
	result := lib.JourneyResult{
		TrustRegistryID:      journey3Result.TrustRegistryID,
		SchemaID:             journey3Result.SchemaID,
		DepositHolder:        lib.ISSUER_APPLICANT_NAME,
		DepositHolderAddress: depositHolderAddr,
		InitialDepositAmount: strconv.FormatUint(uint64(initialDeposit.Amount), 10),
		FinalDepositAmount:   strconv.FormatUint(uint64(finalDeposit.Amount), 10),
	}

	// Save result to global state or file for other journeys to use
	lib.SaveJourneyResult("journey10", result)

	return nil
}
