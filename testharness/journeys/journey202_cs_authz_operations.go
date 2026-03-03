package journeys

import (
	"context"
	"fmt"
	"strconv"

	"github.com/ignite/cli/v28/ignite/pkg/cosmosclient"

	cschema "github.com/verana-labs/verana/x/cs/types"

	"github.com/verana-labs/verana/testharness/lib"
)

// RunCredentialSchemaAuthzOperationsJourney implements Journey 202: Test all CS operations with operator authorization.
// For each of the 3 CS message types: (a) try without auth → fail, (b) grant auth, (c) try with auth → succeed.
// Depends on Journey 201 (setup) having been run first.
func RunCredentialSchemaAuthzOperationsJourney(ctx context.Context, client cosmosclient.Client) error {
	fmt.Println("Starting Journey 202: CS Operations with Operator Authorization (fail-then-pass)")

	// Load results from Journey 201
	setup := lib.LoadJourneyResult("journey201")
	policyAddr := setup.GroupPolicyAddr
	operatorAccount := lib.GetAccount(client, csOperatorName)
	operatorAddr := setup.OperatorAddr
	adminAccount := lib.GetAccount(client, csGroupAdminName)
	member1Account := lib.GetAccount(client, csGroupMember1Name)

	fmt.Printf("  Group Policy: %s\n", policyAddr)
	fmt.Printf("  Operator:     %s\n", operatorAddr)

	// =========================================================================
	// PREREQUISITE: Create Trust Registry with controller = group policy
	// To do this, we first grant TR create auth to the operator, then create the TR.
	// =========================================================================
	fmt.Println("\n=== PREREQUISITE: Create Trust Registry (controller = group policy) ===")

	// Grant TR create authorization to the operator
	fmt.Println("\n--- Prerequisite 1: Grant operator auth for CreateTrustRegistry ---")
	err := lib.GrantOperatorAuthorizationViaGroup(
		client, ctx, adminAccount, member1Account,
		policyAddr, operatorAddr, operatorAddr,
		[]string{"/verana.tr.v1.MsgCreateTrustRegistry"},
	)
	if err != nil {
		return fmt.Errorf("prerequisite 1 failed: %w", err)
	}
	fmt.Println("✅ Prerequisite 1: Granted CreateTrustRegistry authorization")
	waitForTx("grant TR create auth")

	// Create TR with controller = policyAddr
	fmt.Println("\n--- Prerequisite 2: Create Trust Registry with controller = group policy ---")
	did := lib.GenerateUniqueDID(client, ctx)
	trIDStr, err := lib.CreateTrustRegistryWithAuthority(
		client, ctx, operatorAccount, policyAddr,
		did,
		"http://cs-test-aka.com",
		"https://cs-test.com/governance-framework.pdf",
		"sha384-MzNNbQTWCSUSi0bbz7dbua+RcENv7C6FvlmYJ1Y+I727HsPOHdzwELMYO9Mz68M26",
		"en",
	)
	if err != nil {
		return fmt.Errorf("prerequisite 2 failed: %w", err)
	}
	trID, _ := strconv.ParseUint(trIDStr, 10, 64)
	fmt.Printf("✅ Prerequisite 2: Trust Registry created with ID: %d, DID: %s\n", trID, did)
	waitForTx("TR creation")

	// Verify TR creation
	verified := lib.VerifyTrustRegistry(client, ctx, trID, did)
	if !verified {
		return fmt.Errorf("prerequisite 2 verification failed: trust registry not found or DID mismatch")
	}

	// =========================================================================
	// TEST 1: CreateCredentialSchema
	// =========================================================================
	fmt.Println("\n=== TEST 1: CreateCredentialSchema ===")

	schemaData := lib.GenerateSimpleSchema(trIDStr)

	// 1a: Try WITHOUT authorization (expect failure)
	fmt.Println("\n--- Step 1a: Operator tries CreateCredentialSchema without auth (expect failure) ---")
	_, err = lib.CreateCredentialSchemaWithAuthority(
		client, ctx, operatorAccount, policyAddr,
		trID, schemaData,
		cschema.CredentialSchemaPermManagementMode_GRANTOR_VALIDATION,
		cschema.CredentialSchemaPermManagementMode_OPEN,
	)
	if err := expectAuthorizationError("Step 1a", err); err != nil {
		return err
	}
	waitForTx("CreateCS rejection")

	// 1b: Grant authorization for CreateCredentialSchema
	fmt.Println("\n--- Step 1b: Grant operator auth for CreateCredentialSchema ---")
	err = lib.GrantOperatorAuthorizationViaGroup(
		client, ctx, adminAccount, member1Account,
		policyAddr, operatorAddr, operatorAddr,
		[]string{"/verana.cs.v1.MsgCreateCredentialSchema"},
	)
	if err != nil {
		return fmt.Errorf("step 1b failed: %w", err)
	}
	fmt.Println("✅ Step 1b: Granted CreateCredentialSchema authorization")
	waitForTx("grant CreateCS auth")

	// 1c: Try WITH authorization (expect success)
	fmt.Println("\n--- Step 1c: Operator creates credential schema with auth (expect success) ---")
	csIDStr, err := lib.CreateCredentialSchemaWithAuthority(
		client, ctx, operatorAccount, policyAddr,
		trID, schemaData,
		cschema.CredentialSchemaPermManagementMode_GRANTOR_VALIDATION,
		cschema.CredentialSchemaPermManagementMode_OPEN,
	)
	if err != nil {
		return fmt.Errorf("step 1c failed: %w", err)
	}
	csID, _ := strconv.ParseUint(csIDStr, 10, 64)
	fmt.Printf("✅ Step 1c: Credential Schema created with ID: %d\n", csID)
	waitForTx("CS creation")

	// Verify CS creation
	verified = lib.VerifyCredentialSchema(client, ctx, csID, trID)
	if !verified {
		return fmt.Errorf("step 1c verification failed: credential schema not found or TR mismatch")
	}

	// =========================================================================
	// TEST 2: UpdateCredentialSchema
	// =========================================================================
	fmt.Println("\n=== TEST 2: UpdateCredentialSchema ===")

	// 2a: Try WITHOUT authorization (expect failure)
	fmt.Println("\n--- Step 2a: Operator tries UpdateCredentialSchema without auth (expect failure) ---")
	err = lib.UpdateCredentialSchemaWithAuthority(
		client, ctx, operatorAccount, policyAddr,
		csID, 365, 365, 180, 180, 90,
	)
	if err := expectAuthorizationError("Step 2a", err); err != nil {
		return err
	}
	waitForTx("UpdateCS rejection")

	// 2b: Grant authorization for UpdateCredentialSchema
	fmt.Println("\n--- Step 2b: Grant operator auth for UpdateCredentialSchema ---")
	err = lib.GrantOperatorAuthorizationViaGroup(
		client, ctx, adminAccount, member1Account,
		policyAddr, operatorAddr, operatorAddr,
		[]string{"/verana.cs.v1.MsgUpdateCredentialSchema"},
	)
	if err != nil {
		return fmt.Errorf("step 2b failed: %w", err)
	}
	fmt.Println("✅ Step 2b: Granted UpdateCredentialSchema authorization")
	waitForTx("grant UpdateCS auth")

	// 2c: Try WITH authorization (expect success)
	fmt.Println("\n--- Step 2c: Operator updates credential schema with auth (expect success) ---")
	err = lib.UpdateCredentialSchemaWithAuthority(
		client, ctx, operatorAccount, policyAddr,
		csID, 365, 365, 180, 180, 90,
	)
	if err != nil {
		return fmt.Errorf("step 2c failed: %w", err)
	}
	fmt.Println("✅ Step 2c: Successfully updated credential schema")
	waitForTx("UpdateCS success")

	// Verify update
	updatedSchema, err := lib.QueryCredentialSchema(client, ctx, csID)
	if err != nil {
		return fmt.Errorf("step 2c verification query failed: %w", err)
	}
	if updatedSchema.Schema.IssuerGrantorValidationValidityPeriod != 365 {
		return fmt.Errorf("step 2c verification failed: IssuerGrantorValidationValidityPeriod should be 365, got %d",
			updatedSchema.Schema.IssuerGrantorValidationValidityPeriod)
	}
	fmt.Printf("✅ Step 2c: Verified IssuerGrantorValidationValidityPeriod = %d days\n",
		updatedSchema.Schema.IssuerGrantorValidationValidityPeriod)

	// =========================================================================
	// TEST 3: ArchiveCredentialSchema
	// =========================================================================
	fmt.Println("\n=== TEST 3: ArchiveCredentialSchema ===")

	// 3a: Try WITHOUT authorization (expect failure)
	fmt.Println("\n--- Step 3a: Operator tries ArchiveCredentialSchema without auth (expect failure) ---")
	err = lib.ArchiveCredentialSchemaWithAuthority(
		client, ctx, operatorAccount, policyAddr,
		csID, true,
	)
	if err := expectAuthorizationError("Step 3a", err); err != nil {
		return err
	}
	waitForTx("ArchiveCS rejection")

	// 3b: Grant authorization for ArchiveCredentialSchema
	fmt.Println("\n--- Step 3b: Grant operator auth for ArchiveCredentialSchema ---")
	err = lib.GrantOperatorAuthorizationViaGroup(
		client, ctx, adminAccount, member1Account,
		policyAddr, operatorAddr, operatorAddr,
		[]string{"/verana.cs.v1.MsgArchiveCredentialSchema"},
	)
	if err != nil {
		return fmt.Errorf("step 3b failed: %w", err)
	}
	fmt.Println("✅ Step 3b: Granted ArchiveCredentialSchema authorization")
	waitForTx("grant ArchiveCS auth")

	// 3c: Try WITH authorization — archive (expect success)
	fmt.Println("\n--- Step 3c: Operator archives credential schema with auth (expect success) ---")
	err = lib.ArchiveCredentialSchemaWithAuthority(
		client, ctx, operatorAccount, policyAddr,
		csID, true,
	)
	if err != nil {
		return fmt.Errorf("step 3c failed: %w", err)
	}
	fmt.Println("✅ Step 3c: Credential schema archived")
	waitForTx("ArchiveCS success")

	// Verify archived state
	archivedSchema, err := lib.QueryCredentialSchema(client, ctx, csID)
	if err != nil {
		return fmt.Errorf("step 3c verification query failed: %w", err)
	}
	if archivedSchema.Schema.Archived == nil {
		return fmt.Errorf("step 3c verification failed: credential schema should be archived")
	}
	fmt.Println("✅ Step 3c: Verified credential schema is archived")

	// 3d: Unarchive (same msg type, already authorized — expect success)
	fmt.Println("\n--- Step 3d: Operator unarchives credential schema (already authorized) ---")
	err = lib.ArchiveCredentialSchemaWithAuthority(
		client, ctx, operatorAccount, policyAddr,
		csID, false,
	)
	if err != nil {
		return fmt.Errorf("step 3d failed: %w", err)
	}
	fmt.Println("✅ Step 3d: Credential schema unarchived")
	waitForTx("UnarchiveCS success")

	// Verify unarchived state
	unarchivedSchema, err := lib.QueryCredentialSchema(client, ctx, csID)
	if err != nil {
		return fmt.Errorf("step 3d verification query failed: %w", err)
	}
	if unarchivedSchema.Schema.Archived != nil {
		return fmt.Errorf("step 3d verification failed: credential schema should not be archived")
	}
	fmt.Println("✅ Step 3d: Verified credential schema is unarchived")

	// =========================================================================
	// TEST 4: Unauthorized operator (negative test)
	// =========================================================================
	fmt.Println("\n=== TEST 4: Unauthorized operator (negative test) ===")
	fmt.Println("\n--- Step 4: Unauthorized operator tries CreateCredentialSchema (expect failure) ---")

	// Use cooluser as an unauthorized operator (has funds but no CS authorization)
	cooluser := lib.GetAccount(client, lib.COOLUSER_NAME)

	_, err = lib.CreateCredentialSchemaWithAuthority(
		client, ctx, cooluser, policyAddr,
		trID, schemaData,
		cschema.CredentialSchemaPermManagementMode_OPEN,
		cschema.CredentialSchemaPermManagementMode_OPEN,
	)
	if err := expectAuthorizationError("Step 4", err); err != nil {
		return err
	}

	// Save results for potential downstream journeys
	result := lib.JourneyResult{
		TrustRegistryID: trIDStr,
		SchemaID:        csIDStr,
		DID:             did,
		GroupID:         setup.GroupID,
		GroupPolicyAddr: policyAddr,
		OperatorAddr:    operatorAddr,
	}
	lib.SaveJourneyResult("journey202", result)

	fmt.Println("\n========================================")
	fmt.Println("Journey 202 completed successfully! ✨")
	fmt.Println("All 3 CS operations tested: fail without auth, pass with auth.")
	fmt.Println("========================================")

	return nil
}
