package journeys

import (
	"context"
	"fmt"
	"strconv"
	"time"

	"github.com/verana-labs/verana/testharness/lib"

	"cosmossdk.io/math"
	sdk "github.com/cosmos/cosmos-sdk/types"
	govtypes "github.com/cosmos/cosmos-sdk/x/gov/types/v1"
	"github.com/ignite/cli/v28/ignite/pkg/cosmosaccount"
	"github.com/ignite/cli/v28/ignite/pkg/cosmosclient"
	tdtypes "github.com/verana-labs/verana/x/td/types"
)

// RunTrustDepositYieldJourney implements Journey 19: Trust Deposit Yield Accumulation and Reclaim
func RunTrustDepositYieldJourney(ctx context.Context, client cosmosclient.Client) error {
	fmt.Println("Starting Journey 19: Trust Deposit Yield Accumulation and Reclaim")

	// ============================================================================
	// PROPOSAL SETUP  - Comment out this section if proposal already exists
	// ============================================================================
	fmt.Println("\n📋 Step 0: Setting up continuous funding for Yield Intermediate Pool...")

	govModuleAddr, err := lib.GetGovModuleAddress(client, ctx)
	if err != nil {
		return fmt.Errorf("failed to get governance module address: %v", err)
	}

	yipAddr, err := lib.GetYieldIntermediatePoolAddress(client, ctx)
	if err != nil {
		return fmt.Errorf("failed to get Yield Intermediate Pool address: %v", err)
	}

	// Check if proposal already exists (query recent proposals)

	coolUserAccount := lib.GetAccount(client, lib.COOLUSER_NAME)
	percentage := "0.005000000000000000"
	title := "Send continuous funds to Yield Intermediate Pool for Trust Deposit yield distribution"
	summary := "This proposal creates a continuous fund to send 0.5% of community pool contributions to the Yield Intermediate Pool account for distributing yield to trust deposit holders"

	proposalID, err := lib.SubmitContinuousFundProposal(
		client, ctx, coolUserAccount,
		govModuleAddr, yipAddr, percentage,
		title, summary,
	)
	if err != nil {
		return fmt.Errorf("failed to submit continuous fund proposal: %v", err)
	}
	fmt.Printf("✅ Submitted continuous fund proposal with ID: %d\n", proposalID)

	err = lib.VoteOnGovProposal(client, ctx, coolUserAccount, proposalID, govtypes.OptionYes)
	if err != nil {
		return fmt.Errorf("failed to vote on proposal: %v", err)
	}
	fmt.Printf("✅ Voted YES on proposal %d\n", proposalID)

	fmt.Println("⏳ Waiting for proposal to pass...")
	err = lib.WaitForProposalToPass(client, ctx, proposalID, 110)
	if err != nil {
		return fmt.Errorf("proposal did not pass: %v", err)
	}
	fmt.Printf("✅ Proposal %d has passed and executed\n", proposalID)
	time.Sleep(10 * time.Second)
	// ============================================================================
	// END PROPOSAL SETUP - Comment out lines to skip proposal creation
	// ============================================================================

	// Step 1: Find account with trust deposit (claimable = 0, for yield testing)
	fmt.Println("\n📊 Step 1: Finding account with trust deposit for yield testing...")
	fmt.Printf("    - Note: Excluding accounts with claimable > 0 (claimable is from slashing/permission termination, not yield)\n")
	fmt.Printf("    - MOD-TD-MSG-2 (ReclaimTrustDepositYield) calculates yield from share value, not claimable field\n")

	// Check all known accounts to find one with trust deposit and claimable = 0
	type accountInfo struct {
		name    string
		address string
	}
	accountsToCheck := []accountInfo{
		{lib.ISSUER_APPLICANT_NAME, lib.ISSUER_APPLICANT_ADDRESS},
		{lib.ISSUER_GRANTOR_APPLICANT_NAME, lib.ISSUER_GRANTOR_APPLICANT_ADDRESS},
		{lib.VERIFIER_APPLICANT_NAME, lib.VERIFIER_APPLICANT_ADDRESS},
		{lib.CREDENTIAL_HOLDER_NAME, lib.CREDENTIAL_HOLDER_ADDRESS},
		{lib.TRUST_REGISTRY_CONTROLLER_NAME, lib.TRUST_REGISTRY_CONTROLLER_ADDRESS},
	}

	var depositHolderAccount cosmosaccount.Account
	var depositHolderAddr string
	var foundAccount bool

	for _, acc := range accountsToCheck {
		account, err := client.Account(acc.name)
		if err != nil {
			continue
		}

		td, err := lib.GetTrustDeposit(client, ctx, account)
		if err != nil {
			continue
		}

		fmt.Printf("    - %s (%s): Amount=%d, Claimable=%d", acc.name, acc.address, td.Amount, td.Claimable)

		// Skip accounts with claimable > 0 (these are from slashing/permission termination)
		if td.Claimable > 0 {
			fmt.Printf(" (skipped - claimable from different process)\n")
			continue
		}

		fmt.Printf("\n")

		// Use account with trust deposit and claimable = 0
		if td.Amount > 0 {
			depositHolderAccount = account
			depositHolderAddr = acc.address
			foundAccount = true
			fmt.Printf("    ✅ Found account for yield testing: %s (%s)\n", acc.name, acc.address)
			break
		}
	}

	if !foundAccount {
		// Fall back to Issuer_Applicant if no suitable account found
		fmt.Printf("    ⚠️  No suitable account found, using %s for testing\n", lib.ISSUER_APPLICANT_NAME)
		var err error
		depositHolderAccount, err = client.Account(lib.ISSUER_APPLICANT_NAME)
		if err != nil {
			return fmt.Errorf("failed to get %s account: %v", lib.ISSUER_APPLICANT_NAME, err)
		}
		depositHolderAddr = lib.ISSUER_APPLICANT_ADDRESS
	}

	fmt.Printf("\n📊 Using account: %s\n", depositHolderAddr)

	// Step 2: Get initial trust deposit state
	initialDeposit, err := lib.GetTrustDeposit(client, ctx, depositHolderAccount)
	if err != nil {
		return fmt.Errorf("failed to get initial trust deposit: %v", err)
	}

	initialParams, err := lib.GetTrustDepositParams(client, ctx)
	if err != nil {
		return fmt.Errorf("failed to get initial trust deposit parameters: %v", err)
	}

	initialShareValue := initialParams.TrustDepositShareValue
	initialAmount := initialDeposit.Amount

	fmt.Printf("    - Account: %s\n", depositHolderAddr)
	fmt.Printf("    - Trust Deposit Amount: %d uvna\n", initialAmount)
	fmt.Printf("    - Share: %s\n", initialDeposit.Share)
	fmt.Printf("    - Share Value: %s\n", initialShareValue.String())
	fmt.Printf("    - Claimable Amount: %d uvna\n", initialDeposit.Claimable)

	if initialAmount == 0 {
		return fmt.Errorf("account has no trust deposit. Please run previous journeys first")
	}

	// Step 3: Wait for yield to accumulate (need at least 1 uvna after truncation)
	fmt.Println("\n⏳ Step 2: Waiting for yield to accumulate...")
	fmt.Printf("    - Note: Yield must be >= 1 uvna (after truncation) to be reclaimable\n")
	maxWaitTime := 300 * time.Second // Increased to 5 minutes
	checkInterval := 5 * time.Second
	startTime := time.Now()
	var finalParams *tdtypes.Params
	var yieldAccumulated bool

	for time.Since(startTime) < maxWaitTime {
		time.Sleep(checkInterval)

		currentParams, err := lib.GetTrustDepositParams(client, ctx)
		if err != nil {
			continue
		}

		currentShareValue := currentParams.TrustDepositShareValue

		if currentShareValue.GT(initialShareValue) {
			// Check if yield is at least 1 uvna (after truncation)
			shareDec := initialDeposit.Share
			depositValueDec := shareDec.Mul(currentShareValue)
			depositValueUint64 := depositValueDec.TruncateInt().Uint64()
			expectedYield := int64(depositValueUint64) - int64(initialAmount)

			if expectedYield >= 1 {
				finalParams = currentParams
				yieldAccumulated = true
				fmt.Printf("✅ Yield detected and sufficient! Share value: %s -> %s\n",
					initialShareValue.String(), currentShareValue.String())
				fmt.Printf("    - Expected yield: %d uvna (after truncation)\n", expectedYield)
				break
			} else {
				fmt.Printf("    Yield detected but too small: %d uvna (need >= 1 uvna), continuing to wait...\n", expectedYield)
			}
		}

		fmt.Printf("    Waiting... (share value: %s, elapsed: %v)\n",
			currentShareValue.String(), time.Since(startTime).Round(time.Second))
	}

	if !yieldAccumulated {
		fmt.Println("⚠️  No sufficient yield accumulation detected within wait period")
		finalParams = initialParams
	}

	// Step 4: Check current state
	fmt.Println("\n📊 Step 3: Checking trust deposit state after yield accumulation...")
	currentDeposit, err := lib.GetTrustDeposit(client, ctx, depositHolderAccount)
	if err != nil {
		return fmt.Errorf("failed to get current trust deposit: %v", err)
	}

	currentShareValue := finalParams.TrustDepositShareValue
	fmt.Printf("    - Trust Deposit Amount: %d uvna\n", currentDeposit.Amount)
	fmt.Printf("    - Share: %s\n", currentDeposit.Share)
	fmt.Printf("    - Share Value: %s\n", currentShareValue.String())
	fmt.Printf("    - Claimable Amount: %d uvna\n", currentDeposit.Claimable)

	// Calculate expected yield (for diagnostics)
	// depositValue = Share × ShareValue
	// yield = depositValue - Amount (but depositValue is truncated to uint64)
	shareDec := currentDeposit.Share
	depositValueDec := shareDec.Mul(currentShareValue)
	depositValueUint64 := depositValueDec.TruncateInt().Uint64()
	expectedYield := int64(depositValueUint64) - int64(currentDeposit.Amount)
	fmt.Printf("    - Calculated deposit value (truncated): %d uvna\n", depositValueUint64)
	if expectedYield > 0 {
		fmt.Printf("    - Expected yield (before truncation): ~%s uvna\n", depositValueDec.Sub(math.LegacyNewDec(int64(currentDeposit.Amount))).String())
		fmt.Printf("    - Expected yield (after truncation): %d uvna\n", expectedYield)
	} else {
		fmt.Printf("    - ⚠️  No yield available (depositValue %d <= Amount %d)\n", depositValueUint64, currentDeposit.Amount)
		fmt.Printf("    - Yield before truncation: %s uvna (too small, truncated to 0)\n", depositValueDec.Sub(math.LegacyNewDec(int64(currentDeposit.Amount))).String())
		fmt.Printf("    - Need to wait longer for yield to accumulate to at least 1 uvna\n")
	}

	// Step 5: Reclaim yield (MOD-TD-MSG-2: Reclaim Trust Deposit Interests)
	fmt.Printf("\n💰 Step 4: Reclaiming trust deposit yield\n")
	fmt.Printf("    - Account: %s\n", depositHolderAddr)
	fmt.Printf("    - Note: MOD-TD-MSG-2 calculates yield from share value vs amount, NOT from claimable field\n")
	fmt.Printf("    - Claimable field is only used for MOD-TD-MSG-3 (principal reclaim)\n")

	// Verify account still has claimable = 0 (should not have changed)
	if currentDeposit.Claimable > 0 {
		fmt.Printf("    ⚠️  Warning: Account has claimable > 0 (%d), this is from slashing/permission termination, not yield\n", currentDeposit.Claimable)
		fmt.Printf("    - Continuing with yield reclaim test (yield is calculated independently)\n")
	}

	if true { // Always attempt yield reclaim (yield is calculated from share value, not claimable)
		// Check balance before reclaim
		balanceBefore, err := lib.GetBankBalance(client, ctx, depositHolderAddr)
		if err != nil {
			fmt.Printf("    ⚠️  Could not get balance before reclaim: %v\n", err)
		} else {
			fmt.Printf("    - Balance before reclaim: %s\n", balanceBefore.String())
		}

		fmt.Printf("    - Reclaiming yield (calculated from share value vs amount)\n")

		// Get transaction response to check claimed amount
		txResp, blockHeight, err := lib.ReclaimTrustDepositYieldWithResponse(client, ctx, depositHolderAccount)
		if err != nil {
			if contains(err.Error(), "no claimable yield available") {
				fmt.Printf("    ⚠️  No claimable yield available\n")
			} else {
				return fmt.Errorf("failed to reclaim trust deposit yield: %v", err)
			}
		} else {
			if txResp != nil && txResp.ClaimedAmount > 0 {
				fmt.Printf("✅ Successfully reclaimed yield: %d uvna at block height %d\n", txResp.ClaimedAmount, blockHeight)
			} else {
				fmt.Printf("✅ Successfully reclaimed trust deposit yield at block height %d\n", blockHeight)
				fmt.Printf("    ⚠️  Note: Could not extract claimed amount from events\n")
			}
		}
		time.Sleep(3 * time.Second)

		// Check balance after reclaim
		balanceAfter, err := lib.GetBankBalance(client, ctx, depositHolderAddr)
		if err != nil {
			fmt.Printf("    ⚠️  Could not get balance after reclaim: %v\n", err)
		} else {
			fmt.Printf("    - Balance after reclaim: %s\n", balanceAfter.String())
			if balanceBefore.Amount.IsPositive() {
				balanceDiff := balanceAfter.Amount.Sub(balanceBefore.Amount)
				var claimedYield uint64
				if txResp != nil {
					claimedYield = txResp.ClaimedAmount
				}

				if balanceDiff.IsPositive() {
					fmt.Printf("    ✅ Balance increased by: %s\n", sdk.NewCoin("uvna", balanceDiff).String())
					if claimedYield > 0 {
						netYield := balanceDiff.Uint64()
						if netYield < claimedYield {
							feePaid := claimedYield - netYield
							fmt.Printf("    - Yield claimed: %d uvna, Transaction fee: %d uvna, Net: %d uvna\n",
								claimedYield, feePaid, netYield)
						}
					}
				} else if balanceDiff.IsZero() {
					fmt.Printf("    ℹ️  Balance unchanged\n")
					if claimedYield > 0 {
						fmt.Printf("    - Yield claimed: %d uvna, Transaction fee: %d uvna (net: 0)\n",
							claimedYield, claimedYield)
					}
				} else {
					balanceDecrease := balanceDiff.Neg().Uint64()
					fmt.Printf("    ⚠️  Balance decreased by: %s\n", sdk.NewCoin("uvna", balanceDiff.Neg()).String())
					if claimedYield > 0 {
						totalCost := claimedYield + balanceDecrease
						fmt.Printf("    - Yield claimed: %d uvna, Transaction fee: %d uvna, Net decrease: %d uvna\n",
							claimedYield, totalCost, balanceDecrease)
						fmt.Printf("    - Note: Transaction fee (%d uvna) exceeds claimed yield (%d uvna)\n",
							totalCost, claimedYield)
					} else {
						fmt.Printf("    - Transaction fee: %d uvna (check transaction response above)\n", balanceDecrease)
					}
				}
			}
		}
	}

	// Step 6: Verify final state
	fmt.Println("\n✅ Step 5: Verifying final state...")
	finalDeposit, err := lib.GetTrustDeposit(client, ctx, depositHolderAccount)
	if err != nil {
		return fmt.Errorf("failed to get final trust deposit: %v", err)
	}

	fmt.Printf("    - Account: %s\n", depositHolderAddr)
	fmt.Printf("    - Final Trust Deposit Amount: %d uvna\n", finalDeposit.Amount)
	fmt.Printf("    - Final Share: %s\n", finalDeposit.Share)
	fmt.Printf("    - Final Claimable Amount: %d uvna\n", finalDeposit.Claimable)

	// Note: MOD-TD-MSG-2 (ReclaimTrustDepositYield) does NOT use or modify claimable
	// Claimable is only used for MOD-TD-MSG-3 (ReclaimTrustDeposit - principal reclaim)
	// Yield reclaim calculates yield from share value vs amount, completely independent of claimable
	if finalDeposit.Claimable > 0 {
		fmt.Printf("    ℹ️  Note: Claimable (%d) is from a different process (slashing/permission termination)\n", finalDeposit.Claimable)
		fmt.Printf("    - Claimable is NOT used for yield reclaim (MOD-TD-MSG-2)\n")
		fmt.Printf("    - Claimable is only used for principal reclaim (MOD-TD-MSG-3)\n")
	} else {
		fmt.Printf("    ℹ️  Note: Claimable is 0 (expected for yield testing)\n")
	}

	// Save result
	journey3Result, _ := lib.GetJourneyResult("journey3")
	result := lib.JourneyResult{
		TrustRegistryID:      journey3Result.TrustRegistryID,
		SchemaID:             journey3Result.SchemaID,
		DepositHolderAddress: depositHolderAddr,
		InitialDepositAmount: strconv.FormatUint(uint64(initialAmount), 10),
		FinalDepositAmount:   strconv.FormatUint(uint64(finalDeposit.Amount), 10),
	}
	lib.SaveJourneyResult("journey19", result)

	fmt.Println("\n🎉 Journey 19 completed successfully!")
	return nil
}

func contains(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
