package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"strconv"

	"github.com/verana-labs/verana/testharness/journeys"
	"github.com/verana-labs/verana/testharness/journeys/simulations/td_yield"
	"github.com/verana-labs/verana/testharness/lib"

	"github.com/ignite/cli/v28/ignite/pkg/cosmosclient"
)

func main() {
	if len(os.Args) < 2 {
		printUsage()
		os.Exit(1)
	}

	ctx := context.Background()

	// Initialize client
	config := lib.DefaultConfig()
	client, err := lib.NewClient(ctx, config)
	if err != nil {
		log.Fatalf("Failed to create client: %v", err)
	}

	journeyID, err := strconv.Atoi(os.Args[1])
	if err != nil {
		log.Fatalf("Invalid journey ID: %v", err)
	}

	// Run the specified journey
	err = runJourney(ctx, client, journeyID)
	if err != nil {
		log.Fatalf("Journey %d failed: %v", journeyID, err)
	}
}

func runJourney(ctx context.Context, client cosmosclient.Client, journeyID int) error {
	switch journeyID {
	case 1:
		return journeys.RunTrustRegistryJourney(ctx, client)
	case 2:
		return journeys.RunIssuerGrantorJourney(ctx, client)
	case 3:
		return journeys.RunIssuerValidationJourney(ctx, client)
	case 4:
		return journeys.RunVerifierValidationJourney(ctx, client)
	case 5:
		return journeys.RunCredentialIssuanceJourney(ctx, client)
	case 6:
		return journeys.RunCredentialVerificationJourney(ctx, client)
	case 7:
		return journeys.RunPermissionRenewalJourney(ctx, client)
	case 8:
		return journeys.RunPermissionTerminationJourney(ctx, client)
	case 9:
		return journeys.RunGovernanceFrameworkUpdateJourney(ctx, client)
	case 10:
		return journeys.RunTrustDepositManagementJourney(ctx, client)
	case 11:
		return journeys.RunDIDManagementJourney(ctx, client)
	case 12:
		return journeys.RunPermissionRevocationJourney(ctx, client)
	case 13:
		return journeys.RunPermissionExtensionJourney(ctx, client)
	case 14:
		return journeys.RunCredentialSchemaUpdateJourney(ctx, client)
	case 15:
		return journeys.RunFailedValidationJourney(ctx, client)
	case 16:
		return journeys.RunSlashPermissionTrustDepositJourney(ctx, client)
	case 17:
		return journeys.RunRepayPermissionSlashedTrustDepositJourney(ctx, client)
	case 18:
		return journeys.RunCreatePermissionJourney(ctx, client)
	case 19:
		return journeys.RunTrustDepositYieldJourney(ctx, client)
	case 20:
		// TD Yield Simulation - Proposal Setup
		// Using very small percentage (~1.27e-10%) so that 30 VNA TD is sufficient for insufficient funding scenario
		// This results in YIP per-block ~200 uvna, which is less than allowance for 30M uvna TD (~260 uvna)
		// Calculation: 0.05% / 395,000,000 = 1.265822784810127e-12, rounded to 18 decimal places
		_, err := td_yield.SetupFundingProposal(ctx, client, "0.000000000001265823") // ~1.27e-10% (18 decimals max)
		return err
	case 21:
		// TD Yield Simulation - Sufficient Funding Scenario
		return td_yield.RunSufficientFundingSimulation(ctx, client)
	case 22:
		// TD Yield Simulation - Insufficient Funding Scenario
		return td_yield.RunInsufficientFundingSimulation(ctx, client)
	case 23:
		// Error Scenario Tests for Issues #191, #193, #196
		return journeys.RunErrorScenarioTestsJourney(ctx, client)
	case 101:
		// Trust Registry Operator Authorization Setup (Group + Fund)
		return journeys.RunTrustRegistryAuthzSetupJourney(ctx, client)
	case 102:
		// Trust Registry Operations with Operator Authorization (fail-then-pass)
		return journeys.RunTrustRegistryAuthzOperationsJourney(ctx, client)
	default:
		return fmt.Errorf("unknown journey ID: %d", journeyID)
	}
}

func printUsage() {
	fmt.Println("Usage: verana-test-harness JOURNEY_ID")
	fmt.Println("Available journeys:")
	fmt.Println("  1 - Trust Registry Controller Journey")
	fmt.Println("  2 - Issuer Grantor Validation Journey")
	fmt.Println("  3 - Issuer Validation Journey (via Issuer Grantor)")
	fmt.Println("  4 - Verifier Validation Journey (via Trust Registry)")
	fmt.Println("  5 - Credential Issuance Journey")
	fmt.Println("  6 - Credential Verification Journey")
	fmt.Println("  7 - Permission Renewal Journey")
	fmt.Println("  8 - Permission Termination Journey")
	fmt.Println("  9 - Governance Framework Update Journey")
	fmt.Println("  10 - Trust Deposit Management Journey")
	fmt.Println("  11 - DID Management Journey")
	fmt.Println("  12 - Revoke Permission Journey")
	fmt.Println("  13 - Permission Extension Journey")
	fmt.Println("  14 - Credential Schema Update Journey")
	fmt.Println("  15 - Failed Validation Journey")
	fmt.Println("  16 - Slash Permission Trust Deposit Journey")
	fmt.Println("  17 - Repay Permission Slashed Trust Deposit Journey")
	fmt.Println("  18 - Create Permission Journey")
	fmt.Println("  19 - Trust Deposit Yield Accumulation and Reclaim Journey")
	fmt.Println("\n  TD Yield Simulations:")
	fmt.Println("  20 - Setup Funding Proposal (0.05% of block rewards)")
	fmt.Println("  21 - Sufficient Funding Simulation (allowance < YIP funding)")
	fmt.Println("  22 - Insufficient Funding Simulation (allowance > YIP funding)")
	fmt.Println("\n  Error Scenario Tests:")
	fmt.Println("  23 - Error Scenario Tests (Issues #191, #193, #196)")
	fmt.Println("\n  Trust Registry Authorization Journeys:")
	fmt.Println("  101 - TR Operator Authorization Setup (Group + Fund)")
	fmt.Println("  102 - TR Operations with Operator Authorization (fail-then-pass)")
}
