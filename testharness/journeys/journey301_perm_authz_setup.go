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
	permGroupAdminName   = "perm_group_admin"
	permGroupMember1Name = "perm_group_member1"
	permGroupMember2Name = "perm_group_member2"
	permOperatorName     = "perm_operator"
)

// RunPermissionAuthzSetupJourney implements Journey 301: Setup group and fund accounts for Perm operations.
// Creates a group with 3 members, threshold=2, 60s voting period. Funds all accounts.
// Does NOT grant any operator authorizations — that's tested in Journey 302.
func RunPermissionAuthzSetupJourney(ctx context.Context, client cosmosclient.Client) error {
	fmt.Println("Starting Journey 301: Perm Operator Authorization Setup")

	// =========================================================================
	// Step 1: Create accounts and fund them
	// =========================================================================
	fmt.Println("\n--- Step 1: Fund accounts ---")

	adminAccount := getOrCreateAccount(client, permGroupAdminName)
	member1Account := getOrCreateAccount(client, permGroupMember1Name)
	member2Account := getOrCreateAccount(client, permGroupMember2Name)
	operatorAccount := getOrCreateAccount(client, permOperatorName)

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
	waitForTx("perm_admin funding")
	lib.SendFunds(client, ctx, lib.COOLUSER_ADDRESS, member1Addr, fundAmount)
	waitForTx("perm_member1 funding")
	lib.SendFunds(client, ctx, lib.COOLUSER_ADDRESS, member2Addr, fundAmount)
	waitForTx("perm_member2 funding")
	lib.SendFunds(client, ctx, lib.COOLUSER_ADDRESS, operatorAddr, fundAmount)
	waitForTx("perm_operator funding")
	fmt.Println("✅ Step 1: Funded all Perm accounts with 50 VNA each")

	// =========================================================================
	// Step 2: Create group with 3 members, threshold=2, voting_period=60s
	// =========================================================================
	fmt.Println("\n--- Step 2: Create group with policy ---")

	memberAddresses := []string{adminAddr, member1Addr, member2Addr}
	groupID, policyAddr, err := lib.CreateGroupWithPolicy(
		client, ctx, adminAccount, memberAddresses,
		"2",             // threshold
		300*time.Second, // voting period
	)
	if err != nil {
		return fmt.Errorf("step 2 failed: %w", err)
	}
	fmt.Printf("✅ Step 2: Created group ID: %d, policy address: %s\n", groupID, policyAddr)
	waitForTx("Perm group creation")

	// =========================================================================
	// Step 3: Fund the group policy address
	// =========================================================================
	fmt.Println("\n--- Step 3: Fund group policy address ---")

	lib.SendFunds(client, ctx, lib.COOLUSER_ADDRESS, policyAddr, math.NewInt(50000000)) // 50 VNA
	fmt.Printf("✅ Step 3: Funded Perm group policy address %s with 50 VNA\n", policyAddr)
	waitForTx("Perm policy funding")

	// =========================================================================
	// Save results for Journey 302
	// =========================================================================
	result := lib.JourneyResult{
		GroupID:         strconv.FormatUint(groupID, 10),
		GroupPolicyAddr: policyAddr,
		OperatorAddr:    operatorAddr,
		AdminAddr:       adminAddr,
		Member1Addr:     member1Addr,
		Member2Addr:     member2Addr,
	}
	lib.SaveJourneyResult("journey301", result)

	fmt.Println("\n========================================")
	fmt.Println("Journey 301 completed successfully!")
	fmt.Println("Perm group created, all accounts funded.")
	fmt.Println("========================================")

	return nil
}
