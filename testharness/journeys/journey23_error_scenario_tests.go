package journeys

import (
	"context"
	"fmt"
	"strings"
	"time"

	"cosmossdk.io/math"
	"github.com/ignite/cli/v28/ignite/pkg/cosmosclient"

	"github.com/verana-labs/verana/testharness/lib"
	permtypes "github.com/verana-labs/verana/x/perm/types"
)

// =============================================================================
// JOURNEY 23: Error Scenario Tests
// =============================================================================
// This journey tests error scenarios for Issues #191, #193, and #196:
// - Issue #191: CreateRootPermission requires effective_from to be set
// - Issue #193: StartPermissionVP requires validator permission to be ACTIVE
// - Issue #196: RevokePermission allows revoking not-yet-active permissions
//
// Each test case broadcasts a transaction, queries the transaction result by hash,
// and verifies the expected error message is returned.

// RunErrorScenarioTestsJourney implements Journey 23: Error Scenario Tests
func RunErrorScenarioTestsJourney(ctx context.Context, client cosmosclient.Client) error {
	fmt.Println("Starting Journey 23: Error Scenario Tests")
	fmt.Println("=========================================")
	fmt.Println("Testing error handling for Issues #191, #193, and #196")
	fmt.Println()

	// Setup: Load data from Journey 1
	journey1Result, err := lib.GetJourneyResult("journey1")
	if err != nil {
		return fmt.Errorf("failed to load Journey 1 results (run Journey 1 first): %v", err)
	}

	// Get Trust Registry Controller account
	trController, err := client.Account(lib.TRUST_REGISTRY_CONTROLLER_NAME)
	if err != nil {
		return fmt.Errorf("failed to get Trust_Registry_Controller account: %v", err)
	}

	trControllerAddr, err := trController.Address(lib.GetAddressPrefix())
	if err != nil {
		return fmt.Errorf("failed to get Trust_Registry_Controller address: %v", err)
	}

	// Ensure accounts have funds
	lib.SendFunds(client, ctx, lib.COOLUSER_ADDRESS, trControllerAddr, math.NewInt(30000000)) // 30 VNA

	// ==========================================================================
	// ISSUE #191 TESTS: CreateRootPermission requires effective_from
	// ==========================================================================
	fmt.Println("\n=== ISSUE #191 TESTS: CreateRootPermission requires effective_from ===")

	schemaID := journey1Result.SchemaID
	testDID := lib.GenerateUniqueDID(client, ctx)

	// Test 1: CreateRootPermission with nil effective_from should fail
	fmt.Println("\n📋 Test 1: CreateRootPermission with nil effective_from")
	fmt.Println("   Expected: Transaction should fail with 'effective_from is required'")

	err191_1 := lib.CreateRootPermissionWithError(client, ctx, trController, permtypes.MsgCreateRootPermission{
		SchemaId:       lib.ParseSchemaID(schemaID),
		Did:            testDID,
		EffectiveFrom:  nil, // NIL - should fail
		EffectiveUntil: nil,
	})

	if err191_1 != nil && strings.Contains(err191_1.Error(), "effective_from is required") {
		fmt.Println("   ✅ PASS: Transaction correctly rejected with expected error message")
		fmt.Printf("   Error: %s\n", err191_1.Error())
	} else if err191_1 != nil {
		fmt.Println("   ⚠️  PARTIAL: Transaction failed but with different error")
		fmt.Printf("   Error: %s\n", err191_1.Error())
	} else {
		fmt.Println("   ❌ FAIL: Transaction unexpectedly succeeded")
	}

	// Test 2: CreateRootPermission with past effective_from should fail
	fmt.Println("\n📋 Test 2: CreateRootPermission with past effective_from")
	fmt.Println("   Expected: Transaction should fail with 'effective_from must be in the future'")

	pastTime := time.Now().Add(-1 * time.Hour)
	err191_2 := lib.CreateRootPermissionWithError(client, ctx, trController, permtypes.MsgCreateRootPermission{
		SchemaId:       lib.ParseSchemaID(schemaID),
		Did:            testDID,
		EffectiveFrom:  &pastTime, // PAST - should fail
		EffectiveUntil: nil,
	})

	if err191_2 != nil && strings.Contains(err191_2.Error(), "effective_from must be in the future") {
		fmt.Println("   ✅ PASS: Transaction correctly rejected with expected error message")
		fmt.Printf("   Error: %s\n", err191_2.Error())
	} else if err191_2 != nil {
		fmt.Println("   ⚠️  PARTIAL: Transaction failed but with different error")
		fmt.Printf("   Error: %s\n", err191_2.Error())
	} else {
		fmt.Println("   ❌ FAIL: Transaction unexpectedly succeeded")
	}

	// Test 3: CreateRootPermission with future effective_from should succeed
	fmt.Println("\n📋 Test 3: CreateRootPermission with future effective_from")
	fmt.Println("   Expected: Transaction should succeed")

	futureTime := time.Now().Add(1 * time.Hour)
	farFutureTime := time.Now().Add(24 * time.Hour)
	err191_3 := lib.CreateRootPermissionWithError(client, ctx, trController, permtypes.MsgCreateRootPermission{
		SchemaId:       lib.ParseSchemaID(schemaID),
		Did:            testDID,
		EffectiveFrom:  &futureTime, // FUTURE - should succeed
		EffectiveUntil: &farFutureTime,
	})

	if err191_3 == nil {
		fmt.Println("   ✅ PASS: Transaction succeeded as expected")
	} else {
		fmt.Println("   ❌ FAIL: Transaction unexpectedly failed")
		fmt.Printf("   Error: %s\n", err191_3.Error())
	}

	// ==========================================================================
	// ISSUE #193 TESTS: StartPermissionVP requires ACTIVE validator
	// ==========================================================================
	fmt.Println("\n=== ISSUE #193 TESTS: StartPermissionVP requires ACTIVE validator ===")

	// Get an applicant account
	applicantAccount, err := client.Account(lib.ISSUER_APPLICANT_NAME)
	if err != nil {
		return fmt.Errorf("failed to get Issuer_Applicant account: %v", err)
	}

	applicantAddr, err := applicantAccount.Address(lib.GetAddressPrefix())
	if err != nil {
		return fmt.Errorf("failed to get applicant address: %v", err)
	}

	// Ensure applicant has funds
	lib.SendFunds(client, ctx, lib.COOLUSER_ADDRESS, applicantAddr, math.NewInt(30000000)) // 30 VNA

	// Create an INACTIVE validator permission (no effective_from)
	fmt.Println("\n📋 Creating INACTIVE validator permission for testing...")
	inactiveValidatorDID := lib.GenerateUniqueDID(client, ctx)
	inactiveValidatorPermID, err := lib.CreateInactiveValidatorPermission(client, ctx, trController, schemaID, inactiveValidatorDID)
	if err != nil {
		fmt.Printf("   ⚠️  Could not create inactive validator permission: %v\n", err)
		fmt.Println("   Skipping Issue #193 tests...")
	} else {
		// Test 4: StartPermissionVP with INACTIVE validator should fail
		fmt.Println("\n📋 Test 4: StartPermissionVP with INACTIVE validator (null effective_from)")
		fmt.Println("   Expected: Transaction should fail with 'validator perm is not valid'")

		applicantDID := lib.GenerateUniqueDID(client, ctx)
		err193_1 := lib.StartPermissionVPWithError(client, ctx, applicantAccount, permtypes.MsgStartPermissionVP{
			Type:            permtypes.PermissionType_ISSUER,
			ValidatorPermId: inactiveValidatorPermID,
			Did:             applicantDID,
			Country:         "US",
		})

		if err193_1 != nil && strings.Contains(err193_1.Error(), "validator perm is not valid") {
			fmt.Println("   ✅ PASS: Transaction correctly rejected with expected error message")
			fmt.Printf("   Error: %s\n", err193_1.Error())
		} else if err193_1 != nil {
			fmt.Println("   ⚠️  PARTIAL: Transaction failed but with different error")
			fmt.Printf("   Error: %s\n", err193_1.Error())
		} else {
			fmt.Println("   ❌ FAIL: Transaction unexpectedly succeeded")
		}
	}

	// ==========================================================================
	// ISSUE #196 TESTS: RevokePermission allows not-yet-active permissions
	// ==========================================================================
	fmt.Println("\n=== ISSUE #196 TESTS: RevokePermission allows not-yet-active permissions ===")

	// Create a FUTURE permission (effective_from in the future)
	fmt.Println("\n📋 Creating FUTURE permission for testing...")

	futureDID := lib.GenerateUniqueDID(client, ctx)
	futureEffective := time.Now().Add(24 * time.Hour)
	farFuture := time.Now().Add(48 * time.Hour)

	futurePermID, err := lib.CreateRootPermissionAndGetID(client, ctx, trController, permtypes.MsgCreateRootPermission{
		SchemaId:       lib.ParseSchemaID(schemaID),
		Did:            futureDID,
		EffectiveFrom:  &futureEffective,
		EffectiveUntil: &farFuture,
	})

	if err != nil {
		fmt.Printf("   ⚠️  Could not create future permission: %v\n", err)
		fmt.Println("   Skipping Issue #196 tests...")
	} else {
		fmt.Printf("   Created future permission ID: %d\n", futurePermID)

		// Test 5: RevokePermission on FUTURE permission should succeed
		fmt.Println("\n📋 Test 5: RevokePermission on FUTURE permission (not yet active)")
		fmt.Println("   Expected: Transaction should succeed (per Issue #196 fix)")

		err196_1 := lib.RevokePermissionWithError(client, ctx, trController, permtypes.MsgRevokePermission{
			Id: futurePermID,
		})

		if err196_1 == nil {
			fmt.Println("   ✅ PASS: Transaction succeeded as expected (Issue #196 fix confirmed)")

			// Verify the permission was revoked
			perm, queryErr := lib.GetPermission(client, ctx, futurePermID)
			if queryErr == nil && perm.Revoked != nil {
				fmt.Printf("   Permission was revoked at: %v\n", perm.Revoked)
				fmt.Printf("   Permission was revoked by: %s\n", perm.RevokedBy)
			}
		} else {
			fmt.Println("   ❌ FAIL: Transaction failed - Issue #196 may not be properly fixed")
			fmt.Printf("   Error: %s\n", err196_1.Error())
		}
	}

	// ==========================================================================
	// SUMMARY
	// ==========================================================================
	fmt.Println("\n" + strings.Repeat("=", 60))
	fmt.Println("JOURNEY 23: Error Scenario Tests - COMPLETED")
	fmt.Println(strings.Repeat("=", 60))
	fmt.Println("\nSummary of tested scenarios:")
	fmt.Println("  - Issue #191: CreateRootPermission requires effective_from")
	fmt.Println("  - Issue #193: StartPermissionVP requires ACTIVE validator")
	fmt.Println("  - Issue #196: RevokePermission allows not-yet-active permissions")
	fmt.Println("\nJourney 23 completed! ✨")

	// Save result
	result := lib.JourneyResult{
		TrustRegistryID: journey1Result.TrustRegistryID,
		SchemaID:        journey1Result.SchemaID,
	}
	lib.SaveJourneyResult("journey23", result)

	return nil
}
