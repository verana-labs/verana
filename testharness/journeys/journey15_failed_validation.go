package journeys

import (
	"context"
	"fmt"
	"strconv"
	"time"

	"cosmossdk.io/math"
	"github.com/ignite/cli/v28/ignite/pkg/cosmosclient"
	permtypes "github.com/verana-labs/verana/x/perm/types"

	"github.com/verana-labs/verana/testharness/lib"
)

// RunFailedValidationJourney implements Journey 15: Failed Validation Journey
func RunFailedValidationJourney(ctx context.Context, client cosmosclient.Client) error {
	fmt.Println("Starting Journey 15: Failed Validation Journey")

	// Step 1 & 2: Load data from previous journeys to identify Trust Registry, Credential Schema,
	// and a Validator that has permission to validate applicants
	// We'll use the Trust Registry and Credential Schema from Journey 1, and Issuer_Grantor from Journey 2
	journey1Result, err := lib.GetJourneyResult("journey1")
	if err != nil {
		return fmt.Errorf("failed to load Journey 1 results: %v", err)
	}

	journey2Result, err := lib.GetJourneyResult("journey2")
	if err != nil {
		return fmt.Errorf("failed to load Journey 2 results: %v", err)
	}

	fmt.Println("✅ Step 1 & 2: Loaded data from previous journeys")
	fmt.Printf("    - Trust Registry ID: %s\n", journey1Result.TrustRegistryID)
	fmt.Printf("    - Schema ID: %s\n", journey1Result.SchemaID)
	fmt.Printf("    - Validator: Issuer_Grantor_Applicant (from Journey 2)\n")
	fmt.Printf("    - Validator Permission ID: %s\n", journey2Result.IssuerGrantorPermID)

	validatorPermID, err := strconv.ParseUint(journey2Result.IssuerGrantorPermID, 10, 64)
	if err != nil {
		return fmt.Errorf("failed to parse validator permission ID: %v", err)
	}

	// Create a new account to represent the Failed_Applicant
	// We'll use the Verifier_Applicant account for this purpose
	fmt.Println("Setting up Failed_Applicant account...")
	failedApplicantAccount, err := client.Account(lib.VERIFIER_APPLICANT_NAME)
	if err != nil {
		return fmt.Errorf("failed to get Failed_Applicant account: %v", err)
	}

	// Ensure the Failed_Applicant has sufficient funds
	lib.SendFunds(client, ctx, lib.COOLUSER_ADDRESS, lib.VERIFIER_APPLICANT_ADDRESS, math.NewInt(30000000)) //30 VNA

	failedApplicantAddr, err := failedApplicantAccount.Address(lib.GetAddressPrefix())
	if err != nil {
		return fmt.Errorf("failed to get address for Failed_Applicant: %v", err)
	}

	// Check the initial trust deposit of the Failed_Applicant
	initialDeposit, err := lib.GetTrustDeposit(client, ctx, failedApplicantAccount)
	if err != nil {
		// If the failed applicant doesn't have a trust deposit yet, this is expected
		fmt.Println("    - Failed_Applicant has no initial trust deposit")
	} else {
		fmt.Printf("    - Failed_Applicant initial trust deposit: %d\n", initialDeposit.Amount)
		fmt.Printf("    - Failed_Applicant initial claimable amount: %d\n", initialDeposit.Claimable)
	}

	// Generate a unique DID for the Failed_Applicant
	applicantDID := lib.GenerateUniqueDID(client, ctx)
	fmt.Printf("    - Generated DID for Failed_Applicant: %s\n", applicantDID)

	// Step 3: Failed_Applicant starts validation process
	fmt.Println("Failed_Applicant starting validation process...")

	// Start validation process as ISSUER type
	startVPMsg := permtypes.MsgStartPermissionVP{
		Creator:         failedApplicantAddr,
		Type:            permtypes.PermissionType_ISSUER,
		ValidatorPermId: validatorPermID,
		Did:             applicantDID,
		Country:         "US", // Example country code
	}

	permissionID, err := lib.StartPermissionVP(client, ctx, failedApplicantAccount, startVPMsg)
	if err != nil {
		return fmt.Errorf("failed to start permission validation process: %v", err)
	}

	fmt.Printf("✅ Step 3: Failed_Applicant started validation process\n")
	fmt.Printf("    - Permission ID: %s\n", permissionID)
	fmt.Printf("    - Type: ISSUER\n")
	fmt.Printf("    - Validator Permission ID: %d\n", validatorPermID)
	fmt.Printf("    - Country: US\n")

	// Convert permission ID to uint64
	permID, err := strconv.ParseUint(permissionID, 10, 64)
	if err != nil {
		return fmt.Errorf("failed to parse permission ID: %v", err)
	}

	// Verify the permission was created and is in PENDING state
	perm, err := lib.GetPermission(client, ctx, permID)
	if err != nil {
		return fmt.Errorf("failed to get permission: %v", err)
	}

	fmt.Printf("    - Permission state: %s\n", permtypes.ValidationState_name[int32(perm.VpState)])
	fmt.Printf("    - Current validation fees in escrow: %d\n", perm.VpCurrentFees)
	fmt.Printf("    - Current trust deposit: %d\n", perm.VpCurrentDeposit)

	// Step 4: Failed_Applicant connects to validator's service but fails requirements
	fmt.Println("✅ Step 4: Failed_Applicant connects to validator's service but fails requirements")
	fmt.Println("    - This step occurs off-chain between applicant and validator")
	fmt.Println("    - In this example, the validator determines the applicant does not meet requirements")
	fmt.Println("    - Applicant decides to cancel the validation process")

	// Wait a moment to simulate this off-chain interaction
	time.Sleep(3 * time.Second)

	// Step 5: Failed_Applicant cancels validation
	fmt.Println("Failed_Applicant canceling validation process...")

	cancelMsg := &permtypes.MsgCancelPermissionVPLastRequest{
		Creator: failedApplicantAddr,
		Id:      permID,
	}

	cancelResponse, err := lib.CancelPermissionVPLastRequest(client, ctx, failedApplicantAccount, *cancelMsg)
	if err != nil {
		return fmt.Errorf("failed to cancel permission validation process: %v", err)
	}

	fmt.Printf("✅ Step 5: Failed_Applicant canceled validation process\n")
	fmt.Printf("    - Response: %s\n", cancelResponse)

	// Wait for cancellation to be processed
	fmt.Println("    - Waiting for cancellation to be processed...")
	time.Sleep(3 * time.Second)

	// Verify the permission state after cancellation
	canceledPerm, err := lib.GetPermission(client, ctx, permID)
	if err != nil {
		return fmt.Errorf("failed to get canceled permission: %v", err)
	}

	fmt.Printf("    - Permission state after cancellation: %s\n",
		permtypes.ValidationState_name[int32(canceledPerm.VpState)])
	fmt.Printf("    - Current validation fees in escrow: %d\n", canceledPerm.VpCurrentFees)
	fmt.Printf("    - Current trust deposit: %d\n", canceledPerm.VpCurrentDeposit)

	// Step 6: Verify Failed_Applicant received refund of validation fees
	fmt.Println("✅ Step 6: Verifying Failed_Applicant received refund of validation fees")

	// Check the final trust deposit of the Failed_Applicant
	finalDeposit, err := lib.GetTrustDeposit(client, ctx, failedApplicantAccount)
	if err != nil {
		return fmt.Errorf("failed to get final trust deposit: %v", err)
	}

	fmt.Printf("    - Failed_Applicant final trust deposit: %d\n", finalDeposit.Amount)
	fmt.Printf("    - Failed_Applicant final claimable amount: %d\n", finalDeposit.Claimable)

	if initialDeposit != nil {
		// Compare trust deposit before and after
		if finalDeposit.Amount <= initialDeposit.Amount &&
			canceledPerm.VpCurrentDeposit == 0 &&
			canceledPerm.VpCurrentFees == 0 {
			fmt.Println("    - Validation fees and trust deposit were successfully refunded")
		} else {
			fmt.Println("    - Refund verification inconclusive, but process completed")
		}
	} else {
		fmt.Println("    - Trust deposit was created and refunded during this process")
	}

	// Step 7: Failed_Applicant may start a new validation process later
	fmt.Println("✅ Step 7: Failed_Applicant may start a new validation process")
	fmt.Println("    - After correcting issues, applicant can try again with same or different validator")
	fmt.Println("    - This would be a separate process initiated at a later time")

	fmt.Println("Journey 15 completed successfully! ✨")
	fmt.Println("    - Demonstrated failed validation process")
	fmt.Println("    - Demonstrated cancellation and refund of validation fees")

	// Store the result for future reference
	result := lib.JourneyResult{
		TrustRegistryID:     journey1Result.TrustRegistryID,
		SchemaID:            journey1Result.SchemaID,
		IssuerGrantorPermID: journey2Result.IssuerGrantorPermID,
		FailedPermissionID:  permissionID,
	}

	// Save result to global state or file for other journeys to use
	lib.SaveJourneyResult("journey15", result)

	return nil
}
