package journeys

import (
	"context"
	"fmt"
	"strconv"
	"time"

	"cosmossdk.io/math"
	"github.com/ignite/cli/v28/ignite/pkg/cosmosclient"

	"github.com/verana-labs/verana/testharness/lib"
)

const (
	csGroupAdminName   = "cs_group_admin"
	csGroupMember1Name = "cs_group_member1"
	csGroupMember2Name = "cs_group_member2"
	csOperatorName     = "cs_operator"
)

// RunCredentialSchemaAuthzSetupJourney implements Journey 201: Setup group and fund accounts for CS operations.
// Creates a group with 3 members, threshold=2, 60s voting period. Funds all accounts.
// Does NOT grant any operator authorizations — that's tested in Journey 202.
func RunCredentialSchemaAuthzSetupJourney(ctx context.Context, client cosmosclient.Client) error {
	fmt.Println("Starting Journey 201: CS Operator Authorization Setup")

	// =========================================================================
	// Step 1: Create accounts and fund them
	// =========================================================================
	fmt.Println("\n--- Step 1: Fund accounts ---")

	adminAccount := getOrCreateAccount(client, csGroupAdminName)
	member1Account := getOrCreateAccount(client, csGroupMember1Name)
	member2Account := getOrCreateAccount(client, csGroupMember2Name)
	operatorAccount := getOrCreateAccount(client, csOperatorName)

	adminAddr, _ := adminAccount.Address(lib.GetAddressPrefix())
	member1Addr, _ := member1Account.Address(lib.GetAddressPrefix())
	member2Addr, _ := member2Account.Address(lib.GetAddressPrefix())
	operatorAddr, _ := operatorAccount.Address(lib.GetAddressPrefix())

	fmt.Printf("  Admin:    %s\n", adminAddr)
	fmt.Printf("  Member1:  %s\n", member1Addr)
	fmt.Printf("  Member2:  %s\n", member2Addr)
	fmt.Printf("  Operator: %s\n", operatorAddr)

	// Fund all accounts from cooluser (sequential sends from same account need waits)
	fundAmount := math.NewInt(50000000) // 50 VNA each
	lib.SendFunds(client, ctx, lib.COOLUSER_ADDRESS, adminAddr, fundAmount)
	waitForTx("cs_admin funding")
	lib.SendFunds(client, ctx, lib.COOLUSER_ADDRESS, member1Addr, fundAmount)
	waitForTx("cs_member1 funding")
	lib.SendFunds(client, ctx, lib.COOLUSER_ADDRESS, member2Addr, fundAmount)
	waitForTx("cs_member2 funding")
	lib.SendFunds(client, ctx, lib.COOLUSER_ADDRESS, operatorAddr, fundAmount)
	waitForTx("cs_operator funding")
	fmt.Println("✅ Step 1: Funded all CS accounts with 50 VNA each")

	// =========================================================================
	// Step 2: Create group with 3 members, threshold=2, voting_period=60s
	// =========================================================================
	fmt.Println("\n--- Step 2: Create group with policy ---")

	memberAddresses := []string{adminAddr, member1Addr, member2Addr}
	groupID, policyAddr, err := lib.CreateGroupWithPolicy(
		client, ctx, adminAccount, memberAddresses,
		"2",            // threshold
		60*time.Second, // voting period
	)
	if err != nil {
		return fmt.Errorf("step 2 failed: %w", err)
	}
	fmt.Printf("✅ Step 2: Created group ID: %d, policy address: %s\n", groupID, policyAddr)
	waitForTx("CS group creation")

	// =========================================================================
	// Step 3: Fund the group policy address
	// =========================================================================
	fmt.Println("\n--- Step 3: Fund group policy address ---")

	lib.SendFunds(client, ctx, lib.COOLUSER_ADDRESS, policyAddr, math.NewInt(50000000)) // 50 VNA
	fmt.Printf("✅ Step 3: Funded CS group policy address %s with 50 VNA\n", policyAddr)
	waitForTx("CS policy funding")

	// =========================================================================
	// Save results for Journey 202
	// =========================================================================
	result := lib.JourneyResult{
		GroupID:         strconv.FormatUint(groupID, 10),
		GroupPolicyAddr: policyAddr,
		OperatorAddr:    operatorAddr,
		AdminAddr:       adminAddr,
		Member1Addr:     member1Addr,
		Member2Addr:     member2Addr,
	}
	lib.SaveJourneyResult("journey201", result)

	fmt.Println("\n========================================")
	fmt.Println("Journey 201 completed successfully! ✨")
	fmt.Println("CS group created, all accounts funded.")
	fmt.Println("========================================")

	return nil
}
