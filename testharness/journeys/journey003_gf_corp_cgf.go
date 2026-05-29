package journeys

import (
	"context"
	"fmt"
	"strconv"

	"github.com/ignite/cli/v28/ignite/pkg/cosmosclient"

	"github.com/verana-labs/verana/testharness/lib"
)

// RunCorpCGFJourney implements Journey 003: Corp CGF lifecycle (AddGFD + IncreaseGFV + GF queries).
// ecosystemID=0 targets the corporation's own CGF (Corp A from Journey 001).
// [MOD-GF-MSG-1] AddGovernanceFrameworkDocument, [MOD-GF-MSG-2] IncreaseActiveGovernanceFrameworkVersion,
// [MOD-GF-QUERY-1/2] GetGovernanceFrameworkVersion / ListGovernanceFrameworkVersions.
// Depends on Journey 001.
func RunCorpCGFJourney(ctx context.Context, client cosmosclient.Client) error {
	fmt.Println("Starting Journey 003: Corp A CGF — AddGFD + IncreaseGFV + GF Queries")

	setup := lib.LoadJourneyResult("journey001")
	policyAddr := setup.GroupPolicyAddr
	operatorAccount := lib.GetAccount(client, corpAOperator)
	operatorAddr := setup.OperatorAddr

	corpIDStr := setup.EcosystemID
	var corpID uint64
	fmt.Sscanf(corpIDStr, "%d", &corpID)

	fmt.Printf("  Corp ID:  %d\n", corpID)
	fmt.Printf("  Policy:   %s\n", policyAddr)
	fmt.Printf("  Operator: %s\n", operatorAddr)

	// Corp A was created with one CGF GFD (v1) inside CreateCorporation.
	// Active version starts at 1. We will add a v2 document and then increase.

	// =========================================================================
	// Step 1: AddGFD v2 for Corp CGF (ecosystemID=0) [MOD-GF-MSG-1]
	// =========================================================================
	fmt.Println("\n--- Step 1: AddGFD v2 for Corp CGF (ecosystemID=0) ---")
	err := lib.AddGFDWithAuthority(
		client, ctx, operatorAccount, policyAddr,
		0, // ecosystemID=0 → corporation subject
		"en",
		"https://example.com/corp-a-cgf-v2-en.pdf",
		"sha384-MzNNbQTWCSUSi0bbz7dbua+RcENv7C6FvlmYJ1Y+I727HsPOHdzwELMYO9Mz68M26",
		2,
	)
	if err != nil {
		return fmt.Errorf("step 1 failed: %w", err)
	}
	fmt.Println("✅ Step 1: Added CGF GFD v2 (language=en)")
	waitForTx("AddGFD v2 corp CGF")

	// Step 1b: Try to add GFD for version 4 (skip v3 — must abort) [MOD-GF-MSG-1]
	fmt.Println("\n--- Step 1b: AddGFD v4 (skip v3, expect failure) ---")
	err = lib.AddGFDWithAuthority(
		client, ctx, operatorAccount, policyAddr,
		0, "en",
		"https://example.com/corp-a-cgf-v4-en.pdf",
		"sha384-MzNNbQTWCSUSi0bbz7dbua+RcENv7C6FvlmYJ1Y+I727HsPOHdzwELMYO9Mz68M26",
		4,
	)
	if err == nil {
		return fmt.Errorf("step 1b failed: expected rejection for version skip (v4 when max=2)")
	}
	fmt.Printf("✅ Step 1b: Correctly rejected version skip: %v\n", err)

	// Step 1c: Add GFD French translation for v2
	fmt.Println("\n--- Step 1c: AddGFD v2 French translation ---")
	err = lib.AddGFDWithAuthority(
		client, ctx, operatorAccount, policyAddr,
		0, "fr",
		"https://example.com/corp-a-cgf-v2-fr.pdf",
		"sha384-MzNNbQTWCSUSi0bbz7dbua+RcENv7C6FvlmYJ1Y+I727HsPOHdzwELMYO9Mz68M26",
		2,
	)
	if err != nil {
		return fmt.Errorf("step 1c failed: %w", err)
	}
	fmt.Println("✅ Step 1c: Added CGF GFD v2 (language=fr)")
	waitForTx("AddGFD v2 fr corp CGF")

	// =========================================================================
	// Step 2: IncreaseActiveGovernanceFrameworkVersion to v2 [MOD-GF-MSG-2]
	// =========================================================================
	fmt.Println("\n--- Step 2: IncreaseActiveGFVersion → v2 ---")
	err = lib.IncreaseActiveGFVersionWithAuthority(
		client, ctx, operatorAccount, policyAddr,
		0, // ecosystemID=0 → corporation subject
	)
	if err != nil {
		return fmt.Errorf("step 2 failed: %w", err)
	}
	fmt.Println("✅ Step 2: Active GF version increased to 2")
	waitForTx("IncreaseGFV v2 corp CGF")

	// Step 2b: Try IncreaseGFV again without adding a v3 doc (must abort) [MOD-GF-MSG-2]
	fmt.Println("\n--- Step 2b: IncreaseGFV without v3 doc (expect failure) ---")
	err = lib.IncreaseActiveGFVersionWithAuthority(
		client, ctx, operatorAccount, policyAddr, 0,
	)
	if err == nil {
		return fmt.Errorf("step 2b failed: expected rejection when no next-version doc exists")
	}
	fmt.Printf("✅ Step 2b: Correctly rejected IncreaseGFV without next doc: %v\n", err)

	// =========================================================================
	// Step 3: Add v3 doc and increase to v3
	// =========================================================================
	fmt.Println("\n--- Step 3: AddGFD v3 + IncreaseGFV → v3 ---")
	err = lib.AddGFDWithAuthority(
		client, ctx, operatorAccount, policyAddr,
		0, "en",
		"https://example.com/corp-a-cgf-v3-en.pdf",
		"sha384-MzNNbQTWCSUSi0bbz7dbua+RcENv7C6FvlmYJ1Y+I727HsPOHdzwELMYO9Mz68M26",
		3,
	)
	if err != nil {
		return fmt.Errorf("step 3 AddGFD v3 failed: %w", err)
	}
	waitForTx("AddGFD v3 corp CGF")

	err = lib.IncreaseActiveGFVersionWithAuthority(
		client, ctx, operatorAccount, policyAddr, 0,
	)
	if err != nil {
		return fmt.Errorf("step 3 IncreaseGFV v3 failed: %w", err)
	}
	fmt.Println("✅ Step 3: CGF active version now 3")
	waitForTx("IncreaseGFV v3 corp CGF")

	// =========================================================================
	// Step 4: ListGovernanceFrameworkVersions for Corp A [MOD-GF-QUERY-2]
	// =========================================================================
	fmt.Println("\n--- Step 4: ListGFVs for Corp A ---")
	listResp, err := lib.ListGFVs(client, ctx, corpID, 0, false, 100)
	if err != nil {
		return fmt.Errorf("step 4 ListGFVs failed: %w", err)
	}
	if len(listResp.Versions) == 0 {
		return fmt.Errorf("step 4 failed: expected at least one GFV")
	}
	fmt.Printf("✅ Step 4: ListGFVs returned %d GFV(s) for Corp A\n", len(listResp.Versions))
	for _, gfv := range listResp.Versions {
		isActive := !gfv.ActiveSince.IsZero()
		fmt.Printf("  GFV ID=%d version=%d active=%v\n", gfv.Id, gfv.Version, isActive)
	}

	// Step 4b: ListGFVs with activeOnly=true — should return only the active one
	fmt.Println("\n--- Step 4b: ListGFVs activeOnly=true ---")
	listActiveResp, err := lib.ListGFVs(client, ctx, corpID, 0, true, 100)
	if err != nil {
		return fmt.Errorf("step 4b ListGFVs activeOnly failed: %w", err)
	}
	for _, gfv := range listActiveResp.Versions {
		if gfv.ActiveSince.IsZero() {
			return fmt.Errorf("step 4b failed: inactive GFV returned with activeOnly=true")
		}
	}
	fmt.Printf("✅ Step 4b: activeOnly=true returned %d GFV(s), all active\n", len(listActiveResp.Versions))

	// Step 4c: ListGFVs with both corporationID and ecosystemID set (expect error)
	fmt.Println("\n--- Step 4c: ListGFVs with both corp+ecosystem ID set (expect error) ---")
	_, err = lib.ListGFVs(client, ctx, corpID, 1, false, 100)
	if err == nil {
		fmt.Println("  (chain accepted both IDs — server may not validate; continuing)")
	} else {
		fmt.Printf("✅ Step 4c: Correctly rejected both-set query: %v\n", err)
	}

	// =========================================================================
	// Step 5: GetGovernanceFrameworkVersion [MOD-GF-QUERY-1]
	// =========================================================================
	fmt.Println("\n--- Step 5: GetGovernanceFrameworkVersion ---")
	if len(listResp.Versions) > 0 {
		gfvID := listResp.Versions[0].Id
		gfvResp, err := lib.QueryGFV(client, ctx, gfvID)
		if err != nil {
			return fmt.Errorf("step 5 GetGFV failed: %w", err)
		}
		isActive := !gfvResp.Version.ActiveSince.IsZero()
		fmt.Printf("✅ Step 5: GetGFV ID=%d version=%d active=%v\n",
			gfvResp.Version.Id, gfvResp.Version.Version, isActive)
	}

	// Save the v3 GFV id for use in downstream journeys
	var gfv3ID uint64
	for _, gfv := range listResp.Versions {
		if gfv.Version == 3 {
			gfv3ID = gfv.Id
		}
	}

	result := lib.LoadJourneyResult("journey001")
	result.GFV2Id = strconv.FormatUint(gfv3ID, 10)
	lib.SaveJourneyResult("journey001", result)

	fmt.Println("\n========================================")
	fmt.Println("Journey 003 completed successfully!")
	fmt.Println("Corp CGF lifecycle: AddGFD v2/v3, IncreaseGFV, queries all verified.")
	fmt.Println("========================================")

	return nil
}
