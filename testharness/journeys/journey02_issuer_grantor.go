package journeys

import (
	"context"
	"fmt"
	"strconv"

	"cosmossdk.io/math"
	"github.com/ignite/cli/v28/ignite/pkg/cosmosclient"

	permtypes "github.com/verana-labs/verana/x/perm/types"

	"github.com/verana-labs/verana/testharness/lib"
)

// RunIssuerGrantorJourney implements Journey 2: Issuer Grantor Validation Journey
func RunIssuerGrantorJourney(ctx context.Context, client cosmosclient.Client) error {
	fmt.Println("Starting Journey 2: Issuer Grantor Validation Journey")

	// Step 1: Get account and fund it
	fmt.Println("Sending funds from cooluser to Issuer_Grantor_Applicant...")
	issuerGrantorAccount := lib.GetAccount(client, lib.ISSUER_GRANTOR_APPLICANT_NAME)
	lib.SendFunds(client, ctx, lib.COOLUSER_ADDRESS, lib.ISSUER_GRANTOR_APPLICANT_ADDRESS, math.NewInt(10000000)) // 10 VNA
	fmt.Println("✅ Step 1: Funded Issuer_Grantor_Applicant account with sufficient funds")

	// Step 2: Load the Trust Registry data created in Journey 1
	journey1Result := lib.LoadJourneyResult("journey1")
	fmt.Println("✅ Step 2: Loaded Trust Registry data created by Trust_Registry_Controller")
	fmt.Printf("    - Trust Registry ID: %s\n", journey1Result.TrustRegistryID)
	fmt.Printf("    - Schema ID: %s\n", journey1Result.SchemaID)
	fmt.Printf("    - Root Permission ID: %s\n", journey1Result.RootPermissionID)

	// Verify the Trust Registry and Schema from Journey 1 exist
	trID, _ := strconv.ParseUint(journey1Result.TrustRegistryID, 10, 64)
	lib.VerifyTrustRegistry(client, ctx, trID, journey1Result.DID)

	csID, _ := strconv.ParseUint(journey1Result.SchemaID, 10, 64)
	lib.VerifyCredentialSchema(client, ctx, csID, trID)

	rootPermID, _ := strconv.ParseUint(journey1Result.RootPermissionID, 10, 64)
	lib.VerifyPermission(client, ctx, rootPermID, csID, "ECOSYSTEM")

	// Step 3: Issuer_Grantor_Applicant starts validation process
	issuerGrantorDID := lib.GenerateUniqueDID(client, ctx)
	fmt.Printf("    - Generated Issuer Grantor DID: %s\n", issuerGrantorDID)

	fmt.Println("Issuer_Grantor_Applicant starting Permission Validation Process...")
	countryCode := "US" // Example country code

	startVPMsg := permtypes.MsgStartPermissionVP{
		Type:            permtypes.PermissionType_ISSUER_GRANTOR,
		ValidatorPermId: rootPermID,
		Country:         countryCode,
		Did:             issuerGrantorDID,
	}

	permissionID := lib.StartValidationProcess(client, ctx, issuerGrantorAccount, startVPMsg)
	fmt.Printf("✅ Step 3: Issuer_Grantor_Applicant started validation process with permission ID: %s\n", permissionID)

	// Verify validation process was started correctly
	permID, _ := strconv.ParseUint(permissionID, 10, 64)
	lib.VerifyPendingValidation(client, ctx, permID, issuerGrantorDID, "ISSUER_GRANTOR")

	// Step 4: Issuer_Grantor_Applicant connects to Trust_Registry_Controller's validation service
	fmt.Println("✅ Step 4: Issuer_Grantor_Applicant connects to Trust_Registry_Controller's validation service (simulated)")
	fmt.Println("    - Validation occurs off-chain between applicant and validator")

	// Step 5: Trust_Registry_Controller validates the applicant
	fmt.Println("Trust_Registry_Controller validates the Issuer_Grantor_Applicant...")

	// Set fees
	validationFees := uint64(7)
	issuanceFees := uint64(5)
	verificationFees := uint64(2)

	trustRegistryControllerAccount := lib.GetAccount(client, lib.TRUST_REGISTRY_CONTROLLER_NAME)
	lib.ValidatePermission(
		client,
		ctx,
		trustRegistryControllerAccount,
		permID,
		validationFees,
		issuanceFees,
		verificationFees,
		countryCode,
	)

	fmt.Printf("✅ Step 5: Trust_Registry_Controller validated Issuer_Grantor_Applicant with permission ID: %s\n", permissionID)
	fmt.Printf("    - Validation fees: %d\n", validationFees)
	fmt.Printf("    - Issuance fees: %d\n", issuanceFees)
	fmt.Printf("    - Verification fees: %d\n", verificationFees)

	// Verify permission was validated correctly
	lib.VerifyValidatedPermission(
		client,
		ctx,
		permID,
		issuerGrantorDID,
		"ISSUER_GRANTOR",
		validationFees,
		issuanceFees,
		verificationFees,
	)

	fmt.Println("Journey 2 completed successfully! ✨")

	// Store the result for future reference
	finalResult := lib.JourneyResult{
		IssuerGrantorDID:    issuerGrantorDID,
		IssuerGrantorPermID: permissionID,
		TrustRegistryID:     journey1Result.TrustRegistryID,
		SchemaID:            journey1Result.SchemaID,
		RootPermissionID:    journey1Result.RootPermissionID,
	}

	// Save result for other journeys to use
	lib.SaveJourneyResult("journey2", finalResult)

	return nil
}
