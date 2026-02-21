package journeys

import (
	"context"
	"fmt"
	"strconv"
	"time"

	"cosmossdk.io/math"
	"github.com/ignite/cli/v28/ignite/pkg/cosmosaccount"
	"github.com/ignite/cli/v28/ignite/pkg/cosmosclient"

	"github.com/verana-labs/verana/testharness/lib"
)

const (
	groupAdminName   = "group_admin"
	groupMember1Name = "group_member1"
	groupMember2Name = "group_member2"
	trOperatorName   = "tr_operator"
)

// Mnemonics for reproducible key generation (same as scripts/setup_group.sh)
var accountMnemonics = map[string]string{
	groupAdminName:   "nature noble gospel breeze flight salt clerk shuffle match secret cheese alarm artwork unit luxury can other vehicle wall wagon view tiger blue strong",
	groupMember1Name: "wagon crater tent spawn year north beach menu item unhappy damage spin flush south tackle van hat rabbit virtual holiday quote antique lock cereal",
	groupMember2Name: "camera nice autumn border illegal drill robot final elevator usage device unhappy blast enough weather ordinary clean document acoustic pistol behind equal what local",
}

// getOrCreateAccount gets an account from keyring, recovers from mnemonic, or creates a new one.
func getOrCreateAccount(client cosmosclient.Client, name string) cosmosaccount.Account {
	account, err := client.Account(name)
	if err == nil {
		return account
	}
	// Account doesn't exist — try to recover from mnemonic if available
	if mnemonic, ok := accountMnemonics[name]; ok {
		account, err = client.AccountRegistry.Import(name, mnemonic, "")
		if err != nil {
			panic(fmt.Sprintf("Failed to import account %s: %v", name, err))
		}
		fmt.Printf("  Recovered account %s from mnemonic\n", name)
		return account
	}
	// No mnemonic — create a new account
	account, _, err = client.AccountRegistry.Create(name)
	if err != nil {
		panic(fmt.Sprintf("Failed to create account %s: %v", name, err))
	}
	fmt.Printf("  Created new account %s\n", name)
	return account
}

// waitForTx waits for a transaction to be fully processed before the next step.
func waitForTx(description string) {
	fmt.Printf("    - Waiting for %s to be processed...\n", description)
	time.Sleep(3 * time.Second)
}

// RunTrustRegistryAuthzSetupJourney implements Journey 101: Setup group and fund accounts
// Creates group with 3 members, threshold=2, 60s voting period. Funds all accounts.
// Does NOT grant any operator authorizations — that's tested in Journey 102.
func RunTrustRegistryAuthzSetupJourney(ctx context.Context, client cosmosclient.Client) error {
	fmt.Println("Starting Journey 101: TR Operator Authorization Setup")

	// =========================================================================
	// Step 1: Create accounts and fund them
	// =========================================================================
	fmt.Println("\n--- Step 1: Fund accounts ---")

	adminAccount := getOrCreateAccount(client, groupAdminName)
	member1Account := getOrCreateAccount(client, groupMember1Name)
	member2Account := getOrCreateAccount(client, groupMember2Name)
	operatorAccount := getOrCreateAccount(client, trOperatorName)

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
	waitForTx("admin funding")
	lib.SendFunds(client, ctx, lib.COOLUSER_ADDRESS, member1Addr, fundAmount)
	waitForTx("member1 funding")
	lib.SendFunds(client, ctx, lib.COOLUSER_ADDRESS, member2Addr, fundAmount)
	waitForTx("member2 funding")
	lib.SendFunds(client, ctx, lib.COOLUSER_ADDRESS, operatorAddr, fundAmount)
	waitForTx("operator funding")
	fmt.Println("✅ Step 1: Funded all accounts with 50 VNA each")

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
	waitForTx("group creation")

	// =========================================================================
	// Step 3: Fund the group policy address (for trust deposits)
	// =========================================================================
	fmt.Println("\n--- Step 3: Fund group policy address ---")

	lib.SendFunds(client, ctx, lib.COOLUSER_ADDRESS, policyAddr, math.NewInt(50000000)) // 50 VNA
	fmt.Printf("✅ Step 3: Funded group policy address %s with 50 VNA\n", policyAddr)
	waitForTx("policy funding")

	// =========================================================================
	// Save results for Journey 102
	// =========================================================================
	result := lib.JourneyResult{
		GroupID:         strconv.FormatUint(groupID, 10),
		GroupPolicyAddr: policyAddr,
		OperatorAddr:    operatorAddr,
		AdminAddr:       adminAddr,
		Member1Addr:     member1Addr,
		Member2Addr:     member2Addr,
	}
	lib.SaveJourneyResult("journey101", result)

	fmt.Println("\n========================================")
	fmt.Println("Journey 101 completed successfully! ✨")
	fmt.Println("Group created, all accounts funded.")
	fmt.Println("========================================")

	return nil
}
