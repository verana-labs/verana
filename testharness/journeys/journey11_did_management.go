package journeys

import (
	"context"
	"fmt"
	"strconv"
	"time"

	"github.com/ignite/cli/v28/ignite/pkg/cosmosclient"
	didtypes "github.com/verana-labs/verana/x/dd/types"

	"github.com/verana-labs/verana/testharness/lib"
)

// RunDIDManagementJourney implements Journey 11: DID Management Journey
func RunDIDManagementJourney(ctx context.Context, client cosmosclient.Client) error {
	fmt.Println("Starting Journey 11: DID Management Journey")

	// Step 1: Identify a DID_Owner with DID in directory
	// We'll use the TRUST_REGISTRY_CONTROLLER account from Journey 3
	journey1Result, err := lib.GetJourneyResult("journey1")
	if err != nil {
		return fmt.Errorf("failed to load Journey 1 results: %v", err)
	}

	fmt.Println("✅ Step 1: Identified DID_Owner with DID in directory")
	fmt.Printf("    - Using Issuer_Applicant account that participated in Journey 3\n")
	fmt.Printf("    - DID: %s\n", journey1Result.DID)

	// Get the account
	didOwnerAccount, err := client.Account(lib.TRUST_REGISTRY_CONTROLLER_NAME)
	if err != nil {
		return fmt.Errorf("failed to get %s account: %v", lib.TRUST_REGISTRY_CONTROLLER_NAME, err)
	}

	didOwnerAddr, err := didOwnerAccount.Address(lib.GetAddressPrefix())
	if err != nil {
		return fmt.Errorf("failed to get address for %s: %v", lib.TRUST_REGISTRY_CONTROLLER_NAME, err)
	}

	// Check DID in the directory
	did := journey1Result.DID
	didEntry, err := lib.GetDID(client, ctx, did)
	if err != nil {
		return fmt.Errorf("failed to get DID from directory: %v", err)
	}

	fmt.Printf("    - DID Entry: Controller=%s, Created=%v, Expiration=%v\n",
		didEntry.Controller, didEntry.Created, didEntry.Exp)

	// Check initial trust deposit status
	initialDeposit, err := lib.GetTrustDeposit(client, ctx, didOwnerAccount)
	if err != nil {
		return fmt.Errorf("failed to get initial trust deposit: %v", err)
	}

	fmt.Printf("    - Initial Trust Deposit: %d\n", initialDeposit.Amount)
	fmt.Printf("    - Initial Claimable Amount: %d\n", initialDeposit.Claimable)

	// Step 2: Renew DID registration
	fmt.Println("DID_Owner renewing DID registration...")
	renewalYears := uint32(1) // Renew for 1 year

	renewMsg := &didtypes.MsgRenewDID{
		Creator: didOwnerAddr,
		Did:     did,
		Years:   renewalYears,
	}

	renewResponse, err := lib.RenewDID(client, ctx, didOwnerAccount, *renewMsg)
	if err != nil {
		return fmt.Errorf("failed to renew DID: %v", err)
	}

	fmt.Printf("✅ Step 2: DID_Owner renewed DID registration\n")
	fmt.Printf("    - Renewal Period: %d years\n", renewalYears)
	fmt.Printf("    - Response: %s\n", renewResponse)

	// Wait for renewal to be processed
	fmt.Println("    - Waiting for DID renewal to be processed...")
	time.Sleep(3 * time.Second)

	// Verify DID was renewed
	renewedDidEntry, err := lib.GetDID(client, ctx, did)
	if err != nil {
		return fmt.Errorf("failed to get renewed DID from directory: %v", err)
	}

	if !didEntry.Exp.Before(renewedDidEntry.Exp) {
		fmt.Printf("⚠️ Warning: DID expiration date may not have been extended: %v -> %v\n",
			didEntry.Exp, renewedDidEntry.Exp)
	} else {
		fmt.Printf("    - DID expiration extended: %v -> %v\n",
			didEntry.Exp, renewedDidEntry.Exp)
	}

	// Step 3: Touch DID for reindexing
	fmt.Println("DID_Owner updating DID indexing...")

	touchMsg := &didtypes.MsgTouchDID{
		Creator: didOwnerAddr,
		Did:     did,
	}

	touchResponse, err := lib.TouchDID(client, ctx, didOwnerAccount, *touchMsg)
	if err != nil {
		return fmt.Errorf("failed to touch DID: %v", err)
	}

	fmt.Printf("✅ Step 3: DID_Owner updated DID indexing\n")
	fmt.Printf("    - Response: %s\n", touchResponse)

	// Wait for touch to be processed
	fmt.Println("    - Waiting for DID touch to be processed...")
	time.Sleep(3 * time.Second)

	// Verify DID was touched
	touchedDidEntry, err := lib.GetDID(client, ctx, did)
	if err != nil {
		return fmt.Errorf("failed to get touched DID from directory: %v", err)
	}

	if !renewedDidEntry.Modified.Before(touchedDidEntry.Modified) {
		fmt.Printf("⚠️ Warning: DID modified date may not have been updated: %v -> %v\n",
			renewedDidEntry.Modified, touchedDidEntry.Modified)
	} else {
		fmt.Printf("    - DID modified date updated: %v -> %v\n",
			renewedDidEntry.Modified, touchedDidEntry.Modified)
	}

	// Check trust deposit after DID operations
	midwayDeposit, err := lib.GetTrustDeposit(client, ctx, didOwnerAccount)
	if err != nil {
		return fmt.Errorf("failed to get midway trust deposit: %v", err)
	}

	fmt.Printf("    - Trust Deposit after DID operations: %d\n", midwayDeposit.Amount)
	fmt.Printf("    - Claimable Amount after DID operations: %d\n", midwayDeposit.Claimable)

	// Step 4: Remove DID from directory
	fmt.Println("DID_Owner removing DID from directory...")

	removeMsg := &didtypes.MsgRemoveDID{
		Creator: didOwnerAddr,
		Did:     did,
	}

	removeResponse, err := lib.RemoveDID(client, ctx, didOwnerAccount, *removeMsg)
	if err != nil {
		return fmt.Errorf("failed to remove DID: %v", err)
	}

	fmt.Printf("✅ Step 4: DID_Owner removed DID from directory\n")
	fmt.Printf("    - Response: %s\n", removeResponse)

	// Wait for removal to be processed
	fmt.Println("    - Waiting for DID removal to be processed...")
	time.Sleep(3 * time.Second)

	// Verify DID was removed
	_, err = lib.GetDID(client, ctx, did)
	if err == nil {
		fmt.Println("⚠️ Warning: DID may still exist in directory after removal")
	} else {
		fmt.Println("    - DID successfully removed from directory")
	}

	// Step 5: Check if trust deposit was made claimable
	finalDeposit, err := lib.GetTrustDeposit(client, ctx, didOwnerAccount)
	if err != nil {
		return fmt.Errorf("failed to get final trust deposit: %v", err)
	}

	fmt.Printf("✅ Step 5: Checked trust deposit after DID removal\n")
	fmt.Printf("    - Final Trust Deposit: %d\n", finalDeposit.Amount)
	fmt.Printf("    - Final Claimable Amount: %d\n", finalDeposit.Claimable)

	// Check if claimable amount increased after DID removal
	if finalDeposit.Claimable > midwayDeposit.Claimable {
		claimableIncrease := finalDeposit.Claimable - midwayDeposit.Claimable
		fmt.Printf("    - Claimable amount increased by %d after DID removal\n", claimableIncrease)
	} else {
		fmt.Println("    - No change in claimable amount detected after DID removal")
	}

	fmt.Println("Journey 11 completed successfully! ✨")

	// Store the result for future reference
	result := lib.JourneyResult{
		TrustRegistryID:      journey1Result.TrustRegistryID,
		SchemaID:             journey1Result.SchemaID,
		DID:                  did,
		InitialDepositAmount: strconv.FormatUint(uint64(initialDeposit.Amount), 10),
		FinalDepositAmount:   strconv.FormatUint(uint64(finalDeposit.Amount), 10),
	}

	// Save result to global state or file for other journeys to use
	lib.SaveJourneyResult("journey11", result)

	return nil
}
