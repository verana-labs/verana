package journeys

import (
	"context"
	"fmt"
	"strconv"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/ignite/cli/v28/ignite/pkg/cosmosclient"

	permtypes "github.com/verana-labs/verana/x/pp/types"

	"github.com/verana-labs/verana/testharness/lib"
)

// RunPermissionTriggerResolverJourney implements Journey 311: PP Trigger Resolver
// (MOD-PP-MSG-15). It reuses the VALIDATED child participant created by Journey
// 302 (whose validator ancestor is the root permission in the same Corporation),
// grants the operator MsgTriggerResolver authorization via the group, then
// broadcasts MsgTriggerResolver and asserts the trigger_resolver event.
// Authorization resolves via Path 2 (ancestor validator + AUTHZ-CHECK-1).
// Depends on Journey 302.
func RunPermissionTriggerResolverJourney(ctx context.Context, client cosmosclient.Client) error {
	fmt.Println("Starting Journey 311: PP Trigger Resolver (MOD-PP-MSG-15)")

	setup := lib.LoadJourneyResult("journey302")
	policyAddr := setup.GroupPolicyAddr
	operatorAddr := setup.OperatorAddr
	operatorAccount := lib.GetAccount(client, permOperatorName)
	adminAccount := lib.GetAccount(client, permGroupAdminName)
	member1Account := lib.GetAccount(client, permGroupMember1Name)

	childID, err := strconv.ParseUint(setup.PermissionID, 10, 64)
	if err != nil {
		return fmt.Errorf("step 0 failed: parse participant id: %w", err)
	}

	fmt.Printf("  Corporation: %s\n", policyAddr)
	fmt.Printf("  Operator:    %s\n", operatorAddr)
	fmt.Printf("  Target participant: %d\n", childID)

	// =========================================================================
	// Step 1: Confirm the target participant has a did and is VALIDATED/active.
	// MOD-PP-MSG-15-2-1 requires an active participant with a non-empty did.
	// =========================================================================
	fmt.Println("\n--- Step 1: Inspect target participant ---")
	target, err := lib.GetParticipant(client, ctx, childID)
	if err != nil {
		return fmt.Errorf("step 1 failed: query participant %d: %w", childID, err)
	}
	fmt.Printf("  role=%s op_state=%s did=%q validator=%d corp=%d\n",
		target.Role.String(), target.OpState.String(), target.Did,
		target.ValidatorParticipantId, target.CorporationId)
	if target.Did == "" {
		return fmt.Errorf("step 1 failed: target participant has no did")
	}
	if target.OpState != permtypes.OnboardingState_VALIDATED {
		// Re-validate so the active precondition holds for the trigger.
		fmt.Println("  participant not VALIDATED; validating it")
		if _, err := lib.SetPermissionVPToValidated(client, ctx, operatorAccount, permtypes.MsgSetParticipantOPToValidated{
			Corporation: policyAddr,
			Id:          childID,
		}); err != nil {
			return fmt.Errorf("step 1 failed: validate participant: %w", err)
		}
		waitForTx("validate target participant")
	}

	// =========================================================================
	// Step 2: Grant the operator MsgTriggerResolver authorization via the group.
	// Grants are in-place replacements, so re-include the prior msg types.
	// =========================================================================
	fmt.Println("\n--- Step 2: Grant operator MsgTriggerResolver authz via group ---")
	msgTypes := []string{
		"/verana.ec.v1.MsgCreateEcosystem",
		"/verana.cs.v1.MsgCreateCredentialSchema",
		"/verana.pp.v1.MsgSetParticipantOPToValidated",
		"/verana.pp.v1.MsgCreateRootParticipant",
		"/verana.pp.v1.MsgSetParticipantEffectiveUntil",
		"/verana.pp.v1.MsgStartParticipantOP",
		"/verana.pp.v1.MsgRenewParticipantOP",
		"/verana.pp.v1.MsgTriggerResolver",
	}
	if err := lib.GrantOperatorAuthorizationViaGroup(
		client, ctx, adminAccount, member1Account,
		policyAddr, operatorAddr, operatorAddr, msgTypes,
	); err != nil {
		return fmt.Errorf("step 2 failed: grant trigger-resolver authz: %w", err)
	}
	fmt.Println("OK Step 2: Granted MsgTriggerResolver authorization")
	waitForTx("grant trigger-resolver authz")

	// =========================================================================
	// Step 3: Broadcast MsgTriggerResolver as the operator (authorized via the
	// ancestor validator path).
	// =========================================================================
	fmt.Println("\n--- Step 3: Broadcast MsgTriggerResolver ---")
	msg := &permtypes.MsgTriggerResolver{
		Corporation: policyAddr,
		Operator:    operatorAddr,
		Id:          childID,
	}
	txResp, err := client.BroadcastTx(ctx, operatorAccount, msg)
	if err != nil {
		return fmt.Errorf("step 3 failed: broadcast: %w", err)
	}
	if txResp.TxResponse.Code != 0 {
		return fmt.Errorf("step 3 failed: tx code %d: %s", txResp.TxResponse.Code, txResp.TxResponse.RawLog)
	}
	fmt.Printf("OK Step 3: MsgTriggerResolver broadcast (txhash %s)\n", txResp.TxResponse.TxHash)
	waitForTx("trigger resolver")

	// =========================================================================
	// Step 4: Assert the trigger_resolver event was emitted with participant_id.
	// =========================================================================
	fmt.Println("\n--- Step 4: Assert trigger_resolver event ---")
	var txResponse sdk.TxResponse
	b, err := client.Context().Codec.MarshalJSON(txResp.TxResponse)
	if err != nil {
		return fmt.Errorf("step 4 failed: marshal tx response: %w", err)
	}
	if err := client.Context().Codec.UnmarshalJSON(b, &txResponse); err != nil {
		return fmt.Errorf("step 4 failed: unmarshal tx response: %w", err)
	}
	found := false
	for _, ev := range txResponse.Events {
		if ev.Type != permtypes.EventTypeTriggerResolver {
			continue
		}
		for _, a := range ev.Attributes {
			if a.Key == "participant_id" && a.Value == setup.PermissionID {
				found = true
			}
		}
	}
	if !found {
		return fmt.Errorf("step 4 failed: %q event with participant_id=%s not found",
			permtypes.EventTypeTriggerResolver, setup.PermissionID)
	}
	fmt.Printf("OK Step 4: %q event emitted with participant_id=%s\n",
		permtypes.EventTypeTriggerResolver, setup.PermissionID)

	fmt.Println("\n========================================")
	fmt.Println("Journey 311 completed successfully!")
	fmt.Println("PP Trigger Resolver (MOD-PP-MSG-15) validated via ancestor-validator authorization.")
	fmt.Println("========================================")
	return nil
}
