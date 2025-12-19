package journeys

import (
	"context"
	"fmt"
	"strconv"

	"cosmossdk.io/math"

	"github.com/verana-labs/verana/testharness/lib"

	govtypes "github.com/cosmos/cosmos-sdk/x/gov/types/v1"
	"github.com/ignite/cli/v28/ignite/pkg/cosmosclient"
	permtypes "github.com/verana-labs/verana/x/perm/types"
)

// RunSlashPermissionTrustDepositJourney implements Journey 16: Slash Permission Trust Deposit via Governance Proposal
func RunSlashPermissionTrustDepositJourney(ctx context.Context, client cosmosclient.Client) error {
	fmt.Println("Starting Journey 16: Slash Permission Trust Deposit via Governance Proposal")
	lib.SendFunds(client, ctx, lib.COOLUSER_ADDRESS, lib.ISSUER_APPLICANT_ADDRESS, math.NewInt(30000000)) //30 VNA

	issuerDID := lib.GenerateUniqueDID(client, ctx)
	fmt.Printf("    - Generated Issuer DID: %s\n", issuerDID)
	journey2Result := lib.LoadJourneyResult("journey2")

	fmt.Println("Issuer_Applicant starting Permission Validation Process...")
	countryCode := "US" // Example country code
	issuerApplicantAccount := lib.GetAccount(client, lib.ISSUER_APPLICANT_NAME)
	issuerGrantorPermID, _ := strconv.ParseUint(journey2Result.IssuerGrantorPermID, 10, 64)

	startVPMsg := permtypes.MsgStartPermissionVP{
		Type:            permtypes.PermissionType_ISSUER,
		ValidatorPermId: issuerGrantorPermID,
		Country:         countryCode,
		Did:             issuerDID,
	}

	permissionID := lib.StartValidationProcess(client, ctx, issuerApplicantAccount, startVPMsg)
	fmt.Printf("✅ Step 1: Issuer_Applicant started validation process with permission ID: %s\n", permissionID)
	issuerPermID, _ := strconv.ParseUint(permissionID, 10, 64)

	// Query the permission to get its deposit amount
	perm, err := lib.GetPermission(client, ctx, issuerPermID)
	if err != nil {
		return fmt.Errorf("failed to query permission: %v", err)
	}
	fmt.Printf("Queried permission %d, deposit: %d\n", issuerPermID, perm.Deposit)

	// Query the trust deposit before slashing
	trustDepositBefore, err := lib.GetTrustDeposit(client, ctx, issuerApplicantAccount)
	if err != nil {
		return fmt.Errorf("failed to get trust deposit before slashing: %v", err)
	}
	fmt.Printf("Trust deposit before slashing - Amount: %d, Slash Count: %d\n",
		trustDepositBefore.Amount, trustDepositBefore.SlashCount)
	initialSlashCount := trustDepositBefore.SlashCount

	if perm.Deposit == 0 {
		fmt.Println("Permission deposit is 0, skipping slash step.")
		return nil
	}

	// Get issuer applicant address for the proposal
	issuerApplicantAddr, err := issuerApplicantAccount.Address(lib.GetAddressPrefix())
	if err != nil {
		return fmt.Errorf("failed to get issuer applicant address: %v", err)
	}

	// Get governance module address
	govModuleAddr, err := lib.GetGovModuleAddress(client, ctx)
	if err != nil {
		return fmt.Errorf("failed to get governance module address: %v", err)
	}
	fmt.Printf("Governance module address: %s\n", govModuleAddr)

	// Step 2: Submit the governance proposal directly
	amountToSlash := perm.Deposit // Slash the full available deposit for demo
	coolUserAccount := lib.GetAccount(client, lib.COOLUSER_NAME)

	title := "Slash Trust Deposit for Permission Violation"
	summary := fmt.Sprintf("This proposal requests to slash %d uvna from the trust deposit of account %s for permission %d. The account will need to repay this slashed amount before being able to participate in the VPR again.", amountToSlash, issuerApplicantAddr, issuerPermID)

	proposalID, err := lib.SubmitSlashTrustDepositProposal(
		client, ctx, coolUserAccount,
		govModuleAddr, issuerApplicantAddr, amountToSlash,
		title, summary,
	)
	if err != nil {
		return fmt.Errorf("failed to submit governance proposal: %v", err)
	}
	fmt.Printf("✅ Step 2: Submitted governance proposal with ID: %d\n", proposalID)

	// Step 3: Vote on the proposal (cooluser has most voting power)
	err = lib.VoteOnGovProposal(client, ctx, coolUserAccount, proposalID, govtypes.OptionYes)
	if err != nil {
		return fmt.Errorf("failed to vote on proposal: %v", err)
	}
	fmt.Printf("✅ Step 3: Voted YES on proposal %d\n", proposalID)

	// Step 4: Wait for voting period to end (110 seconds as voting period is 100 seconds)
	err = lib.WaitForProposalToPass(client, ctx, proposalID, 110)
	if err != nil {
		return fmt.Errorf("proposal did not pass: %v", err)
	}
	fmt.Printf("✅ Step 4: Proposal %d has passed and executed\n", proposalID)

	// Step 5: Verify the slash - check that slash count has increased
	trustDepositAfter, err := lib.GetTrustDeposit(client, ctx, issuerApplicantAccount)
	if err != nil {
		return fmt.Errorf("failed to get trust deposit after slashing: %v", err)
	}
	fmt.Printf("Trust deposit after slashing - Amount: %d, Slash Count: %d\n",
		trustDepositAfter.Amount, trustDepositAfter.SlashCount)

	// Verify slash count increased
	if trustDepositAfter.SlashCount <= initialSlashCount {
		return fmt.Errorf("slash count verification failed: expected > %d, got %d",
			initialSlashCount, trustDepositAfter.SlashCount)
	}
	fmt.Printf("✅ Step 5: Verified slash count increased from %d to %d\n",
		initialSlashCount, trustDepositAfter.SlashCount)

	// Verify slashed amount is correct
	expectedSlashedAmount := trustDepositBefore.SlashedDeposit + amountToSlash
	if trustDepositAfter.SlashedDeposit != expectedSlashedAmount {
		return fmt.Errorf("slashed deposit verification failed: expected %d, got %d",
			expectedSlashedAmount, trustDepositAfter.SlashedDeposit)
	}
	fmt.Printf("✅ Verified slashed deposit amount: %d\n", trustDepositAfter.SlashedDeposit)

	// Save journey result for use in future journeys
	result := lib.JourneyResult{
		PermissionID:        permissionID,
		IssuerDID:           issuerDID,
		IssuerGrantorPermID: journey2Result.IssuerGrantorPermID,
	}
	lib.SaveJourneyResult("journey16", result)

	return nil
}
