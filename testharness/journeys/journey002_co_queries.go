package journeys

import (
	"context"
	"fmt"

	"github.com/ignite/cli/v28/ignite/pkg/cosmosclient"

	"github.com/verana-labs/verana/testharness/lib"
)

// RunCorpQueriesJourney implements Journey 002: UpdateCorporation + CO query coverage.
// [MOD-CO-MSG-2] UpdateCorporation (fail-then-pass) and [MOD-CO-QUERY-1/2] queries.
// Depends on Journey 001.
func RunCorpQueriesJourney(ctx context.Context, client cosmosclient.Client) error {
	fmt.Println("Starting Journey 002: UpdateCorporation + CO Queries")

	setup := lib.LoadJourneyResult("journey001")
	policyAddr := setup.GroupPolicyAddr
	operatorAccount := lib.GetAccount(client, corpAOperator)
	operatorAddr := setup.OperatorAddr

	corpIDStr := setup.EcosystemID // reused field: stores the corporation ID for J001
	var corpID uint64
	fmt.Sscanf(corpIDStr, "%d", &corpID)

	fmt.Printf("  Corp ID:     %d\n", corpID)
	fmt.Printf("  Policy Addr: %s\n", policyAddr)
	fmt.Printf("  Operator:    %s\n", operatorAddr)

	// =========================================================================
	// Step 1: GetCorporation [MOD-CO-QUERY-1]
	// =========================================================================
	fmt.Println("\n--- Step 1: GetCorporation ---")
	corpResp, err := lib.QueryCorporation(client, ctx, corpID)
	if err != nil {
		return fmt.Errorf("step 1 failed: %w", err)
	}
	fmt.Printf("✅ Step 1: Corp ID=%d DID=%s language=%s active_version=%d\n",
		corpID, corpResp.Corporation.Did, corpResp.Corporation.Language, corpResp.Corporation.ActiveVersion)

	// Step 1b: GetCorporation with non-existent ID (expect error)
	fmt.Println("\n--- Step 1b: GetCorporation with non-existent ID (expect error) ---")
	_, err = lib.QueryCorporation(client, ctx, 999999)
	if err == nil {
		return fmt.Errorf("step 1b failed: expected error for non-existent corporation")
	}
	fmt.Printf("✅ Step 1b: Correctly returned error for non-existent corp: %v\n", err)

	// =========================================================================
	// Step 2: ListCorporations [MOD-CO-QUERY-2]
	// =========================================================================
	fmt.Println("\n--- Step 2: ListCorporations ---")
	listResp, err := lib.ListCorporations(client, ctx, 100)
	if err != nil {
		return fmt.Errorf("step 2 failed: %w", err)
	}
	found := false
	for _, corp := range listResp.Corporations {
		if corp.Id == corpID {
			found = true
			break
		}
	}
	if !found {
		return fmt.Errorf("step 2 failed: Corp A (ID=%d) not found in list", corpID)
	}
	fmt.Printf("✅ Step 2: ListCorporations returned %d corps, Corp A found\n", len(listResp.Corporations))

	// =========================================================================
	// Step 3: UpdateCorporation fail-then-pass [MOD-CO-MSG-2]
	// =========================================================================
	fmt.Println("\n--- Step 3a: Operator tries UpdateCorporation without UpdateCorporation authz (expect failure) ---")
	// Operator has all authz except UpdateCorporation (bootstrapped in J001 for other msgs).
	// Actually J001 bootstrapped UpdateCorporation too — so here we test with a fresh account
	// that has no authz at all.
	cooluser := lib.GetAccount(client, lib.COOLUSER_NAME)
	err = lib.UpdateCorporationWithAuthority(
		client, ctx, cooluser, policyAddr,
		"did:example:corp-a-updated",
	)
	if err == nil {
		return fmt.Errorf("step 3a failed: expected rejection for unauthorized UpdateCorporation")
	}
	fmt.Printf("✅ Step 3a: Correctly rejected unauthorized UpdateCorporation: %v\n", err)
	waitForTx("UpdateCorporation rejection")

	// Step 3b: UpdateCorporation authz was bootstrapped in J001 — no re-grant needed.
	fmt.Println("\n--- Step 3b: UpdateCorporation authz already granted in Journey 001 ---")
	fmt.Println("✅ Step 3b: Skipping re-grant; operator authz set in Journey 001")

	// Step 3c: UpdateCorporation with authorized operator
	fmt.Println("\n--- Step 3c: Authorized operator updates corporation DID ---")
	updatedDID := fmt.Sprintf("%s-updated", setup.DID)
	err = lib.UpdateCorporationWithAuthority(
		client, ctx, operatorAccount, policyAddr,
		updatedDID,
	)
	if err != nil {
		return fmt.Errorf("step 3c failed: %w", err)
	}
	fmt.Printf("✅ Step 3c: UpdateCorporation succeeded — new DID=%s\n", updatedDID)
	waitForTx("UpdateCorporation success")

	// Step 3d: Verify update
	fmt.Println("\n--- Step 3d: Verify updated DID via GetCorporation ---")
	corpResp2, err := lib.QueryCorporation(client, ctx, corpID)
	if err != nil {
		return fmt.Errorf("step 3d query failed: %w", err)
	}
	if corpResp2.Corporation.Did != updatedDID {
		return fmt.Errorf("step 3d failed: expected DID %s, got %s", updatedDID, corpResp2.Corporation.Did)
	}
	fmt.Printf("✅ Step 3d: Verified updated DID=%s\n", corpResp2.Corporation.Did)

	fmt.Println("\n========================================")
	fmt.Println("Journey 002 completed successfully!")
	fmt.Println("UpdateCorporation tested. CO query coverage done.")
	fmt.Println("========================================")

	return nil
}
