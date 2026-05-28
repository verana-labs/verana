package journeys

import (
	"context"
	"fmt"
	"strconv"

	"github.com/ignite/cli/v28/ignite/pkg/cosmosclient"

	"github.com/verana-labs/verana/testharness/lib"
)

// RunEcosystemCreateJourney implements Journey 020: CreateEcosystem (ec-alpha) for Corp A.
// [MOD-EC-MSG-1] CreateEcosystem — fail-then-pass pattern plus query verification.
// Depends on Journey 001 (Corp A + operator authz).
func RunEcosystemCreateJourney(ctx context.Context, client cosmosclient.Client) error {
	fmt.Println("Starting Journey 020: CreateEcosystem (ec-alpha for Corp A)")

	setup := lib.LoadJourneyResult("journey001")
	policyAddr := setup.GroupPolicyAddr
	operatorAccount := lib.GetAccount(client, corpAOperator)
	operatorAddr := setup.OperatorAddr

	fmt.Printf("  Policy Addr: %s\n", policyAddr)
	fmt.Printf("  Operator:    %s\n", operatorAddr)

	// =========================================================================
	// Step 1: CreateEcosystem without authorization (expect failure)
	// =========================================================================
	fmt.Println("\n--- Step 1: Unauthorized caller tries CreateEcosystem (expect failure) ---")
	cooluser := lib.GetAccount(client, lib.COOLUSER_NAME)
	unauthorizedDID := lib.GenerateUniqueDID(client, ctx)
	_, err := lib.CreateEcosystemWithAuthority(
		client, ctx, cooluser, policyAddr,
		unauthorizedDID,
		"https://example.com/ec-alpha-v1.pdf",
		"sha384-MzNNbQTWCSUSi0bbz7dbua+RcENv7C6FvlmYJ1Y+I727HsPOHdzwELMYO9Mz68M26",
		"en",
	)
	if err == nil {
		return fmt.Errorf("step 1 failed: expected rejection for unauthorized CreateEcosystem")
	}
	fmt.Printf("✅ Step 1: Correctly rejected unauthorized CreateEcosystem: %v\n", err)

	// =========================================================================
	// Step 2: CreateEcosystem with authorized operator [MOD-EC-MSG-1]
	// =========================================================================
	fmt.Println("\n--- Step 2: Authorized operator creates ec-alpha ---")
	ecAlphaDID := lib.GenerateUniqueDID(client, ctx)
	ecIDStr, err := lib.CreateEcosystemWithAuthority(
		client, ctx, operatorAccount, policyAddr,
		ecAlphaDID,
		"https://example.com/ec-alpha-v1.pdf",
		"sha384-MzNNbQTWCSUSi0bbz7dbua+RcENv7C6FvlmYJ1Y+I727HsPOHdzwELMYO9Mz68M26",
		"en",
	)
	if err != nil {
		return fmt.Errorf("step 2 failed: %w", err)
	}
	ecID, _ := strconv.ParseUint(ecIDStr, 10, 64)
	fmt.Printf("✅ Step 2: ec-alpha created — ID=%d DID=%s\n", ecID, ecAlphaDID)
	waitForTx("CreateEcosystem ec-alpha")

	// =========================================================================
	// Step 3: GetEcosystem [MOD-EC-QUERY-1]
	// =========================================================================
	fmt.Println("\n--- Step 3: GetEcosystem ---")
	ecResp, err := lib.QueryEcosystem(client, ctx, ecID)
	if err != nil {
		return fmt.Errorf("step 3 GetEcosystem failed: %w", err)
	}
	if ecResp.Ecosystem.Did != ecAlphaDID {
		return fmt.Errorf("step 3 failed: DID mismatch — got %s", ecResp.Ecosystem.Did)
	}
	fmt.Printf("✅ Step 3: GetEcosystem ID=%d DID=%s language=%s active_version=%d\n",
		ecID, ecResp.Ecosystem.Did, ecResp.Ecosystem.Language, ecResp.Ecosystem.ActiveVersion)

	// Step 3b: GetEcosystem with non-existent ID (expect error)
	fmt.Println("\n--- Step 3b: GetEcosystem with non-existent ID (expect error) ---")
	_, err = lib.QueryEcosystem(client, ctx, 999999)
	if err == nil {
		return fmt.Errorf("step 3b failed: expected error for non-existent ecosystem")
	}
	fmt.Printf("✅ Step 3b: Correctly returned error for non-existent ecosystem: %v\n", err)

	// =========================================================================
	// Step 4: ListEcosystems [MOD-EC-QUERY-2]
	// =========================================================================
	fmt.Println("\n--- Step 4: ListEcosystems ---")
	listResp, err := lib.ListEcosystems(client, ctx, 0, 100)
	if err != nil {
		return fmt.Errorf("step 4 ListEcosystems failed: %w", err)
	}
	found := false
	for _, ec := range listResp.Ecosystems {
		if ec.Id == ecID {
			found = true
			break
		}
	}
	if !found {
		return fmt.Errorf("step 4 failed: ec-alpha (ID=%d) not found in ListEcosystems", ecID)
	}
	fmt.Printf("✅ Step 4: ListEcosystems returned %d ecosystems, ec-alpha found\n", len(listResp.Ecosystems))

	// Step 4b: ListEcosystems filtered by Corp A's corporation_id
	corpIDStr := setup.EcosystemID // reused field holds corp ID in J001
	var corpID uint64
	fmt.Sscanf(corpIDStr, "%d", &corpID)
	fmt.Printf("\n--- Step 4b: ListEcosystems filtered by corp_id=%d ---\n", corpID)
	filtResp, err := lib.ListEcosystems(client, ctx, corpID, 100)
	if err != nil {
		return fmt.Errorf("step 4b ListEcosystems(corp_id) failed: %w", err)
	}
	found = false
	for _, ec := range filtResp.Ecosystems {
		if ec.Id == ecID {
			found = true
			break
		}
	}
	if !found {
		return fmt.Errorf("step 4b failed: ec-alpha not in Corp A filtered list")
	}
	fmt.Printf("✅ Step 4b: Corp A filtered list has %d ecosystem(s), ec-alpha present\n", len(filtResp.Ecosystems))

	// =========================================================================
	// Save results
	// =========================================================================
	result := lib.LoadJourneyResult("journey001")
	result.EcAlphaId = strconv.FormatUint(ecID, 10)
	lib.SaveJourneyResult("journey001", result)

	lib.SaveJourneyResult("journey020", lib.JourneyResult{
		EcosystemID: strconv.FormatUint(ecID, 10),
		DID:         ecAlphaDID,
	})

	fmt.Println("\n========================================")
	fmt.Println("Journey 020 completed successfully!")
	fmt.Println("ec-alpha created and verified via GetEcosystem + ListEcosystems.")
	fmt.Println("========================================")

	return nil
}
