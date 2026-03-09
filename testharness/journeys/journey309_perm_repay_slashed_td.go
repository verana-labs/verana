package journeys

import (
	"context"
	"fmt"
	"strconv"

	"github.com/ignite/cli/v28/ignite/pkg/cosmosclient"

	"github.com/verana-labs/verana/testharness/lib"
)

// RunPermissionRepaySlashedTDJourney implements Journey 309: Test RepayPermissionSlashedTrustDeposit
// with operator authorization (authority/operator pattern).
//
// Depends on Journey 308 (slash TD) which creates a slashed ISSUER perm.
//
// TEST 1: RepayPermissionSlashedTrustDeposit (fail without auth, grant auth, succeed)
// TEST 2: Verify repaid permission fields
// TEST 3: Unauthorized operator (negative test)
// TEST 4: Wrong authority (negative test)
func RunPermissionRepaySlashedTDJourney(ctx context.Context, client cosmosclient.Client) error {
	fmt.Println("Starting Journey 309: RepayPermissionSlashedTrustDeposit with Operator Authorization")

	// Load results from prior journeys
	setup301 := lib.LoadJourneyResult("journey301")
	setup302 := lib.LoadJourneyResult("journey302")
	setup308 := lib.LoadJourneyResult("journey308")
	policyAddr := setup302.GroupPolicyAddr
	operatorAccount := lib.GetAccount(client, permOperatorName)
	operatorAddr := setup302.OperatorAddr
	adminAccount := lib.GetAccount(client, permGroupAdminName)
	member1Account := lib.GetAccount(client, permGroupMember1Name)

	issuerPermID, _ := strconv.ParseUint(setup308.PermissionID, 10, 64)

	fmt.Printf("  Group Policy: %s\n", policyAddr)
	fmt.Printf("  Operator:     %s\n", operatorAddr)
	fmt.Printf("  Slashed Perm: %d\n", issuerPermID)

	// Verify the perm is slashed from journey 308
	slashedPerm, err := lib.GetPermission(client, ctx, issuerPermID)
	if err != nil {
		return fmt.Errorf("could not load slashed perm from journey 308: %w", err)
	}
	if slashedPerm.SlashedDeposit == 0 {
		return fmt.Errorf("perm %d is not slashed (slashed_deposit=0), journey 308 may not have run", issuerPermID)
	}
	fmt.Printf("  Slashed deposit: %d\n", slashedPerm.SlashedDeposit)

	// =========================================================================
	// TEST 1: RepayPermissionSlashedTrustDeposit (fail without auth, grant auth, succeed)
	// =========================================================================
	fmt.Println("\n=== TEST 1: RepayPermissionSlashedTrustDeposit ===")

	// 1a: Try WITHOUT authorization (expect failure)
	fmt.Println("\n--- Step 1a: Operator tries RepayPermissionSlashedTrustDeposit without auth (expect failure) ---")
	err = lib.RepayPermissionSlashedTrustDeposit(client, ctx, operatorAccount, policyAddr, issuerPermID)
	if err := expectAuthorizationError("Step 1a", err); err != nil {
		return err
	}
	fmt.Println("OK Step 1a: RepayPermissionSlashedTrustDeposit correctly rejected without authorization")
	waitForTx("repay rejection")

	// 1b: Grant authorization for RepayPermissionSlashedTrustDeposit
	fmt.Println("\n--- Step 1b: Grant operator auth for RepayPermissionSlashedTrustDeposit ---")
	err = lib.GrantOperatorAuthorizationViaGroup(
		client, ctx, adminAccount, member1Account,
		policyAddr, operatorAddr, operatorAddr,
		[]string{"/verana.perm.v1.MsgRepayPermissionSlashedTrustDeposit"},
	)
	if err != nil {
		return fmt.Errorf("step 1b failed: %w", err)
	}
	fmt.Println("OK Step 1b: Granted RepayPermissionSlashedTrustDeposit authorization")
	waitForTx("grant repay auth")

	// 1c: Try WITH authorization (expect success)
	fmt.Println("\n--- Step 1c: Operator repays slashed trust deposit with auth (expect success) ---")
	err = lib.RepayPermissionSlashedTrustDeposit(client, ctx, operatorAccount, policyAddr, issuerPermID)
	if err != nil {
		return fmt.Errorf("step 1c failed: %w", err)
	}
	fmt.Printf("OK Step 1c: RepayPermissionSlashedTrustDeposit succeeded for perm %d\n", issuerPermID)
	waitForTx("repay success")

	// =========================================================================
	// TEST 2: Verify repaid permission fields
	// =========================================================================
	fmt.Println("\n=== TEST 2: Verify repaid permission fields ===")
	repaidPerm, err := lib.GetPermission(client, ctx, issuerPermID)
	if err != nil {
		return fmt.Errorf("step 2 query failed: %w", err)
	}

	// Verify repaid timestamp is set
	if repaidPerm.Repaid == nil {
		return fmt.Errorf("step 2 failed: repaid timestamp is nil")
	}

	// Verify repaid_deposit equals slashed_deposit
	if repaidPerm.RepaidDeposit != slashedPerm.SlashedDeposit {
		return fmt.Errorf("step 2 failed: expected repaid_deposit=%d, got %d", slashedPerm.SlashedDeposit, repaidPerm.RepaidDeposit)
	}

	// Verify modified timestamp is set
	if repaidPerm.Modified == nil {
		return fmt.Errorf("step 2 failed: modified timestamp is nil")
	}

	fmt.Printf("OK Step 2: Verified repaid fields (repaid_deposit=%d)\n", repaidPerm.RepaidDeposit)

	// =========================================================================
	// TEST 3: Unauthorized operator (negative test)
	// =========================================================================
	fmt.Println("\n=== TEST 3: Unauthorized operator (negative test) ===")

	fmt.Println("\n--- Step 3a: Unauthorized operator tries RepayPermissionSlashedTrustDeposit (expect failure) ---")
	coolusrAcct := lib.GetAccount(client, lib.COOLUSER_NAME)
	err = lib.RepayPermissionSlashedTrustDeposit(client, ctx, coolusrAcct, policyAddr, issuerPermID)
	if err := expectAuthorizationError("Step 3a", err); err != nil {
		return err
	}
	fmt.Println("OK Step 3a: Unauthorized operator correctly rejected")

	// =========================================================================
	// TEST 4: Wrong authority (negative test)
	// =========================================================================
	fmt.Println("\n=== TEST 4: Wrong authority (negative test) ===")

	fmt.Println("\n--- Step 4a: Correct operator but wrong authority (expect failure) ---")
	// operatorAddr is not the owner of the perm (policyAddr is)
	err = lib.RepayPermissionSlashedTrustDeposit(client, ctx, operatorAccount, operatorAddr, issuerPermID)
	if err == nil {
		return fmt.Errorf("step 4a failed: expected error for wrong authority, got nil")
	}
	fmt.Printf("OK Step 4a: Wrong authority correctly rejected: %s\n", err.Error())

	// Save results
	result := lib.JourneyResult{
		TrustRegistryID: setup308.TrustRegistryID,
		SchemaID:        setup308.SchemaID,
		DID:             setup308.DID,
		PermissionID:    setup308.PermissionID,
		GroupID:         setup301.GroupID,
		GroupPolicyAddr: policyAddr,
		OperatorAddr:    operatorAddr,
	}
	lib.SaveJourneyResult("journey309", result)

	fmt.Println("\n========================================")
	fmt.Println("Journey 309 completed successfully!")
	fmt.Println("RepayPermissionSlashedTrustDeposit tested:")
	fmt.Println("  - Unauthorized operator rejected")
	fmt.Println("  - Authorized operator succeeded")
	fmt.Println("  - Repaid fields verified")
	fmt.Println("  - Unauthorized operator rejected")
	fmt.Println("  - Wrong authority rejected")
	fmt.Println("========================================")

	return nil
}
