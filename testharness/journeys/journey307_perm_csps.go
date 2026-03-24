package journeys

import (
	"context"
	"fmt"
	"strconv"
	"time"

	"github.com/google/uuid"
	"github.com/ignite/cli/v28/ignite/pkg/cosmosclient"

	cschema "github.com/verana-labs/verana/x/cs/types"
	permtypes "github.com/verana-labs/verana/x/perm/types"

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

	vsOperatorAccount := lib.GetAccount(client, "cooluser")
	vsOperatorAddr, _ := vsOperatorAccount.Address("verana")

	fmt.Printf("  Operator: %s\n", operatorAddr)
	fmt.Printf("  VS Operator: %s\n", vsOperatorAddr)

	trID, _ := strconv.ParseUint(setup302.TrustRegistryID, 10, 64)

	// =========================================================================
	// PREREQUISITES: All operations use operatorAddr as authority (self-delegation)
	// to avoid mutual exclusivity between OperatorAuthorization and
	// VSOperatorAuthorization in the DE module.
	// =========================================================================
	fmt.Println("\n=== PREREQUISITES: Create CS, root perm, and ISSUER perms ===")

	// --- Prerequisite 1: Grant self-delegation for direct operations ---
	fmt.Println("\n--- Prerequisite 1: Grant self-delegation ---")
	err := lib.GrantSelfDelegation(client, ctx, operatorAccount, []string{
		"/verana.cs.v1.MsgCreateCredentialSchema",
		"/verana.perm.v1.MsgSetPermissionVPToValidated",
		"/verana.perm.v1.MsgCreateRootPermission",
		"/verana.perm.v1.MsgStartPermissionVP",
	})
	if err != nil {
		return fmt.Errorf("prerequisite 1 failed: %w", err)
	}
	fmt.Println("OK Prerequisite 1: Granted self-delegation")
	waitForTx("self-delegation")

	// --- Prerequisite 2: Create CS for ISSUER perm ---
	fmt.Println("\n--- Prerequisite 2: Create Credential Schema (CS1) ---")
	schemaData := lib.GenerateSimpleSchema(setup302.TrustRegistryID)
	cs1IDStr, err := lib.CreateCredentialSchema(client, ctx, operatorAccount, cschema.MsgCreateCredentialSchema{
		TrId:                                    trID,
		JsonSchema:                              schemaData,
		IssuerPermManagementMode:                uint32(cschema.CredentialSchemaPermManagementMode_ECOSYSTEM),
		VerifierPermManagementMode:              uint32(cschema.CredentialSchemaPermManagementMode_ECOSYSTEM),
		PricingAssetType:                        uint32(cschema.PricingAssetType_TU),
		PricingAsset:                            "tu",
		DigestAlgorithm:                         "sha256",
		IssuerGrantorValidationValidityPeriod:   &cschema.OptionalUInt32{Value: 0},
		VerifierGrantorValidationValidityPeriod: &cschema.OptionalUInt32{Value: 0},
		IssuerValidationValidityPeriod:          &cschema.OptionalUInt32{Value: 0},
		VerifierValidationValidityPeriod:        &cschema.OptionalUInt32{Value: 0},
		HolderValidationValidityPeriod:          &cschema.OptionalUInt32{Value: 0},
	})
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
	rootPermIDStr, err := lib.CreateRootPermission(client, ctx, operatorAccount, permtypes.MsgCreateRootPermission{
		SchemaId:       cs1ID,
		Did:            rootPermDID,
		EffectiveFrom:  &effectiveFrom,
		EffectiveUntil: &effectiveUntil,
	})
	if err != nil {
		return fmt.Errorf("prerequisite 3 failed: could not create root permission: %w", err)
	}
	rootPermID, _ := strconv.ParseUint(rootPermIDStr, 10, 64)
	fmt.Printf("OK Prerequisite 3: Root permission created with ID: %d\n", rootPermID)
	waitForTx("root perm creation")

	// Wait for root perm to become effective (block time may lag behind system clock)
	fmt.Println("  Waiting for root permission to become effective...")
	time.Sleep(15 * time.Second)

	// --- Prerequisite 4: Create ISSUER perm with vs_operator (authority=operatorAddr) ---
	fmt.Println("\n--- Prerequisite 4: Create ISSUER perm with vs_operator ---")
	issuerDID := lib.GenerateUniqueDID(client, ctx)
	issuerPermIDStr, err := lib.StartPermissionVP(client, ctx, operatorAccount, permtypes.MsgStartPermissionVP{
		// Authority defaults to operatorAddr
		Type:                   permtypes.PermissionType_ISSUER,
		ValidatorPermId:        rootPermID,
		Did:                    issuerDID,
		VsOperator:             vsOperatorAddr,
		VsOperatorAuthzEnabled: true,
	})
	if err != nil {
		return fmt.Errorf("prerequisite 4 failed: could not start issuer perm VP: %w", err)
	}
	issuerPermID, _ := strconv.ParseUint(issuerPermIDStr, 10, 64)
	fmt.Printf("OK Prerequisite 4: ISSUER perm started with ID: %d (vs_operator=%s)\n", issuerPermID, vsOperatorAddr)
	waitForTx("issuer perm start")

	// --- Prerequisite 5: Validate ISSUER perm (grants VS operator auth) ---
	fmt.Println("\n--- Prerequisite 5: Validate ISSUER perm (grants VS operator auth) ---")
	_, err = lib.SetPermissionVPToValidated(client, ctx, operatorAccount, permtypes.MsgSetPermissionVPToValidated{
		Id: issuerPermID,
	})
	if err != nil {
		return fmt.Errorf("prerequisite 5 failed: could not validate issuer perm: %w", err)
	}
	fmt.Printf("OK Prerequisite 5: ISSUER perm %d validated (VS operator auth granted)\n", issuerPermID)
	waitForTx("validate issuer perm")

	// Verify issuer perm is VALIDATED
	issuerPerm, err := lib.GetPermission(client, ctx, issuerPermID)
	if err != nil {
		return fmt.Errorf("prerequisite verification failed: %w", err)
	}
	if issuerPerm.VpState != permtypes.ValidationState_VALIDATED {
		return fmt.Errorf("prerequisite verification failed: expected VALIDATED, got %s", issuerPerm.VpState.String())
	}
	fmt.Printf("  Verified: ISSUER perm is VALIDATED, vs_operator=%s, vs_operator_authz_enabled=%v\n",
		issuerPerm.VsOperator, issuerPerm.VsOperatorAuthzEnabled)

	// --- Prerequisite 6: Create CS2 + root2 + agent perm (authority=operatorAddr) ---
	// Use a second CS to avoid overlap with the issuer perm on CS1
	fmt.Println("\n--- Prerequisite 6: Create CS2 for agent perm ---")
	schemaData2 := lib.GenerateSimpleSchema(setup302.TrustRegistryID)
	cs2IDStr, err := lib.CreateCredentialSchema(client, ctx, operatorAccount, cschema.MsgCreateCredentialSchema{
		TrId:                                    trID,
		JsonSchema:                              schemaData2,
		IssuerPermManagementMode:                uint32(cschema.CredentialSchemaPermManagementMode_ECOSYSTEM),
		VerifierPermManagementMode:              uint32(cschema.CredentialSchemaPermManagementMode_ECOSYSTEM),
		PricingAssetType:                        uint32(cschema.PricingAssetType_TU),
		PricingAsset:                            "tu",
		DigestAlgorithm:                         "sha256",
		IssuerGrantorValidationValidityPeriod:   &cschema.OptionalUInt32{Value: 0},
		VerifierGrantorValidationValidityPeriod: &cschema.OptionalUInt32{Value: 0},
		IssuerValidationValidityPeriod:          &cschema.OptionalUInt32{Value: 0},
		VerifierValidationValidityPeriod:        &cschema.OptionalUInt32{Value: 0},
		HolderValidationValidityPeriod:          &cschema.OptionalUInt32{Value: 0},
	})
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
	rootPerm2IDStr, err := lib.CreateRootPermission(client, ctx, operatorAccount, permtypes.MsgCreateRootPermission{
		SchemaId:       cs2ID,
		Did:            rootPerm2DID,
		EffectiveFrom:  &effectiveFrom2,
		EffectiveUntil: &effectiveUntil2,
	})
	if err != nil {
		return fmt.Errorf("prerequisite 6b failed: could not create root perm on CS2: %w", err)
	}
	rootPerm2ID, _ := strconv.ParseUint(rootPerm2IDStr, 10, 64)
	fmt.Printf("OK Prerequisite 6b: Root perm 2 created with ID: %d\n", rootPerm2ID)
	waitForTx("root perm 2 creation")

	// Wait for root perm 2 to become effective
	fmt.Println("  Waiting for root perm 2 to become effective...")
	time.Sleep(15 * time.Second)

	fmt.Println("\n--- Prerequisite 6c: Create agent ISSUER perm on CS2 ---")
	agentDID := lib.GenerateUniqueDID(client, ctx)
	agentPermIDStr, err := lib.StartPermissionVP(client, ctx, operatorAccount, permtypes.MsgStartPermissionVP{
		Type:            permtypes.PermissionType_ISSUER,
		ValidatorPermId: rootPerm2ID,
		Did:             agentDID,
	})
	if err != nil {
		return fmt.Errorf("prerequisite 6c failed: could not start agent perm VP: %w", err)
	}
	agentPermID, _ := strconv.ParseUint(agentPermIDStr, 10, 64)
	fmt.Printf("OK Prerequisite 6c: Agent perm started with ID: %d\n", agentPermID)
	waitForTx("agent perm start")

	fmt.Println("\n--- Prerequisite 6d: Validate agent perm ---")
	_, err = lib.SetPermissionVPToValidated(client, ctx, operatorAccount, permtypes.MsgSetPermissionVPToValidated{
		Id: agentPermID,
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

	// 1a: Unauthorized operator (controllerB) tries CSPS (expect failure)
	fmt.Println("\n--- Step 1a: Unauthorized operator tries CSPS (expect failure) ---")
	controllerB := lib.GetAccount(client, "controllerB")
	err = lib.CreatePermissionSession(
		client, ctx, controllerB, operatorAddr,
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
		client, ctx, vsOperatorAccount, operatorAddr,
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
		client, ctx, vsOperatorAccount, operatorAddr,
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
		TrustRegistryID: setup302.TrustRegistryID,
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
