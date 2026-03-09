package journeys

import (
	"context"
	"fmt"
	"strconv"
	"time"

	"cosmossdk.io/math"
	"github.com/ignite/cli/v28/ignite/pkg/cosmosclient"

	cschema "github.com/verana-labs/verana/x/cs/types"
	permtypes "github.com/verana-labs/verana/x/perm/types"

	"github.com/verana-labs/verana/testharness/lib"
)

// RunPermissionCreatePermJourney implements Journey 310: Test CreatePermission (Self Create)
// with operator authorization (authority/operator pattern).
//
// TEST 1: CreatePermission (fail without auth, grant auth, succeed)
// TEST 2: Verify created permission fields
// TEST 3: Unauthorized operator (negative test)
// TEST 4: Wrong authority - non-OPEN mode (negative test)
func RunPermissionCreatePermJourney(ctx context.Context, client cosmosclient.Client) error {
	fmt.Println("Starting Journey 310: CreatePermission (Self Create) with Operator Authorization")

	// Load results from prior journeys
	setup301 := lib.LoadJourneyResult("journey301")
	setup302 := lib.LoadJourneyResult("journey302")
	setup304 := lib.LoadJourneyResult("journey304")
	policyAddr := setup302.GroupPolicyAddr
	operatorAccount := lib.GetAccount(client, permOperatorName)
	operatorAddr := setup302.OperatorAddr
	adminAccount := lib.GetAccount(client, permGroupAdminName)
	member1Account := lib.GetAccount(client, permGroupMember1Name)

	trID, _ := strconv.ParseUint(setup304.TrustRegistryID, 10, 64)

	fmt.Printf("  Group Policy: %s\n", policyAddr)
	fmt.Printf("  Operator:     %s\n", operatorAddr)

	// =========================================================================
	// PREREQUISITES: Create CS with OPEN mode + ECOSYSTEM root perm
	// =========================================================================
	fmt.Println("\n=== PREREQUISITES: Create OPEN-mode CS + ECOSYSTEM root perm ===")

	// Re-grant CreateCredentialSchema auth
	fmt.Println("\n--- Prerequisite 1: Re-grant CreateCredentialSchema auth ---")
	err := lib.GrantOperatorAuthorizationViaGroup(
		client, ctx, adminAccount, member1Account,
		policyAddr, operatorAddr, operatorAddr,
		[]string{"/verana.cs.v1.MsgCreateCredentialSchema"},
	)
	if err != nil {
		return fmt.Errorf("prerequisite 1 failed: %w", err)
	}
	fmt.Println("OK Prerequisite 1: Re-granted CreateCredentialSchema authorization")
	waitForTx("re-grant CS auth")

	// Create a new CS with OPEN mode for both issuer and verifier
	fmt.Println("\n--- Prerequisite 2: Create OPEN-mode Credential Schema ---")
	schemaData := lib.GenerateSimpleSchema(setup304.TrustRegistryID)
	csIDStr, err := lib.CreateCredentialSchemaWithAuthority(
		client, ctx, operatorAccount, policyAddr,
		trID, schemaData,
		cschema.CredentialSchemaPermManagementMode_OPEN,
		cschema.CredentialSchemaPermManagementMode_OPEN,
	)
	if err != nil {
		return fmt.Errorf("prerequisite 2 failed: %w", err)
	}
	fmt.Printf("OK Prerequisite 2: OPEN-mode CS created with ID: %s\n", csIDStr)
	waitForTx("CS creation")

	// Re-grant CreateRootPermission auth
	fmt.Println("\n--- Prerequisite 3: Re-grant CreateRootPermission auth ---")
	err = lib.GrantOperatorAuthorizationViaGroup(
		client, ctx, adminAccount, member1Account,
		policyAddr, operatorAddr, operatorAddr,
		[]string{"/verana.perm.v1.MsgCreateRootPermission"},
	)
	if err != nil {
		return fmt.Errorf("prerequisite 3 failed: %w", err)
	}
	fmt.Println("OK Prerequisite 3: Re-granted CreateRootPermission authorization")
	waitForTx("re-grant root perm auth")

	// Fund group policy for root perm creation
	fmt.Println("\n--- Prerequisite 3b: Fund group policy ---")
	cooluserAddr, _ := lib.GetAccount(client, lib.COOLUSER_NAME).Address("verana")
	err = lib.SendBankTransaction(client, ctx, cooluserAddr, policyAddr, math.NewInt(200000000))
	if err != nil {
		return fmt.Errorf("prerequisite 3b failed: %w", err)
	}
	fmt.Println("OK Prerequisite 3b: Funded group policy with 200 VNA")
	waitForTx("fund policy")

	// Create root permission (ECOSYSTEM) for the OPEN-mode CS
	fmt.Println("\n--- Prerequisite 4: Create ECOSYSTEM root permission ---")
	csID, _ := strconv.ParseUint(csIDStr, 10, 64)
	rootPermDID := lib.GenerateUniqueDID(client, ctx)
	effectiveFrom := time.Now().Add(5 * time.Second)
	effectiveUntil := effectiveFrom.Add(360 * 24 * time.Hour)
	rootPermID, err := lib.CreateRootPermissionWithAuthority(
		client, ctx, operatorAccount, policyAddr,
		csID, rootPermDID, &effectiveFrom, &effectiveUntil,
		0, 0, 0, // no fees for ecosystem root
	)
	if err != nil {
		return fmt.Errorf("prerequisite 4 failed: %w", err)
	}
	fmt.Printf("OK Prerequisite 4: ECOSYSTEM root permission created with ID: %d\n", rootPermID)
	waitForTx("create root perm")

	// Wait for root permission to become effective
	fmt.Println("  Waiting for root permission to become effective...")
	time.Sleep(15 * time.Second)

	// =========================================================================
	// TEST 1: CreatePermission (fail without auth, grant auth, succeed)
	// =========================================================================
	fmt.Println("\n=== TEST 1: CreatePermission (Self Create) ===")

	issuerDID := lib.GenerateUniqueDID(client, ctx)
	permEffectiveFrom := time.Now().Add(30 * time.Second)
	permEffectiveUntil := effectiveUntil // same as root perm

	// 1a: Try WITHOUT authorization (expect failure)
	fmt.Println("\n--- Step 1a: Operator tries CreatePermission without auth (expect failure) ---")
	_, err = lib.CreatePermission(client, ctx, operatorAccount, policyAddr, permtypes.MsgCreatePermission{
		Type:            permtypes.PermissionType_ISSUER,
		ValidatorPermId: rootPermID,
		Did:             issuerDID,
		EffectiveFrom:   &permEffectiveFrom,
		EffectiveUntil:  &permEffectiveUntil,
	})
	if err := expectAuthorizationError("Step 1a", err); err != nil {
		return err
	}
	fmt.Println("OK Step 1a: CreatePermission correctly rejected without authorization")
	waitForTx("create perm rejection")

	// 1b: Grant authorization for CreatePermission
	fmt.Println("\n--- Step 1b: Grant operator auth for CreatePermission ---")
	err = lib.GrantOperatorAuthorizationViaGroup(
		client, ctx, adminAccount, member1Account,
		policyAddr, operatorAddr, operatorAddr,
		[]string{"/verana.perm.v1.MsgCreatePermission"},
	)
	if err != nil {
		return fmt.Errorf("step 1b failed: %w", err)
	}
	fmt.Println("OK Step 1b: Granted CreatePermission authorization")
	waitForTx("grant create perm auth")

	// 1c: Try WITH authorization (expect success)
	fmt.Println("\n--- Step 1c: Operator creates permission with auth (expect success) ---")
	_, err = lib.CreatePermission(client, ctx, operatorAccount, policyAddr, permtypes.MsgCreatePermission{
		Type:             permtypes.PermissionType_ISSUER,
		ValidatorPermId:  rootPermID,
		Did:              issuerDID,
		EffectiveFrom:    &permEffectiveFrom,
		EffectiveUntil:   &permEffectiveUntil,
		VerificationFees: 50,
		ValidationFees:   25,
	})
	if err != nil {
		return fmt.Errorf("step 1c failed: %w", err)
	}
	fmt.Println("OK Step 1c: CreatePermission succeeded")
	waitForTx("create perm success")

	// =========================================================================
	// TEST 2: Verify created permission fields
	// =========================================================================
	fmt.Println("\n=== TEST 2: Verify created permission fields ===")

	// Find the newly created permission by walking permissions
	// We know it's the latest one for this authority with ISSUER type
	perms, err := lib.ListPermissions(client, ctx)
	if err != nil {
		return fmt.Errorf("step 2 query failed: %w", err)
	}

	var createdPerm *permtypes.Permission
	for i := len(perms) - 1; i >= 0; i-- {
		if perms[i].Authority == policyAddr && perms[i].Type == permtypes.PermissionType_ISSUER && perms[i].Did == issuerDID {
			createdPerm = &perms[i]
			break
		}
	}

	if createdPerm == nil {
		return fmt.Errorf("step 2 failed: created permission not found")
	}

	// Verify fields
	if createdPerm.ValidatorPermId != rootPermID {
		return fmt.Errorf("step 2 failed: expected validator_perm_id=%d, got %d", rootPermID, createdPerm.ValidatorPermId)
	}
	if createdPerm.Authority != policyAddr {
		return fmt.Errorf("step 2 failed: expected authority=%s, got %s", policyAddr, createdPerm.Authority)
	}
	if createdPerm.Did != issuerDID {
		return fmt.Errorf("step 2 failed: expected did=%s, got %s", issuerDID, createdPerm.Did)
	}
	if createdPerm.Created == nil {
		return fmt.Errorf("step 2 failed: created timestamp is nil")
	}
	if createdPerm.Modified == nil {
		return fmt.Errorf("step 2 failed: modified timestamp is nil")
	}

	fmt.Printf("OK Step 2: Verified created permission fields (id=%d, validator_perm_id=%d, authority=%s)\n",
		createdPerm.Id, createdPerm.ValidatorPermId, createdPerm.Authority)

	// =========================================================================
	// TEST 3: Unauthorized operator (negative test)
	// =========================================================================
	fmt.Println("\n=== TEST 3: Unauthorized operator (negative test) ===")

	fmt.Println("\n--- Step 3a: Unauthorized operator tries CreatePermission (expect failure) ---")
	coolusrAcct := lib.GetAccount(client, lib.COOLUSER_NAME)
	_, err = lib.CreatePermission(client, ctx, coolusrAcct, policyAddr, permtypes.MsgCreatePermission{
		Type:            permtypes.PermissionType_ISSUER,
		ValidatorPermId: rootPermID,
		Did:             "did:example:unauthorized",
		EffectiveFrom:   &permEffectiveFrom,
		EffectiveUntil:  &permEffectiveUntil,
	})
	if err := expectAuthorizationError("Step 3a", err); err != nil {
		return err
	}
	fmt.Println("OK Step 3a: Unauthorized operator correctly rejected")

	// =========================================================================
	// TEST 4: Wrong authority (negative test)
	// =========================================================================
	fmt.Println("\n=== TEST 4: Wrong authority (negative test) ===")

	fmt.Println("\n--- Step 4a: Correct operator but wrong authority (expect failure) ---")
	_, err = lib.CreatePermission(client, ctx, operatorAccount, operatorAddr, permtypes.MsgCreatePermission{
		Type:            permtypes.PermissionType_ISSUER,
		ValidatorPermId: rootPermID,
		Did:             "did:example:wrongauth",
		EffectiveFrom:   &permEffectiveFrom,
		EffectiveUntil:  &permEffectiveUntil,
	})
	if err == nil {
		return fmt.Errorf("step 4a failed: expected error for wrong authority, got nil")
	}
	fmt.Printf("OK Step 4a: Wrong authority correctly rejected: %s\n", err.Error())

	// Save results
	result := lib.JourneyResult{
		TrustRegistryID: setup304.TrustRegistryID,
		SchemaID:        csIDStr,
		DID:             rootPermDID,
		PermissionID:    strconv.FormatUint(rootPermID, 10),
		GroupID:         setup301.GroupID,
		GroupPolicyAddr: policyAddr,
		OperatorAddr:    operatorAddr,
	}
	lib.SaveJourneyResult("journey310", result)

	fmt.Println("\n========================================")
	fmt.Println("Journey 310 completed successfully!")
	fmt.Println("CreatePermission (Self Create) tested:")
	fmt.Println("  - Unauthorized operator rejected")
	fmt.Println("  - Authorized operator succeeded")
	fmt.Println("  - Created permission fields verified")
	fmt.Println("  - Unauthorized operator rejected")
	fmt.Println("  - Wrong authority rejected")
	fmt.Println("========================================")

	return nil
}
