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
	case 101:
		// Trust Registry Operator Authorization Setup (Group + Fund)
		return journeys.RunTrustRegistryAuthzSetupJourney(ctx, client)
	case 102:
		// Trust Registry Operations with Operator Authorization (fail-then-pass)
		return journeys.RunTrustRegistryAuthzOperationsJourney(ctx, client)
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
	default:
		return fmt.Errorf("unknown journey ID: %d", journeyID)
	}
}

func printUsage() {
	fmt.Println("Usage: verana-test-harness JOURNEY_ID")
	fmt.Println("Available journeys:")
	fmt.Println("\n  Trust Registry Authorization Journeys:")
	fmt.Println("  101 - TR Operator Authorization Setup (Group + Fund)")
	fmt.Println("  102 - TR Operations with Operator Authorization (fail-then-pass)")
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
}
