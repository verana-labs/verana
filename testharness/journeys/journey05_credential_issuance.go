package journeys

import (
	"context"
	"fmt"

	"cosmossdk.io/math"
	"github.com/google/uuid"

	//"github.com/google/uuid"
	"strconv"

	"github.com/ignite/cli/v28/ignite/pkg/cosmosclient"

	permtypes "github.com/verana-labs/verana/x/perm/types"

	"github.com/verana-labs/verana/testharness/lib"
)

// RunCredentialIssuanceJourney implements Journey 5: Credential Issuance Journey
func RunCredentialIssuanceJourney(ctx context.Context, client cosmosclient.Client) error {
	fmt.Println("Starting Journey 5: Credential Issuance Journey")

	// Step 1: Get Credential_Holder account and fund it
	fmt.Println("Sending funds from cooluser to Credential_Holder...")
	holderAccount := lib.GetAccount(client, lib.CREDENTIAL_HOLDER_NAME)
	lib.SendFunds(client, ctx, lib.COOLUSER_ADDRESS, lib.CREDENTIAL_HOLDER_ADDRESS, math.NewInt(10000000)) //10 VNA
	lib.SendFunds(client, ctx, lib.COOLUSER_ADDRESS, lib.ISSUER_APPLICANT_ADDRESS, math.NewInt(20000000))  // 20 VNA for txn fees
	fmt.Println("✅ Step 1: Funded Credential_Holder account with sufficient funds")

	// Step 2 & 3: Load Trust Registry, Credential Schema, and Issuer data
	journey1Result := lib.LoadJourneyResult("journey1")
	journey3Result := lib.LoadJourneyResult("journey3")

	fmt.Println("✅ Step 2 & 3: Loaded data from previous journeys")
	fmt.Printf("    - Trust Registry ID: %s\n", journey1Result.TrustRegistryID)
	fmt.Printf("    - Schema ID: %s\n", journey1Result.SchemaID)
	fmt.Printf("    - Issuer DID: %s\n", journey3Result.IssuerDID)
	fmt.Printf("    - Issuer Permission ID: %s\n", journey3Result.IssuerPermID)

	// Verify the Trust Registry and Schema from Journey 1 exist
	trID, _ := strconv.ParseUint(journey1Result.TrustRegistryID, 10, 64)
	lib.VerifyTrustRegistry(client, ctx, trID, journey1Result.DID)

	csID, _ := strconv.ParseUint(journey1Result.SchemaID, 10, 64)
	lib.VerifyCredentialSchema(client, ctx, csID, trID)

	// Verify the Issuer permission from Journey 3 exists
	issuerPermID, _ := strconv.ParseUint(journey3Result.IssuerPermID, 10, 64)
	lib.VerifyValidatedPermission(
		client,
		ctx,
		issuerPermID,
		journey3Result.IssuerDID,
		"ISSUER",
		7,
		5,
		2,
	)

	// Step 4: Simulate Credential_Holder having a compatible wallet
	fmt.Println("✅ Step 4: Credential_Holder has a compatible wallet application (simulated)")

	// Step 5: Credential_Holder requests credential from Issuer_Applicant
	fmt.Println("✅ Step 5: Credential_Holder requests credential from Issuer_Applicant (simulated)")
	fmt.Println("    - Request occurs off-chain between holder and issuer")

	// Step 6: Create a HOLDER permission through validation process using the issuer as validator
	fmt.Println("Creating HOLDER permission for Credential_Holder's wallet...")
	holderWalletDID := lib.GenerateUniqueDID(client, ctx)
	fmt.Printf("    - Generated wallet DID: %s\n", holderWalletDID)

	// Start validation process for HOLDER type
	countryCode := "US"
	startVPMsg := permtypes.MsgStartPermissionVP{
		Type:            permtypes.PermissionType_HOLDER,
		ValidatorPermId: issuerPermID,
		Country:         countryCode,
		Did:             holderWalletDID,
	}
	fmt.Println("..........issuerPermID......", issuerPermID)

	holderPermID := lib.StartValidationProcess(client, ctx, holderAccount, startVPMsg)
	fmt.Printf("    - Started HOLDER validation process with ID: %s\n", holderPermID)

	// Verify validation process was started correctly
	holderPermIDUint, _ := strconv.ParseUint(holderPermID, 10, 64)
	lib.VerifyPendingValidation(client, ctx, holderPermIDUint, holderWalletDID, "HOLDER")

	// Validate the HOLDER permission (by the issuer)
	issuerAccount := lib.GetAccount(client, lib.ISSUER_APPLICANT_NAME)
	lib.ValidatePermission(
		client,
		ctx,
		issuerAccount,
		holderPermIDUint,
		0,
		0,
		0,
		countryCode,
	)

	// Verify HOLDER permission was validated correctly
	lib.VerifyValidatedPermission(
		client,
		ctx,
		holderPermIDUint,
		holderWalletDID,
		"HOLDER",
		0, // No validation fees for holder
		0, // No issuance fees for holder
		0, // No verification fees for holder
	)

	fmt.Printf("✅ Step 6: Created and validated HOLDER permission ID: %s\n", holderPermID)

	// Generate a session UUID
	sessionID := uuid.New().String()
	fmt.Printf("    - Generated session ID: %s\n", sessionID)

	// Step 7: Issuer_Applicant creates permission session
	fmt.Println("Issuer_Applicant creating permission session...")

	// Create permission session
	lib.CreatePermissionSession(
		client,
		ctx,
		issuerAccount,
		sessionID,
		issuerPermID, // Issuer permission ID
		0,            // No verifier for credential issuance
		issuerPermID, // Holder permission as agent
		issuerPermID, // Holder permission as wallet agent
	)

	// Verify the permission session was created correctly
	issuerAddr, err := issuerAccount.Address(lib.GetAddressPrefix())
	if err != nil {
		return fmt.Errorf("failed to get issuer address: %v", err)
	}

	verificationResult := lib.VerifyPermissionSession(
		client,
		ctx,
		sessionID,
		issuerAddr,   // Expected controller (issuer address)
		issuerPermID, // Expected agent permission ID
		issuerPermID, // Expected issuer permission ID
		0,            // No verifier for credential issuance
	)

	if !verificationResult {
		return fmt.Errorf("permission session verification failed")
	}

	fmt.Printf("✅ Step 7: Issuer_Applicant created permission session with ID: %s\n", sessionID)

	fmt.Println("Journey 5 completed successfully! ✨")

	// Store the result for future reference
	result := lib.JourneyResult{
		TrustRegistryID:    journey1Result.TrustRegistryID,
		SchemaID:           journey1Result.SchemaID,
		IssuerDID:          journey3Result.IssuerDID,
		IssuerPermID:       journey3Result.IssuerPermID,
		HolderWalletDID:    holderWalletDID,
		HolderWalletPermID: holderPermID,
		//CredentialSessionID: sessionID,
	}

	// Save result for other journeys to use
	lib.SaveJourneyResult("journey5", result)

	return nil
}
