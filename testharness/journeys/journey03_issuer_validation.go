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

// RunIssuerValidationJourney implements Journey 3: Issuer Validation Journey (via Issuer Grantor)
func RunIssuerValidationJourney(ctx context.Context, client cosmosclient.Client) error {
	fmt.Println("Starting Journey 3: Issuer Validation Journey (via Issuer Grantor)")

	// Step 1: Get Issuer_Applicant account and fund it
	fmt.Println("Sending funds from cooluser to Issuer_Applicant...")
	issuerApplicantAccount := lib.GetAccount(client, lib.ISSUER_APPLICANT_NAME)
	lib.SendFunds(client, ctx, lib.COOLUSER_ADDRESS, lib.ISSUER_APPLICANT_ADDRESS, math.NewInt(10000000)) //10 VNA
	fmt.Println("✅ Step 1: Funded Issuer_Applicant account with sufficient funds")

	// Step 2 & 3: Load Trust Registry, Credential Schema, and Issuer Grantor data
	journey1Result := lib.LoadJourneyResult("journey1")
	journey2Result := lib.LoadJourneyResult("journey2")

	fmt.Println("✅ Step 2 & 3: Loaded data from previous journeys")
	fmt.Printf("    - Trust Registry ID: %s (created by Trust_Registry_Controller)\n", journey1Result.TrustRegistryID)
	fmt.Printf("    - Schema ID: %s (created by Trust_Registry_Controller)\n", journey1Result.SchemaID)
	fmt.Printf("    - Issuer Grantor Permission ID: %s\n", journey2Result.IssuerGrantorPermID)
	fmt.Printf("    - Issuer Grantor DID: %s\n", journey2Result.IssuerGrantorDID)

	// Verify data from previous journeys
	trID, _ := strconv.ParseUint(journey1Result.TrustRegistryID, 10, 64)
	lib.VerifyTrustRegistry(client, ctx, trID, journey1Result.DID)

	csID, _ := strconv.ParseUint(journey1Result.SchemaID, 10, 64)
	lib.VerifyCredentialSchema(client, ctx, csID, trID)

	issuerGrantorPermID, _ := strconv.ParseUint(journey2Result.IssuerGrantorPermID, 10, 64)
	lib.VerifyValidatedPermission(
		client,
		ctx,
		issuerGrantorPermID,
		journey2Result.IssuerGrantorDID,
		"ISSUER_GRANTOR",
		7, // Expected validation fees from Journey 2
		5, // Expected issuance fees from Journey 2
		2, // Expected verification fees from Journey 2
	)

	// Step 4: Issuer_Applicant starts validation process
	issuerDID := lib.GenerateUniqueDID(client, ctx)
	fmt.Printf("    - Generated Issuer DID: %s\n", issuerDID)

	fmt.Println("Issuer_Applicant starting Permission Validation Process...")
	countryCode := "US" // Example country code

	startVPMsg := permtypes.MsgStartPermissionVP{
		Type:            permtypes.PermissionType_ISSUER,
		ValidatorPermId: issuerGrantorPermID,
		Country:         countryCode,
		Did:             issuerDID,
	}

	permissionID := lib.StartValidationProcess(client, ctx, issuerApplicantAccount, startVPMsg)
	fmt.Printf("✅ Step 4: Issuer_Applicant started validation process with permission ID: %s\n", permissionID)

	// Verify validation process was started correctly
	permID, _ := strconv.ParseUint(permissionID, 10, 64)
	lib.VerifyPendingValidation(client, ctx, permID, issuerDID, "ISSUER")

	// Step 5: Issuer_Applicant connects to Issuer_Grantor_Applicant's validation service
	fmt.Println("✅ Step 5: Issuer_Applicant connects to Issuer_Grantor_Applicant's validation service (simulated)")
	fmt.Println("    - Validation occurs off-chain between applicant and validator")

	// Step 6: Issuer_Grantor_Applicant validates the applicant
	fmt.Println("Issuer_Grantor_Applicant validates the Issuer_Applicant...")

	// Set fees
	validationFees := uint64(7)
	issuanceFees := uint64(5)
	verificationFees := uint64(2)

	issuerGrantorAccount := lib.GetAccount(client, lib.ISSUER_GRANTOR_APPLICANT_NAME)
	lib.ValidatePermission(
		client,
		ctx,
		issuerGrantorAccount,
		permID,
		validationFees,
		issuanceFees,
		verificationFees,
		countryCode,
	)

	fmt.Printf("✅ Step 6: Issuer_Grantor_Applicant validated Issuer_Applicant with permission ID: %s\n", permissionID)
	fmt.Printf("    - Validation fees: %d\n", validationFees)
	fmt.Printf("    - Issuance fees: %d\n", issuanceFees)
	fmt.Printf("    - Verification fees: %d\n", verificationFees)

	// Verify permission was validated correctly
	lib.VerifyValidatedPermission(
		client,
		ctx,
		permID,
		issuerDID,
		"ISSUER",
		validationFees,
		issuanceFees,
		verificationFees,
	)

	fmt.Println("Journey 3 completed successfully! ✨")

	// Store the result for future reference
	result := lib.JourneyResult{
		TrustRegistryID:     journey1Result.TrustRegistryID,
		SchemaID:            journey1Result.SchemaID,
		RootPermissionID:    journey1Result.RootPermissionID,
		IssuerGrantorDID:    journey2Result.IssuerGrantorDID,
		IssuerGrantorPermID: journey2Result.IssuerGrantorPermID,
		IssuerDID:           issuerDID,
		IssuerPermID:        permissionID,
	}

	// Save result for other journeys to use
	lib.SaveJourneyResult("journey3", result)

	return nil
}
