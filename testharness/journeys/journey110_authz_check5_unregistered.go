package journeys

import (
	"context"
	"fmt"
	"strings"

	"cosmossdk.io/math"
	"github.com/ignite/cli/v28/ignite/pkg/cosmosclient"

	detypes "github.com/verana-labs/verana/x/de/types"

	"github.com/verana-labs/verana/testharness/lib"
)

const authzCheck5UnregName = "authz5_unregistered"

// RunAuthzCheck5UnregisteredJourney implements Journey 110: AUTHZ-CHECK-5 negative path.
//
// Per spec v4-rc2 AUTHZ-CHECK-5, a delegable Msg whose signing `corporation`
// account is NOT the policy_address of a registered Corporation MUST abort with
// ErrCorporationNotRegistered (see MOD-CO-MSG-1).
//
// MsgGrantOperatorAuthorization with operator="" is the cleanest live probe:
// AUTHZ-CHECK-1 short-circuits for the empty operator (corporation acting alone),
// so AUTHZ-CHECK-5 is the primary gate. The message's signer IS `corporation`,
// so a brand-new account that was never registered via MOD-CO-MSG-1 can sign it
// directly (no group proposal needed) and MUST be rejected.
func RunAuthzCheck5UnregisteredJourney(ctx context.Context, client cosmosclient.Client) error {
	fmt.Println("Starting Journey 110: AUTHZ-CHECK-5 negative (unregistered corporation)")

	// =========================================================================
	// Step 1: a fresh account that has never been registered as a Corporation.
	// =========================================================================
	fmt.Println("\n--- Step 1: Fund a fresh, unregistered account ---")
	unregistered := getOrCreateAccount(client, authzCheck5UnregName)
	addr, err := unregistered.Address(lib.GetAddressPrefix())
	if err != nil {
		return fmt.Errorf("failed to derive unregistered account address: %w", err)
	}
	lib.SendFunds(client, ctx, lib.COOLUSER_ADDRESS, addr, math.NewInt(10_000_000))
	fmt.Printf("  Unregistered account: %s\n", addr)
	waitForTx("fund unregistered account")

	// =========================================================================
	// Step 2: self-grant from the unregistered account → expect AUTHZ-CHECK-5 abort.
	// =========================================================================
	fmt.Println("\n--- Step 2: Self-grant from unregistered corporation (expect ErrCorporationNotRegistered) ---")
	msg := &detypes.MsgGrantOperatorAuthorization{
		Corporation: addr,
		Operator:    "", // self-grant: AUTHZ-CHECK-1 short-circuits, AUTHZ-CHECK-5 is the gate
		Grantee:     addr,
		MsgTypes:    []string{"/verana.di.v1.MsgStoreDigest"},
	}

	txResp, err := client.BroadcastTx(ctx, unregistered, msg)
	if err == nil && txResp.TxResponse.Code != 0 {
		err = fmt.Errorf("code %d: %s", txResp.TxResponse.Code, txResp.TxResponse.RawLog)
	}
	if err == nil {
		return fmt.Errorf("step 2 failed: expected ErrCorporationNotRegistered but the grant succeeded")
	}
	if !strings.Contains(err.Error(), "policy_address") || !strings.Contains(err.Error(), "Corporation") {
		return fmt.Errorf("step 2 failed: expected corporation-not-registered error, got: %w", err)
	}
	fmt.Printf("✅ Step 2: AUTHZ-CHECK-5 correctly rejected the unregistered corporation:\n    %v\n", err)

	fmt.Println("\n========================================")
	fmt.Println("Journey 110 completed successfully!")
	fmt.Println("AUTHZ-CHECK-5 negative path verified:")
	fmt.Println("  - delegable Msg signed by an unregistered policy_address → ErrCorporationNotRegistered")
	fmt.Println("========================================")
	return nil
}
