package journeys

import (
	"context"
	"fmt"
	"strconv"
	"time"

	"github.com/ignite/cli/v28/ignite/pkg/cosmosclient"

	cschema "github.com/verana-labs/verana/x/cs/types"
	permtypes "github.com/verana-labs/verana/x/perm/types"

	"github.com/verana-labs/verana/testharness/lib"
)

// RunPermissionRevokeJourney implements Journey 306: Test RevokePermission
// with operator authorization. For the operation: (a) try without auth -> fail,
// (b) grant auth, (c) try with auth -> succeed, (d) verify revoked fields,
// (e) unauthorized operator rejected.
// Depends on Journey 301 (setup), 302 (group/operator), and 304 (root permission).
func RunPermissionRevokeJourney(ctx context.Context, client cosmosclient.Client) error {
	fmt.Println("Starting Journey 306: RevokePermission with Operator Authorization")

	// Load results from prior journeys
	setup301 := lib.LoadJourneyResult("journey301")
	setup302 := lib.LoadJourneyResult("journey302")
	setup304 := lib.LoadJourneyResult("journey304")
	policyAddr := setup302.GroupPolicyAddr
	operatorAccount := lib.GetAccount(client, permOperatorName)
	operatorAddr := setup302.OperatorAddr
	adminAccount := lib.GetAccount(client, permGroupAdminName)
	member1Account := lib.GetAccount(client, permGroupMember1Name)

	fmt.Printf("  Group Policy: %s\n", policyAddr)
	fmt.Printf("  Operator:     %s\n", operatorAddr)

	// =========================================================================
	// PREREQUISITE: Create a new CS and ECOSYSTEM root permission to revoke.
	// We need a new CS to avoid overlap check with the permission from journey 304/305
	// (same schema_id + type + authority would trigger the overlap check).
	// =========================================================================
	fmt.Println("\n=== PREREQUISITE: Create new CS and root permission to revoke ===")

	trID, _ := strconv.ParseUint(setup304.TrustRegistryID, 10, 64)

	// Re-grant CreateCredentialSchema auth (may have been overwritten)
	fmt.Println("\n--- Prerequisite 1: Re-grant CreateCredentialSchema auth ---")
	err := lib.GrantOperatorAuthorizationViaGroup(
		client, ctx, adminAccount, member1Account,
		policyAddr, operatorAddr, operatorAddr,
		[]string{"/verana.cs.v1.MsgCreateCredentialSchema"},
	)
	if err != nil {
		return fmt.Errorf("prerequisite 1 failed: could not grant CS auth: %w", err)
	}
	fmt.Println("OK Prerequisite 1: Re-granted CreateCredentialSchema authorization")
	waitForTx("re-grant CS auth")

	// Create a new CS on the same TR
	fmt.Println("\n--- Prerequisite 2: Create new Credential Schema ---")
	schemaData := lib.GenerateSimpleSchema(setup304.TrustRegistryID)
	csIDStr, err := lib.CreateCredentialSchemaWithAuthority(
		client, ctx, operatorAccount, policyAddr,
		trID, schemaData,
		cschema.CredentialSchemaPermManagementMode_GRANTOR_VALIDATION,
		cschema.CredentialSchemaPermManagementMode_GRANTOR_VALIDATION,
	)
	if err != nil {
		return fmt.Errorf("prerequisite 2 failed: could not create CS: %w", err)
	}
	csID, _ := strconv.ParseUint(csIDStr, 10, 64)
	fmt.Printf("OK Prerequisite 2: Credential Schema created with ID: %d\n", csID)
	waitForTx("CS creation for journey 306")

	// Re-grant CreateRootPermission auth (may have been overwritten by journey 305's AdjustPermission grant)
	fmt.Println("\n--- Prerequisite 3: Re-grant CreateRootPermission auth ---")
	err = lib.GrantOperatorAuthorizationViaGroup(
		client, ctx, adminAccount, member1Account,
		policyAddr, operatorAddr, operatorAddr,
		[]string{"/verana.perm.v1.MsgCreateRootPermission"},
	)
	if err != nil {
		return fmt.Errorf("prerequisite 3 failed: could not grant CreateRootPermission auth: %w", err)
	}
	fmt.Println("OK Prerequisite 3: Re-granted CreateRootPermission authorization")
	waitForTx("re-grant CreateRootPerm auth")

	// Create root permission on the new CS
	rootPermDID := lib.GenerateUniqueDID(client, ctx)
	effectiveFrom := time.Now().Add(10 * time.Second)
	effectiveUntil := effectiveFrom.Add(360 * 24 * time.Hour)
	permID, err := lib.CreateRootPermissionWithAuthority(
		client, ctx, operatorAccount, policyAddr,
		csID, rootPermDID, &effectiveFrom, &effectiveUntil,
		100, 50, 25,
	)
	if err != nil {
		return fmt.Errorf("prerequisite failed: could not create root permission: %w", err)
	}
	fmt.Printf("OK Prerequisite 4: Root permission created with ID: %d (schema=%d)\n", permID, csID)
	waitForTx("create root perm for revoke test")

	// Wait for the permission to become effective
	perm, err := lib.GetPermission(client, ctx, permID)
	if err != nil {
		return fmt.Errorf("prerequisite failed: could not load permission %d: %w", permID, err)
	}
	if perm.EffectiveFrom != nil {
		waitDuration := time.Until(*perm.EffectiveFrom)
		if waitDuration > 0 {
			fmt.Printf("  Waiting %v for permission to become effective...\n", waitDuration.Round(time.Second))
			time.Sleep(waitDuration + 2*time.Second)
		}
	}
	fmt.Printf("OK Prerequisite: Permission %d is now effective\n", permID)

	// =========================================================================
	// TEST 1: RevokePermission (fail without auth, grant auth, succeed)
	// =========================================================================
	fmt.Println("\n=== TEST 1: RevokePermission ===")

	// 1a: Try WITHOUT authorization (expect failure)
	fmt.Println("\n--- Step 1a: Operator tries RevokePermission without auth (expect failure) ---")
	_, err = lib.RevokePermission(client, ctx, operatorAccount, policyAddr, permID)
	if err := expectAuthorizationError("Step 1a", err); err != nil {
		return err
	}
	fmt.Println("OK Step 1a: RevokePermission correctly rejected without authorization")
	waitForTx("RevokePermission rejection")

	// 1b: Grant authorization for RevokePermission
	fmt.Println("\n--- Step 1b: Grant operator auth for RevokePermission ---")
	err = lib.GrantOperatorAuthorizationViaGroup(
		client, ctx, adminAccount, member1Account,
		policyAddr, operatorAddr, operatorAddr,
		[]string{"/verana.perm.v1.MsgRevokePermission"},
	)
	if err != nil {
		return fmt.Errorf("step 1b failed: %w", err)
	}
	fmt.Println("OK Step 1b: Granted RevokePermission authorization")
	waitForTx("grant RevokePermission auth")

	// 1c: Try WITH authorization (expect success)
	fmt.Println("\n--- Step 1c: Operator revokes permission with auth (expect success) ---")
	_, err = lib.RevokePermission(client, ctx, operatorAccount, policyAddr, permID)
	if err != nil {
		return fmt.Errorf("step 1c failed: %w", err)
	}
	fmt.Printf("OK Step 1c: RevokePermission succeeded for perm %d\n", permID)
	waitForTx("RevokePermission success")

	// =========================================================================
	// TEST 2: Verify revoked permission fields
	// =========================================================================
	fmt.Println("\n=== TEST 2: Verify revoked permission fields ===")
	revokedPerm, err := lib.GetPermission(client, ctx, permID)
	if err != nil {
		return fmt.Errorf("step 2 query failed: %w", err)
	}

	// Verify revoked timestamp is set
	if revokedPerm.Revoked == nil {
		return fmt.Errorf("step 2 failed: revoked timestamp is nil")
	}

	// Verify revoked_by is set to the authority (policy address)
	if revokedPerm.RevokedBy != policyAddr {
		return fmt.Errorf("step 2 failed: expected revoked_by=%s, got %s", policyAddr, revokedPerm.RevokedBy)
	}

	// Verify modified timestamp is set
	if revokedPerm.Modified == nil {
		return fmt.Errorf("step 2 failed: modified timestamp is nil")
	}

	// Verify type unchanged
	if revokedPerm.Type != permtypes.PermissionType_ECOSYSTEM {
		return fmt.Errorf("step 2 failed: expected ECOSYSTEM type, got %s", revokedPerm.Type.String())
	}

	// Verify authority unchanged
	if revokedPerm.Authority != policyAddr {
		return fmt.Errorf("step 2 failed: authority changed unexpectedly")
	}

	fmt.Printf("OK Step 2: Verified revoked fields (revoked_by=%s, revoked is set)\n", policyAddr)

	// =========================================================================
	// TEST 3: Unauthorized operator (negative test)
	// Use the same permID — AUTHZ check happens before basic checks,
	// so "already revoked" doesn't matter; the unauthorized operator is rejected first.
	// =========================================================================
	fmt.Println("\n=== TEST 3: Unauthorized operator (negative test) ===")

	fmt.Println("\n--- Step 3a: Unauthorized operator tries RevokePermission (expect failure) ---")
	cooluser := lib.GetAccount(client, lib.COOLUSER_NAME)
	_, err = lib.RevokePermission(client, ctx, cooluser, policyAddr, permID)
	if err := expectAuthorizationError("Step 3a", err); err != nil {
		return err
	}
	fmt.Println("OK Step 3a: Unauthorized operator correctly rejected")

	// Save results
	result := lib.JourneyResult{
		TrustRegistryID: setup304.TrustRegistryID,
		SchemaID:        setup304.SchemaID,
		DID:             rootPermDID,
		PermissionID:    strconv.FormatUint(permID, 10),
		GroupID:         setup301.GroupID,
		GroupPolicyAddr: policyAddr,
		OperatorAddr:    operatorAddr,
	}
	lib.SaveJourneyResult("journey306", result)

	fmt.Println("\n========================================")
	fmt.Println("Journey 306 completed successfully!")
	fmt.Println("RevokePermission tested: fail without auth, pass with auth, verify fields,")
	fmt.Println("unauthorized operator rejected.")
	fmt.Println("========================================")

	return nil
}
