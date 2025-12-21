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

// RunTrustRegistryJourney implements Journey 1: Trust Registry Controller Journey
func RunTrustRegistryJourney(ctx context.Context, client cosmosclient.Client) error {
	fmt.Println("Starting Journey 1: Trust Registry Controller Journey")

	// Step 1: Fund account
	fmt.Println("Sending funds from cooluser to Trust_Registry_Controller...")
	trustRegistryControllerAccount := lib.GetAccount(client, lib.TRUST_REGISTRY_CONTROLLER_NAME)
	lib.SendFunds(client, ctx, lib.COOLUSER_ADDRESS, lib.TRUST_REGISTRY_CONTROLLER_ADDRESS, math.NewInt(30000000)) // 30 VNA
	fmt.Println("✅ Step 1: Funded Trust_Registry_Controller account with sufficient funds")

	// Step 2: Generate a unique DID for the trust registry
	did := lib.GenerateUniqueDID(client, ctx)
	fmt.Println("✅ Step 2: Generated unique DID for Trust Registry:", did)

	// Step 3: Trust_Registry_Controller creates a new Trust Registry
	fmt.Println("Trust_Registry_Controller creating Trust Registry...")
	trustRegistryID := lib.CreateNewTrustRegistry(client, ctx, trustRegistryControllerAccount, did)
	fmt.Printf("✅ Step 3: Trust Registry created with ID: %s\n", trustRegistryID)

	// Verify Trust Registry creation
	trID, _ := strconv.ParseUint(trustRegistryID, 10, 64)
	verified := lib.VerifyTrustRegistry(client, ctx, trID, did)
	if !verified {
		fmt.Println("  Trust Registry verification failed")
	}

	// Step 4: Trust_Registry_Controller creates a Credential Schema
	fmt.Println("Trust_Registry_Controller creating Credential Schema...")
	schemaData := lib.GenerateSimpleSchema(trustRegistryID)
	schemaID := lib.CreateSimpleCredentialSchema(
		client, ctx, trustRegistryControllerAccount, trustRegistryID, schemaData,
		cschema.CredentialSchemaPermManagementMode_GRANTOR_VALIDATION,
		cschema.CredentialSchemaPermManagementMode_OPEN,
	)
	fmt.Printf("✅ Step 4: Credential Schema created with ID: %s\n", schemaID)

	// Verify Credential Schema creation
	csID, _ := strconv.ParseUint(schemaID, 10, 64)
	trID, _ = strconv.ParseUint(trustRegistryID, 10, 64)
	verified = lib.VerifyCredentialSchema(client, ctx, csID, trID)
	if !verified {
		fmt.Println("  Credential Schema verification failed")
	}

	// Step 5: Trust_Registry_Controller creates Root Permission
	fmt.Println("Trust_Registry_Controller creating Root Permission...")
	effectiveFrom := time.Now().Add(time.Second * 10)
	effectiveUntil := effectiveFrom.Add(time.Hour * 24 * 360)
	validationFees := uint64(5)
	verificationFees := uint64(5)
	issuanceFees := uint64(5)

	rootPermissionID := lib.CreateRootPermissionWithDates(
		client, ctx, trustRegistryControllerAccount, schemaID, did,
		effectiveFrom, effectiveUntil, validationFees, verificationFees, issuanceFees,
	)
	fmt.Printf("✅ Step 5: Root Permission created with ID: %s\n", rootPermissionID)

	// Verify Permission creation
	rpID, _ := strconv.ParseUint(rootPermissionID, 10, 64)
	csID, _ = strconv.ParseUint(schemaID, 10, 64)
	verified = lib.VerifyPermission(client, ctx, rpID, csID, "ECOSYSTEM")
	if !verified {
		fmt.Println("  Permission verification failed")
	}

	// Step 6: Trust_Registry_Controller adds DID to directory
	fmt.Println("Trust_Registry_Controller adding DID to directory...")
	lib.RegisterDID(client, ctx, trustRegistryControllerAccount, did, 1)
	fmt.Printf("✅ Step 6: DID %s registered in directory\n", did)

	// Verify DID registration
	verified = lib.VerifyDID(client, ctx, did, lib.TRUST_REGISTRY_CONTROLLER_ADDRESS)
	if !verified {
		fmt.Println("  DID verification failed")
	}

	fmt.Println("Journey 1 completed successfully! ✨")

	// Store the result for future reference
	result := lib.JourneyResult{
		TrustRegistryID:  trustRegistryID,
		SchemaID:         schemaID,
		RootPermissionID: rootPermissionID,
		DID:              did,
	}
	lib.SaveJourneyResult("journey1", result)

	return nil
}
