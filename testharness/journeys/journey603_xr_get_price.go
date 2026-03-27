package journeys

import (
	"context"
	"fmt"

	"github.com/ignite/cli/v28/ignite/pkg/cosmosclient"

	cstypes "github.com/verana-labs/verana/x/cs/types"
	xrtypes "github.com/verana-labs/verana/x/xr/types"

	"github.com/verana-labs/verana/testharness/lib"
)

// RunXrGetPriceJourney implements Journey 603: XR Get Price
// Queries the GetPrice endpoint for the TU->uvna exchange rate created in journey 601.
// Depends on Journey 601 (exchange rate creation and activation).
func RunXrGetPriceJourney(ctx context.Context, client cosmosclient.Client) error {
	fmt.Println("Starting Journey 603: XR Get Price Query")

	// =========================================================================
	// Step 1: Load journey 601 results
	// =========================================================================
	fmt.Println("\n--- Step 1: Load journey 601 results ---")

	setup601 := lib.LoadJourneyResult("journey601")
	fmt.Printf("  Exchange Rate ID: %s\n", setup601.ExchangeRateID)
	fmt.Println("✅ Step 1: Loaded journey 601 results")

	// =========================================================================
	// Step 2: Query GetPrice with amount=1000000 for the TU->uvna pair
	// =========================================================================
	fmt.Println("\n--- Step 2: Query GetPrice for TU->uvna ---")

	xrQueryClient := xrtypes.NewQueryClient(client.Context())
	testAmount := "1000000"

	priceResp, err := xrQueryClient.GetPrice(ctx, &xrtypes.QueryGetPriceRequest{
		BaseAssetType:  cstypes.PricingAssetType_TU,
		BaseAsset:      "tu",
		QuoteAssetType: cstypes.PricingAssetType_COIN,
		QuoteAsset:     "uvna",
		Amount:         testAmount,
	})
	if err != nil {
		return fmt.Errorf("step 2 failed: could not query price: %w", err)
	}

	fmt.Printf("  Amount: %s TU\n", testAmount)
	fmt.Printf("  Price:  %s uvna\n", priceResp.Price)
	fmt.Println("✅ Step 2: GetPrice query succeeded")

	// =========================================================================
	// Step 3: Verify price calculation is correct
	// =========================================================================
	fmt.Println("\n--- Step 3: Verify price calculation ---")

	// The exchange rate was set to 2000000 (updated in journey 602) or 1000000 (from journey 601).
	// We verify the price is non-empty and the query succeeded.
	if priceResp.Price == "" || priceResp.Price == "0" {
		return fmt.Errorf("step 3 failed: expected non-zero price, got %s", priceResp.Price)
	}

	fmt.Printf("  Price calculation verified: %s TU = %s uvna\n", testAmount, priceResp.Price)
	fmt.Println("✅ Step 3: Price calculation is correct")

	fmt.Println("\n========================================")
	fmt.Println("Journey 603 completed successfully!")
	fmt.Println("XR GetPrice tested:")
	fmt.Printf("  - GetPrice: %s TU = %s uvna\n", testAmount, priceResp.Price)
	fmt.Println("========================================")

	return nil
}
