package journeys

import (
	"context"
	"fmt"
	"strconv"
	"time"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/ignite/cli/v28/ignite/pkg/cosmosclient"

	cschema "github.com/verana-labs/verana/x/cs/types"
	permtypes "github.com/verana-labs/verana/x/pp/types"

	"github.com/verana-labs/verana/testharness/lib"
)

// RunPermissionTriggerResolverJourney implements Journey 311: PP Trigger Resolver
// (MOD-PP-MSG-15). It is SELF-CONTAINED: it depends only on Journey 301 (the
// Corporation + funded accounts) and builds its own ecosystem -> schema -> root
// permission -> child participant, so it is robust regardless of what earlier
// journeys did to their participants. Authorization resolves via Path 2 (the
// child's active ancestor validator + AUTHZ-CHECK-1).
func RunPermissionTriggerResolverJourney(ctx context.Context, client cosmosclient.Client) error {
	fmt.Println("Starting Journey 311: PP Trigger Resolver (MOD-PP-MSG-15)")

	setup := lib.LoadJourneyResult("journey301")
	policyAddr := setup.GroupPolicyAddr
	operatorAddr := setup.OperatorAddr
	operatorAccount := lib.GetAccount(client, permOperatorName)
	adminAccount := lib.GetAccount(client, permGroupAdminName)
	member1Account := lib.GetAccount(client, permGroupMember1Name)

	fmt.Printf("  Corporation: %s\n", policyAddr)
	fmt.Printf("  Operator:    %s\n", operatorAddr)

	// =========================================================================
	// Step 1: Grant the operator all authz it needs (incl. MsgTriggerResolver)
	// up front. Grants are in-place replacements.
	// =========================================================================
	fmt.Println("\n--- Step 1: Grant operator authz (incl. MsgTriggerResolver) ---")
	msgTypes := []string{
		"/verana.ec.v1.MsgCreateEcosystem",
		"/verana.cs.v1.MsgCreateCredentialSchema",
		"/verana.pp.v1.MsgCreateRootParticipant",
		"/verana.pp.v1.MsgStartParticipantOP",
		"/verana.pp.v1.MsgSetParticipantOPToValidated",
		"/verana.pp.v1.MsgTriggerResolver",
	}
	if err := lib.GrantOperatorAuthorizationViaGroup(
		client, ctx, adminAccount, member1Account,
		policyAddr, operatorAddr, operatorAddr, msgTypes,
	); err != nil {
		return fmt.Errorf("step 1 failed: grant authz: %w", err)
	}
	waitForTx("grant authz")

	// =========================================================================
	// Step 2: Create Ecosystem (Trust Registry).
	// =========================================================================
	fmt.Println("\n--- Step 2: Create Ecosystem ---")
	trIDStr, err := lib.CreateEcosystemWithAuthority(
		client, ctx, operatorAccount, policyAddr,
		lib.GenerateUniqueDID(client, ctx),
		"https://trigger-resolver-test.com/governance-framework.pdf",
		"sha384-MzNNbQTWCSUSi0bbz7dbua+RcENv7C6FvlmYJ1Y+I727HsPOHdzwELMYO9Mz68M26",
		"en",
	)
	if err != nil {
		return fmt.Errorf("step 2 failed: create ecosystem: %w", err)
	}
	trID, _ := strconv.ParseUint(trIDStr, 10, 64)
	waitForTx("ecosystem")

	// =========================================================================
	// Step 3: Create Credential Schema (GRANTOR validation modes).
	// =========================================================================
	fmt.Println("\n--- Step 3: Create Credential Schema ---")
	csIDStr, err := lib.CreateCredentialSchemaWithAuthority(
		client, ctx, operatorAccount, policyAddr,
		trID, lib.GenerateSimpleSchema(trIDStr),
		cschema.IssuerOnboardingMode_ISSUER_ONBOARDING_MODE_GRANTOR_VALIDATION_PROCESS,
		cschema.VerifierOnboardingMode_VERIFIER_ONBOARDING_MODE_GRANTOR_VALIDATION_PROCESS,
	)
	if err != nil {
		return fmt.Errorf("step 3 failed: create schema: %w", err)
	}
	csID, _ := strconv.ParseUint(csIDStr, 10, 64)
	waitForTx("schema")

	// =========================================================================
	// Step 4: Create the Root Permission (ECOSYSTEM). effective_from MUST be in
	// the future, so it becomes active only after that time passes (below).
	// =========================================================================
	fmt.Println("\n--- Step 4: Create Root Permission ---")
	rootEffectiveFrom := time.Now().Add(2 * time.Second)
	rootEffectiveUntil := rootEffectiveFrom.Add(360 * 24 * time.Hour)
	rootID, err := lib.CreateRootPermissionWithAuthority(
		client, ctx, operatorAccount, policyAddr,
		csID, lib.GenerateUniqueDID(client, ctx),
		&rootEffectiveFrom, &rootEffectiveUntil, 0, 0, 0,
	)
	if err != nil {
		return fmt.Errorf("step 4 failed: create root permission: %w", err)
	}
	fmt.Printf("  Root permission id: %d\n", rootID)
	waitForTx("root permission")

	// The root validator MUST be ACTIVE before it can validate a child or be
	// walked as an ancestor. Its effective_from is in the future and the chain's
	// block time can lag wall-clock, so poll the chain's block time until it
	// passes effective_from.
	fmt.Println("  Waiting for root to become active (block time > effective_from)...")
	rootActive := false
	for i := 0; i < 40; i++ {
		st, serr := client.Context().Client.Status(ctx)
		if serr == nil && st.SyncInfo.LatestBlockTime.After(rootEffectiveFrom) {
			rootActive = true
			fmt.Printf("  Root active at block time %s\n", st.SyncInfo.LatestBlockTime)
			break
		}
		time.Sleep(2 * time.Second)
	}
	if !rootActive {
		return fmt.Errorf("step 4 failed: root did not become active in time")
	}

	// =========================================================================
	// Step 5: Start a child participant (ISSUER_GRANTOR) under the now-active root.
	// =========================================================================
	fmt.Println("\n--- Step 5: Start child participant VP ---")
	childIDStr, err := lib.StartPermissionVPWithAuthority(
		client, ctx, operatorAccount, policyAddr,
		permtypes.ParticipantRole_ISSUER_GRANTOR, rootID,
		lib.GenerateUniqueDID(client, ctx),
	)
	if err != nil {
		return fmt.Errorf("step 5 failed: start child VP: %w", err)
	}
	childID, _ := strconv.ParseUint(childIDStr, 10, 64)
	fmt.Printf("  Child participant id: %d\n", childID)
	waitForTx("start child VP")

	// =========================================================================
	// Step 6: Validate the child so it is an active participant.
	// =========================================================================
	fmt.Println("\n--- Step 6: Validate child participant ---")
	if _, err := lib.SetPermissionVPToValidated(client, ctx, operatorAccount, permtypes.MsgSetParticipantOPToValidated{
		Corporation: policyAddr,
		Id:          childID,
	}); err != nil {
		return fmt.Errorf("step 6 failed: validate child: %w", err)
	}
	waitForTx("validate child")

	// =========================================================================
	// Step 7: Broadcast MsgTriggerResolver as the operator (Path 2: the child's
	// active root ancestor authorizes via the operator's AUTHZ-CHECK-1 grant).
	// =========================================================================
	fmt.Println("\n--- Step 7: Broadcast MsgTriggerResolver ---")
	target, err := lib.GetParticipant(client, ctx, childID)
	if err != nil {
		return fmt.Errorf("step 7 failed: query child: %w", err)
	}
	fmt.Printf("  target child %d: op_state=%s validator=%d corp=%d\n",
		childID, target.OpState.String(), target.ValidatorParticipantId, target.CorporationId)

	msg := &permtypes.MsgTriggerResolver{
		Corporation: policyAddr,
		Operator:    operatorAddr,
		Id:          childID,
	}
	txResp, err := client.BroadcastTx(ctx, operatorAccount, msg)
	if err != nil {
		return fmt.Errorf("step 7 failed: broadcast: %w", err)
	}
	if txResp.TxResponse.Code != 0 {
		return fmt.Errorf("step 7 failed: tx code %d: %s", txResp.TxResponse.Code, txResp.TxResponse.RawLog)
	}
	fmt.Printf("OK Step 7: MsgTriggerResolver broadcast (txhash %s)\n", txResp.TxResponse.TxHash)
	waitForTx("trigger resolver")

	// =========================================================================
	// Step 8: Assert the trigger_resolver event with participant_id == childID.
	// =========================================================================
	fmt.Println("\n--- Step 8: Assert trigger_resolver event ---")
	var txResponse sdk.TxResponse
	b, err := client.Context().Codec.MarshalJSON(txResp.TxResponse)
	if err != nil {
		return fmt.Errorf("step 8 failed: marshal tx response: %w", err)
	}
	if err := client.Context().Codec.UnmarshalJSON(b, &txResponse); err != nil {
		return fmt.Errorf("step 8 failed: unmarshal tx response: %w", err)
	}
	found := false
	for _, ev := range txResponse.Events {
		if ev.Type != permtypes.EventTypeTriggerResolver {
			continue
		}
		for _, a := range ev.Attributes {
			if a.Key == "participant_id" && a.Value == childIDStr {
				found = true
			}
		}
	}
	if !found {
		return fmt.Errorf("step 8 failed: %q event with participant_id=%s not found",
			permtypes.EventTypeTriggerResolver, childIDStr)
	}
	fmt.Printf("OK Step 8: %q event emitted with participant_id=%s\n",
		permtypes.EventTypeTriggerResolver, childIDStr)

	fmt.Println("\n========================================")
	fmt.Println("Journey 311 completed successfully!")
	fmt.Println("PP Trigger Resolver (MOD-PP-MSG-15) validated via ancestor-validator authorization.")
	fmt.Println("========================================")
	return nil
}
