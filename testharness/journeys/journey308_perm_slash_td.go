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

// RunPermissionSlashTDJourney implements Journey 308: Test SlashPermissionTrustDeposit
// with operator authorization (authority/operator pattern).
//
// TEST 1: SlashPermissionTrustDeposit (fail without auth, grant auth, succeed)
// TEST 2: Verify slashed permission fields
// TEST 3: Unauthorized operator (negative test)
// TEST 4: Wrong authority (negative test)
// Depends on Journey 301 (setup), 302 (group/operator), 304 (root permission).
func RunPermissionSlashTDJourney(ctx context.Context, client cosmosclient.Client) error {
	fmt.Println("Starting Journey 308: SlashPermissionTrustDeposit with Operator Authorization")

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

	trID, _ := strconv.ParseUint(setup304.TrustRegistryID, 10, 64)

	// =========================================================================
	// PREREQUISITES: Create a new CS + ECOSYSTEM root + ISSUER child perm
	// (ISSUER perm has a deposit and will be the target for slashing)
	// =========================================================================
	fmt.Println("\n=== PREREQUISITES: Create CS, root perm, and ISSUER perm with deposit ===")

	// Re-grant CreateCredentialSchema auth (may have been overwritten by prior journeys)
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
		cschema.CredentialSchemaPermManagementMode_ECOSYSTEM,
		cschema.CredentialSchemaPermManagementMode_ECOSYSTEM,
	)
	if err != nil {
		return fmt.Errorf("prerequisite 2 failed: could not create CS: %w", err)
	}
	csID, _ := strconv.ParseUint(csIDStr, 10, 64)
	fmt.Printf("OK Prerequisite 2: Credential Schema created with ID: %d\n", csID)
	waitForTx("CS creation for journey 308")

	// Re-grant CreateRootPermission auth
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

	// Create root permission on the new CS (ECOSYSTEM)
	fmt.Println("\n--- Prerequisite 4: Create root permission ---")
	rootPermDID := lib.GenerateUniqueDID(client, ctx)
	effectiveFrom := time.Now().Add(5 * time.Second)
	effectiveUntil := effectiveFrom.Add(360 * 24 * time.Hour)
	rootPermID, err := lib.CreateRootPermissionWithAuthority(
		client, ctx, operatorAccount, policyAddr,
		csID, rootPermDID, &effectiveFrom, &effectiveUntil,
		100, 50, 25,
	)
	if err != nil {
		return fmt.Errorf("prerequisite 4 failed: could not create root permission: %w", err)
	}
	fmt.Printf("OK Prerequisite 4: Root permission created with ID: %d\n", rootPermID)
	waitForTx("create root perm for slash test")

	// Wait for root permission to become effective
	fmt.Println("  Waiting for root permission to become effective...")
	time.Sleep(15 * time.Second)

	// Re-grant StartPermissionVP + SetPermissionVPToValidated auth for creating the ISSUER child
	fmt.Println("\n--- Prerequisite 5: Grant StartPermissionVP + SetPermissionVPToValidated auth ---")
	err = lib.GrantOperatorAuthorizationViaGroup(
		client, ctx, adminAccount, member1Account,
		policyAddr, operatorAddr, operatorAddr,
		[]string{
			"/verana.perm.v1.MsgStartPermissionVP",
			"/verana.perm.v1.MsgSetPermissionVPToValidated",
		},
	)
	if err != nil {
		return fmt.Errorf("prerequisite 5 failed: could not grant VP auth: %w", err)
	}
	fmt.Println("OK Prerequisite 5: Granted StartPermissionVP + SetPermissionVPToValidated auth")
	waitForTx("grant VP auth")

	// Fund the group policy address with additional tokens for VP fees
	fmt.Println("\n--- Prerequisite 5b: Fund group policy for VP fees ---")
	cooluserAddr, _ := lib.GetAccount(client, lib.COOLUSER_NAME).Address("verana")
	err = lib.SendBankTransaction(client, ctx, cooluserAddr, policyAddr, math.NewInt(200000000))
	if err != nil {
		return fmt.Errorf("prerequisite 5b failed: could not fund policy for VP fees: %w", err)
	}
	fmt.Println("OK Prerequisite 5b: Funded group policy with 200 VNA for VP fees")
	waitForTx("fund policy for VP")

	// Create ISSUER child permission (via start VP + validate)
	fmt.Println("\n--- Prerequisite 6: Start ISSUER permission VP ---")
	issuerDID := lib.GenerateUniqueDID(client, ctx)
	issuerPermIDStr, err := lib.StartPermissionVP(client, ctx, operatorAccount, permtypes.MsgStartPermissionVP{
		Authority:       policyAddr,
		Type:            permtypes.PermissionType_ISSUER,
		ValidatorPermId: rootPermID,
		Did:             issuerDID,
	})
	if err != nil {
		return fmt.Errorf("prerequisite 6 failed: could not start ISSUER perm VP: %w", err)
	}
	issuerPermID, _ := strconv.ParseUint(issuerPermIDStr, 10, 64)
	fmt.Printf("OK Prerequisite 6: ISSUER perm started with ID: %d\n", issuerPermID)
	waitForTx("start issuer perm VP")

	// Validate the ISSUER perm
	fmt.Println("\n--- Prerequisite 7: Validate ISSUER permission ---")
	_, err = lib.SetPermissionVPToValidated(client, ctx, operatorAccount, permtypes.MsgSetPermissionVPToValidated{
		Authority: policyAddr,
		Id:        issuerPermID,
	})
	if err != nil {
		return fmt.Errorf("prerequisite 7 failed: could not validate ISSUER perm: %w", err)
	}
	fmt.Printf("OK Prerequisite 7: ISSUER perm %d validated\n", issuerPermID)
	waitForTx("validate issuer perm")

	// Verify deposit exists on the ISSUER perm
	issuerPerm, err := lib.GetPermission(client, ctx, issuerPermID)
	if err != nil {
		return fmt.Errorf("prerequisite verification failed: could not load ISSUER perm: %w", err)
	}
	fmt.Printf("  ISSUER perm deposit: %d, authority: %s\n", issuerPerm.Deposit, issuerPerm.Authority)

	// =========================================================================
	// TEST 1: SlashPermissionTrustDeposit (fail without auth, grant auth, succeed)
	// =========================================================================
	fmt.Println("\n=== TEST 1: SlashPermissionTrustDeposit ===")

	// Use the root perm's authority (policyAddr) as the slasher - it's a validator ancestor
	slashAmount := uint64(10) // small amount to keep the perm deposit positive

	// 1a: Try WITHOUT authorization (expect failure)
	fmt.Println("\n--- Step 1a: Operator tries SlashPermissionTrustDeposit without auth (expect failure) ---")
	err = lib.SlashPermissionTrustDeposit(client, ctx, operatorAccount, policyAddr, issuerPermID, slashAmount)
	if err := expectAuthorizationError("Step 1a", err); err != nil {
		return err
	}
	fmt.Println("OK Step 1a: SlashPermissionTrustDeposit correctly rejected without authorization")
	waitForTx("slash rejection")

	// 1b: Grant authorization for SlashPermissionTrustDeposit
	fmt.Println("\n--- Step 1b: Grant operator auth for SlashPermissionTrustDeposit ---")
	err = lib.GrantOperatorAuthorizationViaGroup(
		client, ctx, adminAccount, member1Account,
		policyAddr, operatorAddr, operatorAddr,
		[]string{"/verana.perm.v1.MsgSlashPermissionTrustDeposit"},
	)
	if err != nil {
		return fmt.Errorf("step 1b failed: %w", err)
	}
	fmt.Println("OK Step 1b: Granted SlashPermissionTrustDeposit authorization")
	waitForTx("grant slash auth")

	// 1c: Try WITH authorization (expect success - validator ancestor)
	fmt.Println("\n--- Step 1c: Operator slashes perm trust deposit with auth (expect success) ---")
	err = lib.SlashPermissionTrustDeposit(client, ctx, operatorAccount, policyAddr, issuerPermID, slashAmount)
	if err != nil {
		return fmt.Errorf("step 1c failed: %w", err)
	}
	fmt.Printf("OK Step 1c: SlashPermissionTrustDeposit succeeded for perm %d (amount=%d)\n", issuerPermID, slashAmount)
	waitForTx("slash success")

	// =========================================================================
	// TEST 2: Verify slashed permission fields
	// =========================================================================
	fmt.Println("\n=== TEST 2: Verify slashed permission fields ===")
	slashedPerm, err := lib.GetPermission(client, ctx, issuerPermID)
	if err != nil {
		return fmt.Errorf("step 2 query failed: %w", err)
	}

	// Verify slashed timestamp is set
	if slashedPerm.Slashed == nil {
		return fmt.Errorf("step 2 failed: slashed timestamp is nil")
	}

	// Verify slashed_by is the authority (policy address)
	if slashedPerm.SlashedBy != policyAddr {
		return fmt.Errorf("step 2 failed: expected slashed_by=%s, got %s", policyAddr, slashedPerm.SlashedBy)
	}

	// Verify slashed_deposit is set
	if slashedPerm.SlashedDeposit != slashAmount {
		return fmt.Errorf("step 2 failed: expected slashed_deposit=%d, got %d", slashAmount, slashedPerm.SlashedDeposit)
	}

	// Verify modified timestamp is set
	if slashedPerm.Modified == nil {
		return fmt.Errorf("step 2 failed: modified timestamp is nil")
	}

	fmt.Printf("OK Step 2: Verified slashed fields (slashed_by=%s, slashed_deposit=%d)\n",
		slashedPerm.SlashedBy, slashedPerm.SlashedDeposit)

	// =========================================================================
	// TEST 3: Unauthorized operator (negative test)
	// =========================================================================
	fmt.Println("\n=== TEST 3: Unauthorized operator (negative test) ===")

	fmt.Println("\n--- Step 3a: Unauthorized operator tries SlashPermissionTrustDeposit (expect failure) ---")
	coolusrAcct := lib.GetAccount(client, lib.COOLUSER_NAME)
	err = lib.SlashPermissionTrustDeposit(client, ctx, coolusrAcct, policyAddr, issuerPermID, slashAmount)
	if err := expectAuthorizationError("Step 3a", err); err != nil {
		return err
	}
	fmt.Println("OK Step 3a: Unauthorized operator correctly rejected")

	// =========================================================================
	// TEST 4: Wrong authority (negative test)
	// =========================================================================
	fmt.Println("\n=== TEST 4: Wrong authority (negative test) ===")

	fmt.Println("\n--- Step 4a: Correct operator but wrong authority (expect failure) ---")
	// The operator has a self-delegation but the self-delegation authority is operatorAddr,
	// not an ancestor validator or TR controller for the issuer perm
	err = lib.SlashPermissionTrustDeposit(client, ctx, operatorAccount, operatorAddr, issuerPermID, slashAmount)
	if err == nil {
		return fmt.Errorf("step 4a failed: expected error for wrong authority, got nil")
	}
	fmt.Printf("OK Step 4a: Wrong authority correctly rejected: %s\n", err.Error())

	// Save results
	result := lib.JourneyResult{
		TrustRegistryID: setup304.TrustRegistryID,
		SchemaID:        csIDStr,
		DID:             rootPermDID,
		PermissionID:    strconv.FormatUint(issuerPermID, 10),
		GroupID:         setup301.GroupID,
		GroupPolicyAddr: policyAddr,
		OperatorAddr:    operatorAddr,
	}
	lib.SaveJourneyResult("journey308", result)

	fmt.Println("\n========================================")
	fmt.Println("Journey 308 completed successfully!")
	fmt.Println("SlashPermissionTrustDeposit tested:")
	fmt.Println("  - Unauthorized operator rejected")
	fmt.Println("  - Authorized operator succeeded (validator ancestor)")
	fmt.Println("  - Slashed fields verified")
	fmt.Println("  - Unauthorized operator rejected")
	fmt.Println("  - Wrong authority rejected")
	fmt.Println("========================================")

	return nil
}
