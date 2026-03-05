package journeys

import (
	"context"
	"fmt"
	"strconv"
	"time"

	"github.com/ignite/cli/v28/ignite/pkg/cosmosclient"

	permtypes "github.com/verana-labs/verana/x/perm/types"

	"github.com/verana-labs/verana/testharness/lib"
)

// RunPermissionAdjustJourney implements Journey 305: Test AdjustPermission
// with operator authorization. For the operation: (a) try without auth -> fail,
// (b) grant auth, (c) try with auth -> succeed, (d) verify fields,
// (e) unauthorized operator rejected.
// Depends on Journey 301 (setup), 302 (group/operator), and 304 (root permission).
func RunPermissionAdjustJourney(ctx context.Context, client cosmosclient.Client) error {
	fmt.Println("Starting Journey 305: AdjustPermission with Operator Authorization")

	// Load results from prior journeys
	setup301 := lib.LoadJourneyResult("journey301")
	setup302 := lib.LoadJourneyResult("journey302")
	setup304 := lib.LoadJourneyResult("journey304")
	policyAddr := setup302.GroupPolicyAddr
	operatorAccount := lib.GetAccount(client, permOperatorName)
	operatorAddr := setup302.OperatorAddr
	adminAccount := lib.GetAccount(client, permGroupAdminName)
	member1Account := lib.GetAccount(client, permGroupMember1Name)

	permID, _ := strconv.ParseUint(setup304.PermissionID, 10, 64)

	fmt.Printf("  Group Policy: %s\n", policyAddr)
	fmt.Printf("  Operator:     %s\n", operatorAddr)
	fmt.Printf("  Permission:   %d\n", permID)

	// =========================================================================
	// PREREQUISITE: Verify the root permission from journey 304 exists
	// =========================================================================
	fmt.Println("\n=== PREREQUISITE: Verify root permission from Journey 304 ===")
	perm, err := lib.GetPermission(client, ctx, permID)
	if err != nil {
		return fmt.Errorf("prerequisite failed: could not load permission %d: %w", permID, err)
	}
	if perm.Type != permtypes.PermissionType_ECOSYSTEM {
		return fmt.Errorf("prerequisite failed: expected ECOSYSTEM type, got %s", perm.Type.String())
	}
	fmt.Printf("OK Prerequisite: Root permission %d exists (ECOSYSTEM, schema=%d)\n", permID, perm.SchemaId)

	// Wait for the permission to become effective (journey 304 sets effectiveFrom = now + 10s)
	if perm.EffectiveFrom != nil {
		waitDuration := time.Until(*perm.EffectiveFrom)
		if waitDuration > 0 {
			fmt.Printf("  Waiting %v for permission to become effective...\n", waitDuration.Round(time.Second))
			time.Sleep(waitDuration + 2*time.Second)
		}
	}

	// =========================================================================
	// TEST 1: AdjustPermission (fail without auth, grant auth, succeed)
	// =========================================================================
	fmt.Println("\n=== TEST 1: AdjustPermission ===")

	// 1a: Try WITHOUT authorization (expect failure)
	fmt.Println("\n--- Step 1a: Operator tries AdjustPermission without auth (expect failure) ---")
	newEffectiveUntil := time.Now().Add(720 * 24 * time.Hour) // 720 days from now
	_, err = lib.AdjustPermission(
		client, ctx, operatorAccount, policyAddr,
		permID, &newEffectiveUntil,
	)
	if err := expectAuthorizationError("Step 1a", err); err != nil {
		return err
	}
	fmt.Println("OK Step 1a: AdjustPermission correctly rejected without authorization")
	waitForTx("AdjustPermission rejection")

	// 1b: Grant authorization for AdjustPermission
	fmt.Println("\n--- Step 1b: Grant operator auth for AdjustPermission ---")
	err = lib.GrantOperatorAuthorizationViaGroup(
		client, ctx, adminAccount, member1Account,
		policyAddr, operatorAddr, operatorAddr,
		[]string{"/verana.perm.v1.MsgAdjustPermission"},
	)
	if err != nil {
		return fmt.Errorf("step 1b failed: %w", err)
	}
	fmt.Println("OK Step 1b: Granted AdjustPermission authorization")
	waitForTx("grant AdjustPermission auth")

	// 1c: Try WITH authorization (expect success)
	fmt.Println("\n--- Step 1c: Operator adjusts permission with auth (expect success) ---")
	newEffectiveUntil = time.Now().Add(720 * 24 * time.Hour)
	_, err = lib.AdjustPermission(
		client, ctx, operatorAccount, policyAddr,
		permID, &newEffectiveUntil,
	)
	if err != nil {
		return fmt.Errorf("step 1c failed: %w", err)
	}
	fmt.Printf("OK Step 1c: AdjustPermission succeeded for perm %d\n", permID)
	waitForTx("AdjustPermission success")

	// =========================================================================
	// TEST 2: Verify adjusted permission fields
	// =========================================================================
	fmt.Println("\n=== TEST 2: Verify adjusted permission fields ===")
	adjustedPerm, err := lib.GetPermission(client, ctx, permID)
	if err != nil {
		return fmt.Errorf("step 2 query failed: %w", err)
	}

	// Verify effective_until was updated (compare Unix timestamps with tolerance)
	if adjustedPerm.EffectiveUntil == nil {
		return fmt.Errorf("step 2 failed: effective_until is nil after adjustment")
	}
	diff := adjustedPerm.EffectiveUntil.Unix() - newEffectiveUntil.Unix()
	if diff < -5 || diff > 5 {
		return fmt.Errorf("step 2 failed: effective_until mismatch: got %v, expected ~%v",
			adjustedPerm.EffectiveUntil, newEffectiveUntil)
	}

	// Verify adjusted timestamp is set
	if adjustedPerm.Adjusted == nil {
		return fmt.Errorf("step 2 failed: adjusted timestamp is nil")
	}

	// Verify adjusted_by is set to the authority (policy address)
	if adjustedPerm.AdjustedBy != policyAddr {
		return fmt.Errorf("step 2 failed: expected adjusted_by=%s, got %s", policyAddr, adjustedPerm.AdjustedBy)
	}

	// Verify modified timestamp is set
	if adjustedPerm.Modified == nil {
		return fmt.Errorf("step 2 failed: modified timestamp is nil")
	}

	// Verify other fields unchanged
	if adjustedPerm.Type != permtypes.PermissionType_ECOSYSTEM {
		return fmt.Errorf("step 2 failed: expected ECOSYSTEM type, got %s", adjustedPerm.Type.String())
	}
	if adjustedPerm.Authority != policyAddr {
		return fmt.Errorf("step 2 failed: authority changed unexpectedly")
	}

	fmt.Printf("OK Step 2: Verified adjusted fields (effective_until updated, adjusted_by=%s)\n", policyAddr)

	// =========================================================================
	// TEST 3: Reduce effective_until (v4 allows reduction)
	// =========================================================================
	fmt.Println("\n=== TEST 3: Reduce effective_until (v4 allows reduction) ===")
	reducedEffectiveUntil := time.Now().Add(180 * 24 * time.Hour) // 180 days — less than 720
	_, err = lib.AdjustPermission(
		client, ctx, operatorAccount, policyAddr,
		permID, &reducedEffectiveUntil,
	)
	if err != nil {
		return fmt.Errorf("step 3 failed: reducing effective_until should be allowed in v4: %w", err)
	}
	fmt.Println("OK Step 3: Reduced effective_until successfully (v4 allows reduction)")
	waitForTx("reduce effective_until")

	// Verify the reduction took effect
	reducedPerm, err := lib.GetPermission(client, ctx, permID)
	if err != nil {
		return fmt.Errorf("step 3 verify failed: %w", err)
	}
	if reducedPerm.EffectiveUntil == nil {
		return fmt.Errorf("step 3 verify failed: effective_until is nil")
	}
	diffReduced := reducedPerm.EffectiveUntil.Unix() - reducedEffectiveUntil.Unix()
	if diffReduced < -5 || diffReduced > 5 {
		return fmt.Errorf("step 3 verify failed: effective_until not reduced correctly")
	}
	fmt.Println("OK Step 3: Verified effective_until was reduced")

	// =========================================================================
	// TEST 4: Unauthorized operator (negative test)
	// =========================================================================
	fmt.Println("\n=== TEST 4: Unauthorized operator (negative test) ===")
	fmt.Println("\n--- Step 4a: Unauthorized operator tries AdjustPermission (expect failure) ---")
	cooluser := lib.GetAccount(client, lib.COOLUSER_NAME)
	futureTime := time.Now().Add(365 * 24 * time.Hour)
	_, err = lib.AdjustPermission(
		client, ctx, cooluser, policyAddr,
		permID, &futureTime,
	)
	if err := expectAuthorizationError("Step 4a", err); err != nil {
		return err
	}
	fmt.Println("OK Step 4a: Unauthorized operator correctly rejected")

	// =========================================================================
	// TEST 5: Wrong authority (negative test)
	// =========================================================================
	fmt.Println("\n=== TEST 5: Wrong authority (negative test) ===")
	fmt.Println("\n--- Step 5a: Correct operator but wrong authority (expect failure) ---")
	wrongAuthority := operatorAddr // operatorAddr is not the perm authority (policyAddr is)
	_, err = lib.AdjustPermission(
		client, ctx, operatorAccount, wrongAuthority,
		permID, &futureTime,
	)
	if err == nil {
		return fmt.Errorf("step 5a: expected failure with wrong authority but operation succeeded")
	}
	fmt.Printf("OK Step 5a: Wrong authority correctly rejected: %s\n", err.Error())

	// Save results
	result := lib.JourneyResult{
		TrustRegistryID: setup304.TrustRegistryID,
		SchemaID:        setup304.SchemaID,
		DID:             setup304.DID,
		PermissionID:    setup304.PermissionID,
		GroupID:         setup301.GroupID,
		GroupPolicyAddr: policyAddr,
		OperatorAddr:    operatorAddr,
	}
	lib.SaveJourneyResult("journey305", result)

	fmt.Println("\n========================================")
	fmt.Println("Journey 305 completed successfully!")
	fmt.Println("AdjustPermission tested: fail without auth, pass with auth, verify fields,")
	fmt.Println("reduce effective_until, unauthorized operator rejected, wrong authority rejected.")
	fmt.Println("========================================")

	return nil
}
