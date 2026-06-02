package journeys

import (
	"context"
	"fmt"
	"strconv"
	"time"

	"github.com/ignite/cli/v28/ignite/pkg/cosmosclient"

	cschema "github.com/verana-labs/verana/x/cs/types"
	permtypes "github.com/verana-labs/verana/x/pp/types"

	"github.com/verana-labs/verana/testharness/lib"
)

// RunPermissionAuthzOperationsJourney implements Journey 302: Test StartPermissionVP and RenewPermissionVP
// with operator authorization. For each operation: (a) try without auth -> fail, (b) grant auth, (c) try with auth -> succeed.
// Depends on Journey 301 (setup) having been run first.
func RunPermissionAuthzOperationsJourney(ctx context.Context, client cosmosclient.Client) error {
	fmt.Println("Starting Journey 302: Perm Operations with Operator Authorization (fail-then-pass)")

	// Load results from Journey 301
	setup := lib.LoadJourneyResult("journey301")
	policyAddr := setup.GroupPolicyAddr
	operatorAccount := lib.GetAccount(client, permOperatorName)
	operatorAddr := setup.OperatorAddr
	adminAccount := lib.GetAccount(client, permGroupAdminName)
	member1Account := lib.GetAccount(client, permGroupMember1Name)

	fmt.Printf("  Group Policy: %s\n", policyAddr)
	fmt.Printf("  Operator:     %s\n", operatorAddr)

	// =========================================================================
	// PREREQUISITES: Create TR, CS, and Root Permission
	// The operator grants itself a self-delegation, then creates TR, CS, and
	// Root Permission directly (controller = operatorAddr). This avoids complex
	// group proposal chains for prerequisite setup. The actual authz tests
	// below focus on StartPermissionVP and RenewPermissionVP.
	// =========================================================================
	fmt.Println("\n=== PREREQUISITES: Create TR, CS, Root Permission ===")

	// --- Prerequisite 1: Grant operator authorization from the Corporation ---
	// Spec v4 (AUTHZ-CHECK-5): the operator acts on behalf of the registered
	// Corporation (policyAddr); authz is granted by the corp's group, not via
	// self-delegation (a plain account is not a registered Corporation). Grants
	// are in-place replacements, so each later test grant re-includes this set.
	prereqMsgTypes := []string{
		"/verana.ec.v1.MsgCreateEcosystem",
		"/verana.cs.v1.MsgCreateCredentialSchema",
		"/verana.pp.v1.MsgSetParticipantOPToValidated",
		"/verana.pp.v1.MsgCreateRootParticipant",
		"/verana.pp.v1.MsgSetParticipantEffectiveUntil",
	}
	fmt.Println("\n--- Prerequisite 1: Grant operator authz via the Corporation group ---")
	err := lib.GrantOperatorAuthorizationViaGroup(
		client, ctx, adminAccount, member1Account,
		policyAddr, operatorAddr, operatorAddr, prereqMsgTypes,
	)
	if err != nil {
		return fmt.Errorf("prerequisite 1 failed: %w", err)
	}
	fmt.Println("OK Prerequisite 1: Granted operator authz from Corporation for prereq operations")
	waitForTx("operator authz grant")

	// --- Prerequisite 2: Create Trust Registry (controller = operatorAddr) ---
	fmt.Println("\n--- Prerequisite 2: Create Trust Registry ---")
	did := lib.GenerateUniqueDID(client, ctx)
	trIDStr, err := lib.CreateEcosystemWithAuthority(
		client, ctx, operatorAccount, policyAddr,
		did,
		"https://perm-test.com/governance-framework.pdf",
		"sha384-MzNNbQTWCSUSi0bbz7dbua+RcENv7C6FvlmYJ1Y+I727HsPOHdzwELMYO9Mz68M26",
		"en",
	)
	if err != nil {
		return fmt.Errorf("prerequisite 2 failed: %w", err)
	}
	trID, _ := strconv.ParseUint(trIDStr, 10, 64)
	fmt.Printf("OK Prerequisite 2: Trust Registry created with ID: %d, DID: %s\n", trID, did)
	waitForTx("TR creation")

	// --- Prerequisite 3: Create Credential Schema with GRANTOR_VALIDATION modes ---
	fmt.Println("\n--- Prerequisite 3: Create Credential Schema ---")
	schemaData := lib.GenerateSimpleSchema(trIDStr)
	csIDStr, err := lib.CreateCredentialSchemaWithAuthority(
		client, ctx, operatorAccount, policyAddr,
		trID, schemaData,
		cschema.IssuerOnboardingMode_ISSUER_ONBOARDING_MODE_GRANTOR_VALIDATION_PROCESS,
		cschema.VerifierOnboardingMode_VERIFIER_ONBOARDING_MODE_GRANTOR_VALIDATION_PROCESS,
	)
	if err != nil {
		return fmt.Errorf("prerequisite 3 failed: %w", err)
	}
	csID, _ := strconv.ParseUint(csIDStr, 10, 64)
	fmt.Printf("OK Prerequisite 3: Credential Schema created with ID: %d\n", csID)
	waitForTx("CS creation")

	// --- Prerequisite 4: Create Root Permission (ECOSYSTEM type) ---
	// Operator is the TR controller, so creator=operatorAddr matches.
	fmt.Println("\n--- Prerequisite 4: Create Root Permission (ECOSYSTEM type) ---")
	rootPermDID := lib.GenerateUniqueDID(client, ctx)
	effectiveFrom := time.Now().Add(10 * time.Second)
	effectiveUntil := effectiveFrom.Add(360 * 24 * time.Hour)
	rootPermID, err := lib.CreateRootPermissionWithAuthority(
		client, ctx, operatorAccount, policyAddr,
		csID, rootPermDID, &effectiveFrom, &effectiveUntil, 0, 0, 0,
	)
	if err != nil {
		return fmt.Errorf("prerequisite 4 failed: %w", err)
	}
	fmt.Printf("OK Prerequisite 4: Root Permission created with ID: %d\n", rootPermID)
	waitForTx("Root perm creation")

	// Verify root permission
	rootPerm, err := lib.GetParticipant(client, ctx, rootPermID)
	if err != nil {
		return fmt.Errorf("prerequisite 4 verification failed: %w", err)
	}
	fmt.Printf("  Root Permission type: %s, schema_id: %d, op_state: %s\n",
		rootPerm.Role.String(), rootPerm.SchemaId, rootPerm.OpState.String())

	// =========================================================================
	// TEST 1: StartPermissionVP
	// =========================================================================
	fmt.Println("\n=== TEST 1: StartPermissionVP ===")

	startPermDID := lib.GenerateUniqueDID(client, ctx)

	// 1a: Try WITHOUT authorization (expect failure)
	fmt.Println("\n--- Step 1a: Operator tries StartPermissionVP without auth (expect failure) ---")
	_, err = lib.StartPermissionVPWithAuthority(
		client, ctx, operatorAccount, policyAddr,
		permtypes.ParticipantRole_ISSUER_GRANTOR,
		rootPermID,
		startPermDID,
	)
	if err := expectAuthorizationError("Step 1a", err); err != nil {
		return err
	}
	waitForTx("StartPermVP rejection")

	// 1b: Grant authorization for StartPermissionVP
	fmt.Println("\n--- Step 1b: Grant operator auth for StartPermissionVP ---")
	err = lib.GrantOperatorAuthorizationViaGroup(
		client, ctx, adminAccount, member1Account,
		policyAddr, operatorAddr, operatorAddr,
		append(append([]string{}, prereqMsgTypes...), "/verana.pp.v1.MsgStartParticipantOP"),
	)
	if err != nil {
		return fmt.Errorf("step 1b failed: %w", err)
	}
	fmt.Println("OK Step 1b: Granted StartPermissionVP authorization")
	waitForTx("grant StartPermVP auth")

	// 1c: Try WITH authorization (expect success)
	fmt.Println("\n--- Step 1c: Operator starts permission VP with auth (expect success) ---")
	permIDStr, err := lib.StartPermissionVPWithAuthority(
		client, ctx, operatorAccount, policyAddr,
		permtypes.ParticipantRole_ISSUER_GRANTOR,
		rootPermID,
		startPermDID,
	)
	if err != nil {
		return fmt.Errorf("step 1c failed: %w", err)
	}
	permID, _ := strconv.ParseUint(permIDStr, 10, 64)
	fmt.Printf("OK Step 1c: StartPermissionVP succeeded, permission ID: %d\n", permID)
	waitForTx("StartPermVP success")

	// Verify the permission was created in PENDING state
	perm, err := lib.GetParticipant(client, ctx, permID)
	if err != nil {
		return fmt.Errorf("step 1c verification query failed: %w", err)
	}
	if perm.OpState != permtypes.OnboardingState_PENDING {
		return fmt.Errorf("step 1c verification failed: expected PENDING state, got %s", perm.OpState.String())
	}
	if perm.CorporationId == 0 {
		return fmt.Errorf("step 1c verification failed: expected authority=%s, got %d", policyAddr, perm.CorporationId)
	}
	fmt.Printf("OK Step 1c: Verified permission is PENDING, authority=%d\n", perm.CorporationId)

	// =========================================================================
	// TEST 2: RenewPermissionVP
	// To test renewal, we first need to set the permission to VALIDATED state.
	// The validator (root permission authority = operatorAddr) validates it.
	// =========================================================================
	fmt.Println("\n=== TEST 2: RenewPermissionVP ===")

	// 2-prereq: Validate the permission (set to VALIDATED state)
	fmt.Println("\n--- Step 2-prereq: Validate the permission (set op_state=VALIDATED) ---")
	_, err = lib.SetPermissionVPToValidated(client, ctx, operatorAccount, permtypes.MsgSetParticipantOPToValidated{
		Corporation:    policyAddr,
		Id:             permID,
		ValidationFees: 0,
		IssuanceFees:   0,
	})
	if err != nil {
		return fmt.Errorf("step 2-prereq failed: %w", err)
	}
	waitForTx("SetPermissionVPToValidated")

	// Verify the permission is now VALIDATED
	perm, err = lib.GetParticipant(client, ctx, permID)
	if err != nil {
		return fmt.Errorf("step 2-prereq verification failed: %w", err)
	}
	if perm.OpState != permtypes.OnboardingState_VALIDATED {
		return fmt.Errorf("step 2-prereq verification failed: expected VALIDATED state, got %s", perm.OpState.String())
	}
	fmt.Printf("OK Step 2-prereq: Permission is now VALIDATED\n")

	// 2a: Try WITHOUT authorization (expect failure)
	fmt.Println("\n--- Step 2a: Operator tries RenewPermissionVP without auth (expect failure) ---")
	err = lib.RenewPermissionVPWithAuthority(
		client, ctx, operatorAccount, policyAddr,
		permID,
	)
	if err := expectAuthorizationError("Step 2a", err); err != nil {
		return err
	}
	waitForTx("RenewPermVP rejection")

	// 2b: Grant authorization for RenewPermissionVP
	fmt.Println("\n--- Step 2b: Grant operator auth for RenewPermissionVP ---")
	err = lib.GrantOperatorAuthorizationViaGroup(
		client, ctx, adminAccount, member1Account,
		policyAddr, operatorAddr, operatorAddr,
		append(append([]string{}, prereqMsgTypes...), "/verana.pp.v1.MsgStartParticipantOP", "/verana.pp.v1.MsgRenewParticipantOP"),
	)
	if err != nil {
		return fmt.Errorf("step 2b failed: %w", err)
	}
	fmt.Println("OK Step 2b: Granted RenewPermissionVP authorization")
	waitForTx("grant RenewPermVP auth")

	// 2c: Try WITH authorization (expect success)
	fmt.Println("\n--- Step 2c: Operator renews permission VP with auth (expect success) ---")
	err = lib.RenewPermissionVPWithAuthority(
		client, ctx, operatorAccount, policyAddr,
		permID,
	)
	if err != nil {
		return fmt.Errorf("step 2c failed: %w", err)
	}
	fmt.Println("OK Step 2c: RenewPermissionVP succeeded")
	waitForTx("RenewPermVP success")

	// Verify the permission is back to PENDING state
	perm, err = lib.GetParticipant(client, ctx, permID)
	if err != nil {
		return fmt.Errorf("step 2c verification query failed: %w", err)
	}
	if perm.OpState != permtypes.OnboardingState_PENDING {
		return fmt.Errorf("step 2c verification failed: expected PENDING state after renewal, got %s", perm.OpState.String())
	}
	fmt.Printf("OK Step 2c: Verified permission is PENDING after renewal\n")

	// =========================================================================
	// TEST 3: Unauthorized operator (negative test)
	// =========================================================================
	fmt.Println("\n=== TEST 3: Unauthorized operator (negative test) ===")

	// First, validate the permission again so we can test renewal rejection
	_, err = lib.SetPermissionVPToValidated(client, ctx, operatorAccount, permtypes.MsgSetParticipantOPToValidated{
		Corporation: policyAddr,
		Id:          permID,
	})
	if err != nil {
		return fmt.Errorf("step 3-prereq failed: %w", err)
	}
	waitForTx("Re-validate for test 3")

	fmt.Println("\n--- Step 3a: Unauthorized operator tries StartPermissionVP (expect failure) ---")
	cooluser := lib.GetAccount(client, lib.COOLUSER_NAME)
	unauthorizedDID := lib.GenerateUniqueDID(client, ctx)
	_, err = lib.StartPermissionVPWithAuthority(
		client, ctx, cooluser, policyAddr,
		permtypes.ParticipantRole_ISSUER_GRANTOR,
		rootPermID,
		unauthorizedDID,
	)
	if err := expectAuthorizationError("Step 3a", err); err != nil {
		return err
	}

	fmt.Println("\n--- Step 3b: Unauthorized operator tries RenewPermissionVP (expect failure) ---")
	err = lib.RenewPermissionVPWithAuthority(
		client, ctx, cooluser, policyAddr,
		permID,
	)
	if err := expectAuthorizationError("Step 3b", err); err != nil {
		return err
	}

	// Save results
	result := lib.JourneyResult{
		EcosystemID:     trIDStr,
		SchemaID:        csIDStr,
		DID:             did,
		PermissionID:    permIDStr,
		GroupID:         setup.GroupID,
		GroupPolicyAddr: policyAddr,
		OperatorAddr:    operatorAddr,
	}
	lib.SaveJourneyResult("journey302", result)

	fmt.Println("\n========================================")
	fmt.Println("Journey 302 completed successfully!")
	fmt.Println("StartPermissionVP and RenewPermissionVP tested: fail without auth, pass with auth.")
	fmt.Println("========================================")

	return nil
}
