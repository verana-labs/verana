package journeys

import (
	"context"
	"fmt"
	"time"

	"github.com/ignite/cli/v28/ignite/pkg/cosmosclient"

	ditypes "github.com/verana-labs/verana/x/di/types"

	"github.com/verana-labs/verana/testharness/lib"
)

// RunDiStoreDigestJourney implements Journey 501: DI Store Digest
// Tests MsgStoreDigest with operator authorization (fail without auth, grant, succeed with auth).
// Depends on Journey 301 (group setup with operator).
func RunDiStoreDigestJourney(ctx context.Context, client cosmosclient.Client) error {
	fmt.Println("Starting Journey 501: DI Store Digest with Operator Authorization")

	// =========================================================================
	// Step 1: Load journey 301 results (group, operator, authority)
	// =========================================================================
	fmt.Println("\n--- Step 1: Load journey 301 results ---")

	setup301 := lib.LoadJourneyResult("journey301")
	policyAddr := setup301.GroupPolicyAddr
	operatorAddr := setup301.OperatorAddr

	operatorAccount := lib.GetAccount(client, permOperatorName)
	adminAccount := lib.GetAccount(client, permGroupAdminName)
	member1Account := lib.GetAccount(client, permGroupMember1Name)

	fmt.Printf("  Group Policy:  %s\n", policyAddr)
	fmt.Printf("  Operator:      %s\n", operatorAddr)
	fmt.Println("✅ Step 1: Loaded journey 301 results")

	// =========================================================================
	// Step 2: Test StoreDigest WITHOUT authorization (expect failure)
	// =========================================================================
	fmt.Println("\n--- Step 2: Operator tries StoreDigest without auth (expect failure) ---")

	testDigest := fmt.Sprintf("sha256-test-digest-%d", time.Now().UnixNano())
	storeMsg := &ditypes.MsgStoreDigest{
		Authority: policyAddr,
		Operator:  operatorAddr,
		Digest:    testDigest,
	}

	txResp, err := client.BroadcastTx(ctx, operatorAccount, storeMsg)
	if err == nil && txResp.TxResponse.Code != 0 {
		err = fmt.Errorf("transaction failed with code %d: %s", txResp.TxResponse.Code, txResp.TxResponse.RawLog)
	}
	if err := expectAuthorizationError("Step 2", err); err != nil {
		return err
	}
	fmt.Println("✅ Step 2: StoreDigest correctly rejected without authorization")
	waitForTx("store digest rejection")

	// =========================================================================
	// Step 3: Grant operator authorization for MsgStoreDigest via group proposal
	// =========================================================================
	fmt.Println("\n--- Step 3: Grant operator auth for MsgStoreDigest ---")

	err = lib.GrantOperatorAuthorizationViaGroup(
		client, ctx, adminAccount, member1Account,
		policyAddr, operatorAddr, operatorAddr,
		[]string{"/verana.di.v1.MsgStoreDigest"},
	)
	if err != nil {
		return fmt.Errorf("step 3 failed: %w", err)
	}
	fmt.Println("✅ Step 3: Granted MsgStoreDigest authorization")
	waitForTx("grant store digest auth")

	// =========================================================================
	// Step 4: Test StoreDigest WITH authorization (expect success)
	// =========================================================================
	fmt.Println("\n--- Step 4: Operator stores digest with auth ---")

	storeMsg = &ditypes.MsgStoreDigest{
		Authority: policyAddr,
		Operator:  operatorAddr,
		Digest:    testDigest,
	}

	txResp, err = client.BroadcastTx(ctx, operatorAccount, storeMsg)
	if err != nil {
		return fmt.Errorf("step 4 failed: %w", err)
	}
	if txResp.TxResponse.Code != 0 {
		return fmt.Errorf("step 4 failed with code %d: %s", txResp.TxResponse.Code, txResp.TxResponse.RawLog)
	}
	fmt.Printf("✅ Step 4: StoreDigest succeeded (digest: %s)\n", testDigest)
	waitForTx("store digest success")

	// =========================================================================
	// Step 5: Query the digest via GetDigest to verify it was stored
	// =========================================================================
	fmt.Println("\n--- Step 5: Query stored digest ---")

	diQueryClient := ditypes.NewQueryClient(client.Context())
	digestResp, err := diQueryClient.GetDigest(ctx, &ditypes.QueryGetDigestRequest{
		Digest: testDigest,
	})
	if err != nil {
		return fmt.Errorf("step 5 failed: could not query digest: %w", err)
	}

	if digestResp.Digest == nil {
		return fmt.Errorf("step 5 failed: digest response is nil")
	}

	if digestResp.Digest.Digest != testDigest {
		return fmt.Errorf("step 5 failed: expected digest %s, got %s", testDigest, digestResp.Digest.Digest)
	}

	fmt.Printf("✅ Step 5: Verified stored digest:\n")
	fmt.Printf("    Digest:  %s\n", digestResp.Digest.Digest)
	fmt.Printf("    Created: %s\n", digestResp.Digest.Created.Format(time.RFC3339))

	// =========================================================================
	// Save results for downstream journeys
	// =========================================================================
	result := lib.JourneyResult{
		GroupID:         setup301.GroupID,
		GroupPolicyAddr: policyAddr,
		OperatorAddr:    operatorAddr,
	}
	lib.SaveJourneyResult("journey501", result)

	fmt.Println("\n========================================")
	fmt.Println("Journey 501 completed successfully!")
	fmt.Println("DI StoreDigest tested:")
	fmt.Println("  - StoreDigest: unauthorized rejected")
	fmt.Println("  - StoreDigest: authorized succeeded")
	fmt.Println("  - GetDigest: verified stored digest")
	fmt.Println("========================================")

	return nil
}
