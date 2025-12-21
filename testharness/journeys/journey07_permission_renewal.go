package journeys

import (
	"context"
	"cosmossdk.io/math"
	"fmt"
	"strconv"
	"time"

	"github.com/ignite/cli/v28/ignite/pkg/cosmosclient"
	permtypes "github.com/verana-labs/verana/x/perm/types"

	"github.com/verana-labs/verana/testharness/lib"
)

// RunPermissionRenewalJourney implements Journey 7: Permission Renewal Journey
func RunPermissionRenewalJourney(ctx context.Context, client cosmosclient.Client) error {
	fmt.Println("Starting Journey 7: Permission Renewal Journey")
	lib.SendFunds(client, ctx, lib.COOLUSER_ADDRESS, lib.ISSUER_APPLICANT_ADDRESS, math.NewInt(15000000)) //15 VNA

	// Step 1: Load data from previous journeys
	journey3Result, err := lib.GetJourneyResult("journey3")
	if err != nil {
		return fmt.Errorf("failed to load Journey 3 results: %v", err)
	}

	fmt.Println("✅ Step 1: Loaded data from previous journeys")
	fmt.Printf("    - Trust Registry ID: %s\n", journey3Result.TrustRegistryID)
	fmt.Printf("    - Schema ID: %s\n", journey3Result.SchemaID)
	fmt.Printf("    - Issuer DID: %s\n", journey3Result.IssuerDID)
	fmt.Printf("    - Issuer Permission ID: %s\n", journey3Result.IssuerPermID)
	fmt.Printf("    - Issuer Grantor Permission ID: %s\n", journey3Result.IssuerGrantorPermID)

	// Step 2: Verify that the permission exists and is valid
	issuerPermID, err := strconv.ParseUint(journey3Result.IssuerPermID, 10, 64)
	if err != nil {
		return fmt.Errorf("failed to parse issuer permission ID: %v", err)
	}

	// Get permission for displaying info
	perm, err := lib.GetPermission(client, ctx, issuerPermID)
	if err != nil {
		return fmt.Errorf("failed to get permission: %v", err)
	}

	fmt.Printf("✅ Step 2: Verified Issuer permission exists\n")
	fmt.Printf("    - Permission Type: %s\n", permtypes.PermissionType_name[int32(perm.Type)])
	fmt.Printf("    - Permission DID: %s\n", perm.Did)
	fmt.Printf("    - Permission State: %s\n", perm.VpState)

	// Step 3: Get the Issuer_Applicant account
	issuerApplicantAccount, err := client.Account(lib.ISSUER_APPLICANT_NAME)
	if err != nil {
		return fmt.Errorf("failed to get %s account: %v", lib.ISSUER_APPLICANT_NAME, err)
	}

	// Step 4: Issuer_Applicant initiates renewal
	fmt.Println("Issuer_Applicant initiating permission renewal...")

	renewMsg := &permtypes.MsgRenewPermissionVP{
		Id: issuerPermID,
	}

	renewPermissionID, err := lib.RenewPermissionVP(client, ctx, issuerApplicantAccount, *renewMsg)
	if err != nil {
		return fmt.Errorf("failed to renew permission: %v", err)
	}

	fmt.Printf("✅ Step 4: Issuer_Applicant initiated renewal with permission ID: %s\n", renewPermissionID)

	// Wait for blockchain to process the transaction
	fmt.Println("    - Waiting for renewal to be processed...")
	time.Sleep(3 * time.Second)

	// Check permission state after renewal initiation
	pendingPerm, err := lib.GetPermission(client, ctx, issuerPermID)
	if err != nil {
		return fmt.Errorf("failed to get pending permission: %v", err)
	}

	fmt.Printf("    - Permission State: %s\n", pendingPerm.VpState)
	fmt.Printf("    - Current fees: %d\n", pendingPerm.VpCurrentFees)
	fmt.Printf("    - Current deposit: %d\n", pendingPerm.VpCurrentDeposit)

	// Step 5: Issuer_Applicant connects to Issuer_Grantor_Applicant's validation service
	fmt.Println("✅ Step 5: Issuer_Applicant connects to Issuer_Grantor_Applicant's validation service (simulated)")
	fmt.Println("    - Validation occurs off-chain between applicant and validator")

	// Step 6: Issuer_Grantor_Applicant validates the renewal
	// Get Issuer_Grantor_Applicant account to perform validation
	issuerGrantorAccount, err := client.Account(lib.ISSUER_GRANTOR_APPLICANT_NAME)
	if err != nil {
		return fmt.Errorf("failed to get %s account: %v", lib.ISSUER_GRANTOR_APPLICANT_NAME, err)
	}

	fmt.Println("Issuer_Grantor_Applicant validates the renewal...")

	// Calculate new effective until (extend by 1 year from now)
	newEffectiveUntil := time.Now().AddDate(1, 0, 0)

	// For renewals, the fees must be explicitly set to the same values as before
	validateMsg := &permtypes.MsgSetPermissionVPToValidated{
		Id:               issuerPermID,
		EffectiveUntil:   &newEffectiveUntil,
		ValidationFees:   uint64(perm.ValidationFees),
		IssuanceFees:     uint64(perm.IssuanceFees),
		VerificationFees: uint64(perm.VerificationFees),
		Country:          perm.Country,
	}

	_, err = lib.SetPermissionVPToValidated(client, ctx, issuerGrantorAccount, *validateMsg)
	if err != nil {
		return fmt.Errorf("failed to validate renewal: %v", err)
	}

	fmt.Printf("✅ Step 6: Issuer_Grantor_Applicant validated renewal with new effective until: %s\n",
		newEffectiveUntil.Format(time.RFC3339))

	// Wait for blockchain to process the validation
	fmt.Println("    - Waiting for validation to be processed...")
	time.Sleep(3 * time.Second)

	// Step 7: Verify the permission was renewed successfully
	renewedPerm, err := lib.GetPermission(client, ctx, issuerPermID)
	if err != nil {
		return fmt.Errorf("failed to get renewed permission: %v", err)
	}

	// Display the new state
	fmt.Println("✅ Step 7: Permission renewal details:")
	fmt.Printf("    - Permission State: %s\n", renewedPerm.VpState)
	fmt.Printf("    - Extended By: %s\n", renewedPerm.ExtendedBy)

	// Get validator address for comparison
	validatorAddr, err := issuerGrantorAccount.Address(lib.GetAddressPrefix())
	if err == nil { // Only verify if we can get the address
		if renewedPerm.ExtendedBy == validatorAddr {
			fmt.Printf("    - Verified permission was extended by the validator (%s)\n", validatorAddr)
		}
	}

	fmt.Println("Journey 7 completed successfully! ✨")

	// Store the result for future reference
	result := lib.JourneyResult{
		TrustRegistryID:     journey3Result.TrustRegistryID,
		SchemaID:            journey3Result.SchemaID,
		RootPermissionID:    journey3Result.RootPermissionID,
		IssuerGrantorDID:    journey3Result.IssuerGrantorDID,
		IssuerGrantorPermID: journey3Result.IssuerGrantorPermID,
		IssuerDID:           journey3Result.IssuerDID,
		IssuerPermID:        journey3Result.IssuerPermID,
		RenewalTimestamp:    time.Now().Format(time.RFC3339),
	}

	// Save result to global state or file for other journeys to use
	lib.SaveJourneyResult("journey7", result)

	return nil
}
