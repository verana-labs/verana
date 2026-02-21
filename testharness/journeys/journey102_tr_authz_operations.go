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

// RunTrustRegistryAuthzOperationsJourney implements Journey 102: Test all TR operations with operator authorization
// For each of the 5 TR message types: (a) try without auth → fail, (b) grant auth, (c) try with auth → succeed.
// Depends on Journey 101 (setup) having been run first.
func RunTrustRegistryAuthzOperationsJourney(ctx context.Context, client cosmosclient.Client) error {
	fmt.Println("Starting Journey 102: TR Operations with Operator Authorization (fail-then-pass)")

	// Load results from Journey 101
	setup := lib.LoadJourneyResult("journey101")
	policyAddr := setup.GroupPolicyAddr
	operatorAccount := lib.GetAccount(client, trOperatorName)
	operatorAddr := setup.OperatorAddr
	adminAccount := lib.GetAccount(client, groupAdminName)
	member1Account := lib.GetAccount(client, groupMember1Name)

	fmt.Printf("  Group Policy: %s\n", policyAddr)
	fmt.Printf("  Operator:     %s\n", operatorAddr)

	// =========================================================================
	// TEST 1: CreateTrustRegistry
	// =========================================================================
	fmt.Println("\n=== TEST 1: CreateTrustRegistry ===")

	// 1a: Try WITHOUT authorization (expect failure)
	fmt.Println("\n--- Step 1a: Operator tries CreateTrustRegistry without auth (expect failure) ---")
	did := lib.GenerateUniqueDID(client, ctx)
	_, err := lib.CreateTrustRegistryWithAuthority(
		client, ctx, operatorAccount, policyAddr,
		did,
		"http://example-aka.com",
		"https://example.com/governance-framework.pdf",
		"sha384-MzNNbQTWCSUSi0bbz7dbua+RcENv7C6FvlmYJ1Y+I727HsPOHdzwELMYO9Mz68M26",
		"en",
	)
	if err := expectAuthorizationError("Step 1a", err); err != nil {
		return err
	}
	waitForTx("CreateTR rejection")

	// 1b: Grant authorization for CreateTrustRegistry
	fmt.Println("\n--- Step 1b: Grant operator auth for CreateTrustRegistry ---")
	err = lib.GrantOperatorAuthorizationViaGroup(
		client, ctx, adminAccount, member1Account,
		policyAddr, operatorAddr, operatorAddr,
		[]string{"/verana.tr.v1.MsgCreateTrustRegistry"},
	)
	if err != nil {
		return fmt.Errorf("step 1b failed: %w", err)
	}
	fmt.Println("✅ Step 1b: Granted CreateTrustRegistry authorization")
	waitForTx("grant CreateTR auth")

	// 1c: Try WITH authorization (expect success)
	fmt.Println("\n--- Step 1c: Operator creates trust registry with auth (expect success) ---")
	did = lib.GenerateUniqueDID(client, ctx)
	trIDStr, err := lib.CreateTrustRegistryWithAuthority(
		client, ctx, operatorAccount, policyAddr,
		did,
		"http://example-aka.com",
		"https://example.com/governance-framework.pdf",
		"sha384-MzNNbQTWCSUSi0bbz7dbua+RcENv7C6FvlmYJ1Y+I727HsPOHdzwELMYO9Mz68M26",
		"en",
	)
	if err != nil {
		return fmt.Errorf("step 1c failed: %w", err)
	}
	trID, _ := strconv.ParseUint(trIDStr, 10, 64)
	fmt.Printf("✅ Step 1c: Trust Registry created with ID: %d, DID: %s\n", trID, did)
	waitForTx("TR creation")

	// Verify TR creation
	verified := lib.VerifyTrustRegistry(client, ctx, trID, did)
	if !verified {
		return fmt.Errorf("step 1c verification failed: trust registry not found or DID mismatch")
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
		[]string{"/verana.tr.v1.MsgAddGovernanceFrameworkDocument"},
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
		[]string{"/verana.tr.v1.MsgIncreaseActiveGovernanceFrameworkVersion"},
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
	// TEST 4: UpdateTrustRegistry
	// =========================================================================
	fmt.Println("\n=== TEST 4: UpdateTrustRegistry ===")

	// 4a: Try WITHOUT authorization (expect failure)
	fmt.Println("\n--- Step 4a: Operator tries UpdateTrustRegistry without auth (expect failure) ---")
	newDID := lib.GenerateUniqueDID(client, ctx)
	err = lib.UpdateTrustRegistryWithAuthority(
		client, ctx, operatorAccount, policyAddr,
		trID, newDID, "http://updated-aka.com",
	)
	if err := expectAuthorizationError("Step 4a", err); err != nil {
		return err
	}
	waitForTx("UpdateTR rejection")

	// 4b: Grant authorization for UpdateTrustRegistry
	fmt.Println("\n--- Step 4b: Grant operator auth for UpdateTrustRegistry ---")
	err = lib.GrantOperatorAuthorizationViaGroup(
		client, ctx, adminAccount, member1Account,
		policyAddr, operatorAddr, operatorAddr,
		[]string{"/verana.tr.v1.MsgUpdateTrustRegistry"},
	)
	if err != nil {
		return fmt.Errorf("step 4b failed: %w", err)
	}
	fmt.Println("✅ Step 4b: Granted UpdateTrustRegistry authorization")
	waitForTx("grant UpdateTR auth")

	// 4c: Try WITH authorization (expect success)
	fmt.Println("\n--- Step 4c: Operator updates trust registry with auth (expect success) ---")
	newDID = lib.GenerateUniqueDID(client, ctx)
	err = lib.UpdateTrustRegistryWithAuthority(
		client, ctx, operatorAccount, policyAddr,
		trID, newDID, "http://updated-aka.com",
	)
	if err != nil {
		return fmt.Errorf("step 4c failed: %w", err)
	}
	fmt.Printf("✅ Step 4c: Updated trust registry DID to: %s\n", newDID)
	waitForTx("UpdateTR success")

	// Verify update
	verified = lib.VerifyTrustRegistry(client, ctx, trID, newDID)
	if !verified {
		return fmt.Errorf("step 4c verification failed: DID should be updated")
	}

	// =========================================================================
	// TEST 5: ArchiveTrustRegistry
	// =========================================================================
	fmt.Println("\n=== TEST 5: ArchiveTrustRegistry ===")

	// 5a: Try WITHOUT authorization (expect failure)
	fmt.Println("\n--- Step 5a: Operator tries ArchiveTrustRegistry without auth (expect failure) ---")
	err = lib.ArchiveTrustRegistryWithAuthority(
		client, ctx, operatorAccount, policyAddr,
		trID, true,
	)
	if err := expectAuthorizationError("Step 5a", err); err != nil {
		return err
	}
	waitForTx("ArchiveTR rejection")

	// 5b: Grant authorization for ArchiveTrustRegistry
	fmt.Println("\n--- Step 5b: Grant operator auth for ArchiveTrustRegistry ---")
	err = lib.GrantOperatorAuthorizationViaGroup(
		client, ctx, adminAccount, member1Account,
		policyAddr, operatorAddr, operatorAddr,
		[]string{"/verana.tr.v1.MsgArchiveTrustRegistry"},
	)
	if err != nil {
		return fmt.Errorf("step 5b failed: %w", err)
	}
	fmt.Println("✅ Step 5b: Granted ArchiveTrustRegistry authorization")
	waitForTx("grant ArchiveTR auth")

	// 5c: Try WITH authorization — archive (expect success)
	fmt.Println("\n--- Step 5c: Operator archives trust registry with auth (expect success) ---")
	err = lib.ArchiveTrustRegistryWithAuthority(
		client, ctx, operatorAccount, policyAddr,
		trID, true,
	)
	if err != nil {
		return fmt.Errorf("step 5c failed: %w", err)
	}
	fmt.Println("✅ Step 5c: Trust registry archived")
	waitForTx("ArchiveTR success")

	// Verify archived state
	trResp, err := lib.QueryTrustRegistry(client, ctx, trID)
	if err != nil {
		return fmt.Errorf("step 5c verification query failed: %w", err)
	}
	if trResp.TrustRegistry.Archived == nil {
		return fmt.Errorf("step 5c verification failed: trust registry should be archived")
	}
	fmt.Println("✅ Step 5c: Verified trust registry is archived")

	// 5d: Unarchive (same msg type, already authorized — expect success)
	fmt.Println("\n--- Step 5d: Operator unarchives trust registry (already authorized) ---")
	err = lib.ArchiveTrustRegistryWithAuthority(
		client, ctx, operatorAccount, policyAddr,
		trID, false,
	)
	if err != nil {
		return fmt.Errorf("step 5d failed: %w", err)
	}
	fmt.Println("✅ Step 5d: Trust registry unarchived")
	waitForTx("UnarchiveTR success")

	// Verify unarchived state
	trResp, err = lib.QueryTrustRegistry(client, ctx, trID)
	if err != nil {
		return fmt.Errorf("step 5d verification query failed: %w", err)
	}
	if trResp.TrustRegistry.Archived != nil {
		return fmt.Errorf("step 5d verification failed: trust registry should not be archived")
	}
	fmt.Println("✅ Step 5d: Verified trust registry is unarchived")

	// =========================================================================
	// TEST 6: Unauthorized operator (negative test)
	// =========================================================================
	fmt.Println("\n=== TEST 6: Unauthorized operator (negative test) ===")
	fmt.Println("\n--- Step 6: Unauthorized operator tries CreateTrustRegistry (expect failure) ---")

	// Use cooluser as an unauthorized operator (has funds but no DE authorization)
	cooluser := lib.GetAccount(client, lib.COOLUSER_NAME)

	unauthorizedDID := lib.GenerateUniqueDID(client, ctx)
	_, err = lib.CreateTrustRegistryWithAuthority(
		client, ctx, cooluser, policyAddr,
		unauthorizedDID,
		"http://example-aka.com",
		"https://example.com/governance-framework.pdf",
		"sha384-MzNNbQTWCSUSi0bbz7dbua+RcENv7C6FvlmYJ1Y+I727HsPOHdzwELMYO9Mz68M26",
		"en",
	)
	if err := expectAuthorizationError("Step 6", err); err != nil {
		return err
	}

	fmt.Println("\n========================================")
	fmt.Println("Journey 102 completed successfully! ✨")
	fmt.Println("All 5 TR operations tested: fail without auth, pass with auth.")
	fmt.Println("========================================")

	return nil
}
