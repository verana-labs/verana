package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"strconv"

	"github.com/verana-labs/verana/testharness/journeys"
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
		// CreateCorporation (Corp A) + bootstrap operator authz
		return journeys.RunCorpCreateJourney(ctx, client)
	case 2:
		// UpdateCorporation + CO queries
		return journeys.RunCorpQueriesJourney(ctx, client)
	case 3:
		// Corp CGF AddGFD + IncreaseGFV + GF queries
		return journeys.RunCorpCGFJourney(ctx, client)
	case 20:
		// CreateEcosystem (ec-alpha) + EC queries
		return journeys.RunEcosystemCreateJourney(ctx, client)
	case 21:
		// EC AddGovernanceFrameworkDocument for ec-alpha
		return journeys.RunEcGFAddGFDJourney(ctx, client)
	case 22:
		// EC IncreaseActiveGovernanceFrameworkVersion for ec-alpha
		return journeys.RunEcGFIncreaseGFVJourney(ctx, client)
	case 23:
		// UpdateEcosystem for ec-alpha
		return journeys.RunEcosystemUpdateJourney(ctx, client)
	case 24:
		// ArchiveEcosystem for ec-alpha (archive + unarchive)
		return journeys.RunEcosystemArchiveJourney(ctx, client)
	case 25:
		// EC + GF query coverage
		return journeys.RunEcosystemQueriesJourney(ctx, client)
	case 101:
		// Ecosystem Operator Authorization Setup (Group + Fund)
		return journeys.RunEcosystemAuthzSetupJourney(ctx, client)
	case 102:
		// Ecosystem Operations with Operator Authorization (fail-then-pass)
		return journeys.RunEcosystemAuthzOperationsJourney(ctx, client)
	case 201:
		// Credential Schema Operator Authorization Setup (Group + Fund)
		return journeys.RunCredentialSchemaAuthzSetupJourney(ctx, client)
	case 202:
		// Credential Schema Operations with Operator Authorization (fail-then-pass)
		return journeys.RunCredentialSchemaAuthzOperationsJourney(ctx, client)
	case 301:
		// Permission Operator Authorization Setup (Group + Fund)
		return journeys.RunPermissionAuthzSetupJourney(ctx, client)
	case 302:
		// Permission Operations with Operator Authorization (fail-then-pass)
		return journeys.RunPermissionAuthzOperationsJourney(ctx, client)
	case 303:
		// Permission Cancel VP Last Request with Operator Authorization
		return journeys.RunPermissionCancelVPJourney(ctx, client)
	case 304:
		// Permission Create Root Permission with Operator Authorization
		return journeys.RunPermissionCreateRootJourney(ctx, client)
	case 305:
		// Permission Adjust Permission with Operator Authorization
		return journeys.RunPermissionAdjustJourney(ctx, client)
	case 306:
		// Permission Revoke Permission with Operator Authorization
		return journeys.RunPermissionRevokeJourney(ctx, client)
	case 307:
		// Permission CreateOrUpdatePermissionSession with VS Operator Authorization
		return journeys.RunPermissionCSPSJourney(ctx, client)
	case 308:
		// Permission Slash Trust Deposit with Operator Authorization
		return journeys.RunPermissionSlashTDJourney(ctx, client)
	case 309:
		// Permission Repay Slashed Trust Deposit with Operator Authorization
		return journeys.RunPermissionRepaySlashedTDJourney(ctx, client)
	case 310:
		// Permission CreatePermission (Self Create) with Operator Authorization
		return journeys.RunPermissionCreatePermJourney(ctx, client)
	case 401:
		// Trust Deposit ReclaimYield + RepaySlashed with Operator Authorization
		return journeys.RunTDReclaimYieldJourney(ctx, client)
	case 501:
		// DI Store Digest with Operator Authorization
		return journeys.RunDiStoreDigestJourney(ctx, client)
	case 601:
		// XR Create Exchange Rate via Governance
		return journeys.RunXrCreateExchangeRateJourney(ctx, client)
	case 602:
		// XR Update Exchange Rate with Operator Authorization
		return journeys.RunXrUpdateExchangeRateJourney(ctx, client)
	case 603:
		// XR Get Price Query
		return journeys.RunXrGetPriceJourney(ctx, client)
	default:
		return fmt.Errorf("unknown journey ID: %d", journeyID)
	}
}

func printUsage() {
	fmt.Println("Usage: verana-test-harness JOURNEY_ID")
	fmt.Println("Available journeys:")
	fmt.Println("\n  Corporation (CO) Journeys:")
	fmt.Println("  1  - CreateCorporation (Corp A) + bootstrap operator authz")
	fmt.Println("  2  - UpdateCorporation + CO queries")
	fmt.Println("  3  - Corp CGF AddGFD + IncreaseGFV + GF queries")
	fmt.Println("\n  Ecosystem (EC) Journeys:")
	fmt.Println("  20 - CreateEcosystem (ec-alpha) + EC queries")
	fmt.Println("  21 - EC AddGovernanceFrameworkDocument for ec-alpha")
	fmt.Println("  22 - EC IncreaseActiveGovernanceFrameworkVersion for ec-alpha")
	fmt.Println("  23 - UpdateEcosystem for ec-alpha")
	fmt.Println("  24 - ArchiveEcosystem for ec-alpha (archive + unarchive)")
	fmt.Println("  25 - EC + GF query coverage")
	fmt.Println("\n  Ecosystem Authorization Journeys (legacy):")
	fmt.Println("  101 - EC Operator Authorization Setup (Group + Fund)")
	fmt.Println("  102 - EC Operations with Operator Authorization (fail-then-pass)")
	fmt.Println("\n  Credential Schema Authorization Journeys:")
	fmt.Println("  201 - CS Operator Authorization Setup (Group + Fund)")
	fmt.Println("  202 - CS Operations with Operator Authorization (fail-then-pass)")
	fmt.Println("\n  Permission Authorization Journeys:")
	fmt.Println("  301 - Perm Operator Authorization Setup (Group + Fund)")
	fmt.Println("  302 - Perm Operations with Operator Authorization (fail-then-pass)")
	fmt.Println("  303 - Perm Cancel VP Last Request with Operator Authorization")
	fmt.Println("  304 - Perm Create Root Permission with Operator Authorization")
	fmt.Println("  305 - Perm Adjust Permission with Operator Authorization")
	fmt.Println("  306 - Perm Revoke Permission with Operator Authorization")
	fmt.Println("  307 - Perm CreateOrUpdatePermissionSession with VS Operator Authorization")
	fmt.Println("  308 - Perm Slash Permission Trust Deposit with Operator Authorization")
	fmt.Println("  309 - Perm Repay Slashed Trust Deposit with Operator Authorization")
	fmt.Println("  310 - Perm CreatePermission (Self Create) with Operator Authorization")
	fmt.Println("\n  Trust Deposit Authorization Journeys:")
	fmt.Println("  401 - TD ReclaimYield + RepaySlashed with Operator Authorization")
	fmt.Println("\n  Digest (DI) Journeys:")
	fmt.Println("  501 - DI Store Digest with Operator Authorization")
	fmt.Println("\n  Exchange Rate (XR) Journeys:")
	fmt.Println("  601 - XR Create Exchange Rate via Governance")
	fmt.Println("  602 - XR Update Exchange Rate with Operator Authorization")
	fmt.Println("  603 - XR Get Price Query")
}
