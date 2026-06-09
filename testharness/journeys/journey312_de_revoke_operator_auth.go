package journeys

import (
	"context"
	"fmt"
	"time"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/ignite/cli/v28/ignite/pkg/cosmosclient"

	detypes "github.com/verana-labs/verana/x/de/types"

	"github.com/verana-labs/verana/testharness/lib"
)

// RunDeRevokeOperatorAuthJourney implements Journey 312: DE Revoke Operator Authorization.
// Grants an operator authorization to a fresh grantee, then revokes it via a group
// proposal (MOD-DE-MSG: MsgRevokeOperatorAuthorization, signer = corporation).
// Depends on Journey 301 (corp + group admin/member).
func RunDeRevokeOperatorAuthJourney(ctx context.Context, client cosmosclient.Client) error {
	fmt.Println("Starting Journey 312: DE Revoke Operator Authorization")

	setup301 := lib.LoadJourneyResult("journey301")
	policyAddr := setup301.GroupPolicyAddr
	adminAccount := lib.GetAccount(client, permGroupAdminName)
	member1Account := lib.GetAccount(client, permGroupMember1Name)

	// Fresh grantee dedicated to this journey (does not need to be a keyring account;
	// only the authz record is created/removed for it).
	granteeAddr := sdk.AccAddress([]byte("de_revoke_grantee_01")).String()
	msgType := "/verana.cs.v1.MsgCreateCredentialSchema"
	fmt.Printf("  Corporation: %s\n  Grantee: %s\n", policyAddr, granteeAddr)

	// Step 1: Grant operator authorization to the grantee
	fmt.Println("\n--- Step 1: Grant operator authorization to grantee ---")
	if err := lib.GrantOperatorAuthorizationViaGroup(
		client, ctx, adminAccount, member1Account,
		policyAddr, "", granteeAddr,
		[]string{msgType},
	); err != nil {
		return fmt.Errorf("step 1 failed: %w", err)
	}
	fmt.Println("OK Step 1: operator authorization granted to grantee")
	waitForTx("grant operator authz")

	// Step 2: Revoke that operator authorization via group proposal
	fmt.Println("\n--- Step 2: RevokeOperatorAuthorization via group proposal ---")
	revokeMsg := &detypes.MsgRevokeOperatorAuthorization{
		Corporation: policyAddr,
		Operator:    "",
		Grantee:     granteeAddr,
	}
	proposalID, err := lib.SubmitGroupProposal(
		client, ctx, adminAccount, policyAddr,
		[]sdk.Msg{revokeMsg},
		"Revoke operator auth",
		"Revoke operator authorization via group proposal",
	)
	if err != nil {
		return fmt.Errorf("step 2 failed: submit revoke proposal: %w", err)
	}
	time.Sleep(3 * time.Second)
	if err := lib.VoteOnGroupProposal(client, ctx, adminAccount, proposalID, false); err != nil {
		return fmt.Errorf("step 2 failed: admin vote: %w", err)
	}
	time.Sleep(3 * time.Second)
	if err := lib.VoteOnGroupProposal(client, ctx, member1Account, proposalID, true); err != nil {
		return fmt.Errorf("step 2 failed: member vote: %w", err)
	}
	time.Sleep(3 * time.Second)
	fmt.Println("OK Step 2: operator authorization revoked")
	waitForTx("revoke operator authz")

	fmt.Println("\n========================================")
	fmt.Println("Journey 312 completed successfully!")
	fmt.Println("DE: grant then revoke operator authorization")
	fmt.Println("========================================")
	return nil
}
