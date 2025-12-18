package td_yield

import (
	"context"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/verana-labs/verana/testharness/lib"

	"cosmossdk.io/math"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/ignite/cli/v28/ignite/pkg/cosmosclient"
)

// RunInsufficientFundingSimulation runs simulation where allowance > YIP per-block funding
// This means YIP receives less than needed, all available is transferred, YIP stays empty
func RunInsufficientFundingSimulation(ctx context.Context, client cosmosclient.Client) error {
	fmt.Println("\n" + strings.Repeat("=", 80))
	fmt.Println("TD Yield Simulation - Insufficient Funding Scenario")
	fmt.Println(strings.Repeat("=", 80))
	fmt.Println("\nScenario: allowance > YIP per-block funding")
	fmt.Println("Expected: All YIP balance is transferred, YIP stays empty")

	metrics := &SimulationMetrics{}

	// ============================================================================
	// STEP 1: Setup test account and grow trust deposit
	// ============================================================================
	fmt.Println("\n📋 Step 1: Setting up test account and growing trust deposit...")

	// Use Issuer_Applicant account (same as sufficient funding sim)
	testAccount := lib.GetAccount(client, lib.ISSUER_APPLICANT_NAME)
	testAccountAddr, err := testAccount.Address(lib.GetAddressPrefix())
	if err != nil {
		return fmt.Errorf("failed to get account address: %v", err)
	}

	// Fund the account
	fmt.Printf("    Funding account %s...\n", testAccountAddr)
	lib.SendFunds(client, ctx, lib.COOLUSER_ADDRESS, testAccountAddr, math.NewInt(50000000)) // 50 VNA
	time.Sleep(2 * time.Second)

	// Create DID with increased years to grow TD (instead of multiple DIDs)
	// Each year adds 5M uvna, so we'll calculate years needed
	did := lib.GenerateUniqueDID(client, ctx)
	fmt.Printf("    Creating DID: %s\n", did)
	// We'll register with years calculated later, for now use 1
	lib.RegisterDID(client, ctx, testAccount, did, 1)

	// Get initial TD module balance IMMEDIATELY after DID registration (before any blocks pass)
	// This is critical for accurate calculations
	tdModuleAddr, err := lib.GetTrustDepositModuleAddress(client, ctx)
	if err != nil {
		return fmt.Errorf("failed to get trust deposit module address: %v", err)
	}

	initialModuleBalance, err := lib.GetBankBalance(client, ctx, tdModuleAddr)
	if err != nil {
		return fmt.Errorf("failed to get trust deposit module balance: %v", err)
	}

	metrics.InitialTDModuleBalance = initialModuleBalance.Amount
	fmt.Printf("    ✅ Initial TD Module Balance (captured immediately): %s\n", initialModuleBalance.String())

	// Get initial user balance IMMEDIATELY (same block as TD module balance)
	initialUserBalance, err := lib.GetBankBalance(client, ctx, testAccountAddr)
	if err != nil {
		return fmt.Errorf("failed to get initial user balance: %v", err)
	}
	fmt.Printf("    ✅ Initial User Balance (captured immediately): %s\n", initialUserBalance.String())

	// Get initial trust deposit
	initialTD, err := lib.GetTrustDeposit(client, ctx, testAccount)
	if err != nil {
		fmt.Printf("    ⚠️  No trust deposit found yet\n")
	} else {
		fmt.Printf("    ✅ Initial Trust Deposit: %d uvna\n", initialTD.Amount)
	}

	// ============================================================================
	// STEP 2: Get initial state
	// ============================================================================
	fmt.Println("\n📊 Step 2: Getting initial state...")

	yipAddr, err := lib.GetYieldIntermediatePoolAddress(client, ctx)
	if err != nil {
		return fmt.Errorf("failed to get YIP address: %v", err)
	}

	params, err := lib.GetTrustDepositParams(client, ctx)
	if err != nil {
		return fmt.Errorf("failed to get trust deposit parameters: %v", err)
	}

	blocksPerYear, err := lib.GetBlocksPerYear(client, ctx)
	if err != nil {
		return fmt.Errorf("failed to get blocks_per_year: %v", err)
	}

	dust, _ := lib.GetDust(client, ctx)

	fmt.Printf("    - Initial Share Value: %s\n", params.TrustDepositShareValue.String())
	fmt.Printf("    - Blocks Per Year: %d\n", blocksPerYear)

	// ============================================================================
	// STEP 2.5: Query YIP per-block incoming amount from events
	// ============================================================================
	fmt.Println("\n📊 Step 2.5: Querying YIP per-block incoming amount from events...")
	fmt.Println("    Note: YIP balance monitoring won't work because excess is returned to protocol pool in the same block")
	fmt.Println("    Using yield_distribution events to get accurate YIP incoming amounts...")

	// Get current block height
	currentHeight, err := lib.GetLatestBlockHeight(client, ctx)
	if err != nil {
		return fmt.Errorf("failed to get current block height: %v", err)
	}
	fmt.Printf("    Current block height: %d\n", currentHeight)

	// Wait for a few blocks to accumulate events, then query them
	fmt.Println("    Waiting for a few blocks to accumulate events...")
	time.Sleep(15 * time.Second) // Wait for ~3 blocks

	// Query recent blocks to get YIP incoming amounts
	monitorBlocksForYIP := 5
	var yipIncomingAmounts []math.Int

	for i := 0; i < monitorBlocksForYIP; i++ {
		blockHeight := currentHeight + int64(i) + 1
		amount, err := lib.GetYIPIncomingAmountFromBlockResults(blockHeight)
		if err == nil {
			yipIncomingAmounts = append(yipIncomingAmounts, amount)
			fmt.Printf("    [Block %d] YIP incoming amount: %s\n", blockHeight, sdk.NewCoin("uvna", amount).String())
		} else {
			fmt.Printf("    [Block %d] Could not query events: %v\n", blockHeight, err)
		}
		time.Sleep(2 * time.Second) // Small delay between queries
	}

	if len(yipIncomingAmounts) == 0 {
		return fmt.Errorf("could not query any YIP incoming amounts from events")
	}

	// Calculate average YIP per-block amount (they should be similar)
	var totalAmount math.Int = math.ZeroInt()
	for _, amount := range yipIncomingAmounts {
		totalAmount = totalAmount.Add(amount)
	}
	yipPerBlockAmount := totalAmount.Quo(math.NewInt(int64(len(yipIncomingAmounts))))

	fmt.Printf("    ✅ Average YIP per-block incoming amount (from %d blocks): %s\n", len(yipIncomingAmounts), sdk.NewCoin("uvna", yipPerBlockAmount).String())

	// ============================================================================
	// STEP 2.6: Grow TD to make allowance > YIP per-block funding
	// ============================================================================
	fmt.Println("\n📈 Step 2.6: Growing trust deposit to ensure allowance > YIP per-block funding...")

	// Target: allowance should be at least 1.5x YIP per-block amount to ensure it's greater
	targetAllowanceDec := math.LegacyNewDecFromInt(yipPerBlockAmount).Mul(math.LegacyNewDecWithPrec(15, 1)) // 1.5x

	// Calculate required TD balance: allowance = dust + TD * max_yield_rate / blocks_per_year
	// TD = (allowance - dust) * blocks_per_year / max_yield_rate
	blocksPerYearDec := math.LegacyNewDec(int64(blocksPerYear))
	requiredTDDec := targetAllowanceDec.Sub(dust).Mul(blocksPerYearDec).Quo(params.TrustDepositMaxYieldRate)
	requiredTDInt := requiredTDDec.TruncateInt()

	fmt.Printf("    - Target allowance: %s (1.5x YIP per-block: %s)\n", targetAllowanceDec.String(), sdk.NewCoin("uvna", yipPerBlockAmount).String())
	fmt.Printf("    - Required TD balance: %s\n", sdk.NewCoin("uvna", requiredTDInt).String())

	// Get current TD balance
	currentTD, err := lib.GetTrustDeposit(client, ctx, testAccount)
	if err != nil {
		return fmt.Errorf("failed to get current trust deposit: %v", err)
	}

	currentTDInt := math.NewInt(int64(currentTD.Amount))
	currentTDModuleBalance, err := lib.GetBankBalance(client, ctx, tdModuleAddr)
	if err != nil {
		return fmt.Errorf("failed to get TD module balance: %v", err)
	}

	fmt.Printf("    - Current TD balance: %d uvna\n", currentTD.Amount)
	fmt.Printf("    - Current TD module balance: %s\n", currentTDModuleBalance.String())

	// Calculate how much more TD we need
	neededTDInt := requiredTDInt.Sub(currentTDInt)
	if neededTDInt.IsPositive() {
		fmt.Printf("    - Need to add: %s to TD\n", sdk.NewCoin("uvna", neededTDInt).String())

		// Check if we need more than 30 VNA (30M uvna) - if so, something is wrong with the funding percentage
		maxReasonableTD := math.NewInt(30_000_000) // 30 VNA
		if neededTDInt.GT(maxReasonableTD) {
			return fmt.Errorf("required TD (%s) exceeds reasonable limit (30 VNA). Funding percentage may be too high. Current TD: %d uvna, Required: %s",
				sdk.NewCoin("uvna", requiredTDInt).String(), currentTD.Amount, sdk.NewCoin("uvna", requiredTDInt).String())
		}

		// Calculate years needed: Each year adds 5M uvna (5,000,000 uvna)
		// years = ceil(neededTD / 5M)
		yearsNeeded := neededTDInt.Quo(math.NewInt(5000000)).Int64()
		if neededTDInt.Mod(math.NewInt(5000000)).GT(math.ZeroInt()) {
			yearsNeeded++ // Round up
		}
		yearsNeeded++ // +1 to ensure we exceed

		// Limit to maximum 6 years (30 VNA total: 5M * 6 = 30M)
		maxYears := int64(6)
		if yearsNeeded > maxYears {
			fmt.Printf("    ⚠️  Warning: Would need %d years, but limiting to %d years (30 VNA max)\n", yearsNeeded, maxYears)
			yearsNeeded = maxYears
		}

		fmt.Printf("    - Registering DID with %d years to grow TD (5 VNA per year)...\n", yearsNeeded)
		newDID := lib.GenerateUniqueDID(client, ctx)
		fmt.Printf("      Creating DID: %s with %d years\n", newDID, yearsNeeded)
		lib.RegisterDID(client, ctx, testAccount, newDID, uint32(yearsNeeded))
		time.Sleep(2 * time.Second)

		// Update TD module balance after growing TD
		updatedTDModuleBalance, err := lib.GetBankBalance(client, ctx, tdModuleAddr)
		if err == nil {
			initialModuleBalance = updatedTDModuleBalance
			metrics.InitialTDModuleBalance = updatedTDModuleBalance.Amount
			fmt.Printf("    ✅ Updated initial TD module balance: %s\n", updatedTDModuleBalance.String())
		}
	} else {
		fmt.Printf("    ✅ Current TD is already sufficient (allowance will be > YIP per-block)\n")
	}

	// Recalculate allowance with updated TD
	updatedTDModuleBalance, _ := lib.GetBankBalance(client, ctx, tdModuleAddr)
	allowance := lib.CalculateYieldAllowance(
		updatedTDModuleBalance.Amount,
		params.TrustDepositMaxYieldRate,
		blocksPerYear,
		dust,
	)
	fmt.Printf("    - Updated Allowance per block: %s\n", allowance.String())
	fmt.Printf("    - YIP per-block incoming: %s\n", sdk.NewCoin("uvna", yipPerBlockAmount).String())

	if allowance.LTE(math.LegacyNewDecFromInt(yipPerBlockAmount)) {
		return fmt.Errorf("allowance (%s) is not greater than YIP per-block (%s). TD growth may not have been sufficient", allowance.String(), sdk.NewCoin("uvna", yipPerBlockAmount).String())
	}
	fmt.Printf("    ✅ Scenario verified: allowance (%s) > YIP per-block (%s)\n", allowance.String(), sdk.NewCoin("uvna", yipPerBlockAmount).String())

	// ============================================================================
	// STEP 3: Monitor blocks and reclaim yield as soon as it's available
	// ============================================================================
	fmt.Println("\n⏳ Step 3: Monitoring blocks and reclaiming yield as it accumulates...")
	fmt.Println("    Will start reclaiming as soon as yield is detected (>= 1 uvna)")
	fmt.Println("    Note: In this scenario, YIP should stay near-empty (all transferred)")

	monitorBlocks := 50
	checkInterval := 5 * time.Second

	var previousModuleBalance math.Int = initialModuleBalance.Amount
	totalTransferred := math.ZeroInt()
	yipEmptyCount := 0
	reclaimAttempts := 0
	maxReclaims := 20

	// Track actual vs expected yield for comparison
	var actualYieldAmounts []math.Int
	var expectedYieldPerBlock math.LegacyDec = allowance // Expected yield = allowance per block

	for i := 0; i < monitorBlocks; i++ {
		time.Sleep(checkInterval)

		currentModuleBalance, err := lib.GetBankBalance(client, ctx, tdModuleAddr)
		if err != nil {
			continue
		}

		yipBalance, err := lib.GetBankBalance(client, ctx, yipAddr)
		if err == nil {
			if yipBalance.Amount.LT(math.NewInt(1000)) { // Near empty (< 1000 uvna)
				yipEmptyCount++
			}
		}

		// Calculate how much was transferred this block
		transferAmount := currentModuleBalance.Amount.Sub(previousModuleBalance)
		if transferAmount.IsPositive() {
			totalTransferred = totalTransferred.Add(transferAmount)
		}

		previousModuleBalance = currentModuleBalance.Amount

		// Attempt reclaim if we haven't hit the max yet
		if reclaimAttempts < maxReclaims {
			// Try to reclaim yield
			txResp, blockHeight, err := lib.ReclaimTrustDepositYieldWithResponse(client, ctx, testAccount)
			if err == nil && txResp != nil && txResp.ClaimedAmount > 0 {
				claimedAmount := txResp.ClaimedAmount
				metrics.TotalReclaimed += claimedAmount
				metrics.ReclaimCount++
				reclaimAttempts++

				// Query state at the block BEFORE reclaim (blockHeight - 1)
				blockBeforeReclaim := blockHeight - 1
				if blockBeforeReclaim < 1 {
					blockBeforeReclaim = 1
				}

				userBalanceBefore, _ := lib.GetBankBalanceAtHeight(client, ctx, testAccountAddr, blockBeforeReclaim)
				tdModuleBalanceBefore, _ := lib.GetBankBalanceAtHeight(client, ctx, tdModuleAddr, blockBeforeReclaim)

				// Query state at the block OF reclaim (blockHeight)
				userBalanceAtReclaim, _ := lib.GetBankBalanceAtHeight(client, ctx, testAccountAddr, blockHeight)
				tdModuleBalanceAtReclaim, _ := lib.GetBankBalanceAtHeight(client, ctx, tdModuleAddr, blockHeight)

				// Calculate balance changes: Block N - Block N-1
				userBalanceChange := userBalanceAtReclaim.Amount.Sub(userBalanceBefore.Amount)
				tdModuleBalanceChange := tdModuleBalanceAtReclaim.Amount.Sub(tdModuleBalanceBefore.Amount)

				// Get BeginBlock transfer amount from block events
				beginBlockTransfer, errBeginBlock := lib.GetBeginBlockTransferAmountFromBlockResults(blockHeight)
				if errBeginBlock != nil {
					beginBlockTransfer = math.ZeroInt() // If we can't get it, assume 0
				}

				// Track actual yield received for comparison
				if beginBlockTransfer.IsPositive() {
					actualYieldAmounts = append(actualYieldAmounts, beginBlockTransfer)
				}

				// Calculate intermediate state: TD balance after BeginBlock (before reclaim)
				tdModuleBalanceAfterBeginBlock := tdModuleBalanceBefore.Amount.Add(beginBlockTransfer)

				// Calculate what reclaim actually removed from TD module
				// TD balance after reclaim = TD balance after BeginBlock - Reclaim removal
				// So: Reclaim removal = TD balance after BeginBlock - TD balance after reclaim
				reclaimRemoval := tdModuleBalanceAfterBeginBlock.Sub(tdModuleBalanceAtReclaim.Amount)

				// Format balance changes as strings (can be negative, no need for Coin)
				userBalanceChangeStr := ""
				if userBalanceChange.IsNegative() {
					userBalanceChangeStr = fmt.Sprintf("-%suvna", userBalanceChange.Neg().String())
				} else {
					userBalanceChangeStr = fmt.Sprintf("+%suvna", userBalanceChange.String())
				}

				tdModuleBalanceChangeStr := ""
				if tdModuleBalanceChange.IsNegative() {
					tdModuleBalanceChangeStr = fmt.Sprintf("-%suvna", tdModuleBalanceChange.Neg().String())
				} else {
					tdModuleBalanceChangeStr = fmt.Sprintf("+%suvna", tdModuleBalanceChange.String())
				}

				fmt.Printf("\n    ✅ Reclaim %d: Claimed %d uvna at block height %d\n", reclaimAttempts, claimedAmount, blockHeight)
				fmt.Printf("    📊 Reclaim Details:\n")
				fmt.Printf("      1. User Balance:\n")
				fmt.Printf("         - Block %d (end): %s\n", blockBeforeReclaim, userBalanceBefore.String())
				fmt.Printf("         - Block %d (after reclaim): %s\n", blockHeight, userBalanceAtReclaim.String())
				fmt.Printf("         - Balance Change: %s\n", userBalanceChangeStr)
				fmt.Printf("      2. TD Module Balance:\n")
				fmt.Printf("         - Block %d (end): %s\n", blockBeforeReclaim, tdModuleBalanceBefore.String())
				fmt.Printf("         - Block %d (after BeginBlock, before reclaim): %suvna\n", blockHeight, tdModuleBalanceAfterBeginBlock.String())
				fmt.Printf("         - Block %d (after reclaim): %s\n", blockHeight, tdModuleBalanceAtReclaim.String())
				fmt.Printf("         - BeginBlock Addition: +%suvna\n", beginBlockTransfer.String())
				fmt.Printf("         - Reclaim Removal: -%suvna\n", reclaimRemoval.String())
				fmt.Printf("         - Net Change (Block %d → Block %d): %s\n", blockBeforeReclaim, blockHeight, tdModuleBalanceChangeStr)
				fmt.Printf("      3. Total Transferred from TD Module (cumulative reclaims): %d uvna\n", metrics.TotalReclaimed)

				// Verify invariants at reclaim block (get params first for yield comparison)
				paramsAtReclaim, errParams := lib.GetTrustDepositParamsAtHeight(client, ctx, blockHeight)

				// Show yield comparison: Actual vs Expected
				if beginBlockTransfer.IsPositive() {
					actualYieldDec := math.LegacyNewDecFromInt(beginBlockTransfer)
					shortfall := expectedYieldPerBlock.Sub(actualYieldDec)

					// Calculate actual annual yield rate: (actual_yield_per_block * blocks_per_year) / TD_balance
					blocksPerYearDec := math.LegacyNewDec(int64(blocksPerYear))
					actualAnnualYieldRate := actualYieldDec.Mul(blocksPerYearDec).Quo(math.LegacyNewDecFromInt(tdModuleBalanceBefore.Amount))
					actualAnnualYieldRatePercent := actualAnnualYieldRate.Mul(math.LegacyNewDec(100))

					// Max allowed annual yield rate (use paramsAtReclaim if available, otherwise use initial params)
					maxYieldRate := params.TrustDepositMaxYieldRate
					if paramsAtReclaim != nil {
						maxYieldRate = paramsAtReclaim.TrustDepositMaxYieldRate
					}
					// maxYieldRate is stored as decimal (0.15 for 15%), convert to percentage
					// Use string conversion to avoid precision issues with MustFloat64()
					maxYieldRateStr := maxYieldRate.String()
					maxYieldRateFloat, err := strconv.ParseFloat(maxYieldRateStr, 64)
					if err != nil {
						maxYieldRateFloat = 0.15 // Fallback to default if parsing fails
					}
					maxAnnualYieldRatePercentFloat := maxYieldRateFloat * 100.0
					maxAnnualYieldRatePercent := maxYieldRate.Mul(math.LegacyNewDec(100))

					// Yield rate as percentage of max allowed
					yieldRateOfMaxPercent := actualAnnualYieldRatePercent.Quo(maxAnnualYieldRatePercent).Mul(math.LegacyNewDec(100))

					fmt.Printf("      4. Yield Comparison (Block %d):\n", blockHeight)
					fmt.Printf("         - Expected Yield per block: %suvna (based on allowance)\n", expectedYieldPerBlock.TruncateInt().String())
					fmt.Printf("         - Actual Yield received: %suvna (from YIP transfer)\n", beginBlockTransfer.String())
					fmt.Printf("         - Actual Annual Yield Rate: %.4f%% (actual_yield * blocks_per_year / TD_balance)\n", actualAnnualYieldRatePercent.MustFloat64())
					fmt.Printf("         - Max Allowed Annual Yield Rate: %.2f%%\n", maxAnnualYieldRatePercentFloat)
					fmt.Printf("         - Yield Rate: %.2f%% of max allowed (actual/max)\n", yieldRateOfMaxPercent.MustFloat64())
					fmt.Printf("         - Shortfall: %suvna per block (expected - actual)\n", shortfall.TruncateInt().String())
					fmt.Printf("         - Note: Actual < Expected because allowance > YIP per-block funding\n")
				}
				allTDsAtReclaim, errTDs := lib.GetAllKnownTrustDepositsAtHeight(client, ctx, blockHeight)
				if errParams == nil && errTDs == nil && paramsAtReclaim != nil && len(allTDsAtReclaim) > 0 {
					finalShareValue := paramsAtReclaim.TrustDepositShareValue
					if finalShareValue.IsNil() {
						fmt.Printf("      ⚠️  Cannot verify invariants: share value is nil\n")
					} else {
						var sumShareValue math.LegacyDec = math.LegacyZeroDec()
						var sumAmount uint64 = 0
						for _, td := range allTDsAtReclaim {
							if td.TD == nil {
								continue
							}
							if td.TD.Share.IsNil() {
								continue
							}
							shareValue := td.TD.Share.Mul(finalShareValue)
							sumShareValue = sumShareValue.Add(shareValue)
							sumAmount += td.TD.Amount
						}

						fmt.Printf("      🔒 Invariant Check at Block %d:\n", blockHeight)
						if tdModuleBalanceAtReclaim.Amount.GTE(sumShareValue.TruncateInt()) {
							fmt.Printf("         ✅ Invariant 1 HOLDS: module_balance (%s) >= sum(share * shareValue) (%s)\n",
								tdModuleBalanceAtReclaim.String(), sdk.NewCoin("uvna", sumShareValue.TruncateInt()).String())
						} else {
							fmt.Printf("         ❌ Invariant 1 VIOLATED: module_balance (%s) < sum(share * shareValue) (%s)\n",
								tdModuleBalanceAtReclaim.String(), sdk.NewCoin("uvna", sumShareValue.TruncateInt()).String())
						}

						if tdModuleBalanceAtReclaim.Amount.GTE(math.NewInt(int64(sumAmount))) {
							fmt.Printf("         ✅ Invariant 2 HOLDS: module_balance (%s) >= sum(amount) (%d uvna)\n",
								tdModuleBalanceAtReclaim.String(), sumAmount)
						} else {
							fmt.Printf("         ❌ Invariant 2 VIOLATED: module_balance (%s) < sum(amount) (%d uvna)\n",
								tdModuleBalanceAtReclaim.String(), sumAmount)
						}
					}
				}
			} else if err != nil && !strings.Contains(err.Error(), "no claimable yield available") {
				// Only log non-truncation errors
				fmt.Printf("      ⚠️  Reclaim failed: %v\n", err)
			}
		}

		// Exit early if we've reached max reclaims
		if reclaimAttempts >= maxReclaims {
			fmt.Printf("    ✅ Reached max reclaims (%d), exiting monitoring loop early (at block %d/%d)\n", maxReclaims, i+1, monitorBlocks)
			metrics.BlocksMonitored = i + 1
			break
		}

		// Log progress every 10 blocks
		if (i+1)%10 == 0 {
			// Calculate average actual yield and annual yield rate if we have data
			avgActualYieldStr := "N/A"
			if len(actualYieldAmounts) > 0 {
				var totalActual math.Int = math.ZeroInt()
				for _, amt := range actualYieldAmounts {
					totalActual = totalActual.Add(amt)
				}
				avgActual := totalActual.Quo(math.NewInt(int64(len(actualYieldAmounts))))
				avgActualDec := math.LegacyNewDecFromInt(avgActual)

				// Calculate actual annual yield rate
				blocksPerYearDec := math.LegacyNewDec(int64(blocksPerYear))
				actualAnnualYieldRate := avgActualDec.Mul(blocksPerYearDec).Quo(math.LegacyNewDecFromInt(currentModuleBalance.Amount))
				actualAnnualYieldRatePercent := actualAnnualYieldRate.Mul(math.LegacyNewDec(100))
				// maxYieldRate is stored as decimal (0.15 for 15%), convert to percentage
				// Convert to float64 first to avoid precision issues
				maxYieldRateFloat := params.TrustDepositMaxYieldRate.MustFloat64()
				maxAnnualYieldRatePercentFloat := maxYieldRateFloat * 100.0
				maxAnnualYieldRatePercent := params.TrustDepositMaxYieldRate.Mul(math.LegacyNewDec(100))
				yieldRateOfMaxPercent := actualAnnualYieldRatePercent.Quo(maxAnnualYieldRatePercent).Mul(math.LegacyNewDec(100))

				avgActualYieldStr = fmt.Sprintf("%suvna (%.2f%% of max allowed)", avgActual.String(), yieldRateOfMaxPercent.MustFloat64())
				// Use the float64 value for display
				_ = maxAnnualYieldRatePercentFloat // Use this for display if needed
			}

			fmt.Printf("    Block %d/%d: TD Module Balance: %s, YIP: %s, Total Transferred: %s, Reclaims: %d\n",
				i+1, monitorBlocks, currentModuleBalance.String(), yipBalance.String(), sdk.NewCoin("uvna", totalTransferred).String(), reclaimAttempts)
			fmt.Printf("      📊 Yield Rate: Expected: %suvna/block, Avg Actual: %s\n",
				expectedYieldPerBlock.TruncateInt().String(), avgActualYieldStr)
		}
	}

	// Calculate TotalSentToTDModule from actual BeginBlock transfers (more accurate than balance changes)
	var totalActualTransfers math.Int = math.ZeroInt()
	for _, amt := range actualYieldAmounts {
		totalActualTransfers = totalActualTransfers.Add(amt)
	}

	// Use actual transfers if available, otherwise fall back to balance-based calculation
	if totalActualTransfers.IsPositive() {
		metrics.TotalSentToTDModule = totalActualTransfers
		fmt.Printf("\n    ✅ Monitoring complete: %d blocks monitored\n", monitorBlocks)
		fmt.Printf("    ✅ Total transferred to TD module (from BeginBlock events): %s\n", sdk.NewCoin("uvna", totalActualTransfers).String())
		fmt.Printf("    ⚠️  Note: Balance-based calculation was %s (may be inaccurate due to reclaims in same blocks)\n", sdk.NewCoin("uvna", totalTransferred).String())
	} else {
		metrics.TotalSentToTDModule = totalTransferred
		fmt.Printf("\n    ✅ Monitoring complete: %d blocks monitored\n", monitorBlocks)
		fmt.Printf("    ✅ Total transferred to TD module (balance-based): %s\n", sdk.NewCoin("uvna", totalTransferred).String())
		fmt.Printf("    ⚠️  Note: Using balance-based calculation (no BeginBlock events captured)\n")
	}

	// Only update BlocksMonitored if we didn't break early (it was already set in the break case)
	if metrics.BlocksMonitored == 0 {
		metrics.BlocksMonitored = monitorBlocks
	}
	fmt.Printf("    ✅ YIP was empty/near-empty for %d/%d blocks (%.1f%%)\n",
		yipEmptyCount, monitorBlocks, float64(yipEmptyCount)/float64(monitorBlocks)*100)
	fmt.Printf("    ✅ Total reclaims performed: %d\n", reclaimAttempts)

	// Continue reclaiming if we haven't hit the max yet
	if reclaimAttempts < maxReclaims {
		fmt.Println("\n💰 Step 4: Continuing to reclaim remaining yield...")

		for reclaimAttempts < maxReclaims {
			// Try to reclaim yield
			txResp, blockHeight, err := lib.ReclaimTrustDepositYieldWithResponse(client, ctx, testAccount)
			if err != nil {
				if strings.Contains(err.Error(), "no claimable yield available") {
					fmt.Printf("    ⚠️  No claimable yield available (truncation may have reduced it to 0), waiting more...\n")
					time.Sleep(30 * time.Second)
					continue
				}
				return fmt.Errorf("failed to reclaim yield: %v", err)
			}

			if txResp == nil || txResp.ClaimedAmount == 0 {
				fmt.Printf("    ⚠️  No yield claimed, waiting for more accumulation...\n")
				time.Sleep(30 * time.Second)
				continue
			}

			claimedAmount := txResp.ClaimedAmount
			metrics.TotalReclaimed += claimedAmount
			metrics.ReclaimCount++
			reclaimAttempts++

			// Query state at the block BEFORE reclaim (blockHeight - 1)
			blockBeforeReclaim := blockHeight - 1
			if blockBeforeReclaim < 1 {
				blockBeforeReclaim = 1
			}

			userBalanceBefore, _ := lib.GetBankBalanceAtHeight(client, ctx, testAccountAddr, blockBeforeReclaim)
			tdModuleBalanceBefore, _ := lib.GetBankBalanceAtHeight(client, ctx, tdModuleAddr, blockBeforeReclaim)

			// Query state at the block OF reclaim (blockHeight)
			userBalanceAtReclaim, _ := lib.GetBankBalanceAtHeight(client, ctx, testAccountAddr, blockHeight)
			tdModuleBalanceAtReclaim, _ := lib.GetBankBalanceAtHeight(client, ctx, tdModuleAddr, blockHeight)

			// Calculate balance changes: Block N - Block N-1
			userBalanceChange := userBalanceAtReclaim.Amount.Sub(userBalanceBefore.Amount)
			tdModuleBalanceChange := tdModuleBalanceAtReclaim.Amount.Sub(tdModuleBalanceBefore.Amount)

			// Get BeginBlock transfer amount from block events
			beginBlockTransfer, errBeginBlock := lib.GetBeginBlockTransferAmountFromBlockResults(blockHeight)
			if errBeginBlock != nil {
				beginBlockTransfer = math.ZeroInt() // If we can't get it, assume 0
			}

			// Track actual yield received for comparison
			if beginBlockTransfer.IsPositive() {
				actualYieldAmounts = append(actualYieldAmounts, beginBlockTransfer)
			}

			// Calculate intermediate state: TD balance after BeginBlock (before reclaim)
			tdModuleBalanceAfterBeginBlock := tdModuleBalanceBefore.Amount.Add(beginBlockTransfer)

			// Calculate what reclaim actually removed from TD module
			// TD balance after reclaim = TD balance after BeginBlock - Reclaim removal
			// So: Reclaim removal = TD balance after BeginBlock - TD balance after reclaim
			reclaimRemoval := tdModuleBalanceAfterBeginBlock.Sub(tdModuleBalanceAtReclaim.Amount)

			// Format balance changes as strings (can be negative, no need for Coin)
			userBalanceChangeStr := ""
			if userBalanceChange.IsNegative() {
				userBalanceChangeStr = fmt.Sprintf("-%suvna", userBalanceChange.Neg().String())
			} else {
				userBalanceChangeStr = fmt.Sprintf("+%suvna", userBalanceChange.String())
			}

			tdModuleBalanceChangeStr := ""
			if tdModuleBalanceChange.IsNegative() {
				tdModuleBalanceChangeStr = fmt.Sprintf("-%suvna", tdModuleBalanceChange.Neg().String())
			} else {
				tdModuleBalanceChangeStr = fmt.Sprintf("+%suvna", tdModuleBalanceChange.String())
			}

			fmt.Printf("\n    ✅ Reclaim %d: Claimed %d uvna at block height %d\n", reclaimAttempts, claimedAmount, blockHeight)
			fmt.Printf("    📊 Reclaim Details:\n")
			fmt.Printf("      1. User Balance:\n")
			fmt.Printf("         - Block %d (end): %s\n", blockBeforeReclaim, userBalanceBefore.String())
			fmt.Printf("         - Block %d (after reclaim): %s\n", blockHeight, userBalanceAtReclaim.String())
			fmt.Printf("         - Balance Change: %s\n", userBalanceChangeStr)
			fmt.Printf("      2. TD Module Balance:\n")
			fmt.Printf("         - Block %d (end): %s\n", blockBeforeReclaim, tdModuleBalanceBefore.String())
			fmt.Printf("         - Block %d (after BeginBlock, before reclaim): %suvna\n", blockHeight, tdModuleBalanceAfterBeginBlock.String())
			fmt.Printf("         - Block %d (after reclaim): %s\n", blockHeight, tdModuleBalanceAtReclaim.String())
			fmt.Printf("         - BeginBlock Addition: +%suvna\n", beginBlockTransfer.String())
			fmt.Printf("         - Reclaim Removal: -%suvna\n", reclaimRemoval.String())
			fmt.Printf("         - Net Change (Block %d → Block %d): %s\n", blockBeforeReclaim, blockHeight, tdModuleBalanceChangeStr)
			fmt.Printf("      3. Total Transferred from TD Module (cumulative reclaims): %d uvna\n", metrics.TotalReclaimed)

			// Verify invariants at reclaim block (get params first for yield comparison)
			paramsAtReclaim, errParams := lib.GetTrustDepositParamsAtHeight(client, ctx, blockHeight)

			// Show yield comparison: Actual vs Expected
			if beginBlockTransfer.IsPositive() {
				actualYieldDec := math.LegacyNewDecFromInt(beginBlockTransfer)
				shortfall := expectedYieldPerBlock.Sub(actualYieldDec)

				// Calculate actual annual yield rate: (actual_yield_per_block * blocks_per_year) / TD_balance
				blocksPerYearDec := math.LegacyNewDec(int64(blocksPerYear))
				actualAnnualYieldRate := actualYieldDec.Mul(blocksPerYearDec).Quo(math.LegacyNewDecFromInt(tdModuleBalanceBefore.Amount))
				actualAnnualYieldRatePercent := actualAnnualYieldRate.Mul(math.LegacyNewDec(100))

				// Yield rate as percentage of max allowed

				fmt.Printf("      4. Yield Comparison (Block %d):\n", blockHeight)
				fmt.Printf("         - Expected Yield per block: %suvna (based on allowance)\n", expectedYieldPerBlock.TruncateInt().String())
				fmt.Printf("         - Actual Yield received: %suvna (from YIP transfer)\n", beginBlockTransfer.String())
				fmt.Printf("         - Actual Annual Yield Rate: %.4f%% (actual_yield * blocks_per_year / TD_balance)\n", actualAnnualYieldRatePercent.MustFloat64())
				fmt.Printf("         - Max Allowed Annual Yield Rate: %.2f%%\n", 15.0)
				fmt.Printf("         - Shortfall: %suvna per block (expected - actual)\n", shortfall.TruncateInt().String())
				fmt.Printf("         - Note: Actual < Expected because allowance > YIP per-block funding\n")
			}
			allTDsAtReclaim, errTDs := lib.GetAllKnownTrustDepositsAtHeight(client, ctx, blockHeight)
			if errParams == nil && errTDs == nil && paramsAtReclaim != nil && len(allTDsAtReclaim) > 0 {
				finalShareValue := paramsAtReclaim.TrustDepositShareValue
				if finalShareValue.IsNil() {
					fmt.Printf("      ⚠️  Cannot verify invariants: share value is nil\n")
				} else {
					var sumShareValue math.LegacyDec = math.LegacyZeroDec()
					var sumAmount uint64 = 0
					for _, td := range allTDsAtReclaim {
						if td.TD == nil {
							continue
						}
						if td.TD.Share.IsNil() {
							continue
						}
						shareValue := td.TD.Share.Mul(finalShareValue)
						sumShareValue = sumShareValue.Add(shareValue)
						sumAmount += td.TD.Amount
					}

					fmt.Printf("      🔒 Invariant Check at Block %d:\n", blockHeight)
					if tdModuleBalanceAtReclaim.Amount.GTE(sumShareValue.TruncateInt()) {
						fmt.Printf("         ✅ Invariant 1 HOLDS: module_balance (%s) >= sum(share * shareValue) (%s)\n",
							tdModuleBalanceAtReclaim.String(), sdk.NewCoin("uvna", sumShareValue.TruncateInt()).String())
					} else {
						fmt.Printf("         ❌ Invariant 1 VIOLATED: module_balance (%s) < sum(share * shareValue) (%s)\n",
							tdModuleBalanceAtReclaim.String(), sdk.NewCoin("uvna", sumShareValue.TruncateInt()).String())
					}

					if tdModuleBalanceAtReclaim.Amount.GTE(math.NewInt(int64(sumAmount))) {
						fmt.Printf("         ✅ Invariant 2 HOLDS: module_balance (%s) >= sum(amount) (%d uvna)\n",
							tdModuleBalanceAtReclaim.String(), sumAmount)
					} else {
						fmt.Printf("         ❌ Invariant 2 VIOLATED: module_balance (%s) < sum(amount) (%d uvna)\n",
							tdModuleBalanceAtReclaim.String(), sumAmount)
					}
				}
			}

			// Wait before next reclaim (removed unnecessary 50 block wait as requested)
			time.Sleep(30 * time.Second)
		}
	}

	// ============================================================================
	// STEP 5: Final state and verification
	// ============================================================================
	fmt.Println("\n📊 Step 5: Final state and verification...")

	finalModuleBalance, err := lib.GetBankBalance(client, ctx, tdModuleAddr)
	if err != nil {
		return fmt.Errorf("failed to get final module balance: %v", err)
	}

	metrics.FinalTDModuleBalance = finalModuleBalance.Amount

	// Calculate net change in TD module
	netChange := finalModuleBalance.Amount.Sub(initialModuleBalance.Amount)

	fmt.Printf("\n    📈 Simulation Metrics:\n")
	fmt.Printf("      - Blocks Monitored: %d\n", metrics.BlocksMonitored)
	fmt.Printf("      - Total Sent to TD Module: %s\n", sdk.NewCoin("uvna", metrics.TotalSentToTDModule).String())
	fmt.Printf("      - Total Reclaimed by Users: %d uvna\n", metrics.TotalReclaimed)
	fmt.Printf("      - Reclaim Count: %d\n", metrics.ReclaimCount)
	fmt.Printf("      - Initial TD Module Balance: %s\n", sdk.NewCoin("uvna", metrics.InitialTDModuleBalance).String())
	fmt.Printf("      - Final TD Module Balance: %s\n", sdk.NewCoin("uvna", metrics.FinalTDModuleBalance).String())

	// Calculate and display yield rate comparison
	if len(actualYieldAmounts) > 0 {
		var totalActual math.Int = math.ZeroInt()
		for _, amt := range actualYieldAmounts {
			totalActual = totalActual.Add(amt)
		}
		avgActualYield := totalActual.Quo(math.NewInt(int64(len(actualYieldAmounts))))
		avgActualYieldDec := math.LegacyNewDecFromInt(avgActualYield)
		totalExpectedYield := expectedYieldPerBlock.Mul(math.LegacyNewDec(int64(len(actualYieldAmounts))))
		totalShortfall := totalExpectedYield.Sub(math.LegacyNewDecFromInt(totalActual))

		// Calculate actual annual yield rate: (avg_actual_yield_per_block * blocks_per_year) / TD_balance
		blocksPerYearDec := math.LegacyNewDec(int64(blocksPerYear))
		avgTDBalance := initialModuleBalance.Amount.Add(finalModuleBalance.Amount).Quo(math.NewInt(2)) // Average TD balance
		actualAnnualYieldRate := avgActualYieldDec.Mul(blocksPerYearDec).Quo(math.LegacyNewDecFromInt(avgTDBalance))
		actualAnnualYieldRatePercent := actualAnnualYieldRate.Mul(math.LegacyNewDec(100))

		// Max allowed annual yield rate
		// maxYieldRate is stored as decimal (0.15 for 15%), convert to percentage
		// Use string conversion to avoid precision issues with MustFloat64()
		maxYieldRateStr := params.TrustDepositMaxYieldRate.String()
		maxYieldRateFloat, err := strconv.ParseFloat(maxYieldRateStr, 64)
		if err != nil {
			maxYieldRateFloat = 0.15 // Fallback to default if parsing fails
		}
		maxAnnualYieldRatePercentFloat := maxYieldRateFloat * 100.0
		maxAnnualYieldRatePercent := params.TrustDepositMaxYieldRate.Mul(math.LegacyNewDec(100))

		// Yield rate as percentage of max allowed
		yieldRateOfMaxPercent := actualAnnualYieldRatePercent.Quo(maxAnnualYieldRatePercent).Mul(math.LegacyNewDec(100))

		fmt.Printf("\n    📊 Yield Rate Analysis:\n")
		fmt.Printf("      - Expected Yield per block: %suvna (allowance)\n", expectedYieldPerBlock.TruncateInt().String())
		fmt.Printf("      - Average Actual Yield per block: %suvna (from %d blocks)\n", avgActualYield.String(), len(actualYieldAmounts))
		fmt.Printf("      - Actual Annual Yield Rate: %.4f%% (avg_actual_yield * blocks_per_year / avg_TD_balance)\n", actualAnnualYieldRatePercent.MustFloat64())
		fmt.Printf("      - Max Allowed Annual Yield Rate: %.2f%%\n", maxAnnualYieldRatePercentFloat)
		fmt.Printf("      - Yield Rate: %.2f%% of max allowed (actual/max)\n", yieldRateOfMaxPercent.MustFloat64())
		fmt.Printf("      - Total Expected Yield: %suvna\n", totalExpectedYield.TruncateInt().String())
		fmt.Printf("      - Total Actual Yield: %suvna\n", totalActual.String())
		fmt.Printf("      - Total Shortfall: %suvna\n", totalShortfall.TruncateInt().String())
		fmt.Printf("      - Note: Actual yield is lower because YIP receives less than allowance per block\n")
	}
	// Format net change (can be negative)
	netChangeDisplayStr := ""
	if netChange.IsNegative() {
		netChangeDisplayStr = fmt.Sprintf("-%s", sdk.NewCoin("uvna", netChange.Neg()).String())
	} else {
		netChangeDisplayStr = sdk.NewCoin("uvna", netChange).String()
	}
	fmt.Printf("      - Net Change in TD Module: %s\n", netChangeDisplayStr)

	// Verify invariants
	fmt.Printf("\n    🔒 Invariant Verification:\n")

	allTDs, err := lib.GetAllKnownTrustDeposits(client, ctx)
	if err == nil {
		finalParams, errParams := lib.GetTrustDepositParams(client, ctx)
		if errParams == nil && finalParams != nil {
			finalShareValue := finalParams.TrustDepositShareValue
			if finalShareValue.IsNil() {
				fmt.Printf("      ⚠️  Cannot verify invariants: share value is nil\n")
			} else {
				var sumShareValue math.LegacyDec = math.LegacyZeroDec()
				var sumAmount uint64 = 0
				for _, td := range allTDs {
					if td.TD == nil {
						continue
					}
					if td.TD.Share.IsNil() {
						continue
					}
					shareValue := td.TD.Share.Mul(finalShareValue)
					sumShareValue = sumShareValue.Add(shareValue)
					sumAmount += td.TD.Amount
				}

				if finalModuleBalance.Amount.GTE(sumShareValue.TruncateInt()) {
					fmt.Printf("      ✅ Invariant 1 HOLDS: module_balance >= sum(share * shareValue)\n")
				} else {
					return fmt.Errorf("❌ Invariant 1 VIOLATED: module_balance < sum(share * shareValue)")
				}

				if finalModuleBalance.Amount.GTE(math.NewInt(int64(sumAmount))) {
					fmt.Printf("      ✅ Invariant 2 HOLDS: module_balance >= sum(amount)\n")
				} else {
					return fmt.Errorf("❌ Invariant 2 VIOLATED: module_balance < sum(amount)")
				}
			}
		}
	}

	fmt.Println("\n" + strings.Repeat("=", 80))
	fmt.Println("✅ Insufficient Funding Simulation completed successfully!")
	fmt.Println(strings.Repeat("=", 80))

	return nil
}
