package journeys

import (
	"context"
	"fmt"
	"strconv"

	"github.com/google/uuid"

	"github.com/ignite/cli/v28/ignite/pkg/cosmosclient"

	"github.com/verana-labs/verana/testharness/lib"
)

// RunCredentialVerificationJourney implements Journey 6: Credential Verification Journey
func RunCredentialVerificationJourney(ctx context.Context, client cosmosclient.Client) error {
	fmt.Println("Starting Journey 6: Credential Verification Journey")

	// Step 1 & 2: Load data from previous journeys
	journey1Result := lib.LoadJourneyResult("journey1")
	journey3Result := lib.LoadJourneyResult("journey3")
	journey4Result := lib.LoadJourneyResult("journey4")
	journey5Result := lib.LoadJourneyResult("journey5")

	fmt.Println("✅ Step 1 & 2: Loaded data from previous journeys")
	fmt.Printf("    - Trust Registry ID: %s\n", journey1Result.TrustRegistryID)
	fmt.Printf("    - Schema ID: %s\n", journey1Result.SchemaID)
	fmt.Printf("    - Issuer DID: %s\n", journey3Result.IssuerDID)
	fmt.Printf("    - Issuer Permission ID: %s\n", journey3Result.IssuerPermID)
	fmt.Printf("    - Verifier DID: %s\n", journey4Result.VerifierDID)
	fmt.Printf("    - Verifier Permission ID: %s\n", journey4Result.VerifierPermID)
	fmt.Printf("    - Holder Wallet DID: %s\n", journey5Result.HolderWalletDID)
	fmt.Printf("    - Holder Wallet Permission ID: %s\n", journey5Result.HolderWalletPermID)

	// Verify the essential components from previous journeys
	trID, _ := strconv.ParseUint(journey1Result.TrustRegistryID, 10, 64)
	lib.VerifyTrustRegistry(client, ctx, trID, journey1Result.DID)

	csID, _ := strconv.ParseUint(journey1Result.SchemaID, 10, 64)
	lib.VerifyCredentialSchema(client, ctx, csID, trID)

	// Verify the Issuer permission
	issuerPermID, _ := strconv.ParseUint(journey3Result.IssuerPermID, 10, 64)
	lib.VerifyValidatedPermission(
		client,
		ctx,
		issuerPermID,
		journey3Result.IssuerDID,
		"ISSUER",
		7, // Expected validation fees from Journey 3
		5, // Expected issuance fees from Journey 3
		2, // Expected verification fees from Journey 3
	)

	// Verify the Verifier permission
	verifierPermID, _ := strconv.ParseUint(journey4Result.VerifierPermID, 10, 64)
	lib.VerifyValidatedPermission(
		client,
		ctx,
		verifierPermID,
		journey4Result.VerifierDID,
		"VERIFIER",
		80, // Expected validation fees from Journey 4
		30, // Expected issuance fees from Journey 4
		15, // Expected verification fees from Journey 4
	)

	// Verify the Holder permission
	holderPermID, _ := strconv.ParseUint(journey5Result.HolderWalletPermID, 10, 64)
	lib.VerifyValidatedPermission(
		client,
		ctx,
		holderPermID,
		journey5Result.HolderWalletDID,
		"HOLDER",
		0,
		0,
		0,
	)

	// Step 3 & 4: Confirm Credential_Holder already has a credential
	fmt.Println("✅ Step 3 & 4: Credential_Holder already has a credential in wallet (from Journey 5)")

	// Step 5: Credential_Holder presents credential to Verifier_Applicant
	fmt.Println("✅ Step 5: Credential_Holder presents credential to Verifier_Applicant (simulated)")
	fmt.Println("    - Presentation request and response occur off-chain between holder and verifier")

	// Generate a session UUID for verification
	verificationSessionID := uuid.New().String()
	fmt.Printf("    - Generated verification session ID: %s\n", verificationSessionID)

	fmt.Println("Journey 6 completed successfully! ✨")

	// Store the result for future reference
	result := lib.JourneyResult{
		TrustRegistryID:       journey1Result.TrustRegistryID,
		SchemaID:              journey1Result.SchemaID,
		IssuerDID:             journey3Result.IssuerDID,
		IssuerPermID:          journey3Result.IssuerPermID,
		VerifierDID:           journey4Result.VerifierDID,
		VerifierPermID:        journey4Result.VerifierPermID,
		HolderWalletDID:       journey5Result.HolderWalletDID,
		HolderWalletPermID:    journey5Result.HolderWalletPermID,
		CredentialSessionID:   journey5Result.CredentialSessionID,
		VerificationSessionID: verificationSessionID,
	}

	// Save result for other journeys to use
	lib.SaveJourneyResult("journey6", result)

	return nil
}
