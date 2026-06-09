package journeys

import (
	"context"
	"fmt"
	"strconv"
	"time"

	authtypes "github.com/cosmos/cosmos-sdk/x/auth/types"
	"github.com/ignite/cli/v28/ignite/pkg/cosmosclient"

	xrtypes "github.com/verana-labs/verana/x/xr/types"

	"github.com/verana-labs/verana/testharness/lib"
)

// RunXrUpdateExchangeRateJourney implements Journey 602: XR Update Exchange Rate (operator).
// Per v4-rc3, MsgUpdateExchangeRate authorizes via an ExchangeRateAuthorization
// (xr_id, operator) granted by governance (MOD-XR-MSG-4), not corporation delegation.
// Flow: update without authz (fail), grant authz via gov (MOD-XR-MSG-4), update
// (succeed), revoke via gov (MOD-XR-MSG-5), update again (fail).
// Depends on Journey 601 (exchange rate ID) and Journey 301 (operator account).
func RunXrUpdateExchangeRateJourney(ctx context.Context, client cosmosclient.Client) error {
	fmt.Println("Starting Journey 602: XR Update Exchange Rate (ExchangeRateAuthorization)")

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

	operatorAddr := setup301.OperatorAddr
	operatorAccount := lib.GetAccount(client, permOperatorName)
	cooluser := lib.GetAccount(client, lib.COOLUSER_NAME)
	govModuleAddr := authtypes.NewModuleAddress("gov").String()

	fmt.Printf("  Exchange Rate ID: %d\n", exchangeRateID)
	fmt.Printf("  Operator:         %s\n", operatorAddr)
	fmt.Printf("  Gov Module:       %s\n", govModuleAddr)
	fmt.Println("✅ Step 1: Loaded journey results")

	// =========================================================================
	// Step 2: Test UpdateExchangeRate WITHOUT authorization (expect failure)
	// =========================================================================
	fmt.Println("\n--- Step 2: Operator updates without ExchangeRateAuthorization (expect failure) ---")

	updateMsg := &xrtypes.MsgUpdateExchangeRate{
		Operator: operatorAddr,
		Id:       exchangeRateID,
		Rate:     "2000000",
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
	// Step 3: Grant ExchangeRateAuthorization via governance (MOD-XR-MSG-4)
	// =========================================================================
	fmt.Println("\n--- Step 3: Grant ExchangeRateAuthorization via governance ---")

	expiration := time.Now().Add(24 * time.Hour)
	grantMsg := &xrtypes.MsgGrantExchangeRateAuthorization{
		Authority:  govModuleAddr,
		XrId:       exchangeRateID,
		Operator:   operatorAddr,
		Expiration: &expiration,
	}

	grantProposalID, err := submitXrGovProposal(
		client, ctx, lib.COOLUSER_ADDRESS, cooluser,
		grantMsg,
		"Grant XR Authorization",
		"Authorize operator to update exchange rate",
	)
	if err != nil {
		return fmt.Errorf("step 3 failed: %w", err)
	}
	if err := voteAndPassGovProposal(client, ctx, grantProposalID); err != nil {
		return fmt.Errorf("step 3 failed: %w", err)
	}
	fmt.Println("✅ Step 3: ExchangeRateAuthorization granted")

	// =========================================================================
	// Step 4: Test UpdateExchangeRate WITH authorization (expect success)
	// =========================================================================
	fmt.Println("\n--- Step 4: Operator updates exchange rate with authorization ---")

	newRate := "2000000"
	updateMsg = &xrtypes.MsgUpdateExchangeRate{
		Operator: operatorAddr,
		Id:       exchangeRateID,
		Rate:     newRate,
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
	// Step 5: Query to verify rate updated and authorization listed
	// =========================================================================
	fmt.Println("\n--- Step 5: Query exchange rate to verify update + authorizations ---")

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
	if len(getResp.Authorizations) == 0 {
		return fmt.Errorf("step 5 failed: expected at least one authorization in response")
	}
	fmt.Printf("  Rate:           %s\n", getResp.ExchangeRate.Rate)
	fmt.Printf("  Authorizations: %d (operator: %s)\n", len(getResp.Authorizations), getResp.Authorizations[0].Operator)
	fmt.Println("✅ Step 5: Exchange rate updated and authorization listed")

	// =========================================================================
	// Step 6: Revoke ExchangeRateAuthorization via governance (MOD-XR-MSG-5)
	// =========================================================================
	fmt.Println("\n--- Step 6: Revoke ExchangeRateAuthorization via governance ---")

	revokeMsg := &xrtypes.MsgRevokeExchangeRateAuthorization{
		Authority: govModuleAddr,
		XrId:      exchangeRateID,
		Operator:  operatorAddr,
	}
	revokeProposalID, err := submitXrGovProposal(
		client, ctx, lib.COOLUSER_ADDRESS, cooluser,
		revokeMsg,
		"Revoke XR Authorization",
		"Revoke operator authorization for exchange rate",
	)
	if err != nil {
		return fmt.Errorf("step 6 failed: %w", err)
	}
	if err := voteAndPassGovProposal(client, ctx, revokeProposalID); err != nil {
		return fmt.Errorf("step 6 failed: %w", err)
	}
	fmt.Println("✅ Step 6: ExchangeRateAuthorization revoked")

	// =========================================================================
	// Step 7: Update again after revoke (expect failure)
	// =========================================================================
	fmt.Println("\n--- Step 7: Operator updates after revoke (expect failure) ---")

	updateMsg = &xrtypes.MsgUpdateExchangeRate{
		Operator: operatorAddr,
		Id:       exchangeRateID,
		Rate:     "3000000",
	}
	txResp, err = client.BroadcastTx(ctx, operatorAccount, updateMsg)
	if err == nil && txResp.TxResponse.Code != 0 {
		err = fmt.Errorf("transaction failed with code %d: %s", txResp.TxResponse.Code, txResp.TxResponse.RawLog)
	}
	if err := expectAuthorizationError("Step 7", err); err != nil {
		return err
	}
	fmt.Println("✅ Step 7: UpdateExchangeRate correctly rejected after revoke")

	// =========================================================================
	// Save results
	// =========================================================================
	result := lib.JourneyResult{
		ExchangeRateID:  setup601.ExchangeRateID,
		GroupID:         setup301.GroupID,
		GroupPolicyAddr: setup301.GroupPolicyAddr,
		OperatorAddr:    operatorAddr,
	}
	lib.SaveJourneyResult("journey602", result)

	fmt.Println("\n========================================")
	fmt.Println("Journey 602 completed successfully!")
	fmt.Println("XR UpdateExchangeRate + ExchangeRateAuthorization tested:")
	fmt.Println("  - UpdateExchangeRate: rejected without authorization")
	fmt.Println("  - GrantExchangeRateAuthorization (gov): granted")
	fmt.Println("  - UpdateExchangeRate: authorized succeeded")
	fmt.Println("  - GetExchangeRate: authorizations listed")
	fmt.Println("  - RevokeExchangeRateAuthorization (gov): revoked")
	fmt.Println("  - UpdateExchangeRate: rejected after revoke")
	fmt.Println("========================================")

	return nil
}
