package journeys

import (
	"context"
	"fmt"
	"time"

	"github.com/ignite/cli/v28/ignite/pkg/cosmosclient"

	"github.com/verana-labs/verana/testharness/lib"
)

// RunEcosystemQueriesJourney implements Journey 025: Comprehensive EC + GF query coverage.
// [MOD-EC-QUERY-1] GetEcosystem, [MOD-EC-QUERY-2] ListEcosystems (filters),
// [MOD-GF-QUERY-1] GetGFV, [MOD-GF-QUERY-2] ListGFVs (filters).
// Depends on Journey 020-022 (ec-alpha created, GFD added, GFV increased).
func RunEcosystemQueriesJourney(ctx context.Context, client cosmosclient.Client) error {
	fmt.Println("Starting Journey 025: EC + GF Query Coverage")

	setup001 := lib.LoadJourneyResult("journey001")
	setup020 := lib.LoadJourneyResult("journey020")

	var ecID uint64
	fmt.Sscanf(setup020.EcosystemID, "%d", &ecID)

	var corpID uint64
	fmt.Sscanf(setup001.EcosystemID, "%d", &corpID)

	fmt.Printf("  Corp ID:     %d\n", corpID)
	fmt.Printf("  ec-alpha ID: %d\n", ecID)

	// =========================================================================
	// Step 1: GetEcosystem — happy path and not-found [MOD-EC-QUERY-1]
	// =========================================================================
	fmt.Println("\n--- Step 1: GetEcosystem ---")
	ecResp, err := lib.QueryEcosystem(client, ctx, ecID)
	if err != nil {
		return fmt.Errorf("step 1 GetEcosystem failed: %w", err)
	}
	fmt.Printf("✅ Step 1: ID=%d DID=%s language=%s active_version=%d archived=%v\n",
		ecID,
		ecResp.Ecosystem.Did,
		ecResp.Ecosystem.Language,
		ecResp.Ecosystem.ActiveVersion,
		ecResp.Ecosystem.Archived,
	)

	_, err = lib.QueryEcosystem(client, ctx, 999999)
	if err == nil {
		return fmt.Errorf("step 1b failed: expected error for non-existent ecosystem")
	}
	fmt.Printf("✅ Step 1b: Non-existent ecosystem returned error: %v\n", err)

	// =========================================================================
	// Step 2: ListEcosystems — unfiltered and corp-filtered [MOD-EC-QUERY-2]
	// =========================================================================
	fmt.Println("\n--- Step 2: ListEcosystems (unfiltered) ---")
	allResp, err := lib.ListEcosystems(client, ctx, 0, 100)
	if err != nil {
		return fmt.Errorf("step 2 ListEcosystems failed: %w", err)
	}
	fmt.Printf("✅ Step 2: ListEcosystems returned %d ecosystem(s) total\n", len(allResp.Ecosystems))

	fmt.Printf("\n--- Step 2b: ListEcosystems filtered by corp_id=%d ---\n", corpID)
	corpResp, err := lib.ListEcosystems(client, ctx, corpID, 100)
	if err != nil {
		return fmt.Errorf("step 2b ListEcosystems(corp) failed: %w", err)
	}
	found := false
	for _, ec := range corpResp.Ecosystems {
		if ec.Id == ecID {
			found = true
		}
	}
	if !found {
		return fmt.Errorf("step 2b failed: ec-alpha not in Corp A filtered list")
	}
	fmt.Printf("✅ Step 2b: Corp A has %d ecosystem(s), ec-alpha present\n", len(corpResp.Ecosystems))

	// =========================================================================
	// Step 3: ListCredentialSchemas (modified_after=epoch) — cross-module sanity
	// =========================================================================
	fmt.Println("\n--- Step 3: ListCredentialSchemas (modified_after=epoch) ---")
	csResp, err := lib.ListCredentialSchemas(client, ctx, time.Unix(0, 0), 100)
	if err != nil {
		return fmt.Errorf("step 3 ListCredentialSchemas failed: %w", err)
	}
	fmt.Printf("✅ Step 3: ListCredentialSchemas returned %d schema(s)\n", len(csResp.Schemas))

	// =========================================================================
	// Step 4: ListGFVs for ec-alpha [MOD-GF-QUERY-2]
	// =========================================================================
	fmt.Println("\n--- Step 4: ListGFVs for ec-alpha (all) ---")
	gfvResp, err := lib.ListGFVs(client, ctx, 0, ecID, false, 100)
	if err != nil {
		return fmt.Errorf("step 4 ListGFVs failed: %w", err)
	}
	fmt.Printf("✅ Step 4: ListGFVs returned %d GFV(s) for ec-alpha\n", len(gfvResp.Versions))
	for _, gfv := range gfvResp.Versions {
		isActive := !gfv.ActiveSince.IsZero()
		fmt.Printf("  GFV ID=%d version=%d active=%v\n", gfv.Id, gfv.Version, isActive)
	}

	fmt.Println("\n--- Step 4b: ListGFVs for ec-alpha (activeOnly=true) ---")
	activeResp, err := lib.ListGFVs(client, ctx, 0, ecID, true, 100)
	if err != nil {
		return fmt.Errorf("step 4b ListGFVs(activeOnly) failed: %w", err)
	}
	for _, gfv := range activeResp.Versions {
		if gfv.ActiveSince.IsZero() {
			return fmt.Errorf("step 4b failed: inactive GFV returned with activeOnly=true")
		}
	}
	fmt.Printf("✅ Step 4b: activeOnly=true returned %d GFV(s), all active\n", len(activeResp.Versions))

	// Step 4c: ListGFVs for Corp A CGF
	fmt.Printf("\n--- Step 4c: ListGFVs for Corp A CGF (corp_id=%d, ecosystem_id=0) ---\n", corpID)
	corpGFVResp, err := lib.ListGFVs(client, ctx, corpID, 0, false, 100)
	if err != nil {
		return fmt.Errorf("step 4c ListGFVs(corp) failed: %w", err)
	}
	fmt.Printf("✅ Step 4c: Corp A CGF has %d GFV(s)\n", len(corpGFVResp.Versions))

	// =========================================================================
	// Step 5: GetGovernanceFrameworkVersion [MOD-GF-QUERY-1]
	// =========================================================================
	fmt.Println("\n--- Step 5: GetGovernanceFrameworkVersion (active GFV of ec-alpha) ---")
	if len(activeResp.Versions) > 0 {
		gfvID := activeResp.Versions[0].Id
		singleGFVResp, err := lib.QueryGFV(client, ctx, gfvID)
		if err != nil {
			return fmt.Errorf("step 5 GetGFV failed: %w", err)
		}
		isActive := !singleGFVResp.Version.ActiveSince.IsZero()
		fmt.Printf("✅ Step 5: GFV ID=%d version=%d active=%v docs=%d\n",
			singleGFVResp.Version.Id,
			singleGFVResp.Version.Version,
			isActive,
			len(singleGFVResp.Version.Documents),
		)
	}

	// Step 5b: GetGFV with non-existent ID
	fmt.Println("\n--- Step 5b: GetGFV with non-existent ID (expect error) ---")
	_, err = lib.QueryGFV(client, ctx, 999999)
	if err == nil {
		return fmt.Errorf("step 5b failed: expected error for non-existent GFV")
	}
	fmt.Printf("✅ Step 5b: Non-existent GFV returned error: %v\n", err)

	fmt.Println("\n========================================")
	fmt.Println("Journey 025 completed successfully!")
	fmt.Println("Full EC + GF query coverage: GetEcosystem, ListEcosystems, GetGFV, ListGFVs all verified.")
	fmt.Println("========================================")

	return nil
}
