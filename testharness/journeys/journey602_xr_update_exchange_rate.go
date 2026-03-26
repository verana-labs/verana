package journeys

import (
	"context"
	"fmt"
	"strconv"
	"time"

	"github.com/ignite/cli/v28/ignite/pkg/cosmosclient"

	xrtypes "github.com/verana-labs/verana/x/xr/types"

	"github.com/verana-labs/verana/testharness/lib"
)

// RunXrUpdateExchangeRateJourney implements Journey 602: XR Update Exchange Rate (operator)
// Tests MsgUpdateExchangeRate with operator authorization (fail without auth, grant, succeed with auth).
// Depends on Journey 601 (exchange rate ID) and Journey 301 (group, operator).
func RunXrUpdateExchangeRateJourney(ctx context.Context, client cosmosclient.Client) error {
	fmt.Println("Starting Journey 602: XR Update Exchange Rate with Operator Authorization")

	// =========================================================================
	// Step 1: Load journey 601 and 301 results
	// =========================================================================
	fmt.Println("\n--- Step 1: Load journey results ---")

	setup601 := lib.LoadJourneyResult("journey601")
	setup301 := lib.LoadJourneyResult("journey301")

	exchangeRateID, err := strconv.ParseUint(setup601.ExchangeRateID, 10, 64)
	if err != nil {
		return fmt.Errorf("step 1 failed: could not parse exchange rate ID: %w", err)
	}

	policyAddr := setup301.GroupPolicyAddr
	operatorAddr := setup301.OperatorAddr

	operatorAccount := lib.GetAccount(client, permOperatorName)
	adminAccount := lib.GetAccount(client, permGroupAdminName)
	member1Account := lib.GetAccount(client, permGroupMember1Name)

	fmt.Printf("  Exchange Rate ID: %d\n", exchangeRateID)
	fmt.Printf("  Group Policy:     %s\n", policyAddr)
	fmt.Printf("  Operator:         %s\n", operatorAddr)
	fmt.Println("✅ Step 1: Loaded journey results")

	// =========================================================================
	// Step 2: Test UpdateExchangeRate WITHOUT authorization (expect failure)
	// =========================================================================
	fmt.Println("\n--- Step 2: Operator tries UpdateExchangeRate without auth (expect failure) ---")

	updateMsg := &xrtypes.MsgUpdateExchangeRate{
		Authority: policyAddr,
		Operator:  operatorAddr,
		Id:        exchangeRateID,
		Rate:      "2000000",
	}

	txResp, err := client.BroadcastTx(ctx, operatorAccount, updateMsg)
	if err == nil && txResp.TxResponse.Code != 0 {
		err = fmt.Errorf("transaction failed with code %d: %s", txResp.TxResponse.Code, txResp.TxResponse.RawLog)
	}
	if err := expectAuthorizationError("Step 2", err); err != nil {
		return err
	}
	fmt.Println("✅ Step 2: UpdateExchangeRate correctly rejected without authorization")
	waitForTx("update exchange rate rejection")

	// =========================================================================
	// Step 3: Grant operator authorization for MsgUpdateExchangeRate via group proposal
	// =========================================================================
	fmt.Println("\n--- Step 3: Grant operator auth for MsgUpdateExchangeRate ---")

	err = lib.GrantOperatorAuthorizationViaGroup(
		client, ctx, adminAccount, member1Account,
		policyAddr, operatorAddr, operatorAddr,
		[]string{"/verana.xr.v1.MsgUpdateExchangeRate"},
	)
	if err != nil {
		return fmt.Errorf("step 3 failed: %w", err)
	}
	fmt.Println("✅ Step 3: Granted MsgUpdateExchangeRate authorization")
	waitForTx("grant update exchange rate auth")

	// =========================================================================
	// Step 4: Test UpdateExchangeRate WITH authorization (expect success)
	// =========================================================================
	fmt.Println("\n--- Step 4: Operator updates exchange rate with auth ---")

	newRate := "2000000"
	updateMsg = &xrtypes.MsgUpdateExchangeRate{
		Authority: policyAddr,
		Operator:  operatorAddr,
		Id:        exchangeRateID,
		Rate:      newRate,
	}

	txResp, err = client.BroadcastTx(ctx, operatorAccount, updateMsg)
	if err != nil {
		return fmt.Errorf("step 4 failed: %w", err)
	}
	if txResp.TxResponse.Code != 0 {
		return fmt.Errorf("step 4 failed with code %d: %s", txResp.TxResponse.Code, txResp.TxResponse.RawLog)
	}
	fmt.Printf("✅ Step 4: UpdateExchangeRate succeeded (new rate: %s)\n", newRate)
	waitForTx("update exchange rate success")

	// =========================================================================
	// Step 5: Query to verify rate was updated
	// =========================================================================
	fmt.Println("\n--- Step 5: Query exchange rate to verify update ---")

	xrQueryClient := xrtypes.NewQueryClient(client.Context())
	getResp, err := xrQueryClient.GetExchangeRate(ctx, &xrtypes.QueryGetExchangeRateRequest{
		Id: exchangeRateID,
	})
	if err != nil {
		return fmt.Errorf("step 5 failed: could not query exchange rate: %w", err)
	}

	if getResp.ExchangeRate.Rate != newRate {
		return fmt.Errorf("step 5 failed: expected rate=%s, got rate=%s", newRate, getResp.ExchangeRate.Rate)
	}

	fmt.Printf("  Rate:    %s (expected: %s)\n", getResp.ExchangeRate.Rate, newRate)
	fmt.Printf("  State:   %v\n", getResp.ExchangeRate.State)
	fmt.Printf("  Updated: %s\n", getResp.ExchangeRate.Updated.Format(time.RFC3339))
	fmt.Println("✅ Step 5: Exchange rate updated successfully")

	// =========================================================================
	// Save results
	// =========================================================================
	result := lib.JourneyResult{
		ExchangeRateID: setup601.ExchangeRateID,
		GroupID:        setup301.GroupID,
		GroupPolicyAddr: policyAddr,
		OperatorAddr:   operatorAddr,
	}
	lib.SaveJourneyResult("journey602", result)

	fmt.Println("\n========================================")
	fmt.Println("Journey 602 completed successfully!")
	fmt.Println("XR UpdateExchangeRate with Operator Authorization tested:")
	fmt.Println("  - UpdateExchangeRate: unauthorized rejected")
	fmt.Println("  - UpdateExchangeRate: authorized succeeded")
	fmt.Println("  - Exchange rate updated and verified")
	fmt.Println("========================================")

	return nil
}
