package journeys

import (
	"context"
	"fmt"

	"github.com/ignite/cli/v28/ignite/pkg/cosmosclient"

	"github.com/verana-labs/verana/testharness/lib"
)

// RunEcosystemUpdateJourney implements Journey 023: UpdateEcosystem for ec-alpha.
// [MOD-EC-MSG-2] — fail-then-pass with a fresh DID update.
// Depends on Journey 020.
func RunEcosystemUpdateJourney(ctx context.Context, client cosmosclient.Client) error {
	fmt.Println("Starting Journey 023: UpdateEcosystem (ec-alpha)")

	setup001 := lib.LoadJourneyResult("journey001")
	setup020 := lib.LoadJourneyResult("journey020")

	policyAddr := setup001.GroupPolicyAddr
	operatorAccount := lib.GetAccount(client, corpAOperator)
	operatorAddr := setup001.OperatorAddr

	var ecID uint64
	fmt.Sscanf(setup020.EcosystemID, "%d", &ecID)
	originalDID := setup020.DID

	fmt.Printf("  ec-alpha ID:  %d\n", ecID)
	fmt.Printf("  Original DID: %s\n", originalDID)
	fmt.Printf("  Operator:     %s\n", operatorAddr)

	// =========================================================================
	// Step 1: Unauthorized caller tries UpdateEcosystem (expect failure)
	// =========================================================================
	fmt.Println("\n--- Step 1: Unauthorized caller tries UpdateEcosystem (expect failure) ---")
	cooluser := lib.GetAccount(client, lib.COOLUSER_NAME)
	err := lib.UpdateEcosystemWithAuthority(
		client, ctx, cooluser, policyAddr,
		ecID, "did:example:intruder-should-not-land",
	)
	if err == nil {
		return fmt.Errorf("step 1 failed: expected rejection for unauthorized UpdateEcosystem")
	}
	fmt.Printf("✅ Step 1: Correctly rejected unauthorized UpdateEcosystem: %v\n", err)
	waitForTx("UpdateEcosystem rejection")

	// =========================================================================
	// Step 2: Authorized operator updates DID [MOD-EC-MSG-2]
	// =========================================================================
	fmt.Println("\n--- Step 2: Authorized operator updates ec-alpha DID ---")
	updatedDID := lib.GenerateUniqueDID(client, ctx)
	err = lib.UpdateEcosystemWithAuthority(
		client, ctx, operatorAccount, policyAddr,
		ecID, updatedDID,
	)
	if err != nil {
		return fmt.Errorf("step 2 failed: %w", err)
	}
	fmt.Printf("✅ Step 2: UpdateEcosystem succeeded — new DID=%s\n", updatedDID)
	waitForTx("UpdateEcosystem success")

	// =========================================================================
	// Step 3: Verify update via GetEcosystem [MOD-EC-QUERY-1]
	// =========================================================================
	fmt.Println("\n--- Step 3: Verify updated DID via GetEcosystem ---")
	verified := lib.VerifyEcosystem(client, ctx, ecID, updatedDID)
	if !verified {
		return fmt.Errorf("step 3 verification failed: ecosystem DID should be %s", updatedDID)
	}
	fmt.Printf("✅ Step 3: Verified updated DID=%s\n", updatedDID)

	// =========================================================================
	// Step 4: Verify old DID no longer matches
	// =========================================================================
	fmt.Println("\n--- Step 4: Confirm old DID no longer matches ---")
	ecResp, err := lib.QueryEcosystem(client, ctx, ecID)
	if err != nil {
		return fmt.Errorf("step 4 query failed: %w", err)
	}
	if ecResp.Ecosystem.Did == originalDID {
		return fmt.Errorf("step 4 failed: old DID %s still present after update", originalDID)
	}
	fmt.Printf("✅ Step 4: Old DID no longer present. Current DID=%s\n", ecResp.Ecosystem.Did)

	fmt.Println("\n========================================")
	fmt.Println("Journey 023 completed successfully!")
	fmt.Println("UpdateEcosystem tested: unauthorized rejected, authorized DID update verified.")
	fmt.Println("========================================")

	return nil
}
