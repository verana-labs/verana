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

// RunVerifierValidationJourney implements Journey 4: Verifier Validation Journey (via Trust Registry)
func RunVerifierValidationJourney(ctx context.Context, client cosmosclient.Client) error {
	fmt.Println("Starting Journey 4: Verifier Validation Journey (via Trust Registry)")

	// Step 1: Get account and fund it
	fmt.Println("Sending funds from cooluser to Verifier_Applicant...")
	verifierApplicantAccount := lib.GetAccount(client, lib.VERIFIER_APPLICANT_NAME)
	lib.SendFunds(client, ctx, lib.COOLUSER_ADDRESS, lib.VERIFIER_APPLICANT_ADDRESS, math.NewInt(10000000)) //10 VNA
	fmt.Println("✅ Step 1: Funded Verifier_Applicant account with sufficient funds")

	// Step 2 & 3: Load Trust Registry, Credential Schema, and Root Permission data
	journey1Result := lib.LoadJourneyResult("journey1")

	fmt.Println("✅ Step 2 & 3: Trust Registry and Credential Schema exist with Root Permission")
	fmt.Printf("    - Trust Registry ID: %s (created by Trust_Registry_Controller)\n", journey1Result.TrustRegistryID)
	fmt.Printf("    - Schema ID: %s (created by Trust_Registry_Controller)\n", journey1Result.SchemaID)
	fmt.Printf("    - Root Permission ID: %s\n", journey1Result.RootPermissionID)

	// Verify the Trust Registry and Schema from Journey 1 exist
	trID, _ := strconv.ParseUint(journey1Result.TrustRegistryID, 10, 64)
	lib.VerifyTrustRegistry(client, ctx, trID, journey1Result.DID)

	csID, _ := strconv.ParseUint(journey1Result.SchemaID, 10, 64)
	lib.VerifyCredentialSchema(client, ctx, csID, trID)

	rootPermID, _ := strconv.ParseUint(journey1Result.RootPermissionID, 10, 64)
	lib.VerifyPermission(client, ctx, rootPermID, csID, "ECOSYSTEM")

	// Step 4: Verifier_Applicant starts validation process
	verifierDID := lib.GenerateUniqueDID(client, ctx)
	fmt.Printf("    - Generated Verifier DID: %s\n", verifierDID)

	fmt.Println("Verifier_Applicant starting Permission Validation Process...")
	countryCode := "US" // Example country code

	startVPMsg := permtypes.MsgStartPermissionVP{
		Type:            permtypes.PermissionType_VERIFIER,
		ValidatorPermId: rootPermID,
		Country:         countryCode,
		Did:             verifierDID,
	}

	permissionID := lib.StartValidationProcess(client, ctx, verifierApplicantAccount, startVPMsg)
	fmt.Printf("✅ Step 4: Verifier_Applicant started validation process with permission ID: %s\n", permissionID)

	// Verify validation process was started correctly
	permID, _ := strconv.ParseUint(permissionID, 10, 64)
	lib.VerifyPendingValidation(client, ctx, permID, verifierDID, "VERIFIER")

	// Step 5: Verifier_Applicant connects to Trust_Registry_Controller's validation service
	fmt.Println("✅ Step 5: Verifier_Applicant connects to Trust_Registry_Controller's validation service (simulated)")
	fmt.Println("    - Validation occurs off-chain between applicant and validator")

	// Step 6: Trust_Registry_Controller validates the applicant
	fmt.Println("Trust_Registry_Controller validates the Verifier_Applicant...")

	// Set fees
	validationFees := uint64(80)
	issuanceFees := uint64(30)
	verificationFees := uint64(15)

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

	fmt.Printf("✅ Step 6: Trust_Registry_Controller validated Verifier_Applicant with permission ID: %s\n", permissionID)
	fmt.Printf("    - Validation fees: %d\n", validationFees)
	fmt.Printf("    - Issuance fees: %d\n", issuanceFees)
	fmt.Printf("    - Verification fees: %d\n", verificationFees)

	// Verify permission was validated correctly
	lib.VerifyValidatedPermission(
		client,
		ctx,
		permID,
		verifierDID,
		"VERIFIER",
		validationFees,
		issuanceFees,
		verificationFees,
	)

	fmt.Println("Journey 4 completed successfully! ✨")

	// Store the result for future reference
	result := lib.JourneyResult{
		TrustRegistryID:  journey1Result.TrustRegistryID,
		SchemaID:         journey1Result.SchemaID,
		RootPermissionID: journey1Result.RootPermissionID,
		VerifierDID:      verifierDID,
		VerifierPermID:   permissionID,
	}

	// Save result for other journeys to use
	lib.SaveJourneyResult("journey4", result)

	return nil
}
