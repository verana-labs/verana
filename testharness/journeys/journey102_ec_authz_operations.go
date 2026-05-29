package journeys

import (
	"context"
	"fmt"
	"strconv"
	"strings"

	"github.com/ignite/cli/v28/ignite/pkg/cosmosclient"

	"github.com/verana-labs/verana/testharness/lib"
)

// expectAuthorizationError checks that an error contains an authorization-related message.
func expectAuthorizationError(stepName string, err error) error {
	if err == nil {
		return fmt.Errorf("%s: expected authorization error but operation succeeded", stepName)
	}
	errMsg := err.Error()
	if strings.Contains(errMsg, "authorization") ||
		strings.Contains(errMsg, "unauthorized") ||
		strings.Contains(errMsg, "not authorized") ||
		strings.Contains(errMsg, "failed") {
		fmt.Printf("✅ %s: Correctly rejected: %v\n", stepName, err)
		return nil
	}
	return fmt.Errorf("%s: unexpected error: %w", stepName, err)
}

// RunEcosystemAuthzOperationsJourney implements Journey 102: Test all ecosystem operations with operator authorization.
// For each of the 5 EC message types: (a) try without auth → fail, (b) grant auth, (c) try with auth → succeed.
// Depends on Journey 101 (setup) having been run first.
func RunEcosystemAuthzOperationsJourney(ctx context.Context, client cosmosclient.Client) error {
	fmt.Println("Starting Journey 102: EC Operations with Operator Authorization (fail-then-pass)")

	// Load results from Journey 101
	setup := lib.LoadJourneyResult("journey101")
	policyAddr := setup.GroupPolicyAddr
	operatorAccount := lib.GetAccount(client, ecOperatorName)
	operatorAddr := setup.OperatorAddr
	adminAccount := lib.GetAccount(client, groupAdminName)
	member1Account := lib.GetAccount(client, groupMember1Name)

	fmt.Printf("  Group Policy: %s\n", policyAddr)
	fmt.Printf("  Operator:     %s\n", operatorAddr)

	// =========================================================================
	// TEST 1: CreateEcosystem
	// =========================================================================
	fmt.Println("\n=== TEST 1: CreateEcosystem ===")

	// 1a: Try WITHOUT authorization (expect failure)
	fmt.Println("\n--- Step 1a: Operator tries CreateEcosystem without auth (expect failure) ---")
	did := lib.GenerateUniqueDID(client, ctx)
	_, err := lib.CreateEcosystemWithAuthority(
		client, ctx, operatorAccount, policyAddr,
		did,
		"https://example.com/governance-framework.pdf",
		"sha384-MzNNbQTWCSUSi0bbz7dbua+RcENv7C6FvlmYJ1Y+I727HsPOHdzwELMYO9Mz68M26",
		"en",
	)
	if err := expectAuthorizationError("Step 1a", err); err != nil {
		return err
	}
	waitForTx("CreateEcosystem rejection")

	// 1b: Grant authorization for CreateEcosystem
	fmt.Println("\n--- Step 1b: Grant operator auth for CreateEcosystem ---")
	err = lib.GrantOperatorAuthorizationViaGroup(
		client, ctx, adminAccount, member1Account,
		policyAddr, operatorAddr, operatorAddr,
		[]string{"/verana.ec.v1.MsgCreateEcosystem"},
	)
	if err != nil {
		return fmt.Errorf("step 1b failed: %w", err)
	}
	fmt.Println("✅ Step 1b: Granted CreateEcosystem authorization")
	waitForTx("grant CreateEcosystem auth")

	// 1c: Try WITH authorization (expect success)
	fmt.Println("\n--- Step 1c: Operator creates ecosystem with auth (expect success) ---")
	did = lib.GenerateUniqueDID(client, ctx)
	trIDStr, err := lib.CreateEcosystemWithAuthority(
		client, ctx, operatorAccount, policyAddr,
		did,
		"https://example.com/governance-framework.pdf",
		"sha384-MzNNbQTWCSUSi0bbz7dbua+RcENv7C6FvlmYJ1Y+I727HsPOHdzwELMYO9Mz68M26",
		"en",
	)
	if err != nil {
		return fmt.Errorf("step 1c failed: %w", err)
	}
	trID, _ := strconv.ParseUint(trIDStr, 10, 64)
	fmt.Printf("✅ Step 1c: Ecosystem created with ID: %d, DID: %s\n", trID, did)
	waitForTx("ecosystem creation")

	// Verify ecosystem creation
	verified := lib.VerifyEcosystem(client, ctx, trID, did)
	if !verified {
		return fmt.Errorf("step 1c verification failed: ecosystem not found or DID mismatch")
	}

	// =========================================================================
	// TEST 2: AddGovernanceFrameworkDocument
	// =========================================================================
	fmt.Println("\n=== TEST 2: AddGovernanceFrameworkDocument ===")

	// 2a: Try WITHOUT authorization (expect failure)
	fmt.Println("\n--- Step 2a: Operator tries AddGFD without auth (expect failure) ---")
	err = lib.AddGFDWithAuthority(
		client, ctx, operatorAccount, policyAddr,
		trID, "en",
		"https://example.com/gf-v2-en.pdf",
		"sha384-MzNNbQTWCSUSi0bbz7dbua+RcENv7C6FvlmYJ1Y+I727HsPOHdzwELMYO9Mz68M26",
		2,
	)
	if err := expectAuthorizationError("Step 2a", err); err != nil {
		return err
	}
	waitForTx("AddGFD rejection")

	// 2b: Grant authorization for AddGFD
	fmt.Println("\n--- Step 2b: Grant operator auth for AddGFD ---")
	err = lib.GrantOperatorAuthorizationViaGroup(
		client, ctx, adminAccount, member1Account,
		policyAddr, operatorAddr, operatorAddr,
		[]string{"/verana.gf.v1.MsgAddGovernanceFrameworkDocument"},
	)
	if err != nil {
		return fmt.Errorf("step 2b failed: %w", err)
	}
	fmt.Println("✅ Step 2b: Granted AddGFD authorization")
	waitForTx("grant AddGFD auth")

	// 2c: Try WITH authorization (expect success)
	fmt.Println("\n--- Step 2c: Operator adds GFD with auth (expect success) ---")
	err = lib.AddGFDWithAuthority(
		client, ctx, operatorAccount, policyAddr,
		trID, "en",
		"https://example.com/gf-v2-en.pdf",
		"sha384-MzNNbQTWCSUSi0bbz7dbua+RcENv7C6FvlmYJ1Y+I727HsPOHdzwELMYO9Mz68M26",
		2,
	)
	if err != nil {
		return fmt.Errorf("step 2c failed: %w", err)
	}
	fmt.Println("✅ Step 2c: Successfully added GFD for version 2")
	waitForTx("AddGFD success")

	// =========================================================================
	// TEST 3: IncreaseActiveGovernanceFrameworkVersion
	// =========================================================================
	fmt.Println("\n=== TEST 3: IncreaseActiveGovernanceFrameworkVersion ===")

	// 3a: Try WITHOUT authorization (expect failure)
	fmt.Println("\n--- Step 3a: Operator tries IncreaseActiveGFVersion without auth (expect failure) ---")
	err = lib.IncreaseActiveGFVersionWithAuthority(
		client, ctx, operatorAccount, policyAddr, trID,
	)
	if err := expectAuthorizationError("Step 3a", err); err != nil {
		return err
	}
	waitForTx("IncreaseGFV rejection")

	// 3b: Grant authorization for IncreaseActiveGFVersion
	fmt.Println("\n--- Step 3b: Grant operator auth for IncreaseActiveGFVersion ---")
	err = lib.GrantOperatorAuthorizationViaGroup(
		client, ctx, adminAccount, member1Account,
		policyAddr, operatorAddr, operatorAddr,
		[]string{"/verana.gf.v1.MsgIncreaseActiveGovernanceFrameworkVersion"},
	)
	if err != nil {
		return fmt.Errorf("step 3b failed: %w", err)
	}
	fmt.Println("✅ Step 3b: Granted IncreaseActiveGFVersion authorization")
	waitForTx("grant IncreaseGFV auth")

	// 3c: Try WITH authorization (expect success)
	fmt.Println("\n--- Step 3c: Operator increases active GF version with auth (expect success) ---")
	err = lib.IncreaseActiveGFVersionWithAuthority(
		client, ctx, operatorAccount, policyAddr, trID,
	)
	if err != nil {
		return fmt.Errorf("step 3c failed: %w", err)
	}
	fmt.Println("✅ Step 3c: Successfully increased active GF version to 2")
	waitForTx("IncreaseGFV success")

	// Verify active version is now 2
	verified = lib.VerifyGovernanceFrameworkUpdate(client, ctx, trID, 2)
	if !verified {
		return fmt.Errorf("step 3c verification failed: active version should be 2")
	}

	// =========================================================================
	// TEST 4: UpdateEcosystem
	// =========================================================================
	fmt.Println("\n=== TEST 4: UpdateEcosystem ===")

	updatedDid := fmt.Sprintf("%s-updated", did)

	// 4a: Try WITHOUT authorization (expect failure)
	fmt.Println("\n--- Step 4a: Operator tries UpdateEcosystem without auth (expect failure) ---")
	err = lib.UpdateEcosystemWithAuthority(
		client, ctx, operatorAccount, policyAddr,
		trID, updatedDid,
	)
	if err := expectAuthorizationError("Step 4a", err); err != nil {
		return err
	}
	waitForTx("UpdateEcosystem rejection")

	// 4b: Grant authorization for UpdateEcosystem
	fmt.Println("\n--- Step 4b: Grant operator auth for UpdateEcosystem ---")
	err = lib.GrantOperatorAuthorizationViaGroup(
		client, ctx, adminAccount, member1Account,
		policyAddr, operatorAddr, operatorAddr,
		[]string{"/verana.ec.v1.MsgUpdateEcosystem"},
	)
	if err != nil {
		return fmt.Errorf("step 4b failed: %w", err)
	}
	fmt.Println("✅ Step 4b: Granted UpdateEcosystem authorization")
	waitForTx("grant UpdateEcosystem auth")

	// 4c: Try WITH authorization (expect success)
	fmt.Println("\n--- Step 4c: Operator updates ecosystem with auth (expect success) ---")
	err = lib.UpdateEcosystemWithAuthority(
		client, ctx, operatorAccount, policyAddr,
		trID, updatedDid,
	)
	if err != nil {
		return fmt.Errorf("step 4c failed: %w", err)
	}
	fmt.Printf("✅ Step 4c: Updated ecosystem did=%s\n", updatedDid)
	waitForTx("UpdateEcosystem success")

	// Verify update: DID is now the updated value.
	verified = lib.VerifyEcosystem(client, ctx, trID, updatedDid)
	if !verified {
		return fmt.Errorf("step 4c verification failed: ecosystem should now have updated DID %s", updatedDid)
	}

	// =========================================================================
	// TEST 5: ArchiveEcosystem
	// =========================================================================
	fmt.Println("\n=== TEST 5: ArchiveEcosystem ===")

	// 5a: Try WITHOUT authorization (expect failure)
	fmt.Println("\n--- Step 5a: Operator tries ArchiveEcosystem without auth (expect failure) ---")
	err = lib.ArchiveEcosystemWithAuthority(
		client, ctx, operatorAccount, policyAddr,
		trID, true,
	)
	if err := expectAuthorizationError("Step 5a", err); err != nil {
		return err
	}
	waitForTx("ArchiveEcosystem rejection")

	// 5b: Grant authorization for ArchiveEcosystem
	fmt.Println("\n--- Step 5b: Grant operator auth for ArchiveEcosystem ---")
	err = lib.GrantOperatorAuthorizationViaGroup(
		client, ctx, adminAccount, member1Account,
		policyAddr, operatorAddr, operatorAddr,
		[]string{"/verana.ec.v1.MsgArchiveEcosystem"},
	)
	if err != nil {
		return fmt.Errorf("step 5b failed: %w", err)
	}
	fmt.Println("✅ Step 5b: Granted ArchiveEcosystem authorization")
	waitForTx("grant ArchiveEcosystem auth")

	// 5c: Try WITH authorization — archive (expect success)
	fmt.Println("\n--- Step 5c: Operator archives ecosystem with auth (expect success) ---")
	err = lib.ArchiveEcosystemWithAuthority(
		client, ctx, operatorAccount, policyAddr,
		trID, true,
	)
	if err != nil {
		return fmt.Errorf("step 5c failed: %w", err)
	}
	fmt.Println("✅ Step 5c: Ecosystem archived")
	waitForTx("ArchiveEcosystem success")

	// Verify archived state
	trResp, err := lib.QueryEcosystem(client, ctx, trID)
	if err != nil {
		return fmt.Errorf("step 5c verification query failed: %w", err)
	}
	if !trResp.Ecosystem.Archived {
		return fmt.Errorf("step 5c verification failed: ecosystem should be archived")
	}
	fmt.Println("✅ Step 5c: Verified ecosystem is archived")

	// 5d: [MOD-EC-MSG-5] archive is bidirectional; archive=false unarchives.
	fmt.Println("\n--- Step 5d: Operator unarchives ecosystem (expect success) ---")
	err = lib.ArchiveEcosystemWithAuthority(
		client, ctx, operatorAccount, policyAddr,
		trID, false,
	)
	if err != nil {
		return fmt.Errorf("step 5d failed: expected unarchive to succeed: %w", err)
	}
	trResp, err = lib.QueryEcosystem(client, ctx, trID)
	if err != nil {
		return fmt.Errorf("step 5d verification query failed: %w", err)
	}
	if trResp.Ecosystem.Archived {
		return fmt.Errorf("step 5d verification failed: ecosystem should be unarchived")
	}
	fmt.Println("✅ Step 5d: Verified ecosystem is unarchived per [MOD-EC-MSG-5]")

	// 5e: [MOD-EC-MSG-5] unarchiving a non-archived ecosystem must abort.
	fmt.Println("\n--- Step 5e: Unarchive a non-archived ecosystem (expect failure) ---")
	err = lib.ArchiveEcosystemWithAuthority(
		client, ctx, operatorAccount, policyAddr,
		trID, false,
	)
	if err == nil {
		return fmt.Errorf("step 5e failed: expected rejection for unarchiving a non-archived ecosystem")
	}
	fmt.Printf("✅ Step 5e: Correctly rejected unarchive on non-archived ecosystem: %v\n", err)

	// =========================================================================
	// TEST 6: Unauthorized operator (negative test)
	// =========================================================================
	fmt.Println("\n=== TEST 6: Unauthorized operator (negative test) ===")
	fmt.Println("\n--- Step 6: Unauthorized operator tries CreateEcosystem (expect failure) ---")

	// Use cooluser as an unauthorized operator (has funds but no DE authorization)
	cooluser := lib.GetAccount(client, lib.COOLUSER_NAME)

	unauthorizedDID := lib.GenerateUniqueDID(client, ctx)
	_, err = lib.CreateEcosystemWithAuthority(
		client, ctx, cooluser, policyAddr,
		unauthorizedDID,
		"https://example.com/governance-framework.pdf",
		"sha384-MzNNbQTWCSUSi0bbz7dbua+RcENv7C6FvlmYJ1Y+I727HsPOHdzwELMYO9Mz68M26",
		"en",
	)
	if err := expectAuthorizationError("Step 6", err); err != nil {
		return err
	}

	fmt.Println("\n========================================")
	fmt.Println("Journey 102 completed successfully! ✨")
	fmt.Println("All 5 ecosystem operations tested: fail without auth, pass with auth.")
	fmt.Println("========================================")

	return nil
}
