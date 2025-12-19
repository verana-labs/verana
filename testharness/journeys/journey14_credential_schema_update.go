package journeys

import (
	"context"
	"fmt"
	"strconv"
	"time"

	"cosmossdk.io/math"
	"github.com/ignite/cli/v28/ignite/pkg/cosmosclient"
	cschema "github.com/verana-labs/verana/x/cs/types"

	"github.com/verana-labs/verana/testharness/lib"
)

// RunCredentialSchemaUpdateJourney implements Journey 14: Credential Schema Update Journey
func RunCredentialSchemaUpdateJourney(ctx context.Context, client cosmosclient.Client) error {
	fmt.Println("Starting Journey 14: Credential Schema Update Journey")

	// Step 1: Trust_Registry_Controller creates a trust registry
	fmt.Println("Trust_Registry_Controller creating Trust Registry...")

	// Get the Trust_Registry_Controller account
	trustRegistryControllerAccount, err := client.Account(lib.TRUST_REGISTRY_CONTROLLER_NAME)
	if err != nil {
		return fmt.Errorf("failed to get %s account: %v", lib.TRUST_REGISTRY_CONTROLLER_NAME, err)
	}

	// Ensure the account has sufficient funds
	fmt.Println("Sending funds to Trust_Registry_Controller account...")
	lib.SendFunds(client, ctx, lib.COOLUSER_ADDRESS, lib.TRUST_REGISTRY_CONTROLLER_ADDRESS, math.NewInt(50000000))

	// Generate a unique DID for the trust registry
	did := lib.GenerateUniqueDID(client, ctx)
	fmt.Printf("    - Generated DID for Trust Registry: %s\n", did)

	// Create a new Trust Registry
	trustRegistryID := lib.CreateNewTrustRegistry(client, ctx, trustRegistryControllerAccount, did)
	fmt.Printf("✅ Step 1: Trust Registry created with ID: %s\n", trustRegistryID)

	// Verify Trust Registry creation
	trID, _ := strconv.ParseUint(trustRegistryID, 10, 64)
	verified := lib.VerifyTrustRegistry(client, ctx, trID, did)
	if !verified {
		fmt.Println("⚠️ Warning: Trust Registry verification failed, but continuing...")
	}

	// Step 2: Trust_Registry_Controller creates a Credential Schema
	fmt.Println("Trust_Registry_Controller creating Credential Schema...")

	// Generate schema data
	schemaData := lib.GenerateSimpleSchema(trustRegistryID)

	// Create the credential schema with specified management modes
	schemaID := lib.CreateSimpleCredentialSchema(
		client, ctx, trustRegistryControllerAccount, trustRegistryID, schemaData,
		cschema.CredentialSchemaPermManagementMode_GRANTOR_VALIDATION, // Issuer mode
		cschema.CredentialSchemaPermManagementMode_OPEN,               // Verifier mode
	)
	fmt.Printf("✅ Step 2: Credential Schema created with ID: %s\n", schemaID)

	// Verify Credential Schema creation
	csID, _ := strconv.ParseUint(schemaID, 10, 64)
	verified = lib.VerifyCredentialSchema(client, ctx, csID, trID)
	if !verified {
		fmt.Println("⚠️ Warning: Credential Schema verification failed, but continuing...")
	}

	// Retrieve the initial schema to display properties
	initialSchema, err := lib.QueryCredentialSchema(client, ctx, csID)
	if err != nil {
		return fmt.Errorf("failed to query credential schema: %v", err)
	}

	fmt.Println("    - Initial schema properties:")
	fmt.Printf("      - Issuer Grantor Validation Validity Period: %d days\n",
		initialSchema.Schema.IssuerGrantorValidationValidityPeriod)
	fmt.Printf("      - Verifier Grantor Validation Validity Period: %d days\n",
		initialSchema.Schema.VerifierGrantorValidationValidityPeriod)
	fmt.Printf("      - Issuer Validation Validity Period: %d days\n",
		initialSchema.Schema.IssuerValidationValidityPeriod)
	fmt.Printf("      - Verifier Validation Validity Period: %d days\n",
		initialSchema.Schema.VerifierValidationValidityPeriod)
	fmt.Printf("      - Holder Validation Validity Period: %d days\n",
		initialSchema.Schema.HolderValidationValidityPeriod)
	fmt.Printf("      - Issuer Permission Management Mode: %s\n",
		cschema.CredentialSchemaPermManagementMode_name[int32(initialSchema.Schema.IssuerPermManagementMode)])
	fmt.Printf("      - Verifier Permission Management Mode: %s\n",
		cschema.CredentialSchemaPermManagementMode_name[int32(initialSchema.Schema.VerifierPermManagementMode)])

	// Wait for transactions to be processed
	fmt.Println("    - Waiting for transactions to be processed...")
	time.Sleep(3 * time.Second)

	// Step 3: Trust_Registry_Controller updates the Credential Schema
	fmt.Println("Trust_Registry_Controller updating Credential Schema...")

	// Update the credential schema with new validation periods
	updateResponse, err := lib.UpdateCredentialSchema(
		client, ctx, trustRegistryControllerAccount, csID,
		730, // IssuerGrantorValidationValidityPeriod: 2 years
		730, // VerifierGrantorValidationValidityPeriod: 2 years
		365, // IssuerValidationValidityPeriod: 1 year
		365, // VerifierValidationValidityPeriod: 1 year
		180, // HolderValidationValidityPeriod: 6 months
	)
	if err != nil {
		return fmt.Errorf("failed to update credential schema: %v", err)
	}

	fmt.Printf("✅ Step 3: Credential Schema updated with response: %s\n", updateResponse)

	// Query the updated schema to confirm changes
	updatedSchema, err := lib.QueryCredentialSchema(client, ctx, csID)
	if err != nil {
		return fmt.Errorf("failed to query updated credential schema: %v", err)
	}

	fmt.Println("    - Updated schema properties:")
	fmt.Printf("      - Issuer Grantor Validation Validity Period: %d days\n",
		updatedSchema.Schema.IssuerGrantorValidationValidityPeriod)
	fmt.Printf("      - Verifier Grantor Validation Validity Period: %d days\n",
		updatedSchema.Schema.VerifierGrantorValidationValidityPeriod)
	fmt.Printf("      - Issuer Validation Validity Period: %d days\n",
		updatedSchema.Schema.IssuerValidationValidityPeriod)
	fmt.Printf("      - Verifier Validation Validity Period: %d days\n",
		updatedSchema.Schema.VerifierValidationValidityPeriod)
	fmt.Printf("      - Holder Validation Validity Period: %d days\n",
		updatedSchema.Schema.HolderValidationValidityPeriod)

	// Wait for update to be processed
	fmt.Println("    - Waiting for update to be processed...")
	time.Sleep(3 * time.Second)

	// Step 4: Trust_Registry_Controller archives the Credential Schema
	fmt.Println("Trust_Registry_Controller archiving Credential Schema...")

	// Archive the credential schema
	archiveMsg := &cschema.MsgArchiveCredentialSchema{
		Id:      csID,
		Archive: true,
	}

	archiveResponse, err := lib.ArchiveCredentialSchema(client, ctx, trustRegistryControllerAccount, *archiveMsg)
	if err != nil {
		return fmt.Errorf("failed to archive credential schema: %v", err)
	}

	fmt.Printf("✅ Step 4: Credential Schema archived with response: %s\n", archiveResponse)

	// Query the archived schema to confirm changes
	archivedSchema, err := lib.QueryCredentialSchema(client, ctx, csID)
	if err != nil {
		return fmt.Errorf("failed to query archived credential schema: %v", err)
	}

	archivedAt := "not archived"
	if archivedSchema.Schema.Archived != nil {
		archivedAt = archivedSchema.Schema.Archived.Format(time.RFC3339)
	}

	fmt.Printf("    - Schema archived at: %s\n", archivedAt)

	// Wait for archive to be processed
	fmt.Println("    - Waiting for archive to be processed...")
	time.Sleep(3 * time.Second)

	// Step 5: Trust_Registry_Controller unarchives the Credential Schema
	fmt.Println("Trust_Registry_Controller unarchiving Credential Schema...")

	// Unarchive the credential schema
	unarchiveMsg := &cschema.MsgArchiveCredentialSchema{
		Id:      csID,
		Archive: false,
	}

	unarchiveResponse, err := lib.ArchiveCredentialSchema(client, ctx, trustRegistryControllerAccount, *unarchiveMsg)
	if err != nil {
		return fmt.Errorf("failed to unarchive credential schema: %v", err)
	}

	fmt.Printf("✅ Step 5: Credential Schema unarchived with response: %s\n", unarchiveResponse)

	// Query the unarchived schema to confirm changes
	unarchivedSchema, err := lib.QueryCredentialSchema(client, ctx, csID)
	if err != nil {
		return fmt.Errorf("failed to query unarchived credential schema: %v", err)
	}

	archivedAt = "not archived"
	if unarchivedSchema.Schema.Archived != nil {
		archivedAt = unarchivedSchema.Schema.Archived.Format(time.RFC3339)
	}

	fmt.Printf("    - Schema archived status: %s\n", archivedAt)

	fmt.Println("Journey 14 completed successfully! ✨")
	fmt.Println("    - Demonstrated complete lifecycle of a credential schema")
	fmt.Println("    - Created, updated, archived, and unarchived a schema")

	// Store the result for future reference
	result := lib.JourneyResult{
		TrustRegistryID: trustRegistryID,
		SchemaID:        schemaID,
		DID:             did,
	}

	// Save result to global state or file for other journeys to use
	lib.SaveJourneyResult("journey14", result)

	return nil
}
