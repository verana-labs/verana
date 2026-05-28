package journeys

import (
	"context"
	"fmt"

	"github.com/ignite/cli/v28/ignite/pkg/cosmosclient"

	"github.com/verana-labs/verana/testharness/lib"
)

// RunEcosystemArchiveJourney implements Journey 024: ArchiveEcosystem (archive + unarchive) for ec-alpha.
// [MOD-EC-MSG-3] — archive=true then archive=false; invalid-state rejections verified.
// Depends on Journey 020.
func RunEcosystemArchiveJourney(ctx context.Context, client cosmosclient.Client) error {
	fmt.Println("Starting Journey 024: ArchiveEcosystem (ec-alpha)")

	setup001 := lib.LoadJourneyResult("journey001")
	setup020 := lib.LoadJourneyResult("journey020")

	policyAddr := setup001.GroupPolicyAddr
	operatorAccount := lib.GetAccount(client, corpAOperator)
	operatorAddr := setup001.OperatorAddr

	var ecID uint64
	fmt.Sscanf(setup020.EcosystemID, "%d", &ecID)
	fmt.Printf("  ec-alpha ID: %d\n", ecID)
	fmt.Printf("  Operator:    %s\n", operatorAddr)

	// =========================================================================
	// Step 1: Unauthorized caller tries ArchiveEcosystem (expect failure)
	// =========================================================================
	fmt.Println("\n--- Step 1: Unauthorized caller tries ArchiveEcosystem (expect failure) ---")
	cooluser := lib.GetAccount(client, lib.COOLUSER_NAME)
	err := lib.ArchiveEcosystemWithAuthority(
		client, ctx, cooluser, policyAddr,
		ecID, true,
	)
	if err == nil {
		return fmt.Errorf("step 1 failed: expected rejection for unauthorized ArchiveEcosystem")
	}
	fmt.Printf("✅ Step 1: Correctly rejected unauthorized ArchiveEcosystem: %v\n", err)

	// =========================================================================
	// Step 2: Try to unarchive a non-archived ecosystem (expect failure) [MOD-EC-MSG-3]
	// =========================================================================
	fmt.Println("\n--- Step 2: Unarchive a non-archived ecosystem (expect failure) ---")
	err = lib.ArchiveEcosystemWithAuthority(
		client, ctx, operatorAccount, policyAddr,
		ecID, false,
	)
	if err == nil {
		return fmt.Errorf("step 2 failed: expected rejection for unarchiving a non-archived ecosystem")
	}
	fmt.Printf("✅ Step 2: Correctly rejected unarchive on non-archived ecosystem: %v\n", err)

	// =========================================================================
	// Step 3: Archive the ecosystem [MOD-EC-MSG-3]
	// =========================================================================
	fmt.Println("\n--- Step 3: Authorized operator archives ec-alpha ---")
	err = lib.ArchiveEcosystemWithAuthority(
		client, ctx, operatorAccount, policyAddr,
		ecID, true,
	)
	if err != nil {
		return fmt.Errorf("step 3 failed: %w", err)
	}
	fmt.Println("✅ Step 3: Ecosystem archived")
	waitForTx("ArchiveEcosystem archive")

	// Verify archived state
	ecResp, err := lib.QueryEcosystem(client, ctx, ecID)
	if err != nil {
		return fmt.Errorf("step 3 verification query failed: %w", err)
	}
	if !ecResp.Ecosystem.Archived {
		return fmt.Errorf("step 3 verification failed: ecosystem should be archived")
	}
	fmt.Println("✅ Step 3: Verified ecosystem.archived=true")

	// =========================================================================
	// Step 4: Try to archive an already-archived ecosystem (expect failure)
	// =========================================================================
	fmt.Println("\n--- Step 4: Archive an already-archived ecosystem (expect failure) ---")
	err = lib.ArchiveEcosystemWithAuthority(
		client, ctx, operatorAccount, policyAddr,
		ecID, true,
	)
	if err == nil {
		return fmt.Errorf("step 4 failed: expected rejection for archiving an already-archived ecosystem")
	}
	fmt.Printf("✅ Step 4: Correctly rejected double-archive: %v\n", err)

	// =========================================================================
	// Step 5: Unarchive the ecosystem [MOD-EC-MSG-3]
	// =========================================================================
	fmt.Println("\n--- Step 5: Authorized operator unarchives ec-alpha ---")
	err = lib.ArchiveEcosystemWithAuthority(
		client, ctx, operatorAccount, policyAddr,
		ecID, false,
	)
	if err != nil {
		return fmt.Errorf("step 5 failed: %w", err)
	}
	fmt.Println("✅ Step 5: Ecosystem unarchived")
	waitForTx("ArchiveEcosystem unarchive")

	// Verify unarchived state
	ecResp, err = lib.QueryEcosystem(client, ctx, ecID)
	if err != nil {
		return fmt.Errorf("step 5 verification query failed: %w", err)
	}
	if ecResp.Ecosystem.Archived {
		return fmt.Errorf("step 5 verification failed: ecosystem should be unarchived")
	}
	fmt.Println("✅ Step 5: Verified ecosystem.archived=false")

	// =========================================================================
	// Step 6: Try to unarchive again (non-archived → must abort)
	// =========================================================================
	fmt.Println("\n--- Step 6: Unarchive a non-archived ecosystem again (expect failure) ---")
	err = lib.ArchiveEcosystemWithAuthority(
		client, ctx, operatorAccount, policyAddr,
		ecID, false,
	)
	if err == nil {
		return fmt.Errorf("step 6 failed: expected rejection for double-unarchive")
	}
	fmt.Printf("✅ Step 6: Correctly rejected double-unarchive: %v\n", err)

	fmt.Println("\n========================================")
	fmt.Println("Journey 024 completed successfully!")
	fmt.Println("ArchiveEcosystem: archive, double-archive rejection, unarchive, double-unarchive rejection verified.")
	fmt.Println("========================================")

	return nil
}
