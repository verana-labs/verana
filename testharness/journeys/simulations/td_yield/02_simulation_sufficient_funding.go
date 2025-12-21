package td_yield

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/verana-labs/verana/testharness/lib"

	"cosmossdk.io/math"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/ignite/cli/v28/ignite/pkg/cosmosclient"
)

// SimulationMetrics tracks the metrics for the simulation
type SimulationMetrics struct {
	TotalReclaimed         uint64
	TotalSentToTDModule    math.Int
	InitialTDModuleBalance math.Int
	FinalTDModuleBalance   math.Int
	ReclaimCount           int
	BlocksMonitored        int
}

// RunSufficientFundingSimulation runs simulation where allowance < YIP per-block funding
// This means YIP receives more than needed, excess is returned to protocol pool
func RunSufficientFundingSimulation(ctx context.Context, client cosmosclient.Client) error {
	fmt.Println("\n" + strings.Repeat("=", 80))
	fmt.Println("TD Yield Simulation - Sufficient Funding Scenario")
	fmt.Println(strings.Repeat("=", 80))
	fmt.Println("\nScenario: allowance < YIP per-block funding")
	fmt.Println("Expected: Only allowance is transferred, excess returned to protocol pool")

	metrics := &SimulationMetrics{}

	// ============================================================================
	// STEP 1: Setup test account and grow trust deposit
	// ============================================================================
	fmt.Println("\n📋 Step 1: Setting up test account and growing trust deposit...")

	// Use Issuer_Applicant account
	testAccount := lib.GetAccount(client, lib.ISSUER_APPLICANT_NAME)
	testAccountAddr, err := testAccount.Address(lib.GetAddressPrefix())
	if err != nil {
		return fmt.Errorf("failed to get account address: %v", err)
	}

	// Fund the account
	fmt.Printf("    Funding account %s...\n", testAccountAddr)
	lib.SendFunds(client, ctx, lib.COOLUSER_ADDRESS, testAccountAddr, math.NewInt(50000000)) // 50 VNA
	time.Sleep(2 * time.Second)

	// Create DID
	did := lib.GenerateUniqueDID(client, ctx)
	fmt.Printf("    Creating DID: %s\n", did)
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
		fmt.Printf("    ⚠️  No trust deposit found yet, this is expected if DID registration doesn't create TD\n")
		fmt.Printf("    Continuing with assumption that TD will be created through other operations...\n")
	} else {
		fmt.Printf("    ✅ Initial Trust Deposit: %d uvna\n", initialTD.Amount)
	}

	// ============================================================================
	// STEP 2: Get initial state
	// ============================================================================
	fmt.Println("\n📊 Step 2: Getting initial state...")

	params, err := lib.GetTrustDepositParams(client, ctx)
	if err != nil {
		return fmt.Errorf("failed to get trust deposit parameters: %v", err)
	}

	blocksPerYear, err := lib.GetBlocksPerYear(client, ctx)
	if err != nil {
		return fmt.Errorf("failed to get blocks_per_year: %v", err)
	}

	fmt.Printf("    - Initial Share Value: %s\n", params.TrustDepositShareValue.String())
	fmt.Printf("    - Blocks Per Year: %d\n", blocksPerYear)

	// ============================================================================
	// STEP 3: Monitor blocks and reclaim yield as soon as it's available
	// ============================================================================
	fmt.Println("\n⏳ Step 3: Monitoring blocks and reclaiming yield as it accumulates...")
	fmt.Println("    Will start reclaiming as soon as yield is detected (>= 1 uvna)")

	monitorBlocks := 50
	checkInterval := 5 * time.Second // ~1 block per 5 seconds

	var previousModuleBalance math.Int = initialModuleBalance.Amount
	totalTransferred := math.ZeroInt()
	reclaimAttempts := 0
	maxReclaims := 20

	for i := 0; i < monitorBlocks; i++ {
		time.Sleep(checkInterval)

		currentModuleBalance, err := lib.GetBankBalance(client, ctx, tdModuleAddr)
		if err != nil {
			continue
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

				// Verify invariants at reclaim block
				paramsAtReclaim, errParams := lib.GetTrustDepositParamsAtHeight(client, ctx, blockHeight)
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
			fmt.Printf("    Block %d/%d: TD Module Balance: %s, Total Transferred: %s, Reclaims: %d\n",
				i+1, monitorBlocks, currentModuleBalance.String(), sdk.NewCoin("uvna", totalTransferred).String(), reclaimAttempts)
		}
	}

	metrics.TotalSentToTDModule = totalTransferred
	// Only update BlocksMonitored if we didn't break early (it was already set in the break case)
	if metrics.BlocksMonitored == 0 {
		metrics.BlocksMonitored = monitorBlocks
	}

	fmt.Printf("\n    ✅ Monitoring complete: %d blocks monitored\n", monitorBlocks)
	fmt.Printf("    ✅ Total transferred to TD module: %s\n", sdk.NewCoin("uvna", totalTransferred).String())
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

			// Calculate balance changes: Block 327 - Block 326
			userBalanceChange := userBalanceAtReclaim.Amount.Sub(userBalanceBefore.Amount)
			tdModuleBalanceChange := tdModuleBalanceAtReclaim.Amount.Sub(tdModuleBalanceBefore.Amount)

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
			fmt.Printf("         - Block %d (before reclaim): %s\n", blockBeforeReclaim, userBalanceBefore.String())
			fmt.Printf("         - Block %d (at reclaim): %s\n", blockHeight, userBalanceAtReclaim.String())
			fmt.Printf("         - Balance Change: %s\n", userBalanceChangeStr)
			fmt.Printf("      2. TD Module Balance:\n")
			fmt.Printf("         - Block %d (before reclaim): %s\n", blockBeforeReclaim, tdModuleBalanceBefore.String())
			fmt.Printf("         - Block %d (at reclaim): %s\n", blockHeight, tdModuleBalanceAtReclaim.String())
			fmt.Printf("         - Balance Change: %s\n", tdModuleBalanceChangeStr)
			fmt.Printf("      3. Total Transferred from TD Module (cumulative reclaims): %d uvna\n", metrics.TotalReclaimed)

			// Verify invariants at reclaim block
			paramsAtReclaim, errParams := lib.GetTrustDepositParamsAtHeight(client, ctx, blockHeight)
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

			// Wait before next reclaim
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
	fmt.Println("✅ Sufficient Funding Simulation completed successfully!")
	fmt.Println(strings.Repeat("=", 80))

	return nil
}
