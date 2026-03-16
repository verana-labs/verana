package journeys

import (
	"context"
	"fmt"
	"strconv"

	"github.com/ignite/cli/v28/ignite/pkg/cosmosclient"

	permtypes "github.com/verana-labs/verana/x/perm/types"

	"github.com/verana-labs/verana/testharness/lib"
)

// RunPermissionCancelVPJourney implements Journey 303: Test CancelPermissionVPLastRequest
// with operator authorization. For the operation: (a) try without auth -> fail, (b) grant auth, (c) try with auth -> succeed.
// Depends on Journey 302 having been run first (provides a permission in PENDING state after renewal).
func RunPermissionCancelVPJourney(ctx context.Context, client cosmosclient.Client) error {
	fmt.Println("Starting Journey 303: CancelPermissionVPLastRequest with Operator Authorization")

	// Load results from Journey 302
	setup302 := lib.LoadJourneyResult("journey302")
	setup301 := lib.LoadJourneyResult("journey301")
	policyAddr := setup302.GroupPolicyAddr
	operatorAccount := lib.GetAccount(client, permOperatorName)
	operatorAddr := setup302.OperatorAddr
	adminAccount := lib.GetAccount(client, permGroupAdminName)
	member1Account := lib.GetAccount(client, permGroupMember1Name)

	permID, _ := strconv.ParseUint(setup302.PermissionID, 10, 64)

	fmt.Printf("  Group Policy: %s\n", policyAddr)
	fmt.Printf("  Operator:     %s\n", operatorAddr)
	fmt.Printf("  Permission:   %d\n", permID)

	// =========================================================================
	// VERIFY PREREQUISITE: Permission must be in PENDING state
	// Journey 302 leaves the permission in VALIDATED state (test 3 re-validates it).
	// We need to renew it first to get it back to PENDING.
	// =========================================================================
	fmt.Println("\n=== PREREQUISITE: Ensure permission is in PENDING state ===")

	perm, err := lib.GetPermission(client, ctx, permID)
	if err != nil {
		return fmt.Errorf("prerequisite failed: could not query permission: %w", err)
	}

	if perm.VpState == permtypes.ValidationState_VALIDATED {
		fmt.Println("Permission is VALIDATED, renewing to get PENDING state...")
		err = lib.RenewPermissionVPWithAuthority(client, ctx, operatorAccount, policyAddr, permID)
		if err != nil {
			return fmt.Errorf("prerequisite failed: could not renew permission: %w", err)
		}
		waitForTx("renew permission for cancel test")

		perm, err = lib.GetPermission(client, ctx, permID)
		if err != nil {
			return fmt.Errorf("prerequisite verification failed: %w", err)
		}
	}

	if perm.VpState != permtypes.ValidationState_PENDING {
		return fmt.Errorf("prerequisite failed: expected PENDING state, got %s", perm.VpState.String())
	}
	fmt.Printf("OK Prerequisite: Permission %d is in PENDING state\n", permID)

	// =========================================================================
	// TEST 1: CancelPermissionVPLastRequest
	// =========================================================================
	fmt.Println("\n=== TEST 1: CancelPermissionVPLastRequest ===")

	// 1a: Try WITHOUT authorization (expect failure)
	fmt.Println("\n--- Step 1a: Operator tries CancelPermissionVPLastRequest without auth (expect failure) ---")
	err = lib.CancelPermissionVPLastRequestWithAuthority(
		client, ctx, operatorAccount, policyAddr, permID,
	)
	if err := expectAuthorizationError("Step 1a", err); err != nil {
		return err
	}
	waitForTx("CancelPermVP rejection")

	// 1b: Grant authorization for CancelPermissionVPLastRequest
	fmt.Println("\n--- Step 1b: Grant operator auth for CancelPermissionVPLastRequest ---")
	err = lib.GrantOperatorAuthorizationViaGroup(
		client, ctx, adminAccount, member1Account,
		policyAddr, operatorAddr, operatorAddr,
		[]string{"/verana.perm.v1.MsgCancelPermissionVPLastRequest"},
	)
	if err != nil {
		return fmt.Errorf("step 1b failed: %w", err)
	}
	fmt.Println("OK Step 1b: Granted CancelPermissionVPLastRequest authorization")
	waitForTx("grant CancelPermVP auth")

	// 1c: Try WITH authorization (expect success)
	fmt.Println("\n--- Step 1c: Operator cancels permission VP with auth (expect success) ---")
	err = lib.CancelPermissionVPLastRequestWithAuthority(
		client, ctx, operatorAccount, policyAddr, permID,
	)
	if err != nil {
		return fmt.Errorf("step 1c failed: %w", err)
	}
	fmt.Println("OK Step 1c: CancelPermissionVPLastRequest succeeded")
	waitForTx("CancelPermVP success")

	// Verify the permission state transition
	perm, err = lib.GetPermission(client, ctx, permID)
	if err != nil {
		return fmt.Errorf("step 1c verification query failed: %w", err)
	}
	// This perm has no VpExp (was validated without expiration), so cancel transitions to TERMINATED
	if perm.VpState != permtypes.ValidationState_TERMINATED {
		return fmt.Errorf("step 1c verification failed: expected TERMINATED state (no vp_exp), got %s", perm.VpState.String())
	}
	fmt.Printf("OK Step 1c: Verified permission is TERMINATED after cancel (no vp_exp set)\n")

	// Verify fees were refunded (vp_current_fees and vp_current_deposit should be 0)
	if perm.VpCurrentFees != 0 {
		return fmt.Errorf("step 1c verification failed: expected vp_current_fees=0, got %d", perm.VpCurrentFees)
	}
	if perm.VpCurrentDeposit != 0 {
		return fmt.Errorf("step 1c verification failed: expected vp_current_deposit=0, got %d", perm.VpCurrentDeposit)
	}
	fmt.Println("OK Step 1c: Verified vp_current_fees=0, vp_current_deposit=0 (fees refunded)")

	// =========================================================================
	// TEST 2: Unauthorized operator (negative test)
	// =========================================================================
	fmt.Println("\n=== TEST 2: Unauthorized operator (negative test) ===")

	// Permission is now TERMINATED (no vp_exp). The unauthorized operator test
	// will still work because the AUTHZ-CHECK rejects before the state check.
	fmt.Println("\n--- Step 2a: Unauthorized operator tries CancelPermissionVPLastRequest (expect failure) ---")
	cooluser := lib.GetAccount(client, lib.COOLUSER_NAME)
	err = lib.CancelPermissionVPLastRequestWithAuthority(
		client, ctx, cooluser, policyAddr, permID,
	)
	if err := expectAuthorizationError("Step 2a", err); err != nil {
		return err
	}
	fmt.Println("OK Step 2a: Unauthorized operator correctly rejected")

	// Save results
	result := lib.JourneyResult{
		TrustRegistryID: setup302.TrustRegistryID,
		SchemaID:        setup302.SchemaID,
		DID:             setup302.DID,
		PermissionID:    setup302.PermissionID,
		GroupID:         setup301.GroupID,
		GroupPolicyAddr: policyAddr,
		OperatorAddr:    operatorAddr,
	}
	lib.SaveJourneyResult("journey303", result)

	fmt.Println("\n========================================")
	fmt.Println("Journey 303 completed successfully!")
	fmt.Println("CancelPermissionVPLastRequest tested: fail without auth, pass with auth, unauthorized operator rejected.")
	fmt.Println("========================================")

	return nil
}
