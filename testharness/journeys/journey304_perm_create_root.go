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

// RunPermissionCreateRootJourney implements Journey 304: Test CreateRootPermission
// with operator authorization. For the operation: (a) try without auth -> fail,
// (b) grant auth, (c) try with auth -> succeed.
// Depends on Journey 301 (setup) and Journey 302 (for group/operator info).
func RunPermissionCreateRootJourney(ctx context.Context, client cosmosclient.Client) error {
	fmt.Println("Starting Journey 304: CreateRootPermission with Operator Authorization")

	// Load results from Journey 301 and 302
	setup301 := lib.LoadJourneyResult("journey301")
	setup302 := lib.LoadJourneyResult("journey302")
	policyAddr := setup302.GroupPolicyAddr
	operatorAccount := lib.GetAccount(client, permOperatorName)
	operatorAddr := setup302.OperatorAddr
	adminAccount := lib.GetAccount(client, permGroupAdminName)
	member1Account := lib.GetAccount(client, permGroupMember1Name)

	fmt.Printf("  Group Policy: %s\n", policyAddr)
	fmt.Printf("  Operator:     %s\n", operatorAddr)

	// =========================================================================
	// PREREQUISITE 1: Create TR with controller = group policy
	// Journey 302's TR has controller = operatorAddr (self-delegation).
	// CreateRootPermission checks tr.Controller == msg.Authority, so we need
	// a TR where controller = policyAddr.
	// =========================================================================
	fmt.Println("\n=== PREREQUISITE 1: Create Trust Registry (controller = group policy) ===")

	// Grant TR create authorization to the operator
	fmt.Println("\n--- Prerequisite 1a: Grant operator auth for CreateTrustRegistry ---")
	err := lib.GrantOperatorAuthorizationViaGroup(
		client, ctx, adminAccount, member1Account,
		policyAddr, operatorAddr, operatorAddr,
		[]string{"/verana.tr.v1.MsgCreateTrustRegistry"},
	)
	if err != nil {
		return fmt.Errorf("prerequisite 1a failed: %w", err)
	}
	fmt.Println("OK Prerequisite 1a: Granted CreateTrustRegistry authorization")
	waitForTx("grant TR create auth")

	// Create TR with controller = policyAddr
	fmt.Println("\n--- Prerequisite 1b: Create Trust Registry with controller = group policy ---")
	did := lib.GenerateUniqueDID(client, ctx)
	trIDStr, err := lib.CreateTrustRegistryWithAuthority(
		client, ctx, operatorAccount, policyAddr,
		did,
		"http://perm304-test-aka.com",
		"https://perm304-test.com/governance-framework.pdf",
		"sha384-MzNNbQTWCSUSi0bbz7dbua+RcENv7C6FvlmYJ1Y+I727HsPOHdzwELMYO9Mz68M26",
		"en",
	)
	if err != nil {
		return fmt.Errorf("prerequisite 1b failed: %w", err)
	}
	trID, _ := strconv.ParseUint(trIDStr, 10, 64)
	fmt.Printf("OK Prerequisite 1b: Trust Registry created with ID: %d\n", trID)
	waitForTx("TR creation for journey 304")

	// =========================================================================
	// PREREQUISITE 2: Create CS on the new TR
	// =========================================================================
	fmt.Println("\n=== PREREQUISITE 2: Create Credential Schema on new TR ===")

	// Grant CS create authorization to the operator
	fmt.Println("\n--- Prerequisite 2a: Grant operator auth for CreateCredentialSchema ---")
	err = lib.GrantOperatorAuthorizationViaGroup(
		client, ctx, adminAccount, member1Account,
		policyAddr, operatorAddr, operatorAddr,
		[]string{"/verana.cs.v1.MsgCreateCredentialSchema"},
	)
	if err != nil {
		return fmt.Errorf("prerequisite 2a failed: %w", err)
	}
	fmt.Println("OK Prerequisite 2a: Granted CreateCredentialSchema authorization")
	waitForTx("grant CS create auth")

	// Create CS with authority = policyAddr on the new TR
	fmt.Println("\n--- Prerequisite 2b: Create Credential Schema ---")
	schemaData := lib.GenerateSimpleSchema(trIDStr)
	csIDStr, err := lib.CreateCredentialSchemaWithAuthority(
		client, ctx, operatorAccount, policyAddr,
		trID, schemaData,
		cschema.CredentialSchemaPermManagementMode_GRANTOR_VALIDATION,
		cschema.CredentialSchemaPermManagementMode_GRANTOR_VALIDATION,
	)
	if err != nil {
		return fmt.Errorf("prerequisite 2b failed: %w", err)
	}
	csID, _ := strconv.ParseUint(csIDStr, 10, 64)
	fmt.Printf("OK Prerequisite 2b: Credential Schema created with ID: %d\n", csID)
	waitForTx("CS creation for journey 304")

	// =========================================================================
	// TEST 1: CreateRootPermission
	// =========================================================================
	fmt.Println("\n=== TEST 1: CreateRootPermission ===")

	rootPermDID := lib.GenerateUniqueDID(client, ctx)

	// 1a: Try WITHOUT authorization (expect failure)
	// Use a far-future time since this will be rejected by authz check anyway
	fmt.Println("\n--- Step 1a: Operator tries CreateRootPermission without auth (expect failure) ---")
	ef1a := time.Now().Add(5 * time.Minute)
	eu1a := ef1a.Add(360 * 24 * time.Hour)
	_, err = lib.CreateRootPermissionWithAuthority(
		client, ctx, operatorAccount, policyAddr,
		csID, rootPermDID, &ef1a, &eu1a,
		0, 0, 0,
	)
	if err := expectAuthorizationError("Step 1a", err); err != nil {
		return err
	}
	waitForTx("CreateRootPerm rejection")

	// 1b: Grant authorization for CreateRootPermission
	fmt.Println("\n--- Step 1b: Grant operator auth for CreateRootPermission ---")
	err = lib.GrantOperatorAuthorizationViaGroup(
		client, ctx, adminAccount, member1Account,
		policyAddr, operatorAddr, operatorAddr,
		[]string{"/verana.perm.v1.MsgCreateRootPermission"},
	)
	if err != nil {
		return fmt.Errorf("step 1b failed: %w", err)
	}
	fmt.Println("OK Step 1b: Granted CreateRootPermission authorization")
	waitForTx("grant CreateRootPerm auth")

	// 1c: Try WITH authorization (expect success)
	// Set effectiveFrom NOW (after all grants are done) to ensure it's in the future
	fmt.Println("\n--- Step 1c: Operator creates root permission with auth (expect success) ---")
	effectiveFrom := time.Now().Add(5 * time.Minute)
	effectiveUntil := effectiveFrom.Add(360 * 24 * time.Hour)
	permID, err := lib.CreateRootPermissionWithAuthority(
		client, ctx, operatorAccount, policyAddr,
		csID, rootPermDID, &effectiveFrom, &effectiveUntil,
		100, 50, 25,
	)
	if err != nil {
		return fmt.Errorf("step 1c failed: %w", err)
	}
	fmt.Printf("OK Step 1c: CreateRootPermission succeeded with ID: %d\n", permID)
	waitForTx("CreateRootPerm success")

	// Verify the created permission
	perm, err := lib.GetPermission(client, ctx, permID)
	if err != nil {
		return fmt.Errorf("step 1c verification query failed: %w", err)
	}
	if perm.Type != permtypes.PermissionType_ECOSYSTEM {
		return fmt.Errorf("step 1c verification failed: expected ECOSYSTEM type, got %s", perm.Type.String())
	}
	if perm.SchemaId != csID {
		return fmt.Errorf("step 1c verification failed: expected schema_id=%d, got %d", csID, perm.SchemaId)
	}
	if perm.Authority != policyAddr {
		return fmt.Errorf("step 1c verification failed: expected authority=%s, got %s", policyAddr, perm.Authority)
	}
	if perm.Did != rootPermDID {
		return fmt.Errorf("step 1c verification failed: expected did=%s, got %s", rootPermDID, perm.Did)
	}
	if perm.ValidationFees != 100 {
		return fmt.Errorf("step 1c verification failed: expected validation_fees=100, got %d", perm.ValidationFees)
	}
	if perm.IssuanceFees != 50 {
		return fmt.Errorf("step 1c verification failed: expected issuance_fees=50, got %d", perm.IssuanceFees)
	}
	if perm.VerificationFees != 25 {
		return fmt.Errorf("step 1c verification failed: expected verification_fees=25, got %d", perm.VerificationFees)
	}
	if perm.Deposit != 0 {
		return fmt.Errorf("step 1c verification failed: expected deposit=0, got %d", perm.Deposit)
	}
	fmt.Printf("OK Step 1c: Verified permission fields (ECOSYSTEM, schema=%d, fees=100/50/25, deposit=0)\n", csID)

	// =========================================================================
	// TEST 2: Unauthorized operator (negative test)
	// =========================================================================
	fmt.Println("\n=== TEST 2: Unauthorized operator (negative test) ===")

	fmt.Println("\n--- Step 2a: Unauthorized operator tries CreateRootPermission (expect failure) ---")
	cooluser := lib.GetAccount(client, lib.COOLUSER_NAME)
	_, err = lib.CreateRootPermissionWithAuthority(
		client, ctx, cooluser, policyAddr,
		csID, rootPermDID, &effectiveFrom, &effectiveUntil,
		0, 0, 0,
	)
	if err := expectAuthorizationError("Step 2a", err); err != nil {
		return err
	}
	fmt.Println("OK Step 2a: Unauthorized operator correctly rejected")

	// Save results
	result := lib.JourneyResult{
		TrustRegistryID: trIDStr,
		SchemaID:        csIDStr,
		DID:             rootPermDID,
		PermissionID:    strconv.FormatUint(permID, 10),
		GroupID:         setup301.GroupID,
		GroupPolicyAddr: policyAddr,
		OperatorAddr:    operatorAddr,
	}
	lib.SaveJourneyResult("journey304", result)

	fmt.Println("\n========================================")
	fmt.Println("Journey 304 completed successfully!")
	fmt.Println("CreateRootPermission tested: fail without auth, pass with auth, unauthorized operator rejected.")
	fmt.Println("========================================")

	return nil
}
