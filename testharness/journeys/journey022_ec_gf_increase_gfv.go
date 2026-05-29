package journeys

import (
	"context"
	"fmt"

	"github.com/ignite/cli/v28/ignite/pkg/cosmosclient"

	"github.com/verana-labs/verana/testharness/lib"
)

// RunEcGFIncreaseGFVJourney implements Journey 022: IncreaseActiveGovernanceFrameworkVersion for ec-alpha.
// [MOD-GF-MSG-2] — increases from v1→v2, verifies, then v2→v3.
// Depends on Journey 021 (GFD v2 and v3 already added for ec-alpha).
func RunEcGFIncreaseGFVJourney(ctx context.Context, client cosmosclient.Client) error {
	fmt.Println("Starting Journey 022: IncreaseActiveGFVersion for ec-alpha")

	setup001 := lib.LoadJourneyResult("journey001")
	setup020 := lib.LoadJourneyResult("journey020")

	policyAddr := setup001.GroupPolicyAddr
	operatorAccount := lib.GetAccount(client, corpAOperator)

	var ecID uint64
	fmt.Sscanf(setup020.EcosystemID, "%d", &ecID)
	fmt.Printf("  ec-alpha ID: %d\n", ecID)
	fmt.Printf("  Policy:      %s\n", policyAddr)

	// =========================================================================
	// Step 1: IncreaseActiveGFVersion → v2 [MOD-GF-MSG-2]
	// =========================================================================
	fmt.Println("\n--- Step 1: IncreaseActiveGFVersion v1 → v2 ---")
	err := lib.IncreaseActiveGFVersionWithAuthority(
		client, ctx, operatorAccount, policyAddr, ecID,
	)
	if err != nil {
		return fmt.Errorf("step 1 failed: %w", err)
	}
	fmt.Println("✅ Step 1: Active GF version increased to 2")
	waitForTx("IncreaseGFV v2 ec-alpha")

	// Verify active version is now 2
	verified := lib.VerifyGovernanceFrameworkUpdate(client, ctx, ecID, 2)
	if !verified {
		return fmt.Errorf("step 1 verification failed: active version should be 2")
	}
	fmt.Println("✅ Step 1: Verified active version = 2")

	// =========================================================================
	// Step 2: IncreaseActiveGFVersion → v3 [MOD-GF-MSG-2]
	// =========================================================================
	fmt.Println("\n--- Step 2: IncreaseActiveGFVersion v2 → v3 ---")
	err = lib.IncreaseActiveGFVersionWithAuthority(
		client, ctx, operatorAccount, policyAddr, ecID,
	)
	if err != nil {
		return fmt.Errorf("step 2 failed: %w", err)
	}
	fmt.Println("✅ Step 2: Active GF version increased to 3")
	waitForTx("IncreaseGFV v3 ec-alpha")

	verified = lib.VerifyGovernanceFrameworkUpdate(client, ctx, ecID, 3)
	if !verified {
		return fmt.Errorf("step 2 verification failed: active version should be 3")
	}
	fmt.Println("✅ Step 2: Verified active version = 3")

	// =========================================================================
	// Step 3: Try IncreaseGFV without a v4 doc (must abort) [MOD-GF-MSG-2]
	// =========================================================================
	fmt.Println("\n--- Step 3: IncreaseGFV with no v4 doc (expect failure) ---")
	err = lib.IncreaseActiveGFVersionWithAuthority(
		client, ctx, operatorAccount, policyAddr, ecID,
	)
	if err == nil {
		return fmt.Errorf("step 3 failed: expected rejection when no next-version doc exists")
	}
	fmt.Printf("✅ Step 3: Correctly rejected IncreaseGFV without next doc: %v\n", err)

	// =========================================================================
	// Step 4: AddGFD v4 (en + fr), then IncreaseGFV → v4
	// =========================================================================
	fmt.Println("\n--- Step 4a: AddGFD v4 English (default language, required for IncreaseGFV) ---")
	err = lib.AddGFDWithAuthority(
		client, ctx, operatorAccount, policyAddr,
		ecID, "en",
		"https://example.com/ec-alpha-gf-v4-en.pdf",
		"sha384-MzNNbQTWCSUSi0bbz7dbua+RcENv7C6FvlmYJ1Y+I727HsPOHdzwELMYO9Mz68M26",
		4,
	)
	if err != nil {
		return fmt.Errorf("step 4a AddGFD v4 en failed: %w", err)
	}
	fmt.Println("✅ Step 4a: Added ec-alpha GFD v4 (language=en)")
	waitForTx("AddGFD v4 en ec-alpha")

	fmt.Println("\n--- Step 4a-ii: AddGFD v4 French (additional translation) ---")
	err = lib.AddGFDWithAuthority(
		client, ctx, operatorAccount, policyAddr,
		ecID, "fr",
		"https://example.com/ec-alpha-gf-v4-fr.pdf",
		"sha384-MzNNbQTWCSUSi0bbz7dbua+RcENv7C6FvlmYJ1Y+I727HsPOHdzwELMYO9Mz68M26",
		4,
	)
	if err != nil {
		return fmt.Errorf("step 4a-ii AddGFD v4 fr failed: %w", err)
	}
	fmt.Println("✅ Step 4a-ii: Added ec-alpha GFD v4 (language=fr)")
	waitForTx("AddGFD v4 fr ec-alpha")

	fmt.Println("\n--- Step 4b: IncreaseGFV v3 → v4 ---")
	err = lib.IncreaseActiveGFVersionWithAuthority(
		client, ctx, operatorAccount, policyAddr, ecID,
	)
	if err != nil {
		return fmt.Errorf("step 4b IncreaseGFV v4 failed: %w", err)
	}
	fmt.Println("✅ Step 4b: Active GF version increased to 4")
	waitForTx("IncreaseGFV v4 ec-alpha")

	verified = lib.VerifyGovernanceFrameworkUpdate(client, ctx, ecID, 4)
	if !verified {
		return fmt.Errorf("step 4b verification failed: active version should be 4")
	}
	fmt.Println("✅ Step 4b: Verified active version = 4")

	// =========================================================================
	// Step 5: GetGovernanceFrameworkVersion for the active GFV [MOD-GF-QUERY-1]
	// =========================================================================
	fmt.Println("\n--- Step 5: GetGFV — list and fetch active one ---")
	listResp, err := lib.ListGFVs(client, ctx, 0, ecID, true, 100)
	if err != nil {
		return fmt.Errorf("step 5 ListGFVs failed: %w", err)
	}
	if len(listResp.Versions) == 0 {
		return fmt.Errorf("step 5 failed: no active GFV found")
	}
	activeGFV := listResp.Versions[0]
	gfvResp, err := lib.QueryGFV(client, ctx, activeGFV.Id)
	if err != nil {
		return fmt.Errorf("step 5 GetGFV failed: %w", err)
	}
	isActive := !gfvResp.Version.ActiveSince.IsZero()
	fmt.Printf("✅ Step 5: Active GFV ID=%d version=%d active=%v\n",
		gfvResp.Version.Id,
		gfvResp.Version.Version,
		isActive)

	fmt.Println("\n========================================")
	fmt.Println("Journey 022 completed successfully!")
	fmt.Println("ec-alpha GF version increased v1→v2→v3→v4; no-doc rejection verified.")
	fmt.Println("========================================")

	return nil
}
