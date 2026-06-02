package journeys

import (
	"context"
	"fmt"
	"strconv"

	"github.com/ignite/cli/v28/ignite/pkg/cosmosclient"

	permtypes "github.com/verana-labs/verana/x/pp/types"

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

	perm, err := lib.GetParticipant(client, ctx, permID)
	if err != nil {
		return fmt.Errorf("prerequisite failed: could not query permission: %w", err)
	}

	// Grant the operator Renew authz from the Corporation so the prereq renew
	// works regardless of what prior journeys left in the shared operator
	// authorization (MSG-3 grants replace in place).
	if err := lib.GrantOperatorAuthorizationViaGroup(
		client, ctx, adminAccount, member1Account,
		policyAddr, operatorAddr, operatorAddr,
		[]string{"/verana.pp.v1.MsgRenewParticipantOP"},
	); err != nil {
		return fmt.Errorf("prerequisite failed: could not grant renew authz: %w", err)
	}
	waitForTx("grant renew authz for cancel prereq")

	if perm.OpState == permtypes.OnboardingState_VALIDATED {
		fmt.Println("Permission is VALIDATED, renewing to get PENDING state...")
		err = lib.RenewPermissionVPWithAuthority(client, ctx, operatorAccount, policyAddr, permID)
		if err != nil {
			return fmt.Errorf("prerequisite failed: could not renew permission: %w", err)
		}
		waitForTx("renew permission for cancel test")

		perm, err = lib.GetParticipant(client, ctx, permID)
		if err != nil {
			return fmt.Errorf("prerequisite verification failed: %w", err)
		}
	}

	if perm.OpState != permtypes.OnboardingState_PENDING {
		return fmt.Errorf("prerequisite failed: expected PENDING state, got %s", perm.OpState.String())
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
		[]string{"/verana.pp.v1.MsgCancelParticipantOPLastRequest"},
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
	perm, err = lib.GetParticipant(client, ctx, permID)
	if err != nil {
		return fmt.Errorf("step 1c verification query failed: %w", err)
	}
	// [MOD-PP-MSG-6-3] spec v4 draft 13: when op_exp is null (never validated to expiration),
	// cancel transitions op_state to TERMINATED.
	if perm.OpState != permtypes.OnboardingState_TERMINATED {
		return fmt.Errorf("step 1c verification failed: expected TERMINATED state (no op_exp), got %s", perm.OpState.String())
	}
	fmt.Printf("OK Step 1c: Verified permission is TERMINATED after cancel (no op_exp set)\n")

	// Verify fees were refunded (op_current_fees and op_current_deposit should be 0)
	if perm.OpCurrentFees != 0 {
		return fmt.Errorf("step 1c verification failed: expected op_current_fees=0, got %d", perm.OpCurrentFees)
	}
	if perm.OpCurrentDeposit != 0 {
		return fmt.Errorf("step 1c verification failed: expected op_current_deposit=0, got %d", perm.OpCurrentDeposit)
	}
	fmt.Println("OK Step 1c: Verified op_current_fees=0, op_current_deposit=0 (fees refunded)")

	// =========================================================================
	// TEST 2: Unauthorized operator (negative test)
	// =========================================================================
	fmt.Println("\n=== TEST 2: Unauthorized operator (negative test) ===")

	// Permission is now TERMINATED (no op_exp). The unauthorized operator test
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
		EcosystemID:     setup302.EcosystemID,
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
