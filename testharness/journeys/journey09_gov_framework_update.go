package journeys

import (
	"context"
	"fmt"
	"strconv"

	"github.com/ignite/cli/v28/ignite/pkg/cosmosclient"
	trtypes "github.com/verana-labs/verana/x/tr/types"

	"github.com/verana-labs/verana/testharness/lib"
)

// RunGovernanceFrameworkUpdateJourney implements Journey 9: Governance Framework Update Journey
func RunGovernanceFrameworkUpdateJourney(ctx context.Context, client cosmosclient.Client) error {
	fmt.Println("Starting Journey 9: Governance Framework Update Journey")

	// Step 1: Load data from previous journeys to get an existing Trust Registry
	journey1Result, err := lib.GetJourneyResult("journey1")
	if err != nil {
		return fmt.Errorf("failed to load Journey 1 results: %v", err)
	}

	fmt.Println("✅ Step 1: Loaded data from Journey 1")
	fmt.Printf("    - Trust Registry ID: %s\n", journey1Result.TrustRegistryID)
	fmt.Printf("    - Trust Registry DID: %s\n", journey1Result.DID)

	// Get the Trust_Registry_Controller account
	trustRegistryControllerAccount, err := client.Account(lib.TRUST_REGISTRY_CONTROLLER_NAME)
	if err != nil {
		return fmt.Errorf("failed to get %s account: %v", lib.TRUST_REGISTRY_CONTROLLER_NAME, err)
	}

	// Step 2: Verify the Trust Registry exists and get its current active version
	trustRegistryID, err := strconv.ParseUint(journey1Result.TrustRegistryID, 10, 64)
	if err != nil {
		return fmt.Errorf("failed to parse trust registry ID: %v", err)
	}

	// Query the Trust Registry to get its current state
	trustRegistry, err := lib.QueryTrustRegistry(client, ctx, trustRegistryID)
	if err != nil {
		return fmt.Errorf("failed to query trust registry: %v", err)
	}

	currentActiveVersion := trustRegistry.TrustRegistry.ActiveVersion
	fmt.Printf("✅ Step 2: Verified Trust Registry exists with active version: %d\n", currentActiveVersion)

	// Step 3: Trust_Registry_Controller adds a new governance framework document
	fmt.Println("Trust_Registry_Controller adding new governance framework document...")

	// Create a new governance framework document with version = currentActiveVersion + 1
	newVersion := currentActiveVersion + 1
	documentURL := "https://example.com/governance-framework-v2.pdf"
	documentDigestSRI := "sha384-UpdatedDigestForTheGovernanceFrameworkDocument123456789"
	language := "en" // Use the same language as the original

	// Create the message to add a new governance framework document
	addDocMsg := &trtypes.MsgAddGovernanceFrameworkDocument{
		Id:           trustRegistryID,
		DocLanguage:  language,
		DocUrl:       documentURL,
		DocDigestSri: documentDigestSRI,
		Version:      newVersion,
	}

	addDocResponse, err := lib.AddGovernanceFrameworkDocument(client, ctx, trustRegistryControllerAccount, *addDocMsg)
	if err != nil {
		return fmt.Errorf("failed to add governance framework document: %v", err)
	}

	fmt.Printf("✅ Step 3: Added new governance framework document with version %d\n", newVersion)
	fmt.Printf("    - Document URL: %s\n", documentURL)
	fmt.Printf("    - Document Digest SRI: %s\n", documentDigestSRI)
	fmt.Printf("    - Response: %s\n", addDocResponse)

	// Verify the new document was added
	updatedTrustRegistry, err := lib.QueryTrustRegistry(client, ctx, trustRegistryID)
	if err != nil {
		return fmt.Errorf("failed to query updated trust registry: %v", err)
	}

	// Check that the trust registry has multiple versions
	hasNewVersion := false
	for _, version := range updatedTrustRegistry.TrustRegistry.Versions {
		if version.Version == newVersion {
			hasNewVersion = true
			break
		}
	}

	if !hasNewVersion {
		return fmt.Errorf("new governance framework version not found after adding document")
	}

	fmt.Println("    - Verified new governance framework document was successfully added")

	// Step 4: Trust_Registry_Controller activates the new version
	fmt.Println("Trust_Registry_Controller activating new governance framework version...")

	// Create the message to increase the active version
	increaseVersionMsg := &trtypes.MsgIncreaseActiveGovernanceFrameworkVersion{
		Id: trustRegistryID,
	}

	increaseVersionResponse, err := lib.IncreaseActiveGovernanceFrameworkVersion(client, ctx, trustRegistryControllerAccount, *increaseVersionMsg)
	if err != nil {
		return fmt.Errorf("failed to increase active governance framework version: %v", err)
	}

	fmt.Printf("✅ Step 4: Activated new governance framework version\n")
	fmt.Printf("    - Response: %s\n", increaseVersionResponse)

	// Verify the active version has been increased
	if !lib.VerifyGovernanceFrameworkUpdate(client, ctx, trustRegistryID, uint32(newVersion)) {
		return fmt.Errorf("governance framework update verification failed")
	}

	fmt.Println("    - All ecosystem participants now operate under the updated governance framework")

	fmt.Println("Journey 9 completed successfully! ✨")

	// Store the result for future reference
	result := lib.JourneyResult{
		TrustRegistryID:  journey1Result.TrustRegistryID,
		SchemaID:         journey1Result.SchemaID,
		RootPermissionID: journey1Result.RootPermissionID,
		DID:              journey1Result.DID,
		GfVersion:        strconv.FormatUint(uint64(newVersion), 10),
		GfDocumentURL:    documentURL,
	}

	// Save result to global state or file for other journeys to use
	lib.SaveJourneyResult("journey9", result)

	return nil
}
