package journeys

import (
	"context"
	"fmt"

	"github.com/ignite/cli/v28/ignite/pkg/cosmosclient"

	"github.com/verana-labs/verana/testharness/lib"
)

// RunEcGFAddGFDJourney implements Journey 021: AddGovernanceFrameworkDocument for ec-alpha.
// [MOD-GF-MSG-1] — ecosystemID=ec-alpha, various language + version combos.
// Depends on Journey 020.
func RunEcGFAddGFDJourney(ctx context.Context, client cosmosclient.Client) error {
	fmt.Println("Starting Journey 021: AddGFD for ec-alpha")

	setup001 := lib.LoadJourneyResult("journey001")
	setup020 := lib.LoadJourneyResult("journey020")

	policyAddr := setup001.GroupPolicyAddr
	operatorAccount := lib.GetAccount(client, corpAOperator)

	var ecID uint64
	fmt.Sscanf(setup020.EcosystemID, "%d", &ecID)
	fmt.Printf("  ec-alpha ID: %d\n", ecID)
	fmt.Printf("  Policy:      %s\n", policyAddr)

	// ec-alpha was created with v1 active. We add v2 docs and v3 doc here.

	// =========================================================================
	// Step 1: AddGFD v2 (English) for ec-alpha [MOD-GF-MSG-1]
	// =========================================================================
	fmt.Println("\n--- Step 1: AddGFD v2 English for ec-alpha ---")
	err := lib.AddGFDWithAuthority(
		client, ctx, operatorAccount, policyAddr,
		ecID, "en",
		"https://example.com/ec-alpha-gf-v2-en.pdf",
		"sha384-MzNNbQTWCSUSi0bbz7dbua+RcENv7C6FvlmYJ1Y+I727HsPOHdzwELMYO9Mz68M26",
		2,
	)
	if err != nil {
		return fmt.Errorf("step 1 failed: %w", err)
	}
	fmt.Println("✅ Step 1: Added ec-alpha GFD v2 (language=en)")
	waitForTx("AddGFD v2 en ec-alpha")

	// Step 1b: AddGFD for version that skips ahead (v4 when max=2) — must abort
	fmt.Println("\n--- Step 1b: AddGFD v4 (skip v3, expect failure) ---")
	err = lib.AddGFDWithAuthority(
		client, ctx, operatorAccount, policyAddr,
		ecID, "en",
		"https://example.com/ec-alpha-gf-v4-en.pdf",
		"sha384-MzNNbQTWCSUSi0bbz7dbua+RcENv7C6FvlmYJ1Y+I727HsPOHdzwELMYO9Mz68M26",
		4,
	)
	if err == nil {
		return fmt.Errorf("step 1b failed: expected rejection for version skip")
	}
	fmt.Printf("✅ Step 1b: Correctly rejected version skip to v4: %v\n", err)

	// =========================================================================
	// Step 2: AddGFD v2 additional languages
	// =========================================================================
	fmt.Println("\n--- Step 2: AddGFD v2 French + Spanish translations ---")
	for _, lang := range []string{"fr", "es"} {
		err = lib.AddGFDWithAuthority(
			client, ctx, operatorAccount, policyAddr,
			ecID, lang,
			fmt.Sprintf("https://example.com/ec-alpha-gf-v2-%s.pdf", lang),
			"sha384-MzNNbQTWCSUSi0bbz7dbua+RcENv7C6FvlmYJ1Y+I727HsPOHdzwELMYO9Mz68M26",
			2,
		)
		if err != nil {
			return fmt.Errorf("step 2 AddGFD v2 %s failed: %w", lang, err)
		}
		fmt.Printf("✅ Step 2: Added ec-alpha GFD v2 (language=%s)\n", lang)
		waitForTx(fmt.Sprintf("AddGFD v2 %s ec-alpha", lang))
	}

	// =========================================================================
	// Step 3: AddGFD v3 (prepare for IncreaseGFV in Journey 022)
	// =========================================================================
	fmt.Println("\n--- Step 3: AddGFD v3 English (for use in Journey 022) ---")
	err = lib.AddGFDWithAuthority(
		client, ctx, operatorAccount, policyAddr,
		ecID, "en",
		"https://example.com/ec-alpha-gf-v3-en.pdf",
		"sha384-MzNNbQTWCSUSi0bbz7dbua+RcENv7C6FvlmYJ1Y+I727HsPOHdzwELMYO9Mz68M26",
		3,
	)
	if err != nil {
		return fmt.Errorf("step 3 AddGFD v3 failed: %w", err)
	}
	fmt.Println("✅ Step 3: Added ec-alpha GFD v3 (language=en) — ready for IncreaseGFV")
	waitForTx("AddGFD v3 en ec-alpha")

	// =========================================================================
	// Step 4: ListGFVs for ec-alpha to verify
	// =========================================================================
	fmt.Println("\n--- Step 4: ListGFVs for ec-alpha ---")
	listResp, err := lib.ListGFVs(client, ctx, 0, ecID, false, 100)
	if err != nil {
		return fmt.Errorf("step 4 ListGFVs failed: %w", err)
	}
	fmt.Printf("✅ Step 4: ListGFVs returned %d GFV(s) for ec-alpha\n", len(listResp.Versions))
	for _, gfv := range listResp.Versions {
		isActive := !gfv.ActiveSince.IsZero()
		fmt.Printf("  GFV ID=%d version=%d active=%v\n", gfv.Id, gfv.Version, isActive)
	}

	fmt.Println("\n========================================")
	fmt.Println("Journey 021 completed successfully!")
	fmt.Println("ec-alpha GFD v2/v3 added, version-skip rejection verified.")
	fmt.Println("========================================")

	return nil
}
