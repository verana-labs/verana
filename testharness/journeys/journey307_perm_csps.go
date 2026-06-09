package journeys

import (
	"context"
	"fmt"
	"strconv"
	"time"

	"github.com/google/uuid"
	"github.com/ignite/cli/v28/ignite/pkg/cosmosclient"

	cschema "github.com/verana-labs/verana/x/cs/types"
	permtypes "github.com/verana-labs/verana/x/pp/types"

	"github.com/verana-labs/verana/testharness/lib"
)

// RunPermissionCSPSJourney implements Journey 307: Test CreateOrUpdatePermissionSession (CSPS)
// with VS Operator Authorization.
//
// This journey uses the operator's own address as authority (not the group policy)
// to avoid mutual exclusivity conflicts between OperatorAuthorization and
// VSOperatorAuthorization in the DE module.
//
// TEST 1: CreateOrUpdatePermissionSession (unauthorized operator → fail, authorized → succeed)
// TEST 2: Verify session fields
// TEST 3: Update existing session
// TEST 4: Negative tests (wrong authority)
// Depends on Journey 301 (setup), 302 (group/operator), 304 (root permission).
func RunPermissionCSPSJourney(ctx context.Context, client cosmosclient.Client) error {
	fmt.Println("Starting Journey 307: CreateOrUpdatePermissionSession with VS Operator Authorization")

	// Load results from prior journeys
	// Use journey302's TR (controller = operatorAddr) since we use self-delegation
	setup302 := lib.LoadJourneyResult("journey302")
	operatorAccount := lib.GetAccount(client, permOperatorName)
	operatorAddr := setup302.OperatorAddr
	// Corp pattern (spec v4 / AUTHZ-CHECK-5): the operator acts on behalf of the
	// registered Corporation (policyAddr from journey302); the vs_operator
	// (cooluser) is a distinct account, so no OperatorAuthorization/VSOA mutual
	// exclusivity conflict.
	policyAddr := setup302.GroupPolicyAddr
	adminAccount := lib.GetAccount(client, permGroupAdminName)
	member1Account := lib.GetAccount(client, permGroupMember1Name)

	vsOperatorAccount := lib.GetAccount(client, "cooluser")
	vsOperatorAddr, _ := vsOperatorAccount.Address("verana")

	fmt.Printf("  Operator: %s\n", operatorAddr)
	fmt.Printf("  VS Operator: %s\n", vsOperatorAddr)

	trID, _ := strconv.ParseUint(setup302.EcosystemID, 10, 64)

	// =========================================================================
	// PREREQUISITES: All operations use operatorAddr as authority (self-delegation)
	// to avoid mutual exclusivity between OperatorAuthorization and
	// VSOperatorAuthorization in the DE module.
	// =========================================================================
	fmt.Println("\n=== PREREQUISITES: Create CS, root perm, and ISSUER perms ===")

	// --- Prerequisite 1: Grant operator authz from the Corporation group ---
	fmt.Println("\n--- Prerequisite 1: Grant operator authz via the Corporation group ---")
	err := lib.GrantOperatorAuthorizationViaGroup(
		client, ctx, adminAccount, member1Account,
		policyAddr, operatorAddr, operatorAddr,
		[]string{
			"/verana.cs.v1.MsgCreateCredentialSchema",
			"/verana.pp.v1.MsgSetParticipantOPToValidated",
			"/verana.pp.v1.MsgCreateRootParticipant",
			"/verana.pp.v1.MsgStartParticipantOP",
		},
	)
	if err != nil {
		return fmt.Errorf("prerequisite 1 failed: %w", err)
	}
	fmt.Println("OK Prerequisite 1: Granted operator authz from Corporation")
	waitForTx("operator authz grant")

	// --- Prerequisite 2: Create CS for ISSUER perm ---
	fmt.Println("\n--- Prerequisite 2: Create Credential Schema (CS1) ---")
	schemaData := lib.GenerateSimpleSchema(setup302.EcosystemID)
	cs1IDStr, err := lib.CreateCredentialSchemaWithAuthority(
		client, ctx, operatorAccount, policyAddr,
		trID, schemaData,
		cschema.IssuerOnboardingMode_ISSUER_ONBOARDING_MODE_ECOSYSTEM_VALIDATION_PROCESS,
		cschema.VerifierOnboardingMode_VERIFIER_ONBOARDING_MODE_ECOSYSTEM_VALIDATION_PROCESS,
	)
	if err != nil {
		return fmt.Errorf("prerequisite 2 failed: could not create CS1: %w", err)
	}
	cs1ID, _ := strconv.ParseUint(cs1IDStr, 10, 64)
	fmt.Printf("OK Prerequisite 2: CS1 created with ID: %d\n", cs1ID)
	waitForTx("CS1 creation")

	// --- Prerequisite 3: Create root permission on CS1 (authority=operatorAddr) ---
	fmt.Println("\n--- Prerequisite 3: Create root permission on CS1 ---")
	rootPermDID := lib.GenerateUniqueDID(client, ctx)
	effectiveFrom := time.Now().Add(5 * time.Second)
	effectiveUntil := effectiveFrom.Add(360 * 24 * time.Hour)
	rootPermID, err := lib.CreateRootPermissionWithAuthority(
		client, ctx, operatorAccount, policyAddr,
		cs1ID, rootPermDID, &effectiveFrom, &effectiveUntil, 0, 0, 0,
	)
	if err != nil {
		return fmt.Errorf("prerequisite 3 failed: could not create root permission: %w", err)
	}
	fmt.Printf("OK Prerequisite 3: Root permission created with ID: %d\n", rootPermID)
	waitForTx("root perm creation")

	// Wait for root perm to become effective (block time may lag behind system clock)
	fmt.Println("  Waiting for root permission to become effective...")
	time.Sleep(15 * time.Second)

	// --- Prerequisite 4: Create ISSUER perm with vs_operator (authority=operatorAddr) ---
	fmt.Println("\n--- Prerequisite 4: Create ISSUER perm with vs_operator ---")
	issuerDID := lib.GenerateUniqueDID(client, ctx)
	issuerPermIDStr, err := lib.StartPermissionVP(client, ctx, operatorAccount, permtypes.MsgStartParticipantOP{
		Corporation:             policyAddr,
		Role:                    permtypes.ParticipantRole_ISSUER,
		ValidatorParticipantId:  rootPermID,
		Did:                     issuerDID,
		VsOperator:              vsOperatorAddr,
		VsOperatorAuthzMsgTypes: []string{permtypes.MsgCreateOrUpdateParticipantSessionTypeURL},
	})
	if err != nil {
		return fmt.Errorf("prerequisite 4 failed: could not start issuer perm VP: %w", err)
	}
	issuerPermID, _ := strconv.ParseUint(issuerPermIDStr, 10, 64)
	fmt.Printf("OK Prerequisite 4: ISSUER perm started with ID: %d (vs_operator=%s)\n", issuerPermID, vsOperatorAddr)
	waitForTx("issuer perm start")

	// --- Prerequisite 5: Validate ISSUER perm (activates the VSOA record) ---
	// Spec v4-rc2: the disabled record created at MSG-1 is activated at MSG-3 by
	// setting record.expiration = effective_until (via MOD-DE-MSG-9). Without an
	// effective_until the record stays disabled and AUTHZ-CHECK-3 would reject CSPS.
	fmt.Println("\n--- Prerequisite 5: Validate ISSUER perm (activates VS operator auth) ---")
	issuerEffUntil := time.Now().Add(365 * 24 * time.Hour)
	_, err = lib.SetPermissionVPToValidated(client, ctx, operatorAccount, permtypes.MsgSetParticipantOPToValidated{
		Corporation:    policyAddr,
		Id:             issuerPermID,
		EffectiveUntil: &issuerEffUntil,
	})
	if err != nil {
		return fmt.Errorf("prerequisite 5 failed: could not validate issuer perm: %w", err)
	}
	fmt.Printf("OK Prerequisite 5: ISSUER perm %d validated (VS operator auth granted)\n", issuerPermID)
	waitForTx("validate issuer perm")

	// Verify issuer perm is VALIDATED
	issuerPerm, err := lib.GetParticipant(client, ctx, issuerPermID)
	if err != nil {
		return fmt.Errorf("prerequisite verification failed: %w", err)
	}
	if issuerPerm.OpState != permtypes.OnboardingState_VALIDATED {
		return fmt.Errorf("prerequisite verification failed: expected VALIDATED, got %s", issuerPerm.OpState.String())
	}
	fmt.Printf("  Verified: ISSUER perm is VALIDATED, vs_operator=%s\n", issuerPerm.VsOperator)

	// --- Prerequisite 6: Create CS2 + root2 + agent perm (authority=operatorAddr) ---
	// Use a second CS to avoid overlap with the issuer perm on CS1
	fmt.Println("\n--- Prerequisite 6: Create CS2 for agent perm ---")
	schemaData2 := lib.GenerateSimpleSchema(setup302.EcosystemID)
	cs2IDStr, err := lib.CreateCredentialSchemaWithAuthority(
		client, ctx, operatorAccount, policyAddr,
		trID, schemaData2,
		cschema.IssuerOnboardingMode_ISSUER_ONBOARDING_MODE_ECOSYSTEM_VALIDATION_PROCESS,
		cschema.VerifierOnboardingMode_VERIFIER_ONBOARDING_MODE_ECOSYSTEM_VALIDATION_PROCESS,
	)
	if err != nil {
		return fmt.Errorf("prerequisite 6 failed: could not create CS2: %w", err)
	}
	cs2ID, _ := strconv.ParseUint(cs2IDStr, 10, 64)
	fmt.Printf("OK Prerequisite 6: CS2 created with ID: %d\n", cs2ID)
	waitForTx("CS2 creation")

	fmt.Println("\n--- Prerequisite 6b: Create root perm on CS2 ---")
	rootPerm2DID := lib.GenerateUniqueDID(client, ctx)
	effectiveFrom2 := time.Now().Add(5 * time.Second)
	effectiveUntil2 := effectiveFrom2.Add(360 * 24 * time.Hour)
	rootPerm2ID, err := lib.CreateRootPermissionWithAuthority(
		client, ctx, operatorAccount, policyAddr,
		cs2ID, rootPerm2DID, &effectiveFrom2, &effectiveUntil2, 0, 0, 0,
	)
	if err != nil {
		return fmt.Errorf("prerequisite 6b failed: could not create root perm on CS2: %w", err)
	}
	fmt.Printf("OK Prerequisite 6b: Root perm 2 created with ID: %d\n", rootPerm2ID)
	waitForTx("root perm 2 creation")

	// Wait for root perm 2 to become effective
	fmt.Println("  Waiting for root perm 2 to become effective...")
	time.Sleep(15 * time.Second)

	fmt.Println("\n--- Prerequisite 6c: Create agent ISSUER perm on CS2 ---")
	agentDID := lib.GenerateUniqueDID(client, ctx)
	agentPermIDStr, err := lib.StartPermissionVP(client, ctx, operatorAccount, permtypes.MsgStartParticipantOP{
		Corporation:            policyAddr,
		Role:                   permtypes.ParticipantRole_ISSUER,
		ValidatorParticipantId: rootPerm2ID,
		Did:                    agentDID,
	})
	if err != nil {
		return fmt.Errorf("prerequisite 6c failed: could not start agent perm VP: %w", err)
	}
	agentPermID, _ := strconv.ParseUint(agentPermIDStr, 10, 64)
	fmt.Printf("OK Prerequisite 6c: Agent perm started with ID: %d\n", agentPermID)
	waitForTx("agent perm start")

	fmt.Println("\n--- Prerequisite 6d: Validate agent perm ---")
	_, err = lib.SetPermissionVPToValidated(client, ctx, operatorAccount, permtypes.MsgSetParticipantOPToValidated{
		Corporation: policyAddr,
		Id:          agentPermID,
	})
	if err != nil {
		return fmt.Errorf("prerequisite 6d failed: could not validate agent perm: %w", err)
	}
	fmt.Printf("OK Prerequisite 6d: Agent perm %d validated\n", agentPermID)
	waitForTx("validate agent perm")

	// Use the same agent perm for wallet_agent role (handler allows this)
	walletAgentPermID := agentPermID

	fmt.Println("\n=== Prerequisites complete ===")
	fmt.Printf("  Issuer perm ID:       %d (authority=%s, vs_operator=%s)\n", issuerPermID, operatorAddr, vsOperatorAddr)
	fmt.Printf("  Agent perm ID:        %d (authority=%s)\n", agentPermID, operatorAddr)
	fmt.Printf("  Wallet agent perm ID: %d (same as agent)\n", walletAgentPermID)

	// =========================================================================
	// TEST 1: CreateOrUpdatePermissionSession
	// (fail with unauthorized operator, succeed with authorized operator)
	// =========================================================================
	fmt.Println("\n=== TEST 1: CreateOrUpdatePermissionSession ===")

	sessionID := uuid.New().String()

	// 1a: Unauthorized operator (ec_operator) tries CSPS (expect failure)
	// Note: cooluser is the vs_operator (authorized), so we use ec_operator instead.
	fmt.Println("\n--- Step 1a: Unauthorized operator tries CSPS (expect failure) ---")
	// A funded account with no VS-operator authorization (guaranteed present from
	// journey301's group setup) serves as the unauthorized signer.
	unauthorizedAccount := lib.GetAccount(client, permGroupMember2Name)
	err = lib.CreatePermissionSession(
		client, ctx, unauthorizedAccount, policyAddr,
		sessionID, issuerPermID, 0, agentPermID, walletAgentPermID,
	)
	if err := expectAuthorizationError("Step 1a", err); err != nil {
		return err
	}
	fmt.Println("OK Step 1a: Unauthorized operator correctly rejected")
	waitForTx("CSPS rejection")

	// 1b: Authorized vs_operator tries CSPS (expect success — VS operator auth was granted during validation)
	fmt.Println("\n--- Step 1b: Authorized vs_operator creates permission session (expect success) ---")
	err = lib.CreatePermissionSession(
		client, ctx, vsOperatorAccount, policyAddr,
		sessionID, issuerPermID, 0, agentPermID, walletAgentPermID,
	)
	if err != nil {
		return fmt.Errorf("step 1b failed: %w", err)
	}
	fmt.Printf("OK Step 1b: CreateOrUpdatePermissionSession succeeded (session_id=%s)\n", sessionID)
	waitForTx("CSPS success")

	// =========================================================================
	// TEST 2: Verify session fields
	// =========================================================================
	fmt.Println("\n=== TEST 2: Verify session fields ===")
	verified := lib.VerifyPermissionSession(
		client, ctx, sessionID,
		operatorAddr, agentPermID, issuerPermID, 0,
	)
	if !verified {
		return fmt.Errorf("step 2 failed: session verification failed")
	}
	fmt.Println("OK Step 2: Session fields verified")

	// =========================================================================
	// TEST 3: Update existing session (add a second record by calling again)
	// =========================================================================
	fmt.Println("\n=== TEST 3: Update existing session ===")
	err = lib.CreatePermissionSession(
		client, ctx, vsOperatorAccount, policyAddr,
		sessionID, issuerPermID, 0, agentPermID, walletAgentPermID,
	)
	if err != nil {
		return fmt.Errorf("step 3 failed: %w", err)
	}
	fmt.Printf("OK Step 3: Session updated (session_id=%s)\n", sessionID)
	waitForTx("CSPS update")

	// Verify the session still has correct fields after update
	verified = lib.VerifyPermissionSession(
		client, ctx, sessionID,
		operatorAddr, agentPermID, issuerPermID, 0,
	)
	if !verified {
		return fmt.Errorf("step 3 verification failed: session verification after update failed")
	}
	fmt.Println("OK Step 3: Updated session verified")

	// =========================================================================
	// TEST 4: Negative tests
	// =========================================================================
	fmt.Println("\n=== TEST 4: Negative tests ===")

	// 4a: Wrong authority
	fmt.Println("\n--- Step 4a: Wrong authority (expect failure) ---")
	wrongSessionID := uuid.New().String()
	err = lib.CreatePermissionSession(
		client, ctx, vsOperatorAccount, vsOperatorAddr,
		wrongSessionID, issuerPermID, 0, agentPermID, walletAgentPermID,
	)
	if err == nil {
		return fmt.Errorf("step 4a failed: expected error for wrong authority but succeeded")
	}
	fmt.Printf("OK Step 4a: Wrong authority correctly rejected: %v\n", err)

	// Save results
	result := lib.JourneyResult{
		EcosystemID:     setup302.EcosystemID,
		SchemaID:        cs1IDStr,
		DID:             issuerDID,
		PermissionID:    strconv.FormatUint(issuerPermID, 10),
		GroupID:         setup302.GroupID,
		GroupPolicyAddr: setup302.GroupPolicyAddr,
		OperatorAddr:    operatorAddr,
	}
	lib.SaveJourneyResult("journey307", result)

	fmt.Println("\n========================================")
	fmt.Println("Journey 307 completed successfully!")
	fmt.Println("CreateOrUpdatePermissionSession tested:")
	fmt.Println("  - Unauthorized operator rejected")
	fmt.Println("  - Authorized operator succeeded (VS operator auth)")
	fmt.Println("  - Session fields verified")
	fmt.Println("  - Session update succeeded")
	fmt.Println("  - Wrong authority rejected")
	fmt.Println("========================================")

	return nil
}
